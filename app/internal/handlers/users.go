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
	Model	*models.UsersModel
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
    
	// In your CreateUser method, replace the debug section with:
fmt.Printf("🔍 CreateUser - User creation request received\n")

// Test email configuration
fmt.Printf("🔍 CreateUser - Testing email configuration...\n")
h.EmailService.DebugConfig()

// Test SMTP connection
if err := h.EmailService.TestConnection(); err != nil {
    fmt.Printf("❌ CreateUser - SMTP connection test failed: %v\n", err)
} else {
    fmt.Printf("✅ CreateUser - SMTP connection test passed\n")
}

// Check if we should send email (note: it's SendEmail, not SendWelcomeEmail)
fmt.Printf("🔍 CreateUser - SendEmail flag: %v\n", input.SendEmail)

if input.SendEmail {
    fmt.Printf("📧 CreateUser - Sending welcome email to: %s\n", input.Email)
    
    err = h.EmailService.SendWelcomeEmail(input.Email, input.Username, input.Password)
    if err != nil {
        fmt.Printf("❌ CreateUser - Welcome email failed: %v\n", err)
        // Don't return error, just log it - user creation should still succeed
    } else {
        fmt.Printf("✅ CreateUser - Welcome email sent successfully\n")
    }
} else {
    fmt.Printf("ℹ️ CreateUser - SendEmail flag is false, skipping email\n")
}

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
    idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/users/")
    idStr = strings.TrimSuffix(idStr, "/reset-password")
    
    fmt.Printf("🔍 ResetPassword - User ID: %s\n", idStr)
    
    id, err := strconv.ParseInt(idStr, 10, 64)
    if err != nil {
        fmt.Printf("❌ ResetPassword - Invalid user ID: %v\n", err)
        http.Error(w, "Invalid user ID", http.StatusBadRequest)
        return
    }

    // Get current user from context
    userID, ok := r.Context().Value(middleware.ContextUserID).(int)
    if !ok {
        http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
        return
    }

    roleID, ok := r.Context().Value(middleware.ContextRoleID).(int)
    if !ok {
        http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
        return
    }

    fmt.Printf("🔍 ResetPassword - Request by User ID: %d, Role ID: %d\n", userID, roleID)

    // Check permissions
    if roleID != 1 && roleID != 2 { // Only Admin and IT Staff
        http.Error(w, "Forbidden: Only administrators can reset passwords", http.StatusForbidden)
        return
    }

    // Parse the request body
    var input struct {
        NewPassword string `json:"new_password"`
        SendEmail   bool   `json:"send_email"`
    }

    if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
        fmt.Printf("❌ ResetPassword - Invalid input: %v\n", err)
        http.Error(w, "Invalid input", http.StatusBadRequest)
        return
    }

    // Validate the new password
    if input.NewPassword == "" {
        http.Error(w, "New password is required", http.StatusBadRequest)
        return
    }

    if len(input.NewPassword) < 8 {
        http.Error(w, "Password must be at least 8 characters long", http.StatusBadRequest)
        return
    }

    // Get the user who is resetting the password
    var resetByUsername string
    err = h.Model.DB.QueryRow("SELECT username FROM users WHERE id = $1", userID).Scan(&resetByUsername)
    if err != nil {
        resetByUsername = "System Administrator"
    }

    // Get user details for email
    var userEmail, userName string
    err = h.Model.DB.QueryRow(`
        SELECT email, username, full_name 
        FROM users WHERE id = $1
    `, id).Scan(&userEmail, &userName, &userName)
    
    if err != nil {
        fmt.Printf("❌ ResetPassword - Failed to get user details: %v\n", err)
        http.Error(w, "User not found", http.StatusNotFound)
        return
    }

    // Update password in database
    hash, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
    if err != nil {
        fmt.Printf("❌ ResetPassword - Password hash error: %v\n", err)
        http.Error(w, "Password hash error", http.StatusInternalServerError)
        return
    }

    query := `UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2`
    result, err := h.Model.DB.Exec(query, string(hash), id)
    if err != nil {
        fmt.Printf("❌ ResetPassword - Database update error: %v\n", err)
        http.Error(w, "Failed to update password", http.StatusInternalServerError)
        return
    }

    rowsAffected, _ := result.RowsAffected()
    fmt.Printf("✅ ResetPassword - Password updated for user %d by user %d, rows affected: %d\n", id, userID, rowsAffected)

    // ✅ SEND EMAIL NOTIFICATION with the ACTUAL password if requested
    emailSent := false
    if input.SendEmail && userEmail != "" {
        fmt.Printf("📧 ResetPassword - Sending password reset email to: %s\n", userEmail)
        fmt.Printf("📧 ResetPassword - Using ACTUAL password: %s\n", input.NewPassword)
        
        go func() {
            err := h.sendPasswordResetEmail(userEmail, userName, input.NewPassword, resetByUsername, true)
            if err != nil {
                fmt.Printf("❌ ResetPassword - Failed to send email: %v\n", err)
            } else {
                fmt.Printf("✅ ResetPassword - Password reset email sent successfully to %s\n", userEmail)
            }
        }()
        emailSent = true
    } else if input.SendEmail && userEmail == "" {
        fmt.Printf("⚠️ ResetPassword - Send email requested but user has no email address\n")
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "message":    "Password reset successfully",
        "email_sent": emailSent,
        "password_used": input.NewPassword, // For debugging
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

    fmt.Printf("📧 SendCredentials - Sending CURRENT credentials to: %s\n", user.Email)
    fmt.Printf("📧 SendCredentials - Username: %s\n", user.Username)

    // ✅ Send email with CURRENT username and password reset instructions
    go h.sendCurrentCredentialsEmail(user.Email, user.Username, user.FullName, currentUsername)

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "message":    "Current credentials sent successfully",
        "user_id":    id,
        "email":      user.Email,
        "username_sent": user.Username,
        "password_reset_instructions": true,
    })
}

// New function to send current credentials email
func (h *UsersHandler) sendCurrentCredentialsEmail(to, username, fullName, sentBy string) error {
    subject := "Your Current Account Credentials - Internal Inventory Tracker"
    
    htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <style>
        body { 
            font-family: Arial, sans-serif; 
            line-height: 1.6; 
            color: #333; 
            margin: 0;
            padding: 0;
            background-color: #f9f9f9;
        }
        .container {
            max-width: 600px;
            margin: 0 auto;
            background: white;
            border-radius: 10px;
            overflow: hidden;
            box-shadow: 0 4px 15px rgba(0,0,0,0.1);
        }
        .header { 
            background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); 
            color: white; 
            padding: 40px 30px; 
            text-align: center; 
        }
        .header h1 {
            margin: 0;
            font-size: 28px;
            font-weight: 600;
        }
        .header p {
            margin: 10px 0 0 0;
            font-size: 16px;
            opacity: 0.9;
        }
        .content { 
            padding: 40px 30px; 
        }
        .credentials { 
            background: white; 
            padding: 25px; 
            border-radius: 10px; 
            box-shadow: 0 2px 10px rgba(0,0,0,0.1); 
            margin: 20px 0; 
            border: 1px solid #eaeaea;
        }
        .credential-item { 
            margin: 15px 0; 
            padding: 12px; 
            background: #f8f9fa; 
            border-left: 4px solid #667eea; 
            border-radius: 5px;
        }
        .credential-item strong {
            color: #667eea;
            display: block;
            margin-bottom: 5px;
            font-size: 14px;
        }
        .password-help { 
            background: #e3f2fd; 
            border: 1px solid #bbdefb; 
            padding: 20px; 
            border-radius: 8px; 
            margin: 25px 0; 
        }
        .password-help strong {
            color: #1565c0;
            display: block;
            margin-bottom: 8px;
            font-size: 16px;
        }
        .security-note { 
            background: #fff3cd; 
            border: 1px solid #ffeaa7; 
            padding: 20px; 
            border-radius: 8px; 
            margin: 25px 0; 
        }
        .security-note strong {
            color: #856404;
            display: block;
            margin-bottom: 8px;
            font-size: 16px;
        }
        .button { 
            background: #667eea; 
            color: white; 
            padding: 14px 35px; 
            text-decoration: none; 
            border-radius: 6px; 
            display: inline-block; 
            font-weight: 600;
            font-size: 16px;
            text-align: center;
            transition: all 0.3s ease;
            border: none;
            cursor: pointer;
        }
        .button:hover {
            background: #5a6fd8;
            transform: translateY(-2px);
            box-shadow: 0 4px 12px rgba(102, 126, 234, 0.3);
        }
        .footer { 
            text-align: center; 
            padding: 30px; 
            color: #666; 
            font-size: 14px; 
            background: #f8f9fa;
            border-top: 1px solid #eaeaea;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Your Account Credentials</h1>
            <p>Current login information</p>
        </div>
        
        <div class="content">
            <p>Hello <strong>%s</strong>,</p>
            
            <p>Here are your current login credentials for the Internal Inventory Tracker system:</p>
            
            <div class="credentials">
                <div class="credential-item">
                    <strong>🔗 Login URL:</strong>
                    <a href="http://localhost:8081" style="color: #667eea; text-decoration: none;">http://localhost:8081</a>
                </div>
                
                <div class="credential-item">
                    <strong>👤 Username:</strong>
                    <code style="font-size: 16px; background: none; border: none; padding: 0; color: #333;">%s</code>
                </div>
            </div>
            
            <div class="password-help">
                <strong>🔑 Forgot Your Password?</strong>
                <p>If you don't remember your password, please contact your IT administrator to reset it. They can generate a new temporary password for you.</p>
            </div>
            
            <div class="security-note">
                <strong>⚠️ Security Reminder:</strong>
                <p>For security reasons, we cannot include your password in this email. Please use your existing password to log in.</p>
                <p>If you suspect any unauthorized access to your account, contact IT support immediately.</p>
            </div>
            
            <div style="text-align: center; margin: 30px 0;">
                <a href="http://localhost:8081" class="button">Login to System</a>
            </div>
            
            <p style="text-align: center; color: #666; font-size: 14px; margin-top: 30px;">
                <strong>Requested By:</strong> %s (IT Support Team)
            </p>
        </div>
        
        <div class="footer">
            <p>This email was sent automatically. Please do not reply to this message.</p>
            <p><strong>IT Support Team</strong></p>
        </div>
    </div>
</body>
</html>
    `, fullName, username, sentBy)

    textBody := fmt.Sprintf(`
Your Account Credentials - Internal Inventory Tracker

Hello %s,

Here are your current login credentials for the Internal Inventory Tracker system:

CURRENT CREDENTIALS:
====================
Login URL: http://localhost:8081
Username: %s

FORGOT YOUR PASSWORD?
=====================
If you don't remember your password, please contact your IT administrator to reset it. They can generate a new temporary password for you.

SECURITY REMINDER:
==================
For security reasons, we cannot include your password in this email. Please use your existing password to log in.

If you suspect any unauthorized access to your account, contact IT support immediately.

Login to the system: http://localhost:8081

Requested By: %s (IT Support Team)

This email was sent automatically. Please do not reply to this message.

IT Support Team
    `, fullName, username, sentBy)

    return h.EmailService.SendHTMLEmail(to, subject, htmlBody, textBody)
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
// Updated sendWelcomeEmail with the same CSS as reset email
func (h *UsersHandler) sendWelcomeEmail(to, username, password, createdBy string, passwordSetByAdmin bool) error {
    subject := "Welcome to Internal Inventory Tracker - Your Login Credentials"
    
    var passwordMessage string
    if passwordSetByAdmin {
        passwordMessage = "An administrator has set your initial password. Please use the credentials below to log in."
    } else {
        passwordMessage = "Your account has been created. Here are your login credentials:"
    }
    
    htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <style>
        body { 
            font-family: Arial, sans-serif; 
            line-height: 1.6; 
            color: #333; 
            margin: 0;
            padding: 0;
            background-color: #f9f9f9;
        }
        .container {
            max-width: 600px;
            margin: 0 auto;
            background: white;
            border-radius: 10px;
            overflow: hidden;
            box-shadow: 0 4px 15px rgba(0,0,0,0.1);
        }
        .header { 
            background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); 
            color: white; 
            padding: 40px 30px; 
            text-align: center; 
        }
        .header h1 {
            margin: 0;
            font-size: 28px;
            font-weight: 600;
        }
        .header p {
            margin: 10px 0 0 0;
            font-size: 16px;
            opacity: 0.9;
        }
        .content { 
            padding: 40px 30px; 
        }
        .credentials { 
            background: white; 
            padding: 25px; 
            border-radius: 10px; 
            box-shadow: 0 2px 10px rgba(0,0,0,0.1); 
            margin: 20px 0; 
            border: 1px solid #eaeaea;
        }
        .credential-item { 
            margin: 15px 0; 
            padding: 12px; 
            background: #f8f9fa; 
            border-left: 4px solid #667eea; 
            border-radius: 5px;
        }
        .credential-item strong {
            color: #667eea;
            display: block;
            margin-bottom: 5px;
            font-size: 14px;
        }
        .credential-item code {
            font-size: 18px;
            font-weight: bold;
            color: #e74c3c;
            background: none;
            border: none;
            padding: 0;
        }
        .password-warning { 
            background: #fff3cd; 
            border: 1px solid #ffeaa7; 
            padding: 20px; 
            border-radius: 8px; 
            margin: 25px 0; 
        }
        .password-warning strong {
            color: #856404;
            display: block;
            margin-bottom: 8px;
            font-size: 16px;
        }
        .admin-note { 
            background: #e8f5e8; 
            border: 1px solid #c8e6c9; 
            padding: 20px; 
            border-radius: 8px; 
            margin: 25px 0; 
        }
        .admin-note strong {
            color: #2e7d32;
            display: block;
            margin-bottom: 8px;
            font-size: 16px;
        }
        .button { 
            background: #667eea; 
            color: white; 
            padding: 14px 35px; 
            text-decoration: none; 
            border-radius: 6px; 
            display: inline-block; 
            font-weight: 600;
            font-size: 16px;
            text-align: center;
            transition: all 0.3s ease;
            border: none;
            cursor: pointer;
        }
        .button:hover {
            background: #5a6fd8;
            transform: translateY(-2px);
            box-shadow: 0 4px 12px rgba(102, 126, 234, 0.3);
        }
        .footer { 
            text-align: center; 
            padding: 30px; 
            color: #666; 
            font-size: 14px; 
            background: #f8f9fa;
            border-top: 1px solid #eaeaea;
        }
        .login-instructions {
            background: #e3f2fd;
            border: 1px solid #bbdefb;
            padding: 20px;
            border-radius: 8px;
            margin: 20px 0;
        }
        .login-instructions strong {
            color: #1565c0;
            display: block;
            margin-bottom: 8px;
            font-size: 16px;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Welcome to Internal Inventory Tracker</h1>
            <p>Your account has been created successfully</p>
        </div>
        
        <div class="content">
            <p>Hello <strong>%s</strong>,</p>
            
            <p>%s</p>
            
            <div class="credentials">
                <div class="credential-item">
                    <strong>🔗 Login URL:</strong>
                    <a href="http://localhost:8081" style="color: #667eea; text-decoration: none;">http://localhost:8081</a>
                </div>
                
                <div class="credential-item">
                    <strong>👤 Username:</strong>
                    %s
                </div>
                
                <div class="credential-item">
                    <strong>🔑 Password:</strong>
                    <code>%s</code>
                </div>
            </div>
            
            <div class="login-instructions">
                <strong>🚀 Getting Started:</strong>
                <p>Use the credentials above to log in to the system. We recommend exploring the dashboard to familiarize yourself with the available features.</p>
            </div>
            
            <div class="password-warning">
                <strong>⚠️ Security Notice:</strong>
                %s
            </div>
            
            <div class="admin-note">
                <strong>👨‍💼 Account Created By:</strong>
                %s (IT Support Team)
            </div>
            
            <div style="text-align: center; margin: 30px 0;">
                <a href="http://localhost:8081" class="button">Login to System</a>
            </div>
            
            <p style="text-align: center; color: #666; font-size: 14px;">
                If you have any questions or need assistance, please contact the IT support team.
            </p>
        </div>
        
        <div class="footer">
            <p>This email was sent automatically. Please do not reply to this message.</p>
            <p><strong>IT Support Team</strong></p>
        </div>
    </div>
</body>
</html>
    `, username, passwordMessage, username, password, 
       getSecurityMessage(passwordSetByAdmin), createdBy)

    textBody := fmt.Sprintf(`
Welcome to Internal Inventory Tracker

Hello %s,

%s

GETTING STARTED:
================
Login URL: http://localhost:8081
Username: %s
Password: %s

SECURITY NOTICE:
================
%s

Your account was created by: %s (IT Support Team)

Use the credentials above to log in to the system. If you have any questions or need assistance, please contact the IT support team.

This email was sent automatically. Please do not reply to this message.

IT Support Team
    `, username, passwordMessage, username, password, 
       getSecurityMessage(passwordSetByAdmin), createdBy)

    return h.EmailService.SendHTMLEmail(to, subject, htmlBody, textBody)
}

// POST /api/v1/users/{id}/send-password-change
func (h *UsersHandler) SendPasswordChangeEmail(w http.ResponseWriter, r *http.Request) {
    idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/users/")
    idStr = strings.TrimSuffix(idStr, "/send-password-change")
    
    id, err := strconv.ParseInt(idStr, 10, 64)
    if err != nil {
        http.Error(w, "Invalid user ID", http.StatusBadRequest)
        return
    }

    // Get current user for authorization
    currentUserID, ok := r.Context().Value(middleware.ContextUserID).(int)
    if !ok {
        http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
        return
    }

    // Get current user's role
    var currentUserRoleID int
    err = h.Model.DB.QueryRow("SELECT role_id FROM users WHERE id = $1", currentUserID).Scan(&currentUserRoleID)
    if err != nil {
        http.Error(w, "Failed to verify user permissions", http.StatusInternalServerError)
        return
    }

    // Only Admin (1) and IT (2) can send password change emails
    if currentUserRoleID != 1 && currentUserRoleID != 2 {
        http.Error(w, "Only Administrators and IT staff can send password change emails", http.StatusForbidden)
        return
    }

    var currentUsername string
    h.Model.DB.QueryRow("SELECT username FROM users WHERE id = $1", currentUserID).Scan(&currentUsername)

    // Parse request to get the actual password
    var input struct {
        Password string `json:"password"`
    }

    if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
        http.Error(w, "Invalid input", http.StatusBadRequest)
        return
    }

    if input.Password == "" {
        http.Error(w, "Password is required", http.StatusBadRequest)
        return
    }

    // Get user details
    var userEmail, userName string
    err = h.Model.DB.QueryRow(`
        SELECT email, username, full_name 
        FROM users WHERE id = $1
    `, id).Scan(&userEmail, &userName, &userName)
    
    if err != nil {
        if err == sql.ErrNoRows {
            http.Error(w, "User not found", http.StatusNotFound)
            return
        }
        http.Error(w, "Database error", http.StatusInternalServerError)
        return
    }

    // Check if user has email
    if userEmail == "" {
        http.Error(w, "User does not have an email address", http.StatusBadRequest)
        return
    }

    // ✅ Send email with the ACTUAL password (not a generated one)
    go h.sendPasswordResetEmail(userEmail, userName, input.Password, currentUsername, true)

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "message":    "Password change email sent successfully",
        "user_id":    id,
        "email":      userEmail,
    })
}


// Replace the TestEmail handler with this corrected version
func (h *UsersHandler) TestEmail(w http.ResponseWriter, r *http.Request) {
    // Get current user from context (and actually use the variable)
    userID, ok := r.Context().Value(middleware.ContextUserID).(int)
    if !ok {
        http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
        return
    }

    // Use the userID variable to log who is testing
    fmt.Printf("🔍 User %d is testing email service\n", userID)

    // Test email
    err := h.EmailService.SendWelcomeEmail(
        "test@example.com", 
        "testuser", 
        "testpassword123",
    )
    
    if err != nil {
        http.Error(w, fmt.Sprintf(`{"error": "Email failed: %v"}`, err), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "message": "Test email sent successfully",
        "tested_by": userID, // Use the variable here
    })
}