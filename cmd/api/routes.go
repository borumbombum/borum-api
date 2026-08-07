// Router setup: apiRoutes is the single source of truth for the custom API,
// and routes() builds the chi router from it. New endpoints are added to the
// table in apiRoutes, which also feeds the startup banner in server.go.
package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// apiRoute declares one endpoint: the HTTP method, the path (prefixed with
// apiVersion by routes(), except for "/"), and the handler to call.
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
		{http.MethodGet, "/web/admin", a.webHandler},
		{http.MethodGet, "/cms", a.cmsHandler},
	}
}

// routes sets up the chi router and endpoints.
func (a *app) routes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	for _, rt := range a.apiRoutes() {
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
