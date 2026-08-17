// Router setup: apiRoutes is the single source of truth for the custom API.
// router() builds the chi router from it and printRoutes (server.go) feeds the
// startup banner from the same slice.
package main

import (
	"context"
	"net/http"

	"github.com/borumbombum/borum-api/internal/i18n"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// apiRoute declares one endpoint: the HTTP method, the path and the handler to call.
type apiRoute struct {
	method  string
	path    string
	handler http.HandlerFunc
}

// apiRoutes is the single source of truth for the routes.
// The web site lives at the root (/); the API under /api with the health
// endpoint unversioned and versioned routes under /api/v1.
// Both route registration and the startup banner are generated from it.
func (a *app) apiRoutes() []apiRoute {
	routes := []apiRoute{
		// API routes (no language prefix)
		{http.MethodGet, "/api/health", a.healthHandler},
		{http.MethodPost, "/api/v1/auth/login", a.loginHandler},
		{http.MethodPost, "/api/v1/auth/logout", a.requireAPI(a.logoutHandler)},
		{http.MethodGet, "/api/v1/auth/me", a.meHandler},
		{http.MethodPost, "/api/v1/articles/draft", a.requireAPI(a.articleCreateDraftHandler)},
		{http.MethodPut, "/api/v1/articles/{slug}/draft", a.requireAPI(a.articleUpdateDraftHandler)},

		// Admin routes (no language prefix)
		{http.MethodGet, "/login", a.loginPageHandler},
		{http.MethodGet, "/god", a.requirePage(a.godHandler)},
		{http.MethodGet, "/god/articles", a.requirePage(a.godArticlesHandler)},
		{http.MethodGet, "/god/experiments", a.requirePage(a.godExperimentsHandler)},
		{http.MethodPost, "/god/experiments/{slug}/toggle", a.requirePage(a.godExperimentToggleHandler)},
		{http.MethodPost, "/god/experiments/{slug}/move", a.requirePage(a.godExperimentMoveHandler)},
		{http.MethodPost, "/god/experiments/{slug}/intro", a.requirePage(a.godExperimentIntroHandler)},
		{http.MethodGet, "/god/articles/new", a.requirePage(a.godArticleNewHandler)},
		{http.MethodPost, "/god/articles", a.requirePage(a.godArticleCreateHandler)},
		{http.MethodPost, "/god/articles/draft", a.requirePage(a.godArticleDraftHandler)},
		{http.MethodGet, "/god/articles/{slug}/edit", a.requirePage(a.godArticleEditHandler)},
		{http.MethodPost, "/god/articles/{slug}/edit", a.requirePage(a.godArticleUpdateHandler)},
		{http.MethodPost, "/god/articles/{slug}/delete", a.requirePage(a.godArticleDeleteHandler)},
		{http.MethodPost, "/god/articles/{slug}/preview", a.requirePage(a.godPreviewTokenHandler)},
		{http.MethodGet, "/god/articles/preview/{token}", a.requirePage(a.godPreviewHandler)},

		// Data endpoints (no language prefix)
		{http.MethodGet, "/data/articles.json", a.articlesJSONHandler},

		// Experiment API routes (no language prefix)
		{http.MethodPost, "/experiments/img2webp/convert", a.img2webpConvertHandler},
	}

	// Add language-specific routes for each supported language
	for _, lang := range i18n.Supported {
		prefix := i18n.URLFor(lang, "")

		if lang == i18n.DefaultLang {
			// English routes (no prefix)
			routes = append(routes, []apiRoute{
				{http.MethodGet, "/", a.homeHandler},
				{http.MethodGet, "/blog/{slug}", a.articleHandler},
				{http.MethodGet, "/tags/{tag}", a.tagHandler},
				{http.MethodGet, "/experiments/{slug}", a.experimentHandler},
			}...)
		} else {
			// Other language routes (with prefix)
			routes = append(routes, []apiRoute{
				{http.MethodGet, prefix, a.homeHandler},
				{http.MethodGet, prefix + "/blog/{slug}", a.articleHandler},
				{http.MethodGet, prefix + "/tags/{tag}", a.tagHandler},
				{http.MethodGet, prefix + "/experiments/{slug}", a.experimentHandler},
			}...)
		}
	}

	return routes
}

// router builds the chi router from the provided route table, mounts the
// static assets from static/ and serves the site's 404 for unknown paths.
func router(a *app, routes []apiRoute) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(securityHeaders)
	r.Use(a.auth.Peek)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"http://localhost:5173"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	}))

	// Images never change once published; cache them client-side for 7 days so
	// repeat visits skip the round-trip over the tunnel. Other assets (JS/CSS)
	// always revalidate so deploys and edits serve fresh styles.
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case len(r.URL.Path) > len("/assets/experiments/") && r.URL.Path[:len("/assets/experiments/")] == "/assets/experiments/":
				// Experiment scripts change with the code; never cache them.
				w.Header().Set("Cache-Control", "no-cache")
			case len(r.URL.Path) > len("/assets/") && r.URL.Path[:len("/assets/")] == "/assets/":
				w.Header().Set("Cache-Control", "public, max-age=604800, stale-while-revalidate=86400")
			case r.URL.Path == "/app.js" || r.URL.Path == "/god.js" || r.URL.Path == "/borum-loader.js" || r.URL.Path == "/god-editor.js" || r.URL.Path == "/god-autosave.js" || r.URL.Path == "/styles.css":
				w.Header().Set("Cache-Control", "no-cache")
			default:
				if len(r.URL.Path) > len("/vendor/") && r.URL.Path[:len("/vendor/")] == "/vendor/" {
					w.Header().Set("Cache-Control", "no-cache")
				}
			}
			next.ServeHTTP(w, r)
		})
	})

	r.Use(middleware.Compress(5))

	// Language detection middleware
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			lang := i18n.LangFromPath(r.URL.Path)
			ctx := context.WithValue(r.Context(), "lang", lang)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})

	// Register all routes
	for _, rt := range routes {
		r.Method(rt.method, rt.path, rt.handler)
	}

	// Static assets served from disk (no embed).
	fs := noDirListing(http.FileServer(http.Dir("static")))
	r.Handle("/assets/*", http.StripPrefix("/", fs))
	r.Handle("/vendor/*", http.StripPrefix("/", fs))
	r.Handle("/app.js", fs)
	r.Handle("/god.js", fs)
	r.Handle("/borum-loader.js", fs)
	r.Handle("/god-editor.js", fs)
	r.Handle("/god-autosave.js", fs)
	r.Handle("/favicon.svg", fs)

	// /styles.css is the concat of static/css/*.css, computed at startup
	// (concatCSS in web.go) so the browser gets a single stylesheet.
	r.Handle("/styles.css", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		w.Write(a.css)
	}))

	r.NotFound(a.notFoundHandler)

	return r
}
