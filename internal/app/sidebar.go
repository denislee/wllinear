package app

import (
	"fmt"
	"image"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/denislee/wllinear/internal/ui"
)

func (a *App) layoutSidebar(gtx layout.Context, r image.Rectangle) {
	defer op.Offset(r.Min).Push(gtx.Ops).Pop()
	w, h := r.Dx(), r.Dy()
	rect(gtx, image.Rect(0, 0, w, h), a.Th.Panel)
	rect(gtx, image.Rect(w-gtx.Dp(unit.Dp(1)), 0, w, h), a.Th.Border)

	gtx.Constraints = layout.Constraints{Min: image.Point{}, Max: image.Pt(w, h)}

	layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		// Build a flat list of "rows" so the entire sidebar scrolls as one
		// unit when the workspace has many teams/filters/projects.
		rows := a.buildSidebarRows()
		return material.List(a.Th.M, &a.listSidebar).Layout(gtx, len(rows), func(gtx layout.Context, idx int) layout.Dimensions {
			return rows[idx](gtx)
		})
	})
}

// sidebarRow is a self-contained renderer for one row in the unified sidebar
// list. Each entry encloses its own click target and the App theme.
type sidebarRow = layout.Widget

func (a *App) buildSidebarRows() []sidebarRow {
	st := a.State
	rows := []sidebarRow{}

	rows = append(rows, sectionRow(a.Th, a.Th.Fonts.Sidebar, "FILTERS"))
	rows = append(rows, gapRow(2))

	// FILTERS section.
	if cap(a.filterClicks) < len(st.Filters) {
		a.filterClicks = make([]widget.Clickable, len(st.Filters))
	}
	a.filterClicks = a.filterClicks[:len(st.Filters)]
	for i := range st.Filters {
		i := i
		f := st.Filters[i]
		if f == "---" {
			rows = append(rows, dividerRow(a.Th))
			continue
		}
		click := &a.filterClicks[i]
		rows = append(rows, func(gtx layout.Context) layout.Dimensions {
			if click.Clicked(gtx) {
				st.PostEvent(FilterSelected{Filter: f})
			}
			count := ""
			if c, ok := st.FilterCounts[f]; ok && c > 0 {
				count = fmt.Sprintf("%d", c)
			}
			selected := st.ActiveFilter == f
			return drawRow(gtx, a.Th, a.Th.Fonts.Sidebar, click, selected, f, count)
		})
	}

	// MY PROJECTS section.
	if len(st.LeadingProjects) > 0 {
		rows = append(rows, gapRow(10))
		rows = append(rows, sectionRow(a.Th, a.Th.Fonts.Sidebar, "MY PROJECTS"))
		rows = append(rows, gapRow(2))
		if cap(a.leadingClicks) < len(st.LeadingProjects) {
			a.leadingClicks = make([]widget.Clickable, len(st.LeadingProjects))
		}
		a.leadingClicks = a.leadingClicks[:len(st.LeadingProjects)]
		for i := range st.LeadingProjects {
			i := i
			p := st.LeadingProjects[i]
			click := &a.leadingClicks[i]
			rows = append(rows, func(gtx layout.Context) layout.Dimensions {
				if click.Clicked(gtx) {
					st.PostEvent(ProjectSelected{Project: p})
					go CopyProjectLastCycle(st, p)
				}
				selected := st.ActiveFilter == "Project: "+p.Name
				return drawRow(gtx, a.Th, a.Th.Fonts.Sidebar, click, selected, "▶ "+cleanProjectName(p.Name), "")
			})
		}
	}

	return rows
}

func gapRow(dp int) sidebarRow {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Spacer{Height: unit.Dp(float32(dp))}.Layout(gtx)
	}
}

// rigidSpace returns a vertical spacer suitable for layout.Rigid.
func rigidSpace(dp int) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Spacer{Height: unit.Dp(float32(dp))}.Layout(gtx)
	}
}

func sectionRow(th *ui.Theme, fs ui.FontStyle, s string) sidebarRow {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Left: unit.Dp(4), Top: unit.Dp(2), Bottom: unit.Dp(2)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			l := th.LabelColor(fs, unit.Sp(11), th.TextMuted, s)
			l.Font.Weight = 700
			l.MaxLines = 1
			return l.Layout(gtx)
		})
	}
}

func dimRow(th *ui.Theme, fs ui.FontStyle, s string) sidebarRow {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Left: unit.Dp(8), Top: unit.Dp(2), Bottom: unit.Dp(2)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			l := th.LabelColor(fs, unit.Sp(12), th.TextMuted, s)
			l.MaxLines = 1
			return l.Layout(gtx)
		})
	}
}

func dividerRow(th *ui.Theme) sidebarRow {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: unit.Dp(4), Bottom: unit.Dp(4), Left: unit.Dp(4), Right: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			h := gtx.Dp(unit.Dp(1))
			rect(gtx, image.Rect(0, 0, gtx.Constraints.Max.X, h), th.Border)
			return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, h)}
		})
	}
}

// drawRow renders a clickable row with an optional right-aligned count.
// We record the content, measure its natural height, then paint the
// background at that exact size and play the content back on top — this
// avoids the Stack/Expanded sizing pitfalls we hit using Gio's Stack.
func drawRow(gtx layout.Context, th *ui.Theme, fs ui.FontStyle, click *widget.Clickable, selected bool, label, right string) layout.Dimensions {
	return click.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		macro := op.Record(gtx.Ops)
		dims := layout.Inset{
			Top: unit.Dp(3), Bottom: unit.Dp(3),
			Left: unit.Dp(8), Right: unit.Dp(8),
		}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					col := th.Text
					if !selected {
						col = th.TextDim
					}
					l := th.LabelColor(fs, unit.Sp(13), col, label)
					l.MaxLines = 1
					return l.Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if right == "" {
						return layout.Dimensions{}
					}
					c := th.TextMuted
					if selected {
						c = th.Text
					}
					return th.LabelColor(fs, unit.Sp(11), c, right).Layout(gtx)
				}),
			)
		})
		content := macro.Stop()

		size := image.Pt(gtx.Constraints.Max.X, dims.Size.Y)
		if selected {
			rect(gtx, image.Rect(0, 0, size.X, size.Y), th.Selected)
		} else if click.Hovered() {
			rect(gtx, image.Rect(0, 0, size.X, size.Y), th.PanelAlt)
		}
		content.Add(gtx.Ops)
		return layout.Dimensions{Size: size}
	})
}

func drawDimText(gtx layout.Context, th *ui.Theme, fs ui.FontStyle, s string) layout.Dimensions {
	return layout.Inset{Left: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		l := th.LabelColor(fs, unit.Sp(12), th.TextMuted, s)
		l.MaxLines = 1
		return l.Layout(gtx)
	})
}
