package code

import (
	"testing"
)

func TestUnitsKeepEmojiClustersWhole(t *testing.T) {
	got := units("a👨‍👩‍👧‍👦b")
	want := []string{"a", "👨‍👩‍👧‍👦", "b"}
	if len(got) != len(want) {
		t.Fatalf("units = %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("unit %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestRenderWithEmoji(t *testing.T) {
	src := "fmt.Println(\"deploy 🚀 done ✅ 👨‍👩‍👧‍👦\")"
	img, err := New(src).WithLanguage("go").Render()
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Empty() {
		t.Fatal("empty image")
	}
}

func TestRenderWrappedEmojiDoesNotError(t *testing.T) {
	src := "// 😀😀😀😀😀😀😀😀😀😀😀😀😀😀😀😀😀😀😀😀"
	img, err := New(src).WithLanguage("go").WithMaxWidth(120).Render()
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() > 120 {
		t.Errorf("image width %d exceeds max width 120", img.Bounds().Dx())
	}
}
