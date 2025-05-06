package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStepsParser(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "steps-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create step definition files
	testFiles := map[string]string{
		"javascript_steps.js": `const { Given, When, Then, And } = require('@cucumber/cucumber');

Given('I have a user with name {string}', function(name) {
    this.user = { name };
});

When("I update the user's email to {string}", function(email) {
    this.user.email = email;
});

Then(/^I should have (\d+) items total$/, function(count) {
    assert.equal(this.items.length, parseInt(count));
});

And('the user should be active', function() {
    assert.equal(this.user.active, true);
});`,

		"typescript_steps.ts": `import { Given, When, Then, And } from '@cucumber/cucumber';

@Given('I have a TypeScript user with name {string}')
async function createUser(name: string) {
    this.user = { name };
}

@When(/^I add (\d+) more TypeScript items$/)
async function addItems(count: number) {
    this.items.push(...Array(count).fill({}));
}`,

		"coffee_steps.coffee": `{ Given, When, Then, And } = require '@cucumber/cucumber'

Given 'I have a CoffeeScript user with name {string}', (name) ->
  @user = { name }

When /^I add (\d+) more CoffeeScript items$/, (count) ->
  @items.push ...Array(parseInt(count)).fill {}`,
	}

	// Write test files
	for filename, content := range testFiles {
		filePath := filepath.Join(tempDir, "step_definitions", filename)
		err := os.MkdirAll(filepath.Dir(filePath), 0755)
		if err != nil {
			t.Fatalf("Failed to create directory for %s: %v", filename, err)
		}
		err = os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to write %s: %v", filename, err)
		}
	}

	// Run the Go parser
	stepsFile, err := parseStepsFiles(tempDir)
	if err != nil {
		t.Fatalf("Failed to parse steps files: %v", err)
	}

	// Verify the results
	assert.Equal(t, 3, stepsFile.TotalFiles, "Should find 3 step definition files")
	assert.Greater(t, stepsFile.TotalSteps, 0, "Should find steps in the files")

	// Verify each file's steps
	for _, stepDef := range stepsFile.StepDefinitions {
		switch filepath.Base(stepDef.File) {
		case "javascript_steps.js":
			assert.Equal(t, 4, len(stepDef.Steps), "JavaScript file should have 4 steps")
			verifyStepTypes(t, stepDef.Steps, []string{"Given", "When", "And", "Then"})
			verifyPatterns(t, stepDef.Steps, []string{
				"I have a user with name {string}",
				"I update the user",
				"the user should be active",
				"^I should have (\\d+) items total$",
			})
		case "typescript_steps.ts":
			assert.Equal(t, 4, len(stepDef.Steps), "TypeScript file should have 4 steps")
			verifyStepTypes(t, stepDef.Steps, []string{"Given", "When", "Given", "When"})
			verifyPatterns(t, stepDef.Steps, []string{
				"I have a TypeScript user with name {string}",
				"^I add (\\d+) more TypeScript items$",
				"I have a TypeScript user with name {string}",
				"^I add (\\d+) more TypeScript items$",
			})
		case "coffee_steps.coffee":
			assert.Equal(t, 2, len(stepDef.Steps), "CoffeeScript file should have 2 steps")
			verifyStepTypes(t, stepDef.Steps, []string{"Given", "When"})
			verifyPatterns(t, stepDef.Steps, []string{
				"I have a CoffeeScript user with name {string}",
				"^I add (\\d+) more CoffeeScript items$",
			})
		}
	}
}

func verifyStepTypes(t *testing.T, steps []StepDefinition, expectedTypes []string) {
	assert.Equal(t, len(expectedTypes), len(steps), "Number of steps should match expected types")
	for i, step := range steps {
		assert.Equal(t, expectedTypes[i], step.StepType, "Step type mismatch")
		assert.NotEmpty(t, step.Pattern, "Step pattern should not be empty")
		assert.NotEmpty(t, step.FileType, "File type should not be empty")
		assert.NotEmpty(t, step.PatternType, "Pattern type should not be empty")
	}
}

func verifyPatterns(t *testing.T, steps []StepDefinition, expectedPatterns []string) {
	assert.Equal(t, len(expectedPatterns), len(steps), "Number of steps should match expected patterns")
	for i, step := range steps {
		assert.Equal(t, expectedPatterns[i], step.Pattern, "Step pattern mismatch")
	}
}

// Helper function to parse steps files
func parseStepsFiles(projectPath string) (*StepsFile, error) {
	var matches []string
	err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !filepath.HasPrefix(path, filepath.Join(projectPath, "step_definitions")) {
			return nil
		}
		switch filepath.Ext(path) {
		case ".js", ".coffee", ".ts":
			matches = append(matches, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	stepsFile := &StepsFile{
		SearchPath: projectPath,
		TotalFiles: len(matches),
	}

	for _, file := range matches {
		var steps []StepDefinition
		var parseErr error

		switch filepath.Ext(file) {
		case ".js", ".ts":
			steps, parseErr = parseJavaScriptFile(file)
		case ".coffee":
			steps, parseErr = parseCoffeeScriptFile(file)
		}

		if parseErr != nil {
			return nil, parseErr
		}

		if len(steps) > 0 {
			relPath, err := filepath.Rel(projectPath, file)
			if err != nil {
				relPath = file
			}

			stepsFile.StepDefinitions = append(stepsFile.StepDefinitions, StepDefinitionFile{
				File:  relPath,
				Steps: steps,
			})
			stepsFile.TotalSteps += len(steps)
		}
	}

	return stepsFile, nil
}
