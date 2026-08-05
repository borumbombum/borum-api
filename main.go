package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

const (
	apiVersion = "v1"
	apiPort    = "8091"
)

// APIServer holds our dependencies and server state,
// removing the need for a global pocketbaseApp variable.
type APIServer struct {
	pb    *pocketbase.PocketBase
	srv   *http.Server
	start sync.Once
}

type apiRoute struct {
	method  string
	path    string
	handler http.HandlerFunc
}

// healthHandler uses the injected PocketBase instance.
func (api *APIServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, err := api.pb.CountRecords("_superusers")
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"status":"degraded","service":"borum-api","database":"unreachable"}`))
		return
	}
	w.Write([]byte(`{"status":"ok","service":"borum-api","database":"ok"}`))
}

// apiRoutes is the single source of truth for the custom API.
// Both route registration and the startup banner are generated from it.
func (api *APIServer) apiRoutes() []apiRoute {
	return []apiRoute{
		{http.MethodGet, "/", api.healthHandler},
		{http.MethodGet, "/cms", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"endpoint":"cms"}`))
		}},
	}
}

// routes sets up the chi router and endpoints.
func (api *APIServer) routes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	for _, rt := range api.apiRoutes() {
		path := rt.path
		if path != "/" {
			path = "/" + apiVersion + path
		}
		r.Method(rt.method, path, rt.handler)
	}

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"status":"not_found"}`))
	})

	return r
}

// startAPIServer spins up the HTTP server. It is guarded by sync.Once
// because OnServe can fire multiple times (e.g., HTTP + HTTPS).
func (api *APIServer) startAPIServer() {
	api.start.Do(func() {
		routes := api.apiRoutes()

		api.srv = &http.Server{
			Addr:         "127.0.0.1:" + apiPort,
			Handler:      api.routes(),
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  15 * time.Second,
		}

		go func() {
			if err := api.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Printf("API server error: %v", err)
			}
		}()

		// Keep the 50ms delay in a goroutine so our API banner
		// prints cleanly *after* the PocketBase startup banner.
		go api.printRoutes(routes)
	})
}

func (api *APIServer) printRoutes(routes []apiRoute) {
	time.Sleep(50 * time.Millisecond)
	baseURL := "http://127.0.0.1:" + apiPort
	fmt.Println("Borum API:")
	for i, rt := range routes {
		path := rt.path
		if path != "/" {
			path = "/" + apiVersion + path
		}
		branch := "├─"
		if i == len(routes)-1 {
			branch = "└─"
		}
		fmt.Printf("%s %s %s  %s%s\n", branch, rt.method, path, baseURL, path)
	}
}

func main() {
	pb := pocketbase.New()

	api := &APIServer{
		pb: pb,
	}

	// Triggered after DB initializes, safe to start custom API.
	pb.OnServe().BindFunc(func(se *core.ServeEvent) error {
		api.startAPIServer()
		return se.Next()
	})

	// Graceful shutdown with a 5-second context timeout.
	pb.OnTerminate().BindFunc(func(te *core.TerminateEvent) error {
		if api.srv != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := api.srv.Shutdown(ctx); err != nil {
				log.Printf("Borum API shutdown error: %v", err)
			}
		}
		return te.Next()
	})

	if err := pb.Start(); err != nil {
		log.Fatal(err)
	}
}
