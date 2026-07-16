package youlyplus

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"sync"

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

// Provider is the youlyplus lyrics provider.
var Provider = &provider.LyricProvider{
	Name: "youlyplus",
	Fetch: func(ctx context.Context, wg *sync.WaitGroup, metadata *player.Metadata, out chan<- provider.Result) {
		defer wg.Done()
		for _, host := range Hosts {
			wg.Go(func() {
				var res provider.Result
				res.Provider = fmt.Sprintf("youlyplus [%s]", host)
				res.Lyrics, res.Score, res.Err = genericProvider(ctx, host, metadata)
				out <- res
			})
		}
	},
}

func genericProvider(ctx context.Context, host string, metadata *player.Metadata) (lyrics models.Lyrics, score float64, err error) {
	lyrics.Metadata = metadata

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, host, nil)
	if err != nil {
		return lyrics, score, err
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
}
