package helpers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gen2brain/beeep"
	"github.com/ktr0731/go-fuzzyfinder"
)

// GeneratePortNumber handle generating port number
func GeneratePortNumber() int {
	rand.Seed(time.Now().UnixNano())

	min := 1000
	max := 99999

	port := rand.Intn(max-min+1) + min

	return port
}

func IsMediatoolRoot(path string) bool {
	// check if we have a package.json file and if the name field in it is @mediatool/root
	packageJsonPath := filepath.Join(path, "package.json")
	if _, err := os.Stat(packageJsonPath); os.IsNotExist(err) {
		return false
	}

	packageJson, err := os.ReadFile(packageJsonPath)
	if err != nil {
		return false
	}

	return strings.Contains(string(packageJson), "@mediatool/root")
}

// PackageJson represents the structure of a package.json file
type PackageJson struct {
	Name             string            `json:"name"`
	Scripts          map[string]string `json:"scripts"`
	PackageManager   string            `json:"packageManager"`
	Dependencies     map[string]string `json:"dependencies"`
	PeerDependencies map[string]string `json:"peerDependencies"`
	// Add other fields as needed
}

// LinkingStrategy represents different strategies for linking packages
type LinkingStrategy string

const (
	TRANSPILED        LinkingStrategy = "TRANSPILED"
	TRANSPILED_LEGACY LinkingStrategy = "TRANSPILED_LEGACY"
	AMEND_NATIVE      LinkingStrategy = "AMEND_NATIVE"
	MAKEFILE_BUILD    LinkingStrategy = "MAKEFILE_BUILD"
	TRANSPILED_YARN   LinkingStrategy = "TRANSPILED_YARN"
	MT_INTEGRATIONS   LinkingStrategy = "MT_INTEGRATIONS"
)

// GetPackageJsonForPath reads and parses the package.json file at the given path
func GetPackageJsonForPath(absolutePath string, required bool) (*PackageJson, error) {
	packageJsonPath := filepath.Join(absolutePath, "package.json")

	data, err := os.ReadFile(packageJsonPath)
	if err != nil {
		if os.IsNotExist(err) && !required {
			return nil, nil
		}
		return nil, err
	}

	var packageJson PackageJson
	if err := json.Unmarshal(data, &packageJson); err != nil {
		return nil, err
	}

	return &packageJson, nil
}

// strategyChecker is a function that checks if a strategy applies
type strategyChecker func(folderItems map[string]bool, packageJson *PackageJson, absolutePath string) bool

// getStrategyCheckers returns a map of strategy checkers
func getStrategyCheckers() map[LinkingStrategy]strategyChecker {
	return map[LinkingStrategy]strategyChecker{
		TRANSPILED_YARN: func(folderItems map[string]bool, packageJson *PackageJson, absolutePath string) bool {
			return (folderItems["rollup.config.mjs"] ||
				folderItems["rollup.config.js"]) &&
				strings.Contains(packageJson.PackageManager, "yarn")
		},
		TRANSPILED: func(folderItems map[string]bool, packageJson *PackageJson, absolutePath string) bool {
			return folderItems["rollup.config.mjs"] ||
				folderItems["rollup.config.js"]
		},
		MT_INTEGRATIONS: func(folderItems map[string]bool, packageJson *PackageJson, absolutePath string) bool {
			if packageJson == nil || packageJson.Scripts == nil {
				return false
			}
			_, hasBuild := packageJson.Scripts["build"]
			return folderItems["amend"] && folderItems["lib"] && hasBuild
		},
		TRANSPILED_LEGACY: func(_ map[string]bool, packageJson *PackageJson, absolutePath string) bool {
			if packageJson == nil || packageJson.Scripts == nil {
				return false
			}
			_, hasBuild := packageJson.Scripts["build"]
			return hasBuild
		},
		AMEND_NATIVE: func(folderItems map[string]bool, _ *PackageJson, absolutePath string) bool {
			return folderItems["amend"] && folderItems["lib"]
		},
		MAKEFILE_BUILD: func(folderItems map[string]bool, _ *PackageJson, absolutePath string) bool {
			return folderItems["Makefile"]
		},
	}
}

// GetOptimalStrategy determines the optimal linking strategy based on folder contents and package.json
func GetOptimalStrategy(folderItems map[string]bool, packageJson *PackageJson, absolutePath string) LinkingStrategy {
	checkers := getStrategyCheckers()
	strategies := getOrderedStrategies()

	// Iterate through strategies in a defined order
	for _, strategy := range strategies {
		if checker, exists := checkers[strategy]; exists {
			if checker(folderItems, packageJson, absolutePath) {
				return strategy
			}
		}
	}

	return "UNKNOWN"
}

// This function would define the order of strategy evaluation
func getOrderedStrategies() []LinkingStrategy {
	return []LinkingStrategy{
		// List strategies in priority order
		TRANSPILED_YARN,
		TRANSPILED,
		MT_INTEGRATIONS,
		TRANSPILED_LEGACY,
		AMEND_NATIVE,
		MAKEFILE_BUILD,
	}
}

// GetLinkingStrategyForPackage analyzes a package directory and determines the appropriate linking strategy
func GetLinkingStrategyForPackage(absolutePath string) (LinkingStrategy, error) {
	packageJson, err := GetPackageJsonForPath(absolutePath, false)
	if err != nil {
		return "", err
	}

	if packageJson == nil {
		return "", nil
	}

	// Read directory contents
	files, err := os.ReadDir(absolutePath)
	if err != nil {
		return "", err
	}

	// Convert to a map for easier lookup
	folderItems := make(map[string]bool)
	for _, file := range files {
		folderItems[file.Name()] = true
	}

	return GetOptimalStrategy(folderItems, packageJson, absolutePath), nil
}

// NodePackage represents a Node.js package with its path and package.json data
type NodePackage struct {
	Path            string          `json:"path"`
	PackageJson     *PackageJson    `json:"packageJson"`
	Strategy        LinkingStrategy `json:"strategy"`
	IsMediatoolRoot bool            `json:"isMediatoolRoot"`
	FolderItems     map[string]bool `json:"folderItems"`
	IsFrontend      bool            `json:"isFrontend"`
}

// FindNodePackages recursively finds all Node.js packages in the given directory
// and its subdirectories, ignoring node_modules folders
func FindNodePackages(rootDir string) ([]NodePackage, error) {
	var packages []NodePackage

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip node_modules directories
		if info.IsDir() && info.Name() == "node_modules" {
			return filepath.SkipDir
		}

		// Check if this directory contains a package.json file
		if info.IsDir() {
			packageJsonPath := filepath.Join(path, "package.json")
			if _, err := os.Stat(packageJsonPath); err == nil {
				// Found a package.json file, read it
				absPath, err := filepath.Abs(path)
				if err != nil {
					return err
				}

				packageJson, err := GetPackageJsonForPath(absPath, false)
				if err != nil {
					// Just skip this package if we can't read its package.json
					return nil
				}

				if packageJson != nil {
					folderItems := GetFolderItems(absPath)
					strategy := GetOptimalStrategy(folderItems, packageJson, absPath)
					isFrontend := strings.Contains(packageJson.Name, "frontend") || packageJson.Dependencies["react"] != "" || packageJson.PeerDependencies["react"] != ""
					packages = append(packages, NodePackage{
						Path:            absPath,
						PackageJson:     packageJson,
						Strategy:        strategy,
						IsMediatoolRoot: IsMediatoolRoot(absPath),
						FolderItems:     folderItems,
						IsFrontend:      isFrontend,
					})
				}
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return packages, nil
}

func GetFolderItems(path string) map[string]bool {
	files, err := os.ReadDir(path)
	if err != nil {
		return nil
	}

	folderItems := make(map[string]bool)
	for _, file := range files {
		folderItems[file.Name()] = true
	}
	return folderItems
}

func SelectPackages(packages []NodePackage) []NodePackage {
	idx, err := fuzzyfinder.FindMulti(
		packages,
		func(i int) string {
			return packages[i].PackageJson.Name
		},
		fuzzyfinder.WithPreviewWindow(func(i, w, h int) string {
			if i == -1 {
				return ""
			}
			return fmt.Sprintf("%s: %s",
				packages[i].PackageJson.Name,
				packages[i].Strategy)
		}))
	if err != nil {
		log.Fatal(err)
	}

	// Create a new slice to hold the selected packages
	selected := make([]NodePackage, len(idx))
	for i, index := range idx {
		selected[i] = packages[index]
	}
	return selected
}

// SelectPackagesWithExisting allows selecting additional packages while showing currently watched packages
// It marks already-watched packages and shows a two-column preview
func SelectPackagesWithExisting(packages []NodePackage, existing []NodePackage) []NodePackage {
	// Create a map of existing package names for quick lookup
	existingMap := make(map[string]bool)
	for _, pkg := range existing {
		existingMap[pkg.PackageJson.Name] = true
	}

	// Create display names with [WATCHING] prefix for existing packages
	displayNames := make([]string, len(packages))
	for i, pkg := range packages {
		if existingMap[pkg.PackageJson.Name] {
			displayNames[i] = "[WATCHING] " + pkg.PackageJson.Name
		} else {
			displayNames[i] = pkg.PackageJson.Name
		}
	}

	idx, err := fuzzyfinder.FindMulti(
		packages,
		func(i int) string {
			return displayNames[i]
		},
		fuzzyfinder.WithPreviewWindow(func(i, w, h int) string {
			// Calculate column widths for two-column preview
			// Left column: ~60%, Right column: ~40%
			leftColWidth := int(float64(w) * 0.6)
			rightColWidth := int(float64(w) * 0.4)
			if i == -1 {
				// Show currently watching packages in right column
				var rightCol strings.Builder
				rightCol.WriteString("Currently Watching:\n")
				rightCol.WriteString(strings.Repeat("-", rightColWidth-2))
				rightCol.WriteString("\n")
				if len(existing) == 0 {
					rightCol.WriteString("(none)")
				} else {
					for _, pkg := range existing {
						rightCol.WriteString(fmt.Sprintf("• %s\n", pkg.PackageJson.Name))
					}
				}
				return rightCol.String()
			}

			// Left column: package details
			var leftCol strings.Builder
			pkg := packages[i]
			leftCol.WriteString(fmt.Sprintf("Package: %s\n", pkg.PackageJson.Name))
			leftCol.WriteString(fmt.Sprintf("Strategy: %s\n", pkg.Strategy))
			if pkg.IsFrontend {
				leftCol.WriteString("Type: Frontend\n")
			}
			leftCol.WriteString(fmt.Sprintf("Path: %s\n", pkg.Path))

			// Right column: currently watching packages
			var rightCol strings.Builder
			rightCol.WriteString("Currently Watching:\n")
			rightCol.WriteString(strings.Repeat("-", rightColWidth-2))
			rightCol.WriteString("\n")
			if len(existing) == 0 {
				rightCol.WriteString("(none)")
			} else {
				for _, existingPkg := range existing {
					rightCol.WriteString(fmt.Sprintf("• %s\n", existingPkg.PackageJson.Name))
				}
			}

			// Combine columns
			leftLines := strings.Split(leftCol.String(), "\n")
			rightLines := strings.Split(rightCol.String(), "\n")
			maxLines := len(leftLines)
			if len(rightLines) > maxLines {
				maxLines = len(rightLines)
			}

			var result strings.Builder
			for j := 0; j < maxLines; j++ {
				leftLine := ""
				if j < len(leftLines) {
					leftLine = leftLines[j]
				}
				rightLine := ""
				if j < len(rightLines) {
					rightLine = rightLines[j]
				}

				// Truncate lines to fit column widths
				if len(leftLine) > leftColWidth {
					leftLine = leftLine[:leftColWidth-3] + "..."
				}
				if len(rightLine) > rightColWidth {
					rightLine = rightLine[:rightColWidth-3] + "..."
				}

				// Format with padding
				result.WriteString(fmt.Sprintf("%-*s  %s\n", leftColWidth, leftLine, rightLine))
			}

			return result.String()
		}))
	if err != nil {
		log.Fatal(err)
	}

	// Filter out packages that are already being watched
	var newPackages []NodePackage
	for _, index := range idx {
		pkg := packages[index]
		if !existingMap[pkg.PackageJson.Name] {
			newPackages = append(newPackages, pkg)
		}
	}

	return newPackages
}

func GetAbsolutePath(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return absPath, nil
}

func GetProjectPath(path string) (string, error) {
	if path != "" {
		if IsMediatoolRoot(path) {
			return GetAbsolutePath(path)
		}
	}

	if IsMediatoolRoot(".") {
		return ".", nil
	}
	return "", errors.New("not mediatool root")
}

func GetBuildablePackages(packages []NodePackage) []NodePackage {
	//filter out root packages webapp and oackages without strategy
	buildablePackages := []NodePackage{}
	for _, pkg := range packages {
		if pkg.Strategy != "UNKNOWN" && pkg.PackageJson.Name != "mediatool-webapp" && !pkg.IsMediatoolRoot {
			buildablePackages = append(buildablePackages, pkg)
		}
	}
	return buildablePackages
}

func SendNotification(title string, message string) error {
	err := beeep.Notify(title, message, "")
	if err != nil {
		log.Println("Error sending notification:", err)
	}
	return err
}
