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
	"strconv"
	"syscall"
	"time"

	"github.com/borumbombum/borum-api/internal/auth"
	"github.com/borumbombum/borum-api/internal/battery"
	"github.com/borumbombum/borum-api/internal/content"
	"github.com/borumbombum/borum-api/internal/db"
	"github.com/borumbombum/borum-api/internal/experiments"
	"github.com/borumbombum/borum-api/internal/ratelimit"
	"github.com/borumbombum/borum-api/internal/tasks"

	"github.com/joho/godotenv"
	flag "github.com/spf13/pflag"
	turso "turso.tech/database/tursogo-serverless"
)

// app holds our dependencies and server state.
type app struct {
	srv            *http.Server
	db             *sql.DB
	auth           *auth.Service
	css            []byte
	version        string
	errorLogger    *log.Logger
	convertLimiter *ratelimit.Limiter
}

// sessionTTL reads SESSION_TTL_HOURS from the environment, defaulting to 30
// days when unset or unparsable.
func sessionTTL() time.Duration {
	if h := os.Getenv("SESSION_TTL_HOURS"); h != "" {
		if n, err := strconv.Atoi(h); err == nil && n > 0 {
			return time.Duration(n) * time.Hour
		}
	}
	return 720 * time.Hour
}

func main() {
	// Env first: every later step reads configuration from it. Overload so
	// .env is the single source of truth even when a launcher exports the same
	// variables into the process environment.
	if err := godotenv.Overload(); err != nil {
		log.Fatal("Error loading Turso variables")
	}

	// Custom loggers.
	infoLog := log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	infoLog.Printf("Starting server someday")
	errorLog := log.New(os.Stdout, "ERROR\t", log.Ldate|log.Ltime|log.Llongfile)

	// Turso database: apply pending schema migrations, then point the article
	// store and the auth service at it. All three must succeed before any
	// request is served.
	sqlDB := sql.OpenDB(turso.NewConnector(os.Getenv("TURSO_URL"), os.Getenv("TURSO_TOKEN")))
	defer sqlDB.Close()

	if err := db.Migrate(context.Background(), sqlDB); err != nil {
		log.Fatal(err)
	}
	content.Init(sqlDB)
	if err := experiments.Init(sqlDB); err != nil {
		log.Fatal(err)
	}

	// Wire the single-user auth service from the environment. The admin
	// password hash is generated once and stored in ADMIN_PASSWORD_HASH.
	auth.RegisterPassword(os.Getenv("ADMIN_EMAIL"), os.Getenv("ADMIN_PASSWORD_HASH"))

	a := &app{
		db:             sqlDB,
		version:        readVersion(),
		errorLogger:    errorLog,
		convertLimiter: ratelimit.New(5, time.Minute),
		auth: auth.New(sqlDB, auth.Config{
			AdminEmail:        os.Getenv("ADMIN_EMAIL"),
			AdminPasswordHash: os.Getenv("ADMIN_PASSWORD_HASH"),
			SessionTTL:        sessionTTL(),
			CookieName:        "borum_session",
		}),
	}

	// Load templates and the concatenated stylesheet before serving.
	if err := loadTemplates(); err != nil {
		log.Fatal(err)
	}
	css, err := concatCSS()
	if err != nil {
		log.Fatal(err)
	}
	a.css = css

	// Routes depend on the auth service, so they are built only after it.
	routes := a.apiRoutes()

	// Listen address and port flags.
	apiAddress := flag.StringP("address", "a", "127.0.0.1", "API Address")
	apiPort := flag.StringP("port", "p", "8091", "API Port")
	flag.Parse()

	// Bind the listener before announcing anything, so a port conflict is a
	// fatal error instead of a process that is alive but serving nothing.
	addr := *apiAddress + ":" + *apiPort
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", addr, err)
	}

	// Build the server.
	a.srv = &http.Server{
		Handler:      router(a, routes),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
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
	if err := sqlDB.Close(); err != nil {
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
