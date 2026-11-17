package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/terminal"
	"github.com/egoist/mygo/ui"

	"multiplexer/internal/rex"
)

// settings are the user's preferences, persisted immediately to settings.json
// beside the session server's state (no Save button, per 12-desktop.md & SettingsModal.swift).
type settings struct {
	// Appearance is "light", "dark", or "" to follow the system.
	Appearance string `json:"appearance,omitempty"`
	// ProductTheme is "" (NEAR AI Liquid Glass), "trace" (Trace), or "classic" (Classic Glass).
	ProductTheme string `json:"productTheme,omitempty"`
	// Density is "" (Compact desktop default) or "dense" (Dense chrome).
	Density string `json:"density,omitempty"`
	// ReduceTransparency replaces glass vibrancy and floating blurs with opaque surfaces.
	ReduceTransparency bool `json:"reduceTransparency,omitempty"`
	// HighContrast enables 7:1 ink contrast and 1pt edge-hc borders alongside OS settings.
	HighContrast bool `json:"highContrast,omitempty"`
	// FontSize is the terminal font size in points (8..32).
	FontSize float32 `json:"fontSize,omitempty"`
	// CursorStyle is "block", "bar", or "underline".
	CursorStyle string `json:"cursorStyle,omitempty"`
	// CursorBlink controls whether the terminal cursor blinks by default.
	CursorBlink bool `json:"cursorBlink"`
	// cursorBlinkSet distinguishes a saved false from an uninitialized settings.json.
	CursorBlinkSet bool `json:"cursorBlinkSet,omitempty"`
	// OptionAsAlt sends Meta/Alt escape sequences for macOS Option key presses.
	OptionAsAlt bool `json:"optionAsAlt,omitempty"`
}

const defaultFontSize = 12.0

var prefs settings

func settingsPath() string { return filepath.Join(rex.Dir(), "settings.json") }

func loadSettings() {
	prefs = settings{CursorBlink: true, CursorBlinkSet: true}
	if b, err := os.ReadFile(settingsPath()); err == nil {
		var raw settings
		if json.Unmarshal(b, &raw) == nil {
			if !raw.CursorBlinkSet {
				raw.CursorBlink = true
				raw.CursorBlinkSet = true
			}
			prefs = raw
		}
	}
	if prefs.FontSize < 6 || prefs.FontSize > 40 {
		prefs.FontSize = defaultFontSize
	}
	if prefs.CursorStyle == "" {
		prefs.CursorStyle = "block"
	}
	termFont.Size = prefs.FontSize
	applyAppearance()
}

func saveSettings() {
	prefs.CursorBlinkSet = true
	b, _ := json.MarshalIndent(prefs, "", "  ")
	os.MkdirAll(rex.Dir(), 0o700)
	os.WriteFile(settingsPath(), b, 0o600)
}

func cursorStyleOf(s string) terminal.CursorStyle {
	switch s {
	case "bar":
		return terminal.CursorBar
	case "underline":
		return terminal.CursorUnderline
	default:
		return terminal.CursorBlock
	}
}

var appRunning bool

func applyAppearance() {
	if !appRunning {
		return
	}
	switch prefs.Appearance {
	case "light":
		mygo.Theme.SetSource(mygo.ThemeLight)
	case "dark":
		mygo.Theme.SetSource(mygo.ThemeDark)
	default:
		mygo.Theme.SetSource(mygo.ThemeSystem)
	}
}

func (a *App) syncMenus() {
	if a.win != nil {
		mygo.App.SetMenu(a.menu())
	}
}

func (a *App) setAppearance(v string) {
	prefs.Appearance = v
	applyAppearance()
	saveSettings()
	a.syncMenus()
	a.invalidateAll()
}

func (a *App) setProductTheme(v string) {
	prefs.ProductTheme = v
	saveSettings()
	a.syncMenus()
	a.invalidateAll()
}

func (a *App) setDensity(v string) {
	prefs.Density = v
	saveSettings()
	a.syncMenus()
	a.invalidateAll()
}

func (a *App) setReduceTransparency(v bool) {
	prefs.ReduceTransparency = v
	saveSettings()
	a.invalidateAll()
}

func (a *App) setHighContrast(v bool) {
	prefs.HighContrast = v
	saveSettings()
	a.refreshTerminals(false)
	a.invalidateAll()
}

func (a *App) setFontSize(size float32) {
	size = min(max(size, 8), 32)
	prefs.FontSize = size
	termFont.Size = size
	a.refreshTerminals(false)
	saveSettings()
	a.invalidateAll()
}

func (a *App) setCursorStyle(style string) {
	prefs.CursorStyle = style
	saveSettings()
	a.refreshTerminals(true)
	a.invalidateAll()
}

func (a *App) setCursorBlink(blink bool) {
	prefs.CursorBlink = blink
	saveSettings()
	a.refreshTerminals(true)
	a.invalidateAll()
}

func (a *App) setOptionAsAlt(v bool) {
	prefs.OptionAsAlt = v
	saveSettings()
	a.refreshTerminals(true)
	a.invalidateAll()
}

// openSettings opens the single-instance native Settings window (⌘,, 780×640,
// matching TraceCommonsAppMain.swift:155), or focuses the existing Settings window.
func (a *App) openSettings() {
	if a.settingsSection == "" {
		a.settingsSection = "general"
	}
	if a.settingsWin != nil {
		a.settingsWin.Show()
		a.settingsWin.Focus()
		a.settingsWin.Update(func() {})
		return
	}
	win := mygo.NewWindow(mygo.WindowOptions{
		Title:           "Settings",
		Width:           int(sizeModalWidth),
		Height:          int(sizeModalHeight),
		MinWidth:        540,
		MinHeight:       420,
		StateKey:        "settings",
		BackgroundColor: "light-dark(#f2f3f6, #0d1124)",
		Content:         ui.View(a.settingsView),
	})
	a.settingsWin = win
	a.invalidateAll()
	win.OnClosed(func() {
		a.settingsWin = nil
		a.invalidateAll()
	})
}

var settingsSections = []struct {
	id, title, group string
}{
	{"general", "General", "app"},
	{"terminal", "Terminal", "app"},
	{"sessions", "Sessions", "runtime"},
	{"about", "About", "meta"},
}

// settingsView renders TCDesign's SettingsModal (SettingsModal.swift + SectionNav.swift):
//   - Edge-to-edge paneOpaque surface (no rounded pane framed inside the window).
//   - Header (padding 14 16 10 18): 15pt bold "Settings" title + mono GlassChip status pill + 1pt rule.
//   - Left 180pt GlassSectionNavRowStyle list (12pt plain text rows, hairline between groups,
//     ink(0.08) hover pill, active section in textPrimary semibold with NO background fill and NO icon/bar),
//     separated by a 1px ink(0.10) vertical rule from one continuous scrolling body of GlassSectionRule
//     headings and GlassEyebrowCard cards.
func (a *App) settingsView(c *ui.Context) {
	if a.settingsSection == "" {
		a.settingsSection = "general"
	}
	ts := themeOf(c)
	ts.Opaque = true
	c.Root().Background(ts.paneOpaque)

	if c.Shortcut(0, ui.KeyEscape) && a.settingsWin != nil {
		a.settingsWin.Close()
		return
	}

	sessions, running := 0, 0
	for _, t := range a.tabs {
		for _, p := range t.panes() {
			sessions++
			if !p.info.Idle && p.info.Program != "" {
				running++
			}
		}
	}

	ui.Column(c).Fill().Background(ts.paneOpaque).Children(func() {
		// Header (.tc-modal__header: top 14, trailing 16, bottom 10, leading 18)
		ui.Row(c).Padding(space14, space16, space10, space18).Gap(space12).AlignItems(ui.Center).
			BorderWidth(0, 0, 1, 0).BorderColor(ts.rule).Shrink(0).Children(func() {
			applyType(ui.Text(c, "Settings"), ts, stepTitle()).FontWeight(700).TextColor(ts.textPrimary)
			ui.Spacer(c)
			if a.err != "" {
				statusChip(c, ts, "critical", "Disconnected")
			} else {
				statusChip(c, ts, "positive", fmt.Sprintf("Connected · %d sessions", sessions))
			}
		})

		// Columns: 180pt section list + ink(0.10) vertical rule + continuous scrolling body
		ui.Row(c).Grow(1).MinHeight(0).AlignItems(ui.Stretch).Children(func() {
			ui.Scroll(c).Width(sizeModalNav).Shrink(0).Padding(space10, space8).
				BorderWidth(0, 1, 0, 0).BorderColor(ts.inkAlpha(0.10)).
				Role(ui.RoleTabList).Label("Settings sections").Children(func() {
				ui.Column(c).Gap(1).Children(func() {
					for idx, sec := range settingsSections {
						sec := sec
						if idx > 0 && settingsSections[idx-1].group != sec.group {
							ui.Box(c.Key("div-"+sec.id)).Height(1).Margin(space6, space4).Background(ts.inkAlpha(0.10))
						}
						active := a.settingsSection == sec.id
						row := ui.Row(c.Key(sec.id)).Padding(space6, space10).
							Radius(ts.controlRadius()).AlignItems(ui.Center).Cursor(ui.CursorPointer).
							Role(ui.RoleTab).Checked(active).Label(sec.title)

						hovered := row.Hovered()
						if hovered {
							fill := ts.inkAlpha(0.08)
							if row.Pressed() {
								fill = darkenPress(fill)
							}
							row.Background(fill)
						}
						row.Transition(ui.ElementTransition{Colors: true, Duration: durFast, Ease: easeStandard})

						textCol := ts.textSecondary
						weight := 400
						if active {
							textCol = ts.textPrimary
							weight = 600
						} else if hovered {
							textCol = ts.textPrimary
						}
						row.Children(func() {
							applyType(ui.Text(c, sec.title), ts, stepLabel()).FontWeight(weight).
								TextColor(textCol).SingleLine().Ellipsis("…")
						})
						if row.Clicked() {
							a.settingsSection = sec.id
							a.settingsScrollReq = sec.id
						}
					}
				})
			})

			scrollReq := a.settingsScrollReq
			a.settingsScrollReq = ""

			ui.Scroll(c).Grow(1).MinWidth(0).Children(func() {
				ui.Column(c).Padding(0, space20, space24, space20).Gap(0).Children(func() {
					a.settingsSectionBlock(c, ts, "general", "General", scrollReq, func() {
						a.settingsGeneralSection(c, ts)
					})
					a.settingsSectionBlock(c, ts, "terminal", "Terminal", scrollReq, func() {
						a.settingsTerminalSection(c, ts)
					})
					a.settingsSectionBlock(c, ts, "sessions", "Sessions", scrollReq, func() {
						a.settingsSessionsSection(c, ts, sessions, running)
					})
					a.settingsSectionBlock(c, ts, "about", "About", scrollReq, func() {
						a.settingsAboutSection(c, ts)
					})
				})
			})
		})
	})
}

func (a *App) settingsSectionBlock(c *ui.Context, ts *themeState, id, heading, scrollReq string, body func()) {
	sec := ui.Column(c.Key("sec-" + id)).Padding(space14, 0, space6, 0).Gap(cardGap)
	if scrollReq == id {
		sec.ScrollIntoView()
	}
	sec.Children(func() {
		glassSectionRule(c, ts, heading)
		body()
	})
}

// settingsToggleRow renders TCDesign's GlassToggleStyle(.settings):
// label + caption wrapping on the left, 40×24 switch aligned to the trailing edge.
func settingsToggleRow(c *ui.Context, ts *themeState, title, caption string, on bool, onToggle func(bool)) {
	ui.Row(c).Gap(space12).AlignItems(ui.Center).Children(func() {
		ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
			applyType(ui.Text(c, title), ts, stepBody()).TextColor(ts.textPrimary)
			if caption != "" {
				applyType(ui.Text(c, caption), ts, stepCaption()).TextColor(ts.textSecondary)
			}
		})
		nearSwitch(c, ts, title, on, onToggle)
	})
}

func (a *App) settingsGeneralSection(c *ui.Context, ts *themeState) {
	glassEyebrowCard(c, ts, "APPEARANCE", "", func() {
		segmentedControl(c, ts, "Appearance", [][2]string{
			{"light", "Light"},
			{"dark", "Dark"},
			{"", "System"},
		}, prefs.Appearance, a.setAppearance)
		applyType(ui.Text(c, "Follow the system appearance or pin light or dark Liquid Glass."), ts, stepCaption()).
			TextColor(ts.textSecondary)
	})

	glassEyebrowCard(c, ts, "THEME & MATERIAL", "", func() {
		segmentedControl(c, ts, "Theme", [][2]string{
			{"", "NEAR AI"},
			{"trace", "Trace"},
			{"classic", "Classic"},
		}, prefs.ProductTheme, a.setProductTheme)
		applyType(ui.Text(c, "NEAR AI is the flat Liquid Glass baseline; Trace applies the Trace purple accent; Classic draws specular rim highlights and gradients."), ts, stepCaption()).
			TextColor(ts.textSecondary)
	})

	glassEyebrowCard(c, ts, "WORKSPACE", "Density", func() {
		segmentedControl(c, ts, "Density", [][2]string{
			{"", "Compact"},
			{"dense", "Dense"},
		}, prefs.Density, a.setDensity)
		applyType(ui.Text(c, "Dense tightens pane insets and split dividers for multi-pane terminal layouts."), ts, stepCaption()).
			TextColor(ts.textSecondary)
	})

	glassEyebrowCard(c, ts, "ACCESSIBILITY", "", func() {
		settingsToggleRow(c, ts, "Reduce transparency",
			"Replace Liquid Glass and floating backdrop blurs with solid opaque surfaces.",
			prefs.ReduceTransparency, a.setReduceTransparency)
		ui.Box(c).Height(1).Background(ts.hairline)
		settingsToggleRow(c, ts, "Increase contrast",
			"Use 7:1 ink contrast and 1pt edge-hc borders across all panes, cards, and controls.",
			prefs.HighContrast, a.setHighContrast)
	})
}

func (a *App) settingsTerminalSection(c *ui.Context, ts *themeState) {
	glassEyebrowCard(c, ts, "TYPOGRAPHY", "Terminal font", func() {
		ui.Row(c).Gap(space12).AlignItems(ui.Center).Children(func() {
			applyType(ui.Text(c, "SF Mono via Core Text with embedded JetBrains Mono fallback (8–32 pt)."), ts, stepCaption()).
				TextColor(ts.textSecondary).Grow(1).MinWidth(0)
			if prefs.FontSize != defaultFontSize {
				if linkAction(c, ts, "Reset").Clicked() {
					a.setFontSize(defaultFontSize)
				}
			}
			stepper := ui.Row(c).Height(sizeSegmentedTrack).Padding(2).Gap(2).
				Radius(radiusPill).Background(ts.well).AlignItems(ui.Center).Shrink(0)
			ts.hcEdge(stepper)
			stepper.Children(func() {
				if iconButton(c, ts, "minus", "Decrease font size", sizeControl, 13).Clicked() {
					a.setFontSize(prefs.FontSize - 1)
				}
				applyType(ui.Text(c, fmt.Sprintf("%.1f pt", prefs.FontSize)), ts, stepMono()).
					FontWeight(600).TextColor(ts.textPrimary).Width(56).TextAlign(ui.Center).SingleLine()
				if iconButton(c, ts, "plus", "Increase font size", sizeControl, 13).Clicked() {
					a.setFontSize(prefs.FontSize + 1)
				}
			})
		})
	})

	glassEyebrowCard(c, ts, "CURSOR", "", func() {
		segmentedControl(c, ts, "Cursor shape", [][2]string{
			{"block", "Block"},
			{"bar", "Bar"},
			{"underline", "Underline"},
		}, prefs.CursorStyle, a.setCursorStyle)
		ui.Box(c).Height(1).Background(ts.hairline)
		settingsToggleRow(c, ts, "Blink cursor",
			"Allow the resting cursor to blink while a terminal pane has keyboard focus.",
			prefs.CursorBlink, a.setCursorBlink)
	})

	glassEyebrowCard(c, ts, "KEYBOARD", "", func() {
		settingsToggleRow(c, ts, "Use Option as Meta key",
			"Send Alt/Escape sequences for readline, vim, and emacs instead of macOS symbols.",
			prefs.OptionAsAlt, a.setOptionAsAlt)
	})
}

func (a *App) settingsSessionsSection(c *ui.Context, ts *themeState, sessions, running int) {
	glassEyebrowCard(c, ts, "DAEMON", "Persistent session server", func() {
		pid, uptime := "—", "—"
		proto := fmt.Sprintf("v%d", rex.ProtocolVersion)
		if a.hello.PID > 0 {
			pid = fmt.Sprintf("%d", a.hello.PID)
			uptime = roundDur(time.Since(a.hello.Started))
			proto = fmt.Sprintf("v%d", a.hello.Version)
		}
		statusText := "Connected"
		if a.err != "" {
			statusText = "Needs attention"
		}
		ui.Column(c).Gap(space6).Children(func() {
			kvRow(c, ts, "Status", statusText, 88, false)
			kvRow(c, ts, "Server PID", pid, 88, true)
			kvRow(c, ts, "Uptime", uptime, 88, true)
			kvRow(c, ts, "Protocol", proto, 88, true)
			kvRow(c, ts, "Open tabs", fmt.Sprintf("%d", len(a.tabs)), 88, true)
			kvRow(c, ts, "Open panes", fmt.Sprintf("%d (%d running)", sessions, running), 88, true)
			kvRow(c, ts, "Data folder", shortDir(rex.Dir()), 88, true)
		})
		ui.Box(c).Height(1).Background(ts.hairline)
		ui.Row(c).Gap(space8).AlignItems(ui.Center).Children(func() {
			if glassButton(c, ts, "Show in Finder", "glass", true, GroundPane).Clicked() && a.win != nil {
				_ = mygo.Shell.OpenPath(rex.Dir())
			}
			ui.Spacer(c)
			if glassButton(c, ts, "End All Sessions", "destructive", true, GroundPane).Clicked() {
				a.quitAndEnd()
			}
		})
	})
}

func (a *App) settingsAboutSection(c *ui.Context, ts *themeState) {
	glassEyebrowCard(c, ts, "APPLICATION", "", func() {
		ui.Row(c).Gap(space14).AlignItems(ui.Center).Children(func() {
			logo := nearSecondaryLogo(ts.Dark)
			lw, lh := logo.Size()
			logoH := float32(20)
			logoW := logoH * (lw / max(lh, 1))
			ui.Image(c, logo).Size(logoW, logoH).Label("NEAR AI").Shrink(0)
			ui.Column(c).Gap(2).Children(func() {
				applyType(ui.Text(c, "Multiplexer 0.1.0"), ts, stepHeading()).TextColor(ts.textPrimary)
				applyType(ui.Text(c, "Native persistent terminal · Near AI OS Liquid Glass"), ts, stepCaption()).
					TextColor(ts.textSecondary)
			})
		})
		ui.Box(c).Height(1).Background(ts.hairline)
		applyType(ui.Text(c, "Built in pure Go with MyGo Metal GPU UI, AppKit NSGlassEffectView Liquid Glass, and Ghostty's libghostty-vt terminal engine."), ts, stepBody()).
			TextColor(ts.textSecondary)
	})

	glassEyebrowCard(c, ts, "ENVIRONMENT", "", func() {
		h := a.hello.Host
		mem := "—"
		if h.Memory > 0 {
			mem = fmt.Sprintf("%d GB", h.Memory>>30)
		}
		motion := "Standard"
		if ts.ReduceMotion {
			motion = "Reduced"
		}
		surface := "Liquid Glass (NSGlassEffectView)"
		if ts.Opaque && prefs.ReduceTransparency {
			surface = "Opaque (Reduce Transparency)"
		} else if ts.Classic {
			surface = "Classic Specular Glass"
		}
		ui.Column(c).Gap(space6).Children(func() {
			kvRow(c, ts, "Active theme", string(ts.Name), 88, true)
			kvRow(c, ts, "Material", surface, 88, false)
			kvRow(c, ts, "Motion", motion, 88, false)
			kvRow(c, ts, "Host", h.Name, 88, false)
			kvRow(c, ts, "Chip", h.Chip, 88, false)
			kvRow(c, ts, "Memory", mem, 88, true)
			kvRow(c, ts, "OS", h.OS, 88, false)
		})
	})
}
