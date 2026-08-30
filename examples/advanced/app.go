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
	views, err := thema.New("./themes", "khotel")
	if err != nil {
		return nil, err
	}
	return &application{
		views:      views,
		assetFiles: http.FileServer(http.Dir("./themes/khotel/assets")),
		logger:     log.Default(),
	}, nil
}

func (app *application) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", app.home)
	mux.HandleFunc("GET /rooms", app.rooms)
	mux.HandleFunc("GET /rooms/{slug}", app.room)
	mux.HandleFunc("GET /amenities", app.amenities)
	mux.HandleFunc("GET /local-area", app.localArea)
	mux.HandleFunc("GET /getting-here", app.gettingHere)
	mux.HandleFunc("GET /health", app.health)
	mux.Handle("GET /assets/khotel/", http.StripPrefix("/assets/khotel/", app.assetHandler()))
	mux.HandleFunc("GET /", app.notFound)
	return mux
}

func (app *application) assetHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		app.assetFiles.ServeHTTP(w, r)
	})
}
