package models

import (
	"database/sql"
	"time"
)

type Notification struct {
	ID         int64          `json:"id"`
	UserID     int64          `json:"user_id"`
	Title      string         `json:"title"`
	Message    string         `json:"message"`
	Type       string         `json:"type"` // ticket_created, ticket_updated, asset_created, etc.
	RelatedID  *int64         `json:"related_id"`
	RelatedType *string       `json:"related_type"` // ticket, asset, user
	IsRead     bool           `json:"is_read"`
	CreatedAt  time.Time      `json:"created_at"`
	
	// Joined fields
	User *User `json:"user,omitempty"`
}

type NotificationModel struct {
	DB *sql.DB
}

func NewNotificationModel(db *sql.DB) *NotificationModel {
	return &NotificationModel{DB: db}
}

// Create a new notification
func (m *NotificationModel) Create(notification *Notification) error {
	query := `
		INSERT INTO notifications (user_id, title, message, type, related_id, related_type, is_read)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at
	`
	
	err := m.DB.QueryRow(
		query,
		notification.UserID,
		notification.Title,
		notification.Message,
		notification.Type,
		notification.RelatedID,
		notification.RelatedType,
		notification.IsRead,
	).Scan(&notification.ID, &notification.CreatedAt)
	
	return err
}

// Create multiple notifications in bulk
func (m *NotificationModel) CreateBulk(notifications []Notification) error {
	tx, err := m.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	
	stmt, err := tx.Prepare(`
		INSERT INTO notifications (user_id, title, message, type, related_id, related_type, is_read)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	
	for _, notification := range notifications {
		_, err := stmt.Exec(
			notification.UserID,
			notification.Title,
			notification.Message,
			notification.Type,
			notification.RelatedID,
			notification.RelatedType,
			notification.IsRead,
		)
		if err != nil {
			return err
		}
	}
	
	return tx.Commit()
}

// Get notifications for a user
func (m *NotificationModel) GetByUserID(userID int64, unreadOnly bool) ([]Notification, error) {
	query := `
		SELECT 
			n.id, n.user_id, n.title, n.message, n.type, n.related_id, 
			n.related_type, n.is_read, n.created_at,
			u.username, u.full_name
		FROM notifications n
		LEFT JOIN users u ON n.user_id = u.id
		WHERE n.user_id = $1
	`
	
	if unreadOnly {
		query += " AND n.is_read = false"
	}
	
	query += " ORDER BY n.created_at DESC"
	
	rows, err := m.DB.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var notifications []Notification
	for rows.Next() {
		var notification Notification
		var username, fullName sql.NullString
		
		err := rows.Scan(
			&notification.ID,
			&notification.UserID,
			&notification.Title,
			&notification.Message,
			&notification.Type,
			&notification.RelatedID,
			&notification.RelatedType,
			&notification.IsRead,
			&notification.CreatedAt,
			&username, &fullName,
		)
		if err != nil {
			return nil, err
		}
		
		if username.Valid {
			notification.User = &User{
				Username: username.String,
				FullName: fullName.String,
			}
		}
		
		notifications = append(notifications, notification)
	}
	
	return notifications, nil
}

// Mark notification as read
func (m *NotificationModel) MarkAsRead(notificationID, userID int64) error {
	query := `
		UPDATE notifications 
		SET is_read = true 
		WHERE id = $1 AND user_id = $2
	`
	
	result, err := m.DB.Exec(query, notificationID, userID)
	if err != nil {
		return err
	}
	
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	
	return nil
}

// Mark all notifications as read for a user
func (m *NotificationModel) MarkAllAsRead(userID int64) error {
	query := `UPDATE notifications SET is_read = true WHERE user_id = $1`
	_, err := m.DB.Exec(query, userID)
	return err
}

// Get unread count for a user
func (m *NotificationModel) GetUnreadCount(userID int64) (int, error) {
	var count int
	err := m.DB.QueryRow(
		"SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = false",
		userID,
	).Scan(&count)
	return count, err
}