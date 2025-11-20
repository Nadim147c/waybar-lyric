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

// URL wraps net.URL to provide JSON marshaling/unmarshaling
type URL struct {
	*url.URL
}

// NewURL creates a new URL from a string
func NewURL(rawURL string) (*URL, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	return &URL{URL: u}, nil
}

// MarshalJSON implements json.Marshaler
func (u URL) MarshalJSON() ([]byte, error) {
	if u.URL == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.URL.String())
}

// UnmarshalJSON implements json.Unmarshaler
func (u *URL) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	// Handle null case
	if s == "" {
		u.URL = nil
		return nil
	}

	parsedURL, err := url.Parse(s)
	if err != nil {
		return err
	}

	u.URL = parsedURL
	return nil
}

// String returns the string representation of the URL
// Implements the fmt.Stringer interface
func (u URL) String() string {
	if u.URL == nil {
		return ""
	}
	return u.URL.String()
}

// IsNil checks if the underlying URL is nil
func (u *URL) IsNil() bool {
	return u == nil || u.URL == nil
}

// Metadata holds all information of currently playing track metadata
type Metadata struct {
	Player string `json:"player"`
	ID     string `json:"id"`
	Artist string `json:"artist"`
	Title  string `json:"title"`
	Album  string `json:"album"`
	Cover  string `json:"cover"`
	URL    *URL   `json:"url"`

	Metadata map[string]dbus.Variant `json:"-"`

	Volume   float64       `json:"volume"`
	Position time.Duration `json:"position"`
	Length   time.Duration `json:"length"`
	Shuffle  bool          `json:"shuffle"`

	Status mpris.PlaybackStatus `json:"status"`
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
		(p.URL.IsNil() && strings.Contains(p.URL.Host, "music.youtube.com")) {
		slog.Debug("Adding 1.1 second to adjust mpris delay")
		p.Position += 1100 * time.Millisecond
	}

	return nil
}
