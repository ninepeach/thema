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
	views, err := thema.New("./themes", "basic", thema.WithFuncs(template.FuncMap{
		"join": strings.Join,
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
	mux.HandleFunc("GET /templates", app.templates)
	mux.HandleFunc("GET /data", app.data)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("GET /assets/", app.serveAsset)
	return mux
}

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	data := newPage("Getting Started", "/")
	data.Description = "Build and render a small theme with standard Go html/template semantics."
	app.render(w, r, "pages/home", data)
}

func (app *application) templates(w http.ResponseWriter, r *http.Request) {
	data := newPage("Templates", "/templates")
	data.Description = "Compose pages and reusable components with native Go template actions."
	data.Examples = examples[1:]
	app.render(w, r, "pages/templates", data)
}

func (app *application) data(w http.ResponseWriter, r *http.Request) {
	data := newPage("Data & Logic", "/data")
	data.Description = "Use typed application data with familiar Go template control actions."
	data.Examples = examples
	app.render(w, r, "pages/data", data)
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

func (app *application) serveAsset(w http.ResponseWriter, r *http.Request) {
	prefix := "/assets/"
	if strings.HasPrefix(r.URL.Path, "/assets/basic/") {
		prefix = "/assets/basic/"
	}
	http.StripPrefix(prefix, app.assets).ServeHTTP(w, r)
}
