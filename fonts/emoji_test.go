package fonts

import (
	"image"
	"image/color"
	"testing"
)

func TestIsEmoji(t *testing.T) {
	cases := []struct {
		cluster string
		want    bool
	}{
		{"a", false},
		{"→", false},
		{"漢", false},
		{"😀", true},
		{"👍🏽", true},      // skin tone modifier
		{"👨‍👩‍👧‍👦", true}, // ZWJ family
		{"🇺🇸", true},      // regional indicator pair
		{"❤", false},      // default text presentation
		{"❤️", true},      // VS16 forces emoji
		{"1", false},
		{"1️⃣", true}, // keycap sequence
		{"☀", false},  // default text presentation
		{"⌚︎", false}, // VS15 forces text
		{"⚡", true},   // default emoji presentation
	}
	for _, c := range cases {
		if got := IsEmoji(c.cluster); got != c.want {
			t.Errorf("IsEmoji(%q) = %v, want %v", c.cluster, got, c.want)
		}
	}
}

func TestSplitEmoji(t *testing.T) {
	runs := SplitEmoji("hello")
	if len(runs) != 1 || runs[0].Emoji || runs[0].Text != "hello" {
		t.Errorf("SplitEmoji(hello) = %v", runs)
	}

	runs = SplitEmoji("a😀b👨‍👩‍👧‍👦")
	want := []Run{{"a", false}, {"😀", true}, {"b", false}, {"👨‍👩‍👧‍👦", true}}
	if len(runs) != len(want) {
		t.Fatalf("SplitEmoji = %v, want %v", runs, want)
	}
	for i := range want {
		if runs[i] != want[i] {
			t.Errorf("run %d = %v, want %v", i, runs[i], want[i])
		}
	}
}

func TestEmojiRender(t *testing.T) {
	e := Emoji()
	if e == nil {
		t.Skip("no system emoji font installed")
	}
	for _, cluster := range []string{"😀", "👍🏽", "🇺🇸", "👨‍👩‍👧‍👦", "❤️", "1️⃣"} {
		img, ok := e.Render(cluster, 32, color.Black)
		if !ok {
			t.Errorf("Render(%q) failed", cluster)
			continue
		}
		if img.Bounds().Dx() < 4 || img.Bounds().Dy() < 4 {
			t.Errorf("Render(%q) produced a tiny image: %v", cluster, img.Bounds())
		}
	}
}

// Apple Color Emoji's mid-size sbix strikes lack some ZWJ ligature
// glyphs; rendering must fall back to a strike that has them.
func TestEmojiRenderZWJAcrossSizes(t *testing.T) {
	e := Emoji()
	if e == nil {
		t.Skip("no system emoji font installed")
	}
	for size := 16.0; size <= 64; size++ {
		if _, ok := e.Render("👨‍👩‍👧‍👦", size, color.Black); !ok {
			t.Errorf("Render(family ZWJ) failed at size %v", size)
		}
	}
}

func TestEmojiRenderRejectsUnknown(t *testing.T) {
	e := Emoji()
	if e == nil {
		t.Skip("no system emoji font installed")
	}
	// A cluster no emoji font maps to a glyph.
	if _, ok := e.Render("￿", 32, color.Black); ok {
		t.Error("Render of a non-emoji rune unexpectedly succeeded")
	}
}

func TestEmojiDraw(t *testing.T) {
	e := Emoji()
	if e == nil {
		t.Skip("no system emoji font installed")
	}
	dst := image.NewRGBA(image.Rect(0, 0, 40, 20))
	if !e.Draw(dst, "😀", image.Rect(0, 0, 40, 20), color.Black) {
		t.Fatal("Draw failed")
	}
	painted := false
	for y := 0; y < 20 && !painted; y++ {
		for x := 0; x < 40; x++ {
			if _, _, _, a := dst.At(x, y).RGBA(); a > 0 {
				painted = true
				break
			}
		}
	}
	if !painted {
		t.Error("Draw painted no pixels")
	}
}
