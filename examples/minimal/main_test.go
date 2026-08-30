package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHome(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()

	newHandler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if !strings.Contains(response.Body.String(), "Hello, Thema") {
		t.Fatalf("body does not contain native root data: %s", response.Body.String())
	}
}

func TestCSSAsset(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/assets/minimal/css/app.css", nil)
	response := httptest.NewRecorder()

	newHandler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if !strings.Contains(response.Body.String(), "font-family") {
		t.Fatalf("unexpected CSS body: %s", response.Body.String())
	}
}
