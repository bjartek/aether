package ui

import (
	"fmt"

	"github.com/bjartek/aether/pkg/aether"
	"github.com/bjartek/aether/pkg/config"
	"github.com/bjartek/aether/pkg/splitview"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TransactionsViewV2 is the splitview-based implementation
type TransactionsViewV2 struct {
	sv               *splitview.SplitViewModel
	keys             TransactionsKeyMap
	width            int
	height           int
	accountRegistry  *aether.AccountRegistry
	showEventFields  bool
	showRawAddresses bool
}

// NewTransactionsViewV2WithConfig creates a new v2 transactions view based on splitview
func NewTransactionsViewV2WithConfig(cfg *config.Config) *TransactionsViewV2 {
	// Fallback to defaults when cfg is nil
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	columns := []splitview.ColumnConfig{
		{Name: "Time", Width: 8},  // Execution time
		{Name: "ID", Width: 9},    // Truncated hex (first 3 + ... + last 3)
		{Name: "Block", Width: 5}, // Block numbers
		{Name: "Auth", Width: 18}, // Authorizer
		{Name: "Type", Width: 5},  // Transaction type
		{Name: "Status", Width: 8},
	}

	// Table styles (reuse v1 styles)
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(borderColor).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(base03).
		Background(solarYellow).
		Bold(false)

	// Build splitview with options
	sv := splitview.NewSplitView(
		columns,
		splitview.WithTableStyles(s),
		splitview.WithTableSplitPercent(float64(cfg.UI.Layout.Transactions.TableWidthPercent)/100.0),
	)

	return &TransactionsViewV2{
		sv:               sv,
		keys:             DefaultTransactionsKeyMap(),
		showEventFields:  cfg.UI.Defaults.ShowEventFields,
		showRawAddresses: cfg.UI.Defaults.ShowRawAddresses,
	}
}

// Init returns the init command for inner splitview
func (tv *TransactionsViewV2) Init() tea.Cmd { return tv.sv.Init() }

// Update implements tea.Model interface - forwards all messages to splitview
func (tv *TransactionsViewV2) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Capture dimensions for View() wrapping
	if wsMsg, ok := msg.(tea.WindowSizeMsg); ok {
		tv.width = wsMsg.Width
		tv.height = wsMsg.Height
	}
	
	_, cmd := tv.sv.Update(msg)
	return tv, cmd
}

// View delegates to splitview
func (tv *TransactionsViewV2) View() string {
	return tv.sv.View()
}

// Name implements TabbedModel interface
func (tv *TransactionsViewV2) Name() string {
	return "Transactions"
}

// KeyMap implements TabbedModel interface
func (tv *TransactionsViewV2) KeyMap() help.KeyMap {
	return tv.sv.KeyMap()
}

// FooterView implements TabbedModel interface
func (tv *TransactionsViewV2) FooterView() string {
	// No custom footer for transactions view
	return ""
}

// IsCapturingInput implements TabbedModel interface
func (tv *TransactionsViewV2) IsCapturingInput() bool {
	// Transactions view doesn't capture input
	return false
}

// AddTransaction accepts prebuilt TransactionData and converts it to a splitview row
func (tv *TransactionsViewV2) AddTransaction(txData aether.TransactionData) {
	// Build table row
	authDisplay := "N/A"
	if len(txData.Authorizers) > 0 {
		addr := txData.Authorizers[0]
		if tv.showRawAddresses || tv.accountRegistry == nil {
			authDisplay = truncateHex(addr, 6, 4)
		} else {
			name := tv.accountRegistry.GetName(addr)
			if name != addr {
				authDisplay = name
			} else {
				authDisplay = truncateHex(addr, 6, 4)
			}
		}
		if len(txData.Authorizers) > 1 {
			authDisplay += fmt.Sprintf(" +%d", len(txData.Authorizers)-1)
		}
	}

	row := table.Row{
		txData.Timestamp.Format("15:04:05"),
		truncateHex(txData.ID, 3, 3),
		fmt.Sprintf("%d", txData.BlockHeight),
		authDisplay,
		string(txData.Type),
		txData.Status,
	}

	// Build detail content/code using the extracted helpers
	content := buildTransactionDetailContent(txData, tv.accountRegistry, tv.showEventFields, tv.showRawAddresses)
	code := buildTransactionDetailCode(txData)

	// Add to splitview
	tv.sv.AddRow(splitview.NewRowData(row).WithContent(content).WithCode(code))
}
