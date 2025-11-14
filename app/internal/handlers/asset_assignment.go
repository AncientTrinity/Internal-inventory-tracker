package handlers

import (
	"database/sql"
	"encoding/json"
	//"fmt"
	"net/http"
	"strconv"
	"strings"

	"victortillett.net/internal-inventory-tracker/internal/models"
)

type AssetAssignmentHandler struct {
	AssetsModel *models.AssetsModel
	UsersModel  *models.UsersModel
}

func NewAssetAssignmentHandler(db *sql.DB) *AssetAssignmentHandler {
	return &AssetAssignmentHandler{
		AssetsModel: models.NewAssetsModel(db),
		UsersModel:  models.NewUsersModel(db),
	}
}

// POST /api/v1/assets/{id}/assign
func (h *AssetAssignmentHandler) AssignAsset(w http.ResponseWriter, r *http.Request) {
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
	
	var input struct {
		UserID int64 `json:"user_id"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid input: "+err.Error(), http.StatusBadRequest)
		return
	}
	
	// Assign asset to user
	err = h.AssetsModel.AssignAsset(assetID, input.UserID)
	if err != nil {
		if err.Error() == "user not found" {
			http.Error(w, "User not found", http.StatusBadRequest)
			return
		}
		if err.Error() == "asset not found or cannot be assigned (might be retired or in repair)" {
			http.Error(w, "Asset not found or cannot be assigned", http.StatusBadRequest)
			return
		}
		http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	// Get updated asset to return
	asset, err := h.AssetsModel.GetByID(assetID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Asset assigned successfully",
		"asset":   asset,
	})
}

// POST /api/v1/assets/{id}/unassign
func (h *AssetAssignmentHandler) UnassignAsset(w http.ResponseWriter, r *http.Request) {
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
	
	// Unassign asset
	err = h.AssetsModel.UnassignAsset(assetID)
	if err != nil {
		if err.Error() == "asset not found" {
			http.Error(w, "Asset not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	// Get updated asset to return
	asset, err := h.AssetsModel.GetByID(assetID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Asset unassigned successfully",
		"asset":   asset,
	})
}

// GET /api/v1/users/{id}/assets
func (h *AssetAssignmentHandler) GetUserAssets(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from URL
	userIDStr := strings.TrimPrefix(r.URL.Path, "/api/v1/users/")
	userIDStr = strings.TrimSuffix(userIDStr, "/assets")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}
	
	// Verify user exists
	var userExists bool
	err = h.AssetsModel.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", userID).Scan(&userExists)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	if !userExists {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	
	// Get assets assigned to user
	assets, err := h.AssetsModel.GetAssetsByUser(userID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(assets)
}

// GET /api/v1/assets/available
func (h *AssetAssignmentHandler) GetAvailableAssets(w http.ResponseWriter, r *http.Request) {
	assetType := r.URL.Query().Get("type")
	
	assets, err := h.AssetsModel.GetAvailableAssets(assetType)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(assets)
}

// POST /api/v1/assets/bulk-assign
func (h *AssetAssignmentHandler) BulkAssignAssets(w http.ResponseWriter, r *http.Request) {
	var input struct {
		UserID   int64   `json:"user_id"`
		AssetIDs []int64 `json:"asset_ids"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid input: "+err.Error(), http.StatusBadRequest)
		return
	}
	
	// Verify user exists
	var userExists bool
	err := h.AssetsModel.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", input.UserID).Scan(&userExists)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	if !userExists {
		http.Error(w, "User not found", http.StatusBadRequest)
		return
	}
	
	// Define a type for failed assignments
	type failedAssignment struct {
		AssetID int64  `json:"asset_id"`
		Error   string `json:"error"`
	}
	
	results := struct {
		Success []int64           `json:"success"`
		Failed  []failedAssignment `json:"failed"`
	}{
		Success: []int64{},
		Failed:  []failedAssignment{},
	}
	
	// Assign each asset
	for _, assetID := range input.AssetIDs {
		err := h.AssetsModel.AssignAsset(assetID, input.UserID)
		if err != nil {
			results.Failed = append(results.Failed, failedAssignment{
				AssetID: assetID,
				Error:   err.Error(),
			})
		} else {
			results.Success = append(results.Success, assetID)
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}