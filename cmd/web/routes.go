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

// apiRoutes is the single source of truth for the routes.
// The web site lives at the root (/); the API under /api with the health
// endpoint unversioned and versioned routes under /api/v1.
// Both route registration and the startup banner are generated from it.
func (a *app) apiRoutes() []apiRoute {
	return []apiRoute{
		{http.MethodGet, "/", a.homeHandler},
		{http.MethodGet, "/blog/{slug}", a.articleHandler},
		{http.MethodGet, "/tags/{tag}", a.tagHandler},
		{http.MethodGet, "/api/health", a.healthHandler},
		{http.MethodGet, "/api/v1/cms", a.cmsHandler},
	}
}

// router builds the chi router from the provided route table, mounts the
// static assets from static/ and serves the site's 404 for unknown paths.
func router(a *app, routes []apiRoute) http.Handler {
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

	// Static assets served from disk (no embed).
	fs := http.FileServer(http.Dir("static"))
	r.Handle("/assets/*", http.StripPrefix("/", fs))
	r.Handle("/vendor/*", http.StripPrefix("/", fs))
	r.Handle("/app.js", fs)
	r.Handle("/favicon.svg", fs)

	// The data folder is served so the client-side command palette can read
	// data/articles.json directly (the same file the server loads at startup).
	ds := http.FileServer(http.Dir("data"))
	r.Handle("/data/*", http.StripPrefix("/data/", ds))

	// /styles.css is the concat of static/css/*.css, computed at startup
	// (concatCSS in web.go) so the browser gets a single stylesheet.
	r.Handle("/styles.css", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		w.Write(a.css)
	}))

	r.NotFound(a.notFoundHandler)

	return r
}
