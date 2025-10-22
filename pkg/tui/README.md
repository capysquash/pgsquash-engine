# TUI Package - Public API for pgsquash Terminal User Interface

The `pkg/tui` package provides a public API for integrating the pgsquash Terminal User Interface (TUI) into external applications like capysquash-cli.

## Overview

This package exports a clean, stable API for the interactive terminal interface, allowing:
- ✅ Interactive migration analysis
- ✅ Real-time dependency visualization
- ✅ Configuration wizard
- ✅ Progress monitoring
- ✅ Keyboard-driven navigation

The actual TUI implementation remains in `internal/tui/` (private), while this package provides a public wrapper that external tools can depend on.

## Installation

```bash
go get github.com/CAPYSQUASH/pgsquash-engine/pkg/tui
```

## Quick Start

### Simple Usage

The easiest way to launch the TUI:

```go
package main

import (
    "log"
    "github.com/CAPYSQUASH/pgsquash-engine/pkg/tui"
)

func main() {
    if err := tui.Launch("./migrations", "pgsquash.config.json"); err != nil {
        log.Fatal(err)
    }
}
```

### With Options

For more control over the TUI behavior:

```go
package main

import (
    "log"
    "os"
    "github.com/CAPYSQUASH/pgsquash-engine/pkg/tui"
)

func main() {
    // Verify migration directory exists
    if _, err := os.Stat("./migrations"); err != nil {
        log.Fatalf("Migration directory not found: %v", err)
    }

    // Create TUI with custom options
    model := tui.New(tui.Options{
        MigrationDir: "./migrations",
        ConfigPath:   ".pgsquash.config.json",
        AltScreen:    true,
    })

    // Run the TUI
    if err := model.Run(); err != nil {
        log.Fatalf("TUI error: %v", err)
    }
}
```

### Direct View Navigation

Launch directly into a specific view:

```go
package main

import (
    "log"
    "github.com/CAPYSQUASH/pgsquash-engine/pkg/tui"
)

func main() {
    // Launch directly into analysis view
    err := tui.LaunchWithView(
        "./migrations",
        "",
        tui.ViewAnalysis,
    )
    if err != nil {
        log.Fatal(err)
    }
}
```

## Available Views

| View | Constant | Description |
|------|----------|-------------|
| **Dashboard** | `tui.ViewDashboard` | Main dashboard with quick actions and overview |
| **Analysis** | `tui.ViewAnalysis` | Detailed migration analysis with lifecycle patterns |
| **Configuration** | `tui.ViewConfig` | Interactive configuration wizard |
| **Dependency Graph** | `tui.ViewDependencyGraph` | Visual dependency graph visualization |
| **Progress** | `tui.ViewProgress` | Real-time operation progress tracking |
| **Help** | `tui.ViewHelp` | Keyboard shortcuts and help documentation |

## API Reference

### Types

#### `Model`
```go
type Model struct { /* opaque */ }
```
Represents the TUI application model. This is an opaque type wrapping the internal implementation.

#### `Options`
```go
type Options struct {
    MigrationDir string   // Directory containing migration files
    ConfigPath   string   // Path to configuration file
    InitialView  ViewType // Optional: which view to show on startup
    AltScreen    bool     // Enable alternate screen buffer (recommended)
}
```
Configuration options for creating a new TUI instance.

#### `ViewType`
```go
type ViewType int
```
Represents different views in the TUI. Use the exported constants (`ViewDashboard`, `ViewAnalysis`, etc.).

### Functions

#### `New(opts Options) *Model`
Creates a new TUI model with the given options.

```go
model := tui.New(tui.Options{
    MigrationDir: "./migrations",
    ConfigPath:   "pgsquash.config.json",
    AltScreen:    true,
})
```

#### `Launch(migrationDir, configPath string) error`
Convenience function that creates and runs a TUI in one step with default settings.

```go
if err := tui.Launch("./migrations", "pgsquash.config.json"); err != nil {
    log.Fatal(err)
}
```

#### `LaunchWithView(migrationDir, configPath string, view ViewType) error`
Convenience function that creates and runs a TUI, immediately navigating to the specified view.

```go
// Launch directly into analysis view
if err := tui.LaunchWithView("./migrations", "", tui.ViewAnalysis); err != nil {
    log.Fatal(err)
}
```

### Methods

#### `(*Model) Run() error`
Starts the TUI and blocks until the user exits. Returns an error if the TUI encounters a fatal error.

```go
model := tui.New(tui.Options{
    MigrationDir: "./migrations",
    AltScreen:    true,
})
if err := model.Run(); err != nil {
    log.Fatal(err)
}
```

#### `(*Model) RunWithView(view ViewType) error`
Starts the TUI and immediately navigates to the specified view.

```go
model := tui.New(tui.Options{
    MigrationDir: "./migrations",
    AltScreen:    true,
})
// Launch directly into analysis view
if err := model.RunWithView(tui.ViewAnalysis); err != nil {
    log.Fatal(err)
}
```

#### `(*Model) RunWithOptions(opts ...tea.ProgramOption) error`
Starts the TUI with custom Bubbletea program options for advanced users.

```go
model := tui.New(tui.Options{
    MigrationDir: "./migrations",
})

opts := []tea.ProgramOption{
    tea.WithAltScreen(),
    tea.WithMouseCellMotion(),
}

if err := model.RunWithOptions(opts...); err != nil {
    log.Fatal(err)
}
```

## Integration Examples

### Cobra CLI Integration

```go
package main

import (
    "fmt"
    "os"
    "github.com/CAPYSQUASH/pgsquash-engine/pkg/tui"
    "github.com/spf13/cobra"
)

var tuiCmd = &cobra.Command{
    Use:   "tui [migrations-dir]",
    Short: "Launch interactive TUI",
    RunE:  runTUI,
}

func runTUI(cmd *cobra.Command, args []string) error {
    migrationDir := "."
    if len(args) > 0 {
        migrationDir = args[0]
    }

    // Verify directory exists
    if _, err := os.Stat(migrationDir); err != nil {
        if os.IsNotExist(err) {
            return fmt.Errorf("migration directory does not exist: %s", migrationDir)
        }
        return fmt.Errorf("failed to access migration directory: %w", err)
    }

    configPath, _ := cmd.Flags().GetString("config")
    return tui.Launch(migrationDir, configPath)
}
```

### Fallback to Non-Interactive

```go
func runAnalysis(migrationDir string) error {
    // Try interactive TUI first
    if isTTY() {
        if err := tui.Launch(migrationDir, ""); err == nil {
            return nil
        }
    }
    
    // Fallback to non-interactive mode
    return runNonInteractiveAnalysis(migrationDir)
}

func isTTY() bool {
    stat, _ := os.Stdout.Stat()
    return (stat.Mode() & os.ModeCharDevice) != 0
}
```

### With Error Recovery

```go
func launchTUIWithRecovery(migrationDir string) error {
    defer func() {
        if r := recover(); r != nil {
            log.Printf("TUI panic recovered: %v", r)
            log.Println("Falling back to non-interactive mode...")
            runNonInteractiveMode()
        }
    }()

    return tui.Launch(migrationDir, "pgsquash.config.json")
}
```

## Keyboard Shortcuts

### Global Shortcuts (All Views)

| Key | Action |
|-----|--------|
| `q`, `Ctrl+C` | Quit the TUI |
| `?` | Toggle help view |
| `ESC` | Return to dashboard |
| `Tab` | Cycle focus between elements |
| `↑`/`↓` or `j`/`k` | Navigate up/down |
| `Enter` | Select/confirm |

### View-Specific Shortcuts

View-specific keyboard shortcuts are displayed in the help view (press `?` from any view).

## Terminal Requirements

The TUI requires:
- ✅ Terminal with ANSI color support
- ✅ Minimum terminal size: 80x24
- ✅ UTF-8 encoding support

The TUI automatically detects terminal capabilities and adjusts rendering for the best experience.

## Error Handling

The TUI returns errors for:
- Missing or inaccessible migration directories
- Invalid configuration files
- Terminal compatibility issues
- Internal TUI failures

Always check and handle errors appropriately:

```go
if err := tui.Launch("./migrations", ""); err != nil {
    log.Printf("TUI failed: %v", err)
    // Fallback to non-interactive mode
    runNonInteractive()
}
```

## Configuration

The TUI reads configuration from the specified config file (JSON format). If no config file exists, the TUI will use sensible defaults and offer to create one through the configuration wizard.

Example configuration structure:

```json
{
    "safetyLevel": "standard",
    "autoValidate": true,
    "excludePatterns": ["*_test.sql"],
    "customRules": []
}
```

## Thread Safety

⚠️ **Important**: The TUI `Model` is not thread-safe. Each instance should be used from a single goroutine. The underlying Bubbletea framework handles concurrency internally through its message passing system.

## Dependencies

This package uses:
- `github.com/charmbracelet/bubbletea` - TUI framework
- `github.com/charmbracelet/lipgloss` - Styling and layout

These dependencies are handled automatically through the internal implementation and are not exposed in the public API.

## Examples

See the [examples](./examples/) directory for complete, runnable examples:

- **[simple](./examples/simple/main.go)** - Basic TUI usage
- **[advanced](./examples/advanced/main.go)** - Cobra CLI integration with subcommands

Run examples:

```bash
# Simple example
go run pkg/tui/examples/simple/main.go ./migrations

# Advanced example (Cobra integration)
go run pkg/tui/examples/advanced/main.go tui ./migrations
go run pkg/tui/examples/advanced/main.go tui analyze ./migrations
```

## Architecture

```
pkg/tui/              # Public API (this package)
├── api.go            # Main API implementation
├── doc.go            # Package documentation
└── examples/         # Usage examples

internal/tui/         # Private implementation
├── model.go          # TUI application model
├── types.go          # Type definitions
├── styles/           # Visual styling
├── views/            # Different TUI views
└── viewtypes/        # View type definitions
```

The public API (`pkg/tui`) wraps the internal implementation (`internal/tui`) to provide a stable interface that external tools can depend on without being affected by internal refactoring.

## Migration from Internal Package

If you were previously importing `internal/tui` directly, update your imports:

**Before:**
```go
import "github.com/CAPYSQUASH/pgsquash-engine/internal/tui"
```

**After:**
```go
import "github.com/CAPYSQUASH/pgsquash-engine/pkg/tui"
```

**Changes:**
- Use `tui.New()` instead of `tui.NewModel()`
- Use `model.Run()` instead of manually creating a Bubbletea program
- View types are exported with `tui.View` prefix (e.g., `tui.ViewDashboard`)

## Support

- 📖 [Full Documentation](https://pkg.go.dev/github.com/CAPYSQUASH/pgsquash-engine/pkg/tui)
- 🐛 [Report Issues](https://github.com/CAPYSQUASH/pgsquash-engine/issues)
- 💬 [Discussions](https://github.com/CAPYSQUASH/pgsquash-engine/discussions)

## License

MIT License - see [LICENSE](../../LICENSE) for details.