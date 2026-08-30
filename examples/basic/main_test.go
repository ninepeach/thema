package main

import (
	"bytes"
	"context"
	"errors"
	"html/template"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/ninepeach/thema"
)

func TestPages(t *testing.T) {
	app, err := newApplication()
	if err != nil {
		t.Fatal(err)
	}
	handler := app.routes()

	tests := []struct {
		name     string
		path     string
		status   int
		contains []string
	}{
		{
			name: "getting started", path: "/", status: http.StatusOK,
			contains: []string{"Getting Started · Thema", "Declare the manifest", `thema.New("./themes", "basic")`, `"thema": "0.1"`},
		},
		{
			name: "templates", path: "/templates", status: http.StatusOK,
			contains: []string{"Templates &amp; Data · Thema", "Typed view data", "Start here", "Next step", "<li>Data</li>", "Topics: Data, ViewModel", "Rendered for Thema", `class="example-card"`},
		},
		{
			name: "runtime", path: "/runtime", status: http.StatusOK,
			contains: []string{"Runtime Features · Thema", "Thema runtime note", "tutorial-note", "Welcome to the Thema runtime tutorial.", "Slots and Contributions", "Refresh"},
		},
		{
			name: "reference", path: "/reference", status: http.StatusOK,
			contains: []string{"Core API · Thema", "WithDefaultLocale", "WithStrictMissingKeys", "WithAssetBaseURL", "WithFuncs", "WithLocale", "ErrTemplateNotFound", "ErrRefresh"},
		},
		{name: "health", path: "/health", status: http.StatusOK, contains: []string{"ok"}},
		{name: "unknown", path: "/does-not-exist", status: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := request(t, handler, tt.path)
			if response.Code != tt.status {
				t.Fatalf("status = %d, want %d", response.Code, tt.status)
			}
			for _, value := range tt.contains {
				if !strings.Contains(response.Body.String(), value) {
					t.Errorf("body does not contain %q", value)
				}
			}
		})
	}
}

func TestTemplatesUseNativeDataAndEscaping(t *testing.T) {
	app, err := newApplication()
	if err != nil {
		t.Fatal(err)
	}
	body := request(t, app.routes(), "/templates").Body.String()
	if !strings.Contains(body, "&lt;html/template&gt; escapes ordinary application strings.") {
		t.Fatalf("dynamic ViewModel string was not escaped: %s", body)
	}

	data := newPage("Templates & Data", "/templates")
	empty := renderTemplate(t, app.views, "pages/templates", data)
	if !strings.Contains(empty, "No examples available.") {
		t.Fatalf("range else output missing: %s", empty)
	}
}

func TestGeneratedAssetIsServed(t *testing.T) {
	app, err := newApplication()
	if err != nil {
		t.Fatal(err)
	}
	handler := app.routes()
	home := request(t, handler, "/")
	match := regexp.MustCompile(`href="([^"]*/assets/basic/css/app\.css\?v=[^"]+)"`).FindStringSubmatch(home.Body.String())
	if len(match) != 2 {
		t.Fatalf("generated asset URL not found: %s", home.Body.String())
	}
	asset := request(t, handler, match[1])
	if asset.Code != http.StatusOK {
		t.Fatalf("generated asset status = %d, want %d", asset.Code, http.StatusOK)
	}
	if !strings.Contains(asset.Body.String(), ".example-card") {
		t.Fatal("CSS response does not contain example styles")
	}
}

func TestI18nExampleUsesBuiltInTranslator(t *testing.T) {
	app, err := newApplication()
	if err != nil {
		t.Fatal(err)
	}
	data := newPage("Runtime Features", "/runtime")
	body := renderTemplate(t, app.views, "pages/runtime", data, thema.WithLocale("en-US"))
	if !strings.Contains(body, "Welcome to the Thema runtime tutorial.") {
		t.Fatalf("English translation with interpolation missing: %s", body)
	}
	fallback := renderTemplate(t, app.views, "pages/runtime", data, thema.WithLocale("fr"))
	if !strings.Contains(fallback, "Welcome to the Thema runtime tutorial.") {
		t.Fatalf("default-locale fallback missing: %s", fallback)
	}
	escaped := data
	escaped.Site.Name = "<Thema>"
	escapedBody := renderTemplate(t, app.views, "pages/runtime", escaped, thema.WithLocale("en"))
	if !strings.Contains(escapedBody, "Welcome to the &lt;Thema&gt; runtime tutorial.") {
		t.Fatalf("translation interpolation was not escaped: %s", escapedBody)
	}

	repository := testTheme(t, `{{t "missing.key"}}`)
	views := thema.Must(thema.New(repository, "basic"))
	if got := renderTemplate(t, views, "pages/home", nil); got != "missing.key" {
		t.Fatalf("missing translation = %q, want key", got)
	}
}

func TestContributions(t *testing.T) {
	app, err := newApplication()
	if err != nil {
		t.Fatal(err)
	}
	data := newPage("Runtime Features", "/runtime")
	before := renderTemplate(t, app.views, "pages/runtime", data)
	if !strings.Contains(before, "Thema runtime note") {
		t.Fatal("Contribution did not receive Slot page data")
	}
	if err := app.views.Contribute("page.notice", thema.Contribution{
		ID: "tutorial-note", Template: "contributions/tutorial-note",
	}); !errors.Is(err, thema.ErrDuplicateContribution) {
		t.Fatalf("duplicate Contribution error = %v", err)
	}
	if err := app.views.Contribute("page.notice", thema.Contribution{
		ID: "missing", Template: "contributions/missing",
	}); !errors.Is(err, thema.ErrTemplateNotFound) {
		t.Fatalf("missing Contribution template error = %v", err)
	}
	if !app.views.RemoveContribution("page.notice", "tutorial-note") {
		t.Fatal("RemoveContribution returned false")
	}
	after := renderTemplate(t, app.views, "pages/runtime", data)
	if strings.Contains(after, "Thema runtime note") {
		t.Fatal("removed Contribution still rendered")
	}
	if app.views.RemoveContribution("page.notice", "tutorial-note") {
		t.Fatal("second RemoveContribution returned true")
	}
}

func TestRefreshActivatesValidThemeAndRollsBackInvalidCandidate(t *testing.T) {
	repository := testTheme(t, "old")
	views := thema.Must(thema.New(repository, "basic"))
	if changed, err := views.Refresh(context.Background()); err != nil || changed {
		t.Fatalf("unchanged Refresh = %v, %v", changed, err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if changed, err := views.Refresh(cancelled); changed || !errors.Is(err, thema.ErrRefresh) || !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled Refresh = %v, %v", changed, err)
	}

	page := filepath.Join(repository, "basic", "templates", "pages", "home.html")
	writeFile(t, page, "new")
	if changed, err := views.Refresh(context.Background()); err != nil || !changed {
		t.Fatalf("valid Refresh = %v, %v", changed, err)
	}
	if got := renderTemplate(t, views, "pages/home", nil); got != "new" {
		t.Fatalf("refreshed output = %q, want new", got)
	}

	writeFile(t, page, "{{")
	changed, err := views.Refresh(context.Background())
	if changed || !errors.Is(err, thema.ErrRefresh) || !errors.Is(err, thema.ErrInvalidTheme) {
		t.Fatalf("invalid Refresh = %v, %v", changed, err)
	}
	if got := renderTemplate(t, views, "pages/home", nil); got != "new" {
		t.Fatalf("failed Refresh replaced valid Snapshot: %q", got)
	}
}

func TestFuncsAddsTrustedHelperAfterInitialization(t *testing.T) {
	app, err := newApplication()
	if err != nil {
		t.Fatal(err)
	}
	if err := app.views.Funcs(template.FuncMap{
		"join": func([]string, string) string { return "updated safely" },
	}); err != nil {
		t.Fatal(err)
	}
	data := newPage("Templates & Data", "/templates")
	data.Examples = examples
	body := renderTemplate(t, app.views, "pages/templates", data)
	if !strings.Contains(body, "Topics: updated safely") {
		t.Fatalf("Funcs helper output missing: %s", body)
	}
	if err := app.views.Funcs(template.FuncMap{"asset": func() string { return "unsafe" }}); !errors.Is(err, thema.ErrInvalidTheme) {
		t.Fatalf("reserved runtime helper error = %v", err)
	}
}

func TestEngineOptionsAndSafeRenderCommit(t *testing.T) {
	t.Run("strict missing keys", func(t *testing.T) {
		repository := testTheme(t, `before{{.Missing}}after`)
		views := thema.Must(thema.New(repository, "basic", thema.WithStrictMissingKeys()))
		output := bytes.NewBufferString("unchanged")
		err := views.Render(context.Background(), output, "pages/home", map[string]string{})
		if !errors.Is(err, thema.ErrRender) {
			t.Fatalf("strict Render error = %v", err)
		}
		if output.String() != "unchanged" {
			t.Fatalf("failed Render changed Writer to %q", output.String())
		}
	})

	t.Run("asset base URL", func(t *testing.T) {
		repository := testTheme(t, `{{asset "css/app.css"}}`)
		writeFile(t, filepath.Join(repository, "basic", "assets", "css", "app.css"), "body{}")
		views := thema.Must(thema.New(repository, "basic", thema.WithAssetBaseURL("/static")))
		got := renderTemplate(t, views, "pages/home", nil)
		if !strings.HasPrefix(got, "/static/basic/css/app.css?v=") {
			t.Fatalf("custom asset URL = %q", got)
		}
	})
}

func TestStableErrorsAndContextCancellation(t *testing.T) {
	app, err := newApplication()
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := app.views.Render(context.Background(), &output, "pages/missing", nil); !errors.Is(err, thema.ErrTemplateNotFound) {
		t.Fatalf("missing template error = %v", err)
	}
	if err := app.views.Render(context.Background(), &output, "../home", nil); !errors.Is(err, thema.ErrInvalidPath) {
		t.Fatalf("invalid path error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := app.views.Render(ctx, &output, "pages/home", nil); !errors.Is(err, thema.ErrRender) || !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled Render error = %v", err)
	}

	if _, err := thema.New(t.TempDir(), "basic"); !errors.Is(err, thema.ErrInvalidTheme) {
		t.Fatalf("missing Theme error = %v", err)
	}
	repository := testTheme(t, "valid")
	writeFile(t, filepath.Join(repository, "basic", "theme.json"), `{"name":"Basic","version":"1.0.0","thema":"0.2"}`)
	if _, err := thema.New(repository, "basic"); !errors.Is(err, thema.ErrIncompatibleTheme) {
		t.Fatalf("incompatible Theme error = %v", err)
	}
}

func renderTemplate(t *testing.T, views *thema.Engine, name string, data any, opts ...thema.RenderOption) string {
	t.Helper()
	var output bytes.Buffer
	if err := views.Render(context.Background(), &output, name, data, opts...); err != nil {
		t.Fatal(err)
	}
	return output.String()
}

func testTheme(t *testing.T, page string) string {
	t.Helper()
	repository := t.TempDir()
	writeFile(t, filepath.Join(repository, "basic", "theme.json"), `{"name":"Basic","version":"1.0.0","thema":"0.1"}`)
	writeFile(t, filepath.Join(repository, "basic", "templates", "pages", "home.html"), page)
	return repository
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func request(t *testing.T, handler http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
	return recorder
}
