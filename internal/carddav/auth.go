package carddav

import (
	"net/http"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// BasicAuth middleware authenticates DAV clients using HTTP Basic Auth.
// It uses a separate app-scoped credential (not the admin password) for security.
func BasicAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the expected token hash from environment
		tokenHash := os.Getenv("CARDDAV_TOKEN_HASH")
		if tokenHash == "" {
			http.Error(w, "CardDAV not configured", http.StatusServiceUnavailable)
			return
		}

		// Parse Basic Auth header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Basic ") {
			w.Header().Set("WWW-Authenticate", `Basic realm="CardDAV"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Decode the credentials
		decoded := strings.TrimPrefix(authHeader, "Basic ")
		// Basic Auth is base64 encoded "username:password"
		// For CardDAV, we use a single token as the password
		// The username is ignored for single-user setup
		parts := strings.SplitN(decoded, ":", 2)
		if len(parts) != 2 {
			w.Header().Set("WWW-Authenticate", `Basic realm="CardDAV"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Use the password as the token (username is ignored for single-user)
		token := parts[1]

		// Compare with stored hash
		err := bcrypt.CompareHashAndPassword([]byte(tokenHash), []byte(token))
		if err != nil {
			w.Header().Set("WWW-Authenticate", `Basic realm="CardDAV"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Authentication successful - set user ID in context
		ctx := WithUserID(r.Context(), "default")
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireHTTPS middleware rejects non-TLS connections unless X-Forwarded-Proto indicates HTTPS.
// This is critical for Basic Auth security.
func RequireHTTPS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if we're behind a reverse proxy
		if r.Header.Get("X-Forwarded-Proto") == "https" {
			// Behind proxy, trust the header
			next.ServeHTTP(w, r)
			return
		}

		// Direct connection - check TLS
		if r.TLS == nil {
			http.Error(w, "HTTPS required for Basic Auth", http.StatusUpgradeRequired)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// SecurityHeaders adds HSTS and other security headers for DAV endpoints.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}
