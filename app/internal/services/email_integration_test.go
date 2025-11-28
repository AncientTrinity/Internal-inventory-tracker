// file: app/internal/services/email_integration_test.go
package services

import (
	"testing"

	"victortillett.net/internal-inventory-tracker/internal/config"

	"github.com/stretchr/testify/assert"
)

// TestEmailService_Integration tests the actual email service with Mailpit
// These tests require Mailpit to be running in your Docker environment
func TestEmailService_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := &config.Config{
		SMTPHost: "mailpit",
		SMTPPort: "1025",
		SMTPFrom: "noreply@example.com",
		// No auth for Mailpit
	}

	service := NewEnhancedEmailService(cfg)

	t.Run("test connection to mailpit", func(t *testing.T) {
		err := service.TestConnection()
		assert.NoError(t, err, "Should be able to connect to Mailpit")
	})

	t.Run("send test email to mailpit", func(t *testing.T) {
		err := service.SendEmailWithRetry(
			"test@example.com",
			"Integration Test Email",
			"This is a test email from the integration test suite.",
			1,
		)
		assert.NoError(t, err, "Should be able to send email to Mailpit")
	})
}