// app/internal/models/tickets.go
package models

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type Ticket struct {
	ID          int64      `json:"id"`
	TicketNum   string     `json:"ticket_num"`   // TCK-2025-0001
	Title       string     `json:"title"`        // Subject line
	Description string     `json:"description"`  // Issue details
	Type        string     `json:"type"`         // activation, deactivation, it_help, transition
	Priority    string     `json:"priority"`     // low, normal, high, critical
	Status      string     `json:"status"`       // open, received, in_progress, Investigating, resolved, closed
	Completion  int        `json:"completion"`   // 0-100 percentage
	CreatedBy   *int64     `json:"created_by"`   // Team lead who created
	AssignedTo  *int64     `json:"assigned_to"`  // IT staff assigned
	AssetID     *int64     `json:"asset_id"`     // Related asset (optional)
	IsInternal  bool       `json:"is_internal"`  // Internal ticket

	// Verification fields
	VerificationStatus string     `json:"verification_status"` // not_required, pending, verified, rejected
	VerificationNotes  string     `json:"verification_notes"`
	VerifiedBy         *int64     `json:"verified_by"`
	VerifiedAt         *time.Time `json:"verified_at"`

	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	ClosedAt    *time.Time `json:"closed_at"`
	
	// Joined fields for display
	CreatedByUser  *User `json:"created_by_user,omitempty"`
	AssignedToUser *User `json:"assigned_to_user,omitempty"`
	Asset          *Asset `json:"asset,omitempty"`
	VerifiedByUser *User `json:"verified_by_user,omitempty"`
}

type TicketModel struct {
	DB *sql.DB
}

func NewTicketModel(db *sql.DB) *TicketModel {
	return &TicketModel{DB: db}
}

// GenerateTicketNum generates a unique ticket number
// GenerateTicketNum generates a unique ticket number
// GenerateTicketNum generates a unique ticket number
func (m *TicketModel) GenerateTicketNum() (string, error) {
	year := time.Now().Year()
	
	// Try multiple approaches to get the next number
	var nextNum int
	
	// Approach 1: Get max number from existing tickets
	var maxNum sql.NullInt64
	err := m.DB.QueryRow(`
		SELECT MAX(NULLIF(REGEXP_REPLACE(ticket_num, '^TCK-[0-9]+-', ''), '')::INTEGER)
		FROM tickets 
		WHERE ticket_num ~ '^TCK-[0-9]+-[0-9]+$'
	`).Scan(&maxNum)
	
	if err != nil {
		return "", fmt.Errorf("failed to query max ticket number: %v", err)
	}
	
	if maxNum.Valid {
		nextNum = int(maxNum.Int64) + 1
	} else {
		nextNum = 1
	}
	
	// Generate the ticket number
	ticketNum := fmt.Sprintf("TCK-%d-%04d", year, nextNum)
	
	// Double-check it doesn't exist (race condition protection)
	var exists bool
	err = m.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM tickets WHERE ticket_num = $1)", ticketNum).Scan(&exists)
	if err != nil {
		return "", fmt.Errorf("failed to check ticket number existence: %v", err)
	}
	
	if exists {
		// If it exists (shouldn't happen), try with timestamp
		timestamp := time.Now().Unix() % 10000
		ticketNum = fmt.Sprintf("TCK-%d-%04d", year, timestamp)
	}
	
	return ticketNum, nil
}

// Insert a new ticket
func (m *TicketModel) Insert(ticket *Ticket) error {
	// Generate ticket number
	ticketNum, err := m.GenerateTicketNum()
	if err != nil {
		return err
	}
	ticket.TicketNum = ticketNum
	
	query := `
		INSERT INTO tickets (
			ticket_num, title, description, type, priority, status, 
			completion, created_by, assigned_to, asset_id, is_internal
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at, updated_at
	`
	
	err = m.DB.QueryRow(
		query,
		ticket.TicketNum,
		ticket.Title,
		ticket.Description,
		ticket.Type,
		ticket.Priority,
		ticket.Status,
		ticket.Completion,
		ticket.CreatedBy,
		ticket.AssignedTo,
		ticket.AssetID,
		ticket.IsInternal,
	).Scan(&ticket.ID, &ticket.CreatedAt, &ticket.UpdatedAt)
	
	return err
}

// Get ticket by ID with user and asset details
// Now with verification details
func (m *TicketModel) GetByID(id int64) (*Ticket, error) {
	query := `
		SELECT 
			t.id, t.ticket_num, t.title, t.description, t.type, t.priority,
			t.status, t.completion, t.created_by, t.assigned_to, t.asset_id,
			t.is_internal, t.verification_status, t.verification_notes,
			t.verified_by, t.verified_at, t.created_at, t.updated_at, t.closed_at,
			creator.id, creator.username, creator.full_name, creator.email,
			assignee.id, assignee.username, assignee.full_name, assignee.email,
			verifier.id, verifier.username, verifier.full_name, verifier.email,
			a.id, a.internal_id, a.asset_type, a.manufacturer, a.model
		FROM tickets t
		LEFT JOIN users creator ON t.created_by = creator.id
		LEFT JOIN users assignee ON t.assigned_to = assignee.id
		LEFT JOIN users verifier ON t.verified_by = verifier.id
		LEFT JOIN assets a ON t.asset_id = a.id
		WHERE t.id = $1
	`
	
	var ticket Ticket
	var creatorID, assigneeID, verifiedByID, assetID sql.NullInt64
	var creatorUsername, creatorFullName, creatorEmail sql.NullString
	var assigneeUsername, assigneeFullName, assigneeEmail sql.NullString
	var verifierUsername, verifierFullName, verifierEmail sql.NullString
	var assetInternalID, assetType, assetManufacturer, assetModel sql.NullString
	var verifiedAt sql.NullTime
	
	err := m.DB.QueryRow(query, id).Scan(
		&ticket.ID,
		&ticket.TicketNum,
		&ticket.Title,
		&ticket.Description,
		&ticket.Type,
		&ticket.Priority,
		&ticket.Status,
		&ticket.Completion,
		&creatorID,
		&assigneeID,
		&assetID,
		&ticket.IsInternal,
		&ticket.VerificationStatus,
		&ticket.VerificationNotes,
		&verifiedByID,
		&verifiedAt,
		&ticket.CreatedAt,
		&ticket.UpdatedAt,
		&ticket.ClosedAt,
		&creatorID, &creatorUsername, &creatorFullName, &creatorEmail,
		&assigneeID, &assigneeUsername, &assigneeFullName, &assigneeEmail,
		&verifiedByID, &verifierUsername, &verifierFullName, &verifierEmail,
		&assetID, &assetInternalID, &assetType, &assetManufacturer, &assetModel,
	)
	
	if err == sql.ErrNoRows {
		return nil, errors.New("ticket not found")
	} else if err != nil {
		return nil, err
	}
	
	// Populate joined user data
	if creatorID.Valid {
		ticket.CreatedByUser = &User{
			ID:       creatorID.Int64,
			Username: creatorUsername.String,
			FullName: creatorFullName.String,
			Email:    creatorEmail.String,
		}
		ticket.CreatedBy = &creatorID.Int64
	}
	
	if assigneeID.Valid {
		ticket.AssignedToUser = &User{
			ID:       assigneeID.Int64,
			Username: assigneeUsername.String,
			FullName: assigneeFullName.String,
			Email:    assigneeEmail.String,
		}
		ticket.AssignedTo = &assigneeID.Int64
	}
	
	if verifiedByID.Valid {
		ticket.VerifiedByUser = &User{
			ID:       verifiedByID.Int64,
			Username: verifierUsername.String,
			FullName: verifierFullName.String,
			Email:    verifierEmail.String,
		}
		ticket.VerifiedBy = &verifiedByID.Int64
	}
	
	if verifiedAt.Valid {
		ticket.VerifiedAt = &verifiedAt.Time
	}
	
	if assetID.Valid {
		ticket.Asset = &Asset{
			ID:         assetID.Int64,
			InternalID: assetInternalID.String,
			AssetType:  assetType.String,
			Manufacturer: assetManufacturer.String,
			Model:      assetModel.String,
		}
		ticket.AssetID = &assetID.Int64
	}
	
	return &ticket, nil
}

// Get all tickets with filtering
func (m *TicketModel) GetAll(filters TicketFilters) ([]Ticket, error) {
	query := `
		SELECT 
			t.id, t.ticket_num, t.title, t.description, t.type, t.priority,
			t.status, t.completion, t.created_by, t.assigned_to, t.asset_id,
			t.is_internal, t.created_at, t.updated_at, t.closed_at,
			creator.username, creator.full_name,
			assignee.username, assignee.full_name,
			a.internal_id
		FROM tickets t
		LEFT JOIN users creator ON t.created_by = creator.id
		LEFT JOIN users assignee ON t.assigned_to = assignee.id
		LEFT JOIN assets a ON t.asset_id = a.id
		WHERE 1=1
	`
	
	args := []interface{}{}
	argPos := 1
	
	// Apply filters
	if filters.Status != "" {
		query += fmt.Sprintf(" AND t.status = $%d", argPos)
		args = append(args, filters.Status)
		argPos++
	}
	
	if filters.Type != "" {
		query += fmt.Sprintf(" AND t.type = $%d", argPos)
		args = append(args, filters.Type)
		argPos++
	}
	
	if filters.Priority != "" {
		query += fmt.Sprintf(" AND t.priority = $%d", argPos)
		args = append(args, filters.Priority)
		argPos++
	}
	
	if filters.AssignedTo != nil {
		query += fmt.Sprintf(" AND t.assigned_to = $%d", argPos)
		args = append(args, *filters.AssignedTo)
		argPos++
	}
	
	if filters.CreatedBy != nil {
		query += fmt.Sprintf(" AND t.created_by = $%d", argPos)
		args = append(args, *filters.CreatedBy)
		argPos++
	}
	
	// For agents: can only see tickets they created
	if filters.AgentView != nil {
		query += fmt.Sprintf(" AND t.created_by = $%d", argPos)
		args = append(args, *filters.AgentView)
		argPos++
	}
	
	query += " ORDER BY t.created_at DESC"
	
	// Pagination
	if filters.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argPos)
		args = append(args, filters.Limit)
		argPos++
		
		if filters.Offset > 0 {
			query += fmt.Sprintf(" OFFSET $%d", argPos)
			args = append(args, filters.Offset)
		}
	}
	
	rows, err := m.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var tickets []Ticket
	for rows.Next() {
		var ticket Ticket
		var creatorUsername, creatorFullName sql.NullString
		var assigneeUsername, assigneeFullName sql.NullString
		var assetInternalID sql.NullString
		var createdBy, assignedTo, assetID sql.NullInt64
		
		err := rows.Scan(
			&ticket.ID,
			&ticket.TicketNum,
			&ticket.Title,
			&ticket.Description,
			&ticket.Type,
			&ticket.Priority,
			&ticket.Status,
			&ticket.Completion,
			&createdBy,
			&assignedTo,
			&assetID,
			&ticket.IsInternal,
			&ticket.CreatedAt,
			&ticket.UpdatedAt,
			&ticket.ClosedAt,
			&creatorUsername, &creatorFullName,
			&assigneeUsername, &assigneeFullName,
			&assetInternalID,
		)
		if err != nil {
			return nil, err
		}
		
		// Populate user info
		if createdBy.Valid {
			ticket.CreatedBy = &createdBy.Int64
			ticket.CreatedByUser = &User{
				Username: creatorUsername.String,
				FullName: creatorFullName.String,
			}
		}
		
		if assignedTo.Valid {
			ticket.AssignedTo = &assignedTo.Int64
			ticket.AssignedToUser = &User{
				Username: assigneeUsername.String,
				FullName: assigneeFullName.String,
			}
		}
		
		if assetID.Valid {
			ticket.AssetID = &assetID.Int64
			ticket.Asset = &Asset{
				InternalID: assetInternalID.String,
			}
		}
		
		tickets = append(tickets, ticket)
	}
	
	return tickets, nil
}

// Update ticket
func (m *TicketModel) Update(ticket *Ticket) error {
	query := `
		UPDATE tickets 
		SET 
			title = $1, description = $2, type = $3, priority = $4,
			status = $5, completion = $6, assigned_to = $7, asset_id = $8,
			is_internal = $9, updated_at = NOW(),
			closed_at = CASE WHEN $5 = 'closed' AND closed_at IS NULL THEN NOW() ELSE closed_at END
		WHERE id = $10
		RETURNING updated_at, closed_at
	`
	
	err := m.DB.QueryRow(
		query,
		ticket.Title,
		ticket.Description,
		ticket.Type,
		ticket.Priority,
		ticket.Status,
		ticket.Completion,
		ticket.AssignedTo,
		ticket.AssetID,
		ticket.IsInternal,
		ticket.ID,
	).Scan(&ticket.UpdatedAt, &ticket.ClosedAt)
	
	if err == sql.ErrNoRows {
		return errors.New("ticket not found")
	}
	return err
}

// Update ticket status and completion
func (m *TicketModel) UpdateStatus(id int64, status string, completion int, assignedTo *int64) error {
	query := `
		UPDATE tickets 
		SET status = $1, completion = $2, assigned_to = $3, updated_at = NOW(),
			closed_at = CASE WHEN $1 = 'closed' AND closed_at IS NULL THEN NOW() ELSE closed_at END
		WHERE id = $4
	`
	
	result, err := m.DB.Exec(query, status, completion, assignedTo, id)
	if err != nil {
		return err
	}
	
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return errors.New("ticket not found")
	}
	return nil
}

// Reassign ticket
func (m *TicketModel) ReassignTicket(ticketID, newAssigneeID int64) error {
	query := `
		UPDATE tickets 
		SET assigned_to = $1, updated_at = NOW()
		WHERE id = $2
	`
	
	result, err := m.DB.Exec(query, newAssigneeID, ticketID)
	if err != nil {
		return err
	}
	
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return errors.New("ticket not found")
	}
	return nil
}

// TicketFilters for querying tickets
type TicketFilters struct {
	Status     string
	Type       string
	Priority   string
	AssignedTo *int64
	CreatedBy  *int64
	AgentView  *int64 // For agent-specific view
	Limit      int
	Offset     int
}

// Delete ticket by ID
func (m *TicketModel) Delete(id int64) error {
	// First, delete related comments to maintain referential integrity
	_, err := m.DB.Exec("DELETE FROM ticket_comments WHERE ticket_id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete ticket comments: %v", err)
	}

	// Then delete the ticket
	result, err := m.DB.Exec("DELETE FROM tickets WHERE id = $1", id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return errors.New("ticket not found")
	}

	return nil
}

// Request verification for a ticket
// RequestVerification - Allow any user to request verification for tickets they created
func (m *TicketModel) RequestVerification(ticketID int64, notes string) error {
	query := `
		UPDATE tickets 
		SET 
			verification_status = 'pending',
			verification_notes = $1,
			status = 'resolved',
			completion = 90,
			updated_at = NOW()
		WHERE id = $2
	`
	
	result, err := m.DB.Exec(query, notes, ticketID)
	if err != nil {
		return err
	}
	
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return errors.New("ticket not found")
	}
	return nil
}

// VerifyTicket - Allow verification by creator OR Admin/IT staff
func (m *TicketModel) VerifyTicket(ticketID, userID int64, approved bool, notes string, userRoleID int) error {
	var newStatus string
	if approved {
		newStatus = "verified"
	} else {
		newStatus = "rejected"
	}
	
	// Build query based on who is verifying
	query := `
		UPDATE tickets 
		SET 
			verification_status = $1,
			verification_notes = COALESCE($2, verification_notes),
			verified_by = $3,
			verified_at = NOW(),
			status = CASE 
				WHEN $4 = true THEN 'closed' 
				ELSE 'in_progress' 
			END,
			completion = CASE 
				WHEN $4 = true THEN 100 
				ELSE 50 
			END,
			closed_at = CASE 
				WHEN $4 = true THEN NOW() 
				ELSE closed_at 
			END,
			updated_at = NOW()
		WHERE id = $5
	`
	
	result, err := m.DB.Exec(query, newStatus, notes, userID, approved, ticketID)
	if err != nil {
		return err
	}
	
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return errors.New("ticket not found")
	}
	return nil
}

// CanVerifyTicket - Check if user can verify this ticket
func (m *TicketModel) CanVerifyTicket(ticketID, userID int64, userRoleID int) (bool, error) {
	var createdBy sql.NullInt64
	err := m.DB.QueryRow(
		"SELECT created_by FROM tickets WHERE id = $1", 
		ticketID,
	).Scan(&createdBy)
	
	if err != nil {
		return false, err
	}
	
	// Admin/IT staff can verify any ticket
	if userRoleID == 1 || userRoleID == 2 { // Admin or IT Staff
		return true, nil
	}
	
	// Regular users can only verify tickets they created
	if createdBy.Valid && createdBy.Int64 == userID {
		return true, nil
	}
	
	return false, nil
}
// Skip verification for a ticket
func (m *TicketModel) SkipVerification(ticketID int64) error {
	query := `
		UPDATE tickets 
		SET 
			verification_status = 'not_required',
			status = 'closed',
			completion = 100,
			closed_at = NOW(),
			updated_at = NOW()
		WHERE id = $1
	`
	
	result, err := m.DB.Exec(query, ticketID)
	if err != nil {
		return err
	}
	
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return errors.New("ticket not found")
	}
	return nil
}

func (m *TicketModel) SetupVerification(ticketID int64, status, notes string) error {
	query := `
		UPDATE tickets 
		SET 
			verification_status = $1,
			verification_notes = $2,
			updated_at = NOW()
		WHERE id = $3
	`
	
	result, err := m.DB.Exec(query, status, notes, ticketID)
	if err != nil {
		return err
	}
	
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return errors.New("ticket not found")
	}
	return nil
}

// ResetVerification resets a ticket's verification status to pending
func (m *TicketModel) ResetVerification(ticketID, userID int64) error {
    query := `
        UPDATE tickets 
        SET verification_status = 'pending', 
            verification_notes = $1,
            verified_by = NULL,
            verified_at = NULL,
            updated_at = NOW()
        WHERE id = $2
    `
    
    _, err := m.DB.Exec(query, "Verification reset by user", ticketID)
    return err
}