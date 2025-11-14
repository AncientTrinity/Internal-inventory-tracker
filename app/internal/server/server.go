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
	usersHandler := handlers.NewUsersHandler(db)// New users handler
	rolesHandler := handlers.NewRolesHandler(db)// New roles handler
	assetsHandler := handlers.NewAssetsHandler(db)// New assets handler
	assetServiceHandler := handlers.NewAssetServiceHandler(db)// New asset service handler
	assetAssignmentHandler := handlers.NewAssetAssignmentHandler(db) // New asset assignment handler
	authHandler := handlers.NewAuthHandler(db, cfg.JWTSecret)// New auth handler

	// Register routes using handlers and JWT secret
	router := routes.RegisterRoutes(usersHandler, rolesHandler, assetsHandler, assetServiceHandler, assetAssignmentHandler,
		                           authHandler, cfg.JWTSecret) // Register routes

	return &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}// Return configured server
}