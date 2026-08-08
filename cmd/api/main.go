// Command borum-api runs a standalone chi HTTP API on 127.0.0.1:8091.
// No PocketBase: the data layer (Turso) is wired in separately.
package main

import (
	"context"
	"database/sql"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/borumbombum/borum-api/internal/tasks"
	"github.com/joho/godotenv"

	// _ "turso.tech/database/tursogo"
	turso "turso.tech/database/tursogo-serverless"
)

const (
	// apiVersion prefixes all custom API routes except the root health check.
	apiVersion = "v1"
	// apiPort is the only port the API listens on.
	apiPort = "8091"
)

// app holds our dependencies and server state.
type app struct {
	srv *http.Server
	db  *sql.DB
}

func main() {
	a := &app{}
	routes := a.apiRoutes()

	// Bind the listener before announcing anything, so a port conflict is a
	// fatal error instead of a process that is alive but serving nothing.
	addr := "127.0.0.1:" + apiPort
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", addr, err)
	}

	// Turso stuff
	err = godotenv.Load()
	if err != nil {
		log.Fatal("Error loading Turso variables")
	}
	// log.Printf("TURSO_TOKEN %v", os.Getenv("TURSO_TOKEN"))
	db := sql.OpenDB(turso.NewConnector(os.Getenv("TURSO_URL"), os.Getenv("TURSO_TOKEN")))
	a.db = db
	defer db.Close()

	// start the server
	a.srv = &http.Server{
		Handler:      router(routes),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	// Start the scheduler. A zero-job scheduler is suspicious (the heartbeat is
	// auto-registered), so surface it with a warning instead of failing hard.
	if n := tasks.Register(); n == 0 {
		log.Println("borum-api: warning: scheduler started with 0 jobs")
	} else {
		log.Printf("borum-api: scheduler started with %d job(s)", n)
	}

	// The listener is bound: print the banner and start serving.
	printRoutes(routes)

	go func() {
		if err := a.srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Printf("API server error: %v", err)
		}
	}()

	// Wait for SIGINT/SIGTERM, then shut down gracefully with a 5s timeout.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	log.Println("Closing Turso connection...")
	if err := db.Close(); err != nil {
		log.Printf("error closing Turso: %v", err)
	} else {
		log.Printf("Turso connection closed successfully")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := a.srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Borum API shutdown error: %v", err)
	}
	log.Println("Borum API stopped")
}
