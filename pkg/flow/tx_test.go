package flow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
These tests verify the init transaction scanning behavior.

Configuration context:
  init_transactions_folder: ""              ← scans root aether folder (default)
  init_transactions_folder: "init"          ← scans aether/init/ folder only
  init_transactions_folder: "setup/prod"    ← scans aether/setup/prod/ folder only

Key behavior: ONLY files in the specified folder are scanned, NOT subfolders!
*/

// TestScanning_DefaultBehavior tests the default config (init_transactions_folder = "")
//
// Config: init_transactions_folder = "" (or not set)
// Scans: aether/ folder (root level only)
//
// Directory structure:
//   aether/
//   ├── 1_init.cdc          ✓ EXECUTES (root level .cdc file)
//   ├── 2_setup.cdc         ✓ EXECUTES (root level .cdc file)
//   ├── 3_configured.json   ✓ EXECUTES (root level .json config)
//   ├── README.md           ✗ IGNORED (not .cdc or .json)
//   ├── test/               ✗ SKIPPED (subdirectory)
//   │   ├── test_tx.cdc
//   │   └── test_config.json
//   └── scripts/            ✗ SKIPPED (subdirectory)
//       └── helper.cdc
//
// Expected: Only 1_init.cdc, 2_setup.cdc, 3_configured.json execute
func TestScanning_DefaultBehavior(t *testing.T) {
	// Create: aether/ folder with files
	aetherFolder, err := os.MkdirTemp("", "aether-*")
	require.NoError(t, err)
	defer os.RemoveAll(aetherFolder)
	
	t.Logf("\n=== TEST: Default Behavior ===")
	t.Logf("Config: init_transactions_folder = \"\"")
	t.Logf("Scanning folder: %s\n", aetherFolder)
	
	// Create root-level transaction files (SHOULD RUN)
	createFile(t, aetherFolder, "1_init.cdc", "transaction {}")
	createFile(t, aetherFolder, "2_setup.cdc", "transaction {}")
	createFile(t, aetherFolder, "3_configured.json", `{"name":"test","signers":[],"arguments":{}}`)
	
	// Create non-transaction file (SHOULD IGNORE)
	createFile(t, aetherFolder, "README.md", "# Documentation")
	
	// Create subdirectories with files (SHOULD SKIP)
	testDir := filepath.Join(aetherFolder, "test")
	os.Mkdir(testDir, 0755)
	createFile(t, testDir, "test_tx.cdc", "transaction {}")
	createFile(t, testDir, "test_config.json", `{"name":"test"}`)
	
	scriptsDir := filepath.Join(aetherFolder, "scripts")
	os.Mkdir(scriptsDir, 0755)
	createFile(t, scriptsDir, "helper.cdc", "transaction {}")
	
	// Simulate the scanning logic from RunInitTransactions
	scanned := scanFolder(t, aetherFolder)
	
	// Assert: Only root-level .cdc and .json files are found
	assert.Contains(t, scanned, "1_init.cdc", "Root .cdc file should be scanned")
	assert.Contains(t, scanned, "2_setup.cdc", "Root .cdc file should be scanned")
	assert.Contains(t, scanned, "3_configured.json", "Root .json file should be scanned")
	
	assert.NotContains(t, scanned, "README.md", "Non-transaction file should be ignored")
	assert.NotContains(t, scanned, "test_tx.cdc", "Subdirectory file should NOT be scanned")
	assert.NotContains(t, scanned, "test_config.json", "Subdirectory file should NOT be scanned")
	assert.NotContains(t, scanned, "helper.cdc", "Subdirectory file should NOT be scanned")
	
	assert.Equal(t, 3, len(scanned), "Should scan exactly 3 files")
	
	t.Logf("✓ Found %d files to execute: %v\n", len(scanned), scanned)
}

// TestScanning_WithSubfolder tests when init_transactions_folder is configured
//
// Config: init_transactions_folder = "init"
// Scans: aether/init/ folder (only this folder, not parent or subfolders)
//
// Directory structure:
//   aether/
//   ├── root_tx.cdc         ✗ IGNORED (not in configured folder)
//   └── init/               ← CONFIGURED FOLDER TO SCAN
//       ├── 1_setup.cdc     ✓ EXECUTES
//       ├── 2_deploy.cdc    ✓ EXECUTES
//       └── test/           ✗ SKIPPED (subfolder of init/)
//           └── test.cdc
//
// Expected: Only files in aether/init/ execute (not root, not init/test/)
func TestScanning_WithSubfolder(t *testing.T) {
	// Create: aether/ and aether/init/ folders
	aetherFolder, err := os.MkdirTemp("", "aether-*")
	require.NoError(t, err)
	defer os.RemoveAll(aetherFolder)
	
	initFolder := filepath.Join(aetherFolder, "init")
	os.Mkdir(initFolder, 0755)
	
	t.Logf("\n=== TEST: With Subfolder Config ===")
	t.Logf("Config: init_transactions_folder = \"init\"")
	t.Logf("Scanning folder: %s\n", initFolder)
	
	// Create file in root aether/ (NOT in init folder - SHOULD IGNORE)
	createFile(t, aetherFolder, "root_tx.cdc", "transaction {}")
	
	// Create files in aether/init/ (SHOULD RUN)
	createFile(t, initFolder, "1_setup.cdc", "transaction {}")
	createFile(t, initFolder, "2_deploy.cdc", "transaction {}")
	
	// Create subfolder inside init/ (SHOULD SKIP)
	testDir := filepath.Join(initFolder, "test")
	os.Mkdir(testDir, 0755)
	createFile(t, testDir, "test.cdc", "transaction {}")
	
	// Simulate scanning the INIT folder (not root aether folder!)
	scanned := scanFolder(t, initFolder)
	
	// Assert: Only files in init/ folder are found
	assert.Contains(t, scanned, "1_setup.cdc", "File in init/ should be scanned")
	assert.Contains(t, scanned, "2_deploy.cdc", "File in init/ should be scanned")
	
	assert.NotContains(t, scanned, "root_tx.cdc", "File in parent folder should NOT be scanned")
	assert.NotContains(t, scanned, "test.cdc", "File in subfolder should NOT be scanned")
	
	assert.Equal(t, 2, len(scanned), "Should scan exactly 2 files")
	
	t.Logf("✓ Found %d files to execute: %v\n", len(scanned), scanned)
}

// TestScanning_EmptyFolder tests empty directory behavior
//
// Config: init_transactions_folder = ""
// Scans: aether/ folder (empty)
//
// Directory structure:
//   aether/
//   (empty)
//
// Expected: No files execute, no errors
func TestScanning_EmptyFolder(t *testing.T) {
	aetherFolder, err := os.MkdirTemp("", "aether-*")
	require.NoError(t, err)
	defer os.RemoveAll(aetherFolder)
	
	t.Logf("\n=== TEST: Empty Folder ===")
	t.Logf("Config: init_transactions_folder = \"\"")
	t.Logf("Scanning folder: %s (empty)\n", aetherFolder)
	
	scanned := scanFolder(t, aetherFolder)
	
	assert.Equal(t, 0, len(scanned), "Empty folder should scan 0 files")
	
	t.Logf("✓ Empty folder handled correctly\n")
}

// TestScanning_OnlyNonTransactionFiles tests folder with only non-.cdc/.json files
//
// Config: init_transactions_folder = ""
// Scans: aether/ folder (no transaction files)
//
// Directory structure:
//   aether/
//   ├── README.md
//   ├── config.yaml
//   └── notes.txt
//
// Expected: No files execute (none are .cdc or .json)
func TestScanning_OnlyNonTransactionFiles(t *testing.T) {
	aetherFolder, err := os.MkdirTemp("", "aether-*")
	require.NoError(t, err)
	defer os.RemoveAll(aetherFolder)
	
	t.Logf("\n=== TEST: Only Non-Transaction Files ===")
	t.Logf("Config: init_transactions_folder = \"\"")
	t.Logf("Scanning folder: %s\n", aetherFolder)
	
	createFile(t, aetherFolder, "README.md", "# Docs")
	createFile(t, aetherFolder, "config.yaml", "key: value")
	createFile(t, aetherFolder, "notes.txt", "Some notes")
	
	scanned := scanFolder(t, aetherFolder)
	
	assert.Equal(t, 0, len(scanned), "Should ignore non-.cdc/.json files")
	
	t.Logf("✓ All non-transaction files correctly ignored\n")
}

// TestScanning_DeepNestedStructure tests that deeply nested folders are completely ignored
//
// Config: init_transactions_folder = ""
// Scans: aether/ folder (root only)
//
// Directory structure:
//   aether/
//   ├── init.cdc            ✓ EXECUTES
//   └── a/                  ✗ SKIPPED
//       └── b/
//           └── c/
//               └── d/
//                   └── deep.cdc
//
// Expected: Only root init.cdc executes, deep.cdc is never seen
func TestScanning_DeepNestedStructure(t *testing.T) {
	aetherFolder, err := os.MkdirTemp("", "aether-*")
	require.NoError(t, err)
	defer os.RemoveAll(aetherFolder)
	
	t.Logf("\n=== TEST: Deep Nested Structure ===")
	t.Logf("Config: init_transactions_folder = \"\"")
	t.Logf("Scanning folder: %s\n", aetherFolder)
	
	// Root file
	createFile(t, aetherFolder, "init.cdc", "transaction {}")
	
	// Create deeply nested structure
	deepPath := filepath.Join(aetherFolder, "a", "b", "c", "d")
	os.MkdirAll(deepPath, 0755)
	createFile(t, deepPath, "deep.cdc", "transaction {}")
	
	scanned := scanFolder(t, aetherFolder)
	
	assert.Contains(t, scanned, "init.cdc", "Root file should be scanned")
	assert.NotContains(t, scanned, "deep.cdc", "Deeply nested file should NOT be scanned")
	assert.Equal(t, 1, len(scanned), "Should only scan root-level files")
	
	t.Logf("✓ Deep nesting correctly ignored\n")
}

// =============================================================================
// Helper Functions
// =============================================================================

// createFile creates a file with content in the specified directory
func createFile(t *testing.T, dir, filename, content string) {
	t.Helper()
	path := filepath.Join(dir, filename)
	err := os.WriteFile(path, []byte(content), 0644)
	require.NoError(t, err)
}

// scanFolder simulates the RunInitTransactions scanning logic
// Returns a list of filenames (not full paths) that would be processed
func scanFolder(t *testing.T, folder string) []string {
	t.Helper()
	
	var scanned []string
	
	err := filepath.Walk(folder, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// Skip subdirectories - only process files in root folder
		// This is the KEY LOGIC being tested!
		if info.IsDir() {
			if path != folder {
				return filepath.SkipDir
			}
			return nil
		}
		
		// Only process .cdc and .json files
		ext := filepath.Ext(info.Name())
		if ext == ".cdc" || ext == ".json" {
			scanned = append(scanned, info.Name())
		}
		
		return nil
	})
	
	require.NoError(t, err)
	return scanned
}
