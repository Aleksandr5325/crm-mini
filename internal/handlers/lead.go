package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"back/internal/models"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

type LeadHandler struct {
	DB *gorm.DB
}

// stageOrder задаёт порядок стадий в воронке.
// Терминальные стадии (won/lost) имеют одинаковый порядок 4 — это нормально,
// потому что из них всё равно нельзя никуда уйти (проверяем отдельно).
var stageOrder = map[string]int{
	"new":       1,
	"qualified": 2,
	"proposal":  3,
	"won":       4,
	"lost":      4,
}

// writeJSON — короткий helper, чтобы не повторять три строки в каждом хендлере.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError — то же самое, но возвращает {"error": "..."}.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// =====================================================================
// GET /leads — список лидов с фильтрами
// =====================================================================
func (h *LeadHandler) GetLeads(w http.ResponseWriter, r *http.Request) {
	query := h.DB.Model(&models.Lead{})

	q := r.URL.Query()

	// Фильтр по источнику
	if source := q.Get("source"); source != "" {
		query = query.Where("source = ?", source)
	}
	// Фильтр по менеджеру
	if owner := q.Get("owner"); owner != "" {
		query = query.Where("owner = ?", owner)
	}
	// Фильтр по стадии — может быть несколько через запятую: stage=new,qualified
	if stage := q.Get("stage"); stage != "" {
		stages := strings.Split(stage, ",")
		// Чистим пробелы и пустые элементы
		clean := make([]string, 0, len(stages))
		for _, s := range stages {
			s = strings.TrimSpace(s)
			if s != "" {
				clean = append(clean, s)
			}
		}
		if len(clean) > 0 {
			query = query.Where("current_stage IN ?", clean)
		}
	}
	// Фильтр по периоду создания
	if from := q.Get("created_from"); from != "" {
		// Принимаем YYYY-MM-DD
		if t, err := time.Parse("2006-01-02", from); err == nil {
			query = query.Where("created_at >= ?", t)
		}
	}
	if to := q.Get("created_to"); to != "" {
		if t, err := time.Parse("2006-01-02", to); err == nil {
			// Включительно: до конца дня
			query = query.Where("created_at < ?", t.AddDate(0, 0, 1))
		}
	}
	// Поиск q — LIKE по full_name, email, phone
	if needle := q.Get("q"); needle != "" {
		like := "%" + needle + "%"
		query = query.Where(
			"full_name ILIKE ? OR email ILIKE ? OR phone ILIKE ?",
			like, like, like,
		)
	}

	var leads []models.Lead
	if err := query.Order("created_at DESC").Find(&leads).Error; err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, leads)
}

// =====================================================================
// GET /leads/{id} — один лид
// =====================================================================
func (h *LeadHandler) GetLead(w http.ResponseWriter, r *http.Request) {
	leadID := mux.Vars(r)["id"]

	var lead models.Lead
	if err := h.DB.Where("lead_id = ?", leadID).First(&lead).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(w, http.StatusNotFound, "lead not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, lead)
}

// =====================================================================
// GET /leads/{id}/stages — история стадий лида
// =====================================================================
func (h *LeadHandler) GetLeadStages(w http.ResponseWriter, r *http.Request) {
	leadID := mux.Vars(r)["id"]

	var stages []models.LeadStage
	if err := h.DB.Where("lead_id = ?", leadID).Order("entered_at").Find(&stages).Error; err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if len(stages) == 0 {
		writeError(w, http.StatusNotFound, "lead not found")
		return
	}

	writeJSON(w, http.StatusOK, stages)
}

// =====================================================================
// POST /leads — создание лида
// =====================================================================
type createLeadRequest struct {
	FullName     string `json:"full_name"`
	Phone        string `json:"phone"`
	Email        string `json:"email"`
	Source       string `json:"source"`
	Owner        string `json:"owner"`
	CurrentStage string `json:"current_stage"`
	Comment      string `json:"comment"`
}

func (h *LeadHandler) CreateLead(w http.ResponseWriter, r *http.Request) {
	var req createLeadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if req.Source == "" {
		writeError(w, http.StatusBadRequest, "source is required")
		return
	}

	// Дефолт стадии
	if req.CurrentStage == "" {
		req.CurrentStage = "new"
	}
	if _, ok := stageOrder[req.CurrentStage]; !ok {
		writeError(w, http.StatusBadRequest, "invalid current_stage")
		return
	}

	// Проверяем, что source существует
	var src models.Source
	if err := h.DB.Where("code = ?", req.Source).First(&src).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(w, http.StatusBadRequest, "source not found: "+req.Source)
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	now := time.Now()
	newID, err := h.generateLeadID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate lead_id: "+err.Error())
		return
	}

	lead := models.Lead{
		LeadID:       newID,
		CreatedAt:    now,
		UpdatedAt:    now,
		Source:       req.Source,
		CurrentStage: req.CurrentStage,
		Owner:        req.Owner,
		FullName:     req.FullName,
		Phone:        req.Phone,
		Email:        req.Email,
		Comment:      req.Comment,
	}

	// 1. Сохраняем в PostgreSQL (ваш существующий код с транзакцией)
	err = h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&lead).Error; err != nil {
			return err
		}
		stage := models.LeadStage{
			LeadID:    lead.LeadID,
			Stage:     lead.CurrentStage,
			EnteredAt: now,
		}
		return tx.Create(&stage).Error
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create lead: "+err.Error())
		return
	}

	// 2. СРАЗУ отвечаем фронтенду, чтобы пользователь не ждал синхронизацию с 1С
	writeJSON(w, http.StatusCreated, lead)
}

// generateLeadID ищет максимальный номер существующих ID вида LEAD-NNN
// и возвращает следующий. Если ничего не нашли — LEAD-001.
func (h *LeadHandler) generateLeadID() (string, error) {
	var ids []string

	// Вытаскиваем все существующие lead_id из базы
	if err := h.DB.Model(&models.Lead{}).Pluck("lead_id", &ids).Error; err != nil {
		return "", err
	}

	max := 0
	for _, id := range ids {
		// Пробуем конвертировать строку в число
		n, err := strconv.Atoi(id)
		if err != nil {
			continue // Если в базе остались старые ID типа "LEAD-001", просто пропускаем их
		}
		if n > max {
			max = n
		}
	}

	// Возвращаем максимальное число + 1 в виде строки (фронт ожидает строку)
	return strconv.Itoa(max + 1), nil
}

// =====================================================================
// PATCH /leads/{id} — частичное обновление
// =====================================================================
// Используем *string чтобы отличить "поле не передано" от "поле передано пустым".
// В JSON это работает так:
//
//	{"phone": ""}    -> Phone = &"", обнуляем
//	{}               -> Phone = nil, не трогаем
type patchLeadRequest struct {
	FullName *string `json:"full_name"`
	Phone    *string `json:"phone"`
	Email    *string `json:"email"`
	Source   *string `json:"source"`
	Owner    *string `json:"owner"`
	Comment  *string `json:"comment"`

	// Эти поля менять нельзя — если придут, ругаемся
	CurrentStage *string `json:"current_stage"`
	LeadID       *string `json:"lead_id"`
	CreatedAt    *string `json:"created_at"`
}

func (h *LeadHandler) PatchLead(w http.ResponseWriter, r *http.Request) {
	leadID := mux.Vars(r)["id"]

	var req patchLeadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if req.CurrentStage != nil {
		writeError(w, http.StatusBadRequest, "current_stage cannot be changed via PATCH, use POST /leads/{id}/move")
		return
	}
	if req.LeadID != nil || req.CreatedAt != nil {
		writeError(w, http.StatusBadRequest, "lead_id and created_at are immutable")
		return
	}

	// Достаём существующего лида
	var lead models.Lead
	if err := h.DB.Where("lead_id = ?", leadID).First(&lead).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(w, http.StatusNotFound, "lead not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Если меняется source — проверяем, что он есть в справочнике
	if req.Source != nil && *req.Source != lead.Source {
		var src models.Source
		if err := h.DB.Where("code = ?", *req.Source).First(&src).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				writeError(w, http.StatusBadRequest, "source not found: "+*req.Source)
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	// Собираем map с полями для обновления.
	// Используем Updates(map) вместо Save(&lead), чтобы GORM апдейтил только
	// то, что мы реально передали (а не перезаписывал всё).
	updates := map[string]interface{}{}
	if req.FullName != nil {
		updates["full_name"] = *req.FullName
	}
	if req.Phone != nil {
		updates["phone"] = *req.Phone
	}
	if req.Email != nil {
		updates["email"] = *req.Email
	}
	if req.Source != nil {
		updates["source"] = *req.Source
	}
	if req.Owner != nil {
		updates["owner"] = *req.Owner
	}
	if req.Comment != nil {
		updates["comment"] = *req.Comment
	}
	updates["updated_at"] = time.Now()

	if err := h.DB.Model(&lead).Updates(updates).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update: "+err.Error())
		return
	}
	// Перечитываем, чтобы вернуть актуальное состояние
	if err := h.DB.Where("lead_id = ?", leadID).First(&lead).Error; err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, lead)
}

// =====================================================================
// POST /leads/{id}/move — смена стадии
// =====================================================================
type moveRequest struct {
	Stage string `json:"stage"`
}

func (h *LeadHandler) MoveStage(w http.ResponseWriter, r *http.Request) {
	leadID := mux.Vars(r)["id"]

	var req moveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if req.Stage == "" {
		writeError(w, http.StatusBadRequest, "stage is required")
		return
	}

	// Проверяем, что целевая стадия вообще существует
	newOrder, ok := stageOrder[req.Stage]
	if !ok {
		writeError(w, http.StatusBadRequest, "unknown stage: "+req.Stage)
		return
	}

	// Достаём лида
	var lead models.Lead
	if err := h.DB.Where("lead_id = ?", leadID).First(&lead).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(w, http.StatusNotFound, "lead not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	curOrder, ok := stageOrder[lead.CurrentStage]
	if !ok {
		// На всякий случай: если в БД невалидная стадия — это баг данных
		writeError(w, http.StatusInternalServerError, "current_stage is invalid in DB: "+lead.CurrentStage)
		return
	}

	// Валидация перехода
	if lead.CurrentStage == "won" || lead.CurrentStage == "lost" {
		writeError(w, http.StatusConflict, "lead is closed")
		return
	}
	if req.Stage == lead.CurrentStage {
		writeError(w, http.StatusConflict, "already in this stage")
		return
	}
	if newOrder <= curOrder {
		writeError(w, http.StatusConflict, "backwards move forbidden")
		return
	}

	now := time.Now()

	// Транзакция:
	// 1) закрыть текущий открытый этап (left_at = NOW())
	// 2) создать новый этап (entered_at = NOW(), left_at = NULL)
	// 3) обновить current_stage и updated_at у самого лида
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.LeadStage{}).
			Where("lead_id = ? AND left_at IS NULL", leadID).
			Update("left_at", now).Error; err != nil {
			return err
		}

		newStage := models.LeadStage{
			LeadID:    leadID,
			Stage:     req.Stage,
			EnteredAt: now,
			LeftAt:    nil,
		}
		if err := tx.Create(&newStage).Error; err != nil {
			return err
		}

		return tx.Model(&models.Lead{}).
			Where("lead_id = ?", leadID).
			Updates(map[string]interface{}{
				"current_stage": req.Stage,
				"updated_at":    now,
			}).Error
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to move stage: "+err.Error())
		return
	}

	// Возвращаем актуального лида
	if err := h.DB.Where("lead_id = ?", leadID).First(&lead).Error; err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, lead)
}
