package app

import (
	"image"
	"image/color"
	"log"
	"strconv"
	"strings"

	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/denislee/wllinear/internal/linear"
	"github.com/denislee/wllinear/internal/ui"
)

func (a *App) layoutModal(gtx layout.Context, body image.Rectangle) {
	defer op.Offset(body.Min).Push(gtx.Ops).Pop()
	w, h := body.Dx(), body.Dy()
	// Dim background.
	rect(gtx, image.Rect(0, 0, w, h), color.NRGBA{A: 0xC0})

	gtx.Constraints = layout.Constraints{Min: image.Pt(w, h), Max: image.Pt(w, h)}

	switch a.State.Modal {
	case ModalHelp:
		a.layoutHelpModal(gtx)
	case ModalCreate:
		a.layoutCreateModal(gtx)
	case ModalStatus:
		a.layoutStatusModal(gtx)
	case ModalSearch:
		a.layoutSearchModal(gtx)
	case ModalTeam:
		a.layoutTeamModal(gtx)
	case ModalComment:
		a.layoutCommentModal(gtx)
	case ModalProjectInfo:
		a.layoutProjectInfoModal(gtx)
	}
}

// modalCard wraps content in a centered, bordered, padded card.
func (a *App) modalCard(gtx layout.Context, maxW, maxH int, w layout.Widget) layout.Dimensions {
	if maxW > gtx.Constraints.Max.X-gtx.Dp(unit.Dp(40)) {
		maxW = gtx.Constraints.Max.X - gtx.Dp(unit.Dp(40))
	}
	if maxH > gtx.Constraints.Max.Y-gtx.Dp(unit.Dp(40)) {
		maxH = gtx.Constraints.Max.Y - gtx.Dp(unit.Dp(40))
	}

	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Max.X = maxW
		gtx.Constraints.Max.Y = maxH
		gtx.Constraints.Min = image.Point{}
		return layout.Stack{}.Layout(gtx,
			layout.Expanded(func(gtx layout.Context) layout.Dimensions {
				r := image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)
				rrr := gtx.Dp(unit.Dp(8))
				defer clip.UniformRRect(r, rrr).Push(gtx.Ops).Pop()
				rect(gtx, r, a.Th.Panel)
				rectStroke(gtx, r, a.Th.BorderHi)
				return layout.Dimensions{Size: gtx.Constraints.Max}
			}),
			layout.Stacked(func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(20)).Layout(gtx, w)
			}),
		)
	})
}

// --- Help ---

func (a *App) layoutHelpModal(gtx layout.Context) layout.Dimensions {
	sections := a.helpSectionsForContext()

	fs := a.Th.Fonts.Modal
	return a.modalCard(gtx, gtx.Dp(unit.Dp(560)), gtx.Dp(unit.Dp(560)), func(gtx layout.Context) layout.Dimensions {
		children := []layout.FlexChild{
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				l := a.Th.LabelColor(fs, unit.Sp(18), a.Th.AccentDim, "Keyboard Shortcuts")
				l.Font.Weight = 700
				return l.Layout(gtx)
			}),
			layout.Rigid(rigidSpace(12)),
		}
		for _, s := range sections {
			s := s
			children = append(children,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					l := a.Th.LabelColor(fs, unit.Sp(13), a.Th.Text, s.title)
					l.Font.Weight = 700
					return l.Layout(gtx)
				}),
				layout.Rigid(rigidSpace(4)),
			)
			for _, p := range s.keys {
				p := p
				children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.Th.LabelColor(fs, unit.Sp(12), a.Th.Accent, padToWidth(p.k, 20)).Layout(gtx)
						}),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return a.Th.LabelColor(fs, unit.Sp(12), a.Th.TextDim, p.d).Layout(gtx)
						}),
					)
				}))
			}
			children = append(children, layout.Rigid(rigidSpace(8)))
		}
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.Th.LabelColor(fs, unit.Sp(11), a.Th.TextMuted, "Press esc, ?, or F1 to close").Layout(gtx)
		}))
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	})
}

// helpPair is one row in the help overlay: a key combo and its description.
type helpPair struct{ k, d string }

// helpSection groups related shortcuts under a title.
type helpSection struct {
	title string
	keys  []helpPair
}

// helpSectionsForContext returns only the shortcut groups relevant to the
// current screen (view, focus, or open modal), so the lightbox shows what
// the user can actually do right now.
func (a *App) helpSectionsForContext() []helpSection {
	st := a.State

	global := helpSection{"Global", []helpPair{
		{"q / ctrl+c", "quit"},
		{"ctrl+r", "refresh issues & projects"},
		{",", "settings (fonts, hints)"},
		{"? / F1", "this help"},
	}}
	closeModal := helpSection{"Modal", []helpPair{
		{"esc", "close"},
		{"j / k", "navigate"},
		{"enter", "submit"},
	}}

	if st.Modal != ModalNone {
		switch st.Modal {
		case ModalHelp:
			return []helpSection{
				{"Help", []helpPair{
					{"esc / ? / F1", "close"},
				}},
				global,
			}
		case ModalSearch:
			return []helpSection{
				{"Search", []helpPair{
					{"esc / ctrl+k", "close"},
					{"j / k / ↑ / ↓", "navigate results"},
					{"ctrl+n / ctrl+p", "next / previous"},
					{"enter", "open selected"},
				}},
				global,
			}
		case ModalStatus:
			return []helpSection{
				{"Change Status", []helpPair{
					{"esc / s / ctrl+[", "close"},
					{"j / k / ↑ / ↓", "navigate"},
					{"enter", "apply"},
				}},
				global,
			}
		case ModalTeam:
			return []helpSection{
				{"Switch Team", []helpPair{
					{"esc / ctrl+t", "close"},
					{"j / k / ↑ / ↓", "navigate"},
					{"enter", "select team"},
				}},
				global,
			}
		case ModalComment:
			return []helpSection{
				{"Add Update", []helpPair{
					{"esc", "cancel"},
					{"ctrl+enter", "submit"},
				}},
				global,
			}
		case ModalProjectInfo:
			return []helpSection{
				{"Project Info", []helpPair{
					{"esc / h / ctrl+[ / ctrl+i", "close"},
					{"j / k / ↑ / ↓", "navigate fields"},
					{"y", "copy selected value"},
				}},
				global,
			}
		default:
			return []helpSection{closeModal, global}
		}
	}

	switch st.View {
	case ViewCreateIssue:
		return []helpSection{
			{"Create Issue", []helpPair{
				{"esc / ctrl+[", "cancel"},
				{"tab / shift+tab", "next / previous field"},
				{"j / k / ↑ / ↓ / ← / →", "adjust selected field"},
				{"enter", "submit (on Submit button)"},
			}},
			{"Editors", []helpPair{
				{"ctrl+w / ctrl+bksp", "delete word"},
				{"ctrl+e", "caret to end"},
			}},
			global,
		}
	case ViewEditIssue:
		return []helpSection{
			{"Edit Issue", []helpPair{
				{"esc / ctrl+[", "cancel"},
				{"tab / shift+tab", "next / previous field"},
				{"j / k / ↑ / ↓ / ← / →", "adjust selected field"},
				{"enter", "submit (on Submit button)"},
			}},
			{"Editors", []helpPair{
				{"ctrl+w / ctrl+bksp", "delete word"},
				{"ctrl+e", "caret to end"},
			}},
			global,
		}
	case ViewIssueDetail:
		return []helpSection{
			{"Issue Detail", []helpPair{
				{"esc / h", "back"},
				{"j / k / ↑ / ↓", "scroll"},
				{"ctrl+f / ctrl+b", "page down / up"},
				{"e", "edit"},
				{"s", "change status"},
				{"n", "add update"},
				{"enter / ctrl+o", "open in browser"},
				{"y", "copy issue link"},
			}},
			global,
		}
	case ViewProjectCycles:
		return []helpSection{
			{"Project Cycles", []helpPair{
				{"j / k / ↑ / ↓", "navigate"},
				{"space / enter", "expand / collapse"},
				{"ctrl+i", "project info"},
				{"y", "copy issues"},
				{"r", "refresh"},
				{"tab", "switch panel"},
			}},
			global,
		}
	}

	// ViewIssueList — depends on focus.
	if st.Focus == FocusSidebar {
		return []helpSection{
			{"Sidebar", []helpPair{
				{"j / k / ↑ / ↓", "navigate"},
				{"ctrl+f / ctrl+b", "page down / up"},
				{"enter / l", "select filter"},
				{"tab", "switch to issues"},
			}},
			{"Anywhere", []helpPair{
				{"c", "create issue"},
				{"ctrl+k", "search issues"},
				{"ctrl+t", "switch team"},
				{"r / ctrl+r", "refresh"},
			}},
			global,
		}
	}

	main := []helpPair{
		{"j / k / ↑ / ↓", "navigate"},
		{"ctrl+f / ctrl+b", "page down / up"},
		{"enter", "open in browser"},
		{"l", "open detail"},
		{"e", "edit"},
		{"s", "change status"},
		{"c", "create issue"},
		{"y", "copy issue link"},
		{"v", "toggle compact"},
		{"p", "toggle priority"},
		{"t", "toggle labels"},
		{"r", "refresh"},
		{"h / tab", "back to sidebar"},
	}
	if st.ActiveFilter == "My Unlabeled Issues" {
		main = append([]helpPair{{"t", "auto-label visible issues"}}, main...)
	}
	return []helpSection{
		{"Issues", main},
		{"Anywhere", []helpPair{
			{"ctrl+k", "search issues"},
			{"ctrl+t", "switch team"},
			{"w", "copy weekly-report prompt"},
		}},
		global,
	}
}

// --- Status ---

func (a *App) layoutStatusModal(gtx layout.Context) layout.Dimensions {
	m, ok := a.State.ModalState.(*StatusModal)
	if !ok {
		return layout.Dimensions{}
	}
	th := a.Th
	fs := th.Fonts.Modal
	return a.modalCardCompact(gtx, gtx.Dp(unit.Dp(340)), gtx.Constraints.Max.Y, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			// Compact header: identifier + title on a single line.
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Baseline}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						l := th.LabelColor(fs, unit.Sp(12), th.AccentDim, m.Issue.Identifier)
						l.Font.Weight = 700
						return l.Layout(gtx)
					}),
					layout.Rigid(hSpace(8)),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						l := th.LabelColor(fs, unit.Sp(12), th.TextDim, m.Issue.Title)
						l.MaxLines = 1
						return l.Layout(gtx)
					}),
					layout.Rigid(hSpace(8)),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						l := th.LabelColor(fs, unit.Sp(10), th.TextMuted, "esc")
						return l.Layout(gtx)
					}),
				)
			}),
			layout.Rigid(rigidSpace(8)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				h := gtx.Dp(unit.Dp(1))
				rect(gtx, image.Rect(0, 0, gtx.Constraints.Max.X, h), th.Border)
				return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, h)}
			}),
			layout.Rigid(rigidSpace(4)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if len(m.States) == 0 {
					return drawDimText(gtx, th, fs, "Loading workflow states…")
				}
				return material.List(th.M, &m.List).Layout(gtx, len(m.States), func(gtx layout.Context, i int) layout.Dimensions {
					st := m.States[i]
					click := &m.Clicks[i]
					if click.Clicked(gtx) {
						m.Idx = i
						a.confirmStatus()
						return layout.Dimensions{}
					}
					selected := i == m.Idx
					current := st.ID == m.Issue.State.ID
					return statusModalRow(gtx, th, fs, click, selected, current, st)
				})
			}),
		)
	})
}

// statusModalRow renders one row of the Change-Status picker: a colored
// state dot, the state name, an optional "current" pill, and a faint type
// hint on the right. Selected/hover states use a tinted rounded background.
func statusModalRow(gtx layout.Context, th *ui.Theme, fs ui.FontStyle, click *widget.Clickable, selected, current bool, st linear.WorkflowState) layout.Dimensions {
	return click.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		macro := op.Record(gtx.Ops)
		dims := layout.Inset{
			Top: unit.Dp(6), Bottom: unit.Dp(6),
			Left: unit.Dp(10), Right: unit.Dp(10),
		}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(statusDot(th, st.Type)),
				layout.Rigid(hSpace(10)),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					col := th.TextDim
					if selected {
						col = th.Text
					}
					l := th.LabelColor(fs, unit.Sp(13), col, st.Name)
					l.Font.Weight = 600
					l.MaxLines = 1
					return l.Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if !current {
						return layout.Dimensions{}
					}
					return layout.Inset{Right: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return currentPill(gtx, th, fs)
					})
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					l := th.LabelColor(fs, unit.Sp(10), th.TextMuted, st.Type)
					l.MaxLines = 1
					return l.Layout(gtx)
				}),
			)
		})
		content := macro.Stop()

		size := image.Pt(gtx.Constraints.Max.X, dims.Size.Y)
		rr := gtx.Dp(unit.Dp(5))
		switch {
		case selected:
			stack := clip.UniformRRect(image.Rect(0, 0, size.X, size.Y), rr).Push(gtx.Ops)
			rect(gtx, image.Rect(0, 0, size.X, size.Y), th.Selected)
			stack.Pop()
		case click.Hovered():
			stack := clip.UniformRRect(image.Rect(0, 0, size.X, size.Y), rr).Push(gtx.Ops)
			rect(gtx, image.Rect(0, 0, size.X, size.Y), th.PanelAlt)
			stack.Pop()
		}
		content.Add(gtx.Ops)
		return layout.Dimensions{Size: size}
	})
}

// currentPill is a small "current" badge shown next to the issue's
// existing state in the change-status modal.
func currentPill(gtx layout.Context, th *ui.Theme, fs ui.FontStyle) layout.Dimensions {
	macro := op.Record(gtx.Ops)
	dims := layout.Inset{
		Top: unit.Dp(1), Bottom: unit.Dp(1),
		Left: unit.Dp(6), Right: unit.Dp(6),
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		l := th.LabelColor(fs, unit.Sp(9), th.TextMuted, "CURRENT")
		l.Font.Weight = 700
		return l.Layout(gtx)
	})
	content := macro.Stop()
	rr := gtx.Dp(unit.Dp(3))
	stack := clip.UniformRRect(image.Rect(0, 0, dims.Size.X, dims.Size.Y), rr).Push(gtx.Ops)
	rect(gtx, image.Rect(0, 0, dims.Size.X, dims.Size.Y), th.PanelAlt)
	stack.Pop()
	content.Add(gtx.Ops)
	return dims
}

// modalCardCompact is like modalCard but with tighter padding for small
// pickers (status, team) where the default 20dp inset feels heavy.
func (a *App) modalCardCompact(gtx layout.Context, maxW, maxH int, w layout.Widget) layout.Dimensions {
	if maxW > gtx.Constraints.Max.X-gtx.Dp(unit.Dp(40)) {
		maxW = gtx.Constraints.Max.X - gtx.Dp(unit.Dp(40))
	}
	if maxH > gtx.Constraints.Max.Y-gtx.Dp(unit.Dp(40)) {
		maxH = gtx.Constraints.Max.Y - gtx.Dp(unit.Dp(40))
	}
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Max.X = maxW
		gtx.Constraints.Max.Y = maxH
		gtx.Constraints.Min = image.Point{}
		return layout.Stack{}.Layout(gtx,
			layout.Expanded(func(gtx layout.Context) layout.Dimensions {
				sz := gtx.Constraints.Min
				r := image.Rect(0, 0, sz.X, sz.Y)
				rrr := gtx.Dp(unit.Dp(8))
				defer clip.UniformRRect(r, rrr).Push(gtx.Ops).Pop()
				rect(gtx, r, a.Th.Panel)
				rectStroke(gtx, r, a.Th.BorderHi)
				return layout.Dimensions{Size: sz}
			}),
			layout.Stacked(func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(12)).Layout(gtx, w)
			}),
		)
	})
}

func (a *App) confirmStatus() {
	m, ok := a.State.ModalState.(*StatusModal)
	if !ok || m.Idx < 0 || m.Idx >= len(m.States) {
		return
	}
	go updateIssueStatus(a.State, m.Issue.ID, m.States[m.Idx].ID)
	a.closeModal()
}

// --- Search ---

func (a *App) layoutSearchModal(gtx layout.Context) layout.Dimensions {
	m, ok := a.State.ModalState.(*SearchModal)
	if !ok {
		return layout.Dimensions{}
	}
	if !m.FocusSet {
		m.FocusSet = true
		gtx.Execute(key.FocusCmd{Tag: &m.Query})
	}
	m.checkIdentifierLookup(a.State)
	return a.modalCard(gtx, gtx.Dp(unit.Dp(640)), gtx.Dp(unit.Dp(560)), func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(modalTitle(a.Th, "Search my issues")),
			layout.Rigid(rigidSpace(8)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Constraints.Max.X
				ed := editorStyle(a.Th, &m.Query, "type to filter…", a.Th.Text, a.Th.Fonts.Modal)
				return widgetBox(gtx, a.Th, ed.Layout)
			}),
			layout.Rigid(rigidSpace(8)),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				issues := m.Filter()
				if len(issues) == 0 {
					msg := "Loading my issues…"
					switch {
					case m.LookupLoading:
						msg = "Looking up " + m.LookupIdentifier + "…"
					case m.LookupNotFound:
						msg = m.LookupIdentifier + " not found"
					}
					return drawDimText(gtx, a.Th, a.Th.Fonts.Modal, msg)
				}
				if m.Selected >= len(issues) {
					m.Selected = 0
				}
				if len(m.Clicks) < len(issues) {
					m.Clicks = make([]widget.Clickable, len(issues))
				}
				children := make([]layout.FlexChild, 0, len(issues))
				for i := range issues {
					i := i
					is := issues[i]
					click := &m.Clicks[i]
					if click.Clicked(gtx) {
						m.Selected = i
						a.confirmSearch()
						return layout.Dimensions{}
					}
					selected := i == m.Selected
					label := is.Identifier + "  " + truncate(is.Title, 80)
					right := is.State.Name
					children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return drawRow(gtx, a.Th, a.Th.Fonts.Modal, click, selected, label, right)
					}))
				}
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
			}),
		)
	})
}

func (a *App) confirmSearch() {
	m, ok := a.State.ModalState.(*SearchModal)
	if !ok {
		return
	}
	issues := m.Filter()
	if m.Selected < 0 || m.Selected >= len(issues) {
		return
	}
	is := issues[m.Selected]
	a.State.Detail = &is
	a.State.View = ViewIssueDetail
	a.closeModal()
}

// --- Team ---

func (a *App) layoutTeamModal(gtx layout.Context) layout.Dimensions {
	m, ok := a.State.ModalState.(*TeamModal)
	if !ok {
		return layout.Dimensions{}
	}
	th := a.Th
	fs := th.Fonts.Modal
	currentID := ""
	if a.State.Team != nil {
		currentID = a.State.Team.ID
	}
	return a.modalCardCompact(gtx, gtx.Dp(unit.Dp(340)), gtx.Dp(unit.Dp(420)), func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			// Compact header: title + faint esc hint.
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Baseline}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						l := th.LabelColor(fs, unit.Sp(12), th.AccentDim, "Select a team")
						l.Font.Weight = 700
						return l.Layout(gtx)
					}),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, 0)}
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						l := th.LabelColor(fs, unit.Sp(10), th.TextMuted, "esc")
						return l.Layout(gtx)
					}),
				)
			}),
			layout.Rigid(rigidSpace(8)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				h := gtx.Dp(unit.Dp(1))
				rect(gtx, image.Rect(0, 0, gtx.Constraints.Max.X, h), th.Border)
				return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, h)}
			}),
			layout.Rigid(rigidSpace(4)),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				teams := m.Teams
				if len(teams) == 0 {
					return drawDimText(gtx, th, fs, "No teams found")
				}
				if m.Selected >= len(teams) {
					m.Selected = 0
				}
				if len(m.Clicks) < len(teams) {
					m.Clicks = make([]widget.Clickable, len(teams))
				}
				children := make([]layout.FlexChild, 0, len(teams))
				for i := range teams {
					i := i
					t := teams[i]
					click := &m.Clicks[i]
					if click.Clicked(gtx) {
						m.Selected = i
						a.confirmTeam()
						return layout.Dimensions{}
					}
					selected := i == m.Selected
					current := t.ID == currentID
					children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return teamModalRow(gtx, th, fs, click, selected, current, t)
					}))
				}
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
			}),
		)
	})
}

// teamModalRow renders one row of the team picker — same visual idiom as
// statusModalRow but with a team-key chip in place of the colored dot.
func teamModalRow(gtx layout.Context, th *ui.Theme, fs ui.FontStyle, click *widget.Clickable, selected, current bool, t linear.Team) layout.Dimensions {
	return click.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		macro := op.Record(gtx.Ops)
		dims := layout.Inset{
			Top: unit.Dp(6), Bottom: unit.Dp(6),
			Left: unit.Dp(10), Right: unit.Dp(10),
		}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return teamKeyChip(gtx, th, fs, t.Key)
				}),
				layout.Rigid(hSpace(10)),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					col := th.TextDim
					if selected {
						col = th.Text
					}
					l := th.LabelColor(fs, unit.Sp(13), col, t.Name)
					l.Font.Weight = 600
					l.MaxLines = 1
					return l.Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if !current {
						return layout.Dimensions{}
					}
					return currentPill(gtx, th, fs)
				}),
			)
		})
		content := macro.Stop()

		size := image.Pt(gtx.Constraints.Max.X, dims.Size.Y)
		rr := gtx.Dp(unit.Dp(5))
		switch {
		case selected:
			stack := clip.UniformRRect(image.Rect(0, 0, size.X, size.Y), rr).Push(gtx.Ops)
			rect(gtx, image.Rect(0, 0, size.X, size.Y), th.Selected)
			stack.Pop()
		case click.Hovered():
			stack := clip.UniformRRect(image.Rect(0, 0, size.X, size.Y), rr).Push(gtx.Ops)
			rect(gtx, image.Rect(0, 0, size.X, size.Y), th.PanelAlt)
			stack.Pop()
		}
		content.Add(gtx.Ops)
		return layout.Dimensions{Size: size}
	})
}

// teamKeyChip is a small accent-colored pill showing the team key (e.g. "ENG")
// — visual stand-in for the colored status dot used in the status modal.
func teamKeyChip(gtx layout.Context, th *ui.Theme, fs ui.FontStyle, key string) layout.Dimensions {
	macro := op.Record(gtx.Ops)
	dims := layout.Inset{
		Top: unit.Dp(2), Bottom: unit.Dp(2),
		Left: unit.Dp(6), Right: unit.Dp(6),
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		l := th.LabelColor(fs, unit.Sp(10), th.AccentDim, key)
		l.Font.Weight = 700
		l.MaxLines = 1
		return l.Layout(gtx)
	})
	content := macro.Stop()
	rr := gtx.Dp(unit.Dp(3))
	stack := clip.UniformRRect(image.Rect(0, 0, dims.Size.X, dims.Size.Y), rr).Push(gtx.Ops)
	rect(gtx, image.Rect(0, 0, dims.Size.X, dims.Size.Y), th.PanelAlt)
	stack.Pop()
	content.Add(gtx.Ops)
	return dims
}

func (a *App) confirmTeam() {
	m, ok := a.State.ModalState.(*TeamModal)
	if !ok {
		return
	}
	teams := m.Teams
	if m.Selected < 0 || m.Selected >= len(teams) {
		return
	}
	t := teams[m.Selected]
	a.State.PostEvent(TeamSelected{Team: t})
	a.closeModal()
}

// --- Project Info (read-only) ---

func (a *App) layoutProjectInfoModal(gtx layout.Context) layout.Dimensions {
	m, ok := a.State.ModalState.(*ProjectInfoModal)
	if !ok {
		return layout.Dimensions{}
	}
	th := a.Th
	fs := th.Fonts.Modal
	if m.Selected >= len(m.Rows) {
		m.Selected = 0
	}
	return a.modalCard(gtx, gtx.Dp(unit.Dp(620)), gtx.Dp(unit.Dp(560)), func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			// Header: project name + faint esc hint.
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Baseline}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						l := th.LabelColor(fs, unit.Sp(16), th.AccentDim, m.Name)
						l.Font.Weight = 700
						l.MaxLines = 1
						return l.Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						hint := "esc"
						if m.Loading {
							hint = "loading… · esc"
						}
						return th.LabelColor(fs, unit.Sp(10), th.TextMuted, hint).Layout(gtx)
					}),
				)
			}),
			layout.Rigid(rigidSpace(8)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				h := gtx.Dp(unit.Dp(1))
				rect(gtx, image.Rect(0, 0, gtx.Constraints.Max.X, h), th.Border)
				return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, h)}
			}),
			layout.Rigid(rigidSpace(6)),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				if len(m.Rows) == 0 {
					return drawDimText(gtx, th, fs, "No project information")
				}
				return material.List(th.M, &m.List).Layout(gtx, len(m.Rows), func(gtx layout.Context, i int) layout.Dimensions {
					return projectInfoRow(gtx, th, fs, m.Rows[i], i == m.Selected)
				})
			}),
			layout.Rigid(rigidSpace(6)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return th.LabelColor(fs, unit.Sp(10), th.TextMuted, "j/k navigate · y copy value · esc close").Layout(gtx)
			}),
		)
	})
}

// projectInfoRow renders one read-only field: an uppercase label above its
// (wrapping) value, with the selected row drawn on a tinted background.
func projectInfoRow(gtx layout.Context, th *ui.Theme, fs ui.FontStyle, row ProjectInfoRow, selected bool) layout.Dimensions {
	macro := op.Record(gtx.Ops)
	dims := layout.Inset{
		Top: unit.Dp(5), Bottom: unit.Dp(5),
		Left: unit.Dp(8), Right: unit.Dp(8),
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				l := th.LabelColor(fs, unit.Sp(10), th.AccentDim, strings.ToUpper(row.Label))
				l.Font.Weight = 700
				l.MaxLines = 1
				return l.Layout(gtx)
			}),
			layout.Rigid(rigidSpace(2)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				col := th.TextDim
				if selected {
					col = th.Text
				}
				return th.LabelColor(fs, unit.Sp(13), col, row.Value).Layout(gtx)
			}),
		)
	})
	content := macro.Stop()

	size := image.Pt(gtx.Constraints.Max.X, dims.Size.Y)
	if selected {
		rr := gtx.Dp(unit.Dp(5))
		stack := clip.UniformRRect(image.Rect(0, 0, size.X, size.Y), rr).Push(gtx.Ops)
		rect(gtx, image.Rect(0, 0, size.X, size.Y), th.Selected)
		stack.Pop()
	}
	content.Add(gtx.Ops)
	return layout.Dimensions{Size: size}
}

// --- Comment ---

func (a *App) layoutCommentModal(gtx layout.Context) layout.Dimensions {
	m, ok := a.State.ModalState.(*CommentModal)
	if !ok {
		return layout.Dimensions{}
	}
	if !m.FocusSet {
		m.FocusSet = true
		gtx.Execute(key.FocusCmd{Tag: &m.Body})
	}
	th := a.Th
	fs := th.Fonts.Modal
	return a.modalCard(gtx, gtx.Dp(unit.Dp(640)), gtx.Dp(unit.Dp(420)), func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Baseline}.Layout(gtx,
					layout.Rigid(modalTitle(th, "Add Update")),
					layout.Rigid(rigidSpace(12)),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						l := th.LabelColor(fs, unit.Sp(12), th.TextMuted, m.Issue.Identifier+" · "+m.Issue.Title)
						l.MaxLines = 1
						return l.Layout(gtx)
					}),
				)
			}),
			layout.Rigid(rigidSpace(16)),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				ed := editorStyle(th, &m.Body, "Write an update…", th.Text, fs)
				gtx.Constraints.Min.X = gtx.Constraints.Max.X
				return widgetBox(gtx, th, ed.Layout)
			}),
			layout.Rigid(rigidSpace(16)),
			layout.Rigid(a.modalButtons(&m.Cancel, &m.Submit, "Cancel", "Post", a.confirmComment, false)),
		)
	})
}

func (a *App) confirmComment() {
	m, ok := a.State.ModalState.(*CommentModal)
	if !ok {
		return
	}
	body := strings.TrimSpace(m.Body.Text())
	if body == "" {
		a.State.StatusText = "Update is empty"
		a.State.StatusKind = StatusWarn
		return
	}
	go createComment(a.State, m.Issue.ID, m.Issue.Identifier, body)
	a.closeModal()
}

// --- Create ---

func (a *App) layoutCreateModal(gtx layout.Context) layout.Dimensions {
	m, ok := a.State.ModalState.(*CreateModal)
	if !ok {
		return layout.Dimensions{}
	}
	th := a.Th
	fs := th.Fonts.CreateIssue
	// Increased size for a more organized, spacious layout.
	return a.modalCard(gtx, gtx.Dp(unit.Dp(960)), gtx.Dp(unit.Dp(720)), func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			// Header
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Baseline}.Layout(gtx,
					layout.Rigid(modalTitle(a.Th, "Create Issue")),
					layout.Rigid(rigidSpace(12)),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						if a.State.Team != nil {
							return a.Th.LabelColor(fs, unit.Sp(12), a.Th.TextMuted, "in "+a.State.Team.Name).Layout(gtx)
						}
						return layout.Dimensions{}
					}),
				)
			}),
			layout.Rigid(rigidSpace(24)),

			// Body: Two-column layout
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					// Left Column: Title and Description
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(fieldLabel(a.Th, fs, "ISSUE TITLE")),
							layout.Rigid(rigidSpace(4)),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								ed := editorStyle(a.Th, &m.Title, "What needs to be done?", a.Th.Text, fs)
								gtx.Constraints.Min.Y = gtx.Dp(unit.Dp(36))
								return widgetBox(gtx, a.Th, ed.Layout)
							}),
							layout.Rigid(rigidSpace(24)),
							layout.Rigid(fieldLabel(a.Th, fs, "DESCRIPTION")),
							layout.Rigid(rigidSpace(4)),
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								ed := editorStyle(a.Th, &m.Description, "Add more details...", a.Th.Text, fs)
								gtx.Constraints.Min.X = gtx.Constraints.Max.X
								return widgetBox(gtx, a.Th, ed.Layout)
							}),
						)
					}),

					layout.Rigid(rigidSpace(32)),

					// Right Column: Properties Sidebar
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Min.X = gtx.Dp(unit.Dp(300))
						gtx.Constraints.Max.X = gtx.Dp(unit.Dp(300))

						if m.Meta == nil {
							return drawDimText(gtx, a.Th, fs, "Loading metadata…")
						}

						// Use a scrollable list for properties
						return material.List(a.Th.M, &m.List).Layout(gtx, 5, func(gtx layout.Context, i int) layout.Dimensions {
							return layout.Inset{Bottom: unit.Dp(20)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								switch i {
								case 0:
									return a.layoutPriorityRowVertical(&m.PrioClicks, &m.Priority)(gtx)
								case 1:
									gtx.Constraints.Max.Y = gtx.Dp(unit.Dp(180))
									gtx.Constraints.Min.Y = gtx.Constraints.Max.Y
									return chooserColumn(gtx, a.Th, fs, "STATUS",
										stateNames(m.Meta.States), m.StateClicks, &m.StateIdx)
								case 2:
									gtx.Constraints.Max.Y = gtx.Dp(unit.Dp(180))
									gtx.Constraints.Min.Y = gtx.Constraints.Max.Y
									return chooserColumn(gtx, a.Th, fs, "ASSIGNEE",
										userNames(m.Meta.Members), m.AssigneeClicks, &m.AssigneeIdx)
								case 3:
									gtx.Constraints.Max.Y = gtx.Dp(unit.Dp(180))
									gtx.Constraints.Min.Y = gtx.Constraints.Max.Y
									return chooserColumn(gtx, a.Th, fs, "PROJECT",
										projectNames(a.State.LeadingProjects), m.ProjectClicks, &m.ProjectIdx)
								case 4:
									gtx.Constraints.Max.Y = gtx.Dp(unit.Dp(180))
									gtx.Constraints.Min.Y = gtx.Constraints.Max.Y
									return chooserColumn(gtx, a.Th, fs, "CYCLE",
										cycleNames(m.Meta.Cycles), m.CycleClicks, &m.CycleIdx)
								default:
									return layout.Dimensions{}
								}
							})
						})
					}),
				)
			}),

			layout.Rigid(rigidSpace(24)),

			// Actions
			layout.Rigid(a.modalButtons(nil, &m.Submit, "Cancel", "Create Issue", a.confirmCreate, m.FocusIdx == 5)),
		)
	})
}

func (a *App) confirmCreate() {
	m, ok := a.State.ModalState.(*CreateModal)
	if !ok {
		return
	}
	in, valid := m.Build(a.State)
	if !valid {
		a.State.StatusText = "Title is required"
		a.State.StatusKind = StatusWarn
		return
	}
	go createIssue(a.State, in)
	a.closeModal()
}

// --- Modal helpers ---

func modalTitle(th *ui.Theme, s string) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		l := th.LabelColor(th.Fonts.Modal, unit.Sp(18), th.AccentDim, s)
		l.Font.Weight = 700
		return l.Layout(gtx)
	}
}

func fieldLabel(th *ui.Theme, fs ui.FontStyle, s string) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		l := th.LabelColor(fs, unit.Sp(10), th.TextMuted, s)
		l.Font.Weight = 700
		return l.Layout(gtx)
	}
}

func (a *App) modalButtons(cancel, ok *widget.Clickable, cancelTxt, okTxt string, onOK func(), okFocused bool) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		if cancel != nil && cancel.Clicked(gtx) {
			a.closeModal()
		}
		if ok != nil && ok.Clicked(gtx) && onOK != nil {
			onOK()
		}
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, 0)}
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if cancel == nil {
					return layout.Dimensions{}
				}
				b := material.Button(a.Th.M, cancel, cancelTxt)
				b.Background = color.NRGBA{A: 0}
				b.Color = a.Th.TextDim
				return b.Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if cancel == nil {
					return layout.Dimensions{}
				}
				return rigidSpace(12)(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if ok == nil {
					return layout.Dimensions{}
				}
				b := material.Button(a.Th.M, ok, okTxt)
				if okFocused {
					b.Background = a.Th.Selected // visual feedback for focus
				} else {
					b.Background = a.Th.Accent
				}
				return b.Layout(gtx)
			}),
		)
	}
}

func (a *App) layoutPriorityRowVertical(clicks *[5]widget.Clickable, priority *int) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		labels := []string{"None", "Urgent", "High", "Medium", "Low"}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(fieldLabel(a.Th, a.Th.Fonts.Modal, "PRIORITY")),
			layout.Rigid(rigidSpace(6)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				children := make([]layout.FlexChild, 0, 5)
				for i := 0; i < 5; i++ {
					i := i
					click := &clicks[i]
					if click.Clicked(gtx) {
						*priority = i
					}
					sel := *priority == i
					children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return drawChip(gtx, a.Th, click, sel, labels[i])
					}))
					children = append(children, layout.Rigid(rigidSpace(4)))
				}
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, children...)
			}),
		)
	}
}

func (a *App) layoutPriorityRow(clicks *[5]widget.Clickable, priority *int) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		labels := []string{"None", "Urgent", "High", "Medium", "Low"}
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(fieldLabel(a.Th, a.Th.Fonts.Modal, "Priority")),
			layout.Rigid(rigidSpace(8)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				children := make([]layout.FlexChild, 0, 5)
				for i := 0; i < 5; i++ {
					i := i
					click := &clicks[i]
					if click.Clicked(gtx) {
						*priority = i
					}
					sel := *priority == i
					children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return drawChip(gtx, a.Th, click, sel, labels[i])
					}))
					children = append(children, layout.Rigid(rigidSpace(4)))
				}
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, children...)
			}),
		)
	}
}

func drawChip(gtx layout.Context, th *ui.Theme, click *widget.Clickable, selected bool, label string) layout.Dimensions {
	return click.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Stack{}.Layout(gtx,
			layout.Expanded(func(gtx layout.Context) layout.Dimensions {
				r := image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)
				bg := th.PanelAlt
				if selected {
					bg = th.Selected
				} else if click.Hovered() {
					bg = th.Border
				}
				rr := gtx.Dp(unit.Dp(4))
				defer clip.UniformRRect(r, rr).Push(gtx.Ops).Pop()
				rect(gtx, r, bg)
				return layout.Dimensions{Size: gtx.Constraints.Max}
			}),
			layout.Stacked(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{
					Top: unit.Dp(4), Bottom: unit.Dp(4),
					Left: unit.Dp(8), Right: unit.Dp(8),
				}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					l := material.Label(th.M, unit.Sp(11), label)
					l.Color = th.Text
					return l.Layout(gtx)
				})
			}),
		)
	})
}

func chooserColumn(gtx layout.Context, th *ui.Theme, fs ui.FontStyle, label string, names []string, clicks []widget.Clickable, idx *int) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(fieldLabel(th, fs, label)),
		layout.Rigid(rigidSpace(2)),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			r := image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)
			rectStroke(gtx, r, th.Border)
			return layout.Inset{
				Top: unit.Dp(4), Bottom: unit.Dp(4),
				Left: unit.Dp(4), Right: unit.Dp(4),
			}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				if label == "CYCLE" {
					log.Printf("[UI] chooserColumn rendering %d cycles", len(names))
				}
				children := make([]layout.FlexChild, 0, len(names)+1)
				// "None" entry to clear selection.
				clearClick := &widget.Clickable{}
				_ = clearClick // we don't persist the clear click; it lives one frame
				for i := range names {
					i := i
					n := names[i]
					if i >= len(clicks) {
						break
					}
					c := &clicks[i]
					if c.Clicked(gtx) {
						if *idx == i {
							*idx = -1
						} else {
							*idx = i
						}
					}
					selected := *idx == i
					children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return drawRow(gtx, th, fs, c, selected, truncate(n, 80), "")
					}))
				}
				if len(children) == 0 {
					return drawDimText(gtx, th, fs, "—")
				}
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
			})
		}),
	)
}

func stateNames(s []linear.WorkflowState) []string {
	out := make([]string, len(s))
	for i, st := range s {
		out[i] = st.Name + " · " + st.Type
	}
	return out
}

func userNames(u []linear.User) []string {
	out := make([]string, len(u))
	for i, x := range u {
		out[i] = x.Name
	}
	return out
}

func projectNames(p []linear.Project) []string {
	out := make([]string, 0, len(p))
	for _, x := range p {
		out = append(out, cleanProjectName(x.Name))
	}
	return out
}

func cycleNames(c []linear.Cycle) []string {
	out := make([]string, 0, len(c))
	for _, x := range c {
		name := x.Name
		if name == "" {
			name = "Cycle " + strconv.Itoa(x.Number)
		}
		out = append(out, name)
	}
	return out
}

// --- Settings ---

// --- Modal key handling ---

func (a *App) handleModalKey(ke key.Event) {
	switch ke.Name {
	case key.NameEscape:
		a.closeModal()
		return
	case "[":
		if ke.Modifiers.Contain(key.ModCtrl) && (a.State.Modal == ModalStatus || a.State.Modal == ModalProjectInfo) {
			a.closeModal()
			return
		}
	case "H":
		if ke.Modifiers == 0 && a.State.Modal == ModalProjectInfo {
			a.closeModal()
			return
		}
	case key.NameReturn:
		switch a.State.Modal {
		case ModalCreate:
			a.confirmCreate()
		case ModalStatus:
			a.confirmStatus()
		case ModalSearch:
			a.confirmSearch()
		case ModalTeam:
			a.confirmTeam()
		case ModalHelp:
			a.closeModal()
		case ModalComment:
			if ke.Modifiers.Contain(key.ModCtrl) {
				a.confirmComment()
			}
		}
		return
	case "?":
		if a.State.Modal == ModalHelp {
			a.closeModal()
			return
		}
	case "S":
		if ke.Modifiers == 0 && a.State.Modal == ModalStatus {
			a.closeModal()
			return
		}
	case "T":
		if ke.Modifiers == key.ModCtrl && a.State.Modal == ModalTeam {
			a.closeModal()
			return
		}
	case "I":
		if ke.Modifiers == key.ModCtrl && a.State.Modal == ModalProjectInfo {
			a.closeModal()
			return
		}
	case "Y":
		if a.State.Modal == ModalProjectInfo {
			if m, ok := a.State.ModalState.(*ProjectInfoModal); ok {
				go CopyProjectInfoRow(a.State, m)
			}
			return
		}
	case "K", key.NameUpArrow:
		if ke.Modifiers == key.ModCtrl && ke.Name == "K" {
			if a.State.Modal == ModalSearch {
				a.closeModal()
			}
			return
		}
		a.modalMove(-1)
	case "F":
		if ke.Modifiers == key.ModCtrl {
			a.modalMove(10)
		}
	case "B":
		if ke.Modifiers == key.ModCtrl {
			a.modalMove(-10)
		}
	case key.NameDownArrow, "J":
		a.modalMove(1)
	case "N":
		if ke.Modifiers == key.ModCtrl {
			a.modalMove(1)
		}
	case "P":
		if ke.Modifiers == key.ModCtrl {
			a.modalMove(-1)
		}
	}
}

func (a *App) modalMove(d int) {
	switch m := a.State.ModalState.(type) {
	case *StatusModal:
		if len(m.States) == 0 {
			return
		}
		m.Idx = clamp(m.Idx+d, 0, len(m.States)-1)
		scrollListIntoView(&m.List, m.Idx)
	case *SearchModal:
		issues := m.Filter()
		if len(issues) == 0 {
			return
		}
		m.Selected = clamp(m.Selected+d, 0, len(issues)-1)
	case *TeamModal:
		teams := m.Teams
		if len(teams) == 0 {
			return
		}
		m.Selected = clamp(m.Selected+d, 0, len(teams)-1)
	case *ProjectInfoModal:
		if len(m.Rows) == 0 {
			return
		}
		m.Selected = clamp(m.Selected+d, 0, len(m.Rows)-1)
		scrollListIntoView(&m.List, m.Selected)
	}
}

// scrollListIntoView scrolls l only when idx is outside the visible range,
// so keyboard navigation within the visible window doesn't scroll prematurely.
func scrollListIntoView(l *widget.List, idx int) {
	pos := &l.Position
	if pos.Count == 0 {
		l.ScrollTo(idx)
		return
	}
	last := pos.First + pos.Count - 1
	switch {
	case idx < pos.First:
		l.ScrollBy(float32(idx - pos.First))
	case idx > last:
		l.ScrollBy(float32(idx - last))
	case idx == pos.First && pos.Offset > 0:
		l.ScrollBy(-1)
	case idx == last && pos.OffsetLast < 0:
		l.ScrollBy(1)
	}
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
