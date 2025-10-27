package ui

import (
	"fmt"

	"github.com/bjartek/aether/pkg/aether"
	"github.com/bjartek/aether/pkg/config"
	"github.com/bjartek/aether/pkg/splitview"
	"github.com/bjartek/aether/pkg/tabbedtui"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rs/zerolog"
)

// EventsKeyMap defines keybindings for the events view
type EventsKeyMap struct {
	LineUp             key.Binding
	LineDown           key.Binding
	GotoTop            key.Binding
	GotoEnd            key.Binding
	ToggleFullDetail   key.Binding
	ToggleRawAddresses key.Binding
	Filter             key.Binding
}

// DefaultEventsKeyMap returns the default keybindings for events view
func DefaultEventsKeyMap() EventsKeyMap {
	return EventsKeyMap{
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
		ToggleRawAddresses: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "toggle raw addresses"),
		),
		Filter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filter"),
		),
	}
}

// EventsView is the splitview-based implementation
type EventsView struct {
	sv               *splitview.SplitViewModel
	keys             EventsKeyMap
	width            int
	height           int
	accountRegistry  *aether.AccountRegistry
	showRawAddresses bool
	timeFormat       string             // Time format from config
	events           []aether.EventData // Store original data for rebuilding
	logger           zerolog.Logger     // Debug logger
}

// NewEventsViewWithConfig creates a new v2 events view based on splitview
func NewEventsViewWithConfig(cfg *config.Config, logger zerolog.Logger) *EventsView {
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
		splitview.WithTableSplitPercent(float64(cfg.UI.Layout.EventsSplitPercent)/100.0),
		splitview.WithSortOrder(cfg.UI.Defaults.Sort),
	)

	return &EventsView{
		sv:               sv,
		keys:             DefaultEventsKeyMap(),
		showRawAddresses: cfg.UI.Defaults.ShowRawAddresses,
		timeFormat:       cfg.UI.Defaults.TimeFormat,
		logger:           logger,
	}
}

// Init returns the init command for inner splitview
func (ev *EventsView) Init() tea.Cmd { return ev.sv.Init() }

// Update implements tea.Model interface - handles toggles then forwards to splitview
func (ev *EventsView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	ev.logger.Debug().Str("method", "Update").Interface("msgType", msg).Msg("EventsView.Update called")
	
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		ev.width = msg.Width
		ev.height = msg.Height

	case aether.BlockEventMsg:
		// Handle incoming event data
		ev.AddEvent(msg.EventData)
		return ev, nil

	case aether.OverflowReadyMsg:
		// Set account registry when ready
		ev.SetAccountRegistry(msg.AccountRegistry)
		return ev, nil

	case tea.KeyMsg:
		// Handle toggle keys before forwarding to splitview
		switch {
		case key.Matches(msg, ev.keys.ToggleRawAddresses):
			ev.showRawAddresses = !ev.showRawAddresses
			// Refresh all rows to update table and detail
			ev.refreshAllRows()
			return ev, tabbedtui.InputHandled()
		}
	}

	_, cmd := ev.sv.Update(msg)
	return ev, cmd
}

// View delegates to splitview
func (ev *EventsView) View() string {
	ev.logger.Debug().Str("method", "View").Msg("EventsView.View called")
	return ev.sv.View()
}

// Name implements TabbedModel interface
func (ev *EventsView) Name() string {
	return "Events"
}

// KeyMap implements TabbedModel interface
func (ev *EventsView) KeyMap() help.KeyMap {
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
func (ev *EventsView) FooterView() string {
	// No custom footer for events view
	return ""
}

// IsCapturingInput implements TabbedModel interface
func (ev *EventsView) IsCapturingInput() bool {
	// Events view doesn't capture input
	return false
}

// SetAccountRegistry sets the account registry for friendly name resolution
func (ev *EventsView) SetAccountRegistry(registry *aether.AccountRegistry) {
	ev.accountRegistry = registry
}

// AddEvent accepts EventData and converts it to a splitview row
func (ev *EventsView) AddEvent(eventData aether.EventData) {
	// Store event data for rebuilding (always append to internal array)
	ev.events = append(ev.events, eventData)

	// Add row to splitview (it handles sort order internally)
	ev.addEventRow(eventData)
}

// addEventRow builds a splitview row from event data
func (ev *EventsView) addEventRow(eventData aether.EventData) {
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
func (ev *EventsView) refreshAllRows() {
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
