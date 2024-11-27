package main

import (
	"image/color"
	"log"

	"github.com/watzon/goshot/pkg/background"
	"github.com/watzon/goshot/pkg/chrome"
	"github.com/watzon/goshot/pkg/content/code"
	"github.com/watzon/goshot/pkg/render"
)

func main() {
	// Simple example showing basic usage with a solid color background
	input := `package main

import (
    "fmt"
    "strings"
)

// This is a very long comment that should wrap to multiple lines when the width is constrained. We'll make it even longer to ensure it wraps at least once or twice when rendered.
func main() {
    message := "Hello, " + strings.Repeat("World! ", 10)
    fmt.Println(message)
}`

	canvas := render.NewCanvas().
		WithChrome(chrome.NewMacChrome(
			chrome.MacStyleSequoia,
			chrome.WithTitle("Basic Example"))).
		WithBackground(
			background.NewColorBackground().
				WithColor(color.RGBA{R: 20, G: 30, B: 40, A: 255}).
				WithPadding(40),
		).
		WithContent(code.DefaultCodeRenderer(input).
			WithLanguage("go").
			WithTheme("dracula").
			WithTabWidth(4).
			WithShowLineNumbers(true).
			WithMaxWidth(400),
		)

	err := canvas.SaveAsPNG("basic.png")
	if err != nil {
		log.Fatal(err)
	}
}
