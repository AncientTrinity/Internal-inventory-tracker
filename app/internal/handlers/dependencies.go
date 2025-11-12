package handlers

import (
	"database/sql"
	"victortillett.net/internal-inventory-tracker/internal/config"
)

type applicationDependencies struct {
	DB     *sql.DB
	Config config.Config
}
