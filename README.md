# Transpiler4 - Mediatool CLI

A powerful command line interface for managing, building, and watching Mediatool packages in a monorepo environment. This tool streamlines the development workflow by automatically detecting changes in packages, building them, and copying the results to the appropriate locations.

## Features

- Automatically detect and build packages when files change
- Support for multiple build strategies based on package configuration
- Real-time log viewing through a web interface
- Cross-package dependency management
- Intelligent package selection via fuzzy finder

## Installation

### Prerequisites

- Go 1.18 or higher
- Node.js and npm/yarn/pnpm depending on your project

### Building from Source

```bash
go build -o mtcli
```

### Installing the CLI

```bash
go install github.com/LajnaLegenden/transpiler4@latest
```

## Usage

```
NAME:
   mtcli - Mediatool CLI

USAGE:
   mtcli [global options] command [command options] [arguments...]

VERSION:
   1.0.5

COMMANDS:
   build, b    Build and copy this project
   watch, w    Watch for changes and build and copy this project
   list, l     List all available packages
   help, h     Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --help, -h     show help
   --version, -v  print the version
```

## Core Commands

### Watch Command

The `watch` command monitors specified packages for changes and automatically builds and deploys them when changes are detected.

```bash
mtcli watch --path <project_path>
```

Options:
- `--path, -p`: Path to the project folder (optional, defaults to current directory)

### Build Command

The `build` command builds specified packages once and deploys them.

```bash
mtcli build --path <project_path>
```

Options:
- `--path, -p`: Path to the project folder (optional, defaults to current directory)

### List Command

The `list` command displays all available packages in the project.

```bash
mtcli list --path <project_path>
```

Options:
- `--path, -p`: Path to the project folder (optional, defaults to current directory)

## Architecture

Transpiler4 is built with a modular architecture to handle different aspects of the build process.

### Modules

#### CLI Module

The CLI module handles command-line interaction and dispatches commands to the appropriate handlers.

- **cli.go**: Sets up the CLI application and defines the available commands
- **build.go**: Implements the build command
- **watch.go**: Implements the watch command with file system monitoring
- **list.go**: Implements the list command to display available packages

#### Helpers Module

The Helpers module provides utility functions for package detection, building, and other operations.

- **helpers.go**: Contains utility functions for working with Node.js packages, including:
  - Package detection and analysis
  - Strategy determination
  - Package filtering and selection 
- **buildCommands.go**: Contains functions for building packages with different strategies:
  - Command execution
  - Strategy-specific build commands
  - Build notification handling
- **timehelper.go**: Provides time-related utility functions

#### LogSocket Module

The LogSocket module provides real-time logging through a web interface.

- **logsocket.go**: Implements a WebSocket server and web interface for viewing logs in real-time

### Linking Strategies

The system supports multiple linking strategies based on package configuration:

1. **TRANSPILED_YARN**: For packages using rollup with Yarn
2. **TRANSPILED**: For packages using rollup with other package managers
3. **TRANSPILED_LEGACY**: For packages with a build script but no rollup config
4. **AMEND_NATIVE**: For packages with an amend and lib directory
5. **MAKEFILE_BUILD**: For packages using a Makefile for building

### Communication Channels

#### Channel Types

1. **Inter-module Communication**: 
   - Modules communicate through function calls and shared data structures
   - The `NodePackage` struct holds package information that's passed between modules

2. **Logging Communication**:
   - The LogSocket module provides WebSocket-based real-time logging
   - Custom `LogWriter` implementations capture and broadcast logs

3. **File System Watching**:
   - The file system watcher sends events to build channels when files change
   - A debounce mechanism prevents excessive builds during rapid changes

#### WebSocket Communication

The LogSocket module implements a WebSocket server for real-time log viewing:

1. **Server Components**:
   - HTTP server with WebSocket upgrade capability
   - Client tracking with connection management
   - Message broadcasting system

2. **Message Flow**:
   - Log messages are captured through custom `LogWriter` implementations
   - Messages are broadcast to all connected WebSocket clients
   - The web interface displays logs in real-time, organized by package

## Development Workflow

1. Package Selection:
   - When starting the CLI, it scans for Node.js packages
   - Users select which packages to watch using a fuzzy finder

2. File Watching:
   - The system monitors selected package directories for changes
   - Changes trigger the appropriate build strategy

3. Building and Deployment:
   - Each package is built according to its detected strategy
   - Built artifacts are copied to the appropriate locations in the webapp
   - Real-time logs are displayed in the web interface

## Extending

### Adding a New Strategy

To add a new build strategy:

1. Add a new constant for the strategy in `helpers.go`
2. Add a strategy checker function in `getStrategyCheckers()`
3. Add the strategy to the ordered list in `getOrderedStrategies()`
4. Implement the build commands in `GetBuildCommand()`

## License

This project is proprietary software. 