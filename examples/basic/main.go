package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/ninepeach/thema"
)

type Product struct {
	Name        string
	Description string
	Price       int
}

type PageView struct {
	Title       string
	CurrentPath string
	Products    []Product
}

type application struct {
	views  *thema.Engine
	assets http.Handler
}

func main() {
	app, err := newApplication()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Thema basic example listening on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", app.routes()))
}

func newApplication() (*application, error) {
	views, err := thema.New("./themes", "basic", thema.WithFuncs(template.FuncMap{
		"formatPrice": func(price int) string { return fmt.Sprintf("$%d", price) },
	}))
	if err != nil {
		return nil, err
	}
	return &application{
		views:  views,
		assets: http.FileServer(http.Dir("./themes/basic/assets")),
	}, nil
}

func (app *application) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", app.home)
	mux.HandleFunc("GET /about", app.about)
	mux.HandleFunc("GET /products", app.products)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.Handle("GET /assets/", http.HandlerFunc(app.serveAsset))
	return mux
}

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	app.render(w, r, "pages/home", PageView{Title: "Acme Studio", CurrentPath: "/"})
}

func (app *application) about(w http.ResponseWriter, r *http.Request) {
	app.render(w, r, "pages/about", PageView{Title: "About · Acme Studio", CurrentPath: "/about"})
}

func (app *application) products(w http.ResponseWriter, r *http.Request) {
	app.render(w, r, "pages/products", PageView{
		Title:       "Products · Acme Studio",
		CurrentPath: "/products",
		Products: []Product{
			{Name: "Starter", Description: "Simple tools for getting started.", Price: 19},
			{Name: "Team", Description: "Collaboration tools for growing teams.", Price: 49},
			{Name: "Scale", Description: "More capacity for larger workloads.", Price: 99},
		},
	})
}

func (app *application) render(w http.ResponseWriter, r *http.Request, name string, data PageView) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := app.views.Render(r.Context(), w, name, data); err != nil {
		log.Printf("render %s: %v", name, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (app *application) serveAsset(w http.ResponseWriter, r *http.Request) {
	prefix := "/assets/"
	if len(r.URL.Path) >= len("/assets/basic/") && r.URL.Path[:len("/assets/basic/")] == "/assets/basic/" {
		prefix = "/assets/basic/"
	}
	http.StripPrefix(prefix, app.assets).ServeHTTP(w, r)
}
