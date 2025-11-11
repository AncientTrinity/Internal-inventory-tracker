package server

import (
	"database/sql"
	"net/http"
	"os"
	"time"

	"victortillett.net/internal-inventory-tracker/internal/routes"
)

func NewServer(db *sql.DB) *http.Server {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	mux := http.NewServeMux()
	routes.RegisterRoutes(mux, db)

	return &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
}
