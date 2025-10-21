package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bjartek/aether/pkg/aether"
	"github.com/bjartek/aether/pkg/chroma"
	"github.com/bjartek/aether/pkg/config"
	"github.com/bjartek/aether/pkg/flow"
	"github.com/bjartek/overflow/v2"
	"github.com/bjartek/underflow"
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
	Up               key.Binding
	Down             key.Binding
	Enter            key.Binding
	Run              key.Binding
	NextField        key.Binding
	PrevField        key.Binding
	Save             key.Binding
	Refresh          key.Binding
	ToggleFullDetail key.Binding
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
			key.WithKeys("enter", " "),
			key.WithHelp("enter/space", "toggle detail"),
		),
		Run: key.NewBinding(
			key.WithKeys("ctrl+r", "r"),
			key.WithHelp("r/ctrl+r", "run"),
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
			key.WithKeys("ctrl+l"),
			key.WithHelp("ctrl+l", "refresh list"),
		),
		ToggleFullDetail: key.NewBinding(
			key.WithKeys("enter", " "),
			key.WithHelp("enter/space", "toggle detail"),
		),
	}
}

// RunnerView manages the script/transaction runner interface
type RunnerView struct {
	table               table.Model
	codeViewport        viewport.Model
	detailViewport      viewport.Model // Viewport for full detail mode
	splitDetailViewport viewport.Model // Viewport for split view detail panel
	keys                RunnerKeyMap
	ready               bool
	scripts             []ScriptFile
	width               int
	height              int
	tableWidthPercent   int  // Configurable table width percentage
	detailWidthPercent  int  // Configurable detail width percentage
	codeWrapWidth       int  // Configurable code wrap width (0 = no wrap)
	fullDetailMode      bool // Toggle between split and full-screen detail view
	accountRegistry     *aether.AccountRegistry
	inputFields         []InputField
	activeFieldIndex    int
	editingField        bool
	availableSigners    []string // List of available signer names
	overflow            *overflow.OverflowState
	spinner             spinner.Model
	isExecuting         bool
	executionResult     string
	executionError      error
	savingConfig        bool            // True when showing save dialog
	saveInput           textinput.Model // Input for save filename
	saveError           string          // Error message from last save attempt
}

// NewRunnerView creates a new runner view with default settings
func NewRunnerView() *RunnerView {
	return NewRunnerViewWithConfig(nil)
}

// NewRunnerViewWithConfig creates a new runner view with configuration
func NewRunnerViewWithConfig(cfg *config.Config) *RunnerView {
	columns := []table.Column{
		{Title: "Type", Width: 12},
		{Title: "Name", Width: 25},
		{Title: "Network", Width: 10},
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

	// Create viewport for full detail mode
	detailVp := viewport.New(0, 0)
	detailVp.Style = lipgloss.NewStyle()

	// Create viewport for split view detail panel
	splitVp := viewport.New(0, 0)
	splitVp.Style = lipgloss.NewStyle()

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	// Create save input
	saveInput := textinput.New()
	saveInput.Placeholder = "config-name"
	saveInput.CharLimit = 50
	saveInput.Width = 40

	// Get width percentages from config
	// Use config defaults, or fallback to DefaultConfig if no config provided
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	tableWidthPercent := cfg.UI.Layout.Runner.TableWidthPercent
	detailWidthPercent := cfg.UI.Layout.Runner.DetailWidthPercent
	codeWrapWidth := cfg.UI.Layout.Runner.CodeWrapWidth

	rv := &RunnerView{
		table:               t,
		codeViewport:        vp,
		detailViewport:      detailVp,
		splitDetailViewport: splitVp,
		keys:                DefaultRunnerKeyMap(),
		ready:               false,
		scripts:             make([]ScriptFile, 0),
		inputFields:         make([]InputField, 0),
		activeFieldIndex:    0,
		editingField:        false,
		tableWidthPercent:   tableWidthPercent,
		detailWidthPercent:  detailWidthPercent,
		codeWrapWidth:       codeWrapWidth,
		fullDetailMode:      false,
		spinner:             sp,
		isExecuting:         false,
		savingConfig:        false,
		saveInput:           saveInput,
	}

	// Scan for scripts and transactions
	rv.scanFiles()

	return rv
}

// findCdcFile searches for a .cdc file by name in common directories
// The name can be a simple name like "message" or a nested path like "nested/message"
func (rv *RunnerView) findCdcFile(name string, scriptType ScriptType) string {
	// Determine which directory to search based on script type
	var dirs []string
	if scriptType == TypeScript {
		dirs = []string{"scripts", "cadence/scripts"}
	} else {
		dirs = []string{"transactions", "cadence/transactions"}
	}

	// Name may already include path like "nested/message", just add .cdc extension
	filename := name + ".cdc"
	for _, dir := range dirs {
		path := filepath.Join(dir, filename)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
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

			ext := filepath.Ext(path)

			// Handle JSON configuration files
			if ext == ".json" {
				config, err := flow.LoadTransactionConfig(path)
				if err != nil {
					// Skip malformed JSON files
					return nil
				}

				// Find the referenced .cdc file
				cdcPath := rv.findCdcFile(config.Name, sp.typ)
				if cdcPath == "" {
					// Referenced .cdc file not found, skip
					return nil
				}

				code, err := os.ReadFile(cdcPath)
				if err != nil {
					return nil
				}

				// Detect network from config name
				configNetwork := rv.detectNetwork(config.Name)
				displayName := strings.TrimSuffix(filepath.Base(path), ".json") + " (config)"

				codeStr := string(code)
				script := ScriptFile{
					Name:            displayName,
					Path:            path,
					Type:            sp.typ,
					Code:            codeStr,
					HighlightedCode: chroma.HighlightCadence(codeStr), // Highlight WITHOUT wrapping - let lipgloss wrap
					Config:          config,
					IsFromJSON:      true,
					Network:         configNetwork,
				}

				// Parse parameters and signers from the .cdc file
				rv.parseScriptFile(&script)

				files = append(files, script)
				return nil
			}
			// Handle .cdc files
			if ext == ".cdc" {
				// Skip test files
				if strings.Contains(path, "_test.cdc") {
					return nil
				}

				code, err := os.ReadFile(path)
				if err != nil {
					return err
				}

				// Calculate relative path from base directory
				// This preserves nested folder structure (e.g., nested/message.cdc -> nested/message)
				relPath, err := filepath.Rel(sp.dir, path)
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

				codeStr := string(code)
				script := ScriptFile{
					Name:            displayName,
					Path:            path,
					Type:            sp.typ,
					Code:            codeStr,
					HighlightedCode: chroma.HighlightCadence(codeStr), // Highlight WITHOUT wrapping - let lipgloss wrap
					IsFromJSON:      false,
					Network:         network,
				}

				// Parse parameters and signers
				rv.parseScriptFile(&script)

				files = append(files, script)
				return nil
			}
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

	rv.scripts = files
	rv.refreshTable()

	// Setup input fields for first script if any
	if len(rv.scripts) > 0 {
		rv.setupInputFields(rv.scripts[0])
		rv.updateCodeViewport(rv.scripts[0])
	}
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

	// If no suffix, check if code contains network-specific imports (addresses)
	// For now, return "any" - we could enhance this later
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
			script.Network,
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

		// Pre-populate from JSON config if available
		if script.Config != nil && script.Config.Arguments != nil {
			if val, ok := script.Config.Arguments[param.Name]; ok {
				// Convert interface{} to string for display
				ti.SetValue(fmt.Sprintf("%v", val))
			}
		}

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

			// Pre-populate from JSON config if available
			if script.Config != nil && script.Config.Signers != nil && i < len(script.Config.Signers) {
				ti.SetValue(script.Config.Signers[i])
			}

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
	// Use pre-wrapped and highlighted code from scan time
	content := script.HighlightedCode
	if content == "" {
		content = script.Code
	}
	rv.codeViewport.SetContent(content)
	rv.codeViewport.GotoTop()
}

// updateDetailViewport updates the viewport content for full detail mode
func (rv *RunnerView) updateDetailViewport() {
	if len(rv.scripts) == 0 {
		rv.detailViewport.SetContent("")
		return
	}

	selectedIdx := rv.table.Cursor()
	if selectedIdx >= 0 && selectedIdx < len(rv.scripts) {
		if rv.detailViewport.Width == 0 || rv.detailViewport.Height == 0 {
			return
		}
		script := rv.scripts[selectedIdx]
		// Render the full detail content with full viewport width
		content := rv.renderDetailText(script, rv.detailViewport.Width)
		rv.detailViewport.SetContent(content)
		rv.detailViewport.GotoTop()
	}
}

func (rv *RunnerView) Init() tea.Cmd {
	return rv.spinner.Tick
}

// SetAccountRegistry sets the account registry for signer name resolution
func (rv *RunnerView) SetAccountRegistry(registry *aether.AccountRegistry) {
	rv.accountRegistry = registry

	// Update available signers list from registry
	if registry != nil {
		rv.availableSigners = registry.GetAllNames()
	}
}

// SetOverflow sets the overflow state for executing transactions/scripts
func (rv *RunnerView) SetOverflow(o *overflow.OverflowState) {
	rv.overflow = o
}

// executeScriptCmd executes the selected script or transaction
func (rv *RunnerView) executeScriptCmd(script ScriptFile) tea.Cmd {
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

		// If this is from a JSON config, use the config's referenced name
		if script.IsFromJSON && script.Config != nil {
			scriptName = script.Config.Name
		}

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

// formatValue recursively formats a value with proper indentation
// Note: Underflow now handles timestamp formatting and human-readable addresses automatically
func (rv *RunnerView) formatValue(value interface{}, indent string) string {
	var b strings.Builder

	switch v := value.(type) {
	case map[string]interface{}:
		if len(v) == 0 {
			b.WriteString("<empty>")
			return b.String()
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
			b.WriteString("<empty>")
			return b.String()
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
		b.WriteString(fmt.Sprintf("%v", value))
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
	// Use underflow to convert the Result to interface{}
	value := underflow.CadenceValueToInterface(result.Result)

	b.WriteString("Output:\n")
	b.WriteString(rv.formatValue(value, ""))

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

// saveCurrentConfig saves the current input values to a JSON config file
func (rv *RunnerView) saveCurrentConfig(filename string, script ScriptFile) error {
	// Build config from current input fields
	config := &flow.TransactionConfig{
		Name:      script.Name,
		Signers:   make([]string, 0),
		Arguments: make(map[string]interface{}),
	}

	// Collect values from input fields
	for _, field := range rv.inputFields {
		value := field.Input.Value()
		if value == "" {
			continue // Skip empty fields
		}

		if field.IsSigner {
			config.Signers = append(config.Signers, value)
		} else {
			config.Arguments[field.Label] = value
		}
	}

	// If this script was loaded from JSON, use the original name for the referenced .cdc file
	if script.IsFromJSON && script.Config != nil {
		config.Name = script.Config.Name
	}

	// Determine directory based on script type
	var dir string
	if script.Type == TypeScript {
		// Try scripts first, fall back to cadence/scripts
		if _, err := os.Stat("scripts"); err == nil {
			dir = "scripts"
		} else {
			dir = "cadence/scripts"
		}
	} else {
		// Try transactions first, fall back to cadence/transactions
		if _, err := os.Stat("transactions"); err == nil {
			dir = "transactions"
		} else {
			dir = "cadence/transactions"
		}
	}

	// Ensure filename has .json extension
	if !strings.HasSuffix(filename, ".json") {
		filename += ".json"
	}

	path := filepath.Join(dir, filename)
	return flow.SaveTransactionConfig(path, config)
}

// Update handles messages for the runner view
func (rv *RunnerView) Update(msg tea.Msg, width, height int) tea.Cmd {
	rv.width = width
	rv.height = height

	switch msg := msg.(type) {
	case ExecutionCompleteMsg:
		rv.isExecuting = false
		if msg.Error != nil {
			rv.executionError = msg.Error
			rv.executionResult = ""
		} else {
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
		return nil

	case RescanFilesMsg:
		// Rescan files synchronously to update table
		rv.scanFiles()
		return nil

	case spinner.TickMsg:
		if rv.isExecuting {
			var cmd tea.Cmd
			rv.spinner, cmd = rv.spinner.Update(msg)
			return cmd
		}
		return nil

	case tea.WindowSizeMsg:
		if !rv.ready {
			rv.ready = true
		}
		// Table width using configured percentage
		tableWidth := int(float64(width) * float64(rv.tableWidthPercent) / 100.0)
		detailWidth := max(10, width-tableWidth-2)
		rv.table.SetWidth(tableWidth)
		rv.table.SetHeight(height)

		// Code viewport at bottom of detail pane
		rv.codeViewport.Width = width - tableWidth - 4
		rv.codeViewport.Height = height / 2 // Half for inputs, half for code

		// Update viewport size for full detail mode
		hint := lipgloss.NewStyle().
			Foreground(mutedColor).
			Render("Press Tab or Esc to return to table view | j/k to scroll | r to run | s to save")
		hintHeight := lipgloss.Height(hint) + 2 // +2 for spacing
		rv.detailViewport.Width = width
		rv.detailViewport.Height = height - hintHeight

		// Update split view detail viewport size
		rv.splitDetailViewport.Width = detailWidth
		rv.splitDetailViewport.Height = height

	case tea.KeyMsg:
		// Handle save dialog if active
		if rv.savingConfig {
			switch {
			case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
				// Cancel save
				rv.savingConfig = false
				rv.saveInput.SetValue("")
				rv.saveInput.Blur()
				rv.saveError = ""
				return nil
			case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
				// Perform save
				filename := rv.saveInput.Value()
				if filename == "" {
					rv.saveError = "Filename cannot be empty"
					return nil
				}

				selectedIdx := rv.table.Cursor()
				if selectedIdx >= 0 && selectedIdx < len(rv.scripts) {
					script := rv.scripts[selectedIdx]
					err := rv.saveCurrentConfig(filename, script)
					if err != nil {
						rv.saveError = err.Error()
					} else {
						rv.savingConfig = false
						rv.saveInput.SetValue("")
						rv.saveInput.Blur()
						rv.saveError = ""
						// Rescan to pick up the new file
						rv.scanFiles()
						return nil
					}
				}
				return nil
			default:
				// Pass input to save textinput
				var cmd tea.Cmd
				rv.saveInput, cmd = rv.saveInput.Update(msg)
				return cmd
			}
		}

		// Check if we're in full detail mode
		inFullDetail := rv.fullDetailMode
		isEditing := rv.editingField
		hasFields := len(rv.inputFields) > 0

		// Handle Esc to exit full detail view (only if not editing a field)
		if inFullDetail && !isEditing && key.Matches(msg, key.NewBinding(key.WithKeys("esc"))) {
			rv.fullDetailMode = false
			return nil
		}

		// If editing a field (only in full detail mode), pass input to textinput
		if inFullDetail && isEditing && hasFields {
			switch {
			case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
				rv.editingField = false
				rv.inputFields[rv.activeFieldIndex].Input.Blur()
				return nil
			case key.Matches(msg, rv.keys.NextField): // tab
				rv.inputFields[rv.activeFieldIndex].Input.Blur()
				// Move to next field and auto-focus it
				if rv.activeFieldIndex < len(rv.inputFields)-1 {
					rv.activeFieldIndex++
					rv.inputFields[rv.activeFieldIndex].Input.Focus()
					return textinput.Blink
				}
				// Last field - exit editing mode
				rv.editingField = false
				return nil
			case key.Matches(msg, rv.keys.PrevField): // shift+tab
				rv.inputFields[rv.activeFieldIndex].Input.Blur()
				// Move to previous field and auto-focus it
				if rv.activeFieldIndex > 0 {
					rv.activeFieldIndex--
					rv.inputFields[rv.activeFieldIndex].Input.Focus()
					return textinput.Blink
				}
				// First field - exit editing mode
				rv.editingField = false
				return nil
			case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
				// Enter just exits editing mode
				rv.editingField = false
				rv.inputFields[rv.activeFieldIndex].Input.Blur()
				return nil
			default:
				var cmd tea.Cmd
				rv.inputFields[rv.activeFieldIndex].Input, cmd = rv.inputFields[rv.activeFieldIndex].Input.Update(msg)
				return cmd
			}
		}

		// Handle navigation when not editing
		switch {
		case key.Matches(msg, rv.keys.Run):
			// Execute the script/transaction
			if !rv.isExecuting {
				selectedIdx := rv.table.Cursor()
				if selectedIdx >= 0 && selectedIdx < len(rv.scripts) {
					rv.isExecuting = true
					rv.executionResult = ""
					rv.executionError = nil
					script := rv.scripts[selectedIdx]
					return tea.Batch(
						rv.executeScriptCmd(script),
						rv.spinner.Tick,
					)
				}
			}
			return nil

		case key.Matches(msg, rv.keys.Enter):
			// Enter/Space toggles full detail mode (when not editing)
			if !isEditing {
				wasFullMode := rv.fullDetailMode
				rv.fullDetailMode = !rv.fullDetailMode
				// When entering full detail mode, auto-focus first field if available
				if !wasFullMode && rv.fullDetailMode {
					rv.updateDetailViewport()
					if len(rv.inputFields) > 0 {
						rv.activeFieldIndex = 0
						rv.editingField = true
						rv.inputFields[0].Input.Focus()
						return textinput.Blink
					}
				}
				return nil
			}
			return nil

		case key.Matches(msg, rv.keys.Save):
			// Activate save dialog
			rv.savingConfig = true
			rv.saveInput.Focus()
			rv.saveError = ""
			return textinput.Blink

		case key.Matches(msg, rv.keys.Refresh):
			// Trigger rescan of files
			return func() tea.Msg {
				return RescanFilesMsg{}
			}

		case key.Matches(msg, key.NewBinding(key.WithKeys("i"))):
			// 'i' to start editing in full detail mode
			if inFullDetail && !isEditing {
				hasFields = len(rv.inputFields) > 0
				activeIdx := rv.activeFieldIndex

				if hasFields {
					rv.editingField = true
					rv.inputFields[activeIdx].Input.Focus()
					return textinput.Blink
				}
			}
			return nil

		case key.Matches(msg, rv.keys.Down), key.Matches(msg, rv.keys.NextField):
			// In full detail mode, scroll viewport or navigate form fields
			if inFullDetail {
				// If not editing, scroll the viewport
				if !isEditing {
					rv.detailViewport.LineDown(1)
					return nil
				}
				// If editing, navigate form fields
				if len(rv.inputFields) > 0 && rv.activeFieldIndex < len(rv.inputFields)-1 {
					rv.activeFieldIndex++
					return nil
				}
				return nil
			}
			// In split view, navigate table
			var cmd tea.Cmd
			rv.table, cmd = rv.table.Update(msg)
			// Update input fields for selected script and clear previous results
			selectedIdx := rv.table.Cursor()
			if selectedIdx >= 0 && selectedIdx < len(rv.scripts) {
				rv.setupInputFields(rv.scripts[selectedIdx])
				rv.updateCodeViewport(rv.scripts[selectedIdx])
				// Clear previous execution results when switching scripts
				rv.executionResult = ""
				rv.executionError = nil
			}
			return cmd

		case key.Matches(msg, rv.keys.Up), key.Matches(msg, rv.keys.PrevField):
			// In full detail mode, scroll viewport or navigate form fields
			if inFullDetail {
				// If not editing, scroll the viewport
				if !isEditing {
					rv.detailViewport.LineUp(1)
					return nil
				}
				// If editing, navigate form fields
				if len(rv.inputFields) > 0 && rv.activeFieldIndex > 0 {
					rv.activeFieldIndex--
					return nil
				}
				return nil
			}
			// In split view, navigate table
			var cmd tea.Cmd
			rv.table, cmd = rv.table.Update(msg)
			// Update input fields for selected script and clear previous results
			selectedIdx := rv.table.Cursor()
			if selectedIdx >= 0 && selectedIdx < len(rv.scripts) {
				rv.setupInputFields(rv.scripts[selectedIdx])
				rv.updateCodeViewport(rv.scripts[selectedIdx])
				// Clear previous execution results when switching scripts
				rv.executionResult = ""
				rv.executionError = nil
			}
			return cmd

		default:
			// In split view, pass to table for other keys
			if !inFullDetail {
				var cmd tea.Cmd
				rv.table, cmd = rv.table.Update(msg)
				return cmd
			}
		}
	}

	return nil
}

// View renders the runner view
func (rv *RunnerView) View() string {
	if !rv.ready {
		return "Loading runner..."
	}

	if len(rv.scripts) == 0 {
		return lipgloss.NewStyle().
			Foreground(mutedColor).
			Render("No scripts or transactions found in scripts/, transactions/, cadence/scripts/, or cadence/transactions/")
	}

	// Full detail mode - show detail with interactive forms in a scrollable viewport
	if rv.fullDetailMode {
		selectedIdx := rv.table.Cursor()
		if selectedIdx >= 0 && selectedIdx < len(rv.scripts) {
			script := rv.scripts[selectedIdx]

			// Set viewport to full screen dimensions
			rv.detailViewport.Width = rv.width
			rv.detailViewport.Height = rv.height - 3 // Leave room for hint

			// Render all content into viewport for scrolling
			content := rv.renderDetailForViewport(script)
			rv.detailViewport.SetContent(content)

			hint := lipgloss.NewStyle().
				Foreground(mutedColor).
				Render("Press Tab or Esc to return to table view")
			return hint + "\n\n" + rv.detailViewport.View()
		}
		return "No script selected"
	}

	// Split view mode - table on left, detail on right
	// Calculate widths using configured percentages
	tableWidth := int(float64(rv.width) * float64(rv.tableWidthPercent) / 100.0)

	// Update split detail viewport with current script (before rendering table - same order as transactions_view)
	selectedIdx := rv.table.Cursor()
	if selectedIdx >= 0 && selectedIdx < len(rv.scripts) {
		currentWidth := rv.splitDetailViewport.Width
		if currentWidth == 0 {
			currentWidth = 100 // Default
		}

		script := rv.scripts[selectedIdx]

		// Just render fresh every time - no caching, no optimization
		content := rv.renderDetailText(script, currentWidth)
		wrappedContent := lipgloss.NewStyle().Width(currentWidth).Render(content)

		rv.splitDetailViewport.SetContent(wrappedContent)
		rv.splitDetailViewport.GotoTop()
	} else {
		rv.splitDetailViewport.SetContent("No script selected")
		rv.splitDetailViewport.GotoTop()
	}

	// Style table
	tableView := lipgloss.NewStyle().
		Width(tableWidth).
		MaxHeight(rv.height).
		Render(rv.table.View())

	// Render split detail viewport (viewport itself handles width/height constraints)
	detailView := rv.splitDetailViewport.View()

	// Combine table and detail side by side
	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		tableView,
		detailView,
	)
}

// renderDetailText renders the full detail content for inspector mode (without input forms)
// width specifies the maximum width for text wrapping (0 = no wrapping)
func (rv *RunnerView) renderDetailText(script ScriptFile, width int) string {
	fieldStyle := lipgloss.NewStyle().Bold(true).Foreground(secondaryColor)
	valueStyleDetail := lipgloss.NewStyle().Foreground(accentColor)

	renderField := func(label, value string) string {
		return fieldStyle.Render(fmt.Sprintf("%-15s", label+":")) + " " + valueStyleDetail.Render(value) + "\n"
	}

	var details strings.Builder

	// Title
	details.WriteString(fieldStyle.Render("Script/Transaction Details") + "\n\n")

	// Basic info
	details.WriteString(renderField("Type", string(script.Type)))
	details.WriteString(renderField("Name", script.Name))
	details.WriteString(renderField("Network", script.Network))

	if script.Signers > 0 {
		details.WriteString(renderField("Signers", fmt.Sprintf("%d", script.Signers)))
	}

	if len(script.Parameters) > 0 {
		details.WriteString(renderField("Parameters", fmt.Sprintf("%d", len(script.Parameters))))
	}

	details.WriteString("\n")

	// Show execution result if present
	if rv.isExecuting {
		details.WriteString(fieldStyle.Render("Status:") + " " + rv.spinner.View() + " Executing...\n\n")
	} else if rv.executionError != nil {
		errorStyle := lipgloss.NewStyle().Foreground(errorColor).Bold(true)
		details.WriteString(fieldStyle.Render("Error:") + "\n")
		details.WriteString(errorStyle.Render(rv.executionError.Error()) + "\n\n")
	} else if rv.executionResult != "" {
		successStyle := lipgloss.NewStyle().Foreground(successColor).Bold(true)
		details.WriteString(fieldStyle.Render("Result:") + "\n")
		details.WriteString(successStyle.Render(rv.executionResult) + "\n\n")
	}

	// Parameters (if any)
	if len(script.Parameters) > 0 {
		details.WriteString(fieldStyle.Render("Parameters:") + "\n")
		for _, param := range script.Parameters {
			details.WriteString(fmt.Sprintf("  • %s: %s\n",
				valueStyleDetail.Render(param.Name),
				lipgloss.NewStyle().Foreground(mutedColor).Render(param.Type)))
		}
		details.WriteString("\n")
	}

	// Code - use pre-highlighted code (just like transactions_view)
	details.WriteString(fieldStyle.Render("Code:") + "\n")
	// Show syntax-highlighted code if available, otherwise raw code
	codeToShow := script.HighlightedCode
	if codeToShow == "" {
		codeToShow = script.Code
	}
	// Don't wrap in valueStyleDetail since highlighted code already has colors
	details.WriteString(codeToShow + "\n")

	return details.String()
}

// renderDetailForViewport renders the detail view with form and code for viewport (full detail mode with input forms)
func (rv *RunnerView) renderDetailForViewport(script ScriptFile) string {
	// Don't apply width constraint to avoid truncating styled content
	detailStyle := lipgloss.NewStyle().
		Padding(1, 2)

	var content strings.Builder

	// Show save dialog if active
	if rv.savingConfig {
		saveTitle := lipgloss.NewStyle().Bold(true).Foreground(secondaryColor).Render("Save Configuration")
		content.WriteString(saveTitle + "\n\n")

		content.WriteString("Enter filename (without .json extension):\n")
		content.WriteString(rv.saveInput.View() + "\n\n")

		if rv.saveError != "" {
			errorStyle := lipgloss.NewStyle().Foreground(errorColor)
			content.WriteString(errorStyle.Render("Error: "+rv.saveError) + "\n\n")
		}

		hintStyle := lipgloss.NewStyle().Foreground(mutedColor).Italic(true)
		content.WriteString(hintStyle.Render("Press Enter to save, Esc to cancel") + "\n")

		return detailStyle.Render(content.String())
	}

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
			content.WriteString(hintStyle.Render("Press Esc to stop editing, Enter/Tab for next field") + "\n\n")
		} else {
			content.WriteString(hintStyle.Render("Press Enter/Tab to edit, j/k to navigate, r to run, s to save") + "\n\n")
		}
	} else {
		// No parameters - show hint to run directly
		hintStyle := lipgloss.NewStyle().Foreground(mutedColor).Italic(true)
		content.WriteString(hintStyle.Render("No parameters required. Press r to run, s to save") + "\n\n")
	}

	// Code section - use pre-highlighted code (just like transactions_view)
	codeTitle := lipgloss.NewStyle().Bold(true).Foreground(secondaryColor).Render("Code:")
	content.WriteString(codeTitle + "\n")

	// Show syntax-highlighted code if available, otherwise raw code
	codeToShow := script.HighlightedCode
	if codeToShow == "" {
		codeToShow = script.Code
	}
	// Don't wrap in valueStyleDetail since highlighted code already has colors
	content.WriteString(codeToShow + "\n")

	return detailStyle.Render(content.String())
}

// Stop is a no-op for the runner view
func (rv *RunnerView) Stop() {
	// No cleanup needed
}
