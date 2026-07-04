// Package chrome draws window decorations — a title bar, window controls,
// and rounded corners — around a content image.
package chrome

import (
	"image"
	"image/color"
	"sort"

	"github.com/fogleman/gg"
	"github.com/watzon/goshot/fonts"
	"golang.org/x/image/font"
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
	style        *Style
	variant      Variant
	title        string
	radius       float64
	titleBar     bool
	barColor     color.Color // overrides the palette's title bar color
	textColor    color.Color // overrides the palette's title text color
	matchContent bool        // color the title bar like the content background
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

// WithTitleBarColor overrides the palette's title bar color.
func (c *Chrome) WithTitleBarColor(col color.Color) *Chrome { c.barColor = col; return c }

// WithTitleTextColor overrides the palette's title text color.
func (c *Chrome) WithTitleTextColor(col color.Color) *Chrome { c.textColor = col; return c }

// WithTitleBarMatchingContent colors the title bar like the content's
// background, so the window reads as one seamless surface. The title and
// controls switch to whichever palette stays readable.
func (c *Chrome) WithTitleBarMatchingContent() *Chrome { c.matchContent = true; return c }

// isLight reports whether a color reads as light (its luminance is
// closer to white than black).
func isLight(c color.Color) bool {
	r, g, b, _ := c.RGBA()
	luminance := 0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)
	return luminance > 0xFFFF/2
}

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
	if c.matchContent {
		p.TitleBar = content.At(content.Bounds().Min.X, content.Bounds().Min.Y)
	}
	if c.barColor != nil {
		p.TitleBar = c.barColor
	}
	if c.matchContent || c.barColor != nil {
		// Keep the title and controls readable on the new bar color.
		readable := c.style.Dark
		if isLight(p.TitleBar) {
			readable = c.style.Light
		}
		p.TitleText, p.Controls = readable.TitleText, readable.Controls
	}
	if c.textColor != nil {
		p.TitleText = c.textColor
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

	metrics := face.Metrics()
	emojiAdv := float64(font.MeasureString(face, "  ").Round())
	runs := fonts.SplitEmoji(c.title)

	drawsAsEmoji := func(r fonts.Run) bool {
		if !r.Emoji {
			return false
		}
		e := fonts.Emoji()
		if e == nil {
			return false
		}
		_, ok := e.Render(r.Text, float64(metrics.Ascent.Ceil()+metrics.Descent.Ceil()), p.TitleText)
		return ok
	}

	total := 0.0
	for _, run := range runs {
		if drawsAsEmoji(run) {
			total += emojiAdv
		} else {
			w, _ := dc.MeasureString(run.Text)
			total += w
		}
	}

	x := float64(width)/2 - total/2
	baseline := float64(barHeight)/2 + 0.35*float64(metrics.Height)/64
	rgba, _ := dc.Image().(*image.RGBA)
	for _, run := range runs {
		if drawsAsEmoji(run) && rgba != nil {
			box := image.Rect(int(x), int(baseline)-metrics.Ascent.Ceil(), int(x+emojiAdv), int(baseline)+metrics.Descent.Ceil())
			fonts.Emoji().Draw(rgba, run.Text, box, p.TitleText)
			x += emojiAdv
			continue
		}
		dc.DrawString(run.Text, x, baseline)
		w, _ := dc.MeasureString(run.Text)
		x += w
	}
}
