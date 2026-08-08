// HTTP handlers. All handlers are methods on *app so they can share
// dependencies (e.g. a database client) added to the app struct. Add new
// endpoints in apiRoutes (routes.go) and their handlers here; split into
// per-domain files as they grow.
package main

import (
	"fmt"
	"net/http"
	"time"
)

// healthHandler reports liveness of the API process. It has no data-layer
// dependency yet; when a database (Turso) is wired in, add a connectivity check
// here and flip to 503 "degraded" when it fails.
func (a *app) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	start := time.Now()
	err := a.db.PingContext(r.Context())
	latency := time.Since(start)

	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(fmt.Sprintf(`{"status": "error", "error": "db-unreachable", "latancy": %d}`, latency.Milliseconds())))
		return
	}

	w.Write([]byte(fmt.Sprintf(`{"status":"ok","db":"ok","latancy":%d}`, latency.Milliseconds())))
}

// webHandler serves a placeholder HTML page for a future admin web UI.
func (a *app) webHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`<body><h1>This is an html response for the unprotected, you read well "unprotected" admin section</h1></body>`))
}

// cmsHandler is a placeholder for the CMS endpoints.
func (a *app) cmsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"endpoint":"cms"}`))
}
