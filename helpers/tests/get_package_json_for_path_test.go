package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/LajnaLegenden/transpiler4/helpers"
)

func TestGetPackageJsonForPath(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "package-json-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir) // Clean up after test

	// Test case 1: No package.json file, required=false
	packageJson, err := helpers.GetPackageJsonForPath(tempDir, false)
	if err != nil {
		t.Errorf("Expected no error for missing package.json with required=false, got: %v", err)
	}
	if packageJson != nil {
		t.Errorf("Expected nil package.json for missing file with required=false, got non-nil")
	}

	// Test case 2: No package.json file, required=true
	packageJson, err = helpers.GetPackageJsonForPath(tempDir, true)
	if err == nil {
		t.Errorf("Expected error for missing package.json with required=true, got nil")
	}

	// Test case 3: Valid package.json
	validPackageJson := `{
		"name": "test-package",
		"scripts": {
			"build": "echo 'building'",
			"test": "echo 'testing'"
		}
	}`

	packageJsonPath := filepath.Join(tempDir, "package.json")
	err = os.WriteFile(packageJsonPath, []byte(validPackageJson), 0644)
	if err != nil {
		t.Fatalf("Failed to write package.json: %v", err)
	}

	packageJson, err = helpers.GetPackageJsonForPath(tempDir, false)
	if err != nil {
		t.Errorf("Expected no error for valid package.json, got: %v", err)
	}
	if packageJson == nil {
		t.Fatalf("Expected non-nil package.json for valid file, got nil")
	}
	if packageJson.Name != "test-package" {
		t.Errorf("Expected package name 'test-package', got '%s'", packageJson.Name)
	}
	if len(packageJson.Scripts) != 2 {
		t.Errorf("Expected 2 scripts, got %d", len(packageJson.Scripts))
	}
	if packageJson.Scripts["build"] != "echo 'building'" {
		t.Errorf("Expected build script 'echo 'building'', got '%s'", packageJson.Scripts["build"])
	}

	// Test case 4: Invalid JSON
	invalidPackageJson := `{ "name": "invalid-json", "scripts": { "missing-brace": "value" }`
	err = os.WriteFile(packageJsonPath, []byte(invalidPackageJson), 0644)
	if err != nil {
		t.Fatalf("Failed to write invalid package.json: %v", err)
	}

	packageJson, err = helpers.GetPackageJsonForPath(tempDir, false)
	if err == nil {
		t.Errorf("Expected error for invalid JSON, got nil")
	}
}
