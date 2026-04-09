package watcher

import (
	"path/filepath"
	"sync"
	"time"

	"github.com/stashapp/stash/pkg/logger"
)

// DirActivityTracker tracks newly-created directories and waits for filesystem
// activity inside them to settle before forwarding to the scan batcher.
//
// This prevents premature scanning of directories that are actively being
// populated (e.g., during archive extraction by tools like SABnzbd).
// The inotify watch is added immediately so events are captured, but the
// scan is deferred until the directory has been quiet for the debounce period.
type DirActivityTracker struct {
	debounceDelay time.Duration
	onQuiesce     func(path string)

	mu      sync.Mutex
	dirs    map[string]*dirState
	stopped bool
}

type dirState struct {
	timer    *time.Timer
	activity int64
}

// NewDirActivityTracker creates a new tracker.
// onQuiesce is called when a tracked directory has had no activity for debounceDelay.
func NewDirActivityTracker(debounceDelay time.Duration, onQuiesce func(path string)) *DirActivityTracker {
	return &DirActivityTracker{
		debounceDelay: debounceDelay,
		onQuiesce:     onQuiesce,
		dirs:          make(map[string]*dirState),
	}
}

// TrackDir starts tracking a directory. A quiesce timer is started; if no
// activity is noted before it fires, onQuiesce is called with the path.
func (t *DirActivityTracker) TrackDir(path string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.stopped {
		return
	}

	// If already tracked, just reset the timer
	if ds, exists := t.dirs[path]; exists {
		ds.timer.Stop()
		ds.timer = time.AfterFunc(t.debounceDelay, func() { t.quiesce(path) })
		return
	}

	t.dirs[path] = &dirState{
		timer: time.AfterFunc(t.debounceDelay, func() { t.quiesce(path) }),
	}

	logger.Debugf("[watcher] Tracking directory for quiescence: %s", path)
}

// NoteActivity resets the quiesce timer for the directory containing the given path.
// Returns true if the path was inside a tracked directory.
func (t *DirActivityTracker) NoteActivity(dirPath string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.stopped {
		return false
	}

	ds, exists := t.dirs[dirPath]
	if !exists {
		return false
	}

	ds.activity++
	ds.timer.Stop()
	ds.timer = time.AfterFunc(t.debounceDelay, func() { t.quiesce(dirPath) })
	return true
}

// TrackedDir returns the tracked directory that contains the given path,
// or empty string if the path is not inside any tracked directory.
func (t *DirActivityTracker) TrackedDir(path string) string {
	t.mu.Lock()
	defer t.mu.Unlock()

	dir := filepath.Dir(path)
	for trackedPath := range t.dirs {
		if dir == trackedPath || isSubpath(dir, trackedPath) {
			return trackedPath
		}
	}
	return ""
}

// Remove cancels tracking for a directory. The quiesce callback will not fire.
func (t *DirActivityTracker) Remove(path string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if ds, exists := t.dirs[path]; exists {
		ds.timer.Stop()
		delete(t.dirs, path)
		logger.Debugf("[watcher] Stopped tracking directory: %s (activity: %d events)", path, ds.activity)
	}
}

// Stop cancels all tracking and prevents new directories from being tracked.
func (t *DirActivityTracker) Stop() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.stopped = true
	for path, ds := range t.dirs {
		ds.timer.Stop()
		delete(t.dirs, path)
	}
}

// TrackedCount returns the number of directories being tracked.
func (t *DirActivityTracker) TrackedCount() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.dirs)
}

func (t *DirActivityTracker) quiesce(path string) {
	t.mu.Lock()
	ds, exists := t.dirs[path]
	if !exists || t.stopped {
		t.mu.Unlock()
		return
	}
	activity := ds.activity
	delete(t.dirs, path)
	t.mu.Unlock()

	logger.Infof("[watcher] Directory quiesced, queueing for scan: %s (saw %d events)", path, activity)

	if t.onQuiesce != nil {
		t.onQuiesce(path)
	}
}

// isSubpath checks if child is a subdirectory of parent.
func isSubpath(child, parent string) bool {
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	// rel should not start with ".." if child is under parent
	return len(rel) > 0 && rel[0] != '.'
}
