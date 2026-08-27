package thema

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
)

type translator struct {
	messages       map[string]map[string]string
	defaultLocale string
}

func newTranslator(messages map[string]map[string]string, fallback string) *translator {
	return &translator{messages: messages, defaultLocale: normalizeLocale(fallback)}
}

func (t *translator) translate(locale, key string, values ...any) (string, error) {
	if key == "" {
		return "", errors.New("translation key is empty")
	}
	message := key
	for _, candidate := range localeFallbacks(locale, t.defaultLocale) {
		if catalog := t.messages[candidate]; catalog != nil {
			if translated, ok := catalog[key]; ok {
				message = translated
				break
			}
		}
	}
	if len(values)%2 != 0 {
		return "", fmt.Errorf("translation %q requires name/value interpolation pairs", key)
	}
	if len(values) == 0 {
		return message, nil
	}
	replacements := make([]string, 0, len(values))
	for i := 0; i < len(values); i += 2 {
		name, ok := values[i].(string)
		if !ok || name == "" {
			return "", fmt.Errorf("translation %q has an invalid interpolation name", key)
		}
		replacements = append(replacements, "{{"+name+"}}", fmt.Sprint(values[i+1]))
	}
	return strings.NewReplacer(replacements...).Replace(message), nil
}

func localeFallbacks(locale, fallback string) []string {
	seen := make(map[string]struct{}, 4)
	result := make([]string, 0, 4)
	add := func(value string) {
		value = normalizeLocale(value)
		if value == "" {
			return
		}
		if _, exists := seen[value]; !exists {
			seen[value] = struct{}{}
			result = append(result, value)
		}
		if index := strings.IndexByte(value, '-'); index > 0 {
			base := value[:index]
			if _, exists := seen[base]; !exists {
				seen[base] = struct{}{}
				result = append(result, base)
			}
		}
	}
	add(locale)
	add(fallback)
	return result
}

func decodeLocale(data []byte) (map[string]string, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var root map[string]any
	if err := decoder.Decode(&root); err != nil {
		return nil, err
	}
	if root == nil {
		return nil, errors.New("locale must be a JSON object")
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, errors.New("multiple JSON values")
		}
		return nil, err
	}
	messages := make(map[string]string)
	if err := flattenMessages(messages, "", root); err != nil {
		return nil, err
	}
	return messages, nil
}

func flattenMessages(dst map[string]string, prefix string, source map[string]any) error {
	keys := make([]string, 0, len(source))
	for key := range source {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if key == "" || strings.HasPrefix(key, ".") || strings.HasSuffix(key, ".") || strings.Contains(key, "..") {
			return fmt.Errorf("invalid translation key %q", key)
		}
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}
		switch value := source[key].(type) {
		case string:
			if _, exists := dst[fullKey]; exists {
				return fmt.Errorf("duplicate translation key %q", fullKey)
			}
			dst[fullKey] = value
		case map[string]any:
			if err := flattenMessages(dst, fullKey, value); err != nil {
				return err
			}
		default:
			return fmt.Errorf("translation %q must be a string or object", fullKey)
		}
	}
	return nil
}
