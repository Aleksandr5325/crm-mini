package request

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/Aleksandr5325/crm-mini/internal/core/apperr"
)

// QueryInt читает целое число из параметра адреса ?name=…
// Если параметра нет — возвращает 0. Если это не число — apperr.ErrInvalid
func QueryInt(r *http.Request, name string) (int, error) {
	v := r.URL.Query().Get(name)
	if v == "" {
		return 0, nil
	}

	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", name, apperr.ErrInvalid)
	}

	return n, nil
}
