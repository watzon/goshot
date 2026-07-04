package cli

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"

	"github.com/watzon/goshot"
	"github.com/watzon/goshot/background"
	"github.com/watzon/goshot/chrome"
	"github.com/watzon/goshot/code"
	"github.com/watzon/goshot/fonts"
	"github.com/watzon/goshot/term"
)

// renderCode renders highlighted source and writes the outputs.
func renderCode(source string) error {
	family, size, err := resolveFont()
	if err != nil {
		return err
	}

	content := code.New(source).
		WithLanguage(detectLanguage()).
		WithTheme(opts.theme).
		WithFont(family).
		WithFontSize(size).
		WithLineHeight(opts.lineHeight).
		WithPadding(opts.codePad[2], opts.codePad[3], opts.codePad[0], opts.codePad[1]).
		WithLineNumbers(!opts.noLineNumbers).
		WithLineNumberPadding(opts.numberPad).
		WithTabWidth(opts.tabWidth).
		WithMinWidth(opts.minWidth).
		WithMaxWidth(opts.maxWidth)

	if ranges, err := parseRanges(opts.lineRanges); err != nil {
		return err
	} else {
		for _, r := range ranges {
			content.WithLineRange(r.Start, r.End)
		}
	}
	if ranges, err := parseRanges(opts.highlights); err != nil {
		return err
	} else {
		for _, r := range ranges {
			content.WithHighlightRange(r.Start, r.End)
		}
	}

	if opts.redact {
		red, err := buildRedaction()
		if err != nil {
			return err
		}
		content.WithRedaction(red)
	}

	return output(canvasFor(content, nil))
}

// renderTerm renders captured terminal output and writes the outputs.
func renderTerm(args []string, captured []byte) error {
	family, size, err := resolveFont()
	if err != nil {
		return err
	}

	content := term.New(captured).
		WithTheme(opts.theme).
		WithFont(family).
		WithFontSize(size).
		WithLineHeight(opts.lineHeight).
		WithSize(opts.cols, opts.rows).
		WithPadding(opts.cellPad[0], opts.cellPad[1], opts.cellPad[2], opts.cellPad[3]).
		WithCellSpacing(opts.cellSpacing).
		WithCommand(args...)
	if opts.autoSize {
		content.WithAutoSize()
	}
	if opts.showPrompt {
		content.WithPrompt(promptFunc(opts.promptTemplate))
	}

	return output(canvasFor(content, args))
}

func canvasFor(content goshot.Content, args []string) (*goshot.Canvas, error) {
	canvas := goshot.New().WithContent(content)

	win, err := buildChrome(args)
	if err != nil {
		return nil, err
	}
	canvas.WithChrome(win)

	bg, err := buildBackground()
	if err != nil {
		return nil, err
	}
	if bg != nil {
		canvas.WithBackground(bg)
	}
	return canvas, nil
}

func buildChrome(args []string) (*chrome.Chrome, error) {
	if opts.noWindow {
		return chrome.Blank().WithCornerRadius(opts.windowRadius), nil
	}
	win, ok := chrome.Named(opts.chromeStyle)
	if !ok {
		return nil, fmt.Errorf("unknown chrome style %q (have: %s)", opts.chromeStyle, strings.Join(chrome.Names(), ", "))
	}
	if !opts.lightMode {
		win.Dark()
	}
	title := opts.windowTitle
	if opts.autoTitle && len(args) > 0 {
		title = args[0]
	}
	return win.WithTitle(title).WithCornerRadius(opts.windowRadius), nil
}

func buildBackground() (*background.Background, error) {
	var bg *background.Background

	switch {
	case opts.bgImage != "":
		img, err := background.ImageFromFile(opts.bgImage)
		if err != nil {
			return nil, err
		}
		mode, err := scaleMode(opts.bgImageFit)
		if err != nil {
			return nil, err
		}
		bg = img.WithScaleMode(mode)

	case opts.gradientType != "":
		kind, err := gradientKind(opts.gradientType)
		if err != nil {
			return nil, err
		}
		stops, err := parseGradientStops(opts.gradientStops)
		if err != nil {
			return nil, err
		}
		bg = background.Gradient(kind, stops...).
			WithAngle(opts.gradientAngle).
			WithCenter(opts.gradientCenterX, opts.gradientCenterY).
			WithIntensity(opts.gradientIntensity)

	case opts.background != "":
		var c color.Color = color.Transparent
		if opts.background != "transparent" {
			var err error
			if c, err = parseHexColor(opts.background); err != nil {
				return nil, fmt.Errorf("invalid background color: %w", err)
			}
		}
		bg = background.Solid(c)

	default:
		return nil, nil
	}

	if opts.bgBlur > 0 {
		style := background.GaussianBlur
		if opts.bgBlurType == "pixelated" {
			style = background.PixelatedBlur
		}
		bg.WithBlur(style, opts.bgBlur)
	}
	if opts.shadowBlur > 0 {
		c, err := parseHexColor(opts.shadowColor)
		if err != nil {
			return nil, fmt.Errorf("invalid shadow color: %w", err)
		}
		bg.WithShadow(background.NewShadow().
			WithBlur(opts.shadowBlur).
			WithSpread(opts.shadowSpread).
			WithOffset(opts.shadowOffsetX, opts.shadowOffsetY).
			WithColor(c))
	}
	return bg.
		WithCornerRadius(opts.cornerRadius).
		WithPaddingDetailed(opts.padVert, opts.padHoriz, opts.padVert, opts.padHoriz), nil
}

func buildRedaction() (*code.Redaction, error) {
	red := code.NewRedaction().WithBlurRadius(opts.redactBlur)
	switch opts.redactStyle {
	case "block":
		red.WithStyle(code.RedactBlock)
	case "blur":
		red.WithStyle(code.RedactBlur)
	default:
		return nil, fmt.Errorf("invalid redaction style %q (block or blur)", opts.redactStyle)
	}
	for _, p := range opts.redactPatterns {
		red.WithPattern(p, "custom")
	}
	for _, area := range opts.redactAreas {
		var x, y, w, h int
		if _, err := fmt.Sscanf(area, "%d,%d,%d,%d", &x, &y, &w, &h); err != nil {
			return nil, fmt.Errorf("invalid redaction area %q (want 'x,y,width,height')", area)
		}
		red.WithArea(x, y, w, h)
	}
	return red, nil
}

// --- parsing helpers ---------------------------------------------------

// resolveFont picks the first available family from a "Name; Name=size"
// list and returns it with its size.
func resolveFont() (*fonts.Collection, float64, error) {
	if opts.font == "" {
		return fonts.Fallback(), 14, nil
	}
	for _, spec := range strings.Split(opts.font, ";") {
		name, sizeStr, _ := strings.Cut(strings.TrimSpace(spec), "=")
		name = strings.TrimSpace(name)
		size := 14.0
		if v, err := strconv.ParseFloat(strings.TrimSpace(sizeStr), 64); err == nil && v > 0 {
			size = v
		}
		if family, err := fonts.Get(name); err == nil {
			return family, size, nil
		}
	}
	return nil, 0, fmt.Errorf("no available font in %q", opts.font)
}

func parseHexColor(s string) (color.Color, error) {
	s = strings.TrimPrefix(s, "#")
	var r, g, b, a uint8 = 0, 0, 0, 255
	switch len(s) {
	case 6:
		if _, err := fmt.Sscanf(s, "%02x%02x%02x", &r, &g, &b); err != nil {
			return nil, err
		}
	case 8:
		if _, err := fmt.Sscanf(s, "%02x%02x%02x%02x", &r, &g, &b, &a); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("invalid hex color %q", s)
	}
	return color.RGBA{r, g, b, a}, nil
}

// parseRanges parses "5", "5..10", "..10", and "5.." into code Ranges.
func parseRanges(specs []string) ([]code.Range, error) {
	var out []code.Range
	for _, spec := range specs {
		start, end, isRange := strings.Cut(spec, "..")
		r := code.Range{Start: 1, End: 0}
		if s := strings.TrimSpace(start); s != "" {
			n, err := strconv.Atoi(s)
			if err != nil {
				return nil, fmt.Errorf("invalid line range %q", spec)
			}
			r.Start = n
			if !isRange {
				r.End = n
			}
		}
		if e := strings.TrimSpace(end); isRange && e != "" {
			n, err := strconv.Atoi(e)
			if err != nil {
				return nil, fmt.Errorf("invalid line range %q", spec)
			}
			r.End = n
		}
		out = append(out, r)
	}
	return out, nil
}

func parseGradientStops(specs []string) ([]background.Stop, error) {
	var stops []background.Stop
	for _, spec := range specs {
		colorStr, posStr, ok := strings.Cut(spec, ";")
		if !ok {
			return nil, fmt.Errorf("invalid gradient stop %q (want '#color;position')", spec)
		}
		c, err := parseHexColor(strings.TrimSpace(colorStr))
		if err != nil {
			return nil, err
		}
		pos, err := strconv.ParseFloat(strings.TrimSpace(posStr), 64)
		if err != nil || pos < 0 || pos > 100 {
			return nil, fmt.Errorf("invalid gradient stop position %q (0-100)", posStr)
		}
		stops = append(stops, background.Stop{Color: c, Position: pos / 100})
	}
	return stops, nil
}

func gradientKind(name string) (background.GradientType, error) {
	kinds := map[string]background.GradientType{
		"linear": background.LinearGradient, "radial": background.RadialGradient,
		"angular": background.AngularGradient, "diamond": background.DiamondGradient,
		"spiral": background.SpiralGradient, "square": background.SquareGradient,
		"star": background.StarGradient,
	}
	if k, ok := kinds[name]; ok {
		return k, nil
	}
	return 0, fmt.Errorf("unknown gradient type %q", name)
}

func scaleMode(name string) (background.ScaleMode, error) {
	modes := map[string]background.ScaleMode{
		"fit": background.Fit, "fill": background.Fill, "cover": background.Cover,
		"stretch": background.Stretch, "tile": background.Tile,
	}
	if m, ok := modes[name]; ok {
		return m, nil
	}
	return 0, fmt.Errorf("unknown image fit %q", name)
}
