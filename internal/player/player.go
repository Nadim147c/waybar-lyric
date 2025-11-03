package player

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"path"
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
func Hash(v ...any) string {
	h := sha256.New()
	fmt.Fprint(h, v...)
	hash := h.Sum(nil)
	return base64.RawURLEncoding.EncodeToString(hash)
}

// trackIDFunc: uses mpris:trackid as ID source
func trackIDFunc(p *mpris.Player) (string, error) {
	meta, err := p.GetMetadata()
	if err != nil {
		return "", err
	}
	val, ok := meta["mpris:trackid"]
	if !ok {
		return "", ErrNoID
	}

	return Hash(val), nil
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
	pu := should(url.Parse(urlStr))

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

	artist := artistList[0]

	title, err := player.GetTitle()
	if err != nil {
		return nil, err
	}

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
		URL:      pu,
		Volume:   volume,
	}

	err = info.UpdatePosition(player)
	return info, err
}
