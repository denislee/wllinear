package app

import (
	"unicode"

	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/widget"

	"github.com/denislee/wllinear/internal/linear"
)

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
	key.Filter{Name: "E", Required: key.ModCtrl},
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

	if ke.Modifiers == key.ModCtrl && ke.Name == "E" {
		if ed := a.getFocusedEditor(gtx); ed != nil {
			runes := []rune(ed.Text())
			n := len(runes)
			ed.SetCaret(n, n)
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
					m.FocusIdx = (m.FocusIdx + 7) % 8
				} else {
					m.FocusIdx = (m.FocusIdx + 1) % 8
				}
				m.FocusReq = true
			case "J", key.NameDownArrow, key.NameRightArrow:
				if m.FocusIdx == 2 { // Priority
					m.Priority = clamp(m.Priority+1, 0, 4)
				} else if m.FocusIdx == 3 && m.Meta != nil { // Status
					m.StateIdx = clamp(m.StateIdx+1, 0, len(m.Meta.States)-1)
				} else if m.FocusIdx == 4 && m.Meta != nil { // Assignee
					m.AssigneeIdx = clamp(m.AssigneeIdx+1, 0, len(m.Meta.Members)-1)
				} else if m.FocusIdx == 5 { // Project
					m.ProjectIdx = clamp(m.ProjectIdx+1, 0, len(a.State.LeadingProjects)-1)
				} else if m.FocusIdx == 6 && m.Meta != nil { // Cycle
					m.CycleIdx = clamp(m.CycleIdx+1, 0, len(m.Meta.Cycles)-1)
				}
			case "K", key.NameUpArrow, key.NameLeftArrow:
				if m.FocusIdx == 2 { // Priority
					m.Priority = clamp(m.Priority-1, 0, 4)
				} else if m.FocusIdx == 3 && m.Meta != nil { // Status
					m.StateIdx = clamp(m.StateIdx-1, 0, len(m.Meta.States)-1)
				} else if m.FocusIdx == 4 && m.Meta != nil { // Assignee
					m.AssigneeIdx = clamp(m.AssigneeIdx-1, 0, len(m.Meta.Members)-1)
				} else if m.FocusIdx == 5 { // Project
					m.ProjectIdx = clamp(m.ProjectIdx-1, 0, len(a.State.LeadingProjects)-1)
				} else if m.FocusIdx == 6 && m.Meta != nil { // Cycle
					m.CycleIdx = clamp(m.CycleIdx-1, 0, len(m.Meta.Cycles)-1)
				}
			case key.NameReturn:
				if m.FocusIdx == 7 {
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
					m.FocusIdx = (m.FocusIdx + 8) % 9
				} else {
					m.FocusIdx = (m.FocusIdx + 1) % 9
				}
				m.FocusReq = true
			case "J", key.NameDownArrow, key.NameRightArrow:
				if m.FocusIdx == 2 {
					m.Priority = clamp(m.Priority+1, 0, 4)
				} else if m.FocusIdx == 3 && m.Meta != nil {
					m.StateIdx = clamp(m.StateIdx+1, 0, len(m.Meta.States)-1)
				} else if m.FocusIdx == 4 && m.Meta != nil {
					m.AssigneeIdx = clamp(m.AssigneeIdx+1, 0, len(m.Meta.Members)-1)
				} else if m.FocusIdx == 5 && m.Meta != nil {
					m.LabelIdx = clamp(m.LabelIdx+1, 0, len(m.Meta.Labels)-1)
				} else if m.FocusIdx == 6 {
					m.ProjectIdx = clamp(m.ProjectIdx+1, 0, len(a.State.LeadingProjects)-1)
				} else if m.FocusIdx == 7 && m.Meta != nil {
					m.CycleIdx = clamp(m.CycleIdx+1, 0, len(m.Meta.Cycles)-1)
				}
			case "K", key.NameUpArrow, key.NameLeftArrow:
				if m.FocusIdx == 2 {
					m.Priority = clamp(m.Priority-1, 0, 4)
				} else if m.FocusIdx == 3 && m.Meta != nil {
					m.StateIdx = clamp(m.StateIdx-1, 0, len(m.Meta.States)-1)
				} else if m.FocusIdx == 4 && m.Meta != nil {
					m.AssigneeIdx = clamp(m.AssigneeIdx-1, 0, len(m.Meta.Members)-1)
				} else if m.FocusIdx == 5 && m.Meta != nil {
					m.LabelIdx = clamp(m.LabelIdx-1, 0, len(m.Meta.Labels)-1)
				} else if m.FocusIdx == 6 {
					m.ProjectIdx = clamp(m.ProjectIdx-1, 0, len(a.State.LeadingProjects)-1)
				} else if m.FocusIdx == 7 && m.Meta != nil {
					m.CycleIdx = clamp(m.CycleIdx-1, 0, len(m.Meta.Cycles)-1)
				}
			case key.NameReturn:
				if m.FocusIdx == 8 {
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
				filter := buildIssueFilter(st.ActiveFilter, st)
				snap := snapshotFilters(st, st.Filters)
				go fetchIssues(st, st.Team.ID, st.ActiveFilter, filter, true)
				go fetchFilterCounts(st, st.Team.ID, snap)
			}
			go fetchLeadingProjects(st)
			return
		}
		if st.Team != nil {
			filter := buildIssueFilter(st.ActiveFilter, st)
			snap := snapshotFilters(st, st.Filters)
			go fetchIssues(st, st.Team.ID, st.ActiveFilter, filter, true)
			go fetchFilterCounts(st, st.Team.ID, snap)
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
