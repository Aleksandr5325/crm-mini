package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/Aleksandr5325/crm-mini/internal/models"

	"gorm.io/gorm"
)

type ImportHandler struct {
	DB *gorm.DB
}

// Структуры для импорта
type ImportRequest struct {
	ImportID string          `json:"import_id"`
	Sources  []models.Source `json:"sources"`
	Leads    []LeadImport    `json:"leads"`
}

type LeadImport struct {
	LeadID       string        `json:"lead_id"`
	CreatedAt    time.Time     `json:"created_at"`
	Source       string        `json:"source"`
	CurrentStage string        `json:"current_stage"`
	Owner        string        `json:"manager"`
	FullName     string        `json:"full_name"`
	Phone        string        `json:"phone"`
	Email        string        `json:"email"`
	Comment      string        `json:"comment"`
	Stages       []StageImport `json:"stages"`
}

// Структура одного источника из CSV
type SourceImport struct {
	CodeName string `json:"code_name"`
	Name     string `json:"name"`
}

// Структура всего запроса (1С присылает объект {"sources": [...]})
type SourceImportRequest struct {
	Sources []SourceImport `json:"sources"`
}

type StageImport struct {
	Stage     string     `json:"stage"`
	EnteredAt time.Time  `json:"entered_at"`
	LeftAt    *time.Time `json:"left_at"`
}

func (h *ImportHandler) ImportLeads(w http.ResponseWriter, r *http.Request) {
	// 1. Декодируем JSON
	var req ImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	// 2. Вычисляем хеш для идемпотентности
	//    Сериализуем структуру в канонический JSON (без лишних пробелов)
	bodyBytes, err := json.Marshal(req)
	if err != nil {
		http.Error(w, "Failed to serialize request", http.StatusInternalServerError)
		return
	}
	hash := sha256.Sum256(bodyBytes)
	hashStr := hex.EncodeToString(hash[:])

	// 3. Проверяем, не было ли уже такого импорта
	var existingLog models.ImportLog
	err = h.DB.Where("file_hash = ?", hashStr).First(&existingLog).Error
	if err == nil {
		// Уже импортировано
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"duplicate":   true,
			"imported_at": existingLog.ImportedAt,
			"rows":        existingLog.RowsImported,
		})
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// 4. Начинаем транзакцию
	tx := h.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 5. Импорт источников (только новых)
	for _, src := range req.Sources {
		var existing models.Source
		err := tx.Where("code = ?", src.Code).First(&existing).Error
		if err == nil {
			continue // уже есть, пропускаем
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			tx.Rollback()
			http.Error(w, "DB error checking source", http.StatusInternalServerError)
			return
		}
		if err := tx.Create(&src).Error; err != nil {
			tx.Rollback()
			http.Error(w, "Failed to create source: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// 6. Импорт лидов и этапов
	rowsImported := 0
	for _, leadImp := range req.Leads {
		// Проверяем существование источника (лид ссылается на источник по коду)
		var source models.Source
		err := tx.Where("code = ?", leadImp.Source).First(&source).Error
		if err != nil {
			tx.Rollback()
			http.Error(w, "Source not found: "+leadImp.Source, http.StatusBadRequest)
			return
		}

		// Создаём или обновляем лида
		var lead models.Lead
		err = tx.Where("lead_id = ?", leadImp.LeadID).First(&lead).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				lead = models.Lead{
					LeadID:       leadImp.LeadID,
					CreatedAt:    leadImp.CreatedAt,
					Source:       leadImp.Source,
					CurrentStage: leadImp.CurrentStage,
					Owner:        leadImp.Owner,
					FullName:     leadImp.FullName,
					Phone:        leadImp.Phone,
					Email:        leadImp.Email,
					Comment:      leadImp.Comment,
				}
				if err := tx.Create(&lead).Error; err != nil {
					tx.Rollback()
					http.Error(w, "Failed to create lead: "+err.Error(), http.StatusInternalServerError)
					return
				}
			} else {
				tx.Rollback()
				http.Error(w, "DB error", http.StatusInternalServerError)
				return
			}
		} else {
			// Обновляем только если изменились важные поля
			if lead.CurrentStage != leadImp.CurrentStage || lead.Owner != leadImp.Owner {
				lead.CurrentStage = leadImp.CurrentStage
				lead.Owner = leadImp.Owner
				if err := tx.Save(&lead).Error; err != nil {
					tx.Rollback()
					http.Error(w, "Failed to update lead: "+err.Error(), http.StatusInternalServerError)
					return
				}
			}
		}

		// Валидация порядка стадий и дат
		if err := validateStages(leadImp.Stages); err != nil {
			tx.Rollback()
			http.Error(w, "Invalid stages for lead "+leadImp.LeadID+": "+err.Error(), http.StatusBadRequest)
			return
		}

		// Добавляем этапы
		for _, st := range leadImp.Stages {
			// Проверка дублирования этапа (по lead_id, stage, entered_at)
			var existingStage models.LeadStage
			err := tx.Where("lead_id = ? AND stage = ? AND entered_at = ?", leadImp.LeadID, st.Stage, st.EnteredAt).First(&existingStage).Error
			if err == nil {
				// уже есть, пропускаем
				continue
			} else if !errors.Is(err, gorm.ErrRecordNotFound) {
				tx.Rollback()
				http.Error(w, "DB error checking stage", http.StatusInternalServerError)
				return
			}

			stage := models.LeadStage{
				LeadID:    leadImp.LeadID,
				Stage:     st.Stage,
				EnteredAt: st.EnteredAt,
				LeftAt:    st.LeftAt,
			}
			if err := tx.Create(&stage).Error; err != nil {
				tx.Rollback()
				http.Error(w, "Failed to create stage: "+err.Error(), http.StatusInternalServerError)
				return
			}
			rowsImported++
		}
	}

	// 7. Фиксируем транзакцию
	if err := tx.Commit().Error; err != nil {
		http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
		return
	}

	// 8. Логируем импорт
	importLog := models.ImportLog{
		FileHash:     hashStr,
		FileName:     "json_import", // можно добавить имя из запроса, если передаётся
		RowsImported: rowsImported,
		ImportedAt:   time.Now(),
	}
	if err := h.DB.Create(&importLog).Error; err != nil {
		log.Printf("Failed to save import log: %v", err)
	}

	// 9. Успешный ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"import_id":     req.ImportID,
		"rows_imported": rowsImported,
	})
}

// validateStages проверяет допустимость статусов и корректность дат
func validateStages(stages []StageImport) error {
	allowed := map[string]bool{
		"new": true, "qualified": true, "proposal": true, "won": true, "lost": true,
	}
	for _, st := range stages {
		if !allowed[st.Stage] {
			return errors.New("invalid stage: " + st.Stage)
		}
		if st.LeftAt != nil && st.EnteredAt.After(*st.LeftAt) {
			return errors.New("entered_at > left_at for stage " + st.Stage)
		}
	}
	// Дополнительно: можно проверить порядок стадий (новый -> квалифицирован -> ...)
	// для простоты оставим так
	return nil
}
func (h *ImportHandler) ImportSources(w http.ResponseWriter, r *http.Request) {
	// 1. Декодируем JSON, который прислала 1С
	var req SourceImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	if len(req.Sources) == 0 {
		http.Error(w, "Массив sources пуст", http.StatusBadRequest)
		return
	}

	// 2. Начинаем транзакцию в БД
	tx := h.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 3. Проверяем и сохраняем источники
	importedCount := 0
	for _, s := range req.Sources {
		var existing models.Source
		// Судя по твоему коду (строка 76), в БД поле называется "code"
		err := tx.Where("code = ?", s.CodeName).First(&existing).Error

		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// Если такого источника еще нет — создаем
				newSource := models.Source{
					Code: s.CodeName,
					Name: s.Name,
				}
				if err := tx.Create(&newSource).Error; err != nil {
					tx.Rollback()
					http.Error(w, "Failed to create source: "+err.Error(), http.StatusInternalServerError)
					return
				}
				importedCount++
			} else {
				// Если произошла другая ошибка базы
				tx.Rollback()
				http.Error(w, "DB error", http.StatusInternalServerError)
				return
			}
		}
		// Если ошибка nil, значит источник уже есть — просто пропускаем
	}

	// 4. Фиксируем транзакцию
	if err := tx.Commit().Error; err != nil {
		http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
		return
	}

	// 5. Успешный ответ для 1С
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":       true,
		"rows_imported": importedCount,
	})
}

// Структура одного сотрудника из JSON от 1С
type EmployeeImport struct {
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

// Структура всего запроса импорта сотрудников
type EmployeeImportRequest struct {
	Employees []EmployeeImport `json:"employees"`
}

// ImportEmployees обрабатывает JSON-пакет сотрудников от 1С и сохраняет в PostgreSQL
func (h *ImportHandler) ImportEmployees(w http.ResponseWriter, r *http.Request) {
	var req EmployeeImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Некорректный JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	if len(req.Employees) == 0 {
		http.Error(w, "Массив employees пуст", http.StatusBadRequest)
		return
	}

	// Начинаем транзакцию в БД
	tx := h.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	importedCount := 0
	for _, emp := range req.Employees {
		if emp.Email == "" {
			continue // Email обязателен для идентификации
		}

		var existing models.Employee // Убедитесь, что структура Employee объявлена в models
		err := tx.Where("email = ?", emp.Email).First(&existing).Error

		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// Если сотрудника нет в БД — создаём нового
				newEmp := models.Employee{
					FullName: emp.FullName,
					Email:    emp.Email,
					Role:     emp.Role,
				}
				if err := tx.Create(&newEmp).Error; err != nil {
					tx.Rollback()
					http.Error(w, "Не удалось создать сотрудника: "+err.Error(), http.StatusInternalServerError)
					return
				}
				importedCount++
			} else {
				tx.Rollback()
				http.Error(w, "Ошибка базы данных при поиске", http.StatusInternalServerError)
				return
			}
		} else {
			// Если сотрудник найден — обновляем его ФИО и Роль актуальными данными из 1С
			existing.FullName = emp.FullName
			existing.Role = emp.Role
			if err := tx.Save(&existing).Error; err != nil {
				tx.Rollback()
				http.Error(w, "Не удалось обновить сотрудника: "+err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}

	// Фиксируем транзакцию
	if err := tx.Commit().Error; err != nil {
		http.Error(w, "Не удалось зафиксировать транзакцию: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Возвращаем успешный статус
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "success",
		"imported": importedCount,
		"total":    len(req.Employees),
	})
}

// Структура одного контакта из JSON от 1С
type ContactImport struct {
	LeadID   string `json:"lead_id"`
	FullName string `json:"full_name"`
	Email    string `json:"email"`
}

// Структура всего запроса
type ContactImportRequest struct {
	Contacts []ContactImport `json:"contacts"`
}

// ImportContacts обновляет ФИО и Email у существующих лидов
func (h *ImportHandler) ImportContacts(w http.ResponseWriter, r *http.Request) {
	var req ContactImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Некорректный JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	if len(req.Contacts) == 0 {
		http.Error(w, "Массив contacts пуст", http.StatusBadRequest)
		return
	}

	tx := h.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	updatedCount := 0
	for _, c := range req.Contacts {
		var lead models.Lead
		// Ищем лида по lead_id
		err := tx.Where("lead_id = ?", c.LeadID).First(&lead).Error

		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// Если лид не найден, просто пропускаем его (не к кому привязывать контакт)
				continue
			} else {
				tx.Rollback()
				http.Error(w, "Ошибка базы данных", http.StatusInternalServerError)
				return
			}
		}

		// Если лид найден, обновляем его контакты
		lead.FullName = c.FullName
		lead.Email = c.Email

		if err := tx.Save(&lead).Error; err != nil {
			tx.Rollback()
			http.Error(w, "Не удалось обновить контакт: "+err.Error(), http.StatusInternalServerError)
			return
		}
		updatedCount++
	}

	// Фиксируем транзакцию
	if err := tx.Commit().Error; err != nil {
		http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
		return
	}

	// Возвращаем успешный статус 1С
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":      true,
		"rows_updated": updatedCount,
	})
}
