// Package middleware содержит обертки для запросов.
package middleware

import (
	"fmt"
	"net/http"

	"github.com/Aleksandr5325/crm-mini/internal/core/respond"
)

// Recover перехватывает панику в обработчике и превращает её в ответ 500
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}

			respond.Error(w, r, fmt.Errorf("panic: %v", rec))
		}()

		next.ServeHTTP(w, r)
	})
}
