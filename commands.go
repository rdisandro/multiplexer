package main

import (
	"fmt"
	"runtime"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// command is an action of the menu bar and the command palette.
type command struct {
	Title string
	// Accel is the menu item's shortcut, and Keys how the palette shows it.
	Accel, Keys string
	// Other is the shortcut elsewhere than on macOS.
	Other  shortcut
	Run    func(a *App)
	Hidden bool
}

var (
	cmdNewTab       = command{Title: "New Tab", Accel: "CmdOrCtrl+T", Keys: "⌘T", Other: ctrlShift(ui.KeyT), Run: func(a *App) { a.newTab(a.currentDir()) }}
	cmdSplitRight   = command{Title: "Split Right", Accel: "CmdOrCtrl+D", Keys: "⌘D", Other: ctrlShift(ui.KeyD), Run: func(a *App) { a.split(false) }}
	cmdSplitDown    = command{Title: "Split Down", Accel: "CmdOrCtrl+Shift+D", Keys: "⇧⌘D", Other: ctrlShift(ui.KeyE), Run: func(a *App) { a.split(true) }}
	cmdInspector    = command{Title: "Toggle Inspector", Accel: "CmdOrCtrl+Shift+I", Keys: "⇧⌘I", Other: ctrlShift(ui.KeyI), Run: func(a *App) { a.inspectorOpen = !a.inspectorOpen }}
	cmdClosePane    = command{Title: "Close Pane", Accel: "CmdOrCtrl+W", Keys: "⌘W", Other: ctrlShift(ui.KeyW), Run: func(a *App) { a.closeFocused() }}
	cmdCloseTab     = command{Title: "Close Tab", Accel: "CmdOrCtrl+Shift+W", Keys: "⇧⌘W", Run: func(a *App) { a.closeActiveTab() }}
	cmdZoom         = command{Title: "Zoom Pane", Accel: "CmdOrCtrl+Shift+Enter", Keys: "⇧⌘↩", Other: ctrlShift(ui.KeyEnter), Run: func(a *App) { a.toggleZoom() }}
	cmdEqualize     = command{Title: "Equalize Panes", Accel: "CmdOrCtrl+Ctrl+=", Keys: "⌃⌘=", Run: func(a *App) { a.equalize() }}
	cmdNextTab      = command{Title: "Next Tab", Accel: "CmdOrCtrl+Shift+]", Keys: "⇧⌘]", Other: shortcut{ui.Ctrl, ui.KeyPageDown}, Run: func(a *App) { a.cycleTab(1) }}
	cmdPrevTab      = command{Title: "Previous Tab", Accel: "CmdOrCtrl+Shift+[", Keys: "⇧⌘[", Other: shortcut{ui.Ctrl, ui.KeyPageUp}, Run: func(a *App) { a.cycleTab(-1) }}
	cmdRenameTab    = command{Title: "Rename Tab…", Accel: "CmdOrCtrl+Shift+R", Keys: "⇧⌘R", Other: ctrlShift(ui.KeyR), Run: func(a *App) { a.startRename() }}
	cmdSettings     = command{Title: "Settings…", Accel: "CmdOrCtrl+,", Keys: "⌘,", Other: shortcut{ui.Ctrl, ui.KeyComma}}
	cmdPalette      = command{Title: "Command Palette…", Accel: "CmdOrCtrl+Shift+P", Keys: "⇧⌘P", Other: ctrlShift(ui.KeyP), Run: func(a *App) { a.openPalette() }, Hidden: true}
	cmdPalette2     = command{Title: "Go to Pane…", Accel: "CmdOrCtrl+P", Keys: "⌘P", Run: func(a *App) { a.openPalette() }, Hidden: true}
	cmdFocusLeft    = command{Title: "Focus Pane Left", Accel: "CmdOrCtrl+Alt+Left", Keys: "⌥⌘←", Other: shortcut{ui.Alt, ui.KeyLeft}, Run: func(a *App) { a.moveFocus(-1, 0) }}
	cmdFocusRight   = command{Title: "Focus Pane Right", Accel: "CmdOrCtrl+Alt+Right", Keys: "⌥⌘→", Other: shortcut{ui.Alt, ui.KeyRight}, Run: func(a *App) { a.moveFocus(1, 0) }}
	cmdFocusUp      = command{Title: "Focus Pane Above", Accel: "CmdOrCtrl+Alt+Up", Keys: "⌥⌘↑", Other: shortcut{ui.Alt, ui.KeyUp}, Run: func(a *App) { a.moveFocus(0, -1) }}
	cmdFocusDown    = command{Title: "Focus Pane Below", Accel: "CmdOrCtrl+Alt+Down", Keys: "⌥⌘↓", Other: shortcut{ui.Alt, ui.KeyDown}, Run: func(a *App) { a.moveFocus(0, 1) }}
	cmdGrowLeft     = command{Title: "Move Divider Left", Accel: "CmdOrCtrl+Ctrl+Left", Keys: "⌃⌘←", Other: shortcut{ui.Alt | ui.Shift, ui.KeyLeft}, Run: func(a *App) { a.resizeFocused(-1, 0) }}
	cmdGrowRight    = command{Title: "Move Divider Right", Accel: "CmdOrCtrl+Ctrl+Right", Keys: "⌃⌘→", Other: shortcut{ui.Alt | ui.Shift, ui.KeyRight}, Run: func(a *App) { a.resizeFocused(1, 0) }}
	cmdGrowUp       = command{Title: "Move Divider Up", Accel: "CmdOrCtrl+Ctrl+Up", Keys: "⌃⌘↑", Other: shortcut{ui.Alt | ui.Shift, ui.KeyUp}, Run: func(a *App) { a.resizeFocused(0, -1) }}
	cmdGrowDown     = command{Title: "Move Divider Down", Accel: "CmdOrCtrl+Ctrl+Down", Keys: "⌃⌘↓", Other: shortcut{ui.Alt | ui.Shift, ui.KeyDown}, Run: func(a *App) { a.resizeFocused(0, 1) }}
	cmdClear        = command{Title: "Clear Screen and Scrollback", Accel: "CmdOrCtrl+K", Keys: "⌘K", Other: ctrlShift(ui.KeyK), Run: func(a *App) { a.clearFocused() }, Hidden: true}
	cmdClearScroll  = command{Title: "Clear Scrollback", Keys: "⌘K", Run: func(a *App) { a.clearFocused() }}
	cmdRestart      = command{Title: "Restart Shell in Pane", Run: func(a *App) { a.restartFocused() }}
	cmdBigger       = command{Title: "Bigger Text", Accel: "CmdOrCtrl+=", Keys: "⌘+", Other: shortcut{ui.Ctrl, ui.KeyEqual}, Run: func(a *App) { a.setFontSize(termFont.Size + 1) }}
	cmdSmaller      = command{Title: "Smaller Text", Accel: "CmdOrCtrl+-", Keys: "⌘−", Other: shortcut{ui.Ctrl, ui.KeyMinus}, Run: func(a *App) { a.setFontSize(termFont.Size - 1) }}
	cmdActualSize   = command{Title: "Actual Size", Accel: "CmdOrCtrl+0", Keys: "⌘0", Other: shortcut{ui.Ctrl, ui.Key0}, Run: func(a *App) { a.setFontSize(defaultFontSize) }}
	cmdLight        = command{Title: "Appearance: Light", Run: func(a *App) { a.setAppearance("light") }}
	cmdDark         = command{Title: "Appearance: Dark", Run: func(a *App) { a.setAppearance("dark") }}
	cmdSystem       = command{Title: "Appearance: System", Run: func(a *App) { a.setAppearance("") }}
	cmdThemeNear    = command{Title: "Theme: NEAR AI", Run: func(a *App) { a.setProductTheme("") }}
	cmdThemeTrace   = command{Title: "Theme: Trace", Run: func(a *App) { a.setProductTheme("trace") }}
	cmdThemeClassic = command{Title: "Theme: Classic Glass", Run: func(a *App) { a.setProductTheme("classic") }}
	cmdCompact      = command{Title: "Density: Compact", Run: func(a *App) { a.setDensity("") }}
	cmdDense        = command{Title: "Density: Dense", Run: func(a *App) { a.setDensity("dense") }}
	cmdEndAll       = command{Title: "Quit and End All Sessions", Accel: "CmdOrCtrl+Alt+Q", Keys: "⌥⌘Q", Run: func(a *App) { a.quitAndEnd() }}
)

// paletteCommands lists every menu/toolbar capability in the Command Palette.
var (
	paletteCommands []*command
	commands        []*command
)

func init() {
	cmdSettings.Run = (*App).openSettings
	paletteCommands = []*command{
		&cmdNewTab, &cmdSplitRight, &cmdSplitDown, &cmdInspector, &cmdZoom, &cmdEqualize, &cmdClosePane, &cmdCloseTab,
		&cmdNextTab, &cmdPrevTab, &cmdRenameTab,
		&cmdFocusLeft, &cmdFocusRight, &cmdFocusUp, &cmdFocusDown,
		&cmdGrowLeft, &cmdGrowRight, &cmdGrowUp, &cmdGrowDown,
		&cmdSettings,
		&cmdBigger, &cmdSmaller, &cmdActualSize,
		&cmdLight, &cmdDark, &cmdSystem,
		&cmdThemeNear, &cmdThemeTrace, &cmdThemeClassic,
		&cmdCompact, &cmdDense,
		&cmdClearScroll, &cmdRestart, &cmdEndAll,
	}
	commands = append(slices.Clone(paletteCommands), &cmdPalette, &cmdPalette2, &cmdClear)
	if runtime.GOOS == "darwin" {
		return
	}
	for _, cmd := range commands {
		cmd.Accel, cmd.Keys = "", ""
		if cmd.Other != (shortcut{}) {
			cmd.Accel = cmd.Other.String()
			cmd.Keys = cmd.Accel
		}
	}
}

type shortcut struct {
	Mods ui.Modifiers
	Key  ui.Key
}

func ctrlShift(k ui.Key) shortcut { return shortcut{ui.Ctrl | ui.Shift, k} }

func showTabKey(n int) shortcut { return ctrlShift(ui.Key0 + ui.Key(n)) }

func (s shortcut) String() string {
	var parts []string
	for _, m := range []struct {
		mod  ui.Modifiers
		name string
	}{{ui.Ctrl, "Ctrl"}, {ui.Alt, "Alt"}, {ui.Shift, "Shift"}} {
		if s.Mods&m.mod != 0 {
			parts = append(parts, m.name)
		}
	}
	var key string
	switch k := s.Key; {
	case k >= ui.KeyA && k <= ui.KeyZ:
		key = string(rune('A' + k - ui.KeyA))
	case k >= ui.Key0 && k <= ui.Key9:
		key = string(rune('0' + k - ui.Key0))
	default:
		key = map[ui.Key]string{
			ui.KeyEnter: "Enter", ui.KeyPageUp: "PageUp", ui.KeyPageDown: "PageDown",
			ui.KeyLeft: "Left", ui.KeyRight: "Right", ui.KeyUp: "Up", ui.KeyDown: "Down",
			ui.KeyEqual: "=", ui.KeyMinus: "-", ui.KeyComma: ",",
		}[k]
	}
	return strings.Join(append(parts, key), "+")
}

// menu builds the native macOS / cross-platform menu bar following
// Near AI OS 11-navigation.md (App, File, Edit, View, Tabs, Window, Help).
func (a *App) menu() *mygo.Menu {
	item := func(cmd *command) *mygo.MenuItem {
		return &mygo.MenuItem{Label: cmd.Title, Accelerator: cmd.Accel, Click: func(_ *mygo.MenuItem, win *mygo.Window) {
			if cmd == &cmdClosePane && win != nil && win == a.settingsWin {
				win.Close()
				return
			}
			a.run(cmd)
		}}
	}

	tabItems := []*mygo.MenuItem{item(&cmdNextTab), item(&cmdPrevTab), mygo.Separator()}
	for i := 1; i <= 9; i++ {
		i := i
		accel := fmt.Sprintf("CmdOrCtrl+%d", i)
		if runtime.GOOS != "darwin" {
			accel = showTabKey(i).String()
		}
		tabItems = append(tabItems, &mygo.MenuItem{
			Label: fmt.Sprintf("Show Tab %d", i), Accelerator: accel,
			Click: func(*mygo.MenuItem, *mygo.Window) {
				a.do(func() { a.showTab(i) })
			},
		})
	}

	appearance := func(label, v string) *mygo.MenuItem {
		return &mygo.MenuItem{Label: label, Type: mygo.MenuItemRadio, Checked: prefs.Appearance == v, Click: func(*mygo.MenuItem, *mygo.Window) {
			a.setAppearance(v)
		}}
	}
	productTheme := func(label, v string) *mygo.MenuItem {
		return &mygo.MenuItem{Label: label, Type: mygo.MenuItemRadio, Checked: prefs.ProductTheme == v, Click: func(*mygo.MenuItem, *mygo.Window) {
			a.setProductTheme(v)
		}}
	}
	density := func(label, v string) *mygo.MenuItem {
		return &mygo.MenuItem{Label: label, Type: mygo.MenuItemRadio, Checked: prefs.Density == v, Click: func(*mygo.MenuItem, *mygo.Window) {
			a.setDensity(v)
		}}
	}

	fileSubmenu := []*mygo.MenuItem{
		item(&cmdNewTab),
		item(&cmdSplitRight),
		item(&cmdSplitDown),
		mygo.Separator(),
		item(&cmdRestart),
		mygo.Separator(),
		item(&cmdClosePane),
		item(&cmdCloseTab),
	}
	if runtime.GOOS != "darwin" {
		fileSubmenu = append(fileSubmenu,
			mygo.Separator(),
			item(&cmdSettings),
			mygo.Separator(),
			item(&cmdEndAll),
			&mygo.MenuItem{Role: mygo.RoleQuit},
		)
	}

	var top []*mygo.MenuItem
	if runtime.GOOS == "darwin" {
		top = append(top, &mygo.MenuItem{
			Label: "Multiplexer",
			Submenu: []*mygo.MenuItem{
				{Role: mygo.RoleAbout},
				mygo.Separator(),
				item(&cmdSettings),
				mygo.Separator(),
				{Role: mygo.RoleServices},
				mygo.Separator(),
				{Role: mygo.RoleHide},
				{Role: mygo.RoleHideOthers},
				{Role: mygo.RoleUnhide},
				mygo.Separator(),
				item(&cmdEndAll),
				{Role: mygo.RoleQuit},
			},
		})
	}
	top = append(top,
		&mygo.MenuItem{Label: "File", Submenu: fileSubmenu},
		&mygo.MenuItem{Label: "Edit", Submenu: []*mygo.MenuItem{
			{Role: mygo.RoleUndo},
			{Role: mygo.RoleRedo},
			mygo.Separator(),
			{Role: mygo.RoleCut},
			{Role: mygo.RoleCopy},
			{Role: mygo.RolePaste},
			{Role: mygo.RoleSelectAll},
			mygo.Separator(),
			item(&cmdClear),
		}},
		&mygo.MenuItem{Label: "View", Submenu: []*mygo.MenuItem{
			item(&cmdPalette),
			item(&cmdPalette2),
			mygo.Separator(),
			item(&cmdInspector),
			item(&cmdZoom),
			item(&cmdEqualize),
			mygo.Separator(),
			{Label: "Focus", Submenu: []*mygo.MenuItem{item(&cmdFocusLeft), item(&cmdFocusRight), item(&cmdFocusUp), item(&cmdFocusDown)}},
			{Label: "Resize", Submenu: []*mygo.MenuItem{item(&cmdGrowLeft), item(&cmdGrowRight), item(&cmdGrowUp), item(&cmdGrowDown)}},
			mygo.Separator(),
			{Label: "Appearance", Submenu: []*mygo.MenuItem{
				appearance("System", ""),
				appearance("Light", "light"),
				appearance("Dark", "dark"),
			}},
			{Label: "Theme", Submenu: []*mygo.MenuItem{
				productTheme("NEAR AI", ""),
				productTheme("Trace", "trace"),
				productTheme("Classic Glass", "classic"),
			}},
			{Label: "Density", Submenu: []*mygo.MenuItem{
				density("Compact", ""),
				density("Dense", "dense"),
			}},
			mygo.Separator(),
			item(&cmdBigger),
			item(&cmdSmaller),
			item(&cmdActualSize),
			mygo.Separator(),
			{Role: mygo.RoleToggleFullScreen},
			{Role: mygo.RoleToggleDevTools, Label: "Web / UI Inspector", Hidden: !mygo.IsDev()},
		}},
		&mygo.MenuItem{Label: "Tabs", Submenu: append(tabItems, mygo.Separator(), item(&cmdRenameTab))},
		&mygo.MenuItem{Role: mygo.RoleWindowMenu},
		&mygo.MenuItem{Role: mygo.RoleHelp, Label: "Help", Submenu: []*mygo.MenuItem{
			{Label: "About Multiplexer & Persistent Sessions", Click: func(*mygo.MenuItem, *mygo.Window) {
				mygo.RunOnMain(func() {
					a.settingsSection = "about"
					a.settingsScrollReq = "about"
					a.openSettings()
				})
			}},
		}},
	)
	return mygo.NewMenu(top)
}

func (a *App) do(fn func()) {
	if a.win == nil {
		return
	}
	a.win.Update(fn)
}

// run executes a menu command on the main thread. Settings and New Tab work
// even when the main terminal window has been closed on macOS.
func (a *App) run(cmd *command) {
	if cmd == &cmdSettings {
		mygo.RunOnMain(a.openSettings)
		return
	}
	if a.win == nil {
		if cmd == &cmdNewTab {
			mygo.RunOnMain(a.open)
		}
		return
	}
	a.do(func() {
		if a.win != nil {
			cmd.Run(a)
		}
	})
}

func (a *App) closeFocused() {
	if t := a.tab(); t != nil && t.Focus != nil {
		a.closePane(t.Focus)
	}
}

func (a *App) closeActiveTab() {
	if t := a.tab(); t != nil {
		a.closeTab(t)
	}
}

func (a *App) showTab(n int) {
	if n == 9 {
		a.selectTab(len(a.tabs) - 1)
	} else {
		a.selectTab(n - 1)
	}
}

func (a *App) cycleTab(d int) {
	if n := len(a.tabs); n > 1 {
		a.selectTab(((a.active+d)%n + n) % n)
	}
}

func (a *App) startRename() {
	if t := a.tab(); t != nil {
		a.renaming, a.renameText, a.renameFocus = t, t.Name, false
		a.focusReq = nil
	}
}

func (a *App) clearFocused() {
	if t := a.tab(); t != nil && t.Focus != nil && t.Focus.term != nil {
		t.Focus.term.Feed([]byte("\x1b[H\x1b[2J\x1b[3J"))
		if t.Focus.info.Idle {
			t.Focus.term.Send([]byte{0x0c})
		}
	}
}

func (a *App) restartFocused() {
	t := a.tab()
	if t == nil || t.Focus == nil {
		return
	}
	old := t.Focus
	cols, rows := 80, 24
	if old.term != nil {
		cols, rows = old.term.Size()
	}
	p := a.newPane(a.currentDir(), cols, rows)
	p.Tab, p.Node = t, old.Node
	old.Node.Pane = p
	old.closed = true
	if old.term != nil {
		old.term.Close()
	}
	go a.client.Kill(old.SID)
	if t.Zoom == old {
		t.Zoom = p
	}
	t.Focus = nil
	t.setFocus(p)
	a.focusReq = p
	a.changed()
}

func (a *App) quitAndEnd() {
	a.quitting = true
	for _, t := range a.tabs {
		for _, p := range t.panes() {
			p.closed = true
		}
	}
	if a.settingsWin != nil {
		a.settingsWin.Close()
	}
	c := a.client
	go func() {
		if c != nil {
			c.Shutdown()
		}
		mygo.App.Quit()
	}()
}

func (a *App) openPalette() {
	a.paletteOpen, a.paletteQuery, a.paletteSel = true, "", 0
	a.focusReq = nil
}

func (a *App) shortcuts(c *ui.Context) {
	if c.Shortcut(ui.Ctrl, ui.KeyTab) {
		a.cycleTab(1)
	}
	if c.Shortcut(ui.Ctrl|ui.Shift, ui.KeyTab) {
		a.cycleTab(-1)
	}
	if runtime.GOOS != "windows" {
		return
	}
	for _, cmd := range commands {
		if cmd.Other != (shortcut{}) && c.Shortcut(cmd.Other.Mods, cmd.Other.Key) {
			a.later(c, func() { cmd.Run(a) })
		}
	}
	for i := 1; i <= 9; i++ {
		if k := showTabKey(i); c.Shortcut(k.Mods, k.Key) {
			a.later(c, func() { a.showTab(i) })
		}
	}
}

type paletteItem struct {
	group               string
	title, detail, keys string
	glyph               string
	run                 func()
}

func (a *App) paletteItems() []paletteItem {
	var items []paletteItem
	q := strings.ToLower(strings.TrimSpace(a.paletteQuery))
	match := func(s string) bool { return q == "" || fuzzy(strings.ToLower(s), q) }
	for ti, t := range a.tabs {
		for _, p := range t.panes() {
			name, detail := p.label()
			if !match(fmt.Sprintf("%s %s tab %d", name, detail, ti+1)) {
				continue
			}
			ti, t, p := ti, t, p
			items = append(items, paletteItem{
				group: "PANES",
				title: name, detail: detail, glyph: paneProgram(p).Glyph,
				keys: fmt.Sprintf("Tab %d", ti+1),
				run: func() {
					a.selectTab(ti)
					t.setFocus(p)
					a.focusReq = p
				},
			})
		}
	}
	for _, cmd := range paletteCommands {
		if cmd.Hidden || !match(cmd.Title) {
			continue
		}
		cmd := cmd
		glyph := "command"
		switch cmd {
		case &cmdSettings:
			glyph = "tc-gear"
		case &cmdInspector:
			glyph = "tc-inspector"
		case &cmdSplitRight:
			glyph = "columns-2"
		case &cmdSplitDown:
			glyph = "rows-2"
		case &cmdNewTab:
			glyph = "plus"
		}
		items = append(items, paletteItem{
			group: "COMMANDS",
			title: cmd.Title, keys: cmd.Keys, glyph: glyph,
			run: func() { cmd.Run(a) },
		})
	}
	return items
}

func fuzzy(s, q string) bool {
	if strings.Contains(s, q) {
		return true
	}
	i := 0
	for _, r := range s {
		if i < len(q) && byte(r) == q[i] {
			i++
		}
	}
	return i == len(q)
}

// matchSpans builds RichText spans bolding characters matched by query q.
func matchSpans(ts *themeState, title, q string, fg ui.Color) []ui.Span {
	step := stepBody()
	size := step.size * ts.TextScale
	q = strings.ToLower(strings.TrimSpace(q))
	if q == "" {
		return []ui.Span{{Text: title, Size: size, Weight: 500, Color: fg}}
	}
	lower := strings.ToLower(title)
	if idx := strings.Index(lower, q); idx >= 0 {
		var out []ui.Span
		if idx > 0 {
			out = append(out, ui.Span{Text: title[:idx], Size: size, Weight: 500, Color: fg})
		}
		end := min(idx+len(q), len(title))
		out = append(out, ui.Span{Text: title[idx:end], Size: size, Weight: 700, Color: fg})
		if end < len(title) {
			out = append(out, ui.Span{Text: title[end:], Size: size, Weight: 500, Color: fg})
		}
		return out
	}
	var out []ui.Span
	qi := 0
	for i := 0; i < len(title); {
		r, n := utf8.DecodeRuneInString(title[i:])
		lr := unicodeLowerASCII(r)
		w := 500
		if qi < len(q) && byte(lr) == q[qi] {
			w = 700
			qi++
		}
		if len(out) > 0 && out[len(out)-1].Weight == w {
			out[len(out)-1].Text += title[i : i+n]
		} else {
			out = append(out, ui.Span{Text: title[i : i+n], Size: size, Weight: w, Color: fg})
		}
		i += n
	}
	return out
}

func unicodeLowerASCII(r rune) rune {
	if r >= 'A' && r <= 'Z' {
		return r + ('a' - 'A')
	}
	return r
}

// palette renders the Near AI OS / TCDesign CommandPalette (10-actions.css):
// modal-inset-top = 52, 94%-opaque menu fill + 24px blur + shadow-float,
// radius-card-quiet, 40px search row over 1px rule, uppercase eyebrow group headers,
// menuHover + menuHoverText highlighted row with controlOnSelection keycap pills.
func (a *App) palette(c *ui.Context, ts *themeState) {
	if !a.paletteOpen {
		return
	}
	items := a.paletteItems()
	a.paletteSel = min(max(a.paletteSel, 0), max(len(items)-1, 0))
	was := a.paletteOpen
	reveal := false
	open := true
	backdropClicked := false

	ui.DialogBase(c, &open, func(backdrop, panel ui.Element) {
		backdropClicked = backdrop.Clicked()
		backdrop.Background(ts.scrim).Justify(ui.Start).Padding(modalInsetTop, space16, 0, space16)
		styleFloating(panel.Width(600).MaxWidth(sizeReadableWidth).Clip(),
			ts, ts.menu, ts.cardQuietRadius(), false)

		// 40px search header
		ui.Row(c).Height(sizeListRow).Padding(0, cardPaddingX).Gap(space10).AlignItems(ui.Center).
			BorderWidth(0, 0, 1, 0).BorderColor(ts.rule).Children(func() {
			ui.Icon(c, icon("search")).Size(sizeGlyph, sizeGlyph).TextColor(ts.textSecondary).Shrink(0)
			in := ui.TextInputBase(c, &a.paletteQuery).Grow(1).
				FontSize(stepTitle().size * ts.TextScale).TextColor(ts.textPrimary).
				Placeholder("Search panes and commands…").AutoFocus()
			if in.Changed() {
				a.paletteSel, reveal = 0, true
			}
			if in.Shortcut(0, ui.KeyDown) && len(items) > 0 {
				a.paletteSel, reveal = min(a.paletteSel+1, len(items)-1), true
			}
			if in.Shortcut(0, ui.KeyUp) && len(items) > 0 {
				a.paletteSel, reveal = max(a.paletteSel-1, 0), true
			}
			if in.Submitted() && len(items) > 0 {
				a.paletteOpen = false
				items[a.paletteSel].run()
			}
			if a.paletteQuery != "" {
				if glassRoundButton(c, ts, "x", 11, "Clear search", true, false, GroundMenu).Clicked() {
					a.paletteQuery, a.paletteSel, reveal = "", 0, true
				}
			}
		})

		listH := float32(min(max(len(items)*36+36, 88), 340))
		ui.Scroll(c).Height(listH).Padding(space6).Children(func() {
			if len(items) == 0 {
				ui.Row(c).Height(72).Center().Children(func() {
					applyType(ui.Text(c, "No matching panes or commands."), ts, stepBody()).TextColor(ts.textTertiary)
				})
				return
			}
			lastGroup := ""
			for i, it := range items {
				if it.group != "" && it.group != lastGroup {
					lastGroup = it.group
					topPad := float32(4)
					if i > 0 {
						topPad = 10
					}
					ui.Row(c.Key("grp-"+it.group)).Padding(topPad, 10, 3, 10).Children(func() {
						applyType(ui.Text(c, it.group), ts, stepEyebrow()).TextColor(ts.textTertiary)
					})
				}
				selected := i == a.paletteSel
				rowRadius := max(ts.cardQuietRadius()-space4, radiusTile)
				row := ui.Row(c.Key(i)).Height(36).Padding(0, 10).Gap(space10).
					Radius(rowRadius).AlignItems(ui.Center).Cursor(ui.CursorPointer)
				if selected {
					row.Background(ts.menuHover)
					if reveal {
						row.ScrollIntoView()
					}
				} else if row.Hovered() {
					row.Background(ts.rowHover)
				}
				if row.Clicked() {
					a.paletteOpen = false
					it.run()
				}
				row.Children(func() {
					iconCol := ts.textSecondary
					titleCol := ts.textPrimary
					detailCol := ts.textTertiary
					keyFg := ts.textSecondary
					keyBg := ts.controlOnMenu
					if selected {
						iconCol = ts.menuHoverText
						titleCol = ts.menuHoverText
						detailCol = ts.menuHoverText.Alpha(0.82)
						keyFg = ts.menuHoverText
						keyBg = ts.controlOnSelection
					}
					ui.Icon(c, icon(it.glyph)).Size(15, 15).TextColor(iconCol).Shrink(0)
					if strings.TrimSpace(a.paletteQuery) == "" {
						applyType(ui.Text(c, it.title), ts, stepBody()).FontWeight(500).
							TextColor(titleCol).SingleLine().Ellipsis("…").Shrink(0.3).MinWidth(0)
					} else {
						spans := matchSpans(ts, it.title, a.paletteQuery, titleCol)
						w, _ := c.MeasureText(0, spans...)
						tb := ui.Box(c).Width(w + 2).Height(20).Shrink(0.3).MinWidth(0).ClipX().Label(it.title)
						tb.Draw(func(p *ui.Painter, r ui.Rect) {
							_, th := p.MeasureText(0, spans...)
							p.RichText(r.X, r.Y+(r.H-th)/2, 0, spans...)
						})
					}
					if it.detail != "" {
						applyType(ui.Text(c, it.detail), ts, stepMono()).
							TextColor(detailCol).SingleLine().Ellipsis("…").Shrink(1).MinWidth(0)
					}
					ui.Spacer(c)
					if it.keys != "" {
						applyType(ui.Text(c, it.keys), ts, stepMono()).
							TextColor(keyFg).Padding(2, 8).Radius(radiusPill).Background(keyBg).Shrink(0)
					}
				})
			}
		})

		// Keyboard legend footer over 1px rule
		ui.Row(c).Height(30).Padding(0, cardPaddingX).Gap(space14).AlignItems(ui.Center).
			BorderWidth(1, 0, 0, 0).BorderColor(ts.rule).Children(func() {
			paletteHint(c, ts, "↑↓", "Move")
			paletteHint(c, ts, "↩", "Run")
			if a.paletteQuery != "" {
				paletteHint(c, ts, "esc", "Clear")
			} else {
				paletteHint(c, ts, "esc", "Close")
			}
		})
	})

	if !open {
		if !backdropClicked && a.paletteQuery != "" {
			// First Esc clears a non-empty query; second Esc closes (CommandPalette/README.md).
			a.paletteQuery = ""
			a.paletteSel = 0
		} else {
			a.paletteOpen = false
		}
	}
	if was && !a.paletteOpen {
		if t := a.tab(); t != nil && a.focusReq == nil {
			a.focusReq = t.Focus
		}
	}
}

func paletteHint(c *ui.Context, ts *themeState, keys, label string) {
	ui.Row(c).Gap(5).AlignItems(ui.Center).Children(func() {
		applyType(ui.Text(c, keys), ts, stepMono()).TextColor(ts.textSecondary).
			Padding(1, 6).Radius(radiusTile).Background(ts.controlOnMenu)
		applyType(ui.Text(c, label), ts, stepCaption()).TextColor(ts.textTertiary)
	})
}
