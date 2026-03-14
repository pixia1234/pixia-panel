package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	_ "modernc.org/sqlite"

	"github.com/robfig/cron/v3"

	"pixia-panel/internal/db"
	"pixia-panel/internal/flow"
	"pixia-panel/internal/gost"
	httpapi "pixia-panel/internal/http"
	"pixia-panel/internal/migrate"
	"pixia-panel/internal/outbox"
	"pixia-panel/internal/store"
	"pixia-panel/internal/tasks"
)

func main() {
	dbPath := getenvDefault("PIXIA_DB_PATH", filepath.Join(".", "pixia.db"))
	addr := getenvDefault("PIXIA_HTTP_ADDR", ":6365")
	wsPath := getenvDefault("PIXIA_WS_PATH", "/system-info")
	interval := getenvDurationDefault("PIXIA_OUTBOX_INTERVAL", 500*time.Millisecond)
	retryDelay := getenvDurationDefault("PIXIA_OUTBOX_RETRY_DELAY", 5*time.Second)
	maxRetryDelay := getenvDurationDefault("PIXIA_OUTBOX_MAX_RETRY_DELAY", 5*time.Minute)
	maxRetries := getenvInt64Default("PIXIA_OUTBOX_MAX_RETRIES", 24)
	batchSize := getenvIntDefault("PIXIA_OUTBOX_BATCH_SIZE", 20)
	maxProcessingAge := getenvDurationDefault("PIXIA_OUTBOX_MAX_PROCESSING_AGE", 2*time.Minute)
	staleCheckInterval := getenvDurationDefault("PIXIA_OUTBOX_STALE_CHECK_INTERVAL", 30*time.Second)
	jwtSecret := []byte(getenvDefault("PIXIA_JWT_SECRET", "pixia-secret"))
	jwtTTL := getenvDurationDefault("PIXIA_JWT_TTL", 24*time.Hour)

	conn, err := db.Open(dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer conn.Close()

	if err := migrate.Apply(conn, filepath.Join(".", "migrations")); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	store := store.New(conn)
	flowService := flow.New(store)
	hub := gost.NewHub()
	hub.SetJWTSecret(jwtSecret)

	server := httpapi.NewServer(store, flowService, hub, jwtSecret, jwtTTL)
	router := http.NewServeMux()
	server.Register(router)
	lookup := nodeLookup{store: store, api: server}
	router.Handle(wsPath, hub.ServeWS(lookup))
	if wsPath != "/ws" {
		router.Handle("/ws", hub.ServeWS(lookup))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	worker := outbox.NewWorker(store, hub, outbox.WorkerOptions{
		Interval:           interval,
		RetryDelay:         retryDelay,
		MaxRetryDelay:      maxRetryDelay,
		MaxRetries:         maxRetries,
		BatchSize:          batchSize,
		MaxProcessingAge:   maxProcessingAge,
		StaleCheckInterval: staleCheckInterval,
	})
	go worker.Run(ctx)

	scheduler := tasks.New(store, server)
	c := cron.New()
	_, _ = c.AddFunc("0 0 * * *", func() { scheduler.DailyReset(ctx) })
	_, _ = c.AddFunc("0 * * * *", func() { scheduler.HourlyStatistics(ctx) })
	c.Start()

	handler := httpapi.WithCORS(router)
	log.Printf("pixia-panel listening on %s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func getenvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getenvDurationDefault(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

func getenvIntDefault(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getenvInt64Default(key string, def int64) int64 {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return n
		}
	}
	return def
}

type nodeLookup struct {
	store *store.Store
	api   *httpapi.Server
}

func (n nodeLookup) LookupBySecret(ctx context.Context, secret string) (int64, error) {
	return n.store.LookupBySecret(ctx, secret)
}

func (n nodeLookup) UpdateNodeStatus(ctx context.Context, nodeID int64, status int64, version *string, http, tls, socks *int64) error {
	return n.store.UpdateNodeStatus(ctx, nodeID, status, version, http, tls, socks)
}

func (n nodeLookup) ResyncNode(ctx context.Context, nodeID int64) {
	n.api.ResyncNode(ctx, nodeID)
}
