package main

import (
	"net/http"
	"strings"
)

// securityHeaders hardens every response with cheap, site-wide headers and
// adds the stronger ones where a session matters: /god and /login are marked
// non-framable (blocks clickjacking overlays on real forms) and no-store (so
// admin HTML never lingers in a browser or proxy cache).
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		if strings.HasPrefix(r.URL.Path, "/god") || r.URL.Path == "/login" {
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("Cache-Control", "no-store")
		}
		next.ServeHTTP(w, r)
	})
}

// noDirListing wraps a file server so paths ending in "/" return 404 instead
// of an index listing. Every static file is a leaf, so nothing legitimately
// requests a directory.
func noDirListing(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/") {
			http.NotFound(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// requirePage wraps a page handler so only signed-in visitors reach it. See
// auth.Service.RequirePage: anonymous callers are redirected to /login.
func (a *app) requirePage(next http.HandlerFunc) http.HandlerFunc {
	return a.auth.RequirePage(next)
}

// requireAPI wraps a JSON handler so only signed-in clients reach it. See
// auth.Service.RequireAPI: anonymous callers get a 401.
func (a *app) requireAPI(next http.HandlerFunc) http.HandlerFunc {
	return a.auth.RequireAPI(next)
}
