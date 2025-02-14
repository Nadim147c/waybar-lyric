package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

func SaveCache(lines []LyricLine, filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, line := range lines {
		_, err := fmt.Fprintf(file, "%d,%s\n", line.Timestamp, line.Text)
		if err != nil {
			return err
		}
	}
	return nil
}

func LoadCache(filePath string) ([]LyricLine, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lyrics []LyricLine
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ",", 2)
		if len(parts) != 2 {
			continue // Skip invalid lines
		}

		timestamp, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			return nil, err
		}

		lyric := LyricLine{
			Timestamp: time.Duration(timestamp),
			Text:      parts[1],
		}

		lyrics = append(lyrics, lyric)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if len(lyrics) == 0 {
		return nil, fmt.Errorf("Number of line found is zero.")
	}

	return lyrics, nil
}
