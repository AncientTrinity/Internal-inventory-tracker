package services

import (
	"fmt"
	"net/smtp"
	"time"
	"net"
	"victortillett.net/internal-inventory-tracker/internal/config"
)

type EnhancedEmailService struct {
	config *config.Config
}

func NewEnhancedEmailService(cfg *config.Config) *EnhancedEmailService {
	return &EnhancedEmailService{config: cfg}
}

// SendEmailWithRetry attempts to send email with retry logic
func (es *EnhancedEmailService) SendEmailWithRetry(to, subject, body string, maxRetries int) error {
	var lastErr error
	
	for i := 0; i < maxRetries; i++ {
		err := es.sendSingleEmail(to, subject, body)
		if err == nil {
			return nil // Success
		}
		
		lastErr = err
		fmt.Printf("⚠️ Email send attempt %d failed: %v\n", i+1, err)
		
		// Wait before retry (exponential backoff)
		waitTime := time.Duration(i*i) * time.Second
		time.Sleep(waitTime)
	}
	
	return fmt.Errorf("failed to send email after %d attempts: %v", maxRetries, lastErr)
}

// sendSingleEmail sends a single email attempt
func (es *EnhancedEmailService) sendSingleEmail(to, subject, body string) error {
	from := es.config.SMTPFrom
	
	// Enhanced email headers
	msg := []byte(fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nDate: %s\r\n\r\n%s",
		from, to, subject, time.Now().Format(time.RFC1123Z), body,
	))

	// Smart authentication - only use credentials if provided
	var auth smtp.Auth
	if es.config.SMTPUsername != "" && es.config.SMTPPassword != "" {
		// Use authentication for real SMTP servers
		auth = smtp.PlainAuth("", es.config.SMTPUsername, es.config.SMTPPassword, es.config.SMTPHost)
	} else {
		// No authentication for Mailpit/local development
		auth = nil
	}
	
	return smtp.SendMail(
		fmt.Sprintf("%s:%s", es.config.SMTPHost, es.config.SMTPPort),
		auth,
		from,
		[]string{to},
		msg,
	)
}

// SendBulkEmails sends multiple emails efficiently
func (es *EnhancedEmailService) SendBulkEmails(emails []struct {
	To      string
	Subject string
	Body    string
}) error {
	for _, email := range emails {
		go func(e struct { To, Subject, Body string }) {
			err := es.SendEmailWithRetry(e.To, e.Subject, e.Body, 3)
			if err != nil {
				fmt.Printf("❌ Failed to send email to %s: %v\n", e.To, err)
			}
		}(email)
	}
	return nil
}


func (es *EmailService) DebugConfig() {
    fmt.Printf("🔍 Email Config - Host: %s, Port: %s, From: %s\n", 
        es.config.SMTPHost, es.config.SMTPPort, es.config.SMTPFrom)
}

// Add this method to check connection
func (es *EmailService) TestConnection() error {
    fmt.Printf("🔍 Testing connection to %s:%s\n", es.config.SMTPHost, es.config.SMTPPort)
    
    conn, err := net.Dial("tcp", fmt.Sprintf("%s:%s", es.config.SMTPHost, es.config.SMTPPort))
    if err != nil {
        return fmt.Errorf("SMTP connection failed: %v", err)
    }
    defer conn.Close()
    
    fmt.Printf("✅ SMTP connection successful\n")
    return nil
}

func (es *EmailService) SendCurrentCredentials(to, username string) error {
    subject := "Your Account Credentials - Internal Inventory Tracker"
    
    body := fmt.Sprintf(`
Hello %s,

Here are your current login credentials for the Internal Inventory Tracker system:

Username: %s

Please use your existing password to login.

If you have forgotten your password, please contact your IT administrator to reset it.

Best regards,
IT Support Team
    `, username, username)

    fmt.Printf("📧 SendCurrentCredentials - Sending current credentials to: %s\n", to)
    return es.SendEmail(to, subject, body)
}

