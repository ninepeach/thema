package main

import (
	"context"
	"html/template"
	"log"
	"os"
	"strings"

	"github.com/ninepeach/thema"
)

func main() {
	views := thema.Must(thema.New(
		"examples/basic/themes",
		"default",
		thema.WithFuncs(template.FuncMap{"upper": strings.ToUpper}),
	))
	data := struct{ Title string }{Title: "Thema"}
	if err := views.Render(context.Background(), os.Stdout, "pages/home", data); err != nil {
		log.Fatal(err)
	}
}
