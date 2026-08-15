// Auth handlers: the login page and the login/logout/me endpoints that drive
// the session cookie. The admin pages themselves live in admin_handlers.go.
package main

import (
	"encoding/json"
	"net/http"

	"github.com/borumbombum/borum-api/internal/auth"
)

// loginRequest is the JSON body accepted by POST /api/v1/auth/login.
type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// loginPageHandler renders the sign-in page.
func (a *app) loginPageHandler(w http.ResponseWriter, r *http.Request) {
	renderPage(w, http.StatusOK, "login", a.newPageData(r, ""))
}

// loginHandler checks the submitted email/password, rate-limited per client,
// and on success issues the session cookie. Failures are generic so the page
// cannot be probed for valid emails; the real reason is written to the server
// log (never the password) for the owner to debug.
func (a *app) loginHandler(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	if !a.auth.AllowLogin(clientIP(r) + "|" + req.Email) {
		a.errorLogger.Printf("login rate-limited for %q from %s", req.Email, clientIP(r))
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "too many attempts, slow down"})
		return
	}

	sess, err := a.auth.Login(r.Context(), "password", auth.Credentials{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		a.errorLogger.Printf("login failed for %q from %s: %v", req.Email, r.RemoteAddr, err)
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		return
	}

	a.auth.SetSessionCookie(w, r, sess)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// logoutHandler revokes the current session and clears the cookie.
func (a *app) logoutHandler(w http.ResponseWriter, r *http.Request) {
	if err := a.auth.Logout(w, r); err != nil {
		a.errorLogger.Print(err.Error())
	}
	writeJSON(w, http.StatusNoContent, nil)
}

// meHandler reports whether the visitor holds a live session; the article
// pages use it to show the edit button. The identity is already in the request
// context via the Peek middleware.
func (a *app) meHandler(w http.ResponseWriter, r *http.Request) {
	_, ok := auth.SessionFrom(r.Context())
	writeJSON(w, http.StatusOK, map[string]bool{"authenticated": ok})
}
