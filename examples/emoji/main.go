// Emoji render automatically through the system emoji font, and the
// title bar can blend into the content with WithTitleBarMatchingContent.
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
	greeting := "Hello, world! 👋🏽"
	flags := []string{"🇺🇸", "🇯🇵", "🇩🇪"}
	fmt.Println(greeting, "🚀", flags)
	fmt.Println("family: 👨‍👩‍👧‍👦 done: ✅ love: ❤️")
}
`

func main() {
	err := goshot.New().
		WithContent(code.New(source).WithLanguage("go").WithTheme("dracula")).
		WithChrome(chrome.Mac().WithTitle("emoji.go 🎉").Dark().WithTitleBarMatchingContent()).
		WithBackground(background.Solid(color.RGBA{40, 42, 54, 255}).
			WithPadding(60).
			WithCornerRadius(10).
			WithShadow(background.NewShadow().WithBlur(20).WithOffset(0, 8))).
		Save("emoji.png")
	if err != nil {
		log.Fatal(err)
	}
}
