package main

import (
	"fmt"
	"runtime"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
)

const (
	tabRowH  = 32
	tabMinW  = 124
	tabMaxW  = 224
	tileSide = 22
	tileStep = 5
)

type (
	emberXKey struct{}
	emberWKey struct{}
)

// tabStrip renders TCDesign's GlassRuleTabs (Navigation.swift:222, 30-navigation.css:24):
//   - A 1pt rule bleeding edge-to-edge across the entire GlassPane (bleed: .toPaneEdge).
//   - Equal-flex / proportional tabs with ZERO press fill and ZERO hover fill box;
//     hover lifts textSecondary (500) to textPrimary, active tab is textPrimary (600).
//   - Active tab lit from below by a 26pt elliptical tabEmberHalo glow rising behind
//     the title AND a 2pt tabEmber gradient line reaching 30% past each side of the tab
//     (total width = 1.6× tab width), gliding on --spring-ember (488ms).
func (a *App) tabStrip(c *ui.Context, ts *themeState, sideInset float32) {
	outer := ui.Row(c).Height(tabRowH).Padding(0, sideInset).Gap(space6).AlignItems(ui.Center)
	outerBounds := outer.Bounds()

	var track ui.Element
	var trackBounds, activeBounds ui.Rect
	tabs := slices.Clone(a.tabs)

	outer.Children(func() {
		track = ui.Row(c).Grow(1).MinWidth(0).Height(tabRowH).Gap(space6).
			AlignItems(ui.Center).ClipX().Role(ui.RoleTabList).Label("Tabs")
		trackBounds = track.Bounds()
		track.Children(func() {
			for i, t := range tabs {
				b := a.tabItem(c, ts, i, t)
				if i == a.active {
					activeBounds = b
				}
			}
		})
		addBtn := iconButton(c, ts, "plus", "New Tab", 24, 13).Tooltip(tip("New Tab", &cmdNewTab))
		if addBtn.Clicked() {
			a.newTab(a.currentDir())
		}
	})

	fallbackW := float32(180)
	if n := len(tabs); n > 0 && trackBounds.W > 0 {
		fallbackW = min(trackBounds.W/float32(n), tabMaxW)
	}
	targetX := float32(a.active) * (fallbackW + space4)
	targetW := fallbackW
	if trackBounds.W > 0 && activeBounds.W > 0 && outerBounds.W > 0 {
		targetX = activeBounds.X - outerBounds.X
		targetW = activeBounds.W
	} else {
		targetX += sideInset
	}

	emberX := outer.AnimateWith(emberXKey{}, targetX, durSpringEmber, springEmber)
	emberW := outer.AnimateWith(emberWKey{}, targetW, durSpringEmber, springEmber)

	outer.DrawOver(func(p *ui.Painter, r ui.Rect) {
		// Full-bleed 1pt rule from left pane edge to right pane edge (toPaneEdge).
		p.Fill(ui.Rect{X: r.X, Y: r.Y + r.H - 1, W: r.W, H: 1}, ts.rule, 0)
		if len(tabs) == 0 || emberW <= 0 {
			return
		}
		ex := r.X + emberX
		ew := emberW

		// 26pt elliptical tabEmberHalo rising from the rule behind the selected title:
		// layered concentric soft pills fade smoothly upward and outward.
		const haloLayers = 5
		for step := range haloLayers {
			frac := float32(step+1) / float32(haloLayers)
			hw := ew * (0.42 + 0.66*frac)
			hh := 6 + 20*frac
			hx := ex + (ew-hw)/2
			hy := r.Y + r.H - 1 - hh
			// Clamp horizontally inside the pane's edge.
			if hx < r.X+1 {
				hw -= (r.X + 1 - hx)
				hx = r.X + 1
			}
			if hx+hw > r.X+r.W-1 {
				hw = (r.X + r.W - 1) - hx
			}
			if hw > 4 && hh > 2 {
				layerCol := ts.tabEmberHalo.Alpha(float32(ts.tabEmberHalo.A) / 255.0 * (0.34 * (1 - 0.14*float32(step))))
				p.FillGradient(ui.Rect{X: hx, Y: hy, W: hw, H: hh}, ui.LinearGradient{
					From:  layerCol.Alpha(0),
					To:    layerCol,
					Angle: 180,
				}, hh*0.48)
			}
		}

		// 2pt tabEmber line laid on the rule, reaching 30% past each side of the selected tab
		// (GlassRuleTabs.emberSpan = 1.6, emberOffset = -0.3 * tabWidth), clipped to pane bounds.
		lx := ex - ew*0.30
		lw := ew * 1.60
		mid := ex + ew*0.50
		leftX := max(lx, r.X+1)
		rightX := min(lx+lw, r.X+r.W-1)
		y := r.Y + r.H - 2
		if mid > leftX {
			leftStartAlpha := float32(0)
			if leftX > lx && (mid-lx) > 0 {
				leftStartAlpha = (leftX - lx) / (mid - lx)
			}
			fromCol := ts.tabEmber.Alpha(float32(ts.tabEmber.A) / 255.0 * leftStartAlpha)
			p.FillGradient(ui.Rect{X: leftX, Y: y, W: mid - leftX + 0.5, H: 2}, ui.LinearGradient{
				From:  fromCol,
				To:    ts.tabEmber,
				Angle: 90,
			}, 0)
		}
		if rightX > mid {
			rightEndAlpha := float32(0)
			if rightX < lx+lw && (lx+lw-mid) > 0 {
				rightEndAlpha = 1 - (rightX-mid)/(lx+lw-mid)
			}
			toCol := ts.tabEmber.Alpha(float32(ts.tabEmber.A) / 255.0 * rightEndAlpha)
			p.FillGradient(ui.Rect{X: mid, Y: y, W: rightX - mid, H: 2}, ui.LinearGradient{
				From:  ts.tabEmber,
				To:    toCol,
				Angle: 90,
			}, 0)
		}
	})
}

func tabLabelSpans(ts *themeState, name, detail string, active bool, nameColor, detailColor ui.Color) []ui.Span {
	step := stepLabel()
	weight := 500
	if active {
		weight = 600
	}
	spans := []ui.Span{
		{Text: name, Size: step.size * ts.TextScale, Weight: weight, Color: nameColor},
	}
	if detail != "" {
		spans = append(spans, ui.Span{
			Text: "  " + detail, Size: (step.size - 1) * ts.TextScale, Weight: 400, Color: detailColor,
		})
	}
	return spans
}

func tabNaturalWidth(c *ui.Context, ts *themeState, t *Tab) float32 {
	name, detail := t.label()
	w, _ := c.MeasureText(0, tabLabelSpans(ts, name, detail, true, ui.Color{}, ui.Color{})...)
	tiles := float32(tileSide + tileStep*min(max(len(t.panes())-1, 0), 2))
	return min(max(10+tiles+8+w+8+20+10, tabMinW), tabMaxW)
}

// tabItem renders one GlassRuleTabs item and returns its layout bounds.
func (a *App) tabItem(c *ui.Context, ts *themeState, i int, t *Tab) ui.Rect {
	active := i == a.active
	natW := tabNaturalWidth(c, ts, t)
	e := ui.Row(c.Key(t.ID)).Height(tabRowH).Basis(natW).Shrink(1).MinWidth(72).
		Padding(0, 8, 4, 8).Gap(space8).AlignItems(ui.Center).
		Cursor(ui.CursorPointer).Role(ui.RoleTab).Checked(active)

	name, detail := t.label()
	e.Label(strings.TrimSpace(name + " " + detail))

	hasAttn := t.attention()
	isBusy := !hasAttn && a.tabBusy(c, t)
	if hasAttn {
		e.Description("Needs a look")
	} else if isBusy {
		e.Description("Running")
	}

	// TCDesign GlassRuleTabs rule: NO press fill and NO hover rectangle;
	// only a dragged tab lifts onto a control capsule with a soft shadow.
	e.Drag(t)
	if dragged, ok := ui.Drop[*Tab](e); ok && dragged != t {
		a.moveTab(dragged, i)
	}
	_, dragOver := ui.DragOver[*Tab](e)
	if dragOver {
		e.DrawOver(func(p *ui.Painter, r ui.Rect) {
			p.Fill(ui.Rect{X: r.X, Y: r.Y + 5, W: 2, H: r.H - 12}, ts.accent, 1)
		})
	}
	if e.Dragging() {
		e.Radius(radiusPill).Background(ts.control).Shadow(0, 6, 16, 0, ui.RGBA(0, 0, 0, 0.28))
	}
	e.Transition(ui.ElementTransition{Colors: true, Position: true, Duration: durFast, Ease: easeStandard})

	if e.Clicked() && !active {
		a.selectTab(i)
	}
	if e.DoubleClicked() {
		a.selectTab(i)
		a.startRename()
	}
	e.ContextMenu(func(m *ui.Menu) { a.tabMenu(m, t) })

	hovered := e.Hovered()
	e.Children(func() {
		a.tiles(c, ts, t)
		if a.renaming == t {
			a.renameField(c, ts, t, name)
			return
		}
		nameColor := ts.textSecondary
		if active || hovered {
			nameColor = ts.textPrimary
		}
		detailColor := ts.textTertiary
		label := ui.Box(c).Grow(1).MinWidth(0).Height(18)
		label.Draw(func(p *ui.Painter, r ui.Rect) {
			fadeText(p, r, tabLabelSpans(ts, name, detail, active, nameColor, detailColor))
		})
		if hovered && len(a.tabs) > 1 {
			x := iconButton(c, ts, "x", "Close Tab", sizeControlDense, 12).Tooltip("Close Tab")
			if x.Clicked() {
				a.later(c, func() { a.closeTab(t) })
			}
		} else if hasAttn {
			ui.Box(c).Size(7, 7).Radius(radiusPill).Background(ts.caution).Margin(0, 3, 0, 0).Tooltip("Needs a look")
		} else if isBusy {
			ui.Box(c).Size(7, 7).Radius(radiusPill).Background(ts.positive).Margin(0, 3, 0, 0).Tooltip("Running")
		}
	})
	return e.Bounds()
}

// renameField edits a tab's title inline in a TCDesign GlassTextField well.
func (a *App) renameField(c *ui.Context, ts *themeState, t *Tab, name string) {
	in := ui.TextInput(c, &a.renameText).Placeholder(name).Grow(1).MinWidth(0).
		Height(22).Padding(0, 8).Radius(ts.controlRadius()).Background(ts.field).
		FontSize(stepLabel().size * ts.TextScale).TextColor(ts.textPrimary).AutoFocus()
	ts.hcEdge(in)
	done := func(keep bool) {
		if keep {
			t.Name = strings.TrimSpace(a.renameText)
			a.changed()
		}
		a.renaming, a.renameFocus = nil, false
		a.focusReq = t.Focus
	}
	switch {
	case in.Shortcut(0, ui.KeyEscape):
		done(false)
	case in.Submitted():
		done(true)
	case in.Focused():
		a.renameFocus = true
	case a.renameFocus:
		done(true)
	}
}

func (t *Tab) attention() bool {
	for _, p := range t.panes() {
		if p.attention {
			return true
		}
	}
	return false
}

func (a *App) tabBusy(c *ui.Context, t *Tab) bool {
	for _, p := range t.panes() {
		if a.statusOf(c, p) == statusRunning {
			return true
		}
	}
	return false
}

func (a *App) moveTab(t *Tab, to int) {
	from := slices.Index(a.tabs, t)
	if from < 0 || to == from {
		return
	}
	activeTab := a.tab()
	a.tabs = slices.Delete(a.tabs, from, from+1)
	a.tabs = slices.Insert(a.tabs, min(to, len(a.tabs)), t)
	a.active = slices.Index(a.tabs, activeTab)
	a.changed()
}

func (a *App) tabMenu(m *ui.Menu, t *Tab) {
	if m.Item("Rename Tab…").Chosen() {
		a.selectTab(slices.Index(a.tabs, t))
		a.startRename()
	}
	if t.Name != "" && m.Item("Reset Name").Chosen() {
		t.Name = ""
		a.changed()
	}
	m.Separator()
	i := slices.Index(a.tabs, t)
	if m.Item("Move Left").Disabled(i <= 0).Chosen() {
		a.moveTab(t, i-1)
	}
	if m.Item("Move Right").Disabled(i >= len(a.tabs)-1).Chosen() {
		a.moveTab(t, i+1)
	}
	m.Separator()
	mods, key := ui.Cmd, ui.KeyT
	if runtime.GOOS != "darwin" {
		mods, key = cmdNewTab.Other.Mods, cmdNewTab.Other.Key
	}
	if m.Item("New Tab").Shortcut(mods, key).Chosen() {
		a.newTab(a.currentDir())
	}
	if m.Item("Close Tab").Chosen() {
		a.laterService(func() { a.closeTab(t) })
	}
	if m.Item("Close Other Tabs").Disabled(len(a.tabs) < 2).Chosen() {
		a.laterService(func() {
			for _, o := range slices.Clone(a.tabs) {
				if o != t {
					a.closeTab(o)
				}
			}
		})
	}
}

// tiles draws TCDesign's 22pt GlassToolTile (Indicators.swift:251):
// the focused pane's program in front, up to two sibling split panes peeking behind.
func (a *App) tiles(c *ui.Context, ts *themeState, t *Tab) {
	panes := t.panes()
	front := t.Focus
	var back []*Pane
	for _, p := range panes {
		if p != front {
			back = append(back, p)
		}
	}
	if len(back) > 2 {
		back = back[:2]
	}
	w := float32(tileSide + tileStep*len(back))
	ui.Box(c).Size(w, tileSide).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
		for i := len(back) - 1; i >= 0; i-- {
			x := r.X + float32(tileStep*(i+1))
			inset := float32(i+1) * 1.5
			drawToolTile(p, ts, ui.Rect{X: x, Y: r.Y + inset, W: tileSide - inset, H: tileSide - 2*inset}, paneProgram(back[i]), false, len(back) > 0)
		}
		if front != nil {
			drawToolTile(p, ts, ui.Rect{X: r.X, Y: r.Y, W: tileSide, H: tileSide}, paneProgram(front), true, len(back) > 0)
		}
	})
}

func paneProgram(p *Pane) program {
	if p.info.Idle || p.info.Program == "" {
		return programOf(p.info.Shell)
	}
	return programOf(p.info.Program)
}

// drawToolTile renders TCDesign's GlassToolTile (Indicators.swift:281):
// 22×22 rounded square (radiusTile = 6), ink(0.10) fill, 14pt tinted logo/glyph.
func drawToolTile(p *ui.Painter, ts *themeState, r ui.Rect, prog program, showGlyph, stacked bool) {
	const rad = radiusTile
	if stacked && showGlyph {
		p.Fill(r, ts.paneOpaque.Alpha(0.78), rad)
	}
	p.Fill(r, ts.inkAlpha(0.10), rad)
	if !showGlyph {
		p.Fill(r, prog.glyphTint(ts).Alpha(0.22), rad)
	}
	if ts.HC {
		p.Stroke(r, ts.edgeHC, rad, 1)
	}
	if !showGlyph {
		return
	}
	g := prog.TileGlyph
	if g == "" {
		g = prog.Glyph
	}
	const s = 13.5
	p.Icon(icon(g), ui.Rect{X: r.X + (r.W-s)/2, Y: r.Y + (r.H-s)/2, W: s, H: s}, prog.glyphTint(ts))
}

func fadeText(p *ui.Painter, r ui.Rect, spans []ui.Span) {
	key := fmt.Sprintf("%.0f|%v", r.W, spans)
	f, ok := fades[key]
	if !ok {
		f = layoutFade(p, r.W, spans)
		if len(fades) > 512 {
			clear(fades)
		}
		fades[key] = f
	}
	p.RichText(r.X, r.Y+(r.H-f.h)/2, 0, f.spans...)
}

var fades = map[string]fadeLayout{}

type fadeLayout struct {
	spans []ui.Span
	h     float32
}

func layoutFade(p *ui.Painter, width float32, spans []ui.Span) fadeLayout {
	w, h := p.MeasureText(0, spans...)
	if w <= width {
		return fadeLayout{spans, h}
	}
	r := ui.Rect{W: width}
	const fade = 28
	var out []ui.Span
	x := float32(0)
	for _, s := range spans {
		start := 0
		for i := 0; i < len(s.Text); {
			_, n := utf8.DecodeRuneInString(s.Text[i:])
			ch := s
			ch.Text = s.Text[i : i+n]
			cw, _ := p.MeasureText(0, ch)
			if x+cw > r.W-fade {
				if start < i {
					head := s
					head.Text = s.Text[start:i]
					out = append(out, head)
				}
				f := 1 - (x+cw/2-(r.W-fade))/fade
				ch.Color = ch.Color.Alpha(max(f, 0) * max(f, 0))
				if ch.Color.A < 4 {
					return fadeLayout{out, h}
				}
				out = append(out, ch)
				start = i + n
			}
			x += cw
			i += n
		}
		if start < len(s.Text) {
			tail := s
			tail.Text = s.Text[start:]
			out = append(out, tail)
		}
	}
	return fadeLayout{out, h}
}
