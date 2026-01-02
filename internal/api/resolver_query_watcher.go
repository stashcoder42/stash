package api

import (
	"context"

	"github.com/stashapp/stash/internal/manager"
	"github.com/stashapp/stash/internal/watcher"
)

func (r *queryResolver) WatcherStatus(ctx context.Context) (*watcher.Status, error) {
	mgr := manager.GetInstance()
	if mgr.WatcherService == nil {
		// Return empty status if service not initialized
		return &watcher.Status{
			Running:      false,
			WatchedPaths: []string{},
		}, nil
	}
	return mgr.WatcherService.Status(), nil
}
