package middleware

import (
	"context"
	"fmt"
	"net/http"
	//"strconv"
)

// RequireRole middleware checks if user has the required role
func RequireRole(requiredRole string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get role ID from context (set by AuthMiddleware)
			roleID, ok := r.Context().Value(ContextRoleID).(int)
			if !ok {
				http.Error(w, `{"error": "Unauthorized - no role information"}`, http.StatusUnauthorized)
				return
			}

			// Convert role ID to role name (you'll need to query the database)
			roleName, err := getRoleNameByID(r.Context(), roleID)
			if err != nil {
				http.Error(w, `{"error": "Internal server error"}`, http.StatusInternalServerError)
				return
			}

			// Check if user has the required role
			if roleName != requiredRole {
				http.Error(w, `{"error": "Forbidden - insufficient permissions"}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAnyRole middleware checks if user has any of the required roles
func RequireAnyRole(requiredRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			roleID, ok := r.Context().Value(ContextRoleID).(int)
			if !ok {
				http.Error(w, `{"error": "Unauthorized - no role information"}`, http.StatusUnauthorized)
				return
			}

			roleName, err := getRoleNameByID(r.Context(), roleID)
			if err != nil {
				http.Error(w, `{"error": "Internal server error"}`, http.StatusInternalServerError)
				return
			}

			// Check if user has any of the required roles
			hasRole := false
			for _, requiredRole := range requiredRoles {
				if roleName == requiredRole {
					hasRole = true
					break
				}
			}

			if !hasRole {
				http.Error(w, `{"error": "Forbidden - insufficient permissions"}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Helper function to get role name from role ID
func getRoleNameByID(ctx context.Context, roleID int) (string, error) {
	// This is a simplified version - in production, you might want to cache this
	// or get it from the database. For now, we'll use a simple mapping.
	
	// Role ID to name mapping (you should replace this with a database query)
	roleMap := map[int]string{
		1: "admin",
		2: "it", 
		3: "staff",
		4: "agent",
		5: "viewer",
	}

	roleName, exists := roleMap[roleID]
	if !exists {
		return "", fmt.Errorf("role not found for ID: %d", roleID)
	}

	return roleName, nil
}