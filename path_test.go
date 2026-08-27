package thema

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLogicalTemplatePaths(t *testing.T) {
	valid := []string{"pages/home", "components/header", "reservation/detail", "layouts/app"}
	for _, name := range valid {
		if err := validateTemplatePath(name); err != nil {
			t.Errorf("validateTemplatePath(%q) = %v", name, err)
		}
	}
	invalid := []string{"", "/pages/home", "pages/home/", "../home", "pages/../home", "pages//home", "pages/./home", `pages\home`, "pages/home.html"}
	for _, name := range invalid {
		if err := validateTemplatePath(name); !errors.Is(err, ErrInvalidPath) {
			t.Errorf("validateTemplatePath(%q) = %v", name, err)
		}
	}
}

func TestManifestValidation(t *testing.T) {
	for _, test := range []struct {
		name     string
		manifest string
		want     error
	}{
		{name: "missing field", manifest: `{"name":"Default","version":"1.0.0"}`, want: ErrInvalidTheme},
		{name: "bad semver", manifest: `{"name":"Default","version":"1.0","thema":"0.1"}`, want: ErrInvalidTheme},
		{name: "bad contract", manifest: `{"name":"Default","version":"1.0.0","thema":"0.2"}`, want: ErrIncompatibleTheme},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository := newTestTheme(t, map[string]string{"pages/home.html": "home"}, nil, nil)
			writeTestFile(t, filepath.Join(repository, "default", "theme.json"), []byte(test.manifest))
			_, err := New(repository, "default")
			if !errors.Is(err, test.want) {
				t.Fatalf("New() error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestThemeSymlinkIsRejected(t *testing.T) {
	repository := newTestTheme(t, map[string]string{"pages/home.html": "home"}, nil, nil)
	target := filepath.Join(repository, "outside.html")
	writeTestFile(t, target, []byte("outside"))
	link := filepath.Join(repository, "default", "templates", "pages", "linked.html")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	_, err := New(repository, "default")
	if !errors.Is(err, ErrInvalidTheme) {
		t.Fatalf("expected ErrInvalidTheme, got %v", err)
	}
}
