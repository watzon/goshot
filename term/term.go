// Package term renders captured terminal output — ANSI escapes included —
// to an image.
package term

import (
	"fmt"
	"image"
	"image/draw"
	"strings"

	"github.com/watzon/goshot/fonts"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// Renderer renders terminal output. Build one with New and chain the
// With* methods.
type Renderer struct {
	output []byte

	theme      string
	font       *fonts.Collection
	fontSize   float64
	lineHeight float64

	cols, rows  int
	autoSize    bool
	cellSpacing int
	pad         [4]int // left, right, top, bottom — in cells

	args   []string
	prompt func(command string) string
}

// New creates a renderer for raw terminal output with sensible defaults.
func New(output []byte) *Renderer {
	return &Renderer{
		output:     output,
		theme:      "dracula",
		font:       fonts.Fallback(),
		fontSize:   14,
		lineHeight: 1.25,
		cols:       120,
		rows:       40,
		pad:        [4]int{1, 1, 1, 1},
	}
}

// WithTheme selects a terminal color theme by name.
func (r *Renderer) WithTheme(name string) *Renderer { r.theme = name; return r }

// WithFont sets the font family.
func (r *Renderer) WithFont(f *fonts.Collection) *Renderer { r.font = f; return r }

// WithFontSize sets the font size in points.
func (r *Renderer) WithFontSize(size float64) *Renderer { r.fontSize = size; return r }

// WithLineHeight sets the line height multiplier.
func (r *Renderer) WithLineHeight(h float64) *Renderer { r.lineHeight = h; return r }

// WithSize sets the terminal dimensions in cells.
func (r *Renderer) WithSize(cols, rows int) *Renderer { r.cols, r.rows = cols, rows; return r }

// WithAutoSize grows the canvas to fit the content instead of using a
// fixed number of rows and columns.
func (r *Renderer) WithAutoSize() *Renderer { r.autoSize = true; return r }

// WithPadding sets the padding around the content, in cells.
func (r *Renderer) WithPadding(left, right, top, bottom int) *Renderer {
	r.pad = [4]int{left, right, top, bottom}
	return r
}

// WithCellSpacing adds extra horizontal space between cells, in pixels.
func (r *Renderer) WithCellSpacing(px int) *Renderer { r.cellSpacing = px; return r }

// WithCommand records the command line so a prompt can be rendered above
// the output (see WithPrompt).
func (r *Renderer) WithCommand(args ...string) *Renderer { r.args = args; return r }

// WithPrompt prepends a prompt line built by fn from the command set via
// WithCommand.
func (r *Renderer) WithPrompt(fn func(command string) string) *Renderer { r.prompt = fn; return r }

// Render implements the goshot content interface.
func (r *Renderer) Render() (image.Image, error) {
	theme := GetTheme(r.theme)
	if theme == nil {
		return nil, fmt.Errorf("term: unknown theme %q", r.theme)
	}

	input := r.output
	if r.prompt != nil && len(r.args) > 0 {
		promptLine := r.prompt(strings.Join(r.args, " ")) + "\n"
		input = append([]byte(promptLine), input...)
	}

	g := newGrid(r.cols, r.rows, r.pad, theme, r.autoSize)
	g.parse(input)

	cols, rows := g.width, g.height
	if r.autoSize {
		cols, rows = g.maxX+g.padR, g.maxY+g.padB
	}

	faces, err := r.faces()
	if err != nil {
		return nil, err
	}
	cellW := font.MeasureString(faces[0], "M").Round() + r.cellSpacing
	cellH := int(r.fontSize * r.lineHeight)

	img := image.NewRGBA(image.Rect(0, 0, cols*cellW, rows*cellH))
	draw.Draw(img, img.Bounds(), image.NewUniform(theme.Background), image.Point{}, draw.Src)

	// Backgrounds first, in full: a wide glyph overlaps the continuation
	// cells to its right, which must not repaint over it afterwards.
	for y := 0; y < min(rows, len(g.cells)); y++ {
		for x := 0; x < min(cols, len(g.cells[y])); x++ {
			if bg := g.cells[y][x].bg; bg != theme.Background {
				rect := image.Rect(x*cellW, y*cellH, (x+1)*cellW, (y+1)*cellH)
				draw.Draw(img, rect, image.NewUniform(bg), image.Point{}, draw.Src)
			}
		}
	}

	for y := 0; y < min(rows, len(g.cells)); y++ {
		for x := 0; x < min(cols, len(g.cells[y])); x++ {
			c := g.cells[y][x]
			if c.s == "" || c.s == " " {
				continue
			}
			if fonts.IsEmoji(c.s) && r.drawEmoji(img, g, x, y, cellW, cellH) {
				continue
			}
			d := &font.Drawer{
				Dst:  img,
				Src:  image.NewUniform(c.fg),
				Face: faces[faceIndex(c.attrs)],
				Dot: fixed.Point26_6{
					X: fixed.I(x * cellW),
					Y: fixed.I(y*cellH + int(r.fontSize)),
				},
			}
			d.DrawString(c.s)
		}
	}
	return img, nil
}

// drawEmoji draws the emoji cluster at (x, y) into the cells it spans,
// reporting whether it could be rendered as emoji.
func (r *Renderer) drawEmoji(img *image.RGBA, g *grid, x, y, cellW, cellH int) bool {
	e := fonts.Emoji()
	if e == nil {
		return false
	}
	span := 1
	for x+span < len(g.cells[y]) && g.cells[y][x+span].s == "" {
		span++
	}
	box := image.Rect(x*cellW, y*cellH, (x+span)*cellW, (y+1)*cellH)
	return e.Draw(img, g.cells[y][x].s, box, g.cells[y][x].fg)
}

// faces returns [regular, bold, italic, bold-italic].
func (r *Renderer) faces() ([4]font.Face, error) {
	var out [4]font.Face
	for i, s := range []fonts.Style{
		{Weight: fonts.Regular},
		{Weight: fonts.Bold},
		{Weight: fonts.Regular, Italic: true},
		{Weight: fonts.Bold, Italic: true},
	} {
		face, err := r.font.Face(r.fontSize, s)
		if err != nil {
			return out, err
		}
		out[i] = face
	}
	return out, nil
}

func faceIndex(a attributes) int {
	i := 0
	if a.bold {
		i |= 1
	}
	if a.italic {
		i |= 2
	}
	return i
}
