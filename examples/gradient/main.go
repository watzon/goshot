// Every gradient type on the same snippet.
package main

import (
	"fmt"
	"image/color"
	"log"

	"github.com/watzon/goshot"
	"github.com/watzon/goshot/background"
	"github.com/watzon/goshot/chrome"
	"github.com/watzon/goshot/code"
)

const source = `def fib(n):
    a, b = 0, 1
    for _ in range(n):
        a, b = b, a + b
    return a
`

func main() {
	gradients := map[string]background.GradientType{
		"linear": background.LinearGradient,
		"radial": background.RadialGradient,
		"spiral": background.SpiralGradient,
		"star":   background.StarGradient,
	}
	for name, kind := range gradients {
		bg := background.Gradient(kind,
			background.Stop{Color: color.RGBA{65, 88, 208, 255}, Position: 0},
			background.Stop{Color: color.RGBA{200, 80, 192, 255}, Position: 0.5},
			background.Stop{Color: color.RGBA{255, 204, 112, 255}, Position: 1},
		).WithAngle(45).WithPadding(60).WithCornerRadius(10)

		err := goshot.New().
			WithContent(code.New(source).WithLanguage("python").WithTheme("catppuccin-mocha").WithMinWidth(400)).
			WithChrome(chrome.Gnome().WithTitle("fib.py").Dark()).
			WithBackground(bg).
			Save(fmt.Sprintf("gradient_%s.png", name))
		if err != nil {
			log.Fatal(err)
		}
	}
}
