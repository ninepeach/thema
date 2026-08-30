package main

import (
	"bytes"
	"net/http"

	"github.com/ninepeach/thema"
)

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	app.renderPage(w, r, http.StatusOK, "pages/home", "", "pages.home.title")
}

func (app *application) rooms(w http.ResponseWriter, r *http.Request) {
	app.renderPage(w, r, http.StatusOK, "pages/rooms", "rooms", "pages.rooms.title")
}

func (app *application) room(w http.ResponseWriter, r *http.Request) {
	app.renderPage(w, r, http.StatusOK, "pages/room", "rooms", "pages.room.title")
}

func (app *application) amenities(w http.ResponseWriter, r *http.Request) {
	app.renderPage(w, r, http.StatusOK, "pages/amenities", "amenities", "pages.amenities.title")
}

func (app *application) localArea(w http.ResponseWriter, r *http.Request) {
	app.renderPage(w, r, http.StatusOK, "pages/local-area", "local-area", "pages.local_area.title")
}

func (app *application) gettingHere(w http.ResponseWriter, r *http.Request) {
	app.renderPage(w, r, http.StatusOK, "pages/getting-here", "getting-here", "pages.getting_here.title")
}

func (app *application) notFound(w http.ResponseWriter, r *http.Request) {
	locale := requestedLocale(r)
	data := pageView(r.URL.Path, "", locale, "errors.not_found.title", "errors.not_found.description")
	app.render(w, r, http.StatusNotFound, "pages/404", data)
}

func (app *application) health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write([]byte("ok\n"))
}

func (app *application) renderPage(w http.ResponseWriter, r *http.Request, status int, templateName, activeNav, title string) {
	locale := requestedLocale(r)
	data := pageView(r.URL.Path, activeNav, locale, title, "site.description")
	app.render(w, r, status, templateName, data)
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
	data := pageView(r.URL.Path, "", locale, "errors.internal.title", "errors.internal.description")
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
