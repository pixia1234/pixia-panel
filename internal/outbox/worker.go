package outbox

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	"pixia-panel/internal/gost"
	"pixia-panel/internal/store"
)

type Worker struct {
	store              *store.Store
	hub                *gost.Hub
	interval           time.Duration
	delay              time.Duration
	maxDelay           time.Duration
	maxRetries         int64
	batchSize          int
	maxProcessingAge   time.Duration
	staleCheckInterval time.Duration
	lastStaleCheck     time.Time
}

type WorkerOptions struct {
	Interval           time.Duration
	RetryDelay         time.Duration
	MaxRetryDelay      time.Duration
	MaxRetries         int64
	BatchSize          int
	MaxProcessingAge   time.Duration
	StaleCheckInterval time.Duration
}

func NewWorker(store *store.Store, hub *gost.Hub, opts WorkerOptions) *Worker {
	if opts.Interval <= 0 {
		opts.Interval = 500 * time.Millisecond
	}
	if opts.RetryDelay <= 0 {
		opts.RetryDelay = 5 * time.Second
	}
	if opts.MaxRetryDelay <= 0 {
		opts.MaxRetryDelay = 5 * time.Minute
	}
	if opts.MaxRetryDelay < opts.RetryDelay {
		opts.MaxRetryDelay = opts.RetryDelay
	}
	if opts.BatchSize <= 0 {
		opts.BatchSize = 20
	}
	if opts.MaxProcessingAge <= 0 {
		opts.MaxProcessingAge = 2 * time.Minute
	}
	if opts.StaleCheckInterval <= 0 {
		opts.StaleCheckInterval = 30 * time.Second
	}

	return &Worker{
		store:              store,
		hub:                hub,
		interval:           opts.Interval,
		delay:              opts.RetryDelay,
		maxDelay:           opts.MaxRetryDelay,
		maxRetries:         opts.MaxRetries,
		batchSize:          opts.BatchSize,
		maxProcessingAge:   opts.MaxProcessingAge,
		staleCheckInterval: opts.StaleCheckInterval,
	}
}

type GostMessage struct {
	NodeID int64           `json:"node_id"`
	Action string          `json:"action"`
	Data   json.RawMessage `json:"data"`
}

const commandResponseTimeout = 10 * time.Second

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
	w.requeueStaleProcessing(ctx)

	items, err := w.store.ClaimNextOutboxBatch(ctx, w.batchSize)
	if err != nil {
		if err == store.ErrNotFound {
			return
		}
		log.Printf("outbox claim error: %v", err)
		return
	}

	for i := range items {
		w.processItem(ctx, &items[i])
	}
}

func (w *Worker) processItem(ctx context.Context, item *store.OutboxItem) {
	if item == nil {
		return
	}

	var msg GostMessage
	if err := json.Unmarshal(item.Payload, &msg); err != nil {
		log.Printf("outbox payload invalid: %v", err)
		_ = w.store.MarkOutboxDead(ctx, item.ID, false)
		return
	}

	exists, err := w.store.NodeExists(ctx, msg.NodeID)
	if err != nil {
		log.Printf("outbox node lookup failed: node_id=%d err=%v", msg.NodeID, err)
		w.markFailed(ctx, item)
		return
	}
	if !exists {
		log.Printf("outbox node missing, mark dead: node_id=%d action=%s", msg.NodeID, msg.Action)
		_ = w.store.MarkOutboxDead(ctx, item.ID, false)
		return
	}

	resp, err := w.hub.SendAndWait(ctx, msg.NodeID, msg.Action, msg.Data, commandResponseTimeout)
	if err != nil {
		log.Printf("gost send failed: %v", err)
		w.markFailed(ctx, item)
		return
	}

	if !resp.Success {
		if shouldAcknowledgeAsSuccess(msg.Action, resp.Message) {
			_ = w.store.MarkOutboxSuccess(ctx, item.ID)
			return
		}
		log.Printf("gost command failed: action=%s node_id=%d message=%s", msg.Action, msg.NodeID, resp.Message)
		w.markFailed(ctx, item)
		return
	}

	_ = w.store.MarkOutboxSuccess(ctx, item.ID)
}

func (w *Worker) markFailed(ctx context.Context, item *store.OutboxItem) {
	if item == nil {
		return
	}

	if w.maxRetries > 0 && item.RetryCount+1 >= w.maxRetries {
		_ = w.store.MarkOutboxDead(ctx, item.ID, true)
		return
	}

	delay := w.retryDelay(item.RetryCount)
	_ = w.store.MarkOutboxFailed(ctx, item.ID, delay)
}

func (w *Worker) retryDelay(retryCount int64) time.Duration {
	if retryCount <= 0 {
		return w.delay
	}

	d := w.delay
	for i := int64(0); i < retryCount; i++ {
		if d >= w.maxDelay {
			return w.maxDelay
		}
		if d > w.maxDelay/2 {
			return w.maxDelay
		}
		d *= 2
	}
	if d > w.maxDelay {
		return w.maxDelay
	}
	return d
}

func (w *Worker) requeueStaleProcessing(ctx context.Context) {
	if w.maxProcessingAge <= 0 {
		return
	}
	if w.staleCheckInterval > 0 && !w.lastStaleCheck.IsZero() && time.Since(w.lastStaleCheck) < w.staleCheckInterval {
		return
	}

	affected, err := w.store.RequeueStaleOutboxProcessing(ctx, w.maxProcessingAge)
	w.lastStaleCheck = time.Now()
	if err != nil {
		log.Printf("outbox stale requeue error: %v", err)
		return
	}
	if affected > 0 {
		log.Printf("outbox stale processing requeued: %d", affected)
	}
}

func shouldAcknowledgeAsSuccess(action, message string) bool {
	a := strings.TrimSpace(action)
	m := strings.ToLower(strings.TrimSpace(message))
	if m == "" {
		return false
	}

	if strings.HasPrefix(a, "Add") && strings.Contains(m, "already exists") {
		return true
	}
	if strings.HasPrefix(a, "Delete") && strings.Contains(m, "not found") {
		return true
	}

	switch a {
	case "PauseService", "ResumeService":
		if strings.Contains(m, "not found") {
			return true
		}
	}

	return false
}
