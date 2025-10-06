# Troubleshooting Guide

Guide for resolving common issues with pg-squash.

## Table of Contents

- [Quick Diagnostics](#quick-diagnostics)
- [Common Issues](#common-issues)
- [Parsing Errors](#parsing-errors)
- [Dependency Issues](#dependency-issues)
- [Validation Failures](#validation-failures)
- [Performance Problems](#performance-problems)
- [Docker Issues](#docker-issues)
- [AI Integration Issues](#ai-integration-issues)
- [Getting Help](#getting-help)

## Quick Diagnostics

### Run System Check

```bash
# Check pg-squash version
pgsquash --version

# Check Go version
go version  # Should be 1.25.1+

# Check Docker (for validation)
docker --version
docker ps

# Test AI providers
pgsquash ai-test
```

### Enable Verbose Output

Always use `--verbose` when troubleshooting:

```bash
pgsquash analyze migrations/*.sql --verbose
pgsquash squash migrations/*.sql --verbose --dry-run
```

## Common Issues

### Issue: Command Not Found

**Symptom**:
```bash
$ pgsquash
zsh: command not found: pgsquash
```

**Causes**:
1. pg-squash not installed
2. Not in PATH
3. Wrong installation method

**Solutions**:

**If not installed**:
```bash
# Install from source
cd pg-squash-go-app
go build -o pgsquash cmd/pgsquash/main.go

# Or use go install
go install github.com/capysquash/pg-squash-engine/cmd/pgsquash@latest
```

**If not in PATH**:
```bash
# Add to PATH (if built locally)
export PATH=$PATH:/path/to/pg-squash-go-app

# Or use absolute path
/path/to/pg-squash-go-app/pgsquash analyze migrations/*.sql
```

**If installed via go install**:
```bash
# Ensure GOPATH/bin is in PATH
export PATH=$PATH:$(go env GOPATH)/bin

# Add to shell profile
echo 'export PATH=$PATH:$(go env GOPATH)/bin' >> ~/.zshrc
source ~/.zshrc
```

### Issue: No Migrations Found

**Symptom**:
```bash
$ pgsquash analyze migrations/*.sql
Error: no migration files provided
```

**Causes**:
1. Wrong directory
2. No .sql files
3. Glob pattern not expanding

**Solutions**:

**Check directory**:
```bash
# List SQL files
ls migrations/*.sql

# If no files
ls migrations/
```

**Use correct pattern**:
```bash
# Wrong: quoted glob (shell doesn't expand)
pgsquash analyze "migrations/*.sql"

# Correct: unquoted glob
pgsquash analyze migrations/*.sql

# Or explicit files
pgsquash analyze migrations/001_*.sql migrations/002_*.sql
```

**Check file extensions**:
```bash
# If files are .SQL (uppercase)
pgsquash analyze migrations/*.SQL

# Or rename
rename 's/\.SQL$/.sql/' migrations/*.SQL
```

### Issue: Permission Denied

**Symptom**:
```bash
Error: failed to read migrations/001_init.sql: permission denied
```

**Causes**:
1. File permissions
2. Directory permissions
3. SELinux/AppArmor restrictions

**Solutions**:

```bash
# Check permissions
ls -la migrations/

# Fix file permissions
chmod 644 migrations/*.sql

# Fix directory permissions
chmod 755 migrations/

# If SELinux (rarely needed)
chcon -R -t user_home_t migrations/
```

## Parsing Errors

### Issue: Failed to Parse Migration

**Symptom**:
```bash
Error: parse migration 005_add_vector.sql: syntax error at or near "VECTOR"
```

**Causes**:
1. Unsupported PostgreSQL syntax
2. Extension-specific syntax
3. Malformed SQL
4. Special characters in comments

**Solutions**:

**Check syntax**:
```bash
# Test with psql
psql -d testdb -f migrations/005_add_vector.sql

# If psql works but pg-squash fails, syntax might be too new
```

**For extension-specific syntax**:
```sql
-- Wrap in DO block if needed
DO $$
BEGIN
    -- Extension-specific code here
END $$;
```

**Common syntax issues**:

**Issue: Vector types not recognized**:
```sql
-- Problem
CREATE TABLE embeddings (
    id UUID PRIMARY KEY,
    vector VECTOR(1536)  -- May cause parse error
);

-- Solution: Use text or create after extension
CREATE EXTENSION vector;
CREATE TABLE embeddings (
    id UUID PRIMARY KEY,
    embedding vector(1536)
);
```

**Issue: Complex DO blocks**:
```sql
-- Problem: Very complex DO block
DO $$
DECLARE ...
BEGIN ...
-- 100 lines of code
END $$;

-- Solution: Extract to function
CREATE OR REPLACE FUNCTION complex_migration() RETURNS VOID AS $$
DECLARE ...
BEGIN ...
END;
$$ LANGUAGE plpgsql;

SELECT complex_migration();
```

**Issue: Special characters in comments**:
```sql
-- Problem: Apostrophe in comment breaks parsing
-- This migration adds users' email field

-- Solution: Escape or remove
-- This migration adds users email field
```

### Issue: Statement Splitting Errors

**Symptom**:
```bash
Error: failed to split statements: unexpected token
```

**Causes**:
1. Missing semicolons
2. Dollar-quoted functions
3. Mixed statement terminators

**Solutions**:

```sql
-- Ensure semicolons
CREATE TABLE users (id UUID PRIMARY KEY);  -- Semicolon required

-- Dollar-quote functions properly
CREATE FUNCTION test() RETURNS VOID AS $$
BEGIN
    -- Function body
END;
$$ LANGUAGE plpgsql;  -- Semicolon after $$

-- Consistent quoting
CREATE FUNCTION test() RETURNS VOID AS $BODY$
BEGIN
    -- Use $BODY$ or $$ consistently
END;
$BODY$ LANGUAGE plpgsql;
```

## Dependency Issues

### Issue: Circular Dependency Detected

**Symptom**:
```bash
Warning: Circular dependency: users -> profiles -> users
```

**Causes**:
1. Table mutual references
2. Function call cycles
3. Trigger chains

**Solutions**:

**For table references**:
```sql
-- Problem: Mutual foreign keys
CREATE TABLE users (id UUID PRIMARY KEY, profile_id UUID REFERENCES profiles(id));
CREATE TABLE profiles (id UUID PRIMARY KEY, user_id UUID REFERENCES users(id));

-- Solution: Add foreign key after both tables exist
CREATE TABLE users (id UUID PRIMARY KEY, profile_id UUID);
CREATE TABLE profiles (id UUID PRIMARY KEY, user_id UUID);
ALTER TABLE users ADD FOREIGN KEY (profile_id) REFERENCES profiles(id);
ALTER TABLE profiles ADD FOREIGN KEY (user_id) REFERENCES users(id);
```

**Enable cycle detection**:
```bash
# Get detailed cycle information
pgsquash squash migrations/*.sql --detect-cycles --cycle-details
```

**Review cycle report**:
```
DDL Cycle detected:
  Type: TABLE_FOREIGN_KEY
  Severity: MODERATE
  Objects: users, profiles
  Description: Mutual foreign key references
  Recommendation: Add constraints after table creation
```

### Issue: Missing Dependencies

**Symptom**:
```bash
Error: object 'auth.users' not found
```

**Causes**:
1. Cross-schema references
2. Extension not installed
3. Missing migration files
4. Order of operations

**Solutions**:

**Check schema configuration**:
```json
{
  "include_schemas": ["public", "auth", "storage"]
}
```

**Ensure extension migrations included**:
```bash
# Include extension setup
pgsquash squash migrations/000_extensions.sql migrations/*.sql
```

**Check file order**:
```bash
# Migrations should be in correct sequence
ls -1 migrations/
# 001_extensions.sql
# 002_auth_schema.sql
# 003_users.sql  # References auth schema
```

## Validation Failures

### Issue: Schema Validation Failed

**Symptom**:
```bash
✗ Validation failed: Schemas are different.
Differences found:
- Missing column: users.deleted_at
```

**Causes**:
1. Consolidation error
2. Column dropped incorrectly
3. Safety level too aggressive
4. Data operation preserved

**Solutions**:

**Use more conservative safety**:
```bash
# Try conservative mode
pgsquash squash migrations/*.sql --safety conservative
pgsquash validate migrations/ clean_migrations/
```

**Check specific difference**:
```bash
# Review the consolidation
pgsquash squash migrations/*.sql --dry-run | grep -A5 "users"

# Look for the missing column
grep "deleted_at" migrations/*.sql
grep "deleted_at" clean_migrations/*.sql
```

**Manual review**:
```sql
-- Check original migrations
cat migrations/*_add_deleted_at.sql

-- Check if DROP column exists
grep "DROP COLUMN deleted_at" migrations/*.sql

-- If column was dropped then recreated, verify consolidation
```

**Disable problematic rules**:
```json
{
  "rules": {
    "table_operations": {
      "remove_drop_create_cycles": false  # Be more conservative
    }
  }
}
```

### Issue: Docker Validation Won't Start

**Symptom**:
```bash
Error: failed to start docker container: Cannot connect to Docker daemon
```

**Causes**:
1. Docker not running
2. Docker permissions
3. Port conflicts

**Solutions**:

**Check Docker status**:
```bash
# Start Docker Desktop (macOS)
open -a Docker

# Or start Docker daemon (Linux)
sudo systemctl start docker

# Verify
docker ps
```

**Check permissions**:
```bash
# Add user to docker group (Linux)
sudo usermod -aG docker $USER
newgrp docker

# Or use sudo (temporary)
sudo pgsquash validate migrations/ clean/
```

**Check port availability**:
```bash
# Check if PostgreSQL ports in use
lsof -i :5432
lsof -i :5433

# Kill conflicting processes if needed
# Or use different ports (advanced)
```

### Issue: Extension Not Found in Docker

**Symptom**:
```bash
Error: extension "vector" is not available
```

**Causes**:
1. Extension not in default PostgreSQL image
2. Special extension needs custom image

**Solutions**:

**pg-squash automatically detects extensions**:
```bash
# Validator will detect required extensions
pgsquash validate migrations/ clean/

# Output shows:
# Extensions detected: [vector, postgis]
# Using enhanced PostgreSQL image with extensions
```

**If custom extensions needed**:
```bash
# pg-squash will warn about unsupported extensions
# Manual validation required with custom Docker image
```

## Performance Problems

### Issue: Squashing is Slow

**Symptom**:
```bash
# Taking > 5 minutes for 50 migrations
```

**Causes**:
1. Large migration files
2. Many files
3. AI analysis enabled
4. Verbose logging

**Solutions**:

**Enable streaming mode**:
```bash
pgsquash squash migrations/*.sql --streaming --batch-size 100
```

**Disable AI**:
```bash
# Unset API keys temporarily
unset ANTHROPIC_API_KEY
unset OPENAI_API_KEY

pgsquash squash migrations/*.sql
```

**Increase batch size**:
```bash
pgsquash squash migrations/*.sql \
  --streaming \
  --batch-size 100 \
  --workers 8
```

**Disable progress**:
```bash
pgsquash squash migrations/*.sql --progress=false
```

### Issue: Out of Memory

**Symptom**:
```bash
fatal error: out of memory
```

**Causes**:
1. Too many large migrations
2. Streaming not enabled
3. Memory limit too low

**Solutions**:

**Enable streaming with lower memory limit**:
```bash
pgsquash squash migrations/*.sql \
  --streaming \
  --memory-limit 256 \
  --batch-size 50
```

**Process in chunks**:
```bash
# Process migrations in groups
pgsquash squash migrations/001_*.sql --output clean_1/
pgsquash squash migrations/002_*.sql --output clean_2/
pgsquash squash migrations/003_*.sql --output clean_3/

# Then combine
cat clean_*/*.sql > final_migration.sql
```

**Increase system memory**:
```bash
# macOS: Docker Desktop -> Settings -> Resources -> Memory
# Linux: Ensure enough RAM available
free -h
```

### Issue: High CPU Usage

**Symptom**:
```bash
# CPU at 100%, system unresponsive
```

**Causes**:
1. Too many workers
2. Large parallel processing
3. AI analysis

**Solutions**:

```bash
# Reduce worker count
pgsquash squash migrations/*.sql --workers 2

# Disable parallel processing
# Edit pgsquash.config.json
{
  "performance": {
    "parallel_processing": false
  }
}

# Process serially
pgsquash squash migrations/*.sql --workers 1
```

## Docker Issues

### Issue: Docker Container Won't Stop

**Symptom**:
```bash
Error: container still running after validation
```

**Causes**:
1. pg-squash interrupted
2. Validation crashed
3. Network issues

**Solutions**:

```bash
# List pg-squash containers
docker ps | grep pgsquash

# Stop all pg-squash containers
docker ps | grep pgsquash | awk '{print $1}' | xargs docker stop

# Remove containers
docker ps -a | grep pgsquash | awk '{print $1}' | xargs docker rm

# Clean up volumes
docker volume prune
```

### Issue: Port Already in Use

**Symptom**:
```bash
Error: port 5432 already allocated
```

**Causes**:
1. Local PostgreSQL running
2. Previous container not stopped
3. Another application using port

**Solutions**:

```bash
# Check what's using the port
lsof -i :5432

# Stop local PostgreSQL (macOS)
brew services stop postgresql

# Stop local PostgreSQL (Linux)
sudo systemctl stop postgresql

# Or let pg-squash use different ports (automatic)
```

## AI Integration Issues

### Issue: AI Provider Not Available

**Symptom**:
```bash
Warning: AI analyzer unavailable: no providers configured
```

**Causes**:
1. API key not set
2. Invalid API key
3. Network issues

**Solutions**:

```bash
# Set API key
export ANTHROPIC_API_KEY="sk-ant-api03-..."

# Verify key format
echo $ANTHROPIC_API_KEY | grep "^sk-ant"

# Test provider
pgsquash ai-test

# Check network
curl https://api.anthropic.com/v1/messages \
  -H "x-api-key: $ANTHROPIC_API_KEY" \
  -H "anthropic-version: 2023-06-01"
```

### Issue: AI Analysis Takes Too Long

**Symptom**:
```bash
# AI analysis > 60 seconds
```

**Causes**:
1. Large migrations
2. Many functions to analyze
3. Network latency

**Solutions**:

```bash
# Skip AI for routine operations
pgsquash squash migrations/*.sql --safety standard

# Use AI only for deep analysis
pgsquash analyze-deep migrations/*.sql

# Or disable AI completely
unset ANTHROPIC_API_KEY
unset OPENAI_API_KEY
```

## Getting Help

### Enable Debug Output

```bash
# Maximum verbosity
pgsquash squash migrations/*.sql --verbose 2>&1 | tee debug.log

# Share debug.log when reporting issues
```

### Collect Diagnostic Information

```bash
# System information
uname -a
go version
docker --version

# pg-squash version
pgsquash --version

# Configuration
cat pgsquash.config.json

# Migration summary
ls -lh migrations/*.sql | wc -l
du -sh migrations/
```

### Report an Issue

When reporting issues, include:

1. **pg-squash version**: `pgsquash --version`
2. **Go version**: `go version`
3. **Operating system**: macOS, Linux, Windows
4. **Command used**: Full command line
5. **Error message**: Complete error output
6. **Configuration**: Relevant config settings
7. **Migration summary**: Number and size of files
8. **Verbose log**: Output with `--verbose`

**Example issue report**:

```markdown
## Environment
- pg-squash version: 1.0.0
- Go version: 1.25.1
- OS: macOS 14.0
- Docker: 24.0.0

## Issue
Schema validation fails with missing column error.

## Command
```bash
pgsquash squash migrations/*.sql --safety standard --output clean/
pgsquash validate migrations/ clean/
```

## Error
```
✗ Validation failed: Schemas are different.
- Missing column: users.deleted_at
```

## Configuration
```json
{
  "safety_level": "standard",
  "rules": {
    "table_operations": {
      "remove_drop_create_cycles": true
    }
  }
}
```

## Migration Summary
- Files: 23
- Total size: 150KB
- Contains: tables, indexes, functions

## Verbose Log
[Attached: debug.log]
```

### Community Resources

- **GitHub Issues**: https://github.com/capysquash/pg-squash-engine/issues
- **Documentation**: `/docs` directory
- **Examples**: `/examples` directory

### Common Solutions Checklist

Before asking for help, try:

- [ ] Run with `--verbose` flag
- [ ] Try `--dry-run` to preview
- [ ] Test with `--safety conservative`
- [ ] Validate with `pgsquash validate`
- [ ] Check configuration file
- [ ] Review migration file syntax
- [ ] Test with smaller subset of migrations
- [ ] Check Docker is running (for validation)
- [ ] Verify API keys (for AI features)
- [ ] Check system resources (memory, CPU)

---

Most issues can be resolved by:
1. Using more conservative safety level
2. Enabling verbose output for diagnostics
3. Testing with smaller migration sets
4. Validating output thoroughly
