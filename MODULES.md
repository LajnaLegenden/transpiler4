# Transpiler4 Module Documentation

This document provides detailed information about each module in the Transpiler4 project and explains how they communicate with each other.

## Table of Contents

1. [CLI Module](#cli-module)
2. [Helpers Module](#helpers-module)
3. [LogSocket Module](#logsocket-module)
4. [Module Communication](#module-communication)

## CLI Module

The CLI module is the entry point of the application and handles command-line interaction, implementing the commands available to users.

### Components

#### cli.go

`cli.go` serves as the main entry point for the CLI application:

- Initializes the CLI application using the `urfave/cli` package
- Defines the main command structure and global options
- Registers the available commands (Build, Watch, List)
- Organizes commands and flags for better user experience

```go
func SetupCLI() *cli.App {
    app := &cli.App{
        Name:    "mtcli",
        Usage:   "Mediatool CLI",
        Version: "1.0.5",
        Commands: []*cli.Command{
            BuildCommand(),
            WatchCommand(),
            ListCommand(),
        },
    }
    // ...
    return app
}
```

#### build.go

`build.go` implements the build command for building packages:

- Defines the command structure and flags for the build command
- Provides functionality to build selected packages once
- Uses the Helpers module to find, select, and build packages
- Reports build progress and results

#### watch.go

`watch.go` implements the watch command for continuous development:

- Sets up file system watchers to monitor package directories for changes
- Manages goroutines for watching multiple packages concurrently
- Implements debouncing to prevent excessive builds during rapid changes
- Coordinates communication between file system events and build processes
- Integrates with the LogSocket module for real-time logging

Key components:
- `WatchAction`: The main function that coordinates the watch process
- `watchForChanges`: A goroutine function that monitors a specific package
- `handleEvent`: Processes file system events and schedules builds
- `handleBuilds`: Executes builds when triggered by events

#### list.go

`list.go` implements the list command:

- Shows all available packages in the project
- Displays package information such as name, path, and build strategy
- Helps users identify available packages before using other commands

## Helpers Module

The Helpers module provides utility functions for working with Node.js packages, determining build strategies, and executing build commands.

### Components

#### helpers.go

`helpers.go` contains core utility functions for package management:

- **Package Detection**
  - `FindNodePackages`: Recursively finds all Node.js packages in a directory tree
  - `GetFolderItems`: Lists files and directories in a package folder
  - `GetPackageJsonForPath`: Reads and parses package.json files

- **Strategy Determination**
  - `GetLinkingStrategyForPackage`: Analyzes a package to determine its build strategy
  - `GetOptimalStrategy`: Selects the most appropriate strategy based on package contents
  - `getStrategyCheckers`: Defines rules for identifying build strategies

- **Package Selection**
  - `SelectPackages`: Implements a fuzzy finder for package selection
  - `GetBuildablePackages`: Filters packages that can be built

- **Path Handling**
  - `GetProjectPath`: Resolves and validates project paths
  - `GetAbsolutePath`: Converts relative paths to absolute paths

```go
type NodePackage struct {
    Path            string          `json:"path"`
    PackageJson     *PackageJson    `json:"packageJson"`
    Strategy        LinkingStrategy `json:"strategy"`
    IsMediatoolRoot bool            `json:"isMediatoolRoot"`
    FolderItems     map[string]bool `json:"folderItems"`
    IsFrontend      bool            `json:"isFrontend"`
}
```

#### buildCommands.go

`buildCommands.go` handles executing build commands for different strategies:

- **Command Execution**
  - `RunCommand`: Executes shell commands in a specific directory
  - `RunCommandWithLogger`: Executes commands with custom logging

- **Strategy-Specific Building**
  - `GetBuildCommand`: Returns appropriate build commands based on package strategy
  - `BuildPackage`: Builds a package using its strategy
  - `BuildPackageWithLogger`: Builds a package with custom logging

```go
func GetBuildCommand(pkg NodePackage, webappPath string) []string {
    if pkg.Strategy == "TRANSPILED_YARN" {
        return []string{
            "yarn transpile",
            "rm -rf " + webappPath + "/node_modules/" + pkg.PackageJson.Name + "/dist",
            "cp -R " + pkg.Path + "/dist " + webappPath + "/node_modules/" + pkg.PackageJson.Name,
        }
    }
    // ... other strategies
}
```

#### timehelper.go

`timehelper.go` provides time-related utility functions:

- `GetCurrentTimeMillis`: Returns the current time in milliseconds for logging timestamps

## LogSocket Module

The LogSocket module provides real-time logging through a WebSocket-based web interface.

### Components

#### logsocket.go

`logsocket.go` implements a WebSocket server and web interface:

- **Server Management**
  - `StartServer`: Initializes and starts the WebSocket server
  - `StopServer`: Gracefully stops the server
  - `handleWebSocket`: Handles WebSocket connection lifecycle

- **Client Management**
  - Tracks connected clients in a thread-safe map
  - Manages client connections and disconnections
  - Broadcasts messages to all connected clients

- **Log Capture and Distribution**
  - `LogWriter`: Custom io.Writer implementation for capturing logs
  - `broadcastMessage`: Sends log messages to all connected clients
  - `SendPackageLog`: Formats and sends package-specific logs

- **Web Interface**
  - Serves a Vue.js application with Tailwind CSS for viewing logs
  - Organizes logs by package with timestamps
  - Provides real-time updates without refreshing

```go
// LogWriter is a custom io.Writer that captures logs and sends them to WebSocket clients
type LogWriter struct {
    underlying  io.Writer // The original writer to also write logs to
    packageName string    // The package this writer is associated with
}

// Write implements io.Writer and captures logs to send to WebSocket clients
func (w *LogWriter) Write(p []byte) (n int, err error) {
    // Write to the underlying writer
    if w.underlying != nil {
        w.underlying.Write(p)
    }

    // Send log message to WebSocket clients
    // ...
}
```

## Module Communication

Modules in Transpiler4 communicate through several mechanisms:

### 1. Direct Function Calls

The primary communication method is through direct function calls:

- The CLI module calls functions in the Helpers module to find, select, and build packages
- The Watch command uses the LogSocket module to set up logging for each package

```go
// Example of CLI module calling Helpers module
packages, err := helpers.FindNodePackages(projectPath)
selectedPackages := helpers.SelectPackages(buildablePackages)
```

### 2. Data Structures

Shared data structures facilitate communication between modules:

- The `NodePackage` struct defined in the Helpers module is used by the CLI module
- Log messages are structured as `LogMessage` objects for consistency

```go
// Data structure shared between modules
type LogMessage struct {
    Package string `json:"package"`
    Message string `json:"message"`
    Time    int64  `json:"time"`
}
```

### 3. Channels

Go channels enable asynchronous communication within and between modules:

- The Watch command uses channels to signal build events from file watchers
- A debounce mechanism uses channels to control build timing

```go
// Channel-based communication
buildChan := make(chan struct{}, 1)
stopChan := make(chan struct{})

// Trigger a build through a channel
buildChan <- struct{}{}
```

### 4. Custom Writers

The `LogWriter` interface enables communication between logging and the WebSocket server:

- Custom writers capture logs from different packages
- The captured logs are sent to the WebSocket server for distribution
- Each package has its own writer for isolated logging

```go
// Register a custom logger for a package
packageLogger := log.New(logsocket.NewLogWriter(os.Stdout, packageName), "", log.LstdFlags)
```

### 5. WebSocket Communication

The LogSocket module implements WebSocket-based communication:

- The server manages connections with clients
- Messages are broadcast to all connected clients
- The web interface receives and displays messages in real-time

```go
// Broadcasting messages to WebSocket clients
func broadcastMessage(message LogMessage) {
    // Convert message to JSON
    // Send to all connected clients
}
```

### Communication Flow Example

The following example illustrates the communication flow when a file change is detected:

1. The file system watcher in the Watch command detects a change
2. A debounced event is sent through the `buildChan` channel
3. The `handleBuilds` goroutine receives the event and calls `BuildPackageWithLogger`
4. `BuildPackageWithLogger` in the Helpers module executes build commands
5. Output from the commands is captured by the `LogWriter`
6. The `LogWriter` sends the captured logs to the WebSocket server
7. The WebSocket server broadcasts the logs to all connected clients
8. The web interface receives and displays the logs in real-time

This multi-layered communication approach allows the system to handle multiple packages concurrently while providing real-time feedback to users. 