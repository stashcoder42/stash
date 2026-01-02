//go:build linux

package watcher

import (
	"os"
	"strconv"
	"strings"
)

// GetSystemMaxWatches returns the maximum number of inotify watches allowed on Linux.
func GetSystemMaxWatches() (int, error) {
	data, err := os.ReadFile("/proc/sys/fs/inotify/max_user_watches")
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(data)))
}

// IsWatchLimitError returns true if the error is due to inotify watch limit exhaustion.
func IsWatchLimitError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "no space left on device") ||
		strings.Contains(errStr, "too many open files")
}

// GetWatchLimitRecommendation returns instructions for increasing the inotify watch limit.
func GetWatchLimitRecommendation() string {
	return `To increase the inotify watch limit, run:
  echo fs.inotify.max_user_watches=524288 | sudo tee -a /etc/sysctl.conf
  sudo sysctl -p`
}
