package main

import (
	"log"
	"net/http"

	"github.com/ninepeach/thema"
)

type RoomView struct {
	Name        string
	Description string
	Price       string
}

type HomeView struct {
	Title string
	Rooms []RoomView
}

type RoomsView struct {
	Title string
	Rooms []RoomView
}

var rooms = []RoomView{
	{Name: "Garden Room", Description: "A quiet room overlooking the garden.", Price: "$120"},
	{Name: "Mountain Room", Description: "A bright room with a wide mountain view.", Price: "$145"},
}

func newHandler() http.Handler {
	views := thema.Must(thema.New("./themes", "advanced"))
	if err := views.Contribute("home.announcement", thema.Contribution{
		ID: "seasonal-offer", Template: "contributions/announcement", Order: 10,
	}); err != nil {
		panic(err)
	}

	mux := http.NewServeMux()
	mux.Handle("/assets/advanced/", http.StripPrefix(
		"/assets/advanced/",
		http.FileServer(http.Dir("./themes/advanced/assets")),
	))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		locale := r.URL.Query().Get("lang")
		if locale == "" {
			locale = "en"
		}

		var name string
		var data any
		switch r.URL.Path {
		case "/":
			name = "pages/home"
			data = HomeView{Title: "Thema Guesthouse", Rooms: rooms}
		case "/rooms":
			name = "pages/rooms"
			data = RoomsView{Title: "Thema Guesthouse", Rooms: rooms}
		default:
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := views.Render(r.Context(), w, name, data, thema.WithLocale(locale)); err != nil {
			http.Error(w, "render failed", http.StatusInternalServerError)
		}
	})
	return mux
}

func main() {
	log.Println("advanced example: http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", newHandler()))
}
