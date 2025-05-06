package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/LajnaLegenden/transpiler4/helpers"
	"github.com/urfave/cli/v2"
)

// BuildCommand returns the CLI command for the build operation
func StepsCommand() *cli.Command {
	return &cli.Command{
		Name:    "steps",
		Aliases: []string{"s"},
		Usage:   "Generate a steps file in the current directory",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "path",
				Aliases: []string{"p"},
				Usage:   "Path to the project folder",
			},
		},
		Action: StepsAction,
	}
}

type StepsFile struct {
	SearchPath      string               `json:"searchPath"`
	GeneratedAt     string               `json:"generatedAt"`
	TotalFiles      int                  `json:"totalFiles"`
	TotalSteps      int                  `json:"totalSteps"`
	StepDefinitions []StepDefinitionFile `json:"stepDefinitions"`
}

type StepDefinitionFile struct {
	File  string           `json:"file"`
	Steps []StepDefinition `json:"steps"`
}

type StepDefinition struct {
	StepType    string `json:"type"`        // Then, When, Given
	Pattern     string `json:"pattern"`     // The regex pattern
	FileType    string `json:"fileType"`    // js, ts, coffee
	PatternType string `json:"patternType"` // regex, string, etc
	LineNumber  int    `json:"lineNumber"`  // The line number where the step was found
}

// Regular expressions for finding step definitions
var (
	// JavaScript/TypeScript patterns
	jsStringPattern          = regexp.MustCompile(`(?m)(Given|When|Then|And)\s*\(\s*['"\x60](.*?)['"\x60]`)
	jsRegexPattern           = regexp.MustCompile(`(?m)(Given|When|Then|And)\s*\(\s*/(.*?)/[gim]*\s*[,)]`)
	jsTemplatePattern        = regexp.MustCompile(`(?m)(Given|When|Then|And)\s*\(\s*\x60(.*?)\x60`)
	tsDecoratorStringPattern = regexp.MustCompile(`(?m)@(Given|When|Then|And)\s*\(\s*['"\x60](.*?)['"\x60]\s*\)`)
	tsDecoratorRegexPattern  = regexp.MustCompile(`(?m)@(Given|When|Then|And)\s*\(\s*/(.*?)/[gim]*\s*\)`)
	// CoffeeScript patterns
	coffeeStringPattern = regexp.MustCompile(`(?m)(Given|When|Then|And)\s*\(\s*['"\x60](.*?)['"\x60]`)
	coffeeRegexPattern  = regexp.MustCompile(`(?m)(Given|When|Then|And)\s*\(\s*/(.*?)/[gim]*\s*[,)]`)
)

// parseJavaScriptFile parses a JavaScript/TypeScript step definition file
func parseJavaScriptFile(filePath string) ([]StepDefinition, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	var steps []StepDefinition
	fileContent := string(content)
	fileType := filepath.Ext(filePath)[1:] // Remove the dot

	// Helper function to process matches from a pattern
	processMatches := func(pattern *regexp.Regexp, isRegex bool) {
		matches := pattern.FindAllStringSubmatch(fileContent, -1)
		for _, match := range matches {
			stepType := match[1]
			stepPattern := match[2]

			// Calculate line number
			lineNum := 1 + strings.Count(fileContent[:strings.Index(fileContent, match[0])], "\n")

			patternType := "string"
			if isRegex {
				patternType = "regex"
			}

			steps = append(steps, StepDefinition{
				StepType:    stepType,
				Pattern:     stepPattern,
				FileType:    fileType,
				PatternType: patternType,
				LineNumber:  lineNum,
			})
		}
	}

	// Process each pattern
	processMatches(jsStringPattern, false)
	processMatches(jsRegexPattern, true)
	processMatches(jsTemplatePattern, false)
	processMatches(tsDecoratorStringPattern, false)
	processMatches(tsDecoratorRegexPattern, true)

	return steps, nil
}

// parseCoffeeScriptFile parses a CoffeeScript step definition file
func parseCoffeeScriptFile(filePath string) ([]StepDefinition, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	var steps []StepDefinition
	fileContent := string(content)
	fileType := filepath.Ext(filePath)[1:] // Remove the dot

	// Helper function to process matches from a pattern
	processMatches := func(pattern *regexp.Regexp, isRegex bool) {
		matches := pattern.FindAllStringSubmatch(fileContent, -1)
		for _, match := range matches {
			stepType := match[1]
			stepPattern := match[2]

			// Calculate line number
			lineNum := 1 + strings.Count(fileContent[:strings.Index(fileContent, match[0])], "\n")

			patternType := "string"
			if isRegex {
				patternType = "regex"
			}

			steps = append(steps, StepDefinition{
				StepType:    stepType,
				Pattern:     stepPattern,
				FileType:    fileType,
				PatternType: patternType,
				LineNumber:  lineNum,
			})
		}
	}

	// Process each pattern
	processMatches(coffeeStringPattern, false)
	processMatches(coffeeRegexPattern, true)

	return steps, nil
}

// StepsAction handles the steps command execution
func StepsAction(c *cli.Context) error {
	fmt.Println("We are walking the tree and generating a steps file")
	projectPath, err := helpers.GetProjectPath(c.String("path"))
	if err != nil {
		return fmt.Errorf("failed to get project path: %w", err)
	}
	fmt.Printf("Using project path: %s\n", projectPath)

	var matches []string
	err = filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// only look at files
		if info.IsDir() {
			return nil
		}
		if !strings.Contains(path, "/step_definitions/") {
			return nil
		}
		if strings.Contains(path, "/node_modules/") {
			return nil
		}
		switch filepath.Ext(path) {
		case ".js", ".coffee", ".ts":
			matches = append(matches, path)
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Create StepsFile structure
	stepsFile := StepsFile{
		SearchPath:  projectPath,
		GeneratedAt: time.Now().Format(time.RFC3339),
		TotalFiles:  len(matches),
	}

	// Create channels and wait group for concurrent processing
	type result struct {
		file  string
		steps []StepDefinition
		err   error
	}
	resultChan := make(chan result, len(matches))
	var wg sync.WaitGroup

	// Process each file concurrently
	for _, file := range matches {
		wg.Add(1)
		go func(filePath string) {
			defer wg.Done()
			var steps []StepDefinition
			var parseErr error

			switch filepath.Ext(filePath) {
			case ".js", ".ts":
				steps, parseErr = parseJavaScriptFile(filePath)
			case ".coffee":
				steps, parseErr = parseCoffeeScriptFile(filePath)
			}

			resultChan <- result{
				file:  filePath,
				steps: steps,
				err:   parseErr,
			}
		}(file)
	}

	// Close the result channel when all goroutines are done
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results from the channel
	for res := range resultChan {
		if res.err != nil {
			fmt.Printf("Warning: Failed to parse %s: %v\n", res.file, res.err)
			continue
		}

		if len(res.steps) > 0 {
			// Convert absolute path to relative path
			relPath, err := filepath.Rel(projectPath, res.file)
			if err != nil {
				fmt.Printf("Warning: Failed to convert path to relative: %v\n", err)
				relPath = res.file // Fallback to absolute path if conversion fails
			}

			stepsFile.StepDefinitions = append(stepsFile.StepDefinitions, StepDefinitionFile{
				File:  relPath,
				Steps: res.steps,
			})
			stepsFile.TotalSteps += len(res.steps)
		}
	}

	// Write the JSON file
	jsonData, err := json.MarshalIndent(stepsFile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal steps file: %w", err)
	}

	outputPath := filepath.Join(projectPath, "steps.json")
	if err := os.WriteFile(outputPath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write steps file: %w", err)
	}

	fmt.Printf("Generated steps file at: %s\n", outputPath)
	fmt.Printf("Found %d step definition files with %d total steps\n", stepsFile.TotalFiles, stepsFile.TotalSteps)

	return nil
}
