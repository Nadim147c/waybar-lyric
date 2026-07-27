package lyric

import (
	"math"

	"github.com/Nadim147c/waybar-lyric/internal/lyric/models"
	"github.com/Nadim147c/waybar-lyric/internal/lyric/provider"
	"github.com/Nadim147c/waybar-lyric/internal/match"
)

func matchLines(a, b models.Lines) float64 {
	lenA := len(a)
	lenB := len(b)

	if lenA == 0 && lenB == 0 {
		return 0.0
	}
	if lenA == 0 || lenB == 0 {
		return 0.0
	}

	dp := make([][]float64, lenA+1)
	for i := range dp {
		dp[i] = make([]float64, lenB+1)
	}

	for i := 1; i <= lenA; i++ {
		for j := 1; j <= lenB; j++ {
			scoreMatch := dp[i-1][j-1] +
				match.Strings(a[i-1].Text, b[j-1].Text) +
				match.Durations(a[i-1].Timestamp, b[j-1].Timestamp)

			scoreSkipA := dp[i-1][j]
			scoreSkipB := dp[i][j-1]
			dp[i][j] = math.Max(scoreMatch, math.Max(scoreSkipA, scoreSkipB))
		}
	}

	maxScore := dp[lenA][lenB] * 0.75

	maxPossibleMatches := max(float64(lenA), float64(lenB))

	return maxScore / maxPossibleMatches
}

// filterOutliers compares every slice against all other slices using SliceSimilarity.
// It keeps only slices whose average similarity to the rest meets or exceeds minAvgSimilarity.
func filterOutliers(slices []provider.Result, minAvgSimilarity float64) []provider.Result {
	n := len(slices)

	// If there are 0, 1, or 2 slices, filtering by relative comparison doesn't apply
	if n <= 2 {
		return slices
	}

	// matrix[i][j] holds the similarity between slices[i] and slices[j]
	matrix := make([][]float64, n)
	for i := range matrix {
		matrix[i] = make([]float64, n)
	}

	for i := range n {
		matrix[i][i] = 1.0 // A slice is identical to itself
		for j := i + 1; j < n; j++ {
			sim := matchLines(slices[i].Lyrics.Lines, slices[j].Lyrics.Lines)
			matrix[i][j] = sim
			matrix[j][i] = sim
		}
	}

	var filtered []provider.Result

	for i := range n {
		var totalSim float64
		for j := range n {
			if i != j {
				totalSim += matrix[i][j]
			}
		}

		avgSim := totalSim / float64(n-1)

		if avgSim >= minAvgSimilarity {
			filtered = append(filtered, slices[i])
		}
	}

	return filtered
}
