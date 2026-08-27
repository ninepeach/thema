package thema

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"io"
	"net/url"
	"strings"
)

// Render executes a logical template against data as the native root dot (.).
// The destination is written only after template execution succeeds.
func (e *Engine) Render(ctx context.Context, w io.Writer, name string, data any, opts ...RenderOption) error {
	if ctx == nil {
		return fmt.Errorf("%w: nil context", ErrRender)
	}
	if w == nil {
		return fmt.Errorf("%w: nil Writer", ErrRender)
	}
	if err := validateTemplatePath(name); err != nil {
		return err
	}
	renderCfg := renderConfig{locale: e.config.defaultLocale}
	for _, option := range opts {
		if option == nil {
			return fmt.Errorf("%w: nil RenderOption", ErrRender)
		}
		if err := option(&renderCfg); err != nil {
			return err
		}
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("%w: template %q: %w", ErrRender, name, err)
	}

	active := e.current.Load()
	if active.templates.Lookup(name) == nil {
		return fmt.Errorf("%w: %q", ErrTemplateNotFound, name)
	}
	executable, err := active.templates.Clone()
	if err != nil {
		return fmt.Errorf("%w: template %q: %w", ErrRender, name, err)
	}
	// Go 1.22's Clone does not retain the missingkey option. Reapply the
	// immutable Engine setting so strict behavior is consistent across the
	// supported Go versions.
	executable.Option(e.config.missingKey)
	runtimeFuncs := template.FuncMap{
		"t": func(key string, values ...any) (string, error) {
			if err := ctx.Err(); err != nil {
				return "", err
			}
			return active.translator.translate(renderCfg.locale, key, values...)
		},
		"asset": func(assetName string) (string, error) {
			if err := ctx.Err(); err != nil {
				return "", err
			}
			return e.assetURL(active, assetName)
		},
	}
	runtimeFuncs["slot"] = func(slotName string, slotData any) (template.HTML, error) {
		if err := validateIdentifier("slot", slotName); err != nil {
			return "", err
		}
		var rendered bytes.Buffer
		for _, contribution := range orderedContributions(active.contributions[slotName]) {
			writer := contextWriter{ctx: ctx, writer: &rendered}
			if err := executable.ExecuteTemplate(writer, contribution.Template, slotData); err != nil {
				return "", err
			}
		}
		// Each Contribution was already contextually escaped by html/template.
		return template.HTML(rendered.String()), nil
	}
	executable.Funcs(runtimeFuncs)

	var buffer bytes.Buffer
	writer := contextWriter{ctx: ctx, writer: &buffer}
	if err := executable.ExecuteTemplate(writer, name, data); err != nil {
		return fmt.Errorf("%w: template %q: %w", ErrRender, name, err)
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("%w: template %q: %w", ErrRender, name, err)
	}
	if _, err := io.Copy(w, &buffer); err != nil {
		return fmt.Errorf("%w: committing template %q: %w", ErrRender, name, err)
	}
	return nil
}

func (e *Engine) assetURL(active *snapshot, name string) (string, error) {
	if err := validateAssetPath(name); err != nil {
		return "", err
	}
	if _, exists := active.pkg.assets[name]; !exists {
		return "", fmt.Errorf("%w: asset %q", ErrInvalidPath, name)
	}
	parts := strings.Split(name, "/")
	for index := range parts {
		parts[index] = url.PathEscape(parts[index])
	}
	version := active.pkg.version
	if len(version) > 16 {
		version = version[:16]
	}
	return e.config.assetBaseURL + "/" + url.PathEscape(active.pkg.id) + "/" + strings.Join(parts, "/") + "?v=" + url.QueryEscape(version), nil
}

type contextWriter struct {
	ctx    context.Context
	writer io.Writer
}

func (w contextWriter) Write(value []byte) (int, error) {
	if err := w.ctx.Err(); err != nil {
		return 0, err
	}
	return w.writer.Write(value)
}
