package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"victortillett.net/internal-inventory-tracker/internal/models"
)

type AssetSearchHandler struct {
	AssetsModel *models.AssetsModel
}

func NewAssetSearchHandler(db *sql.DB) *AssetSearchHandler {
	return &AssetSearchHandler{
		AssetsModel: models.NewAssetsModel(db),
	}
}

// GET /api/v1/assets/search
func (h *AssetSearchHandler) SearchAssets(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	assetType := r.URL.Query().Get("type")
	status := r.URL.Query().Get("status")
	manufacturer := r.URL.Query().Get("manufacturer")
	inUseByStr := r.URL.Query().Get("in_use_by")
	
	// Parse date filters
	purchasedAfterStr := r.URL.Query().Get("purchased_after")
	purchasedBeforeStr := r.URL.Query().Get("purchased_before")
	
	// Parse service filters
	needsServiceStr := r.URL.Query().Get("needs_service")
	overdueServiceStr := r.URL.Query().Get("overdue_service")
	
	// Parse pagination and sorting
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	sortBy := r.URL.Query().Get("sort_by")
	sortOrder := r.URL.Query().Get("sort_order")
	
	filters := models.AssetSearchFilters{
		Query:        query,
		AssetType:    assetType,
		Status:       status,
		Manufacturer: manufacturer,
		SortBy:       sortBy,
		SortOrder:    sortOrder,
	}
	
	// Parse in_use_by
	if inUseByStr != "" {
		if inUseBy, err := strconv.ParseInt(inUseByStr, 10, 64); err == nil {
			filters.InUseBy = &inUseBy
		}
	}
	
	// Parse date filters
	if purchasedAfterStr != "" {
		if date, err := time.Parse("2006-01-02", purchasedAfterStr); err == nil {
			filters.PurchasedAfter = date
		}
	}
	
	if purchasedBeforeStr != "" {
		if date, err := time.Parse("2006-01-02", purchasedBeforeStr); err == nil {
			filters.PurchasedBefore = date
		}
	}
	
	// Parse boolean filters
	if needsServiceStr == "true" {
		filters.NeedsService = true
	}
	
	if overdueServiceStr == "true" {
		filters.OverdueService = true
	}
	
	// Parse pagination
	if limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			filters.Limit = limit
		}
	}
	
	if offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			filters.Offset = offset
		}
	}
	
	assets, err := h.AssetsModel.SearchAssets(filters.Query, filters)
	if err != nil {
		http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	// Add pagination info to response
	response := map[string]interface{}{
		"assets": assets,
		"filters": map[string]interface{}{
			"query":        query,
			"asset_type":   assetType,
			"status":       status,
			"manufacturer": manufacturer,
			"in_use_by":    inUseByStr,
			"limit":        filters.Limit,
			"offset":       filters.Offset,
			"sort_by":      sortBy,
			"sort_order":   sortOrder,
		},
		"total": len(assets),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GET /api/v1/assets/stats
func (h *AssetSearchHandler) GetAssetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.AssetsModel.GetAssetStats()
	if err != nil {
		http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// GET /api/v1/assets/types
func (h *AssetSearchHandler) GetAssetTypes(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT DISTINCT asset_type 
		FROM assets 
		ORDER BY asset_type
	`
	
	rows, err := h.AssetsModel.DB.Query(query)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	var assetTypes []string
	for rows.Next() {
		var assetType string
		if err := rows.Scan(&assetType); err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		assetTypes = append(assetTypes, assetType)
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(assetTypes)
}

// GET /api/v1/assets/manufacturers
func (h *AssetSearchHandler) GetManufacturers(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT DISTINCT manufacturer 
		FROM assets 
		WHERE manufacturer IS NOT NULL AND manufacturer != ''
		ORDER BY manufacturer
	`
	
	rows, err := h.AssetsModel.DB.Query(query)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	var manufacturers []string
	for rows.Next() {
		var manufacturer string
		if err := rows.Scan(&manufacturer); err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		manufacturers = append(manufacturers, manufacturer)
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(manufacturers)
}