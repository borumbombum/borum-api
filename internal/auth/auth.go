// Package auth owns login, sessions and CSRF protection for the single-user
// admin area. Auth methods are pluggable: email/password ships today, a Nostr
// provider slots in by registering a new Method. Sessions are stored in the
// database keyed by a SHA-256 hash of a random token held in an HttpOnly cookie.
package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/borumbombum/borum-api/internal/ratelimit"
)

// ErrInvalidCredentials is returned for any failed login attempt. It is
// deliberately generic so callers cannot tell a bad email from a bad password.
var ErrInvalidCredentials = errors.New("invalid credentials")

// Identity is a successful authentication result. For now every method maps
// to the single "god" user; a future multi-user system would add a real
// account id here.
type Identity struct {
	UserID string
	Method string
}

// Credentials carries whatever a login method needs to verify a user.
type Credentials struct {
	Email    string
	Password string
}

// Session is an authenticated session: the raw cookie token plus the identity.
type Session struct {
	Token    string
	Identity Identity
}

// Config holds the fixed auth settings read from the environment at startup.
type Config struct {
	AdminEmail        string // the only email that may log in
	AdminPasswordHash string // bcrypt hash of the admin password
	SessionTTL        time.Duration
	CookieName        string
}

// Service validates credentials, issues and revokes sessions, and guards
// admin routes and state-changing requests.
type Service struct {
	db         *sql.DB
	cfg        Config
	limiter    *ratelimit.Limiter
	csrfSecret []byte
}

// New builds a Service. The CSRF signing secret is random per process, so
// tokens minted by a previous run are invalid after a restart.
func New(db *sql.DB, cfg Config) *Service {
	return &Service{
		db:         db,
		cfg:        cfg,
		limiter:    ratelimit.New(loginMaxAttempts, loginWindow),
		csrfSecret: randomBytes(32),
	}
}

// randomBytes returns n cryptographically random bytes, panicking only if the
// host entropy source is broken (an unrecoverable condition).
func randomBytes(n int) []byte {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic("auth: cannot read entropy: " + err.Error())
	}
	return b
}

// generateToken returns a URL-safe random session token.
func generateToken() string {
	return base64URL(randomBytes(32))
}

// Login verifies the given credentials through the named method and, on
// success, opens a new session.
func (s *Service) Login(ctx context.Context, method string, creds Credentials) (Session, error) {
	m, ok := methods[method]
	if !ok {
		return Session{}, ErrInvalidCredentials
	}
	id, err := m.Authenticate(ctx, creds)
	if err != nil {
		return Session{}, err
	}
	return s.CreateSession(ctx, id)
}

// CreateSession stores a new session row and returns it with a fresh token.
func (s *Service) CreateSession(ctx context.Context, id Identity) (Session, error) {
	token := generateToken()
	expires := time.Now().Add(s.cfg.SessionTTL).UTC().Format(time.RFC3339)

	s.purgeExpired(ctx)

	_, err := s.db.ExecContext(ctx, `INSERT INTO sessions
		(token_hash, user_id, method, created_at, expires_at, last_seen)
		VALUES (?, ?, ?, datetime('now'), ?, datetime('now'))`,
		hashToken(token), id.UserID, id.Method, expires)
	if err != nil {
		return Session{}, err
	}
	return Session{Token: token, Identity: id}, nil
}

// purgeExpired best-effort removes expired sessions on write paths so the
// table never grows unbounded.
func (s *Service) purgeExpired(ctx context.Context) {
	s.db.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at < ?`, time.Now().UTC().Format(time.RFC3339))
}

// Session resolves the request's cookie to a live session, or an error when
// the cookie is missing, unknown, or expired. Expired rows are deleted.
func (s *Service) Session(r *http.Request) (Session, error) {
	cookie, err := r.Cookie(s.cfg.CookieName)
	if err != nil {
		return Session{}, err
	}
	token := cookie.Value
	now := time.Now().UTC().Format(time.RFC3339)

	var id Identity
	var expires string
	err = s.db.QueryRowContext(r.Context(), `SELECT user_id, method, expires_at
		FROM sessions WHERE token_hash = ?`, hashToken(token)).Scan(&id.UserID, &id.Method, &expires)
	if err != nil {
		return Session{}, ErrInvalidCredentials
	}
	if expires < now {
		s.db.ExecContext(r.Context(), `DELETE FROM sessions WHERE token_hash = ?`, hashToken(token))
		return Session{}, ErrInvalidCredentials
	}
	return Session{Token: token, Identity: id}, nil
}

// Logout revokes the session behind the request's cookie, if any, and clears
// the cookie so the browser does not keep sending a dead token.
func (s *Service) Logout(w http.ResponseWriter, r *http.Request) error {
	cookie, err := r.Cookie(s.cfg.CookieName)
	if err == nil {
		if _, err := s.db.ExecContext(r.Context(), `DELETE FROM sessions WHERE token_hash = ?`, hashToken(cookie.Value)); err != nil {
			return err
		}
	}
	s.ClearSessionCookie(w, r)
	return nil
}

// Login rate limits: at most loginMaxAttempts failed-or-any login attempts per
// loginWindow, keyed per client (see clientIP) plus the submitted email.
const (
	loginMaxAttempts = 5
	loginWindow      = time.Minute
)

// AllowLogin reports whether another login attempt from this client fits the
// rate limit. Callers should check it before verifying credentials.
func (s *Service) AllowLogin(key string) bool {
	return s.limiter.Allow(key)
}
