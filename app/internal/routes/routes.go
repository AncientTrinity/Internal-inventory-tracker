package routes

import (
	"database/sql"
	"net/http"

	"victortillett.net/internal-inventory-tracker/internal/handlers"
)

func RegisterRoutes(mux *http.ServeMux, db *sql.DB) {
	// Health check
	mux.HandleFunc("/api/v1/health", handlers.HealthCheckHandler(db))

	// Users CRUD
	usersHandler := handlers.NewUsersHandler(db)
	mux.HandleFunc("/api/v1/users", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			usersHandler.ListUsers(w, r)
		case http.MethodPost:
			usersHandler.CreateUser(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/v1/users/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			usersHandler.GetUser(w, r)
		case http.MethodPut:
			usersHandler.UpdateUser(w, r)
		case http.MethodDelete:
			usersHandler.DeleteUser(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

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
