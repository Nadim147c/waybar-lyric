package lyric

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"time"
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

// LrclibEndpoint is api endpoint for lrclib
const LrclibEndpoint = "https://lrclib.net/api/get"

var (
	//revive:disable
	ErrLyricsNotFound  = errors.New("lyrics not found")
	ErrLyricsNotExists = errors.New("lyrics does not exists")
	ErrLyricsNotSynced = errors.New("lyrics is not synced")
	//revive:enable
)

func request(ctx context.Context, params url.Values, header http.Header) (*http.Response, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, LrclibEndpoint, nil)
	if err != nil {
		return nil, err
	}

	req.URL.RawQuery = params.Encode()
	req.Header = header

	slog.Info("Fetching lyrics from Lrclib", "url", req.URL.String())

	client := http.Client{}

	return client.Do(req)
}
