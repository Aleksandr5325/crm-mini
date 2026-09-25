package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/Aleksandr5325/crm-mini/internal/models"

	"gorm.io/gorm"
)

type SourceHandler struct {
	DB *gorm.DB
}

func (h *SourceHandler) GetSources(w http.ResponseWriter, r *http.Request) {
	var sources []models.Source
	if err := h.DB.Find(&sources).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sources)
}
