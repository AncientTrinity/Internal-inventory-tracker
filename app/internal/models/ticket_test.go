// file: app/internal/models/ticket_test.go
package models

import (
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTicketTest(t *testing.T) (*TicketModel, sqlmock.Sqlmock, func()) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	model := NewTicketModel(db)
	
	teardown := func() {
		db.Close()
	}

	return model, mock, teardown
}

func TestTicketModel_GenerateTicketNum(t *testing.T) {
	model, mock, teardown := setupTicketTest(t)
	defer teardown()

	t.Run("successful generation with existing tickets", func(t *testing.T) {
		// Mock max ticket number query
		mock.ExpectQuery(`SELECT MAX`).
			WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow(5))

		// Mock existence check
		mock.ExpectQuery(`SELECT EXISTS`).
			WithArgs(sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		ticketNum, err := model.GenerateTicketNum()
		assert.NoError(t, err)
		assert.Contains(t, ticketNum, "TCK-")
	})

	t.Run("successful generation with no existing tickets", func(t *testing.T) {
		// Mock max ticket number query - no rows
		mock.ExpectQuery(`SELECT MAX`).
			WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow(nil))

		// Mock existence check
		mock.ExpectQuery(`SELECT EXISTS`).
			WithArgs(sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		ticketNum, err := model.GenerateTicketNum()
		assert.NoError(t, err)
		assert.Contains(t, ticketNum, "TCK-")
	})

	t.Run("handles duplicate ticket number", func(t *testing.T) {
		// Mock max ticket number query
		mock.ExpectQuery(`SELECT MAX`).
			WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow(5))

		// Mock existence check - first number exists
		mock.ExpectQuery(`SELECT EXISTS`).
			WithArgs(sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		ticketNum, err := model.GenerateTicketNum()
		assert.NoError(t, err)
		assert.Contains(t, ticketNum, "TCK-")
	})
}

func TestTicketModel_Insert(t *testing.T) {
	model, mock, teardown := setupTicketTest(t)
	defer teardown()

	now := time.Now()
	userID := int64(1)
	ticket := &Ticket{
		Title:       "Test Ticket",
		Description: "Test Description",
		Type:        "it_help",
		Priority:    "normal",
		Status:      "open",
		Completion:  0,
		CreatedBy:   &userID,
		IsInternal:  false,
	}

	t.Run("successful insert", func(t *testing.T) {
		// Mock GenerateTicketNum calls
		mock.ExpectQuery(`SELECT MAX`).
			WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow(5))
		mock.ExpectQuery(`SELECT EXISTS`).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		// Mock insert
		mock.ExpectQuery(`INSERT INTO tickets`).
			WithArgs(
				sqlmock.AnyArg(), // ticket_num
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
			).
			WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
				AddRow(1, now, now))

		err := model.Insert(ticket)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), ticket.ID)
		assert.Equal(t, now, ticket.CreatedAt)
		assert.Equal(t, now, ticket.UpdatedAt)
		assert.NotEmpty(t, ticket.TicketNum)
	})

	t.Run("database error", func(t *testing.T) {
		// Mock GenerateTicketNum calls
		mock.ExpectQuery(`SELECT MAX`).
			WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow(5))
		mock.ExpectQuery(`SELECT EXISTS`).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		mock.ExpectQuery(`INSERT INTO tickets`).
			WillReturnError(assert.AnError)

		err := model.Insert(ticket)
		assert.Error(t, err)
	})
}

func TestTicketModel_GetByID(t *testing.T) {
	model, mock, teardown := setupTicketTest(t)
	defer teardown()

	now := time.Now()
	userID := int64(1)

	t.Run("successful retrieval", func(t *testing.T) {
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
				"open", 0, userID, nil, nil,
				false, "not_required", "", nil, nil, now, now, nil,
				userID, "testuser", "Test User", "test@example.com",
				nil, nil, nil, nil,
				nil, nil, nil, nil,
				nil, nil, nil, nil, nil,
			))

		ticket, err := model.GetByID(1)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), ticket.ID)
		assert.Equal(t, "TCK-2025-0001", ticket.TicketNum)
		assert.Equal(t, "Test Ticket", ticket.Title)
		assert.NotNil(t, ticket.CreatedByUser)
		assert.Equal(t, "testuser", ticket.CreatedByUser.Username)
	})

	t.Run("ticket not found", func(t *testing.T) {
		mock.ExpectQuery(`SELECT`).
			WithArgs(int64(999)).
			WillReturnError(sql.ErrNoRows)

		ticket, err := model.GetByID(999)
		assert.Nil(t, ticket)
		assert.Error(t, err)
		assert.Equal(t, "ticket not found", err.Error())
	})

	t.Run("with assigned user and asset", func(t *testing.T) {
		assigneeID := int64(2)
		assetID := int64(1)
		
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
				"open", 0, userID, assigneeID, assetID,
				false, "not_required", "", nil, nil, now, now, nil,
				userID, "testuser", "Test User", "test@example.com",
				assigneeID, "itstaff", "IT Staff", "it@example.com",
				nil, nil, nil, nil,
				assetID, "DPA-PC001", "PC", "Dell", "OptiPlex 7070",
			))

		ticket, err := model.GetByID(1)
		assert.NoError(t, err)
		assert.NotNil(t, ticket.AssignedToUser)
		assert.Equal(t, "itstaff", ticket.AssignedToUser.Username)
		assert.NotNil(t, ticket.Asset)
		assert.Equal(t, "DPA-PC001", ticket.Asset.InternalID)
	})
}

func TestTicketModel_GetAll(t *testing.T) {
	model, mock, teardown := setupTicketTest(t)
	defer teardown()

	now := time.Now()
	userID := int64(1)

	t.Run("get all tickets", func(t *testing.T) {
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
				"open", 0, userID, nil, nil,
				false, now, now, nil,
				"user1", "User One",
				nil, nil,
				nil,
			).AddRow(
				2, "TCK-2025-0002", "Ticket 2", "Desc 2", "activation", "high",
				"in_progress", 50, userID, nil, nil,
				false, now, now, nil,
				"user1", "User One",
				nil, nil,
				nil,
			))

		filters := TicketFilters{}
		tickets, err := model.GetAll(filters)
		assert.NoError(t, err)
		assert.Len(t, tickets, 2)
		assert.Equal(t, "TCK-2025-0001", tickets[0].TicketNum)
		assert.Equal(t, "TCK-2025-0002", tickets[1].TicketNum)
	})

	t.Run("get tickets with filters", func(t *testing.T) {
		assignedTo := int64(2)
		filters := TicketFilters{
			Status:     "open",
			Type:       "it_help",
			Priority:   "normal",
			AssignedTo: &assignedTo,
		}

		mock.ExpectQuery(`SELECT.*status = \$1.*type = \$2.*priority = \$3.*assigned_to = \$4`).
			WithArgs("open", "it_help", "normal", assignedTo).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "ticket_num", "title", "description", "type", "priority",
				"status", "completion", "created_by", "assigned_to", "asset_id",
				"is_internal", "created_at", "updated_at", "closed_at",
				"creator_username", "creator_full_name",
				"assignee_username", "assignee_full_name",
				"asset_internal_id",
			}).AddRow(
				1, "TCK-2025-0001", "Ticket 1", "Desc 1", "it_help", "normal",
				"open", 0, userID, &assignedTo, nil,
				false, now, now, nil,
				"user1", "User One",
				"itstaff", "IT Staff",
				nil,
			))

		tickets, err := model.GetAll(filters)
		assert.NoError(t, err)
		assert.Len(t, tickets, 1)
		assert.Equal(t, "open", tickets[0].Status)
		assert.Equal(t, "it_help", tickets[0].Type)
		assert.Equal(t, &assignedTo, tickets[0].AssignedTo)
	})
}

func TestTicketModel_Update(t *testing.T) {
	model, mock, teardown := setupTicketTest(t)
	defer teardown()

	now := time.Now()
	ticket := &Ticket{
		ID:          1,
		Title:       "Updated Title",
		Description: "Updated Description",
		Type:        "activation",
		Priority:    "high",
		Status:      "in_progress",
		Completion:  50,
		IsInternal:  true,
	}

	t.Run("successful update", func(t *testing.T) {
		mock.ExpectQuery(`UPDATE tickets`).
			WithArgs(
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
			).
			WillReturnRows(sqlmock.NewRows([]string{"updated_at", "closed_at"}).
				AddRow(now, nil))

		err := model.Update(ticket)
		assert.NoError(t, err)
		assert.Equal(t, now, ticket.UpdatedAt)
	})

	t.Run("ticket not found", func(t *testing.T) {
		mock.ExpectQuery(`UPDATE tickets`).
			WillReturnError(sql.ErrNoRows)

		err := model.Update(ticket)
		assert.Error(t, err)
		assert.Equal(t, "ticket not found", err.Error())
	})
}

func TestTicketModel_UpdateStatus(t *testing.T) {
	model, mock, teardown := setupTicketTest(t)
	defer teardown()

	t.Run("successful status update", func(t *testing.T) {
		assignedTo := int64(2)
		mock.ExpectExec(`UPDATE tickets`).
			WithArgs("in_progress", 50, &assignedTo, int64(1)).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := model.UpdateStatus(1, "in_progress", 50, &assignedTo)
		assert.NoError(t, err)
	})

	t.Run("ticket not found", func(t *testing.T) {
		mock.ExpectExec(`UPDATE tickets`).
			WithArgs("closed", 100, nil, int64(999)).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := model.UpdateStatus(999, "closed", 100, nil)
		assert.Error(t, err)
		assert.Equal(t, "ticket not found", err.Error())
	})
}

func TestTicketModel_ReassignTicket(t *testing.T) {
	model, mock, teardown := setupTicketTest(t)
	defer teardown()

	t.Run("successful reassignment", func(t *testing.T) {
		mock.ExpectExec(`UPDATE tickets`).
			WithArgs(int64(2), int64(1)).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := model.ReassignTicket(1, 2)
		assert.NoError(t, err)
	})

	t.Run("ticket not found", func(t *testing.T) {
		mock.ExpectExec(`UPDATE tickets`).
			WithArgs(int64(2), int64(999)).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := model.ReassignTicket(999, 2)
		assert.Error(t, err)
		assert.Equal(t, "ticket not found", err.Error())
	})
}

func TestTicketModel_Delete(t *testing.T) {
	model, mock, teardown := setupTicketTest(t)
	defer teardown()

	t.Run("successful deletion", func(t *testing.T) {
		// Mock comments deletion
		mock.ExpectExec(`DELETE FROM ticket_comments`).
			WithArgs(int64(1)).
			WillReturnResult(sqlmock.NewResult(0, 0))

		// Mock ticket deletion
		mock.ExpectExec(`DELETE FROM tickets`).
			WithArgs(int64(1)).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := model.Delete(1)
		assert.NoError(t, err)
	})

	t.Run("ticket not found", func(t *testing.T) {
		mock.ExpectExec(`DELETE FROM ticket_comments`).
			WithArgs(int64(999)).
			WillReturnResult(sqlmock.NewResult(0, 0))

		mock.ExpectExec(`DELETE FROM tickets`).
			WithArgs(int64(999)).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := model.Delete(999)
		assert.Error(t, err)
		assert.Equal(t, "ticket not found", err.Error())
	})
}

func TestTicketModel_RequestVerification(t *testing.T) {
	model, mock, teardown := setupTicketTest(t)
	defer teardown()

	t.Run("successful verification request", func(t *testing.T) {
		mock.ExpectExec(`UPDATE tickets`).
			WithArgs("Test notes", int64(1)).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := model.RequestVerification(1, "Test notes")
		assert.NoError(t, err)
	})

	t.Run("ticket not found", func(t *testing.T) {
		mock.ExpectExec(`UPDATE tickets`).
			WithArgs("Test notes", int64(999)).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := model.RequestVerification(999, "Test notes")
		assert.Error(t, err)
		assert.Equal(t, "ticket not found", err.Error())
	})
}

func TestTicketModel_VerifyTicket(t *testing.T) {
	model, mock, teardown := setupTicketTest(t)
	defer teardown()

	t.Run("successful verification approval", func(t *testing.T) {
		mock.ExpectExec(`UPDATE tickets`).
			WithArgs("verified", "Approved", int64(1), true, int64(1)).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := model.VerifyTicket(1, 1, true, "Approved", 1)
		assert.NoError(t, err)
	})

	t.Run("successful verification rejection", func(t *testing.T) {
		mock.ExpectExec(`UPDATE tickets`).
			WithArgs("rejected", "Rejected", int64(1), false, int64(1)).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := model.VerifyTicket(1, 1, false, "Rejected", 1)
		assert.NoError(t, err)
	})
}

func TestTicketModel_CanVerifyTicket(t *testing.T) {
	model, mock, teardown := setupTicketTest(t)
	defer teardown()

	t.Run("admin can verify any ticket", func(t *testing.T) {
		mock.ExpectQuery(`SELECT created_by FROM tickets`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"created_by"}).AddRow(int64(2)))

		canVerify, err := model.CanVerifyTicket(1, 1, 1) // Admin role
		assert.NoError(t, err)
		assert.True(t, canVerify)
	})

	t.Run("creator can verify own ticket", func(t *testing.T) {
		mock.ExpectQuery(`SELECT created_by FROM tickets`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"created_by"}).AddRow(int64(1)))

		canVerify, err := model.CanVerifyTicket(1, 1, 3) // Regular user role
		assert.NoError(t, err)
		assert.True(t, canVerify)
	})

	t.Run("user cannot verify others ticket", func(t *testing.T) {
		mock.ExpectQuery(`SELECT created_by FROM tickets`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"created_by"}).AddRow(int64(2)))

		canVerify, err := model.CanVerifyTicket(1, 1, 3) // Regular user role
		assert.NoError(t, err)
		assert.False(t, canVerify)
	})
}

func TestTicketModel_ResetVerification(t *testing.T) {
	model, mock, teardown := setupTicketTest(t)
	defer teardown()

	t.Run("successful reset", func(t *testing.T) {
		mock.ExpectExec(`UPDATE tickets`).
			WithArgs("Verification reset by user", int64(1)).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := model.ResetVerification(1, 1)
		assert.NoError(t, err)
	})

	t.Run("database error", func(t *testing.T) {
		mock.ExpectExec(`UPDATE tickets`).
			WithArgs("Verification reset by user", int64(1)).
			WillReturnError(assert.AnError)

		err := model.ResetVerification(1, 1)
		assert.Error(t, err)
	})
}