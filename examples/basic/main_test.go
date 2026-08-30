package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
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
			contains: []string{"Getting Started · Thema", "Start with a theme", `thema.New("./themes", "basic")`, "Getting Started", "Templates", "Data &amp; Logic"},
		},
		{
			name: "templates", path: "/templates", status: http.StatusOK,
			contains: []string{"Templates · Thema", "Standard Go templates", "Template composition", "From Thema page data", `class="example-card"`},
		},
		{
			name: "data and logic", path: "/data", status: http.StatusOK,
			contains: []string{"Data &amp; Logic · Thema", "Typed view data", "Start here", "Next step", "<li>Data</li>", "Topics: Data, ViewModel"},
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

func TestObsoleteRoutesAreNotKept(t *testing.T) {
	app, err := newApplication()
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/about", "/examples", "/products"} {
		if response := request(t, app.routes(), path); response.Code != http.StatusNotFound {
			t.Errorf("GET %s status = %d, want %d", path, response.Code, http.StatusNotFound)
		}
	}
}

func TestDynamicTextUsesHTMLTemplateEscaping(t *testing.T) {
	app, err := newApplication()
	if err != nil {
		t.Fatal(err)
	}
	body := request(t, app.routes(), "/data").Body.String()
	if !strings.Contains(body, "&lt;html/template&gt;") {
		t.Fatalf("dynamic template package name was not escaped: %s", body)
	}
}

func TestDataPageEmptyRange(t *testing.T) {
	app, err := newApplication()
	if err != nil {
		t.Fatal(err)
	}
	data := newPage("Data & Logic", "/data")
	var output bytes.Buffer
	if err := app.views.Render(context.Background(), &output, "pages/data", data); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "No examples available.") {
		t.Fatalf("range else output missing: %s", output.String())
	}
}

func TestAssets(t *testing.T) {
	app, err := newApplication()
	if err != nil {
		t.Fatal(err)
	}
	handler := app.routes()

	direct := request(t, handler, "/assets/css/app.css?v=test-generation")
	if direct.Code != http.StatusOK || !strings.Contains(direct.Body.String(), ".example-card") {
		t.Fatalf("direct asset = %d, %q", direct.Code, direct.Body.String())
	}

	home := request(t, handler, "/")
	match := regexp.MustCompile(`href="([^"]*/assets/basic/css/app\.css\?v=[^"]+)"`).FindStringSubmatch(home.Body.String())
	if len(match) != 2 {
		t.Fatalf("generated asset URL not found: %s", home.Body.String())
	}
	if generated := request(t, handler, match[1]); generated.Code != http.StatusOK {
		t.Fatalf("generated asset status = %d, want %d", generated.Code, http.StatusOK)
	}
}

func request(t *testing.T, handler http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
	return recorder
}
