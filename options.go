package thema

import (
	"fmt"
	"html/template"
	"strings"
)

const defaultLocale = "en"

type config struct {
	defaultLocale string
	missingKey    string
	assetBaseURL  string
	initialFuncs  template.FuncMap
}

// Option configures an Engine before its first Theme Snapshot is compiled.
type Option func(*config) error

// WithDefaultLocale sets the final locale used by the built-in Translator's
// fallback chain. The default is "en".
func WithDefaultLocale(locale string) Option {
	return func(cfg *config) error {
		if err := validateLocale(locale); err != nil {
			return err
		}
		cfg.defaultLocale = normalizeLocale(locale)
		return nil
	}
}

// WithStrictMissingKeys makes map lookups with missing keys fail during
// execution. The default preserves html/template's missing-key behavior.
func WithStrictMissingKeys() Option {
	return func(cfg *config) error {
		cfg.missingKey = "missingkey=error"
		return nil
	}
}

// WithAssetBaseURL changes the URL prefix used by the built-in asset helper.
// The default is "/assets".
func WithAssetBaseURL(base string) Option {
	return func(cfg *config) error {
		base = strings.TrimSpace(base)
		if base == "" {
			return fmt.Errorf("%w: asset base URL is empty", ErrInvalidPath)
		}
		cfg.assetBaseURL = strings.TrimRight(base, "/")
		return nil
	}
}

// WithFuncs registers application helpers before the initial Snapshot is
// compiled. Use Engine.Funcs to add helpers after initialization.
func WithFuncs(funcs template.FuncMap) Option {
	return func(cfg *config) error {
		validated, err := validatedFuncMap(funcs)
		if err != nil {
			return err
		}
		if cfg.initialFuncs == nil {
			cfg.initialFuncs = make(template.FuncMap)
		}
		for name, fn := range validated {
			cfg.initialFuncs[name] = fn
		}
		return nil
	}
}

type renderConfig struct {
	locale string
}

// RenderOption configures one Render call.
type RenderOption func(*renderConfig) error

// WithLocale selects a locale for one Render call.
func WithLocale(locale string) RenderOption {
	return func(cfg *renderConfig) error {
		if err := validateLocale(locale); err != nil {
			return err
		}
		cfg.locale = normalizeLocale(locale)
		return nil
	}
}
