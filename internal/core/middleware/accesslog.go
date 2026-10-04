package middleware

import (
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/Aleksandr5325/crm-mini/internal/core/logger"
)

// statusRecorder запоминает код ответа, который записал обработчик
type statusRecorder struct {
	http.ResponseWriter
	status int
}

// WriteHeader запоминает код и передает его настоящему ResponseWriter
func (rec *statusRecorder) WriteHeader(code int) {
	rec.status = code
	rec.ResponseWriter.WriteHeader(code)
}

// AccessLog пишет одну строку лога на каждый запрос: метод, путь, код ответа и время
func AccessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rec, r)

		logger.FromContext(r.Context()).Info("http request",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Int("status", rec.status),
			zap.Duration("duration", time.Since(start)),
		)
	})
}
