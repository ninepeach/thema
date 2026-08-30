package main

import (
	"bytes"
	"net/http"
	"strings"

	"github.com/ninepeach/thema"
)

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	locale := requestedLocale(r)
	data := pageView("/", locale, "site.title", "site.description")
	data.Services = serviceViews()
	data.Projects = projectViews()
	data.Stats = statViews()
	app.render(w, r, http.StatusOK, "pages/home", data)
}

func (app *application) services(w http.ResponseWriter, r *http.Request) {
	locale := requestedLocale(r)
	data := pageView("/services", locale, "services.page_title", "services.intro")
	data.Services = serviceViews()
	app.render(w, r, http.StatusOK, "pages/services", data)
}

func (app *application) work(w http.ResponseWriter, r *http.Request) {
	locale := requestedLocale(r)
	data := pageView("/work", locale, "work.page_title", "work.intro")
	data.Projects = projectViews()
	data.Stats = statViews()
	app.render(w, r, http.StatusOK, "pages/work", data)
}

func (app *application) about(w http.ResponseWriter, r *http.Request) {
	locale := requestedLocale(r)
	data := pageView("/about", locale, "about.page_title", "about.intro")
	data.Stats = statViews()
	app.render(w, r, http.StatusOK, "pages/about", data)
}

func (app *application) contact(w http.ResponseWriter, r *http.Request) {
	locale := requestedLocale(r)
	data := pageView("/contact", locale, "contact.page_title", "contact.intro")
	app.render(w, r, http.StatusOK, "pages/contact", data)
}

func (app *application) notFound(w http.ResponseWriter, r *http.Request) {
	locale := requestedLocale(r)
	data := pageView(r.URL.Path, locale, "errors.not_found.title", "errors.not_found.description")
	app.render(w, r, http.StatusNotFound, "pages/404", data)
}

func (app *application) health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write([]byte("ok\n"))
}

func (app *application) serveAsset(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	prefix := "/assets/"
	if strings.HasPrefix(r.URL.Path, "/assets/northstar/") {
		prefix = "/assets/northstar/"
	}
	http.StripPrefix(prefix, app.assetFiles).ServeHTTP(w, r)
}

func (app *application) render(w http.ResponseWriter, r *http.Request, status int, name string, data PageView) {
	var body bytes.Buffer
	if err := app.views.Render(r.Context(), &body, name, data, thema.WithLocale(data.Locale)); err != nil {
		app.serverError(w, r, data.Locale, err)
		return
	}
	writeHTML(w, status, body.Bytes())
}

func (app *application) serverError(w http.ResponseWriter, r *http.Request, locale string, renderErr error) {
	app.logger.Printf("render failed: %v", renderErr)
	data := pageView(r.URL.Path, locale, "errors.internal.title", "errors.internal.description")
	var body bytes.Buffer
	if err := app.views.Render(r.Context(), &body, "pages/500", data, thema.WithLocale(locale)); err != nil {
		app.logger.Printf("render themed 500 page: %v", err)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	writeHTML(w, http.StatusInternalServerError, body.Bytes())
}

func writeHTML(w http.ResponseWriter, status int, body []byte) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func requestedLocale(r *http.Request) string {
	if r.URL.Query().Get("lang") == "zh" {
		return "zh"
	}
	return "en"
}
