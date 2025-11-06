package str

import (
	_ "embed"
	"regexp"
	"strings"
	"unicode"

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
	chars := []rune(word)
	if len(chars) <= 3 {
		return strings.Repeat("*", len(word))
	}
	return string(
		chars[0],
	) + strings.Repeat(
		"*",
		len(word)-2,
	) + string(
		chars[len(word)-1],
	)
}
