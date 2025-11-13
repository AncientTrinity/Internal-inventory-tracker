package routes

import (
	"net/http"

	"victortillett.net/internal-inventory-tracker/internal/handlers"
	"victortillett.net/internal-inventory-tracker/internal/middleware"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes sets up all routes using chi.Router
func RegisterRoutes(
	usersHandler *handlers.UsersHandler,
	rolesHandler *handlers.RolesHandler,
	//assetsHandler *handlers.AssetsHandler,
	//ticketsHandler *handlers.TicketsHandler,
	authHandler *handlers.AuthHandler,
	jwtSecret string, // Add JWT secret parameter
) http.Handler {
	r := chi.NewRouter()

	// -----------------------
	// Public routes
	// -----------------------
	r.Get("/api/v1/healthcheck", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	r.Post("/api/v1/login", authHandler.Login)
	r.Post("/api/v1/refresh", authHandler.RefreshToken)

	// -----------------------
	// Protected routes
	// -----------------------
	r.Group(func(protected chi.Router) {
		// Pass JWT secret to the middleware
		protected.Use(middleware.AuthMiddleware(jwtSecret))

		// Users
		protected.Route("/api/v1/users", func(r chi.Router) {
			r.Get("/", usersHandler.ListUsers)
			r.Post("/", usersHandler.CreateUser)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", usersHandler.GetUser)
				r.Put("/", usersHandler.UpdateUser)
				r.Delete("/", usersHandler.DeleteUser)
			})
		})

		// Roles
		protected.Route("/api/v1/roles", func(r chi.Router) {
			r.Get("/", rolesHandler.ListRoles)
			r.Post("/", rolesHandler.CreateRole)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", rolesHandler.GetRole)
				r.Put("/", rolesHandler.UpdateRole)
				r.Delete("/", rolesHandler.DeleteRole)
			})
		})

		// Assets (commented out for now - uncomment when handlers are implemented)
		// protected.Route("/api/v1/assets", func(r chi.Router) {
		// 	r.Get("/", assetsHandler.ListAssets)
		// 	r.Post("/", assetsHandler.CreateAsset)
		// 	r.Route("/{id}", func(r chi.Router) {
		// 		r.Get("/", assetsHandler.GetAsset)
		// 		r.Put("/", assetsHandler.UpdateAsset)
		// 		r.Delete("/", assetsHandler.DeleteAsset)
		// 	})
		// 	r.Get("/search", assetsHandler.SearchAssets)
		// 	r.Post("/{id}/logs", assetsHandler.AssetLogs)
		// })

		// Tickets (commented out for now - uncomment when handlers are implemented)
		// protected.Route("/api/v1/tickets", func(r chi.Router) {
		// 	r.Get("/", ticketsHandler.ListTickets)
		// 	r.Post("/", ticketsHandler.CreateTicket)
		// 	r.Route("/{id}", func(r chi.Router) {
		// 		r.Get("/", ticketsHandler.GetTicket)
		// 		r.Put("/", ticketsHandler.UpdateTicket)
		// 	})
		// 	r.Post("/{id}/comments", ticketsHandler.AddComment)
		// 	r.Get("/{id}/comments", ticketsHandler.ListComments)
		// })
	})

	return r
}