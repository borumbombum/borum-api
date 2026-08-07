// HTTP handlers. All handlers are methods on *app so they share the injected
// PocketBase instance. Add new endpoints in apiRoutes (routes.go) and their
// handlers here; split into per-domain files (e.g. cms.go) as they grow.
package main

import "net/http"

// healthHandler uses the injected PocketBase instance.
func (a *app) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, err := a.pb.CountRecords("_superusers")
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"status":"degraded","service":"borum-api","database":"unreachable"}`))
		return
	}
	w.Write([]byte(`{"status":"ok","service":"borum-api","database":"ok"}`))
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
