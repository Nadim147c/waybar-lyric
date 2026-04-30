package simpmusic

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/Nadim147c/waybar-lyric/internal/lyric/models"
	"github.com/Nadim147c/waybar-lyric/internal/lyric/provider"
	"github.com/Nadim147c/waybar-lyric/internal/player"
)

// Endpoint is simpmusic lyrics api endpoint.
const Endpoint = "https://api-lyrics.simpmusic.org/"

type response struct {
	Type    string `json:"type"`
	Data    []data `json:"data"`
	Success bool   `json:"success"`
}

type data struct {
	ID               string  `json:"id"`
	VideoID          string  `json:"videoId"`
	SongTitle        string  `json:"songTitle"`
	ArtistName       string  `json:"artistName"`
	AlbumName        string  `json:"albumName"`
	DurationSeconds  float64 `json:"durationSeconds"`
	PlainLyric       string  `json:"plainLyric"`
	SyncedLyrics     string  `json:"syncedLyrics"`
	RichSyncLyrics   string  `json:"richSyncLyrics"`
	Vote             int64   `json:"vote"`
	Contributor      string  `json:"contributor"`
	ContributorEmail string  `json:"contributorEmail"`
}

// Provider is the lrclib lyrics provider.
var Provider = provider.NewProvider("simpmusic lyrics",
	func(ctx context.Context, metadata *player.Metadata) (lyrics models.Lyrics, err error) {
		lyrics.Metadata = metadata

		vidoeID := metadata.URL.Query().Get("v")
		isYoutubeVideo := strings.HasSuffix(metadata.URL.Hostname(), "youtube.com")
		if !isYoutubeVideo || vidoeID == "" {
			return lyrics, models.ErrLyricsNotFound
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, Endpoint, nil)
		if err != nil {
			return lyrics, err
		}

		req.URL.Path = "/v1/" + vidoeID

		slog.Info("Fetching lyrics from simpmusic api", "url", req.URL.String())

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

		var responseData response
		err = json.NewDecoder(resp.Body).Decode(&responseData)
		if err != nil {
			return lyrics, fmt.Errorf("failed to parse body: %w", err)
		}

		if !responseData.Success {
			return lyrics, models.ErrSearchResultEmpty
		}

		var best *data
		var bestScore float64

		for item := range slices.Values(responseData.Data) {
			if item.SyncedLyrics == "" {
				continue
			}
			itemScore := provider.Score(metadata, provider.LyricsResult{
				Title:    item.SongTitle,
				Artist:   item.ArtistName,
				Album:    item.AlbumName,
				Duration: time.Duration(item.DurationSeconds * float64(time.Second)),
			})
			if itemScore > bestScore {
				best = &item
				bestScore = itemScore
			}
		}

		if bestScore < provider.MinimumScore {
			return lyrics, &models.LyricsMatchScoreError{
				Score:     bestScore,
				Threshold: provider.MinimumScore,
			}
		}

		lyrics.Lines, err = provider.ParseText(best.SyncedLyrics)
		return lyrics, err
	})
