package leads

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/Aleksandr5325/crm-mini/internal/core/apperr"
)

func TestStageCanMoveTo(t *testing.T) {
	tests := []struct {
		name    string
		from    Stage
		to      Stage
		wantErr error
	}{
		{"new to qualified", StageNew, StageQualified, nil},
		{"qualified to proposal", StageQualified, StageProposal, nil},
		{"proposal to won", StageProposal, StageWon, nil},
		{"new to lost", StageNew, StageLost, nil},
		{"proposal to lost", StageProposal, StageLost, nil},
		{"skip from new to won", StageNew, StageWon, apperr.ErrConflict},
		{"skip from new to proposal", StageNew, StageProposal, apperr.ErrConflict},
		{"back from proposal to new", StageProposal, StageNew, apperr.ErrConflict},
		{"same stage", StageQualified, StageQualified, apperr.ErrConflict},
		{"from won", StageWon, StageLost, apperr.ErrConflict},
		{"from lost", StageLost, StageNew, apperr.ErrConflict},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.from.CanMoveTo(tt.to)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("CanMoveTo(%q -> %q) = %v, want %v", tt.from, tt.to, err, tt.wantErr)
			}
		})
	}
}

func TestLeadValidate(t *testing.T) {
	ptr := func(s string) *string { return &s }

	tests := []struct {
		name   string
		change func(l *Lead)
		wantOK bool
	}{
		{"valid lead", func(l *Lead) {}, true},
		{"no source", func(l *Lead) { l.SourceCode = "" }, false},
		{"no owner", func(l *Lead) { l.OwnerID = uuid.Nil }, false},
		{"empty name", func(l *Lead) { l.FullName = "" }, false},
		{"255 cyrillic letters", func(l *Lead) { l.FullName = strings.Repeat("я", 255) }, true},
		{"256 letters", func(l *Lead) { l.FullName = strings.Repeat("я", 256) }, false},
		{"phone without plus", func(l *Lead) { l.Phone = ptr("89991234567") }, false},
		{"email only", func(l *Lead) { l.Phone = nil; l.Email = ptr("anna@example.ru") }, true},
		{"email without @", func(l *Lead) { l.Email = ptr("example.ru") }, false},
		{"no contacts", func(l *Lead) { l.Phone = nil; l.Email = nil }, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := Lead{
				SourceCode: "site",
				OwnerID:    uuid.New(),
				FullName:   "Иван Петров",
				Phone:      ptr("+79991234567"),
			}
			tt.change(&l)

			err := l.Validate()
			if tt.wantOK && err != nil {
				t.Errorf("Validate() = %v, want nil", err)
			}
			if !tt.wantOK && !errors.Is(err, apperr.ErrInvalid) {
				t.Errorf("Validate() = %v, want apperr.ErrInvalid", err)
			}
		})
	}
}
