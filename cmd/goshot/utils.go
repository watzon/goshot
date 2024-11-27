package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/xpty"
	"github.com/watzon/goshot/pkg/background"
	"github.com/watzon/goshot/pkg/chrome"
	"github.com/watzon/goshot/pkg/fonts"
	"github.com/watzon/goshot/pkg/render"
	"golang.org/x/term"
)

// Helper functions
func parseHexColor(hex string) (color.Color, error) {
	hex = strings.TrimPrefix(hex, "#")
	var r, g, b, a uint8

	switch len(hex) {
	case 6:
		_, err := fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b)
		if err != nil {
			return nil, err
		}
		a = 255
	case 8:
		_, err := fmt.Sscanf(hex, "%02x%02x%02x%02x", &r, &g, &b, &a)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("invalid hex color: %s", hex)
	}

	return color.RGBA{R: r, G: g, B: b, A: a}, nil
}

func parseHighlightLines(input string) ([]int, error) {
	var result []int
	parts := strings.Split(input, ";")

	for _, part := range parts {
		if strings.Contains(part, "-") {
			// Handle range (e.g., "1-3")
			var start, end int
			if _, err := fmt.Sscanf(part, "%d-%d", &start, &end); err != nil {
				return nil, err
			}
			for i := start; i <= end; i++ {
				result = append(result, i)
			}
		} else {
			// Handle single line
			var line int
			if _, err := fmt.Sscanf(part, "%d", &line); err != nil {
				return nil, err
			}
			result = append(result, line)
		}
	}

	return result, nil
}

// parseFonts takes in a string of fonts and returns the first font
// that is available on the system.
// Ex. "JetBrains Mono; DejaVu Sans=30"
func parseFonts(input string) (string, float64) {
	for _, fontSpec := range strings.Split(input, ";") {
		parts := strings.Split(strings.TrimSpace(fontSpec), "=")
		fontName := strings.TrimSpace(parts[0])
		fontSize := 14.0
		if len(parts) > 1 {
			if parsedSize, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64); err == nil {
				fontSize = parsedSize
			}
		}

		if fonts.IsFontAvailable(fontName) {
			return fontName, fontSize
		}
	}

	return "", 14.0
}

// parseGradientStops takes in a string slice of gradient stops and returns
// a slice of background.GradientStop.
func parseGradientStops(input []string) ([]background.GradientStop, error) {
	var result []background.GradientStop
	for _, part := range input {
		parts := strings.Split(part, ";")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid gradient stop format: %s; expected hex color and percentage (e.g., #ff0000;50)", part)
		}

		hexColor := strings.TrimSpace(parts[0])
		positionStr := strings.TrimSpace(parts[1])

		color, err := parseHexColor(hexColor)
		if err != nil {
			return nil, fmt.Errorf("invalid color in gradient stop: %s", err)
		}

		position, err := strconv.ParseFloat(positionStr, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid position in gradient stop: %s", err)
		}

		if position < 0 || position > 100 {
			return nil, fmt.Errorf("gradient stop position must be between 0 and 100: %f", position)
		}

		result = append(result, background.GradientStop{
			Color:    color,
			Position: position / 100, // Convert percentage to decimal
		})
	}
	return result, nil
}

func executeComamand(ctx context.Context, args []string) ([]byte, error) {
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		width = 80
		height = 24
	}

	pty, err := xpty.NewPty(width, height)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = pty.Close()
	}()

	cmd := exec.CommandContext(ctx, args[0], args[1:]...) //nolint: gosec
	if err := pty.Start(cmd); err != nil {
		return nil, err
	}

	var out bytes.Buffer
	var errorOut bytes.Buffer
	go func() {
		_, _ = io.Copy(&out, pty)
		errorOut.Write(out.Bytes())
	}()

	if err := xpty.WaitProcess(ctx, cmd); err != nil {
		return errorOut.Bytes(), err //nolint: wrapcheck
	}
	return out.Bytes(), nil
}

func renderCode(config *Config, echo bool, code string) error {
	canvas, err := makeCanvas(config)
	if err != nil {
		return err
	}

	// Configure highlighted lines
	highlightedLines := []render.LineRange{}
	if config.HighlightLines != "" {
		lines, err := parseHighlightLines(config.HighlightLines)
		if err != nil {
			return err
		}
		for _, line := range lines {
			highlightedLines = append(highlightedLines, render.LineRange{Start: line, End: line})
		}
	}

	// Get font
	fontSize := 14.0
	var requestedFont *fonts.Font
	if config.Font != "" {
		var fontStr string
		fontStr, fontSize = parseFonts(config.Font)
		if fontStr != "" {
			requestedFont, err = fonts.GetFont(fontStr, nil)
			if err != nil {
				return err
			}
		}
	}

	// Configure code style
	canvas.WithCodeStyle(&render.CodeStyle{
		Language:        config.Language,
		Theme:           strings.ToLower(config.Theme),
		FontFamily:      requestedFont,
		FontSize:        fontSize,
		TabWidth:        config.TabWidth,
		PaddingLeft:     config.CodePadLeft,
		PaddingRight:    config.CodePadRight,
		PaddingTop:      config.CodePadTop,
		PaddingBottom:   config.CodePadBottom,
		ShowLineNumbers: !config.NoLineNumbers,
		LineNumberRange: render.LineRange{
			Start: config.StartLine,
			End:   config.EndLine,
		},
		LineHighlightRanges: highlightedLines,
	})

	// Render the image
	img, err := canvas.RenderToImage(code)
	if err != nil {
		return fmt.Errorf("failed to render image: %v", err)
	}

	if config.ToClipboard || config.ToStdout {
		pngBuf := bytes.NewBuffer(nil)
		if err := png.Encode(pngBuf, img); err != nil {
			return fmt.Errorf("failed to encode image to png: %v", err)
		}

		// NOTE: Not all clipboard backends recognize the png header.
		//       wl-clipboard and xclip both should.
		if config.ToClipboard {
			err := clipboard.WriteAll(pngBuf.String())
			if err != nil {
				return fmt.Errorf("failed to copy image to clipboard: %v", err)
			}

			if echo {
				logMessage(styles.successBox, "COPIED", "to clipboard")
			}
		}

		if config.ToStdout {
			_, err := os.Stdout.Write(pngBuf.Bytes())
			if err != nil {
				return fmt.Errorf("failed to write image to stdout: %v", err)
			}

			if echo {
				logMessage(styles.successBox, "WROTE", "to stdout")
			}
		}
		return nil
	}

	err = saveImage(img, config)
	if err == nil {
		if echo {
			logMessage(styles.successBox, "WROTE", config.OutputFile)
		}
	} else {
		return fmt.Errorf("failed to save image: %v", err)
	}

	return nil
}

func makeCanvas(config *Config) (*render.Canvas, error) {
	var err error

	// Create canvas
	canvas := render.NewCanvas()

	// Set window chrome
	themeVariant := chrome.ThemeVariantLight
	if config.DarkMode {
		themeVariant = chrome.ThemeVariantDark
	}

	if config.NoWindowControls {
		window := chrome.NewBlankChrome().
			WithCornerRadius(config.WindowCornerRadius)
		canvas.WithChrome(window)
	} else {
		var window chrome.Chrome
		switch config.WindowChrome {
		case "mac":
			window = chrome.NewMacChrome(chrome.MacStyleSequoia)
		case "windows":
			window = chrome.NewWindowsChrome(chrome.WindowsStyleWin11)
		case "gnome":
			window = chrome.NewGNOMEChrome(chrome.GNOMEStyleAdwaita)
		default:
			return nil, fmt.Errorf("invalid chrome style: %s", config.WindowChrome)
		}

		if config.ChromeThemeName == "" {
			window = window.WithVariant(themeVariant)
		} else {
			window = window.WithThemeByName(config.ChromeThemeName, themeVariant)
		}

		if config.AutoTitle {
			// if len(args) > 0 {
			// 	window = window.WithTitle(filepath.Base(args[0]))
			// } else if config.ExecuteCommand != "" {
			// 	window = window.WithTitle(config.ExecuteCommand)
			// }
		} else {
			window = window.WithTitle(config.WindowTitle)
		}

		window = window.WithCornerRadius(config.WindowCornerRadius)
		canvas.WithChrome(window)
	}

	// Set background
	var bg background.Background
	if config.BackgroundImage != "" {
		file, err := os.Open(config.BackgroundImage)
		if err != nil {
			return nil, fmt.Errorf("failed to open background image: %v", err)
		}
		defer file.Close()
		backgroundImage, _, err := image.Decode(file)
		if err != nil {
			return nil, fmt.Errorf("failed to decode background image: %v", err)
		}
		var fit background.ImageScaleMode
		switch config.BackgroundImageFit {
		case "fit":
			fit = background.ImageScaleFit
		case "cover":
			fit = background.ImageScaleCover
		case "fill":
			fit = background.ImageScaleFill
		case "stretch":
			fit = background.ImageScaleStretch
		case "tile":
			fit = background.ImageScaleTile
		default:
			return nil, fmt.Errorf("invalid background image fit mode: %s", config.BackgroundImageFit)
		}
		bg = background.
			NewImageBackground(backgroundImage).
			WithScaleMode(fit).
			WithPaddingDetailed(config.PadHoriz, config.PadVert, config.PadHoriz, config.PadVert)
	} else if config.GradientType != "" {
		stops, err := parseGradientStops(config.GradientStops)
		if err != nil {
			return nil, fmt.Errorf("invalid gradient stops: %v", err)
		}

		var gradient background.GradientType
		switch config.GradientType {
		case "linear":
			gradient = background.LinearGradient
		case "radial":
			gradient = background.RadialGradient
		case "angular":
			gradient = background.AngularGradient
		case "diamond":
			gradient = background.DiamondGradient
		case "spiral":
			gradient = background.SpiralGradient
		case "square":
			gradient = background.SquareGradient
		case "star":
			gradient = background.StarGradient
		default:
			return nil, fmt.Errorf("invalid gradient type: %s", config.GradientType)
		}

		bg = background.NewGradientBackground(gradient, stops...).
			WithAngle(config.GradientAngle).
			WithCenter(config.GradientCenterX, config.GradientCenterY).
			WithIntensity(config.GradientIntensity).
			WithCenter(config.GradientCenterX, config.GradientCenterY).
			WithPaddingDetailed(config.PadHoriz, config.PadVert, config.PadHoriz, config.PadVert)
	} else if config.BackgroundColor != "" {
		// Parse background color
		var bgColor color.Color
		if config.BackgroundColor == "transparent" {
			bgColor = color.Transparent
		} else {
			bgColor, err = parseHexColor(config.BackgroundColor)
		}

		if err != nil {
			return nil, fmt.Errorf("invalid background color: %v", err)
		}

		bg = background.NewColorBackground().
			WithColor(bgColor).
			WithPaddingDetailed(config.PadHoriz, config.PadVert, config.PadHoriz, config.PadVert)
	}

	if bg != nil {
		// Configure shadow if enabled
		if config.ShadowBlurRadius > 0 {
			shadowCol, err := parseHexColor(config.ShadowColor)
			if err != nil {
				return nil, fmt.Errorf("invalid shadow color: %v", err)
			}
			bg = bg.WithShadow(background.NewShadow().
				WithBlur(config.ShadowBlurRadius).
				WithOffset(config.ShadowOffsetX, config.ShadowOffsetY).
				WithColor(shadowCol).
				WithSpread(config.ShadowSpread))
		}

		// Configure corner radius
		if config.CornerRadius > 0 {
			bg = bg.WithCornerRadius(config.CornerRadius)
		}

		// Set background
		canvas.WithBackground(bg)
	}

	return canvas, nil
}

// logMessage prints a styled message with consistent alignment
func logMessage(box lipgloss.Style, tag string, message string) {
	// Set a consistent width for the tag box and center the text
	const boxWidth = 11 // 9 characters + 2 padding spaces
	paddedTag := fmt.Sprintf("%*s", -boxWidth, tag)
	centeredBox := box.Width(boxWidth).Align(lipgloss.Center)
	fmt.Fprintln(os.Stderr, centeredBox.Render(paddedTag)+" "+styles.info.Render(message))
}

func saveImage(img image.Image, config *Config) error {
	// If no output file is specified, use png as default
	if config.OutputFile == "" {
		config.OutputFile = "output.png"
	}

	// Get the extension from the filename
	ext := strings.ToLower(filepath.Ext(config.OutputFile))
	if ext == "" {
		ext = ".png"
		config.OutputFile += ext
	}

	// Save in the format matching the extension
	switch ext {
	case ".png":
		return render.SaveAsPNG(img, config.OutputFile)
	case ".jpg", ".jpeg":
		return render.SaveAsJPEG(img, config.OutputFile)
	case ".bmp":
		return render.SaveAsBMP(img, config.OutputFile)
	default:
		return fmt.Errorf("unsupported file format: %s", ext)
	}
}
