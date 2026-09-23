package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/odysight/crm/config"
	"github.com/odysight/crm/internal/attendance"
	"github.com/odysight/crm/internal/audit"
	"github.com/odysight/crm/internal/auth"
	"github.com/odysight/crm/internal/bookings"
	"github.com/odysight/crm/internal/checklists"
	"github.com/odysight/crm/internal/cleaners"
	"github.com/odysight/crm/internal/contracts"
	"github.com/odysight/crm/internal/customers"
	"github.com/odysight/crm/internal/expenses"
	"github.com/odysight/crm/internal/feedback"
	"github.com/odysight/crm/internal/invoices"
	"github.com/odysight/crm/internal/leads"
	"github.com/odysight/crm/internal/line"
	"github.com/odysight/crm/internal/notifications"
	"github.com/odysight/crm/internal/payments"
	"github.com/odysight/crm/internal/portal"
	"github.com/odysight/crm/internal/quotes"
	"github.com/odysight/crm/internal/reports"
	"github.com/odysight/crm/internal/servicerecords"
	"github.com/odysight/crm/internal/settings"
	"github.com/odysight/crm/internal/sites"
	"github.com/odysight/crm/internal/users"
	"github.com/odysight/crm/pkg/database"
	"github.com/odysight/crm/pkg/mailer"
	apmw "github.com/odysight/crm/pkg/middleware"
	"github.com/odysight/crm/pkg/migrate"
	"github.com/odysight/crm/pkg/response"
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

	if cfg.RunMigrations {
		if err := migrate.Up(ctx, pool, cfg.MigrationsDir); err != nil {
			return err
		}
	}

	authRepo := auth.NewRepository(pool)
	if cfg.SeedEnabled {
		seeds := []auth.SeedUser{
			{Name: cfg.SeedAdminName, Email: cfg.SeedAdminEmail, Password: cfg.SeedAdminPass, Role: auth.RoleSuperAdmin},
		}
		if cfg.SeedDispatchPass != "" {
			seeds = append(seeds, auth.SeedUser{Name: cfg.SeedDispatchName, Email: cfg.SeedDispatchEmail, Password: cfg.SeedDispatchPass, Role: auth.RoleDispatch})
		}
		if err := authRepo.EnsureSeed(ctx, seeds); err != nil {
			return err
		}
		slog.Info("seed check complete", "enabled", true)
	}
	authService := auth.NewServiceWithTTL(authRepo, cfg.JWTSecret, cfg.TokenTTL)
	authHandler := auth.NewHandler(authService, !cfg.IsDev(), cfg.TokenTTL)
	authorizer := auth.NewAuthorizer(cfg.JWTSecret)

	customerRepo := customers.NewRepository(pool)
	customerService := customers.NewService(customerRepo)
	customerHandler := customers.NewHandler(customerService)

	leadRepo := leads.NewRepository(pool)
	leadService := leads.NewService(leadRepo, customerService)
	leadHandler := leads.NewHandler(leadService)

	cleanerRepo := cleaners.NewRepository(pool)
	cleanerService := cleaners.NewService(cleanerRepo)
	cleanerHandler := cleaners.NewHandler(cleanerService)

	bookingRepo := bookings.NewRepository(pool)

	// Demo portal customer (idempotent) so a seeded install always has someone
	// who can sign in at /portal with somchai@smileclean.com.
	if cfg.SeedEnabled {
		if err := seedPortalDemoCustomer(ctx, pool, bookingRepo); err != nil {
			return err
		}
		slog.Info("portal demo seed check complete", "enabled", true)
	}

	// Recurring-booking job: fires on boot then hourly, cloning each due
	// recurring booking forward and disabling its source row.
	recurrence := bookings.NewRecurrenceRunner(bookingRepo, time.Hour)
	go recurrence.Run(ctx)

	feedbackRepo := feedback.NewRepository(pool)
	feedbackService := feedback.NewService(feedbackRepo)
	feedbackHandler := feedback.NewHandler(feedbackService)

	serviceRecordRepo := servicerecords.NewRepository(pool)
	serviceRecordService := servicerecords.NewService(serviceRecordRepo)
	serviceRecordHandler := servicerecords.NewHandler(serviceRecordService)

	reportRepo := reports.NewRepository(pool)
	reportService := reports.NewService(reportRepo)
	reportHandler := reports.NewHandler(reportService)

	userRepo := users.NewRepository(pool)
	userService := users.NewService(userRepo)
	userHandler := users.NewHandler(userService)

	auditRepo := audit.NewRepository(pool)
	auditService := audit.NewService(auditRepo)
	auditHandler := audit.NewHandler(auditService)

	settingsRepo := settings.NewRepository(pool)
	if err := settingsRepo.EnsureSeed(ctx, settings.Defaults()); err != nil {
		return err
	}
	settingsService := settings.NewService(settingsRepo)
	settingsHandler := settings.NewHandler(settingsService)

	newInvoiceMailer := func() invoices.Emailer {
		cfg, err := settingsService.SMTPConfig(context.Background())
		if err != nil {
			return nil
		}
		return mailer.New(cfg)
	}

	newNotificationMailer := func() notifications.Emailer {
		return newInvoiceMailer()
	}

	notifRepo := notifications.NewRepository(pool)
	notifService := notifications.NewService(notifRepo, settingsService, newNotificationMailer)
	notifHandler := notifications.NewHandler(notifService)

	bookingService := bookings.NewService(bookingRepo, notifService)
	bookingHandler := bookings.NewHandler(bookingService)

	portalRepo := portal.NewRepository(pool)
	portalService := portal.NewService(portalRepo, feedbackService, cfg.JWTSecret).WithNotifier(notifService)
	portalHandler := portal.NewHandler(portalService, !cfg.IsDev(), cfg.TokenTTL)
	portalAuthorizer := portal.NewAuthorizer(cfg.JWTSecret)

	invoiceRepo := invoices.NewRepository(pool)
	invoiceService := invoices.NewService(invoiceRepo, settingsService, newInvoiceMailer, notifService)
	invoiceHandler := invoices.NewHandler(invoiceService)

	// Overdue-invoice reminder job: fires on boot, then every hour. Stamping
	// reminder_sent_at keeps it idempotent. Single-instance only.
	reminders := invoices.NewReminderRunner(invoiceRepo, settingsService, newInvoiceMailer, notifService, time.Hour)
	go reminders.Run(ctx)

	paymentRepo := payments.NewRepository(pool)
	paymentService := payments.NewService(paymentRepo, invoiceService)
	paymentHandler := payments.NewHandler(paymentService)

	attendanceRepo := attendance.NewRepository(pool)
	attendanceService := attendance.NewService(attendanceRepo)
	attendanceHandler := attendance.NewHandler(attendanceService)

	expenseRepo := expenses.NewRepository(pool)
	expenseService := expenses.NewService(expenseRepo)
	expenseHandler := expenses.NewHandler(expenseService)

	siteRepo := sites.NewRepository(pool)
	siteService := sites.NewService(siteRepo)
	siteHandler := sites.NewHandler(siteService)

	contractRepo := contracts.NewRepository(pool)
	contractService := contracts.NewService(contractRepo)
	contractHandler := contracts.NewHandler(contractService)

	quoteRepo := quotes.NewRepository(pool)
	quoteService := quotes.NewService(quoteRepo)
	quoteHandler := quotes.NewHandler(quoteService)

	checklistRepo := checklists.NewRepository(pool)
	checklistService := checklists.NewService(checklistRepo)
	checklistHandler := checklists.NewHandler(checklistService, checklists.NewPhotoStore(cfg.UploadDir, cfg.MaxUploadMB))

	// LINE OA → auto-lead. Mounted outside the auth group: authenticity
	// comes from the X-Line-Signature HMAC. With no channel secret
	// configured the endpoint answers 503 and everything else is unchanged.
	lineClient := line.NewClient(cfg.LineChannelAccessToken)
	lineService := line.NewService(leadRepo, lineClient, lineClient, cfg.LineAutoReply)
	lineHandler := line.NewHandler(lineService, cfg.LineChannelSecret)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(cfg.RequestTimeout))
	r.Use(apmw.SecurityHeaders)
	r.Use(apmw.CORS(cfg.CORSOrigins))
	r.Use(apmw.RateLimit(cfg.RateLimitRPM, cfg.TrustedProxyIPs))

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	r.Get("/ready", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), cfg.ReadyTimeout)
		defer cancel()
		if err := pool.Ping(ctx); err != nil {
			response.Error(w, http.StatusServiceUnavailable, "database not ready")
			return
		}
		response.JSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Mount("/auth", auth.Routes(authHandler, authorizer, apmw.LoginRateLimit(cfg.LoginRateLimitRPM, cfg.TrustedProxyIPs)))
		r.Mount("/portal", portal.Routes(portalHandler, portalAuthorizer))
		r.Mount("/line", line.Routes(lineHandler))
		r.Group(func(r chi.Router) {
			r.Use(authorizer.Authenticate)
			r.Use(auditLog(pool, cfg.ReadyTimeout))
			r.Mount("/leads", leads.Routes(leadHandler, authorizer))
			r.Mount("/customers", customers.Routes(customerHandler, authorizer))
			r.Mount("/cleaners", cleaners.Routes(cleanerHandler, authorizer))
			r.Mount("/bookings", bookings.Routes(bookingHandler, authorizer))
			r.Mount("/service-records", servicerecords.Routes(serviceRecordHandler, authorizer))
			r.Mount("/invoices", invoices.Routes(invoiceHandler, authorizer))
			r.Mount("/payments", payments.Routes(paymentHandler, authorizer))
			r.Mount("/reports", reports.Routes(reportHandler, authorizer))
			r.Mount("/users", users.Routes(userHandler, authorizer))
			r.Mount("/audit-logs", audit.Routes(auditHandler, authorizer))
			r.Mount("/settings", settings.Routes(settingsHandler, authorizer))
			r.Mount("/feedback", feedback.Routes(feedbackHandler, authorizer))
			r.Mount("/notifications", notifications.Routes(notifHandler, authorizer))
			r.Mount("/attendance", attendance.Routes(attendanceHandler, authorizer))
			r.Mount("/expenses", expenses.Routes(expenseHandler, authorizer))
			r.Mount("/sites", sites.Routes(siteHandler, authorizer))
			r.Mount("/contracts", contracts.Routes(contractHandler, authorizer))
			r.Mount("/quotes", quotes.Routes(quoteHandler, authorizer))
			r.Mount("/checklists", checklists.Routes(checklistHandler, authorizer))
		})
	})

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("starting server", "port", cfg.Port, "env", cfg.AppEnv)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		slog.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

// auditLog records mutating requests best-effort into audit_logs.
func auditLog(pool *pgxpool.Pool, timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
			if r.Method != http.MethodPost && r.Method != http.MethodPatch && r.Method != http.MethodDelete {
				return
			}
			// High-frequency GPS pings from the cleaner app must not flood audit_logs.
			if r.URL.Path == "/api/v1/cleaners/me/location" {
				return
			}
			id, _ := auth.IdentityFromContext(r.Context())
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			_, _ = pool.Exec(ctx,
				`INSERT INTO audit_logs (user_id, action, resource, resource_id) VALUES ($1, $2, $3, $4)`,
				id.UserID, r.Method, r.URL.Path, resourceIDFromPath(r.URL.Path))
		})
	}
}

// resourceIDFromPath extracts the trailing numeric ID from a well-formed
// /resource/{id} path, returning nil when the path carries no ID.
func resourceIDFromPath(path string) *int64 {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 {
		return nil
	}
	id, err := strconv.ParseInt(parts[len(parts)-1], 10, 64)
	if err != nil {
		return nil
	}
	return &id
}
