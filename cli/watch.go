package cli

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/urfave/cli/v2"

	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/LajnaLegenden/transpiler4/helpers"
)

// WatchCommand returns the CLI command for the watch operation
func WatchCommand() *cli.Command {
	return &cli.Command{
		Name:    "watch",
		Aliases: []string{"w"},
		Usage:   "Watch for changes and build and copy this project",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "path",
				Aliases: []string{"p"},
				Usage:   "Path to the project folder",
			},
		},
		Action: WatchAction,
	}
}

// WatchAction handles the watch command execution
func WatchAction(c *cli.Context) error {
	fmt.Println("We are watching for changes and building and copying this project")
	projectPath, err := helpers.GetProjectPath(c.String("path"))
	if err != nil {
		return fmt.Errorf("failed to get project path: %w", err)
	}

	projectPath, err = filepath.Abs(projectPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	packages, err := helpers.FindNodePackages(projectPath)
	if err != nil {
		log.Fatal("Error selecting packages: ", err)
	}
	buildablePackages := helpers.GetBuildablePackages(packages)
	selectedPackages := helpers.SelectPackages(buildablePackages)

	var wg sync.WaitGroup
	stopChan := make(chan struct{})

	// Set up signal handling
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-signalChan
		fmt.Println("\nReceived an interrupt, stopping...")
		close(stopChan)
	}()

	for _, pkg := range selectedPackages {
		fmt.Printf("Selected package: %s\n", pkg.PackageJson.Name)
		wg.Add(1)
		go watchForChanges(&wg, stopChan, pkg, projectPath+"/webapp")
	}

	wg.Wait() // Wait for all goroutines to finish
	return nil
}

// addDirsToWatcher recursively adds directories to the watcher, skipping node_modules
func addDirsToWatcher(watcher *fsnotify.Watcher, rootPath string) error {
	return filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		unallowedDirs := []string{"node_modules", ".git", "dist", "build", "test", "tests"}
		if info.IsDir() {
			for _, dir := range unallowedDirs {
				if filepath.Base(path) == dir {
					return filepath.SkipDir
				}
			}
		}
		if info.IsDir() {
			err = watcher.Add(path)
			log.Println("Added path to watcher:", path)
			if err != nil {
				log.Fatal("Failed to add path to watcher: ", err)
			}
		}
		return nil
	})
}

func watchForChanges(wg *sync.WaitGroup, stopChan <-chan struct{}, pkg helpers.NodePackage, webappPath string) {
	defer wg.Done()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("Failed to create watcher: %v", err)
	}
	defer watcher.Close()
	log.Printf("Watching for changes in package: %s", pkg.Path)

	if err := addDirsToWatcher(watcher, pkg.Path); err != nil {
		log.Fatalf("Failed to walk through directories: %v", err)
	}

	buildChan := make(chan struct{}, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Add debounce timer
	var debounceTimer *time.Timer
	debounceTimeout := 1000 * time.Millisecond // Configurable debounce delay

	go handleBuilds(ctx, buildChan, pkg, webappPath)

	for {
		select {
		case <-stopChan:
			log.Printf("Stopping watcher for package: %s", pkg.PackageJson.Name)
			return
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			handleEvent(event, buildChan, &ctx, &cancel, pkg, webappPath, &debounceTimer, debounceTimeout)
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Printf("Watcher error: %v", err)
		}
	}
}

func handleEvent(event fsnotify.Event, buildChan chan struct{}, ctx *context.Context,
	cancel *context.CancelFunc, pkg helpers.NodePackage, webappPath string,
	debounceTimer **time.Timer, debounceTimeout time.Duration) {
	log.Printf("Event: %s %s", event.Op.String(), event.Name)
	if event.Op&fsnotify.Write == fsnotify.Write {
		log.Printf("File %s has been modified", event.Name)

		// If there's an existing timer, stop it
		if *debounceTimer != nil {
			(*debounceTimer).Stop()
		}

		// Create a new timer
		*debounceTimer = time.AfterFunc(debounceTimeout, func() {
			log.Printf("Debounce timer expired, triggering build for %s", pkg.PackageJson.Name)
			select {
			case buildChan <- struct{}{}:
			default:
				// If we can't send to buildChan, reset the build context
				(*cancel)()
				*ctx, *cancel = context.WithCancel(context.Background())
				go handleBuilds(*ctx, buildChan, pkg, webappPath)
			}
		})
	}
}

func handleBuilds(ctx context.Context, buildChan <-chan struct{}, pkg helpers.NodePackage, webappPath string) {
	for range buildChan {
		log.Printf("Starting build for package: %s", pkg.PackageJson.Name)
		err := helpers.BuildPackage(ctx, pkg, webappPath)
		if err != nil {
			log.Printf("Build failed: %v", err)
		}
	}
}
