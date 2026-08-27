package thema

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestContributionsAreDeterministicAndRemovable(t *testing.T) {
	repository := newTestTheme(t, map[string]string{
		"pages/home.html": `{{slot "navigation.main" .}}`,
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
