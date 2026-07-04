// Rendering captured terminal output, ANSI escapes and all.
package main

import (
	"image/color"
	"log"

	"github.com/watzon/goshot"
	"github.com/watzon/goshot/background"
	"github.com/watzon/goshot/chrome"
	"github.com/watzon/goshot/term"
)

var output = []byte(
	"\x1b[1;32m✓\x1b[0m compiled in \x1b[1m1.2s\x1b[0m\n" +
		"\x1b[1;32m✓\x1b[0m 42 tests passed\n" +
		"\x1b[1;31m✗\x1b[0m 1 test \x1b[4mskipped\x1b[0m\n" +
		"\n\x1b[38;5;213mdone.\x1b[0m\n")

func main() {
	content := term.New(output).
		WithTheme("catppuccin-mocha").
		WithAutoSize().
		WithCommand("go", "test", "./...").
		WithPrompt(func(cmd string) string { return "\x1b[1;35m❯\x1b[0m " + cmd })

	err := goshot.New().
		WithContent(content).
		WithChrome(chrome.Mac().WithTitle("tests").Dark()).
		WithBackground(background.Solid(color.RGBA{30, 30, 46, 255}).WithPadding(50).WithCornerRadius(10)).
		Save("terminal.png")
	if err != nil {
		log.Fatal(err)
	}
}
