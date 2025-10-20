# Migration Consolidation Strategy

**pgsquash Engine** consolidates PostgreSQL migration files by tracking object lifecycles, analyzing dependencies, and applying safety-aware transformation rules. This document explains the complete consolidation workflow from input migrations to optimized output SQL.

## Core Principle

**Consolidate database objects to their earliest appearance point while preserving the latest version and maintaining dependency order.**

This means:

- Objects appear in the output at their first creation point
- The definition reflects all subsequent modifications (final state)
- Dependencies are satisfied (tables before foreign keys, types before functions)
- The result is semantically equivalent to applying all migrations sequentially

---

## The 6-Phase Processing Pipeline

pgsquash processes migrations through six distinct phases:

### Phase 0: Plugin Initialization

**Happens BEFORE parsing to enable plugin enrichment**

The plugin system auto-detects third-party patterns in migrations (Clerk, Supabase, Prisma, Drizzle, etc.) and initializes plugins that will modify behavior throughout the pipeline.

**Plugin Priority Hierarchy:**

- **90-100**: Auth Services (Clerk: 95, Supabase: 90)
- **70-89**: ORMs (Prisma: 75, Drizzle: 75)
- **50-69**: Platforms (future: Neon, Railway)

Higher priority wins conflicts. Only the highest-priority plugin in each category is activated.

**Plugin Capabilities:**

- SQL transformations (pre-parse syntax normalization)
- Custom consolidation rules
- Validation compatibility layers (auth function stubs)
- Detection patterns (JWT claims, migration metadata tables)

**Implementation:** [`initializePlugins()` in engine.go:1705-1747](../internal/squasher/engine.go#L1705-L1747)

**See also:** [Plugin System Guide](../internal/plugins/README.md)

---

### Phase 1: Parsing and Tracking

**Goal:** Parse SQL into AST and track object lifecycles

**Process:**

1. Parse each migration file using `pg_query_go` (PostgreSQL's official parser)
2. Extract object metadata: name, type, operation, dependencies
3. Create `ObjectLifecycle` for each database object
4. Record every operation as a `LifecycleEvent` with:
   - Migration sequence number
   - Dependencies at that point in time
   - Risk assessment
   - Source location (file, line number)
5. Build dependency graph

**Key Point:** The tracker maintains **complete history** of every object. You can query:

- "Was this table ever dropped?"
- "What's the final state of this function?"
- "What depends on this view?"

**Implementation:**

- Parser: [`internal/parser/parser.go`](../internal/parser/parser.go)
- Tracker: [`internal/tracking/unified_tracker.go`](../internal/tracking/unified_tracker.go)
- Lifecycle: [`ProcessMigration()` at unified\_tracker.go:656-732](../internal/tracking/unified_tracker.go#L656-L732)

---

### Phase 2: Dependency Analysis

**Goal:** Resolve dependencies and detect cycles

**Process:**

1. Run `UnifiedDependencyResolver` to analyze cross-object dependencies
2. Perform topological sort to determine safe ordering
3. Detect circular dependencies (e.g., table A references B, B references A)
4. Run advanced DDL cycle detection:
   - DROP → CREATE cycles (object recreated)
   - CREATE → DROP cycles (object created then removed)
   - ALTER → ALTER conflicts (conflicting schema changes)
5. Validate consistency of tracked state

**DDL Cycle Severity Levels:**

- **CRITICAL**: Drop followed by incompatible recreate (data loss risk)
- **HIGH**: Complex circular ALTER dependencies
- **MEDIUM**: Simple DROP → CREATE cycles (safe if no data)
- **LOW**: Informational warnings

**Critical cycles mark objects as non-squashable** to prevent unsafe optimizations.

**Implementation:**

- Resolver: [`UnifiedDependencyResolver` in unified\_dependency\_resolver.go](../internal/squasher/unified_dependency_resolver.go)
- Cycle Detection: [`DetectDDLCycles()` at unified\_tracker.go:1091-1128](../internal/tracking/unified_tracker.go#L1091-L1128)
- Analysis: [`analyzeDependenciesAndRisks()` at engine.go:648-719](../internal/squasher/engine.go#L648-L719)

---

### Phase 3: Consolidation Rules Application

**Goal:** Apply safety-level-appropriate transformation rules

**Safety-Driven Rule Application:**

| Safety Level     | Rules Applied                                                    | Use Case                                     |
| ---------------- | ---------------------------------------------------------------- | -------------------------------------------- |
| **Conservative** | Bug fixes + basic consolidation (CREATE+ALTER, column evolution) | Production (minimal optimization)            |
| **Standard**     | + DROP/CREATE cycles, RLS consolidation, transaction boundaries  | Staging/Testing (balanced)                   |
| **Aggressive**   | + Function deduplication                                         | Development (maximum optimization)           |
| **Paranoid**     | + Dead code removal (requires DB connection)                     | Critical production (verify against live DB) |

**Core Consolidation Rules:**

1. **MultipleCreateConsolidationRule**
   - **Pattern:** Multiple `CREATE IF NOT EXISTS` for same object
   - **Action:** Merge all column definitions into single CREATE
   - **Example:** `CREATE TABLE users (id INT)` + `CREATE TABLE users (id INT, name TEXT)` → unified schema with both columns

2. **CreateAlterConsolidationRule**
   - **Pattern:** Single CREATE followed by ALTER operations
   - **Action:** Integrate ALTER operations into CREATE statement
   - **Example:** `CREATE TABLE t (id INT)` + `ALTER TABLE t ADD COLUMN name TEXT` → `CREATE TABLE t (id INT, name TEXT)`

3. **DropCreateCycleRule** (Standard+)
   - **Pattern:** DROP followed by CREATE (object recreation)
   - **Action:** Remove DROP, keep final CREATE
   - **Safety:** Only if no data operations between DROP and CREATE

4. **RLSConsolidationRule** (Standard+)
   - **Pattern:** Multiple RLS ENABLE/DISABLE operations
   - **Action:** Consolidate to final RLS state
   - **Example:** `ENABLE RLS` → `DISABLE RLS` → `ENABLE RLS` → final: `ENABLE RLS`

5. **FunctionDeduplicationRule** (Aggressive+)
   - **Pattern:** Multiple definitions of same function
   - **Action:** Keep only final definition
   - **Safety:** Checks for signature differences

6. **DeadCodeRemovalRule** (Paranoid only)
   - **Pattern:** Objects created but never referenced
   - **Action:** Remove entirely (requires DB query to check usage)
   - **Requirement:** Needs `prod_db_dsn` connection

**Bug Fix Rules (applied at ALL safety levels):**

- `DOBlockEnumTypeRule` - Fixes DO block ENUM type creation issues
- `EnumDeduplicationRule` - Removes duplicate ENUM type definitions

**Implementation:**

- Rule Engine: [`ConsolidationRuleEngine` in tracker\_types.go:335-379](../internal/tracking/tracker_types.go#L335-L379)
- Rule Definitions: [`internal/tracking/tracker_types.go`](../internal/tracking/tracker_types.go)
- Application: [`applyConsolidationRules()` at engine.go:722-855](../internal/squasher/engine.go#L722-L855)

---

### Phase 4: SQL Generation

**Goal:** Generate optimized SQL with proper dependency ordering

**Category-Based Ordering (dependency-first):**

The builder outputs objects in this exact order to ensure dependencies are satisfied:

```go
1. CategoryExtensions     // CREATE EXTENSION (must be first)
2. CategoryFoundation     // Tables, views, sequences
3. CategoryConstraints    // ALTER TABLE ADD CONSTRAINT (after all tables exist)
4. CategoryFunctions      // CREATE FUNCTION (after types/tables)
5. CategoryTriggers       // CREATE TRIGGER (after functions)
6. CategoryIndexes        // CREATE INDEX (after tables)
7. CategorySecurity       // RLS policies, GRANT/REVOKE
8. CategoryData           // INSERT/UPDATE (last - after schema complete)
```

**Within each category:** Objects are sorted by their internal dependencies using topological sort.

**Extension Dependency Handling:**

Some extensions have dependencies on others (e.g., `earthdistance` requires `cube`). The builder ensures correct order using a predefined dependency map:

```
cube → earthdistance
postgis (independent)
uuid-ossp (independent)
etc.
```

**Formatting Options:**

- **organized** (default): Grouped by category with headers
- **sequential**: Minimal organization, fast generation
- **minimal**: Compact output, no comments

**Implementation:**

- Builder: [`SQLBuilder` in builder/sql.go](../internal/builder/sql.go)
- Generation: [`generateOptimizedSQL()` at engine.go:858-990](../internal/squasher/engine.go#L858-L990)
- Category Order: [engine.go:869-879](../internal/squasher/engine.go#L869-L879)

---

### Phase 5: Post-Processing (Critical Fix Phase)

**Goal:** Fix edge cases and syntax issues from consolidation

This phase runs **AFTER** SQL generation and **BEFORE** transformation. It catches consolidation artifacts that would cause PostgreSQL syntax errors.

**The 6 Post-Processors:**

1. **`fixMalformedDropTriggers`**
   - **Problem:** `DROP TRIGGER table.trigger_name` (invalid syntax)
   - **Fix:** `DROP TRIGGER trigger_name ON table_name`
   - **Cause:** Consolidation sometimes uses qualified names incorrectly

2. **`fixExtensionOrder`**
   - **Problem:** Extensions created in wrong dependency order
   - **Fix:** Reorder to ensure `cube` before `earthdistance`, etc.
   - **Cause:** Multiple migrations create extensions in different orders

3. **`removeOrphanedAlterStatements`**
   - **Problem:** `ALTER TABLE t` but table `t` was dropped/never created
   - **Fix:** Remove orphaned ALTER statements
   - **Cause:** CREATE → DROP cycles leave orphaned ALTERs

4. **`fixMalformedFunctions`**
   - **Problem:** Functions missing `AS` keyword, duplicate `LANGUAGE` clauses
   - **Fix:** Insert missing keywords, remove duplicates
   - **Cause:** ALTER function consolidation can corrupt syntax

5. **`fixMissingSemicolons`**
   - **Problem:** `CREATE TABLE (...)\nALTER TABLE` (missing `;`)
   - **Fix:** Add semicolons after closing parentheses
   - **Cause:** Consolidation joins statements without proper terminators

6. **`fixEliminatedEnumReferences`**
   - **Problem:** Column uses `status_enum` but ENUM was deduplicated to `status`
   - **Fix:** Rewrite all references to use primary ENUM name
   - **Cause:** EnumDeduplicationRule eliminates types but references remain

**Why is this phase necessary?**

Consolidation rules operate on AST structures and lifecycle events. Sometimes the merging logic produces syntactically valid AST but generates SQL with subtle issues (missing keywords, wrong order). Post-processors catch these edge cases.

**Implementation:** [Post-processing block at engine.go:959-989](../internal/squasher/engine.go#L959-L989)

---

### Phase 6: Transformation (Optional)

**Goal:** Modernize SQL and apply best practices

If `sqlTransformer` is enabled, this phase applies:

- Conversion to modern PostgreSQL syntax (e.g., `GENERATED ALWAYS AS IDENTITY`)
- Code style normalization
- Security best practices (e.g., STABLE markers on auth functions)

**Implementation:**

- Transformer: [`SQLTransformer` in transformation/sql\_transformer.go](../internal/transformation/sql_transformer.go)
- Application: [engine.go:468-483](../internal/squasher/engine.go#L468-L483)

---

## Version Resolution in Detail

**What does "final version in earliest migration" mean?**

### For Objects with Multiple CREATE Statements

When an object has multiple `CREATE IF NOT EXISTS` (common in migration frameworks):

**Input:**

```sql
-- migration_1.sql
CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY);

-- migration_3.sql
CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY, email TEXT);

-- migration_5.sql
CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY, email TEXT, name TEXT);
```

**Output:** (placed at migration\_1 position)

```sql
CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    email TEXT,
    name TEXT
);
```

**Logic:** All column definitions are **merged** (union of all columns across CREATEs).

### For Objects with CREATE + ALTER

When an object is created then modified:

**Input:**

```sql
-- migration_1.sql
CREATE TABLE profiles (id UUID PRIMARY KEY);

-- migration_2.sql
ALTER TABLE profiles ADD COLUMN name TEXT;

-- migration_4.sql
ALTER TABLE profiles ADD COLUMN created_at TIMESTAMP DEFAULT NOW();
```

**Output:** (placed at migration\_1 position)

```sql
CREATE TABLE profiles (
    id UUID PRIMARY KEY,
    name TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);
```

**Logic:** ALTER operations are **integrated** into CREATE (not appended as separate statements).

**Exception:** Some ALTER operations CANNOT be integrated and remain separate:

- `ALTER TABLE ... ENABLE ROW LEVEL SECURITY`
- `ALTER TABLE ... OWNER TO`
- `ALTER TABLE ... RENAME TO`
- Column modifications (`ALTER COLUMN ... SET DEFAULT`)

These are appended after CREATE:

```sql
CREATE TABLE profiles (...);

ALTER TABLE profiles ENABLE ROW LEVEL SECURITY;
```

**Implementation:**

- Multiple CREATE merge: [`mergeMultipleCreateStatements()` in tracker\_types.go](../internal/tracking/tracker_types.go)
- ALTER integration: [`integrateAlterIntoCreate()` in tracker\_types.go](../internal/tracking/tracker_types.go)

---

## Lifecycle Tracking System

**How does pgsquash track every object?**

### ObjectLifecycle Structure

Each database object (table, function, view, etc.) has an `ObjectLifecycle` that records:

```go
type ObjectLifecycle struct {
    Key          string              // "public.users::TABLE"
    Name         string              // "users"
    Schema       string              // "public"
    Type         types.ObjectType    // TABLE, FUNCTION, etc.
    History      []LifecycleEvent    // Complete history of operations
    Dependencies []ObjectDependency  // What this depends on
    Category     types.Category      // For output ordering
    WasDropped   bool                // Ever dropped?
    RiskLevel    RiskLevel           // LOW, MEDIUM, HIGH, CRITICAL
}
```

### LifecycleEvent Details

Every operation (CREATE, ALTER, DROP, etc.) is recorded as a `LifecycleEvent`:

```go
type LifecycleEvent struct {
    ID           string              // "migration_5.sql:42:3:TABLE:users"
    Migration    string              // "migration_5.sql"
    Sequence     int                 // Migration order: 5
    Operation    types.Operation     // CREATE, ALTER, DROP
    Statement    types.Statement     // Full parsed statement
    Dependencies []string            // Dependencies at this point
    RiskLevel    RiskLevel           // Assessed risk
    SourceRange  *SourceRange        // File location
}
```

### Dependency Graph

The tracker maintains a `DependencyGraph` that supports:

- **Forward dependencies:** "What does X depend on?"
- **Reverse dependencies:** "What depends on X?"
- **Topological sort:** "What order should I create these objects?"
- **Cycle detection:** "Are there circular dependencies?"

**Example:**

```
Table: users
  └─> Depends on: EXTENSION uuid-ossp (for uuid_generate_v4())

Function: get_user_by_email
  └─> Depends on: TABLE users

Trigger: users_updated_at
  ├─> Depends on: TABLE users
  └─> Depends on: FUNCTION update_timestamp
```

**Implementation:**

- Lifecycle: [`ObjectLifecycle` in unified\_tracker.go:39-70](../internal/tracking/unified_tracker.go#L39-L70)
- Events: [`LifecycleEvent` in unified\_tracker.go:72-85](../internal/tracking/unified_tracker.go#L72-L85)
- Graph: [`DependencyGraph` in unified\_tracker.go:529-541](../internal/tracking/unified_tracker.go#L529-L541)

---

## Advanced Features

### Streaming Mode

For large migration sets (500+ files, >5MB total), pgsquash can operate in **streaming mode**:

**Features:**

- Batch processing (configurable batch size)
- Memory limits (prevents OOM on huge datasets)
- Progress tracking (via callback)
- Worker pools (parallel processing)
- Automatic cleanup (frees memory after processing)

**Usage:**

```bash
pgsquash squash migrations/*.sql \
  --streaming \
  --memory-limit 512 \
  --batch-size 100 \
  --workers 8
```

**Auto-enabled when:**

- File count > 100
- Total size > `streaming_threshold_mb` (config)
- `--streaming` flag set

**Implementation:**

- Streaming Tracker: [`StreamingTracker` in streaming\_integration.go](../internal/tracking/streaming_integration.go)
- Engine Integration: [`SquashStreaming()` at engine.go:498-563](../internal/squasher/engine.go#L498-L563)

### Pre-Squash Backup

If `backupGenerator` is enabled and `prod_db_dsn` is configured:

**Process:**

1. Connect to production database
2. Generate full schema dump (pg\_dump)
3. Save to timestamped backup file
4. Continue with squashing

**Config:**

```json
{
  "prod_db_dsn": "postgres://user:pass@localhost/db",
  "backup": {
    "enabled": true,
    "backup_dir": "backups",
    "compression": true
  }
}
```

**Implementation:** [`BackupGenerator` in transformation/backup\_generator.go](../internal/transformation/backup_generator.go)

### Rollback Script Generation

When `--rollback` flag is used:

**Generated artifacts:**

1. `rollback_plans/rollback_<timestamp>.json` - Complete rollback plan with:
   - Original schema state
   - Squashed schema state
   - Reverse operations (DROP → CREATE, ALTER → reverse ALTER)
   - Dependencies and ordering

2. Executable SQL scripts (future feature)

**Usage:**

```bash
pgsquash squash migrations/*.sql --rollback --output clean/
```

**Implementation:** [`RollbackManager` in transformation/rollback\_manager.go](../internal/transformation/rollback_manager.go)

### Docker-Based Validation

After squashing, pgsquash can **validate schema equivalence** by:

**Three Validation Modes:**

1. **TWO\_CONTAINERS** (most accurate)
   - Spin up two PostgreSQL containers
   - Apply original migrations to Container 1
   - Apply squashed migrations to Container 2
   - Compare schemas using `pg_dump --schema-only`
   - Report differences

2. **TWO\_DATABASES** (balanced)
   - Single container, two databases
   - Apply migrations to separate databases
   - Compare schemas
   - Faster than two containers

3. **SCHEMA\_DIFF** (fastest)
   - Single container, schema versioning
   - Apply both sets sequentially
   - Quick diff comparison

**Plugin-Aware Validation:**

If plugins are active (e.g., Clerk), the validator injects **compatibility SQL** to make validation work:

```sql
-- Injected for Clerk validation
CREATE SCHEMA IF NOT EXISTS auth;
CREATE FUNCTION auth.clerk_user_id() RETURNS UUID LANGUAGE sql STABLE AS $$
    SELECT '00000000-0000-0000-0000-000000000000'::UUID;
$$;
```

This allows squashed migrations to validate even if they reference plugin-specific functions.

**Config:**

```json
{
  "validation": {
    "mode": "TWO_CONTAINERS",
    "enable_extension_detection": true,
    "enable_sql_fixes": true
  }
}
```

**Implementation:**

- Validator: [`Validator` in validation/validator.go](../internal/validation/validator.go)
- Extension Detection: [`ExtensionDetector` in squasher/extension\_detector.go](../internal/squasher/extension_detector.go)

---

## Transaction Safety

**What about transactions?**

### Current Behavior

The builder **does NOT automatically wrap output in transactions**. Statements are output sequentially:

```sql
CREATE TABLE users (...);

CREATE TABLE posts (...);

ALTER TABLE posts ADD CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id);
```

### TransactionBoundaryRule

There is a `TransactionBoundaryRule` (available in Standard+ safety) that:

- Groups related DDL operations
- Marks transaction boundaries with comments
- **Does NOT output `BEGIN`/`COMMIT`** (that's up to the user)

**Example output with TransactionBoundaryRule:**

```sql
-- TRANSACTION BOUNDARY: Table creation group
CREATE TABLE users (...);
CREATE TABLE posts (...);

-- TRANSACTION BOUNDARY: Constraint group
ALTER TABLE posts ADD CONSTRAINT fk_user ...;
```

### User Responsibility

Users must wrap output in transactions if needed:

```sql
BEGIN;
\i squashed_migrations.sql
COMMIT;
```

Or use psql's `--single-transaction` flag:

```bash
psql -d mydb --single-transaction -f squashed_migrations.sql
```

**Rationale:** Different deployment strategies need different transaction handling (all-or-nothing vs. incremental).

---

## Data Preservation

**How does pgsquash handle data operations?**

### Data Operation Detection

The parser marks statements with `IsDataOp` flag:

- `INSERT`
- `UPDATE`
- `DELETE`
- `TRUNCATE`
- `COPY`

### Separation from Schema DDL

Data operations are:

1. **Never consolidated** (each INSERT/UPDATE is preserved exactly)
2. **Output last** (CategoryData is the final category)
3. **Sequenced correctly** (original order maintained)

### Column/Table Dependencies

If data operations reference columns or tables that were dropped:

**Example:**

```sql
-- migration_1.sql
CREATE TABLE users (id INT, old_email TEXT);

-- migration_2.sql
INSERT INTO users (id, old_email) VALUES (1, 'test@example.com');

-- migration_3.sql
ALTER TABLE users DROP COLUMN old_email;
ALTER TABLE users ADD COLUMN email TEXT;
```

**Result:**

- The `old_email` column is NOT in the final schema
- The INSERT referencing `old_email` is **preserved with a warning**
- User must manually fix data operations that reference dropped columns

**Why not auto-fix?**

Data transformations are **semantic** (not syntactic). Automatically rewriting `old_email` → `email` could corrupt data. The user must decide how to migrate data.

**Implementation:**

- Data ops marked during parsing: [`parser.go`](../internal/parser/parser.go)
- Preserved during consolidation: [Consolidation rules check `HasDataOps`](../internal/tracking/tracker_types.go)
- Output last: [CategoryData is final](../internal/squasher/engine.go#L879)

---

## Environment Compatibility

### PostgreSQL Version Support

pgsquash supports PostgreSQL 12+ syntax, including:

- Modern identity columns (`GENERATED ALWAYS AS IDENTITY`)
- Advanced RLS policies
- Vector extensions (pgvector)
- Generated columns

**Version detection:** Controlled via `modern_features` config:

```json
{
  "modern_features": {
    "enable_vector_support": true,
    "enable_generated_columns": true,
    "enable_advanced_rls": true
  }
}
```

### Extension Requirements

If migrations use PostgreSQL extensions, the output requires those extensions:

**Auto-detected extensions:**

- `uuid-ossp`
- `pgcrypto`
- `postgis`
- `pg_trgm`
- `earthdistance` (requires `cube`)
- `vector` (pgvector)

**Docker image recommendation:**

For migrations using `earthdistance` + `postgis`:

```
Detected extensions: [earthdistance, postgis, uuid-ossp]
Recommended Docker image: postgis/postgis:16-3.4
```

**Implementation:** [`ExtensionDetector` in squasher/extension\_detector.go](../internal/squasher/extension_detector.go)

---

## Testing Requirements

After squashing, you should:

1. **Run validation:** `pgsquash validate original/ squashed/`
   - Ensures schema equivalence
   - Catches consolidation bugs

2. **Test against production data copy:**
   - Apply squashed migrations to a production snapshot
   - Verify all application functionality works
   - Check performance characteristics

3. **Test rollback path (if generated):**
   - Apply squashed migrations
   - Apply rollback scripts
   - Verify return to original state

4. **Manual inspection:**
   - Review `squashed_migrations.sql` for sanity
   - Check for any warnings in squash output
   - Verify critical objects are present

**Automated testing:**

```bash
#!/bin/bash
# Test script

# 1. Squash with validation
pgsquash squash migrations/*.sql --output squashed/ --validate

# 2. Apply to test database
psql -d test_db --single-transaction -f squashed/squashed_migrations.sql

# 3. Run application test suite
npm test

# 4. Compare schemas
pg_dump --schema-only original_db > original_schema.sql
pg_dump --schema-only test_db > squashed_schema.sql
diff original_schema.sql squashed_schema.sql
```

---

## Cross-References

For deeper technical details, see:

- **[Architecture Documentation](architecture.md)** - Complete system design, data structures, and algorithms
- **[Safety Levels Guide](safety-levels.md)** - Detailed explanation of each safety level and risk assessment
- **[Plugin System Guide](../internal/plugins/README.md)** - How to write custom plugins, plugin lifecycle
- **[CLI Reference](cli-reference.md)** - All commands, flags, and configuration options
- **[Troubleshooting Guide](troubleshooting.md)** - Common issues and solutions
- **[Configuration Reference](configuration.md)** - Complete config file documentation

For questions or issues:

- GitHub Issues: [pgsquash-engine/issues](https://github.com/CAPYSQUASH/pgsquash-engine/issues)
- Documentation: [docs/](../docs/)

---

## Appendix: Complete Phase Flow Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                   INPUT: Migration Files                         │
│              migration_1.sql, migration_2.sql, ...               │
└───────────────────────────┬─────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│ PHASE 0: Plugin Initialization                                  │
│ • Detect third-party patterns (Clerk, Supabase, Prisma)        │
│ • Initialize plugins by priority                                │
│ • Prepare SQL transformations                                   │
└───────────────────────────┬─────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│ PHASE 1: Parsing and Tracking                                   │
│ • Parse SQL with pg_query_go                                    │
│ • Create ObjectLifecycle for each object                        │
│ • Record LifecycleEvents                                        │
│ • Build dependency graph                                        │
└───────────────────────────┬─────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│ PHASE 2: Dependency Analysis                                    │
│ • Run UnifiedDependencyResolver                                 │
│ • Topological sort                                              │
│ • Detect circular dependencies                                  │
│ • DDL cycle detection (DROP → CREATE, etc.)                    │
│ • Validate consistency                                          │
└───────────────────────────┬─────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│ PHASE 3: Consolidation Rules Application                        │
│ • Select rules based on safety level                            │
│ • Apply MultipleCreateConsolidationRule                         │
│ • Apply CreateAlterConsolidationRule                            │
│ • Apply DropCreateCycleRule (if Standard+)                      │
│ • Apply FunctionDeduplicationRule (if Aggressive+)              │
│ • Apply DeadCodeRemovalRule (if Paranoid)                       │
└───────────────────────────┬─────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│ PHASE 4: SQL Generation                                         │
│ • Sort objects by category (Extensions → Data)                  │
│ • Topological sort within categories                            │
│ • Build SQL with proper formatting                              │
│ • Add headers and comments                                      │
└───────────────────────────┬─────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│ PHASE 5: Post-Processing (Fix Phase)                            │
│ • fixMalformedDropTriggers                                      │
│ • fixExtensionOrder                                             │
│ • removeOrphanedAlterStatements                                 │
│ • fixMalformedFunctions                                         │
│ • fixMissingSemicolons                                          │
│ • fixEliminatedEnumReferences                                   │
└───────────────────────────┬─────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│ PHASE 6: Transformation (Optional)                              │
│ • Apply SQL modernization                                       │
│ • Code style normalization                                      │
│ • Security best practices                                       │
└───────────────────────────┬─────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│              OUTPUT: Squashed Migration File(s)                  │
│                    squashed_migrations.sql                       │
└─────────────────────────────────────────────────────────────────┘
```

---

**Document Version:** 1.0
**Last Updated:** 2025-01-13
**Compatibility:** pgsquash Engine v0.8.5-beta+
