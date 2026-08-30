package thema

import (
	"errors"
	"fmt"
	"html/template"
	"sort"
	"strings"
	"text/template/parse"
)

var errStopContextValidation = errors.New("thema: stop context validation execution")

// validateSlotContexts delegates context recognition to html/template. Its
// contextual escape pass appends a context-specific escaper to each output
// pipeline. A Slot is valid only as a direct action whose sole appended
// escaper is the HTML-content escaper.
func validateSlotContexts(compiled *template.Template, funcs template.FuncMap) error {
	templates := compiled.Templates()
	sort.Slice(templates, func(i, j int) bool { return templates[i].Name() < templates[j].Name() })

	declared := make(map[string]struct{}, len(templates))
	declaredNames := make([]string, 0, len(templates))
	directSlots := make(map[string]int, len(templates))
	dependencies := make(map[string][]string, len(templates))
	for _, tmpl := range templates {
		if tmpl.Name() == "_thema" || tmpl.Tree == nil || tmpl.Tree.Root == nil {
			continue
		}
		declared[tmpl.Name()] = struct{}{}
		declaredNames = append(declaredNames, tmpl.Name())
		if err := walkContextNodes(tmpl.Tree.Root, func(pipe *parse.PipeNode) error {
			directSlots[tmpl.Name()] += countIdentifier(pipe, "slot")
			return nil
		}, func(name string) error {
			dependencies[tmpl.Name()] = append(dependencies[tmpl.Name()], name)
			return nil
		}); err != nil {
			return err
		}
	}

	slotReachable := make(map[string]bool, len(templates))
	for name, count := range directSlots {
		slotReachable[name] = count > 0
	}
	for changed := true; changed; {
		changed = false
		for name, names := range dependencies {
			if slotReachable[name] {
				continue
			}
			for _, dependency := range names {
				if slotReachable[dependency] {
					slotReachable[name] = true
					changed = true
					break
				}
			}
		}
	}

	containsSlot := false
	for _, reachable := range slotReachable {
		containsSlot = containsSlot || reachable
	}
	if !containsSlot {
		return nil
	}
	htmlContentEscaper, err := detectHTMLContentEscaper()
	if err != nil {
		return err
	}

	candidate := template.New("_thema").Funcs(funcs)
	for _, tmpl := range templates {
		if tmpl.Name() == "_thema" || tmpl.Tree == nil || tmpl.Tree.Root == nil {
			continue
		}
		tree := tmpl.Tree.Copy()
		tree.Root.Nodes = append([]parse.Node{
			&parse.TextNode{NodeType: parse.NodeText, Text: []byte("thema-context-validation")},
		}, tree.Root.Nodes...)
		if _, err := candidate.AddParseTree(tmpl.Name(), tree); err != nil {
			return fmt.Errorf("cannot copy template %q for Slot validation: %w", tmpl.Name(), err)
		}
	}
	for _, name := range declaredNames {
		if !slotReachable[name] {
			continue
		}
		err := candidate.ExecuteTemplate(stopContextWriter{}, name, nil)
		var escapeError *template.Error
		if errors.As(err, &escapeError) {
			return fmt.Errorf("template %q has an invalid HTML context: %w", name, escapeError)
		}

		escaped := candidate.Lookup(name)
		if escaped == nil || escaped.Tree == nil || escaped.Tree.Root == nil {
			continue
		}
		validatedSlots := 0
		if err := walkContextNodes(escaped.Tree.Root, func(pipe *parse.PipeNode) error {
			count := countIdentifier(pipe, "slot")
			if count == 0 {
				return nil
			}
			validatedSlots += count
			if count != 1 || !isHTMLContentSlot(pipe, htmlContentEscaper) {
				return fmt.Errorf("template %q: slot is only valid as a direct HTML content action", name)
			}
			return nil
		}, func(target string) error {
			if _, exact := declared[target]; exact {
				return nil
			}
			if base, ok := contextualTemplateBase(target, declared); ok && slotReachable[base] {
				return fmt.Errorf("template %q: template %q containing slot is used outside HTML content context", name, base)
			}
			return nil
		}); err != nil {
			return err
		}
		if validatedSlots != directSlots[name] {
			return fmt.Errorf("template %q: slot is only valid as a direct HTML content action", name)
		}
	}
	return nil
}

func isHTMLContentSlot(pipe *parse.PipeNode, htmlContentEscaper string) bool {
	if pipe == nil || len(pipe.Decl) != 0 || pipe.IsAssign || len(pipe.Cmds) != 2 {
		return false
	}
	call, escape := pipe.Cmds[0], pipe.Cmds[1]
	if len(call.Args) != 3 || len(escape.Args) != 1 {
		return false
	}
	callName, ok := call.Args[0].(*parse.IdentifierNode)
	if !ok || callName.Ident != "slot" {
		return false
	}
	escapeName, ok := escape.Args[0].(*parse.IdentifierNode)
	return ok && escapeName.Ident == htmlContentEscaper
}

func detectHTMLContentEscaper() (string, error) {
	const probeName = "_thema_html_content_probe"
	probe, err := template.New(probeName).Funcs(template.FuncMap{
		probeName: func() template.HTML { return "" },
	}).Parse("thema-context-validation{{" + probeName + "}}")
	if err != nil {
		return "", fmt.Errorf("cannot create HTML content context probe: %w", err)
	}
	_ = probe.Execute(stopContextWriter{}, nil)
	root := probe.Tree.Root
	if root == nil || len(root.Nodes) != 2 {
		return "", errors.New("cannot identify html/template HTML content context")
	}
	action, ok := root.Nodes[1].(*parse.ActionNode)
	if !ok || action.Pipe == nil || len(action.Pipe.Cmds) != 2 {
		return "", errors.New("cannot identify html/template HTML content escaper")
	}
	escape := action.Pipe.Cmds[1]
	if len(escape.Args) != 1 {
		return "", errors.New("cannot identify html/template HTML content escaper")
	}
	identifier, ok := escape.Args[0].(*parse.IdentifierNode)
	if !ok {
		return "", errors.New("cannot identify html/template HTML content escaper")
	}
	return identifier.Ident, nil
}

func contextualTemplateBase(name string, declared map[string]struct{}) (string, bool) {
	base := ""
	for candidate := range declared {
		if len(candidate) > len(base) && strings.HasPrefix(name, candidate+"$") {
			base = candidate
		}
	}
	return base, base != ""
}

type stopContextWriter struct{}

func (stopContextWriter) Write([]byte) (int, error) {
	return 0, errStopContextValidation
}

func walkContextNodes(node parse.Node, visitPipe func(*parse.PipeNode) error, visitTemplate func(string) error) error {
	if node == nil {
		return nil
	}
	switch current := node.(type) {
	case *parse.ListNode:
		if current == nil {
			return nil
		}
		for _, child := range current.Nodes {
			if err := walkContextNodes(child, visitPipe, visitTemplate); err != nil {
				return err
			}
		}
	case *parse.ActionNode:
		return visitPipe(current.Pipe)
	case *parse.TemplateNode:
		if err := visitPipe(current.Pipe); err != nil {
			return err
		}
		return visitTemplate(current.Name)
	case *parse.IfNode:
		if err := visitPipe(current.Pipe); err != nil {
			return err
		}
		if err := walkContextNodes(current.List, visitPipe, visitTemplate); err != nil {
			return err
		}
		return walkContextNodes(current.ElseList, visitPipe, visitTemplate)
	case *parse.RangeNode:
		if err := visitPipe(current.Pipe); err != nil {
			return err
		}
		if err := walkContextNodes(current.List, visitPipe, visitTemplate); err != nil {
			return err
		}
		return walkContextNodes(current.ElseList, visitPipe, visitTemplate)
	case *parse.WithNode:
		if err := visitPipe(current.Pipe); err != nil {
			return err
		}
		if err := walkContextNodes(current.List, visitPipe, visitTemplate); err != nil {
			return err
		}
		return walkContextNodes(current.ElseList, visitPipe, visitTemplate)
	}
	return nil
}

func countIdentifier(node parse.Node, name string) int {
	if node == nil {
		return 0
	}
	switch current := node.(type) {
	case *parse.IdentifierNode:
		if current.Ident == name {
			return 1
		}
	case *parse.PipeNode:
		count := 0
		for _, command := range current.Cmds {
			count += countIdentifier(command, name)
		}
		return count
	case *parse.CommandNode:
		count := 0
		for _, argument := range current.Args {
			count += countIdentifier(argument, name)
		}
		return count
	case *parse.ChainNode:
		return countIdentifier(current.Node, name)
	}
	return 0
}
