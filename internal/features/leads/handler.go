package leads

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"

	"github.com/Aleksandr5325/crm-mini/internal/core/apperr"
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
	mux.HandleFunc("GET /api/v1/leads/{id}", h.get)
	mux.HandleFunc("GET /api/v1/leads", h.list)
	mux.HandleFunc("PATCH /api/v1/leads/{id}", h.update)
	mux.HandleFunc("DELETE /api/v1/leads/{id}", h.remove)
	mux.HandleFunc("POST /api/v1/leads/{id}/stage", h.moveStage)
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

// get обрабатывает GET /api/v1/leads/{id}: отдаёт лида или 404, если его нет.
func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respond.Error(w, r, fmt.Errorf("invalid lead id: %w", apperr.ErrInvalid))
		return
	}

	lead, err := h.service.Get(r.Context(), id)
	if err != nil {
		respond.Error(w, r, err)
		return
	}

	respond.JSON(w, r, http.StatusOK, lead)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := ListFilter{
		SourceCode: q.Get("source"),
		Stage:      Stage(q.Get("stage")),
	}

	if v := q.Get("owner_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			respond.Error(w, r, fmt.Errorf("invalid owner_id: %w", apperr.ErrInvalid))
			return
		}
		f.OwnerID = id
	}

	var err error
	if f.Limit, err = request.QueryInt(r, "limit"); err != nil {
		respond.Error(w, r, err)
		return
	}
	if f.Offset, err = request.QueryInt(r, "offset"); err != nil {
		respond.Error(w, r, err)
		return
	}

	leads, err := h.service.List(r.Context(), f)
	if err != nil {
		respond.Error(w, r, err)
		return
	}

	respond.JSON(w, r, http.StatusOK, leads)
}

// update обрабатывает PATCH /api/v1/leads/{id}: меняет присланные поля лида
// Отвечает 200 с обновлённым лидом, 409 если версия устарела
func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respond.Error(w, r, fmt.Errorf("invalid lead id: %w", apperr.ErrInvalid))
		return
	}

	var in UpdateInput
	if err := request.DecodeJSON(w, r, &in); err != nil {
		respond.Error(w, r, err)
		return
	}

	lead, err := h.service.Update(r.Context(), id, in)
	if err != nil {
		respond.Error(w, r, err)
		return
	}

	respond.JSON(w, r, http.StatusOK, lead)
}

// remove обрабатывает DELETE /api/v1/leads/{id}: мягко удаляет лида. Отвечает 204 без тела
func (h *Handler) remove(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respond.Error(w, r, fmt.Errorf("invalid lead id: %w", apperr.ErrInvalid))
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		respond.Error(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// moveStage обрабатывает POST /api/v1/leads/{id}/stage: переводит лида в другую стадию
// Отвечает 200 с лидом, 409 если правила воронки не разрешают переход или версия устарела
func (h *Handler) moveStage(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respond.Error(w, r, fmt.Errorf("invalid lead id: %w", apperr.ErrInvalid))
		return
	}

	var in MoveStageInput
	if err := request.DecodeJSON(w, r, &in); err != nil {
		respond.Error(w, r, err)
		return
	}

	lead, err := h.service.MoveStage(r.Context(), id, in)
	if err != nil {
		respond.Error(w, r, err)
		return
	}

	respond.JSON(w, r, http.StatusOK, lead)
}
