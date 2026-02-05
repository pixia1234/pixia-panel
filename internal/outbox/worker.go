package outbox

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"pixia-panel/internal/gost"
	"pixia-panel/internal/store"
)

type Worker struct {
	store    *store.Store
	hub      *gost.Hub
	interval time.Duration
	delay    time.Duration
}

func NewWorker(store *store.Store, hub *gost.Hub, interval, retryDelay time.Duration) *Worker {
	return &Worker{store: store, hub: hub, interval: interval, delay: retryDelay}
}

type GostMessage struct {
	NodeID int64           `json:"node_id"`
	Action string          `json:"action"`
	Data   json.RawMessage `json:"data"`
}

func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.processOnce(ctx)
		}
	}
}

func (w *Worker) processOnce(ctx context.Context) {
	item, err := w.store.ClaimNextOutbox(ctx)
	if err != nil {
		if err == store.ErrNotFound {
			return
		}
		log.Printf("outbox claim error: %v", err)
		return
	}

	var msg GostMessage
	if err := json.Unmarshal(item.Payload, &msg); err != nil {
		log.Printf("outbox payload invalid: %v", err)
		_ = w.store.MarkOutboxFailed(ctx, item.ID, w.delay)
		return
	}

	if err := w.hub.Send(ctx, msg.NodeID, msg.Action, msg.Data); err != nil {
		log.Printf("gost send failed: %v", err)
		_ = w.store.MarkOutboxFailed(ctx, item.ID, w.delay)
		return
	}

	_ = w.store.MarkOutboxSuccess(ctx, item.ID)
}
