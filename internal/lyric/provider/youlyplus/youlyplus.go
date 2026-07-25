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
				res.Lyrics, res.Err = genericProvider(ctx, host, metadata)
				out <- res
			})
		}
	},
}

func genericProvider(ctx context.Context, host string, metadata *player.Metadata) (models.Lyrics, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, host, nil)
	if err != nil {
		return models.Lyrics{}, err
	}

	params := url.Values{}
	params.Set("title", metadata.RawTitle)
	params.Set("artist", metadata.Artist)
	params.Set("album", metadata.Album)
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

	dur, err := ttml.GetTextLength(data.TTML)
	if err != nil {
		return models.Lyrics{}, err
	}

	score := match.Durations(metadata.Length, dur)

	lines, err := ttml.ParseText(data.TTML)
	if err != nil {
		return models.Lyrics{}, err
	}

	return models.Lyrics{Lines: lines, Score: score}, nil //nolint
}
