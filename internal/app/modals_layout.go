package app

import (
	"image"
	"image/color"
	"log"
	"strconv"

	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
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
	case ModalEdit:
		a.layoutEditModal(gtx)
	case ModalStatus:
		a.layoutStatusModal(gtx)
	case ModalSearch:
		a.layoutSearchModal(gtx)
	case ModalTeam:
		a.layoutTeamModal(gtx)
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
	type pair struct{ k, d string }
	type sec struct {
		title string
		keys  []pair
	}
	sections := []sec{
		{"Global", []pair{
			{"q / ctrl+c", "quit"},
			{"tab / shift+tab", "switch panel"},
			{"c", "create issue"},
			{"ctrl+k", "search issues"},
			{",", "settings (fonts)"},
			{"v", "toggle compact"},
			{"?", "toggle help"},
		}},
		{"Sidebar", []pair{
			{"j / k", "navigate"},
			{"enter / l", "select filter or focus issues"},
		}},
		{"Issue list", []pair{
			{"j / k", "navigate"},
			{"enter", "open in browser"},
			{"l", "open detail"},
			{"e", "edit"},
			{"s", "change status"},
			{"r", "refresh"},
			{"t", "toggle tags or auto-label"},
		}},
		{"Issue detail", []pair{
			{"esc / h", "back"},
			{"e", "edit"},
			{"s", "change status"},
		}},
		{"Modals", []pair{
			{"esc", "cancel"},
			{"enter", "submit"},
		}},
	}

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
			return a.Th.LabelColor(fs, unit.Sp(11), a.Th.TextMuted, "Press esc or ? to close").Layout(gtx)
		}))
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	})
}

// --- Status ---

func (a *App) layoutStatusModal(gtx layout.Context) layout.Dimensions {
	m, ok := a.State.ModalState.(*StatusModal)
	if !ok {
		return layout.Dimensions{}
	}
	th := a.Th
	fs := th.Fonts.Modal
	return a.modalCardCompact(gtx, gtx.Dp(unit.Dp(340)), gtx.Dp(unit.Dp(420)), func(gtx layout.Context) layout.Dimensions {
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
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				if len(m.States) == 0 {
					return drawDimText(gtx, th, fs, "Loading workflow states…")
				}
				children := make([]layout.FlexChild, 0, len(m.States))
				for i := range m.States {
					i := i
					st := m.States[i]
					click := &m.Clicks[i]
					if click.Clicked(gtx) {
						m.Idx = i
						a.confirmStatus()
						return layout.Dimensions{}
					}
					selected := i == m.Idx
					current := st.ID == m.Issue.State.ID
					children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return statusModalRow(gtx, th, fs, click, selected, current, st)
					}))
				}
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
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
				r := image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)
				rrr := gtx.Dp(unit.Dp(8))
				defer clip.UniformRRect(r, rrr).Push(gtx.Ops).Pop()
				rect(gtx, r, a.Th.Panel)
				rectStroke(gtx, r, a.Th.BorderHi)
				return layout.Dimensions{Size: gtx.Constraints.Max}
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
	return a.modalCard(gtx, gtx.Dp(unit.Dp(640)), gtx.Dp(unit.Dp(560)), func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(modalTitle(a.Th, "Search my issues")),
			layout.Rigid(rigidSpace(8)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				ed := editorStyle(a.Th, &m.Query, "type to filter…", a.Th.Text, a.Th.Fonts.Modal)
				return widgetBox(gtx, a.Th, ed.Layout)
			}),
			layout.Rigid(rigidSpace(8)),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				issues := m.Filter()
				if len(issues) == 0 {
					return drawDimText(gtx, a.Th, a.Th.Fonts.Modal, "Loading my issues…")
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
			layout.Rigid(rigidSpace(8)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if m.Cancel.Clicked(gtx) {
					a.closeModal()
				}
				return material.Button(a.Th.M, &m.Cancel, "Close").Layout(gtx)
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

// --- Create ---

func (a *App) layoutCreateModal(gtx layout.Context) layout.Dimensions {
	m, ok := a.State.ModalState.(*CreateModal)
	if !ok {
		return layout.Dimensions{}
	}
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
							return a.Th.LabelColor(a.Th.Fonts.Modal, unit.Sp(12), a.Th.TextMuted, "in "+a.State.Team.Name).Layout(gtx)
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
							layout.Rigid(fieldLabel(a.Th, a.Th.Fonts.Modal, "ISSUE TITLE")),
							layout.Rigid(rigidSpace(4)),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								ed := editorStyle(a.Th, &m.Title, "What needs to be done?", a.Th.Text, a.Th.Fonts.Modal)
								gtx.Constraints.Min.Y = gtx.Dp(unit.Dp(36))
								return widgetBox(gtx, a.Th, ed.Layout)
							}),
							layout.Rigid(rigidSpace(24)),
							layout.Rigid(fieldLabel(a.Th, a.Th.Fonts.Modal, "DESCRIPTION")),
							layout.Rigid(rigidSpace(4)),
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								ed := editorStyle(a.Th, &m.Description, "Add more details...", a.Th.Text, a.Th.Fonts.Modal)
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
							return drawDimText(gtx, a.Th, a.Th.Fonts.Modal, "Loading metadata…")
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
									return chooserColumn(gtx, a.Th, a.Th.Fonts.Modal, "STATUS",
										stateNames(m.Meta.States), m.StateClicks, &m.StateIdx)
								case 2:
									gtx.Constraints.Max.Y = gtx.Dp(unit.Dp(180))
									gtx.Constraints.Min.Y = gtx.Constraints.Max.Y
									return chooserColumn(gtx, a.Th, a.Th.Fonts.Modal, "ASSIGNEE",
										userNames(m.Meta.Members), m.AssigneeClicks, &m.AssigneeIdx)
								case 3:
									gtx.Constraints.Max.Y = gtx.Dp(unit.Dp(180))
									gtx.Constraints.Min.Y = gtx.Constraints.Max.Y
									return chooserColumn(gtx, a.Th, a.Th.Fonts.Modal, "PROJECT",
										projectNames(a.State.LeadingProjects), m.ProjectClicks, &m.ProjectIdx)
								case 4:
									gtx.Constraints.Max.Y = gtx.Dp(unit.Dp(180))
									gtx.Constraints.Min.Y = gtx.Constraints.Max.Y
									return chooserColumn(gtx, a.Th, a.Th.Fonts.Modal, "CYCLE",
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

// --- Edit ---

func (a *App) layoutEditModal(gtx layout.Context) layout.Dimensions {
	m, ok := a.State.ModalState.(*EditModal)
	if !ok {
		return layout.Dimensions{}
	}
	return a.modalCard(gtx, gtx.Dp(unit.Dp(960)), gtx.Dp(unit.Dp(720)), func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			// Header
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Baseline}.Layout(gtx,
					layout.Rigid(modalTitle(a.Th, "Edit "+m.Issue.Identifier)),
					layout.Rigid(rigidSpace(12)),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return a.Th.LabelColor(a.Th.Fonts.Modal, unit.Sp(12), a.Th.TextMuted, m.Issue.Title).Layout(gtx)
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
							layout.Rigid(fieldLabel(a.Th, a.Th.Fonts.Modal, "ISSUE TITLE")),
							layout.Rigid(rigidSpace(4)),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								ed := editorStyle(a.Th, &m.Title, "Issue title", a.Th.Text, a.Th.Fonts.Modal)
								gtx.Constraints.Min.Y = gtx.Dp(unit.Dp(36))
								return widgetBox(gtx, a.Th, ed.Layout)
							}),
							layout.Rigid(rigidSpace(24)),
							layout.Rigid(fieldLabel(a.Th, a.Th.Fonts.Modal, "DESCRIPTION")),
							layout.Rigid(rigidSpace(4)),
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								ed := editorStyle(a.Th, &m.Description, "", a.Th.Text, a.Th.Fonts.Modal)
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
							return drawDimText(gtx, a.Th, a.Th.Fonts.Modal, "Loading metadata…")
						}

						// Use a scrollable list for properties
						return material.List(a.Th.M, &m.List).Layout(gtx, 4, func(gtx layout.Context, i int) layout.Dimensions {
							return layout.Inset{Bottom: unit.Dp(20)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								switch i {
								case 0:
									return a.layoutPriorityRowVertical(&m.PrioClicks, &m.Priority)(gtx)
								case 1:
									gtx.Constraints.Max.Y = gtx.Dp(unit.Dp(180))
									gtx.Constraints.Min.Y = gtx.Constraints.Max.Y
									return chooserColumn(gtx, a.Th, a.Th.Fonts.Modal, "STATUS",
										stateNames(m.Meta.States), m.StateClicks, &m.StateIdx)
								case 2:
									gtx.Constraints.Max.Y = gtx.Dp(unit.Dp(180))
									gtx.Constraints.Min.Y = gtx.Constraints.Max.Y
									return chooserColumn(gtx, a.Th, a.Th.Fonts.Modal, "ASSIGNEE",
										userNames(m.Meta.Members), m.AssigneeClicks, &m.AssigneeIdx)
								case 3:
									gtx.Constraints.Max.Y = gtx.Dp(unit.Dp(180))
									gtx.Constraints.Min.Y = gtx.Constraints.Max.Y
									return chooserColumn(gtx, a.Th, a.Th.Fonts.Modal, "PROJECT",
										projectNames(a.State.LeadingProjects), m.ProjectClicks, &m.ProjectIdx)
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
			layout.Rigid(a.modalButtons(&m.Cancel, &m.Submit, "Cancel", "Save Changes", a.confirmEdit, false)),
		)
	})
}

func (a *App) confirmEdit() {
	m, ok := a.State.ModalState.(*EditModal)
	if !ok {
		return
	}
	id, in, valid := m.Build(a.State)
	if !valid {
		a.State.StatusText = "Title is required"
		a.State.StatusKind = StatusWarn
		return
	}
	go editIssue(a.State, id, in)
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

func (a *App) layoutSettingsSide(gtx layout.Context, r image.Rectangle) {
	m := a.State.Settings
	if m == nil {
		return
	}
	defer op.Offset(r.Min).Push(gtx.Ops).Pop()
	gtx.Constraints = layout.Exact(r.Size())

	// Background + 1dp left border.
	rect(gtx, image.Rect(0, 0, r.Dx(), r.Dy()), a.Th.Panel)
	rect(gtx, image.Rect(0, 0, gtx.Dp(unit.Dp(1)), r.Dy()), a.Th.Border)

	a.applySettingsClicks(gtx, m)
	if m.Reset.Clicked(gtx) {
		a.resetAllFonts(m)
	}
	if m.Close.Clicked(gtx) {
		a.State.ShowSettings = false
	}

	layout.Inset{Top: unit.Dp(18), Bottom: unit.Dp(14), Left: unit.Dp(18), Right: unit.Dp(18)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			// Header
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Baseline}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						l := a.Th.LabelColor(a.Th.Fonts.Modal, unit.Sp(20), a.Th.Text, "Fonts")
						l.Font.Weight = 700
						return l.Layout(gtx)
					}),
					layout.Rigid(rigidSpace(8)),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return a.Th.LabelColor(a.Th.Fonts.Modal, unit.Sp(11), a.Th.TextMuted, "auto-saves · ',' to toggle").Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.miniBtn(gtx, &m.Reset, "reset all", false)
					}),
				)
			}),
			layout.Rigid(rigidSpace(12)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				h := gtx.Dp(unit.Dp(1))
				rect(gtx, image.Rect(0, 0, gtx.Constraints.Max.X, h), a.Th.Border)
				return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, h)}
			}),
			layout.Rigid(rigidSpace(8)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.layoutDefaultStatusRow(gtx, m)
			}),
			layout.Rigid(rigidSpace(8)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				h := gtx.Dp(unit.Dp(1))
				rect(gtx, image.Rect(0, 0, gtx.Constraints.Max.X, h), a.Th.Border)
				return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, h)}
			}),
			layout.Rigid(rigidSpace(8)),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return material.List(a.Th.M, &m.List).Layout(gtx, len(m.Rows), func(gtx layout.Context, i int) layout.Dimensions {
					return a.layoutSettingsRow(gtx, m.Rows[i], i == len(m.Rows)-1)
				})
			}),
			layout.Rigid(rigidSpace(12)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, 0)}
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.miniBtn(gtx, &m.Close, "close", true)
					}),
				)
			}),
		)
	})
}

func (a *App) layoutSettingsRow(gtx layout.Context, r *SettingsRow, last bool) layout.Dimensions {
	customized := r.Target.Face != "" || r.Target.Size > 0
	return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			// Section title row: accent indicator + label + per-row reset.
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						sz := gtx.Dp(unit.Dp(6))
						col := a.Th.Border
						if customized {
							col = a.Th.AccentDim
						}
						rr := clip.UniformRRect(image.Rect(0, 0, sz, sz), sz/2)
						paint.FillShape(gtx.Ops, col, rr.Op(gtx.Ops))
						return layout.Dimensions{Size: image.Pt(sz, sz)}
					}),
					layout.Rigid(rigidSpace(8)),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						l := a.Th.LabelColor(a.Th.Fonts.Modal, unit.Sp(13), a.Th.Text, r.Label)
						l.Font.Weight = 600
						return l.Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if !customized {
							return layout.Dimensions{}
						}
						return a.miniBtn(gtx, &r.Reset, "reset", false)
					}),
				)
			}),
			layout.Rigid(rigidSpace(8)),
			// Controls row: face cycler (flex-grow) + size stepper.
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return a.faceCycler(gtx, r)
					}),
					layout.Rigid(rigidSpace(8)),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.sizeStepper(gtx, r)
					}),
				)
			}),
			layout.Rigid(rigidSpace(8)),
			// Preview using the row's live font style.
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.layoutPreview(gtx, r)
			}),
			layout.Rigid(rigidSpace(10)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if last {
					return layout.Dimensions{}
				}
				h := gtx.Dp(unit.Dp(1))
				rect(gtx, image.Rect(0, 0, gtx.Constraints.Max.X, h), a.Th.Border)
				return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, h)}
			}),
		)
	})
}

// faceCycler is `‹  Face Name  ›` — a single rounded surface containing
// the chevrons and the current face name, expanded to fill its flex slot.
func (a *App) faceCycler(gtx layout.Context, r *SettingsRow) layout.Dimensions {
	face := r.Target.Face
	faceCol := a.Th.Text
	if face == "" {
		face = "Default"
		faceCol = a.Th.TextDim
	}
	return a.surfaceRow(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.chevBtn(gtx, &r.PrevF, "‹")
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					l := a.Th.LabelColor(a.Th.Fonts.Modal, unit.Sp(12), faceCol, face)
					l.MaxLines = 1
					return l.Layout(gtx)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.chevBtn(gtx, &r.NextF, "›")
			}),
		)
	})
}

// sizeStepper is `−  13  +`, a fixed-width rounded surface.
func (a *App) sizeStepper(gtx layout.Context, r *SettingsRow) layout.Dimensions {
	display := "—"
	col := a.Th.TextDim
	if r.Target.Size > 0 {
		display = strconv.Itoa(int(r.Target.Size))
		col = a.Th.Text
	}
	gtx.Constraints.Min.X = gtx.Dp(unit.Dp(120))
	gtx.Constraints.Max.X = gtx.Dp(unit.Dp(120))
	return a.surfaceRow(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.chevBtn(gtx, &r.Smaller, "−")
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					l := a.Th.LabelColor(a.Th.Fonts.Modal, unit.Sp(12), col, display)
					l.Font.Weight = 600
					l.MaxLines = 1
					return l.Layout(gtx)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.chevBtn(gtx, &r.Bigger, "+")
			}),
		)
	})
}

// surfaceRow paints a rounded panel-alt background sized to its content
// height (capped to a uniform pill height) and lets `inner` draw on top.
func (a *App) surfaceRow(gtx layout.Context, inner layout.Widget) layout.Dimensions {
	macro := op.Record(gtx.Ops)
	dims := layout.Inset{Top: unit.Dp(2), Bottom: unit.Dp(2)}.Layout(gtx, inner)
	content := macro.Stop()

	w := dims.Size.X
	if gtx.Constraints.Min.X > w {
		w = gtx.Constraints.Min.X
	}
	h := dims.Size.Y
	if min := gtx.Dp(unit.Dp(28)); h < min {
		h = min
	}
	rr := clip.UniformRRect(image.Rect(0, 0, w, h), gtx.Dp(unit.Dp(6)))
	paint.FillShape(gtx.Ops, a.Th.PanelAlt, rr.Op(gtx.Ops))
	content.Add(gtx.Ops)
	return layout.Dimensions{Size: image.Pt(w, h)}
}

// chevBtn is a 28dp square clickable used as < / > / − / + within surfaceRow.
func (a *App) chevBtn(gtx layout.Context, c *widget.Clickable, label string) layout.Dimensions {
	side := gtx.Dp(unit.Dp(28))
	return c.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints = layout.Exact(image.Pt(side, side))
		if c.Hovered() {
			rr := clip.UniformRRect(image.Rect(0, 0, side, side), gtx.Dp(unit.Dp(4)))
			paint.FillShape(gtx.Ops, a.Th.Border, rr.Op(gtx.Ops))
		}
		col := a.Th.TextDim
		if c.Hovered() {
			col = a.Th.Text
		}
		return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			l := a.Th.LabelColor(a.Th.Fonts.Modal, unit.Sp(14), col, label)
			l.Font.Weight = 600
			return l.Layout(gtx)
		})
	})
}

// miniBtn is a small text button with a rounded background; faster and
// lighter than material.Button for the dense settings panel. accent=true
// uses Linear's purple as the surface (used for the primary "close" action).
func (a *App) miniBtn(gtx layout.Context, c *widget.Clickable, label string, accent bool) layout.Dimensions {
	return c.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		macro := op.Record(gtx.Ops)
		fg := a.Th.TextDim
		if accent {
			fg = a.Th.Text
		}
		if c.Hovered() {
			fg = a.Th.Text
		}
		dims := layout.Inset{Top: unit.Dp(5), Bottom: unit.Dp(5), Left: unit.Dp(10), Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			l := a.Th.LabelColor(a.Th.Fonts.Modal, unit.Sp(11), fg, label)
			l.Font.Weight = 600
			l.MaxLines = 1
			return l.Layout(gtx)
		})
		content := macro.Stop()

		bg := a.Th.PanelAlt
		switch {
		case accent && c.Hovered():
			bg = a.Th.AccentDim
		case accent:
			bg = a.Th.Accent
		case c.Hovered():
			bg = a.Th.Border
		}
		rr := clip.UniformRRect(image.Rect(0, 0, dims.Size.X, dims.Size.Y), gtx.Dp(unit.Dp(5)))
		paint.FillShape(gtx.Ops, bg, rr.Op(gtx.Ops))
		content.Add(gtx.Ops)
		return dims
	})
}

func (a *App) layoutPreview(gtx layout.Context, r *SettingsRow) layout.Dimensions {
	sample := "The quick brown fox jumps over the lazy dog · 0123"
	if r.Mono {
		sample = "for i := 0; i < n; i++ { fmt.Println(i) }"
	}
	l := a.Th.LabelColor(*r.Target, unit.Sp(12), a.Th.TextDim, sample)
	l.MaxLines = 1
	return l.Layout(gtx)
}

// layoutDefaultStatusRow renders the "Default Create Status" cycler:
// label on the left, a `‹ Name ›` cycler on the right.
func (a *App) layoutDefaultStatusRow(gtx layout.Context, m *SettingsModal) layout.Dimensions {
	if a.State.Saved == nil {
		return layout.Dimensions{}
	}
	cur := a.State.Saved.DefaultCreateStatusType
	if cur == "" {
		cur = "started"
	}
	return layout.Inset{Top: unit.Dp(4), Bottom: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						sz := gtx.Dp(unit.Dp(6))
						rr := clip.UniformRRect(image.Rect(0, 0, sz, sz), sz/2)
						paint.FillShape(gtx.Ops, a.Th.Border, rr.Op(gtx.Ops))
						return layout.Dimensions{Size: image.Pt(sz, sz)}
					}),
					layout.Rigid(rigidSpace(8)),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						l := a.Th.LabelColor(a.Th.Fonts.Modal, unit.Sp(13), a.Th.Text, "Default Create Status")
						l.Font.Weight = 600
						return l.Layout(gtx)
					}),
				)
			}),
			layout.Rigid(rigidSpace(8)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.surfaceRow(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.chevBtn(gtx, &m.StatusPrev, "‹")
						}),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								l := a.Th.LabelColor(a.Th.Fonts.Modal, unit.Sp(12), a.Th.Text, createStatusLabel(cur))
								l.MaxLines = 1
								return l.Layout(gtx)
							})
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.chevBtn(gtx, &m.StatusNext, "›")
						}),
					)
				})
			}),
		)
	})
}

func (a *App) cycleDefaultStatus(delta int) {
	if a.State.Saved == nil {
		return
	}
	cur := a.State.Saved.DefaultCreateStatusType
	if cur == "" {
		cur = "started"
	}
	idx := 0
	for i, t := range CreateStatusTypes {
		if t == cur {
			idx = i
			break
		}
	}
	idx = (idx + delta + len(CreateStatusTypes)) % len(CreateStatusTypes)
	a.State.Saved.DefaultCreateStatusType = CreateStatusTypes[idx]
	a.saveState()
}

func (a *App) applySettingsClicks(gtx layout.Context, m *SettingsModal) {
	if m.StatusPrev.Clicked(gtx) {
		a.cycleDefaultStatus(-1)
	}
	if m.StatusNext.Clicked(gtx) {
		a.cycleDefaultStatus(1)
	}
	dirty := false
	for _, r := range m.Rows {
		if r.PrevF.Clicked(gtx) {
			a.cycleFace(r, -1)
			dirty = true
		}
		if r.NextF.Clicked(gtx) {
			a.cycleFace(r, 1)
			dirty = true
		}
		if r.Smaller.Clicked(gtx) {
			a.bumpSize(r, -1)
			dirty = true
		}
		if r.Bigger.Clicked(gtx) {
			a.bumpSize(r, 1)
			dirty = true
		}
		if r.Reset.Clicked(gtx) {
			r.Target.Face = ""
			r.Target.Size = 0
			dirty = true
		}
	}
	if dirty {
		a.saveState()
	}
}

func (a *App) facesFor(r *SettingsRow) []string {
	if r.Mono {
		return a.Th.MonoFaces
	}
	return a.Th.Faces
}

func (a *App) cycleFace(r *SettingsRow, delta int) {
	options := append([]string{""}, a.facesFor(r)...)
	idx := 0
	for i, f := range options {
		if f == r.Target.Face {
			idx = i
			break
		}
	}
	idx = (idx + delta + len(options)) % len(options)
	r.Target.Face = options[idx]
}

func (a *App) bumpSize(r *SettingsRow, delta float32) {
	cur := r.Target.Size
	if cur == 0 {
		cur = 13
	}
	cur += delta
	if cur < 8 {
		cur = 8
	}
	if cur > 32 {
		cur = 32
	}
	r.Target.Size = cur
}

func (a *App) resetAllFonts(m *SettingsModal) {
	for _, r := range m.Rows {
		r.Target.Face = ""
		r.Target.Size = 0
	}
	a.saveState()
}

// --- Modal key handling ---

func (a *App) handleModalKey(ke key.Event) {
	switch ke.Name {
	case key.NameEscape:
		a.closeModal()
		return
	case "[":
		if ke.Modifiers.Contain(key.ModCtrl) && a.State.Modal == ModalStatus {
			a.closeModal()
			return
		}
	case key.NameReturn:
		switch a.State.Modal {
		case ModalCreate:
			a.confirmCreate()
		case ModalEdit:
			a.confirmEdit()
		case ModalStatus:
			a.confirmStatus()
		case ModalSearch:
			a.confirmSearch()
		case ModalTeam:
			a.confirmTeam()
		case ModalHelp:
			a.closeModal()
		}
		return
	case "?":
		if a.State.Modal == ModalHelp {
			a.closeModal()
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
	case key.NameUpArrow, "K":
		a.modalMove(-1)
	}
}

func (a *App) modalMove(d int) {
	switch m := a.State.ModalState.(type) {
	case *StatusModal:
		if len(m.States) == 0 {
			return
		}
		m.Idx = clamp(m.Idx+d, 0, len(m.States)-1)
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
