package auth

import (
	"net/http"
	"time"
)

// SetSessionCookie writes the session cookie for the request with the
// configured lifetime. The Secure flag is set only when the connection is
// HTTPS: detected from r.TLS or the X-Forwarded-Proto header sent by
// Cloudflare Tunnel. Local plain-HTTP testing keeps working while production
// sessions stay Secure.
func (s *Service) SetSessionCookie(w http.ResponseWriter, r *http.Request, sess Session) {
	setCookie(w, s.sessionCookie(r, sess.Token, int(s.cfg.SessionTTL.Seconds())))
}

// ClearSessionCookie deletes the session cookie in the browser.
func (s *Service) ClearSessionCookie(w http.ResponseWriter, r *http.Request) {
	c := s.sessionCookie(r, "", -1)
	c.Expires = time.Unix(1, 0)
	setCookie(w, c)
}

// sessionCookie builds the session cookie for the request.
func (s *Service) sessionCookie(r *http.Request, value string, maxAge int) *http.Cookie {
	secure := r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
	return &http.Cookie{
		Name:     s.cfg.CookieName,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
	}
}

// setCookie writes a cookie to the response.
func setCookie(w http.ResponseWriter, c *http.Cookie) {
	http.SetCookie(w, c)
}
