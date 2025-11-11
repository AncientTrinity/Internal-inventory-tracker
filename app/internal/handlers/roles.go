package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"victortillett.net/internal-inventory-tracker/internal/models"
)

type RolesHandler struct {
	Model *models.RolesModel
}

func NewRolesHandler(db *sql.DB) *RolesHandler {
	return &RolesHandler{
		Model: models.NewRolesModel(db),
	}
}

// GET /api/v1/roles
func (h *RolesHandler) ListRoles(w http.ResponseWriter, r *http.Request) {
	roles, err := h.Model.GetAll()
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(roles)
}

// GET /api/v1/roles/{id}
func (h *RolesHandler) GetRole(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/roles/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	role, err := h.Model.GetByID(id)
	if err != nil {
		http.Error(w, "Role not found", http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(role)
}

// POST /api/v1/roles
func (h *RolesHandler) CreateRole(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	role := &models.Role{Name: input.Name}
	if err := h.Model.Insert(role); err != nil {
		http.Error(w, "DB insert error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(role)
}

// PUT /api/v1/roles/{id}
func (h *RolesHandler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/roles/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var input struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	role := &models.Role{
		ID:   id,
		Name: input.Name,
	}

	if err := h.Model.Update(role); err != nil {
		if err.Error() == "role not found" {
			http.Error(w, "Role not found", http.StatusNotFound)
			return
		}
		http.Error(w, "DB update error", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(role)
}

// DELETE /api/v1/roles/{id}
func (h *RolesHandler) DeleteRole(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/roles/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := h.Model.Delete(id); err != nil {
		if err.Error() == "role not found" {
			http.Error(w, "Role not found", http.StatusNotFound)
			return
		}
		http.Error(w, "DB delete error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
