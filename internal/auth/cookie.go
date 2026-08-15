package auth

import (
	"net"
	"net/http"
	"time"
)

// isLoopback reports whether addr is a localhost address. Used to decide when
// proxy-supplied headers can be trusted: the Cloudflare tunnel runs on the
// same host, so a loopback peer is a trusted proxy.
func isLoopback(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

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
	// Secure only when the connection is genuinely HTTPS. The
	// X-Forwarded-Proto header is trusted only when the peer is loopback:
	// the Cloudflare tunnel runs on this same host, so a local peer is a
	// trusted proxy. A directly exposed server cannot be tricked into
	// marking cookies Secure by spoofing the header.
	secure := r.TLS != nil || (isLoopback(r.RemoteAddr) && r.Header.Get("X-Forwarded-Proto") == "https")
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
