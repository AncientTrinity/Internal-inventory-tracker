package models

import (
	"database/sql"
	"errors"
	"time"
	"strconv"
	"strings"
)

type Asset struct {
	ID              int64      `json:"id"`
	InternalID      string     `json:"internal_id"`      // DPA-PC001, AM-M001, etc.
	AssetType       string     `json:"asset_type"`       // PC, Monitor, Keyboard, Mouse, Headset, UPS
	Manufacturer    string     `json:"manufacturer"`     // Dell, Viewsonic, Acer, Samsung, etc.
	Model           string     `json:"model"`            // Model name
	ModelNumber     string     `json:"model_number"`     // Manufacturer model number
	SerialNumber    string     `json:"serial_number"`    // Serial number
	Status          string     `json:"status"`           // IN_USE, IN_STORAGE, RETIRED, REPAIR
	InUseBy         *int64     `json:"in_use_by"`        // User ID if assigned
	DatePurchased   *time.Time `json:"date_purchased"`   // Purchase date
	LastServiceDate *time.Time `json:"last_service_date"` // Last service date
	NextServiceDate *time.Time `json:"next_service_date"` // Next service date
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type AssetsModel struct {
	DB *sql.DB
}

func NewAssetsModel(db *sql.DB) *AssetsModel {
	return &AssetsModel{DB: db}
}

// Insert a new asset
func (m *AssetsModel) Insert(asset *Asset) error {
	query := `
		INSERT INTO assets (
			internal_id, asset_type, manufacturer, model, model_number, 
			serial_number, status, in_use_by, date_purchased, 
			last_service_date, next_service_date
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at, updated_at
	`
	
	err := m.DB.QueryRow(
		query,
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
	).Scan(&asset.ID, &asset.CreatedAt, &asset.UpdatedAt)
	
	return err
}

// Get asset by ID
func (m *AssetsModel) GetByID(id int64) (*Asset, error) {
	var asset Asset
	query := `
		SELECT 
			id, internal_id, asset_type, manufacturer, model, model_number,
			serial_number, status, in_use_by, date_purchased, last_service_date,
			next_service_date, created_at, updated_at
		FROM assets 
		WHERE id = $1
	`
	
	err := m.DB.QueryRow(query, id).Scan(
		&asset.ID,
		&asset.InternalID,
		&asset.AssetType,
		&asset.Manufacturer,
		&asset.Model,
		&asset.ModelNumber,
		&asset.SerialNumber,
		&asset.Status,
		&asset.InUseBy,
		&asset.DatePurchased,
		&asset.LastServiceDate,
		&asset.NextServiceDate,
		&asset.CreatedAt,
		&asset.UpdatedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, errors.New("asset not found")
	} else if err != nil {
		return nil, err
	}
	
	return &asset, nil
}

// Get all assets with optional filtering
func (m *AssetsModel) GetAll(filters ...AssetFilter) ([]Asset, error) {
	query := `
		SELECT 
			id, internal_id, asset_type, manufacturer, model, model_number,
			serial_number, status, in_use_by, date_purchased, last_service_date,
			next_service_date, created_at, updated_at
		FROM assets 
		WHERE 1=1
	`
	args := []interface{}{}
	argPos := 1
	
	// Apply filters
	for _, filter := range filters {
		if filter.Type != "" {
			query += " AND asset_type = $" + string(rune(argPos+'0'))
			args = append(args, filter.Type)
			argPos++
		}
		if filter.Status != "" {
			query += " AND status = $" + string(rune(argPos+'0'))
			args = append(args, filter.Status)
			argPos++
		}
		if filter.InUseBy != nil {
			query += " AND in_use_by = $" + string(rune(argPos+'0'))
			args = append(args, *filter.InUseBy)
			argPos++
		}
	}
	
	query += " ORDER BY internal_id"
	
	rows, err := m.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var assets []Asset
	for rows.Next() {
		var asset Asset
		err := rows.Scan(
			&asset.ID,
			&asset.InternalID,
			&asset.AssetType,
			&asset.Manufacturer,
			&asset.Model,
			&asset.ModelNumber,
			&asset.SerialNumber,
			&asset.Status,
			&asset.InUseBy,
			&asset.DatePurchased,
			&asset.LastServiceDate,
			&asset.NextServiceDate,
			&asset.CreatedAt,
			&asset.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		assets = append(assets, asset)
	}
	
	return assets, nil
}

// Update an asset
func (m *AssetsModel) Update(asset *Asset) error {
	query := `
		UPDATE assets 
		SET 
			internal_id = $1, asset_type = $2, manufacturer = $3, 
			model = $4, model_number = $5, serial_number = $6, 
			status = $7, in_use_by = $8, date_purchased = $9, 
			last_service_date = $10, next_service_date = $11,
			updated_at = NOW()
		WHERE id = $12
		RETURNING updated_at
	`
	
	err := m.DB.QueryRow(
		query,
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
		asset.ID,
	).Scan(&asset.UpdatedAt)
	
	if err == sql.ErrNoRows {
		return errors.New("asset not found")
	}
	return err
}

// Delete an asset
func (m *AssetsModel) Delete(id int64) error {
	res, err := m.DB.Exec("DELETE FROM assets WHERE id = $1", id)
	if err != nil {
		return err
	}
	
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return errors.New("asset not found")
	}
	return nil
}

// AssetFilter for filtering assets
type AssetFilter struct {
	Type    string
	Status  string
	InUseBy *int64
}


// AssignAsset assigns an asset to a user
func (m *AssetsModel) AssignAsset(assetID, userID int64) error {
	// Verify user exists
	var userExists bool
	err := m.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", userID).Scan(&userExists)
	if err != nil {
		return err
	}
	if !userExists {
		return errors.New("user not found")
	}

	query := `
		UPDATE assets 
		SET in_use_by = $1, status = 'IN_USE', updated_at = NOW()
		WHERE id = $2 AND (status != 'RETIRED' AND status != 'REPAIR')
		RETURNING updated_at
	`
	
	var updatedAt time.Time
	err = m.DB.QueryRow(query, userID, assetID).Scan(&updatedAt)
	if err == sql.ErrNoRows {
		return errors.New("asset not found or cannot be assigned (might be retired or in repair)")
	}
	return err
}

// UnassignAsset removes user assignment from an asset
func (m *AssetsModel) UnassignAsset(assetID int64) error {
	query := `
		UPDATE assets 
		SET in_use_by = NULL, status = 'IN_STORAGE', updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`
	
	var updatedAt time.Time
	err := m.DB.QueryRow(query, assetID).Scan(&updatedAt)
	if err == sql.ErrNoRows {
		return errors.New("asset not found")
	}
	return err
}

// GetAssetsByUser gets all assets assigned to a specific user
func (m *AssetsModel) GetAssetsByUser(userID int64) ([]Asset, error) {
	query := `
		SELECT 
			id, internal_id, asset_type, manufacturer, model, model_number,
			serial_number, status, in_use_by, date_purchased, last_service_date,
			next_service_date, created_at, updated_at
		FROM assets 
		WHERE in_use_by = $1
		ORDER BY asset_type, internal_id
	`
	
	rows, err := m.DB.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var assets []Asset
	for rows.Next() {
		var asset Asset
		err := rows.Scan(
			&asset.ID,
			&asset.InternalID,
			&asset.AssetType,
			&asset.Manufacturer,
			&asset.Model,
			&asset.ModelNumber,
			&asset.SerialNumber,
			&asset.Status,
			&asset.InUseBy,
			&asset.DatePurchased,
			&asset.LastServiceDate,
			&asset.NextServiceDate,
			&asset.CreatedAt,
			&asset.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		assets = append(assets, asset)
	}
	
	return assets, nil
}

// GetAvailableAssets gets assets that are not assigned to any user
func (m *AssetsModel) GetAvailableAssets(assetType string) ([]Asset, error) {
	query := `
		SELECT 
			id, internal_id, asset_type, manufacturer, model, model_number,
			serial_number, status, in_use_by, date_purchased, last_service_date,
			next_service_date, created_at, updated_at
		FROM assets 
		WHERE in_use_by IS NULL AND status = 'IN_STORAGE'
	`
	
	args := []interface{}{}
	if assetType != "" {
		query += " AND asset_type = $1"
		args = append(args, assetType)
	}
	
	query += " ORDER BY asset_type, internal_id"
	
	rows, err := m.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var assets []Asset
	for rows.Next() {
		var asset Asset
		err := rows.Scan(
			&asset.ID,
			&asset.InternalID,
			&asset.AssetType,
			&asset.Manufacturer,
			&asset.Model,
			&asset.ModelNumber,
			&asset.SerialNumber,
			&asset.Status,
			&asset.InUseBy,
			&asset.DatePurchased,
			&asset.LastServiceDate,
			&asset.NextServiceDate,
			&asset.CreatedAt,
			&asset.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		assets = append(assets, asset)
	}
	
	return assets, nil
}


// SearchAssets performs advanced search across multiple fields
func (m *AssetsModel) SearchAssets(query string, filters AssetSearchFilters) ([]Asset, error) {
	baseQuery := `
		SELECT 
			id, internal_id, asset_type, manufacturer, model, model_number,
			serial_number, status, in_use_by, date_purchased, last_service_date,
			next_service_date, created_at, updated_at
		FROM assets 
		WHERE 1=1
	`
	
	args := []interface{}{}
	argPos := 1
	
	// Text search across multiple fields
	if query != "" {
		searchTerm := "%" + strings.ToLower(query) + "%"
		baseQuery += ` AND (
			LOWER(internal_id) LIKE $` + strconv.Itoa(argPos) + ` OR
			LOWER(asset_type) LIKE $` + strconv.Itoa(argPos) + ` OR
			LOWER(manufacturer) LIKE $` + strconv.Itoa(argPos) + ` OR
			LOWER(model) LIKE $` + strconv.Itoa(argPos) + ` OR
			LOWER(model_number) LIKE $` + strconv.Itoa(argPos) + ` OR
			LOWER(serial_number) LIKE $` + strconv.Itoa(argPos) + `
		)`
		args = append(args, searchTerm)
		argPos++
	}
	
	// Apply filters
	if filters.AssetType != "" {
		baseQuery += ` AND asset_type = $` + strconv.Itoa(argPos)
		args = append(args, filters.AssetType)
		argPos++
	}
	
	if filters.Status != "" {
		baseQuery += ` AND status = $` + strconv.Itoa(argPos)
		args = append(args, filters.Status)
		argPos++
	}
	
	if filters.Manufacturer != "" {
		baseQuery += ` AND LOWER(manufacturer) = $` + strconv.Itoa(argPos)
		args = append(args, strings.ToLower(filters.Manufacturer))
		argPos++
	}
	
	if filters.InUseBy != nil {
		baseQuery += ` AND in_use_by = $` + strconv.Itoa(argPos)
		args = append(args, *filters.InUseBy)
		argPos++
	}
	
	// Date range filters
	if !filters.PurchasedAfter.IsZero() {
		baseQuery += ` AND date_purchased >= $` + strconv.Itoa(argPos)
		args = append(args, filters.PurchasedAfter)
		argPos++
	}
	
	if !filters.PurchasedBefore.IsZero() {
		baseQuery += ` AND date_purchased <= $` + strconv.Itoa(argPos)
		args = append(args, filters.PurchasedBefore)
		argPos++
	}
	
	// Service date filters
	if filters.NeedsService {
		baseQuery += ` AND (next_service_date IS NOT NULL AND next_service_date <= CURRENT_DATE)`
	}
	
	if filters.OverdueService {
		baseQuery += ` AND (next_service_date IS NOT NULL AND next_service_date < CURRENT_DATE)`
	}
	
	// Sorting
	sortField := filters.SortBy
	if sortField == "" {
		sortField = "internal_id"
	}
	
	sortOrder := filters.SortOrder
	if sortOrder == "" {
		sortOrder = "ASC"
	}
	
	// Validate sort field to prevent SQL injection
	validSortFields := map[string]bool{
		"internal_id": true, "asset_type": true, "manufacturer": true, 
		"model": true, "status": true, "date_purchased": true,
		"last_service_date": true, "next_service_date": true, "created_at": true,
	}
	
	if !validSortFields[sortField] {
		sortField = "internal_id"
	}
	
	if sortOrder != "ASC" && sortOrder != "DESC" {
		sortOrder = "ASC"
	}
	
	baseQuery += ` ORDER BY ` + sortField + ` ` + sortOrder
	
	// Pagination
	if filters.Limit > 0 {
		baseQuery += ` LIMIT $` + strconv.Itoa(argPos)
		args = append(args, filters.Limit)
		argPos++
		
		if filters.Offset > 0 {
			baseQuery += ` OFFSET $` + strconv.Itoa(argPos)
			args = append(args, filters.Offset)
		}
	}
	
	rows, err := m.DB.Query(baseQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var assets []Asset
	for rows.Next() {
		var asset Asset
		err := rows.Scan(
			&asset.ID,
			&asset.InternalID,
			&asset.AssetType,
			&asset.Manufacturer,
			&asset.Model,
			&asset.ModelNumber,
			&asset.SerialNumber,
			&asset.Status,
			&asset.InUseBy,
			&asset.DatePurchased,
			&asset.LastServiceDate,
			&asset.NextServiceDate,
			&asset.CreatedAt,
			&asset.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		assets = append(assets, asset)
	}
	
	return assets, nil
}

// GetAssetStats returns statistics about assets
func (m *AssetsModel) GetAssetStats() (*AssetStats, error) {
	query := `
		SELECT 
			COUNT(*) as total_assets,
			COUNT(CASE WHEN status = 'IN_USE' THEN 1 END) as in_use,
			COUNT(CASE WHEN status = 'IN_STORAGE' THEN 1 END) as in_storage,
			COUNT(CASE WHEN status = 'REPAIR' THEN 1 END) as in_repair,
			COUNT(CASE WHEN status = 'RETIRED' THEN 1 END) as retired,
			COUNT(CASE WHEN next_service_date <= CURRENT_DATE THEN 1 END) as needs_service,
			COUNT(DISTINCT asset_type) as asset_types_count
		FROM assets
	`
	
	var stats AssetStats
	err := m.DB.QueryRow(query).Scan(
		&stats.TotalAssets,
		&stats.InUse,
		&stats.InStorage,
		&stats.InRepair,
		&stats.Retired,
		&stats.NeedsService,
		&stats.AssetTypesCount,
	)
	
	if err != nil {
		return nil, err
	}
	
	return &stats, nil
}

// AssetSearchFilters for advanced searching
type AssetSearchFilters struct {
	Query           string
	AssetType       string
	Status          string
	Manufacturer    string
	InUseBy         *int64
	PurchasedAfter  time.Time
	PurchasedBefore time.Time
	NeedsService    bool
	OverdueService  bool
	SortBy          string
	SortOrder       string
	Limit           int
	Offset          int
}

// AssetStats for dashboard
type AssetStats struct {
	TotalAssets     int `json:"total_assets"`
	InUse           int `json:"in_use"`
	InStorage       int `json:"in_storage"`
	InRepair        int `json:"in_repair"`
	Retired         int `json:"retired"`
	NeedsService    int `json:"needs_service"`
	AssetTypesCount int `json:"asset_types_count"`
}