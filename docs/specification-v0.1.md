# Thema v0.1 Frozen Specification

This document is the normative, frozen contract for Thema v0.1. If it conflicts
with the README, examples, implementation notes, tests, code comments, or code,
this specification wins.

## 1. Purpose

Thema is a Go-native HTML template and theme runtime built on `html/template`.

Its goal is to make it easy to build:

- websites
- admin panels
- multilingual sites
- themeable applications
- white-label interfaces
- hot-updatable presentation layers

without introducing a new template language.

Core principle:

> Small core. Strong composition. Explicit behavior.

## 2. Template Language

Thema MUST use Go `html/template`.

Thema MUST NOT define a new parser or template language.

Existing Go template semantics MUST remain compatible.

Given:

```go
type Page struct {
    Title string
}
```

this MUST work unchanged:

```gotemplate
<h1>{{ .Title }}</h1>
```

Thema MUST NOT wrap application data as `{{ .Data.Title }}`.

Application data passed to Render remains the root `.`.

## 3. Initialization

The primary initialization API SHALL be:

```go
views, err := thema.New("./themes", "default")
```

Convenience:

```go
views := thema.Must(
    thema.New("./themes", "default"),
)
```

Arguments:

- `./themes` = Theme Repository
- `default` = Active Theme ID

Thema SHALL NOT provide:

- `Default`
- `GetInstance`
- global singleton
- package-level mutable Engine

Multiple independent Engine instances MUST be supported.

## 4. Theme Repository

Standard structure:

```text
themes/
├── default/
│   ├── theme.json
│   ├── templates/
│   ├── locales/
│   └── assets/
│
├── minimal/
│   ├── theme.json
│   ├── templates/
│   ├── locales/
│   └── assets/
│
└── custom/
    ├── theme.json
    ├── templates/
    ├── locales/
    └── assets/
```

A Theme is one coherent presentation unit.

The following resources MUST belong to the same active Theme generation:

- templates
- locales
- assets
- manifest

Thema MUST NOT combine Theme A templates + Theme B locales + Theme C assets as
one active Theme.

## 5. Theme Identity

Theme ID is the directory name.

Example: `themes/default/`

Theme ID: `default`

`theme.json` v0.1:

```json
{
  "name": "Default",
  "version": "1.0.0",
  "thema": "0.1"
}
```

All three fields are required.

`theme.json` is strict. Unknown fields and trailing or multiple JSON values MUST
be rejected.

`name` is the human-readable theme name.

`version` is the Theme release version. It MUST be valid SemVer.

`thema` is the Thema Theme Contract version. v0.1 uses `0.1`.

An incompatible Theme Contract MUST prevent activation.

## 6. Logical Template Paths

Thema MUST distinguish:

- Template logical path
- Template source path
- Asset logical path
- URL/route path

They MUST NOT be treated as the same namespace.

Example:

- Source: `themes/default/templates/pages/home.html`
- Logical template: `pages/home`
- Possible HTTP URL: `/home`

Only the logical template name belongs to Thema's Render API.

```go
views.Render(
    ctx,
    w,
    "pages/home",
    data,
)
```

Application code MUST NOT need to specify `.html`, `themes/`, `templates/`, or
absolute filesystem paths.

## 7. Logical Path Rules

Valid:

- `pages/home`
- `components/header`
- `reservation/detail`
- `layouts/app`

Invalid:

- `/pages/home`
- `pages/home/`
- `../home`
- `pages/../home`
- `pages//home`
- `pages/./home`

Logical paths MUST:

- be relative
- use `/`
- contain no `.` segments
- contain no `..`
- contain no empty segments
- have no leading `/`
- have no trailing `/`
- omit `.html`

Windows filesystem separators MUST NOT alter logical names.

## 8. File Mapping

Only `.html` template files are recognized in v0.1.

Mapping:

```text
templates/pages/home.html
→
pages/home

templates/components/header.html
→
components/header
```

No special handling exists for `index.html`.

`templates/pages/rooms/index.html` maps to `pages/rooms/index`, not
`pages/rooms`.

Routing is outside Thema.

## 9. Automatic Template Naming

A template file does not need to contain `{{ define "pages/home" }}`.

The logical path itself provides the primary template name.

Example file: `templates/pages/home.html`

It can simply contain:

```gotemplate
<h1>{{ .Title }}</h1>
```

and is rendered using:

```go
views.Render(ctx, w, "pages/home", data)
```

Native `{{ define }}` MUST remain supported for additional named templates.

## 10. Template Composition

Go's native `{{ template "components/header" . }}` is the standard composition
mechanism.

Thema SHALL NOT introduce separate runtimes for Page, Partial, Component, or
Layout.

They are templates differentiated primarily through convention.

Examples:

- `layouts/app`
- `pages/home`
- `components/header`
- `partials/room-row`

## 11. Theme Override

Theme override MUST be deterministic.

The winning template MUST be resolved before parsing the final template set.

Conceptually:

```text
scan
↓
logical-name map
↓
resolve override
↓
final source set
↓
parse
↓
compile
```

Thema MUST NOT rely on accidental `html/template` parse order for overriding
templates.

If multiple layers eventually exist, later layers override earlier layers.

## 12. Runtime Snapshot

The running Engine MUST render from an immutable compiled Snapshot.

Conceptually:

```text
Sources
↓
Resolve
↓
Parse
↓
Validate
↓
Compile
↓
Snapshot
```

Render hot path:

```text
Current Snapshot
↓
ExecuteTemplate
↓
HTML
```

Render MUST NOT:

- scan directories
- download themes
- reparse files
- rebuild source maps
- modify shared configuration

## 13. Refresh

External Theme resources MUST be refreshable without recompiling the Go
application.

Public behavior:

```go
changed, err := views.Refresh(ctx)
```

Refresh SHALL:

```text
check source version/change
↓
unchanged
→ return false, nil
changed
↓
load candidate
↓
validate manifest
↓
validate Thema contract
↓
resolve template set
↓
parse
↓
validate
↓
compile
↓
create candidate Snapshot
↓
atomic swap
↓
return true, nil
```

Thema SHALL NOT start an internal refresh ticker or hidden background goroutine.

The caller decides when `Refresh()` executes.

## 14. Failed Refresh

A failed Theme update MUST NOT damage the active runtime.

If the candidate contains invalid manifest, incompatible contract, invalid
paths, missing template dependencies, parse errors, compile errors, or validation
failures, then the candidate is rejected and the current Snapshot is retained.

Existing requests continue using the last valid Snapshot.

## 15. Atomic Activation

Snapshot replacement MUST be atomic.

A Render operation can see the old Snapshot or the new Snapshot, but never an
intermediate partially rebuilt state.

Existing requests using an old Snapshot MAY finish normally after a new
Snapshot has become active.

Manual reference counting SHOULD NOT be required.

## 16. Source Change Detection

Thema SHOULD avoid recompiling templates when resources have not changed.

Source version may be based on manifest version, file metadata, content
fingerprint, or an implementation-specific opaque version.

The Engine should care only whether old source version differs from new source
version.

The Source implementation owns how that version is calculated.

A fixed template TTL is NOT the core model.

## 17. Rendering API

Target API:

```go
func (e *Engine) Render(
    ctx context.Context,
    w io.Writer,
    name string,
    data any,
    opts ...RenderOption,
) error
```

`context.Context` supports cancellation, deadline, per-render runtime state,
and future tracing integration.

Application data remains unchanged as root `.`.

## 18. Safe Render Commit

Thema SHOULD render into an internal buffer first.

```text
Execute template
↓
buffer
↓
success?
├─ no → return error; destination Writer unchanged
└─ yes → copy buffer to Writer
```

This prevents partial responses when template execution fails midway.

## 19. Variables

Thema MUST preserve native Go template variables.

Examples:

```gotemplate
{{ .Title }}
{{ .Guest.Name }}
{{ range .Rooms }}
    {{ .Name }}
{{ end }}
{{ $guest := .Guest }}
{{ $guest.Name }}
{{ with .Property }}
    {{ .Name }}
{{ end }}
```

Thema SHALL NOT introduce a separate Variable Engine.

## 20. Helpers

Helpers are ordinary Go template functions.

Example:

```gotemplate
{{ .Name | trim | upper }}
```

Application registration:

```go
views.Funcs(template.FuncMap{
    "money": formatMoney,
    "date":  formatDate,
})
```

Thema MAY provide a small set of pure presentation helpers.

Helper functionality MUST remain additive to `html/template`.

Thema SHALL NOT create a second Filter language.

## 21. Reserved Runtime Helpers

Runtime-level helper names may include `t`, `asset`, and `slot`.

These names SHOULD be reserved by Thema.

Themes MUST NOT redefine runtime helpers.

Themes do not register Go functions.

Only trusted Go application code can register helpers.

## 22. Dangerous Helpers

Thema MUST NOT ship helpers such as:

- `exec`
- `shell`
- `readFile`
- `writeFile`
- `httpGet`
- `sql`
- `env`
- `raw`
- `safe`
- `unsafeHTML`

Thema MUST NOT use keyword blacklists as its primary security mechanism.

Security is capability-based: if a capability is not registered, a template
cannot use it.

## 23. Escaping

Thema MUST preserve `html/template` contextual escaping.

Normal strings MUST remain automatically escaped according to context:

- HTML text
- attributes
- URLs
- JavaScript context
- CSS context

Thema MUST NOT reimplement escaping.

Thema MUST NOT provide a convenient built-in escape bypass.

Trusted HTML, if required, must be explicitly constructed by trusted Go
application code using standard Go types such as `template.HTML`.

## 24. ViewModel Security

Templates SHOULD receive presentation-focused ViewModels.

Preferred:

```text
Handler
↓
Service
↓
ViewModel
↓
Thema
```

Avoid passing objects exposing powerful methods such as database operations,
deletion methods, network methods, filesystem methods, or system operations.

Theme capability is determined by data exposed + registered template funcs.

## 25. i18n

Basic i18n support is a first-class v0.1 feature.

The standard Theme structure includes:

```text
locales/
├── en.json
├── ja.json
├── zh-CN.json
└── zh-TW.json
```

Templates should support:

```gotemplate
{{ t "home.title" }}
```

and interpolation:

```gotemplate
{{ t "guest.hello" "name" .Guest.Name }}
```

## 26. Default Translator

Thema SHALL provide a basic built-in Translator suitable for normal Theme
localization.

It should support locale lookup, translation key lookup, interpolation
variables, fallback policy, and safe string output.

Users SHOULD NOT need to provide a Translator merely to enable basic i18n.

An advanced replacement Translator MAY be supported through an option.

## 27. Per-render Locale

Locale MUST be per Render.

```go
views.Render(
    ctx,
    w,
    "pages/home",
    data,
    thema.WithLocale("ja"),
)
```

Thema MUST NOT implement `SetGlobalLocale("ja")`.

Concurrent renders MUST safely support Request A using `en`, Request B using
`ja`, and Request C using `zh-CN` without state leakage.

## 28. Application Data Compatibility

Runtime capabilities MUST NOT require wrapping application data.

This MUST remain valid:

```gotemplate
{{ .Title }}
```

not:

```gotemplate
{{ .Data.Title }}
```

Runtime capabilities are exposed through functions such as:

```gotemplate
{{ t "home.title" }}
{{ asset "css/app.css" }}
{{ slot "navigation.main" . }}
```

Extensions MUST be additive, not invasive.

## 29. Slot

Slot is an optional template extension primitive.

Example:

```gotemplate
{{ slot "navigation.main" . }}
```

Slot is an HTML-fragment extension point and MUST only be valid as a direct
action in HTML content context.

Valid:

```gotemplate
<nav>{{ slot "navigation.main" . }}</nav>
```

Slot use in HTML attributes, URL contexts, JavaScript, CSS, or any other
non-HTML-content context MUST be rejected during candidate validation, including
when a template containing Slot is composed into such a context. Validation MUST
preserve and rely on `html/template` contextual analysis; it MUST NOT use a
keyword blacklist.

Slot names are logical identifiers.

Slots do not execute arbitrary external callbacks.

They render already registered Contributions against the current Render
context.

## 30. Contribution

Basic structure:

```go
type Contribution struct {
    ID       string
    Template string
    Order    int
}
```

Requirements:

- ID unique within Slot
- lower Order first
- same Order preserves registration order
- behavior deterministic

Candidate Snapshot validation MUST verify that referenced Contribution
templates exist.

## 31. Remove Contribution

Thema SHOULD support:

```go
RemoveContribution(slot, id)
```

Separate operations such as ReplaceContribution, TransformContribution, and
FilterContribution are unnecessary in v0.1.

Replace can emerge from Remove + Contribute.

## 32. Assets

Theme assets belong under `assets/`.

Example:

```text
assets/
├── css/
├── js/
└── images/
```

Templates may use:

```gotemplate
{{ asset "css/app.css" }}
```

Asset names MUST use normalized logical paths.

Thema does NOT provide bundling, minification, SCSS, TypeScript compilation,
image optimization, or CDN upload.

Asset URL resolution MAY be delegated through a resolver/helper.

## 33. Theme Consistency

HTML templates and asset resolution MUST correspond to the same active Theme
generation where relevant.

An online refresh MUST avoid states such as HTML from Theme 1.3, CSS from Theme
1.2, and JavaScript from Theme 1.4.

Theme resources should switch as one coherent Snapshot/generation.

## 34. External Theme Security

External Themes are presentation packages, not application code.

Themes MAY contain HTML templates, CSS, images, localization data, and
explicitly permitted frontend assets.

Themes MUST NOT gain database access, shell execution, server filesystem access,
arbitrary HTTP access, arbitrary Go execution, or new API handlers.

JavaScript assets are executable browser code and MUST be treated as trusted
Theme content, not harmless data.

## 35. Path Security

External Theme loading MUST defend against path traversal.

Thema MUST reject unsafe logical paths.

Theme packaging/install layers SHOULD prevent symlink escape outside the Theme
root.

Absolute server filesystem paths MUST NOT be accepted as template logical
names.

## 36. Validation

Candidate Theme validation SHOULD include:

- valid manifest
- valid Theme ID/source
- compatible Thema contract
- valid logical paths
- duplicate logical-name checks
- template parse validation
- referenced-template existence
- Contribution template existence
- illegal path detection

Reasonable resource limits SHOULD be supported or documented for number of
templates, individual file size, and total Theme size.

## 37. Missing Values

Thema SHALL preserve Go template compatibility by default.

A stricter missing-key mode SHOULD be available for development and Theme
validation.

The exact public option may be decided during implementation.

Goal: strict validation where useful without breaking normal `html/template`
expectations.

## 38. Error Handling

Internal errors SHOULD preserve useful diagnostics:

- logical template name
- theme ID
- theme version
- source file
- line
- underlying parse/execute error

Production-facing errors MUST NOT automatically expose absolute server paths,
template source, sensitive ViewModel values, or system internals.

Potential stable errors:

- `ErrTemplateNotFound`
- `ErrInvalidTheme`
- `ErrInvalidPath`
- `ErrIncompatibleTheme`
- `ErrDuplicateContribution`
- `ErrRender`
- `ErrRefresh`

## 39. Concurrency

Engine MUST be safe for long-lived concurrent use after successful
initialization.

Requirements:

```text
Render ↔ Render      concurrent
Render ↔ Refresh     safe
Refresh ↔ Refresh    serialized/coalesced
```

Refresh MUST NOT block normal Render for the entire candidate compilation
period.

Existing Snapshot remains usable while candidate is being built.

## 40. Static Rendering

Because Render accepts `io.Writer`, Thema naturally supports static HTML output.

```go
f, err := os.Create("index.html")
if err != nil {
    return err
}
return views.Render(
    ctx,
    f,
    "pages/home",
    data,
)
```

Thema SHALL NOT add a redundant `RenderStatic()` in v0.1.

## 41. Static Site Generation

Thema is NOT a Static Site Generator.

It does not own routes, page discovery, sitemap generation, output directory
planning, or publishing.

A future higher-level site package may use Thema.

## 42. Repository Deliverables

v0.1 repository SHALL include:

- `README.md`
- `LICENSE`
- `go.mod`
- library source
- unit/integration tests
- `examples/`
- `docs/specification-v0.1.md`

Examples are repository artifacts only.

They MUST NOT be runtime/library dependencies.

## 43. Examples

Keep examples small.

Recommended:

```text
examples/
├── basic/
├── theme/
└── i18n/
```

`basic` validates New, Render, native variables, template composition, and
helpers.

`theme` validates Theme Repository, active Theme, override, Refresh, candidate
failure, and atomic activation.

`i18n` validates built-in Translator, WithLocale, interpolation, fallback,
escaping, and concurrent locales.

## 44. README Requirements

README is the primary user-facing usage document.

It MUST include:

1. What Thema is
2. Installation
3. Quick Start
4. Theme directory structure
5. `theme.json`
6. `New(themeRepository, activeTheme)`
7. Render
8. Native Go variables
9. Template composition
10. Helpers
11. i18n
12. Theme override
13. Refresh
14. Error handling
15. Security model
16. Link to specification

The README MUST clearly state:

> Thema uses Go `html/template`; it does not define a new template language.

and:

> Render data remains the native template root dot (`.`).

## 45. Public API Target

v0.1 public API SHOULD stay near:

```go
New(themeRepository, activeTheme string, opts ...Option) (*Engine, error)
Must(*Engine, error) *Engine
(*Engine).Render(
    context.Context,
    io.Writer,
    string,
    any,
    ...RenderOption,
) error
(*Engine).Refresh(context.Context) (bool, error)
(*Engine).Funcs(template.FuncMap) error
(*Engine).Contribute(string, Contribution) error
(*Engine).RemoveContribution(string, string) bool
```

Additional APIs require justification.

Before introducing a new abstraction, ask:

> Can this capability emerge from the existing primitives?

If yes, do not add it.

## 46. Frozen v0.1 Principles

1. `html/template` is the language.
2. Thema accepts a Theme Repository and Active Theme.
3. Themes contain templates, locales, assets, and manifest.
4. Theme ID is the directory name.
5. Theme version uses SemVer.
6. Theme Contract compatibility is explicitly validated.
7. Template references use logical paths.
8. Logical paths never expose filesystem locations.
9. Application data remains root `.`.
10. Helpers are ordinary registered Go template functions.
11. No separate Filter language.
12. No keyword blacklist security model.
13. Contextual escaping comes from `html/template`.
14. Themes control presentation, not capability.
15. i18n works by default.
16. Locale is per-render.
17. Runtime uses immutable compiled Snapshots.
18. External Theme updates use candidate validation and atomic activation.
19. Failed updates retain the last working Snapshot.
20. Render never performs Theme discovery or recompilation.
21. The package owns no global/default/singleton Engine.
22. Examples are optional repository content, never library dependencies.
23. Complex behavior should emerge from a small number of primitives.

## v0.1 Definition

Thema is a small Go-native HTML theme runtime that preserves `html/template`
compatibility while adding structured themes, localization, safe runtime
refresh, helpers, extension slots, and deterministic presentation management.
