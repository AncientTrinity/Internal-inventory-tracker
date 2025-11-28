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

// NotifyVerificationRequested sends notifications when verification is requested
func (s *NotificationService) NotifyVerificationRequested(ticket *models.Ticket) error {
	// Notify ticket creator and assigned IT staff (if any)
	users, err := s.getUsersForVerificationNotifications(ticket)
	if err != nil {
		return err
	}

	var notifications []models.Notification
	ticketID := ticket.ID

	for _, user := range users {
		notification := models.Notification{
			UserID:      user.ID,
			Title:       "Verification Requested",
			Message:     fmt.Sprintf("Ticket #%s is ready for verification: %s", ticket.TicketNum, ticket.Title),
			Type:        "verification_requested",
			RelatedID:   &ticketID,
			RelatedType: stringPtr("ticket"),
			IsRead:      false,
		}
		notifications = append(notifications, notification)
	}

	return s.NotificationModel.CreateBulk(notifications)
}

// NotifyVerificationCompleted sends notifications when verification is completed
func (s *NotificationService) NotifyVerificationCompleted(ticket *models.Ticket, approved bool) error {
	// Notify relevant users about verification result
	users, err := s.getUsersForVerificationNotifications(ticket)
	if err != nil {
		return err
	}

	var notifications []models.Notification
	ticketID := ticket.ID

	action := "approved"
	if !approved {
		action = "rejected"
	}

	for _, user := range users {
		notification := models.Notification{
			UserID:      user.ID,
			Title:       fmt.Sprintf("Verification %s", action),
			Message:     fmt.Sprintf("Ticket #%s verification %s: %s", ticket.TicketNum, action, ticket.Title),
			Type:        "verification_completed",
			RelatedID:   &ticketID,
			RelatedType: stringPtr("ticket"),
			IsRead:      false,
		}
		notifications = append(notifications, notification)
	}

	return s.NotificationModel.CreateBulk(notifications)
}

// Get users who should receive verification notifications
func (s *NotificationService) getUsersForVerificationNotifications(ticket *models.Ticket) ([]models.User, error) {
	var users []models.User

	// Always notify the ticket creator
	if ticket.CreatedBy != nil {
		creator, err := s.UserModel.GetByID(*ticket.CreatedBy)
		if err == nil {
			users = append(users, *creator)
		}
	}

	// Notify assigned IT staff if different from creator
	if ticket.AssignedTo != nil && 
	   (ticket.CreatedBy == nil || *ticket.AssignedTo != *ticket.CreatedBy) {
		assignee, err := s.UserModel.GetByID(*ticket.AssignedTo)
		if err == nil {
			users = append(users, *assignee)
		}
	}

	// Also notify all IT staff for visibility
	itStaff, err := s.getITStaffUsers()
	if err == nil {
		for _, staff := range itStaff {
			// Avoid duplicates
			if !s.containsUser(users, staff.ID) {
				users = append(users, staff)
			}
		}
	}

	return users, nil
}

// Get IT staff users
func (s *NotificationService) getITStaffUsers() ([]models.User, error) {
	query := `
		SELECT id, username, full_name, email, role_id 
		FROM users 
		WHERE role_id IN (1, 2) -- Admin, IT Staff
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

// Helper to check if user already exists in slice
func (s *NotificationService) containsUser(users []models.User, userID int64) bool {
	for _, user := range users {
		if user.ID == userID {
			return true
		}
	}
	return false
}

// NotifyVerificationSetup - Send notifications when verification is set up
func (s *NotificationService) NotifyVerificationSetup(ticket *models.Ticket, setupBy int64) error {
	// Get the user who set up verification
	var setupByUser string
	err := s.UserModel.DB.QueryRow(
		"SELECT full_name FROM users WHERE id = $1", 
		setupBy,
	).Scan(&setupByUser)
	if err != nil {
		setupByUser = "System"
	}

	// Notify ticket creator
	if ticket.CreatedBy != nil {
		notification := models.Notification{
			UserID:      *ticket.CreatedBy,
			Title:       "Ticket Verification Setup",
			Message:     fmt.Sprintf("Verification has been set up for ticket %s by %s", ticket.TicketNum, setupByUser),
			Type:        "ticket_verification_setup",
			RelatedID:   &ticket.ID,
			RelatedType: stringPtr("ticket"),
			IsRead:      false,
		}
		s.NotificationModel.Create(&notification)
	}

	// Notify assigned user (if different from creator)
	if ticket.AssignedTo != nil && (ticket.CreatedBy == nil || *ticket.AssignedTo != *ticket.CreatedBy) {
		notification := models.Notification{
			UserID:      *ticket.AssignedTo,
			Title:       "Ticket Verification Setup",
			Message:     fmt.Sprintf("Verification has been set up for ticket %s by %s", ticket.TicketNum, setupByUser),
			Type:        "ticket_verification_setup",
			RelatedID:   &ticket.ID,
			RelatedType: stringPtr("ticket"),
			IsRead:      false,
		}
		s.NotificationModel.Create(&notification)
	}

	return nil
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