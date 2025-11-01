# pkg/tui package map

## Domain Summary
- Public TUI SDK built on Bubble Tea that powers interactive migration analysis; exposes launch APIs, shared models, view types, and styling primitives for external consumers.
- Mirrors internal `internal/tui` functionality but with exported types for embedding in downstream CLIs.

## Files (alphabetical)

### api.go
- Convenience entry points (`Launch`, `LaunchWithView`, `LaunchWithModel`) for starting the TUI with paths or pre-built models.

### doc.go
- Package documentation including high-level overview and usage examples.

### model.go
- Core Bubble Tea model implementation with state machine, message handling, and view routing.
- Defines `Model`, constructors (`NewModel`), update/view functions, command helpers.

### types.go
- Shared structs/enums for TUI state (e.g., `View`, `ViewRequest`, `MigrationSummary`, `ValidationStatus`).
- Includes exported constants for view navigation.

### styles/
- Lip Gloss style definitions (colors, padding) reused across views.

### views/
- Concrete Bubble Tea views (analysis, config wizard, dependency graph, validation reports).

### examples/
- Sample programs demonstrating integration (e.g., launching analysis view directly).

### viewtypes/
- Additional typed view definitions/helpers (currently empty placeholder for future extensions).

## Subdirectories
- `examples/`
- `styles/`
- `views/`
- `viewtypes/`
