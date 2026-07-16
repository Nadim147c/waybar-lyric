package astext

import (
	"context"
	"os"
	"path/filepath"

	"github.com/Nadim147c/waybar-lyric/internal/lyric/formats/lrc"
	"github.com/Nadim147c/waybar-lyric/internal/lyric/models"
	"github.com/Nadim147c/waybar-lyric/internal/lyric/provider"
	"github.com/Nadim147c/waybar-lyric/internal/player"
)

// Provider is a lyrics provider that gets lyrics from touan's asText metadata.
var Provider = provider.NewProvider("local .lrc file",
	func(ctx context.Context, metadata *player.Metadata) (lyrics models.Lyrics, score float64, err error) {
		lyrics.Metadata = metadata

		if metadata.URL.Scheme != "file" {
			return lyrics, score, models.ErrLyricsNotFound
		}

		path := metadata.URL.Path
		ext := filepath.Ext(path)

		// lyrics file path
		lrcFile := path[:len(path)-len(ext)] + ".lrc"

		f, err := os.Open(lrcFile)
		if err != nil {
			return
		}
		defer f.Close()

		lyrics.Lines, err = lrc.Parse(f)

		// Match score is always max since player ensure lyrics belongs to the track
		const MatchScore = 1.0
		score = provider.CalculateLyricsScore(lyrics.Lines) + MatchScore

		return
	})
