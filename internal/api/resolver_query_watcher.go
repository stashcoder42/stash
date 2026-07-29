package api

import (
	"context"

	"github.com/stashapp/stash/internal/manager"
)

func (r *queryResolver) WatcherStatus(ctx context.Context) (*WatcherStatus, error) {
	mgr := manager.GetInstance()
	if mgr.WatcherService == nil {
		// Return empty status if service not initialized
		return &WatcherStatus{
			Running:      false,
			WatchedPaths: []string{},
		}, nil
	}

	status := mgr.WatcherService.Status()

	var lastError *string
	if status.LastError != "" {
		lastError = &status.LastError
	}

	return &WatcherStatus{
		Running:         status.Running,
		WatchedPaths:    status.WatchedPaths,
		WatchCount:      status.WatchCount,
		PendingEvents:   status.PendingEvents,
		ProcessedEvents: int(status.ProcessedEvents),
		TriggeredScans:  int(status.TriggeredScans),
		TriggeredCleans: int(status.TriggeredCleans),
		LastError:       lastError,
	}, nil
}
