package embedded

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"os/exec"

	"github.com/Nadim147c/waybar-lyric/internal/lyric/formats/lrc"
	"github.com/Nadim147c/waybar-lyric/internal/lyric/models"
	"github.com/Nadim147c/waybar-lyric/internal/lyric/provider"
	"github.com/Nadim147c/waybar-lyric/internal/player"
)

type ffprobeOutput struct {
	Streams []streams `json:"streams"`
}
type streams struct {
	Tags map[string]string `json:"tags"`
}

// Provider is a lyrics provider that gets lyrics from `LYRICS` tags of local
// file.
var Provider = provider.NewProvider("embedded lyrics in audio file",
	func(ctx context.Context, metadata *player.Metadata) (models.Lyrics, error) {
		if metadata.URL.Scheme != "file" {
			return models.Lyrics{}, models.ErrLyricsNotFound
		}

		ffprobe, err := exec.LookPath("ffprobe")
		if err != nil {
			return models.Lyrics{}, err
		}

		path := metadata.URL.Path
		output, err := exec.CommandContext(
			ctx, ffprobe,
			"-v", "quiet",
			"-show_streams",
			"-print_format", "json",
			path,
		).Output()
		if err != nil {
			return models.Lyrics{}, err
		}

		var result ffprobeOutput
		err = json.Unmarshal(output, &result)
		if err != nil {
			return models.Lyrics{}, err
		}

		tags := map[string]string{}
		for _, stream := range result.Streams {
			maps.Copy(tags, stream.Tags)
		}

		var errs []error

		keys := []string{"LYRICS", "SYLT", "USLT", "lyrics", "lyrics-eng"}
		for _, key := range keys {
			value, ok := tags[key]
			if !ok || value == "" {
				continue
			}
			lines, err := lrc.ParseText(value)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			// Match score is always max since player ensure lyrics belongs to the track
			const MatchScore = 1.0
			score := provider.CalculateLyricsScore(lines) + MatchScore

			return models.Lyrics{Lines: lines, Score: score}, nil
		}

		return models.Lyrics{}, errors.Join(errs...)
	})
