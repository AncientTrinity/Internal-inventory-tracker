package main

import (
	"context"
	"fmt"
	"log"
	"net/http" 
	"os"
	"os/signal"
	"syscall"
	"time"

	"victortillett.net/internal-inventory-tracker/internal/db"
	"victortillett.net/internal-inventory-tracker/internal/server"
)

func main() {
	// Connect to DB
	database := db.ConnectDB()
	defer database.Close()

	// Start server
	srv := server.NewServer(database)

	// Run server in a goroutine
	go func() {
		fmt.Printf("Starting server on %s...\n", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop // Wait for interrupt

	fmt.Println("\nShutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}

	fmt.Println("Server stopped gracefully")
}
