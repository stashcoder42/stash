package api

import (
	"context"

	"github.com/stashapp/stash/internal/manager"
	"github.com/stashapp/stash/internal/manager/config"
)

func (r *mutationResolver) ConfigureWatcher(ctx context.Context, input ConfigWatcherInput) (*ConfigWatcherResult, error) {
	c := config.GetInstance()

	if input.ScanMode != nil {
		c.SetString(config.WatcherScanModeKey, input.ScanMode.String())
	}
	if input.DebounceMs != nil {
		c.SetInt(config.WatcherDebounceMs, *input.DebounceMs)
	}
	if input.CleanOnRemove != nil {
		c.SetBool(config.WatcherCleanOnRemove, *input.CleanOnRemove)
	}

	if err := c.Write(); err != nil {
		return nil, err
	}

	// Refresh the watcher service based on new config
	manager.GetInstance().RefreshWatcher()

	return makeConfigWatcherResult(), nil
}
