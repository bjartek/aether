package ui

import (
	"fmt"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			MarginBottom(1)

	sectionStyle = lipgloss.NewStyle().
			MarginTop(1).
			MarginBottom(1)

	labelStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(secondaryColor)

	valueStyle = lipgloss.NewStyle().
			Foreground(accentColor)
)

// ServiceInfo represents information about a running service
type ServiceInfo struct {
	Name   string
	Port   string
	Status string
}

// DashboardView displays service information and guidelines
type DashboardView struct {
	mu       sync.RWMutex
	services []ServiceInfo
	ready    bool
}

// NewDashboardView creates a new dashboard view
func NewDashboardView() *DashboardView {
	return &DashboardView{
		services: []ServiceInfo{
			{Name: "Flow Emulator", Port: "3569", Status: "Running"},
			{Name: "Dev Wallet", Port: "8701", Status: "Running"},
		},
		ready: true,
	}
}

// Init initializes the dashboard view
func (dv *DashboardView) Init() tea.Cmd {
	return nil
}

// Update handles messages for the dashboard view
func (dv *DashboardView) Update(msg tea.Msg, width, height int) tea.Cmd {
	return nil
}

// View renders the dashboard view
func (dv *DashboardView) View() string {
	if !dv.ready {
		return "Loading dashboard..."
	}

	dv.mu.RLock()
	defer dv.mu.RUnlock()

	var content string

	// Title
	content += titleStyle.Render("🌊 Aether - Flow Blockchain Dashboard") + "\n\n"

	// Services section
	content += sectionStyle.Render(
		labelStyle.Render("Running Services:") + "\n" +
			dv.renderServices(),
	)

	// Guidelines section
	content += sectionStyle.Render(
		labelStyle.Render("Guidelines:") + "\n" +
			dv.renderGuidelines(),
	)

	return content
}

func (dv *DashboardView) renderServices() string {
	var services string
	for _, svc := range dv.services {
		statusColor := successColor
		if svc.Status != "Running" {
			statusColor = errorColor
		}

		services += fmt.Sprintf("  • %s - Port %s - %s\n",
			valueStyle.Render(svc.Name),
			valueStyle.Render(svc.Port),
			lipgloss.NewStyle().Foreground(statusColor).Render(svc.Status),
		)
	}
	return services
}

func (dv *DashboardView) renderGuidelines() string {
	guidelines := `  Aether is an alternative cli for the Flow blockchain.
  It combines multiple Flow CLI tools into a unified interface.

  Key Features:
  • Real-time transaction monitoring with detailed inspection
  • Service status tracking
  • Comprehensive logging with auto-scroll

  Navigation:
  • Use Tab/→ and Shift+Tab/← to switch between panes
  • In Transactions: Press Enter or 'd' to toggle full detail view
  • Press ? for help
  • Press q to quit

  Built with:
  • Overflow - Go toolkit for Flow blockchain
    (github.com/bjartek/overflow)

  ---
  💭 Vibe-coded with Windsurf & Claude Sonnet 4.5 Thinking
`
	return valueStyle.Render(guidelines)
}

// Stop is a no-op for the dashboard view
func (dv *DashboardView) Stop() {
	// No cleanup needed
}
