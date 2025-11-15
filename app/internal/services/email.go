package services

import (
	"fmt"
	"net/smtp"
	"victortillett.net/internal-inventory-tracker/internal/config"
)

type EmailService struct {
	config *config.Config
}

func NewEmailService(cfg *config.Config) *EmailService {
	return &EmailService{config: cfg}
}

// SendEmail sends a basic email
func (es *EmailService) SendEmail(to, subject, body string) error {
	from := es.config.SMTPFrom
	
	// Format the email message
	msg := []byte(fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s", from, to, subject, body))

	// Mailpit doesn't require authentication, use empty auth
	auth := smtp.PlainAuth("", "", "", es.config.SMTPHost)
	
	// Send the email
	err := smtp.SendMail(
		fmt.Sprintf("%s:%s", es.config.SMTPHost, es.config.SMTPPort),
		auth,
		from,
		[]string{to},
		msg,
	)
	
	if err != nil {
		return fmt.Errorf("failed to send email: %v", err)
	}
	
	return nil
}

// SendHTMLEmail sends an HTML formatted email
func (es *EmailService) SendHTMLEmail(to, subject, htmlBody, textBody string) error {
	from := es.config.SMTPFrom
	
	// Format the email with HTML content
	msg := []byte(fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		from, to, subject, htmlBody,
	))

	auth := smtp.PlainAuth("", "", "", es.config.SMTPHost)
	
	err := smtp.SendMail(
		fmt.Sprintf("%s:%s", es.config.SMTPHost, es.config.SMTPPort),
		auth,
		from,
		[]string{to},
		msg,
	)
	
	if err != nil {
		return fmt.Errorf("failed to send HTML email: %v", err)
	}
	
	return nil
}

// Email Templates

// SendWelcomeEmail sends welcome email to new users
func (es *EmailService) SendWelcomeEmail(to, username, tempPassword string) error {
	subject := "Welcome to Internal Inventory Tracker"
	body := fmt.Sprintf(`
Hello %s,

Welcome to the Internal Inventory Tracker system!

Your account has been created successfully.
Username: %s
Temporary Password: %s

Please log in and change your password immediately.

Best regards,
IT Support Team
	`, username, username, tempPassword)

	return es.SendEmail(to, subject, body)
}

// SendTicketAssignedEmail notifies IT staff when assigned to a ticket
func (es *EmailService) SendTicketAssignedEmail(to, ticketNumber, ticketTitle, assignedBy string) error {
	subject := fmt.Sprintf("New Ticket Assigned: %s", ticketNumber)
	
	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; }
        .header { background: #f4f4f4; padding: 10px; border-left: 4px solid #007cba; }
        .content { padding: 20px; }
        .ticket-info { background: #f9f9f9; padding: 15px; border-radius: 5px; }
        .button { background: #007cba; color: white; padding: 10px 20px; text-decoration: none; border-radius: 3px; }
    </style>
</head>
<body>
    <div class="header">
        <h2>New Ticket Assigned</h2>
    </div>
    <div class="content">
        <p>Hello,</p>
        <p>You have been assigned to a new support ticket:</p>
        
        <div class="ticket-info">
            <strong>Ticket Number:</strong> %s<br>
            <strong>Title:</strong> %s<br>
            <strong>Assigned by:</strong> %s<br>
            <strong>Assigned at:</strong> %s
        </div>
        
        <p>Please log in to the system to review and update the ticket status.</p>
        
        <p>
            <a href="http://localhost:8081" class="button">View Ticket</a>
        </p>
        
        <p>Best regards,<br>IT Support Team</p>
    </div>
</body>
</html>
	`, ticketNumber, ticketTitle, assignedBy, fmt.Sprintf("%v", es.getCurrentTime()))

	textBody := fmt.Sprintf(`
Hello,

You have been assigned to a new support ticket:

Ticket Number: %s
Title: %s
Assigned by: %s
Assigned at: %s

Please log in to the system to review and update the ticket status.

Best regards,
IT Support Team
	`, ticketNumber, ticketTitle, assignedBy, fmt.Sprintf("%v", es.getCurrentTime()))

	return es.SendHTMLEmail(to, subject, htmlBody, textBody)
}

// SendTicketStatusUpdateEmail notifies about ticket status changes
func (es *EmailService) SendTicketStatusUpdateEmail(to, ticketNumber, ticketTitle, oldStatus, newStatus, updatedBy string) error {
	subject := fmt.Sprintf("Ticket Status Updated: %s", ticketNumber)
	
	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; }
        .header { background: #f4f4f4; padding: 10px; border-left: 4px solid #28a745; }
        .content { padding: 20px; }
        .status-change { background: #f9f9f9; padding: 15px; border-radius: 5px; }
        .status-badge { padding: 5px 10px; border-radius: 3px; font-weight: bold; }
        .status-old { background: #ffc107; color: #000; }
        .status-new { background: #28a745; color: #fff; }
    </style>
</head>
<body>
    <div class="header">
        <h2>Ticket Status Updated</h2>
    </div>
    <div class="content">
        <p>Hello,</p>
        <p>The status of ticket <strong>%s</strong> has been updated:</p>
        
        <div class="status-change">
            <strong>Ticket:</strong> %s<br>
            <strong>Title:</strong> %s<br>
            <strong>Status changed from:</strong> <span class="status-badge status-old">%s</span><br>
            <strong>Status changed to:</strong> <span class="status-badge status-new">%s</span><br>
            <strong>Updated by:</strong> %s<br>
            <strong>Updated at:</strong> %s
        </div>
        
        <p>Please log in to the system for more details.</p>
        
        <p>
            <a href="http://localhost:8081" class="button">View Ticket</a>
        </p>
        
        <p>Best regards,<br>IT Support Team</p>
    </div>
</body>
</html>
	`, ticketNumber, ticketNumber, ticketTitle, oldStatus, newStatus, updatedBy, fmt.Sprintf("%v", es.getCurrentTime()))

	textBody := fmt.Sprintf(`
Hello,

The status of ticket %s has been updated:

Ticket: %s
Title: %s
Status changed from: %s
Status changed to: %s
Updated by: %s
Updated at: %s

Please log in to the system for more details.

Best regards,
IT Support Team
	`, ticketNumber, ticketNumber, ticketTitle, oldStatus, newStatus, updatedBy, fmt.Sprintf("%v", es.getCurrentTime()))

	return es.SendHTMLEmail(to, subject, htmlBody, textBody)
}

// SendTicketCommentEmail notifies about new comments
func (es *EmailService) SendTicketCommentEmail(to, ticketNumber, ticketTitle, comment, commentBy string) error {
	subject := fmt.Sprintf("New Comment on Ticket: %s", ticketNumber)
	
	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; }
        .header { background: #f4f4f4; padding: 10px; border-left: 4px solid #17a2b8; }
        .content { padding: 20px; }
        .comment { background: #f8f9fa; padding: 15px; border-left: 4px solid #17a2b8; margin: 10px 0; }
    </style>
</head>
<body>
    <div class="header">
        <h2>New Ticket Comment</h2>
    </div>
    <div class="content">
        <p>Hello,</p>
        <p>A new comment has been added to ticket <strong>%s</strong>:</p>
        
        <div class="comment">
            <strong>Ticket:</strong> %s<br>
            <strong>Title:</strong> %s<br>
            <strong>Comment by:</strong> %s<br>
            <strong>Comment:</strong><br>%s
        </div>
        
        <p>Please log in to the system to view the full conversation.</p>
        
        <p>
            <a href="http://localhost:8081" class="button">View Ticket</a>
        </p>
        
        <p>Best regards,<br>IT Support Team</p>
    </div>
</body>
</html>
	`, ticketNumber, ticketNumber, ticketTitle, commentBy, comment)

	textBody := fmt.Sprintf(`
Hello,

A new comment has been added to ticket %s:

Ticket: %s
Title: %s
Comment by: %s
Comment: %s

Please log in to the system to view the full conversation.

Best regards,
IT Support Team
	`, ticketNumber, ticketNumber, ticketTitle, commentBy, comment)

	return es.SendHTMLEmail(to, subject, htmlBody, textBody)
}

// SendAssetServiceReminder sends reminder for asset service
func (es *EmailService) SendAssetServiceReminder(to, assetID, assetType, assetModel, nextServiceDate string) error {
	subject := fmt.Sprintf("Service Reminder: %s", assetID)
	
	body := fmt.Sprintf(`
Hello,

This is a reminder that the following asset requires service:

Asset ID: %s
Type: %s
Model: %s
Next Service Date: %s

Please schedule maintenance for this asset.

Best regards,
Asset Management System
	`, assetID, assetType, assetModel, nextServiceDate)

	return es.SendEmail(to, subject, body)
}

// Helper function to get current time
func (es *EmailService) getCurrentTime() string {
	return fmt.Sprintf("%v", time.Now().Format("2006-01-02 15:04:05"))
}