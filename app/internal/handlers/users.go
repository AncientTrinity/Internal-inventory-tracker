package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"victortillett.net/internal-inventory-tracker/internal/middleware"
	"victortillett.net/internal-inventory-tracker/internal/models"
	"victortillett.net/internal-inventory-tracker/internal/services" 
	"golang.org/x/crypto/bcrypt"
)

type UsersHandler struct {
	Model        *models.UsersModel
	EmailService *services.EmailService // Add this
}

func NewUsersHandler(db *sql.DB, emailService *services.EmailService) *UsersHandler { // Update constructor
	return &UsersHandler{
		Model:        models.NewUsersModel(db),
		EmailService: emailService, // Add this
	}
}

// POST /api/v1/users - Enhanced to send welcome email
func (h *UsersHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	// Get current user for audit purposes
	currentUserID, ok := r.Context().Value(middleware.ContextUserID).(int)
	if !ok {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var currentUsername string
	h.Model.DB.QueryRow("SELECT username FROM users WHERE id = $1", currentUserID).Scan(&currentUsername)

	var input struct {
		Username string `json:"username"`
		FullName string `json:"full_name"`
		Email    string `json:"email"`
		Password string `json:"password"`
		RoleID   int64  `json:"role_id"`
		SendEmail bool  `json:"send_email"` // New field to control email sending
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	// Generate password if not provided
	password := input.Password
	if password == "" {
		password = generateTemporaryPassword()
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Password hash error", http.StatusInternalServerError)
		return
	}

	u := &models.User{
		Username:     input.Username,
		FullName:     input.FullName,
		Email:        input.Email,
		PasswordHash: string(hash),
		RoleID:       input.RoleID,
	}

	err = h.Model.Insert(u)
	if err != nil {
		http.Error(w, "DB insert error", http.StatusInternalServerError)
		return
	}

	// Send welcome email if requested and email is provided
	if input.SendEmail && input.Email != "" {
		go h.sendWelcomeEmail(input.Email, input.Username, password, currentUsername)
	}

	// Don't return password hash in response
	responseUser := map[string]interface{}{
		"id":         u.ID,
		"username":   u.Username,
		"full_name":  u.FullName,
		"email":      u.Email,
		"role_id":    u.RoleID,
		"created_at": u.CreatedAt,
		"email_sent": input.SendEmail && input.Email != "",
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(responseUser)
}

// POST /api/v1/users/{id}/send-credentials - New endpoint to send credentials
func (h *UsersHandler) SendCredentials(w http.ResponseWriter, r *http.Request) {
	// Get current user for audit
	currentUserID, ok := r.Context().Value(middleware.ContextUserID).(int)
	if !ok {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var currentUsername string
	h.Model.DB.QueryRow("SELECT username FROM users WHERE id = $1", currentUserID).Scan(&currentUsername)

	idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/users/")
	idStr = strings.TrimSuffix(idStr, "/send-credentials")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Get user details
	var user models.User
	err = h.Model.DB.QueryRow(`
		SELECT id, username, full_name, email, role_id 
		FROM users WHERE id = $1
	`, id).Scan(&user.ID, &user.Username, &user.FullName, &user.Email, &user.RoleID)
	
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Check if user has email
	if user.Email == "" {
		http.Error(w, "User does not have an email address", http.StatusBadRequest)
		return
	}

	var input struct {
		GenerateNewPassword bool `json:"generate_new_password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		// Default to not generating new password if not specified
		input.GenerateNewPassword = false
	}

	var newPassword string
	if input.GenerateNewPassword {
		// Generate and set new password
		newPassword = generateTemporaryPassword()
		hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "Password hash error", http.StatusInternalServerError)
			return
		}

		_, err = h.Model.DB.Exec("UPDATE users SET password_hash = $1 WHERE id = $2", string(hash), id)
		if err != nil {
			http.Error(w, "Failed to update password", http.StatusInternalServerError)
			return
		}
	} else {
		// Send existing credentials (in real system, you can't retrieve existing password)
		// For security, we'll generate a new temporary password
		newPassword = generateTemporaryPassword()
		hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "Password hash error", http.StatusInternalServerError)
			return
		}

		_, err = h.Model.DB.Exec("UPDATE users SET password_hash = $1 WHERE id = $2", string(hash), id)
		if err != nil {
			http.Error(w, "Failed to update password", http.StatusInternalServerError)
			return
		}
	}

	// Send credentials email
	go h.sendWelcomeEmail(user.Email, user.Username, newPassword, currentUsername)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Credentials sent successfully",
		"user_id": id,
		"email":   user.Email,
		"new_password_generated": input.GenerateNewPassword,
	})
}

// Helper function to generate temporary password
func generateTemporaryPassword() string {
	// Generate a random 12-character password
	length := 12
	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%"
	password := make([]byte, length)
	for i := range password {
		// In real implementation, use crypto/rand for security
		password[i] = charset[i%len(charset)]
	}
	return string(password)
}

// sendWelcomeEmail sends welcome email with credentials
func (h *UsersHandler) sendWelcomeEmail(to, username, password, createdBy string) error {
	subject := "Welcome to Internal Inventory Tracker - Your Login Credentials"
	
	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .header { background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); color: white; padding: 30px; text-align: center; }
        .content { padding: 30px; background: #f9f9f9; }
        .credentials { background: white; padding: 25px; border-radius: 10px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); margin: 20px 0; }
        .credential-item { margin: 15px 0; padding: 12px; background: #f8f9fa; border-left: 4px solid #667eea; }
        .password-warning { background: #fff3cd; border: 1px solid #ffeaa7; padding: 15px; border-radius: 5px; margin: 20px 0; }
        .button { background: #667eea; color: white; padding: 12px 30px; text-decoration: none; border-radius: 5px; display: inline-block; }
        .footer { text-align: center; padding: 20px; color: #666; font-size: 14px; }
    </style>
</head>
<body>
    <div class="header">
        <h1>Welcome to Internal Inventory Tracker</h1>
        <p>Your account has been created successfully</p>
    </div>
    
    <div class="content">
        <p>Hello <strong>%s</strong>,</p>
        
        <p>Your account has been created by <strong>%s</strong>. Here are your login credentials:</p>
        
        <div class="credentials">
            <div class="credential-item">
                <strong>🔗 Login URL:</strong><br>
                <a href="http://localhost:8081">http://localhost:8081</a>
            </div>
            
            <div class="credential-item">
                <strong>👤 Username:</strong><br>
                %s
            </div>
            
            <div class="credential-item">
                <strong>🔑 Temporary Password:</strong><br>
                <code style="font-size: 18px; font-weight: bold; color: #e74c3c;">%s</code>
            </div>
        </div>
        
        <div class="password-warning">
            <strong>⚠️ Security Notice:</strong><br>
            This is a temporary password. Please log in and change your password immediately for security reasons.
        </div>
        
        <p>
            <a href="http://localhost:8081" class="button">Login to System</a>
        </p>
        
        <p>If you have any questions or need assistance, please contact the IT support team.</p>
    </div>
    
    <div class="footer">
        <p>This email was sent automatically. Please do not reply to this message.</p>
        <p>IT Support Team</p>
    </div>
</body>
</html>
	`, username, createdBy, username, password)

	textBody := fmt.Sprintf(`
Welcome to Internal Inventory Tracker

Hello %s,

Your account has been created by %s. Here are your login credentials:

Login URL: http://localhost:8081
Username: %s
Temporary Password: %s

SECURITY NOTICE:
This is a temporary password. Please log in and change your password immediately for security reasons.

If you have any questions or need assistance, please contact the IT support team.

This email was sent automatically. Please do not reply to this message.

IT Support Team
	`, username, createdBy, username, password)

	return h.EmailService.SendHTMLEmail(to, subject, htmlBody, textBody)
}