// file: app/internal/handlers/assets_test.go
package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"victortillett.net/internal-inventory-tracker/internal/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupHandlerTest(t *testing.T) (*AssetsHandler, sqlmock.Sqlmock, func()) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	handler := NewAssetsHandler(db)
	
	teardown := func() {
		db.Close()
	}

	return handler, mock, teardown
}

func TestAssetsHandler_CreateAsset(t *testing.T) {
	handler, mock, teardown := setupHandlerTest(t)
	defer teardown()

	t.Run("successful creation", func(t *testing.T) {
		input := map[string]interface{}{
			"internal_id":    "DPA-PC001",
			"asset_type":     "PC",
			"manufacturer":   "Dell",
			"model":          "OptiPlex 7070",
			"model_number":   "OP7070",
			"serial_number":  "ABC123456",
			"status":         "IN_STORAGE",
			"date_purchased": "2024-01-15",
		}

		body, _ := json.Marshal(input)
		req := httptest.NewRequest("POST", "/api/v1/assets", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		// Mock the database insert
		now := time.Now()
		mock.ExpectQuery(`INSERT INTO assets`).
			WithArgs(
				"DPA-PC001", "PC", "Dell", "OptiPlex 7070", "OP7070", 
				"ABC123456", "IN_STORAGE", nil, sqlmock.AnyArg(), 
				nil, nil,
			).
			WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
				AddRow(1, now, now))

		handler.CreateAsset(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)
		
		var response models.Asset
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "DPA-PC001", response.InternalID)
		assert.Equal(t, "PC", response.AssetType)
	})

	t.Run("missing required fields", func(t *testing.T) {
		input := map[string]interface{}{
			"manufacturer": "Dell",
			"model":        "OptiPlex 7070",
			// missing internal_id and asset_type
		}

		body, _ := json.Marshal(input)
		req := httptest.NewRequest("POST", "/api/v1/assets", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		handler.CreateAsset(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "Internal ID and asset type are required")
	})

	t.Run("invalid date format", func(t *testing.T) {
		input := map[string]interface{}{
			"internal_id":    "DPA-PC001",
			"asset_type":     "PC",
			"date_purchased": "invalid-date",
		}

		body, _ := json.Marshal(input)
		req := httptest.NewRequest("POST", "/api/v1/assets", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		handler.CreateAsset(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "invalid date format")
	})
}

func TestAssetsHandler_GetAsset(t *testing.T) {
	handler, mock, teardown := setupHandlerTest(t)
	defer teardown()

	t.Run("successful retrieval", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/assets/1", nil)
		rr := httptest.NewRecorder()

		now := time.Now()
		purchaseDate := now.AddDate(0, -6, 0)
		
		mock.ExpectQuery(`SELECT`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "internal_id", "asset_type", "manufacturer", "model", 
				"model_number", "serial_number", "status", "in_use_by", 
				"date_purchased", "last_service_date", "next_service_date", 
				"created_at", "updated_at",
			}).AddRow(
				1, "DPA-PC001", "PC", "Dell", "OptiPlex 7070", 
				"OP7070", "ABC123456", "IN_USE", int64(2),
				purchaseDate, nil, nil, now, now,
			))

		handler.GetAsset(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		
		var response models.Asset
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), response.ID)
		assert.Equal(t, "DPA-PC001", response.InternalID)
	})

	t.Run("asset not found", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/assets/999", nil)
		rr := httptest.NewRecorder()

		// Mock returns no rows - this should trigger the "asset not found" error
		mock.ExpectQuery(`SELECT`).
			WithArgs(int64(999)).
			WillReturnError(sql.ErrNoRows)

		handler.GetAsset(rr, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
		assert.Contains(t, rr.Body.String(), "Asset not found")
	})

	t.Run("invalid asset ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/assets/invalid", nil)
		rr := httptest.NewRecorder()

		handler.GetAsset(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "Invalid ID")
	})

	t.Run("database error", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/assets/1", nil)
		rr := httptest.NewRecorder()

		// Mock returns a database error (not a "not found" error)
		mock.ExpectQuery(`SELECT`).
			WithArgs(int64(1)).
			WillReturnError(assert.AnError)

		handler.GetAsset(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Contains(t, rr.Body.String(), "Database error")
	})
}

func TestAssetsHandler_ListAssets(t *testing.T) {
	handler, mock, teardown := setupHandlerTest(t)
	defer teardown()

	now := time.Now()

	t.Run("list all assets", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/assets", nil)
		rr := httptest.NewRecorder()

		mock.ExpectQuery(`SELECT`).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "internal_id", "asset_type", "manufacturer", "model", 
				"model_number", "serial_number", "status", "in_use_by", 
				"date_purchased", "last_service_date", "next_service_date", 
				"created_at", "updated_at",
			}).AddRow(
				1, "DPA-PC001", "PC", "Dell", "OptiPlex 7070", 
				"OP7070", "ABC123456", "IN_USE", int64(2),
				now, nil, nil, now, now,
			).AddRow(
				2, "AM-M001", "Monitor", "Viewsonic", "VX3276", 
				"VX3276", "DEF789012", "IN_STORAGE", nil,
				now, nil, nil, now, now,
			))

		handler.ListAssets(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		
		var response []models.Asset
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Len(t, response, 2)
	})

	t.Run("list assets with filters", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/assets?type=PC&status=IN_USE&in_use_by=2", nil)
		rr := httptest.NewRecorder()

		mock.ExpectQuery(`SELECT.*asset_type = \$1.*status = \$2.*in_use_by = \$3`).
			WithArgs("PC", "IN_USE", int64(2)).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "internal_id", "asset_type", "manufacturer", "model", 
				"model_number", "serial_number", "status", "in_use_by", 
				"date_purchased", "last_service_date", "next_service_date", 
				"created_at", "updated_at",
			}).AddRow(
				1, "DPA-PC001", "PC", "Dell", "OptiPlex 7070", 
				"OP7070", "ABC123456", "IN_USE", int64(2),
				now, nil, nil, now, now,
			))

		handler.ListAssets(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		
		var response []models.Asset
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Len(t, response, 1)
		assert.Equal(t, "PC", response[0].AssetType)
	})
}