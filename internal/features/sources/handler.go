package sources

import (
	"net/http"

	"github.com/Aleksandr5325/crm-mini/internal/core/request"
	"github.com/Aleksandr5325/crm-mini/internal/core/respond"
)

// Handler — HTTP-обработчики источников
type Handler struct {
	service *Service
}

// NewHandler создаёт обработчики источников
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Register добавляет маршруты источников в mux
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/sources", h.list)
	mux.HandleFunc("POST /api/v1/sources", h.create)
}

// list обрабатывает GET /api/v1/sources: отдаёт все источники
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	sources, err := h.service.List(r.Context())
	if err != nil {
		respond.Error(w, r, err)
		return
	}

	respond.JSON(w, r, http.StatusOK, sources)
}

// create обрабатывает POST /api/v1/sources: создаёт источник из JSON в теле запроса
// Отвечает 201 с созданным источником, 400 при неверных данных и 409, если код уже занят
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var src Source
	if err := request.DecodeJSON(w, r, &src); err != nil {
		respond.Error(w, r, err)
		return
	}

	created, err := h.service.Create(r.Context(), src)
	if err != nil {
		respond.Error(w, r, err)
		return
	}

	respond.JSON(w, r, http.StatusCreated, created)
}
