package match

import (
	"strings"

	"github.com/texttheater/golang-levenshtein/levenshtein"
)

// Strings calculates a similarity score between two strings. Returns
// 1.0 for an exact match, 0.0 if the distance is greater than the threshold,
// and a scaled value between 0.0 and 1.0 otherwise.
func Strings(a, b string) float64 {
	if a == b {
		return 1.0
	}

	distance := levenshtein.DistanceForStrings(
		[]rune(a), []rune(b),
		levenshtein.DefaultOptions,
	)
	maxLen := max(len(a), len(b))
	threshold := maxLen / 2

	if distance > threshold {
		return 0.0
	}

	score := 1.0 - float64(distance)/float64(threshold)

	if strings.HasPrefix(a, b) || strings.HasPrefix(b, a) {
		score += 0.1
	}

	if score > 1.0 {
		score = 1.0
	}

	return score
}
