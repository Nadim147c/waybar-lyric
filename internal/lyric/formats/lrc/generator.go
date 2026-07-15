package lrc

import (
	"bytes"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"

	"github.com/Nadim147c/waybar-lyric/internal/lyric/models"
)

func formatDuration(w io.Writer, d time.Duration) {
	mm := d / time.Minute
	sec := d % time.Minute
	ss := sec / time.Second
	mil := sec % time.Second
	xx := mil / oneHundredthOfSecond
	fmt.Fprintf(w, "%.2d:%.2d.%.2d", mm, ss, xx) //nolint
}

func joinWords(words []models.Word) (models.Word, bool) {
	if len(words) == 0 {
		return models.Word{}, false
	}

	var sb strings.Builder
	for _, word := range words {
		sb.WriteString(word.Text)
	}

	return models.Word{
		Start: words[0].Start,
		End:   words[len(words)-1].End,
		Text:  sb.String(),
	}, true
}

func mergeSyllables(words []models.Word) []models.Word {
	res := make([]models.Word, 0, len(words)/2)
	var i int
	for i < len(words) {
		j := slices.IndexFunc(words[i:], func(s models.Word) bool {
			return s.IsSeparator()
		})
		if j == 0 {
			i++
			continue
		}
		if j < 0 {
			w, ok := joinWords(words[i:])
			if ok {
				res = append(res, w)
			}
			break
		}
		w, ok := joinWords(words[i : i+j])
		if ok {
			res = append(res, w)
		}
		i += j
	}
	return res
}

func Render(lyrics models.Lyrics) []byte {
	buf := bytes.NewBuffer(nil)
	m := lyrics.Metadata
	if m != nil {
		if m.Title != "" {
			fmt.Fprintf(buf, "[ti:%s]\n", m.Title)
		}
		if m.Artist != "" {
			fmt.Fprintf(buf, "[ar:%s]\n", m.Artist)
		}
		if m.Artist != "" {
			fmt.Fprintf(buf, "[al:%s]\n", m.Album)
		}
		buf.WriteByte('\n')
	}

	for _, line := range lyrics.Lines {
		buf.WriteByte('[')
		formatDuration(buf, line.Timestamp)
		buf.WriteByte(']')

		if len(line.Words) == 0 {
			buf.WriteString(line.Text)
		} else {
			for _, word := range mergeSyllables(line.Words) {
				buf.WriteByte(' ')
				buf.WriteByte('<')
				formatDuration(buf, word.Start)
				buf.WriteByte('>')
				buf.WriteByte(' ')
				buf.WriteString(word.Text)
			}
		}

		buf.WriteByte('\n')
	}

	return buf.Bytes()
}
