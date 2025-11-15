package services

import (
	"database/sql"
	"fmt"

	"victortillett.net/internal-inventory-tracker/internal/models"
)

type NotificationService struct {
	NotificationModel *models.NotificationModel
	UserModel         *models.UsersModel
}

func NewNotificationService(db *sql.DB) *NotificationService {
	return &NotificationService{
		NotificationModel: models.NewNotificationModel(db),
		UserModel:         models.NewUsersModel(db),
	}
}

// NotifyTicketCreated sends notifications when a ticket is created
func (s *NotificationService) NotifyTicketCreated(ticket *models.Ticket) error {
	// Get all users who should be notified
	users, err := s.getUsersForTicketNotifications()
	if err != nil {
		return err
	}

	var notifications []models.Notification
	ticketID := ticket.ID

	for _, user := range users {
		notification := models.Notification{
			UserID:      user.ID,
			Title:       "New Ticket Created",
			Message:     fmt.Sprintf("Ticket #%s: %s", ticket.TicketNum, ticket.Title),
			Type:        "ticket_created",
			RelatedID:   &ticketID,
			RelatedType: stringPtr("ticket"),
			IsRead:      false,
		}
		notifications = append(notifications, notification)
	}

	return s.NotificationModel.CreateBulk(notifications)
}

// NotifyTicketUpdated sends notifications when a ticket is updated
func (s *NotificationService) NotifyTicketUpdated(ticket *models.Ticket, updaterUserID int64, action string) error {
	users, err := s.getUsersForTicketNotifications()
	if err != nil {
		return err
	}

	var notifications []models.Notification
	ticketID := ticket.ID

	for _, user := range users {
		// Don't notify the user who made the update
		if user.ID == updaterUserID {
			continue
		}

		notification := models.Notification{
			UserID:      user.ID,
			Title:       fmt.Sprintf("Ticket %s", action),
			Message:     fmt.Sprintf("Ticket #%s: %s - %s", ticket.TicketNum, ticket.Title, action),
			Type:        "ticket_updated",
			RelatedID:   &ticketID,
			RelatedType: stringPtr("ticket"),
			IsRead:      false,
		}
		notifications = append(notifications, notification)
	}

	return s.NotificationModel.CreateBulk(notifications)
}

// NotifyAssetCreated sends notifications when an asset is created
func (s *NotificationService) NotifyAssetCreated(asset *models.Asset) error {
	// Only notify admins and IT staff for assets
	users, err := s.getUsersForAssetNotifications()
	if err != nil {
		return err
	}

	var notifications []models.Notification
	assetID := asset.ID

	for _, user := range users {
		notification := models.Notification{
			UserID:      user.ID,
			Title:       "New Asset Added",
			Message:     fmt.Sprintf("Asset %s: %s %s", asset.InternalID, asset.Manufacturer, asset.Model),
			Type:        "asset_created",
			RelatedID:   &assetID,
			RelatedType: stringPtr("asset"),
			IsRead:      false,
		}
		notifications = append(notifications, notification)
	}

	return s.NotificationModel.CreateBulk(notifications)
}

// Get users who should receive ticket notifications (Admin, IT, Staff, Agent)
func (s *NotificationService) getUsersForTicketNotifications() ([]models.User, error) {
	query := `
		SELECT id, username, full_name, email, role_id 
		FROM users 
		WHERE role_id IN (1, 2, 3, 4) -- Admin, IT, Staff, Agent
		AND is_active = true
	`
	
	rows, err := s.UserModel.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var users []models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.FullName,
			&user.Email,
			&user.RoleID,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	
	return users, nil
}

// Get users who should receive asset notifications (Admin, IT only)
func (s *NotificationService) getUsersForAssetNotifications() ([]models.User, error) {
	query := `
		SELECT id, username, full_name, email, role_id 
		FROM users 
		WHERE role_id IN (1, 2) -- Admin, IT only
		AND is_active = true
	`
	
	rows, err := s.UserModel.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var users []models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.FullName,
			&user.Email,
			&user.RoleID,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	
	return users, nil
}

// Helper function
func stringPtr(s string) *string {
	return &s
}