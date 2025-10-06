# Safety Levels Guide

Understanding and choosing the right safety level.

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

Four safety levels control consolidation aggressiveness. Each balances safety vs optimization.

**Principle:** Higher safety = fewer changes = lower risk = less optimization

## Safety Level Comparison

| Feature | Paranoid | Conservative | Standard | Aggressive |
|---------|----------|--------------|----------|------------|
| CREATE + ALTER Consolidation | Yes | Yes | Yes | Yes |
| Column Evolution Tracking | Yes | Yes | Yes | Yes |
| DROP/CREATE Cycle Removal | No | No | Yes | Yes |
| RLS Policy Consolidation | No | No | Yes | Yes |
| Function Deduplication | No | No | No | Yes |
| Dead Code Removal | Yes | No | No | Yes |
| AI Analysis (Optional) | Yes | No | No | Yes |
| Database Validation | Required | Recommended | Optional | Optional |
| File Reduction | 15-25% | 20-35% | 35-50% | 50-70% |
| Processing Speed | Slow | Fast | Fast | Medium |
| Production Ready | Yes | Yes | With Testing | No |

## Paranoid Level

### Overview

Ultra-safe mode with minimal changes and extensive validation.

### Configuration

```json
{
  "safety_level": "paranoid",
  "prod_db_dsn": "postgres://user:pass@localhost/db"
}
```

### Rules Applied

1. **CreateAlterConsolidationRule** - Merges CREATE + ALTER sequences
2. **ColumnEvolutionRule** - Tracks column lifecycle
3. **DeadCodeRemovalRule** - Removes provably unused code (requires DB)
4. **AdvancedColumnLifecycleRule** - Handles renames, type changes, defaults

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
-- Paranoid mode: Minimal consolidation
-- Only obvious safe optimizations

-- Migration 001: Create users table
CREATE TABLE users (
    id UUID PRIMARY KEY
);

-- Migration 002: Add email column
ALTER TABLE users ADD COLUMN email VARCHAR(255);

-- Migration 003: Add email constraint
ALTER TABLE users ADD CONSTRAINT email_unique UNIQUE (email);

-- Dead code removal only if proven unused via database analysis
-- DROP FUNCTION old_unused_function(); -- Removed: no references in production
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

1. ✓ **CreateAlterConsolidationRule**
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

2. ✓ **ColumnEvolutionRule**
   ```sql
   -- Tracks column changes across migrations
   -- Preserves important intermediate states
   ```

3. ✓ **ConditionalSchemaRule**
   ```sql
   -- Uses IF NOT EXISTS where appropriate
   CREATE TABLE IF NOT EXISTS users (...);
   ```

4. ✓ **AdvancedColumnLifecycleRule**
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

5. ✓ **DropCreateCycleRule**
   ```sql
   -- Before
   CREATE TABLE temp (id INT);
   DROP TABLE temp;
   CREATE TABLE temp (id UUID);

   -- After
   CREATE TABLE temp (id UUID);  -- Final version only
   ```

6. ✓ **RLSConsolidationRule**
   ```sql
   -- Groups related RLS policies
   -- Consolidates policy definitions
   ```

7. ✓ **TransactionBoundaryRule**
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

8. ✓ **FunctionDeduplicationRule**
   ```sql
   -- Removes duplicate function definitions
   -- Uses semantic equivalence checking
   -- AI-powered comparison (optional)
   ```

9. ✓ **DeadCodeRemovalRule** (Without DB)
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

| Environment | Recommended Level | Alternative |
|-------------|-------------------|-------------|
| Production | Conservative | Paranoid (critical systems) |
| Staging | Conservative | Standard |
| QA/Testing | Standard | Conservative |
| CI/CD | Standard | Conservative |
| Development (Shared) | Standard | Aggressive |
| Development (Local) | Aggressive | Standard |

### By Risk Tolerance

| Risk Tolerance | Safety Level | Characteristics |
|----------------|--------------|-----------------|
| Risk Averse | Paranoid | Minimal changes, maximum validation |
| Low Risk | Conservative | Well-tested consolidations only |
| Moderate Risk | Standard | Balanced optimization |
| High Risk | Aggressive | Maximum optimization |

### By First-Time vs. Experienced

**First Time Using pg-squash**:
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
