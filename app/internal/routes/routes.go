package routes

import (
	"net/http"

	"victortillett.net/internal-inventory-tracker/internal/handlers"
	"victortillett.net/internal-inventory-tracker/internal/middleware"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes sets up all routes using chi.Router
func RegisterRoutes(
	usersHandler *handlers.UsersHandler,// new users handler
	rolesHandler *handlers.RolesHandler,// new roles handler
	assetsHandler *handlers.AssetsHandler,// new assets handler
	assetServiceHandler *handlers.AssetServiceHandler,// new asset service handler
	assetAssignmentHandler *handlers.AssetAssignmentHandler,// new asset assignment handler
	assetSearchHandler *handlers.AssetSearchHandler,// new asset search handler
	ticketsHandler *handlers.TicketsHandler, //tickets handler
	ticketCommentsHandler *handlers.TicketCommentsHandler, // ticket comments handler
	authHandler *handlers.AuthHandler,// new auth handler
	jwtSecret string,
) http.Handler {
	r := chi.NewRouter()

	// Initialize authorization middleware
	authMiddleware := middleware.NewAuthorizationMiddleware(usersHandler.Model.DB)

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
		protected.Use(middleware.AuthMiddleware(jwtSecret))

		// Users - Admin only
		protected.Route("/api/v1/users", func(r chi.Router) {
			// List users - Admin and IT only
			r.With(authMiddleware.RequirePermission("users:read")).Get("/", usersHandler.ListUsers)// List users
			
			// Create user - Admin only
			r.With(authMiddleware.RequirePermission("users:create")).Post("/", usersHandler.CreateUser)// Create user
			
			r.Route("/{id}", func(r chi.Router) {
				// Get user - Admin and IT only
				r.With(authMiddleware.RequirePermission("users:read")).Get("/", usersHandler.GetUser)// Get user
				
				// Update user - Admin only
				r.With(authMiddleware.RequirePermission("users:update")).Put("/", usersHandler.UpdateUser)// Update user
				
				// Delete user - Admin only
				r.With(authMiddleware.RequirePermission("users:delete")).Delete("/", usersHandler.DeleteUser)// Delete user

				r.With(authMiddleware.RequirePermission("assets:read")).Get("/assets", assetAssignmentHandler.GetUserAssets)
			})
		})

		// Roles - Admin only
		protected.Route("/api/v1/roles", func(r chi.Router) {
			// All role operations require admin permissions
			r.With(authMiddleware.RequirePermission("roles:read")).Get("/", rolesHandler.ListRoles)// List roles
			r.With(authMiddleware.RequirePermission("roles:create")).Post("/", rolesHandler.CreateRole)// Create role
			
			r.Route("/{id}", func(r chi.Router) {
				r.With(authMiddleware.RequirePermission("roles:read")).Get("/", rolesHandler.GetRole)// Get role
				r.With(authMiddleware.RequirePermission("roles:update")).Put("/", rolesHandler.UpdateRole)// Update role
				r.With(authMiddleware.RequirePermission("roles:delete")).Delete("/", rolesHandler.DeleteRole)// Delete role
			})
		})
       
		// Assets
		protected.Route("/api/v1/assets", func(r chi.Router) {
		r.With(authMiddleware.RequirePermission("assets:read")).Get("/", assetsHandler.ListAssets)// List assets
		r.With(authMiddleware.RequirePermission("assets:create")).Post("/", assetsHandler.CreateAsset)// Create asset
		r.With(authMiddleware.RequirePermission("assets:read")).Get("/available", assetAssignmentHandler.GetAvailableAssets) // Available assets
		r.With(authMiddleware.RequirePermission("assets:update")).Post("/bulk-assign", assetAssignmentHandler.BulkAssignAssets) // Bulk assign assets
		r.With(authMiddleware.RequirePermission("assets:read")).Get("/search", assetSearchHandler.SearchAssets)// Search assets
		r.With(authMiddleware.RequirePermission("assets:read")).Get("/stats", assetSearchHandler.GetAssetStats)// Asset stats
		r.With(authMiddleware.RequirePermission("assets:read")).Get("/types", assetSearchHandler.GetAssetTypes)// Asset types
		r.With(authMiddleware.RequirePermission("assets:read")).Get("/manufacturers", assetSearchHandler.GetManufacturers)// Manufacturers
		
		r.Route("/{id}", func(r chi.Router) {
			r.With(authMiddleware.RequirePermission("assets:read")).Get("/", assetsHandler.GetAsset)// Get asset
			r.With(authMiddleware.RequirePermission("assets:update")).Put("/", assetsHandler.UpdateAsset)// Update asset
			r.With(authMiddleware.RequirePermission("assets:delete")).Delete("/", assetsHandler.DeleteAsset)// Delete asset
			r.With(authMiddleware.RequirePermission("assets:update")).Post("/assign", assetAssignmentHandler.AssignAsset)// Assign asset
			r.With(authMiddleware.RequirePermission("assets:update")).Post("/unassign", assetAssignmentHandler.UnassignAsset)// Unassign asset
			
			// Service logs for specific asset
			r.Route("/service-logs", func(r chi.Router) {
				r.With(authMiddleware.RequirePermission("assets:update")).Post("/", assetServiceHandler.CreateServiceLog)// Create service log
				r.With(authMiddleware.RequirePermission("assets:read")).Get("/", assetServiceHandler.GetServiceLogs)// Get service logs

				//Tickets route
				
			})
		})
	})
		// Tickets routes
	  	protected.Route("/api/v1/tickets", func(r chi.Router) {
		r.With(authMiddleware.RequirePermission("tickets:read")).Get("/", ticketsHandler.ListTickets)
		r.With(authMiddleware.RequirePermission("tickets:create")).Post("/", ticketsHandler.CreateTicket)
		r.With(authMiddleware.RequirePermission("tickets:read")).Get("/stats", ticketsHandler.GetTicketStats)
		
		r.Route("/{id}", func(r chi.Router) {
			r.With(authMiddleware.RequirePermission("tickets:read")).Get("/", ticketsHandler.GetTicket)
			r.With(authMiddleware.RequirePermission("tickets:update")).Put("/", ticketsHandler.UpdateTicket)
			
			// Ticket status updates
			r.With(authMiddleware.RequirePermission("tickets:update")).Post("/status", ticketsHandler.UpdateTicketStatus)
			r.With(authMiddleware.RequirePermission("tickets:assign")).Post("/reassign", ticketsHandler.ReassignTicket)
			
			// Ticket comments
			r.Route("/comments", func(r chi.Router) {
				r.With(authMiddleware.RequirePermission("tickets:read")).Get("/", ticketCommentsHandler.GetComments)
				r.With(authMiddleware.RequirePermission("tickets:update")).Post("/", ticketCommentsHandler.CreateComment)
			})
		})
	})



	// Individual service log routes
	protected.Route("/api/v1/service-logs", func(r chi.Router) {
		r.Route("/{id}", func(r chi.Router) {
			r.With(authMiddleware.RequirePermission("assets:read")).Get("/", assetServiceHandler.GetServiceLog)// Get service log
		})
	})

    	// Individual comment routes
	protected.Route("/api/v1/comments", func(r chi.Router) {
		r.Route("/{id}", func(r chi.Router) {
			r.With(authMiddleware.RequirePermission("tickets:update")).Put("/", ticketCommentsHandler.UpdateComment)
			r.With(authMiddleware.RequirePermission("tickets:update")).Delete("/", ticketCommentsHandler.DeleteComment)
		})
	})

	})

	return r
}