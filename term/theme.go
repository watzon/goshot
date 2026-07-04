package term

import (
	"embed"
	"fmt"
	"image/color"
	"sort"
	"strconv"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

//go:embed themes/*.yml
var themesFS embed.FS

// Theme is a 16-color terminal color scheme.
type Theme struct {
	Name       string
	Background color.Color
	Foreground color.Color
	Cursor     color.Color
	Palette    [16]color.Color
}

var (
	themeMu    sync.Mutex
	themeIndex map[string]string // normalized name -> embedded filename
	themeCache = map[string]*Theme{}
)

func buildIndex() {
	if themeIndex != nil {
		return
	}
	themeIndex = map[string]string{}
	entries, _ := themesFS.ReadDir("themes")
	for _, e := range entries {
		name := strings.TrimSuffix(e.Name(), ".yml")
		themeIndex[normalizeName(name)] = e.Name()
	}
}

// Themes lists all available terminal theme names.
func Themes() []string {
	themeMu.Lock()
	defer themeMu.Unlock()
	buildIndex()
	names := make([]string, 0, len(themeIndex))
	for n := range themeIndex {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// GetTheme loads a theme by name, or nil if it doesn't exist.
func GetTheme(name string) *Theme {
	themeMu.Lock()
	defer themeMu.Unlock()
	buildIndex()

	key := normalizeName(name)
	if t, ok := themeCache[key]; ok {
		return t
	}
	file, ok := themeIndex[key]
	if !ok {
		return nil
	}
	data, err := themesFS.ReadFile("themes/" + file)
	if err != nil {
		return nil
	}

	var raw map[string]string
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil
	}
	t := &Theme{
		Name:       raw["name"],
		Background: hexColor(raw["background"], color.Black),
		Foreground: hexColor(raw["foreground"], color.White),
		Cursor:     hexColor(raw["cursor"], color.White),
	}
	for i := 0; i < 16; i++ {
		t.Palette[i] = hexColor(raw[fmt.Sprintf("color_%02d", i+1)], color.Black)
	}
	themeCache[key] = t
	return t
}

// Color resolves an xterm 256-color index against the theme.
func (t *Theme) Color(n int) color.Color {
	switch {
	case n < 16:
		return t.Palette[n]
	case n < 232: // 6x6x6 color cube
		n -= 16
		b, n := n%6, n/6
		g, r := n%6, n/6
		level := func(v int) uint8 {
			if v == 0 {
				return 0
			}
			return uint8(55 + v*40)
		}
		return color.RGBA{level(r), level(g), level(b), 255}
	case n < 256: // grayscale ramp
		gray := uint8((n-232)*10 + 8)
		return color.RGBA{gray, gray, gray, 255}
	}
	return t.Foreground
}

func normalizeName(name string) string {
	name = strings.ToLower(name)
	var b strings.Builder
	lastHyphen := false
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			b.WriteRune(r)
			lastHyphen = false
		case !lastHyphen && b.Len() > 0:
			b.WriteByte('-')
			lastHyphen = true
		}
	}
	return strings.TrimSuffix(b.String(), "-")
}

func hexColor(hex string, fallback color.Color) color.Color {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return fallback
	}
	v, err := strconv.ParseUint(hex, 16, 32)
	if err != nil {
		return fallback
	}
	return color.RGBA{uint8(v >> 16), uint8(v >> 8), uint8(v), 255}
}
