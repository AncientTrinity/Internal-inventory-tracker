package handlers

import (
	"encoding/json"
	"net/http"

	"victortillett.net/internal-inventory-tracker/internal/middleware"
)

// GET /api/v1/users/me - Get current user profile
func (h *UsersHandler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	// Get current user from context
	userID, ok := r.Context().Value(middleware.ContextUserID).(int)
	if !ok {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	// Get user by ID
	user, err := h.Model.GetByID(int64(userID))
	if err != nil {
		if err.Error() == "user not found" {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Return user data (without password hash)
	response := map[string]interface{}{
		"id":         user.ID,
		"username":   user.Username,
		"full_name":  user.FullName,
		"email":      user.Email,
		"role_id":    user.RoleID,
		"created_at": user.CreatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}