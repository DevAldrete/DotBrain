package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/devaldrete/dotbrain/internal/api"
	"github.com/devaldrete/dotbrain/internal/db/sqlc"
	"github.com/devaldrete/dotbrain/internal/scheduler"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// 1. Initialize Structured Logging (slog)
	// We use slog because the project instructions specify structured logging
	// and discourage fmt.Println for sensitive information.
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// 2. Load Configuration (Environment Variables)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		logger.Info("PORT environment variable not set, defaulting to 8080")
	}

	// 3. Initialize API Router
	dbURL := os.Getenv("DATABASE_URL")
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		logger.Error(fmt.Sprintf("Unable to connect to database: %v", err))
	}
	a := api.NewAPI(pool)

	// 3a. Crash Recovery: fail any runs left in running/pending from a previous crash
	if err := a.RecoverStaleRuns(context.Background()); err != nil {
		logger.Error("crash recovery failed", "error", err)
	}

	// 3b. Scheduler: cron-based workflow triggers
	queries := db.New(pool)
	sched := scheduler.New(queries, a.TriggerWorkflow)
	if err := sched.LoadFromDB(context.Background()); err != nil {
		logger.Error("scheduler: failed to load schedules from DB", "error", err)
	}
	sched.Start()
	a.SetScheduler(sched)

	// 3c. Watchdog: periodically fail runs that exceed max duration
	maxDuration := 1 * time.Hour
	if v := os.Getenv("RUN_MAX_DURATION"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			maxDuration = d
		} else {
			logger.Error("invalid RUN_MAX_DURATION, using default", "value", v, "error", err)
		}
	}
	watchdogInterval := 5 * time.Minute
	if v := os.Getenv("WATCHDOG_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			watchdogInterval = d
		} else {
			logger.Error("invalid WATCHDOG_INTERVAL, using default", "value", v, "error", err)
		}
	}
	watchdogCtx, watchdogCancel := context.WithCancel(context.Background())
	go a.RunWatchdog(watchdogCtx, maxDuration, watchdogInterval)
	logger.Info("watchdog started", "maxDuration", maxDuration, "interval", watchdogInterval)

	router := a.NewRouter()

	// 4. Configure HTTP Server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: router,
		// Good practice to set timeouts to prevent slowloris attacks
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 5. Start Server in a Goroutine
	// This allows the main goroutine to listen for OS signals and shutdown gracefully
	go func() {
		logger.Info(fmt.Sprintf("Starting server on port %s", port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error(fmt.Sprintf("listen: %s\n", err))
			os.Exit(1)
		}
	}()

	// 6. Graceful Shutdown Implementation (Critical for Kubernetes)
	// We wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 5 seconds.
	quit := make(chan os.Signal, 1)
	// kill (no param) default send syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL but can't be caught, so don't need add it
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutdown Server ...")

	// Stop the watchdog and scheduler before shutting down the server
	watchdogCancel()
	sched.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error(fmt.Sprintf("Server Shutdown: %s", err))
		os.Exit(1)
	}

	// catching ctx.Done(). timeout of 5 seconds.
	<-ctx.Done()
	logger.Info("Server exiting gracefully")
}
