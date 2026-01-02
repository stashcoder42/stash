package watcher

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEventBatcher_Add(t *testing.T) {
	t.Parallel()

	t.Run("single path accumulates in pending", func(t *testing.T) {
		b := NewEventBatcher(time.Hour, nil) // long delay so it won't flush
		defer b.Stop()

		b.Add("/path/to/file.mp4")

		assert.Equal(t, 1, b.PendingCount())
	})

	t.Run("duplicate paths are deduplicated", func(t *testing.T) {
		b := NewEventBatcher(time.Hour, nil)
		defer b.Stop()

		b.Add("/path/to/file.mp4")
		b.Add("/path/to/file.mp4")
		b.Add("/path/to/file.mp4")

		assert.Equal(t, 1, b.PendingCount())
	})

	t.Run("multiple unique paths accumulate", func(t *testing.T) {
		b := NewEventBatcher(time.Hour, nil)
		defer b.Stop()

		b.Add("/path/to/file1.mp4")
		b.Add("/path/to/file2.mp4")
		b.Add("/path/to/file3.mp4")

		assert.Equal(t, 3, b.PendingCount())
	})

	t.Run("adding after stop is ignored", func(t *testing.T) {
		b := NewEventBatcher(time.Hour, nil)

		b.Add("/path/to/file.mp4")
		assert.Equal(t, 1, b.PendingCount())

		b.Stop()

		b.Add("/path/to/another.mp4")
		assert.Equal(t, 0, b.PendingCount())
	})
}

func TestEventBatcher_Debounce(t *testing.T) {
	t.Parallel()

	t.Run("flush called after debounce delay", func(t *testing.T) {
		done := make(chan []string, 1)
		b := NewEventBatcher(50*time.Millisecond, func(paths []string) {
			done <- paths
		})
		defer b.Stop()

		b.Add("/path/a.mp4")
		b.Add("/path/b.mp4")

		select {
		case paths := <-done:
			assert.Len(t, paths, 2)
			assert.Contains(t, paths, "/path/a.mp4")
			assert.Contains(t, paths, "/path/b.mp4")
		case <-time.After(200 * time.Millisecond):
			t.Fatal("timeout waiting for flush")
		}
	})

	t.Run("rapid adds reset timer and batch together", func(t *testing.T) {
		var mu sync.Mutex
		flushCount := 0
		var receivedPaths []string

		b := NewEventBatcher(100*time.Millisecond, func(paths []string) {
			mu.Lock()
			flushCount++
			receivedPaths = paths
			mu.Unlock()
		})
		defer b.Stop()

		// Add paths rapidly (faster than debounce delay)
		for i := 0; i < 5; i++ {
			b.Add("/path/file.mp4")
			time.Sleep(20 * time.Millisecond)
		}

		// Wait for flush
		time.Sleep(150 * time.Millisecond)

		mu.Lock()
		assert.Equal(t, 1, flushCount, "should only flush once")
		assert.Len(t, receivedPaths, 1, "duplicates should be deduplicated")
		mu.Unlock()
	})

	t.Run("pending cleared after flush", func(t *testing.T) {
		done := make(chan struct{})
		b := NewEventBatcher(50*time.Millisecond, func(paths []string) {
			close(done)
		})
		defer b.Stop()

		b.Add("/path/a.mp4")
		assert.Equal(t, 1, b.PendingCount())

		<-done
		assert.Equal(t, 0, b.PendingCount())
	})
}

func TestEventBatcher_PendingCount(t *testing.T) {
	t.Parallel()

	b := NewEventBatcher(time.Hour, nil)
	defer b.Stop()

	assert.Equal(t, 0, b.PendingCount())

	b.Add("/path/a.mp4")
	assert.Equal(t, 1, b.PendingCount())

	b.Add("/path/b.mp4")
	assert.Equal(t, 2, b.PendingCount())

	// Duplicate doesn't increase count
	b.Add("/path/a.mp4")
	assert.Equal(t, 2, b.PendingCount())
}

func TestEventBatcher_Stop(t *testing.T) {
	t.Parallel()

	t.Run("stop cancels pending timer", func(t *testing.T) {
		flushCalled := false
		b := NewEventBatcher(100*time.Millisecond, func(paths []string) {
			flushCalled = true
		})

		b.Add("/path/a.mp4")
		b.Stop()

		// Wait longer than debounce delay
		time.Sleep(150 * time.Millisecond)

		assert.False(t, flushCalled, "flush should not be called after stop")
	})

	t.Run("stop clears pending paths", func(t *testing.T) {
		b := NewEventBatcher(time.Hour, nil)

		b.Add("/path/a.mp4")
		b.Add("/path/b.mp4")
		assert.Equal(t, 2, b.PendingCount())

		b.Stop()
		assert.Equal(t, 0, b.PendingCount())
	})

	t.Run("double stop is safe", func(t *testing.T) {
		b := NewEventBatcher(time.Hour, nil)

		b.Add("/path/a.mp4")
		b.Stop()
		b.Stop() // Should not panic
	})
}

func TestEventBatcher_NilProcessor(t *testing.T) {
	t.Parallel()

	// Should not panic with nil processor
	b := NewEventBatcher(50*time.Millisecond, nil)
	defer b.Stop()

	b.Add("/path/a.mp4")

	// Wait for flush - should not panic
	time.Sleep(100 * time.Millisecond)
}
