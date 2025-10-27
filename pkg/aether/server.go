package aether

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bjartek/aether/pkg/chroma"
	"github.com/bjartek/aether/pkg/config"
	"github.com/bjartek/aether/pkg/flow"
	"github.com/bjartek/overflow/v2"
	"github.com/bjartek/underflow"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/enescakir/emoji"
	"github.com/onflow/flow-evm-gateway/models"
	"github.com/onflow/flow-go/fvm/evm/events"
	"github.com/rs/zerolog"
)

type Aether struct {
	Logger          *zerolog.Logger
	FclCdc          []byte
	Overflow        *overflow.OverflowState
	AccountRegistry *AccountRegistry
	Network         string // "testnet", "mainnet", or "emulator"
	Config          *config.Config
	
	// State for deferred init transaction execution (interactive mode)
	pendingInitTx *pendingInitContext
}

type pendingInitContext struct {
	teaProgram *tea.Program
	o          *overflow.OverflowState
	oR         *overflow.OverflowState
	basePath   string
}

// BlockTransactionMsg is sent when a transaction is processed
type BlockTransactionMsg struct {
	TransactionData TransactionData
}

// BlockEventMsg is sent when an event is processed
type BlockEventMsg struct {
	EventData EventData
}

// EventData holds event information for display
type EventData struct {
	Name             string
	BlockHeight      uint64
	BlockID          string
	TransactionID    string
	TransactionIndex int
	EventIndex       int
	Fields           map[string]interface{}
	Timestamp        time.Time
}

type ArgumentData struct {
	Name  string
	Value interface{} // Keep as interface{} for proper formatting
}

// EVMTransactionData wraps all data returned from decoding an EVM transaction event
type EVMTransactionData struct {
	Transaction models.Transaction
	Receipt     *models.Receipt
	Payload     *events.TransactionEventPayload
}

// TransactionType represents the type of transaction
type TransactionType string

const (
	TransactionTypeFlow  TransactionType = "flow"  // Only Flow/Cadence events
	TransactionTypeEVM   TransactionType = "evm"   // Only EVM events
	TransactionTypeMixed TransactionType = "mixed" // Both Flow and EVM events
)

// TransactionData holds transaction information for display
type TransactionData struct {
	ID                string
	BlockID           string
	BlockHeight       uint64
	Authorizers       []string // Can have multiple authorizers
	Status            string
	Proposer          string
	Payer             string
	GasLimit          uint64
	Script            string // Raw script code
	HighlightedScript string // Syntax-highlighted script with ANSI colors
	Arguments         []ArgumentData
	Events            []overflow.OverflowEvent
	EVMTransactions   []EVMTransactionData // Decoded EVM transactions
	Type              TransactionType      // Transaction type (flow/evm/mixed)
	Error             string
	Timestamp         time.Time
	Index             int
	SourceFile        string // Filename that executed this transaction (from runner or init)
	IsInit            bool   // True if this was an init transaction
}

// TransactionMsg is sent when a new transaction is received
type TransactionMsg struct {
	Transaction TransactionData
}

// OverflowReadyMsg is sent when overflow is initialized and ready
type OverflowReadyMsg struct {
	Overflow        *overflow.OverflowState
	AccountRegistry *AccountRegistry
}

// InitTransactionMsg is sent when an init transaction executes
type InitTransactionMsg struct {
	Filename      string
	Success       bool
	Error         string
	TransactionID string // Transaction ID if available
}

// TransactionSourceMsg tracks which file executed a transaction
type TransactionSourceMsg struct {
	TransactionID string
	SourceFile    string
	IsInit        bool
}

// BlockHeightMsg is sent periodically with the latest block height
type BlockHeightMsg struct {
	Height uint64
}

// InitFolderSelectionMsg prompts the user to select an init transactions folder
type InitFolderSelectionMsg struct {
	Folders     []string // Available folders to choose from
	DefaultPath string   // The base aether path
}

// InitFolderSelectedMsg is sent when user selects a folder
type InitFolderSelectedMsg struct {
	SelectedFolder string // The folder name selected by user (empty string = root)
}

func (a *Aether) Start(teaProgram *tea.Program) error {
	ctx := context.Background()

	// Define the two possible paths
	path1 := "aether"
	path2 := filepath.Join("cadence", "aether")

	// Check which path exists
	var validPath string
	basePath := ""
	if _, err := os.Stat(path1); err == nil {
		validPath = path1
	} else if _, err := os.Stat(path2); err == nil {
		validPath = path2
		basePath = "cadence"
	} else {
		return fmt.Errorf("neither %q nor %q exists", path1, path2)
	}

	// Configure underflow options for human-friendly output
	underflowOptions := underflow.Options{
		ByteArrayAsHex:             true,
		ShowUnixTimestampsAsString: true,
		TimestampFormat:            "2006-01-02 15:04:05 UTC",
	}

	// Initialize overflow based on network mode
	var o *overflow.OverflowState
	if a.Network == "emulator" {
		// Local emulator mode
		o = overflow.Overflow(
			overflow.WithExistingEmulator(),
			overflow.WithLogNone(),
			overflow.WithReturnErrors(),
			overflow.WithTransactionFolderName("aether"),
			overflow.WithBasePath(basePath),
			overflow.WithUnderflowOptions(underflowOptions),
			overflow.WithFlowForNewUsers(a.Config.Flow.NewUserBalance))
	} else {
		// Network mode (testnet or mainnet)
		a.Logger.Info().Str("network", a.Network).Msg("Initializing overflow for network")
		o = overflow.Overflow(
			overflow.WithNetwork(a.Network),
			overflow.WithLogNone(),
			overflow.WithReturnErrors(),
			overflow.WithTransactionFolderName("aether"),
			overflow.WithBasePath(basePath),
			overflow.WithUnderflowOptions(underflowOptions))
	}

	// Only create accounts in local mode
	if a.Network == "emulator" {
		a.Logger.Info().Str("network", o.Network.Name).Msg("emulator")
		_, err := o.CreateAccountsE(ctx)
		if err != nil {
			return err
		}
	}
	a.Overflow = o

	// Initialize account registry after accounts are created
	a.AccountRegistry = NewAccountRegistry(o)
	dump := a.AccountRegistry.DebugDump()
	a.Logger.Info().
		Int("accounts", len(a.AccountRegistry.addressToName)).
		Interface("registry", dump).
		Msg("Initialized account registry")

	// Create second overflow instance for runner view with same underflow options
	var oR *overflow.OverflowState
	if a.Network == "emulator" {
		oR = overflow.Overflow(
			overflow.WithExistingEmulator(),
			overflow.WithLogNone(),
			overflow.WithReturnErrors(),
			overflow.WithBasePath(basePath),
			overflow.WithUnderflowOptions(underflowOptions))
	} else {
		oR = overflow.Overflow(
			overflow.WithNetwork(a.Network),
			overflow.WithLogNone(),
			overflow.WithReturnErrors(),
			overflow.WithBasePath(basePath),
			overflow.WithUnderflowOptions(underflowOptions))
	}

	// Send overflow ready message to UI
	if teaProgram != nil {
		teaProgram.Send(OverflowReadyMsg{
			Overflow:        oR,
			AccountRegistry: a.AccountRegistry,
		})
	}

	overflowChannel := make(chan flow.BlockResult)

	// Determine starting block height and polling interval based on network mode
	var startHeight uint64
	pollInterval := a.Config.Indexer.PollingInterval

	if a.Network == "emulator" {
		// Local emulator mode - start from block 1
		startHeight = 1
	} else {
		// Network mode - start from latest block
		latestBlock, err := o.GetLatestBlock(ctx)
		if err != nil {
			a.Logger.Error().Err(err).Str("network", a.Network).Msg("Failed to get latest block")
			return err
		}
		startHeight = latestBlock.Height

		a.Logger.Info().
			Str("network", a.Network).
			Uint64("startHeight", startHeight).
			Dur("pollInterval", pollInterval).
			Msg("Starting to stream from latest block")
	}

	go func() {
		a.Logger.Info().Msg("Started streaming")

		err := flow.StreamTransactions(ctx, o, pollInterval, startHeight, a.Logger, overflowChannel)
		if err != nil {
			if strings.Contains(err.Error(), "context canceled") {
				a.Logger.Info().Msg("Streaming stopped due to context cancellation")
			} else {
				a.Logger.Warn().Err(err).Msg("Streaming encountered an error")
			}
		}
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				a.Logger.Info().Msg("Block receiver goroutine stopping due to context cancellation")
				return
			case br, ok := <-overflowChannel:
				if !ok {
					a.Logger.Info().Msg("Channel has been closed. Crawler stopped completely.")
					return
				}

				l := br.Logger

				if br.Error != nil {
					l.Warn().Err(br.Error).Msg("Failed fetching block")
					continue
				}

				// Send transactions directly to UI if there are any
				if len(br.Transactions) > 0 && teaProgram != nil {
					for _, ot := range br.Transactions {

						// Extract all authorizers
						authorizers := ot.Authorizers
						if len(authorizers) == 0 {
							authorizers = []string{"N/A"}
						}

						// Extract proposer and payer
						proposer := "N/A"
						if ot.ProposalKey.Address.String() != "" {
							proposer = fmt.Sprintf("0x%s", ot.ProposalKey.Address.Hex())
						}

						payer := "N/A"
						if ot.Payer != "" {
							payer = ot.Payer
						}
						// Determine status
						status := "Unknown"
						if ot.Error != nil {
							status = "Failed"
						} else {
							status = ot.Status
						}

						// Store full script - user can scroll if needed
						script := string(ot.Script)
						highlightedScript := chroma.HighlightCadence(script)

						// Format arguments as structured data
						args := make([]ArgumentData, 0, len(ot.Arguments))
						for i, arg := range ot.Arguments {
							// Use the key field as the argument name, fallback to index if not available
							name := arg.Key
							if name == "" {
								name = fmt.Sprintf("argument%d", i)
							}
							argData := ArgumentData{
								Name:  name,
								Value: arg.Value, // Keep as interface{} for proper formatting
							}
							args = append(args, argData)
						}

						// Create error message
						errMsg := ""
						if ot.Error != nil {
							errMsg = ot.Error.Error()
						}

						// Store events directly
						events := ot.Events

						// Detect and decode EVM transactions from events
						evmTransactions := make([]EVMTransactionData, 0)
						hasEVMEvents := false
						hasNonEVMEvents := false

						for _, event := range events {
							// Check if this is an EVM.TransactionExecuted event
							if strings.Contains(event.Name, "EVM.TransactionExecuted") {
								hasEVMEvents = true
								tx, receipt, payload, err := models.DecodeTransactionEvent(event.RawEvent)
								if err != nil {
									// Skip events that fail to decode
									continue
								}
								evmTx := EVMTransactionData{
									Transaction: tx,
									Receipt:     receipt,
									Payload:     payload,
								}
								evmTransactions = append(evmTransactions, evmTx)
							} else {
								hasNonEVMEvents = true
							}
						}

						// Determine transaction type
						txType := TransactionTypeFlow // Default to flow
						if hasEVMEvents && !hasNonEVMEvents {
							txType = TransactionTypeEVM
						} else if hasEVMEvents && hasNonEVMEvents {
							txType = TransactionTypeMixed
						}

						txData := TransactionData{
							ID:                ot.Id,
							BlockID:           br.Block.ID.String(),
							BlockHeight:       br.Block.Height,
							Authorizers:       authorizers,
							Status:            status,
							Proposer:          proposer,
							Payer:             payer,
							GasLimit:          ot.GasLimit,
							Script:            script,
							HighlightedScript: highlightedScript,
							Arguments:         args,
							Events:            events,
							EVMTransactions:   evmTransactions,
							Type:              txType,
							Error:             errMsg,
							Timestamp:         time.Now(),
							Index:             ot.TransactionIndex,
						}

						teaProgram.Send(BlockTransactionMsg{
							TransactionData: txData,
						})

						// Send individual event messages
						for eventIndex, event := range events {
							eventData := EventData{
								Name:             event.Name,
								BlockHeight:      br.Block.Height,
								BlockID:          br.Block.ID.String(),
								TransactionID:    ot.Id,
								TransactionIndex: ot.TransactionIndex,
								EventIndex:       eventIndex,
								Fields:           event.Fields,
								Timestamp:        time.Now(),
							}
							teaProgram.Send(BlockEventMsg{
								EventData: eventData,
							})
						}
					}
				}

				// Send block height update to dashboard
				teaProgram.Send(BlockHeightMsg{
					Height: br.Block.Height,
				})

				// Log the block processing
				txCount := len(br.Transactions)

				if txCount > 0 {
					l.Info().
						Uint64("height", br.Block.Height).
						Int("txCount", txCount).
						Msg("Processed block")
				}
			}
		}
	}()

	// Only perform local setup in emulator mode
	if a.Network == "emulator" {
		a.Logger.Info().Msgf("%v Created accounts for emulator users in flow.json", emoji.Person)
		o.InitializeContracts(ctx)

		a.Logger.Info().Msgf("%v  Deployed contracts specified in emulator deployment block", emoji.Envelope)
		if err := flow.AddFclContract(o, a.FclCdc); err != nil {
			return err
		}

		accounts := o.GetEmulatorAccounts()
		a.Logger.Debug().Int("filtered_accounts", len(accounts)).Interface("accounts", accounts).Msg("Filtered emulator accounts")

		if len(accounts) > 0 {
			a.Logger.Info().Int("count", len(accounts)).Interface("accounts", accounts).Msgf("%v Adding accounts to FCL", emoji.Person)
			if err := flow.AddFclAccounts(o, accounts); err != nil {
				return err
			}
			a.Logger.Info().Msgf("%v Successfully added %d accounts to FCL", emoji.Person, len(accounts))
		} else {
			a.Logger.Warn().Msg("No accounts to add to FCL")
		}

		// Determine init transactions path - either from config or interactive selection
		initTxPath := validPath
		
		if a.Config.Flow.InitTransactionsInteractive {
			// Interactive mode: scan for available folders and prompt user
			folders, err := scanInitFolders(validPath)
			if err != nil {
				a.Logger.Error().Err(err).Msg("Failed to scan for init folders")
				return err
			}
			
			a.Logger.Info().
				Int("folderCount", len(folders)).
				Strs("folders", folders).
				Msg("Found init transaction folders, prompting user for selection")
			
			// Send folder selection message to UI
			teaProgram.Send(InitFolderSelectionMsg{
				Folders:     folders,
				DefaultPath: validPath,
			})
			
			// Store context for deferred init transaction execution
			a.pendingInitTx = &pendingInitContext{
				teaProgram: teaProgram,
				o:          o,
				oR:         oR,
				basePath:   validPath,
			}
			a.Logger.Info().Msg("Init transaction context stored, waiting for user folder selection...")
			
		} else {
			// Non-interactive mode: run init transactions immediately
			if a.Config.Flow.InitTransactionsFolder != "" {
				initTxPath = filepath.Join(validPath, a.Config.Flow.InitTransactionsFolder)
				a.Logger.Info().Str("folder", a.Config.Flow.InitTransactionsFolder).Msg("Using configured init transactions folder")
			}

			// Use same overflow state for both .cdc and .json files
			if err := flow.RunInitTransactions(o, oR, initTxPath, a.Logger, func(filename string, success bool, errorMsg string, txID string) {
				// Send progress update to UI
				teaProgram.Send(InitTransactionMsg{
					Filename:      filename,
					Success:       success,
					Error:         errorMsg,
					TransactionID: txID,
				})
				// Send transaction source tracking
				if success && txID != "" {
					teaProgram.Send(TransactionSourceMsg{
						TransactionID: txID,
						SourceFile:    filename,
						IsInit:        true,
					})
				}
			}); err != nil {
				return err
			}
		}
	} else {
		a.Logger.Info().Str("network", a.Network).Msg("Following network - skipping local setup steps")
	}
	return nil
}

func (a *Aether) Stop() {
}

// scanInitFolders scans the base aether directory for subdirectories
// Returns a list of folder names (root folder is represented as "." or empty string)
func scanInitFolders(basePath string) ([]string, error) {
	var folders []string
	
	// Add root folder as an option (empty string means root)
	folders = append(folders, "")
	
	// Read directory entries
	entries, err := os.ReadDir(basePath)
	if err != nil {
		return nil, err
	}
	
	// Collect subdirectories
	for _, entry := range entries {
		if entry.IsDir() {
			folders = append(folders, entry.Name())
		}
	}
	
	return folders, nil
}

// RunInitTransactionsWithFolder runs init transactions from the selected folder
// This is called by the UI after user selects a folder
func (a *Aether) RunInitTransactionsWithFolder(selectedFolder string) error {
	if a.pendingInitTx == nil {
		return fmt.Errorf("no pending init transaction context")
	}
	
	ctx := a.pendingInitTx
	teaProgram := ctx.teaProgram
	o := ctx.o
	oR := ctx.oR
	basePath := ctx.basePath
	initTxPath := basePath
	if selectedFolder != "" {
		initTxPath = filepath.Join(basePath, selectedFolder)
	}
	
	a.Logger.Info().
		Str("folder", selectedFolder).
		Str("path", initTxPath).
		Msg("Running init transactions from selected folder")
	
	// Run init transactions
	if err := flow.RunInitTransactions(o, oR, initTxPath, a.Logger, func(filename string, success bool, errorMsg string, txID string) {
		// Send progress update to UI
		teaProgram.Send(InitTransactionMsg{
			Filename:      filename,
			Success:       success,
			Error:         errorMsg,
			TransactionID: txID,
		})
		// Send transaction source tracking
		if success && txID != "" {
			teaProgram.Send(TransactionSourceMsg{
				TransactionID: txID,
				SourceFile:    filename,
				IsInit:        true,
			})
		}
	}); err != nil {
		a.Logger.Error().Err(err).Msg("Failed to run init transactions")
		return err
	}
	
	a.Logger.Info().Msg("Init transactions completed")
	return nil
}
