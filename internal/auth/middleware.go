package auth

import (
	"context"
	"encoding/json"
	"net/http"
)

// ctxKey is the request-context key for the resolved session.
type ctxKey int

const sessionKey ctxKey = 1

// WithSession stores the session in the context.
func WithSession(ctx context.Context, s Session) context.Context {
	return context.WithValue(ctx, sessionKey, s)
}

// SessionFrom returns the session stored by WithSession, if any.
func SessionFrom(ctx context.Context) (Session, bool) {
	s, ok := ctx.Value(sessionKey).(Session)
	return s, ok
}

// Peek is router-level middleware: it resolves the session cookie when present
// and stores the result in the context, but never blocks the request. Public
// pages use it to learn whether a visitor is signed in.
func (s *Service) Peek(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if sess, err := s.Session(r); err == nil {
			r = r.WithContext(WithSession(r.Context(), sess))
		}
		next.ServeHTTP(w, r)
	})
}

// RequirePage wraps a page handler and redirects anonymous visitors to /login.
func (s *Service) RequirePage(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sess, err := s.Session(r)
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		next(w, r.WithContext(WithSession(r.Context(), sess)))
	}
}

// RequireAPI wraps a JSON handler and answers anonymous callers with 401.
func (s *Service) RequireAPI(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sess, err := s.Session(r)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		next(w, r.WithContext(WithSession(r.Context(), sess)))
	}
}

// writeJSON is the package's shared JSON writer.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.Encode(v)
}
