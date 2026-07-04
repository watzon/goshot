// Package fonts loads font families from the binary's embedded fonts and
// from the operating system's font directories.
package fonts

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

//go:embed embedded
var embedded embed.FS

// Weight is a CSS-style numeric font weight.
type Weight int

const (
	Thin       Weight = 100
	ExtraLight Weight = 200
	Light      Weight = 300
	Regular    Weight = 400
	Medium     Weight = 500
	SemiBold   Weight = 600
	Bold       Weight = 700
	ExtraBold  Weight = 800
	Black      Weight = 900
)

// Style identifies one variant within a family.
type Style struct {
	Weight Weight
	Italic bool
}

// Font is a single parsed variant of a family.
type Font struct {
	Family string
	Style  Style
	OTF    *opentype.Font
}

// Collection holds every discovered variant of one family.
type Collection struct {
	Name  string
	Fonts []*Font
}

var (
	cache   = map[string]*Collection{}
	cacheMu sync.Mutex
)

var systemDirs = map[string][]string{
	"linux":   {"~/.fonts", "~/.local/share/fonts", "/usr/share/fonts", "/usr/local/share/fonts"},
	"darwin":  {"/System/Library/Fonts", "/Library/Fonts", "~/Library/Fonts"},
	"windows": {`C:\Windows\Fonts`},
}

// Get loads a family by name, searching embedded fonts first and then the
// system font directories. Results are cached for the life of the process.
func Get(name string) (*Collection, error) {
	key := normalize(name)
	if key == "" {
		return nil, fmt.Errorf("fonts: empty font name")
	}

	cacheMu.Lock()
	defer cacheMu.Unlock()
	if c, ok := cache[key]; ok {
		if c == nil {
			return nil, fmt.Errorf("fonts: %q not found", name)
		}
		return c, nil
	}

	c := &Collection{Name: name}
	if entries, err := embedded.ReadDir("embedded"); err == nil {
		for _, e := range entries {
			if familyKey(e.Name()) != key {
				continue
			}
			if data, err := embedded.ReadFile("embedded/" + e.Name()); err == nil {
				c.add(name, e.Name(), data)
			}
		}
	}
	for _, dir := range expandDirs(systemDirs[runtime.GOOS]) {
		filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !isFontFile(path) || familyKey(filepath.Base(path)) != key {
				return nil
			}
			if data, err := os.ReadFile(path); err == nil {
				c.add(name, filepath.Base(path), data)
			}
			return nil
		})
	}

	if len(c.Fonts) == 0 {
		cache[key] = nil
		return nil, fmt.Errorf("fonts: %q not found", name)
	}
	cache[key] = c
	return c, nil
}

// Fallback returns the embedded JetBrains Mono Nerd Font.
func Fallback() *Collection { return mustGet("JetBrainsMonoNerdFont") }

// FallbackSans returns the embedded Inter font.
func FallbackSans() *Collection { return mustGet("Inter") }

func mustGet(name string) *Collection {
	c, err := Get(name)
	if err != nil {
		panic(err) // embedded fonts are always present
	}
	return c
}

// Available reports whether a family can be loaded.
func Available(name string) bool {
	_, err := Get(name)
	return err == nil
}

// List returns the names of all discoverable font families.
func List() []string {
	seen := map[string]string{}
	if entries, err := embedded.ReadDir("embedded"); err == nil {
		for _, e := range entries {
			seen[familyKey(e.Name())] = displayName(e.Name())
		}
	}
	for _, dir := range expandDirs(systemDirs[runtime.GOOS]) {
		filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !isFontFile(path) {
				return nil
			}
			base := filepath.Base(path)
			if key := familyKey(base); seen[key] == "" {
				seen[key] = displayName(base)
			}
			return nil
		})
	}
	names := make([]string, 0, len(seen))
	for _, n := range seen {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

func (c *Collection) add(family, filename string, data []byte) {
	otf, err := opentype.Parse(data)
	if err != nil {
		return
	}
	c.Fonts = append(c.Fonts, &Font{Family: family, Style: styleOf(filename), OTF: otf})
}

// Face builds a font.Face at the given size from the variant that most
// closely matches the requested style.
func (c *Collection) Face(size float64, style Style) (font.Face, error) {
	if len(c.Fonts) == 0 {
		return nil, fmt.Errorf("fonts: %q has no variants", c.Name)
	}
	best, bestScore := c.Fonts[0], -1<<30
	for _, f := range c.Fonts {
		score := -abs(int(f.Style.Weight) - int(style.Weight))
		if f.Style.Italic == style.Italic {
			score += 500
		}
		if score > bestScore {
			best, bestScore = f, score
		}
	}
	return opentype.NewFace(best.OTF, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingFull})
}

// --- name handling -----------------------------------------------------

var weightTokens = []struct {
	token  string
	weight Weight
}{
	{"extralight", ExtraLight}, {"ultralight", ExtraLight},
	{"semibold", SemiBold}, {"demibold", SemiBold},
	{"extrabold", ExtraBold}, {"ultrabold", ExtraBold},
	{"thin", Thin}, {"light", Light}, {"medium", Medium},
	{"black", Black}, {"heavy", Black}, {"bold", Bold},
	{"regular", Regular}, {"normal", Regular}, {"book", Regular},
}

func styleOf(filename string) Style {
	name := strings.ToLower(strings.TrimSuffix(filename, filepath.Ext(filename)))
	s := Style{Weight: Regular, Italic: strings.Contains(name, "italic") || strings.Contains(name, "oblique")}
	for _, wt := range weightTokens {
		if strings.Contains(name, wt.token) {
			s.Weight = wt.weight
			break
		}
	}
	return s
}

// familyKey reduces a font filename to a comparable family identifier by
// dropping the extension, separators, and trailing style descriptors.
func familyKey(filename string) string {
	return normalize(strings.TrimSuffix(filename, filepath.Ext(filename)))
}

func normalize(name string) string {
	key := strings.ToLower(strings.NewReplacer("-", "", "_", "", " ", "").Replace(name))
	for trimmed := true; trimmed; {
		trimmed = false
		for _, suffix := range styleSuffixes {
			if len(key) > len(suffix) && strings.HasSuffix(key, suffix) {
				key = key[:len(key)-len(suffix)]
				trimmed = true
			}
		}
	}
	return key
}

var styleSuffixes = func() []string {
	s := []string{"italic", "oblique"}
	for _, wt := range weightTokens {
		s = append(s, wt.token)
	}
	return s
}()

// displayName turns a filename into a human-friendly family name.
func displayName(filename string) string {
	base := strings.TrimSuffix(filename, filepath.Ext(filename))
	if i := strings.IndexAny(base, "-_"); i > 0 {
		base = base[:i]
	}
	return base
}

func isFontFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".ttf", ".otf":
		return true
	}
	return false
}

func expandDirs(dirs []string) []string {
	home, _ := os.UserHomeDir()
	out := make([]string, 0, len(dirs))
	for _, d := range dirs {
		if strings.HasPrefix(d, "~") && home != "" {
			d = filepath.Join(home, d[1:])
		}
		out = append(out, d)
	}
	return out
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
