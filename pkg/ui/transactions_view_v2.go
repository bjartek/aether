package ui

import (
	"fmt"

	"github.com/bjartek/aether/pkg/aether"
	"github.com/bjartek/aether/pkg/config"
	"github.com/bjartek/aether/pkg/splitview"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
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
	timeFormat       string                   // Time format from config
	transactions     []aether.TransactionData // Store original data for rebuilding
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
		timeFormat:       cfg.UI.Defaults.TimeFormat,
	}
}

// Init returns the init command for inner splitview
func (tv *TransactionsViewV2) Init() tea.Cmd { return tv.sv.Init() }

// Update implements tea.Model interface - handles toggles then forwards to splitview
func (tv *TransactionsViewV2) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		tv.width = msg.Width
		tv.height = msg.Height
	
	case tea.KeyMsg:
		// Handle toggle keys before forwarding to splitview
		switch {
		case key.Matches(msg, tv.keys.ToggleEventFields):
			tv.showEventFields = !tv.showEventFields
			// Refresh current row to update detail view
			tv.refreshCurrentRow()
			return tv, InputHandled()
		case key.Matches(msg, tv.keys.ToggleRawAddresses):
			tv.showRawAddresses = !tv.showRawAddresses
			// Refresh all rows to update table and detail
			tv.refreshAllRows()
			return tv, InputHandled()
		}
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
	return transactionsKeyMapAdapter{
		splitviewKeys: tv.sv.KeyMap(),
		toggleKeys:    tv.keys,
	}
}

// transactionsKeyMapAdapter combines splitview and toggle keys
type transactionsKeyMapAdapter struct {
	splitviewKeys help.KeyMap
	toggleKeys    TransactionsKeyMap
}

func (k transactionsKeyMapAdapter) ShortHelp() []key.Binding {
	// Combine splitview short help with toggle keys
	svHelp := k.splitviewKeys.ShortHelp()
	return append(svHelp, k.toggleKeys.ToggleEventFields, k.toggleKeys.ToggleRawAddresses)
}

func (k transactionsKeyMapAdapter) FullHelp() [][]key.Binding {
	// Get splitview full help
	svHelp := k.splitviewKeys.FullHelp()
	
	// Add toggle keys as a new row
	toggleRow := []key.Binding{
		k.toggleKeys.ToggleEventFields,
		k.toggleKeys.ToggleRawAddresses,
	}
	
	return append(svHelp, toggleRow)
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

// SetAccountRegistry sets the account registry for friendly name resolution
func (tv *TransactionsViewV2) SetAccountRegistry(registry *aether.AccountRegistry) {
	tv.accountRegistry = registry
}

// AddTransaction accepts prebuilt TransactionData and converts it to a splitview row
func (tv *TransactionsViewV2) AddTransaction(txData aether.TransactionData) {
	// Store transaction data for rebuilding
	tv.transactions = append(tv.transactions, txData)
	
	// Build and add row
	tv.addTransactionRow(txData)
}

// addTransactionRow builds a splitview row from transaction data
func (tv *TransactionsViewV2) addTransactionRow(txData aether.TransactionData) {
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
		txData.Timestamp.Format(tv.timeFormat),
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

// refreshCurrentRow rebuilds only the current row's detail to reflect toggle changes
func (tv *TransactionsViewV2) refreshCurrentRow() {
	if len(tv.transactions) == 0 {
		return
	}
	
	// Get current cursor position
	currentIdx := tv.sv.GetCursor()
	if currentIdx < 0 || currentIdx >= len(tv.transactions) {
		return
	}
	
	// Get the transaction data for the current row
	txData := tv.transactions[currentIdx]
	
	// Rebuild just this row with updated toggle states
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
		txData.Timestamp.Format(tv.timeFormat),
		truncateHex(txData.ID, 3, 3),
		fmt.Sprintf("%d", txData.BlockHeight),
		authDisplay,
		string(txData.Type),
		txData.Status,
	}

	// Build detail content/code with current toggle states
	content := buildTransactionDetailContent(txData, tv.accountRegistry, tv.showEventFields, tv.showRawAddresses)
	code := buildTransactionDetailCode(txData)

	// Update only this row in splitview
	tv.sv.UpdateRow(currentIdx, splitview.NewRowData(row).WithContent(content).WithCode(code))
}

// refreshAllRows rebuilds all rows to reflect toggle changes
func (tv *TransactionsViewV2) refreshAllRows() {
	if len(tv.transactions) == 0 {
		return
	}
	
	// Clear existing rows and rebuild from stored transaction data
	tv.sv.SetRows([]splitview.RowData{})
	
	// Rebuild all rows with current toggle states
	for _, txData := range tv.transactions {
		tv.addTransactionRow(txData)
	}
}
