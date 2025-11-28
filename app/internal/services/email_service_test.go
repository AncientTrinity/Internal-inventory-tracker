// file: app/internal/services/email_service_test.go
package services

import (
	"testing"

	"victortillett.net/internal-inventory-tracker/internal/config"

	"github.com/stretchr/testify/assert"
)

func TestEmailService_SendTicketAssignedEmail(t *testing.T) {
	cfg := &config.Config{
		SMTPHost: "localhost",
		SMTPPort: "1025",
		SMTPFrom: "noreply@example.com",
	}

	service := NewEnhancedEmailService(cfg)

	t.Run("successful ticket assigned email", func(t *testing.T) {
		// Mock the underlying send method
		originalSend := service.SendEmailWithRetry
		defer func() { service.SendEmailWithRetry = originalSend }()
		
		var called bool
		service.SendEmailWithRetry = func(to, subject, body string, maxRetries int) error {
			called = true
			assert.Equal(t, "assignee@example.com", to)
			assert.Contains(t, subject, "TCK-2024-0001")
			assert.Contains(t, subject, "Test Ticket")
			assert.Contains(t, body, "TCK-2024-0001")
			assert.Contains(t, body, "Test Ticket")
			assert.Contains(t, body, "adminuser")
			return nil
		}

		err := service.SendTicketAssignedEmail("assignee@example.com", "TCK-2024-0001", "Test Ticket", "adminuser")
		
		assert.NoError(t, err)
		assert.True(t, called)
	})
}

func TestEmailService_SendTicketStatusUpdateEmail(t *testing.T) {
	cfg := &config.Config{
		SMTPHost: "localhost",
		SMTPPort: "1025",
		SMTPFrom: "noreply@example.com",
	}

	service := NewEnhancedEmailService(cfg)

	t.Run("successful status update email", func(t *testing.T) {
		originalSend := service.SendEmailWithRetry
		defer func() { service.SendEmailWithRetry = originalSend }()
		
		var called bool
		service.SendEmailWithRetry = func(to, subject, body string, maxRetries int) error {
			called = true
			assert.Equal(t, "user@example.com", to)
			assert.Contains(t, subject, "TCK-2024-0001")
			assert.Contains(t, subject, "open")
			assert.Contains(t, subject, "in_progress")
			assert.Contains(t, body, "open")
			assert.Contains(t, body, "in_progress")
			assert.Contains(t, body, "adminuser")
			return nil
		}

		err := service.SendTicketStatusUpdateEmail(
			"user@example.com", 
			"TCK-2024-0001", 
			"Test Ticket", 
			"open", 
			"in_progress", 
			"adminuser",
		)
		
		assert.NoError(t, err)
		assert.True(t, called)
	})
}