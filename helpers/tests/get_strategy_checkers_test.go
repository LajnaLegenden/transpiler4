package tests

import (
	"testing"

	"github.com/LajnaLegenden/transpiler4/helpers"
)

// We can't directly test the private getStrategyCheckers function
// So we'll test it indirectly through GetOptimalStrategy
func TestStrategyCheckers(t *testing.T) {
	// Test TRANSPILED strategy checker
	folderItems := map[string]bool{
		"rollup.config.mjs": true,
	}
	packageJson := &helpers.PackageJson{
		Scripts: map[string]string{
			"build": "echo building", // Even with a build script, rollup.config.mjs takes precedence
		},
	}

	strategy := helpers.GetOptimalStrategy(folderItems, packageJson)
	if strategy != helpers.TRANSPILED {
		t.Errorf("TRANSPILED strategy checker failed, got: %s", strategy)
	}

	// Test TRANSPILED_LEGACY strategy checker - only when no rollup.config.mjs
	folderItems = map[string]bool{}
	packageJson = &helpers.PackageJson{
		Scripts: map[string]string{
			"build": "echo building",
		},
	}

	strategy = helpers.GetOptimalStrategy(folderItems, packageJson)
	if strategy != helpers.TRANSPILED_LEGACY {
		t.Errorf("TRANSPILED_LEGACY strategy checker failed, got: %s", strategy)
	}

	// Test AMEND_NATIVE strategy checker
	folderItems = map[string]bool{
		"amend": true,
		"lib":   true,
	}
	packageJson = &helpers.PackageJson{}

	strategy = helpers.GetOptimalStrategy(folderItems, packageJson)
	if strategy != helpers.AMEND_NATIVE {
		t.Errorf("AMEND_NATIVE strategy checker failed, got: %s", strategy)
	}

	// Test MAKEFILE_BUILD strategy checker
	folderItems = map[string]bool{
		"Makefile": true,
	}
	packageJson = &helpers.PackageJson{}

	strategy = helpers.GetOptimalStrategy(folderItems, packageJson)
	if strategy != helpers.MAKEFILE_BUILD {
		t.Errorf("MAKEFILE_BUILD strategy checker failed, got: %s", strategy)
	}

	// Test with nil packageJson
	folderItems = map[string]bool{
		"Makefile": true,
	}
	packageJson = nil

	strategy = helpers.GetOptimalStrategy(folderItems, packageJson)
	if strategy != helpers.MAKEFILE_BUILD {
		t.Errorf("Strategy checker with nil packageJson failed, got: %s", strategy)
	}

	// Test no matching strategy
	folderItems = map[string]bool{}
	packageJson = &helpers.PackageJson{}

	strategy = helpers.GetOptimalStrategy(folderItems, packageJson)
	if strategy != "" {
		t.Errorf("No matching strategy checker shouldn't match, got: %s", strategy)
	}
}
