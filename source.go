package thema

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	maxTemplateFiles = 1_000
	maxFileSize      = 8 << 20
	maxThemeSize     = 128 << 20
)

type templateSource struct {
	name   string
	source string
	text   string
}

type themePackage struct {
	id           string
	manifest     manifest
	templates    []templateSource
	translations map[string]map[string]string
	assets       map[string]struct{}
	generation   string
}

type sourceResource struct {
	kind    string
	logical string
	data    []byte
}

func loadTheme(ctx context.Context, repository, themeID string) (*themePackage, error) {
	if err := validateThemeID(themeID); err != nil {
		return nil, err
	}

	themeRoot := filepath.Join(repository, themeID)
	info, err := os.Lstat(themeRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, invalidTheme(themeID, "theme directory does not exist")
		}
		return nil, invalidTheme(themeID, "cannot inspect theme directory: %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return nil, invalidTheme(themeID, "theme root must be a real directory, not a symlink")
	}

	var resources []sourceResource
	var totalSize int64
	err = filepath.WalkDir(themeRoot, func(filename string, entry fs.DirEntry, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if walkErr != nil {
			return invalidTheme(themeID, "cannot inspect Theme resource: %v", walkErr)
		}
		if filename == themeRoot {
			return nil
		}
		rel, err := filepath.Rel(themeRoot, filename)
		if err != nil {
			return invalidTheme(themeID, "cannot resolve Theme resource")
		}
		logicalFS := filepath.ToSlash(rel)
		if entry.Type()&os.ModeSymlink != 0 {
			return invalidTheme(themeID, "symlinks are not permitted: %s", logicalFS)
		}
		if entry.IsDir() {
			return nil
		}
		entryInfo, err := entry.Info()
		if err != nil {
			return invalidTheme(themeID, "cannot inspect %s", logicalFS)
		}
		if !entryInfo.Mode().IsRegular() {
			return invalidTheme(themeID, "non-regular files are not permitted: %s", logicalFS)
		}

		kind, logical, recognized, err := classifyResource(logicalFS)
		if err != nil {
			return invalidTheme(themeID, "%v", err)
		}
		if !recognized {
			return nil
		}
		if entryInfo.Size() > maxFileSize {
			return invalidTheme(themeID, "%s exceeds the %d-byte file limit", logicalFS, maxFileSize)
		}
		totalSize += entryInfo.Size()
		if totalSize > maxThemeSize {
			return invalidTheme(themeID, "Theme exceeds the %d-byte resource limit", maxThemeSize)
		}
		data, err := os.ReadFile(filename)
		if err != nil {
			return invalidTheme(themeID, "cannot read %s", logicalFS)
		}
		if len(data) > maxFileSize {
			return invalidTheme(themeID, "%s exceeds the %d-byte file limit", logicalFS, maxFileSize)
		}
		totalSize += int64(len(data)) - entryInfo.Size()
		if totalSize > maxThemeSize {
			return invalidTheme(themeID, "Theme exceeds the %d-byte resource limit", maxThemeSize)
		}
		resources = append(resources, sourceResource{kind: kind, logical: logical, data: data})
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(resources, func(i, j int) bool {
		if resources[i].kind == resources[j].kind {
			return resources[i].logical < resources[j].logical
		}
		return resources[i].kind < resources[j].kind
	})

	pkg := &themePackage{
		id:           themeID,
		translations: make(map[string]map[string]string),
		assets:       make(map[string]struct{}),
	}
	var manifestFound bool
	for _, resource := range resources {
		switch resource.kind {
		case "manifest":
			manifestFound = true
			if err := decodeManifest(resource.data, &pkg.manifest); err != nil {
				return nil, invalidTheme(themeID, "invalid theme.json: %v", err)
			}
		case "template":
			if len(pkg.templates) >= maxTemplateFiles {
				return nil, invalidTheme(themeID, "Theme exceeds the %d-template limit", maxTemplateFiles)
			}
			if err := validateTemplatePath(resource.logical); err != nil {
				return nil, invalidTheme(themeID, "template %q: %v", resource.logical, err)
			}
			pkg.templates = append(pkg.templates, templateSource{
				name:   resource.logical,
				source: "templates/" + resource.logical + ".html",
				text:   string(resource.data),
			})
		case "locale":
			locale := normalizeLocale(resource.logical)
			if _, exists := pkg.translations[locale]; exists {
				return nil, invalidTheme(themeID, "duplicate locale %q", resource.logical)
			}
			messages, err := decodeLocale(resource.data)
			if err != nil {
				return nil, invalidTheme(themeID, "locale %q: %v", resource.logical, err)
			}
			pkg.translations[locale] = messages
		case "asset":
			if err := validateAssetPath(resource.logical); err != nil {
				return nil, invalidTheme(themeID, "asset %q: %v", resource.logical, err)
			}
			if _, exists := pkg.assets[resource.logical]; exists {
				return nil, invalidTheme(themeID, "duplicate asset %q", resource.logical)
			}
			pkg.assets[resource.logical] = struct{}{}
		}
	}

	if !manifestFound {
		return nil, invalidTheme(themeID, "theme.json is required")
	}
	if err := pkg.manifest.validate(themeID); err != nil {
		return nil, err
	}
	if len(pkg.templates) == 0 {
		return nil, invalidTheme(themeID, "at least one .html template is required")
	}
	pkg.generation = fingerprint(themeID, resources)
	return pkg, nil
}

func classifyResource(rel string) (kind, logical string, recognized bool, err error) {
	if strings.Contains(rel, `\`) {
		return "", "", false, fmt.Errorf("unsafe resource path %q", rel)
	}
	if rel == "theme.json" {
		return "manifest", rel, true, nil
	}
	if strings.HasPrefix(rel, "templates/") && strings.HasSuffix(rel, ".html") {
		logical = strings.TrimSuffix(strings.TrimPrefix(rel, "templates/"), ".html")
		return "template", logical, true, nil
	}
	if strings.HasPrefix(rel, "locales/") && strings.HasSuffix(rel, ".json") {
		logical = strings.TrimSuffix(strings.TrimPrefix(rel, "locales/"), ".json")
		if strings.Contains(logical, "/") {
			return "", "", false, fmt.Errorf("locale files must be direct children of locales/: %q", rel)
		}
		if err := validateLocale(logical); err != nil {
			return "", "", false, err
		}
		return "locale", logical, true, nil
	}
	if strings.HasPrefix(rel, "assets/") {
		logical = strings.TrimPrefix(rel, "assets/")
		return "asset", logical, true, nil
	}
	return "", "", false, nil
}

func decodeManifest(data []byte, dst *manifest) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}

func fingerprint(themeID string, resources []sourceResource) string {
	hash := sha256.New()
	writeFingerprintPart(hash, []byte(themeID))
	for _, resource := range resources {
		writeFingerprintPart(hash, []byte(resource.kind))
		writeFingerprintPart(hash, []byte(resource.logical))
		writeFingerprintPart(hash, resource.data)
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func writeFingerprintPart(w io.Writer, value []byte) {
	var size [8]byte
	binary.BigEndian.PutUint64(size[:], uint64(len(value)))
	_, _ = w.Write(size[:])
	_, _ = w.Write(value)
}
