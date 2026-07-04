// Package code renders syntax-highlighted source code to an image.
package code

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"strconv"
	"strings"

	"github.com/watzon/goshot/fonts"
	"golang.org/x/image/font"
)

// Range is an inclusive, 1-based range of source lines. An End of 0 or
// less means "through the last line".
type Range struct {
	Start, End int
}

// Renderer renders highlighted source code. Build one with New and chain
// the With* methods.
type Renderer struct {
	source     string
	language   string
	theme      string
	font       *fonts.Collection
	fontSize   float64
	lineHeight float64

	padTop, padRight, padBottom, padLeft int

	lineNumbers bool
	numberPad   int
	tabWidth    int
	minWidth    int
	maxWidth    int

	ranges     []Range
	highlights []Range

	redaction *Redaction
}

// New creates a renderer for the given source with sensible defaults.
func New(source string) *Renderer {
	return &Renderer{
		source:      source,
		theme:       "dracula",
		font:        fonts.Fallback(),
		fontSize:    14,
		lineHeight:  1.2,
		padTop:      10,
		padRight:    10,
		padBottom:   10,
		padLeft:     10,
		lineNumbers: true,
		numberPad:   10,
		tabWidth:    4,
	}
}

// WithLanguage forces a language instead of auto-detecting one.
func (r *Renderer) WithLanguage(lang string) *Renderer { r.language = lang; return r }

// WithTheme selects a chroma syntax theme by name.
func (r *Renderer) WithTheme(theme string) *Renderer { r.theme = theme; return r }

// WithFont sets the font family used to render the code.
func (r *Renderer) WithFont(f *fonts.Collection) *Renderer { r.font = f; return r }

// WithFontSize sets the font size in points.
func (r *Renderer) WithFontSize(size float64) *Renderer { r.fontSize = size; return r }

// WithLineHeight sets the line height multiplier.
func (r *Renderer) WithLineHeight(h float64) *Renderer { r.lineHeight = h; return r }

// WithPadding sets the padding around the code in pixels.
func (r *Renderer) WithPadding(left, right, top, bottom int) *Renderer {
	r.padLeft, r.padRight, r.padTop, r.padBottom = left, right, top, bottom
	return r
}

// WithLineNumbers shows or hides the line number gutter.
func (r *Renderer) WithLineNumbers(show bool) *Renderer { r.lineNumbers = show; return r }

// WithLineNumberPadding sets the gap between line numbers and code.
func (r *Renderer) WithLineNumberPadding(px int) *Renderer { r.numberPad = px; return r }

// WithTabWidth sets how many spaces a tab expands to.
func (r *Renderer) WithTabWidth(w int) *Renderer { r.tabWidth = w; return r }

// WithMinWidth sets a minimum image width in pixels.
func (r *Renderer) WithMinWidth(w int) *Renderer { r.minWidth = w; return r }

// WithMaxWidth sets a maximum image width; long lines wrap to fit.
func (r *Renderer) WithMaxWidth(w int) *Renderer { r.maxWidth = w; return r }

// WithLineRange limits rendering to a range of lines. Multiple ranges are
// separated by ellipsis lines.
func (r *Renderer) WithLineRange(start, end int) *Renderer {
	r.ranges = append(r.ranges, Range{start, end})
	return r
}

// WithHighlightRange highlights a range of source lines.
func (r *Renderer) WithHighlightRange(start, end int) *Renderer {
	r.highlights = append(r.highlights, Range{start, end})
	return r
}

// WithRedaction enables redaction of sensitive values (see Redaction).
func (r *Renderer) WithRedaction(red *Redaction) *Renderer { r.redaction = red; return r }

// visLine is a line selected for display.
type visLine struct {
	tokens      []token
	number      int // 1-based source line number
	ellipsis    bool
	highlighted bool
	redacted    []span // redacted column ranges
}

// row is one wrapped, drawable line of tokens.
type row struct {
	line     *visLine
	tokens   []token
	first    bool
	startCol int
}

// Render implements the goshot content interface.
func (r *Renderer) Render() (image.Image, error) {
	lines, pal, err := highlight(r.source, r.language, r.theme, r.tabWidth)
	if err != nil {
		return nil, err
	}

	faces, err := r.faces()
	if err != nil {
		return nil, err
	}

	visible, err := r.selectLines(lines, pal)
	if err != nil {
		return nil, err
	}
	r.markRedactions(visible)

	// Vertical metrics.
	metrics := faces.regular.Metrics()
	lineHeight := int(float64(metrics.Height.Round()) * r.lineHeight)
	ascent := metrics.Ascent.Round()

	// Gutter for line numbers.
	gutter := 0
	if r.lineNumbers {
		maxNum := 1
		for i := range visible {
			maxNum = max(maxNum, visible[i].number)
		}
		digits := len(strconv.Itoa(maxNum))
		gutter = font.MeasureString(faces.regular, strings.Repeat("9", digits)).Round() + r.numberPad
	}

	// Wrap tokens into rows and find the widest.
	wrapWidth := 0
	if r.maxWidth > 0 {
		wrapWidth = r.maxWidth - r.padLeft - r.padRight - gutter
	}
	var rows []row
	widest := 0
	for i := range visible {
		for _, rw := range wrapLine(&visible[i], faces, wrapWidth) {
			widest = max(widest, rowWidth(rw.tokens, faces))
			rows = append(rows, rw)
		}
	}

	codeWidth := widest + r.padLeft + r.padRight
	if r.minWidth > 0 {
		codeWidth = max(codeWidth, r.minWidth)
	}
	if r.maxWidth > 0 {
		codeWidth = min(codeWidth, r.maxWidth)
	}
	width := codeWidth + gutter
	height := lineHeight*len(rows) + r.padTop + r.padBottom

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), image.NewUniform(pal.background), image.Point{}, draw.Src)

	// Draw rows.
	blurStyle := r.redaction != nil && r.redaction.Enabled && r.redaction.Style == RedactBlur
	var blurRects []image.Rectangle

	y := r.padTop
	for _, rw := range rows {
		if rw.line.highlighted {
			bar := image.Rect(r.padLeft+gutter, y, width, y+lineHeight)
			if !r.lineNumbers {
				bar = image.Rect(0, y, width, y+lineHeight)
			}
			draw.Draw(img, bar, image.NewUniform(pal.highlight), image.Point{}, draw.Over)
		}

		if r.lineNumbers && rw.first {
			num := strconv.Itoa(rw.line.number)
			w := font.MeasureString(faces.regular, num).Round()
			drawString(img, faces.regular, num, r.padLeft+gutter-r.numberPad-w, y+ascent, pal.gutter, false)
		}

		x := r.padLeft + gutter
		col := rw.startCol
		for _, tok := range rw.tokens {
			face := faces.pick(tok)
			if len(rw.line.redacted) == 0 {
				x += drawText(img, face, tok.text, x, y+ascent, tok.color, tok.underline)
				col += len(tok.text)
				continue
			}
			// Unit-by-unit so redacted spans can be replaced.
			for _, u := range units(tok.text) {
				w := measureText(face, u)
				switch {
				case !covered(col, rw.line.redacted):
					drawText(img, face, u, x, y+ascent, tok.color, tok.underline)
				case blurStyle:
					drawText(img, face, u, x, y+ascent, tok.color, tok.underline)
					blurRects = append(blurRects, image.Rect(x, y, x+w, y+lineHeight))
				default:
					drawString(img, face, blocksFor(face, w), x, y+ascent, tok.color, false)
				}
				x += w
				col += len(u)
			}
		}
		y += lineHeight
	}

	// Apply blur redactions and manual areas.
	if r.redaction != nil && r.redaction.Enabled {
		for _, rect := range mergeRects(blurRects) {
			blurArea(img, rect, r.redaction.BlurRadius)
		}
		for _, a := range r.redaction.Areas {
			rect := image.Rect(a.X, a.Y, a.X+a.Width, a.Y+a.Height)
			if r.redaction.Style == RedactBlur {
				blurArea(img, rect, r.redaction.BlurRadius)
			} else {
				draw.Draw(img, rect.Intersect(img.Bounds()), image.NewUniform(color.Black), image.Point{}, draw.Src)
			}
		}
	}

	return img, nil
}

// selectLines applies line ranges, inserting ellipsis markers between and
// around them, and flags highlighted lines.
func (r *Renderer) selectLines(lines []line, pal palette) ([]visLine, error) {
	ellipsis := func(number int) visLine {
		return visLine{
			tokens:   []token{{text: "...", color: pal.comment, italic: true}},
			number:   number,
			ellipsis: true,
		}
	}

	var visible []visLine
	if len(r.ranges) == 0 {
		for i, l := range lines {
			visible = append(visible, visLine{tokens: l.tokens, number: i + 1})
		}
	} else {
		ranges := make([]Range, len(r.ranges))
		for i, rg := range r.ranges {
			if rg.Start < 1 {
				rg.Start = 1
			}
			if rg.End <= 0 || rg.End > len(lines) {
				rg.End = len(lines)
			}
			if rg.Start > len(lines) {
				return nil, fmt.Errorf("code: line range start %d out of bounds (%d lines)", rg.Start, len(lines))
			}
			ranges[i] = rg
		}
		if ranges[0].Start > 1 {
			visible = append(visible, ellipsis(ranges[0].Start-1))
		}
		for i, rg := range ranges {
			for n := rg.Start; n <= rg.End; n++ {
				visible = append(visible, visLine{tokens: lines[n-1].tokens, number: n})
			}
			if i+1 < len(ranges) && rg.End+1 < ranges[i+1].Start {
				visible = append(visible, ellipsis(rg.End+1))
			}
		}
		if last := ranges[len(ranges)-1].End; last < len(lines) {
			visible = append(visible, ellipsis(last+1))
		}
	}

	for i := range visible {
		if visible[i].ellipsis {
			continue
		}
		for _, hr := range r.highlights {
			end := hr.End
			if end <= 0 {
				end = len(lines)
			}
			if visible[i].number >= hr.Start && visible[i].number <= end {
				visible[i].highlighted = true
			}
		}
	}
	return visible, nil
}

// markRedactions finds sensitive spans and attaches them to their lines.
func (r *Renderer) markRedactions(visible []visLine) {
	if r.redaction == nil || !r.redaction.Enabled {
		return
	}
	var full strings.Builder
	starts := make([]int, len(visible))
	for i := range visible {
		starts[i] = full.Len()
		full.WriteString(line{tokens: visible[i].tokens}.text())
		full.WriteByte('\n')
	}
	for _, sp := range r.redaction.find(full.String()) {
		for i := range visible {
			lineStart := starts[i]
			lineEnd := lineStart + len(line{tokens: visible[i].tokens}.text())
			if sp.start < lineEnd && sp.end > lineStart {
				visible[i].redacted = append(visible[i].redacted, span{
					start: max(0, sp.start-lineStart),
					end:   min(lineEnd, sp.end) - lineStart,
				})
			}
		}
	}
}

// --- fonts & drawing ---------------------------------------------------

type faceSet struct {
	regular, bold, italic, boldItalic font.Face
}

func (f faceSet) pick(t token) font.Face {
	switch {
	case t.bold && t.italic:
		return f.boldItalic
	case t.bold:
		return f.bold
	case t.italic:
		return f.italic
	}
	return f.regular
}

func (r *Renderer) faces() (faceSet, error) {
	var fs faceSet
	var err error
	load := func(dst *font.Face, s fonts.Style) {
		if err == nil {
			*dst, err = r.font.Face(r.fontSize, s)
		}
	}
	load(&fs.regular, fonts.Style{Weight: fonts.Regular})
	load(&fs.bold, fonts.Style{Weight: fonts.Bold})
	load(&fs.italic, fonts.Style{Weight: fonts.Regular, Italic: true})
	load(&fs.boldItalic, fonts.Style{Weight: fonts.Bold, Italic: true})
	return fs, err
}

func drawString(img *image.RGBA, face font.Face, s string, x, y int, col color.Color, underline bool) {
	d := &font.Drawer{Dst: img, Src: image.NewUniform(col), Face: face}
	d.Dot.X, d.Dot.Y = fixedI(x), fixedI(y)
	d.DrawString(s)
	if underline {
		w := font.MeasureString(face, s).Round()
		uy := y + face.Metrics().Descent.Round()/2
		draw.Draw(img, image.Rect(x, uy, x+w, uy+1), image.NewUniform(col), image.Point{}, draw.Over)
	}
}

// drawText draws s at the baseline y, routing emoji clusters to the
// system emoji font, and returns the width drawn.
func drawText(img *image.RGBA, face font.Face, s string, x, y int, col color.Color, underline bool) int {
	start := x
	for _, run := range fonts.SplitEmoji(s) {
		if run.Emoji {
			x += drawEmoji(img, face, run.Text, x, y, col)
			continue
		}
		drawString(img, face, run.Text, x, y, col, false)
		x += font.MeasureString(face, run.Text).Round()
	}
	if underline {
		uy := y + face.Metrics().Descent.Round()/2
		draw.Draw(img, image.Rect(start, uy, x, uy+1), image.NewUniform(col), image.Point{}, draw.Over)
	}
	return x - start
}

// drawEmoji draws one emoji cluster centered in a two-space em box on
// the baseline, falling back to the code font when the cluster cannot be
// rendered as emoji.
func drawEmoji(img *image.RGBA, face font.Face, cluster string, x, baseline int, col color.Color) int {
	if e := fonts.Emoji(); e != nil {
		m := face.Metrics()
		adv := emojiAdvance(face)
		box := image.Rect(x, baseline-m.Ascent.Ceil(), x+adv, baseline+m.Descent.Ceil())
		if e.Draw(img, cluster, box, col) {
			return adv
		}
	}
	drawString(img, face, cluster, x, baseline, col, false)
	return font.MeasureString(face, cluster).Round()
}

// measureText is the measuring counterpart of drawText.
func measureText(face font.Face, s string) int {
	w := 0
	for _, run := range fonts.SplitEmoji(s) {
		if run.Emoji && emojiRenderable(face, run.Text) {
			w += emojiAdvance(face)
		} else {
			w += font.MeasureString(face, run.Text).Round()
		}
	}
	return w
}

// emojiAdvance is the horizontal space an emoji occupies: two cells, as
// in terminals and code editors.
func emojiAdvance(face font.Face) int {
	return font.MeasureString(face, "  ").Round()
}

// blocksFor returns enough full-block characters to cover width pixels,
// so redacting a wide unit (an emoji) leaves no gap.
func blocksFor(face font.Face, width int) string {
	bw := font.MeasureString(face, "█").Round()
	n := 1
	if bw > 0 {
		n = max(1, (width+bw/2)/bw)
	}
	return strings.Repeat("█", n)
}

func emojiRenderable(face font.Face, cluster string) bool {
	e := fonts.Emoji()
	if e == nil {
		return false
	}
	m := face.Metrics()
	_, ok := e.Render(cluster, float64(m.Ascent.Ceil()+m.Descent.Ceil()), color.Black)
	return ok
}

// units splits a string into indivisible drawing units: single runes of
// plain text and whole emoji clusters.
func units(s string) []string {
	var out []string
	for _, run := range fonts.SplitEmoji(s) {
		if run.Emoji {
			out = append(out, run.Text)
			continue
		}
		for _, ch := range run.Text {
			out = append(out, string(ch))
		}
	}
	return out
}

// --- wrapping ----------------------------------------------------------

func rowWidth(tokens []token, faces faceSet) int {
	w := 0
	for _, t := range tokens {
		w += measureText(faces.pick(t), t.text)
	}
	return w
}

// wrapLine splits a line into rows no wider than maxWidth (0 = no limit).
func wrapLine(l *visLine, faces faceSet, maxWidth int) []row {
	if maxWidth <= 0 {
		return []row{{line: l, tokens: l.tokens, first: true}}
	}

	var rows []row
	var current []token
	width, col, startCol := 0, 0, 0

	flush := func() {
		rows = append(rows, row{line: l, tokens: current, first: len(rows) == 0, startCol: startCol})
		current, width, startCol = nil, 0, col
	}

	for _, tok := range l.tokens {
		face := faces.pick(tok)
		w := measureText(face, tok.text)
		if width+w <= maxWidth {
			current = append(current, tok)
			width += w
			col += len(tok.text)
			continue
		}
		// Split the token unit by unit across rows.
		part, partW := "", 0
		for _, u := range units(tok.text) {
			cw := measureText(face, u)
			if width+partW+cw > maxWidth && (len(part) > 0 || len(current) > 0) {
				if part != "" {
					sub := tok
					sub.text = part
					current = append(current, sub)
					col += len(part)
				}
				flush()
				part, partW = "", 0
			}
			part += u
			partW += cw
		}
		if part != "" {
			sub := tok
			sub.text = part
			current = append(current, sub)
			width += partW
			col += len(part)
		}
	}
	if len(current) > 0 || len(rows) == 0 {
		flush()
	}
	return rows
}
