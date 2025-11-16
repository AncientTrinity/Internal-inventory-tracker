package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"victortillett.net/internal-inventory-tracker/internal/middleware"
	"victortillett.net/internal-inventory-tracker/internal/models"
)

type NotificationsHandler struct {
	NotificationModel *models.NotificationModel
}

func NewNotificationsHandler(db *sql.DB) *NotificationsHandler {
	return &NotificationsHandler{
		NotificationModel: models.NewNotificationModel(db),
	}
}

// GET /api/v1/notifications
func (h *NotificationsHandler) GetNotifications(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.ContextUserID).(int)
	if !ok {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	unreadOnly := r.URL.Query().Get("unread") == "true"

	notifications, err := h.NotificationModel.GetByUserID(int64(userID), unreadOnly)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notifications)
}

// GET /api/v1/notifications/unread-count
func (h *NotificationsHandler) GetUnreadCount(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.ContextUserID).(int)
	if !ok {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	count, err := h.NotificationModel.GetUnreadCount(int64(userID))
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"unread_count": count})
}

// GET /api/v1/notifications/types
func (h *NotificationsHandler) GetNotificationTypes(w http.ResponseWriter, r *http.Request) {
	types := []string{
		"ticket_created",
		"ticket_updated", 
		"asset_created",
		"user_created",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(types)
}

// PUT /api/v1/notifications/read-all
func (h *NotificationsHandler) MarkAllAsRead(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.ContextUserID).(int)
	if !ok {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	err := h.NotificationModel.MarkAllAsRead(int64(userID))
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "All notifications marked as read"})
}

// PUT /api/v1/notifications/{id}/read
func (h *NotificationsHandler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.ContextUserID).(int)
	if !ok {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	idStr := r.URL.Path[len("/api/v1/notifications/"):]
	idStr = idStr[:len(idStr)-len("/read")]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid notification ID", http.StatusBadRequest)
		return
	}

	err = h.NotificationModel.MarkAsRead(id, int64(userID))
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Notification not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Notification marked as read"})
}