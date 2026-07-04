package fonts

import (
	"strings"

	"github.com/rivo/uniseg"
)

// emojiPresentation holds the ranges of runes with the Unicode
// Emoji_Presentation property — characters drawn as color emoji by default.
//
// Generated from the Unicode Character Database (emoji-data.txt,
// https://unicode.org/Public/UCD/latest/ucd/emoji/emoji-data.txt).
var emojiPresentation = [...][2]rune{
	{0x231A, 0x231B},
	{0x23E9, 0x23EC},
	{0x23F0, 0x23F0},
	{0x23F3, 0x23F3},
	{0x25FD, 0x25FE},
	{0x2614, 0x2615},
	{0x2648, 0x2653},
	{0x267F, 0x267F},
	{0x2693, 0x2693},
	{0x26A1, 0x26A1},
	{0x26AA, 0x26AB},
	{0x26BD, 0x26BE},
	{0x26C4, 0x26C5},
	{0x26CE, 0x26CE},
	{0x26D4, 0x26D4},
	{0x26EA, 0x26EA},
	{0x26F2, 0x26F3},
	{0x26F5, 0x26F5},
	{0x26FA, 0x26FA},
	{0x26FD, 0x26FD},
	{0x2705, 0x2705},
	{0x270A, 0x270B},
	{0x2728, 0x2728},
	{0x274C, 0x274C},
	{0x274E, 0x274E},
	{0x2753, 0x2755},
	{0x2757, 0x2757},
	{0x2795, 0x2797},
	{0x27B0, 0x27B0},
	{0x27BF, 0x27BF},
	{0x2B1B, 0x2B1C},
	{0x2B50, 0x2B50},
	{0x2B55, 0x2B55},
	{0x1F004, 0x1F004},
	{0x1F0CF, 0x1F0CF},
	{0x1F18E, 0x1F18E},
	{0x1F191, 0x1F19A},
	{0x1F1E6, 0x1F1FF},
	{0x1F201, 0x1F201},
	{0x1F21A, 0x1F21A},
	{0x1F22F, 0x1F22F},
	{0x1F232, 0x1F236},
	{0x1F238, 0x1F23A},
	{0x1F250, 0x1F251},
	{0x1F300, 0x1F320},
	{0x1F32D, 0x1F335},
	{0x1F337, 0x1F37C},
	{0x1F37E, 0x1F393},
	{0x1F3A0, 0x1F3CA},
	{0x1F3CF, 0x1F3D3},
	{0x1F3E0, 0x1F3F0},
	{0x1F3F4, 0x1F3F4},
	{0x1F3F8, 0x1F43E},
	{0x1F440, 0x1F440},
	{0x1F442, 0x1F4FC},
	{0x1F4FF, 0x1F53D},
	{0x1F54B, 0x1F54E},
	{0x1F550, 0x1F567},
	{0x1F57A, 0x1F57A},
	{0x1F595, 0x1F596},
	{0x1F5A4, 0x1F5A4},
	{0x1F5FB, 0x1F64F},
	{0x1F680, 0x1F6C5},
	{0x1F6CC, 0x1F6CC},
	{0x1F6D0, 0x1F6D2},
	{0x1F6D5, 0x1F6D8},
	{0x1F6DC, 0x1F6DF},
	{0x1F6EB, 0x1F6EC},
	{0x1F6F4, 0x1F6FC},
	{0x1F7E0, 0x1F7EB},
	{0x1F7F0, 0x1F7F0},
	{0x1F90C, 0x1F93A},
	{0x1F93C, 0x1F945},
	{0x1F947, 0x1F9FF},
	{0x1FA70, 0x1FA7C},
	{0x1FA80, 0x1FA8A},
	{0x1FA8E, 0x1FAC6},
	{0x1FAC8, 0x1FAC8},
	{0x1FACD, 0x1FADC},
	{0x1FADF, 0x1FAEA},
	{0x1FAEF, 0x1FAF8},
}

func emojiPresentationRune(r rune) bool {
	lo, hi := 0, len(emojiPresentation)-1
	for lo <= hi {
		mid := (lo + hi) / 2
		switch rg := emojiPresentation[mid]; {
		case r < rg[0]:
			hi = mid - 1
		case r > rg[1]:
			lo = mid + 1
		default:
			return true
		}
	}
	return false
}

const (
	vs15   = 0xFE0E // variation selector 15: force text presentation
	vs16   = 0xFE0F // variation selector 16: force emoji presentation
	keycap = 0x20E3 // combining enclosing keycap
)

// IsEmoji reports whether a single grapheme cluster is drawn as emoji:
// either a default emoji-presentation character (😀, 🇺🇸, 👨‍👩‍👧‍👦) or a
// sequence carrying an emoji variation selector or keycap (❤️, 1️⃣).
func IsEmoji(cluster string) bool {
	emoji := false
	for _, r := range cluster {
		switch r {
		case vs15:
			return false
		case vs16, keycap:
			emoji = true
		default:
			if emojiPresentationRune(r) {
				emoji = true
			}
		}
	}
	return emoji
}

// Run is a piece of a string: either plain text or one emoji cluster.
type Run struct {
	Text  string
	Emoji bool
}

// SplitEmoji splits s into runs of plain text and single emoji clusters.
func SplitEmoji(s string) []Run {
	if !mayContainEmoji(s) {
		return []Run{{Text: s}}
	}
	var runs []Run
	var plain strings.Builder
	flush := func() {
		if plain.Len() > 0 {
			runs = append(runs, Run{Text: plain.String()})
			plain.Reset()
		}
	}
	g := uniseg.NewGraphemes(s)
	for g.Next() {
		cluster := g.Str()
		if IsEmoji(cluster) {
			flush()
			runs = append(runs, Run{Text: cluster, Emoji: true})
		} else {
			plain.WriteString(cluster)
		}
	}
	flush()
	return runs
}

// mayContainEmoji is a cheap pre-check: every emoji cluster contains at
// least one rune at or above U+20E3.
func mayContainEmoji(s string) bool {
	for _, r := range s {
		if r >= keycap {
			return true
		}
	}
	return false
}
