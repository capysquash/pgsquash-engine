# Paranoid Mode Validation

pgsquash's **paranoid mode** provides comprehensive database validation by comparing generated SQL against your production database schema using the Metadata Manager system.

## Overview

Paranoid mode is the most conservative safety level, designed for mission-critical production deployments. It performs live database introspection and schema comparison to ensure squashed migrations are 100% compatible with your existing database.

## How It Works

### 1. Database Connection

Paranoid mode requires a connection to your production database:

```bash
export PROD_DB_DSN="postgres://user:password@localhost:5432/dbname"
pgsquash squash migrations/*.sql --safety paranoid
```

Or configure in `pgsquash.config.json`:

```json
{
  "safety_level": "paranoid",
  "prod_db_dsn": "postgres://user:password@localhost:5432/dbname"
}
```

### 2. Metadata Collection

The MetadataManager introspects your database and collects:

- **Extensions**: Installed PostgreSQL extensions (pgvector, pg\_trgm, etc.)
- **Schemas**: All user schemas (excluding system schemas)
- **Tables**: Complete table metadata including:
  - Columns with data types, nullability, defaults
  - Constraints (PRIMARY KEY, FOREIGN KEY, UNIQUE, CHECK)
  - Indexes with methods and predicates
  - Triggers and their functions
  - RLS policies
- **Functions**: Function signatures, parameters, return types, volatility
- **Views**: View definitions and dependencies
- **Materialized Views**: With data status and indexes
- **Sequences**: Sequence parameters and ownership
- **Types**: Custom types (ENUMs, COMPOSITE, DOMAIN, RANGE)

### 3. Schema Comparison

The SchemaComparator performs comprehensive validation:

```
Generated SQL → Parse → Compare Against Database Metadata → Validation Report
```

#### Validation Checks

1. **Extension Validation**
   - Verifies all required extensions exist in database
   - Reports missing extensions as errors

2. **Dependency Validation**
   - Checks that all referenced objects exist
   - Validates cross-schema references
   - Distinguishes between hard errors and optional dependencies

3. **Table Validation**
   - Verifies tables exist in database
   - Detects schema drift (tables in migration but not in database)

4. **Function Validation**
   - Checks function existence in database
   - Reports missing functions as drift

5. **Type Validation**
   - Validates custom type existence
   - Detects missing ENUMs, COMPOSITE types, etc.

6. **Breaking Change Detection**
   - Identifies changes that could break existing applications
   - Provides impact assessment and mitigation strategies

## Validation Results

### Success Output

```
✓ Database validation passed: schema is compatible
Database validation completed: Extensions=0, Dependencies=0, TypeMismatches=0, Constraints=0, Breaking=0, Drift=0
```

### Error Output

```
⚠ Missing extensions: [pgvector]
⚠ Missing dependencies detected: 2
⚠ Type mismatches detected: 1
❌ Breaking changes detected: 1
❌ Database validation failed: schema incompatibilities detected
```

## ComparisonResult Structure

The validation produces a detailed ComparisonResult:

```go
type ComparisonResult struct {
    IsValid            bool
    MissingExtensions  []string
    MissingDependencies []MissingDependency
    TypeMismatches     []TypeMismatch
    ConstraintConflicts []ConstraintConflict
    BreakingChanges    []BreakingChange
    Warnings           []string
    SchemaDrift        []SchemaDrift
}
```

### MissingDependency

```go
type MissingDependency struct {
    ObjectName   string           // e.g., "public.users"
    ObjectType   types.ObjectType // TABLE, FUNCTION, etc.
    ReferencedBy string           // SQL snippet where dependency is used
    Severity     string           // "error" or "warning"
}
```

**Example**:

```
ERROR: TABLE dependency 'public.audit_log' not found in database (referenced by: ALTER TABLE events ADD CONSTRAINT...)
```

### TypeMismatch

```go
type TypeMismatch struct {
    Object         string // "public.users"
    Column         string // "email"
    ExpectedType   string // "text"
    ActualType     string // "varchar(255)"
    IsBreaking     bool   // true if incompatible cast
}
```

**Example**:

```
Type mismatch in public.users.age: migration expects integer but database has text
```

### BreakingChange

```go
type BreakingChange struct {
    Description string // What changed
    Impact      string // Consequences
    Mitigation  string // How to fix
}
```

**Example**:

```
BREAKING: Column public.users.email type mismatch: migration expects text but database has integer
| Impact: Queries may fail or return unexpected results
| Mitigation: Run ALTER TABLE to change column type or adjust migration
```

### SchemaDrift

```go
type SchemaDrift struct {
    Object      string // Object name
    ObjectType  string // TABLE, FUNCTION, TYPE, etc.
    Description string // Drift description
    DriftType   string // "missing_in_db", "extra_in_db", "definition_mismatch"
}
```

**Example**:

```
Schema drift (missing_in_db): TABLE users_new - Table users_new defined in migration but not found in database
```

## Use Cases

### 1. Production Deployment Safety

Before deploying squashed migrations to production:

```bash
# Squash with paranoid validation
pgsquash squash migrations/*.sql --safety paranoid --output clean/

# If validation passes, deploy clean/
psql $PROD_DB_URL < clean/squashed.sql
```

### 2. Schema Drift Detection

Detect when migrations diverge from production:

```bash
pgsquash squash migrations/*.sql --safety paranoid --dry-run
```

Review drift warnings to understand discrepancies between migration definitions and actual database state.

### 3. Breaking Change Analysis

Identify breaking changes before they reach production:

```bash
pgsquash squash migrations/*.sql --safety paranoid 2>&1 | grep "BREAKING:"
```

### 4. Dependency Validation

Ensure all migration dependencies exist:

```bash
pgsquash squash migrations/*.sql --safety paranoid 2>&1 | grep "dependency"
```

## Configuration

### Basic Configuration

```json
{
  "safety_level": "paranoid",
  "prod_db_dsn": "postgres://user:password@localhost:5432/dbname"
}
```

### Advanced Configuration

```json
{
  "safety_level": "paranoid",
  "prod_db_dsn": "postgres://user:password@localhost:5432/dbname",

  "validation": {
    "enable_extension_detection": true,
    "enable_sql_fixes": false
  },

  "rules": {
    "table_operations": {
      "consolidate_create_alter": false,
      "remove_drop_create_cycles": false,
      "preserve_data_operations": true
    }
  }
}
```

## Metadata Caching

The MetadataManager implements intelligent caching:

- **Cache TTL**: 15 minutes (default)
- **Cache Invalidation**: Automatic on TTL expiry
- **Cache Statistics**: Hit/miss ratio tracking

```go
hits, misses, ratio := metadataManager.GetCacheStats()
// Example: hits=100, misses=5, ratio=0.95 (95% hit rate)
```

## Performance Considerations

### Database Queries

Paranoid mode executes database queries to introspect schema:

- **Schemas**: 1 query per run
- **Tables**: 1 query per schema + detailed queries per table
- **Functions**: 1 query per schema
- **Extensions**: 1 query per run
- **Types**: 1 query per schema

**Typical overhead**: 50-200ms for small databases, 500-2000ms for large databases (1000+ tables).

### Optimization Tips

1. **Use Caching**: Metadata is cached for 15 minutes
2. **Limit Schema Scope**: Exclude unused schemas if possible
3. **Run Validation Separately**: Use `--dry-run` to validate without squashing

## Limitations

### Current Limitations

1. **Column Type Comparison**: Simplified type comparison (exact matches and common aliases)
2. **Constraint Comparison**: Basic constraint validation (full constraint definition parsing not yet implemented)
3. **View Dependencies**: View dependency analysis requires full SQL parsing

### Future Enhancements

Planned improvements for schema comparison:

- **Deep Column Comparison**: Full column type, nullability, default value comparison
- **Constraint Equivalence**: Sophisticated constraint definition comparison
- **Index Optimization**: Index usage analysis and recommendations
- **Trigger Validation**: Trigger function signature and body comparison
- **RLS Policy Comparison**: Row-level security policy equivalence checking

## Error Handling

### Non-Fatal Warnings

Validation continues with warnings:

```
WARNING: TABLE dependency 'audit_log' not found (IF NOT EXISTS clause present)
```

### Fatal Errors

Validation fails with errors:

```
ERROR: Extension 'pgvector' required but not installed in database
❌ Database validation failed: found 1 errors, 2 warnings, 1 breaking changes
```

## Security Considerations

### Database Credentials

- Store credentials in environment variables: `PROD_DB_DSN`
- Never commit credentials to version control
- Use read-only database user for validation

### Production Database Access

- Paranoid mode only performs **read queries**
- No DDL or DML statements are executed
- Safe to run against production databases

### Recommended Permissions

```sql
-- Create read-only user for validation
CREATE USER pgsquash_validator WITH PASSWORD 'secure_password';

-- Grant schema-level read access
GRANT USAGE ON SCHEMA public TO pgsquash_validator;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO pgsquash_validator;

-- Grant access to system catalogs
GRANT SELECT ON pg_extension TO pgsquash_validator;
GRANT SELECT ON pg_namespace TO pgsquash_validator;
GRANT SELECT ON pg_class TO pgsquash_validator;
GRANT SELECT ON pg_proc TO pgsquash_validator;
GRANT SELECT ON pg_type TO pgsquash_validator;
```

## Best Practices

### 1. Always Use for Production

Always use paranoid mode before deploying to production:

```bash
pgsquash squash migrations/*.sql --safety paranoid --output production/
```

### 2. Test Against Staging First

Validate against staging database before production:

```bash
export PROD_DB_DSN="$STAGING_DB_URL"
pgsquash squash migrations/*.sql --safety paranoid --dry-run
```

### 3. Review All Warnings

Even non-fatal warnings should be reviewed:

```bash
pgsquash squash migrations/*.sql --safety paranoid 2>&1 | grep "WARNING:"
```

### 4. Document Drift Exceptions

If drift is expected, document it:

```bash
# Expected drift: new table 'analytics' not yet deployed
pgsquash squash migrations/*.sql --safety paranoid 2>&1 | tee validation.log
```

## Troubleshooting

### Connection Errors

```
ERROR: failed to get database metadata: connection refused
```

**Solution**: Verify `PROD_DB_DSN` is correct and database is accessible.

### Permission Denied

```
ERROR: permission denied for schema public
```

**Solution**: Grant read permissions to validation user (see Security Considerations).

### High Validation Time

```
Database validation completed: Extensions=0, Dependencies=0... (took 5.2s)
```

**Solution**: Reduce schema scope or use cached metadata.

## Further Reading

- [Safety Levels](./safety-levels.md) - Complete safety level documentation
- [Configuration](./configuration.md) - Full configuration reference
- [Validation](./validation.md) - Docker-based validation modes
- [Architecture](./architecture.md) - System architecture overview
