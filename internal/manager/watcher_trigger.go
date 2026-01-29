package manager

import (
	"context"

	"github.com/stashapp/stash/internal/identify"
	"github.com/stashapp/stash/internal/manager/config"
	"github.com/stashapp/stash/pkg/logger"
)

// WatcherScanTrigger implements the watcher.ScanTrigger interface
// to trigger scans and identification from the file watcher.
type WatcherScanTrigger struct {
	manager *Manager
}

// NewWatcherScanTrigger creates a new WatcherScanTrigger.
func NewWatcherScanTrigger(mgr *Manager) *WatcherScanTrigger {
	return &WatcherScanTrigger{manager: mgr}
}

// TriggerScan triggers a scan for the given paths.
func (t *WatcherScanTrigger) TriggerScan(ctx context.Context, paths []string) error {
	// Get default scan settings from config
	var scanOpts config.ScanMetadataOptions
	if defaults := t.manager.Config.GetDefaultScanSettings(); defaults != nil {
		scanOpts = *defaults
	}

	input := ScanMetadataInput{
		Paths:               paths,
		ScanMetadataOptions: scanOpts,
	}

	_, err := t.manager.Scan(ctx, input)
	return err
}

// TriggerIdentify triggers identification for scenes in the given paths.
func (t *WatcherScanTrigger) TriggerIdentify(ctx context.Context, paths []string) error {
	// Get default identify settings from config
	var identifyOpts identify.Options
	if defaults := t.manager.Config.GetDefaultIdentifySettings(); defaults != nil {
		identifyOpts = *defaults
	}

	// Set paths to identify
	identifyOpts.Paths = paths

	// Only trigger if sources are configured
	if len(identifyOpts.Sources) == 0 {
		// No sources configured, skip identification
		return nil
	}

	job := CreateIdentifyJob(identifyOpts)
	t.manager.JobManager.Add(ctx, "Auto-identifying...", job)
	return nil
}

// TriggerClean triggers a clean operation for the given paths.
// This removes database entries for files that no longer exist at those paths.
func (t *WatcherScanTrigger) TriggerClean(ctx context.Context, paths []string) error {
	input := CleanMetadataInput{
		Paths:  paths,
		DryRun: false,
	}

	t.manager.Clean(ctx, input)
	logger.Debugf("[watcher] Clean job submitted for paths: %v", paths)
	return nil
}
