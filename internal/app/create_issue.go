package app

import (
	"fmt"
	"image"
	"log"

	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/denislee/wllinear/internal/ui"
)

// Two-column table geometry for the create issue form.
const (
	formLabelColDp = 96
	formColGapDp   = 8
	formRowGapDp   = 14
)

func (a *App) layoutCreateIssue(gtx layout.Context) layout.Dimensions {
	m := a.State.Create
	if m == nil {
		return layout.Dimensions{}
	}
	th := a.Th
	fs := th.Fonts.CreateIssue

	if !m.FocusSet {
		m.FocusSet = true
		m.FocusIdx = 0
		m.FocusReq = true
	}

	if m.FocusReq {
		m.FocusReq = false
		if m.FocusIdx == 0 {
			gtx.Execute(key.FocusCmd{Tag: &m.Title})
		} else if m.FocusIdx == 1 {
			gtx.Execute(key.FocusCmd{Tag: &m.Description})
		} else {
			gtx.Execute(key.FocusCmd{Tag: nil})
		}
	}

	if gtx.Focused(&m.Title) {
		m.FocusIdx = 0
	} else if gtx.Focused(&m.Description) {
		m.FocusIdx = 1
	}

	// Scroll the focused form row into view when focus has changed.
	// Item layout: 0=header, 1=spacer, then alternating (row, spacer) pairs for
	// FocusIdx 0..7 → list item index 2,4,6,...,16. Formula: 2*FocusIdx + 2.
	if m.LastFocusIdx != m.FocusIdx {
		m.LastFocusIdx = m.FocusIdx
		targetIdx := m.FocusIdx*2 + 2
		pos := &a.createList.List.Position
		// Strict interior: item is bounded above and below by other visible items,
		// guaranteeing full visibility regardless of partial-clipping at edges.
		interior := pos.Count > 2 && targetIdx > pos.First && targetIdx < pos.First+pos.Count-1
		if !interior {
			pos.First = targetIdx
			pos.Offset = 0
		}
	}

	items := []layout.Widget{
		// Header
		func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					l := th.LabelColor(fs, unit.Sp(20), th.AccentDim, "Create Issue")
					l.Font.Weight = 700
					return l.Layout(gtx)
				}),
				layout.Rigid(rigidSpace(8)),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					h := gtx.Dp(unit.Dp(1))
					rect(gtx, image.Rect(0, 0, gtx.Constraints.Max.X, h), th.Border)
					return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, h)}
				}),
			)
		},
		rigidSpace(formRowGapDp),

		// Title row
		formRow(th, fs, "Title", m.FocusIdx == 0, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			gtx.Constraints.Min.Y = gtx.Dp(unit.Dp(28))
			gtx.Constraints.Max.Y = gtx.Dp(unit.Dp(28))
			col := th.Text
			if m.FocusIdx != 0 {
				col = th.TextDim
			}
			ed := editorStyle(th, &m.Title, "Issue title", col, fs)
			return widgetBox(gtx, th, ed.Layout)
		}),
		rigidSpace(formRowGapDp),

		// Description row
		formRow(th, fs, "Description", m.FocusIdx == 1, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			gtx.Constraints.Min.Y = gtx.Dp(unit.Dp(96))
			gtx.Constraints.Max.Y = gtx.Dp(unit.Dp(96))
			col := th.Text
			if m.FocusIdx != 1 {
				col = th.TextDim
			}
			ed := editorStyle(th, &m.Description, "Description (optional)", col, fs)
			return widgetBox(gtx, th, ed.Layout)
		}),
		rigidSpace(formRowGapDp),

		// Selectors
		a.layoutFormPriorityRow(m),
		rigidSpace(formRowGapDp),
		a.layoutFormStatusRow(m),
		rigidSpace(formRowGapDp),
		a.layoutFormAssigneeRow(m),
		rigidSpace(formRowGapDp),
		a.layoutFormProjectRow(m),
		rigidSpace(formRowGapDp),
		a.layoutFormCycleRow(m),
		rigidSpace(formRowGapDp),

		// Actions
		func(gtx layout.Context) layout.Dimensions {
			return a.modalButtons(nil, &m.Submit, "Cancel", "Create Issue", a.confirmCreateScreen, m.FocusIdx == 7)(gtx)
		},
	}

	return layout.UniformInset(unit.Dp(20)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return material.List(th.M, &a.createList).Layout(gtx, len(items), func(gtx layout.Context, i int) layout.Dimensions {
			return items[i](gtx)
		})
	})
}

func formLabel(th *ui.Theme, fs ui.FontStyle, s string, focused bool) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		color := th.TextDim
		if focused {
			color = th.Accent
		}
		l := th.LabelColor(fs, unit.Sp(13), color, s)
		l.Font.Weight = 600
		return l.Layout(gtx)
	}
}

// formRow lays out a two-column form row: a fixed-width label on the left and
// the value content flexed on the right. The label is top-aligned with a small
// inset so it sits on the first line of multi-line values (description editor).
func formRow(th *ui.Theme, fs ui.FontStyle, label string, focused bool, value layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Start}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Dp(unit.Dp(formLabelColDp))
				gtx.Constraints.Max.X = gtx.Dp(unit.Dp(formLabelColDp))
				return layout.Inset{Top: unit.Dp(10)}.Layout(gtx, formLabel(th, fs, label, focused))
			}),
			layout.Rigid(hSpace(formColGapDp)),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Constraints.Max.X
				return value(gtx)
			}),
		)
	}
}

func (a *App) layoutFormPriorityRow(m *CreateModal) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		th := a.Th
		fs := th.Fonts.CreateIssue
		labels := []string{"None", "Urgent", "High", "Medium", "Low"}
		return formRow(th, fs, "Priority", m.FocusIdx == 2, func(gtx layout.Context) layout.Dimensions {
			children := make([]layout.FlexChild, 0, 10)
			for i := 0; i < 5; i++ {
				i := i
				click := &m.PrioClicks[i]
				if click.Clicked(gtx) {
					m.Priority = i
					m.FocusIdx = 2
				}
				sel := m.Priority == i
				children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return chipBox(gtx, th, click, sel, func(gtx layout.Context) layout.Dimensions {
						return th.LabelColor(fs, unit.Sp(11), th.Text, labels[i]).Layout(gtx)
					})
				}))
				children = append(children, layout.Rigid(hSpace(6)))
			}
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx, children...)
		})(gtx)
	}
}

func (a *App) layoutFormStatusRow(m *CreateModal) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		th := a.Th
		fs := th.Fonts.CreateIssue

		if m.Meta == nil || len(m.Meta.States) == 0 {
			return formRow(th, fs, "Status", false, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return drawDimText(gtx, th, fs, "[ Default ]")
				})
			})(gtx)
		}

		if m.StatusToggle.Clicked(gtx) {
			log.Printf("[UI] Status toggle clicked! New state: expanded=%v", !m.StatusExpanded)
			m.StatusExpanded = !m.StatusExpanded
			m.FocusIdx = 3
		}

		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(formRow(th, fs, "Status", m.FocusIdx == 3, func(gtx layout.Context) layout.Dimensions {
				selectedName := "Select Status ▾"
				stateType := ""
				if m.StateIdx >= 0 && m.StateIdx < len(m.Meta.States) {
					st := m.Meta.States[m.StateIdx]
					selectedName = st.Name + " ▾"
					stateType = st.Type
					if m.StatusExpanded {
						selectedName = st.Name + " ▴"
					}
				}

				return chipBox(gtx, th, &m.StatusToggle, m.FocusIdx == 3, func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if stateType == "" {
								return layout.Dimensions{}
							}
							return statusDot(th, stateType)(gtx)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if stateType == "" {
								return layout.Dimensions{}
							}
							return hSpace(6)(gtx)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return th.LabelColor(fs, unit.Sp(11), th.Text, selectedName).Layout(gtx)
						}),
					)
				})
			})),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if !m.StatusExpanded {
					return layout.Dimensions{}
				}
				return formRowExpanded(func(gtx layout.Context) layout.Dimensions {
					children := make([]layout.FlexChild, 0, len(m.Meta.States))
					for i := range m.Meta.States {
						i := i
						if i >= len(m.StateClicks) {
							break
						}
						click := &m.StateClicks[i]
						if click.Clicked(gtx) {
							log.Printf("[UI] Status chip %d clicked", i)
							m.StateIdx = i
							m.StatusExpanded = false
						}
						sel := m.StateIdx == i
						st := m.Meta.States[i]

						children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Inset{Bottom: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return chipBox(gtx, th, click, sel, func(gtx layout.Context) layout.Dimensions {
									gtx.Constraints.Min.X = gtx.Constraints.Max.X
									return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
										layout.Rigid(statusDot(th, st.Type)),
										layout.Rigid(hSpace(6)),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return th.LabelColor(fs, unit.Sp(11), th.Text, st.Name).Layout(gtx)
										}),
									)
								})
							})
						}))
					}
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
				})(gtx)
			}),
		)
	}
}

func (a *App) layoutFormAssigneeRow(m *CreateModal) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		th := a.Th
		fs := th.Fonts.CreateIssue

		if m.Meta == nil || len(m.Meta.Members) == 0 {
			return formRow(th, fs, "Assignee", false, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return drawDimText(gtx, th, fs, "[ None ]")
				})
			})(gtx)
		}

		if m.AssigneeToggle.Clicked(gtx) {
			log.Printf("[UI] Assignee toggle clicked! New state: expanded=%v", !m.AssigneeExpanded)
			m.AssigneeExpanded = !m.AssigneeExpanded
			m.FocusIdx = 4
		}

		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(formRow(th, fs, "Assignee", m.FocusIdx == 4, func(gtx layout.Context) layout.Dimensions {
				selectedName := "Select Assignee ▾"
				if m.AssigneeIdx >= 0 && m.AssigneeIdx < len(m.Meta.Members) {
					u := m.Meta.Members[m.AssigneeIdx]
					selectedName = u.Name + " ▾"
					if m.AssigneeExpanded {
						selectedName = u.Name + " ▴"
					}
				}

				return chipBox(gtx, th, &m.AssigneeToggle, m.FocusIdx == 4, func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					return th.LabelColor(fs, unit.Sp(11), th.Text, selectedName).Layout(gtx)
				})
			})),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if !m.AssigneeExpanded {
					return layout.Dimensions{}
				}
				return formRowExpanded(func(gtx layout.Context) layout.Dimensions {
					children := make([]layout.FlexChild, 0, len(m.Meta.Members))
					for i := range m.Meta.Members {
						i := i
						if i >= len(m.AssigneeClicks) {
							break
						}
						click := &m.AssigneeClicks[i]
						if click.Clicked(gtx) {
							log.Printf("[UI] Assignee chip %d clicked", i)
							m.AssigneeIdx = i
							m.AssigneeExpanded = false
						}
						sel := m.AssigneeIdx == i
						u := m.Meta.Members[i]

						children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Inset{Bottom: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return chipBox(gtx, th, click, sel, func(gtx layout.Context) layout.Dimensions {
									gtx.Constraints.Min.X = gtx.Constraints.Max.X
									return th.LabelColor(fs, unit.Sp(11), th.Text, u.Name).Layout(gtx)
								})
							})
						}))
					}
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
				})(gtx)
			}),
		)
	}
}

func (a *App) layoutFormProjectRow(m *CreateModal) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		th := a.Th
		fs := th.Fonts.CreateIssue

		if a.State.LeadingProjects == nil || len(a.State.LeadingProjects) == 0 {
			return formRow(th, fs, "Project", false, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return drawDimText(gtx, th, fs, "[ None ]")
				})
			})(gtx)
		}

		if m.ProjectToggle.Clicked(gtx) {
			log.Printf("[UI] Project toggle clicked! New state: expanded=%v", !m.ProjectExpanded)
			m.ProjectExpanded = !m.ProjectExpanded
			m.FocusIdx = 5
		}

		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(formRow(th, fs, "Project", m.FocusIdx == 5, func(gtx layout.Context) layout.Dimensions {
				selectedName := "Select Project ▾"
				if m.ProjectIdx >= 0 && m.ProjectIdx < len(a.State.LeadingProjects) {
					p := a.State.LeadingProjects[m.ProjectIdx]
					pName := cleanProjectName(p.Name)
					selectedName = pName + " ▾"
					if m.ProjectExpanded {
						selectedName = pName + " ▴"
					}
				}

				return chipBox(gtx, th, &m.ProjectToggle, m.FocusIdx == 5, func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					return th.LabelColor(fs, unit.Sp(11), th.Text, selectedName).Layout(gtx)
				})
			})),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if !m.ProjectExpanded {
					return layout.Dimensions{}
				}
				return formRowExpanded(func(gtx layout.Context) layout.Dimensions {
					children := make([]layout.FlexChild, 0, len(a.State.LeadingProjects))
					for i := range a.State.LeadingProjects {
						i := i
						if i >= len(m.ProjectClicks) {
							break
						}
						click := &m.ProjectClicks[i]
						if click.Clicked(gtx) {
							log.Printf("[UI] Project chip %d clicked", i)
							m.ProjectIdx = i
							m.ProjectExpanded = false
						}
						sel := m.ProjectIdx == i
						p := a.State.LeadingProjects[i]

						children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Inset{Bottom: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return chipBox(gtx, th, click, sel, func(gtx layout.Context) layout.Dimensions {
									gtx.Constraints.Min.X = gtx.Constraints.Max.X
									return th.LabelColor(fs, unit.Sp(11), th.Text, cleanProjectName(p.Name)).Layout(gtx)
								})
							})
						}))
					}
					return layout.Flex{Axis: layout.Vertical, Alignment: layout.Start}.Layout(gtx, children...)
				})(gtx)
			}),
		)
	}
}

func (a *App) layoutFormCycleRow(m *CreateModal) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		th := a.Th
		fs := th.Fonts.CreateIssue

		if m.Meta == nil || len(m.Meta.Cycles) == 0 {
			return formRow(th, fs, "Cycle", false, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return drawDimText(gtx, th, fs, "[ None ]")
				})
			})(gtx)
		}

		if m.CycleToggle.Clicked(gtx) {
			log.Printf("[UI] Cycle toggle clicked! New state: expanded=%v", !m.CycleExpanded)
			m.CycleExpanded = !m.CycleExpanded
			m.FocusIdx = 6
		}

		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(formRow(th, fs, "Cycle", m.FocusIdx == 6, func(gtx layout.Context) layout.Dimensions {
				selectedName := "Select Cycle ▾"
				if m.CycleIdx >= 0 && m.CycleIdx < len(m.Meta.Cycles) {
					c := m.Meta.Cycles[m.CycleIdx]
					name := c.Name
					if name == "" {
						name = fmt.Sprintf("Cycle %d", c.Number)
					}
					selectedName = truncate(name, 24) + " ▾"
					if m.CycleExpanded {
						selectedName = truncate(name, 24) + " ▴"
					}
				}

				return chipBox(gtx, th, &m.CycleToggle, m.FocusIdx == 6, func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					return th.LabelColor(fs, unit.Sp(11), th.Text, selectedName).Layout(gtx)
				})
			})),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if !m.CycleExpanded {
					return layout.Dimensions{}
				}
				return formRowExpanded(func(gtx layout.Context) layout.Dimensions {
					children := make([]layout.FlexChild, 0, len(m.Meta.Cycles))
					for i := range m.Meta.Cycles {
						i := i
						if i >= len(m.CycleClicks) {
							break
						}
						click := &m.CycleClicks[i]
						if click.Clicked(gtx) {
							log.Printf("[UI] Cycle chip %d clicked", i)
							m.CycleIdx = i
							m.CycleExpanded = false
						}
						sel := m.CycleIdx == i
						c := m.Meta.Cycles[i]
						name := c.Name
						if name == "" {
							name = fmt.Sprintf("Cycle %d", c.Number)
						}

						children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Inset{Bottom: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return chipBox(gtx, th, click, sel, func(gtx layout.Context) layout.Dimensions {
									gtx.Constraints.Min.X = gtx.Constraints.Max.X
									return th.LabelColor(fs, unit.Sp(11), th.Text, name).Layout(gtx)
								})
							})
						}))
					}
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
				})(gtx)
			}),
		)
	}
}

// formRowExpanded renders an expanded list (e.g. dropdown options) indented
// to align under the value column of formRow.
func formRowExpanded(content layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{
			Top:  unit.Dp(6),
			Left: unit.Dp(formLabelColDp + formColGapDp),
		}.Layout(gtx, content)
	}
}

// chipBox renders a small rounded background sized to the inner widget.
// It records the inner content, measures it, paints the background at the
// measured size, then replays the content on top — avoiding the
// Stack/Expanded sizing pitfall when outer constraints are large.
func chipBox(gtx layout.Context, th *ui.Theme, click *widget.Clickable, selected bool, w layout.Widget) layout.Dimensions {
	return click.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		macro := op.Record(gtx.Ops)
		dims := layout.Inset{
			Top: unit.Dp(4), Bottom: unit.Dp(4),
			Left: unit.Dp(8), Right: unit.Dp(8),
		}.Layout(gtx, w)
		content := macro.Stop()

		bg := th.PanelAlt
		if selected {
			bg = th.Selected
		} else if click.Hovered() {
			bg = th.Border
		}
		r := image.Rect(0, 0, dims.Size.X, dims.Size.Y)
		rr := gtx.Dp(unit.Dp(4))
		defer clip.UniformRRect(r, rr).Push(gtx.Ops).Pop()
		rect(gtx, r, bg)
		content.Add(gtx.Ops)
		return dims
	})
}

// hSpace is a horizontal Spacer (Width set, not Height).
func hSpace(dp int) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Spacer{Width: unit.Dp(float32(dp))}.Layout(gtx)
	}
}
