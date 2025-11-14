package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"victortillett.net/internal-inventory-tracker/internal/middleware"
	"victortillett.net/internal-inventory-tracker/internal/models"
)

type TicketsHandler struct {
	TicketModel *models.TicketModel
	UsersModel  *models.UsersModel
	AssetsModel *models.AssetsModel
}

func NewTicketsHandler(db *sql.DB) *TicketsHandler {
	return &TicketsHandler{
		TicketModel: models.NewTicketModel(db),
		UsersModel:  models.NewUsersModel(db),
		AssetsModel: models.NewAssetsModel(db),
	}
}

// GET /api/v1/tickets
func (h *TicketsHandler) ListTickets(w http.ResponseWriter, r *http.Request) {
	// Get current user from context (set by auth middleware)
	userID, ok := r.Context().Value(middleware.ContextUserID).(int)
	if !ok {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	roleID, ok := r.Context().Value(middleware.ContextRoleID).(int)
	if !ok {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	// Parse query parameters
	status := r.URL.Query().Get("status")
	typeFilter := r.URL.Query().Get("type")
	priority := r.URL.Query().Get("priority")
	assignedToStr := r.URL.Query().Get("assigned_to")
	createdByStr := r.URL.Query().Get("created_by")

	filters := models.TicketFilters{
		Status:   status,
		Type:     typeFilter,
		Priority: priority,
	}

	// Handle role-based filtering
	switch roleID {
	case 1: // Admin - can see all tickets
		// No additional filters
	case 2: // IT Staff - can see assigned tickets and all open tickets
		if assignedToStr == "" {
			// IT staff sees tickets assigned to them OR unassigned tickets
			filters.AssignedTo = &[]int64{int64(userID), 0}[0] // This needs refinement
		}
	case 3: // Staff/Team Leads - can see tickets they created
		createdBy := int64(userID)
		filters.CreatedBy = &createdBy
	case 4: // Agents - can only see tickets they created
		agentView := int64(userID)
		filters.AgentView = &agentView
	case 5: // Viewers - read-only, similar to agents
		viewerView := int64(userID)
		filters.AgentView = &viewerView
	}

	// Parse assigned_to filter
	if assignedToStr != "" {
		if assignedTo, err := strconv.ParseInt(assignedToStr, 10, 64); err == nil {
			filters.AssignedTo = &assignedTo
		}
	}

	// Parse created_by filter
	if createdByStr != "" {
		if createdBy, err := strconv.ParseInt(createdByStr, 10, 64); err == nil {
			filters.CreatedBy = &createdBy
		}
	}

	// Parse pagination
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
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

	tickets, err := h.TicketModel.GetAll(filters)
	if err != nil {
		http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tickets)
}

// GET /api/v1/tickets/{id}
func (h *TicketsHandler) GetTicket(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/tickets/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid ticket ID", http.StatusBadRequest)
		return
	}

	ticket, err := h.TicketModel.GetByID(id)
	if err != nil {
		if err.Error() == "ticket not found" {
			http.Error(w, "Ticket not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ticket)
}

// POST /api/v1/tickets
func (h *TicketsHandler) CreateTicket(w http.ResponseWriter, r *http.Request) {
	// Get current user from context
	userID, ok := r.Context().Value(middleware.ContextUserID).(int)
	if !ok {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var input struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Type        string `json:"type"`
		Priority    string `json:"priority"`
		AssetID     *int64 `json:"asset_id"`
		IsInternal  bool   `json:"is_internal"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid input: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate required fields
	if input.Title == "" {
		http.Error(w, "Title is required", http.StatusBadRequest)
		return
	}
	if input.Description == "" {
		http.Error(w, "Description is required", http.StatusBadRequest)
		return
	}
	if input.Type == "" {
		http.Error(w, "Type is required", http.StatusBadRequest)
		return
	}

	// Validate asset exists if provided
	if input.AssetID != nil {
		_, err := h.AssetsModel.GetByID(*input.AssetID)
		if err != nil {
			http.Error(w, "Asset not found", http.StatusBadRequest)
			return
		}
	}

	// Set default priority if not provided
	if input.Priority == "" {
		input.Priority = "normal"
	}

	createdBy := int64(userID)
	ticket := &models.Ticket{
		Title:       input.Title,
		Description: input.Description,
		Type:        input.Type,
		Priority:    input.Priority,
		Status:      "open", // Always start as open
		Completion:  0,      // Start at 0%
		CreatedBy:   &createdBy,
		AssetID:     input.AssetID,
		IsInternal:  input.IsInternal,
	}

	err := h.TicketModel.Insert(ticket)
	if err != nil {
		http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ticket)
}

// PUT /api/v1/tickets/{id}
// PUT /api/v1/tickets/{id}
func (h *TicketsHandler) UpdateTicket(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/tickets/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid ticket ID", http.StatusBadRequest)
		return
	}

	// Get current user from context
	userID, ok := r.Context().Value(middleware.ContextUserID).(int)
	if !ok {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	roleID, ok := r.Context().Value(middleware.ContextRoleID).(int)
	if !ok {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	// Get existing ticket
	existingTicket, err := h.TicketModel.GetByID(id)
	if err != nil {
		if err.Error() == "ticket not found" {
			http.Error(w, "Ticket not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Authorization check - only allow updates if:
	// - User is admin (roleID 1)
	// - User is IT staff (roleID 2) and ticket is assigned to them or unassigned
	// - User created the ticket (for non-IT roles)
	switch roleID {
	case 1: // Admin - can update any ticket
		// No restrictions
	case 2: // IT Staff - can update tickets assigned to them or unassigned tickets
		if existingTicket.AssignedTo != nil && *existingTicket.AssignedTo != int64(userID) {
			http.Error(w, "Forbidden: You can only update tickets assigned to you", http.StatusForbidden)
			return
		}
	default: // Other roles - can only update tickets they created
		if existingTicket.CreatedBy == nil || *existingTicket.CreatedBy != int64(userID) {
			http.Error(w, "Forbidden: You can only update tickets you created", http.StatusForbidden)
			return
		}
	}

	var input struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Type        string `json:"type"`
		Priority    string `json:"priority"`
		AssetID     *int64 `json:"asset_id"`
		IsInternal  bool   `json:"is_internal"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid input: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Update fields (only if provided)
	if input.Title != "" {
		existingTicket.Title = input.Title
	}
	if input.Description != "" {
		existingTicket.Description = input.Description
	}
	if input.Type != "" {
		existingTicket.Type = input.Type
	}
	if input.Priority != "" {
		existingTicket.Priority = input.Priority
	}
	if input.AssetID != nil {
		// Validate asset exists
		_, err := h.AssetsModel.GetByID(*input.AssetID)
		if err != nil {
			http.Error(w, "Asset not found", http.StatusBadRequest)
			return
		}
		existingTicket.AssetID = input.AssetID
	}
	existingTicket.IsInternal = input.IsInternal

	err = h.TicketModel.Update(existingTicket)
	if err != nil {
		http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Get updated ticket
	updatedTicket, err := h.TicketModel.GetByID(id)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedTicket)
}

// POST /api/v1/tickets/{id}/status
func (h *TicketsHandler) UpdateTicketStatus(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/tickets/")
	idStr = strings.TrimSuffix(idStr, "/status")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid ticket ID", http.StatusBadRequest)
		return
	}

	var input struct {
		Status      string `json:"status"`
		Completion  int    `json:"completion"`
		AssignedTo  *int64 `json:"assigned_to"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid input: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate status transition
	validStatuses := map[string]bool{
		"open": true, "received": true, "in_progress": true, 
		"resolved": true, "closed": true,
	}
	if !validStatuses[input.Status] {
		http.Error(w, "Invalid status", http.StatusBadRequest)
		return
	}

	// Validate completion percentage
	if input.Completion < 0 || input.Completion > 100 {
		http.Error(w, "Completion must be between 0 and 100", http.StatusBadRequest)
		return
	}

	// Validate assigned user exists if provided
	if input.AssignedTo != nil {
		var userExists bool
		err := h.TicketModel.DB.QueryRow(
			"SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", 
			input.AssignedTo,
		).Scan(&userExists)
		if err != nil || !userExists {
			http.Error(w, "Assigned user not found", http.StatusBadRequest)
			return
		}
	}

	// Update ticket status
	err = h.TicketModel.UpdateStatus(id, input.Status, input.Completion, input.AssignedTo)
	if err != nil {
		if err.Error() == "ticket not found" {
			http.Error(w, "Ticket not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Get updated ticket
	updatedTicket, err := h.TicketModel.GetByID(id)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Ticket status updated successfully",
		"ticket":  updatedTicket,
	})
}

// POST /api/v1/tickets/{id}/reassign
func (h *TicketsHandler) ReassignTicket(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/tickets/")
	idStr = strings.TrimSuffix(idStr, "/reassign")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid ticket ID", http.StatusBadRequest)
		return
	}

	var input struct {
		AssignedTo int64 `json:"assigned_to"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid input: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate user exists
	var userExists bool
	err = h.TicketModel.DB.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", 
		input.AssignedTo,
	).Scan(&userExists)
	if err != nil || !userExists {
		http.Error(w, "User not found", http.StatusBadRequest)
		return
	}

	// Reassign ticket
	err = h.TicketModel.ReassignTicket(id, input.AssignedTo)
	if err != nil {
		if err.Error() == "ticket not found" {
			http.Error(w, "Ticket not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Get updated ticket
	updatedTicket, err := h.TicketModel.GetByID(id)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Ticket reassigned successfully",
		"ticket":  updatedTicket,
	})
}

// GET /api/v1/tickets/stats
func (h *TicketsHandler) GetTicketStats(w http.ResponseWriter, r *http.Request) {
	// Get current user from context for role-based stats
	userID, ok := r.Context().Value(middleware.ContextUserID).(int)
	if !ok {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	roleID, ok := r.Context().Value(middleware.ContextRoleID).(int)
	if !ok {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	// Build stats query based on role
	var query string
	var args []interface{}

	switch roleID {
	case 1: // Admin - all tickets
		query = `
			SELECT 
				COUNT(*) as total,
				COUNT(CASE WHEN status = 'open' THEN 1 END) as open,
				COUNT(CASE WHEN status = 'received' THEN 1 END) as received,
				COUNT(CASE WHEN status = 'in_progress' THEN 1 END) as in_progress,
				COUNT(CASE WHEN status = 'resolved' THEN 1 END) as resolved,
				COUNT(CASE WHEN status = 'closed' THEN 1 END) as closed,
				COUNT(CASE WHEN priority = 'critical' THEN 1 END) as critical
			FROM tickets
		`
	case 2: // IT Staff - assigned tickets + open tickets
		query = `
			SELECT 
				COUNT(*) as total,
				COUNT(CASE WHEN status = 'open' THEN 1 END) as open,
				COUNT(CASE WHEN status = 'received' THEN 1 END) as received,
				COUNT(CASE WHEN status = 'in_progress' THEN 1 END) as in_progress,
				COUNT(CASE WHEN status = 'resolved' THEN 1 END) as resolved,
				COUNT(CASE WHEN status = 'closed' THEN 1 END) as closed,
				COUNT(CASE WHEN priority = 'critical' THEN 1 END) as critical
			FROM tickets
			WHERE assigned_to = $1 OR status = 'open'
		`
		args = append(args, userID)
	default: // Staff/Agents - only their tickets
		query = `
			SELECT 
				COUNT(*) as total,
				COUNT(CASE WHEN status = 'open' THEN 1 END) as open,
				COUNT(CASE WHEN status = 'received' THEN 1 END) as received,
				COUNT(CASE WHEN status = 'in_progress' THEN 1 END) as in_progress,
				COUNT(CASE WHEN status = 'resolved' THEN 1 END) as resolved,
				COUNT(CASE WHEN status = 'closed' THEN 1 END) as closed,
				COUNT(CASE WHEN priority = 'critical' THEN 1 END) as critical
			FROM tickets
			WHERE created_by = $1
		`
		args = append(args, userID)
	}

	var stats struct {
		Total      int `json:"total"`
		Open       int `json:"open"`
		Received   int `json:"received"`
		InProgress int `json:"in_progress"`
		Resolved   int `json:"resolved"`
		Closed     int `json:"closed"`
		Critical   int `json:"critical"`
	}

	err := h.TicketModel.DB.QueryRow(query, args...).Scan(
		&stats.Total,
		&stats.Open,
		&stats.Received,
		&stats.InProgress,
		&stats.Resolved,
		&stats.Closed,
		&stats.Critical,
	)
	if err != nil {
		http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}