package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// context keys for storing user info
type contextKey string

const (
	ContextUserID contextKey = "user_id"
	ContextRoleID contextKey = "role_id"
)

// AuthMiddleware verifies JWT and injects claims into context
func AuthMiddleware(jwtSecret string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, `{"error": "missing authorization header"}`, http.StatusUnauthorized)
				return
			}

			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenStr == authHeader {
				http.Error(w, `{"error": "invalid authorization format"}`, http.StatusUnauthorized)
				return
			}

			// Parse and validate token
			token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(jwtSecret), nil
			})

			if err != nil || !token.Valid {
				http.Error(w, `{"error": "invalid or expired token"}`, http.StatusUnauthorized)
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				http.Error(w, `{"error": "invalid token claims"}`, http.StatusUnauthorized)
				return
			}

			// Check expiration manually
			if exp, ok := claims["exp"].(float64); ok {
				if int64(exp) < time.Now().Unix() {
					http.Error(w, `{"error": "token expired"}`, http.StatusUnauthorized)
					return
				}
			}

			// Extract user info
			userID, _ := claims["user_id"].(float64)
			roleID, _ := claims["role_id"].(float64)

			// Add to context
			ctx := context.WithValue(r.Context(), ContextUserID, int(userID))
			ctx = context.WithValue(ctx, ContextRoleID, int(roleID))

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}