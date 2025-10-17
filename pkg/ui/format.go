package ui

import (
	"fmt"
	"sort"
	"strings"
)

// FormatFieldValue formats a field value with proper indentation for container types.
// This provides clean, readable formatting for maps and arrays while keeping
// simple values on a single line.
// For container types, it returns a string starting with \n followed by indented content.
// baseIndent is the indentation level where this value should start (e.g., "  " for 2 spaces).
func FormatFieldValue(val interface{}, baseIndent string) string {
	// Check if this is a container type at the top level
	switch val.(type) {
	case map[string]interface{}, []interface{}:
		// Container type - format with base indentation and add newline prefix
		return "\n" + formatValue(val, baseIndent)
	default:
		// Simple type - format without scientific notation
		return formatSimpleValue(val)
	}
}

// formatSimpleValue formats a simple value, avoiding scientific notation for numbers
func formatSimpleValue(val interface{}) string {
	switch v := val.(type) {
	case float64:
		// Avoid scientific notation - use fixed-point notation
		// Check if it's a whole number
		if v == float64(int64(v)) {
			return fmt.Sprintf("%.0f", v)
		}
		// Use fixed-point with enough precision, then trim trailing zeros
		s := fmt.Sprintf("%.10f", v)
		s = strings.TrimRight(s, "0")
		s = strings.TrimRight(s, ".")
		return s
	case float32:
		// Avoid scientific notation
		if v == float32(int32(v)) {
			return fmt.Sprintf("%.0f", v)
		}
		s := fmt.Sprintf("%.10f", v)
		s = strings.TrimRight(s, "0")
		s = strings.TrimRight(s, ".")
		return s
	default:
		return fmt.Sprintf("%v", val)
	}
}

// formatValue recursively formats a value with the given indentation
func formatValue(val interface{}, indent string) string {
	switch v := val.(type) {
	case map[string]interface{}:
		if len(v) == 0 {
			return "{}"
		}
		// Sort keys for consistent ordering
		keys := make([]string, 0, len(v))
		for key := range v {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		var lines []string
		for _, key := range keys {
			value := v[key]
			// Check if value is a container type
			switch value.(type) {
			case map[string]interface{}, []interface{}:
				// Container type - format on new line with increased indent
				// The nested content gets the parent's indent + 2 spaces
				formattedValue := formatValue(value, indent+"  ")
				lines = append(lines, fmt.Sprintf("%s%s:", indent, key))
				lines = append(lines, formattedValue)
			default:
				// Simple type - format on same line
				lines = append(lines, fmt.Sprintf("%s%s: %s", indent, key, formatSimpleValue(value)))
			}
		}
		return strings.Join(lines, "\n")

	case []interface{}:
		if len(v) == 0 {
			return "[]"
		}
		var lines []string
		for i, item := range v {
			// Check if item is a container type
			switch item.(type) {
			case map[string]interface{}, []interface{}:
				// Container type - format on new line with increased indent
				formattedItem := formatValue(item, indent+"  ")
				lines = append(lines, fmt.Sprintf("%s- [%d]:", indent, i))
				lines = append(lines, formattedItem)
			default:
				// Simple type - format on same line
				lines = append(lines, fmt.Sprintf("%s- %s", indent, formatSimpleValue(item)))
			}
		}
		return strings.Join(lines, "\n")

	default:
		// For simple types, use formatSimpleValue
		return fmt.Sprintf("%s%s", indent, formatSimpleValue(val))
	}
}
