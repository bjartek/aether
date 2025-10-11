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
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// EventData holds event information for display
type EventData struct {
	Name            string
	BlockHeight     uint64
	BlockID         string
	TransactionID   string
	TransactionIndex int
	EventIndex      int
	Fields          map[string]interface{}
	Timestamp       time.Time
	preRenderedDetail string // Cached detail text for performance
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
			key.WithKeys("tab"),
			key.WithHelp("tab", "toggle full detail"),
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
	mu               sync.RWMutex
	table            table.Model
	detailViewport   viewport.Model
	filterInput      textinput.Model
	keys             EventsKeyMap
	ready            bool
	events           []EventData
	filteredEvents   []EventData
	maxEvents        int
	width            int
	height           int
	fullDetailMode   bool   // Toggle between split and full-screen detail view
	showRawAddresses bool   // Toggle showing raw addresses vs friendly names
	filterMode       bool   // Whether filter input is active
	filterText       string // Current filter text
	accountRegistry  *aether.AccountRegistry
}

// NewEventsView creates a new events view
func NewEventsView() *EventsView {
	columns := []table.Column{
		{Title: "Block", Width: 6},
		{Title: "TX Hash", Width: 20},
		{Title: "Event #", Width: 8},
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

	// Create viewport for detail view
	vp := viewport.New(0, 0)
	vp.Style = lipgloss.NewStyle()

	// Create filter input
	filterInput := textinput.New()
	filterInput.Placeholder = "Filter by event name..."
	filterInput.CharLimit = 50
	filterInput.Width = 50

	const maxEvents = 1000

	return &EventsView{
		table:            t,
		detailViewport:   vp,
		filterInput:      filterInput,
		keys:             DefaultEventsKeyMap(),
		ready:            false,
		events:           make([]EventData, 0, maxEvents),
		filteredEvents:   make([]EventData, 0),
		maxEvents:        maxEvents,
		fullDetailMode:   false,
		showRawAddresses: false, // Show friendly names by default
		filterMode:       false,
		filterText:       "",
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

// formatEventFieldValue formats an event field value for display
func (ev *EventsView) formatEventFieldValue(val interface{}) string {
	switch v := val.(type) {
	case []uint8:
		// Convert uint8 array to hex string if in human-friendly mode
		if !ev.showRawAddresses && len(v) > 0 {
			return "0x" + fmt.Sprintf("%x", v)
		}
		return fmt.Sprintf("%v", v)
	case []interface{}:
		// Check if it's an array of numbers (likely a byte array)
		if len(v) > 0 && !ev.showRawAddresses {
			// Try to convert to bytes
			bytes := make([]byte, 0, len(v))
			isBytes := true
			for _, item := range v {
				switch num := item.(type) {
				case float64:
					if num >= 0 && num <= 255 && num == float64(int(num)) {
						bytes = append(bytes, byte(num))
					} else {
						isBytes = false
					}
				case int:
					if num >= 0 && num <= 255 {
						bytes = append(bytes, byte(num))
					} else {
						isBytes = false
					}
				default:
					isBytes = false
				}
				if !isBytes {
					break
				}
			}
			if isBytes && len(bytes) > 0 {
				return "0x" + fmt.Sprintf("%x", bytes)
			}
		}
		return fmt.Sprintf("%v", v)
	case map[string]interface{}:
		// Handle maps - format as key: value pairs with sorted keys
		if len(v) == 0 {
			return "{}"
		}
		// Sort keys for consistent ordering
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		var parts []string
		for _, k := range keys {
			// Recursively format map values
			formattedVal := ev.formatEventFieldValue(v[k])
			parts = append(parts, fmt.Sprintf("%s: %s", k, formattedVal))
		}
		return "{" + strings.Join(parts, ", ") + "}"
	case string:
		// Check if it's an address and format accordingly
		if !ev.showRawAddresses && ev.accountRegistry != nil && strings.HasPrefix(v, "0x") && len(v) == 18 {
			// For event fields, show only the friendly name
			return ev.accountRegistry.GetName(v)
		}
		return v
	default:
		valStr := fmt.Sprintf("%v", v)
		// Check if the string representation looks like an address
		if !ev.showRawAddresses && ev.accountRegistry != nil && strings.HasPrefix(valStr, "0x") && len(valStr) == 18 {
			// For event fields, show only the friendly name
			return ev.accountRegistry.GetName(valStr)
		}
		return valStr
	}
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

	// Pre-render asynchronously in background
	go func() {
		detail := ev.renderEventDetailText(eventData)
		ev.mu.Lock()
		// Find and update the event
		for i := range ev.events {
			if ev.events[i].TransactionID == eventData.TransactionID && 
			   ev.events[i].EventIndex == eventData.EventIndex {
				ev.events[i].preRenderedDetail = detail
				break
			}
		}
		ev.mu.Unlock()
	}()

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
		content := ev.events[selectedIdx].preRenderedDetail
		if content == "" {
			return
		}
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
		// Truncate transaction hash to show start and end
		txHash := truncateHex(event.TransactionID, 8, 8)
		
		rows[i] = table.Row{
			fmt.Sprintf("%d", event.BlockHeight),
			txHash,
			fmt.Sprintf("%d", event.EventIndex),
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
		// Split width: 60% table, 40% details (events have longer names)
		tableWidth := int(float64(width) * 0.6)
		ev.table.SetWidth(tableWidth)
		ev.table.SetHeight(height)

		// Update viewport size for full detail mode
		ev.detailViewport.Width = width
		ev.detailViewport.Height = height - 3

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
			for i := range ev.events {
				ev.events[i].preRenderedDetail = ev.renderEventDetailText(ev.events[i])
			}
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
			var cmd tea.Cmd
			ev.table, cmd = ev.table.Update(msg)
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

	// Split view mode - table on left, detail on right (60% table, 40% details)
	tableWidth := int(float64(ev.width) * 0.6)
	detailWidth := ev.width - tableWidth - 2

	selectedIdx := ev.table.Cursor()
	var detailView string
	if selectedIdx >= 0 && selectedIdx < len(ev.events) {
		detailView = ev.renderEventDetail(ev.events[selectedIdx], detailWidth, ev.height)
	} else {
		detailView = lipgloss.NewStyle().
			Width(detailWidth).
			Height(ev.height).
			Render("No event selected")
	}

	// Style table
	tableView := lipgloss.NewStyle().
		Width(tableWidth).
		Height(ev.height).
		Render(ev.table.View())

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
func (ev *EventsView) renderEventDetailText(event EventData) string {
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

			// Format value using helper function
			valStr := ev.formatEventFieldValue(val)

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
func (ev *EventsView) renderEventDetail(event EventData, width, height int) string {
	detailStyle := lipgloss.NewStyle().
		Width(width).
		Height(height).
		Padding(1)

	// Render a condensed version that fits in the available height
	content := ev.renderEventDetailCondensed(event, height-2)
	return detailStyle.Render(content)
}

// renderEventDetailCondensed renders a height-aware condensed version
func (ev *EventsView) renderEventDetailCondensed(event EventData, maxLines int) string {
	fieldStyle := lipgloss.NewStyle().Bold(true).Foreground(secondaryColor)
	valueStyleDetail := lipgloss.NewStyle().Foreground(accentColor)

	renderField := func(label, value string) string {
		return fieldStyle.Render(fmt.Sprintf("%-18s", label+":")) + " " + valueStyleDetail.Render(value) + "\n"
	}

	var details strings.Builder
	lineCount := 0

	// Title
	details.WriteString(fieldStyle.Render("Event Details") + "\n\n")
	lineCount += 2

	// Basic info
	details.WriteString(renderField("Event Name", event.Name))
	details.WriteString(renderField("Block Height", fmt.Sprintf("%d", event.BlockHeight)))
	details.WriteString(renderField("TX Index", fmt.Sprintf("%d", event.TransactionIndex)))
	details.WriteString(renderField("Event Index", fmt.Sprintf("%d", event.EventIndex)))
	lineCount += 4

	if lineCount+1 < maxLines {
		details.WriteString("\n")
		lineCount++
	}

	// Fields (show some if there's space)
	if len(event.Fields) > 0 && lineCount+3 < maxLines {
		details.WriteString(fieldStyle.Render(fmt.Sprintf("Fields (%d):", len(event.Fields))) + "\n")
		lineCount++

		// Sort keys
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

		fieldsShown := 0
		for _, key := range keys {
			if lineCount >= maxLines {
				break
			}
			val := event.Fields[key]
			paddedKey := fmt.Sprintf("%-*s", maxKeyLen, key)
			valStr := ev.formatEventFieldValue(val)

			details.WriteString(fmt.Sprintf("  %s: %s\n",
				valueStyleDetail.Render(paddedKey),
				valueStyleDetail.Render(valStr)))
			lineCount++
			fieldsShown++
		}

		if fieldsShown < len(event.Fields) {
			details.WriteString(fmt.Sprintf("  ... and %d more\n", len(event.Fields)-fieldsShown))
		}
	}

	// Hint to view full details
	if lineCount+2 < maxLines {
		details.WriteString("\n")
		details.WriteString(lipgloss.NewStyle().Foreground(mutedColor).Render("Press Tab for full details"))
	}

	return details.String()
}

// Stop is a no-op for the events view
func (ev *EventsView) Stop() {
	// No cleanup needed
}
