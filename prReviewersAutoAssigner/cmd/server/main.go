package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"prReviewersAutoAssigner/internal/config"
	"prReviewersAutoAssigner/internal/database"
	"prReviewersAutoAssigner/internal/handlers"
	"prReviewersAutoAssigner/internal/repository"
	"prReviewersAutoAssigner/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	db, err := database.NewPostgresDB(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer func() {
		_ = db.Close()
	}()

	if err := database.RunMigrations(cfg); err != nil {
		log.Printf("Failed to run migrations: %v\nSkipping...", err)
	}

	repo := repository.NewPostgresRepository(db)
	svc := service.NewService(repo)
	handler := handlers.NewHandlers(svc)

	server := &http.Server{
		Addr:         ":" + cfg.ServerPort,
		Handler:      handler.Routes(),
		ReadTimeout:  cfg.ServerReadTimeout,
		WriteTimeout: cfg.ServerWriteTimeout,
		IdleTimeout:  cfg.ServerIdleTimeout,
	}

	go func() {
		log.Printf("Server starting on port %s", cfg.ServerPort)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
