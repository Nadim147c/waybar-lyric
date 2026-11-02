// Package yrc parses the word-timed lyric format used by NetEase Cloud Music.
//
// A lyric line looks like
//
//	[27360,1290](27360,240,0)I've (27600,90,0)been (27690,360,0)tryna
//
// The brackets hold "start,duration" of the whole line and every parenthesis
// holds "start,duration,flag" of the text that follows it. All numbers are
// milliseconds. Credit lines are JSON objects and carry no timing.
package yrc

import (
	"bufio"
	"cmp"
	"encoding/json"
	"io"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/Nadim147c/waybar-lyric/internal/lyric/models"
)

func ParseText(text string) (models.Lines, error) {
	return Parse(strings.NewReader(text))
}

func Parse(r io.Reader) (models.Lines, error) {
	scanner := bufio.NewScanner(r)

	lyrics := make(models.Lines, 1) // add empty line at the start of the lyrics
	for scanner.Scan() {
		line, ok := parseLine(strings.TrimSpace(scanner.Text()))
		if !ok {
			continue
		}
		lyrics = append(lyrics, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	slices.SortStableFunc(lyrics[1:], func(a, b models.Line) int {
		return cmp.Compare(a.Timestamp, b.Timestamp)
	})

	if len(lyrics) == 1 {
		return nil, models.ErrLyricsNotSynced
	}

	return lyrics, nil
}

// parseLine reads one "[start,span](at,length,0)text..." line. It reports
// false for credit lines and anything else that is not a timed lyric line.
func parseLine(line string) (models.Line, bool) {
	rest, ok := strings.CutPrefix(line, "[")
	if !ok {
		return models.Line{}, false
	}
	header, rest, ok := strings.Cut(rest, "]")
	if !ok {
		return models.Line{}, false
	}
	start, ok := parseHeader(header)
	if !ok {
		return models.Line{}, false
	}

	var (
		words []models.Word
		text  strings.Builder
	)

	// grow adds text that has no timestamp of its own to the previous word.
	grow := func(tail string) {
		if tail == "" {
			return
		}
		text.WriteString(tail)
		if n := len(words); n > 0 {
			words[n-1].Text += tail
		}
	}

	for {
		open := strings.IndexByte(rest, '(')
		if open < 0 {
			break
		}
		tail := rest[open+1:]
		shut := strings.IndexByte(tail, ')')
		if shut < 0 {
			break
		}

		at, length, ok := parseStamp(tail[:shut])
		if !ok {
			// Not a timestamp, just text in parentheses like "(la)".
			grow(rest[:open+shut+2])
			rest = tail[shut+1:]
			continue
		}

		grow(rest[:open])
		rest = tail[shut+1:]

		spoken := rest
		if next := strings.IndexByte(rest, '('); next >= 0 {
			spoken = rest[:next]
		}
		words = append(words, models.Word{
			Start: at,
			End:   at + length,
			Text:  spoken,
		})
		text.WriteString(spoken)
		rest = rest[len(spoken):]
	}
	grow(rest)

	lineText := strings.TrimSpace(text.String())
	if lineText == "" {
		return models.Line{}, false
	}

	return models.Line{
		Timestamp: start,
		Text:      lineText,
		Words:     withSpaces(words),
	}, true
}

// withSpaces trims every word and puts the whitespace between them back as
// untimed " " words, the same shape lrc.Parse produces. Words that were not
// separated by whitespace in the source (CJK lyrics) stay adjacent.
func withSpaces(raw []models.Word) []models.Word {
	if len(raw) == 0 {
		return nil
	}

	words := make([]models.Word, 0, len(raw)*2)
	gap := func() {
		if n := len(words); n > 0 && words[n-1].Start != -1 {
			words = append(words, models.Word{Start: -1, End: -1, Text: " "})
		}
	}

	for _, w := range raw {
		spoken := w.Text
		w.Text = strings.TrimSpace(spoken)
		if w.Text == "" {
			gap()
			continue
		}
		if strings.TrimLeftFunc(spoken, unicode.IsSpace) != spoken {
			gap()
		}
		words = append(words, w)
		if strings.TrimRightFunc(spoken, unicode.IsSpace) != spoken {
			gap()
		}
	}

	if n := len(words); n > 0 && words[n-1].Start == -1 {
		words = words[:n-1]
	}
	return words
}

// parseHeader reads the "start,span" part of a line and returns the start.
// The Go models have no line end time, so the span is only checked.
func parseHeader(header string) (time.Duration, bool) {
	startStr, spanStr, ok := strings.Cut(header, ",")
	if !ok {
		return 0, false
	}
	start, ok := parseMillis(startStr)
	if !ok {
		return 0, false
	}
	if _, ok := parseMillis(spanStr); !ok {
		return 0, false
	}
	return start, true
}

// parseStamp reads the "start,length,flag" part of a word. The flag is
// required to be there but is not used.
func parseStamp(stamp string) (at, length time.Duration, ok bool) {
	parts := strings.Split(stamp, ",")
	if len(parts) < 3 {
		return 0, 0, false
	}
	if at, ok = parseMillis(parts[0]); !ok {
		return 0, 0, false
	}
	if length, ok = parseMillis(parts[1]); !ok {
		return 0, 0, false
	}
	return at, length, true
}

func parseMillis(s string) (time.Duration, bool) {
	n, err := strconv.ParseUint(strings.TrimSpace(s), 10, 32)
	if err != nil {
		return 0, false
	}
	return time.Duration(n) * time.Millisecond, true
}

type credit struct {
	C []struct {
		Tx string `json:"tx"`
	} `json:"c"`
}

// Writers returns the lyricists and composers named in the JSON credit lines
// at the top of a YRC lyric, without duplicates.
func Writers(text string) []string {
	var writers []string
	for line := range strings.SplitSeq(text, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "{") {
			continue
		}

		var c credit
		if err := json.Unmarshal([]byte(line), &c); err != nil {
			continue
		}

		var sb strings.Builder
		for _, piece := range c.C {
			sb.WriteString(piece.Tx)
		}
		joined := sb.String()

		label, names, ok := strings.Cut(joined, ":")
		if !ok {
			label, names, ok = strings.Cut(joined, "：")
		}
		if !ok {
			continue
		}
		if !strings.Contains(label, "作词") && !strings.Contains(label, "作曲") {
			continue
		}

		for name := range strings.SplitSeq(names, "/") {
			name = strings.TrimSpace(name)
			if name != "" && !slices.Contains(writers, name) {
				writers = append(writers, name)
			}
		}
	}
	return writers
}
