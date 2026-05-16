package app

import (
	"image"
	"strconv"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

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
				return a.layoutLoggingRow(gtx, m)
			}),
			layout.Rigid(rigidSpace(8)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.layoutHintsRow(gtx, m)
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
	if minH := gtx.Dp(unit.Dp(28)); h < minH {
		h = minH
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

// layoutLoggingRow renders a toggle for debug/info logs.
func (a *App) layoutLoggingRow(gtx layout.Context, m *SettingsModal) layout.Dimensions {
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
						l := a.Th.LabelColor(a.Th.Fonts.Modal, unit.Sp(13), a.Th.Text, "Debug / Info Logs")
						l.Font.Weight = 600
						return l.Layout(gtx)
					}),
				)
			}),
			layout.Rigid(rigidSpace(8)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.surfaceRow(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return a.Th.LabelColor(a.Th.Fonts.Modal, unit.Sp(12), a.Th.TextDim, "Writes to stderr when enabled").Layout(gtx)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return material.CheckBox(a.Th.M, &m.LogToggle, "").Layout(gtx)
						}),
					)
				})
			}),
		)
	})
}

// layoutHintsRow renders a toggle for the bottom help/hints bar.
func (a *App) layoutHintsRow(gtx layout.Context, m *SettingsModal) layout.Dimensions {
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
						l := a.Th.LabelColor(a.Th.Fonts.Modal, unit.Sp(13), a.Th.Text, "Bottom Help Bar")
						l.Font.Weight = 600
						return l.Layout(gtx)
					}),
				)
			}),
			layout.Rigid(rigidSpace(8)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.surfaceRow(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return a.Th.LabelColor(a.Th.Fonts.Modal, unit.Sp(12), a.Th.TextDim, "Show keyboard shortcuts bar").Layout(gtx)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return material.CheckBox(a.Th.M, &m.HintsToggle, "").Layout(gtx)
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
	if m.LogToggle.Update(gtx) {
		a.State.Saved.EnableLogging = m.LogToggle.Value
		a.syncLogging()
		a.saveState()
	}
	if m.HintsToggle.Update(gtx) {
		a.State.HideHints = !m.HintsToggle.Value
		a.saveState()
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
