// Admin handlers: the /god/* server-rendered pages and their form posts. All
// of them are wrapped in requirePage (auth middleware); anonymous visitors are
// redirected to /login.
package main

import (
	"net/http"
	"strings"

	"github.com/borumbombum/borum-api/internal/auth"
	"github.com/borumbombum/borum-api/internal/content"
	"github.com/go-chi/chi/v5"
)

// godData is the view model for every /god page.
type godData struct {
	Version  string
	LoggedIn bool
	CSRF     string
	Error    string
	IsNew    bool
	Articles []content.ArticleSummary
	Article  *content.Article
	Tags     string // comma-separated, for the form input
}

// newGodData builds the shared admin view model, minting a CSRF token bound to
// the caller's session (requirePage guarantees the session exists).
func (a *app) newGodData(r *http.Request, isNew bool) godData {
	d := godData{
		Version:  a.version,
		LoggedIn: true,
		IsNew:    isNew,
	}
	if sess, ok := auth.SessionFrom(r.Context()); ok {
		d.CSRF = a.auth.CSRF(sess.Token)
	}
	return d
}

// renderGodPage executes an admin template against the god base layout.
func (a *app) renderGodPage(w http.ResponseWriter, status int, name string, data godData) {
	t, ok := godTemplates[name]
	if !ok {
		http.Error(w, "template not found", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := t.ExecuteTemplate(w, "godbase", data); err != nil {
		a.errorLogger.Print(err.Error())
	}
}

// godHandler lands on the article list.
func (a *app) godHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/god/articles", http.StatusFound)
}

// godArticlesHandler renders the admin article list.
func (a *app) godArticlesHandler(w http.ResponseWriter, r *http.Request) {
	data := a.newGodData(r, false)
	data.Articles = content.List(r.Context())
	a.renderGodPage(w, http.StatusOK, "god_list", data)
}

// godArticleNewHandler renders the create form.
func (a *app) godArticleNewHandler(w http.ResponseWriter, r *http.Request) {
	data := a.newGodData(r, true)
	data.Article = &content.Article{}
	a.renderGodPage(w, http.StatusOK, "god_form", data)
}

// godArticleCreateHandler validates and stores a new article, then returns to
// the list. Re-renders the form with an error when validation fails.
func (a *app) godArticleCreateHandler(w http.ResponseWriter, r *http.Request) {
	data := a.newGodData(r, true)
	if !a.validCSRF(r, data.CSRF) {
		data.Error = "invalid request token, try again"
		a.renderGodPage(w, http.StatusForbidden, "god_form", data)
		return
	}
	art := parseArticleForm(r)
	if art.Slug == "" || art.Title == "" || art.Date == "" {
		data.Error = "slug, title and date are required"
		data.Article = &art
		data.Tags = strings.Join(art.Tags, ", ")
		a.renderGodPage(w, http.StatusUnprocessableEntity, "god_form", data)
		return
	}
	if err := content.Save(r.Context(), art); err != nil {
		a.errorLogger.Print(err.Error())
		data.Error = "could not save the article"
		data.Article = &art
		data.Tags = strings.Join(art.Tags, ", ")
		a.renderGodPage(w, http.StatusInternalServerError, "god_form", data)
		return
	}
	http.Redirect(w, r, "/god/articles", http.StatusSeeOther)
}

// godArticleEditHandler renders the edit form, pre-filled from the database.
func (a *app) godArticleEditHandler(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	art := content.GetArticle(r.Context(), slug)
	if art == nil {
		a.notFoundHandler(w, r)
		return
	}
	data := a.newGodData(r, false)
	data.Article = art
	data.Tags = strings.Join(art.Tags, ", ")
	a.renderGodPage(w, http.StatusOK, "god_form", data)
}

// godArticleUpdateHandler validates and stores changes to an existing article.
// The slug comes from the URL and cannot change after creation.
func (a *app) godArticleUpdateHandler(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	data := a.newGodData(r, false)
	if !a.validCSRF(r, data.CSRF) {
		data.Error = "invalid request token, try again"
		data.Article = content.GetArticle(r.Context(), slug)
		a.renderGodPage(w, http.StatusForbidden, "god_form", data)
		return
	}
	art := parseArticleForm(r)
	if art.Title == "" || art.Date == "" {
		data.Error = "title and date are required"
		art.Slug = slug
		data.Article = &art
		data.Tags = strings.Join(art.Tags, ", ")
		a.renderGodPage(w, http.StatusUnprocessableEntity, "god_form", data)
		return
	}
	art.Slug = slug
	if err := content.Save(r.Context(), art); err != nil {
		a.errorLogger.Print(err.Error())
		data.Error = "could not save the article"
		data.Article = &art
		data.Tags = strings.Join(art.Tags, ", ")
		a.renderGodPage(w, http.StatusInternalServerError, "god_form", data)
		return
	}
	http.Redirect(w, r, "/god/articles", http.StatusSeeOther)
}

// godArticleDeleteHandler removes an article by its URL slug and returns to
// the list. Like every admin write, it requires a valid CSRF token.
func (a *app) godArticleDeleteHandler(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	data := a.newGodData(r, false)
	if !a.validCSRF(r, data.CSRF) {
		data.Error = "invalid request token, try again"
		data.Articles = content.List(r.Context())
		a.renderGodPage(w, http.StatusForbidden, "god_list", data)
		return
	}
	if err := content.Delete(r.Context(), slug); err != nil {
		a.errorLogger.Print(err.Error())
		data.Error = "could not delete the article"
		data.Articles = content.List(r.Context())
		a.renderGodPage(w, http.StatusInternalServerError, "god_list", data)
		return
	}
	http.Redirect(w, r, "/god/articles", http.StatusSeeOther)
}

// validCSRF checks the form's _csrf field against the caller's session token.
func (a *app) validCSRF(r *http.Request, _ string) bool {
	token := a.sessionToken(r)
	if token == "" {
		return false
	}
	return a.auth.ValidCSRF(token, r.FormValue("_csrf"))
}

// sessionToken returns the caller's raw session token from the context.
func (a *app) sessionToken(r *http.Request) string {
	if sess, ok := auth.SessionFrom(r.Context()); ok {
		return sess.Token
	}
	return ""
}

// parseArticleForm reads the article fields from a form post. Checkboxes are
// "on" when present. Tags arrive comma-separated and are split and trimmed.
func parseArticleForm(r *http.Request) content.Article {
	r.ParseForm()
	art := content.Article{
		Slug:         strings.TrimSpace(r.FormValue("slug")),
		Title:        strings.TrimSpace(r.FormValue("title")),
		Subtitle:     strings.TrimSpace(r.FormValue("subtitle")),
		Date:         strings.TrimSpace(r.FormValue("date")),
		Excerpt:      strings.TrimSpace(r.FormValue("excerpt")),
		Image:        strings.TrimSpace(r.FormValue("image")),
		ImageCaption: strings.TrimSpace(r.FormValue("image_caption")),
		Star:         r.FormValue("star") == "on",
		Featured:     r.FormValue("featured") == "on",
		Body:         r.FormValue("body"),
	}
	for _, t := range strings.Split(r.FormValue("tags"), ",") {
		if t = strings.TrimSpace(t); t != "" {
			art.Tags = append(art.Tags, t)
		}
	}
	return art
}
