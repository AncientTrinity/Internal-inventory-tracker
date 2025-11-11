package models

import (
	"database/sql"
	"errors"
	"time"
)

type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	FullName     string    `json:"full_name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	RoleID       int64     `json:"role_id"`
	CreatedAt    time.Time `json:"created_at"`
}

type UsersModel struct {
	DB *sql.DB
}

func NewUsersModel(db *sql.DB) *UsersModel {
	return &UsersModel{DB: db}
}

func (m *UsersModel) Insert(u *User) error {
	query := `
		INSERT INTO users (username, full_name, email, password_hash, role_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`
	return m.DB.QueryRow(query, u.Username, u.FullName, u.Email, u.PasswordHash, u.RoleID).
		Scan(&u.ID, &u.CreatedAt)
}

func (m *UsersModel) Update(u *User) error {
	query := `
		UPDATE users
		SET username=$1, full_name=$2, email=$3, role_id=$4
		WHERE id=$5
		RETURNING created_at
	`
	err := m.DB.QueryRow(query, u.Username, u.FullName, u.Email, u.RoleID, u.ID).Scan(&u.CreatedAt)
	if err == sql.ErrNoRows {
		return errors.New("user not found")
	}
	return err
}

func (m *UsersModel) Delete(id int64) error {
	res, err := m.DB.Exec(`DELETE FROM users WHERE id=$1`, id)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return errors.New("user not found")
	}
	return nil
}
