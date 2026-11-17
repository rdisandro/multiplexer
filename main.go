// Multiplexer is a native macOS and cross-platform terminal of tabs and split panes,
// built with MyGo's native UI and Ghostty's libghostty-vt, styled to the
// Near AI OS Design System & TCDesign Liquid Glass architecture. Its shells run
// in a persistent session server so they outlive the app across restarts.
//
//	go tool mygo dev    the app, rebuilt as the code changes
//	go run . -server    the session server alone, in the foreground
package main

import (
	"encoding/json"
	"log"
	"os"
	"runtime"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"multiplexer/internal/rex"
)

func main() {
	// -server runs the session server; other arguments are AppKit's, as
	// -NSWindowResizeTime, which it reads itself.
	if len(os.Args) > 1 && os.Args[1] == "-server" {
		log.SetPrefix("[server] ")
		if err := rex.Serve(); err != nil {
			log.Fatal(err)
		}
		return
	}
	// The state of the sessions lives in the app's data directory, which
	// the server it starts takes from MULTIPLEXER_DIR: "Multiplexer", or
	// "Multiplexer Dev" under mygo dev, keeping installed sessions apart.
	if os.Getenv("MULTIPLEXER_DIR") == "" {
		if dir, err := mygo.App.Path(mygo.PathUserData); err == nil {
			os.Setenv("MULTIPLEXER_DIR", dir)
		}
	}
	registerFonts()
	loadSettings()
	a := &App{settingsSection: "general"}
	mygo.App.SetMenu(a.menu())
	mygo.App.WhenReady(func() {
		appRunning = true
		applyAppearance()
		a.open()
	})
	mygo.App.OnActivate(func(hasVisibleWindows bool) {
		if !hasVisibleWindows && a.win == nil {
			a.open()
		}
	})
	mygo.App.OnWindowAllClosed(func() {
		if runtime.GOOS != "darwin" {
			mygo.App.Quit()
		}
	})
	mygo.App.OnBeforeQuit(func(*mygo.QuitEvent) {
		a.saveNow()
		if a.settingsWin != nil {
			a.settingsWin.Close()
		}
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

// open opens the main window, restoring the tabs the session server kept.
func (a *App) open() {
	client, err := rex.Connect()
	if err != nil {
		log.Print(err)
		mygo.Dialog.Error("Multiplexer could not start its session server", err.Error())
		mygo.App.Quit()
		return
	}
	a.client = client
	a.hello, err = client.Hello()
	if err == nil && mygo.IsDev() && staleServer(a.hello) {
		// A server a build before this one started runs that build's
		// code: start one of this build's, ending its sessions.
		if client, err = client.Restart(a.hello.PID); err == nil {
			a.client = client
			a.hello, err = client.Hello()
		} else {
			log.Print(err)
			mygo.Dialog.Error("Multiplexer could not restart its session server", err.Error())
			mygo.App.Quit()
			return
		}
	}
	if err == nil && a.hello.Version != rex.ProtocolVersion {
		a.err = "Session server version mismatch. Another build of Multiplexer started the background server. Quit and end all sessions to restart it cleanly."
	}
	a.reset()
	opts := mygo.WindowOptions{
		Title:                "Multiplexer",
		Width:                1120,
		Height:               660,
		MinWidth:             580,
		MinHeight:            380,
		StateKey:             "main",
		Transparent:          runtime.GOOS == "darwin",
		TitleBarStyle:        mygo.TitleBarHidden,
		TrafficLightPosition: &mygo.Point{X: 16, Y: 19}, // Centered at y=26pt inside the main GlassPane
		BackgroundColor:      "light-dark(#e6eaf1, #070a1a)",
		Content:              ui.View(a.view),
	}
	if ownControls {
		opts.TitleBarStyle, opts.Frameless = mygo.TitleBarDefault, true
	}
	win := mygo.NewWindow(opts)
	a.win = win
	nsHandle := win.NativeHandle()
	configureGlassWindow(nsHandle)

	if !a.restore() {
		home, _ := os.UserHomeDir()
		a.newTab(home)
	}
	a.changed()
	go a.poll(win, client)
	a.debugHook()
	win.OnClosed(func() {
		removeGlassWindow(nsHandle)
		a.saveNow()
		a.quitting = true
		for _, t := range a.tabs {
			for _, p := range t.panes() {
				p.closed = true
				if p.term != nil {
					p.term.Close() // detaches: the session goes on
				}
			}
		}
		a.client.Close()
		a.win = nil
	})
}

// reconnect reconnects to the session server after an unexpected disconnect.
func (a *App) reconnect() {
	client, err := rex.Connect()
	if err != nil {
		a.err = "Could not reconnect to the session server. " + err.Error()
		return
	}
	if a.client != nil {
		a.client.Close()
	}
	a.client = client
	a.hello, _ = client.Hello()
	a.err = ""
	for _, t := range a.tabs {
		for _, p := range t.panes() {
			cols, rows := 80, 24
			if p.term != nil {
				cols, rows = p.term.Size()
				p.term.Close()
				p.term = nil
			}
			a.attach(p, cols, rows)
		}
	}
	if a.win != nil {
		go a.poll(a.win, client)
	}
}

// saveNow sends the layout to the server and waits for it to be kept.
func (a *App) saveNow() {
	if a.client == nil || len(a.tabs) == 0 && !a.quitting {
		return
	}
	b, err := json.Marshal(a.snapshot())
	if err != nil {
		return
	}
	done := make(chan struct{})
	go func() {
		a.client.SetLayout(b)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
	}
}

// reset forgets the state of a window that closed, keeping the server's
// connection, for the next window.
func (a *App) reset() {
	a.tabs, a.active, a.focusReq = nil, 0, nil
	a.paletteOpen, a.hostOpen, a.renaming = false, false, nil
	a.saveDue, a.quitting, a.lastSnapshot, a.title = false, false, "", ""
}

// staleServer reports whether the server runs another build of the app
// than this one, as after mygo dev rebuilt it.
func staleServer(h rex.Hello) bool {
	exe, at := rex.Executable()
	return h.Exe != exe || !h.ExeTime.Equal(at)
}
