package thema

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestNewRenderPreservesNativeRootCompositionAndHelpers(t *testing.T) {
	repository := newTestTheme(t, map[string]string{
		"components/header.html": `{{define "components/header"}}<header>{{.Title}}</header>{{end}}`,
		"pages/home.html":        `{{template "components/header" .}}<h1>{{upper .Title}}</h1>`,
	}, nil, nil)
	views, err := New(repository, "default", WithFuncs(template.FuncMap{
		"upper": strings.ToUpper,
	}))
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := views.Render(context.Background(), &output, "pages/home", struct{ Title string }{Title: `<Welcome>`}); err != nil {
		t.Fatal(err)
	}
	want := `<header>&lt;Welcome&gt;</header><h1>&lt;WELCOME&gt;</h1>`
	if output.String() != want {
		t.Fatalf("Render() = %q, want %q", output.String(), want)
	}
}

func TestAutomaticLogicalTemplateName(t *testing.T) {
	repository := newTestTheme(t, map[string]string{
		"pages/rooms/index.html": `<p>{{.Name}}</p>`,
	}, nil, nil)
	views, err := New(repository, "default")
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := views.Render(context.Background(), &output, "pages/rooms/index", map[string]string{"Name": "Room"}); err != nil {
		t.Fatal(err)
	}
	if output.String() != "<p>Room</p>" {
		t.Fatalf("unexpected output %q", output.String())
	}
	if err := views.Render(context.Background(), &output, "pages/rooms", nil); !errors.Is(err, ErrTemplateNotFound) {
		t.Fatalf("expected ErrTemplateNotFound, got %v", err)
	}
}

func TestRenderCommitsOnlyAfterSuccessfulExecution(t *testing.T) {
	repository := newTestTheme(t, map[string]string{
		"pages/home.html": `before{{fail}}after`,
	}, nil, nil)
	views, err := New(repository, "default", WithFuncs(template.FuncMap{
		"fail": func() (string, error) { return "", errors.New("boom") },
	}))
	if err != nil {
		t.Fatal(err)
	}
	output := bytes.NewBufferString("existing")
	err = views.Render(context.Background(), output, "pages/home", nil)
	if !errors.Is(err, ErrRender) {
		t.Fatalf("expected ErrRender, got %v", err)
	}
	if output.String() != "existing" {
		t.Fatalf("destination changed to %q", output.String())
	}
}

func TestRefreshActivatesOnlyValidChangedCandidates(t *testing.T) {
	repository := newTestTheme(t, map[string]string{
		"pages/home.html": `old`,
	}, nil, nil)
	views, err := New(repository, "default")
	if err != nil {
		t.Fatal(err)
	}
	if changed, err := views.Refresh(context.Background()); err != nil || changed {
		t.Fatalf("unchanged Refresh() = %v, %v", changed, err)
	}
	page := filepath.Join(repository, "default", "templates", "pages", "home.html")
	writeTestFile(t, page, []byte(`new`))
	if changed, err := views.Refresh(context.Background()); err != nil || !changed {
		t.Fatalf("changed Refresh() = %v, %v", changed, err)
	}
	assertRender(t, views, "pages/home", nil, "new")

	writeTestFile(t, page, []byte(`{{`))
	changed, err := views.Refresh(context.Background())
	if changed || !errors.Is(err, ErrRefresh) || !errors.Is(err, ErrInvalidTheme) {
		t.Fatalf("invalid Refresh() = %v, %v", changed, err)
	}
	assertRender(t, views, "pages/home", nil, "new")
}

func TestRefreshKeepsInFlightRenderOnOldSnapshot(t *testing.T) {
	repository := newTestTheme(t, map[string]string{
		"pages/home.html": `old-{{gate}}`,
	}, nil, nil)
	entered := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	views, err := New(repository, "default", WithFuncs(template.FuncMap{
		"gate": func() string {
			once.Do(func() {
				close(entered)
				<-release
			})
			return "done"
		},
	}))
	if err != nil {
		t.Fatal(err)
	}
	type result struct {
		output string
		err    error
	}
	finished := make(chan result, 1)
	go func() {
		var output bytes.Buffer
		err := views.Render(context.Background(), &output, "pages/home", nil)
		finished <- result{output: output.String(), err: err}
	}()
	<-entered
	page := filepath.Join(repository, "default", "templates", "pages", "home.html")
	writeTestFile(t, page, []byte(`new-{{gate}}`))
	if changed, err := views.Refresh(context.Background()); err != nil || !changed {
		t.Fatalf("Refresh() = %v, %v", changed, err)
	}
	close(release)
	old := <-finished
	if old.err != nil || old.output != "old-done" {
		t.Fatalf("in-flight render = %q, %v", old.output, old.err)
	}
	assertRender(t, views, "pages/home", nil, "new-done")
}

func TestFuncsCanBeAddedAfterInitialization(t *testing.T) {
	repository := newTestTheme(t, map[string]string{
		"pages/home.html": `plain`,
	}, nil, nil)
	views, err := New(repository, "default")
	if err != nil {
		t.Fatal(err)
	}
	if err := views.Funcs(template.FuncMap{"upper": strings.ToUpper}); err != nil {
		t.Fatal(err)
	}
	page := filepath.Join(repository, "default", "templates", "pages", "home.html")
	writeTestFile(t, page, []byte(`{{upper .}}`))
	if changed, err := views.Refresh(context.Background()); err != nil || !changed {
		t.Fatalf("Refresh() = %v, %v", changed, err)
	}
	assertRender(t, views, "pages/home", "ready", "READY")
	if err := views.Funcs(template.FuncMap{"t": strings.ToUpper}); err == nil {
		t.Fatal("expected reserved helper rejection")
	}
}

func TestStrictMissingKeys(t *testing.T) {
	repository := newTestTheme(t, map[string]string{
		"pages/home.html": `before{{.Missing}}after`,
	}, nil, nil)
	views, err := New(repository, "default", WithStrictMissingKeys())
	if err != nil {
		t.Fatal(err)
	}
	output := bytes.NewBufferString("unchanged")
	err = views.Render(context.Background(), output, "pages/home", map[string]string{})
	if !errors.Is(err, ErrRender) {
		t.Fatalf("expected ErrRender, got %v", err)
	}
	if output.String() != "unchanged" {
		t.Fatalf("destination changed to %q", output.String())
	}
}

func TestMissingTemplateDependencyRejectsTheme(t *testing.T) {
	repository := newTestTheme(t, map[string]string{
		"pages/home.html": `{{template "components/missing" .}}`,
	}, nil, nil)
	_, err := New(repository, "default")
	if !errors.Is(err, ErrInvalidTheme) || !strings.Contains(err.Error(), `components/missing`) {
		t.Fatalf("unexpected error %v", err)
	}
	if strings.Contains(err.Error(), repository) {
		t.Fatalf("error exposes absolute repository path: %v", err)
	}
}

func TestContextCancellation(t *testing.T) {
	repository := newTestTheme(t, map[string]string{"pages/home.html": `hello`}, nil, nil)
	views, err := New(repository, "default")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var output bytes.Buffer
	err = views.Render(ctx, &output, "pages/home", nil)
	if !errors.Is(err, context.Canceled) || output.Len() != 0 {
		t.Fatalf("Render() = %q, %v", output.String(), err)
	}
}

func TestMust(t *testing.T) {
	engine := &Engine{}
	if Must(engine, nil) != engine {
		t.Fatal("Must changed Engine")
	}
	defer func() {
		if recover() == nil {
			t.Fatal("Must did not panic")
		}
	}()
	Must(nil, errors.New("boom"))
}

func assertRender(t *testing.T, views *Engine, name string, data any, want string, opts ...RenderOption) {
	t.Helper()
	var output bytes.Buffer
	if err := views.Render(context.Background(), &output, name, data, opts...); err != nil {
		t.Fatal(err)
	}
	if output.String() != want {
		t.Fatalf("Render() = %q, want %q", output.String(), want)
	}
}

func newTestTheme(t *testing.T, templates, locales, assets map[string]string) string {
	t.Helper()
	repository := t.TempDir()
	root := filepath.Join(repository, "default")
	writeTestFile(t, filepath.Join(root, "theme.json"), []byte(`{"name":"Default","version":"1.0.0","thema":"0.1"}`))
	for name, content := range templates {
		writeTestFile(t, filepath.Join(root, "templates", filepath.FromSlash(name)), []byte(content))
	}
	for locale, content := range locales {
		writeTestFile(t, filepath.Join(root, "locales", locale+".json"), []byte(content))
	}
	for name, content := range assets {
		writeTestFile(t, filepath.Join(root, "assets", filepath.FromSlash(name)), []byte(content))
	}
	return repository
}

func writeTestFile(t *testing.T, filename string, content []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, content, 0o644); err != nil {
		t.Fatal(err)
	}
}

func ExampleNew() {
	_ = fmt.Sprintf("%T", New)
	// See examples/basic for a complete executable example.
	// Output:
	// func(string, string, ...thema.Option) (*thema.Engine, error)
}
