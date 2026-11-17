package main

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/egoist/mygo/plugins/terminal"
	"github.com/egoist/mygo/ui"
)

const (
	headerH   = 34 // Pane card header height (matches TCDesign Size.cta = 34)
	activeFor = 1500 * time.Millisecond
)

func (a *App) view(c *ui.Context) {
	a.services = c.Services()
	a.runPosted()
	ts := themeOf(c)
	a.focusedWin = a.win == nil || a.win.IsFocused()

	switch {
	case ts.ReduceTransparency:
		c.Root().Background(ts.paneOpaque)
	case ts.NativeGlass:
		// Clear root so floating 20pt Liquid Glass panes expose the real macOS
		// desktop across the 10pt paneGap and around rounded squircle corners.
		c.Root().Background(ui.Transparent)
	default:
		c.Root().Background(ts.scene)
	}

	sideInset := mainInset
	padBottom := panePadding
	if ts.Dense {
		sideInset = space10
		padBottom = space8
	}

	rootRow := ui.Row(c).Fill().Gap(paneGap).AlignItems(ui.Stretch)
	if !ts.NativeGlass {
		rootRow.Draw(func(p *ui.Painter, r ui.Rect) {
			paintSceneBackdrop(p, r, ts)
		})
	}
	rootBounds := rootRow.Bounds()

	var mainBounds, inspBounds ui.Rect
	showInspector := a.inspectorOpen

	rootRow.Children(func() {
		mainPane := styleGlassPane(ui.Column(c.Key("main-pane")).Grow(1).MinWidth(0).MinHeight(0), ts, a.focusedWin)
		mainPane.Children(func() {
			a.topToolbarRow(c, ts, sideInset)
			ui.Box(c).Margin(cardGap, 0, 0, 0).Children(func() {
				a.tabStrip(c, ts, sideInset)
			})
			stage := ui.Column(c).Grow(1).MinHeight(0).Padding(space12, sideInset, padBottom, sideInset)
			stage.Children(func() {
				if t := a.tab(); t != nil {
					a.tabContent(c, ts, t)
				}
				if a.err != "" {
					// Floating GlassPopoverNotice over the top of the tab (PopoverNotice.swift):
					// pushes nothing down, opaque under its glass so content never bleeds through.
					ui.Box(c).Absolute().Top(space12).Left(sideInset).Right(sideInset).Children(func() {
						a.popoverNotice(c, ts)
					})
				}
			})
		})
		mainBounds = mainPane.Bounds()

		if showInspector {
			inspPane := styleGlassPane(ui.Column(c.Key("inspector-pane")).Width(sizeInspectorWidth).Shrink(0).MinHeight(0), ts, a.focusedWin)
			inspPane.Children(func() {
				a.inspectorView(c, ts)
			})
			inspBounds = inspPane.Bounds()
		}
	})

	if a.win != nil {
		rad := ts.paneRadius()
		var specs []nativePaneSpec
		if mainBounds.W > 0 && mainBounds.H > 0 {
			specs = append(specs, nativePaneSpec{
				X: mainBounds.X, Y: mainBounds.Y, W: mainBounds.W, H: mainBounds.H, Radius: rad,
			})
			if showInspector && inspBounds.W > 0 && inspBounds.H > 0 {
				specs = append(specs, nativePaneSpec{
					X: inspBounds.X, Y: inspBounds.Y, W: inspBounds.W, H: inspBounds.H, Radius: rad,
				})
			}
		} else if rootBounds.W > 0 && rootBounds.H > 0 {
			specs = append(specs, nativePaneSpec{W: rootBounds.W, H: rootBounds.H, Radius: rad})
		}
		syncNativeGlassPanes(a.win.NativeHandle(), specs, ts.Dark, ts.NativeGlass, ts.glassTint)
	}

	a.toastViewport(c, ts)
	a.shortcuts(c)
	a.palette(c, ts)

	if a.saveDue && time.Since(a.lastSave) > time.Second {
		a.save()
	}
	if t := a.tab(); t != nil && a.win != nil {
		name, detail := t.label()
		if title := strings.TrimSpace(name + " " + detail); title != a.title {
			a.title = title
			a.win.SetTitle(title)
		}
	}
}

// ownControls tells that the title bar draws the window's caption controls
// (Windows frameless window).
var ownControls = runtime.GOOS == "windows"

// topToolbarRow renders MonitorMainPane's top row (MonitorWindowView.swift:865-915):
// centered at lightsCentre (26pt) alongside the macOS traffic lights inside the GlassPane,
// holding the Host picker pill, draggable window spacer, GlassToolbarGroup capsule,
// and 28pt round Settings gear button.
func (a *App) topToolbarRow(c *ui.Context, ts *themeState, sideInset float32) {
	bar := c.TitleBar()
	leftClear := sideInset
	if !ownControls && (bar.Left > 0 || isMac) {
		leftClear = max(bar.Left+10, windowControlsWidth)
	}
	rightClear := max(bar.Right, sideInset)
	topPad := max(lightsCentre-sizeControlLarge/2, panePadding)

	ui.Row(c).Height(sizeControlLarge).Margin(topPad, rightClear, 0, leftClear).
		Gap(space8).AlignItems(ui.Center).DragWindow().Children(func() {
		a.hostChip(c, ts)
		ui.Spacer(c).MinWidth(24)
		ui.Row(c).Gap(space8).AlignItems(ui.Center).Shrink(0).Children(func() {
			toolbarGroup(c, ts, func() {
				if toolbarButton(c, ts, "command", 13, 13, "Command Palette", true, a.paletteOpen).
					Tooltip(tip("Command Palette", &cmdPalette)).Clicked() {
					a.openPalette()
				}
				if toolbarButton(c, ts, "columns-2", 14, 14, "Toolbar Split Right", false, false).
					Tooltip(tip("Split Right", &cmdSplitRight)).Clicked() {
					a.split(false)
				}
				if toolbarButton(c, ts, "rows-2", 14, 14, "Toolbar Split Down", false, false).
					Tooltip(tip("Split Down", &cmdSplitDown)).Clicked() {
					a.split(true)
				}
				inspLabel := "Show Inspector"
				if a.inspectorOpen {
					inspLabel = "Hide Inspector"
				}
				if toolbarButton(c, ts, "tc-inspector", 14, 12, inspLabel, true, a.inspectorOpen).
					Tooltip(tip(inspLabel, &cmdInspector)).Clicked() {
					a.inspectorOpen = !a.inspectorOpen
				}
			})
			if glassRoundButton(c, ts, "tc-gear", 15, "Settings", false, a.settingsWin != nil, GroundPane).
				Tooltip(tip("Settings", &cmdSettings)).Clicked() {
				a.openSettings()
			}
		})
		a.windowControls(c, ts)
	})
}

// windowControls renders minimize, maximize/restore, and close on frameless Windows windows.
func (a *App) windowControls(c *ui.Context, ts *themeState) {
	win := a.win
	if !ownControls || win == nil || win.IsFullScreen() {
		return
	}
	maximized := win.IsMaximized()
	ui.Row(c).Gap(2).AlignItems(ui.Center).Shrink(0).Children(func() {
		ui.Box(c).Size(1, 16).Margin(0, 6).Background(ts.rule)
		if iconButton(c, ts, "minus", "Minimize", 30, 16).Tooltip("Minimize").Clicked() {
			win.Minimize()
		}
		glyph, label := "square", "Maximize"
		if maximized {
			glyph, label = "copy", "Restore"
		}
		if iconButton(c, ts, glyph, label, 30, 13).Tooltip(label).Clicked() {
			if maximized {
				win.Unmaximize()
			} else {
				win.Maximize()
			}
		}
		b := ui.Box(c).Size(30, 30).Center().Radius(radiusPill).
			Cursor(ui.CursorPointer).Role(ui.RoleButton).Label("Close").Tooltip("Close")
		col := ts.toolbarGlyph
		if b.Pressed() {
			b.Background(closeRed.Alpha(0.82))
			col = ui.Hex("#ffffff")
		} else if b.Hovered() {
			b.Background(closeRed)
			col = ui.Hex("#ffffff")
		}
		b.Transition(ui.ElementTransition{Colors: true, Duration: durFast, Ease: easeStandard})
		b.Children(func() {
			ui.Icon(c, icon("x")).Size(16, 16).TextColor(col)
		})
		if b.Clicked() {
			win.Close()
		}
	})
}

var closeRed = ui.Hex("#c42b1c")

func tip(label string, cmd *command) string {
	if cmd == nil || cmd.Keys == "" {
		return label
	}
	return label + "  " + cmd.Keys
}

// tabContent lays out the split pane tree of a tab, or its zoomed pane.
func (a *App) tabContent(c *ui.Context, ts *themeState, t *Tab) {
	if t.Zoom != nil {
		a.paneCard(c, ts, t, t.Zoom).Grow(1)
		return
	}
	a.node(c, ts, t, t.Root).Grow(1)
}

// node lays out a branch or leaf of the split tree.
func (a *App) node(c *ui.Context, ts *themeState, t *Tab, n *Node) ui.Element {
	if n.Pane != nil {
		return a.paneCard(c, ts, t, n.Pane)
	}
	splitGap := cardGap
	if ts.Dense {
		splitGap = space6
	}
	var box ui.Element
	if n.Vertical {
		box = ui.Column(c.Key(n.ID))
	} else {
		box = ui.Row(c.Key(n.ID))
	}
	box.MinWidth(0).MinHeight(0).AlignItems(ui.Stretch)
	bounds := box.Bounds()
	first, second, ratio := n.A, n.B, n.Ratio
	box.Children(func() {
		a.node(c, ts, t, first).Grow(ratio).Basis(0).MinWidth(0).MinHeight(0)
		div := ui.Box(c.Key("divider")).Role(ui.RoleSplitter).Label("Divider")
		if n.Vertical {
			div.Height(splitGap).Cursor(ui.CursorResizeRow)
		} else {
			div.Width(splitGap).Cursor(ui.CursorResizeColumn)
		}
		if dx, dy, ok := div.Dragged(); ok {
			total := bounds.W - splitGap
			d := dx
			if n.Vertical {
				total, d = bounds.H-splitGap, dy
			}
			if total > 0 {
				n.Ratio = min(max(n.Ratio+d/total, 0.08), 0.92)
				a.changed()
			}
		}
		if div.DoubleClicked() {
			n.Ratio = 0.5
			a.changed()
		}
		active := div.Dragging() || div.Pressed()
		if div.Hovered() || active {
			handleCol := ts.neutral.Alpha(0.55)
			if active {
				handleCol = ts.accent
			}
			div.Draw(func(p *ui.Painter, r ui.Rect) {
				if n.Vertical {
					p.Fill(ui.Rect{X: r.X + r.W/2 - 18, Y: r.Y + r.H/2 - 1.5, W: 36, H: 3}, handleCol, 1.5)
				} else {
					p.Fill(ui.Rect{X: r.X + r.W/2 - 1.5, Y: r.Y + r.H/2 - 18, W: 3, H: 36}, handleCol, 1.5)
				}
			})
		}
		a.node(c, ts, t, second).Grow(1 - ratio).Basis(0).MinWidth(0).MinHeight(0)
	})
	return box
}

// paneCard renders a terminal pane as a painted TCDesign GlassCard (Surfaces.swift):
// 35% translucent card fill over the pane's Liquid Glass (never glass-on-glass),
// radius-card = 16, 1px hairline under its header, and transparent terminal canvas.
func (a *App) paneCard(c *ui.Context, ts *themeState, t *Tab, p *Pane) ui.Element {
	focused := t.Focus == p
	multi := len(t.panes()) > 1 && t.Zoom == nil
	card := styleGlassCard(ui.Column(c.Key(p.ID)).Clip().MinWidth(0).MinHeight(0), ts, multi && !focused)
	card.Transition(ui.ElementTransition{Colors: true, Duration: durReveal, Ease: easeStandard})

	hovered := card.Hovered()
	card.Children(func() {
		a.paneHeader(c, ts, t, p, focused, hovered)
		ui.Box(c).Height(1).Background(ts.hairline).Shrink(0)
		padX, padBottom := float32(10), float32(8)
		if ts.Dense {
			padX, padBottom = 6, 6
		}
		body := ui.Box(c).Grow(1).MinHeight(0).Padding(6, padX, padBottom, padX)
		body.Children(func() {
			if p.term == nil {
				return
			}
			tv := terminal.View(c, p.term).Fill()
			if a.focusReq == p && a.tab() == t {
				tv.Focus()
				a.focusReq = nil
			}
			if tv.Focused() && t.Focus != p {
				t.setFocus(p)
				a.changed()
			}
			if tv.Focused() {
				p.attention = false
			}
		})
	})
	p.bounds = card.Bounds()
	if card.Pressed() && t.Focus != p {
		t.setFocus(p)
		a.focusReq = p
	}
	return card
}

// paneHeader renders the pane's foreground program glyph, title, working directory,
// semantic StatusLabel, and 26×26 header actions.
func (a *App) paneHeader(c *ui.Context, ts *themeState, t *Tab, p *Pane, focused, hovered bool) {
	name, detail := p.label()
	prog := programOf(p.info.Program)
	if p.info.Idle || p.info.Program == "" {
		prog = programOf(p.info.Shell)
	}
	h := ui.Row(c).Height(headerH).Padding(0, 8, 0, 12).Gap(space8).AlignItems(ui.Center).MinWidth(0)
	if h.DoubleClicked() {
		t.setFocus(p)
		a.toggleZoom()
	}
	if h.Clicked() {
		t.setFocus(p)
		a.focusReq = p
	}
	h.Children(func() {
		glyphCol := ts.textSecondary
		titleCol := ts.textSecondary
		if focused {
			glyphCol = prog.glyphTint(ts)
			titleCol = ts.textPrimary
		}
		ui.Icon(c, icon(prog.Glyph)).Size(15, 15).TextColor(glyphCol).Shrink(0)
		ui.Row(c).Grow(1).MinWidth(0).Gap(inlineGap).AlignItems(ui.Center).ClipX().Children(func() {
			applyType(ui.Text(c, name), ts, stepBodyStrong()).TextColor(titleCol).
				SingleLine().Ellipsis("…").Shrink(0).MaxWidthPercent(75)
			if detail != "" {
				applyType(ui.Text(c, detail), ts, stepMono()).TextColor(ts.textTertiary).
					SingleLine().Ellipsis("…").Shrink(1).MinWidth(0)
			}
			a.activityBadge(c, ts, p)
		})
		show := focused || hovered
		ui.Row(c).Gap(1).AlignItems(ui.Center).Shrink(0).Opacity(map[bool]float32{true: 1, false: 0}[show]).Children(func() {
			if !show {
				return
			}
			if iconButton(c, ts, "columns-2", "Split Right", sizeControl, 15).Tooltip(tip("Split Right", &cmdSplitRight)).Clicked() {
				t.setFocus(p)
				a.split(false)
			}
			if iconButton(c, ts, "rows-2", "Split Down", sizeControl, 15).Tooltip(tip("Split Down", &cmdSplitDown)).Clicked() {
				t.setFocus(p)
				a.split(true)
			}
			zoom, label := "maximize-2", "Zoom"
			if t.Zoom == p {
				zoom, label = "minimize-2", "Unzoom"
			}
			if iconButton(c, ts, zoom, "Zoom", sizeControl, 15).Tooltip(tip(label, &cmdZoom)).Clicked() {
				t.setFocus(p)
				a.toggleZoom()
			}
			if iconButton(c, ts, "x", "Close Pane", sizeControl, 15).Tooltip(tip("Close Pane", &cmdClosePane)).Clicked() {
				a.later(c, func() { a.closePane(p) })
			}
		})
	})
}

type status int

const (
	statusIdle      status = iota // Shell waiting at prompt
	statusRunning                 // Foreground program actively printing
	statusQuiet                   // Foreground program running quietly
	statusAttention               // Finished or rang bell while out of sight
)

func (a *App) statusOf(c *ui.Context, p *Pane) status {
	if p.attention {
		return statusAttention
	}
	if p.info.Idle || p.info.Program == "" || programOf(p.info.Program).Shell {
		return statusIdle
	}
	last := time.Unix(0, p.lastData.Load())
	if since := c.Now().Sub(last); since < activeFor {
		c.After(activeFor - since)
		return statusRunning
	}
	return statusQuiet
}

// activityBadge renders TCDesign's GlassStatusLabel (dot + plain words, never pulsing).
func (a *App) activityBadge(c *ui.Context, ts *themeState, p *Pane) {
	switch a.statusOf(c, p) {
	case statusRunning:
		statusLabel(c, ts, "positive", "Running", false)
	case statusAttention:
		statusLabel(c, ts, "caution", "Needs a look", false)
	}
}

// popoverNotice renders TCDesign's GlassPopoverNotice (PopoverNotice.swift, 50-feedback.css):
// an opaque plate floating over the top of the active tab with shadow-float,
// leading 30pt GlassNoticeMark coin, bodyStrong title, caption summary, capsule CTA,
// link Dismiss, and 26pt small GlassRoundButton close X.
func (a *App) popoverNotice(c *ui.Context, ts *themeState) {
	rad := ts.cardRadius()
	sh := ts.shadowFloat
	card := ui.Row(c).Padding(space12).Gap(space10).Radius(rad).
		Background(ts.paneOpaque).Shadow(0, sh.y, sh.blur, 0, sh.color).
		AlignItems(ui.Start).Role(ui.RoleStatus)
	ts.hcEdge(card)

	card.Children(func() {
		// GlassNoticeMark coin: 30×30 paneOpaque circle with 2pt critical border & 12pt icon.
		coin := ui.Box(c).Size(30, 30).Radius(radiusPill).Background(ts.paneOpaque).
			Border(2, ts.critical).Center().Shrink(0)
		coin.Children(func() {
			ui.Icon(c, icon("alert-triangle")).Size(13, 13).TextColor(ts.critical)
		})

		ui.Column(c).Grow(1).MinWidth(0).Gap(space6).Children(func() {
			applyType(ui.Text(c, "Session server needs attention."), ts, stepBodyStrong()).TextColor(ts.textPrimary)
			applyType(ui.Text(c, a.err), ts, stepCaption()).TextColor(ts.textSecondary)
			ui.Row(c).Gap(space8).Padding(space2, 0, 0, 0).AlignItems(ui.Center).Children(func() {
				if strings.Contains(strings.ToLower(a.err), "reconnect") || strings.Contains(strings.ToLower(a.err), "lost") {
					if glassButton(c, ts, "Reconnect", "primary", true, GroundMenu).Clicked() {
						a.reconnect()
					}
				}
				if linkAction(c, ts, "Dismiss").Clicked() {
					a.err = ""
				}
			})
		})

		if glassRoundButton(c, ts, "x", 11, "Close notice", true, false, GroundMenu).Clicked() {
			a.err = ""
		}
	})
}

// inspectorView renders the floating 300pt right-hand Inspector GlassPane
// (ThreePane.swift + MonitorWindowView.swift:403), separated from the main GlassPane
// by paneGap = 10pt of clear desktop.
func (a *App) inspectorView(c *ui.Context, ts *themeState) {
	ui.Scroll(c).Fill().Padding(inspectorPaddingV, inspectorPaddingH).Children(func() {
		ui.Column(c).Gap(space14).MinWidth(0).Children(func() {
			// Inspector header
			ui.Row(c).AlignItems(ui.Center).Gap(space8).Children(func() {
				applyType(ui.Text(c, "Inspector"), ts, stepTitle()).TextColor(ts.textPrimary).Grow(1)
				if glassRoundButton(c, ts, "x", 11, "Close Inspector", true, false, GroundPane).Clicked() {
					a.inspectorOpen = false
				}
			})

			// Focused Session card
			if t := a.tab(); t != nil && t.Focus != nil {
				p := t.Focus
				name, dir := p.label()
				prog := paneProgram(p)
				glassSectionRule(c, ts, "FOCUSED SESSION")
				card := styleGlassCard(ui.Column(c).Padding(cardPaddingY, cardPaddingX).Gap(space10), ts, false)
				card.Children(func() {
					ui.Row(c).Gap(space10).AlignItems(ui.Center).Children(func() {
						ui.Box(c).Size(sizeToolTileLarge, sizeToolTileLarge).Shrink(0).Draw(func(pt *ui.Painter, r ui.Rect) {
							pt.Fill(r, ts.inkAlpha(0.10), ts.controlRadius())
							g := prog.TileGlyph
							if g == "" {
								g = prog.Glyph
							}
							pt.Icon(icon(g), ui.Rect{X: r.X + 7, Y: r.Y + 7, W: 16, H: 16}, prog.glyphTint(ts))
						})
						ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
							applyType(ui.Text(c, name), ts, stepBodyStrong()).TextColor(ts.textPrimary).SingleLine().Ellipsis("…")
							switch a.statusOf(c, p) {
							case statusRunning:
								statusLabel(c, ts, "positive", "Running", false)
							case statusAttention:
								statusLabel(c, ts, "caution", "Needs a look", false)
							default:
								statusLabel(c, ts, "neutral", "Idle at prompt", false)
							}
						})
					})
					ui.Box(c).Height(1).Background(ts.hairline)
					grid := "—"
					if p.term != nil {
						cols, rows := p.term.Size()
						grid = fmt.Sprintf("%d × %d", cols, rows)
					}
					if dir == "" {
						dir = shortDir(p.info.Dir)
					}
					ui.Column(c).Gap(space6).Children(func() {
						kvRow(c, ts, "Shell", p.info.Shell, 62, true)
						kvRow(c, ts, "Folder", dir, 62, true)
						kvRow(c, ts, "Grid", grid, 62, true)
						kvRow(c, ts, "Session", p.SID, 62, true)
					})
					ui.Row(c).Gap(space6).Padding(space2, 0, 0, 0).AlignItems(ui.Center).Children(func() {
						if glassButton(c, ts, "Split Right", "glass", true, GroundPane).Clicked() {
							a.split(false)
						}
						if glassButton(c, ts, "Restart", "glass", true, GroundPane).Clicked() {
							a.restartFocused()
						}
					})
				})
			}

			// Open Panes list across tabs (TCDesign GlassListRow)
			glassSectionRule(c, ts, "OPEN PANES")
			listCard := styleGlassCard(ui.Column(c).Padding(space6).Gap(2), ts, false)
			listCard.Children(func() {
				for ti, tab := range a.tabs {
					for _, p := range tab.panes() {
						ti, tab, p := ti, tab, p
						selected := ti == a.active && tab.Focus == p
						pname, pdir := p.label()
						prog := paneProgram(p)
						row := ui.Row(c.Key(fmt.Sprintf("insp-%d-%d", tab.ID, p.ID))).
							Height(34).Padding(0, space8).Gap(space8).Radius(ts.controlRadius()).
							AlignItems(ui.Center).Cursor(ui.CursorPointer)
						fg, subFg := ts.textPrimary, ts.textTertiary
						if selected {
							row.Background(ts.selection)
							fg = ts.textOnAccent
							subFg = ts.textOnAccent.Alpha(0.80)
						} else if row.Hovered() {
							row.Background(ts.rowHover)
						}
						if row.Clicked() {
							a.selectTab(ti)
							tab.setFocus(p)
							a.focusReq = p
						}
						row.Children(func() {
							ui.Box(c).Size(sizeToolTile, sizeToolTile).Shrink(0).Draw(func(pt *ui.Painter, r ui.Rect) {
								if selected {
									pt.Fill(r, ts.controlOnSelection, radiusTile)
								} else {
									pt.Fill(r, ts.inkAlpha(0.10), radiusTile)
								}
								g := prog.TileGlyph
								if g == "" {
									g = prog.Glyph
								}
								col := prog.glyphTint(ts)
								if selected {
									col = ts.textOnAccent
								}
								pt.Icon(icon(g), ui.Rect{X: r.X + 4.5, Y: r.Y + 4.5, W: 13, H: 13}, col)
							})
							applyType(ui.Text(c, pname), ts, stepLabel()).FontWeight(500).
								TextColor(fg).SingleLine().Ellipsis("…").Grow(1).MinWidth(0)
							if pdir != "" {
								applyType(ui.Text(c, pdir), ts, stepMono()).
									TextColor(subFg).SingleLine().Ellipsis("…").Shrink(1).MaxWidth(96)
							}
						})
					}
				}
			})

			// Persistent Session Server summary
			glassSectionRule(c, ts, "SESSION SERVER")
			srvCard := styleGlassCard(ui.Column(c).Padding(cardPaddingY, cardPaddingX).Gap(space6), ts, true)
			srvCard.Children(func() {
				pid, up := "—", "—"
				if a.hello.PID > 0 {
					pid = fmt.Sprintf("%d", a.hello.PID)
					up = roundDur(time.Since(a.hello.Started))
				}
				kvRow(c, ts, "Server PID", pid, 72, true)
				kvRow(c, ts, "Uptime", up, 72, true)
				kvRow(c, ts, "Host", a.hello.Host.Name, 72, false)
			})
		})
	})
}

// toastViewport styles transient notifications to Near AI OS .nr-toast:
// bottom-leading 24px inset, floating 94%-opaque popover blur + shadow-float,
// StatusLabel dot + capsule action + small close button.
func (a *App) toastViewport(c *ui.Context, ts *themeState) {
	ui.ToastViewportBase(c, func(viewport ui.Element, toasts []ui.Toast) {
		viewport.Padding(space24).AlignItems(ui.Start).Gap(space8)
		for _, t := range toasts {
			item := ui.ToastBase(c, t)
			styleFloating(item.Root.Row().Gap(space10).Padding(cardPaddingY, cardPaddingX).MaxWidth(450).AlignItems(ui.Center),
				ts, ts.popover, ts.cardRadius(), false)
			tone := "positive"
			switch t.Type {
			case "error", "critical":
				tone = "critical"
			case "warning", "caution":
				tone = "caution"
			case "info", "neutral":
				tone = "neutral"
			}
			item.Root.Children(func() {
				ui.Box(c).Size(8, 8).Radius(radiusPill).Background(ts.toneDotColor(tone)).Shrink(0)
				ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
					applyType(ui.Text(c, t.Title), ts, stepBodyStrong()).TextColor(ts.textPrimary)
					if t.Description != "" {
						applyType(ui.Text(c, t.Description), ts, stepCaption()).TextColor(ts.textSecondary)
					}
				})
				if t.Action != "" {
					btn := item.ActionButton().Height(sizeControlLarge).Padding(0, 12).Radius(radiusPill).
						Background(ts.controlOnMenu).AlignItems(ui.Center).Cursor(ui.CursorPointer)
					ts.hcEdge(btn)
					btn.Children(func() {
						applyType(ui.Text(c, t.Action), ts, stepLabel()).TextColor(ts.textPrimary)
					})
				}
				closeBtn := item.CloseButton().Size(sizeControl, sizeControl).Center().
					Radius(radiusPill).Background(ts.controlOnMenu).Cursor(ui.CursorPointer).Label("Dismiss notification")
				closeBtn.Children(func() {
					ui.Icon(c, icon("x")).Size(11, 11).TextColor(ts.textPrimary)
				})
			})
		}
	})
}
