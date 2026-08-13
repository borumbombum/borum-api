package main

import "net/http"

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
