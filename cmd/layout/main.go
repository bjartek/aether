package main

import (
	"fmt"
	"os"

	"github.com/bjartek/aether/pkg/chroma"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	primaryColor = lipgloss.Color("#268bd2")
	accentColor  = lipgloss.Color("#b58900")
	mutedColor   = lipgloss.Color("#586e75")
)

type model struct {
	table          table.Model
	detailViewport viewport.Model
	width          int
	height         int
	code           string
}

func initialModel() model {
	// Read the schedule_transaction.cdc file
	codeBytes, err := os.ReadFile("example/cadence/transactions/schedule_transaction.cdc")
	if err != nil {
		panic(fmt.Sprintf("Failed to read file: %v", err))
	}

	// Highlight and wrap code at 160 chars
	code := chroma.HighlightCadenceWithStyleAndWidth(string(codeBytes), "solarized-dark", 120)

	// Create table with some dummy data
	columns := []table.Column{
		{Title: "Name", Width: 20},
		{Title: "Type", Width: 15},
		{Title: "Network", Width: 15},
	}

	rows := []table.Row{
		{"schedule_transaction", "Transaction", "testnet"},
		{"get_balance", "Script", "mainnet"},
		{"transfer_tokens", "Transaction", "testnet"},
		{"create_account", "Script", "emulator"},
		{"update_contract", "Transaction", "mainnet"},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(primaryColor).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("#fdf6e3")).
		Background(primaryColor).
		Bold(false)
	t.SetStyles(s)

	// Create viewport for detail panel
	vp := viewport.New(80, 20)
	vp.SetContent(code)

	return model{
		table:          t,
		detailViewport: vp,
		code:           code,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "j", "down":
			m.table, cmd = m.table.Update(msg)
		case "k", "up":
			m.table, cmd = m.table.Update(msg)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Split 30/70
		tableWidth := int(float64(m.width) * 0.3)
		detailWidth := m.width - tableWidth

		// Update table
		m.table.SetWidth(tableWidth)
		m.table.SetHeight(m.height - 4)

		// Update detail viewport
		m.detailViewport.Width = detailWidth
		m.detailViewport.Height = m.height - 4
		m.detailViewport.SetContent(m.code)

		return m, nil
	}

	return m, cmd
}

func (m model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	// Calculate widths
	tableWidth := int(float64(m.width) * 0.3)
	detailWidth := m.width - tableWidth

	// Render table
	tableView := lipgloss.NewStyle().
		Width(tableWidth).
		MaxHeight(m.height).
		Render(m.table.View())

	// Render detail panel with title
	fieldStyle := lipgloss.NewStyle().Bold(true).Foreground(accentColor)
	title := fieldStyle.Render("Code: schedule_transaction.cdc") + "\n\n"

	detailContent := title + m.detailViewport.View()
	detailView := lipgloss.NewStyle().
		Width(detailWidth).
		Height(m.height).
		Render(detailContent)

	// Join horizontally
	mainView := lipgloss.JoinHorizontal(
		lipgloss.Top,
		tableView,
		detailView,
	)

	// Add hint at top
	hint := lipgloss.NewStyle().
		Foreground(mutedColor).
		Render("j/k: navigate | q: quit")

	return hint + "\n\n" + mainView
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
