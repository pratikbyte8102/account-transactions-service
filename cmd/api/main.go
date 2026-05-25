package main

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"

	"account-transactions-service/internal/config"
	httpapi "account-transactions-service/internal/http"
	"account-transactions-service/internal/repository"
	"account-transactions-service/internal/service"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		logger.Error("open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(30 * time.Minute)

	if err := pingDatabase(ctx, db); err != nil {
		logger.Error("connect database", "error", err)
		os.Exit(1)
	}
	if err := repository.Migrate(ctx, db); err != nil {
		logger.Error("migrate database", "error", err)
		os.Exit(1)
	}

	accountRepo := repository.NewAccountPostgresRepository(db)
	transactionRepo := repository.NewTransactionPostgresRepository(db)
	appService := service.New(accountRepo, transactionRepo)
	handler := httpapi.NewHandler(appService, appService, logger)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handler.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("server listening", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown failed", "error", err)
		os.Exit(1)
	}
	logger.Info("server stopped")
}

func pingDatabase(ctx context.Context, db *sql.DB) error {
	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return db.PingContext(pingCtx)
}
