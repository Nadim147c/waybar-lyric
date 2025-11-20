package player

import (
	"encoding/hex"
	"errors"
	"fmt"
	"hash/fnv"
	"log/slog"
	"net/url"
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/Nadim147c/go-mpris"
	"github.com/godbus/dbus/v5"
	"github.com/spf13/cast"
)

var (
	// ErrNoPlayerVolume when failed to get player volume
	ErrNoPlayerVolume = errors.New("failed to get player volume")
	// ErrNoArtists when failed to get artists
	ErrNoArtists = errors.New("failed to get artists")
	// ErrNoTitle when failed to get title
	ErrNoTitle = errors.New("failed to get title")
	// ErrNoID when failed to get id
	ErrNoID = errors.New("failed to get track id")
)

// Parser parses player information from mpris metadata
type Parser func(*mpris.Player) (*Metadata, error)

// IDFunc extracts a stable ID for a player.
type IDFunc func(p *mpris.Player) (string, error)

// Hash return sha256 hash for given string
func Hash(v ...string) string {
	h := fnv.New128a()

	switch len(v) {
	case 0:
		panic("nothing to hash")
	case 1:
		fmt.Fprintf(h, "%s", v[0])
	default:
		fmt.Fprint(h, v[0])
		for _, e := range v {
			fmt.Fprintf(h, ":%s", e)
		}
	}

	hash := h.Sum(nil)
	return hex.EncodeToString(hash)
}

// artistTitleFunc: uses artist+title combo as ID source
func artistTitleFunc(p *mpris.Player) (string, error) {
	artists, err := p.GetArtist()
	if err != nil || len(artists) == 0 {
		return "", ErrNoArtists
	}
	artist := artists[0]

	title, err := p.GetTitle()
	if err != nil || title == "" {
		return "", ErrNoTitle
	}

	return Hash(artist, ":", title), nil
}

// urlIDFunc: derive ID from URL for fallback players like Firefox
func urlIDFunc(p *mpris.Player) (string, error) {
	u, err := p.GetURL()
	if err != nil || u == "" {
		return "", ErrNoID
	}

	parsed, err := url.Parse(u)
	if err != nil {
		return "", err
	}

	host := strings.ToLower(parsed.Host)

	// Only allow music.youtube.com and open.spotify.com
	if !(strings.Contains(host, "music.youtube.com") || strings.Contains(host, "open.spotify.com")) {
		return "", ErrNoID
	}

	id := ""
	if strings.Contains(host, "music.youtube.com") {
		id = parsed.Query().Get("v") // ?v=xxx
	} else if strings.Contains(host, "open.spotify.com") {
		id = path.Base(parsed.Path) // /track/xxx
	}

	if id == "" {
		return "", ErrNoID
	}

	return Hash(host, ":", id), nil
}

type players struct {
	name   string
	idFunc IDFunc
}

var supportedPlayers = []players{
	{"spotify", urlIDFunc},
	{"YoutubeMusic", urlIDFunc},
	{"amarok", artistTitleFunc},
	{"io.bassi.Amberol", artistTitleFunc},
}

// Select selects correct parses for player
func Select(conn *dbus.Conn) (*mpris.Player, Parser, error) {
	players, err := mpris.List(conn)
	if err != nil {
		return nil, nil, err
	}
	slog.Debug("Player names", "players", players)

	if len(players) == 0 {
		return nil, nil, errors.New("No player exists")
	}

	// First: explicitly supported players
	for p := range slices.Values(supportedPlayers) {
		playerName := mpris.BaseInterface + "." + p.name
		if slices.Contains(players, playerName) {
			slog.Debug("Player selected", "name", playerName)
			return mpris.New(
					conn,
					playerName,
				), parserWithIDFunc(
					DefaultParser,
					p.idFunc,
				), nil
		}
	}

	// Fallback: Firefox only if URL is on music.youtube.com or open.spotify.com
	for _, playerName := range players {
		if !strings.Contains(strings.ToLower(playerName), "firefox") {
			continue
		}
		slog.Debug("Checking player url", "for", "firefox")
		fp := mpris.New(conn, playerName)
		u, err := fp.GetURL()
		if err != nil || u == "" {
			slog.Debug("Checking player url", "for", "firefox")
			continue
		}
		pu, err := url.Parse(u)
		if err != nil {
			continue
		}
		host := strings.ToLower(pu.Host)
		if strings.Contains(host, "music.youtube.com") ||
			strings.Contains(host, "open.spotify.com") {
			slog.Debug("Player selected", "name", "firefox")
			return fp, parserWithIDFunc(DefaultParser, urlIDFunc), nil
		}
	}

	return nil, nil, errors.New("No player exists")
}

func parserWithIDFunc(f Parser, i IDFunc) Parser {
	return func(p *mpris.Player) (*Metadata, error) {
		info, err := f(p)
		if err != nil {
			return info, err
		}
		id, err := i(p)
		if err != nil {
			return info, err
		}

		info.ID = id
		return info, nil
	}
}

func should[T any](v T, _ error) T {
	return v
}

// Precompiled regex patterns
var (
	reParen1 = regexp.MustCompile(`\(.*\)`)
	reParen2 = regexp.MustCompile(`（.*）`)
	reQuote1 = regexp.MustCompile(`「.*」`)
	reQuote2 = regexp.MustCompile(`『.*』`)
	reAngle1 = regexp.MustCompile(`<.*>`)
	reAngle2 = regexp.MustCompile(`《.*》`)
	reAngle3 = regexp.MustCompile(`〈.*〉`)
	reAngle4 = regexp.MustCompile(`＜.*＞`)
)

func normalizeTitle(title string) string {
	s := title
	s = reParen1.ReplaceAllString(s, "")
	s = reParen2.ReplaceAllString(s, "")
	s = reQuote1.ReplaceAllString(s, "")
	s = reQuote2.ReplaceAllString(s, "")
	s = reAngle1.ReplaceAllString(s, "")
	s = reAngle2.ReplaceAllString(s, "")
	s = reAngle3.ReplaceAllString(s, "")
	s = reAngle4.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}

func normalizeArtist(artist string) string {
	s := strings.ReplaceAll(artist, ", ", "、")
	s = strings.ReplaceAll(s, " & ", "、")
	s = strings.ReplaceAll(s, ".", "")
	s = strings.ReplaceAll(s, "和", "、")
	s = reParen1.ReplaceAllString(s, "")
	s = reParen2.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}

// DefaultParser takes *mpris.Player of spotify and return *PlayerInfo
func DefaultParser(player *mpris.Player) (*Metadata, error) {
	meta, err := player.GetMetadata()
	if err != nil {
		return nil, err
	}
	for k, v := range meta {
		slog.Debug("MPRIS", k, v)
	}

	shuffle := should(player.GetShuffle())
	cover := should(player.GetCoverURL())
	volume := should(player.GetVolume())
	album := should(player.GetAlbum())

	urlStr := should(player.GetURL())
	trackURL := should(NewURL(urlStr))

	status, err := player.GetPlaybackStatus()
	if err != nil {
		return nil, err
	}

	length, err := player.GetLength()
	if err != nil {
		return nil, err
	}

	artistList, err := player.GetArtist()
	if err != nil {
		return nil, err
	}

	if len(artistList) == 0 {
		return nil, ErrNoArtists
	}

	artist := normalizeArtist(artistList[0])

	title, err := player.GetTitle()
	if err != nil {
		return nil, err
	}

	title = normalizeTitle(title)

	if title == "" {
		return nil, ErrNoArtists
	}

	idValue, _ := meta["mpris:trackid"]
	trackid := cast.ToString(idValue.Value())

	info := &Metadata{
		Player:   player.GetName(),
		Album:    album,
		Artist:   artist,
		Cover:    cover,
		ID:       trackid,
		Length:   length,
		Metadata: meta,
		Shuffle:  shuffle,
		Status:   status,
		Title:    title,
		URL:      trackURL,
		Volume:   volume,
	}

	err = info.UpdatePosition(player)
	return info, err
}
