// file: app/internal/services/email_enhanced_test.go
package services

import (
	"errors"
	"fmt"
	"net/smtp"
	"testing"
	"time"

	"victortillett.net/internal-inventory-tracker/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockSMTPClient implements a mock SMTP client for testing
type MockSMTPClient struct {
	ShouldFail  bool
	FailCount   int
	CallCount   int
	LastFrom    string
	LastTo      []string
	LastMessage []byte
}

func (m *MockSMTPClient) SendMail(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
	m.CallCount++
	m.LastFrom = from
	m.LastTo = to
	m.LastMessage = msg

	if m.ShouldFail && m.CallCount <= m.FailCount {
		return errors.New("mock SMTP failure")
	}
	return nil
}

// Test wrapper that uses our mock
type TestEnhancedEmailService struct {
	*EnhancedEmailService
	mockClient *MockSMTPClient
}

func NewTestEnhancedEmailService(cfg *config.Config, mockClient *MockSMTPClient) *TestEnhancedEmailService {
	service := NewEnhancedEmailService(cfg)
	
	testService := &TestEnhancedEmailService{
		EnhancedEmailService: service,
		mockClient: mockClient,
	}
	
	return testService
}

// Override the sendSingleEmail method for testing
func (tes *TestEnhancedEmailService) sendSingleEmail(to, subject, body string) error {
	return tes.mockClient.SendMail(
		fmt.Sprintf("%s:%s", tes.config.SMTPHost, tes.config.SMTPPort),
		nil, // auth handled in mock if needed
		tes.config.SMTPFrom,
		[]string{to},
		[]byte(fmt.Sprintf("Subject: %s\n\n%s", subject, body)),
	)
}

func TestEnhancedEmailService_SendEmailWithRetry(t *testing.T) {
	cfg := &config.Config{
		SMTPHost:     "localhost",
		SMTPPort:     "1025",
		SMTPFrom:     "noreply@example.com",
		SMTPUsername: "user",
		SMTPPassword: "pass",
	}

	t.Run("successful send on first attempt", func(t *testing.T) {
		mockClient := &MockSMTPClient{ShouldFail: false}
		service := NewTestEnhancedEmailService(cfg, mockClient)
		
		err := service.SendEmailWithRetry("test@example.com", "Test Subject", "Test Body", 3)

		assert.NoError(t, err)
		assert.Equal(t, 1, mockClient.CallCount)
	})

	t.Run("successful send after retries", func(t *testing.T) {
		mockClient := &MockSMTPClient{ShouldFail: true, FailCount: 2}
		service := NewTestEnhancedEmailService(cfg, mockClient)
		
		start := time.Now()
		err := service.SendEmailWithRetry("test@example.com", "Test Subject", "Test Body", 3)
		duration := time.Since(start)

		assert.NoError(t, err)
		assert.Equal(t, 3, mockClient.CallCount)
		assert.GreaterOrEqual(t, duration, 1*time.Second) // Should have waited for retries
	})

	t.Run("failure after max retries", func(t *testing.T) {
		mockClient := &MockSMTPClient{ShouldFail: true, FailCount: 5}
		service := NewTestEnhancedEmailService(cfg, mockClient)
		
		err := service.SendEmailWithRetry("test@example.com", "Test Subject", "Test Body", 3)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to send email after 3 attempts")
		assert.Equal(t, 3, mockClient.CallCount)
	})
}

func TestEnhancedEmailService_SendBulkEmails(t *testing.T) {
	cfg := &config.Config{
		SMTPHost: "localhost",
		SMTPPort: "1025",
		SMTPFrom: "noreply@example.com",
	}

	t.Run("successful bulk email send", func(t *testing.T) {
		mockClient := &MockSMTPClient{ShouldFail: false}
		service := NewTestEnhancedEmailService(cfg, mockClient)
		
		emails := []struct {
			To      string
			Subject string
			Body    string
		}{
			{"user1@example.com", "Welcome 1", "Body 1"},
			{"user2@example.com", "Welcome 2", "Body 2"},
			{"user3@example.com", "Welcome 3", "Body 3"},
		}

		err := service.SendBulkEmails(emails)

		assert.NoError(t, err)
		// Wait a bit for goroutines to complete
		time.Sleep(100 * time.Millisecond)
	})

	t.Run("bulk emails with some failures", func(t *testing.T) {
		mockClient := &MockSMTPClient{ShouldFail: true, FailCount: 1}
		service := NewTestEnhancedEmailService(cfg, mockClient)
		
		emails := []struct {
			To      string
			Subject string
			Body    string
		}{
			{"user1@example.com", "Welcome", "Body"},
		}

		err := service.SendBulkEmails(emails)

		assert.NoError(t, err) // Bulk send doesn't return individual errors
	})
}