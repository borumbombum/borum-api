package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"
)

// Write an error message with traces
func (a *app) errorTrace(w http.ResponseWriter, err error) {
	trace := fmt.Sprintf("%s\n%s", err.Error(), debug.Stack())
	a.errorLogger.Output(2, trace)
	// a.errorLogger.Printf("[ERROR_TRACE] %s", trace)
}

// writeJSON writes v as JSON with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v != nil {
		json.NewEncoder(w).Encode(v)
	}
}
