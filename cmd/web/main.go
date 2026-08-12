// Command borum-api runs a standalone chi HTTP API
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

	"github.com/borumbombum/borum-api/internal/battery"
	"github.com/borumbombum/borum-api/internal/content"
	"github.com/borumbombum/borum-api/internal/tasks"

	"github.com/joho/godotenv"
	flag "github.com/spf13/pflag"
	turso "turso.tech/database/tursogo-serverless"
)

// app holds our dependencies and server state.
type app struct {
	srv         *http.Server
	db          *sql.DB
	css         []byte
	version     string
	errorLogger *log.Logger
}

func main() {
	// Initialize the app
	a := &app{version: readVersion()}

	// Load API routes
	routes := a.apiRoutes()

	if err := loadTemplates(); err != nil {
		log.Fatal(err)
	}

	css, err := concatCSS()
	if err != nil {
		log.Fatal(err)
	}
	a.css = css

	// Articles live in data/articles.json (the same file the command palette
	// fetches); load them into memory before serving requests.
	if err := content.LoadArticles("data/articles.json"); err != nil {
		log.Fatal(err)
	}

	// get configs
	apiAddress := flag.StringP("address", "a", "127.0.0.1", "API Address")
	apiPort := flag.StringP("port", "p", "8091", "API Port")

	flag.Parse()

	// Custom loggers
	infoLog := log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	infoLog.Printf("Starting server someday")
	errorLog := log.New(os.Stdout, "ERROR\t", log.Ldate|log.Ltime|log.Llongfile)

	a.errorLogger = errorLog

	// Bind the listener before announcing anything, so a port conflict is a
	// fatal error instead of a process that is alive but serving nothing.
	addr := *apiAddress + ":" + *apiPort

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
		Handler:      router(a, routes),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
		ErrorLog:     errorLog,
	}

	// Start the scheduler. A zero-job scheduler is suspicious (the heartbeat is
	// auto-registered), so surface it with a warning instead of failing hard.
	// The battery job must be added before Register starts the loop.
	tasks.Add("battery", time.Minute, battery.Refresh)
	go battery.Refresh()
	if n := tasks.Register(); n == 0 {
		log.Println("borum-api: warning: scheduler started with 0 jobs")
	} else {
		log.Printf("borum-api: scheduler started with %d job(s)", n)
	}

	// The listener is bound: print the banner and start serving.
	printRoutes(routes, addr)

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
