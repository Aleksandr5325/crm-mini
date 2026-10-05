package apperr

import (
	"errors"
	"fmt"
	"testing"
)

// TestWrappedErrorKeepsItsKind проверяет ошибки на правильность
func TestWrappedErrorKeepsItsKind(t *testing.T) {
	err := fmt.Errorf("get lead 42: %w", ErrNotFound)

	if !errors.Is(err, ErrNotFound) {
		t.Errorf("errors.Is(%q, ErrNotFound) = false, want true", err)
	}

	if err == ErrNotFound {
		t.Errorf("err == ErrNotFound = true, want false: обёрнутая ошибка это новое значение")
	}

	if errors.Is(err, ErrConflict) {
		t.Errorf("errors.Is(%q, ErrConflict) = true, want false", err)
	}
}
