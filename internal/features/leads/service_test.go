package leads

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Aleksandr5325/crm-mini/internal/core/apperr"
)

// fakeStore — поддельное хранилище для тестов сервиса: вместо базы — поля структуры.
// Встроенный интерфейс Store закрывает методы, которые тесты не используют:
// если сервис вызовет такой метод, тест упадёт с паникой
type fakeStore struct {
	Store

	lead Lead

	updated *Lead
	movedTo Stage
}

func (f *fakeStore) Get(ctx context.Context, id uuid.UUID) (Lead, error) {
	return f.lead, nil
}

func (f *fakeStore) Update(ctx context.Context, l Lead) (Lead, error) {
	f.updated = &l
	l.Version++
	return l, nil
}

func (f *fakeStore) MoveStage(ctx context.Context, l Lead, to Stage, at time.Time) (Lead, error) {
	f.movedTo = to
	l.CurrentStage = to
	l.Version++
	return l, nil
}

// testLead — правильный лид в стадии new, версия 3, только с телефоном.
func testLead() Lead {
	phone := "+79991234567"
	return Lead{
		ID:           uuid.New(),
		Version:      3,
		SourceCode:   "site",
		OwnerID:      uuid.New(),
		CurrentStage: StageNew,
		FullName:     "Иван Петров",
		Phone:        &phone,
	}
}

func TestServiceUpdate(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr error
	}{
		{"changes only sent fields", `{"version":3,"comment":"  Перезвонить  "}`, nil},
		{"stale version", `{"version":2,"comment":"x"}`, apperr.ErrConflict},
		{"full_name cannot be null", `{"version":3,"full_name":null}`, apperr.ErrInvalid},
		{"cannot clear the only contact", `{"version":3,"phone":null}`, apperr.ErrInvalid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeStore{lead: testLead()}
			svc := NewService(store)

			var in UpdateInput
			if err := json.Unmarshal([]byte(tt.body), &in); err != nil {
				t.Fatalf("bad test body: %v", err)
			}

			_, err := svc.Update(context.Background(), store.lead.ID, in)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Update() error = %v, want %v", err, tt.wantErr)
			}

			if tt.wantErr != nil {
				if store.updated != nil {
					t.Error("store.Update was called, but the service returned an error")
				}
				return
			}

			if store.updated == nil {
				t.Fatalf("store.Update was not called")
			}
			if c := store.updated.Comment; c == nil || *c != "Перезвонить" {
				t.Errorf("comment = %v, want trimmed \"Перезвонить\"", c)
			}
			if store.updated.Phone == nil {
				t.Errorf("phone was not in the request, but it was cleared")
			}
		})
	}
}

func TestServiceMoveStage(t *testing.T) {
	tests := []struct {
		name    string
		in      MoveStageInput
		wantErr error
	}{
		{"next stage", MoveStageInput{Version: 3, Stage: StageQualified}, nil},
		{"skip stages", MoveStageInput{Version: 3, Stage: StageWon}, apperr.ErrConflict},
		{"unknown stage", MoveStageInput{Version: 3, Stage: "closed"}, apperr.ErrInvalid},
		{"stale version", MoveStageInput{Version: 2, Stage: StageQualified}, apperr.ErrConflict},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeStore{lead: testLead()}

			lead, err := NewService(store).MoveStage(context.Background(), store.lead.ID, tt.in)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("MoveStage() error = %v, want: %v", err, tt.wantErr)
			}

			if tt.wantErr != nil {
				if store.movedTo != "" {
					t.Errorf("store.MoveStage was called with %q, but the service returned an error", store.movedTo)
				}
				return
			}

			if store.movedTo != tt.in.Stage {
				t.Errorf("store.MoveStage got stage %q, want %q", store.movedTo, tt.in.Stage)
			}
			if lead.CurrentStage != tt.in.Stage {
				t.Errorf("returned lead stage = %q, want %q", lead.CurrentStage, tt.in.Stage)
			}
		})
	}
}
