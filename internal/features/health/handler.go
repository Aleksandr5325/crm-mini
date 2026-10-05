// Package health отвечает на вопрос «жив ли сервис».
package health

import (
	"context"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/Aleksandr5325/crm-mini/internal/core/logger"
	"github.com/Aleksandr5325/crm-mini/internal/core/respond"
)

// Pinger - всё, что умеет проверять связь. Подходит, например, *pgxpool.Pool
type Pinger interface {
	Ping(ctx context.Context) error
}

// Handler — обработчик GET /api/v1/health.
type Handler struct {
	db Pinger
}

// NewHandler создаёт обработчик проверки здоровья.
func NewHandler(db Pinger) *Handler {
	return &Handler{db: db}
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
		logger.FromContext(r.Context()).Warn("health check: database unavailable", zap.Error(err))
		code = http.StatusServiceUnavailable
		resp = response{Status: "unavailable", Database: "unavailable"}
	}

	respond.JSON(w, r, code, resp)
}
