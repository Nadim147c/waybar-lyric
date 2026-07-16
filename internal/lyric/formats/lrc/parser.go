package lrc

import (
	"bufio"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"

	"github.com/Nadim147c/waybar-lyric/internal/lyric/models"
)

func ParseText(text string) (models.Lines, error) {
	return Parse(strings.NewReader(text))
}

func Parse(r io.Reader) (models.Lines, error) {
	scanner := bufio.NewScanner(r)

	lyrics := make(models.Lines, 1) // add empty line a start of the lyrics
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var timestamps []time.Duration
		remaining := line
		for {
			idx, ts := getNextSquareTimestamp(remaining)
			if idx == 0 {
				timestamps = append(timestamps, ts)
				remaining = strings.TrimSpace(remaining[tsLen:])
			} else {
				break
			}
		}

		if len(timestamps) == 0 {
			continue
		}

		var words []models.Word
		temp := remaining

		lastTs := timestamps[0]

		for {
			idx, currentTs := getNextAngleTimestamp(temp)
			if idx == -1 {
				if len(words) > 0 && temp != "" {
					words[len(words)-1].Text += temp
				}
				break
			}
			before := strings.TrimSpace(temp[:idx])
			if before == "" {
				lastTs = currentTs
				continue
			}

			words = append(words, models.Word{
				Start: lastTs,
				End:   currentTs,
				Text:  before,
			})
			temp = temp[idx+tsLen:]
		}

		withSpace := make([]models.Word, len(words)*2)
		var sb strings.Builder
		if len(words) != 0 {
			withSpace = append(withSpace, words[0])
			for _, word := range words[1:] {
				withSpace = append(withSpace, models.Word{
					Start: -1, End: -1,
					Text: " ",
				})
				sb.WriteByte(' ')

				word.Text = strings.TrimSpace(word.Text)
				withSpace = append(withSpace, word)
				sb.WriteString(word.Text)
			}
		} else {
			sb.WriteString(remaining)
		}

		for _, ts := range timestamps {
			lyrics = append(lyrics, models.Line{
				Timestamp: ts,
				Text:      remaining,
				Words:     slices.Clone(withSpace),
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	slices.SortStableFunc(lyrics[1:], func(a, b models.Line) int {
		if a.Timestamp < b.Timestamp {
			return -1
		}
		if a.Timestamp > b.Timestamp {
			return 1
		}
		return 0
	})

	if len(lyrics) == 1 {
		return nil, models.ErrLyricsNotSynced
	}

	return lyrics, nil
}

const tsLen = len("[mm:ss.xx]")

const oneHundredthOfSecond = time.Second / 100

func getNextTimestamp(start, end byte, s string) (int, time.Duration) {
	startIndex := strings.IndexByte(s, start)
	if startIndex < 0 {
		return -1, 0
	}
	endIndex := strings.IndexByte(s[startIndex:], end)
	if endIndex < 0 {
		return -1, 0
	}
	if endIndex+1 != tsLen {
		return -1, 0
	}
	var mm, ss, xx time.Duration
	_, err := fmt.Sscanf(s[startIndex+1:startIndex+endIndex], "%d:%d.%d", &mm, &ss, &xx)
	if err != nil {
		return -1, 0
	}

	return startIndex, mm*time.Minute + ss*time.Second + xx*oneHundredthOfSecond
}

func getNextAngleTimestamp(s string) (int, time.Duration) {
	const start, end byte = '<', '>'
	return getNextTimestamp(start, end, s)
}

func getNextSquareTimestamp(s string) (int, time.Duration) {
	const start, end byte = '[', ']'
	return getNextTimestamp(start, end, s)
}
