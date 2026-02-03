package watcher

import (
	"strings"
	"sync"
	"time"

	"github.com/stashapp/stash/pkg/logger"
)

// RenameTracker buffers directory rename events to pair RENAME (old path) with
// CREATE (new path) events, enabling path translation when the debounce fires.
//
// When a directory is renamed, fsnotify emits:
//  1. RENAME event with the old path
//  2. CREATE event with the new path (shortly after)
//
// This tracker pairs these events by timing and parent directory, then provides
// path translation so queued file paths using the old directory name can be
// converted to use the new directory name.
type RenameTracker struct {
	mutex sync.Mutex

	// pendingRenames holds RENAME events waiting for a matching CREATE
	pendingRenames []pendingRename

	// completedMaps holds successfully paired old->new directory mappings
	completedMaps []RenameMapping

	// pairWindow is the maximum time between RENAME and CREATE to consider them paired
	pairWindow time.Duration

	// mappingTTL is how long to keep completed mappings before cleanup
	mappingTTL time.Duration
}

// pendingRename represents a RENAME event waiting for its matching CREATE
type pendingRename struct {
	Path      string
	Parent    string
	Timestamp time.Time
}

// RenameMapping represents a paired RENAME->CREATE directory rename
type RenameMapping struct {
	OldPath   string
	NewPath   string
	Timestamp time.Time
}

// NewRenameTracker creates a new tracker with specified timing parameters.
//
// pairWindow: maximum time to wait for CREATE after RENAME (e.g., 100ms)
// mappingTTL: how long to keep mappings for translation (e.g., 2x debounce delay)
func NewRenameTracker(pairWindow, mappingTTL time.Duration) *RenameTracker {
	return &RenameTracker{
		pairWindow: pairWindow,
		mappingTTL: mappingTTL,
	}
}

// OnDirectoryRename records a RENAME event for potential pairing with a CREATE.
func (rt *RenameTracker) OnDirectoryRename(oldPath string) {
	rt.mutex.Lock()
	defer rt.mutex.Unlock()

	// Extract parent directory for matching
	parent := parentDir(oldPath)

	rt.pendingRenames = append(rt.pendingRenames, pendingRename{
		Path:      oldPath,
		Parent:    parent,
		Timestamp: time.Now(),
	})

	logger.Debugf("[watcher] RenameTracker: recorded pending rename: %s", oldPath)
}

// OnDirectoryCreate attempts to pair a CREATE event with a pending RENAME.
// If a match is found (same parent directory, within pairWindow), creates a mapping.
func (rt *RenameTracker) OnDirectoryCreate(newPath string) {
	rt.mutex.Lock()
	defer rt.mutex.Unlock()

	now := time.Now()
	parent := parentDir(newPath)

	// Look for a matching pending rename in same parent directory within time window
	for i, pr := range rt.pendingRenames {
		if pr.Parent == parent && now.Sub(pr.Timestamp) <= rt.pairWindow {
			// Found a match - create the mapping
			mapping := RenameMapping{
				OldPath:   pr.Path,
				NewPath:   newPath,
				Timestamp: now,
			}
			rt.completedMaps = append(rt.completedMaps, mapping)

			// Remove the matched pending rename
			rt.pendingRenames = append(rt.pendingRenames[:i], rt.pendingRenames[i+1:]...)

			logger.Infof("[watcher] RenameTracker: paired rename %s -> %s", pr.Path, newPath)
			return
		}
	}

	logger.Debugf("[watcher] RenameTracker: no matching rename for CREATE: %s", newPath)
}

// TranslatePaths converts paths that use old renamed directory prefixes to use new paths.
// Returns the translated paths (unchanged paths are passed through as-is).
func (rt *RenameTracker) TranslatePaths(paths []string) []string {
	rt.mutex.Lock()
	defer rt.mutex.Unlock()

	if len(rt.completedMaps) == 0 {
		return paths
	}

	result := make([]string, len(paths))
	for i, path := range paths {
		result[i] = rt.translatePath(path)
	}
	return result
}

// translatePath translates a single path using completed mappings.
// Must be called with mutex held.
func (rt *RenameTracker) translatePath(path string) string {
	for _, m := range rt.completedMaps {
		// Check if path has old directory as prefix
		if strings.HasPrefix(path, m.OldPath+"/") {
			newPath := m.NewPath + path[len(m.OldPath):]
			logger.Infof("[watcher] RenameTracker: translated path %s -> %s", path, newPath)
			return newPath
		}
		// Also handle exact match (the directory itself)
		if path == m.OldPath {
			logger.Infof("[watcher] RenameTracker: translated path %s -> %s", path, m.NewPath)
			return m.NewPath
		}
	}
	return path
}

// Cleanup removes expired pending renames and old mappings.
// Should be called periodically to prevent memory growth.
func (rt *RenameTracker) Cleanup() {
	rt.mutex.Lock()
	defer rt.mutex.Unlock()

	now := time.Now()

	// Remove expired pending renames (older than pairWindow)
	var activePending []pendingRename
	for _, pr := range rt.pendingRenames {
		if now.Sub(pr.Timestamp) <= rt.pairWindow {
			activePending = append(activePending, pr)
		} else {
			logger.Debugf("[watcher] RenameTracker: expired pending rename: %s", pr.Path)
		}
	}
	rt.pendingRenames = activePending

	// Remove expired mappings (older than mappingTTL)
	var activeMappings []RenameMapping
	for _, m := range rt.completedMaps {
		if now.Sub(m.Timestamp) <= rt.mappingTTL {
			activeMappings = append(activeMappings, m)
		} else {
			logger.Debugf("[watcher] RenameTracker: expired mapping: %s -> %s", m.OldPath, m.NewPath)
		}
	}
	rt.completedMaps = activeMappings
}

// PendingCount returns the number of pending RENAME events awaiting a CREATE match.
func (rt *RenameTracker) PendingCount() int {
	rt.mutex.Lock()
	defer rt.mutex.Unlock()
	return len(rt.pendingRenames)
}

// MappingCount returns the number of completed rename mappings.
func (rt *RenameTracker) MappingCount() int {
	rt.mutex.Lock()
	defer rt.mutex.Unlock()
	return len(rt.completedMaps)
}

// parentDir extracts the parent directory from a path.
// Uses simple string manipulation to avoid syscalls.
func parentDir(path string) string {
	// Find last separator (don't strip trailing slash first - we want parent of the dir)
	lastSep := strings.LastIndex(path, "/")
	if lastSep == -1 {
		return ""
	}

	// Handle trailing slash: /stuff/dir/ -> parent is /stuff/dir's parent = /stuff
	// But /stuff/dir -> parent is /stuff
	// Actually for our use case, paths won't have trailing slashes from fsnotify
	// Just find the last component
	parent := path[:lastSep]
	if parent == "" && lastSep == 0 {
		// Root path like /file
		return ""
	}
	return parent
}
