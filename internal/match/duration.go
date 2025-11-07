package match

import "time"

// Durations return duration similarity ration between two duration for given
// threshold.
func Durations(a, b time.Duration) float64 {
	const threshold = 8 * time.Second
	diff := (a - b).Abs()
	if diff >= threshold {
		return 0
	}
	return 1 - float64(diff)/float64(threshold)
}
