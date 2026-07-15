package provider

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/Nadim147c/waybar-lyric/internal/lyric/models"
	"github.com/Nadim147c/waybar-lyric/internal/match"
	"github.com/Nadim147c/waybar-lyric/internal/player"
)

// Result is the lyrics result.
type Result struct {
	Lyrics   models.Lyrics
	Provider string
	Score    float64 // extra score for word-level synced lyrics
	Err      error
}

// CalculateLyricsScore returns score of lines sync.
func CalculateLyricsScore(lines models.Lines) float64 {
	size := len(lines)
	var count int
	for _, line := range lines {
		if len(line.Words) != 0 {
			count++
		}
	}
	return float64(count) / float64(size)
}

// FetchFunc fetches lyrics from a lyrics source.
type FetchFuncResult func(ctx context.Context, metadata *player.Metadata) (lyrics models.Lyrics, score float64, err error)

// FetchFunc fetches lyrics from a lyrics source.
type FetchFunc func(ctx context.Context, wg *sync.WaitGroup, metadata *player.Metadata, out chan<- Result)

// NewProvider creates a new LyricProvider.
func NewProvider(name string, f FetchFuncResult) *LyricProvider {
	var wrapper FetchFunc = func(ctx context.Context, wg *sync.WaitGroup, metadata *player.Metadata, out chan<- Result) {
		var res Result
		res.Provider = name
		res.Lyrics, res.Score, res.Err = f(ctx, metadata)
		out <- res
		wg.Done()
	}
	return &LyricProvider{name, wrapper}
}

type LyricProvider struct {
	Name  string
	Fetch FetchFunc
}

// MinimumScore is the minimum score required to consider the downloaded lyrics
// as the lyrics of the current song!
const MinimumScore float64 = 3.5

// LyricsResult represents the metadata returned from a lyrics provider.
type LyricsResult struct {
	Title    string
	Artist   string
	Album    string
	Duration time.Duration
}

// Score calculates a similarity score between an MPRIS track and a LyricsResult
// to determine if the lyrics are suitable for the current track. Returns true
// is lyrics is suitable.
func Score(track *player.Metadata, result LyricsResult) float64 {
	durationScore := match.Durations(track.Length, result.Duration) * 2
	titleScore := match.Strings(track.RawTitle, result.Title) * 2
	var artistsScore float64
	if len(track.Artists) > 1 {
		var separate float64
		for _, artist := range track.Artists {
			separate += match.Strings(artist, result.Artist)
		}
		joined := match.Strings(strings.Join(track.Artists, ", "), result.Artist)
		artistsScore = max(separate, joined)
	} else {
		artistsScore = match.Strings(track.RawArtist, result.Artist)
	}
	albumScore := match.Strings(track.Album, result.Album)

	score := durationScore + titleScore + albumScore + artistsScore

	slog.Debug(
		"SmartMatch",
		"score", score,
		"album_want", track.Album,
		"album_got", result.Album,
		"album_score", albumScore,
		"artist_want", track.RawArtist,
		"artist_got", result.Artist,
		"artists_score", artistsScore,
		"duration_score", durationScore,
		"title_want", track.RawTitle,
		"title_got", result.Title,
		"title_score", titleScore,
	)

	return score
}
