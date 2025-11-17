package lyric

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"slices"
	"time"

	"github.com/Nadim147c/waybar-lyric/internal/config"
	"github.com/Nadim147c/waybar-lyric/internal/player"
	"github.com/Nadim147c/waybar-lyric/internal/str"
	"github.com/gofrs/flock"
)

// Line is a line of synchronized lyrics
type Line struct {
	Timestamp time.Duration `json:"time"`
	Text      string        `json:"line"`
	Active    bool          `json:"active"`
}

// MarshalJSON implemetions json.Marshaller interface
func (l Line) MarshalJSON() ([]byte, error) {
	type Alias Line
	return json.Marshal(&struct {
		Alias
		Timestamp float64 `json:"time"`
	}{
		Alias:     (Alias)(l),
		Timestamp: l.Timestamp.Seconds(),
	})
}

// UnmarshalJSON implements json.Unmarshaler interface
func (l *Line) UnmarshalJSON(data []byte) error {
	type Alias Line
	aux := &struct {
		Timestamp float64 `json:"time"`
		*Alias
	}{
		Alias: (*Alias)(l),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	l.Timestamp = time.Duration(aux.Timestamp * float64(time.Second))
	return nil
}

// Lines is a slice of Line
type Lines []Line

// Lyrics is the synchronized structured lyrics
type Lyrics struct {
	Metadata *player.Metadata `json:"metadata,omitempty"`
	Lines    Lines            `json:"lyrics"`
}

const flockPathPrefix = "/tmp/waybar-lyric"

// LyricTimeout is the timeout duration for lyrics download.
//
// NOTE: LyricTimeout doesn't ensure that GetLyrics will only run for given
// duration
//
// TODO: add cli flag for user defined duration
const LyricTimeout = 10 * time.Second

// GetLyrics returns lyrics for given *player.Info
func GetLyrics(ctx context.Context, info *player.Metadata) (Lyrics, error) {
	lyrics := Lyrics{Metadata: info}

	lockCtx, cancel := context.WithTimeout(ctx, LyricTimeout)
	defer cancel()

	lockFile := fmt.Sprintf("%s-%s.lock", flockPathPrefix, info.ID)
	flocker := flock.New(lockFile)
	locked, err := flocker.TryLockContext(lockCtx, 200*time.Millisecond)
	if err != nil {
		Store.NotFound(info.ID)
		return lyrics, fmt.Errorf(
			"failed to take flock for id(%s): %v",
			info.ID, err,
		)
	}
	defer os.Remove(lockFile)
	defer flocker.Close()

	if !locked {
		return lyrics, fmt.Errorf(
			"another instance is trying to download: id(%s)",
			info.ID,
		)
	}

	uri := info.ID
	if l, err := Store.Load(uri); err == nil {
		return l, nil
	}

	queryParams := url.Values{}
	queryParams.Set("track_name", info.Title)
	queryParams.Set("artist_name", info.Artist)
	if info.Album != "" {
		queryParams.Set("album_name", info.Album)
	}
	if info.Length != 0 {
		queryParams.Set("duration", fmt.Sprintf("%.2f", info.Length.Seconds()))
	}

	header := http.Header{}
	header.Set("User-Agent", config.Version)

	resp, err := request(ctx, queryParams, header)
	if err != nil {
		Store.NotFound(uri)
		return lyrics, fmt.Errorf("failed to fetch lyrics: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		Store.NotFound(uri)
		return lyrics, ErrLyricsNotFound
	}

	if resp.StatusCode >= 300 {
		Store.NotFound(uri)
		return lyrics, fmt.Errorf("unexpected HTTP status: %d", resp.StatusCode)
	}

	var resJSON LrcLibResponse
	err = json.NewDecoder(resp.Body).Decode(&resJSON)
	if err != nil {
		Store.NotFound(uri)
		return lyrics, fmt.Errorf("failed to read response body: %w", err)
	}

	if s := score(info, resJSON); s < MinimumScore {
		Store.NotFound(uri)
		return lyrics, &ErrLyricsMatchScore{Score: s, Threshold: MinimumScore}
	}

	lines, err := ParseLyrics(resJSON.SyncedLyrics)
	if err != nil {
		Store.NotFound(uri)
		if errors.Is(err, ErrLyricsNotSynced) {
			return lyrics, err
		}
		return lyrics, fmt.Errorf("failed to parse lyrics: %w", err)
	}

	slices.SortFunc(lines, func(a, b Line) int {
		return int((a.Timestamp - b.Timestamp) / time.Millisecond)
	})

	lyrics.Lines = lines

	if err := Store.Save(lyrics); err != nil {
		return lyrics, fmt.Errorf("failed to save lyrics cache json: %w", err)
	}

	CensorLyrics(lyrics)
	TruncateLyrics(lyrics)
	return lyrics, nil
}

// CensorLyrics censors the lyrics with given filtering type
func CensorLyrics(lyrics Lyrics) {
	if config.FilterProfanity {
		for i, l := range lyrics.Lines {
			lyrics.Lines[i].Text = str.CensorText(l.Text)
		}
	}
}

// TruncateLyrics truncates all lines using utf8 character length from user
// input
func TruncateLyrics(lyrics Lyrics) {
	for i, l := range lyrics.Lines {
		lyrics.Lines[i].Text = str.Truncate(l.Text)
	}
}
