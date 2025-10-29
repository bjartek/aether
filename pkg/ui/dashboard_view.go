package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/bjartek/aether/pkg/aether"
	"github.com/bjartek/aether/pkg/config"
	"github.com/bjartek/aether/pkg/events"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rs/zerolog"
)

// DashboardView displays live system status in three boxes
type DashboardView struct {
	// Services box
	services      []ServiceInfo
	servicesReady bool

	// Init transactions box
	initTransactions []InitTransactionStatus
	initComplete     bool

	// Block height box
	latestBlockHeight uint64
	network           string
	blockTime         string
	indexerPolling    string

	// Accounts box
	accountRegistry *aether.AccountRegistry
	accounts        []string

	// Folder selection (for interactive init transaction mode)
	folderSelection     *aether.InitFolderSelectionMsg
	selectedFolderIndex int
	selectedInitFolder  string         // The folder being used for init transactions (empty = root)
	interactiveMode     bool           // If true, user needs to select folder
	aetherServer        *aether.Aether // Reference to server for running init transactions

	// Frontend box
	frontendCommand string
	frontendStatus  string // "Running" or "Stopped"
	frontendPorts   []string

	// Layout
	width  int
	height int
	logger zerolog.Logger
}

type ServiceInfo struct {
	Name   string
	Port   string
	Status string
}

type InitTransactionStatus struct {
	Filename string
	Success  bool
	Error    string
}

// NewDashboardViewWithConfig creates a new dashboard view with given config
func NewDashboardViewWithConfig(cfg *config.Config, logger zerolog.Logger, aetherServer *aether.Aether) *DashboardView {
	// Fallback to defaults when cfg is nil
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	// Build services list from config ports
	services := []ServiceInfo{
		{Name: "Flow Emulator (gRPC)", Port: fmt.Sprintf("%d", cfg.Ports.Emulator.GRPC), Status: "Starting..."},
		{Name: "Flow Emulator (REST)", Port: fmt.Sprintf("%d", cfg.Ports.Emulator.REST), Status: "Starting..."},
		{Name: "Flow Emulator (Admin)", Port: fmt.Sprintf("%d", cfg.Ports.Emulator.Admin), Status: "Starting..."},
		{Name: "Flow Emulator (Debugger)", Port: fmt.Sprintf("%d", cfg.Ports.Emulator.Debugger), Status: "Starting..."},
		{Name: "Dev Wallet", Port: fmt.Sprintf("%d", cfg.Ports.DevWallet), Status: "Starting..."},
		{Name: "EVM Gateway (JSON-RPC)", Port: fmt.Sprintf("%d", cfg.Ports.EVM.RPC), Status: "Starting..."},
		{Name: "EVM Gateway (Profiler)", Port: fmt.Sprintf("%d", cfg.Ports.EVM.Profiler), Status: "Starting..."},
	}

	frontendStatus := "Not configured"
	if cfg.FrontendCommand != "" {
		frontendStatus = "Starting..."
	}

	return &DashboardView{
		services:            services,
		servicesReady:       false,
		initTransactions:    []InitTransactionStatus{},
		initComplete:        false,
		latestBlockHeight:   0,
		network:             cfg.Network,
		blockTime:           cfg.Flow.BlockTime.String(),
		indexerPolling:      cfg.Indexer.PollingInterval.String(),
		accountRegistry:     nil,
		accounts:            []string{},
		folderSelection:     nil,
		selectedFolderIndex: 0,
		selectedInitFolder:  cfg.Flow.InitTransactionsFolder, // Store configured folder
		interactiveMode:     cfg.Flow.InitTransactionsInteractive,
		aetherServer:        aetherServer,
		frontendCommand:     cfg.FrontendCommand,
		frontendStatus:      frontendStatus,
		frontendPorts:       []string{},
		logger:              logger,
	}
}

// Init implements tea.Model
func (dv *DashboardView) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (dv *DashboardView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		dv.width = msg.Width
		dv.height = msg.Height

	case aether.OverflowReadyMsg:
		// Mark services as ready
		dv.servicesReady = true
		for i := range dv.services {
			dv.services[i].Status = "Running"
		}
		// Store account registry and get account names
		dv.accountRegistry = msg.AccountRegistry
		if dv.accountRegistry != nil {
			dv.accounts = dv.accountRegistry.GetAllNames()
		}
		dv.logger.Debug().Msg("Services marked as ready")

	case aether.InitTransactionMsg:
		// Add init transaction status
		dv.initTransactions = append(dv.initTransactions, InitTransactionStatus{
			Filename: msg.Filename,
			Success:  msg.Success,
			Error:    msg.Error,
		})
		dv.logger.Debug().
			Str("filename", msg.Filename).
			Bool("success", msg.Success).
			Msg("Init transaction status received")

	case aether.BlockHeightMsg:
		// Update block height
		if msg.Height > dv.latestBlockHeight {
			dv.latestBlockHeight = msg.Height
		}

	case aether.InitFolderSelectionMsg:
		// Store folder selection options
		dv.folderSelection = &msg
		dv.selectedFolderIndex = 0
		dv.logger.Debug().
			Int("folderCount", len(msg.Folders)).
			Msg("Folder selection received")

	case events.FrontendPortMsg:
		// Add port to frontend ports list
		dv.frontendPorts = append(dv.frontendPorts, msg.Port)
		dv.frontendStatus = "Running"
		dv.logger.Debug().Str("port", msg.Port).Msg("Frontend port detected")

	case tea.KeyMsg:
		// Handle folder selection navigation
		if dv.folderSelection != nil {
			switch msg.String() {
			case "up", "k":
				if dv.selectedFolderIndex > 0 {
					dv.selectedFolderIndex--
				}
				return dv, nil
			case "down", "j":
				if dv.selectedFolderIndex < len(dv.folderSelection.Folders)-1 {
					dv.selectedFolderIndex++
				}
				return dv, nil
			case "enter":
				// Run init transactions with selected folder
				selectedFolder := dv.folderSelection.Folders[dv.selectedFolderIndex]
				dv.logger.Info().
					Int("index", dv.selectedFolderIndex).
					Str("folder", selectedFolder).
					Msg("User selected init folder")

				// Store selected folder and clear selection UI
				dv.selectedInitFolder = selectedFolder
				dv.folderSelection = nil

				// Run init transactions in background
				if dv.aetherServer != nil {
					go func() {
						if err := dv.aetherServer.RunInitTransactionsWithFolder(selectedFolder); err != nil {
							dv.logger.Error().Err(err).Msg("Failed to run init transactions")
						}
					}()
				}

				return dv, nil
			}
		}
	}

	return dv, nil
}

// View implements tea.Model
func (dv *DashboardView) View() string {
	if dv.width == 0 {
		return "Loading..."
	}

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00D7FF")).
		MarginBottom(1)

	title := titleStyle.Render("🌊 Aether - Flow Blockchain Dashboard") + "\n"

	var boxes string

	// Show different boxes based on network
	if dv.network == "emulator" {
		// Emulator: show all four boxes
		boxWidth := (dv.width - 8) / 4 // -8 for spacing between boxes
		boxHeight := dv.height - 5     // -5 for title and padding

		servicesBox := dv.renderServicesBox(boxWidth, boxHeight)
		accountsBox := dv.renderAccountsBox(boxWidth, boxHeight)
		initBox := dv.renderInitTransactionsBox(boxWidth, boxHeight)
		blockHeightBox := dv.renderBlockHeightBox(boxWidth, boxHeight)

		boxes = lipgloss.JoinHorizontal(
			lipgloss.Top,
			servicesBox,
			"  ", // spacing
			accountsBox,
			"  ", // spacing
			initBox,
			"  ", // spacing
			blockHeightBox,
		)

		// Add frontend box if configured
		if dv.frontendCommand != "" {
			frontendBox := dv.renderFrontendBox(boxWidth, boxHeight)
			// Create a second row with the frontend box
			secondRow := lipgloss.JoinHorizontal(
				lipgloss.Top,
				frontendBox,
			)
			boxes = lipgloss.JoinVertical(lipgloss.Left, boxes, secondRow)
		}
	} else {
		// Testnet/Mainnet: only show accounts and block height
		boxWidth := (dv.width - 4) / 2 // -4 for spacing between boxes
		boxHeight := dv.height - 5     // -5 for title and padding

		accountsBox := dv.renderAccountsBox(boxWidth, boxHeight)
		blockHeightBox := dv.renderBlockHeightBox(boxWidth, boxHeight)

		boxes = lipgloss.JoinHorizontal(
			lipgloss.Top,
			accountsBox,
			"  ", // spacing
			blockHeightBox,
		)
	}

	return title + boxes
}

// renderServicesBox renders the services status box
func (dv *DashboardView) renderServicesBox(width, height int) string {
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(secondaryColor).
		PaddingLeft(1).
		PaddingRight(1)

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(secondaryColor).
		Background(base02).
		PaddingLeft(1).
		PaddingRight(1).
		MarginBottom(1)

	var content strings.Builder

	// Header
	if dv.servicesReady {
		content.WriteString(headerStyle.Render("✓ Services Running") + "\n\n")
	} else {
		content.WriteString(headerStyle.Render("⏳ Starting Services...") + "\n\n")
	}

	// Services list
	if len(dv.services) == 0 {
		content.WriteString(dimStyle.Render("No services configured"))
	} else {
		// Calculate max service name length for alignment
		maxNameLen := 0
		for _, svc := range dv.services {
			nameLen := len(svc.Name) + 2 // +2 for symbol and space
			if nameLen > maxNameLen {
				maxNameLen = nameLen
			}
		}

		for _, svc := range dv.services {
			statusColor := mutedColor
			statusSymbol := "⏳"

			if svc.Status == "Running" {
				statusColor = successColor
				statusSymbol = "✓"
			}

			// Pad service name to align ports
			serviceName := statusSymbol + " " + svc.Name
			paddedName := fmt.Sprintf("%-*s", maxNameLen, serviceName)

			line := fmt.Sprintf("%s - Port %s\n",
				lipgloss.NewStyle().Foreground(statusColor).Render(paddedName),
				dimStyle.Render(svc.Port),
			)
			content.WriteString(line)
		}
	}

	return boxStyle.Render(content.String())
}

// renderAccountsBox renders the accounts box showing dev-wallet accounts and funding status
func (dv *DashboardView) renderAccountsBox(width, height int) string {
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(primaryColor).
		PaddingLeft(1).
		PaddingRight(1)

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(primaryColor).
		Background(base02).
		PaddingLeft(1).
		PaddingRight(1).
		MarginBottom(1)

	var content strings.Builder

	// Header
	if dv.accountRegistry == nil || len(dv.accounts) == 0 {
		content.WriteString(headerStyle.Render("👤 Accounts") + "\n\n")
		content.WriteString(dimStyle.Render("Waiting for accounts..."))
	} else {
		content.WriteString(headerStyle.Render(fmt.Sprintf("👤 Accounts (%d)", len(dv.accounts))) + "\n\n")

		// Get all address-name pairs from registry
		addressToName := dv.accountRegistry.DebugDump()

		// Create sorted list of accounts: service-account first, then alphabetically
		type accountPair struct {
			name    string
			address string
		}
		var accounts []accountPair
		var serviceAccount *accountPair

		for address, name := range addressToName {
			if name == "service-account" {
				serviceAccount = &accountPair{name: name, address: address}
			} else {
				accounts = append(accounts, accountPair{name: name, address: address})
			}
		}

		// Sort non-service accounts alphabetically
		sort.Slice(accounts, func(i, j int) bool {
			return accounts[i].name < accounts[j].name
		})

		// Show service-account first if it exists
		if serviceAccount != nil {
			line := fmt.Sprintf("%s %s\n  %s\n",
				lipgloss.NewStyle().Foreground(successColor).Render("✓"),
				valueStyle.Render(serviceAccount.name),
				dimStyle.Render(serviceAccount.address),
			)
			content.WriteString(line)
		}

		// Show all other accounts in alphabetical order
		for _, account := range accounts {
			line := fmt.Sprintf("%s %s\n  %s\n",
				lipgloss.NewStyle().Foreground(successColor).Render("✓"),
				valueStyle.Render(account.name),
				dimStyle.Render(account.address),
			)
			content.WriteString(line)
		}
	}

	return boxStyle.Render(content.String())
}

// renderInitTransactionsBox renders the init transactions progress box or folder selection
func (dv *DashboardView) renderInitTransactionsBox(width, height int) string {
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(highlightColor).
		PaddingLeft(1).
		PaddingRight(1)

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(highlightColor).
		Background(base02).
		PaddingLeft(1).
		PaddingRight(1).
		MarginBottom(1)

	var content strings.Builder

	// If folder selection is active, show folder selection UI
	if dv.folderSelection != nil {
		content.WriteString(headerStyle.Render("📁 Select Init Folder") + "\n\n")
		content.WriteString(dimStyle.Render("↑/↓ or j/k to navigate, Enter to select") + "\n\n")

		// List folders
		for i, folder := range dv.folderSelection.Folders {
			displayName := folder
			if displayName == "" {
				displayName = ". (root)"
			}

			if i == dv.selectedFolderIndex {
				// Selected item
				line := lipgloss.NewStyle().
					Foreground(highlightColor).
					Bold(true).
					Render("▶ " + displayName)
				content.WriteString(line + "\n")
			} else {
				// Unselected item
				line := dimStyle.Render("  " + displayName)
				content.WriteString(line + "\n")
			}
		}
	} else if len(dv.initTransactions) == 0 {
		// No transactions yet - show status
		content.WriteString(headerStyle.Render("⏳ Init Transactions") + "\n\n")

		// Show appropriate message based on mode and network
		if dv.network == "emulator" {
			if dv.interactiveMode && dv.selectedInitFolder == "" {
				// Interactive mode and no folder selected yet
				content.WriteString(dimStyle.Render("Waiting to select folder..."))
			} else {
				// Non-interactive mode or folder already selected
				folderDisplay := "root"
				if dv.selectedInitFolder != "" {
					folderDisplay = dv.selectedInitFolder
				}
				content.WriteString(dimStyle.Render("Folder: "+folderDisplay) + "\n\n")
				content.WriteString(dimStyle.Render("Waiting for init transactions..."))
			}
		} else {
			content.WriteString(dimStyle.Render("Not available on " + dv.network))
		}
	} else {
		// Show init transaction results
		successCount := 0
		for _, tx := range dv.initTransactions {
			if tx.Success {
				successCount++
			}
		}

		header := fmt.Sprintf("Init Transactions (%d/%d)", successCount, len(dv.initTransactions))
		content.WriteString(headerStyle.Render(header) + "\n\n")

		// Show recent transactions (limit to fit in box)
		maxVisible := (height - 6) / 2 // Rough estimate of lines per transaction
		startIdx := 0
		if len(dv.initTransactions) > maxVisible {
			startIdx = len(dv.initTransactions) - maxVisible
		}

		for i := startIdx; i < len(dv.initTransactions); i++ {
			tx := dv.initTransactions[i]
			status := "✓"
			statusColor := successColor
			if !tx.Success {
				status = "✗"
				statusColor = errorColor
			}

			line := fmt.Sprintf("%s %s\n",
				lipgloss.NewStyle().Foreground(statusColor).Render(status),
				valueStyle.Render(tx.Filename),
			)
			content.WriteString(line)

			if !tx.Success && tx.Error != "" {
				content.WriteString(dimStyle.Render("  "+tx.Error) + "\n")
			}
		}
	}

	return boxStyle.Render(content.String())
}

// renderBlockHeightBox renders the latest block height box
func (dv *DashboardView) renderBlockHeightBox(width, height int) string {
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(successColor).
		PaddingLeft(1).
		PaddingRight(1)

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(successColor).
		Background(base02).
		PaddingLeft(1).
		PaddingRight(1).
		MarginBottom(1)

	var content strings.Builder

	// Header
	content.WriteString(headerStyle.Render("📦 Latest Block") + "\n\n")

	// Network info
	content.WriteString(labelStyle.Render("Network: ") + valueStyle.Render(dv.network) + "\n")

	// Only show block time for emulator (it's configurable there)
	if dv.network == "emulator" {
		content.WriteString(dimStyle.Render(fmt.Sprintf("Block time: %s", dv.blockTime)) + "\n")
	}

	content.WriteString(dimStyle.Render(fmt.Sprintf("Polling: %s", dv.indexerPolling)) + "\n\n")

	// Block height
	if dv.latestBlockHeight == 0 {
		content.WriteString(dimStyle.Render("Waiting for first block..."))
	} else {
		blockStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(successColor)

		content.WriteString(blockStyle.Render(fmt.Sprintf("Height: %d", dv.latestBlockHeight)))
		content.WriteString("\n\n")
		content.WriteString(dimStyle.Render("Blockchain is live"))
	}

	return boxStyle.Render(content.String())
}

// renderFrontendBox renders the frontend status box
func (dv *DashboardView) renderFrontendBox(width, height int) string {
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(tertiaryColor).
		PaddingLeft(1).
		PaddingRight(1)

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(tertiaryColor).
		Background(base02).
		PaddingLeft(1).
		PaddingRight(1).
		MarginBottom(1)

	var content strings.Builder

	content.WriteString(headerStyle.Render("🌐 Frontend") + "\n\n")

	if dv.frontendCommand == "" {
		content.WriteString(dimStyle.Render("Not configured"))
	} else {
		content.WriteString(labelStyle.Render("Command: ") + valueStyle.Render(dv.frontendCommand) + "\n")
		content.WriteString(labelStyle.Render("Status: ") + valueStyle.Render(dv.frontendStatus) + "\n")

		// Show detected ports
		if len(dv.frontendPorts) > 0 {
			content.WriteString("\n" + labelStyle.Render("Ports: ") + "\n")
			for _, port := range dv.frontendPorts {
				content.WriteString(fmt.Sprintf("• %s\n", valueStyle.Render(port)))
			}
		} else {
			content.WriteString(dimStyle.Render("No ports detected yet"))
		}
	}

	return boxStyle.Render(content.String())
}


// Name implements TabbedModel interface
func (dv *DashboardView) Name() string {
	return "Dashboard"
}

// KeyMap implements TabbedModel interface
func (dv *DashboardView) KeyMap() help.KeyMap {
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
func (dv *DashboardView) FooterView() string {
	return ""
}

// IsCapturingInput implements TabbedModel interface
func (dv *DashboardView) IsCapturingInput() bool {
	// Capture input when folder selection is active
	return dv.folderSelection != nil
}
