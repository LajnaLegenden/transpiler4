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
	// Channel to signal appending packages
	appendChan := make(chan struct{})

	// Set up signal handling
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-signalChan
		// Stop the log socket server before exiting
		logsocket.StopServer()
		close(stopChan)
	}()

	return runWatchLoop(c, projectPath, stopChan, reselectChan, appendChan)
}

// startStdinListener starts a goroutine that listens for 'r' key (reselect) and 'a' key (append). Returns a stop channel to terminate the goroutine.
func startStdinListener(reselectChan chan<- struct{}, appendChan chan<- struct{}) chan struct{} {
	stopStdin := make(chan struct{})
	go func() {
		for {
			select {
			case <-stopStdin:
				return
			default:
				var b [1]byte
				os.Stdin.Read(b[:])
				if b[0] == 'r' || b[0] == 'R' {
					fmt.Println("\nReopening package selection menu...")
					reselectChan <- struct{}{}
				} else if b[0] == 'a' || b[0] == 'A' {
					fmt.Println("\nOpening menu to append packages...")
					appendChan <- struct{}{}
				}
			}
		}
	}()
	return stopStdin
}

// runWatchLoop manages the watcher lifecycle and package selection
func runWatchLoop(c *cli.Context, projectPath string, stopChan <-chan struct{}, reselectChan chan struct{}, appendChan chan struct{}) error {
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

	// startAdditionalWatchers starts watchers for newly appended packages
	startAdditionalWatchers := func(startIdx int) {
		for i := startIdx; i < len(selectedPackages); i++ {
			pkg := selectedPackages[i]
			log.Printf("Appended package: %s\n", pkg.PackageJson.Name)
			watcherStopChans[i] = make(chan struct{})
			wg.Add(1)
			go watchForChanges(&wg, watcherStopChans[i], pkg, projectPath+"/webapp", !c.Bool("no-build"))
		}
	}

	stopWatchers := func() {
		for _, ch := range watcherStopChans {
			close(ch)
		}
		wg.Wait()
	}

	// Initial menu selection (no stdin goroutine running yet)
	selectedPackages = helpers.SelectPackages(buildablePackages)
	watcherStopChans = make([]chan struct{}, len(selectedPackages))
	stopStdin := startStdinListener(reselectChan, appendChan)
	startWatchers()
	
	log.Println("Watching for changes. Press 'r' to reselect packages or 'a' to append additional packages.")

	for {
		select {
		case <-stopChan:
			close(stopStdin)
			stopWatchers()
			return nil
		case <-reselectChan:
			close(stopStdin)
			stopWatchers()
			// Reselect packages and restart watchers
			buildablePackages, err = helpers.FindNodePackages(projectPath)
			if err != nil {
				log.Fatal("Error selecting packages: ", err)
			}
			buildablePackages = helpers.GetBuildablePackages(buildablePackages)
			selectedPackages = helpers.SelectPackages(buildablePackages)
			watcherStopChans = make([]chan struct{}, len(selectedPackages))
			stopStdin = startStdinListener(reselectChan, appendChan)
			startWatchers()
			log.Println("Watching for changes. Press 'r' to reselect packages or 'a' to append additional packages.")
		case <-appendChan:
			// Append additional packages without stopping existing watchers
			buildablePackages, err = helpers.FindNodePackages(projectPath)
			if err != nil {
				log.Printf("Error finding packages: %v", err)
				continue
			}
			buildablePackages = helpers.GetBuildablePackages(buildablePackages)
			
			// Select additional packages (excluding already selected ones)
			additionalPackages := helpers.SelectAdditionalPackages(buildablePackages, selectedPackages)
			
			if len(additionalPackages) > 0 {
				// Remember the current size to know where to start new watchers
				previousSize := len(selectedPackages)
				
				// Append to selected packages
				selectedPackages = append(selectedPackages, additionalPackages...)
				
				// Extend watcherStopChans slice
				newStopChans := make([]chan struct{}, len(additionalPackages))
				watcherStopChans = append(watcherStopChans, newStopChans...)
				
				// Start watchers only for the new packages
				startAdditionalWatchers(previousSize)
				
				log.Printf("Successfully appended %d package(s)", len(additionalPackages))
			} else {
				log.Println("No additional packages selected")
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
