//app/internal/handlers/tickets.go
package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"fmt"

	"victortillett.net/internal-inventory-tracker/internal/middleware"
	"victortillett.net/internal-inventory-tracker/internal/models"
	"victortillett.net/internal-inventory-tracker/internal/services"
)

type TicketsHandler struct {
	TicketModel *models.TicketModel
	UsersModel  *models.UsersModel
	AssetsModel *models.AssetsModel
	EmailService *services.EmailService
	NotificationService  *services.NotificationService
}

func NewTicketsHandler(db *sql.DB, emailService *services.EmailService) *TicketsHandler {
	return &TicketsHandler{
		TicketModel: models.NewTicketModel(db),
		UsersModel:  models.NewUsersModel(db),
		AssetsModel: models.NewAssetsModel(db),
		NotificationService: services.NewNotificationService(db),
		EmailService: emailService, // FIXED: Use the parameter
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

	// FIXED: Send notifications for new ticket
	go func() {
		if err := h.NotificationService.NotifyTicketCreated(ticket); err != nil {
			fmt.Printf("Failed to send notifications: %v\n", err)
		}
	}()

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ticket)
}

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

    // Authorization check
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

    // Send notifications for ticket update
    go func() {
        if err := h.NotificationService.NotifyTicketUpdated(updatedTicket, int64(userID), "updated"); err != nil {
            fmt.Printf("Failed to send notifications: %v\n", err)
        }
    }()

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(updatedTicket)
}

// DELETE /api/v1/tickets/{id}
func (h *TicketsHandler) DeleteTicket(w http.ResponseWriter, r *http.Request) {
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

	// Get existing ticket to check permissions
	existingTicket, err := h.TicketModel.GetByID(id)
	if err != nil {
		if err.Error() == "ticket not found" {
			http.Error(w, "Ticket not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Authorization check - only admin can delete tickets
	if roleID != 1 { // 1 = Admin
		http.Error(w, "Forbidden: Only administrators can delete tickets", http.StatusForbidden)
		return
	}

	// Delete the ticket
	err = h.TicketModel.Delete(id)
	if err != nil {
		if err.Error() == "ticket not found" {
			http.Error(w, "Ticket not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Send notification about ticket deletion
	go func() {
    fmt.Printf("Ticket #%s (ID: %d) deleted by user %d\n", 
        existingTicket.TicketNum, existingTicket.ID, userID)
}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Ticket deleted successfully",
		"ticket_id": id,
	})
}


// PUT /api/v1/tickets/{id}
func (h *TicketsHandler) UpdateTicketStatus(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/tickets/")
	idStr = strings.TrimSuffix(idStr, "/status")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid ticket ID", http.StatusBadRequest)
		return
	}

	// Get current user info for email
	userID, ok := r.Context().Value(middleware.ContextUserID).(int)
	if !ok {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var currentUserEmail, currentUsername string
	err = h.TicketModel.DB.QueryRow(
		"SELECT email, username FROM users WHERE id = $1", 
		userID,
	).Scan(&currentUserEmail, &currentUsername)
	if err != nil {
		// Log but don't fail the request
		fmt.Printf("Warning: Could not get current user info: %v\n", err)
	}

	// Get current ticket state for comparison
	currentTicket, err := h.TicketModel.GetByID(id)
	if err != nil {
		if err.Error() == "ticket not found" {
			http.Error(w, "Ticket not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
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

	// Send email notifications (in background goroutine)
	go h.sendStatusUpdateEmails(currentTicket, updatedTicket, currentUsername)

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

// sendStatusUpdateEmails handles all email notifications for ticket updates
func (h *TicketsHandler) sendStatusUpdateEmails(oldTicket, newTicket *models.Ticket, updatedBy string) {
	// Notify assigned user if assignment changed
	if newTicket.AssignedTo != nil && (oldTicket.AssignedTo == nil || *oldTicket.AssignedTo != *newTicket.AssignedTo) {
		var assigneeEmail string
		err := h.TicketModel.DB.QueryRow(
			"SELECT email FROM users WHERE id = $1", 
			newTicket.AssignedTo,
		).Scan(&assigneeEmail)
		
		if err == nil && assigneeEmail != "" {
			h.EmailService.SendTicketAssignedEmail(
				assigneeEmail,
				newTicket.TicketNum,
				newTicket.Title,
				updatedBy,
			)
		}
	}

	// Notify about status change
	if oldTicket.Status != newTicket.Status {
		// Notify ticket creator
		if newTicket.CreatedBy != nil {
			var creatorEmail string
			err := h.TicketModel.DB.QueryRow(
				"SELECT email FROM users WHERE id = $1", 
				newTicket.CreatedBy,
			).Scan(&creatorEmail)
			
			if err == nil && creatorEmail != "" {
				h.EmailService.SendTicketStatusUpdateEmail(
					creatorEmail,
					newTicket.TicketNum,
					newTicket.Title,
					oldTicket.Status,
					newTicket.Status,
					updatedBy,
				)
			}
		}

		// Notify assigned user (if different from creator)
		if newTicket.AssignedTo != nil && (newTicket.CreatedBy == nil || *newTicket.AssignedTo != *newTicket.CreatedBy) {
			var assigneeEmail string
			err := h.TicketModel.DB.QueryRow(
				"SELECT email FROM users WHERE id = $1", 
				newTicket.AssignedTo,
			).Scan(&assigneeEmail)
			
			if err == nil && assigneeEmail != "" {
				h.EmailService.SendTicketStatusUpdateEmail(
					assigneeEmail,
					newTicket.TicketNum,
					newTicket.Title,
					oldTicket.Status,
					newTicket.Status,
					updatedBy,
				)
			}
		}
	}
}

// POST /api/v1/tickets/{id}/request-verification
// POST /api/v1/tickets/{id}/request-verification
func (h *TicketsHandler) RequestVerification(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/tickets/")
	idStr = strings.TrimSuffix(idStr, "/request-verification")
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

	// Authorization: Only ticket creator OR Admin/IT can request verification
	canRequest := false
	if existingTicket.CreatedBy != nil && *existingTicket.CreatedBy == int64(userID) {
		canRequest = true
	} else if roleID == 1 || roleID == 2 { // Admin or IT Staff
		canRequest = true
	}

	if !canRequest {
		http.Error(w, "Forbidden: Only ticket creator or administrators can request verification", http.StatusForbidden)
		return
	}

	// Check if ticket is in a state that can be verified
	if existingTicket.Status != "resolved" && existingTicket.Status != "in_progress" {
		http.Error(w, "Ticket must be resolved or in progress to request verification", http.StatusBadRequest)
		return
	}

	var input struct {
		Notes string `json:"notes"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid input: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Request verification
	err = h.TicketModel.RequestVerification(id, input.Notes)
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

	// Send notifications
	go func() {
		if err := h.NotificationService.NotifyVerificationRequested(updatedTicket); err != nil {
			fmt.Printf("Failed to send verification notifications: %v\n", err)
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Verification requested successfully",
		"ticket":  updatedTicket,
	})
}

// POST /api/v1/tickets/{id}/verify
func (h *TicketsHandler) VerifyTicket(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/tickets/")
	idStr = strings.TrimSuffix(idStr, "/verify")
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

	// Check if user can verify this ticket
	canVerify, err := h.TicketModel.CanVerifyTicket(id, int64(userID), roleID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	if !canVerify {
		http.Error(w, "Forbidden: You cannot verify this ticket", http.StatusForbidden)
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

	// Check if verification is pending
	if existingTicket.VerificationStatus != "pending" {
		http.Error(w, "Ticket is not pending verification", http.StatusBadRequest)
		return
	}

	var input struct {
		Approved bool   `json:"approved"`
		Notes    string `json:"notes"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid input: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Verify ticket
	err = h.TicketModel.VerifyTicket(id, int64(userID), input.Approved, input.Notes, roleID)
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

	// Send notifications
	go func() {
		if err := h.NotificationService.NotifyVerificationCompleted(updatedTicket, input.Approved); err != nil {
			fmt.Printf("Failed to send verification notifications: %v\n", err)
		}
	}()

	action := "approved"
	if !input.Approved {
		action = "rejected"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": fmt.Sprintf("Verification %s successfully", action),
		"ticket":  updatedTicket,
	})
}

// POST /api/v1/tickets/{id}/skip-verification
func (h *TicketsHandler) SkipVerification(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/tickets/")
	idStr = strings.TrimSuffix(idStr, "/skip-verification")
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

	// Get existing ticket (use the variable to avoid "declared and not used" error)
	existingTicket, err := h.TicketModel.GetByID(id)
	if err != nil {
		if err.Error() == "ticket not found" {
			http.Error(w, "Ticket not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Authorization: Only Admin/IT can skip verification
	if roleID != 1 && roleID != 2 { // 1=Admin, 2=IT Staff
		http.Error(w, "Forbidden: Only administrators can skip verification", http.StatusForbidden)
		return
	}

	// Skip verification
	err = h.TicketModel.SkipVerification(id)
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

	// Log who skipped verification (using the userID variable)
	fmt.Printf("Verification skipped for ticket #%s by user %d\n", existingTicket.TicketNum, userID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Verification skipped successfully",
		"ticket":  updatedTicket,
	})
}

// POST /api/v1/tickets/{id}/setup-verification
func (h *TicketsHandler) SetupVerification(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/tickets/")
	idStr = strings.TrimSuffix(idStr, "/setup-verification")
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

	// Check if ticket is in a state that can be verified
	if existingTicket.Status != "resolved" && existingTicket.Status != "in_progress" {
		http.Error(w, "Ticket must be resolved or in progress to setup verification", http.StatusBadRequest)
		return
	}

	// Check if verification is already set
	if existingTicket.VerificationStatus != "not_required" {
		http.Error(w, "Verification is already set up for this ticket", http.StatusBadRequest)
		return
	}

	var input struct {
		VerificationStatus string `json:"verification_status"`
		VerificationNotes  string `json:"verification_notes"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid input: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Setup verification
	err = h.TicketModel.SetupVerification(id, input.VerificationStatus, input.VerificationNotes)
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

	// Send notifications
	go func() {
		if err := h.NotificationService.NotifyVerificationSetup(updatedTicket, int64(userID)); err != nil {
			fmt.Printf("Failed to send verification setup notifications: %v\n", err)
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Verification setup successfully",
		"ticket":  updatedTicket,
	})
}


// POST /api/v1/tickets/{id}/reset-verification
// POST /api/v1/tickets/{id}/reset-verification
func (h *TicketsHandler) ResetVerification(w http.ResponseWriter, r *http.Request) {
    fmt.Printf("🔍 ResetVerification called\n")
    
    idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/tickets/")
    idStr = strings.TrimSuffix(idStr, "/reset-verification")
    fmt.Printf("🔍 Extracted ID string: '%s'\n", idStr)
    
    id, err := strconv.ParseInt(idStr, 10, 64)
    if err != nil {
        fmt.Printf("❌ Error parsing ticket ID: %v\n", err)
        http.Error(w, "Invalid ticket ID", http.StatusBadRequest)
        return
    }
    fmt.Printf("🔍 Parsed ticket ID: %d\n", id)

    // Get current user from context
    userID, ok := r.Context().Value(middleware.ContextUserID).(int)
    if !ok {
        fmt.Printf("❌ Could not get userID from context\n")
        http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
        return
    }
    fmt.Printf("🔍 User ID from context: %d\n", userID)

    roleID, ok := r.Context().Value(middleware.ContextRoleID).(int)
    if !ok {
        fmt.Printf("❌ Could not get roleID from context\n")
        http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
        return
    }
    fmt.Printf("🔍 Role ID from context: %d\n", roleID)

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

    fmt.Printf("🔍 Ticket - CreatedBy: %v, VerifiedBy: %v, Status: %s\n", 
        existingTicket.CreatedBy, existingTicket.VerifiedBy, existingTicket.Status)

    // Prevent resetting verification for closed tickets
    if existingTicket.Status == "closed" {
        http.Error(w, "Cannot reset verification for closed tickets", http.StatusBadRequest)
        return
    }

    // Enhanced permission check
    canReset := false

    // Admin and IT Staff can always reset
    if roleID == 1 || roleID == 2 {
        canReset = true
        fmt.Printf("✅ Permission granted: User is Admin/IT Staff\n")
    } else if existingTicket.CreatedBy != nil && *existingTicket.CreatedBy == int64(userID) {
        // Allow ticket creator to reset their own verification
        canReset = true
        fmt.Printf("✅ Permission granted: User is ticket creator\n")
    } else if existingTicket.VerifiedBy != nil && *existingTicket.VerifiedBy == int64(userID) {
        // Allow the user who verified it to reset their own verification
        canReset = true
        fmt.Printf("✅ Permission granted: User is the verifier\n")
    } else {
        fmt.Printf("❌ Permission denied: User ID %d, CreatedBy: %v, VerifiedBy: %v\n", 
            userID, existingTicket.CreatedBy, existingTicket.VerifiedBy)
    }

    if !canReset {
        http.Error(w, "Forbidden: You cannot reset verification for this ticket", http.StatusForbidden)
        return
    }

    fmt.Printf("🔍 Calling ResetVerification with ticketID: %d, userID: %d\n", id, userID)
    
    // Reset verification to pending
    err = h.TicketModel.ResetVerification(id, int64(userID))
    if err != nil {
        fmt.Printf("❌ Error in ResetVerification model: %v\n", err)
        http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
        return
    }

    // Get updated ticket
    updatedTicket, err := h.TicketModel.GetByID(id)
    if err != nil {
        http.Error(w, "Database error", http.StatusInternalServerError)
        return
    }

    fmt.Printf("✅ Verification reset successfully for ticket #%s\n", existingTicket.TicketNum)

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "message": "Verification reset successfully",
        "ticket":  updatedTicket,
    })
}