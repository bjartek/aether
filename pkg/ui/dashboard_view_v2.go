package ui

import (
	"fmt"

	"github.com/bjartek/aether/pkg/config"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// DashboardViewV2 displays service information and guidelines
type DashboardViewV2 struct {
	services []ServiceInfo
	ready    bool
	width    int
	height   int
}

// NewDashboardViewV2WithConfig creates a new v2 dashboard view
func NewDashboardViewV2WithConfig(cfg *config.Config) *DashboardViewV2 {
	// Fallback to defaults when cfg is nil
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	// Build services list from config ports
	services := []ServiceInfo{
		{Name: "Flow Emulator (gRPC)", Port: fmt.Sprintf("%d", cfg.Ports.Emulator.GRPC), Status: "Running"},
		{Name: "Flow Emulator (REST)", Port: fmt.Sprintf("%d", cfg.Ports.Emulator.REST), Status: "Running"},
		{Name: "Flow Emulator (Admin)", Port: fmt.Sprintf("%d", cfg.Ports.Emulator.Admin), Status: "Running"},
		{Name: "Flow Emulator (Debugger)", Port: fmt.Sprintf("%d", cfg.Ports.Emulator.Debugger), Status: "Running"},
		{Name: "Dev Wallet", Port: fmt.Sprintf("%d", cfg.Ports.DevWallet), Status: "Running"},
		{Name: "EVM Gateway (JSON-RPC)", Port: fmt.Sprintf("%d", cfg.Ports.EVM.RPC), Status: "Running"},
		{Name: "EVM Gateway (Profile)", Port: fmt.Sprintf("%d", cfg.Ports.EVM.Profiler), Status: "Running"},
	}

	return &DashboardViewV2{
		services: services,
		ready:    true,
	}
}

// Init implements tea.Model
func (dv *DashboardViewV2) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (dv *DashboardViewV2) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		dv.width = msg.Width
		dv.height = msg.Height
	}
	return dv, nil
}

// View implements tea.Model
func (dv *DashboardViewV2) View() string {
	if !dv.ready {
		return "Loading dashboard..."
	}

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

// Name implements TabbedModel interface
func (dv *DashboardViewV2) Name() string {
	return "Dashboard"
}

// KeyMap implements TabbedModel interface
func (dv *DashboardViewV2) KeyMap() help.KeyMap {
	return dashboardKeyMapAdapter{}
}

// dashboardKeyMapAdapter provides empty key bindings for dashboard
type dashboardKeyMapAdapter struct{}

func (k dashboardKeyMapAdapter) ShortHelp() []key.Binding {
	return []key.Binding{}
}

func (k dashboardKeyMapAdapter) FullHelp() [][]key.Binding {
	return [][]key.Binding{}
}

// FooterView implements TabbedModel interface
func (dv *DashboardViewV2) FooterView() string {
	return ""
}

// IsCapturingInput implements TabbedModel interface
func (dv *DashboardViewV2) IsCapturingInput() bool {
	return false
}

func (dv *DashboardViewV2) renderServices() string {
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

func (dv *DashboardViewV2) renderGuidelines() string {
	guidelines := `  Aether is an alternative cli for the Flow blockchain.
  It combines multiple Flow CLI tools into a unified interface.

  Key Features:
  • Real-time transaction monitoring with detailed inspection
  • Real-time event monitoring with detailed inspection
  • Real-time log monitoring with auto-scroll
  • Runner for transactions and scripts
  • Save and run transactions and scripts with prefilled parameters
  • Service status tracking
  • Comprehensive logging with auto-scroll

  Press ? to see all keybindings and navigation help

  Built with:
  • Overflow - Go toolkit for Flow blockchain
    (github.com/bjartek/overflow)

  ---
  💭 Vibe-coded with Windsurf & Claude Sonnet 4.5 Thinking
`
	return valueStyle.Render(guidelines)
}
