package main

import (
	"context"
	"errors"
	stdhttp "net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/realtime-chat/internal/config"
	"github.com/realtime-chat/internal/pkg/hasher"
	pkgjwt "github.com/realtime-chat/internal/pkg/jwt"
	"github.com/realtime-chat/internal/pkg/logger"
	pkgvalidator "github.com/realtime-chat/internal/pkg/validator"
	pgrepo "github.com/realtime-chat/internal/repository/postgres"
	redisrepo "github.com/realtime-chat/internal/repository/redis"
	"github.com/realtime-chat/internal/service"
	httpx "github.com/realtime-chat/internal/transport/http"
	wsx "github.com/realtime-chat/internal/transport/websocket"
)

func main() {
	if err := run(); err != nil {
		// Logger may not be ready yet on early-startup errors; fall back to stderr.
		_, _ = os.Stderr.WriteString("fatal: " + err.Error() + "\n")
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	log := logger.New(cfg.Logger.Level, cfg.Logger.Format)
	log.Info("starting realtime-chat", "addr", cfg.HTTP.Addr)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	pool, err := pgrepo.Connect(ctx, cfg.Postgres.DSN,
		int32(cfg.Postgres.MaxConns),
		int32(cfg.Postgres.MinConns),
		cfg.Postgres.MaxConnLifetime,
	)
	if err != nil {
		return err
	}
	defer pool.Close()
	log.Info("postgres connected")

	rds, err := redisrepo.Connect(ctx, cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		return err
	}
	defer rds.Close()
	log.Info("redis connected")

	users := pgrepo.NewUserRepository(pool)
	rooms := pgrepo.NewRoomRepository(pool)
	messages := pgrepo.NewMessageRepository(pool)
	bus := redisrepo.NewEventBus(rds, cfg.Redis.Channel)

	tokens := pkgjwt.NewManager(cfg.JWT.Secret, cfg.JWT.TTL, cfg.JWT.Issuer)
	bcrypt := hasher.NewBcrypt(0)
	validator := pkgvalidator.New()

	authSvc := service.NewAuthService(users, bcrypt, tokens)
	roomSvc := service.NewRoomService(rooms)
	msgSvc := service.NewMessageService(messages, roomSvc, bus, log)

	hub := wsx.NewHub(bus, log)
	hubErr := make(chan error, 1)
	go func() { hubErr <- hub.Run(ctx) }()

	wsHandler := wsx.NewHandler(hub, authSvc, roomSvc, msgSvc, log, cfg.HTTP.AllowedOrigins)

	handler := httpx.NewRouter(httpx.Deps{
		Logger:    log,
		Validator: validator,
		Auth:      authSvc,
		Rooms:     roomSvc,
		Messages:  msgSvc,
		Users:     users,
		WebSocket: wsHandler,
		Origins:   cfg.HTTP.AllowedOrigins,
	})

	srv := &stdhttp.Server{
		Addr:         cfg.HTTP.Addr,
		Handler:      handler,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: 0, // disable for /ws long-lived connections; per-write deadlines applied inside handlers
	}

	serverErr := make(chan error, 1)
	go func() {
		log.Info("http server listening", "addr", cfg.HTTP.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, stdhttp.ErrServerClosed) {
			serverErr <- err
		}
		close(serverErr)
	}()

	select {
	case <-ctx.Done():
		log.Info("shutdown signal received")
	case err := <-serverErr:
		if err != nil {
			return err
		}
	case err := <-hubErr:
		if err != nil {
			log.Error("hub stopped", "err", err)
		}
	}

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer cancelShutdown()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("http shutdown", "err", err)
	}

	// Give the hub a brief moment to drain after server stop.
	select {
	case <-hubErr:
	case <-time.After(2 * time.Second):
	}
	log.Info("bye")
	return nil
}
