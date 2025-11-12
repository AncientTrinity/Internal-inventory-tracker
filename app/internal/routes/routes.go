package routes

import (
	"net/http"

	"victortillett.net/internal-inventory-tracker/internal/handlers"
	"victortillett.net/internal-inventory-tracker/internal/middleware"

	"github.com/go-chi/chi/v5"
)

func SetupRoutes(app *handlers.ApplicationDependencies) http.Handler {
	r := chi.NewRouter()

	// Public routes
	r.Get("/api/v1/healthcheck", app.HealthcheckHandler)
	r.Post("/api/v1/login", app.LoginHandler)

	// Protected routes
	r.Group(func(protected chi.Router) {
		protected.Use(middleware.AuthMiddleware)

		protected.Route("/api/v1/users", func(r chi.Router) {
			r.Get("/", app.GetUsersHandler)
			r.Post("/", app.CreateUserHandler)
		})

		protected.Route("/api/v1/roles", func(r chi.Router) {
			r.Get("/", app.GetRolesHandler)
			r.Post("/", middleware.RequireRole(1, http.HandlerFunc(app.CreateRoleHandler)).ServeHTTP)
		})
	})

	return r



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
