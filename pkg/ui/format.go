package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/bjartek/aether/pkg/aether"
	"github.com/muesli/reflow/wordwrap"
)

// FormatFieldValue formats a field value with proper indentation for container types.
// This provides clean, readable formatting for maps and arrays while keeping
// simple values on a single line.
// For container types, it returns a string starting with \n followed by indented content.
// baseIndent is the indentation level where this value should start (e.g., "  " for 2 spaces).
// registry and showRawAddresses are optional - pass nil and false to disable address mapping.
func FormatFieldValue(val interface{}, baseIndent string) string {
	return FormatFieldValueWithRegistry(val, baseIndent, nil, false, 0)
}

// FormatFieldValueWithRegistry formats a field value with account registry support for address mapping.
// maxWidth specifies the maximum width for text wrapping (0 = no wrapping).
func FormatFieldValueWithRegistry(val interface{}, baseIndent string, registry *aether.AccountRegistry, showRawAddresses bool, maxWidth int) string {
	// Check if this is a container type at the top level
	switch val.(type) {
	case map[string]interface{}, []interface{}:
		// Container type - format with base indentation and add newline prefix
		return "\n" + formatValue(val, baseIndent, registry, showRawAddresses, maxWidth)
	default:
		// Simple type - format without scientific notation
		return formatSimpleValue(val, registry, showRawAddresses, baseIndent, maxWidth)
	}
}

// formatSimpleValue formats a simple value, avoiding scientific notation for numbers
// and mapping addresses to friendly names if registry is provided.
// For strings, wraps text if maxWidth > 0.
func formatSimpleValue(val interface{}, registry *aether.AccountRegistry, showRawAddresses bool, indent string, maxWidth int) string {
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
	case string:
		// Check if this looks like a Flow address and map it if registry is available
		if !showRawAddresses && registry != nil && isFlowAddress(v) {
			name := registry.GetName(v)
			if name != v {
				// Return just the friendly name (consistent with other UI displays)
				return name
			}
		}
		// Wrap text if maxWidth is specified and string is long
		if maxWidth > 0 && len(v) > maxWidth-len(indent) {
			// Calculate available width accounting for indent
			availableWidth := maxWidth - len(indent)
			if availableWidth < 20 {
				availableWidth = 20 // Minimum width to avoid breaking too aggressively
			}
			
			// Wrap the text - wordwrap breaks on word boundaries
			// For strings without spaces (like hex), use simple character-based breaking
			if !strings.Contains(v, " ") {
				// No spaces - break by characters
				var lines []string
				remaining := v
				for len(remaining) > 0 {
					if len(remaining) <= availableWidth {
						lines = append(lines, remaining)
						break
					}
					lines = append(lines, remaining[:availableWidth])
					remaining = remaining[availableWidth:]
				}
				// Add indent to continuation lines
				for i := 1; i < len(lines); i++ {
					lines[i] = indent + lines[i]
				}
				return strings.Join(lines, "\n")
			}
			
			// Has spaces - use word wrapping
			wrapped := wordwrap.String(v, availableWidth)
			// Add indent to continuation lines
			lines := strings.Split(wrapped, "\n")
			for i := 1; i < len(lines); i++ {
				lines[i] = indent + lines[i]
			}
			return strings.Join(lines, "\n")
		}
		return v
	default:
		return fmt.Sprintf("%v", val)
	}
}

// isFlowAddress checks if a string looks like a Flow address (0x followed by hex)
func isFlowAddress(s string) bool {
	if len(s) < 3 {
		return false
	}
	if !strings.HasPrefix(s, "0x") {
		return false
	}
	// Check if the rest are hex characters
	hexPart := s[2:]
	for _, c := range hexPart {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return len(hexPart) > 0
}

// formatValue recursively formats a value with the given indentation
func formatValue(val interface{}, indent string, registry *aether.AccountRegistry, showRawAddresses bool, maxWidth int) string {
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
				formattedValue := formatValue(value, indent+"  ", registry, showRawAddresses, maxWidth)
				lines = append(lines, fmt.Sprintf("%s%s:", indent, key))
				lines = append(lines, formattedValue)
			default:
				// Simple type - format on same line
				// Calculate indent for this line (key + ": ")
				lineIndent := indent + key + ": "
				lines = append(lines, fmt.Sprintf("%s%s: %s", indent, key, formatSimpleValue(value, registry, showRawAddresses, lineIndent, maxWidth)))
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
				formattedItem := formatValue(item, indent+"  ", registry, showRawAddresses, maxWidth)
				lines = append(lines, fmt.Sprintf("%s- [%d]:", indent, i))
				lines = append(lines, formattedItem)
			default:
				// Simple type - format on same line
				lineIndent := indent + "- "
				lines = append(lines, fmt.Sprintf("%s- %s", indent, formatSimpleValue(item, registry, showRawAddresses, lineIndent, maxWidth)))
			}
		}
		return strings.Join(lines, "\n")

	default:
		// For simple types, use formatSimpleValue
		return fmt.Sprintf("%s%s", indent, formatSimpleValue(val, registry, showRawAddresses, indent, maxWidth))
	}
}
