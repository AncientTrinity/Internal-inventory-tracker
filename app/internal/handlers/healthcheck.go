package handlers

import (
	"encoding/json"
	"net/http"
)

// HealthCheckHandler returns system status
func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"status": "available",
		"system_info": map[string]string{
			"environment": "development",
			"version":     "1.0.0",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
	}
}
