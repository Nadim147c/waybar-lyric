package main

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"log/slog"
	"slices"
	"time"

	"github.com/Nadim147c/go-mpris"
	"github.com/godbus/dbus/v5"
)

var (
	ErrNoPlayerVolume = errors.New("failed to get player volume")
	ErrNoArtists      = errors.New("failed to get artists")
	ErrNoTitle        = errors.New("failed to get title")
)

// PlayerParser parses player information from mpris metadata
type PlayerParser func(*mpris.Player) (*PlayerInfo, error)

var supportedPlayers = map[string]PlayerParser{
	"spotify":          DefaultParser,
	"YoutubeMusic":     YouTubeMusicParser,
	"amarok":           DefaultParser,
	"io.bassi.Amberol": DefaultParser,
}

// SelectPlayer selects correct parses for player
func SelectPlayer(conn *dbus.Conn) (*mpris.Player, PlayerParser, error) {
	players, err := mpris.List(conn)
	if err != nil {
		return nil, nil, err
	}
	slog.Debug("Player names", "players", players)

	if len(players) == 0 {
		return nil, nil, errors.New("No player exists")
	}

	for name, parser := range supportedPlayers {
		for player := range slices.Values(players) {
			if mpris.BaseInterface+"."+name == player {
				return mpris.New(conn, player), parser, nil
			}
		}
	}

	return nil, nil, errors.New("No player exists")
}

// StringToMD5 converts a string to its MD5 hash
func StringToMD5(s string) string {
	hash := md5.Sum([]byte(s))
	return hex.EncodeToString(hash[:])
}

// DefaultParser takes *mpris.Player of spotify and return *PlayerInfo
func DefaultParser(player *mpris.Player) (*PlayerInfo, error) {
	meta, err := player.GetMetadata()
	if err != nil {
		return nil, err
	}
	for k, v := range meta {
		slog.Debug("MPRIS", k, v)
	}

	shuffle, err := player.GetShuffle()
	if err != nil {
		return nil, err
	}

	status, err := player.GetPlaybackStatus()
	if err != nil {
		return nil, err
	}

	position, err := player.GetPosition()
	if err != nil {
		return nil, err
	}

	// Cover is optional
	cover, _ := meta["mpris:artUrl"].Value().(string)

	volume, err := player.GetVolume()
	if err != nil {
		return nil, ErrNoPlayerVolume
	}

	artistList, ok := meta["xesam:artist"].Value().([]string)
	if !ok || len(artistList) == 0 {
		return nil, ErrNoArtists
	}
	artist := artistList[0]

	title, ok := meta["xesam:title"].Value().(string)
	if !ok || title == "" {
		return nil, ErrNoArtists
	}

	id, ok := meta["mpris:trackid"].Value().(string)
	if !ok || id == "" {
		id = StringToMD5(artist + title)
	}

	album, _ := meta["xesam:album"].Value().(string)
	length, err := player.GetLength()
	if err != nil {
		return nil, err
	}

	return &PlayerInfo{
		Player:   player.GetName(),
		ID:       id,
		Artist:   artist,
		Title:    title,
		Album:    album,
		Status:   status,
		Volume:   volume,
		Position: position,
		Length:   length,
		Shuffle:  shuffle,
		Cover:    cover,
	}, nil
}

// YouTubeMusicParser parses mpris metadata for YouTubeMusic player
// source: https://github.com/th-ch/youtube-music
func YouTubeMusicParser(player *mpris.Player) (*PlayerInfo, error) {
	info, err := DefaultParser(player)
	if err != nil {
		return nil, err
	}
	info.ID = StringToMD5(info.ID)

	// HACK: YoutubeMusic dbus position ≈ 1.1 slow
	info.Position += 1100 * time.Millisecond

	return info, nil
}
