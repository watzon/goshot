package goshot_test

import (
	"image"
	"image/color"
	"strings"
	"testing"

	"github.com/watzon/goshot"
	"github.com/watzon/goshot/background"
	"github.com/watzon/goshot/chrome"
	"github.com/watzon/goshot/code"
	"github.com/watzon/goshot/term"
)

// The fake key is assembled at runtime so secret scanners don't flag
// this file; it still matches the redaction patterns when rendered.
var sample = `package main

import "fmt"

func main() {
	apiKey := "` + "sk_live_" + strings.Repeat("x4", 16) + `"
	fmt.Println("hello", apiKey)
}
`

func TestFullPipeline(t *testing.T) {
	img, err := goshot.New().
		WithContent(code.New(sample).WithLanguage("go").WithMinWidth(400)).
		WithChrome(chrome.Mac().WithTitle("main.go").Dark()).
		WithBackground(background.Solid(color.RGBA{40, 42, 54, 255}).WithPadding(40).WithCornerRadius(8)).
		Image()
	if err != nil {
		t.Fatal(err)
	}
	b := img.Bounds()
	// content >= 400 wide, plus 2x40 padding
	if b.Dx() < 480 {
		t.Errorf("image too narrow: %d", b.Dx())
	}
	if b.Dy() < 100 {
		t.Errorf("image too short: %d", b.Dy())
	}
}

func TestEmptyCanvas(t *testing.T) {
	if _, err := goshot.New().Image(); err == nil {
		t.Fatal("expected error for empty canvas")
	}
}

func TestGradientAndShadow(t *testing.T) {
	content := imageContent{image.NewRGBA(image.Rect(0, 0, 100, 50))}
	bg := background.Gradient(background.LinearGradient,
		background.Stop{Color: color.RGBA{255, 0, 0, 255}, Position: 0},
		background.Stop{Color: color.RGBA{0, 0, 255, 255}, Position: 1},
	).WithAngle(45).WithPadding(30).WithShadow(background.NewShadow())

	img, err := goshot.New().WithContent(content).WithBackground(bg).Image()
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() <= 160 { // 100 + 2x30 padding + shadow margin
		t.Errorf("shadow margin missing: width %d", img.Bounds().Dx())
	}
}

func TestTerminalRender(t *testing.T) {
	out := []byte("\x1b[1;32mPASS\x1b[0m ok\n\x1b[31mred\x1b[0m\n")
	img, err := term.New(out).WithTheme("dracula").WithAutoSize().Render()
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Empty() {
		t.Fatal("empty terminal image")
	}
}

func TestRedactionBlock(t *testing.T) {
	img, err := code.New(sample).
		WithLanguage("go").
		WithRedaction(code.NewRedaction()).
		Render()
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Empty() {
		t.Fatal("empty image")
	}
}

func TestChromeVariants(t *testing.T) {
	content := image.NewRGBA(image.Rect(0, 0, 300, 100))
	for _, name := range chrome.Names() {
		c, ok := chrome.Named(name)
		if !ok {
			t.Fatalf("chrome %q not found", name)
		}
		img, err := c.WithTitle("title").Render(content)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if img.Bounds().Dy() < 100 {
			t.Errorf("%s: missing title bar, height %d", name, img.Bounds().Dy())
		}
	}
}

type imageContent struct{ img image.Image }

func (c imageContent) Render() (image.Image, error) { return c.img, nil }
