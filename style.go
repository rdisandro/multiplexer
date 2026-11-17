package main

import (
	_ "embed"

	"github.com/egoist/mygo/plugins/glass"
	"github.com/egoist/mygo/plugins/terminal"
	"github.com/egoist/mygo/ui"
)

// paintSceneBackdrop paints the Near AI OS .nr-scene atmospheric backdrop
// when native macOS NSGlassEffectView is not active (Linux/Windows or offscreen
// screenshot capture), giving Liquid Glass panes a rich desktop field to refract.
func paintSceneBackdrop(p *ui.Painter, r ui.Rect, ts *themeState) {
	if ts.ReduceTransparency {
		p.Fill(r, ts.paneOpaque, 0)
		return
	}
	if ts.NativeGlass {
		return
	}
	p.FillGradient(r, ui.LinearGradient{From: ts.sceneWarm, To: ts.scene, Angle: 160}, 0)
	g1 := ts.sceneGlow1.Alpha(0.46)
	g2 := ts.sceneGlow2.Alpha(0.36)
	g3 := ts.sceneGlow3.Alpha(0.28)
	p.FillGradient(r, ui.LinearGradient{From: g1, To: g1.Alpha(0), Angle: 135, End: 0.65}, 0)
	p.FillGradient(r, ui.LinearGradient{From: g2.Alpha(0), To: g2, Angle: 135, Start: 0.35, End: 1}, 0)
	p.FillGradient(r, ui.LinearGradient{From: g3.Alpha(0), To: g3, Angle: 45, Start: 0.22, End: 0.88}, 0)
}

// styleGlassPane applies TCDesign's GlassPane material & edge treatment
// (Surfaces.swift + GlassBackdrop.swift):
//   - Live macOS: AppKit NSGlassEffectView (.regular) sits directly behind the pane's
//     20pt rounded rect; MyGo layers glassVeil + 160° paneFill gradient + specular rim.
//   - Non-macOS / offscreen capture: MyGo's macOS 27 Liquid Glass shader (glass.Regular)
//     refracts the scene backdrop under glassVeil + paneFill + shadow-float.
//   - Reduce Transparency: solid paneOpaque + paneFill gradient.
func styleGlassPane(e ui.Element, ts *themeState, focused bool) ui.Element {
	rad := ts.paneRadius()
	e.Radius(rad).Clip()
	if !ts.ReduceTransparency && !ts.NativeGlass {
		sh := ts.shadowFloat
		e.Shadow(0, sh.y, sh.blur, 0, sh.color)
		e.Material(glass.Glass{Style: glass.Regular, Tint: ts.glassTint})
	}
	e.Draw(func(p *ui.Painter, r ui.Rect) {
		if ts.ReduceTransparency {
			p.Fill(r, ts.paneOpaque, rad)
		} else {
			p.Fill(r, ts.glassVeil, rad)
			if ts.HC && !ts.Classic {
				p.Fill(r, ts.glassVeil.Alpha(0.34), rad)
			}
		}
		p.FillGradient(r, ui.LinearGradient{From: ts.paneFill, To: ts.paneFillEnd, Angle: 160}, rad)
		if !focused && ts.inactiveDim.A > 0 {
			p.Fill(r, ts.inactiveDim, rad)
		}
		if ts.Classic {
			// Classic paneEdge: 1pt top & left specular highlight + bottom shade.
			topA, sideA, botA := float32(0.30), float32(0.12), float32(0.20)
			if !ts.Dark {
				topA, sideA, botA = 0.60, 0.24, 0.09
			}
			p.Stroke(r, ui.RGBA(255, 255, 255, sideA), rad, 1)
			p.Fill(ui.Rect{X: r.X + rad*0.6, Y: r.Y, W: max(r.W-rad*1.2, 0), H: 1}, ui.RGBA(255, 255, 255, topA), 0)
			p.Fill(ui.Rect{X: r.X + rad*0.6, Y: r.Y + r.H - 1, W: max(r.W-rad*1.2, 0), H: 1}, ui.RGBA(0, 0, 0, botA), 0)
		} else if !ts.HC && !ts.ReduceTransparency {
			// Subtle Liquid Glass 1px rim gleam matching macOS 26/27 NSGlassEffectView bezel.
			rimA := float32(0.11)
			if !ts.Dark {
				rimA = 0.42
			}
			p.Stroke(r, ui.RGBA(255, 255, 255, rimA), rad, 1)
		}
	})
	return ts.hcEdge(e)
}

// styleGlassCard applies TCDesign's GlassCard / GlassQuietCard painted surface
// (Surfaces.swift: never glass-on-glass; 35%/28% translucent fill over glass,
// or solid cardOnOpaque inside opaque Settings/ReduceTransparency surfaces).
func styleGlassCard(e ui.Element, ts *themeState, quiet bool) ui.Element {
	rad := ts.cardRadius()
	if quiet {
		rad = ts.cardQuietRadius()
	}
	e.Radius(rad)
	if ts.Opaque {
		if quiet {
			e.Background(ts.cardQuietOnOpaque)
		} else {
			e.Background(ts.cardOnOpaque)
		}
		return ts.hcEdge(e)
	}
	if ts.Classic {
		from, to := ts.card, ts.cardEnd
		topA, botA := float32(0.20), float32(0.12)
		if quiet {
			from, to = ts.cardQuiet, ts.cardQuietEnd
			topA, botA = 0.18, 0.10
		}
		if !ts.Dark {
			topA *= 2
			botA *= 0.45
		}
		e.Draw(func(p *ui.Painter, r ui.Rect) {
			p.FillGradient(r, ui.LinearGradient{From: from, To: to, Angle: 180}, rad)
			p.Stroke(r, ui.RGBA(255, 255, 255, topA*0.5), rad, 1)
			p.Fill(ui.Rect{X: r.X + rad*0.5, Y: r.Y, W: max(r.W-rad, 0), H: 1}, ui.RGBA(255, 255, 255, topA), 0)
			p.Fill(ui.Rect{X: r.X + rad*0.5, Y: r.Y + r.H - 1, W: max(r.W-rad, 0), H: 1}, ui.RGBA(0, 0, 0, botA), 0)
		})
		return ts.hcEdge(e)
	}
	if quiet {
		e.Background(ts.cardQuiet)
	} else {
		e.Background(ts.card)
	}
	return ts.hcEdge(e)
}

//go:embed assets/fonts/JetBrainsMono-Regular.ttf
var fontRegular []byte

//go:embed assets/fonts/JetBrainsMono-Bold.ttf
var fontBold []byte

//go:embed assets/fonts/JetBrainsMono-Italic.ttf
var fontItalic []byte

//go:embed assets/fonts/JetBrainsMono-BoldItalic.ttf
var fontBoldItalic []byte

// registerFonts registers embedded JetBrains Mono as the cross-platform fallback;
// SF Pro and SF Mono are resolved directly from macOS Core Text and never bundled.
func registerFonts() {
	for _, f := range [][]byte{fontRegular, fontBold, fontItalic, fontBoldItalic} {
		ui.RegisterFont(f, "JetBrains Mono")
	}
}

var termFont = terminal.Font{
	Family:     "SF Mono, JetBrains Mono, Menlo, Consolas, monospace",
	Size:       12.0,
	LineHeight: 1.15,
}

// Terminal palettes harmonized with Near AI OS / TCDesign tokens:
// crisp text-primary ink, accent cursor/selection, and balanced semantic ANSI hues.
var lightTerm = &terminal.Theme{
	Foreground: ui.Hex("#1d1d1f"),
	Background: ui.Hex("#ffffff"),
	Cursor:     ui.Hex("#035ca3"),
	CursorText: ui.Hex("#ffffff"),
	Selection:  ui.RGBA(3, 92, 163, 0.20),
	Palette: [16]ui.Color{
		ui.Hex("#1d1d1f"), ui.Hex("#d0342c"), ui.Hex("#1a8f4c"), ui.Hex("#a8690f"),
		ui.Hex("#035ca3"), ui.Hex("#5a48b8"), ui.Hex("#14906a"), ui.Hex("#75757b"),
		ui.Hex("#5c5c63"), ui.Hex("#e04a42"), ui.Hex("#2f8a1f"), ui.Hex("#cf5320"),
		ui.Hex("#2a78d6"), ui.Hex("#c94a7c"), ui.Hex("#199e70"), ui.Hex("#3c3c43"),
	},
}

var darkTerm = &terminal.Theme{
	Foreground: ui.Hex("#f2f3f8"),
	Background: ui.Hex("#0d1124"),
	Cursor:     ui.Hex("#7ebcff"),
	CursorText: ui.Hex("#071a2e"),
	Selection:  ui.RGBA(126, 188, 255, 0.26),
	Palette: [16]ui.Color{
		ui.Hex("#232946"), ui.Hex("#ff6b6b"), ui.Hex("#30d158"), ui.Hex("#ffc75c"),
		ui.Hex("#7ebcff"), ui.Hex("#9085e9"), ui.Hex("#199e70"), ui.Hex("#d6d6dc"),
		ui.Hex("#8d8d97"), ui.Hex("#ff8787"), ui.Hex("#63e685"), ui.Hex("#ffd585"),
		ui.Hex("#9ecaff"), ui.Hex("#d55181"), ui.Hex("#38d9a9"), ui.Hex("#ffffff"),
	},
}

var lightHCTerm = &terminal.Theme{
	Foreground: ui.Hex("#1d1d1f"),
	Background: ui.Hex("#ffffff"),
	Cursor:     ui.Hex("#01599f"),
	CursorText: ui.Hex("#ffffff"),
	Selection:  ui.RGBA(1, 89, 159, 0.28),
	Palette: [16]ui.Color{
		ui.Hex("#1d1d1f"), ui.Hex("#b01218"), ui.Hex("#0b4628"), ui.Hex("#573403"),
		ui.Hex("#013e71"), ui.Hex("#43329e"), ui.Hex("#0b5c43"), ui.Hex("#2a2a2e"),
		ui.Hex("#3c3c43"), ui.Hex("#d0342c"), ui.Hex("#1a8f4c"), ui.Hex("#a8690f"),
		ui.Hex("#01599f"), ui.Hex("#a32e5f"), ui.Hex("#14906a"), ui.Hex("#1d1d1f"),
	},
}

var darkHCTerm = &terminal.Theme{
	Foreground: ui.Hex("#ffffff"),
	Background: ui.Hex("#0d1124"),
	Cursor:     ui.Hex("#7ebcff"),
	CursorText: ui.Hex("#071a2e"),
	Selection:  ui.RGBA(126, 188, 255, 0.34),
	Palette: [16]ui.Color{
		ui.Hex("#2a3154"), ui.Hex("#ff8080"), ui.Hex("#acedbc"), ui.Hex("#ffd585"),
		ui.Hex("#c6e1ff"), ui.Hex("#b8b0f5"), ui.Hex("#5ce6b8"), ui.Hex("#e6e6ec"),
		ui.Hex("#d6d6dc"), ui.Hex("#ffd3d3"), ui.Hex("#c2f5cf"), ui.Hex("#ffe4ad"),
		ui.Hex("#d9ecff"), ui.Hex("#f58cb2"), ui.Hex("#8cf0ce"), ui.Hex("#ffffff"),
	},
}

func activeTerminalThemes() (light, dark *terminal.Theme) {
	if prefs.HighContrast {
		return lightHCTerm, darkHCTerm
	}
	return lightTerm, darkTerm
}
