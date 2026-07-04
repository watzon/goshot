// A minimal goshot pipeline: highlighted code in a macOS window on a
// solid background.
package main

import (
	"image/color"
	"log"

	"github.com/watzon/goshot"
	"github.com/watzon/goshot/background"
	"github.com/watzon/goshot/chrome"
	"github.com/watzon/goshot/code"
)

const source = `package main

import "fmt"

func main() {
	fmt.Println("Hello, goshot!")
}
`

func main() {
	err := goshot.New().
		WithContent(code.New(source).WithLanguage("go").WithTheme("dracula")).
		WithChrome(chrome.Mac().WithTitle("hello.go").Dark()).
		WithBackground(background.Solid(color.RGBA{40, 42, 54, 255}).
			WithPadding(60).
			WithCornerRadius(10).
			WithShadow(background.NewShadow().WithBlur(20).WithOffset(0, 8))).
		Save("basic.png")
	if err != nil {
		log.Fatal(err)
	}
}
