// Package background composes a backdrop — a solid color, gradient, or
// image — behind a piece of content, with optional padding, rounded
// corners, drop shadow, and blur.
package background

import (
	"image"
	"image/color"
	"image/draw"
	"math"

	"github.com/disintegration/imaging"
	"github.com/fogleman/gg"
)

type kind int

const (
	kindSolid kind = iota
	kindGradient
	kindImage
)

// ScaleMode controls how an image backdrop is fitted to the canvas.
type ScaleMode int

const (
	Fit ScaleMode = iota
	Fill
	Cover
	Stretch
	Tile
)

// BlurStyle selects the blur algorithm applied to a backdrop.
type BlurStyle int

const (
	GaussianBlur BlurStyle = iota
	PixelatedBlur
)

// Padding is the space between the content and the canvas edge.
type Padding struct {
	Top, Right, Bottom, Left int
}

// Background is a configurable backdrop. Build one with Solid, Gradient,
// Image, or ImageFromFile and chain the With* methods.
type Background struct {
	kind    kind
	padding Padding
	radius  float64
	shadow  *Shadow

	blurStyle  BlurStyle
	blurRadius float64

	// solid
	color color.Color

	// gradient
	gradient  GradientType
	stops     []Stop
	angle     float64
	centerX   float64
	centerY   float64
	intensity float64

	// image
	image   image.Image
	scale   ScaleMode
	opacity float64
}

// Solid creates a solid color backdrop.
func Solid(c color.Color) *Background {
	b := base(kindSolid)
	b.color = c
	return b
}

// Gradient creates a gradient backdrop of the given type.
func Gradient(t GradientType, stops ...Stop) *Background {
	b := base(kindGradient)
	b.gradient = t
	b.stops = stops
	return b
}

// Image creates an image backdrop.
func Image(img image.Image) *Background {
	b := base(kindImage)
	b.image = img
	return b
}

// ImageFromFile creates an image backdrop from a file on disk.
func ImageFromFile(path string) (*Background, error) {
	img, err := imaging.Open(path)
	if err != nil {
		return nil, err
	}
	return Image(img), nil
}

func base(k kind) *Background {
	return &Background{
		kind:      k,
		padding:   Padding{20, 20, 20, 20},
		color:     color.RGBA{240, 240, 240, 255},
		angle:     0,
		centerX:   0.5,
		centerY:   0.5,
		intensity: 5,
		scale:     Cover,
		opacity:   1,
	}
}

// WithPadding sets equal padding on all sides.
func (b *Background) WithPadding(px int) *Background {
	b.padding = Padding{px, px, px, px}
	return b
}

// WithPaddingDetailed sets the padding of each side individually.
func (b *Background) WithPaddingDetailed(top, right, bottom, left int) *Background {
	b.padding = Padding{top, right, bottom, left}
	return b
}

// WithCornerRadius rounds the corners of the backdrop.
func (b *Background) WithCornerRadius(r float64) *Background { b.radius = r; return b }

// WithShadow draws a drop shadow behind the content.
func (b *Background) WithShadow(s *Shadow) *Background { b.shadow = s; return b }

// WithBlur blurs the backdrop (gradient and image backdrops only).
func (b *Background) WithBlur(style BlurStyle, radius float64) *Background {
	b.blurStyle, b.blurRadius = style, radius
	return b
}

// WithAngle sets the gradient angle in degrees.
func (b *Background) WithAngle(deg float64) *Background { b.angle = deg; return b }

// WithCenter sets the gradient center as fractions of the canvas (0-1).
func (b *Background) WithCenter(x, y float64) *Background { b.centerX, b.centerY = x, y; return b }

// WithIntensity tunes spiral tightness and star point count.
func (b *Background) WithIntensity(v float64) *Background { b.intensity = v; return b }

// WithScaleMode sets how an image backdrop is fitted.
func (b *Background) WithScaleMode(m ScaleMode) *Background { b.scale = m; return b }

// WithOpacity sets the opacity of an image backdrop (0-1).
func (b *Background) WithOpacity(o float64) *Background {
	b.opacity = math.Max(0, math.Min(1, o))
	return b
}

// Render draws the backdrop behind (and around) the content image.
// A nil content produces just the backdrop at its padding size.
func (b *Background) Render(content image.Image) (image.Image, error) {
	if content == nil {
		content = image.NewRGBA(image.Rect(0, 0, 0, 0))
	}
	if b.shadow != nil {
		content = b.shadow.Apply(content, b.radius)
	}

	cb := content.Bounds()
	w := cb.Dx() + b.padding.Left + b.padding.Right
	h := cb.Dy() + b.padding.Top + b.padding.Bottom

	canvas := b.paint(w, h)
	if b.blurRadius > 0 && b.kind != kindSolid {
		canvas = blur(canvas, b.blurStyle, b.blurRadius)
	}
	if b.radius > 0 {
		canvas = maskCorners(canvas, b.radius)
	}

	at := image.Pt(b.padding.Left, b.padding.Top)
	draw.Draw(canvas, cb.Sub(cb.Min).Add(at), content, cb.Min, draw.Over)
	return canvas, nil
}

// paint fills a canvas of the given size according to the backdrop kind.
func (b *Background) paint(w, h int) *image.RGBA {
	canvas := image.NewRGBA(image.Rect(0, 0, w, h))
	switch b.kind {
	case kindSolid:
		draw.Draw(canvas, canvas.Bounds(), image.NewUniform(b.color), image.Point{}, draw.Src)
	case kindGradient:
		b.paintGradient(canvas)
	case kindImage:
		scaled := scaleImage(b.image, w, h, b.scale)
		alpha := image.NewUniform(color.Alpha{A: uint8(b.opacity * 255)})
		draw.DrawMask(canvas, canvas.Bounds(), scaled, image.Point{}, alpha, image.Point{}, draw.Over)
	}
	return canvas
}

func blur(img *image.RGBA, style BlurStyle, radius float64) *image.RGBA {
	var out *image.NRGBA
	switch style {
	case PixelatedBlur:
		w, h := img.Bounds().Dx(), img.Bounds().Dy()
		factor := math.Max(1, radius)
		small := imaging.Resize(img, max(1, int(float64(w)/factor)), max(1, int(float64(h)/factor)), imaging.Box)
		out = imaging.Resize(small, w, h, imaging.NearestNeighbor)
	default:
		out = imaging.Blur(img, radius)
	}
	result := image.NewRGBA(img.Bounds())
	draw.Draw(result, result.Bounds(), out, image.Point{}, draw.Src)
	return result
}

// maskCorners clips an image to a rounded rectangle.
func maskCorners(img *image.RGBA, radius float64) *image.RGBA {
	bounds := img.Bounds()
	out := image.NewRGBA(bounds)
	draw.DrawMask(out, bounds, img, bounds.Min, roundedMask(bounds.Dx(), bounds.Dy(), radius), image.Point{}, draw.Over)
	return out
}

// roundedMask builds an anti-aliased alpha mask for a rounded rectangle.
func roundedMask(w, h int, radius float64) *image.Alpha {
	dc := gg.NewContext(w, h)
	dc.SetRGB(1, 1, 1)
	dc.DrawRoundedRectangle(0, 0, float64(w), float64(h), radius)
	dc.Fill()
	src := dc.Image().(*image.RGBA)
	mask := image.NewAlpha(src.Bounds())
	for i, j := 3, 0; i < len(src.Pix); i, j = i+4, j+1 {
		mask.Pix[j] = src.Pix[i]
	}
	return mask
}

func scaleImage(img image.Image, w, h int, mode ScaleMode) image.Image {
	sb := img.Bounds()
	srcRatio := float64(sb.Dx()) / float64(sb.Dy())
	dstRatio := float64(w) / float64(h)

	switch mode {
	case Stretch:
		return imaging.Resize(img, w, h, imaging.Lanczos)

	case Fill, Cover:
		nw, nh := w, int(float64(w)/srcRatio)
		if srcRatio > dstRatio {
			nw, nh = int(float64(h)*srcRatio), h
		}
		scaled := imaging.Resize(img, nw, nh, imaging.Lanczos)
		if mode == Cover {
			return imaging.CropCenter(scaled, w, h)
		}
		return imaging.Crop(scaled, image.Rect(0, 0, w, h))

	case Tile:
		tiled := image.NewRGBA(image.Rect(0, 0, w, h))
		for y := 0; y < h; y += sb.Dy() {
			for x := 0; x < w; x += sb.Dx() {
				draw.Draw(tiled, image.Rect(x, y, x+sb.Dx(), y+sb.Dy()), img, sb.Min, draw.Over)
			}
		}
		return tiled

	default: // Fit
		nw, nh := w, int(float64(w)/srcRatio)
		if srcRatio <= dstRatio {
			nw, nh = int(float64(h)*srcRatio), h
		}
		scaled := imaging.Resize(img, nw, nh, imaging.Lanczos)
		centered := image.NewRGBA(image.Rect(0, 0, w, h))
		at := image.Pt((w-nw)/2, (h-nh)/2)
		draw.Draw(centered, scaled.Bounds().Add(at), scaled, scaled.Bounds().Min, draw.Over)
		return centered
	}
}
