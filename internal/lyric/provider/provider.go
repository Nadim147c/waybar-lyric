package provider

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/Nadim147c/waybar-lyric/internal/lyric/models"
	"github.com/Nadim147c/waybar-lyric/internal/match"
	"github.com/Nadim147c/waybar-lyric/internal/player"
)

// FetchFunc fetches lyrics from a lyrics source.
type FetchFunc func(ctx context.Context, metadata *player.Metadata) (models.Lyrics, error)

// LyricProvider is fetches lyrics from a lyrics source.
type LyricProvider interface {
	Name() string
	Fetch(ctx context.Context, metadata *player.Metadata) (models.Lyrics, error)
}

// NewProvider creates a new LyricProvider.
func NewProvider(name string, f FetchFunc) LyricProvider {
	return &provider{name, f}
}

type provider struct {
	name string
	f    FetchFunc
}

func (p *provider) Name() string {
	return p.name
}

func (p *provider) Fetch(ctx context.Context, metadata *player.Metadata) (models.Lyrics, error) {
	return p.f(ctx, metadata)
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

	slog.Debug("SmartMatch",
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
