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
	"victortillett.net/internal-inventory-tracker/internal/services"
)

type AssetsHandler struct {
	Model *models.AssetsModel
	NotificationService *services.NotificationService
}

func NewAssetsHandler(db *sql.DB) *AssetsHandler {
	return &AssetsHandler{
		Model: models.NewAssetsModel(db),
		NotificationService: services.NewNotificationService(db),
	}
}

// GET /api/v1/assets
func (h *AssetsHandler) ListAssets(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters for filtering
	assetType := r.URL.Query().Get("type")
	status := r.URL.Query().Get("status")
	inUseByStr := r.URL.Query().Get("in_use_by")
	
	var filters []models.AssetFilter
	
	if assetType != "" {
		filters = append(filters, models.AssetFilter{Type: assetType})
	}
	if status != "" {
		filters = append(filters, models.AssetFilter{Status: status})
	}
	if inUseByStr != "" {
		if inUseBy, err := strconv.ParseInt(inUseByStr, 10, 64); err == nil {
			filters = append(filters, models.AssetFilter{InUseBy: &inUseBy})
		}
	}
	
	assets, err := h.Model.GetAll(filters...)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(assets)
}

// GET /api/v1/assets/{id}
func (h *AssetsHandler) GetAsset(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/assets/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	
	asset, err := h.Model.GetByID(id)
	if err != nil {
		if err.Error() == "asset not found" {
			http.Error(w, "Asset not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(asset)
}

// POST /api/v1/assets
// POST /api/v1/assets
func (h *AssetsHandler) CreateAsset(w http.ResponseWriter, r *http.Request) {
	var input struct {
		InternalID      string  `json:"internal_id"`
		AssetType       string  `json:"asset_type"`
		Manufacturer    string  `json:"manufacturer"`
		Model           string  `json:"model"`
		ModelNumber     string  `json:"model_number"`
		SerialNumber    string  `json:"serial_number"`
		Status          string  `json:"status"`
		InUseBy         *int64  `json:"in_use_by"`
		DatePurchased   string  `json:"date_purchased"`    // Change to string
		LastServiceDate string  `json:"last_service_date"` // Change to string
		NextServiceDate string  `json:"next_service_date"` // Change to string
	}
	
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid input: "+err.Error(), http.StatusBadRequest)
		return
	}
	
	// Validate required fields
	if input.InternalID == "" || input.AssetType == "" {
		http.Error(w, "Internal ID and asset type are required", http.StatusBadRequest)
		return
	}
	
	// Parse dates from string
	parseDate := func(dateStr string) (*time.Time, error) {
		if dateStr == "" {
			return nil, nil
		}
		// Try multiple date formats
		formats := []string{"2006-01-02", "2006-01-02T15:04:05Z", time.RFC3339}
		for _, format := range formats {
			if t, err := time.Parse(format, dateStr); err == nil {
				return &t, nil
			}
		}
		return nil, fmt.Errorf("invalid date format: %s, expected YYYY-MM-DD", dateStr)
	}
	
	datePurchased, err := parseDate(input.DatePurchased)
	if err != nil {
		http.Error(w, "DatePurchased: "+err.Error(), http.StatusBadRequest)
		return
	}
	
	lastServiceDate, err := parseDate(input.LastServiceDate)
	if err != nil {
		http.Error(w, "LastServiceDate: "+err.Error(), http.StatusBadRequest)
		return
	}
	
	nextServiceDate, err := parseDate(input.NextServiceDate)
	if err != nil {
		http.Error(w, "NextServiceDate: "+err.Error(), http.StatusBadRequest)
		return
	}
	
	asset := &models.Asset{
		InternalID:      input.InternalID,
		AssetType:       input.AssetType,
		Manufacturer:    input.Manufacturer,
		Model:           input.Model,
		ModelNumber:     input.ModelNumber,
		SerialNumber:    input.SerialNumber,
		Status:          input.Status,
		InUseBy:         input.InUseBy,
		DatePurchased:   datePurchased,
		LastServiceDate: lastServiceDate,
		NextServiceDate: nextServiceDate,
	}
	
	// Set default status if not provided
	if asset.Status == "" {
		asset.Status = "IN_STORAGE"
	}
	
	err = h.Model.Insert(asset)
	if err != nil {
		// Check for duplicate internal_id
		if strings.Contains(err.Error(), "duplicate key") {
			http.Error(w, "Asset with this internal ID already exists", http.StatusBadRequest)
			return
		}

		go func() {
		if err := h.NotificationService.NotifyAssetCreated(asset); err != nil {
			fmt.Printf("Failed to send asset notifications: %v\n", err)
		}
	}()

		http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(asset)
}

// PUT /api/v1/assets/{id}
// PUT /api/v1/assets/{id}
func (h *AssetsHandler) UpdateAsset(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/assets/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	
	var input struct {
		InternalID      string  `json:"internal_id"`
		AssetType       string  `json:"asset_type"`
		Manufacturer    string  `json:"manufacturer"`
		Model           string  `json:"model"`
		ModelNumber     string  `json:"model_number"`
		SerialNumber    string  `json:"serial_number"`
		Status          string  `json:"status"`
		InUseBy         *int64  `json:"in_use_by"`
		DatePurchased   string  `json:"date_purchased"`
		LastServiceDate string  `json:"last_service_date"`
		NextServiceDate string  `json:"next_service_date"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid input: "+err.Error(), http.StatusBadRequest)
		return
	}
	
	// Get existing asset to preserve fields not being updated
	existingAsset, err := h.Model.GetByID(id)
	if err != nil {
		if err.Error() == "asset not found" {
			http.Error(w, "Asset not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	
	// Parse dates from string (same function as above)
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
	
	// Update fields (only if provided in input)
	if input.InternalID != "" {
		existingAsset.InternalID = input.InternalID
	}
	if input.AssetType != "" {
		existingAsset.AssetType = input.AssetType
	}
	if input.Manufacturer != "" {
		existingAsset.Manufacturer = input.Manufacturer
	}
	if input.Model != "" {
		existingAsset.Model = input.Model
	}
	if input.ModelNumber != "" {
		existingAsset.ModelNumber = input.ModelNumber
	}
	if input.SerialNumber != "" {
		existingAsset.SerialNumber = input.SerialNumber
	}
	if input.Status != "" {
		existingAsset.Status = input.Status
	}
	if input.InUseBy != nil {
		existingAsset.InUseBy = input.InUseBy
	}
	
	// Handle date updates
	if input.DatePurchased != "" {
		datePurchased, err := parseDate(input.DatePurchased)
		if err != nil {
			http.Error(w, "DatePurchased: "+err.Error(), http.StatusBadRequest)
			return
		}
		existingAsset.DatePurchased = datePurchased
	}
	
	if input.LastServiceDate != "" {
		lastServiceDate, err := parseDate(input.LastServiceDate)
		if err != nil {
			http.Error(w, "LastServiceDate: "+err.Error(), http.StatusBadRequest)
			return
		}
		existingAsset.LastServiceDate = lastServiceDate
	}
	
	if input.NextServiceDate != "" {
		nextServiceDate, err := parseDate(input.NextServiceDate)
		if err != nil {
			http.Error(w, "NextServiceDate: "+err.Error(), http.StatusBadRequest)
			return
		}
		existingAsset.NextServiceDate = nextServiceDate
	}
	
	err = h.Model.Update(existingAsset)
	if err != nil {
		http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(existingAsset)
}

// DELETE /api/v1/assets/{id}
func (h *AssetsHandler) DeleteAsset(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/assets/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	
	err = h.Model.Delete(id)
	if err != nil {
		if err.Error() == "asset not found" {
			http.Error(w, "Asset not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	
	w.WriteHeader(http.StatusNoContent)
}