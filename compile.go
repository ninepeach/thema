package thema

import (
	"fmt"
	"html/template"
	"sort"
	"text/template/parse"
)

var reservedFuncNames = map[string]struct{}{
	"t":     {},
	"asset": {},
	"slot":  {},
}

type snapshot struct {
	pkg           *themePackage
	templates     *template.Template
	translator    *translator
	contributions map[string][]registeredContribution
}

func compileSnapshot(pkg *themePackage, funcs template.FuncMap, cfg config, contributions map[string][]registeredContribution) (*snapshot, error) {
	parseFuncs := cloneFuncMap(funcs)
	parseFuncs["t"] = func(string, ...any) (string, error) { return "", nil }
	parseFuncs["asset"] = func(string) (string, error) { return "", nil }
	parseFuncs["slot"] = func(string, any) (template.HTML, error) { return "", nil }

	compiled := template.New("_thema").Funcs(parseFuncs).Option(cfg.missingKey)
	owners := make(map[string]string)
	for _, source := range pkg.templates {
		parsed, err := template.New(source.name).Funcs(parseFuncs).Option(cfg.missingKey).Parse(source.text)
		if err != nil {
			return nil, invalidTheme(pkg.id, "%s: %v", source.source, err)
		}
		definitions := parsed.Templates()
		sort.Slice(definitions, func(i, j int) bool { return definitions[i].Name() < definitions[j].Name() })
		for _, definition := range definitions {
			name := definition.Name()
			if name == "_thema" {
				return nil, invalidTheme(pkg.id, "%s defines reserved template %q", source.source, name)
			}
			if previous, exists := owners[name]; exists {
				return nil, invalidTheme(pkg.id, "duplicate template %q in %s and %s", name, previous, source.source)
			}
			if definition.Tree == nil {
				continue
			}
			if _, err := compiled.AddParseTree(name, definition.Tree); err != nil {
				return nil, invalidTheme(pkg.id, "%s: cannot compile template %q: %v", source.source, name, err)
			}
			owners[name] = source.source
		}
	}
	if err := validateTemplateReferences(compiled); err != nil {
		return nil, invalidTheme(pkg.id, "%v", err)
	}
	if err := validateContributions(compiled, contributions); err != nil {
		return nil, err
	}
	return &snapshot{
		pkg:           pkg,
		templates:     compiled,
		translator:    newTranslator(pkg.translations, cfg.defaultLocale),
		contributions: cloneContributions(contributions),
	}, nil
}

func validateTemplateReferences(compiled *template.Template) error {
	for _, tmpl := range compiled.Templates() {
		if tmpl.Tree == nil || tmpl.Tree.Root == nil {
			continue
		}
		if err := walkTemplateNodes(tmpl.Tree.Root, func(name string) error {
			dependency := compiled.Lookup(name)
			if dependency == nil || dependency.Tree == nil {
				return fmt.Errorf("template %q references missing template %q", tmpl.Name(), name)
			}
			return nil
		}); err != nil {
			return err
		}
	}
	return nil
}

func walkTemplateNodes(node parse.Node, visit func(string) error) error {
	if node == nil {
		return nil
	}
	switch current := node.(type) {
	case *parse.ListNode:
		for _, child := range current.Nodes {
			if err := walkTemplateNodes(child, visit); err != nil {
				return err
			}
		}
	case *parse.TemplateNode:
		return visit(current.Name)
	case *parse.IfNode:
		if err := walkTemplateNodes(current.List, visit); err != nil {
			return err
		}
		return walkTemplateNodes(current.ElseList, visit)
	case *parse.RangeNode:
		if err := walkTemplateNodes(current.List, visit); err != nil {
			return err
		}
		return walkTemplateNodes(current.ElseList, visit)
	case *parse.WithNode:
		if err := walkTemplateNodes(current.List, visit); err != nil {
			return err
		}
		return walkTemplateNodes(current.ElseList, visit)
	}
	return nil
}

func validatedFuncMap(funcs template.FuncMap) (result template.FuncMap, err error) {
	result = cloneFuncMap(funcs)
	for name := range result {
		if _, reserved := reservedFuncNames[name]; reserved {
			return nil, fmt.Errorf("%w: helper name %q is reserved", ErrInvalidTheme, name)
		}
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			result = nil
			err = fmt.Errorf("%w: invalid template helper: %v", ErrInvalidTheme, recovered)
		}
	}()
	template.New("_func_validation").Funcs(result)
	return result, nil
}

func cloneFuncMap(source template.FuncMap) template.FuncMap {
	result := make(template.FuncMap, len(source)+3)
	for name, fn := range source {
		result[name] = fn
	}
	return result
}

