package lyric

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"os"
	"regexp"
	"slices"
	"sync"
	"time"

	"github.com/Nadim147c/waybar-lyric/internal/config"
	"github.com/Nadim147c/waybar-lyric/internal/lyric/models"
	"github.com/Nadim147c/waybar-lyric/internal/lyric/provider"
	asText "github.com/Nadim147c/waybar-lyric/internal/lyric/provider/as_text"
	"github.com/Nadim147c/waybar-lyric/internal/lyric/provider/betterlyrics"
	"github.com/Nadim147c/waybar-lyric/internal/lyric/provider/embedded"
	lrcFile "github.com/Nadim147c/waybar-lyric/internal/lyric/provider/lrc_file"
	"github.com/Nadim147c/waybar-lyric/internal/lyric/provider/lrclib"
	"github.com/Nadim147c/waybar-lyric/internal/lyric/provider/simpmusic"
	"github.com/Nadim147c/waybar-lyric/internal/lyric/provider/youlyplus"
	"github.com/Nadim147c/waybar-lyric/internal/player"
	"github.com/Nadim147c/waybar-lyric/internal/str"
	"github.com/gofrs/flock"
)

const flockPathPrefix = "/tmp/waybar-lyric"

// lyricTimeout is the timeout duration for lyrics download.
//
// NOTE: lyricTimeout doesn't ensure that GetLyrics will only run for given
// duration.
//
// TODO: add cli flag for user defined duration.
const lyricTimeout = 10 * time.Second

var providers = []*provider.LyricProvider{
	asText.Provider,
	lrcFile.Provider,
	embedded.Provider,
	youlyplus.Provider,
	betterlyrics.Provider,
	simpmusic.Provider,
	lrclib.Provider,
}

var reArtists = regexp.MustCompile(`(, | and )`)

// GetLyrics returns lyrics for given *player.Info.
func GetLyrics(ctx context.Context, metadata *player.Metadata) (models.Lyrics, error) {
	lyrics := models.Lyrics{
		Metadata: metadata,
		Lines:    nil,
	}

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

	if metadata.URL != nil || metadata.URL.Hostname() == "music.youtube.com" {
		metadata.RawArtist = reArtists.ReplaceAllLiteralString(metadata.RawArtist, ", ")
	}

	var wg sync.WaitGroup

	ctx, cancel = context.WithTimeout(ctx, lyricTimeout)
	defer cancel()

	out := make(chan provider.Result, 10)
	for _, p := range providers {
		wg.Add(1)
		go p.Fetch(ctx, &wg, metadata, out)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	var errs []error
	var results []provider.Result

	for res := range out {
		if res.Err != nil {
			errs = append(errs, err)
			continue
		}
		results = append(results, res)
		if res.Score > 1 {
			cancel()
		}
	}

	if len(results) == 0 {
		Store.NotFound(metadata.ID)
		return models.Lyrics{}, models.ErrLyricsNotFound
	}

	best := provider.Result{Score: math.Inf(-1)} //nolint

	for _, res := range results {
		if res.Score > best.Score {
			best = res
		}
	}
	slog.Info("lyrics found", "provider", best.Provider, "word-sync", best.Score > 1)

	lyrics = best.Lyrics

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

// CensorLyrics censors the lyrics with given filtering type.
func CensorLyrics(lyrics models.Lyrics) {
	if !config.FilterProfanity {
		return
	}

	for i, line := range lyrics.Lines {
		lyrics.Lines[i].Text = str.CensorText(line.Text)
		if len(lyrics.Lines[i].Words) == 0 {
			continue
		}
		for j, word := range lyrics.Lines[i].Words {
			lyrics.Lines[i].Words[j].Text = str.CensorText(word.Text)
		}
	}
}

// TruncateLyrics truncates all lines using utf8 character length from user
// input.
func TruncateLyrics(lyrics models.Lyrics) {
	for i, l := range lyrics.Lines {
		lyrics.Lines[i].Text = str.Truncate(l.Text)
	}
}
