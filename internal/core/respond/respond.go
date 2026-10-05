// Package respond пишет HTTP-ответы в едином формате.
package respond

import (
	"encoding/json"
	"errors"
	"net/http"

	"go.uber.org/zap"

	"github.com/Aleksandr5325/crm-mini/internal/core/apperr"
	"github.com/Aleksandr5325/crm-mini/internal/core/logger"
)

// JSON отвечает кодом code и телом v в формате JSON.
func JSON(w http.ResponseWriter, r *http.Request, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	if err := json.NewEncoder(w).Encode(v); err != nil {
		logger.FromContext(r.Context()).Debug("write json response", zap.Error(err))
	}
}

type errorBody struct {
	Error string `json:"error"`
}

// Error превращает ошибку в HTTP-ответ. Ошибки из apperr отдаются клиенту
// с понятным кодом и текстом, все остальные — как 500 без подробностей.
func Error(w http.ResponseWriter, r *http.Request, err error) {
	code := http.StatusInternalServerError
	switch {
	case errors.Is(err, apperr.ErrNotFound):
		code = http.StatusNotFound
	case errors.Is(err, apperr.ErrInvalid):
		code = http.StatusBadRequest
	case errors.Is(err, apperr.ErrConflict):
		code = http.StatusConflict
	}

	if code == http.StatusInternalServerError {
		logger.FromContext(r.Context()).Error("internal error",
			zap.Error(err),
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
		)
		JSON(w, r, code, errorBody{Error: "internal server error"})
		return
	}

	JSON(w, r, code, errorBody{Error: err.Error()})
}
