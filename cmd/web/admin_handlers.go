// Admin handlers: the /god/* server-rendered pages and their form posts. All
// of them are wrapped in requirePage (auth middleware); anonymous visitors are
// redirected to /login.
package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/borumbombum/borum-api/internal/auth"
	"github.com/borumbombum/borum-api/internal/content"
	"github.com/borumbombum/borum-api/internal/experiments"
	"github.com/go-chi/chi/v5"
)

// godData is the view model for every /god page.
type godData struct {
	Version     string
	LoggedIn    bool
	CSRF        string
	Error       string
	IsNew       bool
	Articles    []content.ArticleSummary
	Article     *content.Article
	Tags        string // comma-separated, for the form input
	Experiments []experiments.Item
	Lang        string
	Translation *content.ArticleTranslation
}

// newGodData builds the shared admin view model, minting a CSRF token bound to
// the caller's session (requirePage guarantees the session exists).
func (a *app) newGodData(r *http.Request, isNew bool) godData {
	d := godData{
		Version:  a.version,
		LoggedIn: true,
		IsNew:    isNew,
		Lang:     "en",
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

// godExperimentsHandler renders the admin experiment list: every hardcoded
// experiment with its visibility toggle and up/down move controls.
func (a *app) godExperimentsHandler(w http.ResponseWriter, r *http.Request) {
	data := a.newGodData(r, false)
	data.Experiments = experiments.List(r.Context())
	a.renderGodPage(w, http.StatusOK, "god_experiments", data)
}

// godExperimentToggleHandler shows or hides an experiment, then returns to
// the list. The target state arrives as a hidden form field.
func (a *app) godExperimentToggleHandler(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	data := a.newGodData(r, false)
	if !a.validCSRF(r) {
		data.Error = "invalid request token, try again"
		data.Experiments = experiments.List(r.Context())
		a.renderGodPage(w, http.StatusForbidden, "god_experiments", data)
		return
	}
	if err := experiments.SetEnabled(r.Context(), slug, r.FormValue("enabled") == "1"); err != nil {
		a.errorLogger.Print(err.Error())
		data.Error = "could not update the experiment"
		data.Experiments = experiments.List(r.Context())
		a.renderGodPage(w, http.StatusInternalServerError, "god_experiments", data)
		return
	}
	http.Redirect(w, r, "/god/experiments", http.StatusSeeOther)
}

// godExperimentIntroHandler saves the admin-written intro text that renders
// above the experiment's form, then returns to the list.
func (a *app) godExperimentIntroHandler(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	data := a.newGodData(r, false)
	if !a.validCSRF(r) {
		data.Error = "invalid request token, try again"
		data.Experiments = experiments.List(r.Context())
		a.renderGodPage(w, http.StatusForbidden, "god_experiments", data)
		return
	}
	lang := r.FormValue("lang")
	if lang == "" {
		lang = "en"
	}
	intro := r.FormValue("intro")
	if err := experiments.UpdateIntro(r.Context(), slug, lang, intro); err != nil {
		a.errorLogger.Print(err.Error())
		data.Error = "could not save the experiment intro"
		data.Experiments = experiments.List(r.Context())
		a.renderGodPage(w, http.StatusInternalServerError, "god_experiments", data)
		return
	}
	http.Redirect(w, r, "/god/experiments", http.StatusSeeOther)
}

// godExperimentMoveHandler shifts an experiment one position up or down in
// the home page list. Direction arrives as dir=up or dir=down.
func (a *app) godExperimentMoveHandler(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	data := a.newGodData(r, false)
	if !a.validCSRF(r) {
		data.Error = "invalid request token, try again"
		data.Experiments = experiments.List(r.Context())
		a.renderGodPage(w, http.StatusForbidden, "god_experiments", data)
		return
	}
	dir := -1
	if r.FormValue("dir") == "down" {
		dir = 1
	}
	if err := experiments.Move(r.Context(), slug, dir); err != nil {
		a.errorLogger.Print(err.Error())
		data.Error = "could not move the experiment"
		data.Experiments = experiments.List(r.Context())
		a.renderGodPage(w, http.StatusInternalServerError, "god_experiments", data)
		return
	}
	http.Redirect(w, r, "/god/experiments", http.StatusSeeOther)
}

// godArticlesHandler renders the admin article list.
func (a *app) godArticlesHandler(w http.ResponseWriter, r *http.Request) {
	data := a.newGodData(r, false)
	data.Articles = content.ListAll(r.Context())
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
	if !a.validCSRF(r) {
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
	// Save translation if provided
	parseAndSaveTranslation(r, art.Slug, "pt")
	http.Redirect(w, r, "/god/articles", http.StatusSeeOther)
}

// godArticleDraftHandler saves an article as a draft. Unlike the create/update
// handlers it does not require a date and sets status to "draft".
func (a *app) godArticleDraftHandler(w http.ResponseWriter, r *http.Request) {
	data := a.newGodData(r, false)
	if !a.validCSRF(r) {
		data.Error = "invalid request token, try again"
		a.renderGodPage(w, http.StatusForbidden, "god_form", data)
		return
	}
	art := parseArticleForm(r)
	if art.Title == "" {
		data.Error = "title is required"
		data.Article = &art
		data.Tags = strings.Join(art.Tags, ", ")
		a.renderGodPage(w, http.StatusUnprocessableEntity, "god_form", data)
		return
	}
	// For new drafts, generate a slug from the title if empty.
	if art.Slug == "" {
		art.Slug = makeSlug(art.Title)
		// Ensure slug uniqueness.
		for i := 2; content.GetArticleAny(r.Context(), art.Slug) != nil; i++ {
			art.Slug = makeSlug(art.Title) + "-" + fmt.Sprintf("%d", i)
		}
	}
	if art.Date == "" {
		art.Date = time.Now().Format("2006-01-02")
	}
	art.Status = "draft"
	if err := content.Save(r.Context(), art); err != nil {
		a.errorLogger.Print(err.Error())
		data.Error = "could not save draft"
		data.Article = &art
		data.Tags = strings.Join(art.Tags, ", ")
		a.renderGodPage(w, http.StatusInternalServerError, "god_form", data)
		return
	}
	// Save translation if provided
	parseAndSaveTranslation(r, art.Slug, "pt")
	// Redirect to edit form so the user can continue working on the draft.
	http.Redirect(w, r, "/god/articles/"+art.Slug+"/edit", http.StatusSeeOther)
}

// godArticleEditHandler renders the edit form, pre-filled from the database.
func (a *app) godArticleEditHandler(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	art := content.GetArticleAny(r.Context(), slug)
	if art == nil {
		a.notFoundHandler(w, r)
		return
	}
	data := a.newGodData(r, false)
	data.Article = art
	data.Tags = strings.Join(art.Tags, ", ")
	// Load translation for PT
	data.Translation = content.GetTranslation(r.Context(), slug, "pt")
	a.renderGodPage(w, http.StatusOK, "god_form", data)
}

// godArticleUpdateHandler validates and stores changes to an existing article.
// The slug comes from the URL but can change for drafts (not published ones).
func (a *app) godArticleUpdateHandler(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	data := a.newGodData(r, false)
	if !a.validCSRF(r) {
		data.Error = "invalid request token, try again"
		data.Article = content.GetArticleAny(r.Context(), slug)
		data.Translation = content.GetTranslation(r.Context(), slug, "pt")
		a.renderGodPage(w, http.StatusForbidden, "god_form", data)
		return
	}
	art := parseArticleForm(r)
	if art.Title == "" || art.Date == "" {
		data.Error = "title and date are required"
		art.Slug = slug
		data.Article = &art
		data.Tags = strings.Join(art.Tags, ", ")
		data.Translation = content.GetTranslation(r.Context(), slug, "pt")
		a.renderGodPage(w, http.StatusUnprocessableEntity, "god_form", data)
		return
	}
	// Preserve the existing love count and check if slug needs changing.
	existing := content.GetArticleAny(r.Context(), slug)
	if existing != nil {
		art.InitialLove = existing.InitialLove
		// Allow slug changes for drafts only.
		if existing.Status == "draft" && art.Slug != "" && art.Slug != slug {
			if err := content.ChangeSlug(r.Context(), slug, art.Slug); err != nil {
				a.errorLogger.Print(err.Error())
				data.Error = "could not update slug"
				art.Slug = slug
				data.Article = &art
				data.Tags = strings.Join(art.Tags, ", ")
				data.Translation = content.GetTranslation(r.Context(), slug, "pt")
				a.renderGodPage(w, http.StatusInternalServerError, "god_form", data)
				return
			}
			slug = art.Slug
		} else {
			art.Slug = slug
		}
	} else {
		art.Slug = slug
	}
	if err := content.Save(r.Context(), art); err != nil {
		a.errorLogger.Print(err.Error())
		data.Error = "could not save the article"
		data.Article = &art
		data.Tags = strings.Join(art.Tags, ", ")
		data.Translation = content.GetTranslation(r.Context(), slug, "pt")
		a.renderGodPage(w, http.StatusInternalServerError, "god_form", data)
		return
	}
	// Save translation if provided
	parseAndSaveTranslation(r, slug, "pt")
	http.Redirect(w, r, "/god/articles/"+slug+"/edit", http.StatusSeeOther)
}

// godArticleDeleteHandler removes an article by its URL slug and returns to
// the list. Like every admin write, it requires a valid CSRF token.
func (a *app) godArticleDeleteHandler(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	data := a.newGodData(r, false)
	if !a.validCSRF(r) {
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
func (a *app) validCSRF(r *http.Request) bool {
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

// parseAndSaveTranslation parses translation fields from the form and saves them.
func parseAndSaveTranslation(r *http.Request, slug, lang string) {
	r.ParseForm()
	title := strings.TrimSpace(r.FormValue("title_" + lang))
	subtitle := strings.TrimSpace(r.FormValue("subtitle_" + lang))
	excerpt := strings.TrimSpace(r.FormValue("excerpt_" + lang))
	imageCaption := strings.TrimSpace(r.FormValue("image_caption_" + lang))
	body := r.FormValue("body_" + lang)

	// Only save if at least one field is provided
	if title != "" || subtitle != "" || excerpt != "" || imageCaption != "" || body != "" {
		t := content.ArticleTranslation{
			Slug:         slug,
			Lang:         lang,
			Title:        title,
			Subtitle:     subtitle,
			Excerpt:      excerpt,
			ImageCaption: imageCaption,
			Body:         body,
		}
		content.SaveTranslation(r.Context(), t)
	}
}

// previewToken is a short-lived, session-bound token for draft preview.
type previewToken struct {
	Slug      string
	ExpiresAt time.Time
	SessionID string
}

var (
	previewMu    sync.Mutex
	previewStore = map[string]*previewToken{}
	previewTTL   = 15 * time.Minute
)

func genPreviewToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// godPreviewTokenHandler creates a one-time preview token and redirects to the
// preview URL. The token is bound to the caller's session and expires after
// previewTTL.
func (a *app) godPreviewTokenHandler(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	sess := a.sessionToken(r)
	if sess == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	token := genPreviewToken()
	previewMu.Lock()
	previewStore[token] = &previewToken{
		Slug:      slug,
		ExpiresAt: time.Now().Add(previewTTL),
		SessionID: sess,
	}
	previewMu.Unlock()
	http.Redirect(w, r, "/god/articles/preview/"+token, http.StatusSeeOther)
}

// godPreviewHandler renders a draft article for preview. The token must be
// valid, not expired, and belong to the current session. Search engines are
// told not to index the page.
func (a *app) godPreviewHandler(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	sess := a.sessionToken(r)

	previewMu.Lock()
	pt, ok := previewStore[token]
	if ok && (pt.ExpiresAt.Before(time.Now()) || pt.SessionID != sess) {
		delete(previewStore, token)
		ok = false
	}
	previewMu.Unlock()

	if !ok {
		a.notFoundHandler(w, r)
		return
	}

	art := content.GetArticleAny(r.Context(), pt.Slug)
	if art == nil {
		a.notFoundHandler(w, r)
		return
	}

	w.Header().Set("X-Robots-Tag", "noindex")
	data := struct {
		pageData
		Article *content.Article
	}{pageData: a.newPageData(r, "articles"), Article: art}
	renderPage(w, http.StatusOK, "article", data)
}
