//cmd/main.go

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/matheuslm7/notifiq/internal/api"
	"github.com/matheuslm7/notifiq/internal/auth"
	"github.com/matheuslm7/notifiq/internal/campaigns"
	"github.com/matheuslm7/notifiq/internal/config"
	"github.com/matheuslm7/notifiq/internal/contacts"
	database "github.com/matheuslm7/notifiq/internal/database"
	"github.com/matheuslm7/notifiq/internal/emailsettings"
	"github.com/matheuslm7/notifiq/internal/members"
	"github.com/matheuslm7/notifiq/internal/metrics"
	"github.com/matheuslm7/notifiq/internal/notifier"
	"github.com/matheuslm7/notifiq/internal/queue"
	"github.com/matheuslm7/notifiq/internal/scheduler"
	"github.com/matheuslm7/notifiq/internal/security"
	"github.com/matheuslm7/notifiq/internal/worker"
	"github.com/redis/go-redis/v9"
)

func main() {
	cfg := config.Load()

	rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	defer rdb.Close()

	db, err := database.Open(database.Config{
		Addr:     cfg.MySQLAddr,
		Database: cfg.MySQLDatabase,
		User:     cfg.MySQLUser,
		Password: cfg.MySQLPassword,
	})
	if err != nil {
		log.Fatalf("mysql initialization failed: %v", err)
	}
	defer db.Close()

	dbCtx, dbCancel := context.WithTimeout(context.Background(), 90*time.Second)
	if err := waitForDependency(dbCtx, "mysql", func(ctx context.Context) error { return database.Ping(ctx, db) }); err != nil {
		dbCancel()
		log.Fatalf("mysql connection failed: %v", err)
	}
	if err := database.Migrate(dbCtx, db, cfg.MigrationPath); err != nil {
		dbCancel()
		log.Fatalf("database migration failed: %v", err)
	}
	dbCancel()

	tokenService, err := auth.NewTokenService(cfg.JWTSecret, 15*time.Minute)
	if err != nil {
		log.Fatalf("auth initialization failed: %v", err)
	}
	authService, err := auth.NewService(database.NewUserRepository(db), tokenService)
	if err != nil {
		log.Fatalf("auth service initialization failed: %v", err)
	}
	secretBox, err := security.NewSecretBox(cfg.EmailEncryptionKey)
	if err != nil {
		log.Fatalf("email security initialization failed: %v", err)
	}
	emailService, err := emailsettings.NewService(database.NewUserRepository(db), secretBox)
	if err != nil {
		log.Fatalf("email settings initialization failed: %v", err)
	}
	authService.SetPasswordResetNotifier(notifier.PasswordResetSender{Resolver: emailService, BaseURL: cfg.PasswordResetURL})

	pingCtx, pingCancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer pingCancel()
	if err := waitForDependency(pingCtx, "redis", func(ctx context.Context) error { return rdb.Ping(ctx).Err() }); err != nil {
		log.Fatalf("redis connection failed: %v", err)
	}

	q := &queue.RedisQueue{Client: rdb, ListKey: cfg.EventQueue}
	deliveryMetrics := &metrics.Metrics{}
	emailJobs := &queue.EmailJobQueue{Client: rdb, Key: cfg.EmailJobQueue, DeadLetterKey: cfg.EmailDeadLetterQueue}
	schedulerService, err := scheduler.New(emailJobs, database.NewUserRepository(db))
	if err != nil {
		log.Fatalf("scheduler initialization failed: %v", err)
	}

	handler := &api.EventHandler{Publisher: q}
	authHandler := &api.AuthHandler{Service: authService}
	auditRepository := database.NewUserRepository(db)
	authService.SetAuditRecorder(auditRepository)
	emailHandler := &api.EmailSettingsHandler{Service: emailService, MFA: authService, Audit: auditRepository}
	webhookHandler := &api.WebhookHandler{Secret: cfg.WebhookSecret, Repository: database.NewUserRepository(db)}
	contactsService, err := contacts.NewService(database.NewUserRepository(db))
	if err != nil {
		log.Fatalf("contacts service initialization failed: %v", err)
	}
	contactsHandler := &api.ContactsHandler{Service: contactsService}
	campaignsService, err := campaigns.NewService(database.NewUserRepository(db))
	if err != nil {
		log.Fatalf("campaigns service initialization failed: %v", err)
	}
	campaignsHandler := &api.CampaignsHandler{Service: campaignsService}
	membersService, err := members.NewService(database.NewUserRepository(db))
	if err != nil {
		log.Fatalf("members service initialization failed: %v", err)
	}
	membersHandler := &api.MembersHandler{Service: membersService, Audit: auditRepository, Invites: database.NewUserRepository(db), InviteNotifier: notifier.InviteSender{Resolver: emailService, BaseURL: cfg.InviteURL}}
	mux := http.NewServeMux()
	mux.HandleFunc("/events", handler.CreateEvent)
	mux.Handle("/auth/register", security.RateLimit(security.RedisRateLimiter{Client: rdb, Limit: 5, Window: time.Minute}, "ratelimit:register", http.HandlerFunc(authHandler.Register)))
	mux.Handle("/auth/login", security.RateLimit(security.RedisRateLimiter{Client: rdb, Limit: 10, Window: time.Minute}, "ratelimit:login", http.HandlerFunc(authHandler.Login)))
	mux.HandleFunc("/auth/refresh", authHandler.Refresh)
	mux.HandleFunc("/auth/logout", authHandler.Logout)
	mux.Handle("/auth/password-reset/request", security.RateLimit(security.RedisRateLimiter{Client: rdb, Limit: 3, Window: time.Minute}, "ratelimit:password-reset", http.HandlerFunc(authHandler.RequestPasswordReset)))
	mux.Handle("/auth/password-reset/confirm", security.RateLimit(security.RedisRateLimiter{Client: rdb, Limit: 5, Window: time.Minute}, "ratelimit:password-reset-confirm", http.HandlerFunc(authHandler.ResetPassword)))
	mux.Handle("/auth/password/change", auth.RequireAuth(tokenService, http.HandlerFunc(authHandler.ChangePassword)))
	mux.Handle("/me", auth.RequireAuth(tokenService, http.HandlerFunc(api.Me)))
	mux.Handle("/auth/mfa/setup", auth.RequireAuth(tokenService, http.HandlerFunc(authHandler.SetupMFA)))
	mux.Handle("/auth/mfa/confirm", auth.RequireAuth(tokenService, http.HandlerFunc(authHandler.ConfirmMFA)))
	mux.Handle("/settings/email", auth.RequireAuth(tokenService, http.HandlerFunc(emailHandler.Get)))
	mux.Handle("/settings/email/save", auth.RequireRoles(tokenService, []string{"owner", "admin"}, http.HandlerFunc(emailHandler.Save)))
	mux.Handle("/contacts", auth.RequireRolesByMethod(tokenService, []string{"owner", "admin", "editor", "viewer"}, []string{"owner", "admin", "editor"}, http.HandlerFunc(contactsHandler.List)))
	mux.Handle("/contacts/create", auth.RequireRoles(tokenService, []string{"owner", "admin", "editor"}, http.HandlerFunc(contactsHandler.Create)))
	mux.Handle("/contacts/import", auth.RequireRoles(tokenService, []string{"owner", "admin", "editor"}, http.HandlerFunc(contactsHandler.Import)))
	mux.Handle("/templates", auth.RequireRolesByMethod(tokenService, []string{"owner", "admin", "editor", "viewer"}, []string{"owner", "admin", "editor"}, http.HandlerFunc(campaignsHandler.Templates)))
	mux.Handle("/campaigns", auth.RequireRolesByMethod(tokenService, []string{"owner", "admin", "editor", "viewer"}, []string{"owner", "admin", "editor"}, http.HandlerFunc(campaignsHandler.Campaigns)))
	mux.Handle("/campaign-recipients", auth.RequireRoles(tokenService, []string{"owner", "admin", "editor"}, http.HandlerFunc(campaignsHandler.AddRecipients)))
	mux.Handle("/campaigns/update", auth.RequireRoles(tokenService, []string{"owner", "admin", "editor"}, http.HandlerFunc(campaignsHandler.Update)))
	mux.Handle("/campaigns/cancel", auth.RequireRoles(tokenService, []string{"owner", "admin", "editor"}, http.HandlerFunc(campaignsHandler.Cancel)))
	mux.Handle("/campaigns/duplicate", auth.RequireRoles(tokenService, []string{"owner", "admin", "editor"}, http.HandlerFunc(campaignsHandler.Duplicate)))
	mux.Handle("/members", auth.RequireRolesByMethod(tokenService, []string{"owner", "admin", "editor", "viewer"}, []string{"owner", "admin"}, http.HandlerFunc(membersHandler.Members)))
	mux.Handle("/members/remove", auth.RequireRoles(tokenService, []string{"owner", "admin"}, http.HandlerFunc(membersHandler.Remove)))
	mux.Handle("/members/invite", auth.RequireRoles(tokenService, []string{"owner", "admin"}, http.HandlerFunc(membersHandler.Invite)))
	mux.Handle("/members/invite/accept", security.RateLimit(security.RedisRateLimiter{Client: rdb, Limit: 5, Window: time.Minute}, "ratelimit:invite-accept", http.HandlerFunc(membersHandler.AcceptInvite)))
	mux.HandleFunc("/metrics", api.MetricsHandler(deliveryMetrics))
	mux.Handle("/metrics/delivery", auth.RequireAuth(tokenService, api.DeliveryMetricsHandler(database.NewUserRepository(db))))
	mux.Handle("/metrics/delivery/history", auth.RequireAuth(tokenService, api.DeliveryHistoryHandler(database.NewUserRepository(db))))
	dlqHandler := &api.DLQHandler{Store: emailJobs}
	mux.Handle("/admin/dlq", dlqHandler.Router(tokenService))
	mux.HandleFunc("/webhooks/email", webhookHandler.Email)
	mux.HandleFunc("/healthz", healthHandler)
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if err := rdb.Ping(r.Context()).Err(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "not ready"})
			return
		}
		healthHandler(w, r)
	})

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	go q.StartConsumer(ctx)
	go schedulerService.Run(ctx)
	processor := &worker.Processor{Sender: notifier.OrganizationSender{Resolver: emailService}, Repository: database.NewUserRepository(db), Metrics: deliveryMetrics}
	go processor.RunQueue(ctx, emailJobs)

	srv := &http.Server{Addr: cfg.HTTPAddr, Handler: mux}
	serverErr := make(chan error, 1)
	go func() {
		log.Printf("api listening on %s", cfg.HTTPAddr)
		serverErr <- srv.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server failed: %v", err)
		}
	case <-ctx.Done():
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("http shutdown failed: %v", err)
		}
	}
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func waitForDependency(ctx context.Context, name string, check func(context.Context) error) error {
	var lastErr error
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		if err := check(ctx); err == nil {
			return nil
		} else {
			lastErr = err
			log.Printf("waiting for %s: %v", name, err)
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("%s did not become ready: %w (last error: %v)", name, ctx.Err(), lastErr)
		case <-ticker.C:
		}
	}
}
