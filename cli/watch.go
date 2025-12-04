package cli

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/urfave/cli/v2"

	"slices"

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
			&cli.BoolFlag{
				Name:    "no-build",
				Aliases: []string{"n"},
				Usage:   "Disable initial build when starting watch",
			},
			&cli.StringSliceFlag{
				Name:    "pack",
				Aliases: []string{"packages"},
				Usage:   "Package name patterns to watch (fuzzy matched, skips interactive selection)",
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

	// Channel to signal stopping all watchers
	stopChan := make(chan struct{})
	// Channel to signal reselecting packages
	reselectChan := make(chan struct{})
	// Channel to signal adding packages
	addPackagesChan := make(chan struct{})

	// Set up signal handling
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-signalChan
		// Stop the log socket server before exiting
		logsocket.StopServer()
		close(stopChan)
	}()

	return runWatchLoop(c, projectPath, stopChan, reselectChan, addPackagesChan)
}

// stdinListenerResult holds the stop channel and wait group for the stdin listener
type stdinListenerResult struct {
	stopChan chan struct{}
	done     *sync.WaitGroup
}

// startStdinListener starts a goroutine that listens for 'a' key and sends to addPackagesChan. 
// Returns a result struct with stop channel and wait group to ensure clean shutdown.
func startStdinListener(addPackagesChan chan<- struct{}) stdinListenerResult {
	stopStdin := make(chan struct{})
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stopStdin:
				return
			default:
				var b [1]byte
				n, err := os.Stdin.Read(b[:])
				if err != nil || n == 0 {
					// Check if we should stop after read error
					select {
					case <-stopStdin:
						return
					default:
						continue
					}
				}
				// Check if we should stop after reading
				select {
				case <-stopStdin:
					return
				default:
					// Only handle 'a' or 'A' keypress
					if b[0] == 'a' || b[0] == 'A' {
						fmt.Println("\nAdding packages to watch list...")
						addPackagesChan <- struct{}{}
					}
					// All other keypresses are ignored (not consumed), allowing Ctrl+C and other signals to work normally
				}
			}
		}
	}()
	return stdinListenerResult{stopChan: stopStdin, done: wg}
}

// runWatchLoop manages the watcher lifecycle and package selection
func runWatchLoop(c *cli.Context, projectPath string, stopChan <-chan struct{}, reselectChan chan struct{}, addPackagesChan chan struct{}) error {
	buildablePackages, err := helpers.FindNodePackages(projectPath)
	if err != nil {
		log.Fatal("Error selecting packages: ", err)
	}
	buildablePackages = helpers.GetBuildablePackages(buildablePackages)

	var (
		selectedPackages []helpers.NodePackage
		watcherStopChans []chan struct{}
		wg               sync.WaitGroup
	)

	startWatchers := func() {
		for i, pkg := range selectedPackages {
			log.Printf("Selected package: %s\n", pkg.PackageJson.Name)
			watcherStopChans[i] = make(chan struct{})
			wg.Add(1)
			go watchForChanges(&wg, watcherStopChans[i], pkg, projectPath+"/webapp", !c.Bool("no-build"))
		}
	}

	addWatchers := func(newPackages []helpers.NodePackage) {
		startIdx := len(selectedPackages)
		selectedPackages = append(selectedPackages, newPackages...)
		// Extend watcherStopChans slice and start watchers for new packages
		for i, pkg := range newPackages {
			log.Printf("Added package to watch: %s\n", pkg.PackageJson.Name)
			watcherStopChans = append(watcherStopChans, make(chan struct{}))
			wg.Add(1)
			go watchForChanges(&wg, watcherStopChans[startIdx+i], pkg, projectPath+"/webapp", !c.Bool("no-build"))
		}
	}

	stopWatchers := func() {
		for _, ch := range watcherStopChans {
			close(ch)
		}
		wg.Wait()
	}

	// Initial menu selection (no stdin goroutine running yet)
	packQueriesRaw := c.StringSlice("pack")
	var packQueries []string
	// Expand comma-separated values (e.g., "pkg1,pkg2" -> ["pkg1", "pkg2"])
	for _, query := range packQueriesRaw {
		if query == "" {
			continue
		}
		// Split by comma and trim whitespace
		parts := strings.Split(query, ",")
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				packQueries = append(packQueries, trimmed)
			}
		}
	}

	if len(packQueries) > 0 {
		// Use flag-based selection
		selectedPackages = helpers.SelectPackagesByQuery(buildablePackages, packQueries)
		if len(selectedPackages) == 0 {
			return fmt.Errorf("no packages found matching the provided patterns: %v", packQueries)
		}
		log.Printf("Selected %d package(s) based on patterns: %v\n", len(selectedPackages), packQueries)
	} else {
		// Use interactive selection
		selectedPackages = helpers.SelectPackages(buildablePackages)
	}
	watcherStopChans = make([]chan struct{}, len(selectedPackages))
	stdinListener := startStdinListener(addPackagesChan)
	startWatchers()

	for {
		select {
		case <-stopChan:
			close(stdinListener.stopChan)
			stdinListener.done.Wait()
			stopWatchers()
			return nil
		case <-reselectChan:
			close(stdinListener.stopChan)
			stdinListener.done.Wait()
			stopWatchers()
			// Reselect packages and restart watchers
			buildablePackages, err = helpers.FindNodePackages(projectPath)
			if err != nil {
				log.Fatal("Error selecting packages: ", err)
			}
			buildablePackages = helpers.GetBuildablePackages(buildablePackages)
			selectedPackages = helpers.SelectPackages(buildablePackages)
			watcherStopChans = make([]chan struct{}, len(selectedPackages))
			stdinListener = startStdinListener(addPackagesChan)
			startWatchers()
		case <-addPackagesChan:
			// Stop stdin listener to allow fuzzy finder to take control of stdin
			close(stdinListener.stopChan)
			// Wait for the goroutine to exit (it may be blocked on Read, so give it a moment)
			// Use a goroutine to wait so we don't block, but ensure it's stopped before fuzzy finder
			done := make(chan struct{})
			go func() {
				stdinListener.done.Wait()
				close(done)
			}()
			// Wait a short time for the goroutine to exit, or proceed if it's taking too long
			select {
			case <-done:
				// Goroutine exited cleanly
			case <-time.After(100 * time.Millisecond):
				// Timeout - proceed anyway (goroutine will exit eventually)
			}
			// Refresh buildable packages list
			buildablePackages, err = helpers.FindNodePackages(projectPath)
			if err != nil {
				log.Fatal("Error finding packages: ", err)
			}
			buildablePackages = helpers.GetBuildablePackages(buildablePackages)
			// Select new packages (excluding already watched ones)
			newPackages := helpers.SelectPackagesWithExisting(buildablePackages, selectedPackages)
			// Restart stdin listener after selection is complete
			stdinListener = startStdinListener(addPackagesChan)
			if len(newPackages) > 0 {
				addWatchers(newPackages)
				log.Printf("Added %d package(s) to watch list\n", len(newPackages))
			} else {
				log.Println("No new packages selected")
			}
		}
	}
}

// addDirsToWatcher recursively adds directories to the watcher, skipping node_modules
func addDirsToWatcher(watcher *fsnotify.Watcher, rootPath string) error {
	return filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		unallowedDirs := []string{"node_modules", ".git", "dist", "build", "test", "tests", "features"}
		if info.IsDir() {
			if slices.Contains(unallowedDirs, filepath.Base(path)) {
				return filepath.SkipDir
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

func watchForChanges(wg *sync.WaitGroup, stopChan <-chan struct{}, pkg helpers.NodePackage, webappPath string, initialBuild bool) {
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

	// Trigger initial build if enabled
	if initialBuild {
		buildChan <- struct{}{}
	}

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
