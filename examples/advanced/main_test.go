package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRoutes(t *testing.T) {
	handler := newHandler()
	for _, test := range []struct {
		name     string
		path     string
		status   int
		contains []string
	}{
		{
			name: "home", path: "/", status: http.StatusOK,
			contains: []string{"Thema Guesthouse", "Garden Room", "Seasonal offer"},
		},
		{
			name: "rooms", path: "/rooms", status: http.StatusOK,
			contains: []string{"Our rooms", "Mountain Room", "A realistic Thema example"},
		},
		{
			name: "Chinese", path: "/?lang=zh", status: http.StatusOK,
			contains: []string{"欢迎入住", "季节优惠"},
		},
		{name: "unknown", path: "/missing", status: http.StatusNotFound},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, test.path, nil)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)

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

func TestCSSAsset(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/assets/advanced/css/app.css", nil)
	response := httptest.NewRecorder()

	newHandler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if !strings.Contains(response.Body.String(), ".room-card") {
		t.Fatalf("unexpected CSS body: %s", response.Body.String())
	}
}
