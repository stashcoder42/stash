//go:build windows

package watcher

// GetSystemMaxWatches returns 0 on Windows as ReadDirectoryChangesW doesn't have the same limitations as inotify.
func GetSystemMaxWatches() (int, error) {
	return 0, nil
}

// IsWatchLimitError returns false on Windows as it doesn't have watch limits like Linux.
func IsWatchLimitError(err error) bool {
	return false
}

// GetWatchLimitRecommendation returns an empty message on Windows.
func GetWatchLimitRecommendation() string {
	return "Windows doesn't typically have file watcher limitations."
}
