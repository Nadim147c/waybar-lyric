package lyric

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/Nadim147c/waybar-lyric/internal/player"
)

// CacheSize is the max amount lyrics to save into the cache
const CacheSize = 20

// Store is in-memorey lyrics cache
var Store = NewCache()

// Cache is used to cache lyrics in memory
type Cache struct {
	mu    sync.Mutex
	store map[string]Lyrics
}

// NewCache creates a new instance of Cachhe
func NewCache() *Cache {
	c := new(Cache)
	c.store = make(map[string]Lyrics, CacheSize)
	return c
}

// NotFound saves a empty lyrics to Cache
func (s *Cache) NotFound(id string) {
	l := Lyrics{Metadata: &player.Metadata{ID: id}}
	s.Save(l)
}

// Save saves lyrics to Cache
func (s *Cache) Save(lyrics Lyrics) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.store) == CacheSize {
		clear(s.store) // clear in memory cache
	}

	s.store[lyrics.Metadata.ID] = lyrics
	return s.saveCache(lyrics)
}

// Load loads lyrics from Cache
func (s *Cache) Load(id string) (Lyrics, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.store[id]
	if !ok {
		return s.loadCache(id)
	}
	return v, nil
}

func (s *Cache) getCacheDir() (string, error) {
	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("failed to find cache directory: %v", err)
	}

	return filepath.Join(userCacheDir, "waybar-lyric"), nil
}

// SaveCache saves the lyrics to cache
func (s *Cache) saveCache(lyrics Lyrics) error {
	cacheDir, err := s.getCacheDir()

	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return err
	}

	cachePath := filepath.Join(cacheDir, lyrics.Metadata.ID+".json")

	file, err := os.Create(cachePath)
	if err != nil {
		return err
	}
	defer file.Close()
	return json.NewEncoder(file).Encode(lyrics)
}

// LoadCache loads the lyrics from cache
func (s *Cache) loadCache(id string) (Lyrics, error) {
	var lyrics Lyrics

	cacheDir, err := s.getCacheDir()
	if err != nil {
		return lyrics, err
	}

	cachePath := filepath.Join(cacheDir, id+".json")

	file, err := os.Open(cachePath)
	if err != nil {
		return lyrics, err
	}
	defer file.Close()

	err = json.NewDecoder(file).Decode(&lyrics)
	return lyrics, err
}
