package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const LRCLIB_ENDPOINT = "https://lrclib.net/api/get"

func FetchLyrics(url string, uri string) ([]LyricLine, error) {
	notFoundTempDir := filepath.Join(os.TempDir(), "WayTune")
	lyricsNotFoundFile := filepath.Join(notFoundTempDir, uri+"-not-found")

	if _, err := os.Stat(lyricsNotFoundFile); err == nil {
		return nil, fmt.Errorf("Lyrics not found (cached)")
	}

	userCacheDir, _ := os.UserCacheDir()
	cacheDir := filepath.Join(userCacheDir, "WayTune-Lyrics")

	uri = strings.ReplaceAll(uri, "/", "-")
	cacheFile := filepath.Join(cacheDir, uri+".csv")

	if cahcedLyrics, err := LoadCache(cacheFile); err == nil {
		return cahcedLyrics, nil
	} else {
		Log(err)
	}

	Log("Fetching lyrics from LRCLIB:", url)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch lyrics: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		os.WriteFile(lyricsNotFoundFile, []byte(url), 644)
		return nil, fmt.Errorf("Lyrics not found")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected HTTP status: %d", resp.StatusCode)
	}

	var resJson LrcLibResponse
	err = json.NewDecoder(resp.Body).Decode(&resJson)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	lyrics, err := parseLyrics(resJson.SyncedLyrics)
	if err != nil {
		return nil, fmt.Errorf("failed to parse lyrics: %w", err)
	}

	if len(lyrics) == 0 {
		return nil, fmt.Errorf("failed to find sync lyrics lines")
	}

	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	err = SaveCache(lyrics, cacheFile)
	if err != nil {
		return nil, fmt.Errorf("failed to cache lyrics to psudo csv: %w", err)
	}

	return lyrics, nil
}
