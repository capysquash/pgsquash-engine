# pgsquash

**The PostgreSQL migration consolidation engine.** Intelligently reorganizes your migration history into clean, production-ready SQL—without breaking anything.

**Current version:** 0.9.0 (Beta)

> **Note:** pgsquash is the engine that powers capysquash, you can use either `pgsquash` or `capysquash` as the command - they're identical but the pgsquash cli/tui will be phased out over time in favor of capysquash for clearer separation between engine, cli and platform.

> **Heads up:** This tool rewrites your SQL files. Back up your migrations first, run the output through tests, and double-check that everything looks right before deploying to production.

## Why pgsquash?

**Tired of migration archaeology?** As your project grows, migration folders become unmanageable. 100-300 files with overlapping changes, conflicting indexes, and forgotten ALTER statements. Onboarding new developers means explaining migration history instead of building features.

**Keep your vibe.** pgsquash intelligently consolidates and optimizes your migration history while preserving dependencies, respecting safety constraints, and validating every change. Works with your existing setup—Supabase projects, Prisma schemas, Clerk auth. No migration rewrites, no new syntax to learn. Just cleaner, safer SQL.

## What it does

- **Intelligently consolidates** 100-300+ migration files into clean, organized output
- **Parser-grade accuracy** using PostgreSQL's own parser (`pg_query_go`)—the same parser PostgreSQL uses internally
- **Dependency-aware** processing that automatically resolves and orders statements safely
- **Safety-first** approach with multiple levels from paranoid (production) to aggressive (dev)
- **Schema validation** against your original schema using Docker containers
- **AI-powered analysis** for detecting duplicate functions, dead code, and optimization opportunities
- **Production-ready streaming** for handling large migration sets efficiently
- **Auto-detection** of Supabase (RLS policies, storage), Clerk (JWT v2), Prisma, and Drizzle patterns

## Interactive mode

There's a built-in TUI if you prefer a visual interface:

```bash
# Launch the dashboard (use pgsquash or capysquash - they're identical)
pgsquash tui migrations/
# or
capysquash tui migrations/

# Or add --tui to any command
pgsquash analyze migrations/ --tui
pgsquash squash migrations/ --tui

# Jump to specific views
pgsquash tui analyze migrations/     # analysis
pgsquash tui config                  # settings
pgsquash tui deps migrations/        # dependency graph
```

The TUI gives you a dashboard with stats, live analysis, a config wizard, dependency visualization, and real-time progress tracking. Press `?` for keyboard shortcuts.

## Installation

```bash
# Build from source
git clone https://github.com/CAPYSQUASH/pgsquash-engine
cd pgsquash-engine
go build -o pgsquash cmd/pgsquash/main.go

# Or install directly
go install github.com/CAPYSQUASH/pgsquash-engine/cmd/pgsquash@latest
```

## Quick Start by Use Case

### Building with Supabase or Clerk?

```bash
# Auto-detects auth schemas, RLS policies, storage buckets
pgsquash analyze migrations/*.sql

# Preview consolidation (doesn't change files)
pgsquash squash migrations/*.sql --dry-run

# Consolidate and validate against your real schema
pgsquash squash migrations/*.sql --output clean/
```

> **Works with Supabase:** Auto-detects `auth.users`, `storage.buckets`, and RLS policies
> **Clerk-ready:** Preserves JWT v2 organization claims and user metadata

### Managing a team?

```bash
# Safe mode for production deploys
pgsquash safe migrations/*.sql --output production/

# Validate before merging PRs
pgsquash validate migrations/ clean/

# Set up GitHub webhooks for automatic PR analysis
# See docs/github-webhooks.md
```

### Working on multiple projects?

```bash
# Share config across team with version control
pgsquash init-config  # creates pgsquash.config.json

# Consistent squashing across projects
pgsquash squash migrations/*.sql  # uses config automatically
```

### Just need it to work?

```bash
# Five-minute setup
pgsquash analyze migrations/*.sql
pgsquash squash migrations/*.sql --dry-run
pgsquash squash migrations/*.sql --output clean/
```

## Common workflows

```bash
# For production: safe and conservative
pgsquash safe migrations/*.sql --output production/

# For development: more aggressive optimization
pgsquash fast migrations/*.sql --output dev/

# Just analyze without changing anything
pgsquash analyze-deep migrations/*.sql
```

## Safety modes

Pick the mode that matches your risk tolerance:

| Mode         | When to use        | What it does      | Typical reduction |
| ------------ | ------------------ | ----------------- | ----------------- |
| Paranoid     | Production systems | Minimal changes   | 15-25%            |
| Conservative | Production         | Safe merges only  | 20-35%            |
| Standard     | Staging/testing    | Balanced approach | 35-50%            |
| Aggressive   | Local development  | Maximum cleanup   | 50-70%            |

## Example output

Here's what consolidated SQL looks like:

```sql
-- Generated by pgsquash (standard mode)

-- === EXTENSIONS ===
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "vector";

-- === FOUNDATION ===
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(255),
    status VARCHAR(50) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT NOW()
);

-- === INDEXES ===
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_status ON users(status) WHERE status = 'active';

-- === SECURITY ===
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
CREATE POLICY users_select ON users FOR SELECT USING (true);
```

## GitHub integration

Automatically analyze migrations in pull requests:

```yaml
# .github/pgsquash.yml
auto_analyze: true
migration_threshold: 15
safety_level: standard
```

Use bot commands in PR comments:

- `/pgsquash analyze` - run analysis
- `/pgsquash consolidate` - create consolidation PR

See [internal/deployments/github-integration.md](docs/internal/deployments/github-integration.md) for setup.

## Documentation

- [Quickstart](docs/quickstart.md) - get started in 5 minutes
- [CLI Reference](docs/cli-reference.md) - all commands and flags
- [Configuration](docs/configuration.md) - config file options
- [Safety Levels](docs/safety-levels.md) - choosing the right mode
- [TUI Guide](docs/tui-guide.md) - using the interactive interface
- [Architecture](docs/architecture.md) - how it works internally
- [Troubleshooting](docs/troubleshooting.md) - common issues

[See all docs](docs/README.md)

## Configuration

Generate a starter config file:

```bash
pgsquash init-config
```

Example `pgsquash.config.json`:

```json
{
  "safety_level": "standard",
  "output": {
    "format": "organized",
    "directory": "squashed"
  },
  "rules": {
    "table_operations": {
      "consolidate_create_alter": true,
      "remove_drop_create_cycles": true
    }
  },
  "performance": {
    "parallel_processing": true,
    "streaming": true
  }
}
```

Check [configuration.md](docs/configuration.md) for all available options.

## API Server

The engine includes an HTTP API server for programmatic access and web integrations:

```bash
# Build API server
go build -o api-server cmd/api-server/main.go

# Run API server
./api-server
```

**Features:**

- REST endpoints for analyze and squash operations
- GitHub webhook integration for PR automation
- CORS support for Platforms
- Health checks and monitoring

See [cmd/api-server/README.md](cmd/api-server/README.md) for API documentation.

## Building from source

```bash
# Clone and build
git clone https://github.com/CAPYSQUASH/pgsquash-engine
cd pgsquash-engine
go mod tidy
go build -o pgsquash cmd/pgsquash/main.go

# Run tests
go test ./...

# Try it out
./pgsquash analyze test_migrations/*.sql
```

The codebase is organized as:

```
cmd/
├── pgsquash/           # CLI entry
└── api-server/         # HTTP API server
internal/
├── parser/             # SQL parsing via pg_query_go
├── tracking/           # Object lifecycle tracking
├── squasher/           # Consolidation logic
├── validation/         # Docker validation
├── ai/                 # AI integrations
├── github/             # GitHub integration
├── plugins/            # Plugin system
└── transformation/     # SQL transformations
```

## What's next

We're working toward 1.0 with:

- Better test coverage
- Performance benchmarks
- More auth plugins (Auth0, NextAuth)
- Platform-specific plugins (Neon, Railway)
- PostgreSQL 18 support

See the [roadmap](docs/internal/roadmap/ROADMAP.md) for details.

## License

MIT License - see LICENSE file

## Links

- **GitHub**: <https://github.com/CAPYSQUASH/pgsquash-engine>
- **Documentation**: [docs/](docs/)
- **Issues**: <https://github.com/CAPYSQUASH/pgsquash-engine/issues>
