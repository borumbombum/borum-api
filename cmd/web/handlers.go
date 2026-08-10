// HTTP handlers. All handlers are methods on *app so they can share
// dependencies (e.g. a database client) added to the app struct. Add new
// endpoints in apiRoutes (routes.go) and their handlers here; split into
// per-domain files as they grow.
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/borumbombum/borum-api/internal/battery"
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
		fmt.Fprintf(w, `{"status": "error", "error": "db-unreachable", "detail": %q, "latancy": %d}`, err.Error(), latency.Milliseconds())
		return
	}

	batteryData, err := battery.Get()
	var batteryJSON []byte
	if err != nil {
		batteryJSON = []byte("null")
	} else {
		batteryJSON, _ = json.Marshal(batteryData)
	}

	fmt.Fprintf(w, `{"status":"ok","db":"ok","latancy":%d, "battery": %s}`, latency.Milliseconds(), batteryJSON)
}

// cmsHandler is a placeholder for the CMS endpoints.
func (a *app) cmsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"endpoint":"cms"}`))
}
