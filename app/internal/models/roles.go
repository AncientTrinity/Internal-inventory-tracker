package models

import (
	"database/sql"
	"errors"
	"time"
)

type Role struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type RolesModel struct {
	DB *sql.DB
}

func NewRolesModel(db *sql.DB) *RolesModel {
	return &RolesModel{DB: db}
}

// Insert a new role
func (m *RolesModel) Insert(r *Role) error {
	query := `
		INSERT INTO roles (name)
		VALUES ($1)
		RETURNING id, created_at
	`
	return m.DB.QueryRow(query, r.Name).Scan(&r.ID, &r.CreatedAt)
}

// Update an existing role
func (m *RolesModel) Update(r *Role) error {
	query := `
		UPDATE roles
		SET name=$1
		WHERE id=$2
		RETURNING created_at
	`
	err := m.DB.QueryRow(query, r.Name, r.ID).Scan(&r.CreatedAt)
	if err == sql.ErrNoRows {
		return errors.New("role not found")
	}
	return err
}

// Delete a role
func (m *RolesModel) Delete(id int64) error {
	res, err := m.DB.Exec(`DELETE FROM roles WHERE id=$1`, id)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return errors.New("role not found")
	}
	return nil
}

// Get all roles
func (m *RolesModel) GetAll() ([]Role, error) {
	rows, err := m.DB.Query(`SELECT id, name, created_at FROM roles ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []Role
	for rows.Next() {
		var r Role
		if err := rows.Scan(&r.ID, &r.Name, &r.CreatedAt); err != nil {
			return nil, err
		}
		roles = append(roles, r)
	}
	return roles, nil
}

// Get role by ID
func (m *RolesModel) GetByID(id int64) (*Role, error) {
	var r Role
	err := m.DB.QueryRow(`SELECT id, name, created_at FROM roles WHERE id=$1`, id).
		Scan(&r.ID, &r.Name, &r.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, errors.New("role not found")
	} else if err != nil {
		return nil, err
	}
	return &r, nil
}
