package middleware

import (
	"net/http"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/Aleksandr5325/crm-mini/internal/core/logger"
)

// RequestID даёт каждому запросу уникальный номер и кладёт в контекст логгер с этим номером
func RequestID(log *zap.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := uuid.NewString()
		w.Header().Set("X-Request-ID", id)

		reqLog := log.With(zap.String("request_id", id))
		ctx := logger.ToContext(r.Context(), reqLog)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
