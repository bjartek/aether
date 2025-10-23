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
	"github.com/bjartek/overflow/v2"
	"github.com/bjartek/underflow"
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

// RunnerViewV2 is the splitview-based implementation for script/transaction runner
type RunnerViewV2 struct {
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
	logger           zerolog.Logger
}

// NewRunnerViewV2WithConfig creates a new v2 runner view
func NewRunnerViewV2WithConfig(cfg *config.Config) *RunnerViewV2 {
	// Fallback to defaults when cfg is nil
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	columns := []splitview.ColumnConfig{
		{Name: "Type", Width: 12},
		{Name: "Name", Width: 40},
		{Name: "Network", Width: 10},
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
	sv := splitview.NewSplitView(
		columns,
		splitview.WithTableStyles(s),
		splitview.WithTableSplitPercent(float64(cfg.UI.Layout.Runner.TableWidthPercent)/100.0),
	)

	// Create save input
	saveInput := textinput.New()
	saveInput.Placeholder = "config-name"
	saveInput.CharLimit = 50
	saveInput.Width = 40

	// Create file logger for debugging (only if debug mode is enabled)
	var logger zerolog.Logger
	if cfg.UI.Debug {
		logFile, err := os.OpenFile("runner_v2_debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			logFile = os.Stderr // Fallback to stderr if file can't be opened
		}
		
		logger = zerolog.New(logFile).
			With().
			Timestamp().
			Str("component", "runner_v2").
			Logger().
			Level(zerolog.DebugLevel)
	} else {
		// Disabled logger that doesn't write anywhere
		logger = zerolog.Nop()
	}

	rv := &RunnerViewV2{
		sv:               sv,
		keys:             DefaultRunnerKeyMap(),
		scripts:          make([]ScriptFile, 0),
		inputFields:      make([]InputField, 0),
		activeFieldIndex: 0,
		editingField:     false,
		saveInput:        saveInput,
		logger:           logger,
	}

	// Scan for scripts and transactions
	rv.scanFiles()

	// Populate splitview with scripts
	for _, script := range rv.scripts {
		rv.addScriptRow(script)
	}

	return rv
}

// Init implements tea.Model
func (rv *RunnerViewV2) Init() tea.Cmd {
	return rv.sv.Init()
}

// Update implements tea.Model
func (rv *RunnerViewV2) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
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
					rv.saveError = rv.saveConfig(rv.saveInput.Value())
					if rv.saveError == "" {
						rv.savingConfig = false
						rv.saveInput.SetValue("")
					}
				}
				return rv, nil
				
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
				return rv, cmd
			}
		}
		
		// Handle form editing
		if rv.editingField && len(rv.inputFields) > 0 {
			switch {
			case msg.Type == tea.KeyEsc:
				// Exit editing mode
				rv.editingField = false
				rv.inputFields[rv.activeFieldIndex].Input.Blur()
				// Refresh detail to update UI
				selectedIdx := rv.sv.GetCursor()
				if selectedIdx >= 0 && selectedIdx < len(rv.scripts) {
					rv.refreshDetailContent(selectedIdx, rv.scripts[selectedIdx])
				}
				return rv, nil
				
			case key.Matches(msg, rv.keys.Enter):
				// Move to next field or exit editing if last field
				rv.inputFields[rv.activeFieldIndex].Input.Blur()
				if rv.activeFieldIndex < len(rv.inputFields)-1 {
					rv.activeFieldIndex++
					rv.inputFields[rv.activeFieldIndex].Input.Focus()
				} else {
					rv.editingField = false
				}
				return rv, nil
				
			default:
				// Update active input field
				var cmd tea.Cmd
				rv.inputFields[rv.activeFieldIndex].Input, cmd = rv.inputFields[rv.activeFieldIndex].Input.Update(msg)
				return rv, cmd
			}
		}
		
		// Handle navigation between fields (when not editing)
		if !rv.editingField && len(rv.inputFields) > 0 && rv.sv.IsFullscreen() {
			switch {
			case key.Matches(msg, rv.keys.NextField):
				rv.activeFieldIndex = (rv.activeFieldIndex + 1) % len(rv.inputFields)
				selectedIdx := rv.sv.GetCursor()
				if selectedIdx >= 0 && selectedIdx < len(rv.scripts) {
					rv.refreshDetailContent(selectedIdx, rv.scripts[selectedIdx])
				}
				return rv, nil
				
			case key.Matches(msg, rv.keys.PrevField):
				rv.activeFieldIndex = (rv.activeFieldIndex - 1 + len(rv.inputFields)) % len(rv.inputFields)
				selectedIdx := rv.sv.GetCursor()
				if selectedIdx >= 0 && selectedIdx < len(rv.scripts) {
					rv.refreshDetailContent(selectedIdx, rv.scripts[selectedIdx])
				}
				return rv, nil
				
			case key.Matches(msg, rv.keys.Enter):
				// Enter editing mode for current field
				rv.editingField = true
				rv.inputFields[rv.activeFieldIndex].Input.Focus()
				selectedIdx := rv.sv.GetCursor()
				if selectedIdx >= 0 && selectedIdx < len(rv.scripts) {
					rv.refreshDetailContent(selectedIdx, rv.scripts[selectedIdx])
				}
				return rv, nil
			}
		}
		
		// Handle refresh
		if key.Matches(msg, rv.keys.Refresh) {
			rv.scanFiles()
			
			// Rebuild splitview rows
			rows := make([]splitview.RowData, 0)
			for _, script := range rv.scripts {
				typeStr := "Script"
				if script.Type == TypeTransaction {
					typeStr = "Transaction"
				}
				row := table.Row{typeStr, script.Name, script.Network}
				content := rv.buildScriptDetail(script)
				codeToShow := script.HighlightedCode
				if codeToShow == "" {
					codeToShow = script.Code
				}
				rows = append(rows, splitview.NewRowData(row).WithContent(content).WithCode(codeToShow))
			}
			rv.sv.SetRows(rows)
			return rv, nil
		}
		
		// Handle save config
		if key.Matches(msg, rv.keys.Save) && rv.sv.IsFullscreen() {
			rv.savingConfig = true
			rv.saveInput.Focus()
			rv.saveError = ""
			return rv, nil
		}
		
		// Handle enter/space to toggle fullscreen and build forms
		if key.Matches(msg, rv.keys.Enter) || key.Matches(msg, rv.keys.ToggleFullDetail) {
			// Build input fields when entering fullscreen
			if !rv.sv.IsFullscreen() {
				selectedIdx := rv.sv.GetCursor()
				if selectedIdx >= 0 && selectedIdx < len(rv.scripts) {
					script := rv.scripts[selectedIdx]
					rv.buildInputFields(script)
					rv.tryLoadConfigFromJSON(script)
				}
			}
		}
		
		// Handle 'r' key to run selected script/transaction
		if key.Matches(msg, rv.keys.Run) {
			rv.logger.Debug().
				Bool("hasOverflow", rv.overflow != nil).
				Int("selectedIdx", rv.sv.GetCursor()).
				Int("scriptsCount", len(rv.scripts)).
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
	return rv, tea.Batch(cmds...)
}

// View implements tea.Model
func (rv *RunnerViewV2) View() string {
	view := rv.sv.View()
	rv.logger.Debug().
		Int("viewLength", len(view)).
		Bool("hasExecutionResult", rv.executionResult != "").
		Bool("hasExecutionError", rv.executionError != nil).
		Int("scriptsCount", len(rv.scripts)).
		Msg("View rendered")
	return view
}

// Name implements TabbedModel interface
func (rv *RunnerViewV2) Name() string {
	return "Runner"
}

// KeyMap implements TabbedModel interface - combines runner and splitview keys
func (rv *RunnerViewV2) KeyMap() help.KeyMap {
	return NewCombinedKeyMap(rv.keys, rv.sv.KeyMap())
}

// FooterView implements TabbedModel interface - results shown inline in detail
func (rv *RunnerViewV2) FooterView() string {
	return ""
}

// IsCapturingInput implements TabbedModel interface
func (rv *RunnerViewV2) IsCapturingInput() bool {
	return rv.editingField || rv.savingConfig
}

// SetOverflow sets the overflow state for script execution
func (rv *RunnerViewV2) SetOverflow(o *overflow.OverflowState) {
	rv.overflow = o
}

// SetAccountRegistry sets the account registry for signer name resolution
func (rv *RunnerViewV2) SetAccountRegistry(registry *aether.AccountRegistry) {
	rv.accountRegistry = registry
	// Update available signers list from registry
	if registry != nil {
		rv.availableSigners = registry.GetAllNames()
	}
}

// formatValue recursively formats a value with proper indentation
func (rv *RunnerViewV2) formatValue(value interface{}, indent string) string {
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
func (rv *RunnerViewV2) formatScriptResult(result *overflow.OverflowScriptResult) string {
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
func (rv *RunnerViewV2) formatTransactionResult(result *overflow.OverflowResult) string {
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
func (rv *RunnerViewV2) detectNetwork(filename string) string {
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
func (rv *RunnerViewV2) removeNetworkSuffix(filename string) string {
	for _, suffix := range []string{".emulator", ".testnet", ".mainnet"} {
		if strings.HasSuffix(filename, suffix) {
			return strings.TrimSuffix(filename, suffix)
		}
	}
	return filename
}

// refreshDetailContent updates the detail content for the current row
func (rv *RunnerViewV2) refreshDetailContent(idx int, script ScriptFile) {
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
func (rv *RunnerViewV2) buildInputFields(script ScriptFile) {
	rv.inputFields = make([]InputField, 0)

	// Add signer fields for transactions
	if script.Type == TypeTransaction && script.Signers > 0 {
		for i := 0; i < script.Signers; i++ {
			input := textinput.New()
			input.Placeholder = "signer name"
			input.Width = 30
			
			label := "Signer"
			if i > 0 {
				label = fmt.Sprintf("PayloadSigner %d", i)
			}
			
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
func (rv *RunnerViewV2) saveConfig(configName string) string {
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
func (rv *RunnerViewV2) tryLoadConfigFromJSON(script ScriptFile) {
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
func (rv *RunnerViewV2) executeScript(script ScriptFile) tea.Cmd {
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
		
		scriptName := script.Name
		
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
func (rv *RunnerViewV2) scanFiles() {
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
func (rv *RunnerViewV2) findCdcFile(name string, scriptType ScriptType) string {
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
func (rv *RunnerViewV2) scanDirectory(dir string, scriptType ScriptType, files *[]ScriptFile) {
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
			config, err := flow.LoadTransactionConfig(path)
			if err != nil {
				// Skip malformed JSON files
				return nil
			}

			// Find the referenced .cdc file
			cdcPath := rv.findCdcFile(config.Name, scriptType)
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
func (rv *RunnerViewV2) addScriptRow(script ScriptFile) {
	typeStr := "Script"
	if script.Type == TypeTransaction {
		typeStr = "Transaction"
	}

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

	// Add to splitview with syntax-highlighted code
	rv.sv.AddRow(splitview.NewRowData(row).WithContent(content).WithCode(codeToShow))
}

// buildScriptDetail builds the detail content for a script
func (rv *RunnerViewV2) buildScriptDetail(script ScriptFile) string {
	fieldStyle := lipgloss.NewStyle().Bold(true).Foreground(secondaryColor)
	valueStyle := lipgloss.NewStyle().Foreground(accentColor)

	renderField := func(label, value string) string {
		return fieldStyle.Render(fmt.Sprintf("%-15s", label+":")) + " " + valueStyle.Render(value) + "\n"
	}

	var details strings.Builder
	details.WriteString(fieldStyle.Render("Script Details") + "\n\n")

	typeStr := "Script"
	if script.Type == TypeTransaction {
		typeStr = "Transaction"
	}

	details.WriteString(renderField("Type", typeStr))
	details.WriteString(renderField("Name", script.Name))
	details.WriteString(renderField("Path", script.Path))
	details.WriteString(renderField("Network", script.Network))
	details.WriteString("\n")

	if len(script.Parameters) > 0 {
		details.WriteString(fieldStyle.Render(fmt.Sprintf("Parameters (%d):", len(script.Parameters))) + "\n")
		for _, param := range script.Parameters {
			details.WriteString(fmt.Sprintf("  • %s (%s)\n",
				valueStyle.Render(param.Name),
				valueStyle.Render(param.Type)))
		}
		details.WriteString("\n")
	}

	if script.Signers > 0 {
		details.WriteString(renderField("Signers", fmt.Sprintf("%d", script.Signers)))
		details.WriteString("\n")
	}

	// Show input forms if in fullscreen mode and have fields
	if rv.sv.IsFullscreen() && len(rv.inputFields) > 0 {
		details.WriteString("\n" + fieldStyle.Render("Input Fields:") + "\n\n")
		
		for i, field := range rv.inputFields {
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
		
		// Show save dialog if active
		if rv.savingConfig {
			details.WriteString("\n" + fieldStyle.Render("Save Config As:") + "\n")
			details.WriteString(rv.saveInput.View() + "\n")
			if rv.saveError != "" {
				details.WriteString(lipgloss.NewStyle().Foreground(errorColor).Render(rv.saveError) + "\n")
			}
		}
	}
	
	// Show execution results inline
	if rv.executionError != nil {
		details.WriteString(lipgloss.NewStyle().
			Foreground(errorColor).
			Render(fmt.Sprintf("\n❌ Error: %s\n", rv.executionError.Error())))
	} else if rv.executionResult != "" {
		details.WriteString(lipgloss.NewStyle().
			Foreground(successColor).
			Render(fmt.Sprintf("\n✓ %s\n", rv.executionResult)))
	} else if !rv.sv.IsFullscreen() {
		details.WriteString("\n" + fieldStyle.Render("Press 'enter' for fullscreen, 'r' to run") + "\n")
	}

	return details.String()
}

// parseScriptFile extracts parameters and signers from cadence code using AST parser
func (rv *RunnerViewV2) parseScriptFile(script *ScriptFile) {
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
func (rv *RunnerViewV2) parseParameterList(paramList *ast.ParameterList) []Parameter {
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
