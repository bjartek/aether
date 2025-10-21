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
	debugColor   = lipgloss.Color("#dc322f")
)

const (
	tableSplitPercent = 0.3
)

type model struct {
	table              table.Model
	detailViewport     viewport.Model
	width              int
	height             int
	codeOriginal       string // Original unhighlighted code
	codeFullscreen     string // Highlighted code for fullscreen width
	codeDetail         string // Highlighted code for detail panel width
	hexString          string // 256-char hex string for testing
	fullDetailMode     bool   // Toggle between split and fullscreen detail view
}

func initialModel() model {
	// Read the schedule_transaction.cdc file
	codeBytes, err := os.ReadFile("example/cadence/transactions/schedule_transaction.cdc")
	if err != nil {
		panic(fmt.Sprintf("Failed to read file: %v", err))
	}

	// Store original code - will highlight on resize with actual widths
	codeOriginal := string(codeBytes)

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

	// Create a 256-char hex string for testing
	hexString := "a1b2c3d4e5f6071829304a5b6c7d8e9f0a1b2c3d4e5f6071829304a5b6c7d8e9f0a1b2c3d4e5f6071829304a5b6c7d8e9f0a1b2c3d4e5f6071829304a5b6c7d8e9f0a1b2c3d4e5f6071829304a5b6c7d8e9f0a1b2c3d4e5f6071829304a5b6c7d8e9f0a1b2c3d4e5f6071829304a5b6c7d8e9f0a1b2c3d4e5f6071829304a5b6c7d8e9f"

	// Create viewport for detail panel
	vp := viewport.New(80, 20)

	return model{
		table:          t,
		detailViewport: vp,
		codeOriginal:   codeOriginal,
		hexString:      hexString,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

// buildViewportContent creates the full content string for the viewport
func (m model) buildViewportContent(width int, isFullscreen bool) string {
	fieldStyle := lipgloss.NewStyle().Bold(true).Foreground(accentColor)
	valueStyle := lipgloss.NewStyle().Foreground(mutedColor)
	debugStyle := lipgloss.NewStyle().Foreground(debugColor)

	modeLabel := "Detail"
	if isFullscreen {
		modeLabel = "Fullscreen"
	}
	debugInfo := debugStyle.Render(fmt.Sprintf("[DEBUG] %s Width: %d, Viewport: %dx%d, Scroll: %.0f%%",
		modeLabel,
		width,
		m.detailViewport.Width,
		m.detailViewport.Height,
		m.detailViewport.ScrollPercent()*100)) + "\n\n"

	title := fieldStyle.Render("Code: schedule_transaction.cdc") + "\n\n"
	hexLabel := fieldStyle.Render("Hex Data:") + " "
	hexValue := valueStyle.Render(m.hexString) + "\n\n"

	code := m.codeFullscreen
	if !isFullscreen {
		code = m.codeDetail
	}
	return debugInfo + title + hexLabel + hexValue + code
}


func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case " ", "enter": // Space or Enter toggles fullscreen
			m.fullDetailMode = !m.fullDetailMode
			return m, nil
		case "esc": // Esc exits fullscreen
			if m.fullDetailMode {
				m.fullDetailMode = false
				return m, nil
			}
		case "j", "down":
			if m.fullDetailMode {
				// In fullscreen, scroll viewport
				m.detailViewport, cmd = m.detailViewport.Update(msg)
			} else {
				// In split view, navigate table
				m.table, cmd = m.table.Update(msg)
			}
		case "k", "up":
			if m.fullDetailMode {
				// In fullscreen, scroll viewport
				m.detailViewport, cmd = m.detailViewport.Update(msg)
			} else {
				// In split view, navigate table
				m.table, cmd = m.table.Update(msg)
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		
		// Generate highlighted code versions for both modes with actual widths
		tableWidth := int(float64(m.width) * tableSplitPercent)
		detailWidth := m.width - tableWidth
		
		m.codeFullscreen = chroma.HighlightCadenceWithStyleAndWidth(m.codeOriginal, "solarized-dark", m.width)
		m.codeDetail = chroma.HighlightCadenceWithStyleAndWidth(m.codeOriginal, "solarized-dark", detailWidth)
		return m, nil
	}

	return m, cmd
}

func (m model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	// Fullscreen mode - show only detail viewport
	if m.fullDetailMode {
		// Set dimensions and render content fresh
		m.detailViewport.Width = m.width
		m.detailViewport.Height = m.height - 4
		content := m.buildViewportContent(m.width, true)
		wrappedContent := lipgloss.NewStyle().Width(m.width).Render(content)
		m.detailViewport.SetContent(wrappedContent)

		hint := lipgloss.NewStyle().
			Foreground(mutedColor).
			Render("j/k: scroll | space/enter/esc: exit fullscreen | q: quit")

		return hint + "\n\n" + m.detailViewport.View()
	}

	// Split view mode - table on left, detail on right
	tableWidth := int(float64(m.width) * tableSplitPercent)
	detailWidth := m.width - tableWidth

	// Update table dimensions
	m.table.SetWidth(tableWidth)
	m.table.SetHeight(m.height - 4)

	// Set viewport dimensions and render content fresh
	m.detailViewport.Width = detailWidth
	m.detailViewport.Height = m.height - 4
	content := m.buildViewportContent(detailWidth, false)
	wrappedContent := lipgloss.NewStyle().Width(detailWidth).Render(content)
	m.detailViewport.SetContent(wrappedContent)

	// Render table
	tableView := lipgloss.NewStyle().
		Width(tableWidth).
		MaxHeight(m.height).
		Render(m.table.View())

	// Render detail panel - viewport handles its own constraints
	detailView := m.detailViewport.View()

	// Join horizontally
	mainView := lipgloss.JoinHorizontal(
		lipgloss.Top,
		tableView,
		detailView,
	)

	// Add hint at top
	hint := lipgloss.NewStyle().
		Foreground(mutedColor).
		Render("j/k: navigate | space/enter: fullscreen | q: quit")

	return hint + "\n\n" + mainView
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
