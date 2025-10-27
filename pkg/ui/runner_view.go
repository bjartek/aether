package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bjartek/aether/pkg/aether"
	"github.com/bjartek/aether/pkg/chroma"
	"github.com/bjartek/aether/pkg/config"
	"github.com/bjartek/aether/pkg/flow"
	"github.com/bjartek/aether/pkg/splitview"
	"github.com/bjartek/aether/pkg/tabbedtui"
	"github.com/bjartek/overflow/v2"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/parser"
	"github.com/onflow/cadence/sema"
	"github.com/rs/zerolog"
)

// ScriptType represents the type of script
type ScriptType string

const (
	TypeScript      ScriptType = "script"
	TypeTransaction ScriptType = "transaction"
)

// ExecutionCompleteMsg is sent when script/transaction execution completes
type ExecutionCompleteMsg struct {
	ScriptResult *overflow.OverflowScriptResult
	TxResult     *overflow.OverflowResult
	IsScript     bool
	Error        error
}

// RescanFilesMsg triggers a rescan of script/transaction files
type RescanFilesMsg struct{}

// Parameter represents a parameter in a script or transaction
type Parameter struct {
	Name string
	Type string
}

// ScriptFile represents a Cadence script or transaction file
type ScriptFile struct {
	Name            string
	Path            string
	Type            ScriptType
	Parameters      []Parameter
	Signers         int                     // Number of signers needed for transactions
	SignerParams    []Parameter             // Signer parameter names from prepare block
	Code            string                  // Raw code
	HighlightedCode string                  // Syntax-highlighted code with ANSI colors
	Config          *flow.TransactionConfig // Pre-populated config from JSON (if loaded from .json file)
	IsFromJSON      bool                    // True if this was loaded from a JSON config file
	Network         string                  // Network this script is specific to (emulator, testnet, mainnet, or "any")
}

// InputField represents a form input field
type InputField struct {
	Label    string
	TypeHint string
	Input    textinput.Model
	IsSigner bool // True if this is a signer selection field
}

// RunnerKeyMap defines keybindings for the runner view
type RunnerKeyMap struct {
	Up        key.Binding
	Down      key.Binding
	Enter     key.Binding
	Run       key.Binding
	NextField key.Binding
	PrevField key.Binding
	Save      key.Binding
	Refresh   key.Binding
}

// DefaultRunnerKeyMap returns the default keybindings for runner view
func DefaultRunnerKeyMap() RunnerKeyMap {
	return RunnerKeyMap{
		Up: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("k/↑", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("j/↓", "down"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "toggle detail"),
		),
		Run: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "run"),
		),
		NextField: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next field"),
		),
		PrevField: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "prev field"),
		),
		Save: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "save config"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("ctrl+l", "x"),
			key.WithHelp("ctrl+l/x", "refresh list"),
		),
	}
}

// ShortHelp returns keybindings to be shown in the mini help view
func (k RunnerKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Run, k.Refresh}
}

// FullHelp returns keybindings for the expanded help view
func (k RunnerKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Enter},
		{k.Run, k.Save, k.Refresh},
	}
}

// RunnerView is the splitview-based implementation for script/transaction runner
type RunnerView struct {
	sv               *splitview.SplitViewModel
	keys             RunnerKeyMap
	width            int
	height           int
	scripts          []ScriptFile
	overflow         *overflow.OverflowState
	accountRegistry  *aether.AccountRegistry
	executionResult  string
	executionError   error
	inputFields      []InputField
	activeFieldIndex int
	editingField     bool
	availableSigners []string
	savingConfig     bool
	saveInput        textinput.Model
	saveError        string
	executing        bool // True when script is executing
	lastSelectedIdx  int  // Track last selected index to detect navigation
	logger           zerolog.Logger
}

// NewRunnerViewWithConfig creates a new v2 runner view
func NewRunnerViewWithConfig(cfg *config.Config, logger zerolog.Logger) *RunnerView {
	// Fallback to defaults when cfg is nil
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	columns := []splitview.ColumnConfig{
		{Name: "Type", Width: 8},
		{Name: "Name", Width: 40},
		{Name: "Network", Width: 12},
	}

	// Table styles
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
	tableSplitPercent := float64(cfg.UI.Layout.RunnerSplitPercent) / 100.0
	sv := splitview.NewSplitView(
		columns,
		splitview.WithTableStyles(s),
		splitview.WithTableSplitPercent(tableSplitPercent),
	)

	// Create save input
	saveInput := textinput.New()
	saveInput.Placeholder = "config-name"
	saveInput.CharLimit = 50
	saveInput.Width = 40

	rv := &RunnerView{
		sv:               sv,
		keys:             DefaultRunnerKeyMap(),
		scripts:          make([]ScriptFile, 0),
		inputFields:      make([]InputField, 0),
		activeFieldIndex: 0,
		editingField:     false,
		saveInput:        saveInput,
		lastSelectedIdx:  -1,
		logger:           logger,
	}

	return rv
}

// Init implements tea.Model
func (rv *RunnerView) Init() tea.Cmd {
	// Scan for script and transaction files
	rv.scanFiles()

	// Populate splitview with discovered scripts
	for _, script := range rv.scripts {
		rv.addScriptRow(script)
	}

	rv.logger.Debug().
		Int("scriptsFound", len(rv.scripts)).
		Msg("RunnerView initialized with scripts")

	return rv.sv.Init()
}

// Update implements tea.Model
func (rv *RunnerView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	rv.logger.Debug().Str("method", "Update").Interface("msgType", msg).Msg("RunnerView.Update called")

	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		rv.width = msg.Width
		rv.height = msg.Height
		rv.logger.Debug().
			Int("width", msg.Width).
			Int("height", msg.Height).
			Msg("WindowSizeMsg received")

	case aether.OverflowReadyMsg:
		// Set overflow and account registry when ready
		rv.SetOverflow(msg.Overflow)
		rv.SetAccountRegistry(msg.AccountRegistry)
		return rv, nil

	case RescanFilesMsg:
		// Rescan files and rebuild rows
		rv.logger.Info().Msg("RescanFilesMsg received - rescanning files")
		rv.scanFiles()

		// Rebuild splitview rows
		rows := make([]splitview.RowData, 0)
		for _, script := range rv.scripts {
			typeStr := "Script"
			if script.Type == TypeTransaction {
				typeStr = "Tx"
			}
			if script.IsFromJSON && script.Config != nil {
				typeStr += "*"
			}
			row := table.Row{typeStr, script.Name, script.Network}
			content := rv.buildScriptDetail(script)
			codeToShow := script.HighlightedCode
			rows = append(rows, splitview.NewRowData(row).WithContent(content).WithCode(codeToShow))
		}
		rv.sv.SetRows(rows)

		rv.logger.Info().
			Int("scriptsFound", len(rv.scripts)).
			Msg("Rescan complete")

		return rv, nil

	case tea.KeyMsg:
		rv.logger.Debug().
			Str("key", msg.String()).
			Bool("editingField", rv.editingField).
			Bool("savingConfig", rv.savingConfig).
			Bool("isFullscreen", rv.sv.IsFullscreen()).
			Int("inputFieldsCount", len(rv.inputFields)).
			Msg("KeyPress received")

		// Handle save dialog input
		if rv.savingConfig {
			switch {
			case key.Matches(msg, rv.keys.Enter):
				// Save config
				if rv.saveInput.Value() != "" {
					configName := rv.saveInput.Value()
					rv.saveError = rv.saveConfig(configName)
					if rv.saveError == "" {
						rv.savingConfig = false
						rv.saveInput.SetValue("")

						rv.logger.Info().
							Str("configName", configName).
							Msg("Config saved successfully - reloading files")

						// Show success message
						rv.executionResult = fmt.Sprintf("Config saved as '%s.json'", configName)
						rv.executionError = nil

						// Reload files from filesystem to pick up new config
						oldCount := len(rv.scripts)
						rv.scanFiles()

						rv.logger.Info().
							Int("oldCount", oldCount).
							Int("newCount", len(rv.scripts)).
							Msg("Files reloaded after save")

						// Rebuild splitview rows
						rows := make([]splitview.RowData, 0)
						for i, script := range rv.scripts {
							typeStr := "Script"
							if script.Type == TypeTransaction {
								typeStr = "Tx"
							}
							if script.IsFromJSON && script.Config != nil {
								typeStr += "*"
							}
							row := table.Row{typeStr, script.Name, script.Network}
							content := rv.buildScriptDetail(script)
							codeToShow := script.HighlightedCode
							rows = append(rows, splitview.NewRowData(row).WithContent(content).WithCode(codeToShow))

							rv.logger.Debug().
								Int("index", i).
								Str("type", typeStr).
								Str("name", script.Name).
								Str("network", script.Network).
								Bool("hasConfig", script.Config != nil).
								Bool("isFromJSON", script.IsFromJSON).
								Msg("Built row")
						}

						rv.logger.Info().
							Int("rowsBuilt", len(rows)).
							Msg("Setting rows in splitview")

						rv.sv.SetRows(rows)

						rv.logger.Info().Msg("Rows set successfully")

						// Refresh detail to show success message
						selectedIdx := rv.sv.GetCursor()
						if selectedIdx >= 0 && selectedIdx < len(rv.scripts) {
							rv.logger.Info().
								Int("selectedIdx", selectedIdx).
								Str("scriptName", rv.scripts[selectedIdx].Name).
								Msg("Refreshing detail content")
							rv.refreshDetailContent(selectedIdx, rv.scripts[selectedIdx])
						}
					} else {
						rv.logger.Error().
							Str("error", rv.saveError).
							Msg("Failed to save config")
					}
				}
				// Return with a command to force re-render
				return rv, tea.Batch()

			case msg.Type == tea.KeyEsc:
				// Cancel save
				rv.savingConfig = false
				rv.saveInput.SetValue("")
				rv.saveError = ""
				return rv, nil

			default:
				// Update save input
				var cmd tea.Cmd
				rv.saveInput, cmd = rv.saveInput.Update(msg)

				// Refresh detail content to show the updated input
				selectedIdx := rv.sv.GetCursor()
				if selectedIdx >= 0 && selectedIdx < len(rv.scripts) {
					rv.refreshDetailContent(selectedIdx, rv.scripts[selectedIdx])
				}

				return rv, cmd
			}
		}

		// Handle form editing (when actively typing in a field)
		if rv.editingField && len(rv.inputFields) > 0 {
			switch {
			case msg.Type == tea.KeyEsc:
				// Exit editing mode, blur field, stay in field navigation mode
				rv.editingField = false
				rv.inputFields[rv.activeFieldIndex].Input.Blur()
				// Refresh detail to update UI
				selectedIdx := rv.sv.GetCursor()
				if selectedIdx >= 0 && selectedIdx < len(rv.scripts) {
					rv.refreshDetailContent(selectedIdx, rv.scripts[selectedIdx])
				}
				return rv, tabbedtui.InputHandled()

			case key.Matches(msg, rv.keys.Enter):
				// Finish editing current field and move to next field
				rv.inputFields[rv.activeFieldIndex].Input.Blur()
				if rv.activeFieldIndex < len(rv.inputFields)-1 {
					rv.activeFieldIndex++
					rv.inputFields[rv.activeFieldIndex].Input.Focus()
					rv.editingField = true
				} else {
					// Last field - exit editing mode
					rv.editingField = false
				}
				selectedIdx := rv.sv.GetCursor()
				if selectedIdx >= 0 && selectedIdx < len(rv.scripts) {
					rv.refreshDetailContent(selectedIdx, rv.scripts[selectedIdx])
				}
				return rv, tabbedtui.InputHandled()

			case msg.Type == tea.KeyTab:
				// Tab: finish editing and move to next field
				rv.inputFields[rv.activeFieldIndex].Input.Blur()
				rv.activeFieldIndex = (rv.activeFieldIndex + 1) % len(rv.inputFields)
				rv.inputFields[rv.activeFieldIndex].Input.Focus()
				selectedIdx := rv.sv.GetCursor()
				if selectedIdx >= 0 && selectedIdx < len(rv.scripts) {
					rv.refreshDetailContent(selectedIdx, rv.scripts[selectedIdx])
				}
				return rv, tabbedtui.InputHandled()

			case msg.Type == tea.KeyShiftTab:
				// Shift-Tab: finish editing and move to previous field
				rv.inputFields[rv.activeFieldIndex].Input.Blur()
				rv.activeFieldIndex = (rv.activeFieldIndex - 1 + len(rv.inputFields)) % len(rv.inputFields)
				rv.inputFields[rv.activeFieldIndex].Input.Focus()
				selectedIdx := rv.sv.GetCursor()
				if selectedIdx >= 0 && selectedIdx < len(rv.scripts) {
					rv.refreshDetailContent(selectedIdx, rv.scripts[selectedIdx])
				}
				return rv, tabbedtui.InputHandled()

			default:
				// Update active input field
				var cmd tea.Cmd
				rv.inputFields[rv.activeFieldIndex].Input, cmd = rv.inputFields[rv.activeFieldIndex].Input.Update(msg)

				// Refresh detail content to show the updated input
				selectedIdx := rv.sv.GetCursor()
				if selectedIdx >= 0 && selectedIdx < len(rv.scripts) {
					rv.refreshDetailContent(selectedIdx, rv.scripts[selectedIdx])
				}

				return rv, cmd
			}
		}

		// Handle field navigation (when in fullscreen with fields but not editing)
		if !rv.editingField && len(rv.inputFields) > 0 && rv.sv.IsFullscreen() {
			switch {
			case msg.Type == tea.KeyTab:
				// Tab: move to next field
				rv.activeFieldIndex = (rv.activeFieldIndex + 1) % len(rv.inputFields)
				selectedIdx := rv.sv.GetCursor()
				if selectedIdx >= 0 && selectedIdx < len(rv.scripts) {
					rv.refreshDetailContent(selectedIdx, rv.scripts[selectedIdx])
				}
				return rv, tabbedtui.InputHandled()

			case msg.Type == tea.KeyShiftTab:
				// Shift-Tab: move to previous field
				rv.activeFieldIndex = (rv.activeFieldIndex - 1 + len(rv.inputFields)) % len(rv.inputFields)
				selectedIdx := rv.sv.GetCursor()
				if selectedIdx >= 0 && selectedIdx < len(rv.scripts) {
					rv.refreshDetailContent(selectedIdx, rv.scripts[selectedIdx])
				}
				return rv, tabbedtui.InputHandled()

			case key.Matches(msg, rv.keys.Enter):
				// Enter: start editing the current field
				rv.editingField = true
				rv.inputFields[rv.activeFieldIndex].Input.Focus()
				selectedIdx := rv.sv.GetCursor()
				if selectedIdx >= 0 && selectedIdx < len(rv.scripts) {
					rv.refreshDetailContent(selectedIdx, rv.scripts[selectedIdx])
				}
				return rv, tabbedtui.InputHandled()

			case msg.Type == tea.KeyEsc:
				// Esc: clear input fields and allow viewport scrolling
				rv.inputFields = make([]InputField, 0)
				rv.editingField = false
				rv.activeFieldIndex = 0
				selectedIdx := rv.sv.GetCursor()
				if selectedIdx >= 0 && selectedIdx < len(rv.scripts) {
					rv.refreshDetailContent(selectedIdx, rv.scripts[selectedIdx])
				}
				// Don't return InputHandled - let splitview handle scrolling
				_, cmd := rv.sv.Update(msg)
				return rv, cmd
			}
		}

		// Handle refresh
		if key.Matches(msg, rv.keys.Refresh) {
			rv.logger.Info().Msg("Refresh triggered - rescanning files")
			rv.scanFiles()

			// Rebuild splitview rows
			rows := make([]splitview.RowData, 0)
			for _, script := range rv.scripts {
				// Type: "Script" or "Tx", with "*" if prefilled from config
				typeStr := "Script"
				if script.Type == TypeTransaction {
					typeStr = "Tx"
				}
				if script.IsFromJSON && script.Config != nil {
					typeStr += "*"
				}

				// Use script.Name which contains relative path without .cdc extension
				row := table.Row{typeStr, script.Name, script.Network}
				content := rv.buildScriptDetail(script)
				codeToShow := script.HighlightedCode
				rows = append(rows, splitview.NewRowData(row).WithContent(content).WithCode(codeToShow))
			}
			rv.sv.SetRows(rows)

			rv.logger.Info().
				Int("scriptsFound", len(rv.scripts)).
				Msg("Refresh complete")

			return rv, tabbedtui.InputHandled()
		}

		// Handle save config (only when not editing)
		if key.Matches(msg, rv.keys.Save) && rv.sv.IsFullscreen() && !rv.editingField {
			rv.savingConfig = true
			rv.saveInput.Focus()
			rv.saveError = ""

			// Refresh detail content to show save dialog
			selectedIdx := rv.sv.GetCursor()
			if selectedIdx >= 0 && selectedIdx < len(rv.scripts) {
				rv.refreshDetailContent(selectedIdx, rv.scripts[selectedIdx])
			}
			return rv, nil
		}

		// Handle enter/space to toggle fullscreen and build forms
		if key.Matches(msg, rv.keys.Enter) {
			wasFullscreen := rv.sv.IsFullscreen()

			// Build input fields BEFORE toggling fullscreen
			if !wasFullscreen {
				selectedIdx := rv.sv.GetCursor()
				if selectedIdx >= 0 && selectedIdx < len(rv.scripts) {
					script := rv.scripts[selectedIdx]
					rv.buildInputFields(script)
					rv.tryLoadConfigFromJSON(script)

					// Clear any previous execution results when entering fullscreen
					rv.executionResult = ""
					rv.executionError = nil
					rv.executing = false

					// Start in navigation mode (not editing) so user can press 'r' to run
					// or Enter to start editing the first field
					rv.activeFieldIndex = 0
					rv.editingField = false
					if len(rv.inputFields) > 0 {
						// Focus but don't start editing - user presses Enter to edit
						rv.inputFields[0].Input.Focus()
					}
				}
			}

			// Pass to splitview to toggle fullscreen - it will re-render with fields
			_, cmd := rv.sv.Update(msg)
			cmds = append(cmds, cmd)

			// After entering fullscreen, refresh detail content to show input fields
			if !wasFullscreen && rv.sv.IsFullscreen() {
				selectedIdx := rv.sv.GetCursor()
				if selectedIdx >= 0 && selectedIdx < len(rv.scripts) {
					rv.refreshDetailContent(selectedIdx, rv.scripts[selectedIdx])
				}
			}

			// Clear input fields and execution results when exiting fullscreen
			if wasFullscreen && !rv.sv.IsFullscreen() {
				rv.inputFields = make([]InputField, 0)
				rv.editingField = false
				rv.executionResult = ""
				rv.executionError = nil
				rv.executing = false

				// Refresh detail content to show cleared state
				selectedIdx := rv.sv.GetCursor()
				if selectedIdx >= 0 && selectedIdx < len(rv.scripts) {
					rv.refreshDetailContent(selectedIdx, rv.scripts[selectedIdx])
				}
			}

			return rv, tea.Batch(cmds...)
		}

		// Handle 'r' key to run selected script/transaction (only when not actively editing)
		if key.Matches(msg, rv.keys.Run) && !rv.editingField {
			rv.logger.Debug().
				Bool("hasOverflow", rv.overflow != nil).
				Int("selectedIdx", rv.sv.GetCursor()).
				Int("scriptsCount", len(rv.scripts)).
				Bool("isFullscreen", rv.sv.IsFullscreen()).
				Int("inputFieldsCount", len(rv.inputFields)).
				Bool("editingField", rv.editingField).
				Msg("Run key matched!")

			if rv.overflow != nil {
				selectedIdx := rv.sv.GetCursor()
				if selectedIdx >= 0 && selectedIdx < len(rv.scripts) {
					script := rv.scripts[selectedIdx]
					rv.logger.Info().
						Str("scriptName", script.Name).
						Str("scriptType", string(script.Type)).
						Msg("Executing script/transaction")

					// Build input fields if not already built
					if len(rv.inputFields) == 0 {
						rv.buildInputFields(script)
						rv.tryLoadConfigFromJSON(script)
					}

					rv.executionResult = ""
					rv.executionError = nil
					rv.executing = true
					// Refresh to show spinner
					rv.refreshDetailContent(selectedIdx, script)
					return rv, rv.executeScript(script)
				} else {
					rv.logger.Warn().Msg("No script selected or invalid index")
				}
			} else {
				rv.logger.Warn().Msg("Overflow not initialized")
			}
		}

	case ExecutionCompleteMsg:
		rv.logger.Info().
			Bool("hasError", msg.Error != nil).
			Bool("isScript", msg.IsScript).
			Msg("ExecutionCompleteMsg received")

		// Clear executing flag
		rv.executing = false

		if msg.Error != nil {
			rv.executionError = msg.Error
			rv.executionResult = ""
		} else {
			rv.executionError = nil
			// Format result based on type with rich formatting
			if msg.IsScript && msg.ScriptResult != nil {
				if msg.ScriptResult.Err != nil {
					rv.executionError = msg.ScriptResult.Err
					rv.executionResult = ""
				} else {
					rv.executionResult = rv.formatScriptResult(msg.ScriptResult)
				}
			} else if msg.TxResult != nil {
				if msg.TxResult.Err != nil {
					rv.executionError = msg.TxResult.Err
					rv.executionResult = ""
				} else {
					rv.executionResult = rv.formatTransactionResult(msg.TxResult)
				}
			} else {
				rv.executionResult = "✓ Execution successful"
			}
		}

		// Refresh the current row to show results inline
		selectedIdx := rv.sv.GetCursor()
		if selectedIdx >= 0 && selectedIdx < len(rv.scripts) {
			script := rv.scripts[selectedIdx]
			rv.refreshDetailContent(selectedIdx, script)
		}
		return rv, nil
	}

	// If we get here, the message is being passed to splitview
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		rv.logger.Debug().
			Str("key", keyMsg.String()).
			Msg("Passing key message to splitview")
	}

	_, cmd := rv.sv.Update(msg)
	cmds = append(cmds, cmd)

	// Check if cursor position changed after splitview update - clear execution results when navigating
	currentIdx := rv.sv.GetCursor()
	if currentIdx != rv.lastSelectedIdx && rv.lastSelectedIdx != -1 {
		rv.logger.Debug().
			Int("oldIdx", rv.lastSelectedIdx).
			Int("newIdx", currentIdx).
			Msg("Cursor changed - clearing execution results")
		rv.executionResult = ""
		rv.executionError = nil
		rv.executing = false

		// Refresh detail content to show cleared state
		if currentIdx >= 0 && currentIdx < len(rv.scripts) {
			rv.refreshDetailContent(currentIdx, rv.scripts[currentIdx])
		}
	}
	rv.lastSelectedIdx = currentIdx

	return rv, tea.Batch(cmds...)
}

// View implements tea.Model
func (rv *RunnerView) View() string {
	rv.logger.Debug().Str("method", "View").Msg("RunnerView.View called")

	view := rv.sv.View()

	rv.logger.Debug().
		Str("method", "View").
		Str("component", "runner").
		Int("parentWidth", rv.width).
		Int("parentHeight", rv.height).
		Int("svWidth", rv.sv.GetWidth()).
		Int("svHeight", rv.sv.GetHeight()).
		Int("svTableWidth", rv.sv.GetTableWidth()).
		Int("svDetailWidth", rv.sv.GetDetailWidth()).
		Int("viewLength", len(view)).
		Bool("isFullscreen", rv.sv.IsFullscreen()).
		Int("inputFieldsCount", len(rv.inputFields)).
		Int("scriptsCount", len(rv.scripts)).
		Msg("RunnerView.View rendered")
	return view
}

// Name implements TabbedModel interface
func (rv *RunnerView) Name() string {
	return "Runner"
}

// KeyMap implements TabbedModel interface - combines runner and splitview keys
func (rv *RunnerView) KeyMap() help.KeyMap {
	return tabbedtui.NewCombinedKeyMap(rv.keys, rv.sv.KeyMap())
}

// FooterView implements TabbedModel interface - results shown inline in detail
func (rv *RunnerView) FooterView() string {
	return ""
}

// IsCapturingInput implements TabbedModel interface
func (rv *RunnerView) IsCapturingInput() bool {
	return rv.editingField || rv.savingConfig
}

// SetOverflow sets the overflow state for script execution
func (rv *RunnerView) SetOverflow(o *overflow.OverflowState) {
	rv.overflow = o
}

// SetAccountRegistry sets the account registry for signer name resolution
func (rv *RunnerView) SetAccountRegistry(registry *aether.AccountRegistry) {
	rv.accountRegistry = registry
	// Update available signers list from registry
	if registry != nil {
		rv.availableSigners = registry.GetAllNames()
	}
}

// AddScript adds a script to the runner view (called externally like AddTransaction)
func (rv *RunnerView) AddScript(script ScriptFile) {
	// Store script data
	rv.scripts = append(rv.scripts, script)

	// Parse the script to extract parameters and signers
	rv.parseScriptFile(&script)

	// Add to splitview
	rv.addScriptRow(script)

	rv.logger.Debug().
		Str("scriptName", script.Name).
		Str("scriptType", string(script.Type)).
		Int("totalScripts", len(rv.scripts)).
		Msg("Script added to runner view")
}

// formatValue recursively formats a value with proper indentation
func (rv *RunnerView) formatValue(value interface{}, indent string) string {
	var b strings.Builder

	switch v := value.(type) {
	case map[string]interface{}:
		if len(v) == 0 {
			return "{}"
		}
		first := true
		for key, val := range v {
			if !first {
				b.WriteString("\n")
			}
			first = false
			b.WriteString(fmt.Sprintf("%s%s: ", indent, key))
			b.WriteString(rv.formatValue(val, indent+"  "))
		}
	case []interface{}:
		if len(v) == 0 {
			return "[]"
		}
		for i, val := range v {
			if i > 0 {
				b.WriteString("\n")
			}
			b.WriteString(fmt.Sprintf("%s[%d]: ", indent, i))
			b.WriteString(rv.formatValue(val, indent+"  "))
		}
	default:
		// Underflow handles timestamps and addresses automatically
		b.WriteString(fmt.Sprintf("%v", v))
	}

	return b.String()
}

// formatScriptResult formats the script result for display
func (rv *RunnerView) formatScriptResult(result *overflow.OverflowScriptResult) string {
	if result == nil {
		return "✓ Script executed successfully"
	}

	var b strings.Builder

	b.WriteString("✓ Script executed successfully\n\n")

	// Show the Output field which uses underflow options from overflow
	if result.Output != nil {
		b.WriteString("Output:\n")
		b.WriteString(rv.formatValue(result.Output, ""))
	}

	return b.String()
}

// formatTransactionResult formats the transaction result for display
func (rv *RunnerView) formatTransactionResult(result *overflow.OverflowResult) string {
	if result == nil {
		return "✓ Transaction executed successfully"
	}

	var b strings.Builder

	b.WriteString("✓ Transaction executed successfully\n\n")
	b.WriteString(fmt.Sprintf("Transaction ID: %s\n", result.Id))

	// Show events if any
	if len(result.Events) > 0 {
		b.WriteString(fmt.Sprintf("\nEvents (%d):\n", len(result.Events)))
		count := 0
		for eventName, eventList := range result.Events {
			if count >= 5 {
				b.WriteString(fmt.Sprintf("  ... and %d more event types\n", len(result.Events)-5))
				break
			}
			b.WriteString(fmt.Sprintf("  • %s (%d)\n", eventName, len(eventList)))
			count++
		}
	}

	return b.String()
}

// detectNetwork determines the network from a filename
func (rv *RunnerView) detectNetwork(filename string) string {
	// Check for network suffixes
	if strings.HasSuffix(filename, ".emulator") {
		return "emulator"
	} else if strings.HasSuffix(filename, ".testnet") {
		return "testnet"
	} else if strings.HasSuffix(filename, ".mainnet") {
		return "mainnet"
	}
	return "any"
}

// removeNetworkSuffix removes network suffix from filename for display
func (rv *RunnerView) removeNetworkSuffix(filename string) string {
	for _, suffix := range []string{".emulator", ".testnet", ".mainnet"} {
		if strings.HasSuffix(filename, suffix) {
			return strings.TrimSuffix(filename, suffix)
		}
	}
	return filename
}

// refreshDetailContent updates the detail content for the current row
func (rv *RunnerView) refreshDetailContent(idx int, script ScriptFile) {
	content := rv.buildScriptDetail(script)
	rows := rv.sv.GetRows()
	if idx < len(rows) {
		row := rows[idx]
		row.Content = content
		rows[idx] = row
		rv.sv.SetRows(rows)
	}
}

// buildInputFields creates input fields for the selected script
func (rv *RunnerView) buildInputFields(script ScriptFile) {
	rv.logger.Debug().
		Str("scriptName", script.Name).
		Str("scriptType", string(script.Type)).
		Int("signers", script.Signers).
		Int("parameters", len(script.Parameters)).
		Msg("buildInputFields called")

	rv.inputFields = make([]InputField, 0)

	// Add signer fields for transactions
	if script.Type == TypeTransaction && len(script.SignerParams) > 0 {
		for _, signerParam := range script.SignerParams {
			input := textinput.New()
			input.Placeholder = "account name"
			input.Width = 30

			// Use the actual parameter name from the prepare block
			label := signerParam.Name

			rv.inputFields = append(rv.inputFields, InputField{
				Label:    label,
				Input:    input,
				IsSigner: true,
			})
		}
	}

	// Add parameter fields
	for _, param := range script.Parameters {
		input := textinput.New()
		input.Placeholder = param.Type
		input.Width = 30

		rv.inputFields = append(rv.inputFields, InputField{
			Label:    param.Name,
			Input:    input,
			IsSigner: false,
		})
	}

	rv.activeFieldIndex = 0
	if len(rv.inputFields) > 0 {
		rv.inputFields[0].Input.Focus()
	}

	rv.logger.Debug().
		Int("totalFields", len(rv.inputFields)).
		Msg("buildInputFields completed")

	// Pre-populate from Config if this script is from a JSON file
	if script.Config != nil {
		// Load signers from config
		signerIdx := 0
		for i, field := range rv.inputFields {
			if field.IsSigner && signerIdx < len(script.Config.Signers) {
				rv.inputFields[i].Input.SetValue(script.Config.Signers[signerIdx])
				signerIdx++
			}
		}

		// Load arguments from config
		for i, field := range rv.inputFields {
			if !field.IsSigner {
				if val, exists := script.Config.Arguments[field.Label]; exists {
					rv.inputFields[i].Input.SetValue(fmt.Sprintf("%v", val))
				}
			}
		}
	}
}

// saveConfig saves the current input field values using Flow format
func (rv *RunnerView) saveConfig(configName string) string {
	selectedIdx := rv.sv.GetCursor()
	if selectedIdx < 0 || selectedIdx >= len(rv.scripts) {
		return "No script selected"
	}

	script := rv.scripts[selectedIdx]

	// Build config from current input fields using Flow format
	config := &flow.TransactionConfig{
		Name:      script.Name,
		Signers:   make([]string, 0),
		Arguments: make(map[string]interface{}),
	}

	// Collect signers
	for _, field := range rv.inputFields {
		if field.IsSigner && field.Input.Value() != "" {
			config.Signers = append(config.Signers, field.Input.Value())
		}
	}

	// Collect arguments
	for _, field := range rv.inputFields {
		if !field.IsSigner && field.Input.Value() != "" {
			config.Arguments[field.Label] = field.Input.Value()
		}
	}

	// Determine save path
	var dir string
	if script.IsFromJSON {
		// If loaded from JSON, save in same directory
		dir = filepath.Dir(script.Path)
	} else {
		// If .cdc file, save next to it
		dir = filepath.Dir(script.Path)
	}

	// Add .json extension if not present
	filename := configName
	if !strings.HasSuffix(filename, ".json") {
		filename = filename + ".json"
	}
	savePath := filepath.Join(dir, filename)

	// Save using Flow format
	if err := flow.SaveTransactionConfig(savePath, config); err != nil {
		return fmt.Sprintf("Failed to save config: %v", err)
	}

	return "" // Success
}

// tryLoadConfigFromJSON attempts to load config from a .json file next to the .cdc file
func (rv *RunnerView) tryLoadConfigFromJSON(script ScriptFile) {
	// Don't try to load if this script is already from a JSON file
	if script.IsFromJSON {
		return
	}

	// Try to find matching .json file
	jsonPath := strings.TrimSuffix(script.Path, ".cdc") + ".json"
	config, err := flow.LoadTransactionConfig(jsonPath)
	if err != nil {
		return // No JSON file or malformed, that's okay
	}

	// Load signer values
	signerIdx := 0
	for i, field := range rv.inputFields {
		if field.IsSigner && signerIdx < len(config.Signers) {
			rv.inputFields[i].Input.SetValue(config.Signers[signerIdx])
			signerIdx++
		}
	}

	// Load argument values
	for i, field := range rv.inputFields {
		if !field.IsSigner {
			if val, exists := config.Arguments[field.Label]; exists {
				rv.inputFields[i].Input.SetValue(fmt.Sprintf("%v", val))
			}
		}
	}
}

// executeScript runs a script or transaction asynchronously
func (rv *RunnerView) executeScript(script ScriptFile) tea.Cmd {
	// Capture values for async execution
	o := rv.overflow

	// Build overflow options from input fields
	var opts []overflow.OverflowInteractionOption

	// Collect signers and arguments
	signerIndex := 0
	for _, field := range rv.inputFields {
		value := field.Input.Value()
		if value == "" {
			continue
		}

		if field.IsSigner {
			// First signer uses WithSigner, rest use WithPayloadSigner
			if signerIndex == 0 {
				opts = append(opts, overflow.WithSigner(value))
			} else {
				opts = append(opts, overflow.WithPayloadSigner(value))
			}
			signerIndex++
		} else {
			opts = append(opts, overflow.WithArg(field.Label, value))
		}
	}

	return func() tea.Msg {
		if o == nil {
			return ExecutionCompleteMsg{
				Error: fmt.Errorf("overflow not initialized"),
			}
		}

		// Use config.Name for execution if available (for JSON-based scripts)
		// Otherwise use script.Name
		scriptName := script.Name
		if script.Config != nil {
			scriptName = script.Config.Name
		}

		if script.Type == TypeScript {
			// Execute script with options
			result := o.Script(scriptName, opts...)
			return ExecutionCompleteMsg{
				ScriptResult: result,
				IsScript:     true,
				Error:        result.Err,
			}
		} else {
			// Execute transaction with options
			result := o.Tx(scriptName, opts...)
			return ExecutionCompleteMsg{
				TxResult: result,
				IsScript: false,
				Error:    result.Err,
			}
		}
	}
}

// scanFiles scans for .cdc files in scripts and transactions folders
func (rv *RunnerView) scanFiles() {
	var files []ScriptFile

	// Paths to scan
	scriptDirs := []string{"scripts", "cadence/scripts"}
	txDirs := []string{"transactions", "cadence/transactions"}

	// Scan scripts
	for _, dir := range scriptDirs {
		rv.scanDirectory(dir, TypeScript, &files)
	}

	// Scan transactions
	for _, dir := range txDirs {
		rv.scanDirectory(dir, TypeTransaction, &files)
	}

	rv.scripts = files
}

// findCdcFile finds a .cdc file by name in the appropriate directory
func (rv *RunnerView) findCdcFile(name string, scriptType ScriptType) string {
	var searchDirs []string
	if scriptType == TypeScript {
		searchDirs = []string{"scripts", "cadence/scripts"}
	} else {
		searchDirs = []string{"transactions", "cadence/transactions"}
	}

	for _, dir := range searchDirs {
		// Try with and without network suffixes
		for _, suffix := range []string{"", ".emulator", ".testnet", ".mainnet"} {
			path := filepath.Join(dir, name+suffix+".cdc")
			if _, err := os.Stat(path); err == nil {
				return path
			}
		}
	}
	return ""
}

// scanDirectory scans a directory for .cdc and .json files
func (rv *RunnerView) scanDirectory(dir string, scriptType ScriptType, files *[]ScriptFile) {
	// Check if directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return
	}

	// Walk the directory
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		ext := filepath.Ext(path)

		// Handle JSON configuration files
		if ext == ".json" {
			rv.logger.Debug().
				Str("jsonPath", path).
				Msg("Found JSON file")

			config, err := flow.LoadTransactionConfig(path)
			if err != nil {
				rv.logger.Debug().
					Str("jsonPath", path).
					Err(err).
					Msg("Failed to load JSON config - skipping")
				return nil
			}

			rv.logger.Debug().
				Str("jsonPath", path).
				Str("configName", config.Name).
				Msg("Loaded JSON config - looking for .cdc file")

			// Find the referenced .cdc file
			cdcPath := rv.findCdcFile(config.Name, scriptType)
			if cdcPath == "" {
				rv.logger.Debug().
					Str("jsonPath", path).
					Str("configName", config.Name).
					Msg("Referenced .cdc file not found - skipping JSON")
				return nil
			}

			rv.logger.Debug().
				Str("jsonPath", path).
				Str("cdcPath", cdcPath).
				Msg("Found matching .cdc file for JSON config")

			code, err := os.ReadFile(cdcPath)
			if err != nil {
				return nil
			}

			// Detect network from config name
			configNetwork := rv.detectNetwork(config.Name)

			// Use JSON filename (without extension) as display name to make configs unique
			// The actual execution name (config.Name) is stored in the Config object
			jsonFilename := filepath.Base(path)
			displayName := strings.TrimSuffix(jsonFilename, ".json")

			codeStr := string(code)
			script := ScriptFile{
				Name:            displayName, // Use JSON filename for display
				Path:            path,
				Type:            scriptType,
				Code:            codeStr,
				HighlightedCode: chroma.HighlightCadence(codeStr),
				Config:          config,
				IsFromJSON:      true,
				Network:         configNetwork,
			}

			// Parse parameters and signers from the .cdc file
			rv.parseScriptFile(&script)

			*files = append(*files, script)
			return nil
		}

		// Handle .cdc files
		if ext != ".cdc" {
			return nil
		}

		// Skip test files
		if strings.Contains(path, "_test.cdc") {
			return nil
		}

		// Read file content
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		codeStr := string(content)

		// Calculate relative path from base directory to preserve nested folder structure
		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			// Fallback to basename if we can't get relative path
			relPath = filepath.Base(path)
		}
		// Remove .cdc extension to get the name overflow expects
		name := strings.TrimSuffix(relPath, ".cdc")

		// Detect network from filename suffix (use base filename for detection)
		basename := filepath.Base(path)
		basenameWithoutExt := strings.TrimSuffix(basename, ".cdc")
		network := rv.detectNetwork(basenameWithoutExt)
		// Remove network suffix from display name if present
		displayName := rv.removeNetworkSuffix(name)

		script := ScriptFile{
			Name:            displayName,
			Path:            path,
			Type:            scriptType,
			Code:            codeStr,
			HighlightedCode: chroma.HighlightCadence(codeStr),
			Network:         network,
		}

		// Parse parameters and signers from code
		rv.parseScriptFile(&script)

		*files = append(*files, script)
		return nil
	})
}

// addScriptRow adds a script to the splitview
func (rv *RunnerView) addScriptRow(script ScriptFile) {
	// Type: "Script" or "Tx", with "*" if prefilled from config
	typeStr := "Script"
	if script.Type == TypeTransaction {
		typeStr = "Tx"
	}
	if script.IsFromJSON && script.Config != nil {
		typeStr += "*"
	}

	// Use script.Name which contains relative path without .cdc extension
	// e.g., "transfer" or "nft/create_nft"
	row := table.Row{
		typeStr,
		script.Name,
		script.Network,
	}

	// Build detail content
	content := rv.buildScriptDetail(script)

	// Use highlighted code if available, otherwise raw code
	codeToShow := script.HighlightedCode
	if codeToShow == "" {
		codeToShow = script.Code
	}

	// Log row details for debugging
	rv.logger.Debug().
		Str("scriptName", script.Name).
		Int("contentLength", len(content)).
		Int("codeLength", len(codeToShow)).
		Msg("Adding script row")

	// Add to splitview with syntax-highlighted code
	rv.sv.AddRow(splitview.NewRowData(row).WithContent(content).WithCode(codeToShow))
}

// buildScriptDetail builds the detail content for a script
func (rv *RunnerView) buildScriptDetail(script ScriptFile) string {
	fieldStyle := lipgloss.NewStyle().Bold(true).Foreground(secondaryColor)
	valueStyle := lipgloss.NewStyle().Foreground(accentColor)

	renderField := func(label, value string) string {
		return fieldStyle.Render(fmt.Sprintf("%-10s", label+":")) + " " + valueStyle.Render(value) + "\n"
	}

	var details strings.Builder

	typeStr := "Script"
	if script.Type == TypeTransaction {
		typeStr = "Transaction"
	}

	displayPath := script.Path
	displayName := script.Name
	details.WriteString(renderField("Type", typeStr))
	details.WriteString(renderField("Name", displayName))
	details.WriteString(renderField("Path", displayPath))
	details.WriteString(renderField("Network", script.Network))

	// Show indicator if pre-filled from config
	if script.IsFromJSON && script.Config != nil {
		configIcon := lipgloss.NewStyle().Foreground(accentColor).Render("📋")
		details.WriteString(renderField("Config", configIcon+" Pre-filled"))
	}

	details.WriteString("\n")

	// Show input forms if in fullscreen mode and have fields
	rv.logger.Debug().
		Bool("isFullscreen", rv.sv.IsFullscreen()).
		Int("inputFieldsCount", len(rv.inputFields)).
		Msg("buildScriptDetail checking if should show input fields")

	if rv.sv.IsFullscreen() && len(rv.inputFields) > 0 {
		// In fullscreen mode - show interactive input fields separated by type

		// Show signers section
		hasSigners := false
		for _, field := range rv.inputFields {
			if field.IsSigner {
				hasSigners = true
				break
			}
		}

		if hasSigners {
			details.WriteString("\n" + fieldStyle.Render("Signers:") + "\n\n")
			for i, field := range rv.inputFields {
				if !field.IsSigner {
					continue
				}

				// Highlight active field
				labelText := field.Label + ":"
				if i == rv.activeFieldIndex {
					if rv.editingField {
						labelText = fieldStyle.Foreground(primaryColor).Render("▶ " + labelText)
					} else {
						labelText = fieldStyle.Foreground(solarYellow).Render("• " + labelText)
					}
				} else {
					labelText = fieldStyle.Render("  " + labelText)
				}

				details.WriteString(labelText + "\n")
				details.WriteString("  " + field.Input.View() + "\n\n")
			}
		}

		// Show arguments section
		hasArguments := false
		for _, field := range rv.inputFields {
			if !field.IsSigner {
				hasArguments = true
				break
			}
		}

		if hasArguments {
			details.WriteString(fieldStyle.Render("Arguments:") + "\n\n")
			for i, field := range rv.inputFields {
				if field.IsSigner {
					continue
				}

				// Highlight active field
				labelText := field.Label + ":"
				if i == rv.activeFieldIndex {
					if rv.editingField {
						labelText = fieldStyle.Foreground(primaryColor).Render("▶ " + labelText)
					} else {
						labelText = fieldStyle.Foreground(solarYellow).Render("• " + labelText)
					}
				} else {
					labelText = fieldStyle.Render("  " + labelText)
				}

				details.WriteString(labelText + "\n")
				details.WriteString("  " + field.Input.View() + "\n\n")
			}
		}

		// Show save dialog if active
		if rv.savingConfig {
			details.WriteString("\n" + fieldStyle.Render("Save Config As:") + "\n")
			details.WriteString(rv.saveInput.View() + "\n")
			if rv.saveError != "" {
				details.WriteString(lipgloss.NewStyle().Foreground(errorColor).Render(rv.saveError) + "\n")
			}
		} else {
			// Show hint to run in fullscreen mode
			details.WriteString("\n" + fieldStyle.Render("Press 'r' to run, 's' to save config") + "\n")
		}
	} else {
		// In split view mode - show merged parameters/signers info with values if available

		// Show signers with values from config if available
		if len(script.SignerParams) > 0 {
			details.WriteString(fieldStyle.Render("Signers") + "\n")
			for i, signer := range script.SignerParams {
				// Get value from config if available
				value := ""
				if script.Config != nil && i < len(script.Config.Signers) {
					value = fmt.Sprintf(": %s", valueStyle.Render(script.Config.Signers[i]))
				}
				details.WriteString(fmt.Sprintf("  %s%s\n", valueStyle.Render(signer.Name), value))
			}
			details.WriteString("\n")
		}

		// Show arguments with values from config if available
		if len(script.Parameters) > 0 {
			details.WriteString(fieldStyle.Render("Arguments") + "\n")
			for _, param := range script.Parameters {
				// Get value from config if available
				value := ""
				if script.Config != nil {
					if val, exists := script.Config.Arguments[param.Name]; exists {
						valueStr := fmt.Sprintf("%v", val)
						// Truncate long values
						if len(valueStr) > 40 {
							valueStr = valueStr[:37] + "..."
						}
						value = fmt.Sprintf(" = %s", valueStyle.Render(valueStr))
					}
				}
				if value == "" {
					// No value, show type
					value = fmt.Sprintf(" (%s)", param.Type)
				}
				details.WriteString(fmt.Sprintf("  %s%s\n", valueStyle.Render(param.Name), value))
			}
			details.WriteString("\n")
		}
	}

	// Show execution results inline
	// Show execution status
	rv.logger.Debug().
		Bool("executing", rv.executing).
		Bool("hasError", rv.executionError != nil).
		Bool("hasResult", rv.executionResult != "").
		Msg("buildScriptDetail execution status")

	if rv.executing {
		details.WriteString(lipgloss.NewStyle().
			Foreground(accentColor).
			Render("\n⏳ Executing...\n"))
	} else if rv.executionError != nil {
		details.WriteString(lipgloss.NewStyle().
			Foreground(errorColor).
			Render(fmt.Sprintf("\n❌ Error: %s\n\n", rv.executionError.Error())))
	} else if rv.executionResult != "" {
		details.WriteString(lipgloss.NewStyle().
			Foreground(successColor).
			Render(fmt.Sprintf("\n%s\n\n", rv.executionResult)))
	}

	// Add code section header (matches transactions view format)
	if script.Code != "" {
		codeLabel := "Script:"
		if script.Type == TypeTransaction {
			codeLabel = "Transaction:"
		}
		details.WriteString("\n" + fieldStyle.Render(fmt.Sprintf("%-12s", codeLabel)))
	}

	return details.String()
}

// parseScriptFile extracts parameters and signers from cadence code using AST parser
func (rv *RunnerView) parseScriptFile(script *ScriptFile) {
	code := []byte(script.Code)

	// Parse the Cadence program
	program, err := parser.ParseProgram(nil, code, parser.Config{})
	if err != nil {
		// If parsing fails, leave parameters empty
		return
	}

	// Check if it's a transaction (Cadence files can only have one transaction declaration)
	txDeclarations := program.TransactionDeclarations()
	if len(txDeclarations) > 0 {
		txd := txDeclarations[0]
		// Parse transaction parameters
		if txd.ParameterList != nil {
			script.Parameters = rv.parseParameterList(txd.ParameterList)
		}

		// Parse signers from prepare block
		if txd.Prepare != nil && txd.Prepare.FunctionDeclaration.ParameterList != nil {
			prepareParams := txd.Prepare.FunctionDeclaration.ParameterList
			script.SignerParams = rv.parseParameterList(prepareParams)
			script.Signers = len(script.SignerParams)
		}
		return
	}

	// If not a transaction, check for script (function declaration)
	functionDeclaration := sema.FunctionEntryPointDeclaration(program)
	if functionDeclaration != nil && functionDeclaration.ParameterList != nil {
		script.Parameters = rv.parseParameterList(functionDeclaration.ParameterList)
	}
}

// parseParameterList converts AST parameter list to our Parameter slice
func (rv *RunnerView) parseParameterList(paramList *ast.ParameterList) []Parameter {
	var params []Parameter

	if paramList == nil {
		return params
	}

	for _, param := range paramList.Parameters {
		paramName := param.Identifier.Identifier
		paramType := ""

		// Get type annotation as string
		if param.TypeAnnotation != nil && param.TypeAnnotation.Type != nil {
			paramType = param.TypeAnnotation.Type.String()
		}

		params = append(params, Parameter{
			Name: paramName,
			Type: paramType,
		})
	}

	return params
}
