package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

func parseLyrics(file string) ([]LyricLine, error) {
	var lyrics []LyricLine
	lines := strings.Split(file, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "]", 2)
		if len(parts) != 2 {
			continue
		}
		timestampStr := strings.TrimPrefix(parts[0], "[")
		lyricLine := strings.TrimSpace(parts[1])

		timestamp, err := parseTimestamp(timestampStr)
		if err != nil {
			continue
		}

		lyric := LyricLine{
			Timestamp: timestamp,
			Text:      lyricLine,
		}

		lyrics = append(lyrics, lyric)
	}
	return lyrics, nil
}

func parseTimestamp(ts string) (time.Duration, error) {
	parts := strings.Split(ts, ":")

	var seconds time.Duration

	for i := len(parts) - 1; i >= 0; i-- {
		part, err := strconv.ParseFloat(strings.TrimSpace(parts[i]), 64)
		if err != nil {
			return 0, fmt.Errorf("invalid timestamp part: %s", parts[i])
		}

		seconds += time.Duration(part * math.Pow(60, float64(len(parts)-1-i)) * float64(time.Second))
	}

	return seconds, nil
}
