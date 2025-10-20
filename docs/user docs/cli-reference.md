# CLI Reference

Complete command-line reference for pgsquash's intelligent migration consolidation engine.

## Global Options

Available for all commands:

```bash
--config, -c <path>    # Config file (default: pgsquash.config.json)
--verbose, -v          # Verbose output with detailed processing logs
--quiet, -q            # Quiet mode - only show errors and final results (ideal for CI/CD)
--no-emoji             # Disable emoji characters in output (improves terminal compatibility)
--help, -h             # Show help
```

**Note:** The `--tui` flag is **not** a global option. It is only available on the `analyze` and `squash` commands. For the standalone TUI interface, use the `tui` command instead.

## Commands

### analyze

Analyze migration dependencies, redundancies, and optimization potential without making any modifications.

```bash
pgsquash analyze [files...] [options]
```

**What it does:**

- Parses all migrations using PostgreSQL's parser
- Builds complete dependency graph
- Identifies redundant operations
- Detects consolidation opportunities
- Reports potential optimization percentage

**Options:**

| Flag             | Type | Default | Description                                 |
| ---------------- | ---- | ------- | ------------------------------------------- |
| `--tui`          | bool | false   | Launch interactive TUI for analysis         |
| `--progress`     | bool | true    | Show progress with throughput stats         |
| `--streaming`    | bool | false   | Memory-efficient streaming mode (auto >100) |
| `--memory-limit` | int  | 256     | Memory limit in MB for streaming mode       |

**Examples:**

```bash
# Basic analysis
pgsquash analyze migrations/*.sql

# Verbose analysis with processing details
pgsquash analyze migrations/*.sql --verbose

# Large dataset with streaming
pgsquash analyze migrations/*.sql --streaming --memory-limit 512
```

### squash

Intelligently consolidate migrations with dependency resolution and safety validation.

```bash
pgsquash squash [files...] [options]
```

**What it does:**

- Parses migrations with parser-grade accuracy
- Resolves all dependencies automatically
- Applies safety-appropriate consolidation rules
- Handles circular foreign key dependencies
- Detects and resolves DDL cycles
- Validates every transformation
- Generates organized, production-ready output

**Options:**

| Flag              | Type   | Default | Description                                                         |
| ----------------- | ------ | ------- | ------------------------------------------------------------------- |
| `--safety, -s`    | string | config  | Safety level (paranoid/conservative/standard/aggressive)            |
| `--output, -o`    | string | config  | Output directory                                                    |
| `--dry-run`       | bool   | false   | Preview without writing                                             |
| `--explain`       | bool   | false   | Show detailed consolidation plan with reasoning (implies --dry-run) |
| `--progress`      | bool   | true    | Show progress                                                       |
| `--streaming`     | bool   | false   | Streaming mode                                                      |
| `--memory-limit`  | int    | 256     | Memory limit (MB)                                                   |
| `--batch-size`    | int    | 50      | Batch size                                                          |
| `--workers`       | int    | auto    | Worker count                                                        |
| `--backup`        | bool   | false   | Generate backup (requires prod\_db\_dsn)                            |
| `--rollback`      | bool   | false   | Generate rollback scripts (saved to rollbacks/rollback\_plans/)     |
| `--transform`     | bool   | true    | Apply SQL transformations                                           |
| `--detect-cycles` | bool   | true    | DDL cycle detection                                                 |
| `--memory-limit` | int  | 256     | Memory limit in MB for streaming mode       |
| `--tui`          | bool | false   | Launch interactive TUI for squashing        |

**Examples:**

```bash
# Basic consolidation with intelligent defaults
pgsquash squash migrations/*.sql --output clean/

# Production mode - conservative with full validation
pgsquash squash migrations/*.sql --safety conservative --backup --rollback

# Preview consolidation plan with reasoning
pgsquash squash migrations/*.sql --dry-run --explain

# Development mode - aggressive optimization
pgsquash squash migrations/*.sql --safety aggressive

# Large datasets with streaming (auto-enabled for >100 files)
pgsquash squash migrations/*.sql --streaming --batch-size 100 --memory-limit 512

# Full safety suite for production deployment
pgsquash squash migrations/*.sql --backup --rollback --safety conservative
```

### validate

Validate byte-for-byte schema equivalence using Docker containers with extension auto-detection.

```bash
pgsquash validate <original-dir> <squashed-dir>
```

**What it does:**

- Spins up PostgreSQL containers with auto-detected extensions
- Applies original migrations to container 1
- Applies consolidated migrations to container 2
- Performs byte-level schema comparison using pg\_dump
- Auto-generates auth compatibility layers (Supabase/Clerk)
- Reports exact differences if schemas don't match

**Arguments:**

| Argument       | Description                        |
| -------------- | ---------------------------------- |
| `original-dir` | Directory with original migrations |
| `squashed-dir` | Directory with squashed migrations |

**Options:**

| Flag                | Default        | Description                                                       |
| ------------------- | -------------- | ----------------------------------------------------------------- |
| `--validation-mode` | TWO\_DATABASES | Validation approach (TWO\_CONTAINERS/TWO\_DATABASES/SCHEMA\_DIFF) |

**Examples:**

```bash
pgsquash validate migrations/ clean/
pgsquash validate migrations/ clean/ --verbose
```

### tui

Launch the interactive terminal user interface (TUI) for migration analysis and squashing.

The TUI provides a visual interface for:
- Analyzing migrations and viewing lifecycle patterns
- Configuring squashing settings interactively
- Visualizing dependency graphs
- Monitoring squashing progress in real-time

```bash
pgsquash tui [migrations-dir] [options]
```

**Arguments:**

| Argument         | Description                                  |
| ---------------- | -------------------------------------------- |
| `migrations-dir` | Optional path to migrations (default: `.`)   |

**Subcommands:**

| Command   | Description                                    |
| --------- | ---------------------------------------------- |
| `analyze` | Launch TUI directly in the analysis view       |
| `config`  | Launch TUI directly in the configuration wizard|
| `deps`    | Launch TUI directly in the dependency graph view |

**Examples:**

```bash
# Launch TUI in the current directory
pgsquash tui

# Launch TUI for a specific migrations folder
pgsquash tui path/to/migrations/

# Go directly to the analysis view
pgsquash tui analyze path/to/migrations/
```

### init-config
```

**Validation Approaches:**

- **TWO\_CONTAINERS**: Most accurate (separate containers)
- **TWO\_DATABASES**: Best balance (default, single container with two databases)
- **SCHEMA\_DIFF**: Fastest (single container with schema versioning)

### init-config

Generate default configuration file.

```bash
pgsquash init-config [options]
```

**Options:**

| Flag           | Default              | Description                              |
| -------------- | -------------------- | ---------------------------------------- |
| `--config, -c` | pgsquash.config.json | Output path                              |
| `--force, -f`  | false                | Overwrite existing config file if exists |

**Examples:**

```bash
pgsquash init-config
pgsquash init-config --config custom.json
pgsquash init-config --force  # Overwrite existing config
```

### ai-test

Test AI provider integrations.

```bash
pgsquash ai-test
```

Requires environment variables:

```bash
export ANTHROPIC_API_KEY="sk-ant-..."
export OPENAI_API_KEY="sk-..."
```

### ai-demo

Demonstrate AI capabilities with sample code.

```bash
pgsquash ai-demo
```

Requires at least one AI provider configured.

### ai-fix

AI-assisted migration fixing (experimental).

```bash
pgsquash ai-fix <migration-directory> [options]
```

Automatically analyzes broken migrations, uses AI to suggest fixes, and applies them in an interactive loop until validation succeeds.

**Arguments:**

| Argument              | Description                         |
| --------------------- | ----------------------------------- |
| `migration-directory` | Directory containing SQL migrations |

**Options:**

| Flag             | Default | Description                                    |
| ---------------- | ------- | ---------------------------------------------- |
| `--max-attempts` | 5       | Maximum number of fix attempts                 |
| `--auto-apply`   | false   | Automatically apply fixes without confirmation |
| `--verbose`      | false   | Enable verbose output showing AI reasoning     |

**Requirements:**

- ANTHROPIC\_API\_KEY, OPENAI\_API\_KEY, or AZURE\_OPENAI\_ENDPOINT
- Docker (for validation)

**Features:**

- Automatic error detection
- AI-powered fix suggestions
- Interactive fix application
- Validation loop until success
- Creates backups before applying fixes

**Examples:**

```bash
# Basic usage with interactive confirmation
export ANTHROPIC_API_KEY="sk-ant-..."
pgsquash ai-fix migrations/

# Auto-apply with more attempts
pgsquash ai-fix migrations/ --max-attempts 10 --auto-apply

# Verbose mode to see AI reasoning
pgsquash ai-fix migrations/ --verbose
```

**Use Cases:**

- Fixing syntax errors automatically
- Resolving dependency conflicts
- Correcting schema inconsistencies
- Quick migration debugging

### health

Health check endpoint for container orchestration.

```bash
pgsquash health [options]
```

Returns status information in JSON format (default) or plain text.

**Options:**

| Flag         | Default | Description                                          |
| ------------ | ------- | ---------------------------------------------------- |
| `--text`     | false   | Output in plain text format instead of JSON         |
| `--detailed` | false   | Include detailed system information (CPU, memory, etc.) |

**Examples:**

```bash
# Basic health check (JSON output)
pgsquash health

# Plain text output
pgsquash health --text

# Detailed JSON with system info
pgsquash health --detailed
```

**JSON Output (default):**

```json
{
  "status": "healthy",
  "version": "0.8.5-beta",
  "docker": true,
  "timestamp": "2025-10-20T16:25:06Z"
}
```

**Detailed JSON Output:**

```json
{
  "status": "healthy",
  "version": "0.8.5-beta",
  "timestamp": "2025-10-20T16:25:06Z",
  "system": {
    "os": "darwin",
    "arch": "arm64",
    "go_version": "go1.25.3",
    "num_cpu": 8,
    "num_goroutines": 5
  },
  "docker": {
    "available": true
  }
}
```

**Use Cases:**

- Kubernetes liveness/readiness probes
- Docker health checks
- CI/CD pipeline validation
- Monitoring systems

**Example in Docker Compose:**

```yaml
services:
  pgsquash:
    image: pgsquash:latest
    healthcheck:
      test: ["CMD", "pgsquash", "health"]
      interval: 30s
      timeout: 10s
      retries: 3
```

## Standardized Workflows

Pre-configured workflows combining multiple features.

### safe

Production-ready workflow with maximum safety.

```bash
pgsquash safe [files...] [options]
```

**Features:**

- Safety: Conservative
- Validation: TWO\_CONTAINERS
- Backup: Enabled
- Rollback: Enabled
- Auto SQL Fix: Disabled

**Example:**

```bash
pgsquash safe migrations/*.sql --output production/
```

### fast

Development-optimized workflow.

```bash
pgsquash fast [files...] [options]
```

**Features:**

- Safety: Standard
- Validation: SCHEMA\_DIFF
- Streaming: Enabled
- DDL Cycles: Detected
- SQL Transform: Enabled

**Example:**

```bash
pgsquash fast migrations/*.sql --output dev/
```

### analyze-deep

Comprehensive analysis without modifications.

```bash
pgsquash analyze-deep [files...] [options]
```

**Features:**

- Dependency graph analysis
- DDL cycle detection (all types)
- AI-powered semantic analysis
- Auth pattern detection
- Dead code identification
- Performance suggestions

**Example:**

```bash
pgsquash analyze-deep migrations/*.sql
```

## Safety Levels

Set with `--safety` flag or config file.

| Level        | Optimization      | Production        | Reduction |
| ------------ | ----------------- | ----------------- | --------- |
| paranoid     | Minimal           | Yes (requires DB) | 15-25%    |
| conservative | CREATE+ALTER only | Yes               | 20-35%    |
| standard     | Balanced          | Testing           | 35-50%    |
| aggressive   | Maximum           | No                | 50-70%    |

**Examples:**

```bash
pgsquash squash migrations/*.sql --safety conservative
pgsquash squash migrations/*.sql --safety aggressive
```

[See Safety Levels Guide](safety-levels.md)

## Output Organization

Squashed migrations are organized by category:

```sql
-- === EXTENSIONS ===
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- === FOUNDATION ===
CREATE TABLE users (...);

-- === CONSTRAINTS ===
ALTER TABLE users ADD CONSTRAINT ...;

-- === INDEXES ===
CREATE INDEX idx_users_email ON users(email);

-- === FUNCTIONS ===
CREATE FUNCTION active_users_count() ...;

-- === TRIGGERS ===
CREATE TRIGGER update_timestamp ...;

-- === SECURITY ===
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
CREATE POLICY users_select ...;

-- === DATA ===
INSERT INTO users VALUES (...);
```

## Performance Options

### Streaming Mode

For large migration sets (500+ files):

```bash
pgsquash squash migrations/*.sql \
  --streaming \
  --memory-limit 512 \
  --batch-size 100 \
  --workers 8
```

### Parallel Processing

Auto-detected based on CPU cores, or set manually:

```bash
pgsquash squash migrations/*.sql --workers 4
```

## Rollback Scripts

When using `--rollback`, pgsquash generates rollback plans for recovery.

**Location**: `rollbacks/rollback_plans/`

**Format**: JSON files with timestamp naming:

```
rollbacks/rollback_plans/rollback_1759854152_squash_1759854152.json
```

**Usage:**

```bash
# Generate rollback scripts
pgsquash squash migrations/*.sql --rollback --output clean/

# Rollback files saved to rollbacks/rollback_plans/
ls rollbacks/rollback_plans/
```

**Rollback Plan Contents:**

- Original schema state
- Squashed schema state
- Reverse operations
- Dependencies and ordering
- Timestamp metadata

**Use Cases:**

- Emergency rollback planning
- Audit trails
- Migration recovery
- Change documentation

## Exit Codes

| Code | Meaning             |
| ---- | ------------------- |
| 0    | Success             |
| 1    | General error       |
| 2    | Parse error         |
| 3    | Validation failed   |
| 4    | Circular dependency |

## Environment Variables

| Variable            | Purpose                    |
| ------------------- | -------------------------- |
| `ANTHROPIC_API_KEY` | Claude AI provider         |
| `OPENAI_API_KEY`    | OpenAI provider            |
| `PROD_DB_DSN`       | Database for paranoid mode |

Example:

```bash
export ANTHROPIC_API_KEY="sk-ant-..."
export PROD_DB_DSN="postgres://user:pass@localhost/db"
```

## Configuration File

Override defaults with `pgsquash.config.json`:

```json
{
  "safety_level": "standard",
  "output": {
    "directory": "squashed"
  },
  "performance": {
    "streaming": true,
    "parallel_processing": true
  }
}
```

[See Configuration Reference](configuration.md)

## Common Patterns

### CI/CD Integration

```bash
#!/bin/bash
set -e

pgsquash analyze migrations/*.sql
pgsquash squash migrations/*.sql --output clean/ --progress=false
pgsquash validate migrations/ clean/

git add clean/
git commit -m "chore: squash migrations [skip ci]"
```

### Supabase Projects

```bash
pgsquash squash supabase/migrations/*.sql --safety standard
```

### Large Datasets

```bash
pgsquash squash migrations/*.sql \
  --streaming \
  --memory-limit 512 \
  --batch-size 100
```

### AI-Enhanced Squashing

```bash
export ANTHROPIC_API_KEY="sk-ant-..."
pgsquash analyze-deep migrations/*.sql
pgsquash fast migrations/*.sql --output optimized/
```

## Further Reading

- [Quickstart](quickstart.md) - Get started in 5 minutes
- [Configuration](configuration.md) - Full config reference
- [Safety Levels](safety-levels.md) - Detailed safety comparison
- [Troubleshooting](troubleshooting.md) - Common issues
