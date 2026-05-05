package app

import (
	"image"

	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/denislee/wllinear/internal/ui"
)

func (a *App) layoutCreateIssue(gtx layout.Context) layout.Dimensions {
	m := a.State.Create
	if m == nil {
		return layout.Dimensions{}
	}
	th := a.Th
	fs := th.Fonts.IssueDetail

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
			// Clear focus from editors if we navigate to priority, status, etc.
			gtx.Execute(key.FocusCmd{Tag: nil})
		}
	}

	if gtx.Focused(&m.Title) {
		m.FocusIdx = 0
	} else if gtx.Focused(&m.Description) {
		m.FocusIdx = 1
	}

	return layout.UniformInset(unit.Dp(20)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			// Header
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
			layout.Rigid(rigidSpace(20)),

			// Title
			layout.Rigid(formLabel(th, fs, "Title:", m.FocusIdx == 0)),
			layout.Rigid(rigidSpace(4)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				ed := editorStyle(th, &m.Title, "Issue title", th.Text, fs)
				gtx.Constraints.Min.Y = gtx.Dp(unit.Dp(36))
				gtx.Constraints.Max.Y = gtx.Dp(unit.Dp(36))
				if m.FocusIdx == 0 {
					ed.Color = th.Accent
				}
				return widgetBox(gtx, th, ed.Layout)
			}),
			layout.Rigid(rigidSpace(16)),

			// Description (fixed height)
			layout.Rigid(formLabel(th, fs, "Description:", m.FocusIdx == 1)),
			layout.Rigid(rigidSpace(4)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				ed := editorStyle(th, &m.Description, "Description (optional)", th.Text, fs)
				gtx.Constraints.Min.Y = gtx.Dp(unit.Dp(160))
				gtx.Constraints.Max.Y = gtx.Dp(unit.Dp(160))
				if m.FocusIdx == 1 {
					ed.Color = th.Accent
				}
				return widgetBox(gtx, th, ed.Layout)
			}),
			layout.Rigid(rigidSpace(20)),

			// Priority row
			layout.Rigid(a.layoutFormPriorityRow(m)),
			layout.Rigid(rigidSpace(16)),

			// Status row
			layout.Rigid(a.layoutFormStatusRow(m)),
			layout.Rigid(rigidSpace(16)),

			// Project row
			layout.Rigid(a.layoutFormProjectRow(m)),
			layout.Rigid(rigidSpace(16)),

			// Cycle row
			layout.Rigid(a.layoutFormCycleRow(m)),
			layout.Rigid(rigidSpace(24)),

			// Actions
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				// Indicate focus for Submit button if FocusIdx == 6 by drawing a slight indicator or just relying on label color.
				return a.modalButtons(nil, &m.Submit, "Cancel", "Create Issue", a.confirmCreateScreen, m.FocusIdx == 6)(gtx)
			}),
		)
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

func (a *App) layoutFormPriorityRow(m *CreateModal) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		th := a.Th
		fs := th.Fonts.IssueDetail
		labels := []string{"None", "Urgent", "High", "Medium", "Low"}
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Dp(unit.Dp(96))
				return formLabel(th, fs, "Priority:", m.FocusIdx == 2)(gtx)
			}),
			layout.Rigid(rigidSpace(8)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
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
			}),
		)
	}
}

func (a *App) layoutFormStatusRow(m *CreateModal) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		th := a.Th
		fs := th.Fonts.IssueDetail
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Start}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Dp(unit.Dp(96))
				return layout.Inset{Top: unit.Dp(6)}.Layout(gtx, formLabel(th, fs, "Status:", m.FocusIdx == 3))
			}),
			layout.Rigid(rigidSpace(8)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if m.Meta == nil {
					return layout.Inset{Top: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return drawDimText(gtx, th, fs, "[ Default ]")
					})
				}
				if len(m.Meta.States) == 0 {
					return layout.Inset{Top: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return drawDimText(gtx, th, fs, "No states available")
					})
				}

				if m.StatusToggle.Clicked(gtx) {
					m.StatusExpanded = !m.StatusExpanded
					m.FocusIdx = 3
				}

				selectedName := "Select Status ▾"
				if m.StateIdx >= 0 && m.StateIdx < len(m.Meta.States) {
					selectedName = m.Meta.States[m.StateIdx].Name + " ▾"
					if m.StatusExpanded {
						selectedName = m.Meta.States[m.StateIdx].Name + " ▴"
					}
				} else if m.StatusExpanded {
					selectedName = "Select Status ▴"
				}

				toggleBtn := func(gtx layout.Context) layout.Dimensions {
					return chipBox(gtx, th, &m.StatusToggle, m.FocusIdx == 3, func(gtx layout.Context) layout.Dimensions {
						return th.LabelColor(fs, unit.Sp(11), th.Text, selectedName).Layout(gtx)
					})
				}

				if !m.StatusExpanded {
					return toggleBtn(gtx)
				}

				children := make([]layout.FlexChild, 0, len(m.Meta.States)+1)
				children = append(children, layout.Rigid(toggleBtn))
				children = append(children, layout.Rigid(rigidSpace(4)))

				// Wrap chips horizontally
				var currentRow []layout.FlexChild
				for i := range m.Meta.States {
					i := i
					if i >= len(m.StateClicks) {
						break
					}
					click := &m.StateClicks[i]
					if click.Clicked(gtx) {
						m.StateIdx = i
						m.StatusExpanded = false
						m.FocusIdx = 3
					}
					sel := m.StateIdx == i
					stateType := m.Meta.States[i].Type
					name := m.Meta.States[i].Name

					chip := func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Right: unit.Dp(6), Bottom: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return chipBox(gtx, th, click, sel, func(gtx layout.Context) layout.Dimensions {
								return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
									layout.Rigid(statusDot(th, stateType)),
									layout.Rigid(hSpace(6)),
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										return th.LabelColor(fs, unit.Sp(11), th.Text, name).Layout(gtx)
									}),
								)
							})
						})
					}
					currentRow = append(currentRow, layout.Rigid(chip))
				}
				// Since Gioui doesn't have a simple wrap layout, we'll just stack them vertically to be safe, 
				// or horizontally if there are few. For dropdown, vertical is normal.
				for _, c := range currentRow {
					children = append(children, c)
				}

				return layout.Flex{Axis: layout.Vertical, Alignment: layout.Start}.Layout(gtx, children...)
			}),
		)
	}
}

func (a *App) layoutFormProjectRow(m *CreateModal) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		th := a.Th
		fs := th.Fonts.IssueDetail
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Start}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Dp(unit.Dp(96))
				return layout.Inset{Top: unit.Dp(6)}.Layout(gtx, formLabel(th, fs, "Project:", m.FocusIdx == 4))
			}),
			layout.Rigid(rigidSpace(8)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if a.State.LeadingProjects == nil {
					return layout.Inset{Top: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return drawDimText(gtx, th, fs, "[ None ]")
					})
				}
				if len(a.State.LeadingProjects) == 0 {
					return layout.Inset{Top: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return drawDimText(gtx, th, fs, "No projects")
					})
				}

				if m.ProjectToggle.Clicked(gtx) {
					m.ProjectExpanded = !m.ProjectExpanded
					m.FocusIdx = 4
				}

				selectedName := "Select Project ▾"
				if m.ProjectIdx >= 0 && m.ProjectIdx < len(a.State.LeadingProjects) {
					selectedName = truncate(a.State.LeadingProjects[m.ProjectIdx].Name, 24) + " ▾"
					if m.ProjectExpanded {
						selectedName = truncate(a.State.LeadingProjects[m.ProjectIdx].Name, 24) + " ▴"
					}
				} else if m.ProjectExpanded {
					selectedName = "Select Project ▴"
				}

				toggleBtn := func(gtx layout.Context) layout.Dimensions {
					return chipBox(gtx, th, &m.ProjectToggle, m.FocusIdx == 4, func(gtx layout.Context) layout.Dimensions {
						return th.LabelColor(fs, unit.Sp(11), th.Text, selectedName).Layout(gtx)
					})
				}

				if !m.ProjectExpanded {
					return toggleBtn(gtx)
				}

				children := make([]layout.FlexChild, 0, len(a.State.LeadingProjects)+2)
				children = append(children, layout.Rigid(toggleBtn))
				children = append(children, layout.Rigid(rigidSpace(4)))

				for i := range a.State.LeadingProjects {
					i := i
					if i >= len(m.ProjectClicks) {
						break
					}
					click := &m.ProjectClicks[i]
					if click.Clicked(gtx) {
						if m.ProjectIdx == i {
							m.ProjectIdx = -1
						} else {
							m.ProjectIdx = i
						}
						m.ProjectExpanded = false
						m.FocusIdx = 4
					}
					sel := m.ProjectIdx == i
					name := truncate(a.State.LeadingProjects[i].Name, 24)
					
					chip := func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Bottom: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return chipBox(gtx, th, click, sel, func(gtx layout.Context) layout.Dimensions {
								return th.LabelColor(fs, unit.Sp(11), th.Text, name).Layout(gtx)
							})
						})
					}
					children = append(children, layout.Rigid(chip))
				}

				return layout.Flex{Axis: layout.Vertical, Alignment: layout.Start}.Layout(gtx, children...)
			}),
		)
	}
}

func (a *App) layoutFormCycleRow(m *CreateModal) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		th := a.Th
		fs := th.Fonts.IssueDetail
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Start}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Dp(unit.Dp(96))
				return layout.Inset{Top: unit.Dp(6)}.Layout(gtx, formLabel(th, fs, "Cycle:", m.FocusIdx == 5))
			}),
			layout.Rigid(rigidSpace(8)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if m.Meta == nil || len(m.Meta.Cycles) == 0 {
					return layout.Inset{Top: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return drawDimText(gtx, th, fs, "[ None ]")
					})
				}

				if m.CycleToggle.Clicked(gtx) {
					m.CycleExpanded = !m.CycleExpanded
					m.FocusIdx = 5
				}

				selectedName := "Select Cycle ▾"
				if m.CycleIdx >= 0 && m.CycleIdx < len(m.Meta.Cycles) {
					selectedName = truncate(m.Meta.Cycles[m.CycleIdx].Name, 24) + " ▾"
					if m.CycleExpanded {
						selectedName = truncate(m.Meta.Cycles[m.CycleIdx].Name, 24) + " ▴"
					}
				} else if m.CycleExpanded {
					selectedName = "Select Cycle ▴"
				}

				toggleBtn := func(gtx layout.Context) layout.Dimensions {
					return chipBox(gtx, th, &m.CycleToggle, m.FocusIdx == 5, func(gtx layout.Context) layout.Dimensions {
						return th.LabelColor(fs, unit.Sp(11), th.Text, selectedName).Layout(gtx)
					})
				}

				if !m.CycleExpanded {
					return toggleBtn(gtx)
				}

				children := make([]layout.FlexChild, 0, len(m.Meta.Cycles)+2)
				children = append(children, layout.Rigid(toggleBtn))
				children = append(children, layout.Rigid(rigidSpace(4)))

				for i := range m.Meta.Cycles {
					i := i
					if i >= len(m.CycleClicks) {
						break
					}
					click := &m.CycleClicks[i]
					if click.Clicked(gtx) {
						if m.CycleIdx == i {
							m.CycleIdx = -1
						} else {
							m.CycleIdx = i
						}
						m.CycleExpanded = false
						m.FocusIdx = 5
					}
					sel := m.CycleIdx == i
					name := truncate(m.Meta.Cycles[i].Name, 24)
					
					chip := func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Bottom: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return chipBox(gtx, th, click, sel, func(gtx layout.Context) layout.Dimensions {
								return th.LabelColor(fs, unit.Sp(11), th.Text, name).Layout(gtx)
							})
						})
					}
					children = append(children, layout.Rigid(chip))
				}

				return layout.Flex{Axis: layout.Vertical, Alignment: layout.Start}.Layout(gtx, children...)
			}),
		)
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
