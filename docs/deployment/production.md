# Production Deployment Guide

Best practices for deploying pg-squash in production environments.

## Pre-Deployment Checklist

**Configuration:**
- [ ] Safety level set to `conservative` or `paranoid`
- [ ] Validation enabled (TWO_CONTAINERS method)
- [ ] Backup generation enabled
- [ ] Rollback scripts enabled
- [ ] Database connection configured (paranoid mode)

**Testing:**
- [ ] All migrations tested in staging
- [ ] Validation passed in staging
- [ ] Schema equivalence verified
- [ ] Performance tested with realistic data

**Security:**
- [ ] API keys stored in secrets manager
- [ ] Database credentials encrypted
- [ ] Webhook secrets rotated
- [ ] Access logs enabled

**Monitoring:**
- [ ] Logging configured
- [ ] Error tracking setup
- [ ] Performance metrics enabled
- [ ] Alerting configured

## Configuration

### Production Config

`pgsquash.config.json`:

```json
{
  "safety_level": "conservative",
  "prod_db_dsn": "${PROD_DB_DSN}",
  "output": {
    "directory": "squashed",
    "format": "organized",
    "preserve_comments": true,
    "add_consolidation_comments": true
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
  "performance": {
    "parallel_processing": true,
    "show_progress": true,
    "streaming": true,
    "memory_limit_mb": 512
  },
  "validation": {
    "method": "TWO_CONTAINERS",
    "auto_fix_sql": false,
    "strict_mode": true
  }
}
```

### Environment Variables

```bash
# Database
export PROD_DB_DSN="postgres://user:pass@host:5432/db?sslmode=require"

# AI (optional, for paranoid mode)
export ANTHROPIC_API_KEY="sk-ant-..."

# GitHub integration
export GITHUB_TOKEN="ghp_..."
export GITHUB_WEBHOOK_SECRET="..."

# Logging
export LOG_LEVEL="info"
export LOG_FORMAT="json"
```

## Deployment Strategies

### Strategy 1: Scheduled Consolidation

Best for: Regular maintenance, predictable migration volume

```bash
#!/bin/bash
# Run weekly consolidation

set -e

# Backup
pg_dump -Fc production_db > backup_$(date +%Y%m%d).dump

# Clone repository
git clone https://github.com/yourorg/repo.git
cd repo

# Squash migrations
pgsquash safe migrations/*.sql --output squashed/

# Validate
pgsquash validate migrations/ squashed/

# Create PR
git checkout -b squash-$(date +%Y%m%d)
git add squashed/
git commit -m "chore: consolidate migrations $(date +%Y-%m-%d)"
git push origin squash-$(date +%Y%m%d)
gh pr create --title "Migration Consolidation" --body "Automated weekly consolidation"
```

Schedule with cron:
```cron
0 2 * * 0 /path/to/consolidate.sh
```

### Strategy 2: Threshold-Based

Best for: Variable migration volume, automated workflows

```yaml
# .github/workflows/consolidate.yml
name: Auto Consolidation

on:
  push:
    paths:
      - 'migrations/**'

jobs:
  check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Count migrations
        id: count
        run: |
          count=$(ls migrations/*.sql | wc -l)
          echo "count=$count" >> $GITHUB_OUTPUT

      - name: Consolidate if needed
        if: steps.count.outputs.count > 50
        run: |
          docker run --rm \
            -v $(pwd)/migrations:/app/migrations \
            -v $(pwd)/squashed:/app/output \
            -v /var/run/docker.sock:/var/run/docker.sock \
            -e PGSQUASH_SAFETY_LEVEL=conservative \
            pgsquash:latest safe /app/migrations/*.sql --output /app/output

      - name: Create PR
        if: steps.count.outputs.count > 50
        uses: peter-evans/create-pull-request@v5
        with:
          commit-message: 'chore: consolidate migrations'
          title: 'Auto-consolidation: Reached 50 migrations'
          body: 'Automated migration consolidation triggered by threshold'
```

### Strategy 3: Manual Review

Best for: Critical systems, compliance requirements

1. **Developer initiates:**
   ```bash
   pgsquash analyze migrations/*.sql --verbose
   pgsquash squash migrations/*.sql --safety conservative --dry-run
   ```

2. **Review analysis output**

3. **Create consolidation PR:**
   ```bash
   pgsquash squash migrations/*.sql --output squashed/ --backup --rollback
   pgsquash validate migrations/ squashed/
   ```

4. **Team review:**
   - Schema diff review
   - Validation results
   - Rollback plan

5. **Deployment:**
   - Apply in staging first
   - Monitor for 24-48 hours
   - Apply to production

## Safety Features

### Backups

Generate database backup before squashing:

```bash
export PROD_DB_DSN="postgres://..."
pgsquash squash migrations/*.sql \
  --backup \
  --backup-path backups/pre_squash_$(date +%Y%m%d).sql
```

Backup includes:
- Complete schema dump
- All data (optional)
- Extension definitions
- Metadata

### Rollback Scripts

Generate rollback SQL:

```bash
pgsquash squash migrations/*.sql \
  --rollback \
  --rollback-path rollbacks/
```

Rollback script reverses all changes:
```sql
-- Rollback generated 2025-10-06 12:00:00
-- Reverts consolidation changes

DROP TABLE IF EXISTS new_table CASCADE;
ALTER TABLE users DROP COLUMN IF EXISTS new_column;
-- ... etc
```

### Validation

Always validate before production deployment:

```bash
# Most accurate validation
pgsquash validate migrations/ squashed/ --method TWO_CONTAINERS

# Check validation result
if [ $? -eq 0 ]; then
  echo "Validation passed"
else
  echo "Validation failed"
  exit 1
fi
```

## Security Hardening

### Secrets Management

Never commit secrets. Use environment variables or secrets manager:

```bash
# AWS Secrets Manager
export PROD_DB_DSN=$(aws secretsmanager get-secret-value \
  --secret-id prod/db/dsn \
  --query SecretString \
  --output text)

# Vault
export PROD_DB_DSN=$(vault kv get -field=dsn secret/prod/db)
```

### Network Security

- Use SSL/TLS for database connections (`sslmode=require`)
- Restrict API server to internal network
- Use VPN for remote access
- Enable firewall rules

### Access Control

- Limit database user permissions
- Use read-only connections where possible
- Rotate credentials regularly
- Audit access logs

### Docker Security

If using Docker:

```bash
# Don't run as root
docker run --rm \
  --user pgsquash \
  --read-only \
  --security-opt no-new-privileges \
  -v $(pwd)/migrations:/app/migrations:ro \
  -v $(pwd)/output:/app/output \
  pgsquash:latest squash /app/migrations/*.sql --output /app/output
```

## Monitoring and Logging

### Structured Logging

Enable JSON logging for production:

```bash
export LOG_FORMAT="json"
export LOG_LEVEL="info"
```

Output:
```json
{
  "timestamp": "2025-10-06T12:00:00Z",
  "level": "info",
  "message": "Squashing completed",
  "files_processed": 45,
  "duration_ms": 2300,
  "safety_level": "conservative"
}
```

### Performance Metrics

Track key metrics:
- Processing time
- Memory usage
- Statement reduction %
- Validation duration

### Error Tracking

Integrate with error tracking:

```go
// Example with Sentry
import "github.com/getsentry/sentry-go"

sentry.Init(sentry.ClientOptions{
    Dsn: os.Getenv("SENTRY_DSN"),
    Environment: "production",
})

defer sentry.Flush(2 * time.Second)
```

### Alerting

Set up alerts for:
- Validation failures
- Parse errors
- Circular dependencies
- Performance degradation

Example with Prometheus:

```yaml
# alerting_rules.yml
groups:
  - name: pgsquash
    rules:
      - alert: ValidationFailed
        expr: pgsquash_validation_failures_total > 0
        for: 5m
        annotations:
          summary: "pg-squash validation failed"
```

## Performance Optimization

### Large Migration Sets

For 500+ migrations:

```json
{
  "performance": {
    "streaming": true,
    "memory_limit_mb": 1024,
    "batch_size": 100,
    "parallel_processing": true,
    "workers": 8
  }
}
```

### Database Connection Pooling

For paranoid mode with frequent checks:

```bash
export PROD_DB_DSN="postgres://...?pool_max_conns=10&pool_min_conns=2"
```

### Caching

Enable metadata caching:

```json
{
  "metadata": {
    "cache_enabled": true,
    "cache_ttl_minutes": 15
  }
}
```

## Disaster Recovery

### Backup Strategy

1. **Before squashing:**
   ```bash
   pg_dump -Fc prod_db > backup_pre_squash.dump
   ```

2. **After squashing:**
   ```bash
   pg_dump -Fc prod_db > backup_post_squash.dump
   ```

3. **Store backups:**
   - S3/GCS for cloud
   - Network storage for on-prem
   - Retention: 30 days minimum

### Recovery Plan

**If squashed migrations fail:**

1. Stop deployment
2. Restore from backup:
   ```bash
   pg_restore -d prod_db backup_pre_squash.dump
   ```
3. Investigate failure
4. Fix and re-test in staging

**If validation fails:**

1. Don't deploy
2. Review diff output
3. Adjust safety level or rules
4. Re-validate

## Compliance

### Audit Logging

Log all operations:
- Who initiated squashing
- When it occurred
- What files were processed
- Results and validation status

### Change Management

Document:
- Reason for consolidation
- Approval chain
- Testing results
- Rollback procedure

### Data Retention

Preserve:
- Original migrations (never delete)
- Squashed migrations
- Validation results
- Backup files

## Troubleshooting Production Issues

### High Memory Usage

```bash
# Reduce memory limit
pgsquash squash migrations/*.sql \
  --streaming \
  --memory-limit 256 \
  --batch-size 50
```

### Slow Performance

```bash
# Increase workers
pgsquash squash migrations/*.sql --workers 16

# Disable progress tracking
pgsquash squash migrations/*.sql --progress=false
```

### Validation Timeouts

```bash
# Use faster validation method
pgsquash validate migrations/ squashed/ --method SCHEMA_DIFF
```

## Production Runbook

**Weekly Consolidation:**

1. Check migration count: `ls migrations/*.sql | wc -l`
2. If >50, initiate consolidation
3. Run analysis: `pgsquash analyze migrations/*.sql`
4. Review redundancies
5. Create backup: `pg_dump -Fc prod_db > backup.dump`
6. Squash: `pgsquash safe migrations/*.sql --output squashed/`
7. Validate: `pgsquash validate migrations/ squashed/`
8. Review diff if validation fails
9. Test in staging
10. Deploy to production
11. Monitor for 48 hours
12. Archive old migrations

## Further Reading

- [Safety Levels](../safety-levels.md) - Choosing safety mode
- [Docker Guide](docker.md) - Container deployment
- [GitHub Integration](github-integration.md) - PR automation
- [Troubleshooting](../troubleshooting.md) - Common issues
