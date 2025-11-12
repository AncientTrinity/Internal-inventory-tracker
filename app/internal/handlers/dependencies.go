package handlers

import (
	"database/sql"
	"victortillett.net/internal-inventory-tracker/internal/config"
)

// ApplicationDependencies holds all shared app dependencies.
type ApplicationDependencies struct {
	DB     *sql.DB
	Config *config.Config
}
