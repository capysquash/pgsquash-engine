# Quickstart

Get up and running in 5 minutes. pgsquash intelligently consolidates your migration history using PostgreSQL's own parser - no changes to your existing migration files needed.

## What You Need

- Go 1.25.3 or later
- SQL migration files (any format)
- Docker (optional, for schema validation)

> **Using Supabase?** No special config needed - pgsquash auto-detects RLS policies, storage schemas, and auth patterns.
> **Using Prisma/Drizzle?** Your ORM metadata tables are automatically preserved.

## Installation

**Option 1: Build from source** (if you want the latest)

```bash
git clone https://github.com/CAPYSQUASH/pgsquash-engine
cd pgsquash-engine
go build -o pgsquash cmd/pgsquash/main.go
./pgsquash --version
```

**Option 2: Install directly** (easiest)

```bash
go install github.com/CAPYSQUASH/pgsquash-engine/cmd/pgsquash@latest
pgsquash --version
```

## Five-Minute Workflow

### Step 1: Analyze Your Migrations (30 seconds)

```bash
pgsquash analyze migrations/*.sql
```

**What you'll see:**

```
📊 Analysis Results:
   Files: 47
   Statements: 312
   Tables: 23
   Indexes: 45
   Functions: 12

🎯 Consolidation Potential: 65%
   - 15 redundant indexes
   - 8 overlapping ALTER TABLE statements
   - 12 duplicate function definitions

☑ Detected Integrations:
   - Supabase (auth.users, RLS policies)
   - Clerk (JWT v2 organization claims)
```

### Step 2: Preview Consolidation (no changes made)

```bash
pgsquash squash migrations/*.sql --dry-run
```

Shows exactly what will be consolidated, how dependencies will be resolved, and which optimizations will be applied - all without modifying any files.

### Step 3: Consolidate Your Migrations

```bash
pgsquash squash migrations/*.sql --output clean/
```

**Result:** Creates intelligently organized files in `clean/`:

```
clean/
├── 001_extensions.sql       # Extensions first (with dependencies resolved)
├── 002_schema_foundation.sql # Tables and core schema (consolidated)
├── 003_constraints.sql       # Foreign keys, checks (circular FKs handled)
├── 004_indexes.sql          # All indexes (deduplicated)
├── 005_functions.sql        # Functions and triggers (optimized)
├── 006_permissions_security.sql # RLS, grants, policies (organized)
└── 007_data.sql            # Seed data (if any, dependency-sorted)
```

Every dependency is automatically resolved - tables before foreign keys, schemas before objects, functions before triggers, etc. Circular foreign key dependencies are detected and handled with two-phase constraint creation.

### Step 4: Validate (requires Docker)

```bash
pgsquash validate migrations/ clean/
```

**What happens:**

1. Spins up two PostgreSQL containers with auto-detected extensions
2. Applies your original migrations to container 1
3. Applies consolidated migrations to container 2
4. Performs byte-level schema comparison using `pg_dump`
5. Auto-generates compatibility layers for auth services (Supabase, Clerk)

**If schemas match:** ☑ Safe to deploy - schemas are byte-for-byte identical
**If schemas differ:** ☒ Shows detailed diff with exact mismatches and suggestions

## Common workflows

**For production** (safe and conservative with full validation):

```bash
pgsquash safe migrations/*.sql --output production/
```

Uses conservative safety level, generates rollback scripts, runs comprehensive Docker validation.

**For development** (aggressive optimization with fast validation):

```bash
pgsquash fast migrations/*.sql --output dev/
```

Enables streaming mode, DDL cycle detection, and automatic SQL modernization.

**For analysis only** (comprehensive insights without modifications):

```bash
pgsquash analyze-deep migrations/*.sql
```

Deep dependency analysis, AI-powered dead code detection, and optimization recommendations.

## Configuration

Generate a config file:

```bash
pgsquash init-config
```

Then edit `pgsquash.config.json` to set your safety level:

```json
{
  "safety_level": "standard"  // paranoid, conservative, standard, aggressive
}
```

See [safety-levels.md](safety-levels.md) for details on each mode.

## Optional AI features

If you want AI-powered analysis for finding duplicate functions and dead code:

```bash
export ANTHROPIC_API_KEY="sk-ant-..."
pgsquash ai-test
pgsquash analyze-deep migrations/*.sql
```

See [ai-features.md](ai-features.md) for details.

## Common use cases

**Supabase projects:**

```bash
pgsquash squash supabase/migrations/*.sql --safety standard
```

Automatically handles RLS policies, auth schema, and storage buckets.

**Large migration sets (500+ files):**

```bash
pgsquash squash migrations/*.sql --streaming --memory-limit 512
```

Enables memory-efficient processing.

**CI/CD:**

```bash
#!/bin/bash
pgsquash analyze migrations/*.sql || exit 1
pgsquash squash migrations/*.sql --output clean/ || exit 1
pgsquash validate migrations/ clean/ || exit 1
git add clean/
git commit -m "chore: squash migrations"
```

## Troubleshooting

**Parse errors:**

```bash
pgsquash analyze migrations/*.sql --verbose
```

**Circular dependencies:**

```bash
pgsquash squash migrations/*.sql --detect-cycles --cycle-details
```

**Validation failures:**
Try a more conservative mode and check the diff:

```bash
pgsquash squash migrations/*.sql --safety conservative
pgsquash validate migrations/ clean/ --verbose
```

## Next steps

- [CLI Reference](cli-reference.md) - all commands
- [Configuration](configuration.md) - config options
- [Safety Levels](safety-levels.md) - mode comparison
- [Troubleshooting](troubleshooting.md) - detailed fixes
- [Architecture](architecture.md) - how it works
