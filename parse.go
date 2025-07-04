package main

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
)

// ParseLyrics parses a string containing time-synchronized lyrics in the format [MM:SS.ss]Lyric text
// and returns a slice of LyricLine structs. Each line in the input should follow the format
// "[timestamp]lyric text", where timestamp is in a format parseable by ParseTimestamp.
// Empty lines and malformed lines are skipped.
func ParseLyrics(file string) ([]LyricLine, error) {
	lyrics := []LyricLine{{}} // add empty line a start of the lyrics
	for line := range strings.SplitSeq(file, "\n") {
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "]", 2)
		if len(parts) != 2 {
			continue
		}

		timestampStr := strings.TrimPrefix(parts[0], "[")
		lyricLine := strings.TrimSpace(parts[1])

		timestamp, err := ParseTimestamp(timestampStr)
		if err != nil {
			continue
		}

		lyric := LyricLine{Timestamp: timestamp, Text: lyricLine}
		lyrics = append(lyrics, lyric)
	}

	return lyrics, nil
}

// ParseTimestamp converts a timestamp string (in "HH:MM:SS", "MM:SS" or "SS" format)
// into a time.Duration value representing the total number of nanoseconds.
// Example inputs: "1:30:45" (1h 30m 45s), "5:20" (5m 20s), "42" (42s)
func ParseTimestamp(ts string) (time.Duration, error) {
	durationConst := []time.Duration{time.Second, time.Minute, time.Hour}

	var duration time.Duration

	parts := strings.Split(ts, ":")
	if len(parts) > 3 {
		return 0, fmt.Errorf("invalid timestamp: %s", ts)
	}

	// Reverse parts to process seconds first, then minutes, then hours
	// This allows us to handle variable-length timestamps (SS, MM:SS, HH:MM:SS)
	slices.Reverse(parts)

	for i, part := range parts {
		num, err := strconv.ParseFloat(strings.TrimSpace(part), 64)
		if err != nil || num < 0 {
			return 0, fmt.Errorf("invalid timestamp part: %s", parts[i])
		}

		duration += time.Duration(num * float64(durationConst[i]))
	}

	return duration, nil
}
