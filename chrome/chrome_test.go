package chrome

import (
	"image"
	"image/color"
	"image/draw"
	"testing"
)

func solidContent(c color.Color) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 200, 100))
	draw.Draw(img, img.Bounds(), image.NewUniform(c), image.Point{}, draw.Src)
	return img
}

// barPixel samples a title bar pixel clear of controls and title text.
func barPixel(t *testing.T, img image.Image, style *Style) color.RGBA {
	t.Helper()
	r, g, b, a := img.At(100, style.TitleBarHeight/2).RGBA()
	return color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)}
}

func TestTitleBarColorOverride(t *testing.T) {
	want := color.RGBA{10, 20, 30, 255}
	img, err := Mac().WithTitleBarColor(want).Render(solidContent(color.White))
	if err != nil {
		t.Fatal(err)
	}
	if got := barPixel(t, img, MacStyle); got != want {
		t.Errorf("title bar pixel = %v, want %v", got, want)
	}
}

func TestTitleBarMatchesContent(t *testing.T) {
	bg := color.RGBA{40, 42, 54, 255}
	img, err := Mac().WithTitleBarMatchingContent().Render(solidContent(bg))
	if err != nil {
		t.Fatal(err)
	}
	if got := barPixel(t, img, MacStyle); got != bg {
		t.Errorf("title bar pixel = %v, want content background %v", got, bg)
	}
}

func TestTitleBarColorBeatsMatchContent(t *testing.T) {
	want := color.RGBA{200, 0, 0, 255}
	img, err := Mac().
		WithTitleBarMatchingContent().
		WithTitleBarColor(want).
		Render(solidContent(color.White))
	if err != nil {
		t.Fatal(err)
	}
	if got := barPixel(t, img, MacStyle); got != want {
		t.Errorf("title bar pixel = %v, want explicit override %v", got, want)
	}
}

// A dark-variant chrome whose title bar matches light content must not
// paint its (light) title text invisibly on the light bar.
func TestMatchContentKeepsTitleReadable(t *testing.T) {
	img, err := Mac().Dark().
		WithTitle("main.go").
		WithTitleBarMatchingContent().
		Render(solidContent(color.White))
	if err != nil {
		t.Fatal(err)
	}
	dark := 0
	for y := 0; y < MacStyle.TitleBarHeight; y++ {
		for x := 80; x < 120; x++ { // centered title area
			r, g, b, _ := img.At(x, y).RGBA()
			if r < 0x8000 && g < 0x8000 && b < 0x8000 {
				dark++
			}
		}
	}
	if dark == 0 {
		t.Error("no dark title-text pixels on a light matched title bar; the title is invisible")
	}
}

func TestTitleTextColor(t *testing.T) {
	base, err := Mac().WithTitle("Hello").Render(solidContent(color.White))
	if err != nil {
		t.Fatal(err)
	}
	tinted, err := Mac().WithTitle("Hello").WithTitleTextColor(color.RGBA{255, 0, 0, 255}).Render(solidContent(color.White))
	if err != nil {
		t.Fatal(err)
	}
	same := true
	for y := 0; y < MacStyle.TitleBarHeight && same; y++ {
		for x := 0; x < 200; x++ {
			if base.At(x, y) != tinted.At(x, y) {
				same = false
				break
			}
		}
	}
	if same {
		t.Error("overriding the title text color did not change the rendered title bar")
	}
}
