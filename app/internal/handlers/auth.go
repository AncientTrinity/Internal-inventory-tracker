package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// AuthHandler handles authentication routes
type AuthHandler struct {
	DB        *sql.DB
	JWTSecret string
}

// NewAuthHandler creates a new AuthHandler with config
func NewAuthHandler(db *sql.DB, jwtSecret string) *AuthHandler {
	return &AuthHandler{DB: db, JWTSecret: jwtSecret}
}

// Credentials struct for login input
type Credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Claims struct for JWT
type Claims struct {
	UserID int    `json:"user_id"`
	RoleID int    `json:"role_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// Login endpoint
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var creds Credentials
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		h.errorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate input
	if creds.Email == "" || creds.Password == "" {
		h.errorResponse(w, "Email and password are required", http.StatusBadRequest)
		return
	}

	// Query user by email
	var userID int
	var hashedPassword string
	var roleID int
	err := h.DB.QueryRow(`
		SELECT id, password, role_id FROM users WHERE email = $1
	`, creds.Email).Scan(&userID, &hashedPassword, &roleID)
	
	if err != nil {
		if err == sql.ErrNoRows {
			h.errorResponse(w, "Invalid email or password", http.StatusUnauthorized)
			return
		}
		h.errorResponse(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Compare password
	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(creds.Password)); err != nil {
		h.errorResponse(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	// Create JWT claims
	expirationTime := time.Now().Add(24 * time.Hour) // Extended to 24 hours for better UX
	claims := &Claims{
		UserID: userID,
		RoleID: roleID,
		Email:  creds.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "internal-inventory-tracker",
		},
	}

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(h.JWTSecret))
	if err != nil {
		h.errorResponse(w, "Could not create token", http.StatusInternalServerError)
		return
	}

	// Return token
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"token":      tokenString,
		"expires_at": expirationTime,
		"user_id":    userID,
		"role_id":    roleID,
	})
}

// RefreshToken endpoint
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.errorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Parse token without validation to get claims
	token, err := jwt.ParseWithClaims(body.Token, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(h.JWTSecret), nil
	})
	if err != nil || !token.Valid {
		h.errorResponse(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		h.errorResponse(w, "Invalid claims", http.StatusUnauthorized)
		return
	}

	// Verify user still exists and is active
	var userExists bool
	err = h.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)`, claims.UserID).Scan(&userExists)
	if err != nil || !userExists {
		h.errorResponse(w, "User no longer exists", http.StatusUnauthorized)
		return
	}

	// Create new token with extended expiration
	expirationTime := time.Now().Add(24 * time.Hour)
	claims.ExpiresAt = jwt.NewNumericDate(expirationTime)
	claims.IssuedAt = jwt.NewNumericDate(time.Now())
	
	newToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	newTokenString, err := newToken.SignedString([]byte(h.JWTSecret))
	if err != nil {
		h.errorResponse(w, "Could not refresh token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"token":      newTokenString,
		"expires_at": expirationTime,
	})
}

// Helper method for consistent error responses
func (h *AuthHandler) errorResponse(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}