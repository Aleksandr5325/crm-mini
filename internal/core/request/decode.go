// Package request читает данные из HTTP-запросов.
package request

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Aleksandr5325/crm-mini/internal/core/apperr"
)

// maxBodySize — максимальный размер тела запроса: 1 МБ.
const maxBodySize = 1 << 20

// DecodeJSON читает JSON из тела запроса в dst
// Ошибки формата возвращает как apperr.ErrInvalid, чтобы клиент получил 400
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("invalid json body: %v: %w", err, apperr.ErrInvalid)
	}

	return nil
}
