package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

// hostChip renders TCDesign's GlassPickerPill trigger in the top toolbar row
// and opens a floating 94%-opaque GlassPopover with host & session-server diagnostics.
func (a *App) hostChip(c *ui.Context, ts *themeState) {
	h := a.hello.Host
	name, model := h.Name, h.Model
	if name == "" {
		name, _ = os.Hostname()
		name = strings.TrimSuffix(name, ".local")
	}
	if name == "" {
		name = thisMachine()
	}
	glyph := "mac-studio"
	switch {
	case strings.Contains(model, "MacBook"), h.Portable:
		glyph = "laptop"
	case strings.Contains(model, "iMac"), runtime.GOOS == "windows":
		glyph = "monitor"
	case model == "Linux":
		glyph = "server"
	}

	tone := "positive"
	if a.err != "" {
		tone = "critical"
	}
	chip := glassPickerPill(c, ts, tone, name, "Host "+name, a.hostOpen)
	if chip.Clicked() {
		a.hostOpen = !a.hostOpen
	}

	ui.PopoverBase(c, chip, &a.hostOpen, func(panel ui.Element) {
		styleFloating(panel.Margin(space8, 0, 0, 0).Width(sizePopoverWidth).Padding(cardPaddingY, cardPaddingX).Gap(space10),
			ts, ts.popover, ts.cardRadius(), false)
		a.hostPanel(c, ts, name, glyph)
	})
}

func (a *App) hostPanel(c *ui.Context, ts *themeState, name, glyph string) {
	h := a.hello.Host
	ui.Row(c).Gap(space10).AlignItems(ui.Center).Children(func() {
		tile := ui.Box(c).Size(sizeToolTileLarge, sizeToolTileLarge).Radius(ts.controlRadius()).
			Background(ts.inkAlpha(0.10)).Center().Shrink(0)
		ts.hcEdge(tile)
		tile.Children(func() {
			ui.Icon(c, icon(glyph)).Size(18, 18).TextColor(ts.accent)
		})
		ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
			applyType(ui.Text(c, name), ts, stepBodyStrong()).TextColor(ts.textPrimary).SingleLine().Ellipsis("…")
			if a.err != "" {
				statusLabel(c, ts, "critical", thisMachine()+" · Needs attention", false)
			} else {
				statusLabel(c, ts, "positive", thisMachine()+" · Connected", false)
			}
		})
	})

	ui.Box(c).Height(1).Background(ts.hairline)

	mem := "—"
	if h.Memory > 0 {
		mem = fmt.Sprintf("%d GB", h.Memory>>30)
	}
	ui.Column(c).Gap(space6).Children(func() {
		kvRow(c, ts, "Model", h.Model, 64, false)
		kvRow(c, ts, "Chip", h.Chip, 64, false)
		kvRow(c, ts, "Memory", mem, 64, true)
		kvRow(c, ts, "System", h.OS, 64, false)
		kvRow(c, ts, "User", h.User, 64, true)
	})

	ui.Box(c).Height(1).Background(ts.hairline)

	sessions, running := 0, 0
	for _, t := range a.tabs {
		for _, p := range t.panes() {
			sessions++
			if !p.info.Idle && p.info.Program != "" {
				running++
			}
		}
	}
	serverVal := "—"
	if a.hello.PID > 0 {
		serverVal = fmt.Sprintf("pid %d · up %s", a.hello.PID, roundDur(time.Since(a.hello.Started)))
	}
	ui.Column(c).Gap(space6).Children(func() {
		kvRow(c, ts, "Server", serverVal, 64, true)
		kvRow(c, ts, "Sessions", fmt.Sprintf("%d open · %d running", sessions, running), 64, true)
	})

	ui.Row(c).Gap(space8).AlignItems(ui.Center).Children(func() {
		applyType(ui.Text(c, "Sessions persist across quits."), ts, stepCaption()).
			TextColor(ts.textTertiary).Grow(1).MinWidth(0)
		if glassButton(c, ts, "Settings…", "glass", true, GroundMenu).Clicked() {
			a.hostOpen = false
			a.settingsSection = "sessions"
			a.settingsScrollReq = "sessions"
			a.openSettings()
		}
	})
}

func roundDur(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", max(int(d.Seconds()), 0))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 48*time.Hour:
		return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}

func thisMachine() string {
	switch runtime.GOOS {
	case "darwin":
		return "This Mac"
	case "windows":
		return "This PC"
	}
	return "This computer"
}
