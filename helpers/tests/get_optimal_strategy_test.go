package tests

import (
	"testing"

	"github.com/LajnaLegenden/transpiler4/helpers"
)

func TestGetOptimalStrategy(t *testing.T) {
	// Test case 1: TRANSPILED strategy
	folderItems := map[string]bool{
		"rollup.config.mjs": true,
		"package.json":      true,
		"src":               true,
	}
	packageJson := &helpers.PackageJson{
		Name: "test-package",
		Scripts: map[string]string{
			"build": "echo 'building'",
		},
	}

	strategy := helpers.GetOptimalStrategy(folderItems, packageJson)
	if strategy != helpers.TRANSPILED {
		t.Errorf("Expected TRANSPILED strategy, got %s", strategy)
	}

	// Test case 2: TRANSPILED_LEGACY strategy
	folderItems = map[string]bool{
		"package.json": true,
		"src":          true,
	}
	packageJson = &helpers.PackageJson{
		Name: "test-package",
		Scripts: map[string]string{
			"build": "echo 'building'",
		},
	}

	strategy = helpers.GetOptimalStrategy(folderItems, packageJson)
	if strategy != helpers.TRANSPILED_LEGACY {
		t.Errorf("Expected TRANSPILED_LEGACY strategy, got %s", strategy)
	}

	// Test case 3: AMEND_NATIVE strategy
	folderItems = map[string]bool{
		"package.json": true,
		"amend":        true,
		"lib":          true,
	}
	packageJson = &helpers.PackageJson{
		Name:    "test-package",
		Scripts: map[string]string{},
	}

	strategy = helpers.GetOptimalStrategy(folderItems, packageJson)
	if strategy != helpers.AMEND_NATIVE {
		t.Errorf("Expected AMEND_NATIVE strategy, got %s", strategy)
	}

	// Test case 4: MAKEFILE_BUILD strategy
	folderItems = map[string]bool{
		"package.json": true,
		"Makefile":     true,
	}
	packageJson = &helpers.PackageJson{
		Name:    "test-package",
		Scripts: map[string]string{},
	}

	strategy = helpers.GetOptimalStrategy(folderItems, packageJson)
	if strategy != helpers.MAKEFILE_BUILD {
		t.Errorf("Expected MAKEFILE_BUILD strategy, got %s", strategy)
	}

	// Test case 5: No matching strategy
	folderItems = map[string]bool{
		"package.json": true,
		"README.md":    true,
	}
	packageJson = &helpers.PackageJson{
		Name:    "test-package",
		Scripts: map[string]string{},
	}

	strategy = helpers.GetOptimalStrategy(folderItems, packageJson)
	if strategy != "" {
		t.Errorf("Expected empty strategy for no match, got %s", strategy)
	}

	// Test case 6: nil packageJson
	folderItems = map[string]bool{
		"package.json": true,
		"Makefile":     true,
	}

	strategy = helpers.GetOptimalStrategy(folderItems, nil)
	if strategy != helpers.MAKEFILE_BUILD {
		t.Errorf("Expected MAKEFILE_BUILD strategy with nil packageJson, got %s", strategy)
	}
}
