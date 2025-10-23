# pgsquash Public API

This directory contains the public API packages for pgsquash-engine.

## Available Packages

### 📦 `pkg/cli` - CLI API

Programmatically run the pgsquash CLI from Go code.

```go
import "github.com/CAPYSQUASH/pgsquash-engine/pkg/cli"

// Execute the CLI
cli.SetVersionInfo("1.0.0", "2025-10-21", "abc123")
cli.SetBrandName("capysquash")
plugins.RegisterDefault()

if err := cli.Execute(); err != nil {
    log.Fatal(err)
}
```

**Functions:**
- `Execute()` - Run the complete CLI
- `SetVersionInfo(version, buildDate, gitCommit)` - Set version metadata
- `SetBrandName(brandName)` - Set branding (pgsquash/capysquash)

**Use Cases:**
- Building custom CLI wrappers
- Creating branded distributions
- CI/CD integration

[Full Documentation](./cli/README.md)

---

### 🔧 `pkg/engine` - Migration Engine API

Use pgsquash as a library for programmatic migration consolidation.

```go
import "github.com/CAPYSQUASH/pgsquash-engine/pkg/engine"

// Squash migrations
result, err := engine.SquashDirectory("./migrations", nil)
if err != nil {
    log.Fatal(err)
}

fmt.Println(result.SQL)
```

**Functions:**
- `SquashDirectory(dir, config)` - Squash all migrations in a directory
- `SquashFiles(migrations, config)` - Squash specific files
- `AnalyzeDirectory(dir, config)` - Analyze without squashing
- `DefaultConfig()` - Get default configuration

**Use Cases:**
- Custom migration tools
- Automated migration optimization
- Integration with existing build systems
- Batch processing workflows

[Full Documentation](./engine/README.md)

---

### 🔌 `pkg/plugins` - Plugin API

Register migration plugins for third-party integrations.

```go
import "github.com/CAPYSQUASH/pgsquash-engine/pkg/plugins"

// Register all built-in plugins
if err := plugins.RegisterDefault(); err != nil {
    log.Warn("Plugin registration failed: %v", err)
}
```

**Functions:**
- `RegisterDefault()` - Register all built-in plugins (Supabase, Clerk, Prisma, Drizzle)

**Built-in Plugins:**
- **Supabase** - RLS policies, storage, auth
- **Clerk** - JWT v2 auth
- **Prisma** - ORM migrations
- **Drizzle** - ORM migrations

**Use Cases:**
- CLI initialization
- Custom plugin management
- Platform-specific features

[Full Documentation](./plugins/README.md)

---

### 📝 `pkg/utils` - Utility API

Logging and utility functions for pgsquash integration.

```go
import "github.com/CAPYSQUASH/pgsquash-engine/pkg/utils"

// Setup logging
logger := utils.NewLogger(utils.LogLevelInfo, os.Stdout)
utils.SetDefaultLogger(logger)

logger.Info("Application started")
```

**Types:**
- `Logger` - Structured logger
- `LogLevel` - Log verbosity levels

**Functions:**
- `NewLogger(level, output)` - Create logger
- `SetDefaultLogger(logger)` - Set global logger
- `GetDefaultLogger()` - Get global logger

**Log Levels:**
- `LogLevelDebug` - Detailed debugging
- `LogLevelInfo` - General information
- `LogLevelWarn` - Warnings
- `LogLevelError` - Errors
- `LogLevelFatal` - Fatal errors (exits)

**Use Cases:**
- Application logging
- Debug output
- Error reporting

[Full Documentation](./utils/README.md)

---

## Quick Comparison

| Package | Purpose | Use Case |
|---------|---------|----------|
| **cli** | Run CLI programmatically | Custom CLI wrappers, branded binaries |
| **engine** | Library API for squashing | Custom tools, automation, batch processing |
| **plugins** | Plugin registration | Platform integrations, custom plugins |
| **utils** | Logging utilities | Application logging, debugging |

## Installation

```bash
# Get the entire package
go get github.com/CAPYSQUASH/pgsquash-engine

# Import specific packages
import "github.com/CAPYSQUASH/pgsquash-engine/pkg/cli"
import "github.com/CAPYSQUASH/pgsquash-engine/pkg/engine"
import "github.com/CAPYSQUASH/pgsquash-engine/pkg/plugins"
import "github.com/CAPYSQUASH/pgsquash-engine/pkg/utils"
```

## Examples

### Example 1: CLI Wrapper

```go
package main

import (
    "os"
    "github.com/CAPYSQUASH/pgsquash-engine/pkg/cli"
    "github.com/CAPYSQUASH/pgsquash-engine/pkg/plugins"
    "github.com/CAPYSQUASH/pgsquash-engine/pkg/utils"
)

func main() {
    // Setup
    logger := utils.NewLogger(utils.LogLevelInfo, os.Stdout)
    utils.SetDefaultLogger(logger)

    // Configure
    cli.SetVersionInfo("1.0.0", "2025-10-21", "abc123")
    cli.SetBrandName("myapp")

    // Register plugins
    plugins.RegisterDefault()

    // Run CLI
    if err := cli.Execute(); err != nil {
        logger.Error("CLI failed: %v", err)
        os.Exit(1)
    }
}
```

### Example 2: Library Usage

```go
package main

import (
    "fmt"
    "log"
    "github.com/CAPYSQUASH/pgsquash-engine/pkg/engine"
)

func main() {
    // Analyze migrations
    analysis, err := engine.AnalyzeDirectory("./migrations", nil)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Found %d redundancies\n", len(analysis.Redundancies))

    // Squash if needed
    if len(analysis.Redundancies) > 0 {
        result, err := engine.SquashDirectory("./migrations", &engine.Config{
            SafetyLevel: engine.Conservative,
        })
        if err != nil {
            log.Fatal(err)
        }

        fmt.Println("Squashed SQL:")
        fmt.Println(result.SQL)
    }
}
```

### Example 3: Custom Integration

```go
package main

import (
    "log"
    "github.com/CAPYSQUASH/pgsquash-engine/pkg/engine"
    "github.com/CAPYSQUASH/pgsquash-engine/pkg/plugins"
    "github.com/CAPYSQUASH/pgsquash-engine/pkg/utils"
)

func main() {
    // Setup logging with custom prefix
    logger := utils.NewLogger(utils.LogLevelDebug, os.Stdout)
    utils.SetDefaultLogger(logger)

    logger.Info("Starting custom migration processor")

    // Register plugins for third-party detection
    if err := plugins.RegisterDefault(); err != nil {
        logger.Warn("Plugin setup: %v", err)
    }

    // Process migrations with streaming for large datasets
    config := &engine.Config{
        SafetyLevel:     engine.Standard,
        EnableStreaming: true,
        MemoryLimitMB:   512,
        Verbose:         true,
    }

    result, err := engine.SquashDirectory("./migrations", config)
    if err != nil {
        log.Fatal(err)
    }

    logger.Info("Processed %d files", result.FilesProcessed)
}
```

## Architecture

```
pkg/                      # Public API
├── cli/                  # CLI execution
│   └── api.go           # Execute(), SetVersionInfo(), SetBrandName()
├── engine/               # Library API
│   ├── api.go           # SquashDirectory(), AnalyzeDirectory()
│   └── README.md        # Full documentation
├── plugins/              # Plugin system
│   └── api.go           # RegisterDefault()
└── utils/                # Utilities
    └── api.go           # Logger, NewLogger(), etc.

internal/                 # Private implementation (not importable)
├── cli/                  # CLI implementation
├── parser/               # SQL parsing
├── squasher/             # Consolidation engine
├── tracking/             # Dependency tracking
└── ...                   # Other internal packages
```

## Why This Architecture?

### Public API (`pkg/`)
- ✅ Stable, versioned interface
- ✅ Documented for external use
- ✅ Safe to import in other projects
- ✅ Breaking changes are avoided

### Internal Implementation (`internal/`)
- ✅ Free to refactor
- ✅ Can change without breaking external code
- ✅ Not accessible from external projects
- ✅ Implementation details hidden

## Versioning

The public API follows semantic versioning:

- **Major** version: Breaking API changes
- **Minor** version: New features, backward compatible
- **Patch** version: Bug fixes

Current version: `v0.9.5-beta`

## Support

- 📖 [Documentation](https://capysquash.dev/docs)
- 🐛 [Report Issues](https://github.com/CAPYSQUASH/pgsquash-engine/issues)
- 💬 [Discussions](https://github.com/CAPYSQUASH/pgsquash-engine/discussions)
- 📧 [Email Support](mailto:support@capysquash.dev)

## License

MIT License - see LICENSE file for details
