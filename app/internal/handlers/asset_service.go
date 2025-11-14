package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
	"fmt"

	"victortillett.net/internal-inventory-tracker/internal/models"
)

type AssetServiceHandler struct {
	ServiceModel *models.AssetServiceModel
	AssetsModel  *models.AssetsModel
}

func NewAssetServiceHandler(db *sql.DB) *AssetServiceHandler {
	return &AssetServiceHandler{
		ServiceModel: models.NewAssetServiceModel(db),
		AssetsModel:  models.NewAssetsModel(db),
	}
}

// POST /api/v1/assets/{id}/service-logs
func (h *AssetServiceHandler) CreateServiceLog(w http.ResponseWriter, r *http.Request) {
	// Extract asset ID from URL
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/assets/")
	pathParts := strings.Split(path, "/")
	if len(pathParts) < 2 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	
	assetID, err := strconv.ParseInt(pathParts[0], 10, 64)
	if err != nil {
		http.Error(w, "Invalid asset ID", http.StatusBadRequest)
		return
	}
	
	// Verify asset exists
	_, err = h.AssetsModel.GetByID(assetID)
	if err != nil {
		if err.Error() == "asset not found" {
			http.Error(w, "Asset not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	
	var input struct {
		PerformedBy      *int64 `json:"performed_by"` // User ID (can be null)
		PerformedAt      string `json:"performed_at"` // Service date
		ServiceType      string `json:"service_type"` // MAINTENANCE, REPAIR, UPGRADE
		NextServiceDate  string `json:"next_service_date"`
		Notes            string `json:"notes"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid input: "+err.Error(), http.StatusBadRequest)
		return
	}
	
	// Validate required fields
	if input.ServiceType == "" {
		http.Error(w, "Service type is required", http.StatusBadRequest)
		return
	}
	
	// Parse dates
	parseDate := func(dateStr string) (*time.Time, error) {
		if dateStr == "" {
			return nil, nil
		}
		formats := []string{"2006-01-02", "2006-01-02T15:04:05Z", time.RFC3339}
		for _, format := range formats {
			if t, err := time.Parse(format, dateStr); err == nil {
				return &t, nil
			}
		}
		return nil, fmt.Errorf("invalid date format: %s, expected YYYY-MM-DD", dateStr)
	}
	
	performedAt := time.Now() // Default to now
	if input.PerformedAt != "" {
		parsedDate, err := parseDate(input.PerformedAt)
		if err != nil {
			http.Error(w, "PerformedAt: "+err.Error(), http.StatusBadRequest)
			return
		}
		if parsedDate != nil {
			performedAt = *parsedDate
		}
	}
	
	nextServiceDate, err := parseDate(input.NextServiceDate)
	if err != nil {
		http.Error(w, "NextServiceDate: "+err.Error(), http.StatusBadRequest)
		return
	}
	
	// Create service log
	serviceLog := &models.AssetServiceLog{
		AssetID:         assetID,
		PerformedBy:     input.PerformedBy,
		PerformedAt:     performedAt,
		ServiceType:     input.ServiceType,
		NextServiceDate: nextServiceDate,
		Notes:           input.Notes,
	}
	
	err = h.ServiceModel.Insert(serviceLog)
	if err != nil {
		http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	// Update asset's service dates
	err = h.ServiceModel.UpdateAssetServiceDate(assetID, performedAt, nextServiceDate)
	if err != nil {
		// Log error but don't fail the request
		fmt.Printf("Warning: Failed to update asset service dates: %v\n", err)
	}
	
	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(serviceLog)
}

// GET /api/v1/assets/{id}/service-logs
func (h *AssetServiceHandler) GetServiceLogs(w http.ResponseWriter, r *http.Request) {
	// Extract asset ID from URL
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/assets/")
	pathParts := strings.Split(path, "/")
	if len(pathParts) < 2 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	
	assetID, err := strconv.ParseInt(pathParts[0], 10, 64)
	if err != nil {
		http.Error(w, "Invalid asset ID", http.StatusBadRequest)
		return
	}
	
	// Verify asset exists
	_, err = h.AssetsModel.GetByID(assetID)
	if err != nil {
		if err.Error() == "asset not found" {
			http.Error(w, "Asset not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	
	logs, err := h.ServiceModel.GetByAssetID(assetID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

// GET /api/v1/service-logs/{id}
func (h *AssetServiceHandler) GetServiceLog(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/service-logs/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid service log ID", http.StatusBadRequest)
		return
	}
	
	log, err := h.ServiceModel.GetByID(id)
	if err != nil {
		if err.Error() == "service log not found" {
			http.Error(w, "Service log not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(log)
}