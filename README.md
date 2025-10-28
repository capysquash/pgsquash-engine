# pgsquash-engine

> The open-source Go engine that powers CAPYSQUASH

**Catalog-proven equivalence** via double-build validation. Intelligently reorganizes your migration history into clean, production-ready SQL—without breaking anything.

**Current version:** 0.9.5 (Production Ready) ✅

## What is this?

pgsquash-engine is the core library behind [CAPYSQUASH](https://capysquash.dev), the automatic migration cleanup tool for Supabase, Neon, and modern Postgres.

**For most users:** Use [CAPYSQUASH](https://capysquash.dev) for one-click cleanup or [capysquash-cli](https://github.com/CAPYSQUASH/capysquash-cli) for terminal workflows.

**For developers:** Use pgsquash-engine to build custom migration tools. This library provides the core PostgreSQL migration consolidation functionality with comprehensive validation, safety modes, and proven equivalence guarantees.

## About pgsquash-engine

**The technology behind CAPYSQUASH.** This is the open-source Go library that powers both CAPYSQUASH and capysquash-cli.

Intelligently consolidates and optimizes your migration history while preserving dependencies, respecting safety constraints, and validating every change. Works with your existing setup—Supabase projects, Prisma schemas, Clerk auth. No migration rewrites, no new syntax to learn. Just cleaner, safer SQL.

**Production-ready** with comprehensive validation, safety modes, and proven equivalence guarantees.

## What it does

- **Intelligently consolidates** 100-300+ migration files into clean, organized output
- **Catalog-proven equivalence** via double-build validation—proves the output produces an identical schema by running both versions through PostgreSQL and comparing the results
- **Dependency-aware** processing that automatically resolves and orders statements safely
- **Safety-first** approach with multiple levels from paranoid (production) to aggressive (dev)
- **Schema validation** against your original schema using Docker containers
- **AI-powered analysis** for detecting duplicate functions, dead code, and optimization opportunities
- **Streaming architecture** for memory-efficient processing—processes migrations incrementally without loading entire history into memory (tested with 1000+ migration files)
- **Lock level analysis** with PostgreSQL transaction planning and conflict detection
- **Branch safety warnings** with git integration and protected branch enforcement
- **Manual override pragmas** (`-- pgsquash:ignore`) for complex edge cases
- **Auto-detection** of Supabase (RLS policies, storage), Clerk (JWT v2), Prisma, and Drizzle patterns

## Interactive mode

pgsquash-engine includes a built-in TUI for a visual interface:

```bash
# Launch the dashboard (both commands work identically)
pgsquash tui migrations/
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

**For most users:** Try [CAPYSQUASH](https://capysquash.dev) for one-click cleanup or install [capysquash-cli](https://github.com/CAPYSQUASH/capysquash-cli) for terminal workflows.

**For developers building custom tools:**

```bash
# As a Go library
go get github.com/CAPYSQUASH/pgsquash-engine

# Or build from source
git clone https://github.com/CAPYSQUASH/pgsquash-engine
cd pgsquash-engine
go build -o pgsquash cmd/pgsquash/main.go
```

## Quick Start

### For Most Users

**Try CAPYSQUASH** (fastest - 30 seconds):
- Visit https://capysquash.dev
- Upload migrations or connect GitHub
- Get one-click cleanup with visual dashboard

**Or use capysquash-cli** (for terminal workflows):
```bash
brew install capysquash-cli
capysquash analyze migrations/
capysquash squash migrations/ --output clean/
```

### For Developers Using pgsquash-engine

**Building with Supabase or Clerk?**

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

# Advanced features
pgsquash plan migrations/*.sql                    # Show transaction plan
pgsquash explain-locks migrations/*.sql          # Analyze lock conflicts
pgsquash squash migrations/*.sql --branch-check   # Branch safety check
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

## Integration Scripts

Ready-to-use scripts for popular migration tools:

```bash
# Apply squashed migrations and mark as applied in Prisma
scripts/prisma-baseline.sh

# Reset Drizzle migrations and apply squashed versions
scripts/drizzle-reset.sh

# GitHub Actions workflow for automated validation
.github/workflows/pgsquash-validate.yml
```

See [scripts/README.md](scripts/README.md) for detailed usage and setup instructions.

## Documentation

- [Quickstart](docs/user%20docs/quickstart.md) - get started in 5 minutes
- [CLI Reference](docs/user%20docs/cli-reference.md) - all commands and flags
- [Configuration](docs/user%20docs/configuration.md) - config file options
- [Safety Levels](docs/user%20docs/safety-levels.md) - choosing the right mode
- [TUI Guide](docs/user%20docs/tui-guide.md) - using the interactive interface
- [Architecture](docs/user%20docs/architecture.md) - how it works internally
- [Troubleshooting](docs/user%20docs/troubleshooting.md) - common issues

[See all docs](docs/user%20docs/README.md)

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

pgsquash-engine includes an HTTP API server for programmatic access and web integrations:

```bash
# Build API server
go build -o api-server cmd/api-server/main.go

# Run API server
./api-server
```

**Features:**

- REST endpoints for analyze and squash operations
- GitHub webhook integration for PR automation
- CORS support for web platforms
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

## The CAPYSQUASH Ecosystem

pgsquash-engine is the underlying technology that powers the CAPYSQUASH ecosystem:

- **[CAPYSQUASH](https://capysquash.dev)** - The platform. One-click cleanup with GitHub automation, visual dashboards, and team features
- **[capysquash-cli](https://github.com/CAPYSQUASH/capysquash-cli)** - Open-source CLI tool for terminal workflows and CI/CD pipelines
- **pgsquash-engine** (this library) - The core technology that powers everything

**For most users:** Start with CAPYSQUASH for the easiest experience or capysquash-cli for terminal workflows.

**For developers:** Use pgsquash-engine to build custom migration tools and integrations.

---

## Links

- **CAPYSQUASH**: <https://capysquash.dev>
- **GitHub**: <https://github.com/CAPYSQUASH/pgsquash-engine>
- **Documentation**: [docs/user docs/](docs/user%20docs/)
- **Issues**: <https://github.com/CAPYSQUASH/pgsquash-engine/issues>

---

<sub>Powered by pgsquash-engine • Part of the CAPYSQUASH ecosystem</sub>
