package fonts

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg" // decode emoji bitmap glyphs
	_ "image/png"  // decode emoji bitmap glyphs
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"

	"github.com/fogleman/gg"
	"github.com/go-text/typesetting/di"
	gtfont "github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/font/opentype"
	"github.com/go-text/typesetting/language"
	"github.com/go-text/typesetting/shaping"
	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/math/fixed"
	_ "golang.org/x/image/tiff" // decode emoji bitmap glyphs
)

// EmojiFont shapes and rasterizes emoji clusters — 😀, 👍🏽, 🇺🇸, 👨‍👩‍👧‍👦 —
// using an emoji font. Color bitmap fonts (Apple Color Emoji, Noto Color
// Emoji) render in full color; outline-only fonts render monochrome.
type EmojiFont struct {
	mu     sync.Mutex
	face   *gtfont.Face
	shaper shaping.HarfbuzzShaper
	cache  map[emojiKey]image.Image
}

type emojiKey struct {
	cluster string
	size    int // pixel size, quantized to quarter pixels
	fg      color.RGBA
}

var (
	emojiOnce   sync.Once
	systemEmoji *EmojiFont
)

// Emoji returns an emoji font found on the system, or nil when none is
// installed. The result is cached for the life of the process.
func Emoji() *EmojiFont {
	emojiOnce.Do(func() {
		for _, path := range emojiFontPaths() {
			if e, err := LoadEmojiFont(path); err == nil {
				systemEmoji = e
				return
			}
		}
	})
	return systemEmoji
}

// LoadEmojiFont loads an emoji font from a .ttf, .otf, or .ttc file.
func LoadEmojiFont(path string) (*EmojiFont, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	// The face may read from the file lazily, so it stays open.
	var face *gtfont.Face
	if strings.EqualFold(filepath.Ext(path), ".ttc") {
		faces, err := gtfont.ParseTTC(f)
		if err != nil {
			f.Close()
			return nil, err
		}
		face = faces[0]
	} else {
		if face, err = gtfont.ParseTTF(f); err != nil {
			f.Close()
			return nil, err
		}
	}
	if _, ok := face.NominalGlyph('😀'); !ok {
		f.Close()
		return nil, fmt.Errorf("fonts: %s has no emoji glyphs", path)
	}
	return &EmojiFont{face: face, cache: map[emojiKey]image.Image{}}, nil
}

// wellKnownEmojiFonts lists the standard emoji font of each platform.
var wellKnownEmojiFonts = map[string][]string{
	"darwin": {"/System/Library/Fonts/Apple Color Emoji.ttc"},
	"linux": {
		"/usr/share/fonts/truetype/noto/NotoColorEmoji.ttf",
		"/usr/share/fonts/noto/NotoColorEmoji.ttf",
		"/usr/share/fonts/google-noto-emoji/NotoColorEmoji.ttf",
		"/usr/share/fonts/noto-color-emoji/NotoColorEmoji.ttf",
	},
	"windows": {`C:\Windows\Fonts\seguiemj.ttf`},
}

func emojiFontPaths() []string {
	paths := append([]string{}, wellKnownEmojiFonts[runtime.GOOS]...)
	for _, dir := range expandDirs(systemDirs[runtime.GOOS]) {
		filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			base := strings.ToLower(filepath.Base(path))
			if !strings.Contains(base, "emoji") {
				return nil
			}
			switch filepath.Ext(base) {
			case ".ttf", ".otf", ".ttc":
				paths = append(paths, path)
			}
			return nil
		})
	}
	return paths
}

// Draw renders cluster scaled to fit within box — aspect preserved,
// centered — onto dst. Monochrome glyphs are painted with fg. It reports
// whether the font could represent the cluster.
func (e *EmojiFont) Draw(dst draw.Image, cluster string, box image.Rectangle, fg color.Color) bool {
	src, ok := e.Render(cluster, float64(box.Dy()), fg)
	if !ok {
		return false
	}
	sw, sh := src.Bounds().Dx(), src.Bounds().Dy()
	scale := min(float64(box.Dx())/float64(sw), float64(box.Dy())/float64(sh))
	w, h := int(float64(sw)*scale+0.5), int(float64(sh)*scale+0.5)
	x := box.Min.X + (box.Dx()-w)/2
	y := box.Min.Y + (box.Dy()-h)/2
	xdraw.BiLinear.Scale(dst, image.Rect(x, y, x+w, y+h), src, src.Bounds(), draw.Over, nil)
	return true
}

// Render rasterizes cluster at the given pixel em-size and returns the
// result cropped to its content. Monochrome glyphs are painted with fg.
// It reports whether the font could represent the cluster; results are
// cached.
func (e *EmojiFont) Render(cluster string, size float64, fg color.Color) (image.Image, bool) {
	if size <= 0 || cluster == "" {
		return nil, false
	}
	e.mu.Lock()
	defer e.mu.Unlock()

	r, g, b, a := fg.RGBA()
	key := emojiKey{cluster, int(size * 4), color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)}}
	if img, hit := e.cache[key]; hit {
		return img, img != nil
	}
	if len(e.cache) >= emojiCacheMax {
		e.cache = map[emojiKey]image.Image{}
	}
	img := e.render(cluster, size, fg)
	e.cache[key] = img
	return img, img != nil
}

// emojiCacheMax bounds the render cache; long-running hosts rendering
// many distinct clusters, sizes, and colors reset it rather than grow
// without limit.
const emojiCacheMax = 1024

func (e *EmojiFont) render(cluster string, size float64, fg color.Color) image.Image {
	runes := []rune(cluster)
	out := e.shaper.Shape(shaping.Input{
		Text:      runes,
		RunStart:  0,
		RunEnd:    len(runes),
		Face:      e.face,
		Size:      fixed.Int26_6(size * 64),
		Direction: di.DirectionLTR,
		Script:    language.LookupScript(runes[0]),
		Language:  language.DefaultLanguage(),
	})
	if len(out.Glyphs) == 0 {
		return nil
	}
	for _, g := range out.Glyphs {
		if g.GlyphID == 0 {
			return nil // .notdef: the font cannot draw this cluster
		}
	}

	// Ppem drives which bitmap strike sbix/CBDT fonts hand out, and not
	// every strike contains every glyph (Apple Color Emoji lacks some ZWJ
	// ligatures in mid-size strikes). Try the requested size first, then
	// the font's actual strike sizes until one draws.
	for _, ppem := range e.candidatePpems(size) {
		e.face.SetPpem(ppem, ppem)
		if img := e.rasterize(out, size, fg); img != nil {
			return img
		}
	}
	return nil
}

// candidatePpems returns the pixel sizes worth asking the font for: the
// requested size, then nearby bitmap strikes (preferring the smallest
// strike at least as large as the request, which downscales cleanly).
func (e *EmojiFont) candidatePpems(size float64) []uint16 {
	want := uint16(size + 0.5)
	above, below := []uint16{}, []uint16{}
	for _, s := range e.face.BitmapSizes() {
		switch {
		case s.XPpem == want:
		case s.XPpem > want:
			above = append(above, s.XPpem)
		default:
			below = append(below, s.XPpem)
		}
	}
	slices.Sort(above)
	slices.Sort(below)
	slices.Reverse(below)
	return append(append([]uint16{want}, above...), below...)
}

// rasterize draws the shaped cluster at the current ppem, returning nil
// when the font produces no visible glyph images at this strike.
func (e *EmojiFont) rasterize(out shaping.Output, size float64, fg color.Color) image.Image {
	width := out.Advance.Ceil()
	ascent := out.LineBounds.Ascent.Ceil()
	height := (out.LineBounds.Ascent - out.LineBounds.Descent).Ceil()
	if width <= 0 || height <= 0 {
		return nil
	}

	canvas := image.NewRGBA(image.Rect(0, 0, width, height))
	pen := fixed.Int26_6(0)
	for _, g := range out.Glyphs {
		x := f26(pen + g.XOffset)
		y := float64(ascent) - f26(g.YOffset) // baseline, in image coordinates
		if !e.drawGlyph(canvas, g, size, x, y, fg) {
			return nil
		}
		pen += g.Advance
	}
	// An all-transparent canvas means the strike had no real image for
	// some glyph (fonts fall back to empty placeholder outlines).
	return cropToContent(canvas)
}

// drawGlyph paints one glyph and reports whether the font provides a
// drawable form for it.
func (e *EmojiFont) drawGlyph(canvas *image.RGBA, g shaping.Glyph, size, x, y float64, fg color.Color) bool {
	scale := size / float64(e.face.Upem())
	switch data := e.face.GlyphData(g.GlyphID).(type) {
	case gtfont.GlyphBitmap:
		e.drawBitmap(canvas, g, data, x, y)
	case gtfont.GlyphOutline:
		drawOutline(canvas, data, scale, x, y, fg)
	case gtfont.GlyphSVG:
		// SVG documents are not parsed; use the spec-mandated fallback outline.
		drawOutline(canvas, data.Outline, scale, x, y, fg)
	default:
		// COLR paint graphs are not parsed, and their layer outlines are
		// not reachable through the public API.
		return false
	}
	return true
}

func (e *EmojiFont) drawBitmap(canvas *image.RGBA, g shaping.Glyph, bm gtfont.GlyphBitmap, x, y float64) {
	// The glyph box in image coordinates; g.Height is negative.
	top := y - f26(g.YBearing)
	left := x + f26(g.XBearing)
	rect := image.Rect(int(left+0.5), int(top+0.5), int(left+f26(g.Width)+0.5), int(top-f26(g.Height)+0.5))

	switch bm.Format {
	case gtfont.PNG, gtfont.JPG, gtfont.TIFF:
		pix, _, err := image.Decode(bytes.NewReader(bm.Data))
		if err != nil {
			return
		}
		xdraw.BiLinear.Scale(canvas, rect, pix, pix.Bounds(), draw.Over, nil)
	case gtfont.BlackAndWhite:
		mono := image.NewPaletted(image.Rect(0, 0, bm.Width, bm.Height), color.Palette{color.Transparent, color.Black})
		for i := range mono.Pix {
			mono.Pix[i] = (bm.Data[i/8] >> (7 - i%8)) & 1
		}
		xdraw.NearestNeighbor.Scale(canvas, rect, mono, mono.Bounds(), draw.Over, nil)
	}
}

// drawOutline fills a glyph outline. Outline coordinates are in font
// units with Y up; scale converts them to pixels.
func drawOutline(canvas *image.RGBA, outline gtfont.GlyphOutline, scale, x, y float64, fg color.Color) {
	dc := gg.NewContextForRGBA(canvas)
	px := func(p opentype.SegmentPoint) (float64, float64) {
		return x + float64(p.X)*scale, y - float64(p.Y)*scale
	}
	for _, seg := range outline.Segments {
		switch seg.Op {
		case opentype.SegmentOpMoveTo:
			x0, y0 := px(seg.Args[0])
			dc.MoveTo(x0, y0)
		case opentype.SegmentOpLineTo:
			x0, y0 := px(seg.Args[0])
			dc.LineTo(x0, y0)
		case opentype.SegmentOpQuadTo:
			x0, y0 := px(seg.Args[0])
			x1, y1 := px(seg.Args[1])
			dc.QuadraticTo(x0, y0, x1, y1)
		case opentype.SegmentOpCubeTo:
			x0, y0 := px(seg.Args[0])
			x1, y1 := px(seg.Args[1])
			x2, y2 := px(seg.Args[2])
			dc.CubicTo(x0, y0, x1, y1, x2, y2)
		}
	}
	dc.SetColor(fg)
	dc.Fill()
}

// cropToContent trims fully transparent borders, returning nil for an
// empty image.
func cropToContent(img *image.RGBA) image.Image {
	b := img.Bounds()
	minX, minY, maxX, maxY := b.Max.X, b.Max.Y, b.Min.X, b.Min.Y
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if img.RGBAAt(x, y).A == 0 {
				continue
			}
			minX, minY = min(minX, x), min(minY, y)
			maxX, maxY = max(maxX, x+1), max(maxY, y+1)
		}
	}
	if minX >= maxX || minY >= maxY {
		return nil
	}
	return img.SubImage(image.Rect(minX, minY, maxX, maxY))
}

func f26(v fixed.Int26_6) float64 { return float64(v) / 64 }
