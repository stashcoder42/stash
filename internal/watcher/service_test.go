package watcher

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/stashapp/stash/internal/manager/config"
	"github.com/stretchr/testify/assert"
)

// mockFsWatcher implements FsWatcher for testing.
type mockFsWatcher struct {
	mu      sync.Mutex
	added   []string
	removed []string
	events  chan fsnotify.Event
	errors  chan error
	closed  bool
}

func newMockFsWatcher() *mockFsWatcher {
	return &mockFsWatcher{
		events: make(chan fsnotify.Event, 100),
		errors: make(chan error, 10),
	}
}

func (m *mockFsWatcher) Add(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.added = append(m.added, name)
	return nil
}

func (m *mockFsWatcher) Remove(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.removed = append(m.removed, name)
	return nil
}

func (m *mockFsWatcher) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	close(m.events)
	return nil
}

func (m *mockFsWatcher) EventsChan() <-chan fsnotify.Event { return m.events }
func (m *mockFsWatcher) ErrorsChan() <-chan error          { return m.errors }

// newTestService creates a Service with mock watcher for handleEvent testing.
func newTestService(cfg *mockConfig, trigger *mockScanTrigger) (*Service, *mockFsWatcher) {
	s := NewService(cfg, trigger)
	mw := newMockFsWatcher()
	s.watcher = mw
	s.running = true
	debounce := time.Duration(cfg.GetWatcherDebounceMs()) * time.Millisecond
	s.batcher = NewEventBatcher(debounce, s.processBatch)
	s.removeBatcher = NewEventBatcher(debounce, s.processRemovalBatch)
	s.dirTracker = NewDirActivityTracker(debounce, func(path string) {
		s.batcher.Add(path)
	})
	return s, mw
}

// mockConfig implements the Config interface for testing.
type mockConfig struct {
	stashPaths        config.StashConfigs
	scanMode          config.WatcherScanMode
	debounceMs        int
	cleanOnRemove     bool
	videoExtensions   []string
	imageExtensions   []string
	galleryExtensions []string
	audioExtensions   []string
}

func (m *mockConfig) GetStashPaths() config.StashConfigs         { return m.stashPaths }
func (m *mockConfig) GetWatcherScanMode() config.WatcherScanMode { return m.scanMode }
func (m *mockConfig) GetWatcherDebounceMs() int                  { return m.debounceMs }
func (m *mockConfig) GetWatcherCleanOnRemove() bool              { return m.cleanOnRemove }
func (m *mockConfig) IsWatcherEffectivelyEnabled() bool {
	return m.scanMode != config.WatcherScanModeDisabled || m.cleanOnRemove
}
func (m *mockConfig) GetVideoExtensions() []string   { return m.videoExtensions }
func (m *mockConfig) GetImageExtensions() []string   { return m.imageExtensions }
func (m *mockConfig) GetGalleryExtensions() []string { return m.galleryExtensions }
func (m *mockConfig) GetAudioExtensions() []string   { return m.audioExtensions }

func newMockConfig() *mockConfig {
	return &mockConfig{
		stashPaths:        nil,
		scanMode:          config.WatcherScanModeScan,
		debounceMs:        100,
		cleanOnRemove:     true,
		videoExtensions:   []string{"mp4", "mkv", "avi"},
		imageExtensions:   []string{"jpg", "png", "gif"},
		galleryExtensions: []string{"zip", "cbz"},
		audioExtensions:   []string{"mp3", "flac", "wav", "aac", "ogg", "m4a", "wma"},
	}
}

// mockScanTrigger implements the ScanTrigger interface for testing.
type mockScanTrigger struct {
	mu            sync.Mutex
	scanCalls     [][]string
	identifyCalls [][]string
	cleanCalls    [][]string
	scanErr       error
	identifyErr   error
	cleanErr      error
}

func (m *mockScanTrigger) TriggerScan(ctx context.Context, paths []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.scanCalls = append(m.scanCalls, paths)
	return m.scanErr
}

func (m *mockScanTrigger) TriggerIdentify(ctx context.Context, paths []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.identifyCalls = append(m.identifyCalls, paths)
	return m.identifyErr
}

func (m *mockScanTrigger) TriggerClean(ctx context.Context, paths []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cleanCalls = append(m.cleanCalls, paths)
	return m.cleanErr
}

func (m *mockScanTrigger) getScanCalls() [][]string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.scanCalls
}

func (m *mockScanTrigger) getIdentifyCalls() [][]string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.identifyCalls
}

func (m *mockScanTrigger) getCleanCalls() [][]string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.cleanCalls
}

func TestService_NewService(t *testing.T) {
	t.Parallel()

	cfg := newMockConfig()
	trigger := &mockScanTrigger{}

	s := NewService(cfg, trigger)

	assert.NotNil(t, s)
	assert.False(t, s.IsRunning())
}

func TestService_IsRunning(t *testing.T) {
	t.Parallel()

	cfg := newMockConfig()
	trigger := &mockScanTrigger{}
	s := NewService(cfg, trigger)

	assert.False(t, s.IsRunning())
}

func TestService_Status(t *testing.T) {
	t.Parallel()

	cfg := newMockConfig()
	trigger := &mockScanTrigger{}
	s := NewService(cfg, trigger)

	status := s.Status()

	assert.NotNil(t, status)
	assert.False(t, status.Running)
	assert.Equal(t, 0, status.WatchCount)
	assert.Equal(t, int64(0), status.ProcessedEvents)
	assert.Equal(t, int64(0), status.TriggeredScans)
	assert.Equal(t, int64(0), status.TriggeredCleans)
}

func TestService_isMediaFile(t *testing.T) {
	t.Parallel()

	cfg := newMockConfig()
	trigger := &mockScanTrigger{}
	s := NewService(cfg, trigger)

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		// Video extensions
		{"video mp4", "/path/to/video.mp4", true},
		{"video mkv", "/path/to/video.mkv", true},
		{"video avi", "/path/to/video.avi", true},

		// Image extensions
		{"image jpg", "/path/to/image.jpg", true},
		{"image png", "/path/to/image.png", true},
		{"image gif", "/path/to/image.gif", true},

		// Gallery extensions
		{"gallery zip", "/path/to/gallery.zip", true},
		{"gallery cbz", "/path/to/gallery.cbz", true},

		// Audio extensions
		{"audio mp3", "/path/to/audio.mp3", true},
		{"audio flac", "/path/to/audio.flac", true},
		{"audio wav", "/path/to/audio.wav", true},
		{"audio aac", "/path/to/audio.aac", true},
		{"audio ogg", "/path/to/audio.ogg", true},
		{"audio m4a", "/path/to/audio.m4a", true},
		{"audio wma", "/path/to/audio.wma", true},
		{"uppercase FLAC", "/path/to/audio.FLAC", true},

		// Non-media extensions
		{"text file", "/path/to/file.txt", false},
		{"pdf file", "/path/to/file.pdf", false},
		{"go source", "/path/to/main.go", false},

		// Edge cases
		{"no extension", "/path/to/file", false},
		{"hidden file", "/path/to/.hidden", false},
		{"directory like", "/path/to/mp4/", false},

		// Case insensitivity (fsutil.MatchExtension uses strings.EqualFold)
		{"uppercase MP4", "/path/to/video.MP4", true},
		{"mixed case MkV", "/path/to/video.MkV", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := s.isMediaFile(tc.path)
			assert.Equal(t, tc.expected, result, "isMediaFile(%q)", tc.path)
		})
	}
}

func TestService_processBatch(t *testing.T) {
	t.Parallel()

	t.Run("calls TriggerScan when mode is SCAN", func(t *testing.T) {
		cfg := newMockConfig()
		cfg.scanMode = config.WatcherScanModeScan
		trigger := &mockScanTrigger{}
		s := NewService(cfg, trigger)

		paths := []string{"/path/a.mp4", "/path/b.mp4"}
		s.processBatch(paths)

		scanCalls := trigger.getScanCalls()
		assert.Len(t, scanCalls, 1)
		assert.ElementsMatch(t, paths, scanCalls[0])
	})

	t.Run("calls TriggerIdentify when mode is SCAN_AND_IDENTIFY", func(t *testing.T) {
		cfg := newMockConfig()
		cfg.scanMode = config.WatcherScanModeScanAndIdentify
		trigger := &mockScanTrigger{}
		s := NewService(cfg, trigger)

		paths := []string{"/path/a.mp4"}
		s.processBatch(paths)

		identifyCalls := trigger.getIdentifyCalls()
		assert.Len(t, identifyCalls, 1)
		assert.ElementsMatch(t, paths, identifyCalls[0])
	})

	t.Run("does not call TriggerScan when mode is DISABLED", func(t *testing.T) {
		cfg := newMockConfig()
		cfg.scanMode = config.WatcherScanModeDisabled
		trigger := &mockScanTrigger{}
		s := NewService(cfg, trigger)

		s.processBatch([]string{"/path/a.mp4"})

		assert.Len(t, trigger.getScanCalls(), 0)
	})

	t.Run("does not call TriggerIdentify when mode is SCAN only", func(t *testing.T) {
		cfg := newMockConfig()
		cfg.scanMode = config.WatcherScanModeScan
		trigger := &mockScanTrigger{}
		s := NewService(cfg, trigger)

		s.processBatch([]string{"/path/a.mp4"})

		assert.Len(t, trigger.getIdentifyCalls(), 0)
	})

	t.Run("empty paths is no-op", func(t *testing.T) {
		cfg := newMockConfig()
		trigger := &mockScanTrigger{}
		s := NewService(cfg, trigger)

		s.processBatch([]string{})

		assert.Len(t, trigger.getScanCalls(), 0)
	})

	t.Run("increments triggeredScans counter", func(t *testing.T) {
		cfg := newMockConfig()
		cfg.scanMode = config.WatcherScanModeScan
		trigger := &mockScanTrigger{}
		s := NewService(cfg, trigger)

		s.processBatch([]string{"/path/a.mp4"})
		s.processBatch([]string{"/path/b.mp4"})

		status := s.Status()
		assert.Equal(t, int64(2), status.TriggeredScans)
	})
}

func TestService_processRemovalBatch(t *testing.T) {
	t.Parallel()

	t.Run("calls TriggerScan for move detection", func(t *testing.T) {
		cfg := newMockConfig()
		cfg.scanMode = config.WatcherScanModeScan
		trigger := &mockScanTrigger{}
		s := NewService(cfg, trigger)

		paths := []string{"/path/dir/a.mp4", "/path/dir/b.mp4"}
		s.processRemovalBatch(paths)

		scanCalls := trigger.getScanCalls()
		assert.Len(t, scanCalls, 1)
		// Should call with nil paths for full library scan
		assert.Nil(t, scanCalls[0])
	})

	t.Run("calls TriggerClean with parent directories", func(t *testing.T) {
		cfg := newMockConfig()
		cfg.scanMode = config.WatcherScanModeScan
		trigger := &mockScanTrigger{}
		s := NewService(cfg, trigger)

		paths := []string{"/path/dir1/a.mp4", "/path/dir2/b.mp4", "/path/dir1/c.mp4"}
		s.processRemovalBatch(paths)

		cleanCalls := trigger.getCleanCalls()
		assert.Len(t, cleanCalls, 1)
		// Should have unique parent directories
		assert.Len(t, cleanCalls[0], 2)
		assert.Contains(t, cleanCalls[0], "/path/dir1")
		assert.Contains(t, cleanCalls[0], "/path/dir2")
	})

	t.Run("empty paths is no-op", func(t *testing.T) {
		cfg := newMockConfig()
		trigger := &mockScanTrigger{}
		s := NewService(cfg, trigger)

		s.processRemovalBatch([]string{})

		assert.Len(t, trigger.getScanCalls(), 0)
		assert.Len(t, trigger.getCleanCalls(), 0)
	})

	t.Run("increments triggeredCleans counter", func(t *testing.T) {
		cfg := newMockConfig()
		cfg.scanMode = config.WatcherScanModeScan
		trigger := &mockScanTrigger{}
		s := NewService(cfg, trigger)

		s.processRemovalBatch([]string{"/path/a.mp4"})
		s.processRemovalBatch([]string{"/path/b.mp4"})

		status := s.Status()
		assert.Equal(t, int64(2), status.TriggeredCleans)
	})
}

func TestHandleEvent_RenameRemovesFromBatcherAndCountsEvent(t *testing.T) {
	t.Parallel()

	cfg := newMockConfig()
	cfg.scanMode = config.WatcherScanModeScan
	cfg.cleanOnRemove = false
	cfg.debounceMs = 50
	trigger := &mockScanTrigger{}
	s, _ := newTestService(cfg, trigger)
	defer s.batcher.Stop()
	defer s.removeBatcher.Stop()
	defer s.dirTracker.Stop()

	// Pre-queue a path, then rename it — should be removed from batcher
	s.batcher.Add("/library/movies/_UNPACK_Movie")
	assert.Equal(t, 1, s.batcher.PendingCount())

	s.handleEvent(fsnotify.Event{
		Name: "/library/movies/_UNPACK_Movie",
		Op:   fsnotify.Rename,
	})

	// Renamed path should be removed from pending
	assert.Equal(t, 0, s.batcher.PendingCount(), "renamed path should be removed from batcher")
	// Event should still be counted
	assert.Equal(t, int64(1), s.Status().ProcessedEvents, "rename event should be counted")
}

func TestHandleEvent_CreateNonMediaFileIsIgnored(t *testing.T) {
	t.Parallel()

	cfg := newMockConfig()
	cfg.scanMode = config.WatcherScanModeScan
	cfg.debounceMs = 50
	trigger := &mockScanTrigger{}
	s, _ := newTestService(cfg, trigger)
	defer s.batcher.Stop()
	defer s.removeBatcher.Stop()
	defer s.dirTracker.Stop()

	// Create a temp file that is NOT a media file
	tmpDir := t.TempDir()
	nonMediaPath := tmpDir + "/readme.txt"
	if err := os.WriteFile(nonMediaPath, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	s.handleEvent(fsnotify.Event{
		Name: nonMediaPath,
		Op:   fsnotify.Create,
	})

	// Non-media file should not be queued
	assert.Equal(t, 0, s.batcher.PendingCount(), "non-media file should not be queued for scan")
	// processedEvents should still increment (event was handled, just filtered)
	assert.Equal(t, int64(1), s.Status().ProcessedEvents, "event should still be counted as processed")
}

func TestHandleEvent_CreateStatFailureIsHandled(t *testing.T) {
	t.Parallel()

	cfg := newMockConfig()
	cfg.scanMode = config.WatcherScanModeScan
	cfg.debounceMs = 50
	trigger := &mockScanTrigger{}
	s, _ := newTestService(cfg, trigger)
	defer s.batcher.Stop()
	defer s.removeBatcher.Stop()
	defer s.dirTracker.Stop()

	// CREATE for a path that doesn't exist (file already gone)
	s.handleEvent(fsnotify.Event{
		Name: "/nonexistent/path/video.mp4",
		Op:   fsnotify.Create,
	})

	// Should not be queued (file gone), but event should be counted
	assert.Equal(t, 0, s.batcher.PendingCount(), "gone file should not be queued")
	assert.Equal(t, int64(1), s.Status().ProcessedEvents, "event should still be counted")
}

func TestHandleEvent_WriteNonMediaFileIsIgnored(t *testing.T) {
	t.Parallel()

	cfg := newMockConfig()
	cfg.scanMode = config.WatcherScanModeScan
	cfg.debounceMs = 50
	trigger := &mockScanTrigger{}
	s, _ := newTestService(cfg, trigger)
	defer s.batcher.Stop()
	defer s.removeBatcher.Stop()
	defer s.dirTracker.Stop()

	s.handleEvent(fsnotify.Event{
		Name: "/library/notes.txt",
		Op:   fsnotify.Write,
	})

	// Non-media WRITE should not be queued
	assert.Equal(t, 0, s.batcher.PendingCount(), "non-media write should not be queued")
	assert.Equal(t, int64(1), s.Status().ProcessedEvents, "event should still be counted")
}

func TestHandleEvent_WriteMediaFileIsQueued(t *testing.T) {
	t.Parallel()

	cfg := newMockConfig()
	cfg.scanMode = config.WatcherScanModeScan
	cfg.debounceMs = 50
	trigger := &mockScanTrigger{}
	s, _ := newTestService(cfg, trigger)
	defer s.batcher.Stop()
	defer s.removeBatcher.Stop()
	defer s.dirTracker.Stop()

	s.handleEvent(fsnotify.Event{
		Name: "/library/video.mp4",
		Op:   fsnotify.Write,
	})

	assert.Equal(t, 1, s.batcher.PendingCount(), "media write should be queued")
}
