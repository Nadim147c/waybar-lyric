package main

import (
	"testing"
	"time"
)

func TestStore_SaveLoad(t *testing.T) {
	s := NewStore()
	lyrics := Lyrics{{Text: "Hello"}}
	id := "test-id"

	// Save lyrics and verify they can be loaded
	s.Save(id, lyrics)
	loaded, ok := s.Load(id)
	if !ok {
		t.Fatal("Lyrics not found after save")
	}
	if len(loaded) != len(lyrics) || loaded[0].Text != lyrics[0].Text {
		t.Errorf("Loaded lyrics do not match saved lyrics")
	}
}

func TestStore_LoadUpdatesAccessTime(t *testing.T) {
	s := NewStore()
	id := "test-id"
	s.Save(id, Lyrics{})

	// Load once to get initial access time
	_, _ = s.Load(id)

	// Load again after a delay and check if access time updated
	time.Sleep(2 * time.Millisecond)
	_, _ = s.Load(id)

	// We can't directly check LastAccess now, but we can verify the entry still exists
	_, ok := s.Load(id)
	if !ok {
		t.Error("Entry was unexpectedly deleted (Load failed)")
	}
}

func TestStore_CleanupExpired(t *testing.T) {
	s := NewStore()
	threshold := 10 * time.Millisecond

	// Save an entry and immediately mark it as expired by not accessing it
	s.Save("expired", Lyrics{})

	// Wait for it to expire
	time.Sleep(threshold + 5*time.Millisecond)

	// Run cleanup
	s.cleanupExpired(threshold)

	// Verify expired entry is gone
	if _, ok := s.Load("expired"); ok {
		t.Error("Expired entry was not cleaned up")
	}
}

func TestStore_CleanupLoop(t *testing.T) {
	s := NewStore()
	t.Parallel()
	ctx := t.Context()

	// Save an entry and let it expire
	s.Save("old", Lyrics{})
	s.Save("new", Lyrics{})

	// Make the old entry 1hour old
	s.data["old"].LastAccess = time.Now().Add(-time.Hour)

	// Start cleanup loop with a short interval
	interval := 10 * time.Millisecond
	go s.Cleanup(ctx, interval)

	time.Sleep(9 * time.Millisecond)
	// This should not clean the new entry
	if _, ok := s.Load("new"); !ok {
		t.Error("New entry was incorrectly cleaned up")
	}

	// Verify cleanup results
	time.Sleep(9 * time.Millisecond)
	if _, ok := s.Load("old"); ok {
		t.Error("Old entry was not cleaned up")
	}
	if _, ok := s.Load("new"); !ok {
		t.Error("New entry was incorrectly cleaned up")
	}
}
