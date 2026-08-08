// Router setup: apiRoutes is the single source of truth for the custom API.
// router() builds the chi router from it and printRoutes (server.go) feeds the
// startup banner from the same slice. apiVersion is applied in exactly one
// place, routePath().
package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// apiRoute declares one endpoint: the HTTP method, the path (prefixed with
// apiVersion by routePath(), except for "/"), and the handler to call.
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

// routePath prefixes a route path with the API version, except for the root
// "/" route which is served unprefixed. This is the only place apiVersion is
// applied; both the router and the startup banner use it.
func routePath(path string) string {
	if path == "/" {
		return "/"
	}
	return "/" + apiVersion + path
}

// router builds the chi router from the provided route table.
func router(routes []apiRoute) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	for _, rt := range routes {
		r.Method(rt.method, routePath(rt.path), rt.handler)
	}

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"status":"not_found"}`))
	})

	return r
}
