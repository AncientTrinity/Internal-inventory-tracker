// file: app/internal/models/users_test.go
package models

import (
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupUserTest(t *testing.T) (*UsersModel, sqlmock.Sqlmock, func()) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	model := NewUsersModel(db)
	
	teardown := func() {
		db.Close()
	}

	return model, mock, teardown
}

func TestUsersModel_Insert(t *testing.T) {
	model, mock, teardown := setupUserTest(t)
	defer teardown()

	now := time.Now()
	user := &User{
		Username:     "testuser",
		FullName:     "Test User",
		Email:        "test@example.com",
		PasswordHash: "hashed_password",
		RoleID:       1,
	}

	t.Run("successful insert", func(t *testing.T) {
		mock.ExpectQuery(`INSERT INTO users`).
			WithArgs(
				user.Username,
				user.FullName,
				user.Email,
				user.PasswordHash,
				user.RoleID,
			).
			WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).
				AddRow(1, now))

		err := model.Insert(user)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), user.ID)
		assert.Equal(t, now, user.CreatedAt)
	})

	t.Run("database error", func(t *testing.T) {
		mock.ExpectQuery(`INSERT INTO users`).
			WillReturnError(assert.AnError)

		err := model.Insert(user)
		assert.Error(t, err)
	})
}

func TestUsersModel_Update(t *testing.T) {
	model, mock, teardown := setupUserTest(t)
	defer teardown()

	now := time.Now()
	user := &User{
		ID:       1,
		Username: "updateduser",
		FullName: "Updated User",
		Email:    "updated@example.com",
		RoleID:   2,
	}

	t.Run("successful update", func(t *testing.T) {
		mock.ExpectQuery(`UPDATE users`).
			WithArgs(
				user.Username,
				user.FullName,
				user.Email,
				user.RoleID,
				user.ID,
			).
			WillReturnRows(sqlmock.NewRows([]string{"created_at"}).AddRow(now))

		err := model.Update(user)
		assert.NoError(t, err)
		assert.Equal(t, now, user.CreatedAt)
	})

	t.Run("user not found", func(t *testing.T) {
		mock.ExpectQuery(`UPDATE users`).
			WillReturnError(sql.ErrNoRows)

		err := model.Update(user)
		assert.Error(t, err)
		assert.Equal(t, "user not found", err.Error())
	})

	t.Run("database error", func(t *testing.T) {
		mock.ExpectQuery(`UPDATE users`).
			WillReturnError(assert.AnError)

		err := model.Update(user)
		assert.Error(t, err)
	})
}

func TestUsersModel_Delete(t *testing.T) {
	model, mock, teardown := setupUserTest(t)
	defer teardown()

	t.Run("successful deletion", func(t *testing.T) {
		mock.ExpectExec(`DELETE FROM users`).
			WithArgs(int64(1)).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := model.Delete(1)
		assert.NoError(t, err)
	})

	t.Run("user not found", func(t *testing.T) {
		mock.ExpectExec(`DELETE FROM users`).
			WithArgs(int64(999)).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := model.Delete(999)
		assert.Error(t, err)
		assert.Equal(t, "user not found", err.Error())
	})

	t.Run("database error", func(t *testing.T) {
		mock.ExpectExec(`DELETE FROM users`).
			WithArgs(int64(1)).
			WillReturnError(assert.AnError)

		err := model.Delete(1)
		assert.Error(t, err)
	})
}

func TestUsersModel_GetAll(t *testing.T) {
	model, mock, teardown := setupUserTest(t)
	defer teardown()

	now := time.Now()

	t.Run("successful get all users", func(t *testing.T) {
		mock.ExpectQuery(`SELECT`).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "username", "full_name", "email", "role_id", "created_at",
			}).AddRow(
				1, "user1", "User One", "user1@example.com", 1, now,
			).AddRow(
				2, "user2", "User Two", "user2@example.com", 2, now,
			))

		users, err := model.GetAll()
		assert.NoError(t, err)
		assert.Len(t, users, 2)
		assert.Equal(t, "user1", users[0].Username)
		assert.Equal(t, "user2", users[1].Username)
	})

	t.Run("no users found", func(t *testing.T) {
		mock.ExpectQuery(`SELECT`).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "username", "full_name", "email", "role_id", "created_at",
			}))

		users, err := model.GetAll()
		assert.NoError(t, err)
		assert.Len(t, users, 0)
	})

	t.Run("database error", func(t *testing.T) {
		mock.ExpectQuery(`SELECT`).
			WillReturnError(assert.AnError)

		users, err := model.GetAll()
		assert.Error(t, err)
		assert.Nil(t, users)
	})
}

func TestUsersModel_GetByID(t *testing.T) {
	model, mock, teardown := setupUserTest(t)
	defer teardown()

	now := time.Now()

	t.Run("successful get by ID", func(t *testing.T) {
		mock.ExpectQuery(`SELECT`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "username", "full_name", "email", "role_id", "created_at",
			}).AddRow(
				1, "testuser", "Test User", "test@example.com", 1, now,
			))

		user, err := model.GetByID(1)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), user.ID)
		assert.Equal(t, "testuser", user.Username)
		assert.Equal(t, "Test User", user.FullName)
		assert.Equal(t, "test@example.com", user.Email)
		assert.Equal(t, int64(1), user.RoleID)
		assert.Equal(t, now, user.CreatedAt)
	})

	t.Run("user not found", func(t *testing.T) {
		mock.ExpectQuery(`SELECT`).
			WithArgs(int64(999)).
			WillReturnError(sql.ErrNoRows)

		user, err := model.GetByID(999)
		assert.Nil(t, user)
		assert.Error(t, err)
		assert.Equal(t, "user not found", err.Error())
	})

	t.Run("database error", func(t *testing.T) {
		mock.ExpectQuery(`SELECT`).
			WithArgs(int64(1)).
			WillReturnError(assert.AnError)

		user, err := model.GetByID(1)
		assert.Nil(t, user)
		assert.Error(t, err)
	})
}

func TestUsersModel_ResetPassword(t *testing.T) {
	model, mock, teardown := setupUserTest(t)
	defer teardown()

	t.Run("successful password reset", func(t *testing.T) {
		mock.ExpectExec(`UPDATE users SET password_hash`).
			WithArgs(sqlmock.AnyArg(), int64(1)).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := model.ResetPassword(1)
		assert.NoError(t, err)
	})

	t.Run("user not found", func(t *testing.T) {
		mock.ExpectExec(`UPDATE users SET password_hash`).
			WithArgs(sqlmock.AnyArg(), int64(999)).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := model.ResetPassword(999)
		assert.NoError(t, err) // Note: ResetPassword doesn't return error for not found users
	})

	t.Run("database error", func(t *testing.T) {
		mock.ExpectExec(`UPDATE users SET password_hash`).
			WithArgs(sqlmock.AnyArg(), int64(1)).
			WillReturnError(assert.AnError)

		err := model.ResetPassword(1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update password")
	})
}