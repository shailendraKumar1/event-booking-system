package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"event-booking-system/internal/cache"
	"event-booking-system/internal/config"
	"event-booking-system/internal/email"
	"event-booking-system/internal/handler"
	"event-booking-system/internal/memory"
	"event-booking-system/internal/middleware"
	"event-booking-system/internal/router"
	"event-booking-system/internal/service"
	"event-booking-system/internal/worker"
)

func main() {
	cfg := config.Get()

	// Infrastructure layer
	var mailer email.Mailer
	if cfg.SMTPHost != "" && cfg.EmailFrom != "" {
		mailer = email.NewSMTPMailer(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUsername, cfg.SMTPPassword, cfg.EmailFrom)
		log.Printf("[main] SMTP configured: host=%s from=%s", cfg.SMTPHost, cfg.EmailFrom)
	} else {
		mailer = &email.NoopMailer{}
		log.Println("[main] SMTP not configured; emails will be silently skipped")
	}

	// Worker pool (Producer-Consumer pattern for async background tasks)
	emailHandler := worker.NewEmailJobHandler(mailer)
	pool := worker.NewPool(cfg.WorkerPoolSize, cfg.JobQueueSize, emailHandler)

	// Repository layer (in-memory; swap for SQL adapters without touching services)
	userRepo := memory.NewUserRepo()
	eventRepo := memory.NewEventRepo()
	bookingRepo := memory.NewBookingRepo()

	// Cache layer (Decorator pattern)
	eventCache := cache.NewEventCache(cfg.CacheTTL)

	// Service layer (business logic; DIP: depends on abstractions)
	authService := service.NewAuthService(userRepo, cfg)
	eventService := service.NewEventService(eventRepo, bookingRepo, eventCache, pool)
	bookingService := service.NewBookingService(eventRepo, bookingRepo, pool)

	// Handler layer (HTTP concerns only)
	authHandler := handler.NewAuthHandler(authService)
	eventHandler := handler.NewEventHandler(eventService)
	bookingHandler := handler.NewBookingHandler(bookingService)
	healthHandler := handler.NewHealthHandler(pool)

	// Rate limiter
	limiter := middleware.NewRateLimiter(cfg.RateLimitRPS, cfg.RateLimitBurst)

	// Router (route registration + middleware chain)
	h := router.New(authService, authHandler, eventHandler, bookingHandler, healthHandler, limiter)

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("[main] shutting down...")
		pool.Shutdown()
		os.Exit(0)
	}()

	addr := ":" + cfg.Port
	log.Printf("[main] Event Booking System API listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, h))
}
