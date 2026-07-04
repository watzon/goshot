package term

import (
	"testing"

	"github.com/watzon/goshot/fonts"
)

func TestGridStoresClustersWithWidth(t *testing.T) {
	g := newGrid(10, 2, [4]int{0, 0, 0, 0}, GetTheme("dracula"), false)
	g.parse([]byte("a😀b"))

	if got := g.cells[0][0].s; got != "a" {
		t.Errorf("cell 0 = %q, want %q", got, "a")
	}
	if got := g.cells[0][1].s; got != "😀" {
		t.Errorf("cell 1 = %q, want %q", got, "😀")
	}
	if got := g.cells[0][2].s; got != "" {
		t.Errorf("cell 2 = %q, want a continuation cell", got)
	}
	if got := g.cells[0][3].s; got != "b" {
		t.Errorf("cell 3 = %q, want %q", got, "b")
	}
}

func TestGridKeepsZWJSequenceInOneCell(t *testing.T) {
	g := newGrid(10, 2, [4]int{0, 0, 0, 0}, GetTheme("dracula"), false)
	g.parse([]byte("👨‍👩‍👧‍👦x"))

	if got := g.cells[0][0].s; got != "👨‍👩‍👧‍👦" {
		t.Errorf("cell 0 = %q, want the whole ZWJ cluster", got)
	}
	if got := g.cells[0][2].s; got != "x" {
		t.Errorf("cell 2 = %q, want %q", got, "x")
	}
}

// A wide emoji on a colored background must not have its right half
// repainted by the continuation cell's background.
func TestRenderWideGlyphOnColoredBackground(t *testing.T) {
	if fonts.Emoji() == nil {
		t.Skip("no system emoji font installed")
	}
	img, err := New([]byte("\x1b[41m😀\x1b[0m\n")).
		WithAutoSize().
		WithPadding(0, 0, 0, 0).
		Render()
	if err != nil {
		t.Fatal(err)
	}
	// The emoji spans two cells; look for non-background ink in the
	// right half (x in [w/4, w/2) of the two-cell span).
	theme := GetTheme("dracula")
	red := 0
	other := 0
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Dx() / 4; x < b.Dx()/2; x++ {
			c := img.At(x, y)
			switch {
			case c == theme.Background:
			case colorEq(c, 205, 49, 49) || colorEq(c, 255, 0, 0):
				red++
			default:
				other++
			}
		}
	}
	if other == 0 {
		t.Errorf("right half of the wide emoji contains no glyph pixels (red=%d)", red)
	}
}

func colorEq(c interface{ RGBA() (r, g, b, a uint32) }, r, g, b uint8) bool {
	cr, cg, cb, _ := c.RGBA()
	return uint8(cr>>8) == r && uint8(cg>>8) == g && uint8(cb>>8) == b
}

func TestRenderWithEmoji(t *testing.T) {
	img, err := New([]byte("deploy 🚀 done ✅\n")).WithAutoSize().Render()
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Empty() {
		t.Fatal("empty image")
	}
	if fonts.Emoji() == nil {
		t.Skip("no system emoji font; rendered with fallback glyphs")
	}
}
