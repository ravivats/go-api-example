package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-api-example/handler"
	"go-api-example/storage"

	"github.com/gorilla/mux"
)

func main() {
	// Setup signal handling for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Get database connection URL from environment variable
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL environment variable is not set")
	}

	// Initialize storage
	store, err := storage.NewPostgresStore(ctx, databaseURL)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	log.Println("Database connection established and schema initialized.")

	// Initialize handlers
	accountHandler := handler.NewAccountHandler(store)
	transactionHandler := handler.NewTransactionHandler(store)

	// Setup router
	r := mux.NewRouter()
	r.HandleFunc("/accounts", accountHandler.CreateAccountHandler).Methods("POST")
	r.HandleFunc("/accounts/{account_id}", accountHandler.GetAccountHandler).Methods("GET")
	r.HandleFunc("/transactions", transactionHandler.CreateTransactionHandler).Methods("POST")

	// Create and start server
	server := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	go func() {
		log.Println("Starting server on port 8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe error: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-ctx.Done()
	log.Println("Shutting down server...")

	// Create a context for shutdown with a timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}

	log.Println("Server gracefully stopped")
}
