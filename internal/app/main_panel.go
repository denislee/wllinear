package app

import (
	"fmt"
	"image"
	"image/color"
	"strings"
	"time"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/denislee/wllinear/internal/linear"
	"github.com/denislee/wllinear/internal/ui"
)

func (a *App) layoutMain(gtx layout.Context, r image.Rectangle) {
	defer op.Offset(r.Min).Push(gtx.Ops).Pop()
	w, h := r.Dx(), r.Dy()
	rect(gtx, image.Rect(0, 0, w, h), a.Th.BG)
	gtx.Constraints = layout.Constraints{Min: image.Point{}, Max: image.Pt(w, h)}

	if a.State.View == ViewCreateIssue {
		a.layoutCreateIssue(gtx)
		return
	}
	if a.State.View == ViewIssueDetail && a.State.Detail != nil {
		a.layoutIssueDetail(gtx)
		return
	}
	if a.State.View == ViewProjectCycles {
		a.layoutProjectCycles(gtx)
		return
	}
	a.layoutIssueList(gtx)
}

func (a *App) layoutProjectCycles(gtx layout.Context) layout.Dimensions {
	st := a.State
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.layoutProjectCyclesHeader(gtx)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			cycles := st.ProjectCycles
			if len(cycles) == 0 {
				return layout.Inset{Top: unit.Dp(8), Left: unit.Dp(20)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return drawDimText(gtx, a.Th, a.Th.Fonts.IssueList, "No cycles found for this project.")
				})
			}
			return material.List(a.Th.M, &a.listIssues).Layout(gtx, len(cycles), func(gtx layout.Context, idx int) layout.Dimensions {
				c := cycles[idx]
				selected := idx == st.Selected
				expanded := st.ExpandedCycles[c.Cycle.ID]
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.drawCycleRow(gtx, c, selected, expanded)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if !expanded {
							return layout.Dimensions{}
						}
						return a.drawCycleIssues(gtx, c.Issues)
					}),
				)
			})
		}),
	)
}

func (a *App) layoutProjectCyclesHeader(gtx layout.Context) layout.Dimensions {
	st := a.State
	title := strings.TrimPrefix(st.ActiveFilter, "Project: ")
	title = cleanProjectName(title)
	return layout.Inset{Top: unit.Dp(16), Left: unit.Dp(20), Right: unit.Dp(20), Bottom: unit.Dp(12)}.Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Baseline}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							l := a.Th.LabelColor(a.Th.Fonts.IssueList, unit.Sp(16), a.Th.Text, title)
							l.Font.Weight = 700
							l.MaxLines = 1
							return l.Layout(gtx)
						}),
						layout.Rigid(hSpace(10)),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							n := len(st.ProjectCycles)
							word := "cycles"
							if n == 1 {
								word = "cycle"
							}
							return a.Th.LabelColor(a.Th.Fonts.IssueList, unit.Sp(12), a.Th.TextMuted, fmt.Sprintf("%d %s", n, word)).Layout(gtx)
						}),
					)
				}),
				layout.Rigid(rigidSpace(8)),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					h := gtx.Dp(unit.Dp(1))
					rect(gtx, image.Rect(0, 0, gtx.Constraints.Max.X, h), a.Th.Border)
					return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, h)}
				}),
			)
		},
	)
}

// cycleStatus classifies a cycle as active, upcoming, or completed for display.
func cycleStatus(c linear.Cycle) (label, stateType string) {
	now := time.Now()
	switch {
	case c.CompletedAt != nil:
		return "completed", "completed"
	case !c.StartsAt.IsZero() && now.Before(c.StartsAt):
		return "upcoming", "unstarted"
	case !c.EndsAt.IsZero() && now.After(c.EndsAt):
		return "ended", "completed"
	default:
		return "active", "started"
	}
}

func cycleDateRange(c linear.Cycle) string {
	if c.StartsAt.IsZero() && c.EndsAt.IsZero() {
		return ""
	}
	const f = "Jan 2"
	sameYear := c.StartsAt.Year() == c.EndsAt.Year() && c.StartsAt.Year() == time.Now().Year()
	if !sameYear && !c.StartsAt.IsZero() && !c.EndsAt.IsZero() {
		return c.StartsAt.Format("Jan 2, 2006") + " – " + c.EndsAt.Format("Jan 2, 2006")
	}
	if c.StartsAt.IsZero() {
		return "ends " + c.EndsAt.Format(f)
	}
	if c.EndsAt.IsZero() {
		return "from " + c.StartsAt.Format(f)
	}
	return c.StartsAt.Format(f) + " – " + c.EndsAt.Format(f)
}

func (a *App) drawCycleRow(gtx layout.Context, c linear.ProjectCycleIssues, selected, expanded bool) layout.Dimensions {
	th := a.Th
	fs := th.Fonts.IssueList
	_, stateType := cycleStatus(c.Cycle)

	macro := op.Record(gtx.Ops)
	dims := layout.Inset{
		Top: unit.Dp(10), Bottom: unit.Dp(10),
		Left: unit.Dp(20), Right: unit.Dp(20),
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				caret := "▶"
				if expanded {
					caret = "▼"
				}
				gtx.Constraints.Min.X = gtx.Dp(unit.Dp(14))
				gtx.Constraints.Max.X = gtx.Dp(unit.Dp(14))
				return th.LabelColor(fs, unit.Sp(10), th.TextDim, caret).Layout(gtx)
			}),
			layout.Rigid(hSpace(6)),
			layout.Rigid(statusDot(th, stateType)),
			layout.Rigid(hSpace(12)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Dp(unit.Dp(36))
				gtx.Constraints.Max.X = gtx.Dp(unit.Dp(36))
				l := th.LabelColor(fs, unit.Sp(13), th.TextDim, fmt.Sprintf("#%d", c.Cycle.Number))
				l.Font.Weight = 700
				l.MaxLines = 1
				return l.Layout(gtx)
			}),
			layout.Rigid(hSpace(8)),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				name := strings.TrimSpace(c.Cycle.Name)
				if name == "" {
					name = fmt.Sprintf("Cycle %d", c.Cycle.Number)
				}
				l := th.LabelColor(fs, unit.Sp(14), th.Text, name)
				l.Font.Weight = 500
				l.MaxLines = 1
				return l.Layout(gtx)
			}),
			layout.Rigid(hSpace(16)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				dr := cycleDateRange(c.Cycle)
				if dr == "" {
					return layout.Dimensions{}
				}
				return th.LabelColor(fs, unit.Sp(12), th.TextMuted, dr).Layout(gtx)
			}),
			layout.Rigid(hSpace(16)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Dp(unit.Dp(56))
				gtx.Constraints.Max.X = gtx.Dp(unit.Dp(56))
				count := fmt.Sprintf("%d issue", len(c.Issues))
				if len(c.Issues) != 1 {
					count += "s"
				}
				l := th.LabelColor(fs, unit.Sp(12), th.TextDim, count)
				l.Alignment = text.End
				l.MaxLines = 1
				return l.Layout(gtx)
			}),
		)
	})
	content := macro.Stop()

	size := image.Pt(gtx.Constraints.Max.X, dims.Size.Y)
	if selected {
		rect(gtx, image.Rect(0, 0, size.X, size.Y), th.Selected)
	}
	content.Add(gtx.Ops)
	return layout.Dimensions{Size: size}
}

func (a *App) drawCycleIssues(gtx layout.Context, issues []linear.Issue) layout.Dimensions {
	th := a.Th
	fs := th.Fonts.IssueList
	if len(issues) == 0 {
		return layout.Inset{
			Top: unit.Dp(2), Bottom: unit.Dp(8),
			Left: unit.Dp(60), Right: unit.Dp(20),
		}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return th.LabelColor(fs, unit.Sp(12), th.TextMuted, "No issues in this cycle.").Layout(gtx)
		})
	}
	children := make([]layout.FlexChild, 0, len(issues)+1)
	for i := range issues {
		is := issues[i]
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{
				Top: unit.Dp(4), Bottom: unit.Dp(4),
				Left: unit.Dp(60), Right: unit.Dp(20),
			}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(statusDot(th, is.State.Type)),
					layout.Rigid(hSpace(10)),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Min.X = gtx.Dp(unit.Dp(80))
						gtx.Constraints.Max.X = gtx.Dp(unit.Dp(80))
						l := th.LabelColor(fs, unit.Sp(12), th.TextDim, is.Identifier)
						l.Font.Weight = 700
						l.MaxLines = 1
						return l.Layout(gtx)
					}),
					layout.Rigid(hSpace(12)),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						l := th.LabelColor(fs, unit.Sp(13), th.Text, is.Title)
						l.MaxLines = 1
						return l.Layout(gtx)
					}),
				)
			})
		}))
	}
	children = append(children, layout.Rigid(rigidSpace(6)))
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
}

// --- Issue list ---

func (a *App) layoutIssueList(gtx layout.Context) layout.Dimensions {
	st := a.State

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.layoutIssueListHeader(gtx)
		}),
		layout.Rigid(rigidSpace(4)),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			issues := st.Issues
			if len(issues) == 0 {
				return drawDimText(gtx, a.Th, a.Th.Fonts.IssueList, "No issues. Try a different filter.")
			}
			if st.Selected >= len(issues) {
				st.Selected = len(issues) - 1
			}
			if cap(a.issueClicks) < len(issues) {
				a.issueClicks = make([]widget.Clickable, len(issues))
			}
			a.issueClicks = a.issueClicks[:len(issues)]
			return material.List(a.Th.M, &a.listIssues).Layout(gtx, len(issues), func(gtx layout.Context, idx int) layout.Dimensions {
				is := issues[idx]
				click := &a.issueClicks[idx]
				if click.Clicked(gtx) {
					st.Selected = idx
					issue := is
					st.Detail = &issue
					st.View = ViewIssueDetail
					a.updateHints()
				}
				selected := idx == st.Selected
				return a.drawIssueRow(gtx, click, is, selected)
			})
		}),
	)
}

func (a *App) layoutIssueListHeader(gtx layout.Context) layout.Dimensions {
	st := a.State
	title := st.ActiveFilter
	if st.Team != nil {
		title = st.Team.Name + " · " + title
	}
	return layout.Inset{Top: unit.Dp(12), Left: unit.Dp(16), Right: unit.Dp(16), Bottom: unit.Dp(8)}.Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					l := a.Th.LabelColor(a.Th.Fonts.IssueList, unit.Sp(16), a.Th.Text, title)
					l.Font.Weight = 700
					l.MaxLines = 1
					return l.Layout(gtx)
				}),
				layout.Rigid(hSpace(12)),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					n := len(st.Issues)
					return a.Th.LabelColor(a.Th.Fonts.IssueList, unit.Sp(13), a.Th.TextMuted, fmt.Sprintf("(%d)", n)).Layout(gtx)
				}),
			)
		},
	)
}

func (a *App) drawIssueRow(gtx layout.Context, click *widget.Clickable, is linear.Issue, selected bool) layout.Dimensions {
	return click.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		macro := op.Record(gtx.Ops)
		var dims layout.Dimensions
		if a.State.Compact {
			dims = a.drawIssueRowCompact(gtx, is)
		} else {
			dims = a.drawIssueRowFull(gtx, is)
		}
		content := macro.Stop()

		size := image.Pt(gtx.Constraints.Max.X, dims.Size.Y)
		if selected {
			rect(gtx, image.Rect(0, 0, size.X, size.Y), a.Th.Selected)
		} else if click.Hovered() {
			rect(gtx, image.Rect(0, 0, size.X, size.Y), a.Th.PanelAlt)
		}
		content.Add(gtx.Ops)
		return layout.Dimensions{Size: size}
	})
}

func (a *App) drawIssueRowCompact(gtx layout.Context, is linear.Issue) layout.Dimensions {
	fs := a.Th.Fonts.IssueList
	return layout.Inset{
		Top: unit.Dp(6), Bottom: unit.Dp(6),
		Left: unit.Dp(16), Right: unit.Dp(16),
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if !a.State.ShowPriority {
					return layout.Dimensions{}
				}
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(priorityChip(a.Th, fs, is.Priority)),
					layout.Rigid(hSpace(12)),
				)
			}),
			layout.Rigid(statusDot(a.Th, is.State.Type)),
			layout.Rigid(hSpace(12)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Dp(unit.Dp(80))
				gtx.Constraints.Max.X = gtx.Dp(unit.Dp(80))
				l := a.Th.LabelColor(fs, unit.Sp(13), a.Th.TextDim, is.Identifier)
				l.Font.Weight = 700
				l.MaxLines = 1
				return l.Layout(gtx)
			}),
			layout.Rigid(hSpace(16)),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				l := a.Th.LabelColor(fs, unit.Sp(14), a.Th.Text, is.Title)
				l.MaxLines = 1
				return l.Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if !a.State.ShowLabels {
					return layout.Dimensions{}
				}
				width := gtx.Dp(unit.Dp(60))
				gtx.Constraints.Min.X = width
				gtx.Constraints.Max.X = width
				if len(is.Labels.Nodes) == 0 {
					return layout.Dimensions{Size: image.Pt(width, 0)}
				}
				return layout.Inset{Left: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return dimLabel(gtx, a.Th, fs, unit.Sp(12), a.Th.TextMuted, labelString(is.Labels.Nodes, 1))
				})
			}),
			layout.Rigid(hSpace(16)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Dp(unit.Dp(28))
				gtx.Constraints.Max.X = gtx.Dp(unit.Dp(28))
				l := a.Th.LabelColor(fs, unit.Sp(12), a.Th.TextMuted, formatAge(is.CreatedAt))
				l.Alignment = text.End
				return l.Layout(gtx)
			}),
		)
	})
}

func (a *App) drawIssueRowFull(gtx layout.Context, is linear.Issue) layout.Dimensions {
	fs := a.Th.Fonts.IssueList
	return layout.Inset{
		Top: unit.Dp(12), Bottom: unit.Dp(12),
		Left: unit.Dp(16), Right: unit.Dp(16),
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(priorityChip(a.Th, fs, is.Priority)),
					layout.Rigid(hSpace(12)),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						l := a.Th.LabelColor(fs, unit.Sp(14), a.Th.TextDim, is.Identifier)
						l.Font.Weight = 700
						l.MaxLines = 1
						return l.Layout(gtx)
					}),
					layout.Rigid(hSpace(16)),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						l := a.Th.LabelColor(fs, unit.Sp(14), a.Th.Text, is.Title)
						l.MaxLines = 1
						return l.Layout(gtx)
					}),
				)
			}),
			layout.Rigid(rigidSpace(8)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(statusBadge(a.Th, fs, is.State)),
					layout.Rigid(hSpace(16)),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						who := "Unassigned"
						if is.Assignee != nil {
							who = "@" + is.Assignee.Name
						}
						return dimLabel(gtx, a.Th, fs, unit.Sp(12), a.Th.TextDim, who)
					}),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						parts := []string{}
						if is.Project != nil {
							parts = append(parts, "▶ "+cleanProjectName(is.Project.Name))
						}
						labels := labelString(is.Labels.Nodes, 4)
						if labels != "" {
							parts = append(parts, labels)
						}
						parts = append(parts, formatAge(is.CreatedAt))
						txt := "  ·  " + strings.Join(parts, "  ·  ")
						return dimLabel(gtx, a.Th, fs, unit.Sp(12), a.Th.TextMuted, txt)
					}),
				)
			}),
		)
	})
}

// dimLabel renders a small label in the supplied color.
func dimLabel(gtx layout.Context, th *ui.Theme, fs ui.FontStyle, sz unit.Sp, c color.NRGBA, s string) layout.Dimensions {
	l := th.LabelColor(fs, sz, c, s)
	l.MaxLines = 1
	return l.Layout(gtx)
}

// priorityChip returns a small colored pill indicating priority.
func priorityChip(th *ui.Theme, fs ui.FontStyle, p int) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		var s string
		switch p {
		case 1:
			s = "!"
		case 2:
			s = "▆▆▆"
		case 3:
			s = "▆▆▁"
		case 4:
			s = "▆▁▁"
		default:
			s = "–"
		}
		l := th.LabelColor(fs, unit.Sp(11), th.PriorityColor(p), s)
		l.Font.Weight = 700
		return l.Layout(gtx)
	}
}

func statusDot(th *ui.Theme, stateType string) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		sz := gtx.Dp(unit.Dp(8))
		r := image.Rect(0, 0, sz, sz)
		defer clip.Ellipse(r).Push(gtx.Ops).Pop()
		rect(gtx, r, th.StatusColor(stateType))
		return layout.Dimensions{Size: image.Pt(sz, sz)}
	}
}

func statusBadge(th *ui.Theme, fs ui.FontStyle, st linear.WorkflowState) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(statusDot(th, st.Type)),
			layout.Rigid(hSpace(6)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return th.LabelColor(fs, unit.Sp(11), th.StatusColor(st.Type), st.Name).Layout(gtx)
			}),
		)
	}
}

func labelString(labels []linear.Label, max int) string {
	if len(labels) == 0 {
		return ""
	}
	parts := []string{}
	for i, l := range labels {
		if i >= max {
			parts = append(parts, fmt.Sprintf("+%d", len(labels)-i))
			break
		}
		name := l.Name
		r := []rune(name)
		if len(r) > 4 {
			name = string(r[:4])
		}
		parts = append(parts, "● "+name)
	}
	return strings.Join(parts, " ")
}

// formatAge mirrors the lazylinear duration formatting.
func formatAge(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dw", int(d.Hours()/(24*7)))
	case d < 365*24*time.Hour:
		return fmt.Sprintf("%dmo", int(d.Hours()/(24*30)))
	default:
		return fmt.Sprintf("%dy", int(d.Hours()/(24*365)))
	}
}

// widgetBox wraps w in a bordered rectangle. The box is sized to the larger
// of the inner content and the incoming Min constraints, so it fills the
// available width when callers set Min.X = Max.X.
func widgetBox(gtx layout.Context, th *ui.Theme, w layout.Widget) layout.Dimensions {
	macro := op.Record(gtx.Ops)
	contentDims := layout.Inset{
		Top: unit.Dp(4), Bottom: unit.Dp(4),
		Left: unit.Dp(8), Right: unit.Dp(8),
	}.Layout(gtx, w)
	content := macro.Stop()

	sz := contentDims.Size
	if gtx.Constraints.Min.X > sz.X {
		sz.X = gtx.Constraints.Min.X
	}
	if gtx.Constraints.Min.Y > sz.Y {
		sz.Y = gtx.Constraints.Min.Y
	}
	r := image.Rect(0, 0, sz.X, sz.Y)
	rect(gtx, r, th.PanelAlt)
	rectStroke(gtx, r, th.Border)
	content.Add(gtx.Ops)
	return layout.Dimensions{Size: sz}
}

// --- Issue detail ---

func cardBox(gtx layout.Context, th *ui.Theme, w layout.Widget) layout.Dimensions {
	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			r := image.Rect(0, 0, gtx.Constraints.Min.X, gtx.Constraints.Min.Y)
			rect(gtx, r, th.PanelAlt)
			rectStroke(gtx, r, th.Border)
			return layout.Dimensions{Size: gtx.Constraints.Min}
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{
				Top: unit.Dp(16), Bottom: unit.Dp(16),
				Left: unit.Dp(20), Right: unit.Dp(20),
			}.Layout(gtx, w)
		}),
	)
}

func (a *App) layoutIssueDetail(gtx layout.Context) layout.Dimensions {
	is := a.State.Detail
	if is == nil {
		return layout.Dimensions{}
	}
	th := a.Th
	fs := th.Fonts.IssueDetail

	return layout.UniformInset(unit.Dp(24)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return material.List(th.M, &a.detailList).Layout(gtx, 1, func(gtx layout.Context, _ int) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							l := th.LabelColor(fs, unit.Sp(14), th.TextDim, is.Identifier)
							l.Font.Weight = 700
							return l.Layout(gtx)
						}),
						layout.Rigid(hSpace(16)),
						layout.Rigid(statusBadge(th, fs, is.State)),
						layout.Rigid(hSpace(12)),
						layout.Rigid(priorityChip(th, fs, is.Priority)),
					)
				}),
				layout.Rigid(rigidSpace(8)),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					l := th.LabelColor(fs, unit.Sp(24), th.Text, is.Title)
					l.Font.Weight = 700
					return l.Layout(gtx)
				}),
				layout.Rigid(rigidSpace(24)),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return cardBox(gtx, th, func(gtx layout.Context) layout.Dimensions {
						return a.layoutDetailMeta(gtx, *is)
					})
				}),
				layout.Rigid(rigidSpace(32)),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					l := th.LabelColor(fs, unit.Sp(16), th.Text, "Description")
					l.Font.Weight = 700
					return l.Layout(gtx)
				}),
				layout.Rigid(rigidSpace(12)),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if is.Description == "" {
						return th.LabelColor(fs, unit.Sp(14), th.TextMuted, "No description provided.").Layout(gtx)
					}
					return th.LabelColor(fs, unit.Sp(14), th.TextDim, is.Description).Layout(gtx)
				}),
				layout.Rigid(rigidSpace(32)),
			)
		})
	})
}

func (a *App) layoutDetailMeta(gtx layout.Context, is linear.Issue) layout.Dimensions {
	th := a.Th
	fs := th.Fonts.IssueDetail

	row := func(label, value string, valueColor color.NRGBA) layout.Widget {
		return func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					// Enforce a fixed width for the label column to act like a table
					gtx.Constraints.Min.X = gtx.Dp(unit.Dp(100))
					gtx.Constraints.Max.X = gtx.Dp(unit.Dp(100))
					l := th.LabelColor(fs, unit.Sp(13), th.TextMuted, label)
					l.Font.Weight = 600
					return l.Layout(gtx)
				}),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return th.LabelColor(fs, unit.Sp(13), valueColor, value).Layout(gtx)
				}),
			)
		}
	}

	assignee := "Unassigned"
	if is.Assignee != nil {
		assignee = is.Assignee.Name
	}
	project := "—"
	if is.Project != nil {
		project = cleanProjectName(is.Project.Name)
	}
	labels := "None"
	if len(is.Labels.Nodes) > 0 {
		ns := make([]string, len(is.Labels.Nodes))
		for i, l := range is.Labels.Nodes {
			ns[i] = l.Name
		}
		labels = strings.Join(ns, ", ")
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(row("Assignee", assignee, th.Text)),
		layout.Rigid(rigidSpace(8)),
		layout.Rigid(row("Project", project, th.AccentDim)),
		layout.Rigid(rigidSpace(8)),
		layout.Rigid(row("Labels", labels, th.Text)),
		layout.Rigid(rigidSpace(8)),
		layout.Rigid(row("Created", is.CreatedAt.Format("2006-01-02 15:04"), th.TextDim)),
		layout.Rigid(rigidSpace(8)),
		layout.Rigid(row("Updated", is.UpdatedAt.Format("2006-01-02 15:04"), th.TextDim)),
		layout.Rigid(rigidSpace(8)),
		layout.Rigid(row("URL", is.URL, th.AccentDim)),
	)
}
