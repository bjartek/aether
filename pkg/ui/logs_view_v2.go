package ui

import (
	"fmt"
	"strings"

	"github.com/bjartek/aether/pkg/config"
	"github.com/bjartek/aether/pkg/logs"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// LogsKeyMapV2 defines keybindings for the logs view v2
type LogsKeyMapV2 struct {
	LineUp   key.Binding
	LineDown key.Binding
	GotoTop  key.Binding
	GotoEnd  key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	Filter   key.Binding
	Confirm  key.Binding
	Cancel   key.Binding
}

// DefaultLogsKeyMapV2 returns the default keybindings for logs view v2
func DefaultLogsKeyMapV2() LogsKeyMapV2 {
	return LogsKeyMapV2{
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
		Confirm: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "confirm"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel/clear filter"),
		),
	}
}

// LogsViewV2 implements standard tea.Model interface
type LogsViewV2 struct {
	viewport      viewport.Model
	filterInput   textinput.Model
	lines         []string
	filteredLines []string
	maxLines      int
	keys          LogsKeyMapV2
	ready         bool
	autoScroll    bool
	filterMode    bool
	filterText    string
	headerHeight  int
	width         int
	height        int
}

// NewLogsViewV2WithConfig creates a new v2 logs view with configuration
func NewLogsViewV2WithConfig(cfg *config.Config) *LogsViewV2 {
	filterInput := textinput.New()
	filterInput.Placeholder = "Filter logs..."
	filterInput.CharLimit = 100
	filterInput.Width = 50

	// Get max log lines from config or use default
	maxLines := 10000
	if cfg != nil {
		maxLines = cfg.UI.History.MaxLogLines
	}

	return &LogsViewV2{
		lines:         make([]string, 0),
		filteredLines: make([]string, 0),
		maxLines:      maxLines,
		keys:          DefaultLogsKeyMapV2(),
		ready:         false,
		autoScroll:    true,
		filterInput:   filterInput,
		filterMode:    false,
		filterText:    "",
		headerHeight:  2,
	}
}

// Init implements tea.Model
func (lv *LogsViewV2) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model interface
func (lv *LogsViewV2) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		lv.width = msg.Width
		lv.height = msg.Height
		lv.viewport.Width = lv.width
		lv.viewport.Height = lv.height
		lv.ready = true
		lv.updateViewport()
		return lv, nil

	case logs.LogLineMsg:
		lv.lines = append(lv.lines, msg.Line)
		if len(lv.lines) > lv.maxLines {
			lv.lines = lv.lines[len(lv.lines)-lv.maxLines:]
		}
		lv.applyFilter()
		lv.updateViewport()
		if lv.autoScroll {
			lv.viewport.GotoBottom()
		}
		return lv, nil

	case tea.KeyMsg:
		if lv.filterMode {
			switch {
			case key.Matches(msg, lv.keys.Confirm):
				lv.filterMode = false
				lv.filterText = lv.filterInput.Value()
				lv.applyFilter()
				lv.updateViewport()
				return lv, nil
			case key.Matches(msg, lv.keys.Cancel):
				lv.filterMode = false
				lv.filterInput.SetValue(lv.filterText)
				return lv, nil
			default:
				var cmd tea.Cmd
				lv.filterInput, cmd = lv.filterInput.Update(msg)
				return lv, cmd
			}
		} else {
			switch {
			case key.Matches(msg, lv.keys.Filter):
				lv.filterMode = true
				lv.filterInput.Focus()
				return lv, nil
			case key.Matches(msg, lv.keys.Cancel) && lv.filterText != "":
				lv.filterText = ""
				lv.filterInput.SetValue("")
				lv.applyFilter()
				lv.updateViewport()
				return lv, nil
			}
		}
	}

	// Forward to viewport if not in filter mode
	if !lv.filterMode {
		var cmd tea.Cmd
		lv.viewport, cmd = lv.viewport.Update(msg)
		return lv, cmd
	}

	return lv, nil
}

// View implements tea.Model
func (lv *LogsViewV2) View() string {
	if !lv.ready {
		return "Initializing logs..."
	}

	// Filter bar is now handled by FooterView()
	return lv.viewport.View()
}

// applyFilter filters log lines based on current filter text
func (lv *LogsViewV2) applyFilter() {
	if lv.filterText == "" {
		lv.filteredLines = lv.lines
	} else {
		lv.filteredLines = make([]string, 0)
		filterLower := strings.ToLower(lv.filterText)

		for _, line := range lv.lines {
			if strings.Contains(strings.ToLower(line), filterLower) {
				lv.filteredLines = append(lv.filteredLines, line)
			}
		}
	}
}

// updateViewport updates the viewport content
func (lv *LogsViewV2) updateViewport() {
	// Log lines already include newlines, so just concatenate them
	content := strings.Join(lv.filteredLines, "")
	lv.viewport.SetContent(content)
}

// Stop is a no-op for logs view
func (lv *LogsViewV2) Stop() {
	// No cleanup needed
}

// Name implements TabbedModel interface
func (lv *LogsViewV2) Name() string {
	return "Logs"
}

// KeyMap implements TabbedModel interface
func (lv *LogsViewV2) KeyMap() help.KeyMap {
	// Return the logs key map (need to implement help.KeyMap interface)
	return logsKeyMapAdapter{lv.keys}
}

// FooterView implements TabbedModel interface
func (lv *LogsViewV2) FooterView() string {
	// Show filter input when in filter mode
	if lv.filterMode {
		filterStyle := lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true)
		return filterStyle.Render("Filter: ") + lv.filterInput.View()
	}
	
	// Show filter status when filter is active but not editing
	if lv.filterText != "" {
		filterStyle := lipgloss.NewStyle().
			Foreground(mutedColor)
		matchCount := len(lv.filteredLines)
		totalCount := len(lv.lines)
		return filterStyle.Render(fmt.Sprintf("Filter: '%s' (%d/%d lines) • Press / to edit, Esc to clear", lv.filterText, matchCount, totalCount))
	}
	
	return ""
}

// IsCapturingInput implements TabbedModel interface
func (lv *LogsViewV2) IsCapturingInput() bool {
	// Capturing input when in filter mode
	return lv.filterMode
}

// logsKeyMapAdapter adapts LogsKeyMapV2 to help.KeyMap interface
type logsKeyMapAdapter struct {
	keys LogsKeyMapV2
}

func (k logsKeyMapAdapter) ShortHelp() []key.Binding {
	return []key.Binding{k.keys.Filter, k.keys.LineUp, k.keys.LineDown}
}

func (k logsKeyMapAdapter) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.keys.Filter, k.keys.Confirm, k.keys.Cancel},
		{k.keys.LineUp, k.keys.LineDown, k.keys.PageUp, k.keys.PageDown},
		{k.keys.GotoTop, k.keys.GotoEnd},
	}
}
