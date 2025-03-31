package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/LajnaLegenden/transpiler4/helpers"
)

func TestIsMediatoolRoot(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "mediatool-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir) // Clean up after test

	// Test case 1: Directory with no package.json
	result := helpers.IsMediatoolRoot(tempDir)
	if result {
		t.Errorf("Expected IsMediatoolRoot to return false for directory with no package.json, got true")
	}

	// Test case 2: Directory with package.json but not @mediatool/root
	packageJsonPath := filepath.Join(tempDir, "package.json")
	err = os.WriteFile(packageJsonPath, []byte(`{"name": "some-other-package"}`), 0644)
	if err != nil {
		t.Fatalf("Failed to write package.json: %v", err)
	}

	result = helpers.IsMediatoolRoot(tempDir)
	if result {
		t.Errorf("Expected IsMediatoolRoot to return false for non-@mediatool/root package.json, got true")
	}

	// Test case 3: Directory with @mediatool/root package.json
	err = os.WriteFile(packageJsonPath, []byte(`{"name": "@mediatool/root"}`), 0644)
	if err != nil {
		t.Fatalf("Failed to write @mediatool/root package.json: %v", err)
	}

	result = helpers.IsMediatoolRoot(tempDir)
	if !result {
		t.Errorf("Expected IsMediatoolRoot to return true for @mediatool/root package.json, got false")
	}
}
