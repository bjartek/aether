package ui

import (
	"fmt"

	"github.com/bjartek/aether/pkg/aether"
	"github.com/bjartek/aether/pkg/config"
	"github.com/bjartek/aether/pkg/splitview"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// EventsViewV2 is the splitview-based implementation
type EventsViewV2 struct {
	sv               *splitview.SplitViewModel
	keys             EventsKeyMap
	width            int
	height           int
	accountRegistry  *aether.AccountRegistry
	showRawAddresses bool
	timeFormat       string             // Time format from config
	events           []aether.EventData // Store original data for rebuilding
}

// NewEventsViewV2WithConfig creates a new v2 events view based on splitview
func NewEventsViewV2WithConfig(cfg *config.Config) *EventsViewV2 {
	// Fallback to defaults when cfg is nil
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	columns := []splitview.ColumnConfig{
		{Name: "Time", Width: 8},   // Execution time
		{Name: "Block", Width: 5},  // Block height
		{Name: "Idx", Width: 3},    // Event index
		{Name: "Tx", Width: 9},     // Transaction ID (truncated)
		{Name: "Event", Width: 50}, // Event name
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
		splitview.WithTableSplitPercent(float64(cfg.UI.Layout.Events.TableWidthPercent)/100.0),
	)

	return &EventsViewV2{
		sv:               sv,
		keys:             DefaultEventsKeyMap(),
		showRawAddresses: cfg.UI.Defaults.ShowRawAddresses,
		timeFormat:       cfg.UI.Defaults.TimeFormat,
	}
}

// Init returns the init command for inner splitview
func (ev *EventsViewV2) Init() tea.Cmd { return ev.sv.Init() }

// Update implements tea.Model interface - handles toggles then forwards to splitview
func (ev *EventsViewV2) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		ev.width = msg.Width
		ev.height = msg.Height

	case tea.KeyMsg:
		// Handle toggle keys before forwarding to splitview
		switch {
		case key.Matches(msg, ev.keys.ToggleRawAddresses):
			ev.showRawAddresses = !ev.showRawAddresses
			// Refresh all rows to update table and detail
			ev.refreshAllRows()
			return ev, InputHandled()
		}
	}

	_, cmd := ev.sv.Update(msg)
	return ev, cmd
}

// View delegates to splitview
func (ev *EventsViewV2) View() string {
	return ev.sv.View()
}

// Name implements TabbedModel interface
func (ev *EventsViewV2) Name() string {
	return "Events"
}

// KeyMap implements TabbedModel interface
func (ev *EventsViewV2) KeyMap() help.KeyMap {
	return eventsKeyMapAdapter{
		splitviewKeys: ev.sv.KeyMap(),
		toggleKeys:    ev.keys,
	}
}

// eventsKeyMapAdapter combines splitview and toggle keys
type eventsKeyMapAdapter struct {
	splitviewKeys help.KeyMap
	toggleKeys    EventsKeyMap
}

func (k eventsKeyMapAdapter) ShortHelp() []key.Binding {
	// Combine splitview short help with toggle keys
	svHelp := k.splitviewKeys.ShortHelp()
	return append(svHelp, k.toggleKeys.ToggleRawAddresses)
}

func (k eventsKeyMapAdapter) FullHelp() [][]key.Binding {
	// Get splitview full help
	svHelp := k.splitviewKeys.FullHelp()

	// Add toggle keys as a new row
	toggleRow := []key.Binding{
		k.toggleKeys.ToggleRawAddresses,
	}

	return append(svHelp, toggleRow)
}

// FooterView implements TabbedModel interface
func (ev *EventsViewV2) FooterView() string {
	// No custom footer for events view
	return ""
}

// IsCapturingInput implements TabbedModel interface
func (ev *EventsViewV2) IsCapturingInput() bool {
	// Events view doesn't capture input
	return false
}

// SetAccountRegistry sets the account registry for friendly name resolution
func (ev *EventsViewV2) SetAccountRegistry(registry *aether.AccountRegistry) {
	ev.accountRegistry = registry
}

// AddEvent accepts EventData and converts it to a splitview row
func (ev *EventsViewV2) AddEvent(eventData aether.EventData) {
	// Store event data for rebuilding
	ev.events = append(ev.events, eventData)

	// Build and add row
	ev.addEventRow(eventData)
}

// addEventRow builds a splitview row from event data
func (ev *EventsViewV2) addEventRow(eventData aether.EventData) {
	row := table.Row{
		eventData.Timestamp.Format(ev.timeFormat),
		fmt.Sprintf("%d", eventData.BlockHeight),
		fmt.Sprintf("%d", eventData.EventIndex),
		truncateHex(eventData.TransactionID, 3, 3),
		eventData.Name,
	}

	// Build detail content
	content := buildEventDetailContent(eventData, ev.accountRegistry, ev.showRawAddresses)

	// Add to splitview (events don't have code)
	ev.sv.AddRow(splitview.NewRowData(row).WithContent(content))
}

// refreshAllRows rebuilds all rows to reflect toggle changes
func (ev *EventsViewV2) refreshAllRows() {
	if len(ev.events) == 0 {
		return
	}

	// Clear existing rows and rebuild from stored event data
	ev.sv.SetRows([]splitview.RowData{})

	// Rebuild all rows with current toggle states
	for _, eventData := range ev.events {
		ev.addEventRow(eventData)
	}
}
