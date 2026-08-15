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
	"github.com/borumbombum/borum-api/internal/content"
)

// healthHandler reports liveness of the API process. The DB ping is cached
// enough for a heartbeat, the battery reading comes from the cached snapshot
// (never a blocking subprocess per request), and raw error strings are not
// echoed to the caller.
func (a *app) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	start := time.Now()
	err := a.db.PingContext(r.Context())
	latency := time.Since(start)

	if err != nil {
		a.errorLogger.Print(err.Error())
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, `{"status": "error", "error": "db-unreachable", "latency": %d}`, latency.Milliseconds())
		return
	}

	snapshot := battery.Current()
	batteryJSON := "null"
	if snapshot.Available {
		if data, err := json.Marshal(snapshot.Status); err == nil {
			batteryJSON = string(data)
		}
	}

	fmt.Fprintf(w, `{"status":"ok","db":"ok","latency":%d, "battery": %s}`, latency.Milliseconds(), batteryJSON)
}

// articlesJSONHandler serves the minimal article data (slug, title, tags) for
// the client-side command palette, from the database. The result is cached
// server-side (see content.Palette) and served from the same /data/articles.json
// URL the palette has always fetched.
func (a *app) articlesJSONHandler(w http.ResponseWriter, r *http.Request) {
	items := content.Palette(r.Context())
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(items); err != nil {
		a.errorLogger.Print(err.Error())
	}
}
