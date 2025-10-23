package ui

import (
	"fmt"
	"math/big"
	"sort"
	"strings"

	"github.com/bjartek/aether/pkg/aether"
	"github.com/charmbracelet/lipgloss"
)

// buildTransactionDetailContent assembles the non-code portion of the transaction detail text.
// It mirrors the formatting and styling used in renderTransactionDetailText, up to (and including)
// the "Script:" header, but does NOT append the script body. Callers should append the code
// returned by buildTransactionDetailCode.
func buildTransactionDetailContent(tx aether.TransactionData, registry *aether.AccountRegistry, showEventFields bool, showRaw bool) string {
	fieldStyle := lipgloss.NewStyle().Bold(true).Foreground(secondaryColor)
	valueStyleDetail := lipgloss.NewStyle().Foreground(accentColor)

	// Helper function to align fields
	renderField := func(label, value string) string {
		return fieldStyle.Render(fmt.Sprintf("%-12s", label+":")) + " " + valueStyleDetail.Render(value) + "\n"
	}

	var details strings.Builder
	// Title
	details.WriteString(fieldStyle.Render("Transaction Details") + "\n\n")

	// Basic fields
	details.WriteString(renderField("ID", tx.ID))
	details.WriteString(renderField("Block", fmt.Sprintf("%d", tx.BlockHeight)))
	details.WriteString(renderField("Block ID", tx.BlockID))
	details.WriteString(renderField("Status", tx.Status))
	details.WriteString(renderField("Index", fmt.Sprintf("%d", tx.Index)))
	details.WriteString(renderField("Gas Limit", fmt.Sprintf("%d", tx.GasLimit)))
	details.WriteString("\n")

	// Account table headers and values
	colWidth := 20
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(secondaryColor).Width(colWidth)
	valueStyle := lipgloss.NewStyle().Foreground(accentColor).Width(colWidth)

	// Headers
	details.WriteString(headerStyle.Render("Proposer"))
	details.WriteString(headerStyle.Render("Payer"))
	details.WriteString(fieldStyle.Render("Authorizers"))
	details.WriteString("\n")

	// Format addresses with friendly names when available
	proposerDisplay := tx.Proposer
	if !showRaw && registry != nil {
		proposerDisplay = registry.GetName(tx.Proposer)
	}

	payerDisplay := tx.Payer
	if !showRaw && registry != nil {
		payerDisplay = registry.GetName(tx.Payer)
	}

	for i, auth := range tx.Authorizers {
		var authDisplay string
		if !showRaw && registry != nil {
			authDisplay = registry.GetName(auth)
		} else {
			authDisplay = auth
		}

		if i == 0 {
			// First line with proposer, payer, and first authorizer
			details.WriteString(valueStyle.Render(proposerDisplay))
			details.WriteString(valueStyle.Render(payerDisplay))
			details.WriteString(valueStyleDetail.Render(authDisplay))
			details.WriteString("\n")
		} else {
			// Additional authorizers aligned under the authorizer column
			details.WriteString(valueStyle.Render(""))
			details.WriteString(valueStyle.Render(""))
			details.WriteString(valueStyleDetail.Render(authDisplay))
			details.WriteString("\n")
		}
	}

	details.WriteString("\n")

	// Error section
	if tx.Error != "" {
		details.WriteString(fieldStyle.Render(fmt.Sprintf("%-12s", "Error:")) + " " + lipgloss.NewStyle().Foreground(errorColor).Render(tx.Error) + "\n\n")
	}

	// Events section
	if len(tx.Events) > 0 {
		details.WriteString(fieldStyle.Render(fmt.Sprintf("%-12s", fmt.Sprintf("Events (%d):", len(tx.Events)))) + "\n")
		for i, event := range tx.Events {
			details.WriteString(fmt.Sprintf("  %d. %s\n", i+1, fieldStyle.Render(event.Name)))

			// Display event fields if enabled
			if showEventFields && len(event.Fields) > 0 {
				// Sort keys for consistent order
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

				// Display fields aligned on ':'
				for _, key := range keys {
					val := event.Fields[key]
					paddedKey := fmt.Sprintf("%-*s", maxKeyLen, key)
					valStr := FormatFieldValueWithRegistry(val, "       ", registry, showRaw, 0)
					details.WriteString(fmt.Sprintf("     %s: %s\n",
						valueStyleDetail.Render(paddedKey),
						valueStyleDetail.Render(valStr)))
				}
			}
		}
		details.WriteString("\n")
	}

	// EVM transactions section
	if len(tx.EVMTransactions) > 0 {
		details.WriteString(fieldStyle.Render(fmt.Sprintf("%-12s", fmt.Sprintf("EVM Transactions (%d):", len(tx.EVMTransactions)))) + "\n")
		for i, evmTx := range tx.EVMTransactions {
			details.WriteString(fmt.Sprintf("  %d. %s\n", i+1, valueStyleDetail.Render(evmTx.Transaction.Hash().Hex())))
			details.WriteString(fmt.Sprintf("     Type:       %d\n", evmTx.Payload.TransactionType))
			details.WriteString(fmt.Sprintf("     Gas Used:   %d\n", evmTx.Receipt.GasUsed))

			if from, err := evmTx.Transaction.From(); err == nil {
				details.WriteString(fmt.Sprintf("     From:       %s\n", from.Hex()))
			}
			if to := evmTx.Transaction.To(); to != nil {
				details.WriteString(fmt.Sprintf("     To:         %s\n", to.Hex()))
			}

			// Display value if non-zero
			if value := evmTx.Transaction.Value(); value != nil && value.Sign() > 0 {
				weiBig := new(big.Float).SetInt(value)
				divisor := new(big.Float).SetFloat64(1e18)
				flowValue := new(big.Float).Quo(weiBig, divisor)
				details.WriteString(fmt.Sprintf("     Value:      %s FLOW\n", flowValue.Text('f', 6)))
			}

			if evmTx.Payload.ErrorCode != 0 {
				details.WriteString(fmt.Sprintf("     Error:      %s\n",
					lipgloss.NewStyle().Foreground(errorColor).Render(evmTx.Payload.ErrorMessage)))
			} else {
				details.WriteString(fmt.Sprintf("     Status:     %s\n",
					lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render("Success")))
			}

			// Logs if any
			if len(evmTx.Receipt.Logs) > 0 {
				details.WriteString(fmt.Sprintf("     Logs:       %d\n", len(evmTx.Receipt.Logs)))
				for logIdx, log := range evmTx.Receipt.Logs {
					details.WriteString(fmt.Sprintf("       %d. Address: %s\n", logIdx+1, log.Address.Hex()))
					if len(log.Topics) > 0 {
						details.WriteString(fmt.Sprintf("          Topics: %d\n", len(log.Topics)))
						for topicIdx, topic := range log.Topics {
							details.WriteString(fmt.Sprintf("            %d: %s\n", topicIdx, topic.Hex()))
						}
					}
					if len(log.Data) > 0 {
						dataHex := fmt.Sprintf("0x%x", log.Data)
						if len(dataHex) > 66 {
							dataHex = dataHex[:66] + "..."
						}
						details.WriteString(fmt.Sprintf("          Data: %s\n", dataHex))
					}
				}
			}
		}
		details.WriteString("\n")
	}

	// Arguments section
	if len(tx.Arguments) > 0 {
		details.WriteString(fieldStyle.Render(fmt.Sprintf("%-12s", fmt.Sprintf("Arguments (%d):", len(tx.Arguments)))) + "\n")

		// Longest argument name for alignment
		maxNameLen := 0
		for _, arg := range tx.Arguments {
			if len(arg.Name) > maxNameLen {
				maxNameLen = len(arg.Name)
			}
		}

		for _, arg := range tx.Arguments {
			paddedName := fmt.Sprintf("%-*s", maxNameLen, arg.Name)
			valStr := FormatFieldValueWithRegistry(arg.Value, "    ", registry, showRaw, 0)
			details.WriteString(fmt.Sprintf("  %s: %s\n",
				valueStyleDetail.Render(paddedName),
				valueStyleDetail.Render(valStr)))
		}
		details.WriteString("\n")
	}

	// Script header (code body appended by caller)
	if tx.Script != "" {
		details.WriteString(fieldStyle.Render(fmt.Sprintf("%-12s", "Script:")) + "\n")
	}

	return details.String()
}

// buildTransactionDetailCode returns the script body (highlighted when available) with trailing newline.
func buildTransactionDetailCode(tx aether.TransactionData) string {
	if tx.Script == "" {
		return ""
	}
	// Prefer highlighted script when present
	scriptToShow := tx.HighlightedScript
	if scriptToShow == "" {
		scriptToShow = tx.Script
	}
	return scriptToShow + "\n"
}
