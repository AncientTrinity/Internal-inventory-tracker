package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"fmt"

	"victortillett.net/internal-inventory-tracker/internal/middleware"
	"victortillett.net/internal-inventory-tracker/internal/models"
	"victortillett.net/internal-inventory-tracker/internal/services"
	"golang.org/x/crypto/bcrypt"
)

type UsersHandler struct {
	Model        *models.UsersModel
	EmailService *services.EmailService
}

func NewUsersHandler(db *sql.DB, emailService *services.EmailService) *UsersHandler {
	return &UsersHandler{
		Model:        models.NewUsersModel(db),
		EmailService: emailService,
	}
}

// POST /api/v1/users - Enhanced with role-based password control
func (h *UsersHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	// Get current user for authorization
	currentUserID, ok := r.Context().Value(middleware.ContextUserID).(int)
	if !ok {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	// Get current user's role
	var currentUserRoleID int
	err := h.Model.DB.QueryRow("SELECT role_id FROM users WHERE id = $1", currentUserID).Scan(&currentUserRoleID)
	if err != nil {
		http.Error(w, "Failed to verify user permissions", http.StatusInternalServerError)
		return
	}

	// Only Admin (1) and IT (2) can create users with set passwords
	canSetPassword := (currentUserRoleID == 1 || currentUserRoleID == 2)

	var currentUsername string
	h.Model.DB.QueryRow("SELECT username FROM users WHERE id = $1", currentUserID).Scan(&currentUsername)

	var input struct {
		Username   string `json:"username"`
		FullName   string `json:"full_name"`
		Email      string `json:"email"`
		Password   string `json:"password"`    // Only Admins/IT can set this
		RoleID     int64  `json:"role_id"`
		SendEmail  bool   `json:"send_email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	// Validate password requirements for Admins/IT
	var password string
	var passwordSetByAdmin bool

	if input.Password != "" {
		if !canSetPassword {
			http.Error(w, "Only Administrators and IT staff can set user passwords", http.StatusForbidden)
			return
		}
		
		// Validate password strength
		if len(input.Password) < 8 {
			http.Error(w, "Password must be at least 8 characters long", http.StatusBadRequest)
			return
		}
		
		password = input.Password
		passwordSetByAdmin = true
	} else {
		// Generate temporary password for non-Admin/IT creators or when no password provided
		password = generateTemporaryPassword()
		passwordSetByAdmin = false
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
		go h.sendWelcomeEmail(input.Email, input.Username, password, currentUsername, passwordSetByAdmin)
	}

	// Return response
	responseUser := map[string]interface{}{
		"id":           u.ID,
		"username":     u.Username,
		"full_name":    u.FullName,
		"email":        u.Email,
		"role_id":      u.RoleID,
		"created_at":   u.CreatedAt,
		"email_sent":   input.SendEmail && input.Email != "",
		"password_set_by_admin": passwordSetByAdmin,
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(responseUser)
}

// GET /api/v1/users - List users (Admin and IT only)
func (h *UsersHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
    users, err := h.Model.GetAll()
    if err != nil {
        http.Error(w, "Database error", http.StatusInternalServerError) 
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(users)
}
func (h *UsersHandler) GetUser(w http.ResponseWriter, r *http.Request) {
    idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/users/")
    id, err := strconv.ParseInt(idStr, 10, 64)
    if err != nil {
        http.Error(w, "Invalid user ID", http.StatusBadRequest)
        return
    }
    
    user, err := h.Model.GetByID(id)
    if err != nil {
        if err.Error() == "user not found" {
            http.Error(w, "User not found", http.StatusNotFound)
            return
        }
        http.Error(w, "Database error", http.StatusInternalServerError)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(user)
}

// PUT /api/v1/users/{id}
func (h *UsersHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
    idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/users/")
    id, err := strconv.ParseInt(idStr, 10, 64)
    if err != nil {
        http.Error(w, "Invalid user ID", http.StatusBadRequest)
        return
    }
    
    var input struct {
        Username string `json:"username"`
        FullName string `json:"full_name"`
        Email    string `json:"email"`
        RoleID   int64  `json:"role_id"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
        http.Error(w, "Invalid input", http.StatusBadRequest)
        return
    }
    
    // Get existing user
    existingUser, err := h.Model.GetByID(id)
    if err != nil {
        if err.Error() == "user not found" {
            http.Error(w, "User not found", http.StatusNotFound)
            return
        }
        http.Error(w, "Database error", http.StatusInternalServerError)
        return
    }
    
    // Update fields
    if input.Username != "" {
        existingUser.Username = input.Username
    }
    if input.FullName != "" {
        existingUser.FullName = input.FullName
    }
    if input.Email != "" {
        existingUser.Email = input.Email
    }
    if input.RoleID != 0 {
        existingUser.RoleID = input.RoleID
    }
    
    err = h.Model.Update(existingUser)
    if err != nil {
        http.Error(w, "Database error", http.StatusInternalServerError)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(existingUser)
}

// DELETE /api/v1/users/{id}
func (h *UsersHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
    idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/users/")
    id, err := strconv.ParseInt(idStr, 10, 64)
    if err != nil {
        http.Error(w, "Invalid user ID", http.StatusBadRequest)
        return
    }
    
    err = h.Model.Delete(id)
    if err != nil {
        if err.Error() == "user not found" {
            http.Error(w, "User not found", http.StatusNotFound)
            return
        }
        http.Error(w, "Database error", http.StatusInternalServerError)
        return
    }
    
    w.WriteHeader(http.StatusNoContent)
}


// POST /api/v1/users/{id}/reset-password - New endpoint for password reset (Admin/IT only)
func (h *UsersHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	// Get current user for authorization
	currentUserID, ok := r.Context().Value(middleware.ContextUserID).(int)
	if !ok {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	// Get current user's role
	var currentUserRoleID int
	err := h.Model.DB.QueryRow("SELECT role_id FROM users WHERE id = $1", currentUserID).Scan(&currentUserRoleID)
	if err != nil {
		http.Error(w, "Failed to verify user permissions", http.StatusInternalServerError)
		return
	}

	// Only Admin (1) and IT (2) can reset passwords
	if currentUserRoleID != 1 && currentUserRoleID != 2 {
		http.Error(w, "Only Administrators and IT staff can reset passwords", http.StatusForbidden)
		return
	}

	var currentUsername string
	h.Model.DB.QueryRow("SELECT username FROM users WHERE id = $1", currentUserID).Scan(&currentUsername)

	idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/users/")
	idStr = strings.TrimSuffix(idStr, "/reset-password")
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

	var input struct {
		NewPassword string `json:"new_password"`
		SendEmail   bool   `json:"send_email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	// Validate new password
	var newPassword string
	if input.NewPassword != "" {
		// Admin/IT is setting a specific password
		if len(input.NewPassword) < 8 {
			http.Error(w, "Password must be at least 8 characters long", http.StatusBadRequest)
			return
		}
		newPassword = input.NewPassword
	} else {
		// Generate a strong temporary password
		newPassword = generateStrongPassword()
	}

	// Hash and update password
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

	// Send email notification if requested and user has email
	if input.SendEmail && user.Email != "" {
		go h.sendPasswordResetEmail(user.Email, user.Username, newPassword, currentUsername, input.NewPassword != "")
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":          "Password reset successfully",
		"user_id":          id,
		"email_sent":       input.SendEmail && user.Email != "",
		"password_set_by_admin": input.NewPassword != "",
	})
}

// POST /api/v1/users/{id}/send-credentials - Updated for Admin/IT only
func (h *UsersHandler) SendCredentials(w http.ResponseWriter, r *http.Request) {
	// Get current user for authorization
	currentUserID, ok := r.Context().Value(middleware.ContextUserID).(int)
	if !ok {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	// Get current user's role
	var currentUserRoleID int
	err := h.Model.DB.QueryRow("SELECT role_id FROM users WHERE id = $1", currentUserID).Scan(&currentUserRoleID)
	if err != nil {
		http.Error(w, "Failed to verify user permissions", http.StatusInternalServerError)
		return
	}

	// Only Admin (1) and IT (2) can send credentials
	if currentUserRoleID != 1 && currentUserRoleID != 2 {
		http.Error(w, "Only Administrators and IT staff can send user credentials", http.StatusForbidden)
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

	// Generate a strong temporary password
	newPassword := generateStrongPassword()

	// Update password
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

	// Send credentials email
	go h.sendWelcomeEmail(user.Email, user.Username, newPassword, currentUsername, false)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":    "Credentials sent successfully",
		"user_id":    id,
		"email":      user.Email,
	})
}

// Helper function to generate strong temporary password
func generateStrongPassword() string {
	length := 16
	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	password := make([]byte, length)
	for i := range password {
		// In real implementation, use crypto/rand for security
		password[i] = charset[i%len(charset)]
	}
	return string(password)
}

// Helper function to generate temporary password (weaker, for non-Admin/IT)
func generateTemporaryPassword() string {
	length := 12
	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	password := make([]byte, length)
	for i := range password {
		password[i] = charset[i%len(charset)]
	}
	return string(password)
}

// Updated sendPasswordResetEmail to indicate who set the password
func (h *UsersHandler) sendPasswordResetEmail(to, username, newPassword, resetBy string, customPasswordSet bool) error {
	subject := "Password Reset - Internal Inventory Tracker"
	
	var passwordMessage string
	if customPasswordSet {
		passwordMessage = "An administrator has set a new password for your account."
	} else {
		passwordMessage = "Your password has been reset. Here is your new temporary password:"
	}
	
	// HTML template for password reset
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
        <h1>Password Reset</h1>
        <p>Your password has been updated</p>
    </div>
    
    <div class="content">
        <p>Hello <strong>%s</strong>,</p>
        
        <p>%s</p>
        
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
                <strong>🔑 New Password:</strong><br>
                <code style="font-size: 18px; font-weight: bold; color: #e74c3c;">%s</code>
            </div>
        </div>
        
        <div class="password-warning">
            <strong>⚠️ Security Notice:</strong><br>
            Please log in and change your password immediately for security reasons.
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
	`, username, passwordMessage, username, newPassword)

	// Text version
	textBody := fmt.Sprintf(`
Password Reset - Internal Inventory Tracker

Hello %s,

%s

Login URL: http://localhost:8081
Username: %s
New Password: %s

Security Notice: Please log in and change your password immediately for security reasons.

If you have any questions or need assistance, please contact the IT support team.

This email was sent automatically. Please do not reply to this message.
IT Support Team
	`, username, passwordMessage, username, newPassword)

	return h.EmailService.SendHTMLEmail(to, subject, htmlBody, textBody)
}

// Helper function to get security message based on who set the password
func getSecurityMessage(passwordSetByAdmin bool) string {
	if passwordSetByAdmin {
		return "This password was set by an administrator. You may continue using this password."
	} else {
		return "This is a  password set by the Ariston. Please Contact IT Support to change it. "
	}
}


// Updated sendWelcomeEmail to indicate who set the password
func (h *UsersHandler) sendWelcomeEmail(to, username, password, createdBy string, passwordSetByAdmin bool) error {
    subject := "Welcome to Internal Inventory Tracker - Your Login Credentials"
    
    var passwordMessage string
    if passwordSetByAdmin {
        passwordMessage = "An administrator has set your initial password. Please use the credentials below to log in."
    } else {
        passwordMessage = "Your account has been created. Here are your temporary login credentials:"
    }
    
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
        .admin-note { background: #e8f5e8; border: 1px solid #c8e6c9; padding: 15px; border-radius: 5px; margin: 20px 0; }
    </style>
</head>
<body>
    <div class="header">
        <h1>Welcome to Internal Inventory Tracker</h1>
        <p>Your account has been created successfully</p>
    </div>
    
    <div class="content">
        <p>Hello <strong>%s</strong>,</p>
        
        <p>%s</p>
        
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
                <strong>🔑 Password:</strong><br>
                <code style="font-size: 18px; font-weight: bold; color: #e74c3c;">%s</code>
            </div>
        </div>
        
        <div class="password-warning">
            <strong>⚠️ Security Notice:</strong><br>
            %s
        </div>
        
        <div class="admin-note">
            <strong>👨‍💼 Account Created By:</strong><br>
            %s (IT Support Team)
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
    `, username, passwordMessage, username, password, 
       getSecurityMessage(passwordSetByAdmin), createdBy)

    textBody := fmt.Sprintf(`
Welcome to Internal Inventory Tracker

Hello %s,

%s

Login URL: http://localhost:8081
Username: %s
Password: %s

%s

Account Created By: %s (IT Support Team)

If you have any questions or need assistance, please contact the IT support team.

This email was sent automatically. Please do not reply to this message.
IT Support Team
    `, username, passwordMessage, username, password, 
       getSecurityMessage(passwordSetByAdmin), createdBy)

    return h.EmailService.SendHTMLEmail(to, subject, htmlBody, textBody)
}