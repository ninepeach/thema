package main

import (
	"context"
	"log"
	"os"

	"github.com/ninepeach/thema"
)

func main() {
	views := thema.Must(thema.New("examples/theme/themes", "minimal"))
	if err := views.Render(context.Background(), os.Stdout, "pages/home", nil); err != nil {
		log.Fatal(err)
	}
	// Call Refresh from application-owned scheduling or deployment code.
	// Thema never starts a hidden ticker or background goroutine.
	_, _ = views.Refresh(context.Background())
}

