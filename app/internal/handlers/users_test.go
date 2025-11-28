// file: app/internal/handlers/users_test.go
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

// MockUsersEmailService for testing
type MockUsersEmailService struct{}

func (m *MockUsersEmailService) SendTicketAssignedEmail(to, ticketNum, title, assignedBy string) error {
	return nil
}

func (m *MockUsersEmailService) SendTicketStatusUpdateEmail(to, ticketNum, title, oldStatus, newStatus, updatedBy string) error {
	return nil
}

// Add the methods that your UsersHandler expects
func (m *MockUsersEmailService) SendEmailWithRetry(to, subject, body string, maxRetries int) error {
	return nil
}

func (m *MockUsersEmailService) SendBulkEmails(emails []struct {
	To      string
	Subject string
	Body    string
}) error {
	return nil
}

func setupUsersTest(t *testing.T) (*UsersHandler, sqlmock.Sqlmock, func()) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	emailService := &MockUsersEmailService{}
	handler := NewUsersHandler(db, emailService)
	
	teardown := func() {
		db.Close()
	}

	return handler, mock, teardown
}

func TestUsersHandler_ListUsers(t *testing.T) {
	handler, mock, teardown := setupUsersTest(t)
	defer teardown()

	t.Run("successful list users", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/users", nil)
		rr := httptest.NewRecorder()

		// Set admin user in context
		ctx := context.WithValue(req.Context(), middleware.ContextUserID, 1)
		ctx = context.WithValue(ctx, middleware.ContextRoleID, 1) // Admin
		req = req.WithContext(ctx)

		now := time.Now()
		mock.ExpectQuery(`SELECT`).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "username", "full_name", "email", "role_id", "created_at",
			}).AddRow(
				1, "user1", "User One", "user1@example.com", 1, now,
			).AddRow(
				2, "user2", "User Two", "user2@example.com", 2, now,
			))

		handler.ListUsers(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		
		var users []models.User
		err := json.Unmarshal(rr.Body.Bytes(), &users)
		assert.NoError(t, err)
		assert.Len(t, users, 2)
	})

	t.Run("unauthorized access", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/users", nil)
		rr := httptest.NewRecorder()

		handler.ListUsers(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})
}

func TestUsersHandler_GetUser(t *testing.T) {
	handler, mock, teardown := setupUsersTest(t)
	defer teardown()

	t.Run("successful get user", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/users/1", nil)
		rr := httptest.NewRecorder()

		// Set user in context
		ctx := context.WithValue(req.Context(), middleware.ContextUserID, 1)
		ctx = context.WithValue(ctx, middleware.ContextRoleID, 1)
		req = req.WithContext(ctx)

		now := time.Now()
		mock.ExpectQuery(`SELECT`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "username", "full_name", "email", "role_id", "created_at",
			}).AddRow(
				1, "testuser", "Test User", "test@example.com", 1, now,
			))

		handler.GetUser(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		
		var user models.User
		err := json.Unmarshal(rr.Body.Bytes(), &user)
		assert.NoError(t, err)
		assert.Equal(t, "testuser", user.Username)
	})

	t.Run("user not found", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/users/999", nil)
		rr := httptest.NewRecorder()

		// Set user in context
		ctx := context.WithValue(req.Context(), middleware.ContextUserID, 1)
		ctx = context.WithValue(ctx, middleware.ContextRoleID, 1)
		req = req.WithContext(ctx)

		mock.ExpectQuery(`SELECT`).
			WithArgs(int64(999)).
			WillReturnError(sql.ErrNoRows)

		handler.GetUser(rr, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
		assert.Contains(t, rr.Body.String(), "User not found")
	})
}

func TestUsersHandler_CreateUser(t *testing.T) {
	handler, mock, teardown := setupUsersTest(t)
	defer teardown()

	t.Run("successful user creation", func(t *testing.T) {
		input := map[string]interface{}{
			"username":  "newuser",
			"full_name": "New User",
			"email":     "newuser@example.com",
			"password":  "password123",
			"role_id":   1,
		}

		body, _ := json.Marshal(input)
		req := httptest.NewRequest("POST", "/api/v1/users", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		// Set admin user in context
		ctx := context.WithValue(req.Context(), middleware.ContextUserID, 1)
		ctx = context.WithValue(ctx, middleware.ContextRoleID, 1) // Admin
		req = req.WithContext(ctx)

		now := time.Now()
		mock.ExpectQuery(`INSERT INTO users`).
			WithArgs(
				"newuser",
				"New User",
				"newuser@example.com",
				sqlmock.AnyArg(), // password hash
				1,
			).
			WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).
				AddRow(1, now))

		handler.CreateUser(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)
		
		var user models.User
		err := json.Unmarshal(rr.Body.Bytes(), &user)
		assert.NoError(t, err)
		assert.Equal(t, "newuser", user.Username)
		assert.Equal(t, "New User", user.FullName)
	})

	t.Run("missing required fields", func(t *testing.T) {
		input := map[string]interface{}{
			"full_name": "New User",
			// missing username and email
		}

		body, _ := json.Marshal(input)
		req := httptest.NewRequest("POST", "/api/v1/users", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		// Set admin user in context
		ctx := context.WithValue(req.Context(), middleware.ContextUserID, 1)
		ctx = context.WithValue(ctx, middleware.ContextRoleID, 1)
		req = req.WithContext(ctx)

		handler.CreateUser(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}

func TestUsersHandler_UpdateUser(t *testing.T) {
	handler, mock, teardown := setupUsersTest(t)
	defer teardown()

	t.Run("successful user update", func(t *testing.T) {
		input := map[string]interface{}{
			"username":  "updateduser",
			"full_name": "Updated User",
			"email":     "updated@example.com",
			"role_id":   2,
		}

		body, _ := json.Marshal(input)
		req := httptest.NewRequest("PUT", "/api/v1/users/1", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		// Set admin user in context
		ctx := context.WithValue(req.Context(), middleware.ContextUserID, 1)
		ctx = context.WithValue(ctx, middleware.ContextRoleID, 1)
		req = req.WithContext(ctx)

		now := time.Now()
		mock.ExpectQuery(`UPDATE users`).
			WithArgs(
				"updateduser",
				"Updated User",
				"updated@example.com",
				2,
				1,
			).
			WillReturnRows(sqlmock.NewRows([]string{"created_at"}).AddRow(now))

		handler.UpdateUser(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("user not found", func(t *testing.T) {
		input := map[string]interface{}{
			"username": "updateduser",
		}

		body, _ := json.Marshal(input)
		req := httptest.NewRequest("PUT", "/api/v1/users/999", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		// Set admin user in context
		ctx := context.WithValue(req.Context(), middleware.ContextUserID, 1)
		ctx = context.WithValue(ctx, middleware.ContextRoleID, 1)
		req = req.WithContext(ctx)

		mock.ExpectQuery(`UPDATE users`).
			WillReturnError(sql.ErrNoRows)

		handler.UpdateUser(rr, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
		assert.Contains(t, rr.Body.String(), "User not found")
	})
}

func TestUsersHandler_DeleteUser(t *testing.T) {
	handler, mock, teardown := setupUsersTest(t)
	defer teardown()

	t.Run("successful user deletion", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/v1/users/1", nil)
		rr := httptest.NewRecorder()

		// Set admin user in context
		ctx := context.WithValue(req.Context(), middleware.ContextUserID, 1)
		ctx = context.WithValue(ctx, middleware.ContextRoleID, 1)
		req = req.WithContext(ctx)

		mock.ExpectExec(`DELETE FROM users`).
			WithArgs(int64(1)).
			WillReturnResult(sqlmock.NewResult(0, 1))

		handler.DeleteUser(rr, req)

		assert.Equal(t, http.StatusNoContent, rr.Code)
	})

	t.Run("user not found", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/v1/users/999", nil)
		rr := httptest.NewRecorder()

		// Set admin user in context
		ctx := context.WithValue(req.Context(), middleware.ContextUserID, 1)
		ctx = context.WithValue(ctx, middleware.ContextRoleID, 1)
		req = req.WithContext(ctx)

		mock.ExpectExec(`DELETE FROM users`).
			WithArgs(int64(999)).
			WillReturnResult(sqlmock.NewResult(0, 0))

		handler.DeleteUser(rr, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
		assert.Contains(t, rr.Body.String(), "User not found")
	})
}