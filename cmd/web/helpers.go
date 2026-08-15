package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"runtime/debug"
	"strings"
)

// clientIP resolves the visitor's real address: the first entry of
// X-Forwarded-For when present (set by the Cloudflare tunnel), else the
// direct peer address. Rate limits key on this instead of RemoteAddr, which
// is 127.0.0.1:<random-port> for every connection behind the tunnel.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.IndexByte(xff, ','); i >= 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	return r.RemoteAddr
}

// isLoopback reports whether addr is a localhost address. Used to decide when
// headers like X-Forwarded-Proto can be trusted: the Cloudflare tunnel runs on
// the same host, so a loopback peer is a trusted proxy.
func isLoopback(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

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
