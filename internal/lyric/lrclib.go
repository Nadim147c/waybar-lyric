package lyric

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/Nadim147c/waybar-lyric/internal/match"
	"github.com/Nadim147c/waybar-lyric/internal/player"
)

// MinimumScore is the minimum score required to consider the downloaded lyrics
// as the lyrics of the current song!
const MinimumScore float64 = 3.5

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
	// ErrLyricsNotFound indicates that the requested lyrics could not be found.
	ErrLyricsNotFound = errors.New("lyrics not found")
	// ErrLyricsNotExists indicates that the lyrics resource does not exist.
	ErrLyricsNotExists = errors.New("lyrics does not exist")
	// ErrLyricsNotSynced indicates that the lyrics are available but not
	// time-synchronized.
	ErrLyricsNotSynced = errors.New("lyrics is not synced")
)

func request(
	ctx context.Context,
	params url.Values,
	header http.Header,
) (*http.Response, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		LrclibEndpoint,
		nil,
	)
	if err != nil {
		return nil, err
	}

	req.URL.RawQuery = params.Encode()
	req.Header = header

	slog.Info("Fetching lyrics from Lrclib", "url", req.URL.String())

	client := http.Client{}

	return client.Do(req)
}

// score calculates a similarity score between a MPRIS track and LrcLib result
// to find if the lyrics is suitable for current track by scoring duration,
// title, album and artist.
//
// Scoring breakdown:
//   - Duration: Up to 1.5 points for an exact match, decreasing linearly to 0.0
//     when the difference reaches or exceeds 8 seconds.
//   - Title: Up to 2.0 points for an exact match, with lower scores for greater
//     string differences.
//   - Album: Up to 1.0 point for an exact match, with lower scores for greater
//     string differences.
//   - Artists: Up to 1.0 point for an exact match, with lower scores for greater
//     string differences.
func score(track *player.Metadata, lyrics LrcLibResponse) float64 {
	lyricsDur := time.Duration(lyrics.Duration * float64(time.Second))

	durationScore := match.Durations(track.Length, lyricsDur) * 1.5
	titleScore := match.Strings(track.Title, lyrics.TrackName) * 2
	albumScore := match.Strings(track.Album, lyrics.AlbumName)
	artistsScore := match.Strings(track.Artist, lyrics.ArtistName)

	score := durationScore + titleScore + albumScore + artistsScore

	slog.Debug("SmartMatch",
		"score", score,
		"duration_score", durationScore,
		"title_score", titleScore,
		"album_score", albumScore,
		"artists_score", artistsScore,
	)

	return score
}
