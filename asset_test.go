package thema

import (
	"context"
	"path/filepath"
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

func TestAssetGenerationChangesWithThemeContent(t *testing.T) {
	repository := newTestTheme(t, map[string]string{
		"pages/home.html": `{{asset "css/app.css"}}`,
	}, nil, map[string]string{"css/app.css": "old"})
	views, err := New(repository, "default")
	if err != nil {
		t.Fatal(err)
	}
	var before strings.Builder
	if err := views.Render(context.Background(), &before, "pages/home", nil); err != nil {
		t.Fatal(err)
	}
	asset := filepath.Join(repository, "default", "assets", "css", "app.css")
	writeTestFile(t, asset, []byte("new"))
	if changed, err := views.Refresh(context.Background()); err != nil || !changed {
		t.Fatalf("Refresh() = %v, %v", changed, err)
	}
	var after strings.Builder
	if err := views.Render(context.Background(), &after, "pages/home", nil); err != nil {
		t.Fatal(err)
	}
	if before.String() == after.String() {
		t.Fatalf("asset generation did not change: %q", before.String())
	}
}

func TestMissingAssetStillGeneratesURL(t *testing.T) {
	repository := newTestTheme(t, map[string]string{
		"pages/home.html": `before{{asset "css/missing.css"}}after`,
	}, nil, nil)

	views, err := New(repository, "default")
	if err != nil {
		t.Fatal(err)
	}

	var output strings.Builder

	if err := views.Render(
		context.Background(),
		&output,
		"pages/home",
		nil,
	); err != nil {
		t.Fatal(err)
	}

	got := output.String()

	if !strings.HasPrefix(
		got,
		`before/assets/default/css/missing.css?v=`,
	) {
		t.Fatalf("unexpected output %q", got)
	}

	if !strings.HasSuffix(got, "after") {
		t.Fatalf("unexpected output %q", got)
	}
}
