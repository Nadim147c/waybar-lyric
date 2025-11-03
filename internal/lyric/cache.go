package lyric

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Store is in-memorey lyrics cache
var Store = store{}

// store is used to cache lyrics in memory
type store struct{ m sync.Map }

// Save saves lyrics to Store
func (s *store) NotFound(id string) {
	s.m.Store(id, Lyrics{})
}

// Save saves lyrics to Store
func (s *store) Save(id string, lyrics Lyrics) {
	s.m.Store(id, lyrics)
}

// Load loads lyrics from Store
func (s *store) Load(key string) (Lyrics, bool) {
	v, ok := s.m.Load(key)
	if !ok {
		return Lyrics{}, false
	}
	return v.(Lyrics), true
}

// Cleanup runs a blocking loop that periodically removes unused entries
// until the context is canceled.
func (s *store) Cleanup(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return // Exit when context is canceled
		case <-ticker.C:
			s.m.Clear()
		}
	}
}

// CacheDir is waybar-lyric lyrics cache dir
var CacheDir string

func init() {
	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		slog.Error("Failed to find cache directory", "error", err)
		return
	}

	CacheDir = filepath.Join(userCacheDir, "waybar-lyric")

	if err := os.MkdirAll(CacheDir, 0755); err != nil {
		slog.Error("Failed to create cache directory")
	}
}

// SaveCache saves the lyrics to cache
func SaveCache(lyrics Lyrics, filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	return json.NewEncoder(file).Encode(lyrics)
}

// LoadCache loads the lyrics from cache
func LoadCache(filePath string) (Lyrics, error) {
	var lyrics Lyrics

	file, err := os.Open(filePath)
	if err != nil {
		return lyrics, err
	}
	defer file.Close()

	err = json.NewDecoder(file).Decode(&lyrics)
	return lyrics, err
}
