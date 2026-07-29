//go:build darwin

package watcher

// GetSystemMaxWatches returns 0 on macOS as FSEvents doesn't have the same limitations as inotify.
func GetSystemMaxWatches() (int, error) {
	return 0, nil
}

// IsWatchLimitError returns false on macOS as FSEvents doesn't have watch limits.
func IsWatchLimitError(err error) bool {
	return false
}

// GetWatchLimitRecommendation returns an empty message on macOS.
func GetWatchLimitRecommendation() string {
	return "macOS uses FSEvents which typically doesn't require limit adjustments."
}
