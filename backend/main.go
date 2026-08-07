package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/watch-tower-org/watchdog/backend/internal/config"
	"github.com/watch-tower-org/watchdog/backend/internal/database"
	"github.com/watch-tower-org/watchdog/backend/internal/logger"
	"github.com/watch-tower-org/watchdog/backend/pkg"
	"github.com/watch-tower-org/watchdog/backend/router"
)

func main() {
	// Load environment variables
	cfg := config.LoadConfig()

	// Initialize the logger with the configured log level
	logger.InitLogger(cfg.LoggerConfig)

	// Initialize the database connection
	db, err := database.New(cfg.Database)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to connect to the database")
	}
	defer db.Close()

	app, err := pkg.NewApplication(cfg, db)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to initialize the application")
	}

	// Initialize the router with the application context
	r, err := router.AppRouter(app)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to initialize the router")
	}

	// Graceful shutdown on SIGINT / SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	srv := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: r,
	}

	go func() {
		logger.Info().Msgf("Server will run at port: %s", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("Failed to start the server")
		}
	}()

	<-quit
	logger.Info().Msg("Shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		if err := srv.Shutdown(shutdownCtx); err != nil {
			logger.Error().Err(err).Msg("HTTP server shutdown error")
		}
		app.Shutdown()
		close(done)
	}()

	select {
	case <-done:
		logger.Info().Msg("Clean shutdown complete")
		logger.Close()
	case <-shutdownCtx.Done():
		logger.Warn().Msg("Shutdown timed out — forcing exit")
		logger.Close()
	}
}
