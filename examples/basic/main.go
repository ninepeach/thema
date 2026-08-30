package main

import (
	"html/template"
	"log"
	"net/http"
	"strings"

	"github.com/ninepeach/thema"
)

type Site struct {
	Name        string
	Description string
}

type Example struct {
	Name        string
	Description string
	Recommended bool
	Tags        []string
}

type PageView struct {
	Site        Site
	Title       string
	Description string
	CurrentPath string
	Examples    []Example
	EscapedText string
}

type application struct {
	views  *thema.Engine
	assets http.Handler
}

var examples = []Example{
	{
		Name:        "Typed view data",
		Description: "Render typed Go data with native root-dot access.",
		Recommended: true,
		Tags:        []string{"Data", "ViewModel"},
	},
	{
		Name:        "Template composition",
		Description: "Compose pages with native <html/template> templates.",
		Recommended: true,
		Tags:        []string{"template", "components"},
	},
	{
		Name:        "Theme assets",
		Description: "Reference versioned CSS and other theme assets.",
		Recommended: false,
		Tags:        []string{"asset", "CSS"},
	},
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
	views, err := thema.New(
		"./themes",
		"basic",
		thema.WithDefaultLocale("en"),
		thema.WithFuncs(template.FuncMap{"join": strings.Join}),
	)
	if err != nil {
		return nil, err
	}
	if err := views.Contribute("page.notice", thema.Contribution{
		ID:       "tutorial-note",
		Template: "contributions/tutorial-note",
		Order:    10,
	}); err != nil {
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
	mux.HandleFunc("GET /templates", app.templates)
	mux.HandleFunc("GET /runtime", app.runtime)
	mux.HandleFunc("GET /reference", app.reference)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.Handle("GET /assets/basic/", http.StripPrefix("/assets/basic/", app.assets))
	return mux
}

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	data := newPage("Getting Started", "/")
	data.Description = "Build and render a small theme with standard Go html/template semantics."
	app.render(w, r, "pages/home", data)
}

func (app *application) templates(w http.ResponseWriter, r *http.Request) {
	data := newPage("Templates & Data", "/templates")
	data.Description = "Use typed data, Go template actions, components, helpers, escaping, and assets."
	data.Examples = examples
	data.EscapedText = "<html/template> escapes ordinary application strings."
	app.render(w, r, "pages/templates", data)
}

func (app *application) runtime(w http.ResponseWriter, r *http.Request) {
	data := newPage("Runtime Features", "/runtime")
	data.Description = "Add localization, extension points, atomic refresh, and cancellation without changing template data."
	app.render(w, r, "pages/runtime", data)
}

func (app *application) reference(w http.ResponseWriter, r *http.Request) {
	data := newPage("Core API", "/reference")
	data.Description = "A compact map of the complete Thema v0.1 public surface and its observable safety contract."
	app.render(w, r, "pages/reference", data)
}

func newPage(title, path string) PageView {
	return PageView{
		Site: Site{
			Name:        "Thema",
			Description: "A small Go-native theme runtime.",
		},
		Title:       title,
		CurrentPath: path,
	}
}

func (app *application) render(w http.ResponseWriter, r *http.Request, name string, data PageView) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := app.views.Render(r.Context(), w, name, data); err != nil {
		log.Printf("render %s: %v", name, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
