package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

var (
	ErrLyricsNotFound  = errors.New("lyrics not found")
	ErrLyricsNotExists = errors.New("lyrics does not exists")
	ErrLyricsNotSynced = errors.New("lyrics is not synced")
)

var LyricStore = NewStore()

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

func CensorLyrics(lyrics Lyrics) {
	if FilterProfanity {
		for i, l := range lyrics {
			lyrics[i].Text = CensorText(l.Text, FilterProfanityType)
		}
	}
}

func GetLyrics(info *PlayerInfo) (Lyrics, error) {
	uri := filepath.Base(info.ID)
	uri = strings.ReplaceAll(uri, "/", "-")

	if val, exists := LyricStore.Load(uri); exists {
		if len(val) == 0 {
			return val, ErrLyricsNotExists
		}
		slog.Debug("Lyrics found in memory cache", "lines", len(val))
		return val, nil
	}

	cacheFile := filepath.Join(CacheDir, uri+".csv")

	cachedLyrics, err := LoadCache(cacheFile)
	if err == nil {
		CensorLyrics(cachedLyrics)
		LyricStore.Save(uri, cachedLyrics)
		return cachedLyrics, nil
	}
	slog.Warn("Can't find the lyrics in the cache", "error", err)

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
		return nil, ErrLyricsNotFound
	}

	if resp.StatusCode != http.StatusOK {
		LyricStore.Save(uri, []LyricLine{})
		return nil, fmt.Errorf("unexpected HTTP status: %d", resp.StatusCode)
	}

	var resJSON LrcLibResponse
	err = json.NewDecoder(resp.Body).Decode(&resJSON)
	if err != nil {
		LyricStore.Save(uri, []LyricLine{})
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	lyrics, err := ParseLyrics(resJSON.SyncedLyrics)
	if err != nil {
		LyricStore.Save(uri, []LyricLine{})
		return nil, fmt.Errorf("failed to parse lyrics: %w", err)
	}

	if len(lyrics) == 0 {
		LyricStore.Save(uri, []LyricLine{})
		return nil, ErrLyricsNotSynced
	}

	slices.SortFunc(lyrics, func(a, b LyricLine) int {
		return int((a.Timestamp - b.Timestamp) / time.Millisecond)
	})

	if err = SaveCache(lyrics, cacheFile); err != nil {
		return nil, fmt.Errorf("failed to cache lyrics to psudo csv: %w", err)
	}

	CensorLyrics(lyrics)
	LyricStore.Save(uri, lyrics)
	return lyrics, nil
}
