package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/Pauloo27/go-mpris"
	"github.com/fred1268/go-clap/clap"
	"github.com/godbus/dbus/v5"
)

type Cli struct {
	Init      bool `clap:"--init"`
	Toggle    bool `clap:"--toggle"`
	MaxLength int  `clap:"--max-length"`
	Help      bool `clap:"--help,-h"`
}

func main() {
	args := make([]string, 0)
	if len(os.Args) > 1 {
		args = os.Args[1:]
	}
	var err error
	cli := &Cli{MaxLength: 150}
	if _, err = clap.Parse(args, cli); err != nil {
		Log(err)
		os.Exit(1)
	}

	if cli.Help {
		os.Exit(0)
	}

	Execute(*cli)
}

func Log(a ...any) {
	fmt.Fprintln(os.Stderr, a...)
}

func truncate(input string, limit int) string {
	if len(input) <= limit {
		return input
	}

	if limit > 3 {
		return input[:limit-3] + "..."
	}

	return input[:limit]
}

func Execute(cli Cli) {
	if cli.Init {
		fmt.Print(`Put the following object in your waybar config:

"custom/lyrics": {
	"interval": 1,
	"signal": 4,
	"return-type": "json",
	"format": "{icon} {0}",
	"format-icons": {
		"playing": "",
		"paused": "",
		"lyric": "",
	},
	"exec-if": "which waytune",
	"exec": "waytune lyrics --max-length 100",
	"on-click": "waytune lyrics --toggle",
},
`)
		os.Exit(0)
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetEscapeHTML(false)

	lockFile := filepath.Join(os.TempDir(), "WayTune-Lyrics.lock")
	file, err := os.OpenFile(lockFile, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		Log("Failed to open or create lock file:", err)
		os.Exit(1)
	}
	defer file.Close()

	err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		if err == syscall.EWOULDBLOCK {
			Log("Another instance of the CLI is already running. Exiting.")
			os.Exit(0)
		}
		Log("Failed to acquire lock:", err)
		os.Exit(1)
	}

	defer os.Remove(lockFile)

	conn, err := dbus.SessionBus()
	if err != nil {
		Log(err)
		os.Exit(1)
	}
	names, err := mpris.List(conn)
	if err != nil {
		Log(err)
		os.Exit(1)
	}

	searchTerm := "spotify"
	var playerName string
	for _, name := range names {
		if strings.Contains(strings.ToLower(name), strings.ToLower(searchTerm)) {
			playerName = name
			break
		}
	}

	if playerName == "" {
		Log("failed to find player")
		os.Exit(1)
	}

	player := mpris.New(conn, playerName)

	if cli.Toggle {
		player.PlayPause()
		UpdateWaybar()
		os.Exit(0)
	}

	meta, err := player.GetMetadata()
	if err != nil {
		Log(err)
		os.Exit(1)
	}

	status, err := player.GetPlaybackStatus()
	if err != nil {
		Log(err)
		os.Exit(1)
	}

	artist := meta["xesam:artist"].Value().([]string)[0]
	title := meta["xesam:title"].Value().(string)
	album := meta["xesam:album"].Value().(string)

	if title == "" || artist == "" {
		os.Exit(1)
	}

	length := time.Duration(meta["mpris:length"].Value().(uint64)) * time.Microsecond

	pos, err := player.GetPosition()
	if err != nil {
		os.Exit(1)
	}
	position := time.Duration(pos * float64(time.Second))

	if status == "Paused" {
		encoder.Encode(Lyrics{
			Text:       fmt.Sprintf("%s - %s", artist, title),
			Class:      "info",
			Alt:        "paused",
			Tooltip:    "",
			Percentage: int(100 * position / length),
		})
		os.Exit(0)
	}

	if status == "Stopped" {
		os.Exit(0)
	}

	queryParams := url.Values{}
	queryParams.Set("track_name", title)
	queryParams.Set("artist_name", artist)
	if album != "" {
		queryParams.Set("album_name", album)
	}
	if length != 0 {
		queryParams.Set("duration", fmt.Sprintf("%.2f", length.Seconds()))
	}
	params := queryParams.Encode()

	url := fmt.Sprintf("%s?%s", LRCLIB_ENDPOINT, params)
	uri := filepath.Base(meta["mpris:trackid"].Value().(string))

	lyrics, err := FetchLyrics(url, uri)
	if err != nil {
		Log(err)
		encoder.Encode(Lyrics{
			Text:       fmt.Sprintf("%s - %s", artist, title),
			Class:      "info",
			Alt:        "playing",
			Percentage: int(100 * position / length),
		})
		os.Exit(0)
	}

	var idx int
	for i, line := range lyrics {
		if position < line.Timestamp {
			break
		}
		idx = i
	}

	currentLine := lyrics[idx].Text

	if currentLine != "" {
		start := idx - 2
		if start < 0 {
			start = 0
		}

		end := idx + 5
		if end > len(lyrics) {
			end = len(lyrics)
		}

		tooltipLyrics := lyrics[start:end]
		var tooltip strings.Builder
		for i, ttl := range tooltipLyrics {
			lineText := ttl.Text
			if start+i == idx {
				tooltip.WriteString("> ")
			}
			tooltip.WriteString(lineText + "\n")
		}

		encoder.Encode(Lyrics{
			Text:       truncate(currentLine, cli.MaxLength),
			Class:      "lyric",
			Alt:        "lyric",
			Tooltip:    strings.TrimSpace(tooltip.String()),
			Percentage: int(100 * position / length),
		})
		os.Exit(0)
	}

	encoder.Encode(Lyrics{
		Text:       fmt.Sprintf("%s - %s", artist, title),
		Class:      "info",
		Alt:        "playing",
		Tooltip:    "",
		Percentage: int(100 * position / length),
	})
}
