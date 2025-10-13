package ui

import (
	"fmt"
	"strings"

	"github.com/bjartek/aether/pkg/logs"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// LogsKeyMap defines keybindings for the logs view
type LogsKeyMap struct {
	LineUp   key.Binding
	LineDown key.Binding
	GotoTop  key.Binding
	GotoEnd  key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	Filter   key.Binding
}

// DefaultLogsKeyMap returns the default keybindings for logs view
func DefaultLogsKeyMap() LogsKeyMap {
	return LogsKeyMap{
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
		PageUp: key.NewBinding(
			key.WithKeys("ctrl+u", "pgup"),
			key.WithHelp("ctrl+u/pgup", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("ctrl+d", "pgdown"),
			key.WithHelp("ctrl+d/pgdn", "page down"),
		),
		Filter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filter"),
		),
	}
}

// LogsView manages the logs display with a scrollable viewport.
type LogsView struct {
	viewport      viewport.Model
	filterInput   textinput.Model
	lines         []string
	filteredLines []string
	maxLines      int
	keys          LogsKeyMap
	ready         bool
	autoScroll    bool
	filterMode    bool
	filterText    string
	headerHeight  int // Cached header height for viewport calculation
	totalHeight   int // Total height available for the entire logs view
}

// NewLogsView creates a new logs view.
func NewLogsView() *LogsView {
	filterInput := textinput.New()
	filterInput.Placeholder = "Filter logs..."
	filterInput.CharLimit = 100
	filterInput.Width = 50
	
	lv := &LogsView{
		lines:         make([]string, 0),
		filteredLines: make([]string, 0),
		maxLines:      10000, // Keep last 10000 lines in memory
		keys:          DefaultLogsKeyMap(),
		ready:         false,
		autoScroll:    true, // Start with auto-scroll enabled
		filterInput:   filterInput,
		filterMode:    false,
		filterText:    "",
		headerHeight:  2, // Initialize with default header height
	}
	
	return lv
}

func (lv *LogsView) Init() tea.Cmd {
	return nil
}

// applyFilter filters log lines based on current filter text
func (lv *LogsView) applyFilter() {
	if lv.filterText == "" {
		// No filter, show all lines
		lv.filteredLines = lv.lines
	} else {
		// Filter by substring match (case-insensitive)
		lv.filteredLines = make([]string, 0)
		filterLower := strings.ToLower(lv.filterText)
		
		for _, line := range lv.lines {
			if strings.Contains(strings.ToLower(line), filterLower) {
				lv.filteredLines = append(lv.filteredLines, line)
			}
		}
	}
}

// updateHeaderHeight calculates the header height based on current state
func (lv *LogsView) updateHeaderHeight() {
	// Render the header components exactly as they appear in View() to measure actual height
	headerStyle := lipgloss.NewStyle().
		Foreground(primaryColor).
		Bold(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(mutedColor)
	
	header := headerStyle.Render("Application Logs") + "\n"
	
	// Show filter bar if in filter mode
	var filterBar string
	if lv.filterMode {
		filterStyle := lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true)
		filterBar = filterStyle.Render("Filter: ") + lv.filterInput.View() + "\n"
	} else if lv.filterText != "" {
		// Show active filter indicator
		filterStyle := lipgloss.NewStyle().
			Foreground(mutedColor)
		matchCount := len(lv.filteredLines)
		totalCount := len(lv.lines)
		filterBar = filterStyle.Render(fmt.Sprintf("Filter: '%s' (%d/%d lines) • Press / to edit, Esc to clear", lv.filterText, matchCount, totalCount)) + "\n"
	}
	
	// Measure actual rendered height (header + optional filter bar)
	fullHeader := header + filterBar
	lv.headerHeight = lipgloss.Height(fullHeader)
}

// Update handles messages for the logs view.
func (lv *LogsView) Update(msg tea.Msg, width, height int) tea.Cmd {
	var cmd tea.Cmd
	
	// Always store the current height for viewport size validation
	lv.totalHeight = height

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle filter mode
		if lv.filterMode {
			switch {
			case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
				// Exit filter mode and clear filter
				lv.filterMode = false
				lv.filterText = ""
				lv.filterInput.SetValue("")
				lv.applyFilter()
				lv.updateHeaderHeight()
				if lv.ready {
					lv.viewport.Height = height - lv.headerHeight
					lv.viewport.SetContent(strings.Join(lv.filteredLines, "\n"))
				}
				return nil
			case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
				// Apply filter and exit filter mode
				lv.filterMode = false
				lv.filterText = lv.filterInput.Value()
				lv.applyFilter()
				lv.updateHeaderHeight()
				if lv.ready {
					lv.viewport.Height = height - lv.headerHeight
					lv.viewport.SetContent(strings.Join(lv.filteredLines, "\n"))
				}
				return nil
			default:
				// Pass input to filter textinput
				lv.filterInput, cmd = lv.filterInput.Update(msg)
				// Update filter in real-time
				lv.filterText = lv.filterInput.Value()
				lv.applyFilter()
				if lv.ready {
					lv.viewport.SetContent(strings.Join(lv.filteredLines, "\n"))
				}
				return cmd
			}
		}
		
		// Handle filter activation
		if key.Matches(msg, lv.keys.Filter) {
			lv.filterMode = true
			lv.filterInput.Focus()
			lv.updateHeaderHeight()
			// Resize viewport to account for filter bar
			if lv.ready {
				lv.viewport.Height = height - lv.headerHeight
			}
			return textinput.Blink
		}
		
	case logs.LogLineMsg:
		if msg.Err != nil {
			// Handle error - could add error display
			return nil
		}

		if msg.Line != "" {
			// Check if we're at the bottom before adding new content
			atBottom := false
			if lv.ready {
				atBottom = lv.viewport.AtBottom()
			}

			// Add new line, converting escaped newlines to actual newlines for readability
			line := strings.TrimRight(msg.Line, "\n")
			// Replace literal \n, \t with actual newlines and tabs for better formatting
			line = strings.ReplaceAll(line, "\\n", "\n")
			line = strings.ReplaceAll(line, "\\t", "  ") // Convert tabs to spaces for consistent display
			
			lv.lines = append(lv.lines, line)

			// Keep only the last maxLines
			if len(lv.lines) > lv.maxLines {
				lv.lines = lv.lines[len(lv.lines)-lv.maxLines:]
			}

			// Apply filter and update viewport content
			lv.applyFilter()
			if lv.ready {
				lv.viewport.SetContent(strings.Join(lv.filteredLines, "\n"))
				// Only auto-scroll if we were at the bottom
				if atBottom {
					lv.viewport.GotoBottom()
				}
			}
		}

		return nil

	case tea.WindowSizeMsg:
		// Calculate header height including filter bar if active
		lv.updateHeaderHeight()
		viewportHeight := height - lv.headerHeight
		
		// Ensure viewport height is at least 1 to avoid negative or zero height
		if viewportHeight < 1 {
			viewportHeight = 1
		}
		
		if !lv.ready {
			lv.viewport = viewport.New(width, viewportHeight)
			// Set explicit style to prevent any unexpected rendering
			lv.viewport.Style = lipgloss.NewStyle()
			lv.applyFilter()
			lv.viewport.SetContent(strings.Join(lv.filteredLines, "\n"))

			// Configure viewport key bindings to match our custom keys
			lv.viewport.KeyMap = viewport.KeyMap{
				PageDown: lv.keys.PageDown,
				PageUp:   lv.keys.PageUp,
				Down:     lv.keys.LineDown,
				Up:       lv.keys.LineUp,
			}

			lv.ready = true
		} else {
			lv.viewport.Width = width
			lv.viewport.Height = viewportHeight
		}

	}

	// Update viewport (handles all key navigation including our custom bindings) - only if not in filter mode
	if !lv.filterMode {
		lv.viewport, cmd = lv.viewport.Update(msg)
	}
	return cmd
}

// View renders the logs view.
// TODO: KNOWN ISSUE - Header scrolls away when logs overflow
// Attempted fixes:
// - Removed vertical padding from contentStyle
// - Used lipgloss.Place to constrain height
// - Set viewport height = totalHeight - headerHeight
// - Added explicit viewport.Style
// The viewport.View() may be returning more lines than viewport.Height
// or the terminal is scrolling the entire output despite size constraints.
// Consider: using a table instead of viewport, or investigating bubbletea viewport internals
func (lv *LogsView) View() string {
	if !lv.ready {
		return "Loading logs..."
	}

	if len(lv.lines) == 0 {
		return lipgloss.NewStyle().
			Foreground(mutedColor).
			Render("Waiting for log entries...")
	}

	// Render sticky header
	headerStyle := lipgloss.NewStyle().
		Foreground(primaryColor).
		Bold(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(mutedColor)
	
	header := headerStyle.Render("Application Logs") + "\n"

	// Show filter input if in filter mode
	var filterBar string
	if lv.filterMode {
		filterStyle := lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true)
		filterBar = filterStyle.Render("Filter: ") + lv.filterInput.View() + "\n"
	} else if lv.filterText != "" {
		// Show active filter indicator
		filterStyle := lipgloss.NewStyle().
			Foreground(mutedColor)
		matchCount := len(lv.filteredLines)
		totalCount := len(lv.lines)
		filterBar = filterStyle.Render(fmt.Sprintf("Filter: '%s' (%d/%d lines) • Press / to edit, Esc to clear", lv.filterText, matchCount, totalCount)) + "\n"
	}

	// Get the viewport content - this is the scrollable portion
	// The viewport internally manages which lines are visible based on its height and YOffset
	viewportContent := lv.viewport.View()
	
	// Combine header, filter bar, and viewport
	fullContent := header + filterBar + viewportContent
	
	// CRITICAL FIX: Force content into exact height box using Place
	// This absolutely prevents overflow by truncating if needed
	if lv.totalHeight > 0 && lv.viewport.Width > 0 {
		// Use Place to put content in an exact-sized box
		// If content is too tall, it will be truncated from bottom
		fullContent = lipgloss.Place(
			lv.viewport.Width,
			lv.totalHeight,
			lipgloss.Left,
			lipgloss.Top,
			fullContent,
		)
	}
	
	return fullContent
}

// Stop is a no-op for the logs view (kept for interface compatibility).
func (lv *LogsView) Stop() {
	// No cleanup needed
}
