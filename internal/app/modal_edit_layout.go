package app

import (
	"fmt"
	"image"

	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget/material"
)

// --- Edit ---

func (a *App) layoutEditIssue(gtx layout.Context) layout.Dimensions {
	m := a.State.Edit
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
			gtx.Execute(key.FocusCmd{Tag: nil})
		}
	}

	if gtx.Focused(&m.Title) {
		m.FocusIdx = 0
	} else if gtx.Focused(&m.Description) {
		m.FocusIdx = 1
	}

	items := []layout.Widget{
		// Header
		func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Baseline}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							l := th.LabelColor(fs, unit.Sp(20), th.AccentDim, "Edit "+m.Issue.Identifier)
							l.Font.Weight = 700
							return l.Layout(gtx)
						}),
						layout.Rigid(rigidSpace(12)),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return th.LabelColor(fs, unit.Sp(12), th.TextMuted, m.Issue.Title).Layout(gtx)
						}),
					)
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
		a.layoutEditFormPriorityRow(m),
		rigidSpace(formRowGapDp),
		a.layoutEditFormStatusRow(m),
		rigidSpace(formRowGapDp),
		a.layoutEditFormAssigneeRow(m),
		rigidSpace(formRowGapDp),
		a.layoutEditFormLabelRow(m),
		rigidSpace(formRowGapDp),
		a.layoutEditFormProjectRow(m),
		rigidSpace(formRowGapDp),
		a.layoutEditFormCycleRow(m),
		rigidSpace(formRowGapDp),

		// Actions
		func(gtx layout.Context) layout.Dimensions {
			return a.modalButtons(nil, &m.Submit, "Cancel", "Save Changes", a.confirmEditScreen, m.FocusIdx == 8)(gtx)
		},
	}

	return layout.UniformInset(unit.Dp(20)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return material.List(th.M, &a.editList).Layout(gtx, len(items), func(gtx layout.Context, i int) layout.Dimensions {
			return items[i](gtx)
		})
	})
}

func (a *App) layoutEditFormPriorityRow(m *EditModal) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		th := a.Th
		fs := th.Fonts.IssueDetail
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

func (a *App) layoutEditFormStatusRow(m *EditModal) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		th := a.Th
		fs := th.Fonts.IssueDetail

		if m.Meta == nil || len(m.Meta.States) == 0 {
			return formRow(th, fs, "Status", false, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return drawDimText(gtx, th, fs, "[ Default ]")
				})
			})(gtx)
		}

		if m.StatusToggle.Clicked(gtx) {
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

func (a *App) layoutEditFormAssigneeRow(m *EditModal) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		th := a.Th
		fs := th.Fonts.IssueDetail

		if m.Meta == nil || len(m.Meta.Members) == 0 {
			return formRow(th, fs, "Assignee", false, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return drawDimText(gtx, th, fs, "[ None ]")
				})
			})(gtx)
		}

		if m.AssigneeToggle.Clicked(gtx) {
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

func (a *App) layoutEditFormLabelRow(m *EditModal) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		th := a.Th
		fs := th.Fonts.IssueDetail

		if m.Meta == nil || len(m.Meta.Labels) == 0 {
			return formRow(th, fs, "Label", false, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return drawDimText(gtx, th, fs, "[ None ]")
				})
			})(gtx)
		}

		if m.LabelToggle.Clicked(gtx) {
			m.LabelExpanded = !m.LabelExpanded
			m.FocusIdx = 5
		}

		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(formRow(th, fs, "Label", m.FocusIdx == 5, func(gtx layout.Context) layout.Dimensions {
				selectedName := "Select Label ▾"
				if m.LabelIdx >= 0 && m.LabelIdx < len(m.Meta.Labels) {
					l := m.Meta.Labels[m.LabelIdx]
					selectedName = l.Name + " ▾"
					if m.LabelExpanded {
						selectedName = l.Name + " ▴"
					}
				}

				return chipBox(gtx, th, &m.LabelToggle, m.FocusIdx == 5, func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					return th.LabelColor(fs, unit.Sp(11), th.Text, selectedName).Layout(gtx)
				})
			})),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if !m.LabelExpanded {
					return layout.Dimensions{}
				}
				return formRowExpanded(func(gtx layout.Context) layout.Dimensions {
					children := make([]layout.FlexChild, 0, len(m.Meta.Labels))
					for i := range m.Meta.Labels {
						i := i
						if i >= len(m.LabelClicks) {
							break
						}
						click := &m.LabelClicks[i]
						if click.Clicked(gtx) {
							m.LabelIdx = i
							m.LabelExpanded = false
						}
						sel := m.LabelIdx == i
						l := m.Meta.Labels[i]

						children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Inset{Bottom: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return chipBox(gtx, th, click, sel, func(gtx layout.Context) layout.Dimensions {
									gtx.Constraints.Min.X = gtx.Constraints.Max.X
									return th.LabelColor(fs, unit.Sp(11), th.Text, l.Name).Layout(gtx)
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

func (a *App) layoutEditFormProjectRow(m *EditModal) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		th := a.Th
		fs := th.Fonts.IssueDetail

		if len(a.State.LeadingProjects) == 0 {
			return formRow(th, fs, "Project", false, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return drawDimText(gtx, th, fs, "[ None ]")
				})
			})(gtx)
		}

		if m.ProjectToggle.Clicked(gtx) {
			m.ProjectExpanded = !m.ProjectExpanded
			m.FocusIdx = 6
		}

		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(formRow(th, fs, "Project", m.FocusIdx == 6, func(gtx layout.Context) layout.Dimensions {
				selectedName := "Select Project ▾"
				if m.ProjectIdx >= 0 && m.ProjectIdx < len(a.State.LeadingProjects) {
					p := a.State.LeadingProjects[m.ProjectIdx]
					pName := cleanProjectName(p.Name)
					selectedName = pName + " ▾"
					if m.ProjectExpanded {
						selectedName = pName + " ▴"
					}
				}

				return chipBox(gtx, th, &m.ProjectToggle, m.FocusIdx == 6, func(gtx layout.Context) layout.Dimensions {
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

func (a *App) layoutEditFormCycleRow(m *EditModal) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		th := a.Th
		fs := th.Fonts.IssueDetail

		if m.Meta == nil || len(m.Meta.Cycles) == 0 {
			return formRow(th, fs, "Cycle", false, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return drawDimText(gtx, th, fs, "[ None ]")
				})
			})(gtx)
		}

		if m.CycleToggle.Clicked(gtx) {
			m.CycleExpanded = !m.CycleExpanded
			m.FocusIdx = 7
		}

		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(formRow(th, fs, "Cycle", m.FocusIdx == 7, func(gtx layout.Context) layout.Dimensions {
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

				return chipBox(gtx, th, &m.CycleToggle, m.FocusIdx == 7, func(gtx layout.Context) layout.Dimensions {
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
