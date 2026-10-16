package employees

import (
	"net/http"

	"github.com/Aleksandr5325/crm-mini/internal/core/respond"
)

// Handler - HTTP-обработчик сотрудников
type Handler struct {
	service *Service
}

// NewHandler - создает обработчики сотрудников
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Register добавляет маршруты сотрудников в mux
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/employees", h.list)
}

// list обрабатывает GET /api/v1/employees: отдает всех сотрудников
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	employees, err := h.service.List(r.Context())
	if err != nil {
		respond.Error(w, r, err)
		return
	}

	respond.JSON(w, r, http.StatusOK, employees)
}
