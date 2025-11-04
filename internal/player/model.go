package player

import (
	"encoding/json"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/Nadim147c/go-mpris"
	"github.com/godbus/dbus/v5"
)

// Metadata holds all information of currently playing track metadata
type Metadata struct {
	Player string `json:"player"`
	ID     string `json:"id"`
	Artist string `json:"artist"`
	Title  string `json:"title"`
	Album  string `json:"album"`
	Cover  string `json:"cover"`

	URL      *url.URL                `json:"url,omitempty"`
	Metadata map[string]dbus.Variant `json:"-"`

	Volume   float64       `json:"volume"`
	Position time.Duration `json:"position"`
	Length   time.Duration `json:"length"`
	Shuffle  bool          `json:"shuffle"`

	Status mpris.PlaybackStatus `json:"status"`
}

// MarshalJSON encodes PlayerInfo with durations in seconds (float)
func (p Metadata) MarshalJSON() ([]byte, error) {
	p.Player = strings.TrimPrefix(p.Player, mpris.BaseInterface+".")

	var u string
	if p.URL != nil {
		u = p.URL.String()
	}

	type Alias Metadata // create alias to avoid recursion
	return json.Marshal(&struct {
		Alias
		Position float64 `json:"position"`
		Length   float64 `json:"length"`
		URL      string  `json:"url,omitempty"`
	}{
		Alias:    (Alias)(p),
		Position: p.Position.Seconds(),
		Length:   p.Length.Seconds(),
		URL:      u,
	})
}

// UnmarshalJSON decodes PlayerInfo with durations in seconds (float)
func (p *Metadata) UnmarshalJSON(data []byte) error {
	type Alias Metadata // prevent recursion
	aux := &struct {
		Position float64 `json:"position"`
		Length   float64 `json:"length"`
		URL      string  `json:"url,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(p),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Restore durations from seconds
	p.Position = time.Duration(aux.Position * float64(time.Second))
	p.Length = time.Duration(aux.Length * float64(time.Second))

	// Parse URL if provided
	if aux.URL != "" {
		if u, err := url.Parse(aux.URL); err == nil {
			p.URL = u
		} else {
			p.URL = nil // or handle invalid URL error if desired
		}
	}

	return nil
}

// Percentage is player position in percentage rounded to int
func (p *Metadata) Percentage() int {
	return int(((p.Position * 100) / p.Length))
}

// UpdatePosition updates the position of player
func (p *Metadata) UpdatePosition(player *mpris.Player) error {
	pos, err := player.GetPosition()
	if err != nil {
		return err
	}
	p.Position = pos

	// HACK: YoutubeMusic dbus position is rounded to seconds which isn't ideal
	// for realtime lyrics. Add 1.1sec delay make lyrics always appear before
	// the song.
	if player.GetName() == mpris.BaseInterface+".YoutubeMusic" ||
		(p.URL != nil && strings.Contains(p.URL.Host, "music.youtube.com")) {
		slog.Debug("Adding 1.1 second to adjust mpris delay")
		p.Position += 1100 * time.Millisecond
	}

	return nil
}
