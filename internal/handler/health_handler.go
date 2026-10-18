package handler

import (
	"net/http"
	"time"

	"event-booking-system/internal/worker"
)

// HealthHandler exposes liveness and worker pool metrics.
type HealthHandler struct {
	pool    *worker.Pool
	startAt time.Time
}

func NewHealthHandler(pool *worker.Pool) *HealthHandler {
	return &HealthHandler{pool: pool, startAt: time.Now()}
}

func (h *HealthHandler) Health(w http.ResponseWriter, _ *http.Request) {
	processed, failed, queueLen := h.pool.Stats()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":           "ok",
		"uptime":           time.Since(h.startAt).String(),
		"jobs_processed":   processed,
		"jobs_failed":      failed,
		"jobs_queue_length": queueLen,
	})
}
