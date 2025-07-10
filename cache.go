package main

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

type StoreValue struct {
	LastAccess time.Time
	Lyrics     Lyrics
}

// Store is used to cache lyrics in memory
type Store struct {
	mu   sync.RWMutex // Using RWMutex for better read performance
	data map[string]*StoreValue
}

// NewStore creates a new initialized Store
func NewStore() *Store {
	return &Store{
		data: make(map[string]*StoreValue),
	}
}

// Save saves lyrics to Store
func (s *Store) Save(id string, lyrics Lyrics) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[id] = &StoreValue{
		LastAccess: time.Now(),
		Lyrics:     lyrics,
	}
}

// Load loads lyrics from Store
func (s *Store) Load(key string) (Lyrics, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, exists := s.data[key]
	if !exists {
		return nil, false
	}
	v.LastAccess = time.Now() // Update last access time
	return v.Lyrics, true
}

// Cleanup runs a blocking loop that periodically removes unused entries
// until the context is canceled.
func (s *Store) Cleanup(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return // Exit when context is canceled
		case <-ticker.C:
			s.cleanupExpired(interval)
		}
	}
}

// cleanupExpired removes entries not accessed within the interval
func (s *Store) cleanupExpired(threshold time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for key, value := range s.data {
		if time.Since(value.LastAccess) > threshold {
			delete(s.data, key)
		}
	}
}

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

func SaveCache(lines []LyricLine, filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	for line := range slices.Values(lines) {
		_, err := fmt.Fprintf(file, "%d,%s\n", line.Timestamp, line.Text)
		if err != nil {
			return err
		}
	}
	return nil
}

func LoadCache(filePath string) ([]LyricLine, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lyrics []LyricLine
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ",", 2)
		if len(parts) != 2 {
			continue // Skip invalid lines
		}

		ts, err := strconv.Atoi(parts[0])
		if err != nil {
			return nil, err
		}

		timestamp := time.Duration(ts)
		text := strings.TrimSpace(parts[1])

		lyric := LyricLine{Timestamp: timestamp, Text: text}
		lyrics = append(lyrics, lyric)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if len(lyrics) == 0 {
		return nil, fmt.Errorf("Number of line found is zero.")
	}

	return lyrics, nil
}
