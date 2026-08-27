package thema

import (
	"context"
	"errors"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSlotIsValidInHTMLContentContext(t *testing.T) {
	repository := newTestTheme(t, map[string]string{
		"pages/home.html":            `<nav>{{template "components/navigation" .}}</nav>`,
		"components/navigation.html": `{{slot "navigation.main" .}}`,
		"contributions/nav.html":     `<a href="/rooms">Rooms</a>`,
	}, nil, nil)
	views, err := New(repository, "default")
	if err != nil {
		t.Fatal(err)
	}
	if err := views.Contribute("navigation.main", Contribution{ID: "nav", Template: "contributions/nav"}); err != nil {
		t.Fatal(err)
	}
	assertRender(t, views, "pages/home", nil, `<nav><a href="/rooms">Rooms</a></nav>`)
}

func TestSlotValidationDoesNotExecuteApplicationHelpers(t *testing.T) {
	called := false
	repository := newTestTheme(t, map[string]string{
		"pages/home.html": `{{sideEffect}}<nav>{{slot "navigation.main" .}}</nav>`,
	}, nil, nil)
	_, err := New(repository, "default", WithFuncs(template.FuncMap{
		"sideEffect": func() string {
			called = true
			return "called"
		},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("Slot candidate validation executed an application helper")
	}
}

func TestSlotRejectsNonHTMLContentContexts(t *testing.T) {
	for _, test := range []struct {
		name     string
		template string
	}{
		{name: "attribute", template: `<a title="{{slot "x" .}}">x</a>`},
		{name: "URL", template: `<a href="{{slot "x" .}}">x</a>`},
		{name: "JavaScript", template: `<script>const x = {{slot "x" .}};</script>`},
		{name: "CSS", template: `<style>.x { content: {{slot "x" .}}; }</style>`},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository := newTestTheme(t, map[string]string{"pages/home.html": test.template}, nil, nil)
			_, err := New(repository, "default")
			if !errors.Is(err, ErrInvalidTheme) || !strings.Contains(err.Error(), "only valid as a direct HTML content action") {
				t.Fatalf("New() error = %v", err)
			}
		})
	}
}

func TestSlotRejectsTemplateComposedIntoNonHTMLContentContext(t *testing.T) {
	repository := newTestTheme(t, map[string]string{
		"pages/home.html":      `<a title="{{template "components/slot" .}}">x</a>`,
		"components/slot.html": `{{slot "x" .}}`,
	}, nil, nil)
	_, err := New(repository, "default")
	if !errors.Is(err, ErrInvalidTheme) || !strings.Contains(err.Error(), "used outside HTML content context") {
		t.Fatalf("New() error = %v", err)
	}
}

func TestRefreshRejectsUnsafeSlotCandidateAndRetainsSnapshot(t *testing.T) {
	repository := newTestTheme(t, map[string]string{
		"pages/home.html": `<nav>{{slot "navigation.main" .}}</nav>`,
	}, nil, nil)
	views, err := New(repository, "default")
	if err != nil {
		t.Fatal(err)
	}
	page := filepath.Join(repository, "default", "templates", "pages", "home.html")
	writeTestFile(t, page, []byte(`<a title="{{slot "navigation.main" .}}">x</a>`))
	changed, err := views.Refresh(context.Background())
	if changed || !errors.Is(err, ErrRefresh) || !errors.Is(err, ErrInvalidTheme) {
		t.Fatalf("Refresh() = %v, %v", changed, err)
	}
	assertRender(t, views, "pages/home", nil, `<nav></nav>`)
}

func TestContributionsAreDeterministicAndRemovable(t *testing.T) {
	repository := newTestTheme(t, map[string]string{
		"pages/home.html":      `{{slot "navigation.main" .}}`,
		"contributions/a.html": `A{{.}}`,
		"contributions/b.html": `B{{.}}`,
		"contributions/c.html": `C{{.}}`,
	}, nil, nil)
	views, err := New(repository, "default")
	if err != nil {
		t.Fatal(err)
	}
	if err := views.Contribute("navigation.main", Contribution{ID: "b", Template: "contributions/b", Order: 10}); err != nil {
		t.Fatal(err)
	}
	if err := views.Contribute("navigation.main", Contribution{ID: "a", Template: "contributions/a", Order: 0}); err != nil {
		t.Fatal(err)
	}
	if err := views.Contribute("navigation.main", Contribution{ID: "c", Template: "contributions/c", Order: 10}); err != nil {
		t.Fatal(err)
	}
	assertRender(t, views, "pages/home", "!", "A!B!C!")
	if err := views.Contribute("navigation.main", Contribution{ID: "a", Template: "contributions/a"}); !errors.Is(err, ErrDuplicateContribution) {
		t.Fatalf("expected ErrDuplicateContribution, got %v", err)
	}
	if err := views.Contribute("navigation.main", Contribution{ID: "missing", Template: "contributions/missing"}); !errors.Is(err, ErrTemplateNotFound) {
		t.Fatalf("expected ErrTemplateNotFound, got %v", err)
	}
	if !views.RemoveContribution("navigation.main", "b") {
		t.Fatal("RemoveContribution returned false")
	}
	if views.RemoveContribution("navigation.main", "b") {
		t.Fatal("second RemoveContribution returned true")
	}
	assertRender(t, views, "pages/home", "!", "A!C!")
}

func TestRefreshRejectsRemovalOfContributedTemplate(t *testing.T) {
	repository := newTestTheme(t, map[string]string{
		"pages/home.html":        `{{slot "navigation.main" .}}`,
		"contributions/nav.html": `nav`,
	}, nil, nil)
	views, err := New(repository, "default")
	if err != nil {
		t.Fatal(err)
	}
	if err := views.Contribute("navigation.main", Contribution{ID: "nav", Template: "contributions/nav"}); err != nil {
		t.Fatal(err)
	}
	filename := filepath.Join(repository, "default", "templates", "contributions", "nav.html")
	if err := os.Remove(filename); err != nil {
		t.Fatal(err)
	}
	changed, err := views.Refresh(context.Background())
	if changed || !errors.Is(err, ErrRefresh) || !errors.Is(err, ErrTemplateNotFound) {
		t.Fatalf("Refresh() = %v, %v", changed, err)
	}
	assertRender(t, views, "pages/home", nil, "nav")
}
