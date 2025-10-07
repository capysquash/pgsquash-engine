# Configuration Reference

Configuration options for `pgsquash.config.json`.

## Table of Contents

- [Configuration File](#configuration-file)
- [Top-Level Settings](#top-level-settings)
- [Output Configuration](#output-configuration)
- [Rules Configuration](#rules-configuration)
- [Performance Settings](#performance-settings)
- [Modern Features](#modern-features)
- [Conflict Resolution](#conflict-resolution)
- [PostgreSQL Features](#postgresql-features)
- [Third-Party Integrations](#third-party-integrations)
- [Configuration Examples](#configuration-examples)

## Configuration File

### Location

pg-squash searches for configuration in this order:

1. Path specified with `--config` flag
2. `./pgsquash.config.json` in current directory
3. Default embedded configuration

### Generation

```bash
pgsquash init-config
pgsquash init-config --config custom.json  # Custom path
```

### Structure

```json
{
  "safety_level": "standard",
  "prod_db_dsn": "",
  "output": { ... },
  "rules": { ... },
  "exclude_patterns": [],
  "include_schemas": [],
  "performance": { ... },
  "modern_features": { ... },
  "conflict_resolution": { ... },
  "postgresql_features": { ... },
  "third_party_integrations": { ... }
}
```

## Top-Level Settings

### safety_level

**Type**: `string`
**Default**: `"standard"`
**Options**: `"paranoid"`, `"conservative"`, `"standard"`, `"aggressive"`

Controls the level of optimization and consolidation applied.

```json
{
  "safety_level": "standard"
}
```

- `paranoid`: Minimal changes, requires DB connection
- `conservative`: CREATE + ALTER consolidation only
- `standard`: Balanced optimizations (default)
- `aggressive`: Maximum optimization

[Safety Levels Guide](safety-levels.md)

### prod_db_dsn

**Type**: `string`
**Default**: `""` (from environment variable `PROD_DB_DSN`)
**Format**: PostgreSQL connection string

Database connection string for paranoid mode and backup generation.

```json
{
  "prod_db_dsn": "postgres://user:password@localhost:5432/database"
}
```

**Usage**:
- Required for `paranoid` safety level
- Required for `--backup` functionality
- Used for dead code analysis
- Optional for other safety levels

Use environment variable for security:
```bash
export PROD_DB_DSN="postgres://..."
```

### exclude_patterns

**Type**: `array of strings`
**Default**: `["auth.*", "extensions.*"]`

Patterns for objects to exclude from consolidation.

```json
{
  "exclude_patterns": [
    "auth.*",
    "extensions.*",
    "temp_*",
    "test_*"
  ]
}
```

Pattern syntax:
- `*` - Wildcard
- `auth.*` - Matches `auth.` prefix
- `*.temp` - Matches `.temp` suffix

### include_schemas

**Type**: `array of strings`
**Default**: `["public"]`

Schemas to include in processing.

```json
{
  "include_schemas": [
    "public",
    "app",
    "reporting"
  ]
}
```

- `["*"]` - All schemas
- `["public"]` - Public only (default)
- `["public", "app"]` - Specific schemas

## Output Configuration

### format

**Type**: `string`
**Default**: `"organized"`
**Options**: `"organized"`, `"sequential"`, `"minimal"`

Output format style.

```json
{
  "output": {
    "format": "organized"
  }
}
```

- `organized` - Groups by category
- `sequential` - Original order
- `minimal` - Compact output

### preserve_comments

**Type**: `boolean`
**Default**: `true`

Whether to preserve original SQL comments.

```json
{
  "output": {
    "preserve_comments": true
  }
}
```

**Example**:
```sql
-- This is an important user table
CREATE TABLE users (
    id UUID PRIMARY KEY
);
```

### add_consolidation_comments

**Type**: `boolean`
**Default**: `true`

Add comments explaining consolidation actions.

```json
{
  "output": {
    "add_consolidation_comments": true
  }
}
```

**Example Output**:
```sql
-- Consolidated from migrations: 001_create_users.sql, 003_add_user_fields.sql
CREATE TABLE users (
    id UUID PRIMARY KEY,
    email VARCHAR(255) NOT NULL  -- Added in migration 003
);
```

### file_naming

**Type**: `string`
**Default**: `"semantic"`
**Options**: `"semantic"`, `"sequential"`, `"timestamp"`

Output file naming strategy.

```json
{
  "output": {
    "file_naming": "semantic"
  }
}
```

**Options**:
- **semantic**: `001_schema_foundation.sql`, `002_constraints.sql`, etc.
- **sequential**: `001.sql`, `002.sql`, etc.
- **timestamp**: `20250101_120000.sql`

### directory

**Type**: `string`
**Default**: `"squashed"`

Output directory path.

```json
{
  "output": {
    "directory": "clean_migrations"
  }
}
```

## Rules Configuration

Controls consolidation behavior for different object types.

### Table Operations

```json
{
  "rules": {
    "table_operations": {
      "consolidate_create_alter": true,
      "remove_drop_create_cycles": true,
      "preserve_data_operations": true
    }
  }
}
```

#### consolidate_create_alter

**Type**: `boolean`
**Default**: `true`

Merge CREATE TABLE and subsequent ALTER TABLE statements.

**Example**:
```sql
-- Original
CREATE TABLE users (id UUID PRIMARY KEY);
ALTER TABLE users ADD COLUMN email VARCHAR(255);
ALTER TABLE users ADD COLUMN name VARCHAR(255);

-- Consolidated
CREATE TABLE users (
    id UUID PRIMARY KEY,
    email VARCHAR(255),
    name VARCHAR(255)
);
```

#### remove_drop_create_cycles

**Type**: `boolean`
**Default**: `true`

Remove redundant DROP → CREATE sequences.

**Example**:
```sql
-- Original
CREATE TABLE temp_data (...);
INSERT INTO temp_data ...;
DROP TABLE temp_data;
CREATE TABLE temp_data (...);  -- Different structure

-- Consolidated
CREATE TABLE temp_data (...);  -- Final version only
```

#### preserve_data_operations

**Type**: `boolean`
**Default**: `true`

Always preserve INSERT, UPDATE, DELETE operations.

```json
{
  "rules": {
    "table_operations": {
      "preserve_data_operations": true
    }
  }
}
```

### Index Operations

```json
{
  "rules": {
    "index_operations": {
      "consolidate_recreations": true,
      "preserve_unique_constraints": true
    }
  }
}
```

#### consolidate_recreations

**Type**: `boolean`
**Default**: `true`

Remove index DROP → CREATE cycles.

**Example**:
```sql
-- Original
CREATE INDEX idx_email ON users(email);
DROP INDEX idx_email;
CREATE INDEX idx_email ON users(email, status);

-- Consolidated
CREATE INDEX idx_email ON users(email, status);
```

#### preserve_unique_constraints

**Type**: `boolean`
**Default**: `true`

Always preserve UNIQUE constraints even if index is recreated.

### Function Operations

```json
{
  "rules": {
    "function_operations": {
      "remove_duplicate_definitions": true,
      "preserve_signature_changes": false
    }
  }
}
```

#### remove_duplicate_definitions

**Type**: `boolean`
**Default**: `true`

Remove duplicate function definitions (same name and signature).

**Example**:
```sql
-- Original
CREATE FUNCTION count_users() RETURNS INTEGER ...;
CREATE OR REPLACE FUNCTION count_users() RETURNS INTEGER ...;

-- Consolidated
CREATE FUNCTION count_users() RETURNS INTEGER ...;
```

#### preserve_signature_changes

**Type**: `boolean`
**Default**: `false`

Keep all versions if function signature changes.

```json
{
  "rules": {
    "function_operations": {
      "preserve_signature_changes": true
    }
  }
}
```

**When true**:
```sql
-- Keeps both versions if signature differs
CREATE FUNCTION count_users() RETURNS INTEGER ...;
CREATE FUNCTION count_users(status TEXT) RETURNS INTEGER ...;
```

## Performance Settings

Optimize processing performance and resource usage.

```json
{
  "performance": {
    "streaming_threshold_mb": 5,
    "parallel_processing": true,
    "show_progress": true
  }
}
```

### streaming_threshold_mb

**Type**: `integer`
**Default**: `5`

Automatically enable streaming mode when total migration size exceeds this threshold (in MB).

```json
{
  "performance": {
    "streaming_threshold_mb": 10
  }
}
```

**Effect**:
- Below threshold: Load all migrations into memory
- Above threshold: Process in batches for memory efficiency

### parallel_processing

**Type**: `boolean`
**Default**: `true`

Enable parallel processing of migrations.

```json
{
  "performance": {
    "parallel_processing": true
  }
}
```

**Benefits**:
- Faster parsing of multiple files
- Concurrent consolidation processing
- Better CPU utilization

**Workers**: Automatically set to number of CPU cores

### show_progress

**Type**: `boolean`
**Default**: `true`

Display progress indicators during processing.

```json
{
  "performance": {
    "show_progress": true
  }
}
```

## Modern Features

Enable support for modern PostgreSQL features.

```json
{
  "modern_features": {
    "enable_vector_support": true,
    "enable_generated_columns": true,
    "enable_event_sourcing": true,
    "enable_merge_statements": true,
    "enable_multirange_types": true,
    "enable_advanced_rls": true
  }
}
```

### enable_vector_support

**Type**: `boolean`
**Default**: `true`

Support for pgvector extension and vector operations.

```json
{
  "modern_features": {
    "enable_vector_support": true
  }
}
```

**Features**:
- Vector column types
- Vector indexes (ivfflat, hnsw)
- Vector operations (cosine, L2, inner product)

### enable_generated_columns

**Type**: `boolean`
**Default**: `true`

Support for PostgreSQL 12+ generated columns.

```json
{
  "modern_features": {
    "enable_generated_columns": true
  }
}
```

**Example**:
```sql
CREATE TABLE products (
    price NUMERIC,
    tax_rate NUMERIC,
    total_price NUMERIC GENERATED ALWAYS AS (price * (1 + tax_rate)) STORED
);
```

### enable_event_sourcing

**Type**: `boolean`
**Default**: `true`

Support for event sourcing patterns.

```json
{
  "modern_features": {
    "enable_event_sourcing": true
  }
}
```

**Features**:
- Event tables
- Aggregate views
- Event triggers

### enable_merge_statements

**Type**: `boolean`
**Default**: `true`

Support for SQL MERGE statements (PostgreSQL 15+).

```json
{
  "modern_features": {
    "enable_merge_statements": true
  }
}
```

### enable_multirange_types

**Type**: `boolean`
**Default**: `true`

Support for multirange types (PostgreSQL 14+).

```json
{
  "modern_features": {
    "enable_multirange_types": true
  }
}
```

**Example**:
```sql
CREATE TABLE bookings (
    id UUID PRIMARY KEY,
    booked_ranges INT4MULTIRANGE
);
```

### enable_advanced_rls

**Type**: `boolean`
**Default**: `true`

Support for advanced Row Level Security features.

```json
{
  "modern_features": {
    "enable_advanced_rls": true
  }
}
```

**Features**:
- Policy chaining
- Dynamic policies
- Policy-based permissions

## Conflict Resolution

Configure how pg-squash handles rule conflicts.

```json
{
  "conflict_resolution": {
    "enable_priority_system": true,
    "strict_mode_enabled": false,
    "allow_overlapping_rules": false,
    "conflict_log_level": "warn"
  }
}
```

### enable_priority_system

**Type**: `boolean`
**Default**: `true`

Use rule priority to resolve conflicts.

```json
{
  "conflict_resolution": {
    "enable_priority_system": true
  }
}
```

**Behavior**:
- Rules with higher priority apply first
- Lower priority rules skipped if conflict detected

### strict_mode_enabled

**Type**: `boolean`
**Default**: `false`

Fail on any rule conflicts instead of resolving.

```json
{
  "conflict_resolution": {
    "strict_mode_enabled": true
  }
}
```

**When true**:
- Any rule conflict causes processing to fail
- Requires manual conflict resolution
- Maximum safety for critical migrations

### allow_overlapping_rules

**Type**: `boolean`
**Default**: `false`

Allow multiple rules to apply to same object.

```json
{
  "conflict_resolution": {
    "allow_overlapping_rules": true
  }
}
```

### conflict_log_level

**Type**: `string`
**Default**: `"warn"`
**Options**: `"debug"`, `"info"`, `"warn"`, `"error"`

Logging level for conflict detection.

```json
{
  "conflict_resolution": {
    "conflict_log_level": "info"
  }
}
```

## PostgreSQL Features

Configure PostgreSQL version compatibility and features.

```json
{
  "postgresql_features": {
    "target_version": "15",
    "enabled_extensions": [
      "uuid-ossp",
      "vector",
      "pg_stat_statements"
    ],
    "optimize_for_performance": true,
    "use_modern_syntax": true,
    "validate_compatibility": true
  }
}
```

### target_version

**Type**: `string`
**Default**: `"15"`

Target PostgreSQL version for output SQL.

```json
{
  "postgresql_features": {
    "target_version": "16"
  }
}
```

**Supported Versions**: `"12"`, `"13"`, `"14"`, `"15"`, `"16"`, `"17"`

**Effect**:
- Uses version-appropriate syntax
- Enables version-specific features
- Validates compatibility

### enabled_extensions

**Type**: `array of strings`
**Default**: `["uuid-ossp", "vector", "pg_stat_statements"]`

Extensions to expect and validate.

```json
{
  "postgresql_features": {
    "enabled_extensions": [
      "uuid-ossp",
      "vector",
      "postgis",
      "pg_trgm"
    ]
  }
}
```

**Common Extensions**:
- `uuid-ossp`: UUID generation
- `vector`: pgvector for vector operations
- `pg_stat_statements`: Query statistics
- `postgis`: Geographic data
- `pg_trgm`: Fuzzy text matching
- `hstore`: Key-value storage

### optimize_for_performance

**Type**: `boolean`
**Default**: `true`

Apply performance optimizations to output SQL.

```json
{
  "postgresql_features": {
    "optimize_for_performance": true
  }
}
```

**Optimizations**:
- Optimal index placement
- Efficient constraint ordering
- Batch operation consolidation

### use_modern_syntax

**Type**: `boolean`
**Default**: `true`

Use modern PostgreSQL syntax in output.

```json
{
  "postgresql_features": {
    "use_modern_syntax": true
  }
}
```

**Examples**:
```sql
-- Modern syntax
CREATE TABLE IF NOT EXISTS users ...;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_email ...;

-- Legacy syntax
CREATE TABLE users ...;
CREATE INDEX idx_email ...;
```

### validate_compatibility

**Type**: `boolean`
**Default**: `true`

Validate output is compatible with target version.

```json
{
  "postgresql_features": {
    "validate_compatibility": true
  }
}
```

## Third-Party Integrations

Configure integrations with authentication providers and services.

### Supabase Integration

```json
{
  "third_party_integrations": {
    "supabase_integration": {
      "enabled": true,
      "jwt_secret": "",
      "enable_rls": true,
      "storage_integration": true
    }
  }
}
```

**Fields**:
- `enabled`: Enable Supabase-specific handling
- `jwt_secret`: JWT secret for validation (optional)
- `enable_rls`: Process RLS policies
- `storage_integration`: Handle storage policies

### Clerk Integration

```json
{
  "third_party_integrations": {
    "clerk_integration": {
      "enabled": false,
      "jwt_version": "v2",
      "organization_support": true,
      "public_metadata_support": true
    }
  }
}
```

**Fields**:
- `enabled`: Enable Clerk authentication patterns
- `jwt_version`: JWT token version (`"v1"`, `"v2"`)
- `organization_support`: Handle organization data
- `public_metadata_support`: Handle public metadata

### Auth0 Integration

```json
{
  "third_party_integrations": {
    "auth0_integration": {
      "enabled": false,
      "domain": "",
      "custom_claims": ["permissions", "role"],
      "role_claim_path": "https://myapp.com/role"
    }
  }
}
```

### NextAuth Integration

```json
{
  "third_party_integrations": {
    "nextauth_integration": {
      "enabled": false,
      "session_strategy": "database",
      "database_tables": ["accounts", "sessions", "users"]
    }
  }
}
```

### Vector Integration (pgvector)

```json
{
  "third_party_integrations": {
    "vector_integration": {
      "enabled": true,
      "default_index_type": "ivfflat",
      "optimize_queries": true,
      "supported_ops": [
        "vector_cosine_ops",
        "vector_l2_ops",
        "vector_ip_ops"
      ]
    }
  }
}
```

**Index Types**:
- `ivfflat`: Inverted file with flat compression
- `hnsw`: Hierarchical Navigable Small World

### PlanetScale Integration

```json
{
  "third_party_integrations": {
    "planetscale_integration": {
      "enabled": false,
      "disable_foreign_keys": true,
      "optimize_for_replication": true
    }
  }
}
```

## Configuration Examples

### Production Configuration

Conservative, safe configuration for production:

```json
{
  "safety_level": "conservative",
  "output": {
    "format": "organized",
    "preserve_comments": true,
    "add_consolidation_comments": true,
    "directory": "production_migrations"
  },
  "rules": {
    "table_operations": {
      "consolidate_create_alter": true,
      "remove_drop_create_cycles": false,
      "preserve_data_operations": true
    },
    "function_operations": {
      "remove_duplicate_definitions": false,
      "preserve_signature_changes": true
    }
  },
  "conflict_resolution": {
    "strict_mode_enabled": true
  }
}
```

### Development Configuration

Aggressive optimization for development:

```json
{
  "safety_level": "aggressive",
  "output": {
    "format": "organized",
    "directory": "dev_migrations"
  },
  "rules": {
    "table_operations": {
      "consolidate_create_alter": true,
      "remove_drop_create_cycles": true,
      "preserve_data_operations": true
    },
    "function_operations": {
      "remove_duplicate_definitions": true,
      "preserve_signature_changes": false
    }
  },
  "performance": {
    "parallel_processing": true,
    "show_progress": true
  }
}
```

### Large Dataset Configuration

Optimized for large migration sets:

```json
{
  "safety_level": "standard",
  "performance": {
    "streaming_threshold_mb": 2,
    "parallel_processing": true,
    "show_progress": true
  },
  "output": {
    "format": "sequential",
    "preserve_comments": false
  }
}
```

### Supabase Project Configuration

Optimized for Supabase projects:

```json
{
  "safety_level": "standard",
  "include_schemas": ["public", "auth", "storage"],
  "exclude_patterns": [],
  "modern_features": {
    "enable_advanced_rls": true
  },
  "third_party_integrations": {
    "supabase_integration": {
      "enabled": true,
      "enable_rls": true,
      "storage_integration": true
    }
  }
}
```

---

For configuration in specific scenarios, see:
- [Safety Levels Guide](./SAFETY_LEVELS.md)
- [User Guide](./USER_GUIDE.md)
- [AI Integration Guide](./AI_INTEGRATION.md)
