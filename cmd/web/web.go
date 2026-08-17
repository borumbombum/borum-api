// Web views: the site is rendered from Go templates (cmd/web/templates),
// byte-for-byte compatible with the doags static-port reference. Data comes
// from the internal/content package; only static assets (css/js/images) are
// served as files from static/.
package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/borumbombum/borum-api/internal/auth"
	"github.com/borumbombum/borum-api/internal/battery"
	"github.com/borumbombum/borum-api/internal/content"
	"github.com/borumbombum/borum-api/internal/experiments"
	"github.com/go-chi/chi/v5"
)

// templateDir is relative to the working directory (repo root).
const templateDir = "cmd/web/templates"

// readVersion returns the app version from the VERSION file at the repo root,
// falling back to "dev" when it cannot be read.
func readVersion() string {
	b, err := os.ReadFile("VERSION")
	if err != nil {
		return "dev"
	}
	v := strings.TrimSpace(string(b))
	if v == "" {
		return "dev"
	}
	return v
}

// cssFiles is the cascade order of the split stylesheet subsets, matching the
// original single stylesheet: theme (variables/theme) first, utilities last.
var cssFiles = []string{
	"theme.css",
	"base.css",
	"components.css",
	"admin.css",
	"prose.css",
	"editor.css",
	"highlight.css",
	"animations.css",
	"breakpoints.css",
	"utilities.css",
}

// concatCSS reads the split stylesheet subsets in cascade order and joins them
// into one stylesheet, served as /styles.css. It runs at startup so the browser
// makes a single CSS request. Edits to static/css/*.css apply on restart.
func concatCSS() ([]byte, error) {
	var out []byte
	for _, name := range cssFiles {
		b, err := os.ReadFile(filepath.Join("static", "css", name))
		if err != nil {
			return nil, fmt.Errorf("concat css %s: %w", name, err)
		}
		out = append(out, b...)
		if len(out) > 0 && out[len(out)-1] != '\n' {
			out = append(out, '\n')
		}
	}
	return out, nil
}

// pageData is the view model shared by every page that renders the header.
type pageData struct {
	ActiveNav string
	Battery   battery.Snapshot
	Version   string
	LoggedIn  bool
	Lang      string
	Host      string
}

// newPageData builds the shared view model for a page with the given active
// nav entry, carrying the app version so the footer can show it. LoggedIn is
// set from the Peek middleware's session resolution, so public pages can show
// admin affordances (the article edit button) without blocking anyone.
func (a *app) newPageData(r *http.Request, active string) pageData {
	_, loggedIn := auth.SessionFrom(r.Context())
	lang := "en"
	if l, ok := r.Context().Value("lang").(string); ok {
		lang = l
	}
	return pageData{
		ActiveNav: active,
		Battery:   battery.Current(),
		Version:   a.version,
		LoggedIn:  loggedIn,
		Lang:      lang,
		Host:      r.Host,
	}
}

// pageTemplates maps a page key to its parsed template set. Each page is
// parsed together with the shared base/header/footer partials so the page's
// "title", "meta" and "content" blocks override the layout.
var pageTemplates = map[string]*template.Template{}

// godTemplates maps an admin page key to its template set, parsed against the
// god base layout (god_base.html + god_drawer.html) instead of the public one.
var godTemplates = map[string]*template.Template{}

var funcMap = template.FuncMap{
	"safe": func(s string) template.HTML { return template.HTML(s) },
	"shortDate": func(date string) string {
		t, err := time.Parse("2006-01-02", date)
		if err != nil {
			return date
		}
		return t.Format("Jan 2")
	},
	"longDate": func(date string) string {
		t, err := time.Parse("2006-01-02", date)
		if err != nil {
			return date
		}
		return t.Format("January 2, 2006")
	},
	"deviceModel": battery.DeviceModel,
	"dict": func(kv ...any) map[string]any {
		m := make(map[string]any, len(kv)/2)
		for i := 0; i+1 < len(kv); i += 2 {
			m[fmt.Sprintf("%v", kv[i])] = kv[i+1]
		}
		return m
	},
	"batteryIcon": func(pct int, charging bool) string {
		switch {
		case charging:
			return "battery-charging"
		case pct >= 90:
			return "battery-full"
		case pct >= 60:
			return "battery-medium"
		case pct >= 30:
			return "battery-low"
		default:
			return "battery"
		}
	},
}

func loadTemplates() error {
	pages := map[string]string{
		"home":    "home.html",
		"article": "article.html",
		"tag":     "tag.html",
		"login":   "login.html",
		"404":     "404.html",
	}
	common := []string{"base.html", "header.html", "footer.html", "love_bar.html"}
	for name, page := range pages {
		files := append(append([]string{}, common...), page)
		paths := make([]string, 0, len(files))
		for _, f := range files {
			paths = append(paths, filepath.Join(templateDir, f))
		}
		t, err := template.New("").Funcs(funcMap).ParseFiles(paths...)
		if err != nil {
			return fmt.Errorf("template %s: %w", name, err)
		}
		pageTemplates[name] = t
	}

	// Each experiment renders its own template (experiments/<dir>/index.html)
	// against the shared experiment layout (experiments/layout.html) and the
	// public base/header/footer. The layout owns the page chrome — hero, index
	// link, admin intro text — and each experiment only defines its
	// experiment_body block (its form) and optional page_scripts.
	for _, exp := range experiments.All() {
		paths := make([]string, 0, len(common)+2)
		for _, f := range common {
			paths = append(paths, filepath.Join(templateDir, f))
		}
		paths = append(paths, filepath.Join(templateDir, "experiments", "layout.html"))
		paths = append(paths, filepath.Join(templateDir, "experiments", exp.Dir, "index.html"))
		t, err := template.New("").Funcs(funcMap).ParseFiles(paths...)
		if err != nil {
			return fmt.Errorf("experiment template %s: %w", exp.Slug, err)
		}
		pageTemplates["experiment_"+exp.Slug] = t
	}

	// The /god pages share their own layout: god_base.html + the drawer
	// partial, without the public header/footer.
	godPages := map[string]string{
		"god_list":             "god_articles.html",
		"god_form":             "god_article_form.html",
		"god_experiments":      "god_experiments.html",
		"god_carddav":          "god_carddav.html",
		"god_carddav_contacts": "god_carddav_contacts.html",
		"god_dashboard":        "god_dashboard.html",
	}
	godCommon := []string{"god_base.html", "god_drawer.html"}
	for name, page := range godPages {
		files := append(append([]string{}, godCommon...), page)
		paths := make([]string, 0, len(files))
		for _, f := range files {
			paths = append(paths, filepath.Join(templateDir, f))
		}
		t, err := template.New("").Funcs(funcMap).ParseFiles(paths...)
		if err != nil {
			return fmt.Errorf("template %s: %w", name, err)
		}
		godTemplates[name] = t
	}
	return nil
}

func renderPage(w http.ResponseWriter, status int, name string, data any) {
	t, ok := pageTemplates[name]
	if !ok {
		http.Error(w, "template not found", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := t.ExecuteTemplate(w, "base", data); err != nil {
		log.Printf("render %s: %v", name, err)
	}
}

// homeHandler renders the archive home page.
func (a *app) homeHandler(w http.ResponseWriter, r *http.Request) {
	lang := r.Context().Value("lang").(string)
	arts := content.ListByLang(r.Context(), lang)
	all := experiments.List(r.Context())
	var enabled []experiments.Item
	for _, it := range all {
		if it.Enabled {
			enabled = append(enabled, it)
		}
	}
	data := struct {
		pageData
		Featured    *content.ArticleSummary
		Articles    []content.ArticleSummary
		Tags        []string
		Principles  []content.Principle
		Experiments []experiments.Item
		Lang        string
	}{
		pageData:    a.newPageData(r, "home"),
		Articles:    arts,
		Tags:        allTags(arts),
		Principles:  content.Principles(),
		Experiments: enabled,
		Lang:        lang,
	}
	for i := range arts {
		if arts[i].Featured {
			data.Featured = &arts[i]
			break
		}
	}
	renderPage(w, http.StatusOK, "home", data)
}

// articleHandler renders a single blog post.
func (a *app) articleHandler(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	lang := r.Context().Value("lang").(string)
	
	// Get base article (English)
	art := content.GetArticle(r.Context(), slug)
	if art == nil {
		a.notFoundHandler(w, r)
		return
	}
	
	// If non-English, try to load translation and merge
	if lang != "en" {
		if trans := content.GetTranslation(r.Context(), slug, lang); trans != nil {
			// Merge translation into article (non-empty fields override)
			if trans.Title != "" {
				art.Title = trans.Title
			}
			if trans.Subtitle != "" {
				art.Subtitle = trans.Subtitle
			}
			if trans.Excerpt != "" {
				art.Excerpt = trans.Excerpt
			}
			if trans.ImageCaption != "" {
				art.ImageCaption = trans.ImageCaption
			}
			if trans.Body != "" {
				art.Body = trans.Body
			}
		}
	}
	
	translationSlug := content.GetTranslationSlug(r.Context(), slug, lang)
	data := struct {
		pageData
		Article         *content.Article
		Lang            string
		TranslationSlug string
	}{pageData: a.newPageData(r, "articles"), Article: art, Lang: lang, TranslationSlug: translationSlug}
	renderPage(w, http.StatusOK, "article", data)
}

// tagHandler renders the archive filtered by one tag.
func (a *app) tagHandler(w http.ResponseWriter, r *http.Request) {
	tag := chi.URLParam(r, "tag")
	lang := r.Context().Value("lang").(string)
	
	// Get articles for the requested language
	arts := content.ListByLang(r.Context(), lang)
	
	// Filter by tag
	filtered := make([]content.ArticleSummary, 0, len(arts))
	for _, art := range arts {
		for _, t := range art.Tags {
			if t == tag {
				filtered = append(filtered, art)
				break
			}
		}
	}
	if len(filtered) == 0 {
		a.notFoundHandler(w, r)
		return
	}
	data := struct {
		pageData
		Tag      string
		Articles []content.ArticleSummary
		Lang     string
	}{pageData: a.newPageData(r, "articles"), Tag: tag, Articles: filtered, Lang: lang}
	renderPage(w, http.StatusOK, "tag", data)
}

// notFoundHandler is also used as the router's NotFound handler.
func (a *app) notFoundHandler(w http.ResponseWriter, r *http.Request) {
	renderPage(w, http.StatusNotFound, "404", a.newPageData(r, ""))
}

// allTags returns the unique tags across the given articles in the order they
// first appear, matching the static-port reference tag cloud.
func allTags(arts []content.ArticleSummary) []string {
	seen := map[string]bool{}
	var tags []string
	for _, art := range arts {
		for _, t := range art.Tags {
			if !seen[t] {
				seen[t] = true
				tags = append(tags, t)
			}
		}
	}
	return tags
}
