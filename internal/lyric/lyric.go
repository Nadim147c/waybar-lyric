package lyric

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/Nadim147c/waybar-lyric/internal/config"
	"github.com/Nadim147c/waybar-lyric/internal/player"
	"github.com/Nadim147c/waybar-lyric/internal/shared"
	"github.com/Nadim147c/waybar-lyric/internal/str"
	"github.com/Nadim147c/waybar-lyric/internal/waybar"
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

var (
	//revive:disable
	ErrLyricsNotFound  = errors.New("lyrics not found")
	ErrLyricsNotExists = errors.New("lyrics does not exists")
	ErrLyricsNotSynced = errors.New("lyrics is not synced")
	//revive:enable
)

// Store is in memory cache for lyrics
var Store = newStore()

// LrclibEndpoint is api endpoint for lrclib
const LrclibEndpoint = "https://lrclib.net/api/get"

func request(params url.Values, header http.Header) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, LrclibEndpoint, nil)
	if err != nil {
		return nil, err
	}

	req.URL.RawQuery = params.Encode()
	req.Header = header

	slog.Info("Fetching lyrics from Lrclib", "url", req.URL.String())

	client := http.Client{}

	return client.Do(req)
}

// CensorLyrics censors the lyrics with given filtering type
func CensorLyrics(lyrics shared.Lyrics) {
	if config.FilterProfanity {
		for i, l := range lyrics {
			lyrics[i].Text = str.CensorText(l.Text, config.FilterProfanityType)
		}
	}
}

// TruncateLyrics truncates all lines using utf8 character length from user input
func TruncateLyrics(lyrics shared.Lyrics) {
	for i, l := range lyrics {
		lyrics[i].Text = str.Truncate(l.Text, config.MaxTextLength)
	}
}

// GetLyrics returns lyrics for given *player.Info
func GetLyrics(info *player.Info) (shared.Lyrics, error) {
	uri := filepath.Base(info.ID)
	uri = strings.ReplaceAll(uri, "/", "-")

	if val, exists := Store.Load(uri); exists {
		if len(val) == 0 {
			return val, ErrLyricsNotExists
		}
		slog.Debug("Lyrics found in memory cache", "lines", len(val))
		return val, nil
	}

	cacheFile := filepath.Join(CacheDir, uri+".csv")

	cachedLyrics, err := LoadCache(cacheFile)
	if err == nil {
		CensorLyrics(cachedLyrics)
		TruncateLyrics(cachedLyrics)
		Store.Save(uri, cachedLyrics)
		return cachedLyrics, nil
	}
	slog.Warn("Can't find the lyrics in the cache", "error", err)

	w := waybar.ForPlayer(info)
	w.Alt = waybar.Getting
	w.Class = append(w.Class, waybar.Getting)
	w.Encode()

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

	resp, err := request(queryParams, header)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch lyrics: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		Store.Save(uri, shared.Lyrics{})
		return nil, ErrLyricsNotFound
	}

	if resp.StatusCode != http.StatusOK {
		Store.Save(uri, shared.Lyrics{})
		return nil, fmt.Errorf("unexpected HTTP status: %d", resp.StatusCode)
	}

	var resJSON LrcLibResponse
	err = json.NewDecoder(resp.Body).Decode(&resJSON)
	if err != nil {
		Store.Save(uri, shared.Lyrics{})
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	lyrics, err := ParseLyrics(resJSON.SyncedLyrics)
	if err != nil {
		Store.Save(uri, shared.Lyrics{})
		return nil, fmt.Errorf("failed to parse lyrics: %w", err)
	}

	if len(lyrics) == 0 {
		Store.Save(uri, shared.Lyrics{})
		return nil, ErrLyricsNotSynced
	}

	slices.SortFunc(lyrics, func(a, b shared.LyricLine) int {
		return int((a.Timestamp - b.Timestamp) / time.Millisecond)
	})

	if err = SaveCache(info, lyrics, cacheFile); err != nil {
		return nil, fmt.Errorf("failed to cache lyrics to psudo csv: %w", err)
	}

	CensorLyrics(lyrics)
	TruncateLyrics(lyrics)
	Store.Save(uri, lyrics)
	return lyrics, nil
}
