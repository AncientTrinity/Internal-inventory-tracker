package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"victortillett.net/internal-inventory-tracker/internal/models"
)

type UsersHandler struct {
	Model *models.UsersModel
}

func NewUsersHandler(db *sql.DB) *UsersHandler {
	return &UsersHandler{
		Model: models.NewUsersModel(db),
	}
}

// GET /api/v1/users
func (h *UsersHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := h.Model.DB.Query(`SELECT id, username, full_name, email, role_id, created_at FROM users ORDER BY id`)
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	users := []models.User{}
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Username, &u.FullName, &u.Email, &u.RoleID, &u.CreatedAt); err != nil {
			http.Error(w, "DB scan error", http.StatusInternalServerError)
			return
		}
		users = append(users, u)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

// GET /api/v1/users/{id}
func (h *UsersHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/users/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var u models.User
	err = h.Model.DB.QueryRow(`SELECT id, username, full_name, email, role_id, created_at FROM users WHERE id=$1`, id).
		Scan(&u.ID, &u.Username, &u.FullName, &u.Email, &u.RoleID, &u.CreatedAt)
	if err == sql.ErrNoRows {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(u)
}

// POST /api/v1/users
func (h *UsersHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Username string `json:"username"`
		FullName string `json:"full_name"`
		Email    string `json:"email"`
		Password string `json:"password"`
		RoleID   int64  `json:"role_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Password hash error", http.StatusInternalServerError)
		return
	}

	u := &models.User{
		Username:     input.Username,
		FullName:     input.FullName,
		Email:        input.Email,
		PasswordHash: string(hash),
		RoleID:       input.RoleID,
	}

	err = h.Model.Insert(u)
	if err != nil {
		http.Error(w, "DB insert error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(u)
}

// PUT /api/v1/users/{id}
func (h *UsersHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/users/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var input struct {
		Username string `json:"username"`
		FullName string `json:"full_name"`
		Email    string `json:"email"`
		RoleID   int64  `json:"role_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	u := &models.User{
		ID:       id,
		Username: input.Username,
		FullName: input.FullName,
		Email:    input.Email,
		RoleID:   input.RoleID,
	}

	err = h.Model.Update(u)
	if err != nil {
		if err.Error() == "user not found" {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		http.Error(w, "DB update error", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(u)
}

// DELETE /api/v1/users/{id}
func (h *UsersHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/users/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	err = h.Model.Delete(id)
	if err != nil {
		if err.Error() == "user not found" {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		http.Error(w, "DB delete error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
