// Package watcher provides a file watching service that monitors library paths
// for new files and triggers scan/identify operations.
package watcher

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/stashapp/stash/internal/manager/config"
	"github.com/stashapp/stash/pkg/fsutil"
	"github.com/stashapp/stash/pkg/logger"
)

// Config interface for watcher configuration.
type Config interface {
	GetStashPaths() config.StashConfigs
	GetWatcherScanMode() config.WatcherScanMode
	GetWatcherDebounceMs() int
	GetWatcherCleanOnRemove() bool
	IsWatcherEffectivelyEnabled() bool
	GetVideoExtensions() []string
	GetImageExtensions() []string
	GetGalleryExtensions() []string
}

// ScanTrigger interface for triggering scans and identification.
type ScanTrigger interface {
	TriggerScan(ctx context.Context, paths []string) error
	TriggerIdentify(ctx context.Context, paths []string) error
	TriggerClean(ctx context.Context, paths []string) error
}

// Status represents the current state of the file watcher.
type Status struct {
	Running         bool      `json:"running"`
	WatchedPaths    []string  `json:"watchedPaths"`
	WatchCount      int       `json:"watchCount"`
	PendingEvents   int       `json:"pendingEvents"`
	ProcessedEvents int64     `json:"processedEvents"`
	TriggeredScans  int64     `json:"triggeredScans"`
	TriggeredCleans int64     `json:"triggeredCleans"`
	LastError       string    `json:"lastError,omitempty"`
	LastErrorTime   time.Time `json:"lastErrorTime,omitempty"`
}

// Service manages file watching for library paths.
type Service struct {
	config      Config
	scanTrigger ScanTrigger

	watcher       *fsnotify.Watcher
	batcher       *EventBatcher
	removeBatcher *EventBatcher
	renameTracker *RenameTracker

	running bool
	mutex   sync.Mutex
	ctx     context.Context
	cancel  context.CancelFunc

	// Statistics
	watchCount      int
	processedEvents int64
	triggeredScans  int64
	triggeredCleans int64
	lastError       error
	lastErrorTime   time.Time

	// Track root paths for status reporting
	rootPaths []string
}

// NewService creates a new file watcher service.
func NewService(cfg Config, trigger ScanTrigger) *Service {
	return &Service{
		config:      cfg,
		scanTrigger: trigger,
		// Initialize rename tracker with default timing
		// pairWindow: 100ms to pair RENAME+CREATE events
		// mappingTTL: 10s default, will be updated in Start() based on debounce config
		renameTracker: NewRenameTracker(100*time.Millisecond, 10*time.Second),
	}
}

// Start starts the file watcher service.
func (s *Service) Start() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.running {
		return nil
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		s.setError(err)
		return err
	}
	s.watcher = watcher

	s.ctx, s.cancel = context.WithCancel(context.Background())

	// Initialize batchers
	debounceDelay := time.Duration(s.config.GetWatcherDebounceMs()) * time.Millisecond
	s.batcher = NewEventBatcher(debounceDelay, s.processBatch)
	s.removeBatcher = NewEventBatcher(debounceDelay, s.processRemovalBatch)

	// Initialize rename tracker
	// pairWindow: 100ms to pair RENAME+CREATE events (they arrive consecutively)
	// mappingTTL: 2x debounce delay to ensure mappings are available when batch fires
	s.renameTracker = NewRenameTracker(100*time.Millisecond, debounceDelay*2)

	// Reset statistics
	s.watchCount = 0
	s.processedEvents = 0
	s.triggeredScans = 0
	s.triggeredCleans = 0
	s.rootPaths = nil

	// Add watches for all stash paths
	if err := s.addWatchesForPaths(); err != nil {
		s.watcher.Close()
		s.setError(err)
		return err
	}

	s.running = true
	go s.processEvents()

	logger.Infof("[watcher] Service started, watching %d directories", s.watchCount)
	return nil
}

// Stop stops the file watcher service.
func (s *Service) Stop() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.running {
		return
	}

	logger.Info("[watcher] Stopping service")

	if s.cancel != nil {
		s.cancel()
	}
	if s.batcher != nil {
		s.batcher.Stop()
	}
	if s.removeBatcher != nil {
		s.removeBatcher.Stop()
	}
	if s.watcher != nil {
		s.watcher.Close()
	}

	s.running = false
	s.watchCount = 0
	s.rootPaths = nil

	logger.Info("[watcher] Service stopped")
}

// IsRunning returns whether the service is running.
func (s *Service) IsRunning() bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.running
}

// Status returns the current status of the file watcher.
func (s *Service) Status() *Status {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	status := &Status{
		Running:         s.running,
		WatchedPaths:    s.rootPaths,
		WatchCount:      s.watchCount,
		ProcessedEvents: atomic.LoadInt64(&s.processedEvents),
		TriggeredScans:  atomic.LoadInt64(&s.triggeredScans),
		TriggeredCleans: atomic.LoadInt64(&s.triggeredCleans),
	}

	if s.batcher != nil {
		status.PendingEvents += s.batcher.PendingCount()
	}
	if s.removeBatcher != nil {
		status.PendingEvents += s.removeBatcher.PendingCount()
	}

	if s.lastError != nil {
		status.LastError = s.lastError.Error()
		status.LastErrorTime = s.lastErrorTime
	}

	return status
}

// RefreshPaths updates the watched paths based on current configuration.
func (s *Service) RefreshPaths() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.running {
		return nil
	}

	logger.Info("[watcher] Refreshing paths")

	// Get current config paths
	configPaths := make(map[string]bool)
	for _, sp := range s.config.GetStashPaths() {
		configPaths[sp.Path] = true
	}

	// Remove watches for paths no longer in config
	for _, path := range s.rootPaths {
		if !configPaths[path] {
			s.removeWatchRecursive(path)
		}
	}

	// Re-add watches for all paths (will skip already-watched)
	return s.addWatchesForPaths()
}

// addWatchesForPaths adds watches for all configured stash paths.
func (s *Service) addWatchesForPaths() error {
	stashPaths := s.config.GetStashPaths()
	s.rootPaths = make([]string, 0, len(stashPaths))

	for _, sp := range stashPaths {
		s.rootPaths = append(s.rootPaths, sp.Path)
		if err := s.addWatchRecursive(sp.Path); err != nil {
			logger.Warnf("[watcher] Failed to watch path %s: %v", sp.Path, err)
			// Continue with other paths even if one fails
		}
	}

	return nil
}

// addWatchRecursive adds watches for a directory and all its subdirectories.
func (s *Service) addWatchRecursive(path string) error {
	return filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			return nil
		}

		if err := s.watcher.Add(p); err != nil {
			if IsWatchLimitError(err) {
				logger.Warnf("[watcher] Watch limit reached, cannot watch: %s. %s", p, GetWatchLimitRecommendation())
				return filepath.SkipDir
			}
			return err
		}

		s.watchCount++
		return nil
	})
}

// removeWatchRecursive removes watches for a directory and all its subdirectories.
func (s *Service) removeWatchRecursive(path string) {
	_ = filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return err
		}

		if err := s.watcher.Remove(p); err == nil {
			s.watchCount--
		}
		return nil
	})
}

// processEvents handles fsnotify events in a goroutine.
func (s *Service) processEvents() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case event, ok := <-s.watcher.Events:
			if !ok {
				return
			}
			s.handleEvent(event)
		case err, ok := <-s.watcher.Errors:
			if !ok {
				return
			}
			s.setError(err)
			logger.Errorf("[watcher] Error: %v", err)
		}
	}
}

// handleEvent processes a single fsnotify event.
func (s *Service) handleEvent(event fsnotify.Event) {
	atomic.AddInt64(&s.processedEvents, 1)

	// Log events at INFO level (skip WRITE - too noisy during file transfers)
	if event.Op&fsnotify.Write == 0 {
		logger.Infof("[watcher] Event: %s %s", event.Op, event.Name)
	}

	// Handle new directories - add them to the watch list
	if event.Op&fsnotify.Create != 0 {
		info, err := os.Stat(event.Name)
		if err == nil && info.IsDir() {
			logger.Infof("[watcher] New directory created, adding to watch list: %s", event.Name)
			s.mutex.Lock()
			if err := s.addWatchRecursive(event.Name); err != nil {
				logger.Warnf("[watcher] Failed to watch new directory %s: %v", event.Name, err)
			}
			s.mutex.Unlock()

			// Track directory create for rename pairing
			s.renameTracker.OnDirectoryCreate(event.Name)
		}
	}

	// Handle renamed items - the event.Name is the OLD path before rename
	// fsnotify does not provide the new path; we rely on a Create event for the new location
	if event.Op&fsnotify.Rename != 0 {
		logger.Infof("[watcher] Rename event (old path): %s", event.Name)
		// Track directory rename for pairing with subsequent CREATE
		// Note: We can't easily check if it was a directory since it no longer exists,
		// but tracking all renames is safe - non-matching creates just won't pair
		s.renameTracker.OnDirectoryRename(event.Name)
	}

	// Handle removed directories - remove from watch list
	if event.Op&fsnotify.Remove != 0 {
		s.mutex.Lock()
		// Try to remove from watcher - may fail if already removed, that's OK
		_ = s.watcher.Remove(event.Name)
		s.mutex.Unlock()
	}

	// Check if it's a media file we care about
	if !s.isMediaFile(event.Name) {
		return
	}

	// Route to appropriate batcher based on event type
	if event.Op&(fsnotify.Create|fsnotify.Write) != 0 {
		s.batcher.Add(event.Name)
		logger.Infof("[watcher] Queued media file for scan: %s", event.Name)
	} else if event.Op&fsnotify.Remove != 0 && s.config.GetWatcherCleanOnRemove() {
		s.removeBatcher.Add(event.Name)
	}
}

// isMediaFile checks if a file is a media file based on extension.
func (s *Service) isMediaFile(path string) bool {
	return fsutil.MatchExtension(path, s.config.GetVideoExtensions()) ||
		fsutil.MatchExtension(path, s.config.GetImageExtensions()) ||
		fsutil.MatchExtension(path, s.config.GetGalleryExtensions())
}

// processBatch handles a batch of file paths after debouncing.
func (s *Service) processBatch(paths []string) {
	if len(paths) == 0 {
		return
	}

	scanMode := s.config.GetWatcherScanMode()
	if !scanMode.ShouldScan() {
		return
	}

	// Translate paths using rename tracker (handles _UNPACK_ -> final name renames)
	translatedPaths := s.renameTracker.TranslatePaths(paths)

	// Cleanup old mappings
	s.renameTracker.Cleanup()

	atomic.AddInt64(&s.triggeredScans, 1)

	ctx := context.Background()

	// Trigger scan
	logger.Infof("[watcher] Triggering scan for paths: %v", translatedPaths)
	if err := s.scanTrigger.TriggerScan(ctx, translatedPaths); err != nil {
		s.setError(err)
		logger.Errorf("[watcher] Scan failed: %v", err)
		return
	}

	// Trigger identify if mode includes identification
	if scanMode.ShouldIdentify() {
		logger.Infof("[watcher] Triggering identify for paths: %v", translatedPaths)
		if err := s.scanTrigger.TriggerIdentify(ctx, translatedPaths); err != nil {
			s.setError(err)
			logger.Errorf("[watcher] Identify failed: %v", err)
		}
	}
}

// processRemovalBatch handles a batch of removed file paths after debouncing.
// It first triggers a scan to detect moved files (via fingerprint matching),
// then triggers a clean for files that were truly deleted.
func (s *Service) processRemovalBatch(paths []string) {
	if len(paths) == 0 {
		return
	}

	atomic.AddInt64(&s.triggeredCleans, 1)

	ctx := context.Background()

	// First, trigger a scan of all library paths to detect moves.
	// This will find files with matching fingerprints at new locations
	// and update their paths in the database.
	if s.config.GetWatcherScanMode().ShouldScan() {
		logger.Infof("[watcher] Triggering scan for move detection (removed: %v)", paths)
		if err := s.scanTrigger.TriggerScan(ctx, nil); err != nil {
			s.setError(err)
			logger.Errorf("[watcher] Scan for move detection failed: %v", err)
			return
		}
	}

	// Extract unique parent directories from removed file paths.
	// Clean works on directories, not individual files.
	dirSet := make(map[string]bool)
	for _, p := range paths {
		dirSet[filepath.Dir(p)] = true
	}
	dirs := make([]string, 0, len(dirSet))
	for dir := range dirSet {
		dirs = append(dirs, dir)
	}

	// Trigger clean for the parent directories of removed files.
	// The clean operation finds database entries where the file no longer exists.
	// Files that were moved will have updated paths from the scan above and won't be cleaned.
	// Files that were truly deleted will be cleaned.
	logger.Infof("[watcher] Triggering clean for directories: %v", dirs)
	if err := s.scanTrigger.TriggerClean(ctx, dirs); err != nil {
		s.setError(err)
		logger.Errorf("[watcher] Clean failed: %v", err)
	}
}

// setError records an error with timestamp.
func (s *Service) setError(err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.lastError = err
	s.lastErrorTime = time.Now()
}
