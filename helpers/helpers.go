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
