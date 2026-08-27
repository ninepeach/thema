package thema

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestAssetUsesValidatedVersionedThemeURL(t *testing.T) {
	repository := newTestTheme(t, map[string]string{
		"pages/home.html": `<link href="{{asset "css/app.css"}}">`,
	}, nil, map[string]string{"css/app.css": "body{}"})
	views, err := New(repository, "default")
	if err != nil {
		t.Fatal(err)
	}
	var output strings.Builder
	if err := views.Render(context.Background(), &output, "pages/home", nil); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(output.String(), `<link href="/assets/default/css/app.css?v=`) || !strings.HasSuffix(output.String(), `">`) {
		t.Fatalf("unexpected asset URL %q", output.String())
	}
}

func TestMissingAssetFailsWithoutCommit(t *testing.T) {
	repository := newTestTheme(t, map[string]string{
		"pages/home.html": `before{{asset "css/missing.css"}}after`,
	}, nil, nil)
	views, err := New(repository, "default")
	if err != nil {
		t.Fatal(err)
	}
	output := strings.Builder{}
	output.WriteString("existing")
	err = views.Render(context.Background(), &output, "pages/home", nil)
	if !errors.Is(err, ErrRender) || !errors.Is(err, ErrInvalidPath) {
		t.Fatalf("unexpected error %v", err)
	}
	if output.String() != "existing" {
		t.Fatalf("destination changed to %q", output.String())
	}
}
