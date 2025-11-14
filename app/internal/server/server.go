package server

import (
	"database/sql"
	"net/http"
	//"os"
	"time"

	"victortillett.net/internal-inventory-tracker/internal/handlers"
	"victortillett.net/internal-inventory-tracker/internal/routes"
	"victortillett.net/internal-inventory-tracker/internal/config"
)

func NewServer(db *sql.DB, cfg *config.Config) *http.Server {
	port := cfg.Port
	if port == "" {
		port = "8081"
	}

	// Initialize handlers with config
	usersHandler := handlers.NewUsersHandler(db)
	rolesHandler := handlers.NewRolesHandler(db)
	assetsHandler := handlers.NewAssetsHandler(db)
	authHandler := handlers.NewAuthHandler(db, cfg.JWTSecret)

	// Register routes using handlers and JWT secret
	router := routes.RegisterRoutes(usersHandler, rolesHandler, assetsHandler, authHandler, cfg.JWTSecret)

	return &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
}