package thema

import (
	"fmt"
	"path"
	"regexp"
	"strings"
)

var (
	semverPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$`)
	localePattern = regexp.MustCompile(`^[A-Za-z]{2,8}(?:-[A-Za-z0-9]{1,8})*$`)
	identifierPattern = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9._-]*[A-Za-z0-9])?$`)
)

func validateTemplatePath(name string) error {
	if err := validateLogicalPath(name); err != nil {
		return err
	}
	if strings.HasSuffix(name, ".html") {
		return fmt.Errorf("%w: %q must omit .html", ErrInvalidPath, name)
	}
	return nil
}

func validateAssetPath(name string) error {
	return validateLogicalPath(name)
}

func validateLogicalPath(name string) error {
	if name == "" || strings.Contains(name, `\`) || strings.HasPrefix(name, "/") || strings.HasSuffix(name, "/") {
		return fmt.Errorf("%w: %q", ErrInvalidPath, name)
	}
	if path.Clean(name) != name {
		return fmt.Errorf("%w: %q", ErrInvalidPath, name)
	}
	for _, segment := range strings.Split(name, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return fmt.Errorf("%w: %q", ErrInvalidPath, name)
		}
	}
	return nil
}

func validateThemeID(id string) error {
	if err := validateLogicalPath(id); err != nil || strings.Contains(id, "/") {
		return fmt.Errorf("%w: invalid theme ID %q", ErrInvalidTheme, id)
	}
	return nil
}

func validateIdentifier(kind, value string) error {
	if !identifierPattern.MatchString(value) || strings.Contains(value, "..") {
		return fmt.Errorf("%w: invalid %s %q", ErrInvalidPath, kind, value)
	}
	return nil
}

func validateLocale(locale string) error {
	if !localePattern.MatchString(locale) {
		return fmt.Errorf("%w: invalid locale %q", ErrInvalidPath, locale)
	}
	return nil
}

func normalizeLocale(locale string) string {
	return strings.ToLower(locale)
}

func validSemver(version string) bool {
	if !semverPattern.MatchString(version) {
		return false
	}
	withoutBuild := strings.SplitN(version, "+", 2)[0]
	parts := strings.SplitN(withoutBuild, "-", 2)
	if len(parts) == 1 {
		return true
	}
	for _, identifier := range strings.Split(parts[1], ".") {
		numeric := true
		for _, character := range identifier {
			if character < '0' || character > '9' {
				numeric = false
				break
			}
		}
		if numeric && len(identifier) > 1 && identifier[0] == '0' {
			return false
		}
	}
	return true
}
