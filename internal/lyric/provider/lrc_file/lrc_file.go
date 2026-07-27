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
	func(ctx context.Context, metadata *player.Metadata) (models.Lyrics, error) {
		if metadata.URL.Scheme != "file" {
			return models.Lyrics{}, models.ErrLyricsNotFound
		}

		path := metadata.URL.Path
		ext := filepath.Ext(path)

		// lyrics file path
		lrcFile := path[:len(path)-len(ext)] + ".lrc"

		f, err := os.Open(lrcFile)
		if err != nil {
			return models.Lyrics{}, err
		}
		defer f.Close()

		lines, err := lrc.Parse(f)
		if err != nil {
			return models.Lyrics{}, err
		}

		const score = 1.0

		return models.Lyrics{Lines: lines, Score: score}, nil //nolint
	})
