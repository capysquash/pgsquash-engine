# Pattern Detection Guide

How pgsquash understands your SQL and why it doesn't break your schema during consolidation.

## Why Pattern Detection Matters

**The problem:** Naive migration consolidation tools use string matching, which breaks on:

- Schema-qualified names (`public.users` vs `users`)
- Quoted identifiers (`"user"` vs `user`)
- Case variations (`CREATE TABLE` vs `create table`)
- Comments and whitespace differences

**Result:** Broken foreign keys, missing dependencies, wrong statement ordering - production disasters waiting to happen.

**pgsquash's solution:** Use `pg_query_go`, the same parser PostgreSQL uses internally. We parse SQL into an Abstract Syntax Tree (AST) and analyze the actual structure, not string patterns.

## What Pattern Detection Does

1. **Tracks dependencies accurately**: Knows that `posts.user_id` references `users.id`, even with schema qualifications
2. **Safe consolidation**: Only merges operations when dependencies allow it
3. **Auto-detects integrations**: Recognizes Supabase RLS, Clerk schemas, Prisma metadata - no config needed
4. **Catches issues early**: Detects circular dependencies, duplicate indexes, conflicting policies before validation

**Performance:** All patterns are pre-compiled at startup (`internal/patterns/patterns.go`) - 5-10× faster than dynamic regex compilation.

## Pattern Categories

### 1. SQL Parsing Patterns

Used to identify basic SQL structures and extract object names.

| Pattern               | Purpose                      | Example Match                                  |
| --------------------- | ---------------------------- | ---------------------------------------------- |
| `FunctionPattern`     | Extract function names       | `CREATE FUNCTION calculate_total()`            |
| `CreateTablePattern`  | Identify table creation      | `CREATE TABLE users (...)`                     |
| `AlterTablePattern`   | Identify table modifications | `ALTER TABLE users ADD COLUMN email`           |
| `CreateSchemaPattern` | Identify schema creation     | `CREATE SCHEMA IF NOT EXISTS auth`             |
| `CreateIndexPattern`  | Identify index creation      | `CREATE INDEX idx_users_email ON users(email)` |

**Impact on squashing:**

- Tracks object lifecycles (when objects are created/modified/dropped)
- Establishes initial dependency graph
- Enables safe reordering of independent operations

### 2. SQL Transformation Patterns

Used to detect specific types of operations for safety analysis.

| Pattern             | Purpose               | Safety Level Impact                                |
| ------------------- | --------------------- | -------------------------------------------------- |
| `InsertPattern`     | Detect data insertion | Conservative: Preserve order                       |
| `UpdatePattern`     | Detect data updates   | Aggressive: Can consolidate if targeting same rows |
| `DeletePattern`     | Detect data deletion  | Conservative: Preserve (data loss risk)            |
| `DropTablePattern`  | Detect table removal  | Paranoid: Flag as high-risk                        |
| `DropColumnPattern` | Detect column removal | Conservative: Preserve (schema change)             |
| `AlterTypePattern`  | Detect type changes   | Aggressive: Can be dangerous, warn user            |

**Impact on squashing:**

- Determines which operations can be safely consolidated
- Influences safety level recommendations
- Triggers warnings for risky operations

**Example:**

```sql
-- Migration 1
INSERT INTO users (name) VALUES ('Alice');

-- Migration 2
UPDATE users SET name = 'Alice Smith' WHERE name = 'Alice';

-- Result (Aggressive mode):
-- Consolidated: INSERT INTO users (name) VALUES ('Alice Smith');
```

### 3. Dependency Resolution Patterns

**The disaster scenario:** Squash `posts` table creation before `users` table, breaking the foreign key. Deploy fails, rollback required.

**How we prevent it:** Track every reference between objects.

| Pattern                  | Detects                          | Why It Matters                             |
| ------------------------ | -------------------------------- | ------------------------------------------ |
| `ForeignKeyPattern`      | `REFERENCES users(id)`           | Can't create posts before users exist      |
| `InsertIntoPattern`      | `INSERT INTO users`              | Can't insert before table exists           |
| `ExecuteFunctionPattern` | `EXECUTE calculate_tax()`        | Can't call function before it's created    |
| `UpdateTablePattern`     | `UPDATE posts SET user_id = ...` | Can't update column that doesn't exist yet |

**Real-world example:**

```sql
-- Migration 01_users.sql
CREATE TABLE users (id SERIAL PRIMARY KEY, email VARCHAR(255));

-- Migration 02_posts.sql
CREATE TABLE posts (
  id SERIAL PRIMARY KEY,
  user_id INT REFERENCES users(id)  -- Dependency detected
);

-- Migration 03_seed.sql
INSERT INTO users (email) VALUES ('admin@example.com');  -- Dependency detected
INSERT INTO posts (user_id) VALUES (1);  -- Dependency detected

-- Result: Order preserved exactly as written
-- Why: pgsquash builds dependency graph and respects it during consolidation
```

**What about circular dependencies?**
Detected and flagged as errors. You'll need to break the cycle manually (usually by adding the FK constraint after both tables exist).

### 4. Authentication Patterns

**Why it matters:** Auth policies are order-dependent. Wrong consolidation = broken security.

**The problem:** You have two RLS policies on `users`:

```sql
-- Policy 1: Users can see their own data
CREATE POLICY users_own ON users FOR SELECT USING (auth.uid() = id);

-- Policy 2: Admins can see all data
CREATE POLICY users_admin ON users FOR SELECT USING (is_admin = true);
```

**Without pattern detection:** Tool sees two policies on same table, merges them randomly. Admin policy might get evaluated first, letting non-admins see all data. Security bug.

**With pattern detection:** Tool recognizes `auth.uid()` (Supabase) or `auth.jwt()` (Clerk) as auth patterns, preserves exact policy order.

#### Clerk Plugin Patterns

| Pattern                  | What It Detects            | Why It Matters                             |
| ------------------------ | -------------------------- | ------------------------------------------ |
| `auth.jwt()->'o'->>'id'` | JWT v2 organization claims | Multi-tenant apps - wrong org = data leak    |
| `clerk_user_id()`        | Custom helper function     | Breaking this function breaks all policies |
| `auth.jwt()->>'sub'`     | Generic JWT subject        | User identity - used in every auth policy    |

#### Supabase Plugin Patterns

| Pattern                              | What It Detects        | Why It Matters                                 |
| ------------------------------------ | ---------------------- | ---------------------------------------------- |
| `auth.uid()`                         | Supabase auth function | Core auth primitive - every RLS policy uses this |
| `auth.users`                         | Auth schema table      | Can't consolidate if schema doesn't exist      |
| `storage.objects`, `storage.buckets` | Storage tables         | File permissions depend on these               |
| `supabase_realtime`                  | Realtime publication   | Real-time subscriptions break without this     |

**Real-world example:**

```sql
-- Migration with Supabase pattern
CREATE POLICY "users_view_own"
  ON users
  FOR SELECT
  USING (auth.uid() = id);  -- Pattern detected: High priority (90)

CREATE POLICY "users_view_public"
  ON users
  FOR SELECT
  USING (is_public = true);  -- Lower priority

-- Result:
-- ☑ Order preserved (auth policy evaluated first)
-- ☑ Validates auth schema exists during validation
-- ☑ Groups with other auth policies in output (006_permissions_security.sql)
```

### 5. ORM Patterns

Used by plugins to detect ORM framework conventions.

#### Prisma Plugin Patterns

| Pattern               | Detection                                 | Handling             |
| --------------------- | ----------------------------------------- | -------------------- |
| Migration file format | `*_migration.sql` in `prisma/migrations/` | Directory-based      |
| Shadow database       | `prisma_migrations` table                 | Preserve metadata    |
| Relation tables       | `_*_to_*` naming                          | Preserve join tables |

#### Drizzle Plugin Patterns

| Pattern               | Detection                      | Handling                 |
| --------------------- | ------------------------------ | ------------------------ |
| Migration file format | `drizzle/\d{14}_[a-z_]+`       | Timestamp-based          |
| Identity columns      | `GENERATED ALWAYS AS IDENTITY` | Normalize syntax         |
| Serial primary keys   | `SERIAL PRIMARY KEY`           | Convert to standard form |

**Impact on squashing:**

- Maintains ORM metadata compatibility
- Normalizes ORM-specific SQL syntax
- Preserves ORM conventions in output
- Prevents breaking ORM tooling

**Example:**

```sql
-- Drizzle migration (before normalization)
CREATE TABLE users (
  id SERIAL PRIMARY KEY,
  name VARCHAR(255)
);

-- After Drizzle plugin normalization
CREATE TABLE users (
  id INTEGER GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
  name VARCHAR(255)
);
```

### 6. Validation Patterns

Used to detect potential issues before squashing.

| Pattern                   | Purpose                       | Warning Level |
| ------------------------- | ----------------------------- | ------------- |
| `CreateExtensionPattern`  | Detect extension creation     | Info          |
| `DropExtensionPattern`    | Detect extension removal      | Warning       |
| `AlterPublicationPattern` | Detect publication changes    | Info          |
| `TransactionPattern`      | Detect transaction boundaries | Critical      |
| `CreatePolicyPattern`     | Detect RLS policy creation    | Info          |
| `FunctionHeaderPattern`   | Detect function definitions   | Info          |

**Impact on squashing:**

- Warns about potentially breaking changes
- Suggests safer alternatives
- Validates SQL syntax
- Checks for common mistakes

### 7. Backup Generator Patterns

Used to generate rollback scripts for risky operations.

| Pattern                  | Operation          | Rollback Generated          |
| ------------------------ | ------------------ | --------------------------- |
| `AddColumnPattern`       | Add column         | `DROP COLUMN`               |
| `DropColumnPattern`      | Drop column        | Store column definition     |
| `AddConstraintPattern`   | Add constraint     | `DROP CONSTRAINT`           |
| `DropConstraintPattern`  | Drop constraint    | Store constraint definition |
| `AlterColumnTypePattern` | Change column type | Store original type         |
| `RenameTablePattern`     | Rename table       | `RENAME TO` (reverse)       |

**Impact on squashing:**

- Enables safe experimentation with `--rollback` flag
- Provides undo scripts for production issues
- Documents schema changes automatically

## Pattern Usage in Squashing

### Phase 1: Initial Parsing

```
Read migrations → Apply patterns → Extract objects → Build initial graph
```

Patterns used:

- `CreateTablePattern`, `CreateSchemaPattern`, `FunctionPattern`
- Extracts: Tables, schemas, functions, indexes

### Phase 2: Dependency Analysis

```
Analyze SQL → Find references → Build dependency tree → Detect cycles
```

Patterns used:

- `ForeignKeyPattern`, `DirectRefPattern`, `FunctionCallPattern`
- Tracks: Which objects depend on which other objects

### Phase 3: Operation Classification

```
Classify each SQL → Assign safety level → Group related operations
```

Patterns used:

- `InsertPattern`, `UpdatePattern`, `DeletePattern`, `DropTablePattern`
- Determines: Which operations can be consolidated

### Phase 4: Consolidation

```
Group operations → Apply rules → Merge where safe → Preserve order
```

Rules depend on patterns:

- Data operations (INSERT/UPDATE/DELETE): Preserve or consolidate based on safety level
- Schema changes (ALTER TABLE): Merge multiple ALTERs on same table
- Duplicates (CREATE ENUM): Deduplicate identical definitions

### Phase 5: Output Generation

```
Build SQL → Apply formatting → Add comments → Generate rollback
```

Patterns used:

- Plugin-specific patterns for normalization
- Backup patterns for rollback generation

## Configuration

Patterns are **not configurable** by design for performance and consistency reasons. However, you can influence how patterns affect squashing through:

### Safety Levels

Control consolidation aggressiveness:

```bash
# Conservative: Minimal consolidation
pgsquash squash --safety conservative

# Standard: Balanced approach (default)
pgsquash squash --safety standard

# Aggressive: Maximum consolidation
pgsquash squash --safety aggressive

# Paranoid: No consolidation, only validation
pgsquash squash --safety paranoid
```

See [safety-levels.md](./safety-levels.md) for details.

### Plugin Detection

Enable/disable specific plugins:

```json
{
  "plugins": {
    "clerk": {
      "enabled": true,
      "priority": 95
    },
    "supabase": {
      "enabled": true,
      "priority": 90
    },
    "prisma": {
      "enabled": true,
      "priority": 85
    },
    "drizzle": {
      "enabled": true,
      "priority": 80
    }
  }
}
```

See [configuration.md](./configuration.md) for plugin configuration.

## Pattern Detection Examples

### Example 1: Dependency Ordering

**Input migrations:**

```sql
-- Migration 1
CREATE TABLE posts (
  id SERIAL PRIMARY KEY,
  user_id INT REFERENCES users(id)  -- Forward reference!
);

-- Migration 2
CREATE TABLE users (
  id SERIAL PRIMARY KEY
);
```

**Pattern detection:**

1. `CreateTablePattern` identifies both tables
2. `ForeignKeyPattern` detects `posts` → `users` dependency
3. Dependency resolver detects forward reference
4. Engine reorders migrations

**Output:**

```sql
-- Squashed migration (reordered)
CREATE TABLE users (
  id SERIAL PRIMARY KEY
);

CREATE TABLE posts (
  id SERIAL PRIMARY KEY,
  user_id INT REFERENCES users(id)
);
```

### Example 2: Auth Plugin Detection

**Input:**

```sql
-- Migration with Clerk
CREATE POLICY "org_members"
  ON users
  FOR SELECT
  USING ((auth.jwt()->'o'->>'id') = organization_id);
```

**Pattern detection:**

1. Plugin system scans all migrations
2. Clerk plugin detects `auth.jwt()->'o'->>'id'` pattern
3. Clerk plugin activates with priority 95
4. Preserves policy order and validates auth schema

**Output:**

```sql
-- Squashed with Clerk plugin active
-- Clerk JWT v2 auth pattern detected

CREATE POLICY "org_members"
  ON users
  FOR SELECT
  USING ((auth.jwt()->'o'->>'id') = organization_id);
```

### Example 3: Data Operation Consolidation

**Input migrations:**

```sql
-- Migration 1
INSERT INTO users (name, email) VALUES ('Alice', 'alice@old.com');

-- Migration 2
UPDATE users SET email = 'alice@new.com' WHERE name = 'Alice';

-- Migration 3
INSERT INTO users (name, email) VALUES ('Bob', 'bob@test.com');
```

**Pattern detection (Aggressive mode):**

1. `InsertPattern` detects both inserts
2. `UpdatePattern` detects update
3. Data operation tracker links INSERT → UPDATE for same user
4. Consolidation rule merges operations

**Output:**

```sql
-- Squashed migration (consolidated)
INSERT INTO users (name, email) VALUES
  ('Alice', 'alice@new.com'),  -- Merged INSERT + UPDATE
  ('Bob', 'bob@test.com');
```

### Example 4: Circular Dependency Detection

**Input:**

```sql
CREATE TABLE departments (
  id SERIAL PRIMARY KEY,
  manager_id INT REFERENCES employees(id)  -- Circular!
);

CREATE TABLE employees (
  id SERIAL PRIMARY KEY,
  department_id INT REFERENCES departments(id)  -- Circular!
);
```

**Pattern detection:**

1. `ForeignKeyPattern` detects both references
2. Dependency resolver builds graph
3. Cycle detection finds: departments ↔ employees
4. Circular FK handler extracts constraint

**Output:**

```sql
-- Circular dependency resolved
CREATE TABLE departments (
  id SERIAL PRIMARY KEY
);

CREATE TABLE employees (
  id SERIAL PRIMARY KEY,
  department_id INT REFERENCES departments(id)
);

-- Circular foreign key added after tables exist
ALTER TABLE departments
  ADD CONSTRAINT departments_manager_id_fkey
  FOREIGN KEY (manager_id) REFERENCES employees(id);
```

## Advanced: Custom Pattern Detection

While built-in patterns can't be modified, you can extend pgsquash with custom plugins for domain-specific patterns.

### Plugin Interface

```go
type Plugin interface {
    Name() string
    Priority() int
    Detect(migrations []*types.Migration) bool
    Transform(sql string) (string, error)
    Validate(sql string) error
}
```

### Example: Custom Auth Plugin

```go
// CustomAuthPlugin detects custom JWT patterns
type CustomAuthPlugin struct {
    *plugins.BasePlugin
}

func (p *CustomAuthPlugin) Detect(migrations []*types.Migration) bool {
    for _, m := range migrations {
        for _, stmt := range m.Statements {
            // Custom pattern: auth.custom_user_id()
            if strings.Contains(stmt.SQL, "auth.custom_user_id()") {
                return true
            }
        }
    }
    return false
}
```

See [plugin-development.md](./plugin-development.md) for full plugin development guide.

## Performance Considerations

### Pattern Compilation Cost

- All patterns compiled once at package init
- Zero runtime compilation overhead
- Patterns reused across all migrations
- **Speedup:** 5-10× vs dynamic compilation

### Memory Usage

- Pattern objects are singletons
- Shared across all goroutines
- \~50KB total memory for all patterns
- Thread-safe by design (regex.Regexp is thread-safe)

### Large Migrations

For projects with 100+ migrations:

```bash
# Enable streaming mode for better performance
pgsquash squash \
  --stream \
  --workers 8 \
  --batch-size 50
```

See [configuration.md](./configuration.md) for streaming configuration.

## Troubleshooting

### Pattern Not Detecting

**Problem:** Plugin not activating despite matching patterns

**Solutions:**

1. Check plugin priority (higher priority plugins run first)
2. Verify pattern is case-insensitive where needed
3. Check for conflicting plugins
4. Enable debug logging: `PGSQUASH_LOG_LEVEL=debug pgsquash squash`

### Incorrect Dependency Order

**Problem:** Migrations execute in wrong order

**Solutions:**

1. Check for schema-qualified names (`public.users` vs `users`)
2. Verify foreign key syntax is standard
3. Use `--analyze` to inspect dependency graph
4. Consider using `--safety paranoid` to disable reordering

### Plugin Not Normalizing Syntax

**Problem:** ORM plugin not transforming SQL

**Solutions:**

1. Verify plugin is enabled in config
2. Check plugin detection patterns match your migrations
3. Use `pgsquash validate` to test plugin activation
4. Check plugin-specific config in `pgsquash.config.json`

## Related Documentation

- [Configuration Guide](./configuration.md) - Configure plugins and safety levels
- [Safety Levels](./safety-levels.md) - Understand consolidation aggressiveness
- [Plugin Development](./plugin-development.md) - Create custom plugins
- [Architecture](./architecture.md) - System architecture overview
- [Troubleshooting](./troubleshooting.md) - Common issues and solutions

## Summary

Pattern detection is the foundation of pgsquash's intelligence:

1. **Pre-compiled patterns** provide high performance (5-10× speedup)
2. **Category-based organization** makes patterns easy to understand
3. **Plugin system** enables extensibility without modifying core patterns
4. **Safety levels** control how patterns affect consolidation
5. **Dependency tracking** ensures correct migration order
6. **Validation** prevents issues before they reach production

The pattern system enables pgsquash to understand your migrations semantically, not just syntactically, leading to safer and more effective consolidation.
