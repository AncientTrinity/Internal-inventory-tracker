package server

import (
	"database/sql"
	"net/http"
	"os"
	"time"

	"victortillett.net/internal-inventory-tracker/internal/handlers"
	"victortillett.net/internal-inventory-tracker/internal/routes"
)

func NewServer(db *sql.DB) *http.Server {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	// Initialize handlers
	usersHandler := handlers.NewUsersHandler(db)
	rolesHandler := handlers.NewRolesHandler(db)
	authHandler := handlers.NewAuthHandler(db)

	// Register routes using handlers
	router := routes.RegisterRoutes(usersHandler, rolesHandler, authHandler)

	return &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
}
