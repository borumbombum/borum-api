// HTTP handlers. All handlers are methods on *app so they can share
// dependencies (e.g. a database client) added to the app struct. Add new
// endpoints in apiRoutes (routes.go) and their handlers here; split into
// per-domain files as they grow.
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/borumbombum/borum-api/internal/battery"
	"github.com/borumbombum/borum-api/internal/content"
	"github.com/go-chi/chi/v5"
)

// healthHandler reports liveness of the API process. The DB ping is cached
// enough for a heartbeat, the battery reading comes from the cached snapshot
// (never a blocking subprocess per request), and raw error strings are not
// echoed to the caller.
func (a *app) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	start := time.Now()
	err := a.db.PingContext(r.Context())
	latency := time.Since(start)

	if err != nil {
		a.errorLogger.Print(err.Error())
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, `{"status": "error", "error": "db-unreachable", "latency": %d}`, latency.Milliseconds())
		return
	}

	snapshot := battery.Current()
	batteryJSON := "null"
	if snapshot.Available {
		if data, err := json.Marshal(snapshot.Status); err == nil {
			batteryJSON = string(data)
		}
	}

	fmt.Fprintf(w, `{"status":"ok","db":"ok","latency":%d, "battery": %s}`, latency.Milliseconds(), batteryJSON)
}

// articlesJSONHandler serves the minimal article data (slug, title, tags) for
// the client-side command palette, from the database. The result is cached
// server-side (see content.Palette) and served from the same /data/articles.json
// URL the palette has always fetched.
func (a *app) articlesJSONHandler(w http.ResponseWriter, r *http.Request) {
	items := content.Palette(r.Context())
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(items); err != nil {
		a.errorLogger.Print(err.Error())
	}
}

// articleAutoSaveRequest is the JSON body for the draft auto-save endpoint.
type articleAutoSaveRequest struct {
	Slug         string   `json:"slug"`
	Title        string   `json:"title"`
	Subtitle     string   `json:"subtitle"`
	Date         string   `json:"date"`
	Tags         []string `json:"tags"`
	Excerpt      string   `json:"excerpt"`
	Image        string   `json:"image"`
	ImageCaption string   `json:"imageCaption"`
	Star         bool     `json:"star"`
	Featured     bool     `json:"featured"`
	Body         string   `json:"body"`
}

var slugRe = regexp.MustCompile(`[^a-z0-9-]`)
var dashRe = regexp.MustCompile(`-{2,}`)

// makeSlug converts a title to a URL-safe slug.
func makeSlug(title string) string {
	s := strings.ToLower(title)
	s = strings.ReplaceAll(s, " ", "-")
	s = slugRe.ReplaceAllString(s, "")
	s = dashRe.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "untitled"
	}
	return s
}

// articleCreateDraftHandler creates a new draft article from a JSON body.
// Returns the assigned slug so the client can update its URL.
func (a *app) articleCreateDraftHandler(w http.ResponseWriter, r *http.Request) {
	var req articleAutoSaveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return
	}
	if req.Title == "" {
		http.Error(w, `{"error":"title required"}`, http.StatusBadRequest)
		return
	}
	slug := req.Slug
	if slug == "" {
		slug = makeSlug(req.Title)
	}
	// Ensure slug uniqueness by appending a suffix if it already exists.
	existing := content.GetArticleAny(r.Context(), slug)
	if existing != nil {
		for i := 2; ; i++ {
			candidate := fmt.Sprintf("%s-%d", slug, i)
			if content.GetArticleAny(r.Context(), candidate) == nil {
				slug = candidate
				break
			}
		}
	}
	art := content.Article{
		Slug:         slug,
		Title:        req.Title,
		Subtitle:     req.Subtitle,
		Date:         req.Date,
		Tags:         req.Tags,
		Excerpt:      req.Excerpt,
		Image:        req.Image,
		ImageCaption: req.ImageCaption,
		Star:         req.Star,
		Featured:     req.Featured,
		Body:         req.Body,
		Status:       "draft",
	}
	if art.Date == "" {
		art.Date = time.Now().Format("2006-01-02")
	}
	if err := content.Save(r.Context(), art); err != nil {
		a.errorLogger.Print(err.Error())
		http.Error(w, `{"error":"save failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"ok":true,"slug":%q}`, slug)
}

// articleUpdateDraftHandler updates an existing draft from a JSON body.
// Slug changes are allowed for drafts.
func (a *app) articleUpdateDraftHandler(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	var req articleAutoSaveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return
	}
	existing := content.GetArticleAny(r.Context(), slug)
	if existing == nil {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}
	// Allow slug changes for drafts.
	newSlug := req.Slug
	if newSlug == "" {
		newSlug = slug
	}
	if existing.Status == "draft" && newSlug != slug {
		if err := content.ChangeSlug(r.Context(), slug, newSlug); err != nil {
			a.errorLogger.Print(err.Error())
			http.Error(w, `{"error":"slug change failed"}`, http.StatusInternalServerError)
			return
		}
		slug = newSlug
	}
	art := content.Article{
		Slug:         slug,
		Title:        req.Title,
		Subtitle:     req.Subtitle,
		Date:         req.Date,
		Tags:         req.Tags,
		Excerpt:      req.Excerpt,
		Image:        req.Image,
		ImageCaption: req.ImageCaption,
		Star:         req.Star,
		Featured:     req.Featured,
		Body:         req.Body,
		Status:       "draft",
	}
	if art.Date == "" {
		art.Date = time.Now().Format("2006-01-02")
	}
	art.InitialLove = existing.InitialLove
	if err := content.Save(r.Context(), art); err != nil {
		a.errorLogger.Print(err.Error())
		http.Error(w, `{"error":"save failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"ok":true,"slug":%q}`, slug)
}
