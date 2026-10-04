// Package middleware содержит обертки для запросов.
package middleware

import (
	"net/http"

	"go.uber.org/zap"

	"github.com/Aleksandr5325/crm-mini/internal/core/logger"
)

// Recover помогает ловить паники
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}

			logger.FromContext(r.Context()).Error("panic in handler",
				zap.Any("panic", rec),
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
			)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"internal server error"}`))
		}()

		next.ServeHTTP(w, r)
	})
}
