package lrclib

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/Nadim147c/waybar-lyric/internal/config"
	"github.com/Nadim147c/waybar-lyric/internal/lyric/formats/lrc"
	"github.com/Nadim147c/waybar-lyric/internal/lyric/models"
	"github.com/Nadim147c/waybar-lyric/internal/lyric/provider"
	"github.com/Nadim147c/waybar-lyric/internal/player"
)

var retryAfter atomic.Value

func init() {
	retryAfter.Store(time.Now())
}

// Response is the response sent from LrcLib api.
type response struct {
	ID           int     `json:"id"`
	TrackName    string  `json:"trackName"`
	ArtistName   string  `json:"artistName"`
	AlbumName    string  `json:"albumName"`
	Duration     float64 `json:"duration"`
	Instrumental bool    `json:"instrumental"`
	PlainLyrics  string  `json:"plainLyrics"`
	SyncedLyrics string  `json:"syncedLyrics"`
}

// Endpoint is api endpoint for lrclib.
const Endpoint = "https://lrclib.net/api/search"

// Provider is a lyrics provider that fetches lyrics from lrclib.
var Provider = provider.NewProvider("lrclib lyrics api",
	func(ctx context.Context, metadata *player.Metadata) (models.Lyrics, error) {
		if retryAfter.Load().(time.Time).After(time.Now()) {
			return models.Lyrics{}, models.ErrLyricsNotFound
		}

		params := url.Values{}
		params.Set("track_name", metadata.RawTitle)
		params.Set("artist_name", metadata.RawArtist)

		header := http.Header{}
		header.Set("User-Agent", config.Version)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, Endpoint, nil)
		if err != nil {
			return models.Lyrics{}, err
		}

		req.URL.RawQuery = params.Encode()
		req.Header = header

		slog.Info("Fetching lyrics from Lrclib", "url", req.URL.String())

		client := http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return models.Lyrics{}, err
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusNotFound {
			return models.Lyrics{}, models.ErrLyricsNotFound
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			const defaultWait = 10 * time.Minute
			const maxWait = time.Hour
			wait := defaultWait
			h := resp.Header.Get("Retry-After")
			if secs, perr := strconv.ParseInt(h, 10, 64); perr == nil {
				if secs > 0 {
					wait = min(time.Duration(secs)*time.Second, maxWait)
				}
			} else if t, perr := http.ParseTime(h); perr == nil {
				if d := time.Until(t); d > 0 {
					wait = min(d, maxWait)
				}
			}
			retryAfter.Store(time.Now().Add(wait))
			return models.Lyrics{}, fmt.Errorf("HTTP status: %d", resp.StatusCode)
		}

		if resp.StatusCode >= 300 {
			return models.Lyrics{}, fmt.Errorf("unexpected HTTP status: %d", resp.StatusCode)
		}

		var items []response
		err = json.NewDecoder(resp.Body).Decode(&items)
		if err != nil {
			return models.Lyrics{}, err
		}

		if len(items) == 0 {
			return models.Lyrics{}, models.ErrSearchResultEmpty
		}

		var best *response
		var bestScore float64

		for item := range slices.Values(items) {
			if item.SyncedLyrics == "" {
				continue
			}
			itemScore := provider.Score(metadata, provider.LyricsResult{
				Title:    item.TrackName,
				Artist:   item.ArtistName,
				Album:    item.AlbumName,
				Duration: time.Duration(item.Duration * float64(time.Second)),
			})
			if itemScore > bestScore {
				best = &item
				bestScore = itemScore
			}
		}

		if best == nil {
			return models.Lyrics{}, models.ErrLyricsNotSynced
		}

		lines, err := lrc.ParseText(best.SyncedLyrics)
		if err != nil {
			return models.Lyrics{}, err
		}

		score := min(bestScore/5, 1)

		return models.Lyrics{Lines: lines, Score: score}, nil //nolint
	})
