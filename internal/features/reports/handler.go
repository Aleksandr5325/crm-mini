package reports

import (
	"net/http"

	"github.com/Aleksandr5325/crm-mini/internal/core/respond"
)

// Handler — HTTP-обработчики отчётов
type Handler struct {
	service *Service
}

// NewHandler создаёт обработчики отчётов
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Register добавляет маршруты отчётов в mux
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/reports/funnel", h.funnel)
}

// funnel обрабатывает GET /api/v1/reports/funnel: отчёт «Воронка продаж»
func (h *Handler) funnel(w http.ResponseWriter, r *http.Request) {
	f, err := h.service.Funnel(r.Context())
	if err != nil {
		respond.Error(w, r, err)
		return
	}

	respond.JSON(w, r, http.StatusOK, f)
}
