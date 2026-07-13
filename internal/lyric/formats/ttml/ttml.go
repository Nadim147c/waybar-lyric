package ttml

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/Nadim147c/waybar-lyric/internal/lyric/models"
	"golang.org/x/net/html"
)

var ErrBodyNotFound = errors.New("body not found")

func GetTextLength(s string) (time.Duration, error) {
	return GetLength(strings.NewReader(s))
}

func GetLength(r io.Reader) (time.Duration, error) {
	node, err := html.Parse(r)
	if err != nil {
		return 0, err
	}

	body := getElemens(func(n *html.Node) bool { return n.Data == "body" }, node)
	if len(body) != 1 {
		return 0, ErrBodyNotFound
	}
	i := slices.IndexFunc(body[0].Attr, func(a html.Attribute) bool { return a.Key == "dur" })
	if i < 0 {
		return 0, ErrBodyNotFound
	}
	return parseTtmlTimestamp(body[0].Attr[i].Val)
}

func ParseText(s string) (models.Lines, error) {
	return Parse(strings.NewReader(s))
}

func Parse(r io.Reader) (models.Lines, error) {
	node, err := html.Parse(r)
	if err != nil {
		return nil, err
	}

	// we always add a empty line in front
	lines := make(models.Lines, 1)

	ps := getElemens(filterLines, node)
	for _, p := range ps {
		start, _, err := getTimestamps(p.Attr)
		if err != nil {
			continue
		}

		if isLineLevelSynced(p) {
			lines = append(lines, models.Line{
				Timestamp: start,
				Text:      p.FirstChild.Data,
				Words:     nil,
			})
			continue
		}

		line := collectLineWords(p, start)
		lines = append(lines, line)
	}

	if len(lines) == 1 {
		return lines, fmt.Errorf("lyrics lines not found: %w", models.ErrLyricsNotFound)
	}

	return lines, nil
}

func collectLineWords(parentNode *html.Node, lineStart time.Duration) models.Line {
	var buf bytes.Buffer

	words := make([]models.Word, 0, 5)
	for node := parentNode.FirstChild; node != nil; node = node.NextSibling {
		if node.Type == html.TextNode {
			words = append(words, models.Word{
				Start: -1, End: -1,
				Text: node.Data,
			})
			buf.WriteString(node.Data)
			continue
		}

		start, end, err := getTimestamps(node.Attr)
		if err != nil {
			continue
		}

		if isBackground(node) {
			line := collectLineWords(node, start)
			buf.WriteString(line.Text)
			words = append(words, line.Words...)
			continue
		}

		text := node.FirstChild.Data

		words = append(words, models.Word{
			Start: start, End: end,
			Text: text,
		})
		buf.WriteString(text)
	}

	return models.Line{
		Timestamp: lineStart,
		Text:      buf.String(),
		Words:     words,
	}
}

func getTimestamps(attrs []html.Attribute) (start, end time.Duration, err error) {
	var startFound, endFound bool
	for _, a := range attrs {
		if a.Key == "begin" {
			start, err = parseTtmlTimestamp(a.Val)
			if err != nil {
				return
			}
			startFound = true
		}
		if a.Key == "end" {
			end, err = parseTtmlTimestamp(a.Val)
			if err != nil {
				return
			}
			endFound = true
		}
	}
	if startFound && endFound {
		return start, end, nil
	}
	return 0, 0, fmt.Errorf("start or end not found: start=%v end=%v", startFound, endFound)
}

func parseTtmlTimestamp(s string) (time.Duration, error) {
	minStr, secStr, ok := strings.Cut(s, ":")
	if ok {
		min, err := strconv.ParseFloat(minStr, 64)
		if err != nil {
			return 0, err
		}
		sec, err := strconv.ParseFloat(secStr, 64)
		if err != nil {
			return 0, err
		}
		return time.Duration(min*float64(time.Minute) + sec*float64(time.Second)), nil
	}

	sec, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}
	return time.Duration(sec * float64(time.Second)), nil
}

func isLineLevelSynced(node *html.Node) bool {
	n := node.FirstChild
	return n != nil &&
		n.Type == html.TextNode &&
		n.NextSibling == nil
}

func filterLines(node *html.Node) bool {
	return node.Type == html.ElementNode && node.Data == "p"
}

func isBackground(n *html.Node) bool {
	if n == nil {
		return false
	}
	return slices.ContainsFunc(n.Attr, func(a html.Attribute) bool {
		return a.Key == "ttm:role" && a.Val == "x-bg"
	})
}

func getElemens(pred func(*html.Node) bool, nodes ...*html.Node) []*html.Node {
	var res []*html.Node

	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node == nil {
			return
		}
		if pred(node) {
			res = append(res, node)
			return
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}

	for _, node := range nodes {
		walk(node)
	}

	return res
}
