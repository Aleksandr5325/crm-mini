// Package health отвечает на вопрос «жив ли сервис».
package health

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// Pinger - всё, что умеет проверять связь. Подходит, например, *pgxpool.Pool
type Pinger interface {
	Ping(ctx context.Context) error
}

// Handler — обработчик GET /api/v1/health.
type Handler struct {
	db  Pinger
	log *zap.Logger
}

// NewHandler создаёт обработчик проверки здоровья.
func NewHandler(db Pinger, log *zap.Logger) *Handler {
	return &Handler{db: db, log: log}
}

type response struct {
	Status   string `json:"status"`
	Database string `json:"database"`
}

// ServeHTTP отвечает 200, если база доступна, и 503, если нет.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	code := http.StatusOK
	resp := response{Status: "ok", Database: "ok"}

	if err := h.db.Ping(ctx); err != nil {
		h.log.Warn("health check: database unavailable", zap.Error(err))
		code = http.StatusServiceUnavailable
		resp = response{Status: "unavailable", Database: "unavailable"}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(resp)
}
