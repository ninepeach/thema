package main

import (
	"log"
	"net/http"

	"github.com/ninepeach/thema"
)

type pageView struct {
	Title string
}

func newHandler() http.Handler {
	views := thema.Must(thema.New("./themes", "minimal"))
	mux := http.NewServeMux()
	mux.Handle("/assets/minimal/", http.StripPrefix(
		"/assets/minimal/",
		http.FileServer(http.Dir("./themes/minimal/assets")),
	))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := views.Render(r.Context(), w, "pages/home", pageView{Title: "Hello, Thema"}); err != nil {
			http.Error(w, "render failed", http.StatusInternalServerError)
		}
	})
	return mux
}

func main() {
	log.Println("minimal example: http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", newHandler()))
}
