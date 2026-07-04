package code

import (
	"image"
	"image/draw"
	"regexp"
	"sort"

	"github.com/disintegration/imaging"
	"golang.org/x/image/math/fixed"
)

// RedactStyle selects how redacted text is obscured.
type RedactStyle string

const (
	RedactBlock RedactStyle = "block" // replace characters with █
	RedactBlur  RedactStyle = "blur"  // blur the pixels
)

// Pattern is a named regular expression whose first capture group is
// redacted wherever it matches.
type Pattern struct {
	Regexp *regexp.Regexp
	Name   string
}

// Area is a manually redacted rectangle in image coordinates.
type Area struct {
	X, Y, Width, Height int
}

// Redaction configures automatic and manual hiding of sensitive content.
type Redaction struct {
	Enabled    bool
	Style      RedactStyle
	BlurRadius float64
	Patterns   []Pattern
	Areas      []Area
}

// NewRedaction returns an enabled redaction config with the default
// patterns and block style.
func NewRedaction() *Redaction {
	return &Redaction{
		Enabled:    true,
		Style:      RedactBlock,
		BlurRadius: 5,
		Patterns:   DefaultPatterns,
	}
}

// WithStyle sets the redaction style.
func (r *Redaction) WithStyle(s RedactStyle) *Redaction { r.Style = s; return r }

// WithBlurRadius sets the blur strength for RedactBlur.
func (r *Redaction) WithBlurRadius(radius float64) *Redaction { r.BlurRadius = radius; return r }

// WithPattern adds a custom pattern; its first capture group is redacted.
func (r *Redaction) WithPattern(expr, name string) *Redaction {
	if re, err := regexp.Compile(expr); err == nil {
		r.Patterns = append(r.Patterns, Pattern{Regexp: re, Name: name})
	}
	return r
}

// WithArea adds a manual redaction rectangle.
func (r *Redaction) WithArea(x, y, width, height int) *Redaction {
	r.Areas = append(r.Areas, Area{x, y, width, height})
	return r
}

// DefaultPatterns match common shapes of secrets: known token formats,
// basic-auth passwords in URLs, and values assigned to sensitive names.
var DefaultPatterns = []Pattern{
	{regexp.MustCompile(`(?i)"(?P<value>(?:` +
		`sk_(?:live|test)_[\w]{24,}|` + // Stripe
		`sk-[\w]{32,}|` + // OpenAI
		`gh[porsu]_[\w]{36,}|` + // GitHub
		`AKIA[\w]{16}|` + // AWS
		`eyJ[\w\-_=]+\.eyJ[\w\-_=]+\.[\w\-_.+/=]+` + // JWT
		`))"`), "Known Secret Format"},
	{regexp.MustCompile(`(?i)(?:\w+)?://[^:]+:(?P<value>[^@\s]+)@`), "URL Password"},
	{regexp.MustCompile(`(?im)(?:^|\s|"|,)\s*(?:")?[\w]*` +
		`(?:key|token|secret|pass(?:word)?|pwd|auth|cred)[\w]*(?:")?\s*` +
		`(?:[:=]|:=)\s*(?:"|` + "`" + `)(?P<value>(?:[^"` + "`" + `]|(?:\n|\\n)[^"` + "`" + `]*)*?)(?:"|` + "`" + `)`),
		"Sensitive Variable"},
}

// span is a redacted byte range within a text.
type span struct {
	start, end int
}

// find returns the merged spans of all pattern matches in text.
func (r *Redaction) find(text string) []span {
	var spans []span
	for _, p := range r.Patterns {
		for _, m := range p.Regexp.FindAllSubmatchIndex([]byte(text), -1) {
			if len(m) >= 4 && m[2] >= 0 {
				spans = append(spans, span{m[2], m[3]})
			}
		}
	}
	sort.Slice(spans, func(i, j int) bool { return spans[i].start < spans[j].start })

	var merged []span
	for _, s := range spans {
		if n := len(merged); n > 0 && s.start <= merged[n-1].end {
			merged[n-1].end = max(merged[n-1].end, s.end)
		} else {
			merged = append(merged, s)
		}
	}
	return merged
}

func covered(col int, spans []span) bool {
	for _, s := range spans {
		if col >= s.start && col < s.end {
			return true
		}
	}
	return false
}

// mergeRects joins horizontally adjacent per-character rectangles into
// contiguous runs so each run is blurred once.
func mergeRects(rects []image.Rectangle) []image.Rectangle {
	var out []image.Rectangle
	for _, r := range rects {
		if n := len(out); n > 0 && out[n-1].Min.Y == r.Min.Y && out[n-1].Max.X == r.Min.X {
			out[n-1].Max.X = r.Max.X
		} else {
			out = append(out, r)
		}
	}
	return out
}

// blurArea blurs a rectangle of the image in place.
func blurArea(img *image.RGBA, rect image.Rectangle, radius float64) {
	rect = rect.Intersect(img.Bounds())
	if rect.Empty() {
		return
	}
	region := image.NewRGBA(rect.Sub(rect.Min))
	draw.Draw(region, region.Bounds(), img, rect.Min, draw.Src)
	blurred := imaging.Blur(region, radius)
	draw.Draw(img, rect, blurred, image.Point{}, draw.Src)
}

func fixedI(v int) fixed.Int26_6 { return fixed.I(v) }
