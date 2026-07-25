package betterlyrics

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/Nadim147c/waybar-lyric/internal/lyric/formats/ttml"
	"github.com/Nadim147c/waybar-lyric/internal/lyric/models"
	"github.com/Nadim147c/waybar-lyric/internal/lyric/provider"
	"github.com/Nadim147c/waybar-lyric/internal/match"
	"github.com/Nadim147c/waybar-lyric/internal/player"
)

const (
	LyricsAPIEndpoint = "https://lyrics-api.boidu.dev/getLyrics"
	UnisonEndpoint    = "https://unison.boidu.dev/lyrics"
)

// Provider is the betterlyrics lyrics provider.
var Provider = provider.NewProvider(
	"betterlyrics",
	func(ctx context.Context, metadata *player.Metadata) (models.Lyrics, error) {
		primary, primaryErr := genericProvider(ctx, LyricsAPIEndpoint, metadata)
		if primaryErr == nil && provider.WordLevelSyncScore(primary.Lines) > 0 {
			return primary, nil
		}

		// Fetch fallback lyrics
		fallback, fallbackErr := genericProvider(ctx, UnisonEndpoint, metadata)

		if primaryErr != nil && fallbackErr != nil {
			return models.Lyrics{}, errors.Join(primaryErr, fallbackErr)
		}

		if primaryErr != nil {
			return fallback, nil
		}
		if fallbackErr != nil {
			return primary, nil
		}

		// Both calls succeeded: Pick the best result
		if provider.WordLevelSyncScore(fallback.Lines) > 0 {
			return fallback, nil
		}
		if primary.Score > fallback.Score {
			return primary, nil
		}

		return fallback, nil
	},
)

func genericProvider(ctx context.Context, endpoint string, metadata *player.Metadata) (models.Lyrics, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return models.Lyrics{}, err
	}

	params := url.Values{}
	params.Set("song", metadata.RawTitle)
	params.Set("artist", metadata.Artist)
	params.Set("album", metadata.Album)
	req.URL.RawQuery = params.Encode()

	slog.Info("Fetching lyrics from betterlyrics api", "url", req.URL.String())

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return models.Lyrics{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return models.Lyrics{}, fmt.Errorf("[%d] %w", resp.StatusCode, models.ErrLyricsNotFound)
	}

	if resp.StatusCode >= 300 {
		return models.Lyrics{}, fmt.Errorf("unexpected HTTP status: %d", resp.StatusCode)
	}

	var data struct {
		TTML string `json:"ttml"`
	}

	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return models.Lyrics{}, err
	}

	l, err := ttml.GetTextLength(data.TTML)
	if err != nil {
		return models.Lyrics{}, err
	}

	score := match.Durations(metadata.Length, l)

	lines, err := ttml.ParseText(data.TTML)
	if err != nil {
		return models.Lyrics{}, err
	}

	return models.Lyrics{Lines: lines, Score: score}, nil //nolint
}
