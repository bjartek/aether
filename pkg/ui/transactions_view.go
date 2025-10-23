package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bjartek/aether/pkg/aether"
	"github.com/bjartek/aether/pkg/chroma"
	"github.com/bjartek/aether/pkg/config"
	"github.com/bjartek/aether/pkg/flow"
	"github.com/bjartek/overflow/v2"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/onflow/flow-evm-gateway/models"
	"github.com/onflow/flow-go/fvm/evm/events"
)

// ArgumentData holds argument name and value for display
type ArgumentData struct {
	Name  string
	Value interface{} // Keep as interface{} for proper formatting
}

// EVMTransactionData wraps all data returned from decoding an EVM transaction event
type EVMTransactionData struct {
	Transaction models.Transaction
	Receipt     *models.Receipt
	Payload     *events.TransactionEventPayload
}

// TransactionType represents the type of transaction
type TransactionType string

const (
	TransactionTypeFlow  TransactionType = "flow"  // Only Flow/Cadence events
	TransactionTypeEVM   TransactionType = "evm"   // Only EVM events
	TransactionTypeMixed TransactionType = "mixed" // Both Flow and EVM events
)

// TransactionData holds transaction information for display
type TransactionData struct {
	ID                string
	BlockID           string
	BlockHeight       uint64
	Authorizers       []string // Can have multiple authorizers
	Status            string
	Proposer          string
	Payer             string
	GasLimit          uint64
	Script            string // Raw script code
	HighlightedScript string // Syntax-highlighted script with ANSI colors
	Arguments         []ArgumentData
	Events            []overflow.OverflowEvent
	EVMTransactions   []EVMTransactionData // Decoded EVM transactions
	Type              TransactionType      // Transaction type (flow/evm/mixed)
	Error             string
	Timestamp         time.Time
	Index             int
}

// TransactionMsg is sent when a new transaction is received
type TransactionMsg struct {
	Transaction TransactionData
}

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

// TransactionsView manages the transactions table and detail display
type TransactionsView struct {
	table               table.Model
	detailViewport      viewport.Model // For full detail mode
	splitDetailViewport viewport.Model // For split view detail panel
	filterInput         textinput.Model
	saveInput           textinput.Model // Input for save filename
	keys                TransactionsKeyMap
	ready               bool
	transactions        []TransactionData
	filteredTxs         []TransactionData
	maxTxs              int
	width               int
	height              int
	tableWidthPercent   int    // Configurable table width percentage
	detailWidthPercent  int    // Configurable detail width percentage
	fullDetailMode      bool   // Toggle between split and full-screen detail view
	showEventFields     bool   // Toggle showing event field details
	showRawAddresses    bool   // Toggle showing raw addresses vs friendly names
	filterMode          bool   // Whether filter input is active
	filterText          string // Current filter text
	savingMode          bool   // Whether save dialog is active
	saveError           string // Error message from last save attempt
	saveSuccess         string // Success message from last save
	accountRegistry     *aether.AccountRegistry
}

// NewTransactionsView creates a new transactions view with default settings
func NewTransactionsView() *TransactionsView {
	return NewTransactionsViewWithConfig(nil)
}

// NewTransactionsViewWithConfig creates a new transactions view with configuration
func NewTransactionsViewWithConfig(cfg *config.Config) *TransactionsView {
	columns := []table.Column{
		{Title: "Time", Width: 8},  // Execution time
		{Title: "ID", Width: 9},    // Truncated hex (first 3 + ... + last 3)
		{Title: "Block", Width: 5}, // Block numbers
		{Title: "Auth", Width: 18}, // Authorizer
		{Title: "Type", Width: 5},  // Transaction type
		{Title: "Status", Width: 8},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(10),
	)

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

	t.SetStyles(s)

	// Create viewport for full detail mode
	vp := viewport.New(0, 0)
	vp.Style = lipgloss.NewStyle()

	// Create viewport for split view detail panel
	splitVp := viewport.New(0, 0)
	splitVp.Style = lipgloss.NewStyle()

	// Create filter input
	filterInput := textinput.New()
	filterInput.Placeholder = "Filter by authorizer name..."
	if cfg != nil {
		filterInput.CharLimit = cfg.UI.Filter.CharLimit
		filterInput.Width = cfg.UI.Filter.Width
	} else {
		filterInput.CharLimit = 50
		filterInput.Width = 50
	}

	// Create save input
	saveInput := textinput.New()
	saveInput.Placeholder = "transaction-name"
	if cfg != nil {
		saveInput.CharLimit = cfg.UI.Save.FilenameCharLimit
		saveInput.Width = cfg.UI.Save.DialogWidth
	} else {
		saveInput.CharLimit = 50
		saveInput.Width = 40
	}

	// Get max transactions from config or use default
	maxTransactions := 10000
	if cfg != nil {
		maxTransactions = cfg.UI.History.MaxTransactions
	}

	// Get default display modes and layout from config
	// Use config defaults, or fallback to DefaultConfig if no config provided
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	showEventFields := cfg.UI.Defaults.ShowEventFields
	showRawAddresses := cfg.UI.Defaults.ShowRawAddresses
	fullDetailMode := cfg.UI.Defaults.FullDetailMode
	tableWidthPercent := cfg.UI.Layout.Transactions.TableWidthPercent
	detailWidthPercent := cfg.UI.Layout.Transactions.DetailWidthPercent

	return &TransactionsView{
		table:               t,
		detailViewport:      vp,
		splitDetailViewport: splitVp,
		filterInput:         filterInput,
		saveInput:           saveInput,
		keys:                DefaultTransactionsKeyMap(),
		ready:               false,
		transactions:        make([]TransactionData, 0, maxTransactions),
		filteredTxs:         make([]TransactionData, 0),
		maxTxs:              maxTransactions,
		tableWidthPercent:   tableWidthPercent,
		detailWidthPercent:  detailWidthPercent,
		fullDetailMode:      fullDetailMode,
		showEventFields:     showEventFields,
		showRawAddresses:    showRawAddresses,
		filterMode:          false,
		filterText:          "",
		savingMode:          false,
	}
}

func (tv *TransactionsView) Init() tea.Cmd {
	return nil
}

// truncateHex truncates a hex string to show start and end
func truncateHex(s string, startLen, endLen int) string {
	if len(s) <= startLen+endLen {
		return s
	}
	return s[:startLen] + "..." + s[len(s)-endLen:]
}

// AddTransaction adds a new transaction from an OverflowTransaction
func (tv *TransactionsView) AddTransaction(blockHeight uint64, blockID string, ot overflow.OverflowTransaction, registry *aether.AccountRegistry) {
	// Store registry for use in rendering
	if registry != nil {
		tv.accountRegistry = registry
	}

	// Extract all authorizers
	authorizers := ot.Authorizers
	if len(authorizers) == 0 {
		authorizers = []string{"N/A"}
	}

	// Extract proposer and payer
	proposer := "N/A"
	if ot.ProposalKey.Address.String() != "" {
		proposer = fmt.Sprintf("0x%s", ot.ProposalKey.Address.Hex())
	}

	payer := "N/A"
	if ot.Payer != "" {
		payer = ot.Payer
	}
	// Determine status
	status := "Unknown"
	if ot.Error != nil {
		status = "Failed"
	} else {
		status = ot.Status
	}

	// Store full script - user can scroll if needed
	script := string(ot.Script)
	highlightedScript := chroma.HighlightCadence(script)

	// Format arguments as structured data
	args := make([]ArgumentData, 0, len(ot.Arguments))
	for i, arg := range ot.Arguments {
		// Use the key field as the argument name, fallback to index if not available
		name := arg.Key
		if name == "" {
			name = fmt.Sprintf("argument%d", i)
		}
		argData := ArgumentData{
			Name:  name,
			Value: arg.Value, // Keep as interface{} for proper formatting
		}
		args = append(args, argData)
	}

	// Create error message
	errMsg := ""
	if ot.Error != nil {
		errMsg = ot.Error.Error()
	}

	// Store events directly
	events := ot.Events

	// Detect and decode EVM transactions from events
	evmTransactions := make([]EVMTransactionData, 0)
	hasEVMEvents := false
	hasNonEVMEvents := false

	for _, event := range events {
		// Check if this is an EVM.TransactionExecuted event
		if strings.Contains(event.Name, "EVM.TransactionExecuted") {
			hasEVMEvents = true
			tx, receipt, payload, err := models.DecodeTransactionEvent(event.RawEvent)
			if err != nil {
				// Skip events that fail to decode
				continue
			}
			evmTx := EVMTransactionData{
				Transaction: tx,
				Receipt:     receipt,
				Payload:     payload,
			}
			evmTransactions = append(evmTransactions, evmTx)
		} else {
			hasNonEVMEvents = true
		}
	}

	// Determine transaction type
	txType := TransactionTypeFlow // Default to flow
	if hasEVMEvents && !hasNonEVMEvents {
		txType = TransactionTypeEVM
	} else if hasEVMEvents && hasNonEVMEvents {
		txType = TransactionTypeMixed
	}

	txData := TransactionData{
		ID:                ot.Id,
		BlockID:           blockID,
		BlockHeight:       blockHeight,
		Authorizers:       authorizers,
		Status:            status,
		Proposer:          proposer,
		Payer:             payer,
		GasLimit:          ot.GasLimit,
		Script:            script,
		HighlightedScript: highlightedScript,
		Arguments:         args,
		Events:            events,
		EVMTransactions:   evmTransactions,
		Type:              txType,
		Error:             errMsg,
		Timestamp:         time.Now(),
		Index:             ot.TransactionIndex,
	}

	tv.transactions = append(tv.transactions, txData)

	// No pre-rendering - render fresh on demand

	// Keep only the last maxTxs transactions
	if len(tv.transactions) > tv.maxTxs {
		tv.transactions = tv.transactions[len(tv.transactions)-tv.maxTxs:]
	}

	tv.refreshTable()
}

// updateDetailViewport updates the viewport content with current transaction details
func (tv *TransactionsView) updateDetailViewport() {
	// Don't lock here - this is called from locked contexts
	if len(tv.transactions) == 0 {
		tv.detailViewport.SetContent("")
		return
	}

	selectedIdx := tv.table.Cursor()
	if selectedIdx >= 0 && selectedIdx < len(tv.transactions) {
		// Don't update if viewport isn't ready or sized
		if tv.detailViewport.Width == 0 || tv.detailViewport.Height == 0 {
			return
		}
		// Render fresh
		content := tv.renderTransactionDetailText(tv.transactions[selectedIdx])
		tv.detailViewport.SetContent(content)
		tv.detailViewport.GotoTop() // Always start at top
	}
}

// applyFilter filters transactions based on current filter text
func (tv *TransactionsView) applyFilter() {
	if tv.filterText == "" {
		// No filter, show all transactions
		tv.filteredTxs = tv.transactions
	} else {
		// Filter by authorizer friendly name
		tv.filteredTxs = make([]TransactionData, 0)
		filterLower := strings.ToLower(tv.filterText)

		for _, tx := range tv.transactions {
			for _, authAddr := range tx.Authorizers {
				// Get friendly name if available
				name := authAddr
				if tv.accountRegistry != nil {
					name = tv.accountRegistry.GetName(authAddr)
				}

				// Check if name matches filter
				if strings.Contains(strings.ToLower(name), filterLower) {
					tv.filteredTxs = append(tv.filteredTxs, tx)
					break // Only add transaction once even if multiple authorizers match
				}
			}
		}
	}
}

// refreshTable updates the table rows from transactions
func (tv *TransactionsView) refreshTable() {
	// Use filtered list if filter is active
	txList := tv.transactions
	if tv.filterText != "" {
		txList = tv.filteredTxs
	}

	rows := make([]table.Row, len(txList))
	for i, tx := range txList {
		// Show first authorizer in table with friendly name if available
		authDisplay := "N/A"
		if len(tx.Authorizers) > 0 {
			addr := tx.Authorizers[0]
			if tv.showRawAddresses {
				// Always show truncated address
				authDisplay = truncateHex(addr, 6, 4)
			} else if tv.accountRegistry != nil {
				name := tv.accountRegistry.GetName(addr)
				if name != addr {
					// Show friendly name
					authDisplay = name
				} else {
					// No friendly name, show truncated address
					authDisplay = truncateHex(addr, 6, 4)
				}
			} else {
				authDisplay = truncateHex(addr, 6, 4)
			}

			// Add count if multiple authorizers
			if len(tx.Authorizers) > 1 {
				authDisplay += fmt.Sprintf(" +%d", len(tx.Authorizers)-1)
			}
		}

		rows[i] = table.Row{
			tx.Timestamp.Format("15:04:05"), // Show time only
			truncateHex(tx.ID, 3, 3),        // Show first 3 and last 3 of ID
			fmt.Sprintf("%d", tx.BlockHeight),
			authDisplay,     // Show friendly name or truncated address
			string(tx.Type), // Transaction type (flow/evm/mixed)
			tx.Status,
		}
	}
	tv.table.SetRows(rows)
}

// Update handles messages for the transactions view
func (tv *TransactionsView) Update(msg tea.Msg, width, height int) tea.Cmd {
	tv.width = width
	tv.height = height

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if !tv.ready {
			tv.ready = true
		}
		// Split width using configured percentages
		tableWidth := int(float64(width) * float64(tv.tableWidthPercent) / 100.0)
		detailWidth := max(10, width-tableWidth-2)
		tv.table.SetWidth(tableWidth)
		tv.table.SetHeight(height)

		// Update viewport size for full detail mode
		// Calculate hint text height dynamically
		hint := lipgloss.NewStyle().
			Foreground(mutedColor).
			Render("Press Tab or Esc to return to table view | j/k to scroll")
		hintHeight := lipgloss.Height(hint) + 2 // +2 for spacing
		tv.detailViewport.Width = width
		tv.detailViewport.Height = height - hintHeight

		// Update split view detail viewport size
		tv.splitDetailViewport.Width = detailWidth
		tv.splitDetailViewport.Height = height

	case tea.KeyMsg:
		// Handle save mode
		if tv.savingMode {
			switch msg.String() {
			case "esc":
				// Cancel save
				tv.savingMode = false
				tv.saveInput.SetValue("")
				tv.saveInput.Blur()
				tv.saveError = ""
				tv.saveSuccess = ""
				return nil
			case "enter":
				// Perform save
				filename := tv.saveInput.Value()
				if filename == "" {
					tv.saveError = "Filename cannot be empty"
					return nil
				}

				// Get selected transaction from the currently displayed list
				selectedIdx := tv.table.Cursor()

				// Use the same logic as refreshTable - check which list is displayed
				txList := tv.transactions
				if tv.filterText != "" {
					txList = tv.filteredTxs
				}

				if selectedIdx < 0 || selectedIdx >= len(txList) {
					tv.saveError = "No transaction selected"
					return nil
				}
				tx := txList[selectedIdx]

				// Perform save
				err := tv.saveTransaction(filename, tx)
				if err != nil {
					tv.saveError = err.Error()
					tv.saveSuccess = ""
					return nil
				}

				// Success - show message and close dialog
				tv.saveSuccess = fmt.Sprintf("Saved %s.emulator.cdc and %s.json", filename, filename)
				tv.savingMode = false
				tv.saveInput.SetValue("")
				tv.saveInput.Blur()
				tv.saveError = ""
				return nil
			default:
				// Pass input to save textinput
				var cmd tea.Cmd
				tv.saveInput, cmd = tv.saveInput.Update(msg)
				return cmd
			}
		}

		// Handle filter mode
		if tv.filterMode {
			switch {
			case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
				// Exit filter mode and clear filter
				tv.filterMode = false
				tv.filterText = ""
				tv.filterInput.SetValue("")
				tv.applyFilter()
				tv.refreshTable()
				return nil
			case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
				// Apply filter and exit filter mode
				tv.filterMode = false
				tv.filterText = tv.filterInput.Value()
				tv.applyFilter()
				tv.refreshTable()
				return nil
			default:
				// Pass input to filter textinput
				var cmd tea.Cmd
				tv.filterInput, cmd = tv.filterInput.Update(msg)
				// Update filter in real-time
				tv.filterText = tv.filterInput.Value()
				tv.applyFilter()
				tv.refreshTable()
				return cmd
			}
		}

		// Handle filter activation
		if key.Matches(msg, tv.keys.Filter) {
			tv.filterMode = true
			tv.filterInput.Focus()
			return textinput.Blink
		}

		// Handle toggle full detail view
		if key.Matches(msg, tv.keys.ToggleFullDetail) {
			wasFullMode := tv.fullDetailMode
			tv.fullDetailMode = !tv.fullDetailMode
			// Update viewport content when entering full detail mode
			if !wasFullMode && tv.fullDetailMode {
				tv.updateDetailViewport()
			}
			return nil
		}

		// Handle Esc to exit full detail view
		if tv.fullDetailMode && key.Matches(msg, key.NewBinding(key.WithKeys("esc"))) {
			tv.fullDetailMode = false
			return nil
		}

		// Handle toggle event fields
		if key.Matches(msg, tv.keys.ToggleEventFields) {
			tv.showEventFields = !tv.showEventFields
			// Refresh full detail viewport if needed
			if tv.fullDetailMode {
				tv.updateDetailViewport()
			}
			return nil
		}

		// Handle toggle raw addresses
		if key.Matches(msg, tv.keys.ToggleRawAddresses) {
			tv.showRawAddresses = !tv.showRawAddresses
			// Refresh table and full detail viewport if needed
			tv.refreshTable()
			if tv.fullDetailMode {
				tv.updateDetailViewport()
			}
			return nil
		}

		// Handle save activation
		if key.Matches(msg, tv.keys.Save) {
			tv.savingMode = true
			tv.saveInput.Focus()
			tv.saveError = ""
			tv.saveSuccess = "" // Clear previous success message
			return textinput.Blink
		}

		// In full detail mode, pass keys to viewport for scrolling
		if tv.fullDetailMode {
			var cmd tea.Cmd
			tv.detailViewport, cmd = tv.detailViewport.Update(msg)
			return cmd
		} else {
			// Otherwise pass keys to table
			prevCursor := tv.table.Cursor()
			var cmd tea.Cmd
			tv.table, cmd = tv.table.Update(msg)

			// If cursor changed, update viewport content and reset scroll to top
			newCursor := tv.table.Cursor()
			if prevCursor != newCursor {
				tv.updateDetailViewport()
			}
			return cmd
		}
	}

	return nil
}

// saveTransaction saves a transaction to .cdc and .json files
func (tv *TransactionsView) saveTransaction(filename string, tx TransactionData) error {
	// Always use transactions directory, create if needed
	dir := "transactions"

	// Check if cadence/transactions exists instead
	if _, err := os.Stat("cadence/transactions"); err == nil {
		dir = "cadence/transactions"
	} else if _, err := os.Stat("transactions"); os.IsNotExist(err) {
		// Neither exists, create transactions
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Save .cdc file with network suffix (.emulator.cdc)
	cdcFilename := filename + ".emulator.cdc"
	cdcPath := filepath.Join(dir, cdcFilename)
	if err := os.WriteFile(cdcPath, []byte(tx.Script), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", cdcPath, err)
	}

	// Build JSON config with arguments (but empty signers)
	config := &flow.TransactionConfig{
		Name:      filename + ".emulator",
		Signers:   []string{}, // Leave empty as requested
		Arguments: make(map[string]interface{}),
	}

	// Populate arguments from transaction data
	for _, arg := range tx.Arguments {
		// Convert to string for JSON config
		config.Arguments[arg.Name] = fmt.Sprintf("%v", arg.Value)
	}

	// Save JSON config file
	jsonFilename := filename + ".json"
	jsonPath := filepath.Join(dir, jsonFilename)
	if err := flow.SaveTransactionConfig(jsonPath, config); err != nil {
		return fmt.Errorf("failed to write %s: %w", jsonPath, err)
	}

	return nil
}

// View renders the transactions view
func (tv *TransactionsView) View() string {
	if !tv.ready {
		return fmt.Sprintf("Loading transactions... tableWidth:%d detailWidth:%d", tv.tableWidthPercent, tv.detailWidthPercent)
	}

	if len(tv.transactions) == 0 {
		return lipgloss.NewStyle().
			Foreground(mutedColor).
			Render("Waiting for transactions...")
	}

	// Show save dialog if in saving mode
	if tv.savingMode {
		var content strings.Builder
		saveTitle := lipgloss.NewStyle().Bold(true).Foreground(secondaryColor).Render("Save Transaction")
		content.WriteString(saveTitle + "\n\n")

		content.WriteString("Enter filename (without extension):\n")
		content.WriteString(tv.saveInput.View() + "\n\n")

		if tv.saveError != "" {
			errorStyle := lipgloss.NewStyle().Foreground(errorColor)
			content.WriteString(errorStyle.Render("Error: "+tv.saveError) + "\n\n")
		}

		hintStyle := lipgloss.NewStyle().Foreground(mutedColor).Italic(true)
		content.WriteString(hintStyle.Render("Will save as <name>.emulator.cdc and <name>.json") + "\n")
		content.WriteString(hintStyle.Render("Press Enter to save, Esc to cancel") + "\n")

		return content.String()
	}

	// Full detail mode - show only the transaction detail in viewport
	if tv.fullDetailMode {
		hint := lipgloss.NewStyle().
			Foreground(mutedColor).
			Render("Press Tab or Esc to return to table view | j/k to scroll")
		return hint + "\n\n" + tv.detailViewport.View()
	}

	// Show filter input if in filter mode
	var filterBar string
	if tv.filterMode {
		filterStyle := lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true)
		filterBar = filterStyle.Render("Filter: ") + tv.filterInput.View() + "\n"
	} else if tv.filterText != "" {
		// Show active filter indicator
		filterStyle := lipgloss.NewStyle().
			Foreground(mutedColor)
		matchCount := len(tv.filteredTxs)
		filterBar = filterStyle.Render(fmt.Sprintf("Filter: '%s' (%d matches) • Press / to edit, Esc to clear", tv.filterText, matchCount)) + "\n"
	}

	// Show success message if present
	var successBar string
	if tv.saveSuccess != "" {
		successStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")). // Green color
			Bold(true)
		successBar = successStyle.Render("✓ "+tv.saveSuccess) + "\n"
	}

	// Split view mode - table on left, detail on right
	// Calculate widths using configured percentages
	tableWidth := int(float64(tv.width) * float64(tv.tableWidthPercent) / 100.0)

	// Update split detail viewport with current transaction
	selectedIdx := tv.table.Cursor()
	if selectedIdx >= 0 && selectedIdx < len(tv.transactions) {
		currentWidth := tv.splitDetailViewport.Width
		if currentWidth == 0 {
			currentWidth = 100 // Default
		}

		tx := tv.transactions[selectedIdx]

		// Just render fresh every time - no caching, no optimization
		content := tv.renderTransactionDetailText(tx)
		wrappedContent := lipgloss.NewStyle().Width(currentWidth).Render(content)

		tv.splitDetailViewport.SetContent(wrappedContent)
		tv.splitDetailViewport.GotoTop()
	} else {
		tv.splitDetailViewport.SetContent("No transaction selected")
		tv.splitDetailViewport.GotoTop()
	}

	// Style table
	tableView := lipgloss.NewStyle().
		Width(tableWidth).
		MaxHeight(tv.height).
		Render(tv.table.View())

	// Render split detail viewport (viewport itself handles width/height constraints)
	detailView := tv.splitDetailViewport.View()

	// Combine table and detail side by side
	mainView := lipgloss.JoinHorizontal(
		lipgloss.Top,
		tableView,
		detailView,
	)

	// Add filter bar and success message on top if present
	topBars := successBar + filterBar
	if topBars != "" {
		return topBars + mainView
	}
	return mainView
}

// renderTransactionDetailText renders transaction details as plain text (for viewport)
// Delegates to helpers to build content and code sections to enable splitview migration.
func (tv *TransactionsView) renderTransactionDetailText(tx TransactionData) string {
    content := buildTransactionDetailContent(tx, tv.accountRegistry, tv.showEventFields, tv.showRawAddresses)
    code := buildTransactionDetailCode(tx)
    return content + code
}


// Stop is a no-op for the transactions view
func (tv *TransactionsView) Stop() {
    // No cleanup needed
}
