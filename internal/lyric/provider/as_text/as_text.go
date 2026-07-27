package astext

import (
	"context"

	"github.com/Nadim147c/go-mpris"
	"github.com/Nadim147c/waybar-lyric/internal/lyric/formats/lrc"
	"github.com/Nadim147c/waybar-lyric/internal/lyric/models"
	"github.com/Nadim147c/waybar-lyric/internal/lyric/provider"
	"github.com/Nadim147c/waybar-lyric/internal/player"
	"github.com/spf13/cast"
)

// Provider is a lyrics provider that gets lyrics from touan's asText metadata.
var Provider = provider.NewProvider("asText metadata parser",
	func(ctx context.Context, metadata *player.Metadata) (models.Lyrics, error) {
		asText, err := metadata.Metadata.Get(mpris.KeyAsText)
		if err != nil {
			return models.Lyrics{}, err
		}

		text, err := cast.ToStringE(asText)
		if err != nil {
			return models.Lyrics{}, err
		}

		lines, err := lrc.ParseText(text)
		if err != nil {
			return models.Lyrics{}, err
		}

		const score = 1.0

		return models.Lyrics{Lines: lines, Score: score}, nil //nolint
	})
