package main

import (
	"log"
	"net/http"

	"github.com/ninepeach/thema"
)

type application struct {
	views      *thema.Engine
	assetFiles http.Handler
	logger     *log.Logger
}

func newApplication() (*application, error) {
	views, err := thema.New("./themes", "northstar")
	if err != nil {
		return nil, err
	}
	if err := views.Contribute("home.announcement", thema.Contribution{
		ID:       "project-availability",
		Template: "contributions/announcement",
		Order:    10,
	}); err != nil {
		return nil, err
	}
	return &application{
		views:      views,
		assetFiles: http.FileServer(http.Dir("./themes/northstar/assets")),
		logger:     log.Default(),
	}, nil
}

func (app *application) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", app.home)
	mux.HandleFunc("GET /services", app.services)
	mux.HandleFunc("GET /work", app.work)
	mux.HandleFunc("GET /about", app.about)
	mux.HandleFunc("GET /contact", app.contact)
	mux.HandleFunc("GET /health", app.health)
	mux.HandleFunc("GET /assets/", app.serveAsset)
	mux.HandleFunc("GET /", app.notFound)
	return mux
}
