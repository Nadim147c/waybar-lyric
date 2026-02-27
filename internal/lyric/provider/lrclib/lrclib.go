package lrclib

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"time"

	"github.com/Nadim147c/waybar-lyric/internal/config"
	"github.com/Nadim147c/waybar-lyric/internal/lyric/models"
	"github.com/Nadim147c/waybar-lyric/internal/lyric/provider"
	"github.com/Nadim147c/waybar-lyric/internal/player"
)

// Response is the response sent from LrcLib api
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

// Endpoint is api endpoint for lrclib
const Endpoint = "https://lrclib.net/api/search"

// Provider is a lyrics provider that fetches lyrics from lrclib.
var Provider = provider.NewProvider("lrclib lyrics api",
	func(ctx context.Context, metadata *player.Metadata) (lyrics models.Lyrics, err error) {
		lyrics.Metadata = metadata

		params := url.Values{}
		params.Set("track_name", metadata.RawTitle)
		params.Set("artist_name", metadata.RawArtist)
		if metadata.Album != "" {
			params.Set("album_name", metadata.Album)
		}

		header := http.Header{}
		header.Set("User-Agent", config.Version)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, Endpoint, nil)
		if err != nil {
			return
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
			return lyrics, models.ErrLyricsNotFound
		}

		if resp.StatusCode >= 300 {
			return lyrics, fmt.Errorf("unexpected HTTP status: %d", resp.StatusCode)
		}

		var items []response
		err = json.NewDecoder(resp.Body).Decode(&items)
		if err != nil {
			return lyrics, fmt.Errorf("failed to parse body: %w", err)
		}

		if len(items) == 0 {
			return lyrics, models.ErrLyricsNotFound
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

		if bestScore < provider.MinimumScore {
			return lyrics, &models.ErrLyricsMatchScore{
				Score:     bestScore,
				Threshold: provider.MinimumScore,
			}
		}

		lyrics.Lines, err = provider.ParseText(best.SyncedLyrics)
		return
	})
