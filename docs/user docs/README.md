# pgsquash Documentation

Welcome to the pgsquash Engine documentation. pgsquash is an intelligent PostgreSQL migration consolidation engine that uses parser-grade accuracy and dependency-aware processing to safely reorganize your migration history.

## Getting Started

- [Quickstart](quickstart.md) - get up and running in 5 minutes
- [CLI Reference](cli-reference.md) - complete command reference with all options
- [Configuration](configuration.md) - detailed config file options
- [Safety Levels](safety-levels.md) - choosing the right consolidation strategy
- [Error Reference](error-reference.md) - error codes, meanings, and solutions
- [TUI Guide](tui-guide.md) - using the interactive interface
- [AI Features](ai-features.md) - optional AI analysis
- [AI Validation Usage](ai-validation-usage.md) - AI-powered validation
- [Pattern Detection](patterns.md) - understanding pattern detection and consolidation
- [Troubleshooting](troubleshooting.md) - fixing common issues

## Integration & Automation

- [GitHub Webhooks](github-webhooks.md) - automated PR analysis and consolidation
- [Environment Variables](environment-variables.md) - configuration via environment

## Advanced Topics

- [Plugin Development](plugin-development.md) - creating custom plugins for auth/ORM frameworks
- [Advanced Features](advanced-features.md) - SQL builder, metadata, tracking, streaming

## Technical

- [Architecture](architecture.md) - how the engine works

## Quick reference

## Quick Example

```bash
# Analyze your migrations with dependency tracking
pgsquash analyze migrations/*.sql

# Preview intelligent consolidation plan
pgsquash squash migrations/*.sql --dry-run

# Consolidate with dependency resolution and optimization
pgsquash squash migrations/*.sql --output clean/

# Validate schemas match byte-for-byte
pgsquash validate migrations/ clean/

# Generate config file with safety presets
pgsquash init-config
```

## Safety modes

| Mode         | Use case            | Approach         |
| ------------ | ------------------- | ---------------- |
| paranoid     | Production critical | Minimal changes  |
| conservative | Production          | Safe merges only |
| standard     | Staging/testing     | Balanced         |
| aggressive   | Development         | Maximum cleanup  |

---

**Version**: 0.8.5 (beta)
