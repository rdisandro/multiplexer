//go:build !darwin

package main

import "github.com/egoist/mygo/ui"

type nativePaneSpec struct {
	X, Y, W, H, Radius float32
}

func systemReduceTransparency() bool { return false }

func configureGlassWindow(uintptr) {}

func removeGlassWindow(uintptr) {}

func syncNativeGlassPanes(uintptr, []nativePaneSpec, bool, bool, ui.Color) {}

func nativeWindowID(uintptr) int { return 0 }
