package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/LajnaLegenden/transpiler4/helpers"
)

func TestFindNodePackages(t *testing.T) {
	// Create a temporary directory structure for testing
	rootDir, err := os.MkdirTemp("", "find-packages-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(rootDir) // Clean up after test

	// Create a package at the root level
	rootPackageJson := `{"name": "root-package"}`
	err = os.WriteFile(filepath.Join(rootDir, "package.json"), []byte(rootPackageJson), 0644)
	if err != nil {
		t.Fatalf("Failed to write root package.json: %v", err)
	}

	// Create node_modules directory (should be ignored)
	nodeModulesDir := filepath.Join(rootDir, "node_modules")
	err = os.Mkdir(nodeModulesDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create node_modules directory: %v", err)
	}

	// Create a package in node_modules (should be ignored)
	nodeModulesPackageDir := filepath.Join(nodeModulesDir, "fake-module")
	err = os.Mkdir(nodeModulesPackageDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create node_modules package directory: %v", err)
	}

	nodeModulesPackageJson := `{"name": "fake-module"}`
	err = os.WriteFile(filepath.Join(nodeModulesPackageDir, "package.json"), []byte(nodeModulesPackageJson), 0644)
	if err != nil {
		t.Fatalf("Failed to write node_modules package.json: %v", err)
	}

	// Create a nested package (should be found)
	nestedDir := filepath.Join(rootDir, "packages", "nested-package")
	err = os.MkdirAll(nestedDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create nested package directory: %v", err)
	}

	nestedPackageJson := `{"name": "nested-package", "scripts": {"build": "echo building"}}`
	err = os.WriteFile(filepath.Join(nestedDir, "package.json"), []byte(nestedPackageJson), 0644)
	if err != nil {
		t.Fatalf("Failed to write nested package.json: %v", err)
	}

	// Create another nested package with invalid JSON (should be skipped but not error)
	invalidDir := filepath.Join(rootDir, "packages", "invalid-package")
	err = os.MkdirAll(invalidDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create invalid package directory: %v", err)
	}

	invalidPackageJson := `{"name": "invalid-package", "scripts": {`
	err = os.WriteFile(filepath.Join(invalidDir, "package.json"), []byte(invalidPackageJson), 0644)
	if err != nil {
		t.Fatalf("Failed to write invalid package.json: %v", err)
	}

	// Run the function
	packages, err := helpers.FindNodePackages(rootDir)
	if err != nil {
		t.Errorf("Expected no error from FindNodePackages, got: %v", err)
	}

	// We expect to find 2 packages (root and nested), but not the one in node_modules
	// The invalid package should be skipped but not cause an error
	if len(packages) != 2 {
		t.Errorf("Expected to find 2 packages, got: %d", len(packages))
	}

	// Verify the packages that were found
	foundRootPackage := false
	foundNestedPackage := false

	for _, pkg := range packages {
		if pkg.PackageJson != nil {
			switch pkg.PackageJson.Name {
			case "root-package":
				foundRootPackage = true
			case "nested-package":
				foundNestedPackage = true
				// Check if the scripts were properly parsed
				if pkg.PackageJson.Scripts["build"] != "echo building" {
					t.Errorf("Nested package has incorrect build script")
				}
			case "fake-module":
				t.Errorf("Should not have found the package in node_modules")
			}
		}
	}

	if !foundRootPackage {
		t.Errorf("Did not find the root package")
	}

	if !foundNestedPackage {
		t.Errorf("Did not find the nested package")
	}
}
