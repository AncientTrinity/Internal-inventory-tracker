// file: app/internal/models/asset_service_test.go
package models

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupAssetServiceTest(t *testing.T) (*AssetServiceModel, sqlmock.Sqlmock, func()) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	model := NewAssetServiceModel(db)
	
	teardown := func() {
		db.Close()
	}

	return model, mock, teardown
}

func TestAssetServiceModel_Insert(t *testing.T) {
	model, mock, teardown := setupAssetServiceTest(t)
	defer teardown()

	now := time.Now()
	userID := int64(1)
	nextService := now.AddDate(0, 6, 0)

	serviceLog := &AssetServiceLog{
		AssetID:         1,
		PerformedBy:     &userID,
		PerformedAt:     now,
		ServiceType:     "MAINTENANCE",
		NextServiceDate: &nextService,
		Notes:           "Routine maintenance performed",
	}

	t.Run("successful insert", func(t *testing.T) {
		mock.ExpectQuery(`INSERT INTO asset_service`).
			WithArgs(
				serviceLog.AssetID,
				serviceLog.PerformedBy,
				serviceLog.PerformedAt,
				serviceLog.ServiceType,
				serviceLog.NextServiceDate,
				serviceLog.Notes,
			).
			WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).
				AddRow(1, now))

		err := model.Insert(serviceLog)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), serviceLog.ID)
		assert.Equal(t, now, serviceLog.CreatedAt)
	})

	t.Run("database error", func(t *testing.T) {
		mock.ExpectQuery(`INSERT INTO asset_service`).
			WillReturnError(assert.AnError)

		err := model.Insert(serviceLog)
		assert.Error(t, err)
	})
}

func TestAssetServiceModel_GetByAssetID(t *testing.T) {
	model, mock, teardown := setupAssetServiceTest(t)
	defer teardown()

	now := time.Now()
	userID := int64(1)
	nextService := now.AddDate(0, 6, 0)

	t.Run("successful retrieval", func(t *testing.T) {
		mock.ExpectQuery(`SELECT`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "asset_id", "performed_by", "performed_at", "service_type",
				"next_service_date", "notes", "created_at",
			}).AddRow(
				1, 1, &userID, now, "MAINTENANCE",
				&nextService, "Routine maintenance", now,
			).AddRow(
				2, 1, &userID, now.AddDate(0, -6, 0), "REPAIR",
				nil, "Fixed hardware issue", now.AddDate(0, -6, 0),
			))

		logs, err := model.GetByAssetID(1)
		assert.NoError(t, err)
		assert.Len(t, logs, 2)
		assert.Equal(t, "MAINTENANCE", logs[0].ServiceType)
		assert.Equal(t, "REPAIR", logs[1].ServiceType)
	})

	t.Run("database error", func(t *testing.T) {
		mock.ExpectQuery(`SELECT`).
			WithArgs(int64(1)).
			WillReturnError(assert.AnError)

		logs, err := model.GetByAssetID(1)
		assert.Error(t, err)
		assert.Nil(t, logs)
	})
}

func TestAssetServiceModel_GetByID(t *testing.T) {
	model, mock, teardown := setupAssetServiceTest(t)
	defer teardown()

	now := time.Now()
	userID := int64(1)
	nextService := now.AddDate(0, 6, 0)

	t.Run("successful retrieval", func(t *testing.T) {
		mock.ExpectQuery(`SELECT`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "asset_id", "performed_by", "performed_at", "service_type",
				"next_service_date", "notes", "created_at",
			}).AddRow(
				1, 1, &userID, now, "MAINTENANCE",
				&nextService, "Routine maintenance", now,
			))

		log, err := model.GetByID(1)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), log.ID)
		assert.Equal(t, "MAINTENANCE", log.ServiceType)
	})

	t.Run("service log not found", func(t *testing.T) {
		mock.ExpectQuery(`SELECT`).
			WithArgs(int64(999)).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "asset_id", "performed_by", "performed_at", "service_type",
				"next_service_date", "notes", "created_at",
			}))

		log, err := model.GetByID(999)
		assert.Error(t, err)
		assert.Nil(t, log)
		assert.Equal(t, "service log not found", err.Error())
	})
}

func TestAssetServiceModel_UpdateAssetServiceDate(t *testing.T) {
	model, mock, teardown := setupAssetServiceTest(t)
	defer teardown()

	now := time.Now()
	nextService := now.AddDate(0, 6, 0)

	t.Run("successful update", func(t *testing.T) {
		mock.ExpectExec(`UPDATE assets`).
			WithArgs(now, &nextService, int64(1)).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := model.UpdateAssetServiceDate(1, now, &nextService)
		assert.NoError(t, err)
	})

	t.Run("update with nil next service date", func(t *testing.T) {
		mock.ExpectExec(`UPDATE assets`).
			WithArgs(now, nil, int64(1)).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := model.UpdateAssetServiceDate(1, now, nil)
		assert.NoError(t, err)
	})
}