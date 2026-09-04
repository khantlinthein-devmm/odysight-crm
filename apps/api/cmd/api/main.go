package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/odysight/crm/config"
	"github.com/odysight/crm/internal/auth"
	"github.com/odysight/crm/internal/bookings"
	"github.com/odysight/crm/internal/cleaners"
	"github.com/odysight/crm/internal/customers"
	"github.com/odysight/crm/internal/leads"
	"github.com/odysight/crm/internal/payments"
	"github.com/odysight/crm/internal/reports"
	"github.com/odysight/crm/internal/servicerecords"
	"github.com/odysight/crm/pkg/database"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	slog.Info("connected to database")

	authRepo := auth.NewRepository(pool)
	if err := authRepo.EnsureSeed(ctx, []auth.SeedUser{
		{Name: "Admin User", Email: "admin@example.com", Password: "admin123", Role: auth.RoleSuperAdmin},
		{Name: "Dispatcher", Email: "dispatch@example.com", Password: "dispatch123", Role: auth.RoleDispatch},
	}); err != nil {
		return err
	}
	authService := auth.NewService(authRepo, cfg.JWTSecret)
	authHandler := auth.NewHandler(authService)
	authorizer := auth.NewAuthorizer(cfg.JWTSecret)

	leadRepo := leads.NewRepository(pool)
	leadService := leads.NewService(leadRepo)
	leadHandler := leads.NewHandler(leadService)

	customerRepo := customers.NewRepository(pool)
	customerService := customers.NewService(customerRepo)
	customerHandler := customers.NewHandler(customerService)

	cleanerRepo := cleaners.NewRepository(pool)
	cleanerService := cleaners.NewService(cleanerRepo)
	cleanerHandler := cleaners.NewHandler(cleanerService)

	bookingRepo := bookings.NewRepository(pool)
	bookingService := bookings.NewService(bookingRepo)
	bookingHandler := bookings.NewHandler(bookingService)

	serviceRecordRepo := servicerecords.NewRepository(pool)
	serviceRecordService := servicerecords.NewService(serviceRecordRepo)
	serviceRecordHandler := servicerecords.NewHandler(serviceRecordService)

	paymentRepo := payments.NewRepository(pool)
	paymentService := payments.NewService(paymentRepo)
	paymentHandler := payments.NewHandler(paymentService)

	reportRepo := reports.NewRepository(pool)
	reportService := reports.NewService(reportRepo)
	reportHandler := reports.NewHandler(reportService)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Mount("/auth", auth.Routes(authHandler))
		r.Group(func(r chi.Router) {
			r.Use(authorizer.Authenticate)
			r.Mount("/leads", leads.Routes(leadHandler, authorizer))
			r.Mount("/customers", customers.Routes(customerHandler, authorizer))
			r.Mount("/cleaners", cleaners.Routes(cleanerHandler, authorizer))
			r.Mount("/bookings", bookings.Routes(bookingHandler, authorizer))
			r.Mount("/service-records", servicerecords.Routes(serviceRecordHandler, authorizer))
			r.Mount("/payments", payments.Routes(paymentHandler, authorizer))
			r.Mount("/reports", reports.Routes(reportHandler, authorizer))
		})
	})

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("starting server", "port", cfg.Port, "env", cfg.AppEnv)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		slog.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
