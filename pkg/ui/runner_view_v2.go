package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bjartek/aether/pkg/chroma"
	"github.com/bjartek/aether/pkg/config"
	"github.com/bjartek/aether/pkg/splitview"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/parser"
	"github.com/onflow/cadence/sema"
)

// RunnerViewV2 is the splitview-based implementation for script/transaction runner
type RunnerViewV2 struct {
	sv      *splitview.SplitViewModel
	keys    RunnerKeyMap
	width   int
	height  int
	scripts []ScriptFile
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

	rv := &RunnerViewV2{
		sv:      sv,
		keys:    DefaultRunnerKeyMap(),
		scripts: make([]ScriptFile, 0),
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
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		rv.width = msg.Width
		rv.height = msg.Height
	}

	_, cmd := rv.sv.Update(msg)
	return rv, cmd
}

// View implements tea.Model
func (rv *RunnerViewV2) View() string {
	return rv.sv.View()
}

// Name implements TabbedModel interface
func (rv *RunnerViewV2) Name() string {
	return "Runner"
}

// KeyMap implements TabbedModel interface
func (rv *RunnerViewV2) KeyMap() help.KeyMap {
	return rv.sv.KeyMap()
}

// FooterView implements TabbedModel interface
func (rv *RunnerViewV2) FooterView() string {
	return ""
}

// IsCapturingInput implements TabbedModel interface
func (rv *RunnerViewV2) IsCapturingInput() bool {
	return false
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

// scanDirectory scans a directory for .cdc files
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

		// Only process .cdc files
		if !strings.HasSuffix(path, ".cdc") {
			return nil
		}

		// Read file content
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		codeStr := string(content)
		
		// Create script file entry with syntax highlighting
		name := strings.TrimSuffix(filepath.Base(path), ".cdc")
		script := ScriptFile{
			Name:            name,
			Path:            path,
			Type:            scriptType,
			Code:            codeStr,
			HighlightedCode: chroma.HighlightCadence(codeStr),
			Network:         "any", // Default to any network
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

	details.WriteString(fieldStyle.Render("Note:") + " Full execution support coming soon.\n")
	details.WriteString("For now, you can view scripts and their code.\n")

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
