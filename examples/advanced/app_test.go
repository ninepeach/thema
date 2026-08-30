package main

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"slices"
	"strings"
	"testing"
)

func TestPageShells(t *testing.T) {
	app, err := newApplication()
	if err != nil {
		t.Fatal(err)
	}
	handler := app.routes()

	tests := []struct {
		name       string
		path       string
		title      string
		activeHref string
	}{
		{name: "home", path: "/", title: "Home"},
		{name: "rooms", path: "/rooms", title: "Rooms", activeHref: "/rooms"},
		{name: "room detail", path: "/rooms/example-room", title: "Room", activeHref: "/rooms"},
		{name: "amenities", path: "/amenities", title: "Amenities", activeHref: "/amenities"},
		{name: "local area", path: "/local-area", title: "Local Area", activeHref: "/local-area"},
		{name: "getting here", path: "/getting-here", title: "Getting Here", activeHref: "/getting-here"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := advancedRequest(t, handler, tt.path)
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
			}
			body := response.Body.String()
			for _, want := range []string{
				`<a class="skip-link" href="#main">`,
				`<main id="main"`,
				`<a class="brand" href="/"`,
				`<div class="utility-bar">`,
				`<nav class="primary-nav"`,
				`<a class="booking-action" href="/booking">Book Now</a>`,
				`<footer class="site-footer">`,
				`<h1>` + tt.title + `</h1>`,
			} {
				if !strings.Contains(body, want) {
					t.Errorf("body does not contain %q", want)
				}
			}
			if tt.activeHref == "" {
				if strings.Contains(primaryNavigation(t, body), `aria-current="page"`) {
					t.Error("home unexpectedly has an active primary item")
				}
			} else if !strings.Contains(body, `href="`+tt.activeHref+`" aria-current="page"`) {
				t.Errorf("active navigation missing for %s", tt.activeHref)
			}
		})
	}
}

func TestNavigationStructure(t *testing.T) {
	app, err := newApplication()
	if err != nil {
		t.Fatal(err)
	}
	body := advancedRequest(t, app.routes(), "/rooms").Body.String()

	primary := primaryNavigation(t, body)
	hrefs := regexp.MustCompile(`href="([^"]+)"`).FindAllStringSubmatch(primary, -1)
	got := make([]string, 0, len(hrefs))
	for _, match := range hrefs {
		got = append(got, match[1])
	}
	want := []string{"/rooms", "/amenities", "/local-area", "/getting-here"}
	if !slices.Equal(got, want) {
		t.Fatalf("primary navigation hrefs = %v, want %v", got, want)
	}
	for _, label := range []string{"Rooms", "Amenities", "Local Area", "Getting Here"} {
		if !strings.Contains(primary, label) {
			t.Errorf("primary navigation missing %q", label)
		}
	}
	if strings.Contains(primary, ">Home<") {
		t.Fatal("primary navigation must not contain Home")
	}
	mobile := mobileNavigation(t, body)
	for _, href := range append(want, "/booking", "/register") {
		if !strings.Contains(mobile, `href="`+href+`"`) {
			t.Errorf("mobile navigation missing %s", href)
		}
	}

	for _, want := range []string{
		`<nav class="utility-nav"`,
		`>English</a>`,
		`href="/sign-in">Sign In</a>`,
		`href="/register">Register</a>`,
		`<details class="mobile-menu">`,
		`<summary>Menu</summary>`,
		`<nav class="mobile-panel"`,
		`<div class="footer-main">`,
		`<div class="footer-bottom">`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("shell does not contain %q", want)
		}
	}
}

func TestHealthAndNotFound(t *testing.T) {
	app, err := newApplication()
	if err != nil {
		t.Fatal(err)
	}
	handler := app.routes()

	health := advancedRequest(t, handler, "/health")
	if health.Code != http.StatusOK || health.Body.String() != "ok\n" {
		t.Fatalf("health = %d, %q", health.Code, health.Body.String())
	}

	missing := advancedRequest(t, handler, "/does-not-exist")
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing status = %d, want %d", missing.Code, http.StatusNotFound)
	}
	for _, want := range []string{"Page not found", `<main id="main"`, `<footer class="site-footer">`} {
		if !strings.Contains(missing.Body.String(), want) {
			t.Errorf("themed 404 does not contain %q", want)
		}
	}
}

func TestGeneratedAssetIsServed(t *testing.T) {
	app, err := newApplication()
	if err != nil {
		t.Fatal(err)
	}
	handler := app.routes()
	home := advancedRequest(t, handler, "/")
	match := regexp.MustCompile(`href="([^\"]*/assets/khotel/css/app\.css\?v=[^\"]+)"`).FindStringSubmatch(home.Body.String())
	if len(match) != 2 {
		t.Fatalf("generated asset URL not found: %s", home.Body.String())
	}
	asset := advancedRequest(t, handler, match[1])
	if asset.Code != http.StatusOK {
		t.Fatalf("generated asset status = %d, want %d", asset.Code, http.StatusOK)
	}
	if asset.Header().Get("Cache-Control") == "" {
		t.Fatal("asset has no Cache-Control header")
	}
	for _, selector := range []string{".skip-link:focus", ".utility-bar", ".primary-header", ".mobile-menu", ".site-footer"} {
		if !strings.Contains(asset.Body.String(), selector) {
			t.Errorf("CSS does not contain %s", selector)
		}
	}
}

func TestExistingPerRenderLocaleFoundationRemainsUsable(t *testing.T) {
	app, err := newApplication()
	if err != nil {
		t.Fatal(err)
	}
	body := advancedRequest(t, app.routes(), "/?lang=zh").Body.String()
	if !strings.Contains(body, "<h1>首页</h1>") || !strings.Contains(body, `<html lang="zh">`) {
		t.Fatalf("Chinese locale foundation did not render: %s", body)
	}
}

func primaryNavigation(t *testing.T, body string) string {
	t.Helper()
	match := regexp.MustCompile(`(?s)<nav class="primary-nav"[^>]*>(.*?)</nav>`).FindStringSubmatch(body)
	if len(match) != 2 {
		t.Fatalf("primary navigation not found: %s", body)
	}
	return match[1]
}

func mobileNavigation(t *testing.T, body string) string {
	t.Helper()
	match := regexp.MustCompile(`(?s)<nav class="mobile-panel"[^>]*>(.*?)</nav>`).FindStringSubmatch(body)
	if len(match) != 2 {
		t.Fatalf("mobile navigation not found: %s", body)
	}
	return match[1]
}

func advancedRequest(t *testing.T, handler http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
	if response.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Errorf("GET %s missing X-Content-Type-Options", path)
	}
	return response
}
