package ui

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/bjartek/aether/pkg/aether"
	"github.com/bjartek/aether/pkg/config"
	"github.com/bjartek/overflow/v2"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// EventData holds event information for display
type EventData struct {
	Name              string
	BlockHeight       uint64
	BlockID           string
	TransactionID     string
	TransactionIndex  int
	EventIndex        int
	Fields    map[string]interface{}
	Timestamp time.Time
}

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

// EventsView manages the events table and detail display
type EventsView struct {
	mu                  sync.RWMutex
	table               table.Model
	detailViewport      viewport.Model // For full detail mode
	splitDetailViewport viewport.Model // For split view detail panel
	filterInput         textinput.Model
	keys                EventsKeyMap
	ready               bool
	events              []EventData
	filteredEvents      []EventData
	maxEvents           int
	width               int
	height              int
	tableWidthPercent   int    // Configurable table width percentage
	detailWidthPercent  int    // Configurable detail width percentage
	fullDetailMode      bool   // Toggle between split and full-screen detail view
	showRawAddresses    bool   // Toggle showing raw addresses vs friendly names
	filterMode          bool   // Whether filter input is active
	filterText          string // Current filter text
	accountRegistry     *aether.AccountRegistry
}

// NewEventsView creates a new events view with default settings
func NewEventsView() *EventsView {
	return NewEventsViewWithConfig(nil)
}

// NewEventsViewWithConfig creates a new events view with configuration
func NewEventsViewWithConfig(cfg *config.Config) *EventsView {
	columns := []table.Column{
		{Title: "Time", Width: 8}, // Execution time
		{Title: "Block", Width: 6},
		{Title: "#", Width: 3},  // Event index
		{Title: "TX", Width: 9}, // Transaction hash (first 3 + ... + last 3)
		{Title: "Event Name", Width: 70},
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

	// Create viewport for full detail mode
	vp := viewport.New(0, 0)
	vp.Style = lipgloss.NewStyle()

	// Create viewport for split view detail panel
	splitVp := viewport.New(0, 0)
	splitVp.Style = lipgloss.NewStyle()

	// Create filter input
	filterInput := textinput.New()
	filterInput.Placeholder = "Filter by event name..."
	if cfg != nil {
		filterInput.CharLimit = cfg.UI.Filter.CharLimit
		filterInput.Width = cfg.UI.Filter.Width
	} else {
		filterInput.CharLimit = 50
		filterInput.Width = 50
	}

	// Get max events from config or use default
	maxEvents := 10000
	if cfg != nil {
		maxEvents = cfg.UI.History.MaxEvents
	}

	// Get default display modes and layout from config
	// Use config defaults, or fallback to DefaultConfig if no config provided
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	
	showRawAddresses := cfg.UI.Defaults.ShowRawAddresses
	fullDetailMode := cfg.UI.Defaults.FullDetailMode
	tableWidthPercent := cfg.UI.Layout.Events.TableWidthPercent
	detailWidthPercent := cfg.UI.Layout.Events.DetailWidthPercent

	return &EventsView{
		table:               t,
		detailViewport:      vp,
		splitDetailViewport: splitVp,
		filterInput:         filterInput,
		keys:                DefaultEventsKeyMap(),
		ready:               false,
		events:              make([]EventData, 0, maxEvents),
		filteredEvents:      make([]EventData, 0),
		maxEvents:           maxEvents,
		tableWidthPercent:   tableWidthPercent,
		detailWidthPercent:  detailWidthPercent,
		fullDetailMode:      fullDetailMode,
		showRawAddresses:    showRawAddresses,
		filterMode:          false,
		filterText:          "",
	}
}

func (ev *EventsView) Init() tea.Cmd {
	return nil
}

// truncateString truncates a string to maxLen
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// AddEvent adds a new event from transaction processing
func (ev *EventsView) AddEvent(blockHeight uint64, blockID string, txID string, txIndex int, event overflow.OverflowEvent, eventIndex int, registry *aether.AccountRegistry) {
	ev.mu.Lock()
	defer ev.mu.Unlock()

	// Store registry for use in rendering
	if registry != nil {
		ev.accountRegistry = registry
	}

	eventData := EventData{
		Name:             event.Name,
		BlockHeight:      blockHeight,
		BlockID:          blockID,
		TransactionID:    txID,
		TransactionIndex: txIndex,
		EventIndex:       eventIndex,
		Fields:           event.Fields,
		Timestamp:        time.Now(),
	}

	ev.events = append(ev.events, eventData)

	// No pre-rendering - render fresh on demand

	// Keep only the last maxEvents events
	if len(ev.events) > ev.maxEvents {
		ev.events = ev.events[len(ev.events)-ev.maxEvents:]
	}

	ev.refreshTable()
}

// updateDetailViewport updates the viewport content with current event details
func (ev *EventsView) updateDetailViewport() {
	if len(ev.events) == 0 {
		ev.detailViewport.SetContent("")
		return
	}

	selectedIdx := ev.table.Cursor()
	if selectedIdx >= 0 && selectedIdx < len(ev.events) {
		if ev.detailViewport.Width == 0 || ev.detailViewport.Height == 0 {
			return
		}
		// Render fresh
		content := ev.renderEventDetailText(ev.events[selectedIdx], ev.detailViewport.Width)
		ev.detailViewport.SetContent(content)
		ev.detailViewport.GotoTop()
	}
}

// applyFilter filters events based on current filter text
func (ev *EventsView) applyFilter() {
	if ev.filterText == "" {
		ev.filteredEvents = ev.events
	} else {
		ev.filteredEvents = make([]EventData, 0)
		filterLower := strings.ToLower(ev.filterText)

		for _, event := range ev.events {
			if strings.Contains(strings.ToLower(event.Name), filterLower) {
				ev.filteredEvents = append(ev.filteredEvents, event)
			}
		}
	}
}

// refreshTable updates the table rows from events
func (ev *EventsView) refreshTable() {
	// Use filtered list if filter is active
	eventList := ev.events
	if ev.filterText != "" {
		eventList = ev.filteredEvents
	}

	rows := make([]table.Row, len(eventList))
	for i, event := range eventList {
		// Truncate transaction hash to show first 3 and last 3 characters
		txHash := truncateHex(event.TransactionID, 3, 3)

		rows[i] = table.Row{
			event.Timestamp.Format("15:04:05"), // Show time only
			fmt.Sprintf("%d", event.BlockHeight),
			fmt.Sprintf("%d", event.EventIndex),
			txHash,
			truncateString(event.Name, 70),
		}
	}
	ev.table.SetRows(rows)
}

// Update handles messages for the events view
func (ev *EventsView) Update(msg tea.Msg, width, height int) tea.Cmd {
	ev.width = width
	ev.height = height

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if !ev.ready {
			ev.ready = true
		}
		// Split width using configured percentages
		tableWidth := int(float64(width) * float64(ev.tableWidthPercent) / 100.0)
		detailWidth := max(10, width-tableWidth-2)
		ev.table.SetWidth(tableWidth)
		ev.table.SetHeight(height)

		// Update viewport size for full detail mode
		// Calculate hint text height dynamically
		hint := lipgloss.NewStyle().
			Foreground(mutedColor).
			Render("Press Tab or Esc to return to table view | j/k to scroll")
		hintHeight := lipgloss.Height(hint) + 2 // +2 for spacing
		ev.detailViewport.Width = width
		ev.detailViewport.Height = height - hintHeight

		// Update split view detail viewport size
		ev.splitDetailViewport.Width = detailWidth
		ev.splitDetailViewport.Height = height

	case tea.KeyMsg:
		// Handle filter mode
		if ev.filterMode {
			switch {
			case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
				ev.filterMode = false
				ev.filterText = ""
				ev.filterInput.SetValue("")
				ev.mu.Lock()
				ev.applyFilter()
				ev.refreshTable()
				ev.mu.Unlock()
				return nil
			case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
				ev.filterMode = false
				ev.filterText = ev.filterInput.Value()
				ev.mu.Lock()
				ev.applyFilter()
				ev.refreshTable()
				ev.mu.Unlock()
				return nil
			default:
				var cmd tea.Cmd
				ev.filterInput, cmd = ev.filterInput.Update(msg)
				ev.filterText = ev.filterInput.Value()
				ev.mu.Lock()
				ev.applyFilter()
				ev.refreshTable()
				ev.mu.Unlock()
				return cmd
			}
		}

		// Handle filter activation
		if key.Matches(msg, ev.keys.Filter) {
			ev.filterMode = true
			ev.filterInput.Focus()
			return textinput.Blink
		}

		// Handle toggle full detail view
		if key.Matches(msg, ev.keys.ToggleFullDetail) {
			wasFullMode := ev.fullDetailMode
			ev.fullDetailMode = !ev.fullDetailMode
			if !wasFullMode && ev.fullDetailMode {
				ev.mu.RLock()
				ev.updateDetailViewport()
				ev.mu.RUnlock()
			}
			return nil
		}

		// Handle Esc to exit full detail view
		if ev.fullDetailMode && key.Matches(msg, key.NewBinding(key.WithKeys("esc"))) {
			ev.fullDetailMode = false
			return nil
		}

		// Handle toggle raw addresses
		if key.Matches(msg, ev.keys.ToggleRawAddresses) {
			ev.showRawAddresses = !ev.showRawAddresses
			ev.mu.Lock()
			if ev.fullDetailMode {
				ev.updateDetailViewport()
			}
			ev.mu.Unlock()
			return nil
		}

		// In full detail mode, pass keys to viewport for scrolling
		if ev.fullDetailMode {
			var cmd tea.Cmd
			ev.detailViewport, cmd = ev.detailViewport.Update(msg)
			return cmd
		} else {
			// Otherwise pass keys to table
			prevCursor := ev.table.Cursor()
			var cmd tea.Cmd
			ev.table, cmd = ev.table.Update(msg)
			
			// If cursor changed, update viewport content and reset scroll to top
			newCursor := ev.table.Cursor()
			if prevCursor != newCursor {
				ev.mu.RLock()
				ev.updateDetailViewport()
				ev.mu.RUnlock()
			}
			return cmd
		}
	}

	return nil
}

// View renders the events view
func (ev *EventsView) View() string {
	if !ev.ready {
		return "Loading events..."
	}

	ev.mu.RLock()
	defer ev.mu.RUnlock()

	if len(ev.events) == 0 {
		return lipgloss.NewStyle().
			Foreground(mutedColor).
			Render("Waiting for events...")
	}

	// Full detail mode - show only the event detail in viewport
	if ev.fullDetailMode {
		hint := lipgloss.NewStyle().
			Foreground(mutedColor).
			Render("Press Tab or Esc to return to table view | j/k to scroll")
		return hint + "\n\n" + ev.detailViewport.View()
	}

	// Show filter input if in filter mode
	var filterBar string
	if ev.filterMode {
		filterStyle := lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true)
		filterBar = filterStyle.Render("Filter: ") + ev.filterInput.View() + "\n"
	} else if ev.filterText != "" {
		filterStyle := lipgloss.NewStyle().
			Foreground(mutedColor)
		matchCount := len(ev.filteredEvents)
		filterBar = filterStyle.Render(fmt.Sprintf("Filter: '%s' (%d matches) • Press / to edit, Esc to clear", ev.filterText, matchCount)) + "\n"
	}

	// Split view mode - table on left, detail on right
	// Calculate widths using configured percentages
	tableWidth := int(float64(ev.width) * float64(ev.tableWidthPercent) / 100.0)

	// Update split detail viewport with current event
	selectedIdx := ev.table.Cursor()
	if selectedIdx >= 0 && selectedIdx < len(ev.events) {
		currentWidth := ev.splitDetailViewport.Width
		if currentWidth == 0 {
			currentWidth = 100 // Default
		}
		
		event := ev.events[selectedIdx]
		
		// Just render fresh every time - no caching, no optimization
		content := ev.renderEventDetailText(event, currentWidth)
		// Wrap with lipgloss for proper text flow
		wrappedContent := lipgloss.NewStyle().Width(currentWidth).Render(content)
		
		ev.splitDetailViewport.SetContent(wrappedContent)
		ev.splitDetailViewport.GotoTop()
	} else {
		ev.splitDetailViewport.SetContent("No event selected")
		ev.splitDetailViewport.GotoTop()
	}

	// Style table
	tableView := lipgloss.NewStyle().
		Width(tableWidth).
		MaxHeight(ev.height).
		Render(ev.table.View())

	// Render split detail viewport (viewport itself handles width/height constraints)
	detailView := ev.splitDetailViewport.View()

	// Combine table and detail side by side
	mainView := lipgloss.JoinHorizontal(
		lipgloss.Top,
		tableView,
		detailView,
	)

	if filterBar != "" {
		return filterBar + mainView
	}
	return mainView
}

// renderEventDetailText renders event details as plain text (for viewport)
// width specifies the maximum width for text wrapping (0 = no wrapping)
func (ev *EventsView) renderEventDetailText(event EventData, width int) string {
	fieldStyle := lipgloss.NewStyle().Bold(true).Foreground(secondaryColor)
	valueStyleDetail := lipgloss.NewStyle().Foreground(accentColor)

	renderField := func(label, value string) string {
		return fieldStyle.Render(fmt.Sprintf("%-18s", label+":")) + " " + valueStyleDetail.Render(value) + "\n"
	}

	var details strings.Builder
	details.WriteString(fieldStyle.Render("Event Details") + "\n\n")

	details.WriteString(renderField("Event Name", event.Name))
	details.WriteString(renderField("Block Height", fmt.Sprintf("%d", event.BlockHeight)))
	details.WriteString(renderField("Block ID", event.BlockID))
	details.WriteString(renderField("Transaction ID", event.TransactionID))
	details.WriteString(renderField("Transaction Index", fmt.Sprintf("%d", event.TransactionIndex)))
	details.WriteString(renderField("Event Index", fmt.Sprintf("%d", event.EventIndex)))
	details.WriteString("\n")

	if len(event.Fields) > 0 {
		details.WriteString(fieldStyle.Render(fmt.Sprintf("Fields (%d):", len(event.Fields))) + "\n")

		// Sort keys for consistent ordering
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

			// For nested structures, use simple indent
			// The lipgloss width wrapping will handle line breaks for simple values
			valStr := FormatFieldValueWithRegistry(val, "    ", ev.accountRegistry, ev.showRawAddresses, 0)
			details.WriteString(fmt.Sprintf("  %s: %s\n",
				valueStyleDetail.Render(paddedKey),
				valueStyleDetail.Render(valStr)))
		}
	} else {
		details.WriteString(fieldStyle.Render("No fields") + "\n")
	}

	return details.String()
}

// renderEventDetail renders the detailed view of an event (for split view)
// Uses the same full content as inspector view, just in a smaller viewport
func (ev *EventsView) renderEventDetail(event EventData, width, height int) string {
	// Render fresh
	content := ev.renderEventDetailText(event, width)
	
	// Just render with padding - don't constrain width to avoid clipping
	// The parent container will handle the width constraint
	detailStyle := lipgloss.NewStyle().
		Padding(1)

	return detailStyle.Render(content)
}

// Stop is a no-op for the events view
func (ev *EventsView) Stop() {
	// No cleanup needed
}
