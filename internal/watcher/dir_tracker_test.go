package watcher

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDirActivityTracker_TrackDir(t *testing.T) {
	t.Parallel()

	t.Run("tracked directory is counted", func(t *testing.T) {
		tracker := NewDirActivityTracker(time.Hour, nil)
		defer tracker.Stop()

		tracker.TrackDir("/data/library/_UNPACK_Movie")

		assert.Equal(t, 1, tracker.TrackedCount())
	})

	t.Run("duplicate tracking resets timer", func(t *testing.T) {
		tracker := NewDirActivityTracker(time.Hour, nil)
		defer tracker.Stop()

		tracker.TrackDir("/data/library/_UNPACK_Movie")
		tracker.TrackDir("/data/library/_UNPACK_Movie")

		assert.Equal(t, 1, tracker.TrackedCount())
	})

	t.Run("tracking after stop is ignored", func(t *testing.T) {
		tracker := NewDirActivityTracker(time.Hour, nil)
		tracker.Stop()

		tracker.TrackDir("/data/library/_UNPACK_Movie")

		assert.Equal(t, 0, tracker.TrackedCount())
	})
}

func TestDirActivityTracker_Quiesce(t *testing.T) {
	t.Parallel()

	t.Run("quiesce fires after debounce with no activity", func(t *testing.T) {
		done := make(chan string, 1)
		tracker := NewDirActivityTracker(50*time.Millisecond, func(path string) {
			done <- path
		})
		defer tracker.Stop()

		tracker.TrackDir("/data/library/NewDir")

		select {
		case path := <-done:
			assert.Equal(t, "/data/library/NewDir", path)
			assert.Equal(t, 0, tracker.TrackedCount())
		case <-time.After(200 * time.Millisecond):
			t.Fatal("timeout waiting for quiesce")
		}
	})

	t.Run("activity resets quiesce timer", func(t *testing.T) {
		done := make(chan string, 1)
		tracker := NewDirActivityTracker(100*time.Millisecond, func(path string) {
			done <- path
		})
		defer tracker.Stop()

		tracker.TrackDir("/data/library/Unpacking")

		// Note activity at 50ms intervals — should keep resetting the 100ms timer
		for i := 0; i < 3; i++ {
			time.Sleep(50 * time.Millisecond)
			noted := tracker.NoteActivity("/data/library/Unpacking")
			assert.True(t, noted)
		}

		// Now wait for quiesce (100ms after last activity)
		select {
		case path := <-done:
			assert.Equal(t, "/data/library/Unpacking", path)
		case <-time.After(300 * time.Millisecond):
			t.Fatal("timeout waiting for quiesce after activity stopped")
		}
	})

	t.Run("quiesce does not fire after remove", func(t *testing.T) {
		fired := make(chan struct{})
		tracker := NewDirActivityTracker(50*time.Millisecond, func(path string) {
			close(fired)
		})
		defer tracker.Stop()

		tracker.TrackDir("/data/library/_UNPACK_Movie")
		tracker.Remove("/data/library/_UNPACK_Movie")

		select {
		case <-fired:
			t.Fatal("quiesce should not fire after remove")
		case <-time.After(150 * time.Millisecond):
			// Expected
		}
	})

	t.Run("quiesce does not fire after stop", func(t *testing.T) {
		fired := make(chan struct{})
		tracker := NewDirActivityTracker(50*time.Millisecond, func(path string) {
			close(fired)
		})

		tracker.TrackDir("/data/library/_UNPACK_Movie")
		tracker.Stop()

		select {
		case <-fired:
			t.Fatal("quiesce should not fire after stop")
		case <-time.After(150 * time.Millisecond):
			// Expected
		}
	})
}

func TestDirActivityTracker_NoteActivity(t *testing.T) {
	t.Parallel()

	t.Run("returns false for untracked directory", func(t *testing.T) {
		tracker := NewDirActivityTracker(time.Hour, nil)
		defer tracker.Stop()

		noted := tracker.NoteActivity("/data/library/Unknown")

		assert.False(t, noted)
	})

	t.Run("returns true for tracked directory", func(t *testing.T) {
		tracker := NewDirActivityTracker(time.Hour, nil)
		defer tracker.Stop()

		tracker.TrackDir("/data/library/_UNPACK_Movie")
		noted := tracker.NoteActivity("/data/library/_UNPACK_Movie")

		assert.True(t, noted)
	})

	t.Run("returns false after stop", func(t *testing.T) {
		tracker := NewDirActivityTracker(time.Hour, nil)

		tracker.TrackDir("/data/library/_UNPACK_Movie")
		tracker.Stop()

		noted := tracker.NoteActivity("/data/library/_UNPACK_Movie")

		assert.False(t, noted)
	})
}

func TestDirActivityTracker_TrackedDir(t *testing.T) {
	t.Parallel()

	t.Run("finds tracked directory for file inside it", func(t *testing.T) {
		tracker := NewDirActivityTracker(time.Hour, nil)
		defer tracker.Stop()

		tracker.TrackDir("/data/library/_UNPACK_Movie")

		dir := tracker.TrackedDir("/data/library/_UNPACK_Movie/movie.mp4")
		assert.Equal(t, "/data/library/_UNPACK_Movie", dir)
	})

	t.Run("finds tracked directory for file in subdirectory", func(t *testing.T) {
		tracker := NewDirActivityTracker(time.Hour, nil)
		defer tracker.Stop()

		tracker.TrackDir("/data/library/_UNPACK_Movie")

		dir := tracker.TrackedDir("/data/library/_UNPACK_Movie/Subs/english.srt")
		assert.Equal(t, "/data/library/_UNPACK_Movie", dir)
	})

	t.Run("returns empty for untracked path", func(t *testing.T) {
		tracker := NewDirActivityTracker(time.Hour, nil)
		defer tracker.Stop()

		tracker.TrackDir("/data/library/_UNPACK_Movie")

		dir := tracker.TrackedDir("/data/library/other/file.mp4")
		assert.Equal(t, "", dir)
	})

	t.Run("returns empty for parent of tracked directory", func(t *testing.T) {
		tracker := NewDirActivityTracker(time.Hour, nil)
		defer tracker.Stop()

		tracker.TrackDir("/data/library/_UNPACK_Movie")

		dir := tracker.TrackedDir("/data/library/file.mp4")
		assert.Equal(t, "", dir)
	})
}

func TestDirActivityTracker_Remove(t *testing.T) {
	t.Parallel()

	t.Run("removes tracked directory", func(t *testing.T) {
		tracker := NewDirActivityTracker(time.Hour, nil)
		defer tracker.Stop()

		tracker.TrackDir("/data/library/_UNPACK_Movie")
		assert.Equal(t, 1, tracker.TrackedCount())

		tracker.Remove("/data/library/_UNPACK_Movie")
		assert.Equal(t, 0, tracker.TrackedCount())
	})

	t.Run("removing non-tracked path is safe", func(t *testing.T) {
		tracker := NewDirActivityTracker(time.Hour, nil)
		defer tracker.Stop()

		tracker.Remove("/data/library/nonexistent")
		assert.Equal(t, 0, tracker.TrackedCount())
	})
}

func TestDirActivityTracker_Stop(t *testing.T) {
	t.Parallel()

	t.Run("clears all tracked directories", func(t *testing.T) {
		tracker := NewDirActivityTracker(time.Hour, nil)

		tracker.TrackDir("/data/library/dir1")
		tracker.TrackDir("/data/library/dir2")
		assert.Equal(t, 2, tracker.TrackedCount())

		tracker.Stop()
		assert.Equal(t, 0, tracker.TrackedCount())
	})

	t.Run("double stop is safe", func(t *testing.T) {
		tracker := NewDirActivityTracker(time.Hour, nil)
		tracker.TrackDir("/data/library/dir1")
		tracker.Stop()
		tracker.Stop()
	})
}

func TestDirActivityTracker_SABnzbdSequence(t *testing.T) {
	t.Parallel()

	t.Run("SABnzbd unpack sequence produces single quiesce", func(t *testing.T) {
		var mu sync.Mutex
		quiesced := []string{}

		tracker := NewDirActivityTracker(100*time.Millisecond, func(path string) {
			mu.Lock()
			quiesced = append(quiesced, path)
			mu.Unlock()
		})
		defer tracker.Stop()

		// Phase 1: _UNPACK_ directory created
		tracker.TrackDir("/data/library/_UNPACK_Movie.2024")

		// Phase 2: Files being extracted (activity keeps resetting timer)
		for i := 0; i < 5; i++ {
			time.Sleep(30 * time.Millisecond)
			tracker.NoteActivity("/data/library/_UNPACK_Movie.2024")
		}

		// Phase 3: _UNPACK_ dir renamed to final name
		tracker.Remove("/data/library/_UNPACK_Movie.2024")
		tracker.TrackDir("/data/library/Movie.2024")

		// Phase 4: Wait for quiesce of final directory
		time.Sleep(200 * time.Millisecond)

		mu.Lock()
		defer mu.Unlock()

		// Should only get one quiesce for the final directory name
		assert.Equal(t, 1, len(quiesced))
		assert.Contains(t, quiesced, "/data/library/Movie.2024")
	})
}

func TestIsSubpath(t *testing.T) {
	t.Parallel()

	assert.True(t, isSubpath("/data/library/_UNPACK_Movie/Subs", "/data/library/_UNPACK_Movie"))
	assert.False(t, isSubpath("/data/library/_UNPACK_Movie", "/data/library/_UNPACK_Movie"), "same path is not a subpath")
	assert.False(t, isSubpath("/data/library", "/data/library/_UNPACK_Movie"))
	assert.False(t, isSubpath("/data/other", "/data/library/_UNPACK_Movie"))
}
