package app

import (
	"image"
	"image/color"
	"io"
	"log"
	"log/slog"
	"os"
	"strings"
	"unicode"

	"gioui.org/app"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/denislee/wllinear/internal/config"
	"github.com/denislee/wllinear/internal/db"
	"github.com/denislee/wllinear/internal/linear"
	"github.com/denislee/wllinear/internal/ui"
)

// App glues the Gio window to the State and renders every frame.
type App struct {
	W     *app.Window
	Th    *ui.Theme
	State *State

	teamClicks   []widget.Clickable
	filterClicks []widget.Clickable
	issueClicks  []widget.Clickable

	leadingClicks []widget.Clickable

	listIssues  widget.List
	listSidebar widget.List
	detailList  widget.List
	createList  widget.List
	editList    widget.List

	helpClick   widget.Clickable
	createClick widget.Clickable
	searchClick widget.Clickable
	refreshClick widget.Clickable
	compactClick widget.Clickable

	resizing       bool
	dragStartPosX  float32
	dragStartWidth int
	splitterTag    struct{}
}

// NewApp constructs a new App and dispatches the initial fetches.
func NewApp(w *app.Window, client *linear.Client, d *db.DB, cfg *config.Config, saved *config.State) *App {
	th := ui.New()
	th.ApplyFontPrefs(fontPrefsFromConfig(saved.Fonts))

	st := &State{
		Client:       client,
		DB:           d,
		Cfg:          cfg,
		Saved:        saved,
		Events:       make(chan Event, 256),
		ActiveFilter: saved.LastFilter,
		Compact:      saved.CompactMode,
		ShowLabels:   saved.ShowLabels,
		ShowPriority: saved.ShowPriority,
		HideHints:    saved.HideHints,
		SidebarWidth: saved.SidebarWidth,
		Focus:        FocusSidebar,
		HintsText:    "tab: switch panel  •  c: create  •  ctrl+k: search  •  ,: settings  •  ?: toggle bar • F1: help",
		StatusText:   "Connecting to Linear...",
	}
	st.rebuildFilters()

	a := &App{W: w, Th: th, State: st}
	a.syncLogging()
	a.listIssues.Axis = layout.Vertical
	a.listSidebar.Axis = layout.Vertical
	a.detailList.Axis = layout.Vertical
	a.createList.Axis = layout.Vertical
	a.editList.Axis = layout.Vertical

	st.Wakeup = w.Invalidate

	go fetchViewer(st)
	go fetchTeams(st)
	return a
}

func (a *App) syncLogging() {
	if a.State.Saved.EnableLogging {
		log.SetOutput(os.Stderr)
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	} else {
		log.SetOutput(io.Discard)
		slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	}
}

// Run is the main event loop. It blocks until the window closes.
func (a *App) Run() error {
	var ops op.Ops
	for {
		ev := a.W.Event()
		switch ev := ev.(type) {
		case app.DestroyEvent:
			a.saveState()
			return ev.Err
		case app.FrameEvent:
			// Apply any pending state mutations before laying out so the
			// frame reflects the latest data from background goroutines.
			a.State.Drain()
			gtx := app.NewContext(&ops, ev)
			a.layout(gtx)
			ev.Frame(gtx.Ops)
		}
	}
}

func (a *App) saveState() {
	st := a.State
	defaultStatus := "started"
	if st.Saved != nil && st.Saved.DefaultCreateStatusType != "" {
		defaultStatus = st.Saved.DefaultCreateStatusType
	}
	saved := &config.State{
		LastFilter:              st.ActiveFilter,
		CompactMode:             st.Compact,
		ShowLabels:              st.ShowLabels,
		ShowPriority:            st.ShowPriority,
		HideHints:               st.HideHints,
		SidebarWidth:            st.SidebarWidth,
		Fonts:                   fontPrefsToConfig(a.Th.Fonts),
		DefaultCreateStatusType: defaultStatus,
		EnableLogging:           st.Saved.EnableLogging,
	}
	if st.Team != nil {
		saved.LastTeamID = st.Team.ID
	}
	_ = config.SaveState(saved)
}

func fontPrefsFromConfig(p config.FontPrefs) ui.SectionFonts {
	cv := func(f config.FontPref) ui.FontStyle { return ui.FontStyle{Face: f.Face, Size: f.Size} }
	return ui.SectionFonts{
		Global:      cv(p.Global),
		Sidebar:     cv(p.Sidebar),
		IssueList:   cv(p.IssueList),
		IssueDetail: cv(p.IssueDetail),
		CreateIssue: cv(p.CreateIssue),
		StatusBar:   cv(p.StatusBar),
		Modal:       cv(p.Modal),
		Code:        cv(p.Code),
	}
}

func fontPrefsToConfig(s ui.SectionFonts) config.FontPrefs {
	cv := func(f ui.FontStyle) config.FontPref { return config.FontPref{Face: f.Face, Size: f.Size} }
	return config.FontPrefs{
		Global:      cv(s.Global),
		Sidebar:     cv(s.Sidebar),
		IssueList:   cv(s.IssueList),
		IssueDetail: cv(s.IssueDetail),
		CreateIssue: cv(s.CreateIssue),
		StatusBar:   cv(s.StatusBar),
		Modal:       cv(s.Modal),
		Code:        cv(s.Code),
	}
}


// --- Layout ---

func (a *App) layout(gtx layout.Context) {
	a.handleGlobalKeys(gtx)
	paint.Fill(gtx.Ops, a.Th.BG)

	// Statusbar reserves the bottom row.
	statusH := gtx.Dp(unit.Dp(28))
	hintsH := gtx.Dp(unit.Dp(22))
	if a.State.HideHints {
		hintsH = 0
	}
	bottomH := statusH + hintsH

	body := image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y-bottomH)
	bottom := image.Rect(0, body.Max.Y, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)

	splitterW := gtx.Dp(unit.Dp(6))

	// Handle sidebar resizing. During a drag gesture Gio anchors the event
	// area at press time, so Position.X stays in a consistent basis for the
	// whole gesture — we just diff against the press position.
	for {
		ev, ok := gtx.Event(pointer.Filter{
			Target: &a.splitterTag,
			Kinds:  pointer.Press | pointer.Drag | pointer.Release | pointer.Cancel,
		})
		if !ok {
			break
		}
		p, ok := ev.(pointer.Event)
		if !ok {
			continue
		}
		switch p.Kind {
		case pointer.Press:
			a.resizing = true
			a.dragStartPosX = p.Position.X
			a.dragStartWidth = a.State.SidebarWidth
		case pointer.Drag:
			if !a.resizing {
				break
			}
			deltaDp := int((p.Position.X - a.dragStartPosX) / gtx.Metric.PxPerDp)
			newW := a.dragStartWidth + deltaDp
			if newW < 120 {
				newW = 120
			}
			if newW > 600 {
				newW = 600
			}
			a.State.SidebarWidth = newW
		case pointer.Release, pointer.Cancel:
			if a.resizing {
				a.resizing = false
				a.saveState()
			}
		}
	}

	sidebarW := gtx.Dp(unit.Dp(float32(a.State.SidebarWidth)))
	if sidebarW > body.Dx()/2 {
		sidebarW = body.Dx() / 2
	}
	// Auto-hide the sidebar when the main panel has focus, so the issue
	// list / detail view can use the full width.
	if a.State.Focus == FocusMain {
		sidebarW = 0
	}

	settingsW := 0
	if a.State.ShowSettings {
		settingsW = gtx.Dp(unit.Dp(320))
		if settingsW > body.Dx()/3 {
			settingsW = body.Dx() / 3
		}
	}

	a.layoutSidebar(gtx, image.Rect(body.Min.X, body.Min.Y, body.Min.X+sidebarW, body.Max.Y))
	a.layoutMain(gtx, image.Rect(body.Min.X+sidebarW, body.Min.Y, body.Max.X-settingsW, body.Max.Y))

	// Draw resizing handle.
	splitterRect := image.Rect(sidebarW-splitterW/2, 0, sidebarW+splitterW/2, body.Dy())
	stack := clip.Rect(splitterRect).Push(gtx.Ops)
	event.Op(gtx.Ops, &a.splitterTag)
	pointer.CursorEastWestResize.Add(gtx.Ops)
	stack.Pop()

	if a.State.ShowSettings {
		if a.State.Settings == nil {
			a.State.Settings = NewSettingsModal(a.Th, a.State.Saved.EnableLogging, a.State.HideHints)
		}
		a.layoutSettingsSide(gtx, image.Rect(body.Max.X-settingsW, body.Min.Y, body.Max.X, body.Max.Y))
	}
	
	a.layoutStatusBar(gtx, bottom)

	// Modal overlay last (top z-order).
	if a.State.Modal != ModalNone {
		a.layoutModal(gtx, body)
	}
}

func (a *App) layoutStatusBar(gtx layout.Context, r image.Rectangle) {
	defer op.Offset(r.Min).Push(gtx.Ops).Pop()
	w := r.Dx()
	h := r.Dy()
	// Solid background.
	rect(gtx, image.Rect(0, 0, w, h), a.Th.Panel)
	// Top border.
	rect(gtx, image.Rect(0, 0, w, gtx.Dp(unit.Dp(1))), a.Th.Border)

	layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if a.State.HideHints {
				return layout.Dimensions{}
			}
			return layout.Inset{Top: unit.Dp(3), Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return a.drawStatusText(gtx, a.Th.TextDim, a.State.HintsText)
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(4), Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Spacing: layout.SpaceBetween}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						col := a.Th.Text
						switch a.State.StatusKind {
						case StatusOk:
							col = a.Th.Success
						case StatusWarn:
							col = a.Th.Warning
						case StatusErr:
							col = a.Th.Error
						}
						return a.drawStatusText(gtx, col, a.State.StatusText)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						right := ""
						if a.State.Team != nil {
							right = a.State.Team.Key
						}
						if a.State.ActiveFilter != "" {
							if right != "" {
								right += " · "
							}
							right += a.State.ActiveFilter
						}
						if right == "" {
							return layout.Dimensions{}
						}
						return a.drawStatusText(gtx, a.Th.AccentDim, right)
					}),
				)
			})
		}),
	)
}

func (a *App) drawStatusText(gtx layout.Context, c color.NRGBA, s string) layout.Dimensions {
	if s == "" {
		return layout.Dimensions{}
	}
	l := a.Th.LabelColor(a.Th.Fonts.StatusBar, unit.Sp(12), c, s)
	l.MaxLines = 1
	return l.Layout(gtx)
}

func (a *App) statusTextWidth(gtx layout.Context, s string) int {
	macro := op.Record(gtx.Ops)
	dims := a.drawStatusText(gtx, color.NRGBA{A: 0xFF}, s)
	macro.Stop()
	return dims.Size.X
}

// rect fills the rectangle r with the given color.
func rect(gtx layout.Context, r image.Rectangle, c color.NRGBA) {
	defer clip.Rect(r).Push(gtx.Ops).Pop()
	paint.ColorOp{Color: c}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
}

// rectStroke draws a 1dp border around r in color c.
func rectStroke(gtx layout.Context, r image.Rectangle, c color.NRGBA) {
	one := gtx.Dp(unit.Dp(1))
	rect(gtx, image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+one), c)
	rect(gtx, image.Rect(r.Min.X, r.Max.Y-one, r.Max.X, r.Max.Y), c)
	rect(gtx, image.Rect(r.Min.X, r.Min.Y, r.Min.X+one, r.Max.Y), c)
	rect(gtx, image.Rect(r.Max.X-one, r.Min.Y, r.Max.X, r.Max.Y), c)
}


var globalKeyFilters = []event.Filter{
	key.Filter{Name: key.NameTab, Optional: key.ModShift},
	key.Filter{Name: "Q"},
	key.Filter{Name: "C"},
	key.Filter{Name: "V"},
	key.Filter{Name: "?"},
	key.Filter{Name: key.NameF1},
	key.Filter{Name: "/"},
	key.Filter{Name: ","},
	key.Filter{Name: "K", Required: key.ModCtrl},
	key.Filter{Name: "[", Required: key.ModCtrl},
	key.Filter{Name: "F", Required: key.ModCtrl},
	key.Filter{Name: "B", Required: key.ModCtrl},
	key.Filter{Name: key.NameEscape},
	key.Filter{Name: key.NameReturn},
	key.Filter{Name: key.NameSpace},
	key.Filter{Name: key.NameUpArrow},
	key.Filter{Name: key.NameDownArrow},
	key.Filter{Name: "J"},
	key.Filter{Name: "K"},
	key.Filter{Name: "H"},
	key.Filter{Name: "L"},
	key.Filter{Name: "S"},
	key.Filter{Name: "E"},
	key.Filter{Name: "R"},
	key.Filter{Name: "T"},
	key.Filter{Name: "T", Required: key.ModCtrl},
	key.Filter{Name: "W", Required: key.ModCtrl},
	key.Filter{Name: key.NameDeleteBackward, Required: key.ModCtrl},
	key.Filter{Name: "N", Required: key.ModCtrl},
	key.Filter{Name: "P", Required: key.ModCtrl},
	key.Filter{Name: "P"},
	key.Filter{Name: "Y"},
}

// handleGlobalKeys subscribes to keyboard events on the window root.
func (a *App) handleGlobalKeys(gtx layout.Context) {
	// Subscribe by adding key.Filter inputs.
	for {
		ev, ok := gtx.Event(globalKeyFilters...)
		if !ok {
			break
		}
		ke, ok := ev.(key.Event)
		if !ok || ke.State != key.Press {
			continue
		}
		// Don't let printable letter hotkeys steal keystrokes while the user
		// is typing in an editor. Modifier-bearing combos (Ctrl+K, etc.)
		// still pass through, but Shift+letter (uppercase) or Shift+/ (?)
		// are blocked.
		if (ke.Modifiers == 0 || ke.Modifiers == key.ModShift) && a.isEditorFocused(gtx) && isPrintableHotkey(ke.Name) {
			continue
		}
		a.handleKey(gtx, ke)
	}
	event.Op(gtx.Ops, a)
}

// isEditorFocused reports whether any of the app's text editors currently
// holds key focus.
func (a *App) isEditorFocused(gtx layout.Context) bool {
	return a.getFocusedEditor(gtx) != nil
}

func (a *App) getFocusedEditor(gtx layout.Context) *widget.Editor {
	if a.State.Create != nil {
		if gtx.Focused(&a.State.Create.Title) {
			return &a.State.Create.Title
		}
		if gtx.Focused(&a.State.Create.Description) {
			return &a.State.Create.Description
		}
	}
	if a.State.Edit != nil {
		if gtx.Focused(&a.State.Edit.Title) {
			return &a.State.Edit.Title
		}
		if gtx.Focused(&a.State.Edit.Description) {
			return &a.State.Edit.Description
		}
	}
	switch m := a.State.ModalState.(type) {
	case *CreateModal:
		if gtx.Focused(&m.Title) {
			return &m.Title
		}
		if gtx.Focused(&m.Description) {
			return &m.Description
		}
	case *EditModal:
		if gtx.Focused(&m.Title) {
			return &m.Title
		}
		if gtx.Focused(&m.Description) {
			return &m.Description
		}
	case *SearchModal:
		if gtx.Focused(&m.Query) {
			return &m.Query
		}
	}
	return nil
}

func (a *App) deleteWordBeforeCaret(ed *widget.Editor) {
	start, end := ed.Selection()
	if start != end {
		ed.Insert("")
		return
	}

	pos := start
	if pos <= 0 {
		return
	}

	txt := ed.Text()
	runes := []rune(txt)
	if pos > len(runes) {
		pos = len(runes)
	}

	i := pos - 1
	// Skip trailing spaces
	for i >= 0 && unicode.IsSpace(runes[i]) {
		i--
	}
	// Skip the word
	for i >= 0 && !unicode.IsSpace(runes[i]) {
		i--
	}

	newStart := i + 1
	ed.SetCaret(newStart, pos)
	ed.Insert("")
}

func isPrintableHotkey(name key.Name) bool {
	if len(name) != 1 {
		return false
	}
	c := name[0]
	return (c >= 'A' && c <= 'Z') || c == '?' || c == '/' || c == ','
}

func (a *App) handleKey(gtx layout.Context, ke key.Event) {
	st := a.State

	if ke.Modifiers == key.ModCtrl && (ke.Name == "W" || ke.Name == key.NameDeleteBackward) {
		if ed := a.getFocusedEditor(gtx); ed != nil {
			a.deleteWordBeforeCaret(ed)
			return
		}
	}

	// Global toggles that work everywhere (unless blocked by editor focus in handleGlobalKeys).
	switch ke.Name {
	case "?":
		st.HideHints = !st.HideHints
		if st.Settings != nil {
			st.Settings.HintsToggle.Value = !st.HideHints
		}
		a.saveState()
		return
	case key.NameF1:
		a.openHelp()
		return
	case "V":
		if ke.Modifiers == 0 {
			st.Compact = !st.Compact
			a.saveState()
			return
		}
	case "P":
		if ke.Modifiers == 0 {
			st.ShowPriority = !st.ShowPriority
			a.saveState()
			return
		}
	case ",":
		if ke.Modifiers == 0 {
			a.openSettings()
			return
		}
	}

	if st.Modal != ModalNone {
		a.handleModalKey(ke)
		return
	}

	if st.View == ViewCreateIssue {
		m := st.Create
		if m != nil {
			switch ke.Name {
			case key.NameEscape:
				a.closeCreate()
			case "[":
				if ke.Modifiers.Contain(key.ModCtrl) {
					a.closeCreate()
				}
			case key.NameTab:
				if ke.Modifiers.Contain(key.ModShift) {
					m.FocusIdx = (m.FocusIdx + 6) % 7
				} else {
					m.FocusIdx = (m.FocusIdx + 1) % 7
				}
				m.FocusReq = true
			case "J", key.NameDownArrow, key.NameRightArrow:
				if m.FocusIdx == 2 { // Priority
					m.Priority = clamp(m.Priority+1, 0, 4)
				} else if m.FocusIdx == 3 && m.Meta != nil { // Status
					m.StateIdx = clamp(m.StateIdx+1, 0, len(m.Meta.States)-1)
				} else if m.FocusIdx == 4 { // Project
					m.ProjectIdx = clamp(m.ProjectIdx+1, 0, len(a.State.LeadingProjects)-1)
				} else if m.FocusIdx == 5 && m.Meta != nil { // Cycle
					m.CycleIdx = clamp(m.CycleIdx+1, 0, len(m.Meta.Cycles)-1)
				}
			case "K", key.NameUpArrow, key.NameLeftArrow:
				if m.FocusIdx == 2 { // Priority
					m.Priority = clamp(m.Priority-1, 0, 4)
				} else if m.FocusIdx == 3 && m.Meta != nil { // Status
					m.StateIdx = clamp(m.StateIdx-1, 0, len(m.Meta.States)-1)
				} else if m.FocusIdx == 4 { // Project
					m.ProjectIdx = clamp(m.ProjectIdx-1, 0, len(a.State.LeadingProjects)-1)
				} else if m.FocusIdx == 5 && m.Meta != nil { // Cycle
					m.CycleIdx = clamp(m.CycleIdx-1, 0, len(m.Meta.Cycles)-1)
				}
			case key.NameReturn:
				if m.FocusIdx == 6 {
					a.confirmCreateScreen()
				}
			}
		}
		return
	}

	if st.View == ViewEditIssue {
		m := st.Edit
		if m != nil {
			switch ke.Name {
			case key.NameEscape:
				a.closeEdit()
			case "[":
				if ke.Modifiers.Contain(key.ModCtrl) {
					a.closeEdit()
				}
			case key.NameTab:
				if ke.Modifiers.Contain(key.ModShift) {
					m.FocusIdx = (m.FocusIdx + 6) % 7
				} else {
					m.FocusIdx = (m.FocusIdx + 1) % 7
				}
				m.FocusReq = true
			case "J", key.NameDownArrow, key.NameRightArrow:
				if m.FocusIdx == 2 {
					m.Priority = clamp(m.Priority+1, 0, 4)
				} else if m.FocusIdx == 3 && m.Meta != nil {
					m.StateIdx = clamp(m.StateIdx+1, 0, len(m.Meta.States)-1)
				} else if m.FocusIdx == 4 {
					m.ProjectIdx = clamp(m.ProjectIdx+1, 0, len(a.State.LeadingProjects)-1)
				} else if m.FocusIdx == 5 && m.Meta != nil {
					m.CycleIdx = clamp(m.CycleIdx+1, 0, len(m.Meta.Cycles)-1)
				}
			case "K", key.NameUpArrow, key.NameLeftArrow:
				if m.FocusIdx == 2 {
					m.Priority = clamp(m.Priority-1, 0, 4)
				} else if m.FocusIdx == 3 && m.Meta != nil {
					m.StateIdx = clamp(m.StateIdx-1, 0, len(m.Meta.States)-1)
				} else if m.FocusIdx == 4 {
					m.ProjectIdx = clamp(m.ProjectIdx-1, 0, len(a.State.LeadingProjects)-1)
				} else if m.FocusIdx == 5 && m.Meta != nil {
					m.CycleIdx = clamp(m.CycleIdx-1, 0, len(m.Meta.Cycles)-1)
				}
			case key.NameReturn:
				if m.FocusIdx == 6 {
					a.confirmEditScreen()
				}
			}
		}
		return
	}

	switch ke.Name {
	case key.NameEscape:
		if st.View == ViewIssueDetail {
			st.View = ViewIssueList
			st.Detail = nil
		}
		return
	case key.NameTab:
		if st.Focus == FocusSidebar {
			count := len(st.Issues)
			if st.View == ViewProjectCycles {
				count = len(st.ProjectCycles)
			}
			if count > 0 {
				st.Focus = FocusMain
			}
		} else {
			st.Focus = FocusSidebar
		}
		a.updateHints()
		return
	}

	switch ke.Name {
	case "C":
		a.openCreate()
		return
	case "F":
		if ke.Modifiers == key.ModCtrl {
			if st.Focus == FocusSidebar {
				a.moveSidebar(max(1, a.listSidebar.Position.Count-1))
			} else {
				a.moveIssue(max(1, a.listIssues.Position.Count-1))
			}
		}
		return
	case "B":
		if ke.Modifiers == key.ModCtrl {
			if st.Focus == FocusSidebar {
				a.moveSidebar(-max(1, a.listSidebar.Position.Count-1))
			} else {
				a.moveIssue(-max(1, a.listIssues.Position.Count-1))
			}
		}
		return
	case "K", key.NameUpArrow:
		if ke.Modifiers == key.ModCtrl && ke.Name == "K" {
			a.openSearch()
			return
		}
		if st.View == ViewIssueDetail {
			a.detailList.ScrollBy(-1)
			return
		}
		if st.Focus == FocusSidebar {
			a.moveSidebar(-1)
		} else {
			a.moveIssue(-1)
		}
		return
	case "J", key.NameDownArrow:
		if st.View == ViewIssueDetail {
			a.detailList.ScrollBy(1)
			return
		}
		if st.Focus == FocusSidebar {
			a.moveSidebar(1)
		} else {
			a.moveIssue(1)
		}
		return
	case "N":
		if ke.Modifiers == key.ModCtrl {
			if st.Focus == FocusSidebar {
				a.moveSidebar(1)
			} else {
				a.moveIssue(1)
			}
			return
		}
	case "P":
		if ke.Modifiers == key.ModCtrl {
			if st.Focus == FocusSidebar {
				a.moveSidebar(-1)
			} else {
				a.moveIssue(-1)
			}
			return
		}
		if ke.Modifiers == 0 {
			st.ShowPriority = !st.ShowPriority
			a.saveState()
			return
		}
	case "H":
		if st.View == ViewIssueDetail {
			st.View = ViewIssueList
			st.Detail = nil
			return
		}
		st.Focus = FocusSidebar
		a.updateHints()
		return
	case "L":
		if st.Focus == FocusSidebar {
			count := len(st.Issues)
			if st.View == ViewProjectCycles {
				count = len(st.ProjectCycles)
			}
			if count > 0 {
				st.Focus = FocusMain
				a.updateHints()
			}
			return
		}
		a.openSelectedDetail()
		return
	case key.NameReturn:
		if st.Focus == FocusSidebar {
			a.activateSidebar()
			return
		}
		if st.View == ViewProjectCycles {
			a.toggleSelectedCycle()
			return
		}
		a.openSelectedInBrowser()
		return
	case key.NameSpace:
		if st.Focus == FocusMain && st.View == ViewProjectCycles {
			a.toggleSelectedCycle()
		}
		return
	case "S":
		if is := a.currentIssue(); is != nil {
			a.openStatus(*is)
		}
		return
	case "E":
		if is := a.currentIssue(); is != nil {
			a.openEdit(*is)
		}
		return
	case "R":
		if ke.Modifiers == key.ModCtrl {
			st.StatusText = "Refreshing..."
			if st.Team != nil {
				go fetchIssues(st, st.Team.ID, st.ActiveFilter)
				go fetchFilterCounts(st, st.Team.ID, st.Filters)
			}
			go fetchLeadingProjects(st)
			return
		}
		if st.Team != nil {
			go fetchIssues(st, st.Team.ID, st.ActiveFilter)
			go fetchFilterCounts(st, st.Team.ID, st.Filters)
		}
		return
	case "T":
		if ke.Modifiers == key.ModCtrl {
			if st.Modal == ModalTeam {
				a.closeModal()
			} else {
				a.openTeam()
			}
			return
		}
		if st.ActiveFilter == "My Unlabeled Issues" {
			issues := append([]linear.Issue(nil), st.Issues...)
			go AutoLabel(st, issues)
		} else {
			st.ShowLabels = !st.ShowLabels
			a.saveState()
		}
		return
	case "Y":
		if st.View == ViewProjectCycles {
			if st.Focus == FocusMain {
				a.copySelectedCycleIssues()
			}
		} else if is := a.currentIssue(); is != nil {
			CopyIssue(st, *is)
		}
		return
	case "Q":
		a.W.Perform(system.ActionClose)
		return
	case "/":
		// Focus the issue query editor; routed via clicking it.
		gtx := layout.Context{} // not great, but key handling lives outside layout here
		_ = gtx
		// We can't focus from here without a gtx; rely on user clicking the field.
		return
	}
}

func (a *App) updateHints() {
	st := a.State
	if st.Modal != ModalNone {
		st.HintsText = "tab: fields  •  enter: submit  •  esc: cancel  •  ?: toggle bar  •  F1: help"
		return
	}
	if st.View == ViewCreateIssue {
		st.HintsText = "esc: cancel  •  click Create Issue to submit"
		return
	}
	if st.View == ViewEditIssue {
		st.HintsText = "esc: cancel  •  click Save Changes to submit"
		return
	}
	if st.View == ViewIssueDetail {
		st.HintsText = "esc/h: back  •  e: edit  •  s: status  •  ?: toggle bar • F1: help"
		return
	}
	switch st.Focus {
	case FocusSidebar:
		st.HintsText = "j/k: navigate  •  enter/l: select  •  c: create  •  ctrl+k: search  •  ctrl+r: reload  •  tab: issues  •  ?: toggle bar • F1: help"
	case FocusMain:
		hints := "j/k: navigate  •  enter: browser  •  l: open  •  e: edit  •  s: status  •  c: create  •  v: compact  •  t: tags  •  r: refresh  •  ctrl+r: reload  •  ?: toggle bar • F1: help"
		if st.View == ViewProjectCycles {
			hints = "j/k: navigate  •  space/enter: expand  •  y: copy issues  •  r: refresh  •  ctrl+r: reload  •  ?: toggle bar • F1: help"
		} else if st.ActiveFilter == "My Unlabeled Issues" {
			hints = "t: auto-label  •  j/k: navigate  •  enter: browser  •  l: open  •  e: edit  •  s: status  •  c: create  •  v: compact  •  r: refresh  •  ctrl+r: reload  •  ?: toggle bar • F1: help"
		}
		st.HintsText = hints
	}
}

func (a *App) moveSidebar(d int) {
	st := a.State
	// Create a combined list of selectable items: Filters + Projects.
	type item struct {
		name    string
		isProj  bool
		project linear.Project
	}
	var items []item
	for _, f := range st.Filters {
		if f != "---" {
			items = append(items, item{name: f})
		}
	}
	for _, p := range st.LeadingProjects {
		items = append(items, item{name: "Project: " + p.Name, isProj: true, project: p})
	}

	if len(items) == 0 {
		return
	}

	cur := -1
	for i, it := range items {
		if it.name == st.ActiveFilter {
			cur = i
			break
		}
	}

	if cur < 0 {
		cur = 0
	} else {
		cur = (cur + d + len(items)) % len(items)
	}

	it := items[cur]
	st.ActiveFilter = it.name
	if it.isProj {
		st.PostEvent(ProjectSelected{Project: it.project})
	} else {
		st.PostEvent(FilterSelected{Filter: it.name})
	}
}

func (a *App) activateSidebar() {
	st := a.State
	if strings.HasPrefix(st.ActiveFilter, "Project: ") {
		name := strings.TrimPrefix(st.ActiveFilter, "Project: ")
		for _, p := range st.LeadingProjects {
			if p.Name == name {
				st.PostEvent(ProjectSelected{Project: p})
				return
			}
		}
	}
	st.PostEvent(FilterSelected{Filter: st.ActiveFilter})
}

func (a *App) moveIssue(d int) {
	st := a.State
	var count int
	if st.View == ViewProjectCycles {
		count = len(st.ProjectCycles)
	} else {
		count = len(st.Issues)
	}

	if count == 0 {
		return
	}
	st.Selected += d
	if st.Selected < 0 {
		st.Selected = 0
	}
	if st.Selected >= count {
		st.Selected = count - 1
	}

	// Keep list aligned with selection
	first := a.listIssues.Position.First
	visible := a.listIssues.Position.Count
	if visible > 0 {
		if st.Selected < first {
			a.listIssues.Position.First = st.Selected
			a.listIssues.Position.Offset = 0
		} else if st.Selected == first && a.listIssues.Position.Offset != 0 {
			// If it's the first visible item but partially cut off at the top, snap to it
			a.listIssues.Position.Offset = 0
		} else if st.Selected >= first+visible {
			a.listIssues.Position.First = st.Selected - visible + 1
			a.listIssues.Position.Offset = 0
		} else if st.Selected == first+visible-1 && a.listIssues.Position.OffsetLast > 0 {
			// If it's the last visible item but partially cut off at the bottom, scroll down by one
			a.listIssues.Position.First++
			a.listIssues.Position.Offset = 0
		}
	} else {
		a.listIssues.Position.First = st.Selected
		a.listIssues.Position.Offset = 0
	}

	if st.View == ViewIssueList {
		a.updateHints()
	}
}
func (a *App) toggleSelectedCycle() {
	st := a.State
	if st.Selected < 0 || st.Selected >= len(st.ProjectCycles) {
		return
	}
	id := st.ProjectCycles[st.Selected].Cycle.ID
	if st.ExpandedCycles == nil {
		st.ExpandedCycles = make(map[string]bool)
	}
	st.ExpandedCycles[id] = !st.ExpandedCycles[id]
}

func (a *App) copySelectedCycleIssues() {
	st := a.State
	if st.Selected < 0 || st.Selected >= len(st.ProjectCycles) {
		return
	}
	go CopyCycleIssues(st, st.ProjectCycles[st.Selected])
}

func (a *App) currentIssue() *linear.Issue {
	if a.State.View == ViewIssueDetail && a.State.Detail != nil {
		return a.State.Detail
	}
	issues := a.State.Issues
	if a.State.Selected < 0 || a.State.Selected >= len(issues) {
		return nil
	}
	is := issues[a.State.Selected]
	return &is
}

func (a *App) openSelectedDetail() {
	if is := a.currentIssue(); is != nil {
		issue := *is
		a.State.Detail = &issue
		a.State.View = ViewIssueDetail
		a.updateHints()
	}
}

func (a *App) openSelectedInBrowser() {
	if is := a.currentIssue(); is != nil {
		OpenBrowser(is.URL)
	}
}

func (a *App) openCreate() {
	st := a.State
	st.View = ViewCreateIssue
	st.Detail = nil
	st.Create = NewCreateModal(st)
	if st.Team != nil && st.Meta == nil {
		go fetchTeamMetadata(st, st.Team.ID)
	}
	a.updateHints()
}

func (a *App) closeCreate() {
	a.State.Create = nil
	a.State.View = ViewIssueList
	a.updateHints()
}

func (a *App) confirmCreateScreen() {
	st := a.State
	if st.Create == nil {
		return
	}
	in, valid := st.Create.Build(st)
	if !valid {
		st.StatusText = "Title is required"
		st.StatusKind = StatusWarn
		return
	}
	go createIssue(st, in)
	a.closeCreate()
}

func (a *App) openEdit(issue linear.Issue) {
	st := a.State
	st.View = ViewEditIssue
	st.Detail = nil
	st.Edit = NewEditModal(st, issue)
	if st.Team != nil && st.Meta == nil {
		go fetchTeamMetadata(st, st.Team.ID)
	}
	a.updateHints()
}

func (a *App) closeEdit() {
	a.State.Edit = nil
	a.State.View = ViewIssueList
	a.updateHints()
}

func (a *App) confirmEditScreen() {
	st := a.State
	if st.Edit == nil {
		return
	}
	id, in, valid := st.Edit.Build(st)
	if !valid {
		st.StatusText = "Title is required"
		st.StatusKind = StatusWarn
		return
	}
	go editIssue(st, id, in)
	a.closeEdit()
}

func (a *App) openStatus(issue linear.Issue) {
	st := a.State
	st.Modal = ModalStatus
	st.ModalState = NewStatusModal(issue)
	if st.Team != nil {
		go fetchWorkflowStates(st, st.Team.ID)
	}
	a.updateHints()
}
func (a *App) openSettings() {
	st := a.State
	st.ShowSettings = !st.ShowSettings
	if st.ShowSettings && st.Settings == nil {
		st.Settings = NewSettingsModal(a.Th, st.Saved.EnableLogging, st.HideHints)
	}
	a.updateHints()
}

func (a *App) openSearch() {
	st := a.State
	if st.Modal == ModalSearch {
		a.closeModal()
		return
	}
	st.Modal = ModalSearch
	st.ModalState = NewSearchModal()
	go fetchMyIssues(st)
	a.updateHints()
}

func (a *App) openTeam() {
	st := a.State
	st.Modal = ModalTeam
	st.ModalState = NewTeamModal()
	if m, ok := st.ModalState.(*TeamModal); ok {
		m.SetTeams(st.Teams)
	}
	a.updateHints()
}

func (a *App) openHelp() {
	st := a.State
	if st.Modal == ModalHelp {
		a.closeModal()
	} else {
		st.Modal = ModalHelp
		a.updateHints()
	}
}

func (a *App) closeModal() {
	a.State.Modal = ModalNone
	a.State.ModalState = nil
	a.updateHints()
}

// indexOf returns the first index of s in xs, or -1.
func indexOf(xs []string, s string) int {
	for i, x := range xs {
		if x == s {
			return i
		}
	}
	return -1
}

// --- Helpers used by panel files ---

// cleanProjectName removes anything between '[' and ']' inclusive.
func cleanProjectName(s string) string {
	start := strings.IndexByte(s, '[')
	if start != -1 {
		end := strings.IndexByte(s[start:], ']')
		if end != -1 {
			s = s[:start] + s[start+end+1:]
		}
	}
	return strings.TrimSpace(s)
}

// truncate returns s truncated to n runes with an ellipsis if needed.
func truncate(s string, n int) string {
	if n <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 1 {
		return string(r[:n])
	}
	return string(r[:n-1]) + "…"
}

// padToWidth pads s with spaces on the right so its rune count is at least n.
func padToWidth(s string, n int) string {
	r := []rune(s)
	if len(r) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len(r))
}

// editorStyle constructs a styled material editor wrapping ed.
// The optional section overrides typeface and base size.
func editorStyle(th *ui.Theme, ed *widget.Editor, hint string, c color.NRGBA, fs ui.FontStyle) material.EditorStyle {
	e := th.Editor(fs, ed, hint, c)
	_ = text.Start
	return e
}
