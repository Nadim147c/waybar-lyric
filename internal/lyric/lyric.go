package lyric

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"time"

	"github.com/Nadim147c/waybar-lyric/internal/config"
	"github.com/Nadim147c/waybar-lyric/internal/lyric/models"
	"github.com/Nadim147c/waybar-lyric/internal/lyric/provider"
	asText "github.com/Nadim147c/waybar-lyric/internal/lyric/provider/as_text"
	"github.com/Nadim147c/waybar-lyric/internal/lyric/provider/lrclib"
	"github.com/Nadim147c/waybar-lyric/internal/lyric/provider/simpmusic"
	"github.com/Nadim147c/waybar-lyric/internal/player"
	"github.com/Nadim147c/waybar-lyric/internal/str"
	"github.com/gofrs/flock"
)

const flockPathPrefix = "/tmp/waybar-lyric"

// lyricTimeout is the timeout duration for lyrics download.
//
// NOTE: lyricTimeout doesn't ensure that GetLyrics will only run for given
// duration
//
// TODO: add cli flag for user defined duration
const lyricTimeout = 10 * time.Second

var providers = []provider.LyricProvider{
	asText.Provider,
	lrclib.Provider,
	simpmusic.Provider,
}

// GetLyrics returns lyrics for given *player.Info
func GetLyrics(ctx context.Context, metadata *player.Metadata) (models.Lyrics, error) {
	lyrics := models.Lyrics{Metadata: metadata}

	lockCtx, cancel := context.WithTimeout(ctx, lyricTimeout)
	defer cancel()

	lockFile := fmt.Sprintf("%s-%s.lock", flockPathPrefix, metadata.ID)
	flocker := flock.New(lockFile)
	locked, err := flocker.TryLockContext(lockCtx, 200*time.Millisecond)
	if err != nil {
		Store.NotFound(metadata.ID)
		return lyrics, fmt.Errorf("failed to take flock for id(%s): %v", metadata.ID, err)
	}
	defer os.Remove(lockFile)
	defer flocker.Close()

	if !locked {
		return lyrics, fmt.Errorf("another instance is trying to download: id(%s)", metadata.ID)
	}

	uri := metadata.ID
	if l, err := Store.Load(uri); err == nil {
		return l, nil
	}

	for p := range slices.Values(providers) {
		ctx, cancel := context.WithTimeout(ctx, lyricTimeout)
		defer cancel()
		lyrics, err := p.Fetch(ctx, metadata)
		if err != nil {
			slog.Warn("Provider failed", "name", p.Name(), "error", err)
			continue
		}

		slices.SortFunc(lyrics.Lines, func(a, b models.Line) int {
			return int((a.Timestamp - b.Timestamp) / time.Millisecond)
		})

		if err := Store.Save(lyrics); err != nil {
			return lyrics, fmt.Errorf("failed to save lyrics cache json: %w", err)
		}

		CensorLyrics(lyrics)
		TruncateLyrics(lyrics)
		return lyrics, nil
	}

	Store.NotFound(metadata.ID)

	return models.Lyrics{}, models.ErrLyricsNotFound
}

// CensorLyrics censors the lyrics with given filtering type
func CensorLyrics(lyrics models.Lyrics) {
	if config.FilterProfanity {
		for i, l := range lyrics.Lines {
			lyrics.Lines[i].Text = str.CensorText(l.Text)
		}
	}
}

// TruncateLyrics truncates all lines using utf8 character length from user
// input
func TruncateLyrics(lyrics models.Lyrics) {
	for i, l := range lyrics.Lines {
		lyrics.Lines[i].Text = str.Truncate(l.Text)
	}
}
