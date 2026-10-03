package lyricsfile

import (
	"errors"
	"io"
	"strings"
	"time"

	"github.com/Nadim147c/waybar-lyric/internal/lyric/models"
	"gopkg.in/yaml.v3"
)

type Lyrics struct {
	Version  string   `json:"version"`
	Metadata Metadata `json:"metadata"`
	Plain    *string  `json:"plain,omitempty"`
	Lines    []Line   `json:"lines,omitempty"`
}

type Word struct {
	Text    string `json:"text"`
	StartMS int64  `json:"start_ms"`
	EndMS   int64  `json:"end_ms"`
}

type Line struct {
	Text    string `json:"text"`
	StartMS int64  `json:"start_ms"`
	EndMS   int64  `json:"end_ms"`
	Words   []Word `json:"words,omitempty"`
}

type Metadata struct {
	Title        string  `json:"title"`
	Artist       string  `json:"artist"`
	DurationMS   *int64  `json:"duration_ms,omitempty"`
	Language     *string `json:"language,omitempty"`
	Instrumental *bool   `json:"instrumental,omitempty"`
}

func ParseText(text string) (models.Lines, error) {
	return Parse(strings.NewReader(text))
}

func Parse(r io.Reader) (models.Lines, error) {
	var data Lyrics

	err := yaml.NewDecoder(r).Decode(&data)
	if errors.Is(err, io.EOF) {
		return nil, models.ErrLyricsNotSynced
	}
	if err != nil {
		return nil, err
	}

	if len(data.Lines) == 0 {
		return nil, models.ErrLyricsNotSynced
	}

	lines := make(models.Lines, 0, len(data.Lines))
	for _, line := range data.Lines {
		var finalLine models.Line
		finalLine.Text = line.Text
		finalLine.Timestamp = time.Duration(line.StartMS) * time.Millisecond
		if len(line.Words) != 0 {
			words := make([]models.Word, 0, (len(line.Words)*2)-1)
			for i, w := range line.Words {
				if i != 0 {
					words = append(words, models.Word{
						Text:  " ",
						Start: -1,
						End:   -1,
					})
				}
				words = append(words, models.Word{
					Text:  w.Text,
					Start: time.Duration(w.StartMS) * time.Millisecond,
					End:   time.Duration(w.EndMS) * time.Millisecond,
				})
			}
			finalLine.Words = words
		}
		lines = append(lines, finalLine)
	}

	return lines, nil
}
