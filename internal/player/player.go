package player

import (
	"errors"
	"log/slog"
	"net/url"
	"regexp"
	"strings"

	"github.com/Nadim147c/go-mpris"
	"github.com/godbus/dbus/v5"
)

var (
	// ErrNoPlayerVolume when failed to get player volume.
	ErrNoPlayerVolume = errors.New("failed to get player volume")
	// ErrNoArtists when failed to get artists.
	ErrNoArtists = errors.New("failed to get artists")
	// ErrNoTitle when failed to get title.
	ErrNoTitle = errors.New("failed to get title")
	// ErrNoID when failed to get id.
	ErrNoID = errors.New("failed to get track id")
	// ErrNoLength when mpris track length is 0.
	ErrNoLength = errors.New("track length is empty")
	// ErrNoPlayer when there is no active mpris player.
	ErrNoPlayer = errors.New("no preferred player")
)

// Select selects correct parses for player.
func Select(conn *dbus.Conn) (*mpris.Player, error) {
	players, err := mpris.List(conn)
	if err != nil {
		return nil, err
	}
	slog.Debug("Player names", "players", players)

	if len(players) == 0 {
		return nil, ErrNoPlayer
	}

	for _, playerName := range players {
		slog.Debug("Checking player url", "for", playerName)
		player := mpris.New(conn, playerName)

		if hasAlbumAndArtists(player) {
			return player, nil
		}

		// Fallback: Firefox only if URL is on music.youtube.com or open.spotify.com
		if !strings.Contains(strings.ToLower(playerName), "firefox") {
			continue
		}

		rawURL, err := player.GetURL()
		if err != nil || rawURL == "" {
			slog.Debug("Checking player url", "for", "firefox")
			continue
		}
		trackURL, err := url.Parse(rawURL)
		if err != nil {
			continue
		}
		host := strings.ToLower(trackURL.Host)
		if strings.Contains(host, "music.youtube.com") ||
			strings.Contains(host, "open.spotify.com") {
			slog.Debug("Player selected", "name", "firefox")
			return player, nil
		}
	}

	return nil, ErrNoPlayer
}

func should[T any](v T, _ error) T { return v }

// Precompiled regex patterns.
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
	s := strings.ReplaceAll(artist, ", ", ", ")
	s = strings.ReplaceAll(s, " & ", ", ")
	s = strings.ReplaceAll(s, ".", "")
	s = strings.ReplaceAll(s, "和", ", ")
	s = reParen1.ReplaceAllString(s, "")
	s = reParen2.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}

// Parse takes *mpris.Player of supported play and return *Metadata.
func Parse(player *mpris.Player) (*Metadata, error) {
	meta, err := player.GetMetadata()
	if err != nil {
		return nil, err
	}
	for k, v := range meta {
		slog.Debug("MPRIS", k, v)
	}

	shuffle := should(player.GetShuffle())
	cover := should(player.GetArtURL())
	volume := should(player.GetVolume())

	album, err := player.GetAlbum()
	if err != nil {
		return nil, err
	}

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
	if length == 0 {
		return nil, ErrNoLength
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

	trackid := should(player.GetTrackID())

	metadata := &Metadata{
		Artist:    normalizeArtist(artist),
		Artists:   artistList,
		Title:     normalizeTitle(title),
		RawArtist: artist,
		RawTitle:  title,
		Player:    player.GetName(),
		Album:     album,
		Cover:     cover,
		ID:        string(trackid),
		Length:    length,
		Metadata:  meta,
		Shuffle:   shuffle,
		Status:    status,
		URL:       trackURL,
		Volume:    volume,
		Position:  0, // will be updated by UpdatePosition
	}

	metadata.ID = computeID(player, metadata)

	err = metadata.UpdatePosition(player)
	return metadata, err
}
