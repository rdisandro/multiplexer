package main

import (
	"math"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

// ThemeName identifies the active Near AI OS / TCDesign Liquid Glass theme.
type ThemeName string

const (
	ThemeLight        ThemeName = "light"
	ThemeDark         ThemeName = "dark"
	ThemeLightHC      ThemeName = "light-hc"
	ThemeDarkHC       ThemeName = "dark-hc"
	ThemeTraceLight   ThemeName = "trace-light"
	ThemeTraceDark    ThemeName = "trace-dark"
	ThemeClassicLight ThemeName = "classic-light"
	ThemeClassicDark  ThemeName = "classic-dark"
)

// Brand constants from tokens/near-ai.tokens.json and TCDesign/GlassTokens.swift.
// Never use directly on controls; reserve for brand lockups and application icon artwork (NDS-8).
var (
	brandBlue   = ui.Hex("#0091fd")
	brandTile   = ui.Hex("#0b8ff6")
	brandPaper  = ui.Hex("#f3f3f2")
	brandInk    = ui.Hex("#000000")
	tracePurple = ui.Hex("#6d14f3")
)

// Spacing, radius, and sizing tokens from TCDesign/Generated/GlassTokens.swift & Near AI OS 1.0.0.
const (
	space2  float32 = 2
	space4  float32 = 4
	space6  float32 = 6
	space8  float32 = 8
	space10 float32 = 10
	space12 float32 = 12
	space14 float32 = 14
	space16 float32 = 16
	space18 float32 = 18
	space20 float32 = 20
	space24 float32 = 24
	space32 float32 = 32
	space40 float32 = 40

	paneGap              float32 = 10
	windowPadding        float32 = 0
	windowControlsInset  float32 = 30
	windowControlsWidth  float32 = 78
	lightsCentre         float32 = 26
	panePadding          float32 = 12
	inspectorPaddingH    float32 = 16
	inspectorPaddingV    float32 = 18
	mainInset            float32 = 16 // MonitorWindowView.mainInset = inspectorPaddingHorizontal
	cardPaddingY         float32 = 12
	cardPaddingX         float32 = 14
	cardGap              float32 = 10
	inlineGap            float32 = 6
	modalInset           float32 = 28
	modalInsetTop        float32 = 52

	radiusPane      float32 = 20
	radiusCard      float32 = 16
	radiusCardQuiet float32 = 14
	radiusControl   float32 = 11
	radiusTile      float32 = 6
	radiusCheckbox  float32 = 5
	radiusDense     float32 = 3
	radiusWindow    float32 = 22
	radiusPill      float32 = 999

	sizeControlDense   float32 = 20
	sizeSubmitPill     float32 = 24
	sizeControl        float32 = 26
	sizeTab            float32 = 26
	sizeControlLarge   float32 = 28
	sizeCtaSmall       float32 = 30
	sizeSegmentedTrack float32 = 30
	sizeToolTileLarge  float32 = 30
	sizeCta            float32 = 34
	sizeListRow        float32 = 40
	sizeToggleWidth    float32 = 40
	sizeToggleHeight   float32 = 24
	sizeGlyph          float32 = 16
	sizeToolTile       float32 = 22
	sizeToolLogo       float32 = 15
	sizeToolLogoLarge  float32 = 20
	sizeInspectorWidth float32 = 300
	sizeModalWidth     float32 = 780
	sizeModalHeight    float32 = 640
	sizeModalNarrow    float32 = 450
	sizeModalNav       float32 = 180
	sizePopoverWidth   float32 = 288
	sizeReadableWidth  float32 = 680

	opacityDisabled float32 = 0.45
	opacityRowOff   float32 = 0.60
)

// Motion durations and easings from Near AI OS Design System (09-motion.md) & TCDesign.
const (
	durFast          = 150 * time.Millisecond
	durReveal        = 180 * time.Millisecond
	durStandard      = 220 * time.Millisecond
	durSlide         = 250 * time.Millisecond
	durSlow          = 300 * time.Millisecond
	durSpringFast    = 224 * time.Millisecond
	durSpringSpatial = 317 * time.Millisecond
	durSpringEffects = 231 * time.Millisecond
	durSpringEmber   = 488 * time.Millisecond
)

var (
	// easeStandard is cubic-bezier(0.2, 0.8, 0.2, 1).
	easeStandard = cubicBezier(0.2, 0.8, 0.2, 1.0)
	// easeExit is cubic-bezier(0.3, 0, 1, 1).
	easeExit = cubicBezier(0.3, 0.0, 1.0, 1.0)
	// springSpatial matches --spring-spatial (ζ = 0.9, no bounce).
	springSpatial = linearStops([]float32{
		0, 0.101, 0.296, 0.495, 0.66, 0.783, 0.869, 0.925, 0.96, 0.981, 0.992, 0.998, 1, 1.001, 1.002, 1.001, 1,
	})
	// springEmber matches --spring-ember for the active tab ember glide.
	springEmber = linearStops([]float32{
		0, 0.11, 0.322, 0.535, 0.708, 0.831, 0.912, 0.961, 0.987, 1, 1.005, 1.006, 1.005, 1.004, 1.003, 1.002, 1,
	})
)

func linearStops(stops []float32) ui.Easing {
	return func(t float32) float32 {
		if t <= 0 {
			return 0
		}
		if t >= 1 {
			return 1
		}
		f := t * float32(len(stops)-1)
		i := int(f)
		if i >= len(stops)-1 {
			return stops[len(stops)-1]
		}
		frac := f - float32(i)
		return stops[i] + (stops[i+1]-stops[i])*frac
	}
}

func cubicBezier(x1, y1, x2, y2 float64) ui.Easing {
	sampleX := func(u float64) float64 {
		om := 1 - u
		return 3*om*om*u*x1 + 3*om*u*u*x2 + u*u*u
	}
	sampleDX := func(u float64) float64 {
		om := 1 - u
		return 3*om*om*x1 + 6*om*u*(x2-x1) + 3*u*u*(1-x2)
	}
	sampleY := func(u float64) float64 {
		om := 1 - u
		return 3*om*om*u*y1 + 3*om*u*u*y2 + u*u*u
	}
	return func(t float32) float32 {
		if t <= 0 {
			return 0
		}
		if t >= 1 {
			return 1
		}
		x := float64(t)
		u := x
		for range 6 {
			dx := sampleDX(u)
			if math.Abs(dx) < 1e-6 {
				break
			}
			u -= (sampleX(u) - x) / dx
			u = min(max(u, 0), 1)
		}
		return float32(sampleY(u))
	}
}

// shadowSpec describes one elevation shadow token (shadow-float or shadow-modal).
type shadowSpec struct {
	y, blur float32
	color   ui.Color
}

// Tokens holds every semantic color, gradient, and elevation role for a Near AI OS / TCDesign theme.
type Tokens struct {
	Name    ThemeName
	Dark    bool
	HC      bool
	Classic bool

	// Ground & Liquid Glass materials
	scene, sceneWarm                    ui.Color
	sceneGlow1, sceneGlow2, sceneGlow3  ui.Color
	glassVeil, glassTint                ui.Color
	paneFill, paneFillEnd               ui.Color
	paneBase, paneOpaque                ui.Color
	card, cardEnd                       ui.Color
	cardQuiet, cardQuietEnd             ui.Color
	cardHover, cardNested               ui.Color
	cardOnOpaque, cardQuietOnOpaque     ui.Color
	control, controlEnd, controlHover   ui.Color
	controlOnMenu, controlOnMenuHover   ui.Color
	controlOnSelection                  ui.Color
	controlOnSelectionHover             ui.Color
	controlSelected, textOnSelected     ui.Color
	well, field, consent                ui.Color
	popover, menu                       ui.Color
	scrim, scrimModal, inactiveDim      ui.Color
	hairline, rule, edgeHC              ui.Color
	rowHover                            ui.Color
	toggleOff, toggleKnobOff            ui.Color
	toolbarGlyph, toolbarGlyphOff       ui.Color
	toolbarExpanded                     ui.Color
	textPrimary, textSecondary          ui.Color
	textTertiary, textPlaceholder       ui.Color
	textOnStatus                        ui.Color
	accent, accentEnd, accentPressed    ui.Color
	textOnAccent                        ui.Color
	accentText, linkText                ui.Color
	accentSoft, accentTint              ui.Color
	tabEmber, tabEmberHalo              ui.Color
	focusRing, selection                ui.Color
	menuHover, menuHoverText            ui.Color
	positive, positiveText, positiveTint ui.Color
	caution, cautionText, cautionTint   ui.Color
	critical, criticalText, criticalTint ui.Color
	neutral, neutralTint                ui.Color
	destructiveText                     ui.Color
	badgeFill, badgeFillSubtle          ui.Color
	surfaceChart, chartGrid             ui.Color
	data                                [8]ui.Color
	ink                                 ui.Color

	shadowFloat shadowSpec
	shadowModal shadowSpec
}

var (
	tokensLight = Tokens{
		Name:                    ThemeLight,
		scene:                   ui.Hex("#e6eaf1"),
		sceneWarm:               ui.Hex("#f6f7fa"),
		sceneGlow1:              ui.Hex("#c4d7f2"),
		sceneGlow2:              ui.Hex("#dcd3ee"),
		sceneGlow3:              ui.Hex("#cde6dc"),
		glassVeil:               ui.Hex("#e3e6eca8"),
		glassTint:               ui.Hex("#e3e6ec2a"), // 25% of glassVeil alpha (GlassTheme.glassTint)
		paneFill:                ui.Hex("#f4f6f914"),
		paneFillEnd:             ui.Hex("#f4f6f914"),
		paneBase:                ui.Hex("#f4f5f8f5"),
		paneOpaque:              ui.Hex("#f2f3f6"),
		card:                    ui.Hex("#ffffff70"), // Crisp translucent white card over cool light glass
		cardEnd:                 ui.Hex("#ffffff70"),
		cardQuiet:               ui.Hex("#ffffff54"),
		cardQuietEnd:            ui.Hex("#ffffff54"),
		cardHover:               ui.Hex("#f7f8fbcc"),
		cardNested:              ui.Hex("#0000000b"),
		cardOnOpaque:            ui.Hex("#ffffff"),
		cardQuietOnOpaque:       ui.Hex("#f8f9fb"),
		control:                 ui.Hex("#ffffff70"), // 44% white (GlassTokens.Gradient.controlFill flat)
		controlEnd:              ui.Hex("#ffffff70"),
		controlHover:            ui.Hex("#ebebedd6"), // 84% #ebebed (replaces control fill)
		controlOnMenu:           ui.Hex("#00000014"), // 8% black (GlassTokens.Color.controlOnMenu)
		controlOnMenuHover:      ui.Hex("#00000021"), // 13% black
		controlOnSelection:      ui.Hex("#0000002e"), // 18% black
		controlOnSelectionHover: ui.Hex("#00000047"), // 28% black
		controlSelected:         ui.Hex("#035ca3"),
		textOnSelected:          ui.Hex("#ffffff"),
		well:                    ui.Hex("#8a93a547"), // 28% #8a93a5 (GlassTokens.Color.wellFill flat)
		field:                   ui.Hex("#f4f6f999"),
		consent:                 ui.Hex("#f4f6f973"),
		popover:                 ui.Hex("#f7f7f9f0"), // 94% opaque per NDS-24
		menu:                    ui.Hex("#f7f7f9f0"), // 94% opaque per NDS-24
		scrim:                   ui.Hex("#00000040"),
		scrimModal:              ui.Hex("#00000040"),
		inactiveDim:             ui.Hex("#0000000a"),
		hairline:                ui.Hex("#00000014"),
		rule:                    ui.Hex("#0000001f"),
		edgeHC:                  ui.Hex("#1d1d1f8c"),
		rowHover:                ui.Hex("#0000000a"),
		toggleOff:               ui.Hex("#00000029"), // 16% black
		toggleKnobOff:           ui.Hex("#ffffff"),
		toolbarGlyph:            ui.Hex("#1d1d1f"),
		toolbarGlyphOff:         ui.Hex("#8a8a90"),   // GlassTokens.Color.toolbarGlyphHidden
		toolbarExpanded:         ui.Hex("#035ca3"),   // GlassTokens.Color.toolbarExpanded flat
		textPrimary:             ui.Hex("#1d1d1f"),
		textSecondary:           ui.Hex("#3c3c43"),
		textTertiary:            ui.Hex("#3d3d43"),
		textPlaceholder:         ui.Hex("#000000a3"),
		textOnStatus:            ui.Hex("#0c0c0e"),
		accent:                  ui.Hex("#035ca3"),
		accentEnd:               ui.Hex("#035ca3"),
		accentPressed:           ui.Hex("#00488f"),
		textOnAccent:            ui.Hex("#ffffff"),
		accentText:              ui.Hex("#024175"),
		linkText:                ui.Hex("#024175"),
		accentSoft:              ui.Hex("#0070c6"),
		accentTint:              ui.Hex("#0070c624"),
		tabEmber:                ui.Hex("#0070c6"),
		tabEmberHalo:            ui.Hex("#0070c638"), // alpha 0.22
		focusRing:               ui.Hex("#035ca3"),
		selection:               ui.Hex("#035ca3"),
		menuHover:               ui.Hex("#035ca3e6"),
		menuHoverText:           ui.Hex("#ffffff"),
		positive:                ui.Hex("#1a8f4c"),
		positiveText:            ui.Hex("#0b4628"),
		positiveTint:            ui.Hex("#1a8f4c24"),
		caution:                 ui.Hex("#a8690f"),
		cautionText:             ui.Hex("#573403"),
		cautionTint:             ui.Hex("#a8690f24"),
		critical:                ui.Hex("#d0342c"),
		criticalText:            ui.Hex("#771814"),
		criticalTint:            ui.Hex("#d0342c1f"),
		neutral:                 ui.Hex("#8a8a90"),
		neutralTint:             ui.Hex("#0000000f"),
		destructiveText:         ui.Hex("#791313"),
		badgeFill:               ui.Hex("#e04a42"),
		badgeFillSubtle:         ui.Hex("#8a93a547"),
		surfaceChart:            ui.Hex("#f7f8fa"),
		chartGrid:               ui.Hex("#00000014"),
		data: [8]ui.Color{
			ui.Hex("#2a78d6"), ui.Hex("#cf5320"), ui.Hex("#14906a"), ui.Hex("#a97200"),
			ui.Hex("#c94a7c"), ui.Hex("#2f8a1f"), ui.Hex("#5a48b8"), ui.Hex("#d33c3c"),
		},
		ink:         ui.Hex("#000000"),
		shadowFloat: shadowSpec{y: 18, blur: 36, color: ui.Hex("#0000001f")},
		shadowModal: shadowSpec{y: 24, blur: 60, color: ui.Hex("#00000039")},
	}

	tokensDark = Tokens{
		Name:                    ThemeDark,
		Dark:                    true,
		scene:                   ui.Hex("#070a1a"),
		sceneWarm:               ui.Hex("#11152b"),
		sceneGlow1:              ui.Hex("#3f6a8a"),
		sceneGlow2:              ui.Hex("#6a3f7a"),
		sceneGlow3:              ui.Hex("#2f6b5a"),
		glassVeil:               ui.Hex("#070a1aad"), // alpha 0.68
		glassTint:               ui.Hex("#070a1a2b"), // 25% of glassVeil alpha (GlassTheme.glassTint)
		paneFill:                ui.Hex("#ffffff0a"), // 4% white
		paneFillEnd:             ui.Hex("#ffffff0a"),
		paneBase:                ui.Hex("#0d1124f5"),
		paneOpaque:              ui.Hex("#0d1124"),
		card:                    ui.Hex("#0d112459"), // 35% #0d1124 (GlassTokens.Gradient.cardFill flat)
		cardEnd:                 ui.Hex("#0d112459"),
		cardQuiet:               ui.Hex("#0d112447"), // 28% #0d1124 (GlassTokens.Gradient.cardFillQuiet flat)
		cardQuietEnd:            ui.Hex("#0d112447"),
		cardHover:               ui.Hex("#ffffff14"),
		cardNested:              ui.Hex("#ffffff12"),
		cardOnOpaque:            ui.Hex("#171c33"),
		cardQuietOnOpaque:       ui.Hex("#13182d"),
		control:                 ui.Hex("#0000004c"), // 30% black (GlassTokens.Gradient.controlFill flat)
		controlEnd:              ui.Hex("#0000004c"),
		controlHover:            ui.Hex("#00000070"), // 44% black (replaces control fill)
		controlOnMenu:           ui.Hex("#ffffff1a"), // 10% white (GlassTokens.Color.controlOnMenu)
		controlOnMenuHover:      ui.Hex("#ffffff2e"), // 18% white
		controlOnSelection:      ui.Hex("#071a2e29"), // 16% #071a2e
		controlOnSelectionHover: ui.Hex("#071a2e42"), // 26% #071a2e
		controlSelected:         ui.Hex("#7ebcff"),
		textOnSelected:          ui.Hex("#071a2e"),
		well:                    ui.Hex("#0000004c"), // 30% black
		field:                   ui.Hex("#0000004c"),
		consent:                 ui.Hex("#0000004c"),
		popover:                 ui.Hex("#0b0f22f0"), // 94% opaque per NDS-24
		menu:                    ui.Hex("#0b0f22f0"), // 94% opaque per NDS-24
		scrim:                   ui.Hex("#00000059"),
		scrimModal:              ui.Hex("#00000066"),
		inactiveDim:             ui.Hex("#070a1a24"),
		hairline:                ui.Hex("#ffffff14"),
		rule:                    ui.Hex("#ffffff1f"),
		edgeHC:                  ui.Hex("#f2f2f466"),
		rowHover:                ui.Hex("#ffffff0d"), // 5% white
		toggleOff:               ui.Hex("#ffffff61"), // 38% white (GlassTokens.Color.toggleOff flat)
		toggleKnobOff:           ui.Hex("#071a2e"),   // Navy off knob in dark flat (GlassTokens.Color.toggleKnobOff)
		toolbarGlyph:            ui.Hex("#e6e6ec"),
		toolbarGlyphOff:         ui.Hex("#83838c"),   // GlassTokens.Color.toolbarGlyphHidden
		toolbarExpanded:         ui.Hex("#7ebcff"),   // GlassTokens.Color.toolbarExpanded flat
		textPrimary:             ui.Hex("#f2f3f8"),
		textSecondary:           ui.Hex("#f2f3f8e0"),
		textTertiary:            ui.Hex("#f2f3f8d6"),
		textPlaceholder:         ui.Hex("#ffffffcc"),
		textOnStatus:            ui.Hex("#0c0c0e"),
		accent:                  ui.Hex("#7ebcff"),
		accentEnd:               ui.Hex("#7ebcff"),
		accentPressed:           ui.Hex("#6aa8eb"),
		textOnAccent:            ui.Hex("#071a2e"),
		accentText:              ui.Hex("#c6e1ff"),
		linkText:                ui.Hex("#c6e1ff"),
		accentSoft:              ui.Hex("#0091fd"),
		accentTint:              ui.Hex("#0091fd38"),
		tabEmber:                ui.Hex("#0091fd"),
		tabEmberHalo:            ui.Hex("#0091fd4f"), // alpha 0.31
		focusRing:               ui.Hex("#7ebcff"),
		selection:               ui.Hex("#7ebcff"),
		menuHover:               ui.Hex("#7ebcffe6"),
		menuHoverText:           ui.Hex("#071a2e"),
		positive:                ui.Hex("#30d158"),
		positiveText:            ui.Hex("#acedbc"),
		positiveTint:            ui.Hex("#30d1582e"),
		caution:                 ui.Hex("#ffc75c"),
		cautionText:             ui.Hex("#ffd585"),
		cautionTint:             ui.Hex("#ffc75c2e"),
		critical:                ui.Hex("#ff6b6b"),
		criticalText:            ui.Hex("#ffd3d3"),
		criticalTint:            ui.Hex("#ff6b6b2e"),
		neutral:                 ui.Hex("#a9a9b0"),
		neutralTint:             ui.Hex("#ffffff1a"),
		destructiveText:         ui.Hex("#ffd0d0"),
		badgeFill:               ui.Hex("#ff6b6b"),
		badgeFillSubtle:         ui.Hex("#ffffff29"),
		surfaceChart:            ui.Hex("#0d1124"),
		chartGrid:               ui.Hex("#ffffff1a"),
		data: [8]ui.Color{
			ui.Hex("#0091fd"), ui.Hex("#d95926"), ui.Hex("#199e70"), ui.Hex("#c98500"),
			ui.Hex("#d55181"), ui.Hex("#008300"), ui.Hex("#9085e9"), ui.Hex("#e66767"),
		},
		ink:         ui.Hex("#ffffff"),
		shadowFloat: shadowSpec{y: 18, blur: 36, color: ui.Hex("#02030a99")},
		shadowModal: shadowSpec{y: 24, blur: 60, color: ui.Hex("#00000080")},
	}

	tokensLightHC = func() Tokens {
		t := tokensLight
		t.Name = ThemeLightHC
		t.HC = true
		t.glassVeil = ui.Hex("#e3e6ece6")
		t.card = ui.Hex("#ffffff8c")
		t.cardEnd = t.card
		t.cardQuiet = ui.Hex("#ffffff80")
		t.cardQuietEnd = t.cardQuiet
		t.control = ui.Hex("#ffffff99")
		t.controlEnd = t.control
		t.field = ui.Hex("#ffffffb2")
		t.hairline = ui.Hex("#00000073")
		t.rule = ui.Hex("#00000073")
		t.toolbarGlyphOff = ui.Hex("#2a2a2e")
		t.textSecondary = ui.Hex("#2a2a2e")
		t.textTertiary = ui.Hex("#2a2a2e")
		t.textPlaceholder = ui.Hex("#2a2a2e")
		t.textOnStatus = ui.Hex("#ffffff")
		t.accent = ui.Hex("#01599f")
		t.accentEnd = t.accent
		t.accentPressed = ui.Hex("#00458b")
		t.controlSelected = t.accent
		t.toolbarExpanded = t.accent
		t.accentText = ui.Hex("#013e71")
		t.linkText = ui.Hex("#013e71")
		t.focusRing = ui.Hex("#01599f")
		t.selection = ui.Hex("#01599f")
		t.menuHover = ui.Hex("#01599f")
		t.badgeFill = ui.Hex("#b01218")
		return t
	}()

	tokensDarkHC = func() Tokens {
		t := tokensDark
		t.Name = ThemeDarkHC
		t.HC = true
		t.glassVeil = ui.Hex("#070a1ae6")
		t.card = ui.Hex("#0d11248c")
		t.cardEnd = t.card
		t.cardQuiet = ui.Hex("#0d112480")
		t.cardQuietEnd = t.cardQuiet
		t.hairline = ui.Hex("#ffffff66")
		t.rule = ui.Hex("#ffffff66")
		t.toolbarGlyphOff = ui.Hex("#d6d6dc")
		t.textSecondary = ui.Hex("#e6e6ec")
		t.textTertiary = ui.Hex("#d6d6dc")
		t.textPlaceholder = ui.Hex("#e6e6ec")
		t.menuHover = ui.Hex("#7ebcff")
		return t
	}()

	tokensTraceLight = func() Tokens {
		t := tokensLight
		t.Name = ThemeTraceLight
		t.toolbarGlyphOff = ui.Hex("#8a8a90")
		t.accent = ui.Hex("#15608a")
		t.accentEnd = t.accent
		t.accentPressed = ui.Hex("#014c76")
		t.controlSelected = t.accent
		t.toolbarExpanded = t.accent
		t.accentText = ui.Hex("#0c3f5c")
		t.linkText = ui.Hex("#0c3f5c")
		t.accentSoft = ui.Hex("#1f7aa8")
		t.accentTint = ui.Hex("#7a2cf524")
		t.tabEmber = ui.Hex("#2a8fc4")
		t.tabEmberHalo = ui.Hex("#2a8fc44c")
		t.focusRing = ui.Hex("#15608a")
		t.selection = ui.Hex("#15608a")
		t.menuHover = ui.Hex("#15608ae6")
		t.data[0] = ui.Hex("#7a2cf5")
		t.data[1] = ui.Hex("#2f6bc4")
		t.data[2] = ui.Hex("#7d5ce0")
		return t
	}()

	tokensTraceDark = func() Tokens {
		t := tokensDark
		t.Name = ThemeTraceDark
		t.toolbarGlyphOff = ui.Hex("#83838c")
		t.toolbarExpanded = ui.Hex("#6d14f3")
		t.accent = ui.Hex("#6d14f3")
		t.accentEnd = t.accent
		t.accentPressed = ui.Hex("#5900df")
		t.controlSelected = ui.Hex("#6d14f3")
		t.textOnAccent = ui.Hex("#ffffff")
		t.textOnSelected = ui.Hex("#ffffff")
		t.toggleKnobOff = ui.Hex("#ffffff")
		t.accentText = ui.Hex("#e1d5ff")
		t.linkText = ui.Hex("#e1d5ff")
		t.accentSoft = ui.Hex("#8a3dff")
		t.accentTint = ui.Hex("#8a3dff38")
		t.tabEmber = ui.Hex("#8a3dff")
		t.tabEmberHalo = ui.Hex("#8a3dff4f")
		t.focusRing = ui.Hex("#c9b3ff")
		t.selection = ui.Hex("#6d14f3")
		t.menuHover = ui.Hex("#6d14f3e6")
		t.menuHoverText = ui.Hex("#ffffff")
		t.controlOnSelection = ui.Hex("#0000002e")
		t.controlOnSelectionHover = ui.Hex("#00000047")
		t.data[0] = ui.Hex("#8a3dff")
		t.data[1] = ui.Hex("#3a7bd5")
		t.data[2] = ui.Hex("#a78bfa")
		return t
	}()

	// tokensClassicLight and tokensClassicDark implement TCDesign's GlassTheme.classic
	// (specular rim highlights, 160°/180° glass sheen gradients, and purple brand CTAs).
	tokensClassicLight = func() Tokens {
		t := tokensLight
		t.Name = ThemeClassicLight
		t.Classic = true
		t.glassVeil = ui.Hex("#ffffff4d")
		t.glassTint = ui.Hex("#ffffff13")
		t.paneFill = ui.Hex("#ffffff8c")
		t.paneFillEnd = ui.Hex("#ffffff40")
		t.paneBase = ui.Hex("#f4f5f8f5")
		t.paneOpaque = ui.Hex("#f2f3f6")
		t.card = ui.Hex("#ffffffb8")
		t.cardEnd = ui.Hex("#ffffff85")
		t.cardQuiet = ui.Hex("#ffffff99")
		t.cardQuietEnd = ui.Hex("#ffffff6b")
		t.cardHover = ui.Hex("#ffffffdb")
		t.control = ui.Hex("#ffffffeb")
		t.controlEnd = ui.Hex("#ffffffb8")
		t.controlHover = ui.Hex("#ebebedd6")
		t.controlSelected = ui.Hex("#ffffff")
		t.textOnSelected = ui.Hex("#1d1d1f")
		t.well = ui.Hex("#0000000f")
		t.toolbarExpanded = ui.Hex("#d1d1d6")
		t.textSecondary = ui.Hex("#48484f")
		t.textTertiary = ui.Hex("#5c5c63")
		t.accent = ui.Hex("#7a2cf5")
		t.accentEnd = ui.Hex("#6d14f3")
		t.accentPressed = ui.Hex("#5a12c9")
		t.textOnAccent = ui.Hex("#ffffff")
		t.accentText = ui.Hex("#5b10cc")
		t.linkText = ui.Hex("#5b10cc")
		t.accentSoft = ui.Hex("#7a2cf5")
		t.accentTint = ui.Hex("#7a2cf524")
		t.tabEmber = ui.Hex("#7a2cf5d6")
		t.tabEmberHalo = ui.Hex("#7a2cf538")
		t.focusRing = ui.Hex("#7a2cf5")
		t.selection = ui.Hex("#2f6bc4")
		t.menuHover = ui.Hex("#2a62b5e6")
		t.menuHoverText = ui.Hex("#ffffff")
		t.shadowFloat = shadowSpec{y: 12, blur: 36, color: ui.Hex("#00000022")}
		return t
	}()

	tokensClassicDark = func() Tokens {
		t := tokensDark
		t.Name = ThemeClassicDark
		t.Classic = true
		t.scene = ui.Hex("#0f1219")
		t.sceneWarm = ui.Hex("#1d2430")
		t.glassVeil = ui.Hex("#0c0e1447")
		t.glassTint = ui.Hex("#0c0e1412")
		t.paneFill = ui.Hex("#ffffff1a")
		t.paneFillEnd = ui.Hex("#ffffff08")
		t.paneBase = ui.Hex("#161a22f5")
		t.paneOpaque = ui.Hex("#1c1e24")
		t.cardOnOpaque = ui.Hex("#242730")
		t.cardQuietOnOpaque = ui.Hex("#20232b")
		t.card = ui.Hex("#ffffff16")
		t.cardEnd = ui.Hex("#ffffff0b")
		t.cardQuiet = ui.Hex("#ffffff13")
		t.cardQuietEnd = ui.Hex("#ffffff0a")
		t.cardHover = ui.Hex("#ffffff1a")
		t.control = ui.Hex("#ffffff24")
		t.controlEnd = ui.Hex("#ffffff12")
		t.controlHover = ui.Hex("#ffffff29")
		t.controlSelected = ui.Hex("#ffffff29")
		t.textOnSelected = ui.Hex("#f2f2f4")
		t.well = ui.Hex("#00000038")
		t.popover = ui.Hex("#22242af0")
		t.menu = ui.Hex("#22242af0")
		t.inactiveDim = ui.Transparent
		t.toggleOff = ui.Hex("#ffffff2e")
		t.toggleKnobOff = ui.Hex("#ffffff")
		t.toolbarExpanded = ui.Hex("#55555a")
		t.textPrimary = ui.Hex("#f2f2f4")
		t.textSecondary = ui.Hex("#c9c9d0")
		t.textTertiary = ui.Hex("#b4b4bc")
		t.accent = ui.Hex("#8a3dffb8")
		t.accentEnd = ui.Hex("#6d14f39e")
		t.accentPressed = ui.Hex("#5900df")
		t.textOnAccent = ui.Hex("#ffffff")
		t.accentText = ui.Hex("#c9b3ff")
		t.linkText = ui.Hex("#c9b3ff")
		t.accentSoft = ui.Hex("#8a3dff")
		t.accentTint = ui.Hex("#8a3dff38")
		t.tabEmber = ui.Hex("#8a3dff")
		t.tabEmberHalo = ui.Hex("#8a3dff4f")
		t.focusRing = ui.Hex("#c9b3ff")
		t.selection = ui.Hex("#3275d3")
		t.menuHover = ui.Hex("#2f6ac0")
		t.menuHoverText = ui.Hex("#f2f2f4")
		t.controlOnSelection = ui.Hex("#0000002e")
		t.controlOnSelectionHover = ui.Hex("#00000047")
		t.shadowFloat = shadowSpec{y: 12, blur: 36, color: ui.Hex("#0000004d")}
		return t
	}()
)

// AllThemes returns the six official Near AI OS flat-glass themes for WCAG contrast tests.
func AllThemes() []*Tokens {
	return []*Tokens{&tokensLight, &tokensDark, &tokensLightHC, &tokensDarkHC, &tokensTraceLight, &tokensTraceDark}
}

// TokensByName returns the token set for a named theme.
func TokensByName(name ThemeName) *Tokens {
	switch name {
	case ThemeDark:
		return &tokensDark
	case ThemeLightHC:
		return &tokensLightHC
	case ThemeDarkHC:
		return &tokensDarkHC
	case ThemeTraceLight:
		return &tokensTraceLight
	case ThemeTraceDark:
		return &tokensTraceDark
	case ThemeClassicLight:
		return &tokensClassicLight
	case ThemeClassicDark:
		return &tokensClassicDark
	default:
		return &tokensLight
	}
}

func envGlassTheme() string {
	for _, k := range []string{"MULTIPLEXER_GLASS_THEME", "TC_GLASS_THEME"} {
		if v := strings.ToLower(strings.TrimSpace(os.Getenv(k))); v == "classic" || v == "trace" {
			return v
		}
	}
	return ""
}

// resolveThemeName resolves product, appearance, and increased-contrast axes
// per Near AI OS 01-architecture.md & TCDesign/GlassTheme.swift.
func resolveThemeName(dark, highContrast bool, product string) ThemeName {
	if highContrast {
		if dark {
			return ThemeDarkHC
		}
		return ThemeLightHC
	}
	if product == "" {
		product = envGlassTheme()
	}
	switch product {
	case "classic":
		if dark {
			return ThemeClassicDark
		}
		return ThemeClassicLight
	case "trace":
		if dark {
			return ThemeTraceDark
		}
		return ThemeTraceLight
	default:
		if dark {
			return ThemeDark
		}
		return ThemeLight
	}
}

// Ground describes what a control sits on (TCDesign/GlassStyle.swift GlassGround).
type Ground uint8

const (
	GroundPane Ground = iota
	GroundMenu
	GroundSelection
)

// themeState bundles the resolved tokens and active environmental axes for the current frame.
type themeState struct {
	*Tokens
	Vibrancy           bool
	NativeGlass        bool
	ReduceTransparency bool
	ReduceMotion       bool
	Opaque             bool
	Dense              bool
	TextScale          float32
}

// colors is kept as an alias to themeState so internal callers and tests share one signature.
type colors = *themeState

// forceSceneBackdrop is toggled temporarily during offscreen CapturePage() so
// transparent Liquid Glass panes render over the Near AI OS atmospheric scene.
var forceSceneBackdrop bool

// themeOf resolves the active Near AI OS / TCDesign theme and syncs MyGo's widget theme.
func themeOf(c *ui.Context) *themeState {
	p := c.Preferences()
	dark := c.Theme().Dark
	switch prefs.Appearance {
	case "light":
		dark = false
	case "dark":
		dark = true
	}
	hc := p.HighContrast || prefs.HighContrast
	reduceTrans := prefs.ReduceTransparency || systemReduceTransparency()
	nativeGlass := isMac && !reduceTrans && !forceSceneBackdrop
	vib := (c.Vibrancy() || nativeGlass) && !reduceTrans
	scale := p.TextScale
	if scale <= 0 {
		scale = 1
	}
	tk := TokensByName(resolveThemeName(dark, hc, prefs.ProductTheme))
	ts := &themeState{
		Tokens:             tk,
		Vibrancy:           vib,
		NativeGlass:        nativeGlass,
		ReduceTransparency: reduceTrans,
		ReduceMotion:       p.ReduceMotion,
		Opaque:             reduceTrans,
		Dense:              prefs.Density == "dense",
		TextScale:          scale,
	}
	syncUITheme(c, ts)
	return ts
}

func colorsOf(c *ui.Context) *themeState { return themeOf(c) }

func (ts *themeState) paneRadius() float32 {
	if ts.Classic {
		return 16
	}
	return radiusPane
}

func (ts *themeState) cardRadius() float32 {
	if ts.Dense {
		return 12
	}
	if ts.Classic {
		return 14
	}
	return radiusCard
}

func (ts *themeState) cardQuietRadius() float32 {
	if ts.Classic {
		return 12
	}
	return radiusCardQuiet
}

func (ts *themeState) controlRadius() float32 {
	if ts.Classic {
		return 8
	}
	return radiusControl
}

func (ts *themeState) inkAlpha(a float32) ui.Color {
	return ts.ink.Alpha(a)
}

// darkenPress darkens a fill by ~8% on press, matching TCDesign GlassPress (brightness 0.92).
func darkenPress(col ui.Color) ui.Color {
	return ui.Color{
		R: uint8(float32(col.R) * 0.88),
		G: uint8(float32(col.G) * 0.88),
		B: uint8(float32(col.B) * 0.88),
		A: min(uint8(min(int(col.A)+16, 255)), 255),
	}
}

// syncUITheme configures MyGo's ui.Theme so focus rings, text selection,
// scrollbars, and SF Pro system typography follow active tokens.
func syncUITheme(c *ui.Context, ts *themeState) {
	base := float32(14)
	if runtime.GOOS == "darwin" {
		base = 13
	}
	border := ts.rule
	if ts.HC {
		border = ts.edgeHC
	}
	c.SetTheme(&ui.Theme{
		Dark:           ts.Dark,
		Background:     ts.paneOpaque,
		Surface:        ts.control,
		SurfaceHover:   ts.controlHover,
		SurfacePressed: darkenPress(ts.controlHover),
		Border:         border,
		Text:           ts.textPrimary,
		TextMuted:      ts.textSecondary,
		Accent:         ts.accent,
		AccentHover:    ts.accent,
		AccentPressed:  ts.accentPressed,
		AccentText:     ts.textOnAccent,
		Danger:         ts.critical,
		Warning:        ts.caution,
		Success:        ts.positive,
		Selection:      ts.selection.Alpha(0.28),
		Focus:          ts.focusRing,
		Inverse:        ts.popover,
		InverseText:    ts.textPrimary,
		Scrollbar:      ts.neutral.Alpha(0.45),
		ScrollbarWidth: 6,
		Radius:         ts.controlRadius(),
		Spacing:        4,
		FontSize:       base * ts.TextScale,
		Font:           "", // System UI font (SF Pro via Core Text on macOS; never bundled)
	})
}

// cardFill returns the painted card surface fill (35% translucent over Liquid Glass,
// or solid cardOnOpaque when Opaque/Settings/ReduceTransparency is active).
func (ts *themeState) cardFill() ui.Color {
	if ts.Opaque {
		return ts.cardOnOpaque
	}
	return ts.card
}

// cardQuietFill returns the quiet painted card surface fill (28–29% translucent over Liquid Glass,
// or solid cardQuietOnOpaque when Opaque is active).
func (ts *themeState) cardQuietFill() ui.Color {
	if ts.Opaque {
		return ts.cardQuietOnOpaque
	}
	return ts.cardQuiet
}

// hcEdge applies the 1pt solid edge-hc border required in increased-contrast
// themes, leaving standard flat themes borderless (NDS-2, 03-materials.md).
func (ts *themeState) hcEdge(e ui.Element) ui.Element {
	if ts.HC {
		e.Border(1, ts.edgeHC)
	}
	return e
}

func (ts *themeState) scaled(px float32) float32 {
	return px * ts.TextScale
}

// Platform typography steps (04-typography.md & TCDesign/GlassTokens.TypeScale).
type typeStep struct {
	size, line, tracking float32
	weight               int
	mono                 bool
}

var isMac = runtime.GOOS == "darwin"

func stepMicro() typeStep {
	if isMac {
		return typeStep{size: 10, line: 13, weight: 700}
	}
	return typeStep{size: 11, line: 14, weight: 700}
}

func stepEyebrow() typeStep {
	if isMac {
		return typeStep{size: 10, line: 14, tracking: 0.8, weight: 700}
	}
	return typeStep{size: 11, line: 16, tracking: 0.8, weight: 700}
}

func stepCaption() typeStep {
	if isMac {
		return typeStep{size: 11, line: 16, weight: 400}
	}
	return typeStep{size: 12, line: 16, weight: 400}
}

func stepLabel() typeStep {
	if isMac {
		return typeStep{size: 12, line: 16, weight: 500}
	}
	return typeStep{size: 13, line: 18, weight: 500}
}

func stepBody() typeStep {
	if isMac {
		return typeStep{size: 13, line: 18, weight: 400}
	}
	return typeStep{size: 14, line: 20, weight: 400}
}

func stepBodyStrong() typeStep {
	if isMac {
		return typeStep{size: 13, line: 18, weight: 600}
	}
	return typeStep{size: 14, line: 20, weight: 600}
}

func stepTitle() typeStep {
	if isMac {
		return typeStep{size: 15, line: 20, weight: 600}
	}
	return typeStep{size: 16, line: 22, weight: 600}
}

func stepHeading() typeStep {
	if isMac {
		return typeStep{size: 17, line: 22, weight: 600}
	}
	return typeStep{size: 20, line: 26, weight: 600}
}

func stepDisplay() typeStep {
	if isMac {
		return typeStep{size: 22, line: 28, tracking: -0.22, weight: 700}
	}
	return typeStep{size: 28, line: 34, tracking: -0.28, weight: 700}
}

func stepMono() typeStep {
	if isMac {
		return typeStep{size: 11, line: 16, weight: 400, mono: true}
	}
	return typeStep{size: 12, line: 18, weight: 400, mono: true}
}

const monoStack = "SF Mono, JetBrains Mono, Menlo, Consolas, monospace"

func applyType(e ui.Element, ts *themeState, s typeStep) ui.Element {
	e.FontSize(s.size * ts.TextScale).FixedLineHeight(s.line * ts.TextScale).FontWeight(s.weight)
	if s.tracking != 0 {
		e.LetterSpacing(s.tracking)
	}
	if s.mono {
		e.Font(monoStack).FontFeatures("tnum")
	}
	return e
}
