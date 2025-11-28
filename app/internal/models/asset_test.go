// file: app/internal/models/assets_test.go
package models

import (
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupAssetTest(t *testing.T) (*AssetsModel, sqlmock.Sqlmock, func()) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	model := NewAssetsModel(db)
	
	teardown := func() {
		db.Close()
	}

	return model, mock, teardown
}

func TestAssetsModel_Insert(t *testing.T) {
	model, mock, teardown := setupAssetTest(t)
	defer teardown()

	now := time.Now()
	asset := &Asset{
		InternalID:   "DPA-PC001",
		AssetType:    "PC",
		Manufacturer: "Dell",
		Model:        "OptiPlex 7070",
		ModelNumber:  "OP7070",
		SerialNumber: "ABC123456",
		Status:       "IN_STORAGE",
		DatePurchased: &now,
	}

	t.Run("successful insert", func(t *testing.T) {
		mock.ExpectQuery(`INSERT INTO assets`).
			WithArgs(
				asset.InternalID,
				asset.AssetType,
				asset.Manufacturer,
				asset.Model,
				asset.ModelNumber,
				asset.SerialNumber,
				asset.Status,
				asset.InUseBy,
				asset.DatePurchased,
				asset.LastServiceDate,
				asset.NextServiceDate,
			).
			WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
				AddRow(1, now, now))

		err := model.Insert(asset)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), asset.ID)
		assert.Equal(t, now, asset.CreatedAt)
		assert.Equal(t, now, asset.UpdatedAt)
	})

	t.Run("database error", func(t *testing.T) {
		mock.ExpectQuery(`INSERT INTO assets`).
			WillReturnError(assert.AnError)

		err := model.Insert(asset)
		assert.Error(t, err)
	})
}

func TestAssetsModel_GetByID(t *testing.T) {
	model, mock, teardown := setupAssetTest(t)
	defer teardown()

	now := time.Now()
	purchaseDate := now.AddDate(0, -6, 0)

	t.Run("successful get", func(t *testing.T) {
		expectedAsset := &Asset{
			ID:            1,
			InternalID:    "DPA-PC001",
			AssetType:     "PC",
			Manufacturer:  "Dell",
			Model:         "OptiPlex 7070",
			ModelNumber:   "OP7070",
			SerialNumber:  "ABC123456",
			Status:        "IN_USE",
			InUseBy:       int64Ptr(2),
			DatePurchased: &purchaseDate,
			CreatedAt:     now,
			UpdatedAt:     now,
		}

		mock.ExpectQuery(`SELECT`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "internal_id", "asset_type", "manufacturer", "model", 
				"model_number", "serial_number", "status", "in_use_by", 
				"date_purchased", "last_service_date", "next_service_date", 
				"created_at", "updated_at",
			}).AddRow(
				expectedAsset.ID,
				expectedAsset.InternalID,
				expectedAsset.AssetType,
				expectedAsset.Manufacturer,
				expectedAsset.Model,
				expectedAsset.ModelNumber,
				expectedAsset.SerialNumber,
				expectedAsset.Status,
				expectedAsset.InUseBy,
				expectedAsset.DatePurchased,
				expectedAsset.LastServiceDate,
				expectedAsset.NextServiceDate,
				expectedAsset.CreatedAt,
				expectedAsset.UpdatedAt,
			))

		asset, err := model.GetByID(1)
		assert.NoError(t, err)
		assert.Equal(t, expectedAsset.ID, asset.ID)
		assert.Equal(t, expectedAsset.InternalID, asset.InternalID)
		assert.Equal(t, expectedAsset.AssetType, asset.AssetType)
	})

	t.Run("asset not found", func(t *testing.T) {
		mock.ExpectQuery(`SELECT`).
			WithArgs(int64(999)).
			WillReturnError(sql.ErrNoRows)

		asset, err := model.GetByID(999)
		assert.Nil(t, asset)
		assert.Error(t, err)
		assert.Equal(t, "asset not found", err.Error())
	})

	t.Run("database error", func(t *testing.T) {
		mock.ExpectQuery(`SELECT`).
			WithArgs(int64(1)).
			WillReturnError(assert.AnError)

		asset, err := model.GetByID(1)
		assert.Nil(t, asset)
		assert.Error(t, err)
	})
}

func TestAssetsModel_GetAll(t *testing.T) {
	model, mock, teardown := setupAssetTest(t)
	defer teardown()

	now := time.Now()

	t.Run("get all assets", func(t *testing.T) {
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

		assets, err := model.GetAll()
		assert.NoError(t, err)
		assert.Len(t, assets, 2)
		assert.Equal(t, "DPA-PC001", assets[0].InternalID)
		assert.Equal(t, "AM-M001", assets[1].InternalID)
	})

	t.Run("get assets with filters", func(t *testing.T) {
		userID := int64(2)
		filters := []AssetFilter{
			{Type: "PC"},
			{Status: "IN_USE"},
			{InUseBy: &userID},
		}

		mock.ExpectQuery(`SELECT.*asset_type = \$1.*status = \$2.*in_use_by = \$3`).
			WithArgs("PC", "IN_USE", userID).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "internal_id", "asset_type", "manufacturer", "model", 
				"model_number", "serial_number", "status", "in_use_by", 
				"date_purchased", "last_service_date", "next_service_date", 
				"created_at", "updated_at",
			}).AddRow(
				1, "DPA-PC001", "PC", "Dell", "OptiPlex 7070", 
				"OP7070", "ABC123456", "IN_USE", &userID,
				now, nil, nil, now, now,
			))

		assets, err := model.GetAll(filters...)
		assert.NoError(t, err)
		assert.Len(t, assets, 1)
		assert.Equal(t, "PC", assets[0].AssetType)
		assert.Equal(t, "IN_USE", assets[0].Status)
		assert.Equal(t, &userID, assets[0].InUseBy)
	})
}

func TestAssetsModel_AssignAsset(t *testing.T) {
	model, mock, teardown := setupAssetTest(t)
	defer teardown()

	t.Run("successful assignment", func(t *testing.T) {
		// Mock user exists check
		mock.ExpectQuery(`SELECT EXISTS`).
			WithArgs(int64(2)).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		// Mock asset update
		now := time.Now()
		mock.ExpectQuery(`UPDATE assets`).
			WithArgs(int64(2), int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"updated_at"}).AddRow(now))

		err := model.AssignAsset(1, 2)
		assert.NoError(t, err)
	})

	t.Run("user not found", func(t *testing.T) {
		mock.ExpectQuery(`SELECT EXISTS`).
			WithArgs(int64(999)).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		err := model.AssignAsset(1, 999)
		assert.Error(t, err)
		assert.Equal(t, "user not found", err.Error())
	})

	t.Run("asset not found or cannot be assigned", func(t *testing.T) {
		mock.ExpectQuery(`SELECT EXISTS`).
			WithArgs(int64(2)).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		// Mock returns no rows (sql.ErrNoRows scenario)
		mock.ExpectQuery(`UPDATE assets`).
			WithArgs(int64(2), int64(999)).
			WillReturnError(sql.ErrNoRows)

		err := model.AssignAsset(999, 2)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "asset not found or cannot be assigned")
	})
}

func TestAssetsModel_GetAssetStats(t *testing.T) {
	model, mock, teardown := setupAssetTest(t)
	defer teardown()

	t.Run("successful stats retrieval", func(t *testing.T) {
		mock.ExpectQuery(`SELECT`).
			WillReturnRows(sqlmock.NewRows([]string{
				"total_assets", "in_use", "in_storage", "in_repair", 
				"retired", "needs_service", "asset_types_count",
			}).AddRow(100, 75, 15, 5, 5, 10, 8))

		stats, err := model.GetAssetStats()
		assert.NoError(t, err)
		assert.Equal(t, 100, stats.TotalAssets)
		assert.Equal(t, 75, stats.InUse)
		assert.Equal(t, 15, stats.InStorage)
		assert.Equal(t, 5, stats.InRepair)
		assert.Equal(t, 5, stats.Retired)
		assert.Equal(t, 10, stats.NeedsService)
		assert.Equal(t, 8, stats.AssetTypesCount)
	})
}

// Helper function
func int64Ptr(i int64) *int64 {
	return &i
}