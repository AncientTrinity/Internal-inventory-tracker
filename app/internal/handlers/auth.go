package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// Secret key (in production, load this from environment variable)
var jwtSecret = []byte("supersecretkey123")

// AuthHandler handles authentication routes
type AuthHandler struct {
	DB *sql.DB
}

// NewAuthHandler creates a new AuthHandler
func NewAuthHandler(db *sql.DB) *AuthHandler {
	return &AuthHandler{DB: db}
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
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Query user by email
	var userID int
	var hashedPassword string
	var roleID int
	err := h.DB.QueryRow(`SELECT id, password, role_id FROM users WHERE email = $1`, creds.Email).
		Scan(&userID, &hashedPassword, &roleID)
	if err != nil {
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	// Compare password
	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(creds.Password)); err != nil {
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	// Create JWT claims
	expirationTime := time.Now().Add(1 * time.Hour)
	claims := &Claims{
		UserID: userID,
		RoleID: roleID,
		Email:  creds.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		http.Error(w, "Could not create token", http.StatusInternalServerError)
		return
	}

	// Return token
	json.NewEncoder(w).Encode(map[string]string{"token": tokenString})
}

// RefreshToken endpoint
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	token, err := jwt.ParseWithClaims(body.Token, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil || !token.Valid {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		http.Error(w, "Invalid claims", http.StatusUnauthorized)
		return
	}

	// Create new token
	expirationTime := time.Now().Add(1 * time.Hour)
	claims.ExpiresAt = jwt.NewNumericDate(expirationTime)
	newToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	newTokenString, err := newToken.SignedString(jwtSecret)
	if err != nil {
		http.Error(w, "Could not refresh token", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"token": newTokenString})
}
