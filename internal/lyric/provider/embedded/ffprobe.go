package embedded

import (
	"context"
	"encoding/json"
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
	func(ctx context.Context, metadata *player.Metadata) (lyrics models.Lyrics, score float64, err error) {
		lyrics.Metadata = metadata

		if metadata.URL.Scheme != "file" {
			return lyrics, score, models.ErrLyricsNotFound
		}

		ffprobe, err := exec.LookPath("ffprobe")
		if err != nil {
			return
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
			return
		}

		var result ffprobeOutput
		err = json.Unmarshal(output, &result)
		if err != nil {
			return
		}

		tags := map[string]string{}
		for _, stream := range result.Streams {
			maps.Copy(tags, stream.Tags)
		}

		keys := []string{"LYRICS", "SYLT", "USLT", "lyrics", "lyrics-eng"}
		for _, key := range keys {
			value, ok := tags[key]
			if !ok || value == "" {
				continue
			}
			lines, err := lrc.ParseText(value)
			if err != nil {
				continue
			}
			lyrics.Lines = lines
			// Match score is always max since player ensure lyrics belongs to the track
			const MatchScore = 1.0
			score = provider.CalculateLyricsScore(lyrics.Lines) + MatchScore
			return lyrics, score, nil
		}

		return lyrics, score, models.ErrLyricsNotFound
	})
