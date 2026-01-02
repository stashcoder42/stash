package api

import (
	"context"

	"github.com/stashapp/stash/internal/manager"
	"github.com/stashapp/stash/internal/manager/config"
)

func (r *mutationResolver) EnableWatcher(ctx context.Context) (bool, error) {
	mgr := manager.GetInstance()
	if mgr.WatcherService == nil {
		return false, nil
	}
	err := mgr.WatcherService.Start()
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *mutationResolver) DisableWatcher(ctx context.Context) (bool, error) {
	mgr := manager.GetInstance()
	if mgr.WatcherService == nil {
		return false, nil
	}
	mgr.WatcherService.Stop()
	return true, nil
}

func (r *mutationResolver) ConfigureWatcher(ctx context.Context, input ConfigWatcherInput) (*ConfigWatcherResult, error) {
	c := config.GetInstance()

	if input.Enabled != nil {
		c.SetBool(config.WatcherEnabled, *input.Enabled)
	}
	if input.DebounceMs != nil {
		c.SetInt(config.WatcherDebounceMs, *input.DebounceMs)
	}
	if input.ScanOnChange != nil {
		c.SetBool(config.WatcherScanOnChange, *input.ScanOnChange)
	}
	if input.IdentifyOnChange != nil {
		c.SetBool(config.WatcherIdentifyOnChange, *input.IdentifyOnChange)
	}
	if input.CleanOnRemove != nil {
		c.SetBool(config.WatcherCleanOnRemove, *input.CleanOnRemove)
	}

	if err := c.Write(); err != nil {
		return nil, err
	}

	// Refresh the watcher service if enabled setting changed
	manager.GetInstance().RefreshWatcher()

	return makeConfigWatcherResult(), nil
}
