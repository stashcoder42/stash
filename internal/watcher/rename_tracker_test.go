package watcher

import (
	"testing"
	"time"
)

func TestRenameTracker_BasicPairing(t *testing.T) {
	rt := NewRenameTracker(100*time.Millisecond, 1*time.Second)

	// Simulate RENAME followed by CREATE
	rt.OnDirectoryRename("/stuff/_UNPACK_Movie")
	rt.OnDirectoryCreate("/stuff/Movie")

	// Verify the mapping was created
	if rt.MappingCount() != 1 {
		t.Errorf("expected 1 mapping, got %d", rt.MappingCount())
	}
	if rt.PendingCount() != 0 {
		t.Errorf("expected 0 pending, got %d", rt.PendingCount())
	}
}

func TestRenameTracker_PathTranslation(t *testing.T) {
	rt := NewRenameTracker(100*time.Millisecond, 1*time.Second)

	// Create a mapping
	rt.OnDirectoryRename("/stuff/_UNPACK_Movie")
	rt.OnDirectoryCreate("/stuff/Movie")

	tests := []struct {
		input    string
		expected string
	}{
		// File inside renamed directory
		{"/stuff/_UNPACK_Movie/video.mp4", "/stuff/Movie/video.mp4"},
		// File in subdirectory
		{"/stuff/_UNPACK_Movie/subs/english.srt", "/stuff/Movie/subs/english.srt"},
		// Unrelated path - unchanged
		{"/stuff/other/file.mp4", "/stuff/other/file.mp4"},
		// Exact directory match
		{"/stuff/_UNPACK_Movie", "/stuff/Movie"},
	}

	for _, tc := range tests {
		result := rt.TranslatePaths([]string{tc.input})
		if result[0] != tc.expected {
			t.Errorf("TranslatePaths(%q) = %q, want %q", tc.input, result[0], tc.expected)
		}
	}
}

func TestRenameTracker_MultiplePaths(t *testing.T) {
	rt := NewRenameTracker(100*time.Millisecond, 1*time.Second)

	rt.OnDirectoryRename("/stuff/_UNPACK_Movie")
	rt.OnDirectoryCreate("/stuff/Movie")

	paths := []string{
		"/stuff/_UNPACK_Movie/video.mp4",
		"/stuff/_UNPACK_Movie/audio.mp3",
		"/stuff/other/unchanged.mp4",
	}

	expected := []string{
		"/stuff/Movie/video.mp4",
		"/stuff/Movie/audio.mp3",
		"/stuff/other/unchanged.mp4",
	}

	result := rt.TranslatePaths(paths)

	for i, exp := range expected {
		if result[i] != exp {
			t.Errorf("path %d: got %q, want %q", i, result[i], exp)
		}
	}
}

func TestRenameTracker_NoMatchDifferentParent(t *testing.T) {
	rt := NewRenameTracker(100*time.Millisecond, 1*time.Second)

	// RENAME in one directory, CREATE in different directory
	rt.OnDirectoryRename("/stuff/dir1/_UNPACK_Movie")
	rt.OnDirectoryCreate("/stuff/dir2/Movie")

	// Should not pair - different parent directories
	if rt.MappingCount() != 0 {
		t.Errorf("expected 0 mappings (different parents), got %d", rt.MappingCount())
	}
	if rt.PendingCount() != 1 {
		t.Errorf("expected 1 pending (unmatched rename), got %d", rt.PendingCount())
	}
}

func TestRenameTracker_CleanupExpiredPending(t *testing.T) {
	// Use very short window for testing
	rt := NewRenameTracker(10*time.Millisecond, 1*time.Second)

	rt.OnDirectoryRename("/stuff/_UNPACK_Movie")

	// Wait for pending to expire
	time.Sleep(20 * time.Millisecond)

	rt.Cleanup()

	if rt.PendingCount() != 0 {
		t.Errorf("expected 0 pending after cleanup, got %d", rt.PendingCount())
	}
}

func TestRenameTracker_CleanupExpiredMappings(t *testing.T) {
	// Use very short TTL for testing
	rt := NewRenameTracker(100*time.Millisecond, 10*time.Millisecond)

	rt.OnDirectoryRename("/stuff/_UNPACK_Movie")
	rt.OnDirectoryCreate("/stuff/Movie")

	if rt.MappingCount() != 1 {
		t.Errorf("expected 1 mapping initially, got %d", rt.MappingCount())
	}

	// Wait for mapping to expire
	time.Sleep(20 * time.Millisecond)

	rt.Cleanup()

	if rt.MappingCount() != 0 {
		t.Errorf("expected 0 mappings after cleanup, got %d", rt.MappingCount())
	}
}

func TestRenameTracker_PairingWithinWindow(t *testing.T) {
	rt := NewRenameTracker(50*time.Millisecond, 1*time.Second)

	rt.OnDirectoryRename("/stuff/_UNPACK_Movie")

	// Wait but stay within window
	time.Sleep(30 * time.Millisecond)

	rt.OnDirectoryCreate("/stuff/Movie")

	// Should still pair
	if rt.MappingCount() != 1 {
		t.Errorf("expected 1 mapping (within window), got %d", rt.MappingCount())
	}
}

func TestRenameTracker_PairingOutsideWindow(t *testing.T) {
	rt := NewRenameTracker(10*time.Millisecond, 1*time.Second)

	rt.OnDirectoryRename("/stuff/_UNPACK_Movie")

	// Wait beyond the window
	time.Sleep(20 * time.Millisecond)

	rt.OnDirectoryCreate("/stuff/Movie")

	// Should NOT pair - outside window
	if rt.MappingCount() != 0 {
		t.Errorf("expected 0 mappings (outside window), got %d", rt.MappingCount())
	}
}

func TestRenameTracker_MultipleMappings(t *testing.T) {
	rt := NewRenameTracker(100*time.Millisecond, 1*time.Second)

	// Create multiple mappings
	rt.OnDirectoryRename("/stuff/_UNPACK_Movie1")
	rt.OnDirectoryCreate("/stuff/Movie1")

	rt.OnDirectoryRename("/stuff/_UNPACK_Movie2")
	rt.OnDirectoryCreate("/stuff/Movie2")

	if rt.MappingCount() != 2 {
		t.Errorf("expected 2 mappings, got %d", rt.MappingCount())
	}

	// Test translation of both
	paths := []string{
		"/stuff/_UNPACK_Movie1/file.mp4",
		"/stuff/_UNPACK_Movie2/file.mp4",
	}

	result := rt.TranslatePaths(paths)

	if result[0] != "/stuff/Movie1/file.mp4" {
		t.Errorf("path 0: got %q, want %q", result[0], "/stuff/Movie1/file.mp4")
	}
	if result[1] != "/stuff/Movie2/file.mp4" {
		t.Errorf("path 1: got %q, want %q", result[1], "/stuff/Movie2/file.mp4")
	}
}

func TestRenameTracker_EmptyPaths(t *testing.T) {
	rt := NewRenameTracker(100*time.Millisecond, 1*time.Second)

	result := rt.TranslatePaths([]string{})
	if len(result) != 0 {
		t.Errorf("expected empty result for empty input")
	}

	result = rt.TranslatePaths(nil)
	if result != nil {
		t.Errorf("expected nil result for nil input")
	}
}

func TestRenameTracker_NoMappings(t *testing.T) {
	rt := NewRenameTracker(100*time.Millisecond, 1*time.Second)

	paths := []string{"/stuff/file.mp4"}
	result := rt.TranslatePaths(paths)

	// Should return same slice when no mappings
	if result[0] != paths[0] {
		t.Errorf("expected unchanged path when no mappings")
	}
}

func TestParentDir(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"/stuff/dir/file", "/stuff/dir"},
		{"/stuff/dir/", "/stuff/dir"}, // Trailing slash - strips to /stuff/dir (edge case, fsnotify never sends trailing slashes)
		{"/stuff/file", "/stuff"},
		{"/file", ""},
		{"file", ""},
	}

	for _, tc := range tests {
		result := parentDir(tc.path)
		if result != tc.expected {
			t.Errorf("parentDir(%q) = %q, want %q", tc.path, result, tc.expected)
		}
	}
}
