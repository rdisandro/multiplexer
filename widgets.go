package main

import (
	"strings"

	"github.com/egoist/mygo/plugins/glass"
	"github.com/egoist/mygo/ui"
)

// styleFloating applies TCDesign's floating-layer treatment (Surfaces.swift, 03-materials.md):
// 24px within-window backdrop blur + 94% opaque popover/menu fill + shadow-float/shadow-modal,
// falling back to solid paneOpaque when Reduce Transparency is active.
func styleFloating(e ui.Element, ts *themeState, fill ui.Color, radius float32, modal bool) ui.Element {
	sh := ts.shadowFloat
	if modal {
		sh = ts.shadowModal
	}
	e.Radius(radius).Shadow(0, sh.y, sh.blur, 0, sh.color)
	if ts.ReduceTransparency {
		e.Background(ts.paneOpaque)
	} else {
		e.Material(glass.Blur{Radius: 24}).Draw(func(p *ui.Painter, r ui.Rect) {
			p.Fill(r, fill, radius)
			if ts.Classic {
				topA := float32(0.25)
				if !ts.Dark {
					topA = 0.50
				}
				p.Stroke(r, ui.RGBA(255, 255, 255, topA*0.5), radius, 1)
			}
		})
	}
	return ts.hcEdge(e)
}

// paintControlSurface paints a capsule/circle control on GroundPane, GroundMenu, or GroundSelection,
// matching TCDesign's GlassGround + GlassSurface (Surfaces.swift & GlassStyle.swift).
func paintControlSurface(e ui.Element, ts *themeState, ground Ground, hovered, pressed, expanded bool) ui.Color {
	var fill, ink ui.Color
	switch ground {
	case GroundMenu:
		fill, ink = ts.controlOnMenu, ts.textPrimary
		if hovered {
			fill = ts.controlOnMenuHover
		}
	case GroundSelection:
		fill, ink = ts.controlOnSelection, ts.textOnAccent
		if hovered {
			fill = ts.controlOnSelectionHover
		}
	default:
		fill, ink = ts.control, ts.textPrimary
		if hovered {
			fill = ts.controlHover
		}
	}
	if expanded {
		fill = ts.toolbarExpanded
		if !ts.Classic {
			ink = ts.textOnSelected
		}
	}
	if pressed {
		fill = darkenPress(fill)
	}
	if ts.Classic && ground == GroundPane && !hovered && !expanded && !pressed {
		e.Draw(func(p *ui.Painter, r ui.Rect) {
			rad := min(r.W, r.H) / 2
			p.FillGradient(r, ui.LinearGradient{From: ts.control, To: ts.controlEnd, Angle: 180}, rad)
			topA := float32(0.32)
			if !ts.Dark {
				topA = 0.64
			}
			p.Stroke(r, ui.RGBA(255, 255, 255, topA*0.5), rad, 1)
		})
	} else {
		e.Background(fill)
	}
	ts.hcEdge(e)
	return ink
}

// toolbarGroup renders TCDesign's GlassToolbarGroup (Controls.swift:452):
// padding 2, spacing 2, capsule control tier.
func toolbarGroup(c *ui.Context, ts *themeState, children func()) ui.Element {
	g := ui.Row(c).Padding(2).Gap(2).Radius(radiusPill).AlignItems(ui.Center).Shrink(0)
	if ts.Classic {
		g.Draw(func(p *ui.Painter, r ui.Rect) {
			rad := r.H / 2
			p.FillGradient(r, ui.LinearGradient{From: ts.control, To: ts.controlEnd, Angle: 180}, rad)
			topA := float32(0.32)
			if !ts.Dark {
				topA = 0.64
			}
			p.Stroke(r, ui.RGBA(255, 255, 255, topA*0.5), rad, 1)
		})
	} else {
		g.Background(ts.control)
	}
	ts.hcEdge(g)
	g.Children(children)
	return g
}

// toolbarButton renders TCDesign's 28×24 GlassToolbarButton (Controls.swift:469):
// toggle buttons show toolbarGlyphOff when hidden and toolbarExpanded + textOnSelected when open;
// action buttons show toolbarGlyph and controlHover on hover.
func toolbarButton(c *ui.Context, ts *themeState, glyph string, gw, gh float32, label string, isToggle, active bool) ui.Element {
	b := ui.Box(c).Size(sizeControlLarge, 24).Center().Radius(radiusPill).
		Cursor(ui.CursorPointer).Role(ui.RoleButton).Label(label)
	var fg ui.Color
	if active {
		fill := ts.toolbarExpanded
		if b.Pressed() {
			fill = darkenPress(fill)
		}
		b.Background(fill)
		fg = ts.textOnSelected
		if ts.Classic {
			fg = ts.toolbarGlyph
		}
	} else {
		if b.Hovered() {
			fill := ts.controlHover
			if b.Pressed() {
				fill = darkenPress(fill)
			}
			b.Background(fill)
		} else if b.Pressed() {
			b.Background(darkenPress(ts.controlHover))
		}
		if isToggle {
			fg = ts.toolbarGlyphOff
		} else {
			fg = ts.toolbarGlyph
		}
	}
	b.Transition(ui.ElementTransition{Colors: true, Duration: durFast, Ease: easeStandard})
	b.Children(func() {
		ui.Icon(c, icon(glyph)).Size(gw, gh).TextColor(fg)
	})
	return b
}

// glassRoundButton renders TCDesign's GlassRoundButton (Controls.swift:509):
// 28×28 (or 26×26 when small) circular control button, darkened on press.
func glassRoundButton(c *ui.Context, ts *themeState, glyph string, glyphSize float32, label string, small, expanded bool, ground Ground) ui.Element {
	side := sizeControlLarge
	if small {
		side = sizeControl
	}
	b := ui.Box(c).Size(side, side).Center().Radius(radiusPill).
		Cursor(ui.CursorPointer).Role(ui.RoleButton).Label(label).Shrink(0)
	fg := paintControlSurface(b, ts, ground, b.Hovered(), b.Pressed(), expanded)
	b.Transition(ui.ElementTransition{Colors: true, Duration: durFast, Ease: easeStandard})
	b.Children(func() {
		ui.Icon(c, icon(glyph)).Size(glyphSize, glyphSize).TextColor(fg)
	})
	return b
}

// glassPickerPill renders TCDesign's GlassPickerPill (Controls.swift:1072):
// status dot + semibold label + #1146 10pt chevron on a 28pt control capsule.
func glassPickerPill(c *ui.Context, ts *themeState, statusTone, title, label string, open bool) ui.Element {
	pill := ui.Row(c).Height(sizeControlLarge).Gap(inlineGap).Padding(0, 8, 0, 10).
		Radius(radiusPill).AlignItems(ui.Center).Cursor(ui.CursorPointer).
		Role(ui.RoleButton).Label(label).MaxWidth(210).Shrink(0)
	fg := paintControlSurface(pill, ts, GroundPane, pill.Hovered() || open, pill.Pressed(), false)
	pill.Transition(ui.ElementTransition{Colors: true, Duration: durFast, Ease: easeStandard})
	pill.Children(func() {
		if statusTone != "" {
			ui.Box(c).Size(7, 7).Radius(radiusPill).Background(ts.toneDotColor(statusTone)).Shrink(0)
		}
		applyType(ui.Text(c, title), ts, stepLabel()).FontWeight(600).
			TextColor(fg).SingleLine().Ellipsis("…").MinWidth(0)
		ui.Icon(c, icon("tc-chevron-down")).Size(10, 10).TextColor(ts.textTertiary).Shrink(0)
	})
	return pill
}

// iconButton is a quiet capsule/circle icon button for inline pane/tab chrome.
func iconButton(c *ui.Context, ts *themeState, name, label string, size, iconSize float32) ui.Element {
	b := ui.Box(c).Size(size, size).Center().Radius(radiusPill).
		Cursor(ui.CursorPointer).Role(ui.RoleButton).Label(label)
	if b.Pressed() {
		b.Background(darkenPress(ts.controlHover))
	} else if b.Hovered() {
		b.Background(ts.rowHover)
	}
	b.Transition(ui.ElementTransition{Colors: true, Duration: durFast, Ease: easeStandard})
	col := ts.textSecondary
	if b.Hovered() || b.Pressed() {
		col = ts.textPrimary
	}
	b.Children(func() {
		ui.Icon(c, icon(name)).Size(iconSize, iconSize).TextColor(col)
	})
	return b
}

// glassButton renders TCDesign's GlassButtonStyle (Controls.swift:79, 10-actions.css):
// ALWAYS a capsule (radiusPill = 999), ground-aware, darkened on press (never faded).
func glassButton(c *ui.Context, ts *themeState, label, kind string, small bool, ground Ground) ui.Element {
	h, padX := sizeCta, float32(16)
	switch {
	case kind == "primary" && small:
		h, padX = sizeCtaSmall, 14
	case kind != "primary" && small:
		h, padX = sizeControlLarge, 12
	case kind != "primary":
		h, padX = sizeCtaSmall, 14
	}
	b := ui.Row(c).Height(h).Padding(0, padX).Radius(radiusPill).
		AlignItems(ui.Center).Justify(ui.Center).Cursor(ui.CursorPointer).
		Role(ui.RoleButton).Label(label).Shrink(0)

	var fg ui.Color
	weight := 500
	switch kind {
	case "primary":
		weight = 600
		fg = ts.textOnAccent
		if ts.Classic {
			from, to := ts.accent, ts.accentEnd
			if b.Pressed() {
				from, to = darkenPress(from), darkenPress(to)
			}
			b.Shadow(0, 4, 12, 0, tracePurple.Alpha(0.30))
			b.Draw(func(p *ui.Painter, r ui.Rect) {
				rad := r.H / 2
				p.FillGradient(r, ui.LinearGradient{From: from, To: to, Angle: 180}, rad)
				p.Stroke(r, ui.RGBA(255, 255, 255, 0.45), rad, 1)
			})
		} else {
			fill := ts.accent
			if b.Pressed() {
				fill = ts.accentPressed
			} else if b.Hovered() {
				fill = ts.accent.Mix(ts.accentPressed, 0.38)
			}
			b.Background(fill)
		}
	case "destructive":
		weight = 600
		paintControlSurface(b, ts, ground, b.Hovered(), b.Pressed(), false)
		fg = ts.destructiveText
	default:
		fg = paintControlSurface(b, ts, ground, b.Hovered(), b.Pressed(), false)
	}
	b.Transition(ui.ElementTransition{Colors: true, Duration: durFast, Ease: easeStandard})
	b.Children(func() {
		step := stepLabel()
		if kind == "primary" && !small {
			step = stepBodyStrong()
		}
		applyType(ui.Text(c, label), ts, step).FontWeight(weight).TextColor(fg).SingleLine()
	})
	return b
}

// actionButton renders a primary or glass capsule button on the pane ground.
func actionButton(c *ui.Context, ts *themeState, label string, primary, small bool) ui.Element {
	if primary {
		return glassButton(c, ts, label, "primary", small, GroundPane)
	}
	return glassButton(c, ts, label, "glass", small, GroundPane)
}

// linkAction renders TCDesign's GlassButtonKind.link (Controls.swift:175).
func linkAction(c *ui.Context, ts *themeState, label string) ui.Element {
	b := ui.Row(c).Padding(2, 4).Radius(radiusTile).AlignItems(ui.Center).
		Cursor(ui.CursorPointer).Role(ui.RoleButton).Label(label).Shrink(0)
	hovered := b.Hovered()
	col := ts.linkText
	if b.Pressed() {
		col = darkenPress(col)
	}
	b.Children(func() {
		t := applyType(ui.Text(c, label), ts, stepLabel()).TextColor(col).SingleLine()
		if hovered {
			t.Underline()
		}
	})
	return b
}

// toneDotColor returns the semantic mark color for a StatusLabel, Tag, or Chip dot.
func (ts *themeState) toneDotColor(tone string) ui.Color {
	switch tone {
	case "positive":
		return ts.positive
	case "caution":
		return ts.caution
	case "critical":
		return ts.critical
	case "accent":
		return ts.accent
	default:
		return ts.neutral
	}
}

// statusLabel renders TCDesign's GlassStatusLabel (Indicators.swift:34):
// solid dot + plain words in textSecondary (or bodyStrong textPrimary when strong).
func statusLabel(c *ui.Context, ts *themeState, tone, words string, strong bool) ui.Element {
	dotSize := float32(7)
	if strong {
		dotSize = 8
	}
	row := ui.Row(c).Gap(inlineGap).AlignItems(ui.Center).Shrink(0)
	row.Children(func() {
		ui.Box(c).Size(dotSize, dotSize).Radius(radiusPill).Background(ts.toneDotColor(tone)).Shrink(0)
		if strong {
			applyType(ui.Text(c, words), ts, stepBodyStrong()).TextColor(ts.textPrimary).SingleLine()
		} else {
			applyType(ui.Text(c, words), ts, stepLabel()).TextColor(ts.textSecondary).SingleLine()
		}
	})
	return row
}

// statusChip renders TCDesign's GlassChip (Indicators.swift:57):
// mono 11pt capsule in neutralTint (or 0.5pt hairline ring in classic) with 6pt status dot
// and text-safe status ink.
func statusChip(c *ui.Context, ts *themeState, tone, words string) ui.Element {
	ink := ts.textSecondary
	switch tone {
	case "positive":
		ink = ts.positiveText
	case "caution":
		ink = ts.cautionText
	case "critical":
		ink = ts.criticalText
	case "accent":
		ink = ts.accentText
	}
	chip := ui.Row(c).Padding(3, 9).Gap(5).Radius(radiusPill).AlignItems(ui.Center).Shrink(0)
	if ts.Classic {
		chip.Border(1, ink.Alpha(0.55))
	} else {
		chip.Background(ts.neutralTint)
		ts.hcEdge(chip)
	}
	chip.Children(func() {
		if tone != "" {
			ui.Box(c).Size(6, 6).Radius(radiusPill).Background(ts.toneDotColor(tone)).Shrink(0)
		}
		applyType(ui.Text(c, words), ts, stepMono()).TextColor(ink).SingleLine()
	})
	return chip
}

// segmentedControl renders TCDesign's GlassSegmentedTabs (Navigation.swift:32, 20-inputs.css:579):
// capsule well track (radiusPill = 999, height = 30, padding = 2, gap = 2),
// capsule active segment in controlSelected with textOnSelected semibold ink,
// unselected segments in textTertiary lifting to textPrimary on hover with no press fill.
func segmentedControl(c *ui.Context, ts *themeState, groupLabel string, options [][2]string, current string, onChange func(string)) ui.Element {
	track := ui.Row(c).Height(sizeSegmentedTrack).Padding(2).Gap(2).Radius(radiusPill).
		Background(ts.well).AlignItems(ui.Center).AlignSelf(ui.Start).
		Role(ui.RoleRadioGroup).Label(groupLabel).Shrink(0)
	ts.hcEdge(track)
	track.Children(func() {
		for _, opt := range options {
			val, text := opt[0], opt[1]
			selected := val == current
			seg := ui.Row(c.Key(val)).Height(sizeSegmentedTrack - 4).Padding(0, 11).Radius(radiusPill).
				AlignItems(ui.Center).Justify(ui.Center).Cursor(ui.CursorPointer).
				Role(ui.RoleRadio).Checked(selected).Label(text)
			fg := ts.textTertiary
			weight := 500
			if selected {
				fill := ts.controlSelected
				if seg.Pressed() && !ts.Classic {
					fill = ts.accentPressed
				}
				seg.Background(fill)
				if ts.Classic {
					seg.Border(1, ui.RGBA(255, 255, 255, 0.32))
				}
				fg = ts.textOnSelected
				weight = 600
			} else if seg.Hovered() {
				fg = ts.textPrimary
			}
			seg.Transition(ui.ElementTransition{Colors: true, Duration: durFast, Ease: easeStandard})
			seg.Children(func() {
				applyType(ui.Text(c, text), ts, stepLabel()).FontWeight(weight).TextColor(fg).SingleLine()
			})
			if seg.Clicked() && !selected {
				onChange(val)
			}
		}
	})
	return track
}

type switchAnimKey struct{}

// nearSwitch renders TCDesign's GlassToggleStyle (Controls.swift:1110, 20-inputs.css:514):
// 40×24 capsule track (toggleOff 38% white in dark flat; accent when on),
// flat 18×18 circular knob with 3pt inset in toggleKnobOff / textOnAccent (no ring, no shadow).
func nearSwitch(c *ui.Context, ts *themeState, label string, on bool, onToggle func(bool)) ui.Element {
	sw := ui.Box(c).Size(sizeToggleWidth, sizeToggleHeight).Radius(radiusPill).
		Cursor(ui.CursorPointer).Focusable().Role(ui.RoleSwitch).Checked(on).Label(label).Shrink(0)
	fill := ts.toggleOff
	if on {
		fill = ts.accent
		if sw.Pressed() {
			fill = ts.accentPressed
		}
	} else if sw.Pressed() {
		fill = darkenPress(ts.toggleOff)
	}
	sw.Background(fill)
	ts.hcEdge(sw)
	target := float32(0)
	if on {
		target = 1
	}
	pos := sw.AnimateWith(switchAnimKey{}, target, durSpringFast, springSpatial)
	knobCol := ts.toggleKnobOff
	if on {
		knobCol = ts.textOnAccent
	}
	sw.Draw(func(p *ui.Painter, r ui.Rect) {
		const (
			knob  = 18
			inset = 3
		)
		x := r.X + inset + pos*(r.W-knob-2*inset)
		y := r.Y + (r.H-knob)/2
		p.Fill(ui.Rect{X: x, Y: y, W: knob, H: knob}, knobCol, knob/2)
	})
	if sw.Clicked() || sw.Shortcut(0, ui.KeySpace) || sw.Shortcut(0, ui.KeyEnter) {
		onToggle(!on)
	}
	return sw
}

// glassSectionRule renders TCDesign's GlassSectionRule (Containers.swift:287):
// uppercase accentText eyebrow heading followed by a 1px ink(0.14) hairline.
func glassSectionRule(c *ui.Context, ts *themeState, title string) {
	ui.Row(c).Gap(space8).AlignItems(ui.Center).Children(func() {
		applyType(ui.Text(c, strings.ToUpper(title)), ts, stepEyebrow()).TextColor(ts.accentText).Shrink(0)
		ui.Box(c).Height(1).Grow(1).Background(ts.inkAlpha(0.14))
	})
}

// glassEyebrowCard renders TCDesign's GlassEyebrowCard (Containers.swift:100):
// painted card surface with uppercase textTertiary eyebrow, optional title, and body.
func glassEyebrowCard(c *ui.Context, ts *themeState, eyebrow, title string, children func()) {
	card := styleGlassCard(ui.Column(c).Padding(cardPaddingY, cardPaddingX).Gap(space8), ts, false)
	card.Children(func() {
		if eyebrow != "" {
			applyType(ui.Text(c, strings.ToUpper(eyebrow)), ts, stepEyebrow()).TextColor(ts.textTertiary)
		}
		if title != "" {
			applyType(ui.Text(c, title), ts, stepBodyStrong()).TextColor(ts.textPrimary)
		}
		children()
	})
}

// kvRow renders TCDesign's GlassKeyValueList row (Containers.swift:653):
// leading-aligned textTertiary label (12pt regular), 10pt horizontal gap, textPrimary value.
func kvRow(c *ui.Context, ts *themeState, key, val string, keyW float32, monoVal bool) {
	if val == "" {
		val = "—"
	}
	ui.Row(c).Gap(space10).AlignItems(ui.Center).MinWidth(0).Children(func() {
		applyType(ui.Text(c, key), ts, stepLabel()).FontWeight(400).TextColor(ts.textTertiary).
			Width(keyW).TextAlign(ui.Start).Shrink(0).SingleLine()
		v := ui.Text(c, val)
		if monoVal {
			applyType(v, ts, stepMono())
		} else {
			applyType(v, ts, stepLabel()).FontWeight(400)
		}
		v.TextColor(ts.textPrimary).Grow(1).MinWidth(0).SingleLine().Ellipsis("…")
	})
}
