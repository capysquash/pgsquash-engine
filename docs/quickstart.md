# Quickstart Guide

Get pg-squash running in 5 minutes.

## Prerequisites

- Go 1.25.1+
- PostgreSQL migrations in SQL format
- Docker (optional, for validation)

## Installation

### Option 1: Build from Source

```bash
git clone https://github.com/capysquash/pg-squash-engine
cd pg-squash-engine
go build -o pgsquash cmd/pgsquash/main.go
./pgsquash --version
```

### Option 2: Go Install

```bash
go install github.com/capysquash/pg-squash-engine/cmd/pgsquash@latest
pgsquash --version
```

## 5-Minute Workflow

### 1. Analyze Migrations

```bash
pgsquash analyze migrations/*.sql
```

Output shows:
- Number of migration files and statements
- Objects by type (tables, indexes, functions, etc.)
- Redundancy opportunities
- Potential statement reduction

### 2. Preview Squashing

```bash
pgsquash squash migrations/*.sql --dry-run
```

Shows what will be consolidated without making changes.

### 3. Squash Migrations

```bash
pgsquash squash migrations/*.sql --output clean/
```

Creates `clean/001_squashed_migration.sql` with:
- Consolidated CREATE + ALTER statements
- Proper dependency ordering
- Categorized SQL (extensions → tables → constraints → indexes → security → data)

### 4. Validate Results

```bash
pgsquash validate migrations/ clean/
```

Uses Docker to verify schema equivalence:
- Applies original migrations to container 1
- Applies squashed migrations to container 2
- Compares schemas with pg_dump

**Success:** Schemas are equivalent ✓
**Failure:** Shows diff with specific differences

## Next Steps

### Use Standardized Workflows

```bash
# Production: conservative mode, full validation, backups
pgsquash safe migrations/*.sql --output production/

# Development: balanced optimization, fast validation
pgsquash fast migrations/*.sql --output dev/

# Analysis only: no modifications
pgsquash analyze-deep migrations/*.sql
```

### Configure Safety Level

Create `pgsquash.config.json`:

```bash
pgsquash init-config
```

Edit safety level:

```json
{
  "safety_level": "standard"  // paranoid, conservative, standard, aggressive
}
```

| Level | Use Case | File Reduction |
|-------|----------|----------------|
| paranoid | Critical production | 15-25% |
| conservative | Production | 20-35% |
| standard | Staging/Testing | 35-50% |
| aggressive | Development | 50-70% |

[See Safety Levels Guide](safety-levels.md)

### Enable AI Features (Optional)

```bash
export ANTHROPIC_API_KEY="sk-ant-..."
pgsquash ai-test  # Test provider
pgsquash analyze-deep migrations/*.sql  # AI-powered analysis
```

AI features provide:
- Semantic function equivalence detection
- Dead code identification
- Performance optimization suggestions

[See AI Features Guide](ai-features.md)

## Common Patterns

### Supabase Projects

```bash
pgsquash squash supabase/migrations/*.sql --safety standard
```

Special handling for:
- RLS policies
- Auth schema modifications
- Storage bucket policies

### Large Migration Sets (500+ files)

```bash
pgsquash squash migrations/*.sql --streaming --memory-limit 512
```

Enables memory-efficient batch processing.

### CI/CD Integration

```bash
#!/bin/bash
pgsquash analyze migrations/*.sql || exit 1
pgsquash squash migrations/*.sql --output clean/ || exit 1
pgsquash validate migrations/ clean/ || exit 1
git add clean/
git commit -m "chore: squash migrations"
```

## Troubleshooting

### "Failed to parse migration"

Check for unsupported PostgreSQL syntax. Use `--verbose` to see details:

```bash
pgsquash analyze migrations/*.sql --verbose
```

### "Circular dependency detected"

Enable cycle detection:

```bash
pgsquash squash migrations/*.sql --detect-cycles --cycle-details
```

### "Schema validation failed"

Use more conservative safety level:

```bash
pgsquash squash migrations/*.sql --safety conservative
```

Review differences:

```bash
pgsquash validate migrations/ clean/ --verbose
```

## Learn More

- [CLI Reference](cli-reference.md) - All commands and options
- [Configuration](configuration.md) - Full config reference
- [Safety Levels](safety-levels.md) - Detailed safety comparison
- [Troubleshooting](troubleshooting.md) - Common issues
- [Architecture](architecture.md) - How it works
