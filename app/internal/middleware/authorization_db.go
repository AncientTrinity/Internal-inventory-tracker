package middleware

import (
	//"context"
	"database/sql"
	//"fmt"
	"net/http"
)

// AuthorizationMiddleware struct with database connection
type AuthorizationMiddleware struct {
	DB *sql.DB
}

// NewAuthorizationMiddleware creates a new authorization middleware
func NewAuthorizationMiddleware(db *sql.DB) *AuthorizationMiddleware {
	return &AuthorizationMiddleware{DB: db}
}

// RequireRoleWithDB checks if user has required role (with database lookup)
func (am *AuthorizationMiddleware) RequireRole(requiredRole string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			roleID, ok := r.Context().Value(ContextRoleID).(int)
			if !ok {
				http.Error(w, `{"error": "Unauthorized - no role information"}`, http.StatusUnauthorized)
				return
			}

			// Query database for role name
			var roleName string
			err := am.DB.QueryRowContext(r.Context(), 
				"SELECT name FROM roles WHERE id = $1", roleID).Scan(&roleName)
			if err != nil {
				if err == sql.ErrNoRows {
					http.Error(w, `{"error": "Role not found"}`, http.StatusForbidden)
					return
				}
				http.Error(w, `{"error": "Internal server error"}`, http.StatusInternalServerError)
				return
			}

			if roleName != requiredRole {
				http.Error(w, `{"error": "Forbidden - insufficient permissions"}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequirePermission checks if user has specific permission
func (am *AuthorizationMiddleware) RequirePermission(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			roleID, ok := r.Context().Value(ContextRoleID).(int)
			if !ok {
				http.Error(w, `{"error": "Unauthorized - no role information"}`, http.StatusUnauthorized)
				return
			}

			// Check if role has the required permission
			var hasPermission bool
			err := am.DB.QueryRowContext(r.Context(), `
				SELECT EXISTS(
					SELECT 1 FROM role_permissions rp
					JOIN permissions p ON rp.permission_id = p.id
					WHERE rp.role_id = $1 AND p.name = $2
				)`, roleID, permission).Scan(&hasPermission)
			
			if err != nil {
				http.Error(w, `{"error": "Internal server error"}`, http.StatusInternalServerError)
				return
			}

			if !hasPermission {
				http.Error(w, `{"error": "Forbidden - insufficient permissions"}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}