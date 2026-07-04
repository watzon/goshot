package fonts

import "testing"

func TestFallbacks(t *testing.T) {
	for _, c := range []*Collection{Fallback(), FallbackSans()} {
		if len(c.Fonts) == 0 {
			t.Fatalf("%s: no variants", c.Name)
		}
		if _, err := c.Face(14, Style{Weight: Bold}); err != nil {
			t.Fatalf("%s: face: %v", c.Name, err)
		}
	}
}

func TestStyleSelection(t *testing.T) {
	mono := Fallback()
	bold := false
	for _, f := range mono.Fonts {
		if f.Style.Weight == Bold && !f.Style.Italic {
			bold = true
		}
	}
	if !bold {
		t.Error("expected a bold variant in the embedded mono font")
	}
}

func TestNormalize(t *testing.T) {
	cases := map[string]string{
		"JetBrainsMonoNerdFont-Bold.ttf":   "jetbrainsmononerdfont",
		"Cantarell-BoldItalic.ttf":         "cantarell",
		"Inter-Regular.ttf":                "inter",
		"JetBrainsMonoNerdFont-Italic.ttf": "jetbrainsmononerdfont",
	}
	for in, want := range cases {
		if got := familyKey(in); got != want {
			t.Errorf("familyKey(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestUnknownFont(t *testing.T) {
	if _, err := Get("definitely-not-a-real-font-family"); err == nil {
		t.Fatal("expected error")
	}
}
