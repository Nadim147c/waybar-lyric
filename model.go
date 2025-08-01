package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/Nadim147c/go-mpris"
)

// LrcLibResponse is the response sent from LrcLib api
type LrcLibResponse struct {
	ID           int     `json:"id"`
	Name         string  `json:"name"`
	TrackName    string  `json:"trackName"`
	ArtistName   string  `json:"artistName"`
	AlbumName    string  `json:"albumName"`
	Duration     float64 `json:"duration"`
	Instrumental bool    `json:"instrumental"`
	PlainLyrics  string  `json:"plainLyrics"`
	SyncedLyrics string  `json:"syncedLyrics"`
}

// LyricLine is a line of synchronized lyrics
type LyricLine struct {
	Timestamp time.Duration `json:"time"`
	Text      string        `json:"line"`

	// Active is used for detailed context
	Active bool `json:"active"`
}

func (l LyricLine) MarshalJSON() ([]byte, error) {
	type Alias LyricLine
	return json.Marshal(&struct {
		Alias
		Timestamp float64 `json:"time"`
	}{
		Alias:     (Alias)(l),
		Timestamp: l.Timestamp.Seconds(),
	})
}

// Lyrics is a slice of LyricLine
type Lyrics []LyricLine

// Status is the alt/class for waybar
type Status string

const (
	Music   Status = "music"
	Lyric   Status = "lyric"
	Playing Status = "playing"
	Paused  Status = "paused"
	NoLyric Status = "no_lyric"
)

type Class []Status

type Waybar struct {
	Text       string       `json:"text"`
	Class      Class        `json:"class"`
	Alt        Status       `json:"alt"`
	Tooltip    string       `json:"tooltip"`
	Percentage int          `json:"percentage"`
	Info       *PlayerInfo  `json:"info,omitempty"`
	Context    *[]LyricLine `json:"context,omitempty"`
}

func NewWaybar(lyrics []LyricLine, idx int) *Waybar {
	lyric := lyrics[idx]
	start := max(idx-2, 0)
	end := min(idx+TooltipLines-2, len(lyrics))

	lyricsContext := slices.Clone(lyrics[start:end])

	var tooltip strings.Builder

	tooltip.WriteString(fmt.Sprintf("<span foreground=\"%s\">", TooltipColor))

	for i, ttl := range lyricsContext {
		line := BreakLine(ttl.Text, BreakTooltip)
		if ttl.Text == "" {
			line = "󰝚 "
		}

		if start+i == idx {
			lyricsContext[i].Active = true
			newLine := fmt.Sprintf("</span><b><big>%s</big></b>\n<span foreground=\"%s\">", line, TooltipColor)
			tooltip.WriteString(newLine)
			continue
		}

		tooltip.WriteString(line + "\n")
	}

	line := truncate(lyric.Text)
	tt := strings.TrimSpace(tooltip.String()) + "</span>"

	class := Class{Lyric, Playing}
	waybar := &Waybar{Alt: Lyric, Class: class, Text: line, Tooltip: tt}

	if Detailed {
		waybar.Context = &lyricsContext
	}

	return waybar
}

var JSON = json.NewEncoder(os.Stdout)

func init() {
	JSON.SetEscapeHTML(false)
}

func (w *Waybar) Encode() {
	if LyricOnly && (w.Alt == Paused || w.Alt == Music) {
		fmt.Println("")
		return
	}

	if Compact {
		fmt.Println(w.Text)
		return
	}

	if w == (&Waybar{}) {
		fmt.Println("{}")
	}

	JSON.Encode(w)
}

func (w *Waybar) Is(other *Waybar) bool {
	if w == other {
		return true
	}
	if other == nil {
		return false
	}
	if w.Text != other.Text ||
		w.Alt != other.Alt ||
		w.Tooltip != other.Tooltip ||
		w.Percentage != other.Percentage {
		return false
	}

	if len(w.Class) != len(other.Class) {
		return false
	}
	for i := range w.Class {
		if w.Class[i] != other.Class[i] {
			return false
		}
	}

	if Detailed {
		if w.Info == nil && other.Info == nil {
			return true
		}
		if w.Info == nil || other.Info == nil {
			return false
		}
		if w.Info.Shuffle != other.Info.Shuffle {
			return false
		}
		if w.Info.Status != other.Info.Status {
			return false
		}
		if w.Info.Volume != other.Info.Volume {
			return false
		}
	}

	return true
}

func (w *Waybar) Paused(info *PlayerInfo) {
	if !LyricOnly {
		w.Text = fmt.Sprintf("%s - %s", info.Artist, info.Title)
	}
	w.Alt = Paused
	w.Class = Class{Paused}
}

// PlayerInfo holds all information of currently playing track metadata
type PlayerInfo struct {
	Player string `json:"player"`
	ID     string `json:"id"`
	Artist string `json:"artist"`
	Title  string `json:"title"`
	Album  string `json:"album"`
	Cover  string `json:"cover"`

	Volume   float64       `json:"volume"`
	Position time.Duration `json:"position"`
	Length   time.Duration `json:"length"`
	Shuffle  bool          `json:"shuffle"`

	Status mpris.PlaybackStatus `json:"status"`
}

// MarshalJSON encodes PlayerInfo with durations in seconds (float)
func (p PlayerInfo) MarshalJSON() ([]byte, error) {
	p.Player = p.Player[23:]
	type Alias PlayerInfo // create alias to avoid recursion
	return json.Marshal(&struct {
		Alias
		Position float64 `json:"position"`
		Length   float64 `json:"length"`
	}{
		Alias:    (Alias)(p),
		Position: p.Position.Seconds(),
		Length:   p.Length.Seconds(),
	})
}

func (p *PlayerInfo) Percentage() int {
	return int((p.Position * 100) / p.Length)
}

func (p *PlayerInfo) Waybar() *Waybar {
	alt := Status(Playing)
	if p.Status == "Paused" {
		alt = Paused
	}

	var text string
	if !LyricOnly {
		text = fmt.Sprintf("%s - %s", p.Artist, p.Title)
	}

	waybar := &Waybar{
		Class:      Class{alt},
		Text:       text,
		Alt:        alt,
		Percentage: p.Percentage(),
	}

	if Detailed {
		waybar.Info = p
	}

	return waybar
}

// UpdatePosition updates the position of player
func (p *PlayerInfo) UpdatePosition(player *mpris.Player) error {
	pos, err := player.GetPosition()
	if err != nil {
		return err
	}
	p.Position = pos

	// HACK: YoutubeMusic dbus position ≈ 1.1 slow
	if player.GetName() == mpris.BaseInterface+".YoutubeMusic" {
		slog.Debug("Adding 1.1 second to adjust mpris delay")
		p.Position += 1100 * time.Millisecond
	}

	return nil
}

// noopHandler is a empty handler that ignore all logs
type noopHandler struct{}

var _ slog.Handler = (*noopHandler)(nil)

func (h *noopHandler) Enabled(_ context.Context, _ slog.Level) bool  { return false }
func (h *noopHandler) Handle(_ context.Context, _ slog.Record) error { return nil }
func (h *noopHandler) WithAttrs(_ []slog.Attr) slog.Handler          { return h }
func (h *noopHandler) WithGroup(_ string) slog.Handler               { return h }
