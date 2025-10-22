# pgsquash Interactive TUI Guide

## Overview

The pgsquash Interactive TUI (Terminal User Interface) provides a visual, menu-driven interface for analyzing and squashing PostgreSQL migrations. Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lipgloss](https://github.com/charmbracelet/lipgloss), the TUI offers an intuitive alternative to command-line flags.

## Features

- **Dashboard**: Overview of your migration files and configuration
- **Analysis View**: Deep dive into migration patterns and optimization opportunities
- **Configuration Wizard**: Interactive configuration editing
- **Dependency Graph**: Visualize object dependencies
- **Progress Monitor**: Real-time squashing progress
- **Context-sensitive Help**: Built-in keyboard shortcuts and guidance

## Launching the TUI

### Basic Usage

```bash
# Launch TUI in current directory
pgsquash tui

# Launch TUI for specific migration directory
pgsquash tui migrations/

# Launch with custom config
pgsquash tui migrations/ --config custom.config.json
```

### Direct View Access

Jump directly to a specific view:

```bash
# Open directly in analysis view
pgsquash tui analyze migrations/

# Open configuration wizard
pgsquash tui config

# Open dependency graph view
pgsquash tui deps migrations/
```

## Views

### 1. Dashboard View

The main entry point of the TUI. Displays:

- **Migration Overview**: Total files, size, detected plugins
- **Configuration Status**: Current safety level and config file status
- **Action Menu**: Quick access to all TUI features

**Navigation:**

- `↑/↓` or `j/k`: Navigate menu items
- `Enter` or `Space`: Select action
- `ESC`: Return to dashboard (from any view)
- `?`: Toggle help screen
- `q` or `Ctrl+C`: Quit application

### 2. Analysis View

Comprehensive migration analysis with multiple tabs:

#### Tabs:

1. **Overview**: Statistics summary
   - Total migrations and objects
   - Optimization potential percentage
   - File reduction estimates

2. **Lifecycle Patterns**: Object evolution tracking
   - CREATE → ALTER → DROP sequences
   - Optimizable patterns highlighted
   - Warnings for problematic cycles

3. **Dependencies**: Object relationships
   - Dependency graph visualization
   - Source → Target mapping
   - Circular dependency detection

4. **Issues**: Errors and warnings
   - Parse errors
   - Consistency warnings
   - Actionable recommendations

**Navigation:**

- `←/→` or `h/l`: Switch tabs
- `↑/↓` or `j/k`: Scroll content
- `r`: Refresh analysis
- `ESC`: Return to dashboard

### 3. Configuration Wizard

Interactive configuration editing with real-time preview:

**Configurable Options:**

- Safety Level: `paranoid`, `conservative`, `standard`, `aggressive`
- Output Format: `organized`, `sequential`, `minimal`
- Boolean Toggles:
  - Preserve Comments
  - Add Consolidation Comments
  - Consolidate CREATE + ALTER
  - Remove DROP-CREATE Cycles
  - Parallel Processing
  - Show Progress
- Third-Party Integrations:
  - Supabase Integration
  - Clerk Integration

**Navigation:**

- `↑/↓` or `j/k`: Navigate fields
- `Enter`: Edit selected field
- `←/→` or `h/l`: Change value (while editing)
- `ESC`: Cancel editing
- `s` or `Ctrl+S`: Save configuration
- `d`: Reset to defaults

**Visual Indicators:**

- `●` Yellow indicator: Unsaved changes
- `☑` Green message: Configuration saved
- Selected items highlighted in purple
- Editing mode shows `◄ value ►` with border

### 4. Dependency Graph View

Visualize and explore object dependencies:

**Features:**

- Sorted object list with dependency counts
- Forward dependencies (what this object depends on)
- Reverse dependencies (what depends on this object)
- Interactive navigation

**Navigation:**

- `↑/↓` or `j/k`: Navigate objects
- `r`: Toggle reverse dependencies
- `f`: Refresh graph
- `ESC`: Return to dashboard

**Display:**

- Objects shown with dependency count in parentheses
- Selected object highlighted
- Detailed dependency list for selected object
- Scroll indicator for large graphs

### 5. Progress View

Real-time monitoring of squashing operations:

**Displays:**

- Current phase (Parsing, Tracking, Analysis, Consolidation, Generation)
- Progress bar with percentage
- Elapsed time
- Recent activity log (last 10 events)

**Completion Screen:**

- Original vs squashed file count
- Reduction percentage
- Space saved
- Objects optimized
- Total duration
- Output path

**Navigation:**

- Automatically displayed when squashing starts
- `Enter`: Return to dashboard (when complete)
- `ESC`: Cancel operation (returns to dashboard)

### 6. Help View

Comprehensive keyboard shortcuts and usage information:

**Sections:**

- Global shortcuts (available everywhere)
- View-specific shortcuts
- About pgsquash

**Navigation:**

- `↑/↓` or `j/k`: Scroll help content
- `ESC` or `?`: Close help

## Keyboard Shortcuts Reference

### Global (Available in all views)

| Key           | Action              |
| ------------- | ------------------- |
| `q`, `Ctrl+C` | Quit application    |
| `ESC`         | Return to dashboard |
| `?`           | Toggle help screen  |

### Dashboard

| Key              | Action        |
| ---------------- | ------------- |
| `↑/↓`, `j/k`     | Navigate menu |
| `Enter`, `Space` | Select item   |

### Analysis View

| Key          | Action           |
| ------------ | ---------------- |
| `←/→`, `h/l` | Switch tabs      |
| `↑/↓`, `j/k` | Scroll content   |
| `r`          | Refresh analysis |

### Configuration Wizard

| Key           | Action                 |
| ------------- | ---------------------- |
| `↑/↓`, `j/k`  | Navigate fields        |
| `Enter`       | Edit field             |
| `←/→`, `h/l`  | Change value (editing) |
| `ESC`         | Cancel editing         |
| `s`, `Ctrl+S` | Save configuration     |
| `d`           | Reset to defaults      |

### Dependency Graph

| Key          | Action              |
| ------------ | ------------------- |
| `↑/↓`, `j/k` | Navigate objects    |
| `r`          | Toggle reverse deps |
| `f`          | Refresh graph       |

## Color Scheme

The TUI uses a carefully designed color palette for clarity:

- **Purple** (#7C3AED): Primary actions, selections
- **Cyan** (#06B6D4): Secondary information, headings
- **Green** (#10B981): Success messages, optimizable items
- **Orange** (#F59E0B): Warnings, unsaved changes
- **Red** (#EF4444): Errors, critical warnings
- **Gray** (#6B7280): Muted text, descriptions

## Tips & Best Practices

### 1. Start with Analysis

Before squashing, always review the Analysis view:

- Check the Issues tab for errors
- Review lifecycle patterns for optimization opportunities
- Verify dependency graph for circular dependencies

### 2. Use Configuration Wizard

Instead of manually editing `pgsquash.config.json`:

- Use the TUI Configuration Wizard for guided setup
- See real-time preview of option changes
- Avoid syntax errors with validated input

### 3. Monitor Progress

When squashing large migration sets:

- Watch the Progress view for phase information
- Review activity log for warnings
- Check completion statistics for optimization metrics

### 4. Leverage Direct View Access

Save time by jumping directly to needed views:

```bash
# Quick analysis without dashboard navigation
pgsquash tui analyze migrations/

# Jump straight to config editing
pgsquash tui config
```

### 5. Keyboard-First Navigation

The TUI is optimized for keyboard use:

- Learn Vim-style navigation (`hjkl`)
- Use `ESC` as universal "back" button
- Press `?` anytime for context-sensitive help

## Integration with CLI

The TUI complements the command-line interface:

```bash
# TUI for interactive exploration
pgsquash tui analyze migrations/

# CLI for automation and scripting
pgsquash analyze migrations/*.sql --progress

# TUI for configuration
pgsquash tui config

# CLI for actual squashing (after TUI analysis)
pgsquash squash migrations/*.sql --output clean/
```

## Troubleshooting

### TUI Won't Launch

**Issue**: Terminal compatibility

**Solution**:

```bash
# Use no-emoji mode for basic terminals
pgsquash tui --no-emoji migrations/

# Check terminal supports 256 colors
echo $TERM
# Should show: xterm-256color or similar
```

### Garbled Display

**Issue**: Terminal size too small

**Solution**:

- Resize terminal to at least 80x24 characters
- Maximize terminal window
- TUI automatically adjusts to window size changes

### Missing Data in Views

**Issue**: No migrations found

**Solution**:

```bash
# Verify migration directory exists
ls -la migrations/

# Ensure migrations have .sql extension
find migrations/ -name "*.sql"

# Launch TUI with correct path
pgsquash tui migrations/
```

### Configuration Not Saving

**Issue**: Permission denied

**Solution**:

```bash
# Check directory permissions
ls -la .

# Ensure write access to config file
chmod 644 pgsquash.config.json
```

## Advanced Features

### Custom Themes (Future)

While not yet implemented, the TUI architecture supports:

- Custom color schemes
- Light/dark mode toggle
- Accessibility options

### Real-Time Updates (Future Enhancement)

Planned features:

- Live file watching during analysis
- Streaming progress updates
- Background task monitoring

## Examples

### Complete Workflow Example

```bash
# 1. Launch TUI in migration directory
cd my-project
pgsquash tui migrations/

# 2. Navigate to Analysis view (Enter on "Analyze Migrations")
#    - Review Overview tab for statistics
#    - Check Lifecycle Patterns tab for optimizations
#    - Verify Dependencies tab for circular refs
#    - Examine Issues tab for errors

# 3. Go back to Dashboard (ESC)
#    Navigate to Configuration (Enter on "Configure Settings")
#    - Set Safety Level to "conservative"
#    - Enable "Add Consolidation Comments"
#    - Save with 's'

# 4. Return to Dashboard (ESC)
#    Select "Squash Migrations" (Enter)
#    - Monitor Progress view
#    - Wait for completion
#    - Review statistics

# 5. Exit TUI (q or Ctrl+C)
#    Verify output
ls -la squashed/

# 6. Optionally validate
pgsquash validate migrations/ squashed/
```

### Quick Analysis Workflow

```bash
# Direct launch into analysis
pgsquash tui analyze migrations/

# Review tabs:
# - Press → to navigate to Lifecycle Patterns
# - Use ↓ to scroll through patterns
# - Press → again for Dependencies
# - Final → for Issues

# Return to dashboard or quit
# ESC twice to exit
```

### Configuration-Only Workflow

```bash
# Open config wizard directly
pgsquash tui config

# Edit settings:
# - ↓ to "Safety Level"
# - Enter to edit
# - → to change to "aggressive"
# - Enter to confirm
# - s to save

# Exit
# q or Ctrl+C
```

## See Also

- [CLI Reference](cli-reference.md): Command-line interface documentation
- [Configuration](configuration.md): Detailed config file documentation
- [Safety Levels](safety-levels.md): Understanding safety levels
- [Architecture](architecture.md): System design and components

## Version Information

**TUI Version**: 0.8.5-beta
**Bubble Tea**: 1.3.10
**Lipgloss**: 1.1.0

## Feedback & Contributions

Found a bug or have a feature request for the TUI?

- GitHub Issues: <https://github.com/CAPYSQUASH/pgsquash-engine/issues>
- Documentation: See `docs/` directory
- Examples: See `examples/` directory
