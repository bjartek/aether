# Model Conversion to SplitView Abstraction

## Overview
Convert `pkg/ui/model.go` and individual view implementations to use the `pkg/splitview` component, eliminating duplicate layout code and centralizing split-view behavior.

## Applicable Views
- [ ] TransactionsView
- [ ] EventsView  
- [ ] RunnerView
- [ ] DashboardView (no split view needed)
- [ ] LogsView (no split view needed)

## Configuration Cleanup

### Removed From View Structs (Still in Config)
- [ ] `tableWidthPercent` field - read from config, passed via `WithTableSplitPercent()`
- [ ] `detailWidthPercent` field - derived from table split, not stored
- [ ] Individual viewport width/height fields - handled by splitview
- [ ] `fullDetailMode` flag - handled by splitview
- [ ] Dual viewport management (`detailViewport`, `splitDetailViewport`) - single viewport in splitview

### Config Options Flow
- [ ] `tableWidthPercent` - read from config → passed to `WithTableSplitPercent()`
- [ ] `maxTxs`/`maxEvents` - stored in view for data management
- [ ] `showEventFields` - stored in view for toggle behavior
- [ ] `showRawAddresses` - stored in view for toggle behavior
- [ ] Theme/color settings - applied via `WithTableStyles()`

## Footer Integration

### Filter Bar Strategy
- [ ] Move filter textinput to model-level footer
- [ ] Footer shows based on filter mode flag from active view
- [ ] Footer renders below all content, above help bar
- [ ] Adjust content height calculation to account for footer when visible

### Footer State Management
- [ ] Model tracks which view is in filter mode
- [ ] Model passes filter messages to active view
- [ ] Views expose filter state via method (e.g., `IsFilterActive()`, `GetFilterInput()`)
- [ ] Views handle filter logic, model handles rendering

## Key Binding Consolidation

### View KeyMaps
- [ ] Each view exposes `KeyMap()` method returning `help.KeyMap`
- [ ] Remove duplicate navigation keys (j/k/up/down) - handled by splitview
- [ ] Keep view-specific bindings (e.g., toggle fields, save, run)

### Transactions View Keys
- [ ] Remove: `LineUp`, `LineDown`, `GotoTop`, `GotoEnd`, `ToggleFullDetail`
- [ ] Keep: `ToggleEventFields`, `ToggleRawAddresses`, `Filter`, `Save`

### Events View Keys  
- [ ] Remove: `LineUp`, `LineDown`, `GotoTop`, `GotoEnd`, `ToggleFullDetail`
- [ ] Keep: `ToggleRawAddresses`, `Filter`

### Runner View Keys
- [ ] Remove: `Up`, `Down`, `Enter` (if used for navigation)
- [ ] Keep: `Run`, `Save`, `Refresh`, form navigation keys

### Model-Level Key Handling
- [ ] Model delegates navigation to splitview when not in text input mode
- [ ] Model intercepts filter key (`/`) to activate filter footer
- [ ] Model delegates view-specific keys to active view

## View Conversion Pattern

### TransactionsView Structure
- [ ] Replace `table`, `detailViewport`, `splitDetailViewport` with single `*splitview.SplitViewModel`
- [ ] Implement `buildRowData()` to convert `TransactionData` to `splitview.RowData`
- [ ] Move detail rendering to content/code builders for `RowData`
- [ ] Remove manual viewport management
- [ ] Remove fullDetailMode flag - handled by splitview

### EventsView Structure  
- [ ] Same pattern as TransactionsView
- [ ] Convert `EventData` to `splitview.RowData`
- [ ] Remove duplicate layout code

### RunnerView Structure
- [ ] Wrap splitview, but handle form input separately
- [ ] Table shows scripts/transactions list
- [ ] Detail panel shows read-only info in split mode
- [ ] Full detail mode shows interactive forms
- [ ] Add `renderDetailForSplit()` vs `renderDetailForFull()` methods

## Data Flow Updates

### Adding Data
- [ ] `AddTransaction()` creates `RowData` and calls `splitview.AddRow()`
- [ ] `AddEvent()` creates `RowData` and calls `splitview.AddRow()`
- [ ] `AddScript()` creates `RowData` and calls `splitview.AddRow()`

### Filtering
- [ ] Filter logic remains in view
- [ ] Filtered results call `splitview.SetRows()` with filtered `RowData`
- [ ] Filter input managed by model footer

### Selection
- [ ] Current selection tracked by splitview table
- [ ] Views query selection via `splitview.table.Cursor()` when needed
- [ ] Save/export operations use cursor to identify selected item

## Update Method Refactoring

### Model.Update Changes  
- [ ] Calculate footer height when active
- [ ] Pass adjusted content height to views
- [ ] Handle filter activation/deactivation
- [ ] Forward filter input to active view
- [ ] Update `isInTextInputMode()` to check footer filter state

### View.Update Pattern
- [ ] Accept `tea.Msg`, `width`, `height` (unchanged signature)
- [ ] Create `tea.WindowSizeMsg` from width/height
- [ ] Forward all messages to `splitview.Update()`
- [ ] Handle view-specific messages (e.g., TransactionMsg, EventMsg)
- [ ] Return commands from splitview

## View Method Refactoring

### Simplified View.View()
- [ ] Remove layout logic
- [ ] Remove viewport content management  
- [ ] Return `splitview.View()` directly
- [ ] No manual joins or styling

### Detail Content Builders
- [ ] Extract `buildDetailContent(data)` returning markdown/text
- [ ] Extract `buildDetailCode(data)` returning highlighted code
- [ ] Use in `RowData.WithContent()` and `RowData.WithCode()`

## Help System Integration

### CombinedKeyMap Updates
- [ ] Include splitview keys via `splitview.KeyMap()`
- [ ] Merge with view-specific keys
- [ ] Remove navigation key duplication

### Help Rendering  
- [ ] Model continues to handle help toggle
- [ ] Help bar shows combined keys from splitview + view
- [ ] Footer shows separately from help (filter vs help)

## Incremental Migration Strategy

### Approach: Parallel Implementation
Run old and new implementations side-by-side, switch one view at a time.

**Flag Mechanism**: Add to config or use build tags
```go
// Option A: Config flag
cfg.UI.UseSplitViewV2.Transactions = true

// Option B: Simple const toggle in model.go
const (
    useTransactionsV2 = false  // flip to true when ready
    useEventsV2 = false
    useRunnerV2 = false
)
```

### Alternative: Minimal First Step
For even safer migration, start with just the layout wrapper:
- [ ] Keep all existing view logic
- [ ] Only replace table + viewport management with splitview
- [ ] Keep `filterInput`, `saveInput`, `showEventFields`, etc. in view
- [ ] Then extract detail builders in next step

### Step 1: Transactions Prep — Extract Detail Builders (No behavior change)
- [ ] Create `pkg/ui/tx_detail_builders.go` with two pure helpers:
  - [ ] `buildTransactionDetailContent(tx TransactionData, registry *aether.AccountRegistry, showEventFields bool, showRaw bool) string`
  - [ ] `buildTransactionDetailCode(tx TransactionData) string` (returns `tx.HighlightedScript` or fallback to `tx.Script`)
- [ ] Refactor `pkg/ui/transactions_view.go` `TransactionsView.renderTransactionDetailText(tx TransactionData)` to:
  - [ ] Call `buildTransactionDetailContent(...)`
  - [ ] Call `buildTransactionDetailCode(tx)`
  - [ ] Concatenate content + code and return
- [ ] Keep `TransactionsView.updateDetailViewport()` unchanged in call sites; it still calls `renderTransactionDetailText()`
- [ ] Do not touch table/viewport fields or width calculations in this step

### Step 2: New TransactionsView (Parallel)
- [ ] Create `transactions_view_v2.go` with splitview implementation
- [ ] Implement `NewTransactionsViewV2WithConfig()`
- [ ] Use extracted detail builders from Step 1
- [ ] Add `useV2Transactions` flag to model or config
- [ ] Model uses v1 or v2 based on flag

### Step 3: TransactionsView Migration
- [ ] Set flag to use v2 by default
- [ ] Remove v1 implementation and flag
- [ ] Rename `transactions_view_v2.go` → `transactions_view.go`

### Step 4: EventsView (Same Pattern)
- [ ] Extract detail builders from EventsView
- [ ] Create `events_view_v2.go`
- [ ] Add flag for events view
- [ ] Switch flag and remove v1 when ready

### Step 5: RunnerView (Same Pattern)
- [ ] More complex due to forms
- [ ] Extract detail builders (read-only vs forms)
- [ ] Create `runner_view_v2.go`
- [ ] Switch and remove v1

### Step 6: Footer Integration (After Views Stable)
- [ ] Add footer rendering to model
- [ ] Views still manage own filter inputs initially

### Step 7: Footer Migration
- [ ] Move filter input to model footer
- [ ] Update views to use model footer

### Step 8: Cleanup
- [ ] Remove all v1 code
- [ ] Remove feature flags
- [ ] Update documentation

## Migration Checklist

### Phase 1: TransactionsView
- [ ] Create `buildTransactionRowData()` method
- [ ] Initialize splitview with transaction columns
- [ ] Remove old table/viewport fields
- [ ] Update `AddTransaction()` to use splitview
- [ ] Implement filter via model footer
- [ ] Update keybindings
- [ ] Test split and fullscreen modes

### Phase 2: EventsView
- [ ] Same steps as TransactionsView
- [ ] Create `buildEventRowData()` method
- [ ] Update event columns

### Phase 3: RunnerView  
- [ ] Handle dual-mode detail rendering
- [ ] Keep form logic separate from splitview
- [ ] Coordinate between splitview selection and form state

### Phase 4: Model Integration
- [ ] Add footer rendering method
- [ ] Update height calculations
- [ ] Implement filter message routing
- [ ] Update `CombinedKeyMap` structure
- [ ] Remove obsolete view dimension code



## Implementation Notes

### Width Management
- Model calculates available width
- Passes to view Update()
- View passes to splitview via WindowSizeMsg
- No manual width tracking in views

### Height Management  
- Model: `contentHeight = total - header - footer - help`
- View: receives contentHeight parameter
- View: passes to splitview via WindowSizeMsg

### Message Routing
- Global keys: handled by model
- Navigation keys: forwarded to splitview
- Filter key: activates model footer
- View-specific keys: forwarded to view then splitview if not consumed

### State Consistency
- Splitview owns table cursor position
- View owns business logic (filtering, saving, data transforms)
- Model owns global UI state (tabs, help, footer)

## Success Criteria
- [ ] No duplicate viewport management code
- [ ] No manual table width/height calculations in views
- [ ] Single footer implementation used by all filterable views
- [ ] All keybindings represented as `key.Binding`
- [ ] Help system shows complete, non-duplicate key list
- [ ] Code reduction >50% in view layout logic
- [ ] Same or better UX than current implementation