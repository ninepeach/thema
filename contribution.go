package thema

import (
	"fmt"
	"html/template"
	"sort"
)

// Contribution registers a template for rendering inside a named Slot.
type Contribution struct {
	ID       string
	Template string
	Order    int
}

type registeredContribution struct {
	Contribution
	sequence uint64
}

func validateContributions(compiled *template.Template, contributions map[string][]registeredContribution) error {
	for slot, entries := range contributions {
		seen := make(map[string]struct{}, len(entries))
		for _, entry := range entries {
			if _, exists := seen[entry.ID]; exists {
				return fmt.Errorf("%w: slot %q, ID %q", ErrDuplicateContribution, slot, entry.ID)
			}
			seen[entry.ID] = struct{}{}
			if compiled.Lookup(entry.Template) == nil {
				return fmt.Errorf("%w: contribution %q references %q", ErrTemplateNotFound, entry.ID, entry.Template)
			}
		}
	}
	return nil
}

func cloneContributions(source map[string][]registeredContribution) map[string][]registeredContribution {
	result := make(map[string][]registeredContribution, len(source))
	for slot, entries := range source {
		result[slot] = append([]registeredContribution(nil), entries...)
	}
	return result
}

func orderedContributions(entries []registeredContribution) []registeredContribution {
	result := append([]registeredContribution(nil), entries...)
	sort.Slice(result, func(i, j int) bool {
		if result[i].Order == result[j].Order {
			return result[i].sequence < result[j].sequence
		}
		return result[i].Order < result[j].Order
	})
	return result
}

