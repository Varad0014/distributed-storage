package service

import (
	"context"
	"log"
	"time"
)

type ReplicationWorker struct {
	syncFunc func(context.Context) error
	interval time.Duration
}

func NewReplicationWorker(
	syncFunc func(context.Context) error,
	interval time.Duration,
) *ReplicationWorker {
	return &ReplicationWorker{
		syncFunc: syncFunc,
		interval: interval,
	}
}

func (w *ReplicationWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := w.syncFunc(ctx); err != nil {
				log.Printf("replication sync failed: %v", err)
			}

		case <-ctx.Done():
			log.Println("replication worker stopped")
			return
		}
	}
}
