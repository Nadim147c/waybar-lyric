package astext

import (
	"context"
	"os"
	"path/filepath"

	"github.com/Nadim147c/waybar-lyric/internal/lyric/models"
	"github.com/Nadim147c/waybar-lyric/internal/lyric/provider"
	"github.com/Nadim147c/waybar-lyric/internal/player"
)

// Provider is a lyrics provider that gets lyrics from touan's asText metadata.
var Provider = provider.NewProvider("local lrc file",
	func(_ context.Context, metadata *player.Metadata) (lyrics models.Lyrics, err error) {
		lyrics.Metadata = metadata
		lyrics.NoCache = true

		if metadata.URL.Scheme != "file" {
			return lyrics, models.ErrLyricsNotFound
		}

		path := metadata.URL.Path
		ext := filepath.Ext(path)

		// lyrics file path
		lrcFile := path[:len(path)-len(ext)] + ".lrc"

		b, err := os.ReadFile(lrcFile)
		if err != nil {
			return
		}

		lyrics.Lines, err = provider.ParseText(string(b))
		return
	})
