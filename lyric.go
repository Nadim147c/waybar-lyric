package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

var LyricStore = make(Store)

const LrclibEndpoint = "https://lrclib.net/api/get"

func request(params url.Values, header http.Header) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, LrclibEndpoint, nil)
	if err != nil {
		return nil, err
	}

	req.URL.RawQuery = params.Encode()
	req.Header = header

	slog.Info("Fetching lyrics from Lrclib", "url", req.URL.String())

	client := http.Client{}

	return client.Do(req)
}

func GetLyrics(info *PlayerInfo) (Lyrics, error) {
	uri := filepath.Base(info.ID)
	uri = strings.ReplaceAll(uri, "/", "-")

	if val, exists := LyricStore.Load(uri); exists {
		if len(val) == 0 {
			return val, fmt.Errorf("Lyrics doesn't exists")
		}
		slog.Debug("Lyrics found in memory cache", "lines", len(val))
		return val, nil
	}

	userCacheDir, _ := os.UserCacheDir()
	cacheDir := filepath.Join(userCacheDir, "waybar-lyric")

	cacheFile := filepath.Join(cacheDir, uri+".csv")

	if cachedLyrics, err := LoadCache(cacheFile); err == nil {
		LyricStore.Save(uri, cachedLyrics)
		return cachedLyrics, nil
	} else {
		slog.Warn("Can't find the lyrics in the cache", "error", err)
	}

	queryParams := url.Values{}
	queryParams.Set("track_name", info.Title)
	queryParams.Set("artist_name", info.Artist)
	if info.Album != "" {
		queryParams.Set("album_name", info.Album)
	}
	if info.Length != 0 {
		queryParams.Set("duration", fmt.Sprintf("%.2f", info.Length.Seconds()))
	}

	header := http.Header{}
	header.Set("User-Agent", Version)

	resp, err := request(queryParams, header)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch lyrics: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		LyricStore.Save(uri, []LyricLine{})
		return nil, fmt.Errorf("Lyrics not found")
	}

	if resp.StatusCode != http.StatusOK {
		LyricStore.Save(uri, []LyricLine{})
		return nil, fmt.Errorf("unexpected HTTP status: %d", resp.StatusCode)
	}

	var resJson LrcLibResponse
	err = json.NewDecoder(resp.Body).Decode(&resJson)
	if err != nil {
		LyricStore.Save(uri, []LyricLine{})
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	lyrics, err := ParseLyrics(resJson.SyncedLyrics)
	if err != nil {
		LyricStore.Save(uri, []LyricLine{})
		return nil, fmt.Errorf("failed to parse lyrics: %w", err)
	}

	if len(lyrics) == 0 {
		LyricStore.Save(uri, []LyricLine{})
		return nil, fmt.Errorf("failed to find sync lyrics lines")
	}

	slices.SortFunc(lyrics, func(a, b LyricLine) int {
		return int((a.Timestamp - b.Timestamp) / time.Millisecond)
	})

	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	err = SaveCache(lyrics, cacheFile)
	if err != nil {
		return nil, fmt.Errorf("failed to cache lyrics to psudo csv: %w", err)
	}

	LyricStore.Save(uri, lyrics)

	return lyrics, nil
}
