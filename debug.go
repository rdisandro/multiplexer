package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo"
)

// debugHook lets a script drive the app in development mode while MULTIPLEXER_DEBUG
// names a directory: each line of its file "do" is run, and "shot" captures
// the window into shot.png.
func (a *App) debugHook() {
	if !mygo.IsDev() {
		return
	}
	dir := os.Getenv("MULTIPLEXER_DEBUG")
	if dir == "" {
		return
	}
	st, err := os.Lstat(dir)
	if err != nil || !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
		return
	}
	log.SetFlags(log.Ltime | log.Lmicroseconds)
	go func() {
		for range time.Tick(150 * time.Millisecond) {
			req := filepath.Join(dir, "do")
			if st, err := os.Lstat(req); err != nil || !st.Mode().IsRegular() {
				continue
			}
			b, err := os.ReadFile(req)
			if err != nil {
				continue
			}
			os.Remove(req)
			for _, line := range strings.Split(strings.TrimSpace(string(b)), "\n") {
				a.debug(dir, strings.TrimSpace(line))
			}
		}
	}()
}

func captureWindow(win *mygo.Window, sceneBackdrop bool, dir, arg, fallback string) {
	if win == nil {
		return
	}
	if sceneBackdrop {
		win.Update(func() { forceSceneBackdrop = true })
		time.Sleep(120 * time.Millisecond)
	} else {
		time.Sleep(100 * time.Millisecond)
	}
	png, err := win.CapturePage()
	if sceneBackdrop {
		win.Update(func() { forceSceneBackdrop = false })
	}
	if err == nil {
		name := fallback
		if arg != "" {
			if base := filepath.Base(arg); base != "." && base != ".." && base != string(filepath.Separator) {
				name = base
			}
		}
		os.WriteFile(filepath.Join(dir, name), png, 0o600)
	}
}

func (a *App) debug(dir, line string) {
	verb, arg, _ := strings.Cut(line, " ")
	win := a.win
	if win == nil {
		return
	}
	switch verb {
	case "shot":
		captureWindow(win, true, dir, arg, "shot.png")
	case "shot-settings":
		captureWindow(a.settingsWin, false, dir, arg, "settings.png")
	case "sleep":
		d, _ := time.ParseDuration(arg)
		time.Sleep(d)
	case "type":
		arg = strings.ReplaceAll(arg, `\r`, "\r")
		arg = strings.ReplaceAll(arg, `\e`, "\x1b")
		win.Update(func() {
			if t := a.tab(); t != nil && t.Focus != nil && t.Focus.term != nil {
				t.Focus.term.Send([]byte(arg))
			}
		})
	case "size":
		var w, h int
		if _, err := fmtSscan(arg, &w, &h); err == nil {
			win.SetSize(w, h)
		}
	case "appearance":
		mygo.RunOnMain(func() {
			if arg == "system" {
				arg = ""
			}
			a.setAppearance(arg)
		})
	case "theme":
		mygo.RunOnMain(func() {
			if arg == "near" {
				arg = ""
			}
			a.setProductTheme(arg)
		})
	case "hc":
		mygo.RunOnMain(func() { a.setHighContrast(arg == "on") })
	case "density":
		mygo.RunOnMain(func() {
			if arg == "compact" {
				arg = ""
			}
			a.setDensity(arg)
		})
	case "inspector":
		win.Update(func() { a.inspectorOpen = arg != "close" && arg != "off" })
	case "palette":
		win.Update(func() {
			if arg == "close" {
				a.paletteOpen = false
			} else {
				a.openPalette()
				if arg != "" && arg != "open" {
					a.paletteQuery = arg
				}
			}
		})
	case "host":
		win.Update(func() { a.hostOpen = arg != "close" })
	case "settings":
		mygo.RunOnMain(func() {
			if arg == "close" {
				if a.settingsWin != nil {
					a.settingsWin.Close()
				}
				return
			}
			if arg != "" {
				a.settingsSection = arg
				a.settingsScrollReq = arg
			}
			a.openSettings()
		})
	default:
		for _, cmd := range commands {
			if strings.EqualFold(strings.ReplaceAll(cmd.Title, " ", "-"), verb) {
				a.run(cmd)
			}
		}
		switch verb {
		case "tab":
			win.Update(func() { a.selectTab(atoi(arg) - 1) })
		case "focus":
			win.Update(func() {
				if t := a.tab(); t != nil {
					idx := atoi(arg) - 1
					if ps := t.panes(); idx >= 0 && idx < len(ps) {
						t.setFocus(ps[idx])
						a.focusReq = t.Focus
					}
				}
			})
		case "name":
			win.Update(func() {
				if t := a.tab(); t != nil {
					t.Name = arg
				}
			})
		case "close":
			win.Close()
		case "zoom":
			win.Update(func() {
				if win.IsMaximized() {
					win.Unmaximize()
				} else {
					win.Maximize()
				}
			})
		}
	}
}

func atoi(s string) int {
	n := 0
	for _, r := range s {
		if r >= '0' && r <= '9' {
			n = n*10 + int(r-'0')
		}
	}
	return n
}

func fmtSscan(s string, w, h *int) (int, error) {
	a, b, _ := strings.Cut(s, " ")
	*w, *h = atoi(a), atoi(b)
	return 2, nil
}
