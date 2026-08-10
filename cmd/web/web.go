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
	"sort"
	"time"

	"github.com/borumbombum/borum-api/internal/battery"
	"github.com/borumbombum/borum-api/internal/content"
	"github.com/go-chi/chi/v5"
)

// templateDir is relative to the working directory (repo root).
const templateDir = "cmd/web/templates"

// cssFiles is the cascade order of the split stylesheet subsets, matching the
// original single stylesheet: theme (variables/theme) first, utilities last.
var cssFiles = []string{
	"theme.css",
	"base.css",
	"components.css",
	"prose.css",
	"animations.css",
	"breakdowns.css",
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
}

// pageTemplates maps a page key to its parsed template set. Each page is
// parsed together with the shared base/header/footer partials so the page's
// "title", "meta" and "content" blocks override the layout.
var pageTemplates = map[string]*template.Template{}

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
		"404":     "404.html",
	}
	common := []string{"base.html", "header.html", "footer.html"}
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
	arts := append([]content.Article(nil), content.Articles()...)
	sort.SliceStable(arts, func(i, j int) bool { return arts[i].Date > arts[j].Date })

	data := struct {
		pageData
		Featured   *content.Article
		Articles   []content.Article
		Tags       []string
		Principles []content.Principle
	}{
		pageData:   pageData{ActiveNav: "home", Battery: battery.Current()},
		Articles:   arts,
		Tags:       allTags(content.Articles()),
		Principles: content.Principles(),
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
	art := content.FindArticle(slug)
	if art == nil {
		a.notFoundHandler(w, r)
		return
	}
	data := struct {
		pageData
		Article *content.Article
	}{pageData: pageData{ActiveNav: "articles", Battery: battery.Current()}, Article: art}
	renderPage(w, http.StatusOK, "article", data)
}

// tagHandler renders the archive filtered by one tag.
func (a *app) tagHandler(w http.ResponseWriter, r *http.Request) {
	tag := chi.URLParam(r, "tag")
	arts := content.Articles()
	filtered := make([]content.Article, 0, len(arts))
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
	sort.SliceStable(filtered, func(i, j int) bool { return filtered[i].Date > filtered[j].Date })
	data := struct {
		ActiveNav string
		Tag       string
		Articles  []content.Article
	}{ActiveNav: "articles", Tag: tag, Articles: filtered}
	renderPage(w, http.StatusOK, "tag", data)
}

// notFoundHandler is also used as the router's NotFound handler.
func (a *app) notFoundHandler(w http.ResponseWriter, r *http.Request) {
	renderPage(w, http.StatusNotFound, "404", nil)
}

// allTags returns the unique tags across the given articles in the order they
// first appear, matching the static-port reference tag cloud.
func allTags(arts []content.Article) []string {
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
