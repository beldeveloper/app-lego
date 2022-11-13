package app

import "context"

// WatcherJob is a job that is run frequently by the watcher service.
type WatcherJob struct {
	Name string
	Do   func(ctx context.Context) error
}
