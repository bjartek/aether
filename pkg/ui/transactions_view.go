package ui

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/bjartek/overflow/v2"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TransactionData holds transaction information for display
type TransactionData struct {
	ID         string
	BlockID    string
	BlockHeight uint64
	Authorizer string
	Status     string
	Proposer   string
	Payer      string
	GasLimit   uint64
	Script     string
	Arguments  string
	Events     []overflow.OverflowEvent
	Error      string
	Timestamp  time.Time
	Index      int
}

// TransactionMsg is sent when a new transaction is received
type TransactionMsg struct {
	Transaction TransactionData
}

// TransactionsKeyMap defines keybindings for the transactions view
type TransactionsKeyMap struct {
	LineUp   key.Binding
	LineDown key.Binding
	GotoTop  key.Binding
	GotoEnd  key.Binding
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
	}
}

// TransactionsView manages the transactions table and detail display
type TransactionsView struct {
	mu           sync.RWMutex
	table        table.Model
	keys         TransactionsKeyMap
	ready        bool
	transactions []TransactionData
	maxTxs       int
	width        int
	height       int
}

// NewTransactionsView creates a new transactions view
func NewTransactionsView() *TransactionsView {
	columns := []table.Column{
		{Title: "ID", Width: 16},
		{Title: "Block", Width: 10},
		{Title: "Authorizer", Width: 16},
		{Title: "Status", Width: 12},
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

	return &TransactionsView{
		table:        t,
		keys:         DefaultTransactionsKeyMap(),
		ready:        false,
		transactions: make([]TransactionData, 0),
		maxTxs:       1000, // Keep last 1000 transactions
	}
}

// Init initializes the transactions view
func (tv *TransactionsView) Init() tea.Cmd {
	return nil
}

// AddTransaction adds a new transaction from an OverflowTransaction
func (tv *TransactionsView) AddTransaction(blockHeight uint64, blockID string, ot overflow.OverflowTransaction) {
	tv.mu.Lock()
	defer tv.mu.Unlock()

	// Extract authorizer
	authorizer := "N/A"
	if len(ot.Authorizers) > 0 {
		authorizer = ot.Authorizers[0]
		if len(authorizer) > 16 {
			authorizer = authorizer[:16]
		}
	}

	// Extract proposer and payer
	proposer := "N/A"
	if ot.ProposalKey.Address.String() != "" {
		proposer = ot.ProposalKey.Address.Hex()
	}

	payer := "N/A"
	if ot.Payer != "" {
		payer = ot.Payer
		if len(payer) > 16 {
			payer = payer[:16]
		}
	}

	// Determine status
	status := "Unknown"
	if ot.Error != nil {
		status = "Failed"
	} else {
		status = ot.Status
	}

	// Format script (truncate if too long)
	script := string(ot.Script)
	if len(script) > 500 {
		script = script[:500] + "..."
	}

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
		Authorizer:  authorizer,
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
}

// refreshTable updates the table rows from transactions
func (tv *TransactionsView) refreshTable() {
	rows := make([]table.Row, len(tv.transactions))
	for i, tx := range tv.transactions {
		rows[i] = table.Row{
			tx.ID,
			fmt.Sprintf("%d", tx.BlockHeight),
			tx.Authorizer,
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
		// Split width: 60% table, 40% details
		tableWidth := int(float64(width) * 0.6)
		tv.table.SetWidth(tableWidth)
		tv.table.SetHeight(height)

	case tea.KeyMsg:
		var cmd tea.Cmd
		tv.table, cmd = tv.table.Update(msg)
		return cmd
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

	// Calculate widths
	tableWidth := int(float64(tv.width) * 0.6)
	detailWidth := tv.width - tableWidth - 2

	// Get selected transaction
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

// renderTransactionDetail renders the detailed view of a transaction
func (tv *TransactionsView) renderTransactionDetail(tx TransactionData, width, height int) string {
	detailStyle := lipgloss.NewStyle().
		Width(width).
		Height(height).
		Padding(1).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(borderColor).
		BorderLeft(true)

	fieldStyle := lipgloss.NewStyle().Bold(true).Foreground(secondaryColor)
	valueStyleDetail := lipgloss.NewStyle().Foreground(accentColor)

	var details strings.Builder
	details.WriteString(fieldStyle.Render("Transaction Details") + "\n\n")

	details.WriteString(fieldStyle.Render("ID: ") + valueStyleDetail.Render(tx.ID) + "\n")
	details.WriteString(fieldStyle.Render("Block: ") + valueStyleDetail.Render(fmt.Sprintf("%d", tx.BlockHeight)) + "\n")
	details.WriteString(fieldStyle.Render("Block ID: ") + valueStyleDetail.Render(tx.BlockID) + "\n")
	details.WriteString(fieldStyle.Render("Status: ") + valueStyleDetail.Render(tx.Status) + "\n")
	details.WriteString(fieldStyle.Render("Index: ") + valueStyleDetail.Render(fmt.Sprintf("%d", tx.Index)) + "\n\n")

	details.WriteString(fieldStyle.Render("Proposer: ") + valueStyleDetail.Render(tx.Proposer) + "\n")
	details.WriteString(fieldStyle.Render("Payer: ") + valueStyleDetail.Render(tx.Payer) + "\n")
	details.WriteString(fieldStyle.Render("Authorizer: ") + valueStyleDetail.Render(tx.Authorizer) + "\n")
	details.WriteString(fieldStyle.Render("Gas Limit: ") + valueStyleDetail.Render(fmt.Sprintf("%d", tx.GasLimit)) + "\n\n")

	if tx.Error != "" {
		details.WriteString(fieldStyle.Render("Error: ") + lipgloss.NewStyle().Foreground(errorColor).Render(tx.Error) + "\n\n")
	}

	if len(tx.Events) > 0 {
		details.WriteString(fieldStyle.Render(fmt.Sprintf("Events (%d):\n", len(tx.Events))))
		for i, event := range tx.Events {
			if i < 5 { // Show only first 5 events
				details.WriteString(fmt.Sprintf("  %d. %s\n", i+1, valueStyleDetail.Render(event.Name)))
			}
		}
		if len(tx.Events) > 5 {
			details.WriteString(fmt.Sprintf("  ... and %d more\n", len(tx.Events)-5))
		}
		details.WriteString("\n")
	}

	if tx.Arguments != "" {
		details.WriteString(fieldStyle.Render("Arguments:\n"))
		details.WriteString(valueStyleDetail.Render(tx.Arguments) + "\n\n")
	}

	if tx.Script != "" {
		details.WriteString(fieldStyle.Render("Script:\n"))
		scriptPreview := tx.Script
		if len(scriptPreview) > 300 {
			scriptPreview = scriptPreview[:300] + "..."
		}
		details.WriteString(valueStyleDetail.Render(scriptPreview) + "\n")
	}

	return detailStyle.Render(details.String())
}

// Stop is a no-op for the transactions view
func (tv *TransactionsView) Stop() {
	// No cleanup needed
}
