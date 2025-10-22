# Safety Levels Guide

Understanding pgsquash's intelligent consolidation strategies and choosing the right safety level for your workflow.

## Table of Contents

- [Overview](#overview)
- [Safety Level Comparison](#safety-level-comparison)
- [Paranoid Level](#paranoid-level)
- [Conservative Level](#conservative-level)
- [Standard Level](#standard-level)
- [Aggressive Level](#aggressive-level)
- [Choosing a Safety Level](#choosing-a-safety-level)
- [Safety Level Examples](#safety-level-examples)

## Overview

pgsquash offers four safety levels that control consolidation strategies and optimization aggressiveness. Each level balances safety with optimization potential using different rule sets and validation approaches.

**Core Principle:** Higher safety = fewer transformations = lower risk = more conservative consolidation

**Important Note:** Safety levels differ primarily in their **consolidation strategies and rules applied**, not necessarily in raw output file size. Different levels may produce similar line counts but with fundamentally different approaches to statement organization, dependency resolution, and optimization logic.

## Safety Level Comparison

| Feature                          | Paranoid  | Conservative | Standard     | Aggressive |
| -------------------------------- | --------- | ------------ | ------------ | ---------- |
| CREATE + ALTER Consolidation     | Yes       | Yes          | Yes          | Yes        |
| Column Evolution Tracking        | Yes       | Yes          | Yes          | Yes        |
| DROP/CREATE Cycle Removal        | Yes       | No           | Yes          | Yes        |
| RLS Policy Consolidation         | Yes       | No           | Yes          | Yes        |
| Function Deduplication           | Yes       | No           | No           | Yes        |
| Dead Code Removal                | Yes\*     | No           | No           | Yes\*\*    |
| AI-Powered Analysis (Optional)   | Yes       | No           | No           | Yes        |
| Database Validation              | Required  | Recommended  | Optional     | Optional   |
| Circular FK Detection & Handling | Yes       | Yes          | Yes          | Yes        |
| DDL Cycle Detection              | Yes       | Yes          | Yes          | Yes        |
| Optimization Rules Applied       | All 9     | 4            | 7            | All 9      |
| Processing Speed                 | Slowest   | Fast         | Fast         | Medium     |
| Production Ready                 | Yes\*\*\* | Yes          | With Testing | No         |

\* **Paranoid:** Database-validated dead code removal (queries production DB to verify no usage)
\*\* **Aggressive:** Static analysis-based dead code removal (heuristic-based, no DB required)
\*\*\* **Paranoid production use:** Only if you have a production DB connection for validation. Otherwise falls back to Conservative.

### Understanding Safety Level Differences

Safety levels primarily differ in **which consolidation rules are enabled** and **validation requirements**, not output size:

**Paranoid**:

- **All Aggressive-level consolidations** (same rules as Aggressive mode)
- **Plus: Database-validated dead code removal** (requires production DB connection)
- Creates maximally optimized output, but every optimization is verified safe via database queries
- Slowest processing due to database validation overhead
- Most thorough dead code detection (compares against actual production usage)

**Conservative**:

- CREATE + ALTER consolidation
- Column evolution tracking
- Final column ordering may differ from original
- No DROP/CREATE cycle removal

**Standard**:

- All conservative optimizations
- Plus DROP/CREATE cycle removal
- Plus RLS policy consolidation
- Statement reordering for dependency optimization

**Aggressive**:

- All standard optimizations
- Plus function deduplication
- Plus static dead code removal (heuristic-based, no DB required)
- AI-powered semantic analysis

**Visual Example**:

```sql
-- Original migrations
CREATE TABLE users (id INT);
ALTER TABLE users ADD email VARCHAR(255);
ALTER TABLE users ADD name VARCHAR(255);

-- Paranoid/Conservative: Consolidates to final schema
CREATE TABLE users (
    id INT,
    email VARCHAR(255),
    name VARCHAR(255)
);

-- Standard: Same consolidation, may reorder columns
CREATE TABLE users (
    id INT,
    name VARCHAR(255),  -- Reordered for optimization
    email VARCHAR(255)
);

-- Aggressive: Same as Standard but with additional
-- function deduplication and dead code removal
```

## Paranoid Level

### Overview

**Maximum consolidation with database-validated safety.** Paranoid mode applies the same aggressive optimization rules as Aggressive mode, but validates every dead code removal against your actual production database. This provides the most thorough optimization possible while ensuring absolute safety through runtime verification.

**Key Characteristics:**

- Applies **all** Aggressive-level consolidations
- **Plus:** Database-validated dead code removal
- Slowest processing (requires database queries)
- Most accurate dead code detection
- Requires production database connection

### Configuration

```json
{
  "safety_level": "paranoid",
  "prod_db_dsn": "postgres://user:pass@localhost/db"
}
```

### Rules Applied

**Important:** Paranoid mode applies **ALL** the rules that Aggressive mode uses, **PLUS** database-validated dead code removal. This means it performs the most aggressive consolidation possible, but validates every change against your production database.

1. **CreateAlterConsolidationRule** - Merges CREATE + ALTER sequences
2. **ColumnEvolutionRule** - Tracks column lifecycle
3. **ConditionalSchemaRule** - Uses IF NOT EXISTS where appropriate
4. **AdvancedColumnLifecycleRule** - Handles renames, type changes, defaults
5. **DropCreateCycleRule** - Removes DROP/CREATE cycles (including VIEWs)
6. **RLSConsolidationRule** - Consolidates RLS policies
7. **TransactionBoundaryRule** - Optimizes transaction boundaries
8. **FunctionDeduplicationRule** - Removes duplicate functions
9. **DeadCodeRemovalRule** - Removes provably unused code (**requires DB connection**)

**Difference from Aggressive:** Paranoid validates dead code removal against your actual production database, while Aggressive uses static analysis heuristics. This makes Paranoid slower but more accurate.

### Requirements

**Database Connection**: Required

```bash
export PROD_DB_DSN="postgres://user:pass@localhost:5432/database"
```

Without database connection, paranoid mode falls back to conservative.

### Use Cases

- Critical production systems
- Compliance requirements (SOC2, HIPAA, PCI-DSS)
- First-time squashing
- **Financial Systems**: Banking, payment processing
- **Healthcare Systems**: Patient data systems

### Example Output

```sql
-- Paranoid mode: MAXIMUM consolidation with database validation
-- Applies all aggressive optimizations, verified safe via production DB

-- Tables: CREATE + ALTER consolidated (same as Aggressive)
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(255),
    status VARCHAR(50) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT NOW()
);
-- Consolidated from migrations 001-005

-- DROP/CREATE cycles removed (same as Standard/Aggressive)
CREATE TABLE user_sessions (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    expires_at TIMESTAMP NOT NULL
);
-- Removed intermediate DROP/CREATE cycle from migrations 012-015

-- RLS policies consolidated (same as Standard/Aggressive)
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
CREATE POLICY users_select ON users FOR SELECT USING (true);
CREATE POLICY users_insert ON users FOR INSERT WITH CHECK (auth.uid() = id);

-- Function deduplication applied (same as Aggressive)
CREATE FUNCTION count_active_users() RETURNS INTEGER AS $$
BEGIN
    RETURN (SELECT COUNT(*) FROM users WHERE status = 'active');
END;
$$ LANGUAGE plpgsql;
-- Removed duplicate: count_active_users_v2 (semantically equivalent)

-- Dead code removal with DATABASE VALIDATION (Paranoid-specific)
-- These removals are VERIFIED against production database queries:
-- DROP FUNCTION old_unused_function(); -- Removed: VERIFIED no callers in production DB
-- DROP FUNCTION legacy_email_validator(); -- Removed: VERIFIED replaced by new version
-- DROP TRIGGER update_modified_at_trigger; -- Removed: VERIFIED no dependencies in production

-- Note: Without database connection, Paranoid falls back to Conservative mode
```

### Command Usage

```bash
# Paranoid mode with database validation
export PROD_DB_DSN="postgres://user:pass@localhost:5432/db"
pgsquash squash migrations/*.sql --safety paranoid --output production/

# Or use safe workflow (conservative mode)
pgsquash safe migrations/*.sql --output production/
```

### Performance

- **Processing Time**: Slowest (database queries for validation)
- **Memory Usage**: Low to moderate
- **File Size Reduction**: 15-25%

## Conservative Level

### Overview

Production-safe mode with only well-tested consolidations.

### Configuration

```json
{
  "safety_level": "conservative"
}
```

### Rules Applied

1. ☑ **CreateAlterConsolidationRule**

   ```sql
   -- Before
   CREATE TABLE users (id UUID);
   ALTER TABLE users ADD email VARCHAR(255);

   -- After
   CREATE TABLE users (
       id UUID,
       email VARCHAR(255)
   );
   ```

2. ☑ **ColumnEvolutionRule**

   ```sql
   -- Tracks column changes across migrations
   -- Preserves important intermediate states
   ```

3. ☑ **ConditionalSchemaRule**

   ```sql
   -- Uses IF NOT EXISTS where appropriate
   CREATE TABLE IF NOT EXISTS users (...);
   ```

4. ☑ **AdvancedColumnLifecycleRule**

   ```sql
   -- Handles complex column evolution
   -- Preserves data type changes
   ```

### Use Cases

- **Production Deployments**: Standard production systems
- **Staging Databases**: Pre-production environments
- **Shared Databases**: Multi-tenant systems
- **Risk-Averse Organizations**: Enterprises with strict change control
- **Legacy Systems**: Older systems with complex history

### Example Output

```sql
-- Conservative mode: Safe CREATE + ALTER consolidation

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL,
    name VARCHAR(255),
    status VARCHAR(50) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT NOW()
);
-- Consolidated from:
--   001_create_users.sql
--   002_add_email.sql
--   003_add_name.sql
--   004_add_status.sql
--   005_add_timestamps.sql

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_status ON users(status);
```

### Command Usage

```bash
# Conservative mode
pgsquash squash migrations/*.sql --safety conservative

# Or use safe workflow (includes validation)
pgsquash safe migrations/*.sql
```

### Performance

- **Processing Time**: Fast
- **Memory Usage**: Low to moderate
- **File Size Reduction**: 20-35%

## Standard Level

### Overview

Balanced mode with proven optimizations for staging and testing.

### Configuration

```json
{
  "safety_level": "standard"
}
```

### Rules Applied

All Conservative rules plus:

5. ☑ **DropCreateCycleRule**

   ```sql
   -- Before
   CREATE TABLE temp (id INT);
   DROP TABLE temp;
   CREATE TABLE temp (id UUID);

   -- After
   CREATE TABLE temp (id UUID);  -- Final version only
   ```

6. ☑ **RLSConsolidationRule**

   ```sql
   -- Groups related RLS policies
   -- Consolidates policy definitions
   ```

7. ☑ **TransactionBoundaryRule**

   ```sql
   -- Optimizes transaction boundaries
   -- Groups related operations
   ```

### Use Cases

- **Staging Environments**: Pre-production testing
- **Development Databases**: Shared development databases
- **CI/CD Pipelines**: Automated testing pipelines
- **Integration Testing**: System integration tests
- **Performance Testing**: Load and performance tests

### Example Output

```sql
-- Standard mode: Balanced optimization

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL,
    name VARCHAR(255),
    status VARCHAR(50) DEFAULT 'active'
);

-- DROP/CREATE cycles removed
CREATE TABLE user_sessions (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    expires_at TIMESTAMP NOT NULL
);
-- Removed intermediate DROP/CREATE cycle from migrations 012-015

-- RLS policies consolidated
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
CREATE POLICY users_select ON users FOR SELECT USING (true);
CREATE POLICY users_insert ON users FOR INSERT WITH CHECK (auth.uid() = id);
-- Consolidated from migrations 020-025
```

### Command Usage

```bash
# Standard mode (default)
pgsquash squash migrations/*.sql

# Or use fast workflow
pgsquash fast migrations/*.sql
```

### Performance

- **Processing Time**: Fast
- **Memory Usage**: Moderate
- **File Size Reduction**: 35-50%

## Aggressive Level

### Overview

Maximum optimization for development environments.

### Configuration

```json
{
  "safety_level": "aggressive"
}
```

### Rules Applied

All Standard rules plus:

8. ☑ **FunctionDeduplicationRule**

   ```sql
   -- Removes duplicate function definitions
   -- Uses semantic equivalence checking
   -- AI-powered comparison (optional)
   ```

9. ☑ **DeadCodeRemovalRule** (Without DB)

   ```sql
   -- Removes unused functions/triggers
   -- Based on static analysis
   -- More aggressive than paranoid mode
   ```

### Use Cases

- **Development Environments**: Local development
- **Feature Branches**: Experimental features
- **Prototypes**: Proof of concept projects
- **Throwaway Databases**: Temporary test databases
- **Migration Cleanup**: Before production preparation

### Example Output

```sql
-- Aggressive mode: Maximum optimization

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(255),
    status VARCHAR(50) DEFAULT 'active'
);

-- Function deduplication applied
CREATE FUNCTION count_active_users() RETURNS INTEGER AS $$
BEGIN
    RETURN (SELECT COUNT(*) FROM users WHERE status = 'active');
END;
$$ LANGUAGE plpgsql;
-- Removed duplicate: count_active_users_v2 (semantically equivalent)

-- Dead code removed
-- DROP FUNCTION old_user_stats(); -- Removed: no references found
-- DROP FUNCTION legacy_email_validator(); -- Removed: replaced by new version
```

### Command Usage

```bash
# Aggressive mode
pgsquash squash migrations/*.sql --safety aggressive

# With AI analysis
export ANTHROPIC_API_KEY="sk-ant-..."
pgsquash squash migrations/*.sql --safety aggressive
```

### Performance

- **Processing Time**: Medium (AI analysis if enabled)
- **Memory Usage**: Moderate to high
- **File Size Reduction**: 50-70%

### Warnings

⚠️ **Not for Production**: Aggressive mode should NOT be used for production without thorough testing.

⚠️ **Function Analysis**: Function deduplication uses heuristics; review changes carefully.

⚠️ **Dead Code**: Static analysis may miss dynamic code usage.

## Choosing a Safety Level

### Decision Tree

```
START: What's your use case?
│
├─ Production Database?
│  ├─ Critical System? → PARANOID
│  └─ Standard System → CONSERVATIVE
│
├─ Staging/Testing?
│  ├─ Pre-production → CONSERVATIVE
│  └─ Testing Environment → STANDARD
│
└─ Development?
   ├─ Shared Development → STANDARD
   └─ Local Development → AGGRESSIVE
```

### By Environment

| Environment          | Recommended Level | Alternative                 |
| -------------------- | ----------------- | --------------------------- |
| Production           | Conservative      | Paranoid (critical systems) |
| Staging              | Conservative      | Standard                    |
| QA/Testing           | Standard          | Conservative                |
| CI/CD                | Standard          | Conservative                |
| Development (Shared) | Standard          | Aggressive                  |
| Development (Local)  | Aggressive        | Standard                    |

### By Risk Tolerance

| Risk Tolerance | Safety Level | Characteristics                     |
| -------------- | ------------ | ----------------------------------- |
| Risk Averse    | Paranoid     | Minimal changes, maximum validation |
| Low Risk       | Conservative | Well-tested consolidations only     |
| Moderate Risk  | Standard     | Balanced optimization               |
| High Risk      | Aggressive   | Maximum optimization                |

### By First-Time vs. Experienced

**First Time Using pgsquash**:

1. Start with **Conservative**
2. Use **--dry-run** extensively
3. Always run **validate**
4. Review output carefully
5. Test in staging first

**Experienced Users**:

1. **Standard** for most cases
2. **Aggressive** for development
3. **Paranoid** for critical changes
4. Use **fast/safe workflows**

## Safety Level Examples

### Example 1: New SaaS Application

**Scenario**: Startup with rapid development, monthly production deployments

**Recommendation**:

- Development: **Aggressive**
- Staging: **Standard**
- Production: **Conservative**

```bash
# Development
pgsquash fast migrations/*.sql --output dev/

# Staging
pgsquash squash migrations/*.sql --safety standard --output staging/

# Production
pgsquash safe migrations/*.sql --output production/
```

### Example 2: Enterprise System

**Scenario**: Large enterprise, quarterly releases, strict compliance

**Recommendation**:

- All Environments: **Conservative**
- Critical Systems: **Paranoid**

```bash
# Standard deployment
pgsquash safe migrations/*.sql --output production/

# Critical system
export PROD_DB_DSN="postgres://..."
pgsquash squash migrations/*.sql --safety paranoid --validate
```

### Example 3: Open Source Project

**Scenario**: Open source project with many contributors

**Recommendation**:

- Development: **Aggressive**
- CI/CD: **Standard**
- Release: **Conservative**

```bash
# Contributor development
pgsquash squash migrations/*.sql --safety aggressive

# CI pipeline
pgsquash squash migrations/*.sql --safety standard --validate

# Release preparation
pgsquash safe migrations/*.sql --output release/
```

### Example 4: Migration Cleanup

**Scenario**: Years of accumulated migrations, preparing for cleanup

**Process**:

1. **Analyze**: Deep analysis to understand complexity

   ```bash
   pgsquash analyze-deep migrations/*.sql
   ```

2. **Test Aggressive**: See maximum optimization

   ```bash
   pgsquash squash migrations/*.sql --safety aggressive --dry-run
   ```

3. **Use Standard**: Balance optimization and safety

   ```bash
   pgsquash squash migrations/*.sql --safety standard --output clean/
   ```

4. **Validate**: Ensure equivalence

   ```bash
   pgsquash validate migrations/ clean/
   ```

5. **Production**: Use conservative for actual deployment

   ```bash
   pgsquash safe clean/*.sql --output production/
   ```

## Best Practices

### 1. Start Conservative

Always start with a more conservative level:

```bash
# First run
pgsquash squash migrations/*.sql --safety conservative --dry-run

# Review output
# Then proceed with actual squashing
```

### 2. Use Dry Run

Preview changes before committing:

```bash
# Try aggressive optimizations
pgsquash squash migrations/*.sql --safety aggressive --dry-run

# If comfortable, apply standard
pgsquash squash migrations/*.sql --safety standard
```

### 3. Always Validate

Validate squashed migrations:

```bash
pgsquash squash migrations/*.sql --safety standard --output clean/
pgsquash validate migrations/ clean/
```

### 4. Progressive Testing

Test progressively through environments:

```bash
# Development: aggressive
pgsquash squash migrations/*.sql --safety aggressive --output dev/

# Staging: standard
pgsquash squash migrations/*.sql --safety standard --output staging/

# Production: conservative
pgsquash squash migrations/*.sql --safety conservative --output prod/
```

### 5. Review AI Suggestions

When using aggressive mode with AI:

```bash
pgsquash analyze-deep migrations/*.sql > analysis.txt
# Review analysis
# Make informed decision
pgsquash squash migrations/*.sql --safety aggressive
```

## Migrating Between Safety Levels

### Conservative → Standard

Safe progression:

```bash
# Run both and compare
pgsquash squash migrations/*.sql --safety conservative --output conservative/
pgsquash squash migrations/*.sql --safety standard --output standard/

# Validate both
pgsquash validate migrations/ conservative/
pgsquash validate migrations/ standard/

# If both pass, use standard
```

### Standard → Aggressive

Requires careful review:

```bash
# Analyze differences
pgsquash analyze-deep migrations/*.sql

# Test aggressive
pgsquash squash migrations/*.sql --safety aggressive --output aggressive/ --dry-run

# Review function deduplication
# Review dead code removal
# Test thoroughly before using
```

---

Choose the safety level that matches your risk tolerance, environment, and validation capabilities. When in doubt, start conservative and progressively increase optimization as confidence grows.
