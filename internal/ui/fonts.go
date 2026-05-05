package ui

import (
	"image/color"
	"log/slog"
	"os"
	"sort"
	"strings"

	"gioui.org/font"
	"gioui.org/font/opentype"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

// DefaultBaseSp is the body-text anchor used to scale per-section sizes.
// All hardcoded unit.Sp(N) call sites are interpreted relative to this.
const DefaultBaseSp = 13.0

// FontStyle is a per-section override. Empty Face / zero Size means
// "fall through to the global default, then the theme default."
type FontStyle struct {
	Face string  `json:"face,omitempty"`
	Size float32 `json:"size,omitempty"`
}

// SectionFonts is the live, mutable set of overrides used during layout.
// The settings screen mutates these in place.
type SectionFonts struct {
	Global      FontStyle
	Sidebar     FontStyle
	IssueList   FontStyle
	IssueDetail FontStyle
	StatusBar   FontStyle
	Modal       FontStyle
	Code        FontStyle
}

// Sp scales a "natural" base size by the section's body size override.
// If the section has no size override, we fall back to the global override,
// and finally to base unchanged.
func (t *Theme) Sp(s FontStyle, base unit.Sp) unit.Sp {
	size := s.Size
	if size == 0 {
		size = t.Fonts.Global.Size
	}
	if size <= 0 {
		return base
	}
	scale := size / DefaultBaseSp
	return unit.Sp(float32(base) * scale)
}

// Face resolves the typeface for a section, with global + theme fallback.
func (t *Theme) Face(s FontStyle) font.Typeface {
	if s.Face != "" {
		return font.Typeface(s.Face)
	}
	if t.Fonts.Global.Face != "" {
		return font.Typeface(t.Fonts.Global.Face)
	}
	return ""
}

// Label builds a styled material label honoring the section's face/size.
// Callers pass the existing "natural" size (e.g. unit.Sp(13)).
func (t *Theme) Label(s FontStyle, base unit.Sp, txt string) material.LabelStyle {
	l := material.Label(t.M, t.Sp(s, base), txt)
	l.Color = t.Text
	if face := t.Face(s); face != "" {
		l.Font.Typeface = face
	}
	return l
}

// LabelColor is Label with an explicit color set.
func (t *Theme) LabelColor(s FontStyle, base unit.Sp, c color.NRGBA, txt string) material.LabelStyle {
	l := t.Label(s, base, txt)
	l.Color = c
	return l
}

// Editor builds a styled material editor honoring the section's face/size.
func (t *Theme) Editor(s FontStyle, ed *widget.Editor, hint string, c color.NRGBA) material.EditorStyle {
	e := material.Editor(t.M, ed, hint)
	e.TextSize = t.Sp(s, unit.Sp(DefaultBaseSp))
	e.Color = c
	e.HintColor = color.NRGBA{R: 0x66, G: 0x66, B: 0x66, A: 0xFF}
	if face := t.Face(s); face != "" {
		e.Font.Typeface = face
	}
	return e
}

// ApplyFontPrefs overlays user-saved prefs into the theme. Zero fields are
// ignored so users can leave any section at "default".
func (t *Theme) ApplyFontPrefs(p SectionFonts) {
	merge := func(target *FontStyle, pref FontStyle) {
		if pref.Face != "" {
			target.Face = pref.Face
		}
		if pref.Size > 0 {
			target.Size = pref.Size
		}
	}
	merge(&t.Fonts.Global, p.Global)
	merge(&t.Fonts.Sidebar, p.Sidebar)
	merge(&t.Fonts.IssueList, p.IssueList)
	merge(&t.Fonts.IssueDetail, p.IssueDetail)
	merge(&t.Fonts.StatusBar, p.StatusBar)
	merge(&t.Fonts.Modal, p.Modal)
	merge(&t.Fonts.Code, p.Code)
}

// loadSystemFaces tries the usual locations for sharper UI fonts and
// returns whatever it can parse.
func loadSystemFaces() []font.FontFace {
	candidates := []string{
		"/usr/share/fonts/inter/Inter-Regular.otf",
		"/usr/share/fonts/inter/Inter-Bold.otf",
		"/usr/share/fonts/inter/Inter-Italic.otf",
		"/usr/share/fonts/TTF/Inter-Regular.ttf",
		"/usr/share/fonts/TTF/Inter-Bold.ttf",
		"/usr/share/fonts/TTF/Inter-Italic.ttf",
		"/usr/share/fonts/truetype/inter/Inter-Regular.ttf",
		"/usr/share/fonts/truetype/inter/Inter-Bold.ttf",
		"/usr/share/fonts/truetype/inter/Inter-Italic.ttf",
		"/usr/share/fonts/ibm-plex/IBMPlexSans-Regular.otf",
		"/usr/share/fonts/ibm-plex/IBMPlexSans-Bold.otf",
		"/usr/share/fonts/ibm-plex/IBMPlexSans-Italic.otf",
		"/usr/share/fonts/TTF/IBMPlexSans-Regular.ttf",
		"/usr/share/fonts/jetbrains-mono/JetBrainsMono-Regular.ttf",
		"/usr/share/fonts/TTF/JetBrainsMono-Regular.ttf",
		"/usr/share/fonts/truetype/jetbrains-mono/JetBrainsMono-Regular.ttf",
		"/usr/share/fonts/ibm-plex/IBMPlexMono-Regular.otf",
	}
	var out []font.FontFace
	for _, p := range candidates {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		faces, err := opentype.ParseCollection(data)
		if err != nil {
			slog.Warn("ui font parse failed", "path", p, "error", err)
			continue
		}
		out = append(out, faces...)
	}
	return out
}

// uniqueFaces collapses a font collection into a sorted, deduplicated
// list of typeface names; mono is the subset whose name contains "mono".
func uniqueFaces(coll []font.FontFace) (faces []string, mono []string) {
	seen := map[string]bool{}
	for _, f := range coll {
		name := string(f.Font.Typeface)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		faces = append(faces, name)
	}
	sort.Strings(faces)
	for _, name := range faces {
		if strings.Contains(strings.ToLower(name), "mono") {
			mono = append(mono, name)
		}
	}
	if len(mono) == 0 {
		mono = faces
	}
	return faces, mono
}
