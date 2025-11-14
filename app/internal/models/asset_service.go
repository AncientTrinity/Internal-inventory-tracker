package models

import (
	"database/sql"
	"errors"
	"time"
)

type AssetServiceLog struct {
	ID               int64      `json:"id"`
	AssetID          int64      `json:"asset_id"`
	PerformedBy      *int64     `json:"performed_by"`      // User ID who performed the service
	PerformedAt      time.Time  `json:"performed_at"`      // When service was performed
	ServiceType      string     `json:"service_type"`      // MAINTENANCE, REPAIR, UPGRADE, etc.
	NextServiceDate  *time.Time `json:"next_service_date"` // When next service is due
	Notes            string     `json:"notes"`             // Service details
	CreatedAt        time.Time  `json:"created_at"`
}

type AssetServiceModel struct {
	DB *sql.DB
}

func NewAssetServiceModel(db *sql.DB) *AssetServiceModel {
	return &AssetServiceModel{DB: db}
}

// Insert a new service log
func (m *AssetServiceModel) Insert(log *AssetServiceLog) error {
	query := `
		INSERT INTO asset_service (
			asset_id, performed_by, performed_at, service_type, 
			next_service_date, notes
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`
	
	err := m.DB.QueryRow(
		query,
		log.AssetID,
		log.PerformedBy,
		log.PerformedAt,
		log.ServiceType,
		log.NextServiceDate,
		log.Notes,
	).Scan(&log.ID, &log.CreatedAt)
	
	return err
}

// Get service logs for a specific asset
func (m *AssetServiceModel) GetByAssetID(assetID int64) ([]AssetServiceLog, error) {
	query := `
		SELECT 
			id, asset_id, performed_by, performed_at, service_type,
			next_service_date, notes, created_at
		FROM asset_service 
		WHERE asset_id = $1
		ORDER BY performed_at DESC
	`
	
	rows, err := m.DB.Query(query, assetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var logs []AssetServiceLog
	for rows.Next() {
		var log AssetServiceLog
		err := rows.Scan(
			&log.ID,
			&log.AssetID,
			&log.PerformedBy,
			&log.PerformedAt,
			&log.ServiceType,
			&log.NextServiceDate,
			&log.Notes,
			&log.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}
	
	return logs, nil
}

// Get service log by ID
func (m *AssetServiceModel) GetByID(id int64) (*AssetServiceLog, error) {
	var log AssetServiceLog
	query := `
		SELECT 
			id, asset_id, performed_by, performed_at, service_type,
			next_service_date, notes, created_at
		FROM asset_service 
		WHERE id = $1
	`
	
	err := m.DB.QueryRow(query, id).Scan(
		&log.ID,
		&log.AssetID,
		&log.PerformedBy,
		&log.PerformedAt,
		&log.ServiceType,
		&log.NextServiceDate,
		&log.Notes,
		&log.CreatedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, errors.New("service log not found")
	} else if err != nil {
		return nil, err
	}
	
	return &log, nil
}

// Update asset's last_service_date when service is performed
func (m *AssetServiceModel) UpdateAssetServiceDate(assetID int64, serviceDate time.Time, nextServiceDate *time.Time) error {
	query := `
		UPDATE assets 
		SET last_service_date = $1, next_service_date = $2, updated_at = NOW()
		WHERE id = $3
	`
	
	_, err := m.DB.Exec(query, serviceDate, nextServiceDate, assetID)
	return err
}