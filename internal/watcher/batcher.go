package watcher

import (
	"sync"
	"time"
)

// EventBatcher batches file events with debouncing to avoid triggering
// scans for every individual file when copying large numbers of files.
type EventBatcher struct {
	debounceDelay time.Duration
	processor     func(paths []string)

	mutex   sync.Mutex
	pending map[string]struct{}
	timer   *time.Timer
	stopped bool
}

// NewEventBatcher creates a new event batcher.
// debounceDelay is the time to wait after the last event before processing.
// processor is the function to call with accumulated paths when the debounce timer fires.
func NewEventBatcher(debounceDelay time.Duration, processor func([]string)) *EventBatcher {
	return &EventBatcher{
		debounceDelay: debounceDelay,
		processor:     processor,
		pending:       make(map[string]struct{}),
	}
}

// Add adds a path to the pending set and resets the debounce timer.
func (b *EventBatcher) Add(path string) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if b.stopped {
		return
	}

	// Add path to pending set (deduplicates automatically)
	b.pending[path] = struct{}{}

	// Reset or start the debounce timer
	if b.timer != nil {
		b.timer.Stop()
	}
	b.timer = time.AfterFunc(b.debounceDelay, b.flush)
}

// flush processes all pending paths and clears the pending set.
func (b *EventBatcher) flush() {
	b.mutex.Lock()

	if b.stopped || len(b.pending) == 0 {
		b.mutex.Unlock()
		return
	}

	// Collect paths and clear pending
	paths := make([]string, 0, len(b.pending))
	for p := range b.pending {
		paths = append(paths, p)
	}
	b.pending = make(map[string]struct{})

	if b.timer != nil {
		b.timer.Stop()
		b.timer = nil
	}

	b.mutex.Unlock()

	// Process outside the lock
	if b.processor != nil {
		b.processor(paths)
	}
}

// PendingCount returns the number of pending paths.
func (b *EventBatcher) PendingCount() int {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return len(b.pending)
}

// Stop stops the batcher and cancels any pending flush.
func (b *EventBatcher) Stop() {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	b.stopped = true
	if b.timer != nil {
		b.timer.Stop()
		b.timer = nil
	}
	b.pending = make(map[string]struct{})
}
