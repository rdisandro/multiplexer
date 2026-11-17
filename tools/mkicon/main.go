// Mkicon draws Multiplexer's macOS/app icon (resources/icon.png, 1024×1024) in the
// Near AI OS Design System palette: a continuous-curvature macOS squircle on
// deep midnight navy (#070a1a -> #0d1124 -> #11152b) with an Ember Rule tab
// bar and flat-glass split terminal panes accented in NEAR AI blue (#0091fd / #7ebcff).
//
//	go run ./tools/mkicon
package main

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"math"
	"os"

	"golang.org/x/image/vector"
)

const size = 1024

func main() {
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	// Standard macOS 1024px icon grid: 824×824 squircle inset by 100px.
	x0, y0, w := float32(100), float32(100), float32(824)
	shadow(img, x0+8, y0+18, w-16, w-16, 186, 28, color.RGBA{2, 3, 10, 110})

	// Near AI OS dark scene ground with subtle top-left brand-blue radial glow.
	bg := sceneBackdrop(int(x0), int(y0), int(w))
	fill(img, squircle(x0, y0, w, w), bg)

	pad := float32(68)
	innerX := x0 + pad
	innerW := w - 2*pad

	// Top chrome: Ember Rule tab indicator + status pill inset comfortably inside squircle corners.
	barY := y0 + 62
	barX := innerX + 20
	barW := innerW - 40
	// Subtle 1px-equivalent rule under tab strip.
	fill(img, rounded(barX, barY+34, barW, 4, 2), image.NewUniform(color.NRGBA{255, 255, 255, 34}))
	// Active tab label bar + glowing tab-ember bar (#0091fd) underneath.
	fill(img, rounded(barX+20, barY+8, 128, 14, 7), image.NewUniform(rgb(0xf2f3f8)))
	fill(img, rounded(barX+6, barY+22, 156, 12, 6), image.NewUniform(color.NRGBA{0, 145, 253, 52}))
	fill(img, rounded(barX+12, barY+32, 144, 6, 3), image.NewUniform(rgb(0x0091fd)))
	// Inactive tab pill hint + right status dot.
	fill(img, rounded(barX+196, barY+8, 108, 14, 7), image.NewUniform(color.NRGBA{242, 243, 248, 72}))
	fill(img, rounded(barX+barW-26, barY+5, 20, 20, 10), image.NewUniform(rgb(0x30d158)))

	// Split terminal panes: flat cards (no card drop shadows, per NDS-2/NDS-9).
	panesTop := barY + 58
	panesH := (y0 + w - pad) - panesTop
	gap := float32(28)
	lw := innerW*0.55 - gap/2
	th := panesH*0.54 - gap/2

	// Focused top-left pane (card-on-opaque #171c33, lifted slightly for icon legibility).
	c0x, c0y, c0w, c0h := innerX, panesTop, lw, th
	fill(img, rounded(c0x, c0y, c0w, c0h, 40), image.NewUniform(color.NRGBA{26, 34, 64, 248}))

	// Unfocused bottom-left pane (card-quiet #13182d).
	c1x, c1y, c1w, c1h := innerX, panesTop+th+gap, lw, panesH-th-gap
	fill(img, rounded(c1x, c1y, c1w, c1h, 40), image.NewUniform(color.NRGBA{19, 25, 48, 224}))

	// Right split pane (card-quiet #13182d).
	c2x, c2y, c2w, c2h := innerX+lw+gap, panesTop, innerW-lw-gap, panesH
	fill(img, rounded(c2x, c2y, c2w, c2h, 40), image.NewUniform(color.NRGBA{19, 25, 48, 224}))

	// Focused pane prompt: luminous NEAR AI accent chevron (#7ebcff) + primary cursor (#f2f3f8).
	accent := image.NewUniform(rgb(0x7ebcff))
	ink := image.NewUniform(rgb(0xf2f3f8))
	cx, cy := c0x+58, c0y+c0h*0.52
	for _, seg := range stroke([][2]float32{{cx, cy - 52}, {cx + 56, cy}, {cx, cy + 52}}, 26) {
		fill(img, seg, accent)
	}
	fill(img, rounded(cx+88, cy+28, 96, 24, 12), ink)

	// Bottom-left pane: structured CLI output lines.
	lines1 := []struct {
		frac float32
		col  color.NRGBA
	}{
		{0.68, color.NRGBA{242, 243, 248, 185}},
		{0.46, color.NRGBA{0, 145, 253, 220}},
		{0.76, color.NRGBA{242, 243, 248, 120}},
	}
	for i, ln := range lines1 {
		ly := c1y + 50 + float32(i)*46
		fill(img, rounded(c1x+46, ly, (c1w-92)*ln.frac, 20, 10), image.NewUniform(ln.col))
	}

	// Right pane: header hairline + live process rows with positive/accent indicators.
	fill(img, rounded(c2x+36, c2y+68, c2w-72, 3, 1.5), image.NewUniform(color.NRGBA{255, 255, 255, 26}))
	fill(img, rounded(c2x+36, c2y+34, 108, 16, 8), image.NewUniform(color.NRGBA{242, 243, 248, 175}))

	rows := []struct {
		dot  color.NRGBA
		frac float32
	}{
		{rgb(0x30d158), 0.72},
		{rgb(0x0091fd), 0.54},
		{rgb(0x7ebcff), 0.66},
		{color.NRGBA{169, 169, 176, 180}, 0.44},
	}
	for i, rw := range rows {
		ry := c2y + 114 + float32(i)*78
		fill(img, rounded(c2x+40, ry, 22, 22, 11), image.NewUniform(rw.dot))
		fill(img, rounded(c2x+78, ry+2, (c2w-118)*rw.frac, 18, 9), image.NewUniform(color.NRGBA{242, 243, 248, 165}))
		fill(img, rounded(c2x+78, ry+30, (c2w-118)*(rw.frac*0.65), 12, 6), image.NewUniform(color.NRGBA{242, 243, 248, 78}))
	}

	if err := os.MkdirAll("resources", 0o755); err != nil {
		log.Fatal(err)
	}
	f, err := os.Create("resources/icon.png")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		log.Fatal(err)
	}
}

func rgb(v uint32) color.NRGBA {
	return color.NRGBA{uint8(v >> 16), uint8(v >> 8), uint8(v), 255}
}

// sceneBackdrop paints the Near AI OS midnight-navy canvas (#070a1a..#11152b)
// with a subtle radial brand-blue (#0091fd) glow in the upper-left quadrant.
func sceneBackdrop(x0, y0, w int) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	top := rgb(0x111833)
	mid := rgb(0x0d1124)
	bot := rgb(0x070a1a)
	glowX, glowY := float64(x0)+float64(w)*0.24, float64(y0)+float64(w)*0.18
	glowR := float64(w) * 0.72

	for y := y0; y < y0+w; y++ {
		t := float64(y-y0) / float64(w-1)
		var base color.NRGBA
		if t < 0.5 {
			base = lerpColor(top, mid, t/0.5)
		} else {
			base = lerpColor(mid, bot, (t-0.5)/0.5)
		}
		for x := x0; x < x0+w; x++ {
			dx, dy := float64(x)-glowX, float64(y)-glowY
			d := math.Hypot(dx, dy) / glowR
			c := base
			if d < 1 {
				g := (1 - d) * (1 - d) * 0.24
				c = lerpColor(c, rgb(0x0b8ff6), g)
			}
			img.SetNRGBA(x, y, c)
		}
	}
	return img
}

func lerpColor(a, b color.NRGBA, t float64) color.NRGBA {
	t = math.Max(0, math.Min(1, t))
	mix := func(p, q uint8) uint8 { return uint8(float64(p) + (float64(q)-float64(p))*t + 0.5) }
	return color.NRGBA{mix(a.R, b.R), mix(a.G, b.G), mix(a.B, b.B), 255}
}

func fill(dst *image.RGBA, r *vector.Rasterizer, src image.Image) {
	r.Draw(dst, dst.Bounds(), src, image.Point{})
}

func rounded(x, y, w, h, rad float32) *vector.Rasterizer {
	r := vector.NewRasterizer(size, size)
	rad = min(rad, min(w, h)/2)
	k := rad * 0.5523
	r.MoveTo(x+rad, y)
	r.LineTo(x+w-rad, y)
	r.CubeTo(x+w-rad+k, y, x+w, y+rad-k, x+w, y+rad)
	r.LineTo(x+w, y+h-rad)
	r.CubeTo(x+w, y+h-rad+k, x+w-rad+k, y+h, x+w-rad, y+h)
	r.LineTo(x+rad, y+h)
	r.CubeTo(x+rad-k, y+h, x, y+h-rad+k, x, y+h-rad)
	r.LineTo(x, y+rad)
	r.CubeTo(x, y+rad-k, x+rad-k, y, x+rad, y)
	r.ClosePath()
	return r
}

func squircle(x, y, w, h float32) *vector.Rasterizer {
	r := vector.NewRasterizer(size, size)
	const n = 5.0
	cx, cy, a, b := float64(x+w/2), float64(y+h/2), float64(w/2), float64(h/2)
	for i := 0; i <= 720; i++ {
		t := float64(i) / 720 * 2 * math.Pi
		c, s := math.Cos(t), math.Sin(t)
		px := cx + a*math.Copysign(math.Pow(math.Abs(c), 2/n), c)
		py := cy + b*math.Copysign(math.Pow(math.Abs(s), 2/n), s)
		if i == 0 {
			r.MoveTo(float32(px), float32(py))
		} else {
			r.LineTo(float32(px), float32(py))
		}
	}
	r.ClosePath()
	return r
}

func stroke(pts [][2]float32, width float32) []*vector.Rasterizer {
	var out []*vector.Rasterizer
	hw := width / 2
	for i := 0; i+1 < len(pts); i++ {
		r := vector.NewRasterizer(size, size)
		out = append(out, r)
		a, b := pts[i], pts[i+1]
		dx, dy := b[0]-a[0], b[1]-a[1]
		l := float32(math.Hypot(float64(dx), float64(dy)))
		nx, ny := -dy/l*hw, dx/l*hw
		r.MoveTo(a[0]+nx, a[1]+ny)
		r.LineTo(b[0]+nx, b[1]+ny)
		r.LineTo(b[0]-nx, b[1]-ny)
		r.LineTo(a[0]-nx, a[1]-ny)
		r.ClosePath()
	}
	for _, p := range pts {
		r := vector.NewRasterizer(size, size)
		circle(r, p[0], p[1], hw)
		out = append(out, r)
	}
	return out
}

func circle(r *vector.Rasterizer, cx, cy, rad float32) {
	k := rad * 0.5523
	r.MoveTo(cx+rad, cy)
	r.CubeTo(cx+rad, cy+k, cx+k, cy+rad, cx, cy+rad)
	r.CubeTo(cx-k, cy+rad, cx-rad, cy+k, cx-rad, cy)
	r.CubeTo(cx-rad, cy-k, cx-k, cy-rad, cx, cy-rad)
	r.CubeTo(cx+k, cy-rad, cx+rad, cy-k, cx+rad, cy)
	r.ClosePath()
}

func shadow(dst *image.RGBA, x, y, w, h, rad, blur float32, c color.RGBA) {
	steps := 12
	for i := steps; i >= 1; i-- {
		g := blur * float32(i) / float32(steps)
		a := float64(c.A) / float64(steps) * 0.9
		col := color.NRGBA{c.R, c.G, c.B, uint8(a)}
		m := rounded(x-g/2, y-g/2, w+g, h+g, rad+g/2)
		m.DrawOp = draw.Over
		fill(dst, m, image.NewUniform(col))
	}
}
