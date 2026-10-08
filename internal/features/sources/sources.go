// Package sources — справочник источников лидов: сайт, олимпиада и т.п
package sources

import (
	"fmt"
	"regexp"
	"unicode/utf8"

	"github.com/Aleksandr5325/crm-mini/internal/core/apperr"
)

// Source — источник, откуда пришёл лид
type Source struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

var codePattern = regexp.MustCompile(`^[a-z_]{2,50}$`)

// Validate проверяет поля источника по тем же правилам, что и CHECK в базе.
func (s Source) Validate() error {
	if !codePattern.MatchString(s.Code) {
		return fmt.Errorf("code must be 2-50 lowercase latin letters or underscores: %w", apperr.ErrInvalid)
	}

	if n := utf8.RuneCountInString(s.Name); n < 1 || n > 100 {
		return fmt.Errorf("name must be 1-100 characters long: %w", apperr.ErrInvalid)
	}

	return nil
}
