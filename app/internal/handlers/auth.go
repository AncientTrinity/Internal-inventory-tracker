package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// Secret key (in production, load from env)
var jwtSecret = []byte("supersecretkey123")

type Credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (a *applicationDependencies) serverErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}


// POST /api/v1/login
func (a *applicationDependencies) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var creds Credentials
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	

	// Query user by email
	row := a.DB.QueryRow(`SELECT id, password, role_id FROM users WHERE email = $1`, creds.Email)
	var userID int
	var hashedPassword string
	var roleID int

	err := row.Scan(&userID, &hashedPassword, &roleID)
	if err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	// Compare password
	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(creds.Password)); err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	// Generate JWT
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"role_id": roleID,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	})

	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		http.Error(w, "could not generate token", http.StatusInternalServerError)
		return
	}

	a.writeJSON(w, http.StatusOK, envelope{
		"token": tokenString,
	}, nil)
}
