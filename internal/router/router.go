package router

import (
	"net/http"

	"event-booking-system/internal/domain"
	"event-booking-system/internal/handler"
	"event-booking-system/internal/middleware"
	"event-booking-system/internal/service"
)

// New builds the full HTTP handler with route registration and middleware stack.
// Route registration is explicit to keep the dependency graph visible.
func New(
	authService *service.AuthService,
	authHandler *handler.AuthHandler,
	eventHandler *handler.EventHandler,
	bookingHandler *handler.BookingHandler,
	healthHandler *handler.HealthHandler,
	limiter *middleware.TokenBucket,
) http.Handler {
	mux := http.NewServeMux()

	// Public routes
	mux.HandleFunc("GET /health", healthHandler.Health)
	mux.HandleFunc("POST /api/auth/register", authHandler.Register)
	mux.HandleFunc("POST /api/auth/login", authHandler.Login)
	mux.HandleFunc("GET /api/events", eventHandler.List)
	mux.HandleFunc("GET /api/events/{id}", eventHandler.Get)

	// Organizer-only routes
	authMw := middleware.JWTAuth(authService)
	organizerMw := middleware.RequireRole(domain.RoleOrganizer)
	mux.Handle("POST /api/events", middleware.Chain(
		http.HandlerFunc(eventHandler.Create),
		authMw, organizerMw,
	))
	mux.Handle("PATCH /api/events/{id}", middleware.Chain(
		http.HandlerFunc(eventHandler.Update),
		authMw, organizerMw,
	))

	// Customer-only routes
	customerMw := middleware.RequireRole(domain.RoleCustomer)
	mux.Handle("POST /api/events/{id}/bookings", middleware.Chain(
		http.HandlerFunc(bookingHandler.Book),
		authMw, customerMw,
	))
	mux.Handle("GET /api/bookings/me", middleware.Chain(
		http.HandlerFunc(bookingHandler.MyBookings),
		authMw, customerMw,
	))

	// Global middleware stack (outermost first)
	return middleware.Chain(mux,
		middleware.CORS,
		middleware.Recovery,
		middleware.Logging,
		middleware.RateLimit(limiter),
	)
}
