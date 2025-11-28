// file: app/internal/handlers/ticket_test.go
package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"victortillett.net/internal-inventory-tracker/internal/middleware"
	"victortillett.net/internal-inventory-tracker/internal/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockEmailService for testing - implements the minimum required interface
type MockEmailService struct{}

func (m *MockEmailService) SendTicketAssignedEmail(to, ticketNum, title, assignedBy string) error {
	return nil
}

func (m *MockEmailService) SendTicketStatusUpdateEmail(to, ticketNum, title, oldStatus, newStatus, updatedBy string) error {
	return nil
}

// Add any other methods required by your EmailService interface
func (m *MockEmailService) SendEmailWithRetry(to, subject, body string, maxRetries int) error {
	return nil
}

func (m *MockEmailService) SendBulkEmails(emails []struct {
	To      string
	Subject string
	Body    string
}) error {
	return nil
}

func setupTicketsTest(t *testing.T) (*TicketsHandler, sqlmock.Sqlmock, func()) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	emailService := &MockEmailService{}
	handler := NewTicketsHandler(db, emailService)
	
	teardown := func() {
		db.Close()
	}

	return handler, mock, teardown
}

func TestTicketsHandler_ListTickets(t *testing.T) {
	handler, mock, teardown := setupTicketsTest(t)
	defer teardown()

	t.Run("admin can see all tickets", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/tickets", nil)
		rr := httptest.NewRecorder()

		// Set admin user in context
		ctx := context.WithValue(req.Context(), middleware.ContextUserID, 1)
		ctx = context.WithValue(ctx, middleware.ContextRoleID, 1) // Admin
		req = req.WithContext(ctx)

		now := time.Now()
		mock.ExpectQuery(`SELECT`).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "ticket_num", "title", "description", "type", "priority",
				"status", "completion", "created_by", "assigned_to", "asset_id",
				"is_internal", "created_at", "updated_at", "closed_at",
				"creator_username", "creator_full_name",
				"assignee_username", "assignee_full_name",
				"asset_internal_id",
			}).AddRow(
				1, "TCK-2025-0001", "Ticket 1", "Desc 1", "it_help", "normal",
				"open", 0, int64(1), nil, nil,
				false, now, now, nil,
				"admin", "Admin User",
				nil, nil,
				nil,
			))

		handler.ListTickets(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		
		var tickets []models.Ticket
		err := json.Unmarshal(rr.Body.Bytes(), &tickets)
		assert.NoError(t, err)
		assert.Len(t, tickets, 1)
	})

	t.Run("staff can only see their tickets", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/tickets", nil)
		rr := httptest.NewRecorder()

		// Set staff user in context
		ctx := context.WithValue(req.Context(), middleware.ContextUserID, 2)
		ctx = context.WithValue(ctx, middleware.ContextRoleID, 3) // Staff
		req = req.WithContext(ctx)

		now := time.Now()
		mock.ExpectQuery(`SELECT`).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "ticket_num", "title", "description", "type", "priority",
				"status", "completion", "created_by", "assigned_to", "asset_id",
				"is_internal", "created_at", "updated_at", "closed_at",
				"creator_username", "creator_full_name",
				"assignee_username", "assignee_full_name",
				"asset_internal_id",
			}).AddRow(
				1, "TCK-2025-0001", "Ticket 1", "Desc 1", "it_help", "normal",
				"open", 0, int64(2), nil, nil,
				false, now, now, nil,
				"staff", "Staff User",
				nil, nil,
				nil,
			))

		handler.ListTickets(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("unauthorized without user context", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/tickets", nil)
		rr := httptest.NewRecorder()

		handler.ListTickets(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})
}

func TestTicketsHandler_CreateTicket(t *testing.T) {
	handler, mock, teardown := setupTicketsTest(t)
	defer teardown()

	t.Run("successful creation", func(t *testing.T) {
		input := map[string]interface{}{
			"title":       "Test Ticket",
			"description": "Test Description",
			"type":        "it_help",
			"priority":    "normal",
			"is_internal": false,
		}

		body, _ := json.Marshal(input)
		req := httptest.NewRequest("POST", "/api/v1/tickets", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		// Set user in context
		ctx := context.WithValue(req.Context(), middleware.ContextUserID, 1)
		ctx = context.WithValue(ctx, middleware.ContextRoleID, 1)
		req = req.WithContext(ctx)

		// Mock GenerateTicketNum calls
		mock.ExpectQuery(`SELECT MAX`).
			WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow(5))
		mock.ExpectQuery(`SELECT EXISTS`).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		// Mock insert
		now := time.Now()
		mock.ExpectQuery(`INSERT INTO tickets`).
			WithArgs(
				sqlmock.AnyArg(), // ticket_num
				"Test Ticket",
				"Test Description",
				"it_help",
				"normal",
				"open",
				0,
				int64(1),
				nil,
				nil,
				false,
			).
			WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
				AddRow(1, now, now))

		handler.CreateTicket(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)
		
		var ticket models.Ticket
		err := json.Unmarshal(rr.Body.Bytes(), &ticket)
		assert.NoError(t, err)
		assert.Equal(t, "Test Ticket", ticket.Title)
		assert.Equal(t, "Test Description", ticket.Description)
	})

	t.Run("missing required fields", func(t *testing.T) {
		input := map[string]interface{}{
			"description": "Test Description",
			// missing title and type
		}

		body, _ := json.Marshal(input)
		req := httptest.NewRequest("POST", "/api/v1/tickets", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		// Set user in context
		ctx := context.WithValue(req.Context(), middleware.ContextUserID, 1)
		ctx = context.WithValue(ctx, middleware.ContextRoleID, 1)
		req = req.WithContext(ctx)

		handler.CreateTicket(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "Title is required")
	})

	t.Run("invalid asset", func(t *testing.T) {
		input := map[string]interface{}{
			"title":       "Test Ticket",
			"description": "Test Description",
			"type":        "it_help",
			"asset_id":    999,
		}

		body, _ := json.Marshal(input)
		req := httptest.NewRequest("POST", "/api/v1/tickets", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		// Set user in context
		ctx := context.WithValue(req.Context(), middleware.ContextUserID, 1)
		ctx = context.WithValue(ctx, middleware.ContextRoleID, 1)
		req = req.WithContext(ctx)

		// Mock asset check
		mock.ExpectQuery(`SELECT`).
			WithArgs(int64(999)).
			WillReturnError(sql.ErrNoRows)

		handler.CreateTicket(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "Asset not found")
	})
}

func TestTicketsHandler_GetTicket(t *testing.T) {
	handler, mock, teardown := setupTicketsTest(t)
	defer teardown()

	t.Run("successful retrieval", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/tickets/1", nil)
		rr := httptest.NewRecorder()

		now := time.Now()
		mock.ExpectQuery(`SELECT`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "ticket_num", "title", "description", "type", "priority",
				"status", "completion", "created_by", "assigned_to", "asset_id",
				"is_internal", "verification_status", "verification_notes",
				"verified_by", "verified_at", "created_at", "updated_at", "closed_at",
				"creator_id", "creator_username", "creator_full_name", "creator_email",
				"assignee_id", "assignee_username", "assignee_full_name", "assignee_email",
				"verifier_id", "verifier_username", "verifier_full_name", "verifier_email",
				"asset_id", "asset_internal_id", "asset_type", "asset_manufacturer", "asset_model",
			}).AddRow(
				1, "TCK-2025-0001", "Test Ticket", "Test Description", "it_help", "normal",
				"open", 0, int64(1), nil, nil,
				false, "not_required", "", nil, nil, now, now, nil,
				int64(1), "testuser", "Test User", "test@example.com",
				nil, nil, nil, nil,
				nil, nil, nil, nil,
				nil, nil, nil, nil, nil,
			))

		handler.GetTicket(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		
		var ticket models.Ticket
		err := json.Unmarshal(rr.Body.Bytes(), &ticket)
		assert.NoError(t, err)
		assert.Equal(t, "TCK-2025-0001", ticket.TicketNum)
	})

	t.Run("ticket not found", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/tickets/999", nil)
		rr := httptest.NewRecorder()

		mock.ExpectQuery(`SELECT`).
			WithArgs(int64(999)).
			WillReturnError(sql.ErrNoRows)

		handler.GetTicket(rr, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
		assert.Contains(t, rr.Body.String(), "Ticket not found")
	})

	t.Run("invalid ticket ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/tickets/invalid", nil)
		rr := httptest.NewRecorder()

		handler.GetTicket(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "Invalid ticket ID")
	})
}

func TestTicketsHandler_UpdateTicketStatus(t *testing.T) {
	handler, mock, teardown := setupTicketsTest(t)
	defer teardown()

	t.Run("successful status update", func(t *testing.T) {
		input := map[string]interface{}{
			"status":     "in_progress",
			"completion": 50,
		}

		body, _ := json.Marshal(input)
		req := httptest.NewRequest("PUT", "/api/v1/tickets/1/status", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		// Set user in context
		ctx := context.WithValue(req.Context(), middleware.ContextUserID, 1)
		ctx = context.WithValue(ctx, middleware.ContextRoleID, 1)
		req = req.WithContext(ctx)

		// Mock current user info
		mock.ExpectQuery(`SELECT email, username FROM users`).
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{"email", "username"}).
				AddRow("test@example.com", "testuser"))

		// Mock get current ticket
		now := time.Now()
		mock.ExpectQuery(`SELECT`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "ticket_num", "title", "description", "type", "priority",
				"status", "completion", "created_by", "assigned_to", "asset_id",
				"is_internal", "verification_status", "verification_notes",
				"verified_by", "verified_at", "created_at", "updated_at", "closed_at",
				"creator_id", "creator_username", "creator_full_name", "creator_email",
				"assignee_id", "assignee_username", "assignee_full_name", "assignee_email",
				"verifier_id", "verifier_username", "verifier_full_name", "verifier_email",
				"asset_id", "asset_internal_id", "asset_type", "asset_manufacturer", "asset_model",
			}).AddRow(
				1, "TCK-2025-0001", "Test Ticket", "Test Description", "it_help", "normal",
				"open", 0, int64(1), nil, nil,
				false, "not_required", "", nil, nil, now, now, nil,
				int64(1), "testuser", "Test User", "test@example.com",
				nil, nil, nil, nil,
				nil, nil, nil, nil,
				nil, nil, nil, nil, nil,
			))

		// Mock status update
		mock.ExpectExec(`UPDATE tickets`).
			WithArgs("in_progress", 50, nil, int64(1)).
			WillReturnResult(sqlmock.NewResult(0, 1))

		// Mock get updated ticket
		mock.ExpectQuery(`SELECT`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "ticket_num", "title", "description", "type", "priority",
				"status", "completion", "created_by", "assigned_to", "asset_id",
				"is_internal", "verification_status", "verification_notes",
				"verified_by", "verified_at", "created_at", "updated_at", "closed_at",
				"creator_id", "creator_username", "creator_full_name", "creator_email",
				"assignee_id", "assignee_username", "assignee_full_name", "assignee_email",
				"verifier_id", "verifier_username", "verifier_full_name", "verifier_email",
				"asset_id", "asset_internal_id", "asset_type", "asset_manufacturer", "asset_model",
			}).AddRow(
				1, "TCK-2025-0001", "Test Ticket", "Test Description", "it_help", "normal",
				"in_progress", 50, int64(1), nil, nil,
				false, "not_required", "", nil, nil, now, now, nil,
				int64(1), "testuser", "Test User", "test@example.com",
				nil, nil, nil, nil,
				nil, nil, nil, nil,
				nil, nil, nil, nil, nil,
			))

		handler.UpdateTicketStatus(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Ticket status updated successfully", response["message"])
	})
}

func TestTicketsHandler_RequestVerification(t *testing.T) {
	handler, mock, teardown := setupTicketsTest(t)
	defer teardown()

	t.Run("successful verification request", func(t *testing.T) {
		input := map[string]interface{}{
			"notes": "Please verify this ticket",
		}

		body, _ := json.Marshal(input)
		req := httptest.NewRequest("POST", "/api/v1/tickets/1/request-verification", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		// Set user in context
		ctx := context.WithValue(req.Context(), middleware.ContextUserID, 1)
		ctx = context.WithValue(ctx, middleware.ContextRoleID, 1)
		req = req.WithContext(ctx)

		// Mock get existing ticket
		now := time.Now()
		mock.ExpectQuery(`SELECT`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "ticket_num", "title", "description", "type", "priority",
				"status", "completion", "created_by", "assigned_to", "asset_id",
				"is_internal", "verification_status", "verification_notes",
				"verified_by", "verified_at", "created_at", "updated_at", "closed_at",
				"creator_id", "creator_username", "creator_full_name", "creator_email",
				"assignee_id", "assignee_username", "assignee_full_name", "assignee_email",
				"verifier_id", "verifier_username", "verifier_full_name", "verifier_email",
				"asset_id", "asset_internal_id", "asset_type", "asset_manufacturer", "asset_model",
			}).AddRow(
				1, "TCK-2025-0001", "Test Ticket", "Test Description", "it_help", "normal",
				"resolved", 90, int64(1), nil, nil,
				false, "not_required", "", nil, nil, now, now, nil,
				int64(1), "testuser", "Test User", "test@example.com",
				nil, nil, nil, nil,
				nil, nil, nil, nil,
				nil, nil, nil, nil, nil,
			))

		// Mock verification request
		mock.ExpectExec(`UPDATE tickets`).
			WithArgs("Please verify this ticket", int64(1)).
			WillReturnResult(sqlmock.NewResult(0, 1))

		// Mock get updated ticket
		mock.ExpectQuery(`SELECT`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "ticket_num", "title", "description", "type", "priority",
				"status", "completion", "created_by", "assigned_to", "asset_id",
				"is_internal", "verification_status", "verification_notes",
				"verified_by", "verified_at", "created_at", "updated_at", "closed_at",
				"creator_id", "creator_username", "creator_full_name", "creator_email",
				"assignee_id", "assignee_username", "assignee_full_name", "assignee_email",
				"verifier_id", "verifier_username", "verifier_full_name", "verifier_email",
				"asset_id", "asset_internal_id", "asset_type", "asset_manufacturer", "asset_model",
			}).AddRow(
				1, "TCK-2025-0001", "Test Ticket", "Test Description", "it_help", "normal",
				"resolved", 90, int64(1), nil, nil,
				false, "pending", "Please verify this ticket", nil, nil, now, now, nil,
				int64(1), "testuser", "Test User", "test@example.com",
				nil, nil, nil, nil,
				nil, nil, nil, nil,
				nil, nil, nil, nil, nil,
			))

		handler.RequestVerification(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})
}

func TestTicketsHandler_GetTicketStats(t *testing.T) {
	handler, mock, teardown := setupTicketsTest(t)
	defer teardown()

	t.Run("admin stats", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/tickets/stats", nil)
		rr := httptest.NewRecorder()

		// Set admin user in context
		ctx := context.WithValue(req.Context(), middleware.ContextUserID, 1)
		ctx = context.WithValue(ctx, middleware.ContextRoleID, 1) // Admin
		req = req.WithContext(ctx)

		mock.ExpectQuery(`SELECT`).
			WillReturnRows(sqlmock.NewRows([]string{
				"total", "open", "received", "in_progress", "resolved", "closed", "critical",
			}).AddRow(100, 25, 15, 20, 30, 10, 5))

		handler.GetTicketStats(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		
		var stats map[string]interface{}
		err := json.Unmarshal(rr.Body.Bytes(), &stats)
		assert.NoError(t, err)
		assert.Equal(t, float64(100), stats["total"])
		assert.Equal(t, float64(25), stats["open"])
	})
}