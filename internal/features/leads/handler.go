package leads

import (
	"net/http"

	"github.com/Aleksandr5325/crm-mini/internal/core/request"
	"github.com/Aleksandr5325/crm-mini/internal/core/respond"
)

// Handler — HTTP-обработчики лидов
type Handler struct {
	service *Service
}

// NewHandler создаёт обработчики лидов
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Register добавляет маршруты лидов в mux
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/leads", h.create)
}

// create обрабатывает POST /api/v1/leads: создаёт лида в стадии new
// Отвечает 201 с созданным лидом и 400 при неверных данных
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var in CreateInput
	if err := request.DecodeJSON(w, r, &in); err != nil {
		respond.Error(w, r, err)
		return
	}

	lead, err := h.service.Create(r.Context(), in)
	if err != nil {
		respond.Error(w, r, err)
		return
	}

	respond.JSON(w, r, http.StatusCreated, lead)
}
