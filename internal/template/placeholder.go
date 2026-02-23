package template

import (
	"regexp"
	"sort"

	"friday/internal/whatsapp"
)

var placeholderRegex = regexp.MustCompile(`\{\{(\w+)\}\}`)

// ExtractPlaceholders returns all unique placeholder names from the content, sorted.
func ExtractPlaceholders(content string) []string {
	matches := placeholderRegex.FindAllStringSubmatch(content, -1)

	seen := make(map[string]bool)
	for _, match := range matches {
		if len(match) > 1 {
			seen[match[1]] = true
		}
	}

	result := make([]string, 0, len(seen))
	for name := range seen {
		result = append(result, name)
	}

	sort.Strings(result)

	return result
}

// FillPlaceholders replaces {{name}} placeholders with values from the map.
// Returns the filled content and a list of placeholders that had no values.
func FillPlaceholders(content string, values map[string]string) (string, []string) {
	missingMap := make(map[string]bool)

	filled := placeholderRegex.ReplaceAllStringFunc(content, func(match string) string {
		name := match[2 : len(match)-2]

		if value, ok := values[name]; ok {
			return value
		}

		missingMap[name] = true
		return match
	})

	missing := make([]string, 0, len(missingMap))
	for name := range missingMap {
		missing = append(missing, name)
	}
	sort.Strings(missing)

	return filled, missing
}

type PreviewResult struct {
	Original            string   `json:"original"`
	Preview             string   `json:"preview"`
	PlaceholdersFound   []string `json:"placeholders_found"`
	PlaceholdersFilled  []string `json:"placeholders_filled"`
	PlaceholdersMissing []string `json:"placeholders_missing"`
}

// Preview generates a preview of the content with placeholders filled from values.
func Preview(content string, values map[string]string) PreviewResult {
	found := ExtractPlaceholders(content)
	filled, missing := FillPlaceholders(content, values)

	filledList := make([]string, 0, len(found)-len(missing))
	missingSet := make(map[string]bool)
	for _, m := range missing {
		missingSet[m] = true
	}
	for _, f := range found {
		if !missingSet[f] {
			filledList = append(filledList, f)
		}
	}

	return PreviewResult{
		Original:            content,
		Preview:             filled,
		PlaceholdersFound:   found,
		PlaceholdersFilled:  filledList,
		PlaceholdersMissing: missing,
	}
}

// GetBuiltInPlaceholders extracts standard fields from a WhatsApp contact.
// Available: {{phone}}, {{name}}, {{push_name}}, {{first_name}}, {{full_name}}
func GetBuiltInPlaceholders(contact *whatsapp.Contact) map[string]string {
	if contact == nil {
		return map[string]string{}
	}

	return map[string]string{
		"phone":      contact.Phone,
		"name":       contact.Name,
		"push_name":  contact.PushName,
		"first_name": contact.FirstName,
		"full_name":  contact.FullName,
	}
}

// MergePlaceholders combines multiple placeholder maps, with later maps taking precedence.
func MergePlaceholders(maps ...map[string]string) map[string]string {
	result := make(map[string]string)

	for _, m := range maps {
		for k, v := range m {
			if v != "" {
				result[k] = v
			}
		}
	}

	return result
}
