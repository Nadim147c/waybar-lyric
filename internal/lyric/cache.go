package lyric

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
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
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.store) == CacheSize {
		clear(s.store) // clear in memory cache
	}

	s.store[id] = Lyrics{}
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

	if v, ok := s.store[id]; ok {
		return v, nil
	}

	lyrics, err := s.loadCache(id)
	if err != nil {
		return lyrics, err
	}

	if len(s.store) == CacheSize {
		slog.Debug("Flushing memory cache")
		clear(s.store) // clear in memory cache
	}

	s.store[id] = lyrics

	return lyrics, nil
}

func (s *Cache) getCacheDir() (string, error) {
	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("failed to find cache directory: %v", err)
	}

	return filepath.Join(userCacheDir, "waybar-lyric"), nil
}

// CacheExtension is the extension use for cache files
const CacheExtension = ".json.gz"

// SaveCache saves the lyrics to cache
func (s *Cache) saveCache(lyrics Lyrics) error {
	cacheDir, err := s.getCacheDir()

	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return err
	}

	cachePath := filepath.Join(cacheDir, lyrics.Metadata.ID+CacheExtension)

	file, err := os.Create(cachePath)
	if err != nil {
		return err
	}
	defer file.Close()

	gz := gzip.NewWriter(file)
	defer gz.Close()

	if err := json.NewEncoder(gz).Encode(&lyrics); err != nil {
		return err
	}

	return gz.Flush()
}

// LoadCache loads the lyrics from cache
func (s *Cache) loadCache(id string) (Lyrics, error) {
	var lyrics Lyrics

	cacheDir, err := s.getCacheDir()
	if err != nil {
		return lyrics, err
	}

	cachePath := filepath.Join(cacheDir, id+CacheExtension)

	file, err := os.Open(cachePath)
	if err != nil {
		return lyrics, err
	}
	defer file.Close()

	gz, err := gzip.NewReader(file)
	if err != nil {
		return lyrics, err
	}
	defer gz.Close()

	err = json.NewDecoder(gz).Decode(&lyrics)
	return lyrics, err
}
