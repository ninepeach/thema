package main

import (
	"context"
	"log"
	"os"

	"github.com/ninepeach/thema"
)

func main() {
	views := thema.Must(thema.New("examples/i18n/themes", "default"))
	data := struct{ Name string }{Name: "Stone"}
	if err := views.Render(
		context.Background(),
		os.Stdout,
		"pages/home",
		data,
		thema.WithLocale("ja"),
	); err != nil {
		log.Fatal(err)
	}
}
