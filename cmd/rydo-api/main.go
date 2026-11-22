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

	"rydo-backend/internal/pilot"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

const requiredSchemaVersion = 3

func newRouter(checkReadiness func(context.Context) error, apis ...*pilot.Server) http.Handler {
	router := gin.New()
	router.HandleMethodNotAllowed = true
	registerSwagger(router)
	for _, api := range apis {
		api.Register(router)
	}
	router.GET("/health/live", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	router.GET("/health/ready", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		if err := checkReadiness(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	return router
}

func checkSchema(ctx context.Context, pool *pgxpool.Pool) error {
	var ready bool
	err := pool.QueryRow(ctx, "SELECT version >= $1 AND NOT dirty FROM schema_migrations",
		requiredSchemaVersion).Scan(&ready)
	if err != nil {
		return err
	}
	if !ready {
		return errors.New("database schema is outdated or dirty")
	}
	return nil
}

func run(ctx context.Context, logger *slog.Logger) int {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		logger.Error("DATABASE_URL is required")
		return 1
	}

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		logger.Error("invalid PostgreSQL configuration")
		return 1
	}
	defer pool.Close()
	api, err := pilot.New(pool, pilot.Config{Origin: os.Getenv("WEB_APP_URL"), JWTKey: os.Getenv("JWT_SIGNING_KEY"), PreviewKey: os.Getenv("PREVIEW_SIGNING_KEY"), LineChannelID: os.Getenv("LINE_LOGIN_CHANNEL_ID")})
	if err != nil {
		logger.Error("invalid pilot configuration", "error", err.Error())
		return 1
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = os.Getenv("APP_PORT")
	}
	if port == "" {
		port = "8080"
	}
	gin.SetMode(gin.ReleaseMode)
	server := &http.Server{
		Addr: ":" + port,
		Handler: newRouter(func(ctx context.Context) error {
			return checkSchema(ctx, pool)
		}, api),
		ReadHeaderTimeout: 5 * time.Second,
	}
	stopped := make(chan error, 1)
	go func() {
		stopped <- server.ListenAndServe()
	}()

	select {
	case err := <-stopped:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("HTTP listener failed", "error", err)
			return 1
		}
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdown); err != nil {
			logger.Error("HTTP shutdown failed", "error", err)
			return 1
		}
	}
	return 0
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	slog.SetDefault(logger)
	code := run(ctx, logger)
	stop()
	os.Exit(code)
}
