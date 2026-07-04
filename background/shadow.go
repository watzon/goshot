package background

import (
	"image"
	"image/color"
	"image/draw"
	"math"

	"github.com/disintegration/imaging"
	"github.com/fogleman/gg"
)

// Shadow describes a drop shadow drawn behind the content.
type Shadow struct {
	offsetX, offsetY float64
	blur             float64
	spread           float64
	color            color.Color
}

// NewShadow creates a shadow with sensible defaults.
func NewShadow() *Shadow {
	return &Shadow{offsetX: 5, offsetY: 5, blur: 10, color: color.RGBA{0, 0, 0, 128}}
}

// WithOffset moves the shadow relative to the content.
func (s *Shadow) WithOffset(x, y float64) *Shadow { s.offsetX, s.offsetY = x, y; return s }

// WithBlur sets the softness of the shadow edge.
func (s *Shadow) WithBlur(radius float64) *Shadow { s.blur = radius; return s }

// WithSpread grows the shadow beyond the content bounds.
func (s *Shadow) WithSpread(radius float64) *Shadow { s.spread = radius; return s }

// WithColor sets the shadow color.
func (s *Shadow) WithColor(c color.Color) *Shadow { s.color = c; return s }

// Apply returns the content composited over its shadow, expanded to fit.
// cornerRadius should match the corner radius of the content so the shadow
// hugs its silhouette.
func (s *Shadow) Apply(content image.Image, cornerRadius float64) image.Image {
	bounds := content.Bounds()
	margin := int(math.Ceil(2*s.blur + s.spread + math.Max(math.Abs(s.offsetX), math.Abs(s.offsetY))))

	w := bounds.Dx() + 2*margin
	h := bounds.Dy() + 2*margin
	out := image.NewRGBA(image.Rect(0, 0, w, h))

	// Silhouette of the (spread-grown) content, offset and blurred.
	dc := gg.NewContext(w, h)
	dc.SetColor(s.color)
	dc.DrawRoundedRectangle(
		float64(margin)+s.offsetX-s.spread,
		float64(margin)+s.offsetY-s.spread,
		float64(bounds.Dx())+2*s.spread,
		float64(bounds.Dy())+2*s.spread,
		cornerRadius+s.spread,
	)
	dc.Fill()

	silhouette := dc.Image()
	if s.blur > 0 {
		silhouette = imaging.Blur(silhouette, s.blur/2)
	}
	draw.Draw(out, out.Bounds(), silhouette, silhouette.Bounds().Min, draw.Over)
	draw.Draw(out, bounds.Sub(bounds.Min).Add(image.Pt(margin, margin)), content, bounds.Min, draw.Over)
	return out
}
