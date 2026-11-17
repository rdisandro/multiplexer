package main

import (
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
)

func srgbLinear(c uint8) float64 {
	v := float64(c) / 255.0
	if v <= 0.04045 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}

func relLuminance(c ui.Color) float64 {
	return 0.2126*srgbLinear(c.R) + 0.7152*srgbLinear(c.G) + 0.0722*srgbLinear(c.B)
}

func contrastRatio(fg, bg ui.Color) float64 {
	if fg.A < 255 {
		fg = fg.Over(bg)
	}
	l1, l2 := relLuminance(fg), relLuminance(bg)
	if l1 < l2 {
		l1, l2 = l2, l1
	}
	return (l1 + 0.05) / (l2 + 0.05)
}

// TestNearAITokenContrast verifies the WCAG 2.x contrast floors mandated by
// Near AI OS Design System 02-color.md & 22-implementation.md across all 6 themes:
//   - >= 4.5:1 text contrast in standard themes (>= 7.0:1 in -hc) over opaque
//     grounds and worst-case desktop wallpapers behind glass-veil.
//   - >= 4.5:1 text-on-accent on accent, accent-pressed, and menu-hover.
//   - >= 3.0:1 non-text contrast for focus-ring and semantic status dots.
func TestNearAITokenContrast(t *testing.T) {
	for _, tk := range AllThemes() {
		t.Run(string(tk.Name), func(t *testing.T) {
			textFloor := 4.5
			if tk.HC {
				textFloor = 7.0
			}

			// Worst-case desktop wallpapers behind native vibrancy (22-implementation.md):
			// pure white (#ffffff) and dark charcoal (#333333 / #000000).
			wallpapers := []ui.Color{ui.Hex("#ffffff"), ui.Hex("#333333")}
			if tk.Dark {
				wallpapers = []ui.Color{ui.Hex("#ffffff"), ui.Hex("#000000")}
			}

			grounds := []struct {
				name string
				col  ui.Color
			}{
				{"paneOpaque", tk.paneOpaque},
				{"cardOnOpaque", tk.cardOnOpaque},
				{"cardQuietOnOpaque", tk.cardQuietOnOpaque},
			}
			for i, wp := range wallpapers {
				pane := tk.paneFill.Over(tk.glassVeil.Over(wp))
				card := tk.card.Over(pane)
				grounds = append(grounds,
					struct {
						name string
						col  ui.Color
					}{"glassPane", pane},
					struct {
						name string
						col  ui.Color
					}{"glassCard", card},
				)
				_ = i
			}

			inks := []struct {
				name  string
				col   ui.Color
				floor float64
			}{
				{"textPrimary", tk.textPrimary, textFloor},
				{"textSecondary", tk.textSecondary, textFloor},
				{"textTertiary", tk.textTertiary, textFloor},
				{"accentText", tk.accentText, textFloor},
			}

			for _, g := range grounds {
				for _, ink := range inks {
					got := contrastRatio(ink.col, g.col)
					// Allow tiny 8-bit quantization tolerance (0.08).
					if got+0.08 < ink.floor {
						t.Errorf("%s on %s: contrast %.2f < %.1f", ink.name, g.name, got, ink.floor)
					}
				}
			}

			// text-on-accent over accent, accent-pressed, and menu-hover over menu.
			accentGrounds := []struct {
				name string
				col  ui.Color
			}{
				{"accent", tk.accent},
				{"accentPressed", tk.accentPressed},
				{"menuHover", tk.menuHover.Over(tk.menu.Over(tk.paneOpaque))},
			}
			for _, ag := range accentGrounds {
				if got := contrastRatio(tk.textOnAccent, ag.col); got+0.08 < 4.5 {
					t.Errorf("textOnAccent on %s: contrast %.2f < 4.5", ag.name, got)
				}
			}

			// Non-text 3:1 marks against cardOnOpaque.
			marks := []struct {
				name string
				col  ui.Color
			}{
				{"focusRing", tk.focusRing},
				{"positive", tk.positive},
				{"caution", tk.caution},
				{"critical", tk.critical},
			}
			for _, m := range marks {
				if got := contrastRatio(m.col, tk.cardOnOpaque); got+0.05 < 3.0 {
					t.Errorf("%s mark on cardOnOpaque: contrast %.2f < 3.0", m.name, got)
				}
			}
		})
	}
}

// TestToolTileContrast checks that every foreground program tint in programs.go
// holds >= 3.0:1 against the flat neutral-tint tool tile in both light and dark.
func TestToolTileContrast(t *testing.T) {
	for _, tk := range []*Tokens{&tokensLight, &tokensDark} {
		ts := &themeState{Tokens: tk}
		tileBg := tk.neutralTint.Over(tk.cardOnOpaque)
		for name, prog := range programs {
			tint := prog.glyphTint(ts)
			if got := contrastRatio(tint, tileBg); got+0.05 < 3.0 {
				t.Errorf("%s (%s): tool tile tint contrast %.2f < 3.0", tk.Name, name, got)
			}
		}
		// Shell & fallback data[0..7] tints.
		if got := contrastRatio(shellTile.glyphTint(ts), tileBg); got+0.05 < 3.0 {
			t.Errorf("%s (shell): tool tile tint contrast %.2f < 3.0", tk.Name, got)
		}
	}
}

func TestThemeResolution(t *testing.T) {
	cases := []struct {
		dark, hc bool
		product  string
		want     ThemeName
	}{
		{false, false, "", ThemeLight},
		{true, false, "", ThemeDark},
		{false, true, "", ThemeLightHC},
		{true, true, "", ThemeDarkHC},
		{false, false, "trace", ThemeTraceLight},
		{true, false, "trace", ThemeTraceDark},
		// Trace & Classic share light-hc and dark-hc when increased contrast applies (01-architecture.md).
		{false, true, "trace", ThemeLightHC},
		{true, true, "trace", ThemeDarkHC},
		{false, false, "classic", ThemeClassicLight},
		{true, false, "classic", ThemeClassicDark},
		{false, true, "classic", ThemeLightHC},
		{true, true, "classic", ThemeDarkHC},
	}
	for _, tc := range cases {
		if got := resolveThemeName(tc.dark, tc.hc, tc.product); got != tc.want {
			t.Errorf("resolveThemeName(dark=%v, hc=%v, product=%q) = %q, want %q",
				tc.dark, tc.hc, tc.product, got, tc.want)
		}
	}
}
