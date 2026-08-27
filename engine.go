package thema

import (
	"context"
	"fmt"
	"html/template"
	"path/filepath"
	"sync"
	"sync/atomic"
)

// Engine renders one active Theme from a Theme Repository.
type Engine struct {
	repository string
	themeID    string
	config     config

	current atomic.Pointer[snapshot]

	mutationMu      sync.Mutex
	funcs           template.FuncMap
	nextContribution uint64
}

// New loads, validates, and compiles activeTheme from themeRepository.
func New(themeRepository, activeTheme string, opts ...Option) (*Engine, error) {
	if themeRepository == "" {
		return nil, fmt.Errorf("%w: Theme Repository is empty", ErrInvalidTheme)
	}
	if err := validateThemeID(activeTheme); err != nil {
		return nil, err
	}
	repository, err := filepath.Abs(themeRepository)
	if err != nil {
		return nil, fmt.Errorf("%w: cannot resolve Theme Repository", ErrInvalidTheme)
	}
	cfg := config{
		defaultLocale: defaultLocale,
		missingKey:    "missingkey=default",
		assetBaseURL:  "/assets",
		initialFuncs:  make(template.FuncMap),
	}
	for _, option := range opts {
		if option == nil {
			return nil, fmt.Errorf("%w: nil Option", ErrInvalidTheme)
		}
		if err := option(&cfg); err != nil {
			return nil, err
		}
	}
	funcs, err := validatedFuncMap(cfg.initialFuncs)
	if err != nil {
		return nil, err
	}
	pkg, err := loadTheme(context.Background(), repository, activeTheme)
	if err != nil {
		return nil, err
	}
	compiled, err := compileSnapshot(pkg, funcs, cfg, nil)
	if err != nil {
		return nil, err
	}
	engine := &Engine{
		repository: repository,
		themeID:    activeTheme,
		config:     cfg,
		funcs:      funcs,
	}
	engine.current.Store(compiled)
	return engine, nil
}

// Must returns engine or panics when err is non-nil.
func Must(engine *Engine, err error) *Engine {
	if err != nil {
		panic(err)
	}
	return engine
}

// Refresh compiles and atomically activates a changed Theme generation.
// A failed candidate never replaces the active Snapshot.
func (e *Engine) Refresh(ctx context.Context) (bool, error) {
	if ctx == nil {
		return false, fmt.Errorf("%w: nil context", ErrRefresh)
	}
	e.mutationMu.Lock()
	defer e.mutationMu.Unlock()
	if err := ctx.Err(); err != nil {
		return false, fmt.Errorf("%w: %w", ErrRefresh, err)
	}
	candidatePackage, err := loadTheme(ctx, e.repository, e.themeID)
	if err != nil {
		return false, fmt.Errorf("%w: %w", ErrRefresh, err)
	}
	active := e.current.Load()
	if active.pkg.version == candidatePackage.version {
		return false, nil
	}
	if err := ctx.Err(); err != nil {
		return false, fmt.Errorf("%w: %w", ErrRefresh, err)
	}
	candidate, err := compileSnapshot(candidatePackage, e.funcs, e.config, active.contributions)
	if err != nil {
		return false, fmt.Errorf("%w: %w", ErrRefresh, err)
	}
	if err := ctx.Err(); err != nil {
		return false, fmt.Errorf("%w: %w", ErrRefresh, err)
	}
	e.current.Store(candidate)
	return true, nil
}

// Funcs atomically adds trusted application helpers and recompiles the active
// in-memory Theme generation. Reserved runtime helper names cannot be replaced.
func (e *Engine) Funcs(funcs template.FuncMap) error {
	validated, err := validatedFuncMap(funcs)
	if err != nil {
		return err
	}
	e.mutationMu.Lock()
	defer e.mutationMu.Unlock()
	merged := cloneFuncMap(e.funcs)
	for name, fn := range validated {
		merged[name] = fn
	}
	active := e.current.Load()
	candidate, err := compileSnapshot(active.pkg, merged, e.config, active.contributions)
	if err != nil {
		return err
	}
	e.funcs = merged
	e.current.Store(candidate)
	return nil
}

// Contribute registers a template in a Slot. IDs are unique within each Slot.
func (e *Engine) Contribute(slot string, contribution Contribution) error {
	if err := validateIdentifier("slot", slot); err != nil {
		return err
	}
	if err := validateIdentifier("contribution ID", contribution.ID); err != nil {
		return err
	}
	if err := validateTemplatePath(contribution.Template); err != nil {
		return err
	}
	e.mutationMu.Lock()
	defer e.mutationMu.Unlock()
	active := e.current.Load()
	if active.templates.Lookup(contribution.Template) == nil {
		return fmt.Errorf("%w: %q", ErrTemplateNotFound, contribution.Template)
	}
	contributions := cloneContributions(active.contributions)
	for _, existing := range contributions[slot] {
		if existing.ID == contribution.ID {
			return fmt.Errorf("%w: slot %q, ID %q", ErrDuplicateContribution, slot, contribution.ID)
		}
	}
	e.nextContribution++
	contributions[slot] = append(contributions[slot], registeredContribution{
		Contribution: contribution,
		sequence:     e.nextContribution,
	})
	candidate := *active
	candidate.contributions = contributions
	e.current.Store(&candidate)
	return nil
}

// RemoveContribution removes an ID from a Slot and reports whether it existed.
func (e *Engine) RemoveContribution(slot, id string) bool {
	if validateIdentifier("slot", slot) != nil || validateIdentifier("contribution ID", id) != nil {
		return false
	}
	e.mutationMu.Lock()
	defer e.mutationMu.Unlock()
	active := e.current.Load()
	entries := active.contributions[slot]
	index := -1
	for i := range entries {
		if entries[i].ID == id {
			index = i
			break
		}
	}
	if index < 0 {
		return false
	}
	contributions := cloneContributions(active.contributions)
	updated := append([]registeredContribution(nil), entries[:index]...)
	updated = append(updated, entries[index+1:]...)
	if len(updated) == 0 {
		delete(contributions, slot)
	} else {
		contributions[slot] = updated
	}
	candidate := *active
	candidate.contributions = contributions
	e.current.Store(&candidate)
	return true
}
