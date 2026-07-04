package code

import (
	"embed"
	"fmt"
	"image/color"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

//go:embed themes/*.xml
var themesFS embed.FS

func init() {
	entries, _ := themesFS.ReadDir("themes")
	for _, e := range entries {
		if data, err := themesFS.ReadFile("themes/" + e.Name()); err == nil {
			if style, err := chroma.NewXMLStyle(strings.NewReader(string(data))); err == nil {
				styles.Register(style)
			}
		}
	}
}

// Themes lists the available syntax highlighting themes.
func Themes() []string { return styles.Names() }

// Languages lists the supported languages, optionally including aliases.
func Languages(aliases bool) []string { return lexers.Names(aliases) }

// token is a run of styled text within a line.
type token struct {
	text                    string
	color                   color.Color
	bold, italic, underline bool
}

type line struct {
	tokens []token
}

func (l line) text() string {
	var b strings.Builder
	for _, t := range l.tokens {
		b.WriteString(t.text)
	}
	return b.String()
}

// palette holds the theme-derived colors the renderer needs.
type palette struct {
	background color.Color
	gutter     color.Color
	highlight  color.Color
	comment    color.Color
}

// highlight tokenizes source into styled lines with tabs expanded.
func highlight(source, language, theme string, tabWidth int) ([]line, palette, error) {
	var lexer chroma.Lexer
	if language != "" {
		if lexer = lexers.Get(language); lexer == nil {
			return nil, palette{}, fmt.Errorf("code: no lexer for language %q", language)
		}
	} else if lexer = lexers.Analyse(source); lexer == nil {
		lexer = lexers.Fallback
	}

	style := styles.Get(theme)
	if style == nil {
		style = styles.Fallback
	}

	iter, err := lexer.Tokenise(nil, source)
	if err != nil {
		return nil, palette{}, fmt.Errorf("code: tokenize: %w", err)
	}

	var lines []line
	current := line{}
	col := 0
	for _, tok := range iter.Tokens() {
		entry := style.Get(tok.Type)
		for i, part := range strings.Split(tok.Value, "\n") {
			if i > 0 {
				lines = append(lines, current)
				current, col = line{}, 0
			}
			if part == "" {
				continue
			}
			text, next := expandTabs(part, col, tabWidth)
			col = next
			current.tokens = append(current.tokens, token{
				text:      text,
				color:     entryColor(style, entry),
				bold:      entry.Bold == chroma.Yes,
				italic:    entry.Italic == chroma.Yes && !entry.NoInherit,
				underline: entry.Underline == chroma.Yes,
			})
		}
	}
	if len(current.tokens) > 0 {
		lines = append(lines, current)
	}

	return lines, themePalette(style), nil
}

func expandTabs(text string, col, tabWidth int) (string, int) {
	if !strings.Contains(text, "\t") {
		return text, col + len(text)
	}
	if tabWidth <= 0 {
		tabWidth = 4
	}
	var b strings.Builder
	for _, r := range text {
		if r == '\t' {
			n := tabWidth - col%tabWidth
			b.WriteString(strings.Repeat(" ", n))
			col += n
		} else {
			b.WriteRune(r)
			col++
		}
	}
	return b.String(), col
}

// --- theme colors ------------------------------------------------------

func themePalette(style *chroma.Style) palette {
	bg := chromaColor(style.Get(chroma.Background).Background, chromaColor(style.Get(chroma.Background).Colour, color.White))
	light := isLight(bg)

	gutter := chromaColor(style.Get(chroma.LineNumbers).Colour, nil)
	if gutter == nil {
		if light {
			gutter = color.RGBA{110, 110, 110, 255}
		} else {
			gutter = color.RGBA{145, 145, 145, 255}
		}
	}

	hl := style.Get(chroma.LineHighlight)
	var highlightColor color.Color
	switch {
	case hl.Background != 0:
		c := hl.Background
		highlightColor = color.NRGBA{c.Red(), c.Green(), c.Blue(), 128}
	case light:
		highlightColor = color.NRGBA{0, 0, 0, 32}
	default:
		highlightColor = color.NRGBA{255, 255, 255, 32}
	}

	return palette{
		background: bg,
		gutter:     gutter,
		highlight:  highlightColor,
		comment:    entryColor(style, style.Get(chroma.Comment)),
	}
}

// entryColor resolves a style entry's foreground with sensible fallbacks.
func entryColor(style *chroma.Style, entry chroma.StyleEntry) color.Color {
	if c := chromaColor(entry.Colour, nil); c != nil {
		return c
	}
	if c := chromaColor(style.Get(chroma.Other).Colour, nil); c != nil {
		return c
	}
	bg := chromaColor(style.Get(chroma.Background).Background, color.White)
	if isLight(bg) {
		return color.RGBA{74, 74, 74, 255}
	}
	return color.RGBA{204, 204, 204, 255}
}

func chromaColor(c chroma.Colour, fallback color.Color) color.Color {
	if c == 0 {
		return fallback
	}
	return color.RGBA{c.Red(), c.Green(), c.Blue(), 255}
}

func isLight(c color.Color) bool {
	r, g, b, _ := c.RGBA()
	return (float64(r>>8)*0.299 + float64(g>>8)*0.587 + float64(b>>8)*0.114) > 128
}
