package ui

import (
	"fmt"

	"github.com/bjartek/aether/pkg/aether"
	"github.com/bjartek/aether/pkg/config"
	"github.com/bjartek/aether/pkg/splitview"
	"github.com/bjartek/overflow/v2"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rs/zerolog"
)

// TransactionsKeyMap defines keybindings for the transactions view
type TransactionsKeyMap struct {
	LineUp             key.Binding
	LineDown           key.Binding
	GotoTop            key.Binding
	GotoEnd            key.Binding
	ToggleFullDetail   key.Binding
	ToggleEventFields  key.Binding
	ToggleRawAddresses key.Binding
	Filter             key.Binding
	Save               key.Binding
}

// DefaultTransactionsKeyMap returns the default keybindings for transactions view
func DefaultTransactionsKeyMap() TransactionsKeyMap {
	return TransactionsKeyMap{
		LineUp: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("k/↑", "up"),
		),
		LineDown: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("j/↓", "down"),
		),
		GotoTop: key.NewBinding(
			key.WithKeys("g", "home"),
			key.WithHelp("g/home", "go to top"),
		),
		GotoEnd: key.NewBinding(
			key.WithKeys("G", "end"),
			key.WithHelp("G/end", "go to bottom"),
		),
		ToggleFullDetail: key.NewBinding(
			key.WithKeys("enter", " "),
			key.WithHelp("enter/space", "toggle detail"),
		),
		ToggleEventFields: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "toggle event fields"),
		),
		ToggleRawAddresses: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "toggle raw addresses"),
		),
		Filter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filter"),
		),
		Save: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "save transaction"),
		),
	}
}

// TransactionsView is the splitview-based implementation
type TransactionsView struct {
	sv               *splitview.SplitViewModel
	keys             TransactionsKeyMap
	width            int
	height           int
	overflow         *overflow.OverflowState
	accountRegistry  *aether.AccountRegistry
	showEventFields  bool
	showRawAddresses bool
	timeFormat       string                   // Time format from config
	transactions     []aether.TransactionData // Store original data for rebuilding
	savingMode       bool                     // Whether save dialog is active
	saveInput        textinput.Model          // Input for save filename
	saveError        string                   // Error message from last save attempt
	saveSuccess      string                   // Success message from last save
	logger           zerolog.Logger           // Debug logger
}

// NewTransactionsViewWithConfig creates a new v2 transactions view based on splitview
func NewTransactionsViewWithConfig(cfg *config.Config, logger zerolog.Logger) *TransactionsView {
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

	// Initialize save input
	saveInput := textinput.New()
	saveInput.Placeholder = "filename (without .json)"
	saveInput.CharLimit = 50
	saveInput.Width = 40

	return &TransactionsView{
		sv:               sv,
		keys:             DefaultTransactionsKeyMap(),
		showEventFields:  cfg.UI.Defaults.ShowEventFields,
		showRawAddresses: cfg.UI.Defaults.ShowRawAddresses,
		timeFormat:       cfg.UI.Defaults.TimeFormat,
		saveInput:        saveInput,
		logger:           logger,
	}
}

// Init returns the init command for inner splitview
func (tv *TransactionsView) Init() tea.Cmd { return tv.sv.Init() }

// Update implements tea.Model interface - handles toggles then forwards to splitview
func (tv *TransactionsView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	tv.logger.Debug().Str("method", "Update").Interface("msgType", msg).Msg("TransactionsView.Update called")

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		tv.width = msg.Width
		tv.height = msg.Height

	case aether.BlockTransactionMsg:
		// Handle incoming transaction data
		tv.AddTransaction(msg.TransactionData)
		return tv, nil

	case aether.OverflowReadyMsg:
		// Set overflow and account registry when ready
		tv.SetOverflow(msg.Overflow)
		tv.SetAccountRegistry(msg.AccountRegistry)
		return tv, nil

	case tea.KeyMsg:
		// Handle save dialog input
		if tv.savingMode {
			switch {
			case msg.Type == tea.KeyEnter:
				// Save transaction
				if tv.saveInput.Value() != "" {
					filename := tv.saveInput.Value()
					tv.saveError = tv.saveTransaction(filename)
					if tv.saveError == "" {
						tv.saveSuccess = fmt.Sprintf("Transaction saved as '%s.json'", filename)
						tv.savingMode = false
						tv.saveInput.SetValue("")
						// Refresh to show success message
						tv.refreshCurrentRow()
						// Send message to refresh runner view
						return tv, func() tea.Msg { return RescanFilesMsg{} }
					}
				}
				return tv, nil

			case msg.Type == tea.KeyEsc:
				// Cancel save
				tv.savingMode = false
				tv.saveInput.SetValue("")
				tv.saveError = ""
				tv.saveSuccess = ""
				tv.refreshCurrentRow()
				return tv, nil

			default:
				// Update save input
				var cmd tea.Cmd
				tv.saveInput, cmd = tv.saveInput.Update(msg)
				tv.refreshCurrentRow()
				return tv, tea.Batch(cmd, InputHandled())
			}
		}

		// Handle toggle keys before forwarding to splitview
		switch {
		case key.Matches(msg, tv.keys.Save) && tv.sv.IsFullscreen():
			// Activate save dialog
			tv.savingMode = true
			tv.saveInput.Focus()
			tv.saveError = ""
			tv.saveSuccess = ""
			tv.refreshCurrentRow()
			return tv, nil

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
func (tv *TransactionsView) View() string {
	tv.logger.Debug().Str("method", "View").Msg("TransactionsView.View called")

	view := tv.sv.View()
	tv.logger.Debug().
		Str("method", "View").
		Str("component", "transactions").
		Int("parentWidth", tv.width).
		Int("parentHeight", tv.height).
		Int("svWidth", tv.sv.GetWidth()).
		Int("svHeight", tv.sv.GetHeight()).
		Int("svTableWidth", tv.sv.GetTableWidth()).
		Int("svDetailWidth", tv.sv.GetDetailWidth()).
		Int("viewLength", len(view)).
		Msg("TransactionsView.View rendered")
	return view
}

// Name implements TabbedModel interface
func (tv *TransactionsView) Name() string {
	return "Transactions"
}

// KeyMap implements TabbedModel interface
func (tv *TransactionsView) KeyMap() help.KeyMap {
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
func (tv *TransactionsView) FooterView() string {
	// No custom footer for transactions view
	return ""
}

// IsCapturingInput implements TabbedModel interface
func (tv *TransactionsView) IsCapturingInput() bool {
	// Capture input when in saving mode
	return tv.savingMode
}

// SetAccountRegistry sets the account registry for name resolution
func (tv *TransactionsView) SetAccountRegistry(registry *aether.AccountRegistry) {
	tv.accountRegistry = registry
}

// SetOverflow sets the overflow state for network information
func (tv *TransactionsView) SetOverflow(o *overflow.OverflowState) {
	tv.overflow = o
}

// AddTransaction accepts prebuilt TransactionData and converts it to a splitview row
func (tv *TransactionsView) AddTransaction(txData aether.TransactionData) {
	// Store transaction data for rebuilding
	tv.transactions = append(tv.transactions, txData)

	// Build and add row
	tv.addTransactionRow(txData)
}

// addTransactionRow builds a splitview row from transaction data
func (tv *TransactionsView) addTransactionRow(txData aether.TransactionData) {
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

func truncateHex(s string, startLen, endLen int) string {
	if len(s) <= startLen+endLen {
		return s
	}
	return s[:startLen] + "..." + s[len(s)-endLen:]
}

// refreshCurrentRow rebuilds only the current row's detail to reflect toggle changes
func (tv *TransactionsView) refreshCurrentRow() {
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

	// Append save dialog or success message if applicable
	if tv.savingMode {
		fieldStyle := lipgloss.NewStyle().Bold(true).Foreground(secondaryColor)
		content += "\n\n" + fieldStyle.Render("Save Transaction As:") + "\n"
		content += tv.saveInput.View() + "\n"
		if tv.saveError != "" {
			content += lipgloss.NewStyle().Foreground(errorColor).Render(tv.saveError) + "\n"
		}
		hintStyle := lipgloss.NewStyle().Foreground(mutedColor).Italic(true)
		// Get network from overflow for hint text
		network := "emulator"
		if tv.overflow != nil {
			network = tv.overflow.Network.Name
		}
		content += "\n" + hintStyle.Render(fmt.Sprintf("Will save as <name>.%s.cdc and <name>.json", network))
	} else if tv.saveSuccess != "" {
		content += "\n\n" + lipgloss.NewStyle().Foreground(successColor).Render(tv.saveSuccess) + "\n"
	}

	code := buildTransactionDetailCode(txData)

	// Update only this row in splitview
	tv.sv.UpdateRow(currentIdx, splitview.NewRowData(row).WithContent(content).WithCode(code))
}

// refreshAllRows rebuilds all rows to reflect toggle changes
func (tv *TransactionsView) refreshAllRows() {
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
