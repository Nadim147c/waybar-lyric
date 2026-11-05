// This implemetation derived from https://github.com/sonorahq/sonora/blob/e3d4adfe04cd95d9a8b470bd98abc5b2de96c76b/crates/music/src/netease/mod.rs.
// GPL-V3 https://github.com/sonorahq/sonora/blob/e3d4adfe04cd95d9a8b470bd98abc5b2de96c76b/COPYING.

package netease

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Nadim147c/waybar-lyric/internal/config"
	"github.com/Nadim147c/waybar-lyric/internal/lyric/formats/lrc"
	"github.com/Nadim147c/waybar-lyric/internal/lyric/formats/yrc"
	"github.com/Nadim147c/waybar-lyric/internal/lyric/models"
	"github.com/Nadim147c/waybar-lyric/internal/lyric/provider"
	"github.com/Nadim147c/waybar-lyric/internal/player"
)

const (
	// SearchEndpoint is the NetEase song search endpoint.
	SearchEndpoint = "https://music.163.com/api/search/get"
	// LyricEndpoint is the NetEase lyric endpoint.
	LyricEndpoint = "https://music.163.com/api/song/lyric/v1"

	name    = "NetEase"
	referer = "https://music.163.com"

	// candidates is how many search results get their lyrics downloaded.
	candidates = 3
)

// Provider is the NetEase Cloud Music lyrics provider. It searches for the
// track and sends one result for each of the closest matches.
var Provider = &provider.LyricProvider{
	Name: name,
	Fetch: func(ctx context.Context, wg *sync.WaitGroup, metadata *player.Metadata, out chan<- provider.Result) {
		defer wg.Done()
		songs, err := search(ctx, metadata)
		if err != nil {
			out <- provider.Result{Provider: name, Err: err} //nolint
			return
		}

		for _, s := range shortlist(songs, metadata.Length) {
			wg.Go(func() {
				var res provider.Result
				res.Provider = fmt.Sprintf("%s [%d]", name, s.ID)
				res.Lyrics, res.Err = fetchLyrics(ctx, metadata, s)
				out <- res
			})
		}
	},
}

type named struct {
	Name string `json:"name"`
}

type song struct {
	ID       int64   `json:"id"`
	Name     string  `json:"name"`
	Duration int64   `json:"duration"` // milliseconds
	Artists  []named `json:"artists"`
	Album    *named  `json:"album"`
}

func (s song) artistNames() []string {
	names := make([]string, 0, len(s.Artists))
	for _, artist := range s.Artists {
		if artist.Name != "" {
			names = append(names, artist.Name)
		}
	}
	return names
}

func (s song) albumName() string {
	if s.Album == nil {
		return ""
	}
	return s.Album.Name
}

func (s song) length() time.Duration {
	return time.Duration(s.Duration) * time.Millisecond
}

type searchResponse struct {
	Result *struct {
		Songs []song `json:"songs"`
	} `json:"result"`
}

type verse struct {
	Lyric string `json:"lyric"`
}

// text returns the lyric, or "" when there is none or it is only whitespace.
func (v *verse) text() string {
	if v == nil || strings.TrimSpace(v.Lyric) == "" {
		return ""
	}
	return v.Lyric
}

type sheet struct {
	LRC       *verse `json:"lrc"`
	YRC       *verse `json:"yrc"`
	PureMusic bool   `json:"pureMusic"`
}

// parse reads the word-timed lyrics when there are any and the plain lrc
// otherwise.
func (sh *sheet) parse() (models.Lines, error) {
	if text := sh.YRC.text(); text != "" {
		if lines, err := yrc.ParseText(text); err == nil {
			return lines, nil
		}
	}
	if text := sh.LRC.text(); text != "" {
		return lrc.ParseText(text)
	}
	return nil, models.ErrLyricsNotSynced
}

var lineReplacer = strings.NewReplacer(
	"（", "(",
	"）", ")",
	"，", ",",
	"．", ".",
	"。", ".",
	"！", "!",
	"？", "?",
)

// fixText replaces fullwidth punctuation characters with their standard ASCII
// equivalents. It returns the original string unmodified if no target
// characters are found.
func fixText(s string) string {
	if !strings.ContainsAny(s, "（），．。") {
		return s
	}
	return lineReplacer.Replace(s)
}

// lyrics returns the synced lines of the sheet. Instrumentals and sheets
// without usable synced lyrics are reported as not found.
func (sh *sheet) lyrics(s song) (models.Lines, error) {
	if sh.PureMusic {
		return nil, fmt.Errorf("netease marks the track as instrumental: %w", models.ErrLyricsNotFound)
	}

	lines, err := sh.parse()
	if err != nil {
		return nil, err
	}

	if instrumental(lines) {
		return nil, fmt.Errorf("netease lyrics are an instrumental placeholder: %w", models.ErrLyricsNotFound)
	}

	lines, ok := stripHeader(lines, s.Name, s.artistNames())
	if !ok {
		return nil, fmt.Errorf("netease lyrics hold nothing but a header: %w", models.ErrLyricsNotFound)
	}

	for i, line := range lines {
		lines[i].Text = fixText(line.Text)
		for j, word := range line.Words {
			lines[i].Words[j].Text = fixText(word.Text)
		}
	}
	return lines, nil
}

func search(ctx context.Context, metadata *player.Metadata) ([]song, error) {
	params := url.Values{}
	params.Set("s", metadata.RawTitle+" "+metadata.RawArtist)
	params.Set("type", "1")
	params.Set("limit", "5")

	var found searchResponse
	if err := getJSON(ctx, SearchEndpoint, params, &found); err != nil {
		return nil, err
	}
	if found.Result == nil || len(found.Result.Songs) == 0 {
		return nil, models.ErrSearchResultEmpty
	}
	return found.Result.Songs, nil
}

func fetchLyrics(ctx context.Context, metadata *player.Metadata, s song) (models.Lyrics, error) {
	sh, err := fetchSheet(ctx, s.ID)
	if err != nil {
		return models.Lyrics{}, err
	}

	lines, err := sh.lyrics(s)
	if err != nil {
		return models.Lyrics{}, err
	}

	score := provider.Score(metadata, provider.LyricsResult{
		Title:    s.Name,
		Artist:   strings.Join(s.artistNames(), ", "),
		Album:    s.albumName(),
		Duration: s.length(),
	})

	return models.Lyrics{Lines: lines, Score: min(score/5, 1)}, nil //nolint
}

func fetchSheet(ctx context.Context, id int64) (*sheet, error) {
	params := url.Values{}
	params.Set("id", strconv.FormatInt(id, 10))
	params.Set("cp", "false")
	for _, key := range []string{"lv", "tv", "rv", "kv", "yv", "ytv", "yrv"} {
		params.Set(key, "0")
	}

	var sh sheet
	if err := getJSON(ctx, LyricEndpoint, params, &sh); err != nil {
		return nil, err
	}
	return &sh, nil
}

func getJSON(ctx context.Context, endpoint string, params url.Values, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}

	req.URL.RawQuery = params.Encode()
	req.Header.Set("User-Agent", config.Version)
	req.Header.Set("Referer", referer)

	slog.Info("Fetching lyrics from NetEase", "url", req.URL.String())

	var client http.Client

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("cannot reach netease: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("[%d] %w", resp.StatusCode, models.ErrLyricsNotFound)
	}

	if resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected HTTP status: %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("cannot read the netease response: %w", err)
	}
	return nil
}

// shortlist keeps the few songs whose length is closest to want, comparing
// whole seconds. Without a known length the search order is kept.
func shortlist(songs []song, want time.Duration) []song {
	songs = slices.Clone(songs)
	if want > 0 {
		slices.SortStableFunc(songs, func(a, b song) int {
			return cmp.Compare(secondsApart(a, want), secondsApart(b, want))
		})
	}
	return songs[:min(len(songs), candidates)]
}

func secondsApart(s song, want time.Duration) int64 {
	diff := s.Duration/1000 - int64(want/time.Second)
	if diff < 0 {
		return -diff
	}
	return diff
}

// instrumentalMarkers is what NetEase puts in place of the lyrics of an
// instrumental, lower case.
var instrumentalMarkers = []string{"纯音乐", "请欣赏", "instrumental"}

// instrumental reports whether the lyrics are just an instrumental
// placeholder: a handful of lines that say so.
func instrumental(lines models.Lines) bool {
	if len(lines) > 4 { // the empty first line and three real ones
		return false
	}
	for _, line := range lines {
		text := strings.ToLower(line.Text)
		for _, marker := range instrumentalMarkers {
			if strings.Contains(text, marker) {
				return true
			}
		}
	}
	return false
}

// creditWords are what the label of a credit line contains, lower case.
var creditWords = []string{
	"作词", "作曲", "编曲", "制作", "监制", "录音", "混音", "母带",
	"lyrics", "lyricist", "composer", "composed", "arranger", "arranged",
	"producer", "produced", "written",
}

// stripHeader removes the title, artist and credit lines NetEase puts in
// front of the lyrics. It reports false when nothing else is left.
func stripHeader(lines models.Lines, title string, artists []string) (models.Lines, bool) {
	head := 1 // lines[0] is the empty line the parsers put in front of the lyrics
	for head < len(lines) && isHeader(lines[head].Text, title, artists) {
		head++
	}
	if head >= len(lines) {
		return nil, false
	}
	return slices.Delete(lines, 1, head), true
}

func isHeader(text, title string, artists []string) bool {
	text = strings.TrimSpace(text)

	if i := strings.IndexAny(text, ":："); i >= 0 {
		label := strings.ToLower(text[:i])
		if slices.ContainsFunc(creditWords, func(word string) bool {
			return strings.Contains(label, word)
		}) {
			return true
		}
	}

	for _, artist := range artists {
		if strings.EqualFold(text, title+" - "+artist) ||
			strings.EqualFold(text, artist+" - "+title) {
			return true
		}
	}
	return false
}
