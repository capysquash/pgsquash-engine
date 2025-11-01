# internal/fileutil package map

## Domain Summary
- Provides opinionated file I/O helpers (write, append, backup) with consistent permissions and structured error reporting.
- Used across the engine to persist generated SQL, configs, and other artifacts while ensuring directories exist and backups are created when needed.
- Normalizes filesystem interactions for CLI/engine so all failures return `internal/errors.StructuredError` with validation codes and helpful suggestions.

## Files (alphabetical)

### write.go
- **Purpose**: Core file utility implementations including safe writes, backups, append operations, and existence checks.
- **Functions**
  - `WriteSQL`: Convenience wrapper for writing SQL content with `0644` permissions.
  - `WriteConfig`: Writes configuration blobs with standard permissions.
  - `WriteFile`: Low-level writer that ensures parent directories exist and wraps errors with structured metadata.
  - `WriteWithBackup`: Creates `<file>.backup` before overwriting existing files.
  - `AppendToFile`: Appends data (inserting newline separator when appropriate).
  - `EnsureDir`: Creates directories recursively with `0755` permissions.
  - `FileExists`: Checks for existing non-directory file.
  - `DirExists`: Checks for existing directory path.

## Subdirectories
- _None._
