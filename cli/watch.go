package cli

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/urfave/cli/v2"

	"github.com/LajnaLegenden/transpiler4/helpers"
	"github.com/LajnaLegenden/transpiler4/logsocket"
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
	projectPath, err := helpers.GetProjectPath(c.String("path"))
	if err != nil {
		return fmt.Errorf("failed to get project path: %w", err)
	}

	projectPath, err = filepath.Abs(projectPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Start the log socket server
	port, err := logsocket.StartServer()
	if err != nil {
		return fmt.Errorf("failed to start log socket server: %w", err)
	}
	fmt.Printf("Log viewer available at http://localhost:%d\n", port)

	// Create a global log writer for non-package specific logs
	originalLogger := log.Writer()
	globalLogWriter := logsocket.NewLogWriter(originalLogger, "System")
	log.SetOutput(globalLogWriter)

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
		// Stop the log socket server before exiting
		logsocket.StopServer()
		close(stopChan)
	}()

	for _, pkg := range selectedPackages {
		log.Printf("Selected package: %s\n", pkg.PackageJson.Name)
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
			if err != nil {
				log.Fatal("Failed to add path to watcher: ", err)
			}
		}
		return nil
	})
}

func watchForChanges(wg *sync.WaitGroup, stopChan <-chan struct{}, pkg helpers.NodePackage, webappPath string) {
	defer wg.Done()

	// Create a package-specific logger
	packageName := pkg.PackageJson.Name
	packageLogger := log.New(logsocket.NewLogWriter(os.Stdout, packageName), "", log.LstdFlags)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		packageLogger.Fatalf("Failed to create watcher: %v", err)
	}
	defer watcher.Close()
	packageLogger.Printf("Watching for changes in package: %s", pkg.Path)

	if err := addDirsToWatcher(watcher, pkg.Path); err != nil {
		packageLogger.Fatalf("Failed to walk through directories: %v", err)
	}

	buildChan := make(chan struct{}, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Add debounce timer
	var debounceTimer *time.Timer
	debounceTimeout := 1000 * time.Millisecond // Configurable debounce delay

	go handleBuilds(ctx, buildChan, pkg, webappPath, packageLogger)

	for {
		select {
		case <-stopChan:
			packageLogger.Printf("Stopping watcher for package: %s", pkg.PackageJson.Name)
			return
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			handleEvent(event, buildChan, &ctx, &cancel, pkg, webappPath, &debounceTimer, debounceTimeout, packageLogger)
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			packageLogger.Printf("Watcher error: %v", err)
		}
	}
}

func handleEvent(event fsnotify.Event, buildChan chan struct{}, ctx *context.Context,
	cancel *context.CancelFunc, pkg helpers.NodePackage, webappPath string,
	debounceTimer **time.Timer, debounceTimeout time.Duration, logger *log.Logger) {
	if event.Op&fsnotify.Write == fsnotify.Write {
		logger.Printf("File %s has been modified", event.Name)
		// If there's an existing timer, stop it
		if *debounceTimer != nil {
			(*debounceTimer).Stop()
		}

		// Create a new timer
		*debounceTimer = time.AfterFunc(debounceTimeout, func() {
			select {
			case buildChan <- struct{}{}:
			default:
				// If we can't send to buildChan, reset the build context
				(*cancel)()
				*ctx, *cancel = context.WithCancel(context.Background())
				go handleBuilds(*ctx, buildChan, pkg, webappPath, logger)
			}
		})
	}
}

func handleBuilds(ctx context.Context, buildChan <-chan struct{}, pkg helpers.NodePackage, webappPath string, logger *log.Logger) {
	for range buildChan {
		logger.Printf("Starting build for package: %s", pkg.PackageJson.Name)
		err := helpers.BuildPackageWithLogger(ctx, pkg, webappPath, logger)
		if err != nil {
			logger.Printf("Build failed: %v", err)
		}
	}
}
