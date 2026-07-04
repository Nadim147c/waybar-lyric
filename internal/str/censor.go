package str

import (
	_ "embed"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/Nadim147c/waybar-lyric/internal/config"
)

//go:embed profanity.txt
var profanity string

var profanityRe *regexp.Regexp

func profanityRegex() *regexp.Regexp {
	if profanityRe != nil {
		return profanityRe
	}
	profanity = strings.TrimRightFunc(profanity, unicode.IsSpace)
	idx := strings.LastIndex(profanity, "\n")
	line := profanity[idx+1:]
	profanityRe = regexp.MustCompile(`(?i)\b(` + line + `)\b`)
	return profanityRe
}

func CensorText(input string) string {
	re := profanityRegex()
	input = re.ReplaceAllStringFunc(input, func(match string) string {
		switch config.FilterProfanityType {
		case "full":
			return strings.Repeat("*", len(match))
		case "partial":
			return partialCensor(match)
		default:
			return match
		}
	})
	return input
}

func partialCensor(word string) string {
	if len(word) == 0 {
		return ""
	}

	firstRune, firstSize := utf8.DecodeRuneInString(word)
	runeCount := utf8.RuneCountInString(word)

	if runeCount <= 3 {
		return strings.Repeat("*", runeCount)
	}

	lastRune, lastSize := utf8.DecodeLastRuneInString(word)

	asteriskCount := runeCount - 2

	var sb strings.Builder
	sb.Grow(firstSize + asteriskCount + lastSize)

	sb.WriteRune(firstRune)
	for range asteriskCount {
		sb.WriteByte('*')
	}
	sb.WriteRune(lastRune)

	return sb.String()
}
