package splitview

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wrap"
)

const (
	defaultTableSplitPercent = 0.3
)

// ColumnConfig defines a table column with sort/filter capabilities
type ColumnConfig struct {
	Name      string
	Width     int
	SortKey   string
	FilterKey string
}

// RowData contains table row and associated detail content
type RowData struct {
	TableRow table.Row
	Code     string // Original highlighted code (empty = no code to show)
	Content  string // Text to show before code (title, metadata, etc)
}

// NewRowData creates a new RowData with the given table row
func NewRowData(tableRow table.Row) RowData {
	return RowData{
		TableRow: tableRow,
	}
}

// WithCode sets the code content for the row
func (r RowData) WithCode(code string) RowData {
	r.Code = code
	return r
}

// WithContent sets the content/description for the row
func (r RowData) WithContent(content string) RowData {
	r.Content = content
	return r
}

// SplitViewModel is a reusable split view: table on left, detail on right
type SplitViewModel struct {
	columns           []ColumnConfig
	rows              []RowData
	table             table.Model
	detailViewport    viewport.Model
	spinner           spinner.Model
	Keys              KeyMap
	tableSplitPercent float64
	width             int
	height            int
	fullDetailMode    bool

	// Cache highlighted code per row index
	codeFullscreenCache map[int]string
	codeDetailCache     map[int]string

	// Track when viewport content needs updating
	lastSelectedRow int
	lastWidth       int
	lastMode        bool
}

// KeyMap defines key bindings for the split view
type KeyMap struct {
	ToggleFullscreen key.Binding
	ExitFullscreen   key.Binding
	ToggleHelp       key.Binding
	Quit             key.Binding
}

// ShortHelp returns keybindings to be shown in the mini help view
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.ToggleHelp, k.ToggleFullscreen, k.Quit}
}

// FullHelp returns keybindings for the expanded help view
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.ToggleFullscreen, k.ExitFullscreen},
		{k.ToggleHelp, k.Quit},
	}
}

// NewKeyMap creates the default key bindings for the split view
func NewKeyMap() KeyMap {
	return KeyMap{
		ToggleFullscreen: key.NewBinding(
			key.WithKeys(" ", "enter"),
			key.WithHelp("space/enter", "toggle fullscreen"),
		),
		ExitFullscreen: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "exit fullscreen/help"),
		),
		ToggleHelp: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "toggle help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}

// Option is a functional option for configuring the split view
type Option func(*SplitViewModel)

// WithTableStyles sets the table styles
func WithTableStyles(styles table.Styles) Option {
	return func(m *SplitViewModel) {
		m.table.SetStyles(styles)
	}
}

// WithTableSplitPercent sets the percentage of width allocated to the table (0.0 to 1.0)
func WithTableSplitPercent(percent float64) Option {
	return func(m *SplitViewModel) {
		m.tableSplitPercent = percent
	}
}

// WithRows sets the initial rows for the split view
func WithRows(rows []RowData) Option {
	return func(m *SplitViewModel) {
		m.rows = rows
		m.updateTableRows()
	}
}

// NewSplitView creates a new split view model
func NewSplitView(columns []ColumnConfig, opts ...Option) *SplitViewModel {
	// Convert columns to table columns
	tableCols := make([]table.Column, len(columns))
	for i, col := range columns {
		tableCols[i] = table.Column{
			Title: col.Name,
			Width: col.Width,
		}
	}

	// Create table with default styles (no rows initially)
	t := table.New(
		table.WithColumns(tableCols),
		table.WithRows([]table.Row{}),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	// Create viewport
	vp := viewport.New(80, 20)

	// Create spinner
	s := spinner.New()
	s.Spinner = spinner.Dot

	m := &SplitViewModel{
		columns:             columns,
		rows:                []RowData{},
		table:               t,
		detailViewport:      vp,
		spinner:             s,
		Keys:                NewKeyMap(),
		tableSplitPercent:   defaultTableSplitPercent,
		width:               0, // Wait for WindowSizeMsg
		height:              0,
		codeFullscreenCache: make(map[int]string),
		codeDetailCache:     make(map[int]string),
		lastSelectedRow:     -1, // Force initial render
	}

	// Apply functional options
	for _, opt := range opts {
		opt(m)
	}

	// Don't pre-generate all code - do it lazily on first view/selection
	// This makes startup fast even with many rows

	return m
}

// Init implements tea.Model
func (m *SplitViewModel) Init() tea.Cmd {
	return m.spinner.Tick
}

// Update implements tea.Model
func (m *SplitViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle our custom keys first using key.Matches
		switch {
		case key.Matches(msg, m.Keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.Keys.ToggleFullscreen):
			m.fullDetailMode = !m.fullDetailMode
			return m, nil
		case key.Matches(msg, m.Keys.ExitFullscreen):
			if m.fullDetailMode {
				m.fullDetailMode = false
				return m, nil
			}
		}

		// Let components handle their own navigation with built-in keymaps
		if m.fullDetailMode {
			// In fullscreen, viewport handles all navigation (j/k/up/down/pgup/pgdn/etc)
			m.detailViewport, cmd = m.detailViewport.Update(msg)
		} else {
			// In split view, table handles all navigation
			m.table, cmd = m.table.Update(msg)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	}

	// Update spinner
	var spinnerCmd tea.Cmd
	m.spinner, spinnerCmd = m.spinner.Update(msg)
	if cmd == nil {
		cmd = spinnerCmd
	} else {
		cmd = tea.Batch(cmd, spinnerCmd)
	}

	return m, cmd
}

// wrapCode gets or generates highlighted code for the selected row
func (m *SplitViewModel) wrapCode(rowIndex int, width int, isFullscreen bool) string {
	if rowIndex < 0 || rowIndex >= len(m.rows) {
		return ""
	}

	cache := m.codeDetailCache
	if isFullscreen {
		cache = m.codeFullscreenCache
	}

	// Check cache
	if code, ok := cache[rowIndex]; ok {
		return code
	}

	// Generate and cache
	rawCode := m.rows[rowIndex].Code
	if rawCode == "" {
		return ""
	}

	highlighted := wrap.String(rawCode, width)
	cache[rowIndex] = highlighted
	return highlighted
}

// buildViewportContent creates the full content string for the viewport
func (m *SplitViewModel) buildViewportContent(width int, isFullscreen bool) string {
	selectedIdx := m.table.Cursor()
	if selectedIdx < 0 || selectedIdx >= len(m.rows) {
		return "No item selected"
	}

	// Simple unstyled rendering - parent can style CodePrefix if needed
	// Get row data
	rowData := m.rows[selectedIdx]

	// Build content: prefix + code (if present)
	content := ""
	if rowData.Content != "" {
		content += rowData.Content + "\n\n"
	}

	if rowData.Code != "" {
		code := m.wrapCode(selectedIdx, width, isFullscreen)
		content += code
	}

	return content
}

// AddRow adds a new row to the split view
func (m *SplitViewModel) AddRow(row RowData) {
	m.rows = append(m.rows, row)
	m.updateTableRows()
}

// AddRows adds multiple rows to the split view
func (m *SplitViewModel) AddRows(rows []RowData) {
	m.rows = append(m.rows, rows...)
	m.updateTableRows()
}

// SetRows replaces all rows in the split view
func (m *SplitViewModel) SetRows(rows []RowData) {
	m.rows = rows
	m.updateTableRows()
	// Clear caches since we have new rows
	m.codeFullscreenCache = make(map[int]string)
	m.codeDetailCache = make(map[int]string)
	// Force viewport refresh on next render
	m.lastSelectedRow = -1
}

// GetRows returns the current rows
func (m *SplitViewModel) GetRows() []RowData {
	return m.rows
}

// GetCursor returns the current table cursor position
func (m *SplitViewModel) GetCursor() int {
	return m.table.Cursor()
}

// UpdateRow updates a specific row at the given index
func (m *SplitViewModel) UpdateRow(index int, row RowData) {
	if index < 0 || index >= len(m.rows) {
		return
	}
	m.rows[index] = row
	m.updateTableRows()
	// Clear cache for this row
	delete(m.codeFullscreenCache, index)
	delete(m.codeDetailCache, index)
	// Force viewport refresh if this is the selected row
	if index == m.table.Cursor() {
		m.lastSelectedRow = -1
	}
}

// updateTableRows updates the table with current rows
func (m *SplitViewModel) updateTableRows() {
	tableRows := make([]table.Row, len(m.rows))
	for i, row := range m.rows {
		tableRows[i] = row.TableRow
	}
	m.table.SetRows(tableRows)
}

// KeyMap returns the combined key bindings for the current mode
// This includes both the split view's keys and the active component's keys
func (m *SplitViewModel) KeyMap() help.KeyMap {
	if m.fullDetailMode {
		// In fullscreen mode, combine split view keys with viewport keys
		return CombinedKeyMap{
			SplitView: m.Keys,
			Viewport:  m.detailViewport.KeyMap,
		}
	}
	// In split mode, combine split view keys with table keys
	return CombinedKeyMap{
		SplitView: m.Keys,
		Table:     m.table.KeyMap,
	}
}

// CombinedKeyMap implements help.KeyMap by combining split view and component keys
type CombinedKeyMap struct {
	SplitView KeyMap
	Viewport  viewport.KeyMap
	Table     table.KeyMap
}

func (k CombinedKeyMap) ShortHelp() []key.Binding {
	// Return split view keys for short help
	return k.SplitView.ShortHelp()
}

func (k CombinedKeyMap) FullHelp() [][]key.Binding {
	// Combine split view keys with component-specific keys
	result := k.SplitView.FullHelp()

	// Add component-specific navigation keys
	if k.Viewport.Down.Enabled() {
		// Viewport mode
		result = append(result, []key.Binding{
			k.Viewport.Up,
			k.Viewport.Down,
			k.Viewport.PageUp,
			k.Viewport.PageDown,
			k.Viewport.HalfPageUp,
			k.Viewport.HalfPageDown,
		})
	} else if k.Table.LineUp.Enabled() {
		// Table mode
		result = append(result, []key.Binding{
			k.Table.LineUp,
			k.Table.LineDown,
			k.Table.PageUp,
			k.Table.PageDown,
			k.Table.GotoTop,
			k.Table.GotoBottom,
		})
	}

	return result
}

// View implements tea.Model
func (m *SplitViewModel) View() string {
	if m.width == 0 {
		return m.spinner.View() + " Loading..."
	}

	// Help is now handled by the parent - don't render it here
	// The parent can use KeyMap() to get all keys

	// Fullscreen mode - show only detail viewport
	if m.fullDetailMode {
		selectedIdx := m.table.Cursor()

		// Only update content if something changed
		if selectedIdx != m.lastSelectedRow || m.width != m.lastWidth || m.fullDetailMode != m.lastMode {
			// Account for horizontal padding (2 on each side = 4 total)
			contentWidth := m.width - 4
			m.detailViewport.Width = contentWidth
			m.detailViewport.Height = m.height - 4
			content := m.buildViewportContent(contentWidth, true)
			wrappedContent := lipgloss.NewStyle().Width(contentWidth).Render(content)
			m.detailViewport.SetContent(wrappedContent)

			// Update tracking
			m.lastSelectedRow = selectedIdx
			m.lastWidth = m.width
			m.lastMode = m.fullDetailMode
		}

		// Add horizontal padding to match table view
		return lipgloss.NewStyle().Padding(0, 1).Render(m.detailViewport.View())
	}

	// Split view mode - table on left, detail on right
	tableWidth := int(float64(m.width) * m.tableSplitPercent)
	detailWidth := m.width - tableWidth
	selectedIdx := m.table.Cursor()

	// Update table dimensions
	m.table.SetWidth(tableWidth)
	m.table.SetHeight(m.height - 4)

	// Check if we have no rows - show spinner in detail view
	if len(m.rows) == 0 {
		// Render table (empty)
		tableView := lipgloss.NewStyle().
			Width(tableWidth).
			MaxHeight(m.height).
			Render(m.table.View())

		// Show spinner in detail view
		detailView := lipgloss.NewStyle().
			Width(detailWidth).
			Height(m.height).
			Align(lipgloss.Center, lipgloss.Center).
			Render(m.spinner.View() + " Loading...")

		return lipgloss.JoinHorizontal(lipgloss.Top, tableView, detailView)
	}

	// Only update content if something changed
	if selectedIdx != m.lastSelectedRow || detailWidth != m.lastWidth || m.fullDetailMode != m.lastMode {
		m.detailViewport.Width = detailWidth
		m.detailViewport.Height = m.height - 4
		content := m.buildViewportContent(detailWidth, false)
		wrappedContent := lipgloss.NewStyle().Width(detailWidth).Render(content)
		m.detailViewport.SetContent(wrappedContent)

		// Update tracking
		m.lastSelectedRow = selectedIdx
		m.lastWidth = detailWidth
		m.lastMode = m.fullDetailMode
	}

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

	return mainView
}
