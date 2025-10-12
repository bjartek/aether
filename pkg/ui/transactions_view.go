package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/bjartek/aether/pkg/aether"
	"github.com/bjartek/aether/pkg/chroma"
	"github.com/bjartek/aether/pkg/flow"
	"github.com/bjartek/overflow/v2"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ArgumentData holds argument name and value for display
type ArgumentData struct {
	Name  string
	Value string
}

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
	Error             string
	Timestamp         time.Time
	Index             int
	preRenderedDetail string // Cached detail text for performance
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
			key.WithKeys("tab"),
			key.WithHelp("tab", "toggle full detail"),
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
	mu                sync.RWMutex
	table             table.Model
	detailViewport    viewport.Model
	filterInput       textinput.Model
	saveInput         textinput.Model // Input for save filename
	keys              TransactionsKeyMap
	ready             bool
	transactions      []TransactionData
	filteredTxs       []TransactionData
	maxTxs            int
	width             int
	height            int
	fullDetailMode    bool   // Toggle between split and full-screen detail view
	showEventFields   bool   // Toggle showing event field details
	showRawAddresses  bool   // Toggle showing raw addresses vs friendly names
	filterMode        bool   // Whether filter input is active
	filterText        string // Current filter text
	savingMode        bool   // Whether save dialog is active
	saveError         string // Error message from last save attempt
	saveSuccess       string // Success message from last save
	accountRegistry   *aether.AccountRegistry
}

// NewTransactionsView creates a new transactions view
func NewTransactionsView() *TransactionsView {
	columns := []table.Column{
		{Title: "ID", Width: 20},         // Truncated hex (8...8)
		{Title: "Block", Width: 6},       // Slimmer for block numbers
		{Title: "Authorizer", Width: 30}, // Wider to show friendly names
		{Title: "Status", Width: 10},
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

	// Create viewport for detail view
	vp := viewport.New(0, 0)
	vp.Style = lipgloss.NewStyle()

	// Create filter input
	filterInput := textinput.New()
	filterInput.Placeholder = "Filter by authorizer name..."
	filterInput.CharLimit = 50
	filterInput.Width = 50

	// Create save input
	saveInput := textinput.New()
	saveInput.Placeholder = "transaction-name"
	saveInput.CharLimit = 50
	saveInput.Width = 40

	const maxTransactions = 10000
	
	return &TransactionsView{
		table:            t,
		detailViewport:   vp,
		filterInput:      filterInput,
		saveInput:        saveInput,
		keys:             DefaultTransactionsKeyMap(),
		ready:            false,
		transactions:     make([]TransactionData, 0, maxTransactions),
		filteredTxs:      make([]TransactionData, 0),
		maxTxs:           maxTransactions,
		fullDetailMode:   false,
		showEventFields:  false,
		showRawAddresses: false, // Show friendly names by default
		filterMode:       false,
		filterText:       "",
		savingMode:       false,
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

// formatEventFieldValue formats an event field value for display
func (tv *TransactionsView) formatEventFieldValue(val interface{}) string {
	switch v := val.(type) {
	case []uint8:
		// Convert uint8 array to hex string if in human-friendly mode
		if !tv.showRawAddresses && len(v) > 0 {
			return "0x" + fmt.Sprintf("%x", v)
		}
		return fmt.Sprintf("%v", v)
	case []interface{}:
		// Check if it's an array of numbers (likely a byte array)
		if len(v) > 0 && !tv.showRawAddresses {
			// Try to convert to bytes
			bytes := make([]byte, 0, len(v))
			isBytes := true
			for _, item := range v {
				switch num := item.(type) {
				case float64:
					if num >= 0 && num <= 255 && num == float64(int(num)) {
						bytes = append(bytes, byte(num))
					} else {
						isBytes = false
					}
				case int:
					if num >= 0 && num <= 255 {
						bytes = append(bytes, byte(num))
					} else {
						isBytes = false
					}
				default:
					isBytes = false
				}
				if !isBytes {
					break
				}
			}
			if isBytes && len(bytes) > 0 {
				return "0x" + fmt.Sprintf("%x", bytes)
			}
		}
		return fmt.Sprintf("%v", v)
	case map[string]interface{}:
		// Handle maps - format as key: value pairs with sorted keys
		if len(v) == 0 {
			return "{}"
		}
		// Sort keys for consistent ordering
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		var parts []string
		for _, k := range keys {
			// Recursively format map values
			formattedVal := tv.formatEventFieldValue(v[k])
			parts = append(parts, fmt.Sprintf("%s: %s", k, formattedVal))
		}
		return "{" + strings.Join(parts, ", ") + "}"
	case string:
		// Check if it's an address and format accordingly
		if !tv.showRawAddresses && tv.accountRegistry != nil && strings.HasPrefix(v, "0x") && len(v) == 18 {
			// For event fields, show only the friendly name
			return tv.accountRegistry.GetName(v)
		}
		return v
	default:
		valStr := fmt.Sprintf("%v", v)
		// Check if the string representation looks like an address
		if !tv.showRawAddresses && tv.accountRegistry != nil && strings.HasPrefix(valStr, "0x") && len(valStr) == 18 {
			// For event fields, show only the friendly name
			return tv.accountRegistry.GetName(valStr)
		}
		return valStr
	}
}

// AddTransaction adds a new transaction from an OverflowTransaction
func (tv *TransactionsView) AddTransaction(blockHeight uint64, blockID string, ot overflow.OverflowTransaction, registry *aether.AccountRegistry) {
	tv.mu.Lock()
	defer tv.mu.Unlock()

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

	// Debug: Log addresses if registry is available
	if registry != nil {
		// TODO: Remove debug logging once address mapping is working
		// This helps diagnose why friendly names aren't showing
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
			Value: fmt.Sprintf("%v", arg.Value),
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
		Error:             errMsg,
		Timestamp:         time.Now(),
		Index:             ot.TransactionIndex,
	}

	tv.transactions = append(tv.transactions, txData)

	// Pre-render asynchronously in background (don't block)
	go func() {
		detail := tv.renderTransactionDetailText(txData)
		tv.mu.Lock()
		// Find and update the transaction
		for i := range tv.transactions {
			if tv.transactions[i].ID == txData.ID {
				tv.transactions[i].preRenderedDetail = detail
				break
			}
		}
		tv.mu.Unlock()
	}()

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
		// Use pre-rendered detail text for instant display
		content := tv.transactions[selectedIdx].preRenderedDetail
		if content == "" {
			// Fallback if not pre-rendered
			return
		}
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
			truncateHex(tx.ID, 8, 8), // Show start and end of ID
			fmt.Sprintf("%d", tx.BlockHeight),
			authDisplay, // Show friendly name or truncated address
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
		// Split width: 40% table, 60% details (wider details pane)
		tableWidth := int(float64(width) * 0.4)
		tv.table.SetWidth(tableWidth)
		tv.table.SetHeight(height)

		// Update viewport size for full detail mode
		tv.detailViewport.Width = width
		tv.detailViewport.Height = height - 3 // Leave room for hint text

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
				tv.mu.RLock()
				
				// Use the same logic as refreshTable - check which list is displayed
				txList := tv.transactions
				if tv.filterText != "" {
					txList = tv.filteredTxs
				}
				
				if selectedIdx < 0 || selectedIdx >= len(txList) {
					tv.mu.RUnlock()
					tv.saveError = "No transaction selected"
					return nil
				}
				tx := txList[selectedIdx]
				tv.mu.RUnlock()
				
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
				tv.mu.Lock()
				tv.applyFilter()
				tv.refreshTable()
				tv.mu.Unlock()
				return nil
			case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
				// Apply filter and exit filter mode
				tv.filterMode = false
				tv.filterText = tv.filterInput.Value()
				tv.mu.Lock()
				tv.applyFilter()
				tv.refreshTable()
				tv.mu.Unlock()
				return nil
			default:
				// Pass input to filter textinput
				var cmd tea.Cmd
				tv.filterInput, cmd = tv.filterInput.Update(msg)
				// Update filter in real-time
				tv.filterText = tv.filterInput.Value()
				tv.mu.Lock()
				tv.applyFilter()
				tv.refreshTable()
				tv.mu.Unlock()
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
				tv.mu.RLock()
				tv.updateDetailViewport()
				tv.mu.RUnlock()
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
			// Need to re-render all transactions with new setting
			tv.mu.Lock()
			for i := range tv.transactions {
				tv.transactions[i].preRenderedDetail = tv.renderTransactionDetailText(tv.transactions[i])
			}
			if tv.fullDetailMode {
				tv.updateDetailViewport()
			}
			tv.mu.Unlock()
			return nil
		}

		// Handle toggle raw addresses
		if key.Matches(msg, tv.keys.ToggleRawAddresses) {
			tv.showRawAddresses = !tv.showRawAddresses
			// Need to re-render all transactions with new setting
			tv.mu.Lock()
			tv.refreshTable()
			for i := range tv.transactions {
				tv.transactions[i].preRenderedDetail = tv.renderTransactionDetailText(tv.transactions[i])
			}
			if tv.fullDetailMode {
				tv.updateDetailViewport()
			}
			tv.mu.Unlock()
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
			var cmd tea.Cmd
			tv.table, cmd = tv.table.Update(msg)
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
		Arguments: make(map[string]string),
	}

	// Populate arguments from transaction data
	for _, arg := range tx.Arguments {
		config.Arguments[arg.Name] = arg.Value
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
		return "Loading transactions..."
	}

	tv.mu.RLock()
	defer tv.mu.RUnlock()

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
		successBar = successStyle.Render("✓ " + tv.saveSuccess) + "\n"
	}

	// Split view mode - table on left, detail on right
	// Calculate widths: 40% table, 60% details (wider details pane)
	tableWidth := int(float64(tv.width) * 0.4)
	detailWidth := tv.width - tableWidth - 2

	selectedIdx := tv.table.Cursor()
	var detailView string
	if selectedIdx >= 0 && selectedIdx < len(tv.transactions) {
		detailView = tv.renderTransactionDetail(tv.transactions[selectedIdx], detailWidth, tv.height)
	} else {
		detailView = lipgloss.NewStyle().
			Width(detailWidth).
			Height(tv.height).
			Render("No transaction selected")
	}

	// Style table
	tableView := lipgloss.NewStyle().
		Width(tableWidth).
		Height(tv.height).
		Render(tv.table.View())

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
func (tv *TransactionsView) renderTransactionDetailText(tx TransactionData) string {
	fieldStyle := lipgloss.NewStyle().Bold(true).Foreground(secondaryColor)
	valueStyleDetail := lipgloss.NewStyle().Foreground(accentColor)

	// Helper function to align fields
	renderField := func(label, value string) string {
		return fieldStyle.Render(fmt.Sprintf("%-12s", label+":")) + " " + valueStyleDetail.Render(value) + "\n"
	}

	var details strings.Builder
	details.WriteString(fieldStyle.Render("Transaction Details") + "\n\n")

	details.WriteString(renderField("ID", tx.ID))
	details.WriteString(renderField("Block", fmt.Sprintf("%d", tx.BlockHeight)))
	details.WriteString(renderField("Block ID", tx.BlockID))
	details.WriteString(renderField("Status", tx.Status))
	details.WriteString(renderField("Index", fmt.Sprintf("%d", tx.Index)))
	details.WriteString(renderField("Gas Limit", fmt.Sprintf("%d", tx.GasLimit)))
	details.WriteString("\n")
	
	// Account table with fixed-width columns using lipgloss Width
	colWidth := 20
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(secondaryColor).Width(colWidth)
	valueStyle := lipgloss.NewStyle().Foreground(accentColor).Width(colWidth)
	
	// Headers
	details.WriteString(headerStyle.Render("Proposer"))
	details.WriteString(headerStyle.Render("Payer"))
	details.WriteString(fieldStyle.Render("Authorizers"))
	details.WriteString("\n")

	// Format addresses with friendly names (unless raw mode is enabled)
	proposerDisplay := tx.Proposer
	if !tv.showRawAddresses && tv.accountRegistry != nil {
		proposerDisplay = tv.accountRegistry.GetName(tx.Proposer)
	}

	payerDisplay := tx.Payer
	if !tv.showRawAddresses && tv.accountRegistry != nil {
		payerDisplay = tv.accountRegistry.GetName(tx.Payer)
	}

	for i, auth := range tx.Authorizers {
		var authDisplay string
		if !tv.showRawAddresses && tv.accountRegistry != nil {
			authDisplay = tv.accountRegistry.GetName(auth)
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

	if tx.Error != "" {
		details.WriteString(fieldStyle.Render(fmt.Sprintf("%-12s", "Error:")) + " " + lipgloss.NewStyle().Foreground(errorColor).Render(tx.Error) + "\n\n")
	}

	if len(tx.Events) > 0 {
		details.WriteString(fieldStyle.Render(fmt.Sprintf("%-12s", fmt.Sprintf("Events (%d):", len(tx.Events)))) + "\n")
		for i, event := range tx.Events {
			details.WriteString(fmt.Sprintf("  %d. %s\n", i+1, fieldStyle.Render(event.Name)))

			// Display event fields if any and if showEventFields is true
			if tv.showEventFields && len(event.Fields) > 0 {
				// TODO: Use event.FieldOrder when available in overflow library
				// For now, sort keys alphabetically for consistent ordering
				// (Go maps don't preserve insertion order)
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

					// Format value using helper function
					valStr := tv.formatEventFieldValue(val)

					details.WriteString(fmt.Sprintf("     %s: %s\n",
						valueStyleDetail.Render(paddedKey),
						valueStyleDetail.Render(valStr)))
				}
			}
		}
		details.WriteString("\n")
	}

	if len(tx.Arguments) > 0 {
		details.WriteString(fieldStyle.Render(fmt.Sprintf("%-12s", fmt.Sprintf("Arguments (%d):", len(tx.Arguments)))) + "\n")

		// Find the longest argument name for alignment
		maxNameLen := 0
		for _, arg := range tx.Arguments {
			if len(arg.Name) > maxNameLen {
				maxNameLen = len(arg.Name)
			}
		}

		// Display arguments aligned on :
		for _, arg := range tx.Arguments {
			paddedName := fmt.Sprintf("%-*s", maxNameLen, arg.Name)

			// Format value - check if it's an address and show friendly name
			valStr := arg.Value
			if !tv.showRawAddresses && tv.accountRegistry != nil && strings.HasPrefix(valStr, "0x") && len(valStr) == 18 {
				// Looks like an address, show only the friendly name
				valStr = tv.accountRegistry.GetName(valStr)
			}

			details.WriteString(fmt.Sprintf("  %s: %s\n",
				valueStyleDetail.Render(paddedName),
				valueStyleDetail.Render(valStr)))
		}
		details.WriteString("\n")
	}

	if tx.Script != "" {
		details.WriteString(fieldStyle.Render(fmt.Sprintf("%-12s", "Script:")) + "\n")
		// Show syntax-highlighted script if available, otherwise raw script
		scriptToShow := tx.HighlightedScript
		if scriptToShow == "" {
			scriptToShow = tx.Script
		}
		// Don't wrap in valueStyleDetail since highlighted code already has colors
		details.WriteString(scriptToShow + "\n")
	}

	return details.String()
}

// renderTransactionDetail renders the detailed view of a transaction (for split view)
// This version is height-constrained to prevent overflow
func (tv *TransactionsView) renderTransactionDetail(tx TransactionData, width, height int) string {
	detailStyle := lipgloss.NewStyle().
		Width(width).
		Height(height).
		Padding(1)

	// Render a condensed version that fits in the available height
	content := tv.renderTransactionDetailCondensed(tx, height-2) // -2 for padding
	return detailStyle.Render(content)
}

// renderTransactionDetailCondensed renders a height-aware condensed version
func (tv *TransactionsView) renderTransactionDetailCondensed(tx TransactionData, maxLines int) string {
	fieldStyle := lipgloss.NewStyle().Bold(true).Foreground(secondaryColor)
	valueStyleDetail := lipgloss.NewStyle().Foreground(accentColor)

	renderField := func(label, value string) string {
		return fieldStyle.Render(fmt.Sprintf("%-12s", label+":")) + " " + valueStyleDetail.Render(value) + "\n"
	}

	var details strings.Builder
	lineCount := 0

	// Title
	details.WriteString(fieldStyle.Render("Transaction Details") + "\n\n")
	lineCount += 2

	// Basic info (always show)
	details.WriteString(renderField("ID", tx.ID))
	details.WriteString(renderField("Block", fmt.Sprintf("%d", tx.BlockHeight)))
	details.WriteString(renderField("Status", tx.Status))
	lineCount += 3

	if lineCount+1 < maxLines {
		details.WriteString("\n")
		lineCount++
	}

	// Account info table (minimum 2 lines: header + at least one value line)
	minLinesNeeded := 2 + len(tx.Authorizers)
	if lineCount+minLinesNeeded < maxLines {
		// Use same column layout as full detail view
		colWidth := 20
		headerStyle := lipgloss.NewStyle().Bold(true).Foreground(secondaryColor).Width(colWidth)
		valueStyle := lipgloss.NewStyle().Foreground(accentColor).Width(colWidth)
		
		// Headers
		details.WriteString(headerStyle.Render("Proposer"))
		details.WriteString(headerStyle.Render("Payer"))
		details.WriteString(fieldStyle.Render("Authorizers"))
		details.WriteString("\n")
		lineCount++
		
		// Format addresses with friendly names
		proposerDisplay := tx.Proposer
		if !tv.showRawAddresses && tv.accountRegistry != nil {
			proposerDisplay = tv.accountRegistry.GetName(tx.Proposer)
		}
		
		payerDisplay := tx.Payer
		if !tv.showRawAddresses && tv.accountRegistry != nil {
			payerDisplay = tv.accountRegistry.GetName(tx.Payer)
		}
		
		for i, auth := range tx.Authorizers {
			var authDisplay string
			if !tv.showRawAddresses && tv.accountRegistry != nil {
				authDisplay = tv.accountRegistry.GetName(auth)
			} else {
				authDisplay = auth
			}
			
			if i == 0 {
				// First line with proposer, payer, and first authorizer
				details.WriteString(valueStyle.Render(proposerDisplay))
				details.WriteString(valueStyle.Render(payerDisplay))
				details.WriteString(valueStyleDetail.Render(authDisplay))
				details.WriteString("\n")
				lineCount++
			} else if lineCount < maxLines {
				// Additional authorizers aligned under the authorizer column
				details.WriteString(valueStyle.Render(""))
				details.WriteString(valueStyle.Render(""))
				details.WriteString(valueStyleDetail.Render(authDisplay))
				details.WriteString("\n")
				lineCount++
			}
		}
	}

	// Error (if present)
	if tx.Error != "" && lineCount+2 < maxLines {
		if lineCount+1 < maxLines {
			details.WriteString("\n")
			lineCount++
		}
		details.WriteString(fieldStyle.Render(fmt.Sprintf("%-12s", "Error:")) + " " + lipgloss.NewStyle().Foreground(errorColor).Render(tx.Error) + "\n")
		lineCount++
	}

	// Events (show some if there's space)
	if len(tx.Events) > 0 && lineCount+3 < maxLines {
		if lineCount+1 < maxLines {
			details.WriteString("\n")
			lineCount++
		}
		details.WriteString(fieldStyle.Render(fmt.Sprintf("Events (%d):", len(tx.Events))) + "\n")
		lineCount++

		eventsShown := 0
		for i, event := range tx.Events {
			if lineCount+2 >= maxLines {
				break
			}
			details.WriteString(fmt.Sprintf("  %d. %s\n", i+1, fieldStyle.Render(event.Name)))
			lineCount++
			eventsShown++

			// Only show event fields if showEventFields is true and there's space
			if tv.showEventFields && len(event.Fields) > 0 && lineCount+len(event.Fields)+1 < maxLines {
				keys := make([]string, 0, len(event.Fields))
				for key := range event.Fields {
					keys = append(keys, key)
				}
				sort.Strings(keys)

				maxKeyLen := 0
				for _, key := range keys {
					if len(key) > maxKeyLen {
						maxKeyLen = len(key)
					}
				}

				for _, key := range keys {
					if lineCount >= maxLines {
						break
					}
					val := event.Fields[key]
					paddedKey := fmt.Sprintf("%-*s", maxKeyLen, key)

					// Format value using helper function
					valStr := tv.formatEventFieldValue(val)

					details.WriteString(fmt.Sprintf("     %s: %s\n",
						valueStyleDetail.Render(paddedKey),
						valueStyleDetail.Render(valStr)))
					lineCount++
				}
			}
		}

		if eventsShown < len(tx.Events) {
			details.WriteString(fmt.Sprintf("  ... and %d more\n", len(tx.Events)-eventsShown))
		}
	}

	// Hint to view full details
	if lineCount+2 < maxLines {
		details.WriteString("\n")
		details.WriteString(lipgloss.NewStyle().Foreground(mutedColor).Render("Press Tab for full details"))
	}

	return details.String()
}

// Stop is a no-op for the transactions view
func (tv *TransactionsView) Stop() {
	// No cleanup needed
}
