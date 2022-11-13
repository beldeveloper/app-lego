package svc

import (
	"context"
	"github.com/beldeveloper/app-lego/internal/app"
	"github.com/beldeveloper/go-errors-context"
	"log"
	"time"
)

// WatchJobDelay defines the delay between jobs.
const WatchJobDelay = time.Second

// NewWatcher creates a new instance of the watcher service.
func NewWatcher(jobs []app.WatcherJob) Watcher {
	return Watcher{jobs: jobs}
}

// Watcher is a service that runs the sequences of jobs in a loop.
type Watcher struct {
	jobs []app.WatcherJob
}

// Watch runs the watcher.
func (s Watcher) Watch() {
	ctx := context.Background()
	var err error
	for {
		for _, j := range s.jobs {
			time.Sleep(WatchJobDelay)
			err = j.Do(ctx)
			if err != nil {
				log.Println(errors.WrapContext(err, errors.Context{
					Path:   "svc.Watcher.Watch",
					Params: errors.Params{"job": j.Name},
				}))
			}
		}
	}
}
