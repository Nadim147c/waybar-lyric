package main

import "time"

type (
	Lyrics struct {
		Text       string      `json:"text"`
		Class      interface{} `json:"class"`
		Alt        string      `json:"alt"`
		Tooltip    string      `json:"tooltip"`
		Percentage int         `json:"percentage"`
	}

	LrcLibResponse struct {
		ID           int     `json:"id"`
		Name         string  `json:"name"`
		TrackName    string  `json:"trackName"`
		ArtistName   string  `json:"artistName"`
		AlbumName    string  `json:"albumName"`
		Duration     float64 `json:"duration"`
		Instrumental bool    `json:"instrumental"`
		PlainLyrics  string  `json:"plainLyrics"`
		SyncedLyrics string  `json:"syncedLyrics"`
	}

	LyricLine struct {
		Timestamp time.Duration
		Text      string
	}
)
