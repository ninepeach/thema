# Thema

Thema is a small Go-native HTML template and theme runtime built on
[`html/template`](https://pkg.go.dev/html/template). It adds structured Theme
packages, localization, safe runtime refresh, application helpers, assets, and
extension Slots without replacing Go's template model.

**Thema uses Go `html/template`; it does not define a new template language.**

**Render data remains the native template root dot (`.`).** Thema never requires
`{{ .Data.Title }}` wrappers.

## Installation

```sh
go get github.com/ninepeach/thema
```

Thema v0.1 requires Go 1.22 or later and has no third-party runtime dependencies.

## Quick start

```text
themes/
└── default/
    ├── theme.json
    ├── templates/
    │   ├── components/header.html
    │   └── pages/home.html
    ├── locales/
    │   └── en.json
    └── assets/
        └── css/app.css
```

`theme.json`:

```json
{
  "name": "Default",
  "version": "1.0.0",
  "thema": "0.1"
}
```

`templates/components/header.html`:

```gotemplate
<header>{{ .Title }}</header>
```

`templates/pages/home.html`:

```gotemplate
{{ template "components/header" . }}
<h1>{{ .Title | upper }}</h1>
<link rel="stylesheet" href="{{ asset "css/app.css" }}">
```

Go:

```go
package main

import (
    "context"
    "html/template"
    "os"
    "strings"

    "github.com/ninepeach/thema"
)

func main() {
    views := thema.Must(thema.New(
        "./themes",
        "default",
        thema.WithFuncs(template.FuncMap{"upper": strings.ToUpper}),
    ))

    data := struct{ Title string }{Title: "Welcome"}
    if err := views.Render(
        context.Background(),
        os.Stdout,
        "pages/home",
        data,
    ); err != nil {
        panic(err)
    }
}
```

`New(themeRepository, activeTheme)` creates an independent Engine. Thema has no
global Engine, mutable singleton, `Default`, or `GetInstance`.

## Theme Repository

Each direct child of a Theme Repository is a Theme. Its directory name is the
Theme ID. A Theme is one coherent presentation unit:

```text
themes/
├── default/
│   ├── theme.json
│   ├── templates/
│   ├── locales/
│   └── assets/
└── minimal/
    ├── theme.json
    ├── templates/
    ├── locales/
    └── assets/
```

The manifest fields are all required:

- `name`: human-readable Theme name.
- `version`: valid SemVer Theme release version.
- `thema`: Theme Contract version. Thema v0.1 accepts exactly `0.1`.

`theme.json` is strict: unknown fields, missing required fields, and trailing or
multiple JSON values are rejected.

An incompatible contract prevents activation. Templates, locales, assets, and
the manifest always come from the same active Theme generation.

## Logical template paths

Only `.html` files under `templates/` are templates. The source path
`templates/pages/home.html` maps to the logical name `pages/home`. Render calls
never contain `.html`, `themes/`, `templates/`, an absolute path, or a route:

```go
err := views.Render(ctx, w, "pages/home", data)
```

Logical names are relative slash-separated paths. Leading or trailing slashes,
empty segments, `.`, `..`, backslashes, traversal, and a `.html` suffix are
rejected. `index.html` has no special behavior:
`templates/pages/rooms/index.html` is `pages/rooms/index`.

The logical path automatically names the file's primary template, so a file can
contain plain HTML without `{{ define }}`. Native `{{ define }}` remains
available for additional named templates.

## Native Go variables and composition

All normal Go template behavior is preserved:

```gotemplate
{{ .Title }}
{{ .Guest.Name }}
{{ range .Rooms }}{{ .Name }}{{ end }}
{{ $guest := .Guest }}{{ $guest.Name }}
{{ with .Property }}{{ .Name }}{{ end }}
```

Composition uses Go's native template action:

```gotemplate
{{ template "components/header" . }}
```

Page, Component, Partial, and Layout are naming conventions, not separate
runtimes.

## Helpers

Helpers are ordinary trusted Go functions:

```go
views, err := thema.New(
    "./themes",
    "default",
    thema.WithFuncs(template.FuncMap{
        "money": formatMoney,
        "date":   formatDate,
    }),
)
```

Use `WithFuncs` when the initial Theme already refers to the helper. `Funcs`
transactionally adds or replaces application helpers after initialization and
recompiles the current in-memory Theme generation:

```go
err := views.Funcs(template.FuncMap{"upper": strings.ToUpper})
```

If recompilation fails, the helpers and current Snapshot remain unchanged. The
runtime names `t`, `asset`, and `slot` are reserved. Themes cannot register Go
functions.

## i18n

The built-in Translator loads flat or nested JSON catalogs:

```json
{
  "home": { "title": "Welcome" },
  "guest.hello": "Hello, {{name}}!"
}
```

Templates translate keys and interpolate name/value pairs:

```gotemplate
{{ t "home.title" }}
{{ t "guest.hello" "name" .Guest.Name }}
```

Locale is selected per Render, never globally:

```go
err := views.Render(ctx, w, "pages/home", data, thema.WithLocale("ja"))
```

Concurrent Render calls can use different locales without leaking state. The
fallback chain is requested locale, requested base language, configured default
locale, configured default base language, then the translation key itself.
The default locale is `en`; change it with `WithDefaultLocale`.

Translation results are normal strings and remain subject to `html/template`
contextual escaping.

## Assets

Assets live under `assets/` and use normalized logical paths:

```gotemplate
{{ asset "css/app.css" }}
```

The default URL is:

```text
/assets/<theme-id>/<logical-path>?v=<generation>
```

The generation token binds HTML references to the active Theme Snapshot. Use
`WithAssetBaseURL` to replace `/assets`. The application is responsible for
serving or publishing assets; Thema does not bundle, minify, compile, optimize,
or upload them.

## Slots and Contributions

A Theme can expose an extension point:

```gotemplate
{{ slot "navigation.main" . }}
```

A Slot is an HTML-fragment extension point. It is valid only as a direct action
in HTML content context, such as inside a `<nav>` element. Candidate validation
uses `html/template`'s contextual analysis and rejects Slot use in attributes,
URLs, JavaScript, CSS, or any other non-HTML-content context, including indirect
use through `{{ template }}` composition.

```gotemplate
<nav>{{ slot "navigation.main" . }}</nav>            {{/* valid */}}
<a title="{{ slot "navigation.main" . }}">Link</a>  {{/* rejected */}}
```

Trusted application code registers existing Theme templates:

```go
err := views.Contribute("navigation.main", thema.Contribution{
    ID:       "booking-link",
    Template: "contributions/booking-link",
    Order:    20,
})
```

Lower Order renders first. Equal Order preserves registration order. IDs are
unique inside one Slot. `RemoveContribution(slot, id)` reports whether it
removed an entry. A Refresh candidate is rejected if it removes a contributed
template.

## Deterministic source resolution

Thema maps all recognized files to logical names before parsing. Duplicate
template definitions are rejected; behavior never depends on filesystem walk or
`html/template` parse order. v0.1 activates one complete Theme and does not mix
templates, locales, or assets from other Theme directories.

## Refresh

External Theme files can change without recompiling the Go application:

```go
changed, err := views.Refresh(ctx)
```

Refresh computes a content fingerprint over the complete recognized Theme
generation. Unchanged content returns `false, nil`. Changed content is loaded,
validated, parsed, compiled, and checked against registered Contributions before
one atomic Snapshot swap.

A failed candidate returns an error and retains the last valid Snapshot. In-flight
requests may finish against the old Snapshot. Refresh calls are serialized while
normal Render calls continue. Thema starts no ticker and no hidden goroutine; the
caller owns refresh timing.

Render never scans directories, downloads content, reparses files, rebuilds
source maps, or mutates shared configuration. It executes a per-request clone of
the immutable compiled template prototype so the reserved runtime helpers can be
bound to that request's locale and Snapshot without wrapping root data.

## Missing values

The default preserves normal `html/template` missing-key behavior. Development
and Theme validation can opt into map lookup failures:

```go
views, err := thema.New("./themes", "default", thema.WithStrictMissingKeys())
```

## Error handling

Stable errors support `errors.Is`:

- `ErrTemplateNotFound`
- `ErrInvalidTheme`
- `ErrInvalidPath`
- `ErrIncompatibleTheme`
- `ErrDuplicateContribution`
- `ErrRender`
- `ErrRefresh`

Diagnostics identify the Theme, logical template, Theme-relative source, line,
and underlying parse or execution error where available. They do not expose
absolute server paths, template data, or template source.

Render executes into an internal buffer first. An execution error leaves the
destination Writer untouched. Errors from the final Writer can still produce a
partial destination write, as with any `io.Writer` commit.

## Security model

Thema preserves `html/template` contextual escaping for HTML, attributes, URLs,
JavaScript, and CSS. It does not reimplement escaping and provides no convenient
escape bypass.

Thema does not ship helpers for shell execution, filesystem I/O, HTTP, SQL,
environment access, or raw/unsafe HTML. Security is capability-based: a Theme can
only use the ViewModel and trusted functions the Go application provides. Prefer
presentation-focused ViewModels without database, deletion, network, filesystem,
or system methods.

Trusted Go application code can deliberately provide standard typed values such
as `template.HTML`. JavaScript assets are executable browser code and must be
treated as trusted Theme content.

Theme loading rejects unsafe logical paths, traversal, backslashes, absolute
logical names, symlinks, and non-regular files. Default resource limits are
1,000 templates, 8 MiB per recognized file, and 128 MiB per Theme generation.

## Static output

Render accepts `io.Writer`, so static output needs no separate API:

```go
file, err := os.Create("index.html")
if err != nil {
    return err
}
defer file.Close()
return views.Render(ctx, file, "pages/home", data)
```

Thema is not a Static Site Generator. Routes, page discovery, sitemaps, output
planning, and publishing remain application responsibilities.

## Examples

### Minimal

[`examples/minimal`](examples/minimal) is the smallest working browser
integration. It shows `New`, `Must`, `Render`, native root data, the `asset`
helper, and application-owned `net/http` routing and static file serving.

```sh
cd examples/minimal
go run .
```

Open [http://localhost:8080](http://localhost:8080).

### Advanced

[`examples/advanced`](examples/advanced) shows realistic composition with typed
application ViewModels, native `{{ template }}` and `{{ range }}`, multiple
pages, components, per-render English and Chinese localization, assets, and an
HTML-content Slot with an application-registered Contribution.

```sh
cd examples/advanced
go run .
```

Open [http://localhost:8080](http://localhost:8080), then visit `/rooms` or use
`/?lang=en` and `/?lang=zh`.

Both servers belong only to their examples and use Go's standard `net/http`
package. Thema itself does not provide routing, middleware, or static file
serving.

## Specification

The frozen, normative v0.1 contract is
[`docs/specification-v0.1.md`](docs/specification-v0.1.md). If implementation or
other documentation conflicts with it, the frozen specification wins.
