// Router setup: apiRoutes is the single source of truth for the custom API.
// router() builds the chi router from it and printRoutes (server.go) feeds the
// startup banner from the same slice.
package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// apiRoute declares one endpoint: the HTTP method, the path and the handler to call.
type apiRoute struct {
	method  string
	path    string
	handler http.HandlerFunc
}

// apiRoutes is the single source of truth for the custom API.
// Both route registration and the startup banner are generated from it.
func (a *app) apiRoutes() []apiRoute {
	return []apiRoute{
		{http.MethodGet, "/", a.healthHandler},
		{http.MethodGet, "/cms", a.cmsHandler},
	}
}

// router builds the chi router from the provided route table.
func router(routes []apiRoute) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"http://localhost:5173"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	}))

	// Register all routes
	for _, rt := range routes {
		r.Method(rt.method, rt.path, rt.handler)
	}

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"status":"not_found"}`))
	})

	return r
}
