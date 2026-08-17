// Experiment handlers: the public /experiments/{slug} pages and the
// per-experiment form posts. Experiments are registered in
// internal/experiments; each experiment owns its template folder and any
// static assets. The conversion endpoint here is img2webp-specific because
// it is the only experiment so far.
package main

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/borumbombum/borum-api/internal/auth"
	"github.com/borumbombum/borum-api/internal/experiments"
	"github.com/borumbombum/borum-api/internal/imgconv"
	"github.com/go-chi/chi/v5"
)

// maxUploadBytes caps how large a converted upload may be. Anything bigger is
// rejected before it is read into memory.
const maxUploadBytes = 25 << 20 // 25 MB

// maxImagePixels and maxImageEdge bound how large a bitmap may be. The file cap
// alone does not stop a header-lying PNG from decoding to a giant bitmap, so
// dimensions are checked via image.DecodeConfig before any decode happens.
const (
	maxImagePixels = 40_000_000 // 40 MP, ~160 MB in RGBA
	maxImageEdge   = 10_000
)

// experimentHandler renders one experiment's own page against the shared
// experiment layout. Unknown or disabled experiments get the site 404 so
// disabled experiments are unlisted.
func (a *app) experimentHandler(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	lang := r.Context().Value("lang").(string)
	items := experiments.List(r.Context())
	var item *experiments.Item
	for i := range items {
		if items[i].Slug == slug {
			item = &items[i]
			break
		}
	}
	if item == nil || !item.Enabled {
		a.notFoundHandler(w, r)
		return
	}
	
	// Load translation if available
	if lang != "en" {
		trans, err := experiments.GetTranslation(r.Context(), slug, lang)
		if err == nil && trans != nil {
			if trans.Title != "" {
				item.Experiment.Title = trans.Title
			}
			if trans.Description != "" {
				item.Experiment.Description = trans.Description
			}
			if trans.Intro != "" {
				item.IntroPT = trans.Intro
			}
		}
	}
	
	data := struct {
		pageData
		Experiment    *experiments.Item
		ConvertPerMin int
		MaxPixelsMP   int
		MaxEdge       int
		MaxUploadMB   int
		Lang          string
	}{
		pageData:      a.newPageData(r, "experiments"),
		Experiment:    item,
		ConvertPerMin: a.convertLimiter.Max(),
		MaxPixelsMP:   maxImagePixels / 1_000_000,
		MaxEdge:       maxImageEdge,
		MaxUploadMB:   maxUploadBytes >> 20,
		Lang:          lang,
	}
	renderPage(w, http.StatusOK, "experiment_"+slug, data)
}

// img2webpConvertHandler converts one uploaded PNG/JPEG to WebP and returns it
// as a file download. Quality defaults to 80 lossy; advanced form fields can
// override quality, mode (lossy/lossless/auto) and method.
func (a *app) img2webpConvertHandler(w http.ResponseWriter, r *http.Request) {
	if !experiments.Enabled(r.Context(), "img2webp") {
		http.Error(w, "experiment disabled", http.StatusNotFound)
		return
	}
	// The logged-in admin is exempt from the per-visitor conversion limit; it
	// exists only to curb anonymous abuse.
	if _, authed := auth.SessionFrom(r.Context()); !authed {
		if !a.convertLimiter.Allow(clientIP(r)) {
			w.Header().Set("Retry-After", "60")
			http.Error(w, "too many conversions, slow down", http.StatusTooManyRequests)
			return
		}
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "could not read the upload", http.StatusBadRequest)
		return
	}
	file, header, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "choose a PNG or JPEG image first", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Peek at the image header before allocating anything: a tiny file can
	// still claim enormous dimensions, and decode would allocate by pixel
	// count. Reject oversized or non-PNG/JPEG images here.
	cfg, format, err := image.DecodeConfig(file)
	if err != nil {
		http.Error(w, "not a valid image", http.StatusBadRequest)
		return
	}
	if format != "png" && format != "jpeg" {
		http.Error(w, "unsupported format, expected PNG or JPEG", http.StatusBadRequest)
		return
	}
	if cfg.Width > maxImageEdge || cfg.Height > maxImageEdge || cfg.Width*cfg.Height > maxImagePixels {
		http.Error(w, "image too large", http.StatusBadRequest)
		return
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		http.Error(w, "could not read the upload", http.StatusBadRequest)
		return
	}

	opts, err := parseConvertOptions(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	res, err := imgconv.Convert(file, opts)
	if err != nil {
		a.errorLogger.Print(err.Error())
		http.Error(w, "could not convert this image", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "image/webp")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename=%q`, webpName(header.Filename)))
	w.Header().Set("X-WebP-Mode", res.Label)
	w.WriteHeader(http.StatusOK)
	w.Write(res.Data)
}

// parseConvertOptions reads the advanced form fields with safe defaults: 80
// lossy, method 4.
func parseConvertOptions(r *http.Request) (imgconv.Options, error) {
	o := imgconv.Options{Quality: imgconv.DefaultQuality, Method: imgconv.DefaultMethod}

	switch strings.TrimSpace(r.FormValue("mode")) {
	case "", "lossy":
		o.Mode = imgconv.ModeLossy
	case "lossless":
		o.Mode = imgconv.ModeLossless
	case "auto":
		o.Mode = imgconv.ModeAuto
	default:
		return o, fmt.Errorf("unknown mode")
	}

	if q := strings.TrimSpace(r.FormValue("quality")); q != "" {
		n, err := strconv.ParseFloat(q, 32)
		if err != nil || n < 1 || n > 100 {
			return o, fmt.Errorf("quality must be a number between 1 and 100")
		}
		o.Quality = float32(n)
	}

	if m := strings.TrimSpace(r.FormValue("method")); m != "" {
		n, err := strconv.Atoi(m)
		if err != nil || n < 0 || n > 6 {
			return o, fmt.Errorf("method must be between 0 and 6")
		}
		o.Method = n
	}

	return o, nil
}

// webpName derives a download filename from the uploaded one, always ending in
// .webp.
func webpName(name string) string {
	base := filepath.Base(name)
	base = strings.TrimSuffix(base, filepath.Ext(base))
	if base == "" || base == "." {
		base = "image"
	}
	return base + ".webp"
}
