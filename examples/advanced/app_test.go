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
		{
			name: "home", path: "/", status: http.StatusOK,
			contains: []string{"Build better digital products.", "Product Engineering", "Atlas", `class="stat"`, "Projects shipped", "accepting new projects"},
		},
		{
			name: "services", path: "/services", status: http.StatusOK,
			contains: []string{"What we do", "service-card", "Cloud Infrastructure"},
		},
		{
			name: "work", path: "/work", status: http.StatusOK,
			contains: []string{"Selected work", "project-card", "99.99% availability", "40% faster workflows"},
		},
		{
			name: "about", path: "/about", status: http.StatusOK,
			contains: []string{"Who we are", "Stay simple", "Build for the long term"},
		},
		{
			name: "contact", path: "/contact", status: http.StatusOK,
			contains: []string{"Start a conversation", "hello@northstar.example"},
		},
		{
			name: "Chinese", path: "/?lang=zh", status: http.StatusOK,
			contains: []string{"构建更出色的数字产品", "本季度我们正在接受新项目"},
		},
		{
			name: "not found", path: "/missing", status: http.StatusNotFound,
			contains: []string{"Page not found", "error-page"},
		},
		{name: "health", path: "/health", status: http.StatusOK, contains: []string{"ok"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			response := advancedRequest(t, handler, test.path)
			if response.Code != test.status {
				t.Fatalf("status = %d, want %d", response.Code, test.status)
			}
			if response.Header().Get("X-Content-Type-Options") != "nosniff" {
				t.Fatal("missing X-Content-Type-Options header")
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
	for _, path := range []string{"/assets/css/app.css", "/assets/images/project-atlas.svg"} {
		response := advancedRequest(t, handler, path)
		if response.Code != http.StatusOK {
			t.Fatalf("GET %s = %d, want %d", path, response.Code, http.StatusOK)
		}
		if response.Header().Get("Cache-Control") == "" {
			t.Fatalf("GET %s has no Cache-Control header", path)
		}
	}

	home := advancedRequest(t, handler, "/")
	match := regexp.MustCompile(`href="([^"]*/assets/northstar/css/app\.css\?v=[^"]+)"`).FindStringSubmatch(home.Body.String())
	if len(match) != 2 {
		t.Fatalf("generated asset URL not found: %s", home.Body.String())
	}
	if response := advancedRequest(t, handler, match[1]); response.Code != http.StatusOK {
		t.Fatalf("generated asset status = %d, want %d", response.Code, http.StatusOK)
	}
}

func advancedRequest(t *testing.T, handler http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
	return response
}
