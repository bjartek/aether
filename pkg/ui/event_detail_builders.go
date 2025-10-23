package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/bjartek/aether/pkg/aether"
	"github.com/charmbracelet/lipgloss"
)

// buildEventDetailContent builds the detail content for an event
func buildEventDetailContent(event aether.EventData, accountRegistry *aether.AccountRegistry, showRawAddresses bool) string {
	fieldStyle := lipgloss.NewStyle().Bold(true).Foreground(secondaryColor)
	valueStyleDetail := lipgloss.NewStyle().Foreground(accentColor)

	renderField := func(label, value string) string {
		return fieldStyle.Render(fmt.Sprintf("%-18s", label+":")) + " " + valueStyleDetail.Render(value) + "\n"
	}

	var details strings.Builder
	details.WriteString(fieldStyle.Render("Event Details") + "\n\n")

	details.WriteString(renderField("Event Name", event.Name))
	details.WriteString(renderField("Block Height", fmt.Sprintf("%d", event.BlockHeight)))
	details.WriteString(renderField("Block ID", event.BlockID))
	details.WriteString(renderField("Transaction ID", event.TransactionID))
	details.WriteString(renderField("Transaction Index", fmt.Sprintf("%d", event.TransactionIndex)))
	details.WriteString(renderField("Event Index", fmt.Sprintf("%d", event.EventIndex)))
	details.WriteString("\n")

	if len(event.Fields) > 0 {
		details.WriteString(fieldStyle.Render(fmt.Sprintf("Fields (%d):", len(event.Fields))) + "\n")

		// Sort keys for consistent ordering
		keys := make([]string, 0, len(event.Fields))
		for key := range event.Fields {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		// Find the longest key for alignment
		maxKeyLen := 0
		for _, key := range keys {
			if len(key) > maxKeyLen {
				maxKeyLen = len(key)
			}
		}

		// Display fields aligned on :
		for _, key := range keys {
			val := event.Fields[key]
			paddedKey := fmt.Sprintf("%-*s", maxKeyLen, key)

			// For nested structures, use simple indent
			valStr := FormatFieldValueWithRegistry(val, "    ", accountRegistry, showRawAddresses, 0)
			details.WriteString(fmt.Sprintf("  %s: %s\n",
				valueStyleDetail.Render(paddedKey),
				valueStyleDetail.Render(valStr)))
		}
	} else {
		details.WriteString(fieldStyle.Render("No fields") + "\n")
	}

	return details.String()
}
