//go:build darwin

package main

import (
	"runtime"
	"slices"
	"sync"
	"time"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
	"github.com/egoist/mygo/ui"
)

type nsPoint struct{ X, Y float64 }
type nsSize struct{ Width, Height float64 }
type nsRect struct {
	Origin nsPoint
	Size   nsSize
}

type nativePaneSpec struct {
	X, Y, W, H, Radius float32
}

type glassWinState struct {
	host       objc.ID
	panes      []objc.ID
	specs      []nativePaneSpec
	contentW   float64
	contentH   float64
	dark       bool
	active     bool
	tint       ui.Color
	inited     bool
	useNSGlass bool
}

var (
	reduceTransMu    sync.Mutex
	reduceTransAt    time.Time
	reduceTransValue bool

	darwinOnce  sync.Once
	msgRect     func(obj objc.ID, sel objc.SEL) nsRect
	msgSetRect  func(obj objc.ID, sel objc.SEL, r nsRect)
	msgInitRect func(obj objc.ID, sel objc.SEL, r nsRect) objc.ID
	msgSetFloat func(obj objc.ID, sel objc.SEL, v float64)
	msgSetSize  func(obj objc.ID, sel objc.SEL, s nsSize)
	msgColor    func(cls objc.ID, sel objc.SEL, r, g, b, a float64) objc.ID

	selSharedWorkspace    objc.SEL
	selReduceTransparency objc.SEL
	selAlloc              objc.SEL
	selInitWithFrame      objc.SEL
	selBounds             objc.SEL
	selSetFrame           objc.SEL
	selContentView        objc.SEL
	selAddSubviewRel      objc.SEL
	selSetAutoresizing    objc.SEL
	selSetHidden          objc.SEL
	selSetWantsLayer      objc.SEL
	selLayer              objc.SEL
	selSetMasksToBounds   objc.SEL
	selSetCornerRadius    objc.SEL
	selSetCornerCurve     objc.SEL
	selSetStyle           objc.SEL
	selSetTintColor       objc.SEL
	selSetAppearance      objc.SEL
	selAppearanceNamed    objc.SEL
	selColorSRGB          objc.SEL
	selStringUTF8         objc.SEL
	selSetOpaque          objc.SEL
	selSetBgColor         objc.SEL
	selSetHasShadow       objc.SEL
	selSetTitlebarTrans   objc.SEL
	selSetTitleVis        objc.SEL
	selSetTitlebarSep     objc.SEL
	selRespondsTo         objc.SEL
	selSetMaterial        objc.SEL
	selSetBlendingMode    objc.SEL
	selSetState           objc.SEL
	selWindowNumber       objc.SEL

	glassWinMu sync.Mutex
	glassWins  = map[uintptr]*glassWinState{}
)

func initDarwin() {
	darwinOnce.Do(func() {
		libObjC, err := purego.Dlopen("/usr/lib/libobjc.A.dylib", purego.RTLD_GLOBAL|purego.RTLD_NOW)
		if err != nil {
			return
		}
		msgSend, err := purego.Dlsym(libObjC, "objc_msgSend")
		if err != nil {
			return
		}
		stret := msgSend
		if runtime.GOARCH == "amd64" {
			if s, err := purego.Dlsym(libObjC, "objc_msgSend_stret"); err == nil {
				stret = s
			}
		}
		purego.RegisterFunc(&msgRect, stret)
		purego.RegisterFunc(&msgSetRect, msgSend)
		purego.RegisterFunc(&msgInitRect, msgSend)
		purego.RegisterFunc(&msgSetFloat, msgSend)
		purego.RegisterFunc(&msgSetSize, msgSend)
		purego.RegisterFunc(&msgColor, msgSend)

		selSharedWorkspace = objc.RegisterName("sharedWorkspace")
		selReduceTransparency = objc.RegisterName("accessibilityDisplayShouldReduceTransparency")
		selAlloc = objc.RegisterName("alloc")
		selInitWithFrame = objc.RegisterName("initWithFrame:")
		selBounds = objc.RegisterName("bounds")
		selSetFrame = objc.RegisterName("setFrame:")
		selContentView = objc.RegisterName("contentView")
		selAddSubviewRel = objc.RegisterName("addSubview:positioned:relativeTo:")
		selSetAutoresizing = objc.RegisterName("setAutoresizingMask:")
		selSetHidden = objc.RegisterName("setHidden:")
		selSetWantsLayer = objc.RegisterName("setWantsLayer:")
		selLayer = objc.RegisterName("layer")
		selSetMasksToBounds = objc.RegisterName("setMasksToBounds:")
		selSetCornerRadius = objc.RegisterName("setCornerRadius:")
		selSetCornerCurve = objc.RegisterName("setCornerCurve:")
		selSetStyle = objc.RegisterName("setStyle:")
		selSetTintColor = objc.RegisterName("setTintColor:")
		selSetAppearance = objc.RegisterName("setAppearance:")
		selAppearanceNamed = objc.RegisterName("appearanceNamed:")
		selColorSRGB = objc.RegisterName("colorWithSRGBRed:green:blue:alpha:")
		selStringUTF8 = objc.RegisterName("stringWithUTF8String:")
		selSetOpaque = objc.RegisterName("setOpaque:")
		selSetBgColor = objc.RegisterName("setBackgroundColor:")
		selSetHasShadow = objc.RegisterName("setHasShadow:")
		selSetTitlebarTrans = objc.RegisterName("setTitlebarAppearsTransparent:")
		selSetTitleVis = objc.RegisterName("setTitleVisibility:")
		selSetTitlebarSep = objc.RegisterName("setTitlebarSeparatorStyle:")
		selRespondsTo = objc.RegisterName("respondsToSelector:")
		selSetMaterial = objc.RegisterName("setMaterial:")
		selSetBlendingMode = objc.RegisterName("setBlendingMode:")
		selSetState = objc.RegisterName("setState:")
		selWindowNumber = objc.RegisterName("windowNumber")
	})
}

func nsStr(s string) objc.ID {
	cls := objc.GetClass("NSString")
	if cls == 0 {
		return 0
	}
	b := append([]byte(s), 0)
	return objc.Send[objc.ID](objc.ID(cls), selStringUTF8, uintptr(unsafe.Pointer(&b[0])))
}

func nsSRGB(c ui.Color) objc.ID {
	cls := objc.GetClass("NSColor")
	if cls == 0 || msgColor == nil {
		return 0
	}
	return msgColor(objc.ID(cls), selColorSRGB,
		float64(c.R)/255.0, float64(c.G)/255.0, float64(c.B)/255.0, float64(c.A)/255.0)
}

func nsAppearance(dark bool) objc.ID {
	cls := objc.GetClass("NSAppearance")
	if cls == 0 {
		return 0
	}
	name := "NSAppearanceNameAqua"
	if dark {
		name = "NSAppearanceNameDarkAqua"
	}
	return objc.Send[objc.ID](objc.ID(cls), selAppearanceNamed, nsStr(name))
}

// systemReduceTransparency queries NSWorkspace.sharedWorkspace.accessibilityDisplayShouldReduceTransparency
// when AppKit is loaded, caching the result briefly so per-frame checks cost nothing.
func systemReduceTransparency() bool {
	reduceTransMu.Lock()
	defer reduceTransMu.Unlock()
	if time.Since(reduceTransAt) < 500*time.Millisecond {
		return reduceTransValue
	}
	reduceTransAt = time.Now()
	initDarwin()
	cls := objc.GetClass("NSWorkspace")
	if cls == 0 {
		return false
	}
	ws := objc.ID(cls).Send(selSharedWorkspace)
	if ws == 0 {
		return false
	}
	reduceTransValue = objc.Send[bool](ws, selReduceTransparency)
	return reduceTransValue
}

// configureGlassWindow configures the hosting NSWindow identically to
// TCDesign's GlassWindowConfigurator.configure(_:) in trace-commons-fork:
// clear window background (alpha 0.004 so resize/drag work across pane gaps),
// full-size transparent titlebar with no separator line, so floating 20pt
// Liquid Glass panes show the real macOS desktop between them.
func configureGlassWindow(nsWin uintptr) {
	if nsWin == 0 {
		return
	}
	initDarwin()
	if msgColor == nil {
		return
	}
	w := objc.ID(nsWin)
	objc.Send[struct{}](w, selSetOpaque, false)
	// Alpha 0.004 (~1/255) catches window edge resize and gap dragging while remaining invisible.
	clearHit := msgColor(objc.ID(objc.GetClass("NSColor")), selColorSRGB, 0, 0, 0, 0.004)
	objc.Send[struct{}](w, selSetBgColor, clearHit)
	objc.Send[struct{}](w, selSetHasShadow, true)
	objc.Send[struct{}](w, selSetTitlebarTrans, true)
	objc.Send[struct{}](w, selSetTitleVis, uintptr(1)) // NSWindowTitleHidden
	if objc.Send[bool](w, selRespondsTo, selSetTitlebarSep) {
		objc.Send[struct{}](w, selSetTitlebarSep, uintptr(1)) // NSTitlebarSeparatorStyleNone
	}
}

func removeGlassWindow(nsWin uintptr) {
	glassWinMu.Lock()
	delete(glassWins, nsWin)
	glassWinMu.Unlock()
}

// syncNativeGlassPanes synchronizes AppKit NSGlassEffectView (macOS 26+) or
// NSVisualEffectView (.hudWindow fallback) subviews behind MyGoSurfaceView,
// one per floating GlassPane, so real macOS Liquid Glass refracts the desktop
// inside each rounded pane while leaving the 10pt paneGap completely clear.
func syncNativeGlassPanes(nsWin uintptr, specs []nativePaneSpec, dark, active bool, tint ui.Color) {
	if nsWin == 0 {
		return
	}
	initDarwin()
	if msgRect == nil {
		return
	}
	w := objc.ID(nsWin)
	cv := objc.Send[objc.ID](w, selContentView)
	if cv == 0 {
		return
	}
	bounds := msgRect(cv, selBounds)

	glassWinMu.Lock()
	defer glassWinMu.Unlock()
	st := glassWins[nsWin]
	if st == nil {
		nsGlassCls := objc.GetClass("NSGlassEffectView")
		nsViewCls := objc.GetClass("NSView")
		if nsViewCls == 0 {
			return
		}
		host := msgInitRect(objc.ID(nsViewCls).Send(selAlloc), selInitWithFrame, bounds)
		objc.Send[struct{}](host, selSetAutoresizing, uintptr(2|16)) // widthSizable | heightSizable
		objc.Send[struct{}](host, selSetWantsLayer, true)
		if layer := objc.Send[objc.ID](host, selLayer); layer != 0 {
			objc.Send[struct{}](layer, selSetMasksToBounds, false)
		}
		// Insert below MyGoSurfaceView (^uintptr(0) == -1 == NSWindowBelow).
		objc.Send[struct{}](cv, selAddSubviewRel, host, ^uintptr(0), uintptr(0))
		st = &glassWinState{
			host:       host,
			useNSGlass: nsGlassCls != 0,
		}
		glassWins[nsWin] = st
	}

	if st.inited &&
		st.active == active &&
		st.dark == dark &&
		st.tint == tint &&
		st.contentW == bounds.Size.Width &&
		st.contentH == bounds.Size.Height &&
		slices.Equal(st.specs, specs) {
		return
	}

	st.inited = true
	st.active = active
	st.dark = dark
	st.tint = tint
	st.contentW = bounds.Size.Width
	st.contentH = bounds.Size.Height
	st.specs = slices.Clone(specs)

	// Single-pane windows cast the native macOS window shadow around the 20pt glass pane;
	// multi-pane windows turn off the window-wide shadow so the 10pt clear gap stays shadow-free.
	objc.Send[struct{}](w, selSetHasShadow, len(specs) <= 1 || !active)

	objc.Send[struct{}](st.host, selSetHidden, !active)
	if !active {
		return
	}

	app := nsAppearance(dark)
	nsTint := nsSRGB(tint)

	for len(st.panes) < len(specs) {
		var pv objc.ID
		if st.useNSGlass {
			cls := objc.ID(objc.GetClass("NSGlassEffectView"))
			pv = msgInitRect(cls.Send(selAlloc), selInitWithFrame, nsRect{})
			objc.Send[struct{}](pv, selSetStyle, uintptr(0)) // NSGlassEffectViewStyleRegular
		} else {
			cls := objc.ID(objc.GetClass("NSVisualEffectView"))
			pv = msgInitRect(cls.Send(selAlloc), selInitWithFrame, nsRect{})
			objc.Send[struct{}](pv, selSetMaterial, uintptr(13))    // NSVisualEffectMaterialHUDWindow
			objc.Send[struct{}](pv, selSetBlendingMode, uintptr(0)) // BehindWindow
			objc.Send[struct{}](pv, selSetState, uintptr(1))        // Active
			objc.Send[struct{}](pv, selSetWantsLayer, true)
			if layer := objc.Send[objc.ID](pv, selLayer); layer != 0 {
				objc.Send[struct{}](layer, selSetMasksToBounds, true)
				objc.Send[struct{}](layer, selSetCornerCurve, nsStr("continuous"))
			}
		}
		objc.Send[struct{}](st.host, selAddSubviewRel, pv, uintptr(1), uintptr(0))
		st.panes = append(st.panes, pv)
	}

	for i, pv := range st.panes {
		if i >= len(specs) {
			objc.Send[struct{}](pv, selSetHidden, true)
			continue
		}
		sp := specs[i]
		if sp.W <= 0 || sp.H <= 0 {
			objc.Send[struct{}](pv, selSetHidden, true)
			continue
		}
		objc.Send[struct{}](pv, selSetHidden, false)
		macY := bounds.Size.Height - float64(sp.Y+sp.H)
		msgSetRect(pv, selSetFrame, nsRect{
			Origin: nsPoint{X: float64(sp.X), Y: macY},
			Size:   nsSize{Width: float64(sp.W), Height: float64(sp.H)},
		})
		if app != 0 {
			objc.Send[struct{}](pv, selSetAppearance, app)
		}
		if st.useNSGlass {
			msgSetFloat(pv, selSetCornerRadius, float64(sp.Radius))
			if nsTint != 0 {
				objc.Send[struct{}](pv, selSetTintColor, nsTint)
			}
		} else if layer := objc.Send[objc.ID](pv, selLayer); layer != 0 {
			msgSetFloat(layer, selSetCornerRadius, float64(sp.Radius))
		}
	}
}

func nativeWindowID(nsWin uintptr) int {
	if nsWin == 0 {
		return 0
	}
	initDarwin()
	return objc.Send[int](objc.ID(nsWin), selWindowNumber)
}
