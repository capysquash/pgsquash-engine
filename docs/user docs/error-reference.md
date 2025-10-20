# Error Reference

Complete reference for pgsquash error codes, categories, severities, and troubleshooting solutions.

## Table of Contents

- [Understanding pgsquash Errors](#understanding-pgsquash-errors)
- [Error Severity Levels](#error-severity-levels)
- [Error Categories](#error-categories)
- [Error Codes Reference](#error-codes-reference)
- [Exit Codes](#exit-codes)
- [Common Error Scenarios](#common-error-scenarios)
- [Troubleshooting Guide](#troubleshooting-guide)

---

## Understanding pgsquash Errors

pgsquash uses a structured error system that provides detailed context to help you quickly identify and fix issues. Every error includes:

- **Error Code**: Specific identifier (e.g., `SYNTAX_ERROR`, `VALIDATION_FAILED`)
- **Category**: Broad classification (e.g., Parsing, Validation, Dependency)
- **Severity**: Impact level (Info, Warning, Error, Critical)
- **Context**: File location, line number, object name, etc.
- **Suggestion**: Actionable advice to fix the issue

**Example Error Message:**

```
[ERROR:PARSING] code:SYNTAX_ERROR file:migrations/003_add_users.sql line:15
object:users Invalid SQL syntax detected
suggestion: Check for missing semicolons or unbalanced parentheses
```

---

## Error Severity Levels

| Severity | Description                                  | Action Required      |
| -------- | -------------------------------------------- | -------------------- |
| INFO     | Informational message, no action needed      | None                 |
| WARNING  | Potential issue, but processing can continue | Review recommended   |
| ERROR    | Serious issue, operation may fail            | Fix required         |
| CRITICAL | Fatal error, processing cannot continue      | Immediate fix needed |

**Processing Behavior:**

- **INFO/WARNING**: Logged but processing continues
- **ERROR**: Processing continues but operation may fail or produce suboptimal results
- **CRITICAL**: Stops processing immediately and exits with non-zero status code

---

## Error Categories

pgsquash organizes errors into categories to help identify the source of the problem:

### Core Categories

| Category       | Description                       | Common Causes                              |
| -------------- | --------------------------------- | ------------------------------------------ |
| PARSING        | SQL parsing and syntax errors     | Invalid SQL, missing semicolons            |
| VALIDATION     | Schema validation failures        | Schema mismatch, missing objects           |
| DEPENDENCY     | Dependency resolution issues      | Circular dependencies, missing references  |
| CONSOLIDATION  | Consolidation logic errors        | Conflicting rules, unsafe optimizations    |
| TRANSFORMATION | SQL transformation failures       | Backup/rollback generation errors          |
| SYNTAX         | SQL syntax errors                 | PostgreSQL syntax violations               |
| SEMANTIC       | SQL semantic errors               | Invalid object references, type mismatches |
| CONSTRAINT     | Constraint-related issues         | Constraint conflicts, validation failures  |
| FUNCTION       | Function-related errors           | Invalid function definitions               |
| INDEX          | Index-related errors              | Index creation failures                    |
| POLICY         | RLS policy errors                 | Policy conflicts, invalid conditions       |
| EXTENSION      | PostgreSQL extension errors       | Missing extensions, version mismatches     |
| PERFORMANCE    | Performance optimization warnings | Inefficient patterns, missing indexes      |
| NAMING         | Naming convention issues          | Reserved keywords, invalid identifiers     |
| PERMISSION     | Permission and security errors    | Missing grants, RLS issues                 |

### Additional Categories

| Category      | Description                       | When It Appears              |
| ------------- | --------------------------------- | ---------------------------- |
| CYCLE         | Circular dependency detection     | DDL cycle analysis           |
| OPTIMIZATION  | Optimization opportunities        | Analysis and recommendations |
| RISK          | Risk assessment warnings          | Safety level checks          |
| BACKUP        | Backup generation issues          | `--backup` flag              |
| ROLLBACK      | Rollback script generation issues | `--rollback` flag            |
| TYPE          | Type system errors                | Type analysis and validation |
| NORMALIZATION | SQL normalization issues          | SQL transformation           |

---

## Error Codes Reference

### Parsing Errors

#### `SYNTAX_ERROR`

**Category:** PARSING
**Severity:** ERROR

**Cause:** Invalid SQL syntax detected during parsing.

**Common Scenarios:**

- Missing semicolons at end of statements
- Unbalanced parentheses, brackets, or quotes
- Invalid PostgreSQL syntax or keywords
- Malformed CREATE/ALTER statements

**How to Fix:**

1. Run with `--verbose` to see the exact line number and context
2. Check the SQL syntax against [PostgreSQL documentation](https://www.postgresql.org/docs/current/sql.html)
3. Verify all statements end with semicolons
4. Check for balanced parentheses and quotes

**Example:**

```
[ERROR:PARSING] code:SYNTAX_ERROR file:migrations/003_add_users.sql line:15
Invalid SQL syntax detected
suggestion: Check for missing semicolons or unbalanced parentheses

-- Problem SQL:
CREATE TABLE users (id UUID PRIMARY KEY,  -- Missing closing parenthesis

-- Fixed SQL:
CREATE TABLE users (id UUID PRIMARY KEY);
```

---

#### `SEMANTIC_ERROR`

**Category:** PARSING
**Severity:** ERROR

**Cause:** SQL is syntactically valid but semantically incorrect.

**Common Scenarios:**

- Referencing non-existent tables or columns
- Type mismatches in expressions
- Invalid function calls or parameters
- Constraint violations

**How to Fix:**

1. Verify all referenced objects exist in prior migrations
2. Check data types match expected types
3. Ensure function signatures are correct
4. Review constraint definitions

**Example:**

```
[ERROR:PARSING] code:SEMANTIC_ERROR file:migrations/005_add_fk.sql line:8
object:orders Reference to undefined table 'customers'
suggestion: Ensure the 'customers' table is created in an earlier migration

-- Problem SQL:
ALTER TABLE orders ADD FOREIGN KEY (customer_id) REFERENCES customers(id);
-- But 'customers' table doesn't exist yet

-- Fix: Create customers table first or check migration order
```

---

#### `DEPENDENCY_ERROR`

**Category:** DEPENDENCY
**Severity:** ERROR

**Cause:** Unable to resolve object dependencies or circular dependency detected.

**Common Scenarios:**

- Circular foreign key dependencies
- Forward references to objects not yet created
- Mutual dependencies between objects
- DDL cycles (DROP → CREATE → DROP patterns)

**How to Fix:**

1. Run `pgsquash analyze --cycle-details` to see dependency graph
2. Reorder migrations to satisfy dependencies
3. Use two-phase constraint creation for circular foreign keys
4. Review DDL cycles and remove unnecessary DROP/CREATE sequences

**Example:**

```
[ERROR:DEPENDENCY] code:DEPENDENCY_ERROR
Circular dependency detected: users -> orders -> users
suggestion: Use ALTER TABLE to add foreign keys after both tables exist

-- Problem: Circular FK in single migration
CREATE TABLE users (id UUID, order_id UUID REFERENCES orders(id));
CREATE TABLE orders (id UUID, user_id UUID REFERENCES users(id));

-- Fix: Create tables first, add FKs later
CREATE TABLE users (id UUID PRIMARY KEY);
CREATE TABLE orders (id UUID PRIMARY KEY);
ALTER TABLE users ADD COLUMN order_id UUID REFERENCES orders(id);
ALTER TABLE orders ADD COLUMN user_id UUID REFERENCES users(id);
```

---

### Validation Errors

#### `VALIDATION_FAILED`

**Category:** VALIDATION
**Severity:** ERROR

**Cause:** Schema validation detected differences between original and squashed migrations.

**Common Scenarios:**

- Squashed schema differs from original
- Missing constraints in consolidated output
- Index definition mismatch
- Column order differences (may be acceptable)

**How to Fix:**

1. Run `pgsquash validate migrations/ clean/ --verbose` for detailed diff
2. Review the schema differences in the output
3. Try a more conservative safety level: `--safety conservative`
4. If differences are expected (e.g., column reordering), verify they're safe
5. If unexpected differences, file a bug report with the diff

**Example:**

```
[ERROR:VALIDATION] code:VALIDATION_FAILED
Schema mismatch detected between original and squashed migrations
suggestion: Run 'pgsquash validate --verbose' for detailed diff

-- Common cause: Missing constraint
Original schema:
  ALTER TABLE users ADD CONSTRAINT email_unique UNIQUE (email);

Squashed schema:
  -- Constraint missing!

-- Fix: Use conservative safety level or investigate consolidation issue
pgsquash squash migrations/*.sql --safety conservative
```

---

#### `SCHEMA_NOT_FOUND`

**Category:** VALIDATION
**Severity:** ERROR

**Cause:** Referenced schema does not exist.

**Common Scenarios:**

- Schema created in migration not being tracked
- Cross-schema references to missing schemas
- Plugin schemas (auth, storage) not detected

**How to Fix:**

1. Ensure all schema creation statements are in migrations
2. Check `include_schemas` in config includes the schema
3. For plugin schemas (Supabase auth, storage), enable the plugin
4. Verify schema exists in earlier migration

**Example:**

```
[ERROR:VALIDATION] code:SCHEMA_NOT_FOUND schema:reporting
Schema 'reporting' referenced but not found
suggestion: Add 'reporting' to include_schemas in config or create it in migrations

-- Fix in pgsquash.config.json:
{
  "include_schemas": ["public", "reporting"]
}

-- Or create schema in migration:
CREATE SCHEMA IF NOT EXISTS reporting;
```

---

### Transformation Errors

#### `ROLLBACK_GENERATION_FAILED`

**Category:** TRANSFORMATION
**Severity:** ERROR

**Cause:** Failed to generate rollback scripts when `--rollback` flag is used.

**Common Scenarios:**

- Complex transformations that can't be automatically reversed
- Non-reversible operations (DROP without backup)
- Insufficient information to generate reverse operation

**How to Fix:**

1. Review the specific operation that failed
2. Manually create rollback scripts if needed
3. Use `--backup` flag in addition to `--rollback`
4. Consider more conservative safety level

---

#### `BACKUP_GENERATION_FAILED`

**Category:** TRANSFORMATION
**Severity:** ERROR

**Cause:** Failed to generate backup when `--backup` flag is used.

**Common Scenarios:**

- Database connection issues (requires `prod_db_dsn`)
- Insufficient permissions to read schema
- Database unreachable

**How to Fix:**

1. Verify `prod_db_dsn` is set correctly:
   ```bash
   export PROD_DB_DSN="postgresql://user:pass@localhost:5432/db"
   ```
2. Test database connection:
   ```bash
   psql $PROD_DB_DSN -c "SELECT version();"
   ```
3. Check user has SELECT permissions on all tables
4. Ensure database is accessible from current network

---

### Consolidation Errors

#### `CONSOLIDATION_FAILED`

**Category:** CONSOLIDATION
**Severity:** ERROR

**Cause:** Consolidation rules failed to process an object.

**Common Scenarios:**

- Conflicting consolidation rules
- Object state inconsistency
- Unsupported SQL patterns

**How to Fix:**

1. Run with `--verbose` to see which rule failed
2. Try lower safety level: `--safety conservative`
3. Check for unusual SQL patterns or PostgreSQL extensions
4. Report issue with migration samples if bug suspected

---

#### `SQL_GENERATION_FAILED`

**Category:** CONSOLIDATION
**Severity:** ERROR

**Cause:** Failed to generate valid SQL from consolidated objects.

**Common Scenarios:**

- Complex object transformations
- Missing metadata
- Unsupported PostgreSQL features

**How to Fix:**

1. Review the specific object that failed
2. Try `--dry-run` to preview without writing
3. Use more conservative safety level
4. Check PostgreSQL version compatibility in config

---

### Type Errors

#### `INVALID_TYPE`

**Category:** TYPE
**Severity:** ERROR

**Cause:** Invalid data type specified or type not supported.

**Common Scenarios:**

- Typo in type name (e.g., `INTGER` instead of `INTEGER`)
- Custom types not defined
- PostgreSQL version mismatch

**How to Fix:**

1. Verify type spelling matches PostgreSQL documentation
2. Ensure custom types are created before use
3. Check `postgresql_features.target_version` in config
4. Verify required extensions are loaded

---

#### `TYPE_NOT_FOUND`

**Category:** TYPE
**Severity:** ERROR

**Cause:** Referenced type does not exist.

**Common Scenarios:**

- Custom ENUM or COMPOSITE type not created
- Extension type not loaded (e.g., `vector` type)
- Type defined in different schema

**How to Fix:**

1. Create the type before using it
2. Load required extensions:
   ```sql
   CREATE EXTENSION IF NOT EXISTS vector;
   ```
3. Check type is in correct schema
4. Enable auto-extension detection in config

---

## Exit Codes

pgsquash uses standard exit codes for scripting and CI/CD integration:

| Exit Code | Meaning             | Description                            |
| --------- | ------------------- | -------------------------------------- |
| 0         | Success             | Operation completed successfully       |
| 1         | General error       | Unspecified error occurred             |
| 2         | Parse error         | SQL parsing failed                     |
| 3         | Validation failed   | Schema validation detected differences |
| 4         | Circular dependency | Unresolvable circular dependency       |
| 5         | Configuration error | Invalid configuration                  |
| 6         | File I/O error      | Cannot read/write migration files      |
| 7         | Database error      | Database connection or query failed    |

**Usage in Scripts:**

```bash
#!/bin/bash
set -e

pgsquash validate migrations/ clean/
EXIT_CODE=$?

if [ $EXIT_CODE -eq 3 ]; then
    echo "Validation failed - schemas don't match"
    exit 1
elif [ $EXIT_CODE -ne 0 ]; then
    echo "Unexpected error: $EXIT_CODE"
    exit 1
fi

echo "Validation passed!"
```

---

## Common Error Scenarios

### Scenario 1: "Parse error: unexpected token"

**Error:**

```
[ERROR:PARSING] code:SYNTAX_ERROR
Unexpected token at position 45
```

**Causes:**

- Missing semicolon at end of statement
- PostgreSQL-specific syntax not recognized
- Incomplete statement

**Solutions:**

```bash
# 1. Run with verbose to see exact location
pgsquash analyze migrations/*.sql --verbose

# 2. Check for semicolons
# Every statement should end with ;

# 3. Verify PostgreSQL syntax
# Consult PostgreSQL documentation for your version
```

---

### Scenario 2: "Validation failed: schema mismatch"

**Error:**

```
[ERROR:VALIDATION] code:VALIDATION_FAILED
Schema mismatch detected
```

**Solutions:**

```bash
# 1. See detailed diff
pgsquash validate migrations/ clean/ --verbose

# 2. Try conservative mode
pgsquash squash migrations/*.sql --safety conservative --output clean/

# 3. Check Docker is running (required for validation)
docker ps

# 4. Review specific differences
# Validate shows exact schema differences
```

---

### Scenario 3: "Circular dependency detected"

**Error:**

```
[ERROR:DEPENDENCY] code:DEPENDENCY_ERROR
Circular dependency: table_a -> table_b -> table_a
```

**Solutions:**

```bash
# 1. Analyze dependency graph
pgsquash analyze migrations/*.sql --detect-cycles --cycle-details

# 2. Use two-phase constraint creation
# Create tables first, add foreign keys later

# 3. Enable DDL cycle detection
pgsquash squash migrations/*.sql --detect-cycles
```

---

### Scenario 4: "Extension not found"

**Error:**

```
[ERROR:EXTENSION] code:VALIDATION_FAILED
Extension 'vector' not available
```

**Solutions:**

```bash
# 1. Enable extension auto-detection
# In pgsquash.config.json:
{
  "validation": {
    "enable_extension_detection": true,
    "auto_install_extensions": true
  }
}

# 2. Specify extensions manually
{
  "postgresql_features": {
    "enabled_extensions": ["vector", "uuid-ossp"]
  }
}

# 3. Use appropriate Docker image
{
  "validation": {
    "docker_image": "ankane/pgvector:latest"
  }
}
```

---

## Troubleshooting Guide

### Getting More Information

**Enable verbose mode:**

```bash
pgsquash [command] --verbose
```

**Check specific file:**

```bash
pgsquash analyze migrations/003_problem_file.sql --verbose
```

**Analyze dependencies:**

```bash
pgsquash analyze migrations/*.sql --detect-cycles --cycle-details
```

---

### Performance Issues

**For large migration sets (500+ files):**

```bash
# Enable streaming mode
pgsquash squash migrations/*.sql --streaming --memory-limit 512

# Increase batch size
pgsquash squash migrations/*.sql --batch-size 100

# Adjust worker count
pgsquash squash migrations/*.sql --workers 8
```

---

### Validation Troubleshooting

**Validation timeout:**

```json
{
  "validation": {
    "timeout_seconds": 300,
    "container_ready_timeout": 60
  }
}
```

**Extension detection issues:**

```json
{
  "validation": {
    "enable_extension_detection": true,
    "auto_install_extensions": true,
    "verbose": true
  }
}
```

**Try different validation mode:**

```bash
# Fastest
pgsquash validate migrations/ clean/ --validation-mode SCHEMA_DIFF

# Most accurate
pgsquash validate migrations/ clean/ --validation-mode TWO_CONTAINERS
```

---

### AI-Related Errors

**API key not set:**

```bash
export ANTHROPIC_API_KEY="sk-ant-..."
# or
export OPENAI_API_KEY="sk-..."
```

**AI timeout:**

```json
{
  "ai": {
    "timeout_seconds": 120,
    "max_retries": 5
  }
}
```

---

### Configuration Errors

**Invalid config file:**

```bash
# Generate fresh config
pgsquash init-config --force

# Validate config by running
pgsquash analyze migrations/*.sql --config pgsquash.config.json
```

**JSON syntax errors:**

The config loader provides detailed error messages with line numbers and context. Look for:

- Missing or extra commas
- Unbalanced brackets
- Invalid JSON types

---

## Getting Help

If you encounter an error not covered here:

1. **Check verbose output:**
   ```bash
   pgsquash [command] --verbose
   ```

2. **Search documentation:**
   - [Troubleshooting Guide](troubleshooting.md)
   - [Configuration Reference](configuration.md)
   - [CLI Reference](cli-reference.md)

3. **Review migration files:**
   - Check for non-standard SQL
   - Verify PostgreSQL version compatibility
   - Look for plugin-specific patterns

4. **File an issue:**
   - Include error message with full context
   - Provide sample migration (if possible)
   - Specify PostgreSQL version and pgsquash version
   - Include configuration file (remove sensitive data)

---

## See Also

- [CLI Reference](cli-reference.md) - Complete command documentation
- [Configuration](configuration.md) - Configuration options
- [Troubleshooting](troubleshooting.md) - General troubleshooting
- [Safety Levels](safety-levels.md) - Understanding safety modes
- [Architecture](architecture.md) - How pgsquash processes migrations
