package routes

import (
	"database/sql"
	"net/http"

	"victortillett.net/internal-inventory-tracker/internal/handlers"
)

func RegisterRoutes(mux *http.ServeMux, db *sql.DB) {
	// Health check
	mux.HandleFunc("/api/v1/health", handlers.HealthCheckHandler(db))

    // Auth
    //mux.HandleFunc("/api/v1/auth/login", handlers.LoginHandler)
    //mux.HandleFunc("/api/v1/auth/refresh", handlers.RefreshTokenHandler)

    // Users (admins only)
    //mux.HandleFunc("/api/v1/users", handlers.UsersHandler)             // GET list, POST create
    //mux.HandleFunc("/api/v1/users/", handlers.UserByIDHandler)         // GET/PUT/DELETE by ID

    // Assets
    //mux.HandleFunc("/api/v1/assets", handlers.AssetsHandler)           // GET list, POST create
    //mux.HandleFunc("/api/v1/assets/", handlers.AssetByIDHandler)       // GET/PUT/DELETE by ID
    //mux.HandleFunc("/api/v1/assets/", handlers.AssetLogsHandler)       // POST / GET logs

    // Tickets
    //mux.HandleFunc("/api/v1/tickets", handlers.TicketsHandler)         // GET list, POST create
    //mux.HandleFunc("/api/v1/tickets/", handlers.TicketByIDHandler)     // GET/PUT ticket
    //mux.HandleFunc("/api/v1/tickets/", handlers.TicketCommentsHandler) // POST/GET comments

    // Quick linking helpers
    //mux.HandleFunc("/api/v1/agents/", handlers.AgentAssetsHandler)     // GET assets for agent
    //mux.HandleFunc("/api/v1/assets/search", handlers.AssetSearchHandler) // GET search by internal_id
}
