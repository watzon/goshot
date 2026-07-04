package background

import (
	"image"
	"image/color"
	"math"
	"sort"
)

// GradientType selects the shape of a gradient backdrop.
type GradientType int

const (
	LinearGradient GradientType = iota
	RadialGradient
	AngularGradient
	DiamondGradient
	SpiralGradient
	SquareGradient
	StarGradient
)

// Stop is a color at a position (0-1) along a gradient.
type Stop struct {
	Color    color.Color
	Position float64
}

// paintGradient fills the canvas with the configured gradient. Colors are
// sampled from a precomputed ramp for speed.
func (b *Background) paintGradient(canvas *image.RGBA) {
	bounds := canvas.Bounds()
	w, h := float64(bounds.Dx()), float64(bounds.Dy())
	cx, cy := w*b.centerX, h*b.centerY
	ramp := b.ramp(1024)

	pos := b.position(w, h, cx, cy)
	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			p := math.Max(0, math.Min(1, pos(float64(x), float64(y))))
			canvas.SetRGBA(x, y, ramp[int(p*float64(len(ramp)-1))])
		}
	}
}

// position returns a function mapping a pixel to its 0-1 gradient position.
func (b *Background) position(w, h, cx, cy float64) func(x, y float64) float64 {
	angleRad := b.angle * math.Pi / 180
	maxRadial := math.Sqrt(math.Max(cx*cx, (w-cx)*(w-cx)) + math.Max(cy*cy, (h-cy)*(h-cy)))
	maxCenter := math.Sqrt(cx*cx + cy*cy)

	switch b.gradient {
	case RadialGradient:
		return func(x, y float64) float64 { return math.Hypot(x-cx, y-cy) / maxRadial }
	case AngularGradient:
		return func(x, y float64) float64 {
			return math.Mod((math.Atan2(y-cy, x-cx)/math.Pi+1)/2+b.angle/360, 1)
		}
	case DiamondGradient:
		maxDist := math.Max(cx, w-cx) + math.Max(cy, h-cy)
		return func(x, y float64) float64 { return (math.Abs(x-cx) + math.Abs(y-cy)) / maxDist }
	case SpiralGradient:
		return func(x, y float64) float64 {
			angle := math.Atan2(y-cy, x-cx)
			dist := math.Hypot(x-cx, y-cy)
			return math.Mod((angle/math.Pi+1)/2+dist*b.intensity/maxCenter+b.angle/360, 1)
		}
	case SquareGradient:
		maxDist := math.Max(cx, w-cx)
		return func(x, y float64) float64 {
			return math.Max(math.Abs(x-cx), math.Abs(y-cy)) / maxDist
		}
	case StarGradient:
		return func(x, y float64) float64 {
			angle := math.Atan2(y-cy, x-cx)
			dist := math.Hypot(x-cx, y-cy)
			star := math.Abs(math.Sin(angle*b.intensity + angleRad))
			return (dist/maxCenter + star*0.5) / 1.5
		}
	default: // LinearGradient
		span := w*math.Abs(math.Cos(angleRad)) + h*math.Abs(math.Sin(angleRad))
		return func(x, y float64) float64 {
			return (x*math.Cos(angleRad) + y*math.Sin(angleRad)) / span
		}
	}
}

// ramp samples the gradient stops into n evenly spaced colors.
func (b *Background) ramp(n int) []color.RGBA {
	stops := append([]Stop(nil), b.stops...)
	sort.SliceStable(stops, func(i, j int) bool { return stops[i].Position < stops[j].Position })

	out := make([]color.RGBA, n)
	for i := range out {
		out[i] = colorAt(stops, float64(i)/float64(n-1))
	}
	return out
}

func colorAt(stops []Stop, pos float64) color.RGBA {
	switch len(stops) {
	case 0:
		return color.RGBA{A: 255}
	case 1:
		return toRGBA(stops[0].Color)
	}
	if pos <= stops[0].Position {
		return toRGBA(stops[0].Color)
	}
	for i := 0; i < len(stops)-1; i++ {
		a, z := stops[i], stops[i+1]
		if pos >= a.Position && pos <= z.Position {
			t := 0.0
			if z.Position > a.Position {
				t = (pos - a.Position) / (z.Position - a.Position)
			}
			return lerp(toRGBA(a.Color), toRGBA(z.Color), t)
		}
	}
	return toRGBA(stops[len(stops)-1].Color)
}

func lerp(a, z color.RGBA, t float64) color.RGBA {
	mix := func(x, y uint8) uint8 { return uint8(float64(x)*(1-t) + float64(y)*t) }
	return color.RGBA{mix(a.R, z.R), mix(a.G, z.G), mix(a.B, z.B), mix(a.A, z.A)}
}

func toRGBA(c color.Color) color.RGBA {
	r, g, b, a := c.RGBA()
	return color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)}
}
