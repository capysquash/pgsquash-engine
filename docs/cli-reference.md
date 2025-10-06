# CLI Reference

Complete command-line reference for pg-squash.

## Global Options

Available for all commands:

```bash
--config, -c <path>    # Config file (default: pgsquash.config.json)
--verbose, -v          # Verbose output
--help, -h             # Show help
```

## Commands

### analyze

Analyze migrations without modifications.

```bash
pgsquash analyze [files...] [options]
```

**Options:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--progress` | bool | true | Show progress |
| `--streaming` | bool | false | Streaming mode |
| `--memory-limit` | int | 256 | Memory limit (MB) |

**Examples:**

```bash
pgsquash analyze migrations/*.sql
pgsquash analyze migrations/*.sql --verbose
pgsquash analyze migrations/*.sql --streaming --memory-limit 512
```

### squash

Consolidate migrations into optimized output.

```bash
pgsquash squash [files...] [options]
```

**Options:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--safety, -s` | string | config | Safety level (paranoid/conservative/standard/aggressive) |
| `--output, -o` | string | config | Output directory |
| `--dry-run` | bool | false | Preview without writing |
| `--progress` | bool | true | Show progress |
| `--streaming` | bool | false | Streaming mode |
| `--memory-limit` | int | 256 | Memory limit (MB) |
| `--batch-size` | int | 50 | Batch size |
| `--workers` | int | auto | Worker count |
| `--backup` | bool | false | Generate backup |
| `--rollback` | bool | false | Generate rollback scripts |
| `--transform` | bool | true | Apply SQL transformations |
| `--detect-cycles` | bool | true | DDL cycle detection |
| `--cycle-details` | bool | false | Detailed cycle info |

**Examples:**

```bash
# Basic squash
pgsquash squash migrations/*.sql --output clean/

# Production mode
pgsquash squash migrations/*.sql --safety conservative

# Preview changes
pgsquash squash migrations/*.sql --dry-run

# Development mode
pgsquash squash migrations/*.sql --safety aggressive

# Large datasets
pgsquash squash migrations/*.sql --streaming --batch-size 100

# With safety features
pgsquash squash migrations/*.sql --backup --rollback
```

### validate

Validate schema equivalence using Docker.

```bash
pgsquash validate <original-dir> <squashed-dir>
```

**Arguments:**

| Argument | Description |
|----------|-------------|
| `original-dir` | Directory with original migrations |
| `squashed-dir` | Directory with squashed migrations |

**Examples:**

```bash
pgsquash validate migrations/ clean/
pgsquash validate migrations/ clean/ --verbose
```

**Validation Methods:**

- **TWO_CONTAINERS**: Most accurate (separate containers)
- **SCHEMA_DIFF**: Fastest (single container)
- **SINGLE_CONTAINER**: Simplest (sequential apply)

### init-config

Generate default configuration file.

```bash
pgsquash init-config [options]
```

**Options:**

| Flag | Default | Description |
|------|---------|-------------|
| `--config, -c` | pgsquash.config.json | Output path |

**Examples:**

```bash
pgsquash init-config
pgsquash init-config --config custom.json
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

## Standardized Workflows

Pre-configured workflows combining multiple features.

### safe

Production-ready workflow with maximum safety.

```bash
pgsquash safe [files...] [options]
```

**Features:**
- Safety: Conservative
- Validation: TWO_CONTAINERS
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
- Validation: SCHEMA_DIFF
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

| Level | Optimization | Production | Reduction |
|-------|--------------|------------|-----------|
| paranoid | Minimal | Yes (requires DB) | 15-25% |
| conservative | CREATE+ALTER only | Yes | 20-35% |
| standard | Balanced | Testing | 35-50% |
| aggressive | Maximum | No | 50-70% |

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

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Parse error |
| 3 | Validation failed |
| 4 | Circular dependency |

## Environment Variables

| Variable | Purpose |
|----------|---------|
| `ANTHROPIC_API_KEY` | Claude AI provider |
| `OPENAI_API_KEY` | OpenAI provider |
| `PROD_DB_DSN` | Database for paranoid mode |

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
pgsquash squash migrations/*.sql --output clean/ --no-progress
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
