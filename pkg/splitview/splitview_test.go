package splitview

import (
	"reflect"
	"testing"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
)

var testColumns = []ColumnConfig{
	{Name: "Name", Width: 20},
	{Name: "Type", Width: 15},
	{Name: "Network", Width: 15},
}

var testRows = []RowData{
	NewRowData(table.Row{"tx1", "Transaction", "testnet"}).WithCode("code1").WithContent("Content 1"),
	NewRowData(table.Row{"tx2", "Transaction", "mainnet"}).WithCode("code2").WithContent("Content 2"),
	NewRowData(table.Row{"tx3", "Script", "testnet"}).WithContent("Content 3"),
}

func TestNewSplitView(t *testing.T) {
	tests := map[string]struct {
		columns []ColumnConfig
		opts    []Option
		wantErr bool
	}{
		"Default": {
			columns: testColumns,
		},
		"WithRows": {
			columns: testColumns,
			opts: []Option{
				WithRows(testRows),
			},
		},
		"WithTableSplitPercent": {
			columns: testColumns,
			opts: []Option{
				WithTableSplitPercent(0.4),
			},
		},
		"Multiple options": {
			columns: testColumns,
			opts: []Option{
				WithRows(testRows),
				WithTableSplitPercent(0.5),
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			m := NewSplitView(tc.columns, tc.opts...)

			if m == nil {
				t.Fatal("expected non-nil model")
			}

			if len(m.columns) != len(tc.columns) {
				t.Errorf("expected %d columns, got %d", len(tc.columns), len(m.columns))
			}

			// Check if spinner was initialized
			if m.spinner.Spinner.FPS == 0 {
				t.Error("spinner not initialized")
			}

			// Check if keys were initialized
			if len(m.Keys.Quit.Keys()) == 0 {
				t.Error("keys not initialized")
			}
		})
	}
}

func TestRowDataBuilder(t *testing.T) {
	tests := map[string]struct {
		row     table.Row
		code    string
		content string
	}{
		"With all fields": {
			row:     table.Row{"name", "type", "network"},
			code:    "some code",
			content: "some content",
		},
		"Without code": {
			row:     table.Row{"name", "type", "network"},
			content: "some content",
		},
		"Without content": {
			row:  table.Row{"name", "type", "network"},
			code: "some code",
		},
		"Only row": {
			row: table.Row{"name", "type", "network"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			row := NewRowData(tc.row)

			if tc.code != "" {
				row = row.WithCode(tc.code)
			}
			if tc.content != "" {
				row = row.WithContent(tc.content)
			}

			if !reflect.DeepEqual(row.TableRow, tc.row) {
				t.Errorf("expected row %v, got %v", tc.row, row.TableRow)
			}

			if row.Code != tc.code {
				t.Errorf("expected code %q, got %q", tc.code, row.Code)
			}

			if row.Content != tc.content {
				t.Errorf("expected content %q, got %q", tc.content, row.Content)
			}
		})
	}
}

func TestAddRow(t *testing.T) {
	m := NewSplitView(testColumns)

	if len(m.GetRows()) != 0 {
		t.Errorf("expected 0 rows initially, got %d", len(m.GetRows()))
	}

	newRow := NewRowData(table.Row{"new", "row", "test"})
	m.AddRow(newRow)

	if len(m.GetRows()) != 1 {
		t.Errorf("expected 1 row after AddRow, got %d", len(m.GetRows()))
	}

	if !reflect.DeepEqual(m.GetRows()[0], newRow) {
		t.Errorf("added row doesn't match")
	}
}

func TestAddRows(t *testing.T) {
	m := NewSplitView(testColumns)

	m.AddRows(testRows)

	if len(m.GetRows()) != len(testRows) {
		t.Errorf("expected %d rows, got %d", len(testRows), len(m.GetRows()))
	}

	for i, row := range m.GetRows() {
		if !reflect.DeepEqual(row, testRows[i]) {
			t.Errorf("row %d doesn't match", i)
		}
	}
}

func TestSetRows(t *testing.T) {
	initialRows := []RowData{
		NewRowData(table.Row{"old", "row", "1"}),
	}

	m := NewSplitView(testColumns, WithRows(initialRows))

	if len(m.GetRows()) != 1 {
		t.Errorf("expected 1 initial row, got %d", len(m.GetRows()))
	}

	m.SetRows(testRows)

	if len(m.GetRows()) != len(testRows) {
		t.Errorf("expected %d rows after SetRows, got %d", len(testRows), len(m.GetRows()))
	}

	for i, row := range m.GetRows() {
		if !reflect.DeepEqual(row, testRows[i]) {
			t.Errorf("row %d doesn't match after SetRows", i)
		}
	}
}

func TestGetRows(t *testing.T) {
	m := NewSplitView(testColumns, WithRows(testRows))

	rows := m.GetRows()

	if len(rows) != len(testRows) {
		t.Errorf("expected %d rows, got %d", len(testRows), len(rows))
	}

	for i, row := range rows {
		if !reflect.DeepEqual(row, testRows[i]) {
			t.Errorf("row %d doesn't match", i)
		}
	}
}

func TestInit(t *testing.T) {
	m := NewSplitView(testColumns)
	cmd := m.Init()

	if cmd == nil {
		t.Error("expected Init to return spinner tick command")
	}
}

func TestUpdate_WindowSizeMsg(t *testing.T) {
	m := NewSplitView(testColumns, WithRows(testRows))

	if m.width != 0 || m.height != 0 {
		t.Error("expected initial width and height to be 0")
	}

	newWidth, newHeight := 100, 50
	updatedModel, _ := m.Update(tea.WindowSizeMsg{
		Width:  newWidth,
		Height: newHeight,
	})

	updatedSplit := updatedModel.(*SplitViewModel)

	if updatedSplit.width != newWidth {
		t.Errorf("expected width %d, got %d", newWidth, updatedSplit.width)
	}

	if updatedSplit.height != newHeight {
		t.Errorf("expected height %d, got %d", newHeight, updatedSplit.height)
	}

	// Check that caches were cleared
	if len(updatedSplit.codeFullscreenCache) != 0 {
		t.Error("expected fullscreen cache to be cleared")
	}
	if len(updatedSplit.codeDetailCache) != 0 {
		t.Error("expected detail cache to be cleared")
	}
}

func TestUpdate_ToggleFullscreen(t *testing.T) {
	m := NewSplitView(testColumns, WithRows(testRows))

	if m.fullDetailMode {
		t.Error("expected fullDetailMode to be false initially")
	}

	// Toggle to fullscreen
	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	updatedSplit := updatedModel.(*SplitViewModel)

	if !updatedSplit.fullDetailMode {
		t.Error("expected fullDetailMode to be true after space key")
	}

	// Toggle back
	updatedModel, _ = updatedSplit.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updatedSplit = updatedModel.(*SplitViewModel)

	if updatedSplit.fullDetailMode {
		t.Error("expected fullDetailMode to be false after enter key")
	}
}

func TestUpdate_ExitFullscreen(t *testing.T) {
	m := NewSplitView(testColumns, WithRows(testRows))
	m.fullDetailMode = true

	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updatedSplit := updatedModel.(*SplitViewModel)

	if updatedSplit.fullDetailMode {
		t.Error("expected fullDetailMode to be false after esc key")
	}
}

func TestView_Empty(t *testing.T) {
	m := NewSplitView(testColumns)

	// Without WindowSizeMsg, should show loading
	view := m.View()
	if view == "" {
		t.Error("expected non-empty view")
	}
	if view != m.spinner.View()+" Loading..." {
		t.Errorf("expected loading message, got %q", view)
	}
}

func TestView_WithSize(t *testing.T) {
	m := NewSplitView(testColumns)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})

	view := m.View()
	if view == "" {
		t.Error("expected non-empty view after window size set")
	}
}

func TestKeyMap_ShortHelp(t *testing.T) {
	km := NewKeyMap()
	shortHelp := km.ShortHelp()

	if len(shortHelp) != 3 {
		t.Errorf("expected 3 short help bindings, got %d", len(shortHelp))
	}
}

func TestKeyMap_FullHelp(t *testing.T) {
	km := NewKeyMap()
	fullHelp := km.FullHelp()

	if len(fullHelp) != 2 {
		t.Errorf("expected 2 groups in full help, got %d", len(fullHelp))
	}
}

func TestCombinedKeyMap(t *testing.T) {
	m := NewSplitView(testColumns, WithRows(testRows))
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})

	// Test split view mode
	keyMap := m.KeyMap()
	if keyMap == nil {
		t.Error("expected non-nil keymap")
	}

	// Test fullscreen mode
	m.fullDetailMode = true
	keyMap = m.KeyMap()
	if keyMap == nil {
		t.Error("expected non-nil keymap in fullscreen mode")
	}
}

func TestWithTableSplitPercent(t *testing.T) {
	tests := []struct {
		name    string
		percent float64
	}{
		{"default", defaultTableSplitPercent},
		{"custom 40%", 0.4},
		{"custom 50%", 0.5},
		{"custom 25%", 0.25},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			opts := []Option{}
			if tc.percent != defaultTableSplitPercent {
				opts = append(opts, WithTableSplitPercent(tc.percent))
			}

			m := NewSplitView(testColumns, opts...)

			if m.tableSplitPercent != tc.percent {
				t.Errorf("expected tableSplitPercent %f, got %f", tc.percent, m.tableSplitPercent)
			}
		})
	}
}
