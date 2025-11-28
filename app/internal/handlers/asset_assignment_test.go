// file: app/internal/handlers/asset_assignment_test.go
package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupAssignmentTest(t *testing.T) (*AssetAssignmentHandler, sqlmock.Sqlmock, func()) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	handler := NewAssetAssignmentHandler(db)
	
	teardown := func() {
		db.Close()
	}

	return handler, mock, teardown
}

func TestAssetAssignmentHandler_AssignAsset(t *testing.T) {
	handler, mock, teardown := setupAssignmentTest(t)
	defer teardown()

	t.Run("successful assignment", func(t *testing.T) {
		input := map[string]interface{}{
			"user_id": 2,
		}

		body, _ := json.Marshal(input)
		req := httptest.NewRequest("POST", "/api/v1/assets/1/assign", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		// Mock user exists check
		mock.ExpectQuery(`SELECT EXISTS`).
			WithArgs(int64(2)).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		// Mock asset assignment
		now := time.Now()
		mock.ExpectQuery(`UPDATE assets`).
			WithArgs(int64(2), int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"updated_at"}).AddRow(now))

		// Mock getting updated asset
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
				now, nil, nil, now, now,
			))

		handler.AssignAsset(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Asset assigned successfully", response["message"])
	})

	t.Run("user not found", func(t *testing.T) {
		input := map[string]interface{}{
			"user_id": 999,
		}

		body, _ := json.Marshal(input)
		req := httptest.NewRequest("POST", "/api/v1/assets/1/assign", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		mock.ExpectQuery(`SELECT EXISTS`).
			WithArgs(int64(999)).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		handler.AssignAsset(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "User not found")
	})

	t.Run("asset not found or cannot be assigned", func(t *testing.T) {
		input := map[string]interface{}{
			"user_id": 2,
		}

		body, _ := json.Marshal(input)
		req := httptest.NewRequest("POST", "/api/v1/assets/999/assign", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		mock.ExpectQuery(`SELECT EXISTS`).
			WithArgs(int64(2)).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		// Mock returns no rows for the asset assignment
		mock.ExpectQuery(`UPDATE assets`).
			WithArgs(int64(2), int64(999)).
			WillReturnError(sql.ErrNoRows)

		handler.AssignAsset(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "Asset not found or cannot be assigned")
	})
}