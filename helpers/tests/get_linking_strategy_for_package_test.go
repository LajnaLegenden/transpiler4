package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/LajnaLegenden/transpiler4/helpers"
)

func TestGetLinkingStrategyForPackage(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "linking-strategy-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir) // Clean up after test

	// Test case 1: Directory with no package.json
	strategy, err := helpers.GetLinkingStrategyForPackage(tempDir)
	if err != nil {
		t.Errorf("Expected no error for directory with no package.json, got: %v", err)
	}
	if strategy != "" {
		t.Errorf("Expected empty strategy for directory with no package.json, got: %s", strategy)
	}

	// Test case 2: TRANSPILED strategy
	// Create package.json
	packageJsonPath := filepath.Join(tempDir, "package.json")
	err = os.WriteFile(packageJsonPath, []byte(`{"name": "test-package"}`), 0644)
	if err != nil {
		t.Fatalf("Failed to write package.json: %v", err)
	}

	// Create rollup.config.mjs (for TRANSPILED strategy)
	rollupConfigPath := filepath.Join(tempDir, "rollup.config.mjs")
	err = os.WriteFile(rollupConfigPath, []byte("// Fake rollup config"), 0644)
	if err != nil {
		t.Fatalf("Failed to write rollup.config.mjs: %v", err)
	}

	strategy, err = helpers.GetLinkingStrategyForPackage(tempDir)
	if err != nil {
		t.Errorf("Expected no error for TRANSPILED check, got: %v", err)
	}
	if strategy != helpers.TRANSPILED {
		t.Errorf("Expected TRANSPILED strategy, got: %s", strategy)
	}

	// Remove rollup config for next test
	os.Remove(rollupConfigPath)

	// Test case 3: TRANSPILED_LEGACY strategy
	// Update package.json to include build script
	err = os.WriteFile(packageJsonPath, []byte(`{"name": "test-package", "scripts": {"build": "echo building"}}`), 0644)
	if err != nil {
		t.Fatalf("Failed to update package.json: %v", err)
	}

	strategy, err = helpers.GetLinkingStrategyForPackage(tempDir)
	if err != nil {
		t.Errorf("Expected no error for TRANSPILED_LEGACY check, got: %v", err)
	}
	if strategy != helpers.TRANSPILED_LEGACY {
		t.Errorf("Expected TRANSPILED_LEGACY strategy, got: %s", strategy)
	}

	// Test case 4: AMEND_NATIVE strategy
	// Create amend and lib directories
	amendDir := filepath.Join(tempDir, "amend")
	libDir := filepath.Join(tempDir, "lib")
	err = os.Mkdir(amendDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create amend directory: %v", err)
	}
	err = os.Mkdir(libDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create lib directory: %v", err)
	}

	// Reset package.json to remove build script
	err = os.WriteFile(packageJsonPath, []byte(`{"name": "test-package"}`), 0644)
	if err != nil {
		t.Fatalf("Failed to reset package.json: %v", err)
	}

	strategy, err = helpers.GetLinkingStrategyForPackage(tempDir)
	if err != nil {
		t.Errorf("Expected no error for AMEND_NATIVE check, got: %v", err)
	}
	if strategy != helpers.AMEND_NATIVE {
		t.Errorf("Expected AMEND_NATIVE strategy, got: %s", strategy)
	}

	// Clean up amend and lib for next test
	os.RemoveAll(amendDir)
	os.RemoveAll(libDir)

	// Test case 5: MAKEFILE_BUILD strategy
	makefilePath := filepath.Join(tempDir, "Makefile")
	err = os.WriteFile(makefilePath, []byte("all:\n\techo 'building'"), 0644)
	if err != nil {
		t.Fatalf("Failed to write Makefile: %v", err)
	}

	strategy, err = helpers.GetLinkingStrategyForPackage(tempDir)
	if err != nil {
		t.Errorf("Expected no error for MAKEFILE_BUILD check, got: %v", err)
	}
	if strategy != helpers.MAKEFILE_BUILD {
		t.Errorf("Expected MAKEFILE_BUILD strategy, got: %s", strategy)
	}
}
