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
		protected.Use(middleware.AuthMiddleware)

		// Users
		protected.Route("/api/v1/users", func(r chi.Router) {
			r.Get("/", usersHandler.ListUsers)
			r.Post("/", usersHandler.CreateUser)
			// Example future routes:
			// r.Route("/{id}", func(r chi.Router) {
			//     r.Get("/", usersHandler.GetUserByID)
			//     r.Put("/", usersHandler.UpdateUser)
			//     r.Delete("/", usersHandler.DeleteUser)
			// })
		})

		// Roles
		protected.Route("/api/v1/roles", func(r chi.Router) {
			r.Get("/", rolesHandler.ListRoles)
			r.Post("/", rolesHandler.CreateRole)
			// Future routes:
			// r.Route("/{id}", func(r chi.Router) {
			//     r.Get("/", rolesHandler.GetRole)
			//     r.Put("/", rolesHandler.UpdateRole)
			//     r.Delete("/", rolesHandler.DeleteRole)
			// })
		})

		// Assets
		//protected.Route("/api/v1/assets", func(r chi.Router) {
			//r.Get("/", assetsHandler.ListAssets)
			//r.Post("/", assetsHandler.CreateAsset)
			// Future routes:
			// r.Route("/{id}", func(r chi.Router) {
			//     r.Get("/", assetsHandler.GetAsset)
			//     r.Put("/", assetsHandler.UpdateAsset)
			//     r.Delete("/", assetsHandler.DeleteAsset)
			// })
			// r.Get("/search", assetsHandler.SearchAssets)
			// r.Post("/logs", assetsHandler.AssetLogs)
		})

		// Tickets
		//protected.Route("/api/v1/tickets", func(r chi.Router) {
			//r.Get("/", ticketsHandler.ListTickets)
			//r.Post("/", ticketsHandler.CreateTicket)
			// Future routes:
			// r.Route("/{id}", func(r chi.Router) {
			//     r.Get("/", ticketsHandler.GetTicket)
			//     r.Put("/", ticketsHandler.UpdateTicket)
			// })
			// r.Post("/{id}/comments", ticketsHandler.AddComment)
			// r.Get("/{id}/comments", ticketsHandler.ListComments)
		//})

		// Quick linking helpers
		// r.Get("/api/v1/agents/{id}/assets", assetsHandler.AgentAssets)
	//}) 

	return r
}
