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
		breadcrumb bool
	}{
		{name: "home", path: "/", title: "Home"},
		{name: "rooms", path: "/rooms", title: "Rooms", activeHref: "/rooms", breadcrumb: true},
		{name: "room detail", path: "/rooms/example-room", title: "Room", activeHref: "/rooms", breadcrumb: true},
		{name: "amenities", path: "/amenities", title: "Amenities", activeHref: "/amenities", breadcrumb: true},
		{name: "local area", path: "/local-area", title: "Local Area", activeHref: "/local-area", breadcrumb: true},
		{name: "getting here", path: "/getting-here", title: "Getting Here", activeHref: "/getting-here", breadcrumb: true},
		{name: "my stays", path: "/my-stays", title: "My Stays", breadcrumb: true},
		{name: "stay information", path: "/stay-information", title: "Stay Information", breadcrumb: true},
		{name: "cancellation policy", path: "/cancellation-policy", title: "Cancellation Policy", breadcrumb: true},
		{name: "privacy", path: "/privacy", title: "Privacy", breadcrumb: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := advancedRequest(t, handler, tt.path)
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
			}
			body := response.Body.String()
			for _, want := range []string{
				`<a class="skip-link" href="#main">Skip to main content</a>`,
				`<main id="main" class="site-main" tabindex="-1">`,
				`<a class="brand" href="/"`,
				`>KHotel</a>`,
				`<div class="utility-bar">`,
				`<nav class="primary-nav"`,
				`<footer class="site-footer">`,
				`<button class="feedback-control"`,
				`<button class="webchat-launcher"`,
				`<h1>` + tt.title + `</h1>`,
			} {
				if !strings.Contains(body, want) {
					t.Errorf("body does not contain %q", want)
				}
			}

			primary := primaryNavigation(t, body)
			if tt.activeHref == "" {
				if strings.Contains(primary, `aria-current="page"`) {
					t.Error("page unexpectedly has an active primary item")
				}
			} else if !strings.Contains(primary, `href="`+tt.activeHref+`" aria-current="page"`) {
				t.Errorf("active navigation missing for %s", tt.activeHref)
			}

			hasBreadcrumb := strings.Contains(body, `<nav class="breadcrumb"`)
			if hasBreadcrumb != tt.breadcrumb {
				t.Errorf("breadcrumb present = %t, want %t", hasBreadcrumb, tt.breadcrumb)
			}
		})
	}
}

func TestHeaderNavigationAndLanguageMenu(t *testing.T) {
	app, err := newApplication()
	if err != nil {
		t.Fatal(err)
	}
	body := advancedRequest(t, app.routes(), "/rooms").Body.String()

	for _, want := range []string{
		`<button class="utility-chat" type="button" popovertarget="webchat-panel">`,
		`Chat with us`,
		`href="/my-stays">My Stays</a>`,
		`<details class="language-menu">`,
		`<nav class="language-options" aria-label="Language selection">`,
		`<a class="booking-action" href="/booking">Book Now</a>`,
		`<details class="mobile-menu">`,
		`<summary>Menu</summary>`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("header does not contain %q", want)
		}
	}
	for _, unwanted := range []string{"Sign In", "Register", "WhatsApp", "tel:"} {
		if strings.Contains(headerHTML(t, body), unwanted) {
			t.Errorf("header unexpectedly contains %q", unwanted)
		}
	}

	primary := primaryNavigation(t, body)
	if got, want := hrefs(primary), []string{"/rooms", "/amenities", "/local-area", "/getting-here"}; !slices.Equal(got, want) {
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
	for _, href := range []string{"/rooms", "/amenities", "/local-area", "/getting-here", "/my-stays", "/booking"} {
		if !strings.Contains(mobile, `href="`+href+`"`) {
			t.Errorf("mobile navigation missing %s", href)
		}
	}

	languageMenu := elementByClass(t, body, "nav", "language-options")
	if got, want := hrefs(languageMenu), []string{"/rooms?lang=en", "/rooms?lang=ja", "/rooms?lang=zh-Hant", "/rooms?lang=zh-Hans"}; !slices.Equal(got, want) {
		t.Fatalf("language hrefs = %v, want %v", got, want)
	}
	for _, label := range []string{"English", "日本語", "繁體中文", "简体中文"} {
		if !strings.Contains(languageMenu, label) {
			t.Errorf("language menu missing %q", label)
		}
	}
}

func TestBreadcrumbs(t *testing.T) {
	app, err := newApplication()
	if err != nil {
		t.Fatal(err)
	}
	handler := app.routes()

	home := advancedRequest(t, handler, "/").Body.String()
	if strings.Contains(home, `<nav class="breadcrumb"`) {
		t.Fatal("home must not render a breadcrumb")
	}

	rooms := advancedRequest(t, handler, "/rooms").Body.String()
	for _, want := range []string{
		`<nav class="breadcrumb" aria-label="Breadcrumb">`,
		`href="/">`,
		`class="breadcrumb-home-icon"`,
		`<span>Home</span>`,
		`<span aria-current="page">Rooms</span>`,
	} {
		if !strings.Contains(rooms, want) {
			t.Errorf("rooms breadcrumb does not contain %q", want)
		}
	}
	mainIndex := strings.Index(rooms, `<main id="main"`)
	breadcrumbIndex := strings.Index(rooms, `<nav class="breadcrumb"`)
	footerIndex := strings.Index(rooms, `<footer class="site-footer"`)
	if mainIndex < 0 || breadcrumbIndex < 0 || footerIndex < 0 || !(mainIndex < breadcrumbIndex && breadcrumbIndex < footerIndex) {
		t.Fatalf("shell order must be main, breadcrumb, footer; indexes = %d, %d, %d", mainIndex, breadcrumbIndex, footerIndex)
	}

	room := advancedRequest(t, handler, "/rooms/example-room").Body.String()
	for _, want := range []string{`href="/">`, `<span>Home</span>`, `href="/rooms">`, `<span>Rooms</span>`, `<span aria-current="page">Room</span>`} {
		if !strings.Contains(room, want) {
			t.Errorf("room breadcrumb does not contain %q", want)
		}
	}
}

func TestFooterAndWebChat(t *testing.T) {
	app, err := newApplication()
	if err != nil {
		t.Fatal(err)
	}
	body := advancedRequest(t, app.routes(), "/").Body.String()
	footer := footerHTML(t, body)

	if got := strings.Count(footer, `class="footer-group"`); got != 3 {
		t.Fatalf("footer groups = %d, want 3", got)
	}
	for _, want := range []string{
		`class="footer-brand"`,
		`>Social Media</h2>`,
		`>Instagram</a>`,
		`>Facebook</a>`,
		`>LINE</a>`,
		`<h2>KHotel</h2>`,
		`>Stay Information</h2>`,
		`href="/stay-information">Guest Information</a>`,
		`href="/cancellation-policy">Cancellation Policy</a>`,
		`href="/privacy">Privacy</a>`,
		`Some content on this website may be machine translated.`,
		`© 2026 KHotel. All rights reserved.`,
	} {
		if !strings.Contains(footer, want) {
			t.Errorf("footer does not contain %q", want)
		}
	}
	if strings.Contains(footer, `language-menu`) || strings.Contains(footer, `?lang=`) {
		t.Fatal("footer must not contain a language selector")
	}
	translationIndex := strings.Index(footer, `class="translation-bar"`)
	copyrightIndex := strings.Index(footer, `class="footer-bottom"`)
	if translationIndex < 0 || copyrightIndex < 0 || translationIndex >= copyrightIndex {
		t.Fatalf("translation notice must render before copyright bar; indexes = %d, %d", translationIndex, copyrightIndex)
	}

	for _, want := range []string{
		`popovertarget="webchat-panel"`,
		`class="webchat-launcher"`,
		`id="webchat-panel" popover`,
		`aria-label="KHotel WebChat"`,
		`How can we help?`,
		`<button class="feedback-control" type="button" aria-label="Feedback">`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("shared WebChat does not contain %q", want)
		}
	}
	if got := strings.Count(body, `popovertarget="webchat-panel"`); got != 3 {
		t.Fatalf("WebChat target references = %d, want utility, launcher, and close", got)
	}
}

func TestSharedUIUsesAllSupportedLocales(t *testing.T) {
	app, err := newApplication()
	if err != nil {
		t.Fatal(err)
	}
	handler := app.routes()

	tests := []struct {
		locale      string
		chat        string
		myStays     string
		rooms       string
		book        string
		guestInfo   string
		machineNote string
		greeting    string
		feedback    string
	}{
		{locale: "en", chat: "Chat with us", myStays: "My Stays", rooms: "Rooms", book: "Book Now", guestInfo: "Guest Information", machineNote: "Some content on this website may be machine translated.", greeting: "How can we help?", feedback: "Feedback"},
		{locale: "ja", chat: "チャットで相談", myStays: "予約確認", rooms: "客室", book: "予約する", guestInfo: "ご利用案内", machineNote: "このウェブサイトの一部のコンテンツは機械翻訳されている場合があります。", greeting: "どのようなご用件でしょうか？", feedback: "フィードバック"},
		{locale: "zh-Hant", chat: "與我們聊天", myStays: "我的預訂", rooms: "客房", book: "立即預訂", guestInfo: "旅客須知", machineNote: "此網站的部分內容可能由機器翻譯。", greeting: "我們可以如何協助您？", feedback: "意見回饋"},
		{locale: "zh-Hans", chat: "与我们聊天", myStays: "我的预订", rooms: "客房", book: "立即预订", guestInfo: "住客须知", machineNote: "本网站的部分内容可能由机器翻译。", greeting: "我们可以为您做什么？", feedback: "意见反馈"},
	}

	for _, tt := range tests {
		t.Run(tt.locale, func(t *testing.T) {
			body := advancedRequest(t, handler, "/rooms?lang="+tt.locale).Body.String()
			for _, want := range []string{
				`<html lang="` + tt.locale + `">`,
				tt.chat,
				tt.myStays,
				`<h1>` + tt.rooms + `</h1>`,
				tt.book,
				tt.guestInfo,
				tt.machineNote,
				tt.greeting,
				tt.feedback,
				`lang="` + tt.locale + `" aria-current="true"`,
			} {
				if !strings.Contains(body, want) {
					t.Errorf("localized shell does not contain %q", want)
				}
			}
			if tt.locale != "en" {
				for _, href := range []string{"/amenities?lang=" + tt.locale, "/my-stays?lang=" + tt.locale, "/?lang=" + tt.locale} {
					if !strings.Contains(body, `href="`+href+`"`) {
						t.Errorf("localized navigation missing %s", href)
					}
				}
			}
		})
	}
}

func TestHealthNotFoundAndInvalidLocale(t *testing.T) {
	app, err := newApplication()
	if err != nil {
		t.Fatal(err)
	}
	handler := app.routes()

	health := advancedRequest(t, handler, "/health")
	if health.Code != http.StatusOK || health.Body.String() != "ok\n" {
		t.Fatalf("health = %d, %q", health.Code, health.Body.String())
	}

	missing := advancedRequest(t, handler, "/does-not-exist?lang=zh-Hant")
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing status = %d, want %d", missing.Code, http.StatusNotFound)
	}
	for _, want := range []string{"找不到頁面", `<main id="main"`, `<footer class="site-footer">`} {
		if !strings.Contains(missing.Body.String(), want) {
			t.Errorf("localized themed 404 does not contain %q", want)
		}
	}

	invalid := advancedRequest(t, handler, "/?lang=unsupported").Body.String()
	if !strings.Contains(invalid, `<html lang="en">`) || !strings.Contains(invalid, `<h1>Home</h1>`) {
		t.Fatal("unsupported locale must fall back to English")
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
	for _, selector := range []string{
		".skip-link:focus",
		".utility-bar",
		".language-menu",
		".primary-header",
		".mobile-menu",
		".breadcrumb",
		".feedback-control",
		".webchat-launcher",
		".site-footer",
	} {
		if !strings.Contains(asset.Body.String(), selector) {
			t.Errorf("CSS does not contain %s", selector)
		}
	}
	for _, declaration := range []string{
		"max-width: var(--container-width);",
		"margin-inline: auto;",
		"padding-inline: var(--gutter);",
	} {
		if !strings.Contains(asset.Body.String(), declaration) {
			t.Errorf("shared container CSS does not contain %q", declaration)
		}
	}
}

func hrefs(fragment string) []string {
	matches := regexp.MustCompile(`href="([^"]+)"`).FindAllStringSubmatch(fragment, -1)
	result := make([]string, 0, len(matches))
	for _, match := range matches {
		result = append(result, match[1])
	}
	return result
}

func headerHTML(t *testing.T, body string) string {
	t.Helper()
	match := regexp.MustCompile(`(?s)<header class="site-header">(.*?)</header>`).FindStringSubmatch(body)
	if len(match) != 2 {
		t.Fatalf("site header not found: %s", body)
	}
	return match[1]
}

func primaryNavigation(t *testing.T, body string) string {
	t.Helper()
	return elementByClass(t, body, "nav", "primary-nav")
}

func mobileNavigation(t *testing.T, body string) string {
	t.Helper()
	return elementByClass(t, body, "nav", "mobile-panel")
}

func footerHTML(t *testing.T, body string) string {
	t.Helper()
	match := regexp.MustCompile(`(?s)<footer class="site-footer">(.*?)</footer>`).FindStringSubmatch(body)
	if len(match) != 2 {
		t.Fatalf("site footer not found: %s", body)
	}
	return match[1]
}

func elementByClass(t *testing.T, body, element, class string) string {
	t.Helper()
	pattern := `(?s)<` + regexp.QuoteMeta(element) + ` class="` + regexp.QuoteMeta(class) + `"[^>]*>(.*?)</` + regexp.QuoteMeta(element) + `>`
	match := regexp.MustCompile(pattern).FindStringSubmatch(body)
	if len(match) != 2 {
		t.Fatalf("%s.%s not found: %s", element, class, body)
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
