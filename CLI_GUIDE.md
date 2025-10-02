# Transpiler4 CLI Guide

This guide provides detailed instructions on how to use the Transpiler4 CLI tool (mtcli) with practical examples.

## Installation

Before using the CLI, you need to install it:

```bash
# Install directly with Go
go install github.com/LajnaLegenden/transpiler4@latest

# Or clone the repository and build from source
git clone https://github.com/LajnaLegenden/transpiler4.git
cd transpiler4
go build -o mtcli
```

## Basic Usage

The mtcli tool provides several commands for working with packages in a monorepo:

```bash
mtcli [command] [options]
```

Available commands:
- `watch` (or `w`): Watch for changes and build packages
- `build` (or `b`): Build packages once
- `list` (or `l`): List available packages

## Watch Command

The watch command is the most commonly used command. It monitors packages for changes and rebuilds them automatically.

### Basic Usage

```bash
mtcli watch
```

This will:
1. Scan the current directory for Node.js packages
2. Present a fuzzy finder for selecting packages to watch
3. Set up file system watchers for the selected packages
4. Start a web server for viewing build logs
5. Build and deploy packages when changes are detected

### Specifying a Project Path

```bash
mtcli watch --path /path/to/your/project
# or
mtcli w -p /path/to/your/project
```

### Interactive Controls

While the watch command is running, you can interact with it using keyboard shortcuts:

#### Reselect Packages (Press 'r' or 'R')

- Stops all active watchers
- Opens the package selection menu showing all available packages
- Allows you to select packages from scratch
- Restarts watchers for the newly selected packages

This is useful when you want to completely change which packages you're watching.

#### Append Packages (Press 'a' or 'A')

- Keeps all existing watchers running
- Opens the package selection menu showing only unselected packages
- Allows you to add additional packages to the watch list
- Starts watchers only for the newly added packages

This is particularly useful when you want to add more packages without disrupting ongoing build processes.

**Example workflow:**
1. Start by watching `package-a` and `package-b`
2. Press 'a' to append packages
3. Select `package-c` from the filtered list
4. `package-a` and `package-b` continue watching uninterrupted
5. `package-c` watcher starts immediately

#### Rebuild All Packages (Press 'b' or 'B')

- Keeps all watchers running
- Triggers an immediate rebuild of all currently watched packages
- Useful after pulling changes from git or when you want to force a rebuild

This is particularly useful when you've made changes that weren't detected by the file watcher (e.g., git checkout, external file changes) or when you want to verify all packages build correctly.

**Example use cases:**
- After running `git pull` or `git checkout` to rebuild with new code
- After modifying shared configuration files
- When you want to verify all watched packages build successfully
- After changing environment variables or dependencies

### Examples

#### Watch a monorepo from within the project

```bash
# From the root of your monorepo
cd /path/to/monorepo
mtcli watch
```

#### Watch a monorepo from another location

```bash
# From any directory
mtcli watch --path /path/to/monorepo
```

#### Watch specific packages

After running the watch command, you'll be presented with a fuzzy finder:

1. Use arrow keys to navigate packages
2. Press Tab to select/deselect packages
3. Press Enter to confirm selection

Only selected packages will be watched for changes.

### Log Viewer

When you run the watch command, a log viewer is automatically started:

```
Log viewer available at http://localhost:54321
```

Open this URL in your browser to view real-time logs from all watched packages.

## Build Command

The build command builds packages once without watching for changes.

### Basic Usage

```bash
mtcli build
```

This will:
1. Scan the current directory for Node.js packages
2. Present a fuzzy finder for selecting packages to build
3. Build and deploy the selected packages once

### Specifying a Project Path

```bash
mtcli build --path /path/to/your/project
# or
mtcli b -p /path/to/your/project
```

### Examples

#### Build specific packages in a monorepo

```bash
cd /path/to/monorepo
mtcli build
# Select packages using the fuzzy finder
```

#### Build packages from another location

```bash
mtcli build --path /path/to/monorepo
```

## List Command

The list command displays all available packages in the project.

### Basic Usage

```bash
mtcli list
```

This will:
1. Scan the current directory for Node.js packages
2. Display a list of all packages with their paths and build strategies

### Specifying a Project Path

```bash
mtcli list --path /path/to/your/project
# or
mtcli l -p /path/to/your/project
```

### Examples

#### List all packages in a monorepo

```bash
cd /path/to/monorepo
mtcli list
```

#### List packages from another location

```bash
mtcli list --path /path/to/monorepo
```

## Advanced Usage

### Working with Multiple Packages

When working with multiple packages, you can select which ones to build or watch:

1. Run `mtcli watch` or `mtcli build`
2. In the fuzzy finder:
   - Type to filter packages by name
   - Use arrow keys to navigate
   - Press Tab to select/deselect packages
   - Press Enter to confirm selection

### Understanding Build Strategies

The CLI automatically detects the appropriate build strategy for each package:

1. **TRANSPILED_YARN**: For packages using rollup with Yarn
   - Executes `yarn transpile`
   - Copies build artifacts to the webapp's node_modules

2. **TRANSPILED**: For packages using rollup with other package managers
   - Executes `pnpm transpile`
   - Copies build artifacts to the webapp's node_modules

3. **TRANSPILED_LEGACY**: For packages with a build script but no rollup config
   - Executes `pnpm prepublishOnly`
   - Copies build artifacts to the webapp's node_modules

4. **AMEND_NATIVE**: For packages with an amend and lib directory
   - Copies the lib, boundaries, and amend directories to the webapp's node_modules

5. **MAKEFILE_BUILD**: For packages using a Makefile for building
   - Executes `make build`

### Handling Build Errors

If a build fails, the CLI will:
1. Display the error in the terminal
2. Show the error in the log viewer
3. Send a desktop notification
4. Continue watching for changes (in watch mode)

## Troubleshooting

### Common Issues

#### Port Already in Use

If the log viewer fails to start because the port is already in use, the CLI will automatically try a different port.

#### No Packages Found

If no packages are found, check that:
1. You're in the correct directory
2. The project has valid package.json files
3. The path provided with --path is correct

#### Build Failures

If builds are failing, check:
1. The log viewer for detailed error messages
2. That all required dependencies are installed
3. That the package has the correct build scripts defined

## Getting Help

For more information, use the help command:

```bash
mtcli --help
# or for a specific command
mtcli watch --help
``` 