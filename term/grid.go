package term

import (
	"image/color"
	"strconv"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

type attributes struct {
	bold, italic, underline, strike bool
}

// cell holds one terminal cell: a whole grapheme cluster (possibly an
// emoji sequence), or "" for the continuation of a wide cluster to its
// left.
type cell struct {
	s     string
	fg    color.Color
	bg    color.Color
	attrs attributes
}

// grid is a minimal terminal screen: a cell matrix plus cursor and
// current drawing state. Dimensions include the cell padding.
type grid struct {
	cells         [][]cell
	width, height int
	curX, curY    int
	maxX, maxY    int

	fg, bg color.Color
	attrs  attributes

	theme                  *Theme
	autoSize               bool
	padL, padR, padT, padB int
}

func newGrid(cols, rows int, pad [4]int, theme *Theme, autoSize bool) *grid {
	g := &grid{
		theme:    theme,
		autoSize: autoSize,
		padL:     pad[0], padR: pad[1], padT: pad[2], padB: pad[3],
		fg: theme.Foreground,
		bg: theme.Background,
	}
	g.width = cols + g.padL + g.padR
	g.height = rows + g.padT + g.padB
	g.curX, g.curY = g.padL, g.padT
	g.grow(g.width, g.height)
	return g
}

func (g *grid) grow(w, h int) {
	blank := cell{s: " ", fg: g.theme.Foreground, bg: g.theme.Background}
	for y := range g.cells {
		for len(g.cells[y]) < w {
			g.cells[y] = append(g.cells[y], blank)
		}
	}
	for len(g.cells) < h {
		row := make([]cell, w)
		for x := range row {
			row[x] = blank
		}
		g.cells = append(g.cells, row)
	}
	g.width, g.height = max(g.width, w), max(g.height, h)
}

// set writes one grapheme cluster spanning w cells at (x, y), marking
// the cells behind a wide cluster as continuations.
func (g *grid) set(x, y int, s string, w int) {
	if x < g.padL || y < g.padT || w < 1 {
		return
	}
	if g.autoSize {
		g.maxX, g.maxY = max(g.maxX, x+w), max(g.maxY, y+1)
		g.grow(max(g.width, x+w), max(g.height, y+1))
	} else if x+w > g.width || y >= g.height {
		return
	}
	g.cells[y][x] = cell{s: s, fg: g.fg, bg: g.bg, attrs: g.attrs}
	for i := 1; i < w; i++ {
		g.cells[y][x+i] = cell{fg: g.fg, bg: g.bg, attrs: g.attrs}
	}
}

func (g *grid) newline() {
	g.curX = g.padL
	g.curY++
}

// parse feeds raw terminal output (with ANSI escapes) into the grid.
func (g *grid) parse(input []byte) {
	parser := ansi.GetParser()
	defer ansi.PutParser(parser)

	lastLine := g.height - g.padB - 1
	var state byte
	for len(input) > 0 {
		if !g.autoSize && g.curY > lastLine {
			break
		}
		seq, width, n, newState := ansi.DecodeSequence(input, state, parser)
		if n == 0 {
			input = input[1:]
			continue
		}
		if width > 0 {
			g.set(g.curX, g.curY, string(seq), width)
			g.curX += width
			if g.width > 0 && g.curX >= g.width {
				g.newline()
			}
		} else {
			g.control(string(seq))
		}
		input = input[n:]
		state = newState
	}
}

func (g *grid) control(s string) {
	switch {
	case s == "\n":
		g.newline()
	case s == "\r":
		g.curX = g.padL
	case strings.HasPrefix(s, "\x1b["):
		g.csi(s)
	}
}

// csi handles the CSI sequences that matter for static rendering:
// SGR styling, cursor movement, and line erasure.
func (g *grid) csi(s string) {
	body := s[2 : len(s)-1]
	final := s[len(s)-1]
	p := csiParams(body)
	arg := func(i, def int) int {
		if i < len(p) {
			return p[i]
		}
		return def
	}

	switch final {
	case 'm':
		g.sgr(p)
	case 'G': // cursor to column
		g.curX = min(g.width-g.padR-1, g.padL+max(1, arg(0, 1))-1)
	case 'H', 'f': // cursor position
		g.curY = min(g.height-1, max(1, arg(0, 1)))
		g.curX = min(g.width-g.padR-1, max(g.padL, arg(1, 1)-1+g.padL))
	case 'A':
		g.curY = max(1, g.curY-max(1, arg(0, 1)))
	case 'B':
		g.curY = min(g.height-1, g.curY+max(1, arg(0, 1)))
	case 'C':
		g.curX = min(g.width-g.padR-1, g.curX+max(1, arg(0, 1)))
	case 'D':
		g.curX = max(g.padL, g.curX-max(1, arg(0, 1)))
	case 'K': // erase in line
		if g.curY >= len(g.cells) {
			return
		}
		from, to := g.curX, len(g.cells[g.curY])
		switch arg(0, 0) {
		case 1:
			from, to = 0, g.curX+1
		case 2:
			from = 0
		}
		blank := cell{s: " ", fg: g.theme.Foreground, bg: g.theme.Background}
		for x := from; x < to; x++ {
			g.cells[g.curY][x] = blank
		}
	}
}

func (g *grid) sgr(p []int) {
	if len(p) == 0 {
		p = []int{0}
	}
	for i := 0; i < len(p); i++ {
		switch code := p[i]; {
		case code == 0:
			g.attrs = attributes{}
			g.fg, g.bg = g.theme.Foreground, g.theme.Background
		case code == 1:
			g.attrs.bold = true
		case code == 3:
			g.attrs.italic = true
		case code == 4:
			g.attrs.underline = true
		case code == 7:
			g.fg, g.bg = g.bg, g.fg
		case code == 9:
			g.attrs.strike = true
		case code == 22:
			g.attrs.bold = false
		case code == 23:
			g.attrs.italic = false
		case code == 24:
			g.attrs.underline = false
		case code == 29:
			g.attrs.strike = false
		case code >= 30 && code <= 37:
			g.fg = g.theme.Palette[code-30]
		case code >= 40 && code <= 47:
			g.bg = g.theme.Palette[code-40]
		case code >= 90 && code <= 97:
			g.fg = g.theme.Palette[code-90+8]
		case code >= 100 && code <= 107:
			g.bg = g.theme.Palette[code-100+8]
		case code == 38 || code == 48:
			c, skip := extendedColor(p[i+1:], g.theme)
			if c != nil {
				if code == 38 {
					g.fg = c
				} else {
					g.bg = c
				}
			}
			i += skip
		case code == 39:
			g.fg = g.theme.Foreground
		case code == 49:
			g.bg = g.theme.Background
		}
	}
}

// extendedColor parses the tail of a 38/48 SGR: "2;r;g;b" or "5;n".
func extendedColor(p []int, theme *Theme) (color.Color, int) {
	if len(p) >= 4 && p[0] == 2 {
		return color.RGBA{uint8(p[1]), uint8(p[2]), uint8(p[3]), 255}, 4
	}
	if len(p) >= 2 && p[0] == 5 {
		return theme.Color(p[1]), 2
	}
	return nil, len(p)
}

func csiParams(body string) []int {
	if body == "" {
		return nil
	}
	parts := strings.Split(body, ";")
	out := make([]int, 0, len(parts))
	for _, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil {
			n = 0
		}
		out = append(out, n)
	}
	return out
}
