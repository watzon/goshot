// Package chrome draws window decorations — a title bar, window controls,
// and rounded corners — around a content image.
package chrome

import (
	"image"
	"image/color"
	"sort"

	"github.com/fogleman/gg"
	"github.com/watzon/goshot/fonts"
)

// Variant selects the light or dark palette of a style.
type Variant string

const (
	Light Variant = "light"
	Dark  Variant = "dark"
)

// Palette holds the colors a window style needs for one variant.
type Palette struct {
	TitleBar  color.Color
	TitleText color.Color
	Controls  color.Color
}

// Style describes how a family of windows draws itself. The Controls
// function paints the window buttons into the title bar.
type Style struct {
	Name           string
	TitleFont      string // font family for the title; "" uses the embedded sans
	TitleFontSize  float64
	TitleBarHeight int
	CornerRadius   float64
	Light, Dark    Palette
	Controls       func(dc *gg.Context, width, barHeight int, p Palette)
}

// Chrome renders window decorations for a single style.
type Chrome struct {
	style    *Style
	variant  Variant
	title    string
	radius   float64
	titleBar bool
}

// New creates a Chrome for the given style.
func New(s *Style) *Chrome {
	return &Chrome{style: s, variant: Light, radius: s.CornerRadius, titleBar: true}
}

// Named looks up a chrome style by name (see Names).
func Named(name string) (*Chrome, bool) {
	s, ok := registry[name]
	if !ok {
		return nil, false
	}
	return New(s), true
}

// Names lists the registered chrome style names.
func Names() []string {
	names := make([]string, 0, len(registry))
	for n := range registry {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// WithTitle sets the window title text.
func (c *Chrome) WithTitle(title string) *Chrome { c.title = title; return c }

// WithVariant selects the light or dark palette.
func (c *Chrome) WithVariant(v Variant) *Chrome { c.variant = v; return c }

// Dark selects the dark palette.
func (c *Chrome) Dark() *Chrome { return c.WithVariant(Dark) }

// Light selects the light palette.
func (c *Chrome) Light() *Chrome { return c.WithVariant(Light) }

// WithCornerRadius overrides the style's window corner radius.
func (c *Chrome) WithCornerRadius(r float64) *Chrome { c.radius = r; return c }

// WithTitleBar shows or hides the title bar.
func (c *Chrome) WithTitleBar(show bool) *Chrome { c.titleBar = show; return c }

// Style returns the underlying style definition.
func (c *Chrome) Style() *Style { return c.style }

// Render draws the window around the content image.
func (c *Chrome) Render(content image.Image) (image.Image, error) {
	if content == nil {
		content = image.NewRGBA(image.Rect(0, 0, 200, 100))
	}
	w := content.Bounds().Dx()
	h := content.Bounds().Dy()
	bar := 0
	if c.titleBar {
		bar = c.style.TitleBarHeight
	}

	p := c.style.Light
	if c.variant == Dark {
		p = c.style.Dark
	}

	dc := gg.NewContext(w, h+bar)
	dc.DrawRoundedRectangle(0, 0, float64(w), float64(h+bar), c.radius)
	dc.Clip()

	dc.SetColor(p.TitleBar)
	dc.DrawRectangle(0, 0, float64(w), float64(h+bar))
	dc.Fill()

	if bar > 0 {
		if c.style.Controls != nil {
			c.style.Controls(dc, w, bar, p)
		}
		if c.title != "" {
			c.drawTitle(dc, w, bar, p)
		}
	}

	dc.DrawImage(content, 0, bar)
	return dc.Image(), nil
}

func (c *Chrome) drawTitle(dc *gg.Context, width, barHeight int, p Palette) {
	family := fonts.FallbackSans()
	if c.style.TitleFont != "" {
		if f, err := fonts.Get(c.style.TitleFont); err == nil {
			family = f
		}
	}
	face, err := family.Face(c.style.TitleFontSize, fonts.Style{Weight: fonts.Bold})
	if err != nil {
		return
	}
	dc.SetFontFace(face)
	dc.SetColor(p.TitleText)
	dc.DrawStringAnchored(c.title, float64(width)/2, float64(barHeight)/2, 0.5, 0.35)
}
