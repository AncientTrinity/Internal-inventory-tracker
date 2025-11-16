package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"victortillett.net/internal-inventory-tracker/internal/models"
	"victortillett.net/internal-inventory-tracker/internal/middleware"
	"victortillett.net/internal-inventory-tracker/internal/services"
)

type TicketCommentsHandler struct {
	CommentModel *models.TicketCommentModel
	TicketModel  *models.TicketModel
	EmailService  *services.EmailService
}

func NewTicketCommentsHandler(db *sql.DB, emailService *services.EmailService) *TicketCommentsHandler {
	return &TicketCommentsHandler{
		CommentModel: models.NewTicketCommentModel(db),
		TicketModel:  models.NewTicketModel(db),
		EmailService: emailService, // FIXED: Use the parameter
	}
}


// POST /api/v1/tickets/{id}/comments
func (h *TicketCommentsHandler) CreateComment(w http.ResponseWriter, r *http.Request) {
	// Extract ticket ID from URL
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/tickets/")
	pathParts := strings.Split(path, "/")
	if len(pathParts) < 2 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	ticketID, err := strconv.ParseInt(pathParts[0], 10, 64)
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

	// Verify ticket exists
	_, err = h.TicketModel.GetByID(ticketID)
	if err != nil {
		if err.Error() == "ticket not found" {
			http.Error(w, "Ticket not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	var input struct {
		Comment    string `json:"comment"`
		IsInternal bool   `json:"is_internal"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid input: "+err.Error(), http.StatusBadRequest)
		return
	}

	if input.Comment == "" {
		http.Error(w, "Comment is required", http.StatusBadRequest)
		return
	}

	authorID := int64(userID)
	comment := &models.TicketComment{
		TicketID:   ticketID,
		AuthorID:   &authorID,
		Comment:    input.Comment,
		IsInternal: input.IsInternal,
	}

	err = h.CommentModel.Insert(comment)
	if err != nil {
		http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	//send email notification to ticket watchers
	go h.sendCommentNotification(comment, ticketID)

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(comment)

}

// GET /api/v1/tickets/{id}/comments
func (h *TicketCommentsHandler) GetComments(w http.ResponseWriter, r *http.Request) {
	// Extract ticket ID from URL
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/tickets/")
	pathParts := strings.Split(path, "/")
	if len(pathParts) < 2 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	ticketID, err := strconv.ParseInt(pathParts[0], 10, 64)
	if err != nil {
		http.Error(w, "Invalid ticket ID", http.StatusBadRequest)
		return
	}

	// Get current user from context for internal comment visibility
	roleID, ok := r.Context().Value(middleware.ContextRoleID).(int)
	if !ok {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	// Only IT staff and admins can see internal comments
	showInternal := (roleID == 1 || roleID == 2) // Admin or IT staff

	comments, err := h.CommentModel.GetByTicketID(ticketID, showInternal)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(comments)
}

// PUT /api/v1/comments/{id}
func (h *TicketCommentsHandler) UpdateComment(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/comments/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid comment ID", http.StatusBadRequest)
		return
	}

	// Get current user from context
	userID, ok := r.Context().Value(middleware.ContextUserID).(int)
	if !ok {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	// Get existing comment
	existingComment, err := h.CommentModel.GetByID(id)
	if err != nil {
		if err.Error() == "comment not found" {
			http.Error(w, "Comment not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Check if user is the author (or admin)
	if existingComment.AuthorID == nil || *existingComment.AuthorID != int64(userID) {
		// Check if user is admin
		roleID, ok := r.Context().Value(middleware.ContextRoleID).(int)
		if !ok || roleID != 1 {
			http.Error(w, "Forbidden - can only edit your own comments", http.StatusForbidden)
			return
		}
	}

	var input struct {
		Comment    string `json:"comment"`
		IsInternal bool   `json:"is_internal"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid input: "+err.Error(), http.StatusBadRequest)
		return
	}

	if input.Comment == "" {
		http.Error(w, "Comment is required", http.StatusBadRequest)
		return
	}

	// Update comment
	existingComment.Comment = input.Comment
	existingComment.IsInternal = input.IsInternal

	err = h.CommentModel.Update(existingComment)
	if err != nil {
		http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(existingComment)
}

// DELETE /api/v1/comments/{id}
func (h *TicketCommentsHandler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/comments/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid comment ID", http.StatusBadRequest)
		return
	}

	// Get current user from context
	userID, ok := r.Context().Value(middleware.ContextUserID).(int)
	if !ok {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	// Get existing comment
	existingComment, err := h.CommentModel.GetByID(id)
	if err != nil {
		if err.Error() == "comment not found" {
			http.Error(w, "Comment not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Check if user is the author (or admin)
	if existingComment.AuthorID == nil || *existingComment.AuthorID != int64(userID) {
		// Check if user is admin
		roleID, ok := r.Context().Value(middleware.ContextRoleID).(int)
		if !ok || roleID != 1 {
			http.Error(w, "Forbidden - can only delete your own comments", http.StatusForbidden)
			return
		}
	}

	err = h.CommentModel.Delete(id)
	if err != nil {
		http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// sendCommentNotification sends email notifications to ticket watchers about a new comment
func (h *TicketCommentsHandler) sendCommentNotification(comment *models.TicketComment, ticketID int64) {
	// Get ticket details
	ticket, err := h.TicketModel.GetByID(ticketID)
	if err != nil {
		return
	}

	// Get comment author info
	var authorUsername string
	if comment.AuthorID != nil {
		h.TicketModel.DB.QueryRow(
			"SELECT username FROM users WHERE id = $1", 
			comment.AuthorID,
		).Scan(&authorUsername)
	}

	// Notify ticket creator (if different from comment author)
	if ticket.CreatedBy != nil && (comment.AuthorID == nil || *ticket.CreatedBy != *comment.AuthorID) {
		var creatorEmail string
		err := h.TicketModel.DB.QueryRow(
			"SELECT email FROM users WHERE id = $1", 
			ticket.CreatedBy,
		).Scan(&creatorEmail)
		
		if err == nil && creatorEmail != "" {
			h.EmailService.SendTicketCommentEmail(
				creatorEmail,
				ticket.TicketNum,
				ticket.Title,
				comment.Comment,
				authorUsername,
			)
		}
	}

	// Notify assigned user (if different from comment author and creator)
	if ticket.AssignedTo != nil && 
	   (comment.AuthorID == nil || *ticket.AssignedTo != *comment.AuthorID) &&
	   (ticket.CreatedBy == nil || *ticket.AssignedTo != *ticket.CreatedBy) {
		
		var assigneeEmail string
		err := h.TicketModel.DB.QueryRow(
			"SELECT email FROM users WHERE id = $1", 
			ticket.AssignedTo,
		).Scan(&assigneeEmail)
		
		if err == nil && assigneeEmail != "" {
			h.EmailService.SendTicketCommentEmail(
				assigneeEmail,
				ticket.TicketNum,
				ticket.Title,
				comment.Comment,
				authorUsername,
			)
		}
	}
}

// End of file app/internal/handlers/ticket_comments.go