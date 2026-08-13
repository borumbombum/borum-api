package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
)

// CSRF mints a token bound to the given session token. It needs no storage:
// the server signs the session token with a secret only it knows, and the
// client echoes it back in forms. An attacker cannot forge it without reading
// the session cookie (HttpOnly) or the secret.
func (s *Service) CSRF(sessionToken string) string {
	mac := hmac.New(sha256.New, s.csrfSecret)
	mac.Write([]byte(sessionToken))
	return base64URL(mac.Sum(nil))
}

// ValidCSRF reports whether the provided token matches the session token.
func (s *Service) ValidCSRF(sessionToken, provided string) bool {
	mac := hmac.New(sha256.New, s.csrfSecret)
	mac.Write([]byte(sessionToken))
	expected := base64URL(mac.Sum(nil))
	return subtle.ConstantTimeCompare([]byte(expected), []byte(provided)) == 1
}
