package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/bjartek/aether/pkg/aether"
	"github.com/bjartek/overflow/v2"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/parser"
	"github.com/onflow/cadence/sema"
)

// ScriptType represents whether a file is a script or transaction
type ScriptType string

const (
	TypeScript      ScriptType = "Script"
	TypeTransaction ScriptType = "Transaction"
)

// ExecutionCompleteMsg is sent when script/transaction execution completes
type ExecutionCompleteMsg struct {
	ScriptResult *overflow.OverflowScriptResult
	TxResult     *overflow.OverflowResult
	IsScript     bool
	Error        error
}

// Parameter represents a parameter in a script or transaction
type Parameter struct {
	Name string
	Type string
}

// ScriptFile represents a Cadence script or transaction file
type ScriptFile struct {
	Name       string
	Path       string
	Type       ScriptType
	Parameters []Parameter
	Signers    int // Number of signers needed for transactions
	Code       string
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
			key.WithHelp("enter", "edit field"),
		),
		Run: key.NewBinding(
			key.WithKeys("ctrl+r", "r"),
			key.WithHelp("r/ctrl+r", "run"),
		),
		NextField: key.NewBinding(
			key.WithKeys("ctrl+n"),
			key.WithHelp("ctrl+n", "next field"),
		),
		PrevField: key.NewBinding(
			key.WithKeys("ctrl+p"),
			key.WithHelp("ctrl+p", "prev field"),
		),
	}
}

// RunnerView manages the script/transaction runner interface
type RunnerView struct {
	mu               sync.RWMutex
	table            table.Model
	codeViewport     viewport.Model
	keys             RunnerKeyMap
	ready            bool
	scripts          []ScriptFile
	width            int
	height           int
	accountRegistry  *aether.AccountRegistry
	inputFields      []InputField
	activeFieldIndex int
	editingField     bool
	availableSigners []string // List of available signer names
	overflow         *overflow.OverflowState
	spinner          spinner.Model
	isExecuting      bool
	executionResult  string
	executionError   error
}

// NewRunnerView creates a new runner view
func NewRunnerView() *RunnerView {
	columns := []table.Column{
		{Title: "Type", Width: 12},
		{Title: "Name", Width: 30},
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

	// Create viewport for code display
	vp := viewport.New(0, 0)
	vp.Style = lipgloss.NewStyle()

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	rv := &RunnerView{
		table:            t,
		codeViewport:     vp,
		keys:             DefaultRunnerKeyMap(),
		ready:            false,
		scripts:          make([]ScriptFile, 0),
		inputFields:      make([]InputField, 0),
		activeFieldIndex: 0,
		editingField:     false,
		spinner:          sp,
		isExecuting:      false,
	}

	// Scan for scripts and transactions
	rv.scanFiles()

	return rv
}

// scanFiles scans for .cdc files in scripts and transactions folders
func (rv *RunnerView) scanFiles() {
	var files []ScriptFile

	// Paths to scan
	scanPaths := []struct {
		dir string
		typ ScriptType
	}{
		{"scripts", TypeScript},
		{"transactions", TypeTransaction},
		{"cadence/scripts", TypeScript},
		{"cadence/transactions", TypeTransaction},
	}

	for _, sp := range scanPaths {
		if _, err := os.Stat(sp.dir); os.IsNotExist(err) {
			continue
		}

		err := filepath.Walk(sp.dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			if filepath.Ext(path) != ".cdc" {
				return nil
			}

			// Skip test files
			if strings.Contains(path, "_test.cdc") {
				return nil
			}

			code, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			script := ScriptFile{
				Name: strings.TrimSuffix(filepath.Base(path), ".cdc"),
				Path: path,
				Type: sp.typ,
				Code: string(code),
			}

			// Parse parameters and signers
			rv.parseScriptFile(&script)

			files = append(files, script)
			return nil
		})

		if err != nil {
			// Log error but continue
			continue
		}
	}

	// Sort by name
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name < files[j].Name
	})

	rv.mu.Lock()
	rv.scripts = files
	rv.refreshTable()

	// Setup input fields for first script if any
	if len(rv.scripts) > 0 {
		rv.setupInputFields(rv.scripts[0])
		rv.updateCodeViewport(rv.scripts[0])
	}
	rv.mu.Unlock()
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

	// Check if it's a transaction
	for _, txd := range program.TransactionDeclarations() {
		// Parse transaction parameters
		if txd.ParameterList != nil {
			script.Parameters = rv.parseParameterList(txd.ParameterList)
		}

		// Parse signers from prepare block
		if txd.Prepare != nil && txd.Prepare.FunctionDeclaration.ParameterList != nil {
			prepareParams := txd.Prepare.FunctionDeclaration.ParameterList
			script.Signers = len(prepareParams.ParametersByIdentifier())
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

// refreshTable updates the table rows from scripts
func (rv *RunnerView) refreshTable() {
	rows := make([]table.Row, len(rv.scripts))
	for i, script := range rv.scripts {
		rows[i] = table.Row{
			string(script.Type),
			script.Name,
		}
	}
	rv.table.SetRows(rows)
}

// setupInputFields creates input fields for the selected script
func (rv *RunnerView) setupInputFields(script ScriptFile) {
	rv.inputFields = make([]InputField, 0)

	// Add parameter input fields
	for _, param := range script.Parameters {
		ti := textinput.New()
		ti.Placeholder = param.Type
		ti.CharLimit = 200
		ti.Width = 40

		field := InputField{
			Label:    param.Name,
			TypeHint: param.Type,
			Input:    ti,
			IsSigner: false,
		}
		rv.inputFields = append(rv.inputFields, field)
	}

	// Add signer selection fields for transactions
	if script.Type == TypeTransaction {
		for i := 0; i < script.Signers; i++ {
			ti := textinput.New()
			ti.Placeholder = "Select signer (use friendly name)"
			ti.CharLimit = 50
			ti.Width = 40

			field := InputField{
				Label:    fmt.Sprintf("Signer %d", i+1),
				TypeHint: "&Account",
				Input:    ti,
				IsSigner: true,
			}
			rv.inputFields = append(rv.inputFields, field)
		}
	}

	rv.activeFieldIndex = 0
	rv.editingField = false
}

// updateCodeViewport updates the viewport with the selected script's code
func (rv *RunnerView) updateCodeViewport(script ScriptFile) {
	if rv.codeViewport.Width == 0 || rv.codeViewport.Height == 0 {
		return
	}
	rv.codeViewport.SetContent(script.Code)
	rv.codeViewport.GotoTop()
}

func (rv *RunnerView) Init() tea.Cmd {
	return rv.spinner.Tick
}

// SetAccountRegistry sets the account registry for signer name resolution
func (rv *RunnerView) SetAccountRegistry(registry *aether.AccountRegistry) {
	rv.mu.Lock()
	defer rv.mu.Unlock()

	rv.accountRegistry = registry

	// Update available signers list
	if registry != nil {
		// Get all account names from registry
		// Note: We'd need a method to get all names, for now we'll populate as we go
		rv.availableSigners = []string{"alice", "bob", "charlie"} // TODO: Get from registry
	}
}

// SetOverflow sets the overflow state for executing transactions/scripts
func (rv *RunnerView) SetOverflow(o *overflow.OverflowState) {
	rv.mu.Lock()
	defer rv.mu.Unlock()
	rv.overflow = o
}

// executeScriptCmd executes the selected script or transaction
func (rv *RunnerView) executeScriptCmd(script ScriptFile) tea.Cmd {
	// Capture values while holding the lock
	rv.mu.RLock()
	o := rv.overflow

	// Build overflow options from input fields
	var opts []overflow.OverflowInteractionOption

	// Collect signers and arguments
	for _, field := range rv.inputFields {
		value := field.Input.Value()
		if value == "" {
			continue
		}

		if field.IsSigner {
			opts = append(opts, overflow.WithSigner(value))
		} else {
			opts = append(opts, overflow.WithArg(field.Label, value))
		}
	}
	rv.mu.RUnlock()

	// Execute asynchronously without holding the lock
	return func() tea.Msg {
		// Recover from any panics during execution
		var result tea.Msg
		defer func() {
			if r := recover(); r != nil {
				// Send error message if panic occurs
				result = ExecutionCompleteMsg{
					Error: fmt.Errorf("panic during execution: %v", r),
				}
			}
		}()

		if o == nil {
			return ExecutionCompleteMsg{
				Error: fmt.Errorf("overflow not initialized"),
			}
		}

		// Execute based on script type
		// Overflow expects script name without .cdc extension
		scriptName := script.Name

		if script.Type == TypeTransaction {
			txResult := o.Tx(scriptName, opts...)
			result = ExecutionCompleteMsg{
				TxResult: txResult,
				IsScript: false,
				Error:    txResult.Err,
			}
		} else {
			scriptResult := o.Script(scriptName, opts...)
			result = ExecutionCompleteMsg{
				ScriptResult: scriptResult,
				IsScript:     true,
				Error:        scriptResult.Err,
			}
		}

		return result
	}
}

// formatScriptResult formats the script result for display
func (rv *RunnerView) formatScriptResult(result *overflow.OverflowScriptResult) string {
	if result == nil {
		return "✓ Script executed successfully"
	}

	var b strings.Builder

	b.WriteString("✓ Script executed successfully\n\n")
	b.WriteString("Output:\n")
	b.WriteString(fmt.Sprintf("%v", result.Output))

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

// Update handles messages for the runner view
func (rv *RunnerView) Update(msg tea.Msg, width, height int) tea.Cmd {
	rv.mu.Lock()
	rv.width = width
	rv.height = height
	rv.mu.Unlock()

	switch msg := msg.(type) {
	case ExecutionCompleteMsg:
		rv.mu.Lock()
		rv.isExecuting = false
		if msg.Error != nil {
			rv.executionError = msg.Error
			rv.executionResult = ""
		} else {
			rv.executionError = nil
			// Format result based on type
			if msg.IsScript && msg.ScriptResult != nil {
				// Show detailed script output
				rv.executionResult = rv.formatScriptResult(msg.ScriptResult)
			} else if !msg.IsScript && msg.TxResult != nil {
				// Show detailed transaction result
				rv.executionResult = rv.formatTransactionResult(msg.TxResult)
			} else {
				rv.executionResult = "✓ Execution successful"
			}
		}
		rv.mu.Unlock()
		return nil

	case spinner.TickMsg:
		rv.mu.RLock()
		isExecuting := rv.isExecuting
		rv.mu.RUnlock()

		if isExecuting {
			rv.mu.Lock()
			var cmd tea.Cmd
			rv.spinner, cmd = rv.spinner.Update(msg)
			rv.mu.Unlock()
			return cmd
		}
		return nil

	case tea.WindowSizeMsg:
		rv.mu.Lock()
		if !rv.ready {
			rv.ready = true
		}
		// Table takes 30% width, rest for details
		tableWidth := int(float64(width) * 0.3)
		rv.table.SetWidth(tableWidth)
		rv.table.SetHeight(height)

		// Code viewport at bottom of detail pane
		rv.codeViewport.Width = width - tableWidth - 4
		rv.codeViewport.Height = height / 2 // Half for inputs, half for code
		rv.mu.Unlock()

	case tea.KeyMsg:
		// If editing a field, pass input to textinput
		rv.mu.RLock()
		isEditing := rv.editingField
		hasFields := len(rv.inputFields) > 0
		rv.mu.RUnlock()

		if isEditing && hasFields {
			switch {
			case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
				rv.mu.Lock()
				rv.editingField = false
				rv.inputFields[rv.activeFieldIndex].Input.Blur()
				rv.mu.Unlock()
				return nil
			case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
				rv.mu.Lock()
				rv.editingField = false
				rv.inputFields[rv.activeFieldIndex].Input.Blur()
				// Move to next field
				if rv.activeFieldIndex < len(rv.inputFields)-1 {
					rv.activeFieldIndex++
				}
				rv.mu.Unlock()
				return nil
			default:
				rv.mu.Lock()
				var cmd tea.Cmd
				rv.inputFields[rv.activeFieldIndex].Input, cmd = rv.inputFields[rv.activeFieldIndex].Input.Update(msg)
				rv.mu.Unlock()
				return cmd
			}
		}

		// Handle navigation when not editing
		switch {
		case key.Matches(msg, rv.keys.Run):
			// Execute the script/transaction
			rv.mu.Lock()
			if !rv.isExecuting {
				selectedIdx := rv.table.Cursor()
				if selectedIdx >= 0 && selectedIdx < len(rv.scripts) {
					rv.isExecuting = true
					rv.executionResult = ""
					rv.executionError = nil
					script := rv.scripts[selectedIdx]
					rv.mu.Unlock()
					return tea.Batch(
						rv.executeScriptCmd(script),
						rv.spinner.Tick,
					)
				}
			}
			rv.mu.Unlock()
			return nil

		case key.Matches(msg, rv.keys.Enter):
			rv.mu.RLock()
			hasFields = len(rv.inputFields) > 0
			activeIdx := rv.activeFieldIndex
			rv.mu.RUnlock()

			if hasFields {
				rv.mu.Lock()
				rv.editingField = true
				rv.inputFields[activeIdx].Input.Focus()
				rv.mu.Unlock()
				return textinput.Blink
			}
			return nil

		case key.Matches(msg, rv.keys.Down), key.Matches(msg, rv.keys.NextField):
			rv.mu.Lock()
			if len(rv.inputFields) > 0 && rv.activeFieldIndex < len(rv.inputFields)-1 {
				rv.activeFieldIndex++
				rv.mu.Unlock()
				return nil
			}
			// Otherwise navigate table
			var cmd tea.Cmd
			rv.table, cmd = rv.table.Update(msg)
			// Update input fields for selected script
			selectedIdx := rv.table.Cursor()
			if selectedIdx >= 0 && selectedIdx < len(rv.scripts) {
				rv.setupInputFields(rv.scripts[selectedIdx])
				rv.updateCodeViewport(rv.scripts[selectedIdx])
			}
			rv.mu.Unlock()
			return cmd

		case key.Matches(msg, rv.keys.Up), key.Matches(msg, rv.keys.PrevField):
			rv.mu.Lock()
			if len(rv.inputFields) > 0 && rv.activeFieldIndex > 0 {
				rv.activeFieldIndex--
				rv.mu.Unlock()
				return nil
			}
			// Otherwise navigate table
			var cmd tea.Cmd
			rv.table, cmd = rv.table.Update(msg)
			// Update input fields for selected script
			selectedIdx := rv.table.Cursor()
			if selectedIdx >= 0 && selectedIdx < len(rv.scripts) {
				rv.setupInputFields(rv.scripts[selectedIdx])
				rv.updateCodeViewport(rv.scripts[selectedIdx])
			}
			rv.mu.Unlock()
			return cmd

		default:
			// Pass to table for other keys
			rv.mu.Lock()
			var cmd tea.Cmd
			rv.table, cmd = rv.table.Update(msg)
			rv.mu.Unlock()
			return cmd
		}
	}

	return nil
}

// View renders the runner view
func (rv *RunnerView) View() string {
	if !rv.ready {
		return "Loading runner..."
	}

	rv.mu.RLock()

	if len(rv.scripts) == 0 {
		rv.mu.RUnlock()
		return lipgloss.NewStyle().
			Foreground(mutedColor).
			Render("No scripts or transactions found in scripts/, transactions/, cadence/scripts/, or cadence/transactions/")
	}

	// Table on left (30%)
	tableWidth := int(float64(rv.width) * 0.3)
	detailWidth := rv.width - tableWidth - 2

	tableView := lipgloss.NewStyle().
		Width(tableWidth).
		Height(rv.height).
		Render(rv.table.View())

	// Detail view on right
	selectedIdx := rv.table.Cursor()
	var detailView string
	if selectedIdx >= 0 && selectedIdx < len(rv.scripts) {
		script := rv.scripts[selectedIdx]

		// Check if we need to setup fields - if so, upgrade to write lock
		needsSetup := len(rv.inputFields) == 0
		rv.mu.RUnlock()

		if needsSetup {
			rv.mu.Lock()
			// Double-check after acquiring write lock
			if len(rv.inputFields) == 0 {
				rv.setupInputFields(script)
				rv.updateCodeViewport(script)
			}
			rv.mu.Unlock()
		}

		// Re-acquire read lock for rendering
		rv.mu.RLock()
		detailView = rv.renderDetail(script, detailWidth, rv.height)
		rv.mu.RUnlock()
	} else {
		detailView = lipgloss.NewStyle().
			Width(detailWidth).
			Height(rv.height).
			Render("No script selected")
		rv.mu.RUnlock()
	}

	// Combine table and detail side by side
	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		tableView,
		detailView,
	)
}

// renderDetail renders the detail view with form and code
func (rv *RunnerView) renderDetail(script ScriptFile, width, height int) string {
	detailStyle := lipgloss.NewStyle().
		Width(width).
		Height(height).
		Padding(1, 2)

	var content strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(primaryColor)
	content.WriteString(titleStyle.Render(script.Name) + "\n")
	content.WriteString(lipgloss.NewStyle().Foreground(mutedColor).Render(string(script.Type)) + "\n\n")

	// Show spinner or execution result
	if rv.isExecuting {
		content.WriteString(rv.spinner.View() + " Executing...\n\n")
	} else if rv.executionError != nil {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
		content.WriteString(errorStyle.Render("✗ Error: "+rv.executionError.Error()) + "\n\n")
	} else if rv.executionResult != "" {
		successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Bold(true)
		content.WriteString(successStyle.Render(rv.executionResult) + "\n\n")
	}

	// Input form
	if len(rv.inputFields) > 0 {
		formTitle := lipgloss.NewStyle().Bold(true).Foreground(secondaryColor).Render("Parameters:")
		content.WriteString(formTitle + "\n\n")

		for i, field := range rv.inputFields {
			// Highlight active field
			labelStyle := lipgloss.NewStyle().Foreground(accentColor)
			if i == rv.activeFieldIndex {
				labelStyle = labelStyle.Bold(true).Foreground(solarYellow)
			}

			label := labelStyle.Render(fmt.Sprintf("%-20s", field.Label+":"))
			typeHint := lipgloss.NewStyle().Foreground(mutedColor).Render(field.TypeHint)

			content.WriteString(fmt.Sprintf("%s %s\n", label, typeHint))

			// Show input box
			inputStyle := lipgloss.NewStyle().Foreground(base0)
			if i == rv.activeFieldIndex && rv.editingField {
				inputStyle = inputStyle.BorderStyle(lipgloss.NormalBorder()).BorderForeground(primaryColor)
			}
			content.WriteString(inputStyle.Render(field.Input.View()) + "\n\n")
		}

		// Hint
		hintStyle := lipgloss.NewStyle().Foreground(mutedColor).Italic(true)
		if rv.editingField {
			content.WriteString(hintStyle.Render("Press Esc to stop editing, Enter to confirm") + "\n\n")
		} else {
			content.WriteString(hintStyle.Render("Press Enter to edit, j/k to navigate, r to run") + "\n\n")
		}
	} else {
		// No parameters - show hint to run directly
		hintStyle := lipgloss.NewStyle().Foreground(mutedColor).Italic(true)
		content.WriteString(hintStyle.Render("No parameters required. Press r to run") + "\n\n")
	}

	// Code section
	codeTitle := lipgloss.NewStyle().Bold(true).Foreground(secondaryColor).Render("Code:")
	content.WriteString(codeTitle + "\n")

	// Calculate remaining height for code viewport
	usedLines := strings.Count(content.String(), "\n") + 2
	codeHeight := height - usedLines - 4 // Leave some padding
	if codeHeight < 5 {
		codeHeight = 5
	}

	rv.codeViewport.Width = width - 4
	rv.codeViewport.Height = codeHeight

	if rv.codeViewport.Width > 0 && rv.codeViewport.Height > 0 {
		rv.updateCodeViewport(script)
		content.WriteString(rv.codeViewport.View())
	}

	return detailStyle.Render(content.String())
}

// Stop is a no-op for the runner view
func (rv *RunnerView) Stop() {
	// No cleanup needed
}
