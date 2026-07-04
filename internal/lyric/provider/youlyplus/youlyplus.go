package youlyplus

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

var Hosts = []string{
	"https://lyricsplus.binimum.org/",
	"https://lyricsplus.prjktla.my.id/",
	"https://lyricsplus.prjktla.workers.dev/",
	"https://lyricsplus.atomix.one/",
	"https://lyricsplus-seven.vercel.app/",
}

// Provider is the lrclib lyrics provider.
var Provider = provider.NewProvider("youlyplus",
	func(ctx context.Context, metadata *player.Metadata) (models.Lyrics, error) {
		var errs []error
		for _, host := range Hosts {
			lyrics, err := genericProvider(ctx, host, metadata)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			return lyrics, nil
		}
		return models.Lyrics{}, errors.Join(errs...)
	})

func genericProvider(ctx context.Context, host string, metadata *player.Metadata) (lyrics models.Lyrics, err error) {
	lyrics.Metadata = metadata

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, host, nil)
	if err != nil {
		return lyrics, err
	}

	params := url.Values{}
	params.Set("title", lyrics.Metadata.RawTitle)
	params.Set("artist", lyrics.Metadata.Artist)
	params.Set("album", lyrics.Metadata.Album)
	req.URL.RawQuery = params.Encode()
	req.URL.Path = "/v1/ttml/get"

	slog.Info("Fetching lyrics from youlyplus api", "url", req.URL.String())

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return models.Lyrics{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return lyrics, fmt.Errorf("[%d] %w", resp.StatusCode, models.ErrLyricsNotFound)
	}

	if resp.StatusCode >= 300 {
		return lyrics, fmt.Errorf("unexpected HTTP status: %d", resp.StatusCode)
	}

	var data struct {
		TTML string `json:"ttml"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return lyrics, err
	}

	l, err := ttml.GetTextLength(data.TTML)
	if err != nil {
		return lyrics, err
	}

	const minimumScore = 0.67
	if score := match.Durations(metadata.Length, l); score < minimumScore {
		return lyrics, &models.LyricsMatchScoreError{Score: score, Threshold: minimumScore}
	}

	lyrics.Lines, err = ttml.ParseText(data.TTML)
	return lyrics, err
}
