package main

import (
    "fmt"
    "log"
    "net/http"
    "os"
    "time"

    "victortillett.net/internal-inventory-tracker/internal/routes"
)

func main() {
    port := os.Getenv("PORT")
    if port == "" {
        port = "8081"
    }

    mux := http.NewServeMux()

    // Register all routes
    routes.RegisterRoutes(mux)

    srv := &http.Server{
        Addr:         ":" + port,
        Handler:      mux,
        ReadTimeout:  10 * time.Second,
        WriteTimeout: 10 * time.Second,
        IdleTimeout:  120 * time.Second,
    }

    fmt.Printf("Starting server on port %s...\n", port)
    log.Fatal(srv.ListenAndServe())
}
