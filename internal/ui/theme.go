package ui

import (
	"image/color"

	"gioui.org/font/gofont"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget/material"
)

// Theme holds shared colors and the material theme.
type Theme struct {
	M *material.Theme

	BG        color.NRGBA
	Panel     color.NRGBA
	PanelAlt  color.NRGBA
	Border    color.NRGBA
	BorderHi  color.NRGBA
	Text      color.NRGBA
	TextDim   color.NRGBA
	TextMuted color.NRGBA
	Accent    color.NRGBA // #7D56F4 — Linear purple
	AccentDim color.NRGBA
	Selected  color.NRGBA // #5D4399
	Success   color.NRGBA
	Warning   color.NRGBA
	Error     color.NRGBA

	StatusBacklog   color.NRGBA
	StatusUnstarted color.NRGBA
	StatusStarted   color.NRGBA
	StatusCompleted color.NRGBA
	StatusCanceled  color.NRGBA
	StatusTriage    color.NRGBA

	PrioUrgent color.NRGBA
	PrioHigh   color.NRGBA
	PrioMedium color.NRGBA
	PrioLow    color.NRGBA
	PrioNone   color.NRGBA

	// Fonts holds the live per-section overrides (face + size). The settings
	// modal mutates these; layout helpers read them every frame.
	Fonts SectionFonts

	// Faces is the deduplicated list of typeface names known to the shaper.
	// MonoFaces is the subset matching "mono"; used by the Code section.
	Faces     []string
	MonoFaces []string
}

// New builds the default dark theme.
func New() *Theme {
	m := material.NewTheme()
	m.TextSize = unit.Sp(DefaultBaseSp)

	collection := gofont.Collection()
	collection = append(collection, loadSystemFaces()...)
	m.Shaper = text.NewShaper(text.WithCollection(collection))

	faces, monoFaces := uniqueFaces(collection)
	monoDefault := ""
	for _, want := range []string{"JetBrains Mono", "IBM Plex Mono", "Go Mono"} {
		for _, f := range monoFaces {
			if f == want {
				monoDefault = want
				break
			}
		}
		if monoDefault != "" {
			break
		}
	}

	t := &Theme{
		M:               m,
		BG:              rgb(0x141417),
		Panel:           rgb(0x1B1B20),
		PanelAlt:        rgb(0x232329),
		Border:          rgb(0x2C2C33),
		BorderHi:        rgb(0x4A3D80),
		Text:            rgb(0xEDEDED),
		TextDim:         rgb(0xAAAAAA),
		TextMuted:       rgb(0x666666),
		Accent:          rgb(0x7D56F4),
		AccentDim:       rgb(0xA885FF),
		Selected:        rgb(0x5D4399),
		Success:         rgb(0x4ADE80),
		Warning:         rgb(0xF2C94C),
		Error:           rgb(0xEB5757),
		StatusBacklog:   rgb(0x95A2B3),
		StatusUnstarted: rgb(0xE2E2E2),
		StatusStarted:   rgb(0xF2C94C),
		StatusCompleted: rgb(0x5E6AD2),
		StatusCanceled:  rgb(0x95A2B3),
		StatusTriage:    rgb(0xEB5757),
		PrioUrgent:      rgb(0xEB5757),
		PrioHigh:        rgb(0xF2994A),
		PrioMedium:      rgb(0xF2C94C),
		PrioLow:         rgb(0x888888),
		PrioNone:        rgb(0x444444),

		Faces:     faces,
		MonoFaces: monoFaces,
		Fonts: SectionFonts{
			Code: FontStyle{Face: monoDefault},
		},
	}
	m.Bg = t.BG
	m.Fg = t.Text
	m.ContrastBg = t.Accent
	m.ContrastFg = t.Text
	return t
}

func rgb(v uint32) color.NRGBA {
	return color.NRGBA{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v), A: 0xFF}
}

// StatusColor returns the color for a workflow state type.
func (t *Theme) StatusColor(stateType string) color.NRGBA {
	switch stateType {
	case "backlog":
		return t.StatusBacklog
	case "unstarted":
		return t.StatusUnstarted
	case "started":
		return t.StatusStarted
	case "completed":
		return t.StatusCompleted
	case "canceled":
		return t.StatusCanceled
	case "triage":
		return t.StatusTriage
	}
	return t.TextDim
}

// PriorityLabel returns the human-readable name for a priority value.
func PriorityLabel(p int) string {
	switch p {
	case 1:
		return "Urgent"
	case 2:
		return "High"
	case 3:
		return "Medium"
	case 4:
		return "Low"
	}
	return "None"
}

// PriorityColor returns the color for a priority value.
func (t *Theme) PriorityColor(p int) color.NRGBA {
	switch p {
	case 1:
		return t.PrioUrgent
	case 2:
		return t.PrioHigh
	case 3:
		return t.PrioMedium
	case 4:
		return t.PrioLow
	}
	return t.PrioNone
}
