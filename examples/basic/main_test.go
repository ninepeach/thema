package main

import (
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
	for _, test := range []struct {
		name     string
		path     string
		status   int
		contains []string
	}{
		{name: "home", path: "/", status: http.StatusOK, contains: []string{"Acme Studio", "Simple products for modern teams."}},
		{name: "about", path: "/about", status: http.StatusOK, contains: []string{"Who we are", "Keep things simple"}},
		{name: "products", path: "/products", status: http.StatusOK, contains: []string{"Starter", "Team", "$19"}},
		{name: "health", path: "/health", status: http.StatusOK, contains: []string{"ok"}},
		{name: "unknown", path: "/does-not-exist", status: http.StatusNotFound},
	} {
		t.Run(test.name, func(t *testing.T) {
			response := request(t, handler, test.path)
			if response.Code != test.status {
				t.Fatalf("status = %d, want %d", response.Code, test.status)
			}
			for _, value := range test.contains {
				if !strings.Contains(response.Body.String(), value) {
					t.Fatalf("body does not contain %q: %s", value, response.Body.String())
				}
			}
		})
	}
}

func TestAssets(t *testing.T) {
	app, err := newApplication()
	if err != nil {
		t.Fatal(err)
	}
	handler := app.routes()

	direct := request(t, handler, "/assets/css/app.css")
	if direct.Code != http.StatusOK || !strings.Contains(direct.Body.String(), ".product-card") {
		t.Fatalf("direct asset = %d, %q", direct.Code, direct.Body.String())
	}

	home := request(t, handler, "/")
	match := regexp.MustCompile(`href="([^"]*/assets/basic/css/app\.css\?v=[^"]+)"`).FindStringSubmatch(home.Body.String())
	if len(match) != 2 {
		t.Fatalf("generated asset URL not found: %s", home.Body.String())
	}
	generated := request(t, handler, match[1])
	if generated.Code != http.StatusOK {
		t.Fatalf("generated asset status = %d, want %d", generated.Code, http.StatusOK)
	}
}

func request(t *testing.T, handler http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
	return response
}
