package models

import (
	"errors"
	"fmt"
	"time"

	"github.com/Nadim147c/waybar-lyric/internal/player"
)

// Line is a line of synchronized lyrics
type Line struct {
	Timestamp time.Duration `json:"time"`
	Text      string        `json:"line"`
}

// Lines is a slice of Line
type Lines []Line

// Lyrics is the synchronized structured lyrics
type Lyrics struct {
	Metadata *player.Metadata `json:"metadata,omitempty"`
	Lines    Lines            `json:"lyrics"`
}

var (
	// ErrLyricsNotFound indicates that the requested lyrics could not be found.
	ErrLyricsNotFound = errors.New("lyrics not found")
	// ErrLyricsNotSynced indicates that the lyrics are available but not
	// time-synchronized.
	ErrLyricsNotSynced = errors.New("lyrics is not synced")
)

// ErrLyricsMatchScore is error when lyrics response does not satisfies the
// minimum matching score required.
type ErrLyricsMatchScore struct {
	Score, Threshold float64
}

func (e *ErrLyricsMatchScore) Error() string {
	return fmt.Sprintf("insufficient lyrics match score: %.2f < %.2f", e.Score, e.Threshold)
}
