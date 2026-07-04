package chrome

import (
	"image/color"

	"github.com/fogleman/gg"
)

// Mac creates a macOS-style chrome with traffic-light controls.
func Mac() *Chrome { return New(MacStyle) }

// Windows creates a Windows 11-style chrome.
func Windows() *Chrome { return New(WindowsStyle) }

// Gnome creates a GNOME Adwaita-style chrome.
func Gnome() *Chrome { return New(GnomeStyle) }

// Breeze creates a KDE Breeze-style chrome.
func Breeze() *Chrome { return New(BreezeStyle) }

// Blank creates a chrome with no title bar or controls, just a rounded
// window surface.
func Blank() *Chrome { return New(BlankStyle).WithTitleBar(false) }

var registry = map[string]*Style{
	"mac":     MacStyle,
	"windows": WindowsStyle,
	"gnome":   GnomeStyle,
	"breeze":  BreezeStyle,
	"blank":   BlankStyle,
}

func rgb(r, g, b uint8) color.Color { return color.RGBA{r, g, b, 255} }

// MacStyle mimics modern macOS (Sequoia) windows.
var MacStyle = &Style{
	Name:           "mac",
	TitleFont:      "Inter",
	TitleFontSize:  13,
	TitleBarHeight: 30,
	CornerRadius:   9,
	Light:          Palette{TitleBar: rgb(236, 236, 236), TitleText: rgb(76, 76, 76), Controls: rgb(76, 76, 76)},
	Dark:           Palette{TitleBar: rgb(36, 36, 36), TitleText: rgb(255, 255, 255), Controls: rgb(255, 255, 255)},
	Controls: func(dc *gg.Context, width, bar int, p Palette) {
		const size, pad, gap = 13.0, 10.0, 8.0
		y := float64(bar) / 2
		for i, c := range []color.Color{rgb(255, 95, 87), rgb(255, 189, 46), rgb(39, 201, 63)} {
			dc.SetColor(c)
			dc.DrawCircle(pad+size/2+float64(i)*(size+gap), y, size/2)
			dc.Fill()
		}
	},
}

// WindowsStyle mimics Windows 11 windows.
var WindowsStyle = &Style{
	Name:           "windows",
	TitleFont:      "Inter",
	TitleFontSize:  14,
	TitleBarHeight: 36,
	CornerRadius:   8,
	Light:          Palette{TitleBar: rgb(243, 243, 243), TitleText: rgb(0, 0, 0), Controls: rgb(0, 0, 0)},
	Dark:           Palette{TitleBar: rgb(32, 32, 32), TitleText: rgb(255, 255, 255), Controls: rgb(255, 255, 255)},
	Controls: func(dc *gg.Context, width, bar int, p Palette) {
		const button, icon = 48.0, 10.0
		cy := float64(bar) / 2
		dc.SetColor(p.Controls)
		dc.SetLineWidth(1.25)

		x := float64(width) - button/2 // close
		dc.MoveTo(x-icon/2, cy-icon/2)
		dc.LineTo(x+icon/2, cy+icon/2)
		dc.MoveTo(x-icon/2, cy+icon/2)
		dc.LineTo(x+icon/2, cy-icon/2)
		dc.Stroke()

		x -= button // maximize
		dc.DrawRectangle(x-icon/2, cy-icon/2, icon, icon)
		dc.Stroke()

		x -= button // minimize
		dc.MoveTo(x-icon/2, cy)
		dc.LineTo(x+icon/2, cy)
		dc.Stroke()
	},
}

// GnomeStyle mimics GNOME's Adwaita header bars.
var GnomeStyle = &Style{
	Name:           "gnome",
	TitleFont:      "Cantarell",
	TitleFontSize:  14,
	TitleBarHeight: 45,
	CornerRadius:   12,
	Light:          Palette{TitleBar: rgb(242, 242, 242), TitleText: rgb(40, 40, 40), Controls: rgb(40, 40, 40)},
	Dark:           Palette{TitleBar: rgb(36, 36, 36), TitleText: rgb(255, 255, 255), Controls: rgb(255, 255, 255)},
	Controls: func(dc *gg.Context, width, bar int, p Palette) {
		const size, gap, rightPad = 20.0, 16.0, 16.0
		cy := float64(bar) / 2
		icon := size * 0.4
		dc.SetColor(p.Controls)
		dc.SetLineWidth(2)

		x := float64(width) - rightPad - size/2 // close
		dc.MoveTo(x-icon/2, cy-icon/2)
		dc.LineTo(x+icon/2, cy+icon/2)
		dc.MoveTo(x-icon/2, cy+icon/2)
		dc.LineTo(x+icon/2, cy-icon/2)
		dc.Stroke()

		x -= size + gap // maximize
		dc.DrawRectangle(x-icon/2, cy-icon/2, icon, icon)
		dc.Stroke()

		x -= size + gap // minimize
		dc.MoveTo(x-icon/2, cy+icon/2)
		dc.LineTo(x+icon/2, cy+icon/2)
		dc.Stroke()
	},
}

// BreezeStyle mimics KDE's Breeze window decorations.
var BreezeStyle = &Style{
	Name:           "breeze",
	TitleFont:      "Cantarell",
	TitleFontSize:  14,
	TitleBarHeight: 32,
	CornerRadius:   8,
	Light:          Palette{TitleBar: rgb(205, 209, 214), TitleText: rgb(109, 113, 120), Controls: rgb(40, 40, 40)},
	Dark:           Palette{TitleBar: rgb(42, 46, 50), TitleText: rgb(220, 220, 220), Controls: rgb(220, 220, 220)},
	Controls: func(dc *gg.Context, width, bar int, p Palette) {
		const size, gap, pad = 16.0, 8.0, 8.0
		cy := float64(bar) / 2

		// Close: filled circle with an X cut through it.
		cx := float64(width) - pad - size/2
		dc.SetColor(p.Controls)
		dc.DrawCircle(cx, cy, size/2)
		dc.Fill()
		dc.SetColor(p.TitleBar)
		dc.SetLineWidth(1.5)
		x := size / 4
		dc.DrawLine(cx-x, cy-x, cx+x, cy+x)
		dc.DrawLine(cx-x, cy+x, cx+x, cy-x)
		dc.Stroke()

		// Minimize: downward chevron.
		cx -= size + gap
		dc.SetColor(p.Controls)
		dc.DrawLine(cx-size/4, cy-size/8, cx, cy+size/8)
		dc.DrawLine(cx, cy+size/8, cx+size/4, cy-size/8)
		dc.Stroke()
	},
}

// BlankStyle is a bare window surface with no decorations.
var BlankStyle = &Style{
	Name:         "blank",
	CornerRadius: 6,
	Light:        Palette{TitleBar: color.White, TitleText: color.Black, Controls: color.Black},
	Dark:         Palette{TitleBar: rgb(30, 30, 30), TitleText: color.White, Controls: color.White},
}
