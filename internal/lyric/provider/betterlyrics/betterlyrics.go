package betterlyrics

import (
	"context"
	"encoding/json"
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

// Endpoint is simpmusic lyrics api endpoint.
const Endpoint = "https://lyrics-api.boidu.dev/getLyrics"

// Provider is the lrclib lyrics provider.
var Provider = provider.NewProvider("betterlyrics",
	func(ctx context.Context, metadata *player.Metadata) (lyrics models.Lyrics, score float64, err error) {
		lyrics.Metadata = metadata

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, Endpoint, nil)
		if err != nil {
			return lyrics, score, err
		}

		params := url.Values{}
		params.Set("song", lyrics.Metadata.RawTitle)
		params.Set("artist", lyrics.Metadata.Artist)
		params.Set("album", lyrics.Metadata.Album)
		params.Set("duration", fmt.Sprintf("%.0f", lyrics.Metadata.Length.Seconds()))
		req.URL.RawQuery = params.Encode()

		slog.Info("Fetching lyrics from betterlyrics api", "url", req.URL.String())

		client := http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusNotFound {
			return lyrics, score, fmt.Errorf("[%d] %w", resp.StatusCode, models.ErrLyricsNotFound)
		}

		if resp.StatusCode >= 300 {
			return lyrics, score, fmt.Errorf("unexpected HTTP status: %d", resp.StatusCode)
		}

		var data struct {
			TTML string `json:"ttml"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			return lyrics, score, err
		}

		l, err := ttml.GetTextLength(data.TTML)
		if err != nil {
			return lyrics, score, err
		}

		durScore := match.Durations(metadata.Length, l)
		const minimumScore = 0.67
		if durScore < minimumScore {
			return lyrics, score, &models.LyricsMatchScoreError{Score: durScore, Threshold: minimumScore}
		}

		lyrics.Lines, err = ttml.ParseText(data.TTML)
		score = provider.CalculateLyricsScore(lyrics.Lines) + durScore
		return lyrics, score, err
	})
