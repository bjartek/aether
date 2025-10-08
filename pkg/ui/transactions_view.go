package ui

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/bjartek/aether/pkg/aether"
	"github.com/bjartek/overflow/v2"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TransactionData holds transaction information for display
type TransactionData struct {
	ID          string
	BlockID     string
	BlockHeight uint64
	Authorizers []string // Can have multiple authorizers
	Status      string
	Proposer    string
	Payer       string
	GasLimit    uint64
	Script      string
	Arguments   string
	Events      []overflow.OverflowEvent
	Error       string
	Timestamp   time.Time
	Index       int
}

// TransactionMsg is sent when a new transaction is received
type TransactionMsg struct {
	Transaction TransactionData
}

// TransactionsKeyMap defines keybindings for the transactions view
type TransactionsKeyMap struct {
	LineUp           key.Binding
	LineDown         key.Binding
	GotoTop          key.Binding
	GotoEnd          key.Binding
	ToggleFullDetail key.Binding
	ToggleEventFields key.Binding
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
			key.WithKeys("enter", "d"),
			key.WithHelp("enter/d", "toggle full detail"),
		),
		ToggleEventFields: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "toggle event fields"),
		),
	}
}

// TransactionsView manages the transactions table and detail display
type TransactionsView struct {
	mu               sync.RWMutex
	table            table.Model
	detailViewport   viewport.Model
	keys             TransactionsKeyMap
	ready            bool
	transactions     []TransactionData
	maxTxs           int
	width            int
	height           int
	fullDetailMode   bool // Toggle between split and full-screen detail view
	showEventFields  bool // Toggle showing event field details
	accountRegistry  *aether.AccountRegistry
}

// NewTransactionsView creates a new transactions view
func NewTransactionsView() *TransactionsView {
	columns := []table.Column{
		{Title: "ID", Width: 20},           // Truncated hex (8...8)
		{Title: "Block", Width: 6},         // Slimmer for block numbers
		{Title: "Authorizer", Width: 30},   // Wider to show friendly names
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
		Background(solarBlue).
		Bold(false)
	t.SetStyles(s)

	// Create viewport for detail view
	vp := viewport.New(0, 0)
	vp.Style = lipgloss.NewStyle()
	
	return &TransactionsView{
		table:           t,
		detailViewport:  vp,
		keys:            DefaultTransactionsKeyMap(),
		ready:           false,
		transactions:    make([]TransactionData, 0),
		maxTxs:          1000, // Keep last 1000 transactions
		fullDetailMode:  false,
		showEventFields: true, // Show event fields by default
	}
}

// Init initializes the transactions view
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
		proposer = ot.ProposalKey.Address.Hex()
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

	// Format arguments
	args := ""
	if len(ot.Arguments) > 0 {
		argStrs := make([]string, len(ot.Arguments))
		for i, arg := range ot.Arguments {
			argStrs[i] = fmt.Sprintf("%v", arg.Value)
		}
		args = strings.Join(argStrs, ", ")
		if len(args) > 200 {
			args = args[:200] + "..."
		}
	}

	// Create error message
	errMsg := ""
	if ot.Error != nil {
		errMsg = ot.Error.Error()
	}

	// Store events directly
	events := ot.Events

	txData := TransactionData{
		ID:          ot.Id,
		BlockID:     blockID,
		BlockHeight: blockHeight,
		Authorizers: authorizers,
		Status:      status,
		Proposer:    proposer,
		Payer:       payer,
		GasLimit:    ot.GasLimit,
		Script:      script,
		Arguments:   args,
		Events:      events,
		Error:       errMsg,
		Timestamp:   time.Now(),
		Index:       ot.TransactionIndex,
	}

	tv.transactions = append(tv.transactions, txData)

	// Keep only the last maxTxs transactions
	if len(tv.transactions) > tv.maxTxs {
		tv.transactions = tv.transactions[len(tv.transactions)-tv.maxTxs:]
	}

	tv.refreshTable()
	
	// Update viewport if in full detail mode
	if tv.fullDetailMode {
		tv.updateDetailViewport()
	}
}

// updateDetailViewport updates the viewport content with current transaction details
func (tv *TransactionsView) updateDetailViewport() {
	tv.mu.RLock()
	defer tv.mu.RUnlock()
	
	if len(tv.transactions) == 0 {
		tv.detailViewport.SetContent("")
		return
	}
	
	selectedIdx := tv.table.Cursor()
	if selectedIdx >= 0 && selectedIdx < len(tv.transactions) {
		detail := tv.renderTransactionDetailText(tv.transactions[selectedIdx])
		tv.detailViewport.SetContent(detail)
	}
}

// refreshTable updates the table rows from transactions
func (tv *TransactionsView) refreshTable() {
	rows := make([]table.Row, len(tv.transactions))
	for i, tx := range tv.transactions {
		// Show first authorizer in table with friendly name if available
		authDisplay := "N/A"
		if len(tx.Authorizers) > 0 {
			addr := tx.Authorizers[0]
			if tv.accountRegistry != nil {
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
			truncateHex(tx.ID, 8, 8),           // Show start and end of ID
			fmt.Sprintf("%d", tx.BlockHeight),
			authDisplay,                         // Show friendly name or truncated address
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
		// Handle toggle full detail view
		if key.Matches(msg, tv.keys.ToggleFullDetail) {
			tv.fullDetailMode = !tv.fullDetailMode
			// Update viewport content when entering full detail mode
			if tv.fullDetailMode {
				tv.updateDetailViewport()
			}
			return nil
		}
		
		// Handle toggle event fields
		if key.Matches(msg, tv.keys.ToggleEventFields) {
			tv.showEventFields = !tv.showEventFields
			// Refresh viewport content
			if tv.fullDetailMode {
				tv.updateDetailViewport()
			}
			return nil
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

	// Full detail mode - show only the transaction detail in viewport
	if tv.fullDetailMode {
		hint := lipgloss.NewStyle().
			Foreground(mutedColor).
			Render("Press Enter or 'd' to return to table view | j/k to scroll")
		return hint + "\n\n" + tv.detailViewport.View()
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
	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		tableView,
		detailView,
	)
}

// renderTransactionDetailText renders transaction details as plain text (for viewport)
func (tv *TransactionsView) renderTransactionDetailText(tx TransactionData) string {
	fieldStyle := lipgloss.NewStyle().Bold(true).Foreground(secondaryColor)
	valueStyleDetail := lipgloss.NewStyle().Foreground(accentColor)

	// Helper function to align fields
	renderField := func(label, value string) string {
		return fieldStyle.Render(fmt.Sprintf("%-12s", label+":")) + " " + valueStyleDetail.Render(value) + "\n"
	}
	
	// Helper for label on one line, value on next
	renderFieldBlock := func(label, value string) string {
		return fieldStyle.Render(label) + "\n" + valueStyleDetail.Render(value) + "\n"
	}

	var details strings.Builder
	details.WriteString(fieldStyle.Render("Transaction Details") + "\n\n")

	details.WriteString(renderField("ID", tx.ID))
	details.WriteString(renderField("Block", fmt.Sprintf("%d", tx.BlockHeight)))
	details.WriteString(renderField("Block ID", tx.BlockID))
	details.WriteString(renderField("Status", tx.Status))
	details.WriteString(renderField("Index", fmt.Sprintf("%d", tx.Index)))
	details.WriteString("\n")

	// Format addresses with friendly names
	proposerDisplay := tx.Proposer
	if tv.accountRegistry != nil {
		proposerDisplay = tv.accountRegistry.FormatAddress(tx.Proposer)
	}
	details.WriteString(renderFieldBlock("Proposer", proposerDisplay))
	
	payerDisplay := tx.Payer
	if tv.accountRegistry != nil {
		payerDisplay = tv.accountRegistry.FormatAddress(tx.Payer)
	}
	details.WriteString(renderFieldBlock("Payer", payerDisplay))
	
	// Handle multiple authorizers with friendly names
	authDisplayList := make([]string, len(tx.Authorizers))
	for i, auth := range tx.Authorizers {
		if tv.accountRegistry != nil {
			authDisplayList[i] = tv.accountRegistry.FormatAddress(auth)
		} else {
			authDisplayList[i] = auth
		}
	}
	authDisplay := strings.Join(authDisplayList, ", ")
	details.WriteString(renderFieldBlock("Authorizers", authDisplay))
	
	details.WriteString(renderField("Gas Limit", fmt.Sprintf("%d", tx.GasLimit)))
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
				
				// Display fields aligned on ->
				for _, key := range keys {
					val := event.Fields[key]
					paddedKey := fmt.Sprintf("%-*s", maxKeyLen, key)
					details.WriteString(fmt.Sprintf("     %s -> %s\n", 
						valueStyleDetail.Render(paddedKey),
						valueStyleDetail.Render(fmt.Sprintf("%v", val))))
				}
			}
		}
		details.WriteString("\n")
	}

	if tx.Arguments != "" {
		details.WriteString(fieldStyle.Render(fmt.Sprintf("%-12s", "Arguments:")) + "\n")
		details.WriteString(valueStyleDetail.Render(tx.Arguments) + "\n\n")
	}

	if tx.Script != "" {
		details.WriteString(fieldStyle.Render(fmt.Sprintf("%-12s", "Script:")) + "\n")
		// Show full script - user can scroll if needed
		details.WriteString(valueStyleDetail.Render(tx.Script) + "\n")
	}

	return details.String()
}

// renderTransactionDetail renders the detailed view of a transaction (for split view)
// This version is height-constrained to prevent overflow
func (tv *TransactionsView) renderTransactionDetail(tx TransactionData, width, height int) string {
	detailStyle := lipgloss.NewStyle().
		Width(width).
		Height(height).
		Padding(1).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(borderColor).
		BorderLeft(true)

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
	
	renderFieldBlock := func(label, value string) string {
		return fieldStyle.Render(label) + "\n" + valueStyleDetail.Render(value) + "\n"
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

	// Account info
	if lineCount+6 < maxLines {
		proposerDisplay := tx.Proposer
		if tv.accountRegistry != nil {
			proposerDisplay = tv.accountRegistry.FormatAddress(tx.Proposer)
		}
		details.WriteString(renderFieldBlock("Proposer", proposerDisplay))
		lineCount += 2
		
		payerDisplay := tx.Payer
		if tv.accountRegistry != nil {
			payerDisplay = tv.accountRegistry.FormatAddress(tx.Payer)
		}
		details.WriteString(renderFieldBlock("Payer", payerDisplay))
		lineCount += 2
		
		authDisplayList := make([]string, len(tx.Authorizers))
		for i, auth := range tx.Authorizers {
			if tv.accountRegistry != nil {
				authDisplayList[i] = tv.accountRegistry.FormatAddress(auth)
			} else {
				authDisplayList[i] = auth
			}
		}
		authDisplay := strings.Join(authDisplayList, ", ")
		details.WriteString(renderFieldBlock("Authorizers", authDisplay))
		lineCount += 2
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
					details.WriteString(fmt.Sprintf("     %s -> %s\n", 
						valueStyleDetail.Render(paddedKey),
						valueStyleDetail.Render(fmt.Sprintf("%v", val))))
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
		details.WriteString(lipgloss.NewStyle().Foreground(mutedColor).Render("Press Enter/d for full details"))
	}

	return details.String()
}

// Stop is a no-op for the transactions view
func (tv *TransactionsView) Stop() {
	// No cleanup needed
}
