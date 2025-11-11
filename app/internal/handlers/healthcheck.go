package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

// HealthCheckHandler returns system status, including DB connectivity
func HealthCheckHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := "available"
		dbStatus := "ok"

		// Ping the DB to check connection
		if err := db.Ping(); err != nil {
			dbStatus = "unreachable"
			status = "degraded"
		}

		data := map[string]interface{}{
			"status": status,
			"system_info": map[string]string{
				"environment": "development",
				"version":     "1.0.0",
			},
			"database": dbStatus,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(data); err != nil {
			http.Error(w, "Failed to write response", http.StatusInternalServerError)
		}
	}
}
