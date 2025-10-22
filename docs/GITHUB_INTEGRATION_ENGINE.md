# GitHub Integration - Ecosystem Alignment

**Status**: Fully Integrated with CAPYSQUASH Platform ☑

This document describes how the pgsquash-engine GitHub integration works within the CAPYSQUASH ecosystem.

---

## Architecture Overview

The CAPYSQUASH ecosystem consists of three main components:

1. **pgsquash-engine** (this repository) - Core migration analysis and consolidation engine
2. **capysquash-platform** - Web application with project management, authentication, and orchestration
3. **GitHub App / Webhooks** - Automation layer connecting GitHub to the platform and engine

### How They Work Together

```
┌─────────────┐
│   GitHub    │
│  (PR opened)│
└──────┬──────┘
       │ webhook
       ▼
┌─────────────────────┐
│  CAPYSQUASH         │
│  Platform           │◄─── Web UI, Auth, Projects
│  (Next.js)          │
└──────┬──────────────┘
       │ API call
       ▼
┌─────────────────────┐
│  pgsquash-engine    │
│  API Server         │◄─── Analysis Engine
│  (Go)               │
└──────┬──────────────┘
       │ results
       ▼
┌─────────────┐
│   GitHub    │
│  (PR comment)│
└─────────────┘
```

### Integration Modes

The engine supports **two integration modes**:

1. **Platform Mode** (recommended)
   - GitHub webhooks go to CAPYSQUASH platform
   - Platform orchestrates analysis requests to engine
   - Platform manages authentication, projects, and user settings
   - See: [ecosystem docs/GITHUB\_INTEGRATION.md](../ecosystem%20docs/GITHUB_INTEGRATION.md)

2. **Direct Mode** (self-hosted)
   - GitHub webhooks go directly to engine API server
   - Engine handles GitHub App authentication independently
   - Useful for self-hosted deployments without the platform
   - Uses `.capysquash.yml` for per-repository configuration

---

## Configuration: .capysquash.yml

The engine supports per-repository configuration via `.capysquash.yml` files. This aligns with the platform's configuration expectations.

### File Location

Place `.capysquash.yml` in one of these locations:

- Repository root: `.capysquash.yml` or `.capysquash.yaml`
- GitHub folder: `.github/.capysquash.yml` or `.github/.capysquash.yaml`

### Configuration Schema

```yaml
# Enable/disable analysis for this repository
enabled: true

# Safety level: paranoid | conservative | standard | aggressive
safety_level: standard

# Minimum files to trigger consolidation suggestions
migration_threshold: 15

# File patterns to analyze
include:
  - "migrations/**/*.sql"
  - "db/migrate/*.sql"
  - "supabase/migrations/*.sql"
  - "prisma/migrations/*.sql"

# File patterns to exclude
exclude:
  - "**/seeds/**"
  - "**/fixtures/**"

# PR comment formatting
pr_comment:
  enabled: true
  update_existing: true
  include_stats: true
  include_warnings: true
  include_recommendations: true

# Pass/fail thresholds
checks:
  max_warnings: 5
  fail_on_critical: true
  fail_on_warnings: false
  fail_on_data_loss: true
  min_reduction_percent: 0
  require_optimization: false

# Notifications
notifications:
  notify_users: ["@tech-lead"]
  slack_channel: "#migrations"

# Auto-apply (use with caution!)
auto_apply:
  enabled: false
  branches: ["development"]
  exclude_branches: ["main", "production"]
```

See [.capysquash.yml.example](../.capysquash.yml.example) for a complete reference.

### Configuration Precedence

When both configs exist:

1. `.capysquash.yml` - Per-repository settings (GitHub-specific)
2. `pgsquash.config.json` - Engine defaults (local analysis)

**GitHub webhook processing uses this priority:**

```
.capysquash.yml (repo-specific)
    ↓ overrides
pgsquash.config.json (engine defaults)
    ↓ falls back to
DefaultCapySquashConfig() (hardcoded defaults)
```

---

## GitHub Webhook Handling

### Webhook Event Flow

1. **Pull Request Opened/Updated**
   ```
   GitHub → Webhook → Engine API Server → Load .capysquash.yml
                                         → Filter migration files
                                         → Analyze migrations
                                         → Evaluate thresholds
                                         → Post PR comment
                                         → Create check run
   ```

2. **Issue Comment Commands**
   ```
   GitHub Comment: "/pgsquash analyze"
              → Webhook → Engine → Re-analyze → Post results

   GitHub Comment: "/pgsquash consolidate"
              → Webhook → Engine → Create consolidation PR
   ```

### Migration File Detection

The engine automatically detects migrations in these standard paths:

- `migrations/` - Standard migration folder
- `db/migrations/` - Rails-style migrations
- `db/migrate/` - Alternative Rails style
- `supabase/migrations/` - Supabase migrations
- `prisma/migrations/` - Prisma migrations

**Custom paths**: Use `.capysquash.yml` `include` patterns for non-standard locations.

### Filtering Logic

Files must meet ALL criteria:

1. ☑ Has `.sql` extension
2. ☑ Matches at least one `include` pattern (or standard path if no patterns specified)
3. ☑ Does NOT match any `exclude` pattern

Example:

```yaml
include:
  - "migrations/**/*.sql"
exclude:
  - "**/seeds/**"
```

- ☑ `migrations/001_users.sql` - included
- ☑ `migrations/auth/002_roles.sql` - included
- ☒ `migrations/seeds/demo_data.sql` - excluded (matches exclude)
- ☒ `scripts/backup.sql` - excluded (doesn't match include)

---

## PR Comment Format

The engine formats PR comments to align with the CAPYSQUASH platform style:

### Success (No Warnings)

````markdown
## ☑ CAPYSQUASH Migration Analysis

**Status**: Analysis Successful
**Migration Files**: 12
**Potential Consolidation**: 12 → 1 files (92% reduction)

### 📊 Analysis Results

- **Original files**: 12 migration files
- **Optimized**: 1 consolidated file
- **Time saved**: ~120 seconds per deployment

### 💡 Recommendation

You have 12 migration files. Consider using `pgsquash squash` to consolidate them.

```bash
pgsquash squash migrations/*.sql --output consolidated/ --safety standard
````

[View detailed analysis →](https://capysquash.dev/analyze)

---

_Powered by [CAPYSQUASH](https://capysquash.dev) 🦫_

````

### With Warnings

```markdown
## ⚠️ CAPYSQUASH Migration Analysis

**Status**: Analysis Completed with Warnings
**Migration Files**: 8
**Potential Consolidation**: 8 → 1 files (88% reduction)

### ⚠️ Warnings

1. Missing index on `posts.author_id` (foreign key without index)
2. `DROP COLUMN` without `IF EXISTS` may fail in production
3. Function `calculate_total` redefined without version tracking

### 💡 Recommendation

You have 8 migration files. Consider using `pgsquash squash` to consolidate them.

---
_Powered by [CAPYSQUASH](https://capysquash.dev) 🦫_
````

### Failed Checks

```markdown
## ☒ CAPYSQUASH Migration Analysis

**Status**: Analysis Failed
**Migration Files**: 15
**Potential Consolidation**: 15 → 3 files (80% reduction)

### ⚠️ Warnings

1. **CRITICAL**: `DROP TABLE users` will cause data loss
2. **CRITICAL**: Missing foreign key constraint on `posts.author_id`
3. Duplicate index creation on `users.email`
...and 8 more warnings

---
_Powered by [CAPYSQUASH](https://capysquash.dev) 🦫_
```

---

## Pass/Fail Logic

The engine evaluates checks and sets GitHub check run conclusion:

### Check Evaluation Order

1. **Critical Warnings** (`fail_on_critical: true`)
   - Any critical warning → ☒ `failure`

2. **Warning Count** (`max_warnings: 5`)
   - More than 5 warnings → ☒ `failure`

3. **Any Warnings** (`fail_on_warnings: true`)
   - Any warning at all → ☒ `failure`

4. **Data Loss** (`fail_on_data_loss: true`)
   - Data loss detected → ☒ `failure`

5. **Reduction Percentage** (`min_reduction_percent: 20`)
   - Less than 20% reduction → ⚪ `neutral`

6. **Require Optimization** (`require_optimization: true`)
   - No optimization found → ⚪ `neutral`

7. **All Passed**
   - No issues → ☑ `success`
   - Warnings but within limits → ⚪ `neutral`

### Check Run States

- ☑ **success** - No issues, all checks passed
- ⚪ **neutral** - Warnings within acceptable limits
- ☒ **failure** - Critical issues or exceeded thresholds

---

## Ecosystem Responsibilities

### CAPYSQUASH Platform Handles:

- ☑ User authentication and authorization
- ☑ Project management and organization
- ☑ GitHub App installation and OAuth flow
- ☑ Webhook signature verification and routing
- ☑ Rate limiting and quota management
- ☑ Result storage and history
- ☑ Team collaboration features
- ☑ Slack/Discord notifications
- ☑ Usage analytics and billing

### pgsquash-engine Handles:

- ☑ SQL parsing and AST generation
- ☑ Migration analysis and dependency tracking
- ☑ Consolidation logic and optimization
- ☑ Safety level evaluation
- ☑ Warning and recommendation generation
- ☑ `.capysquash.yml` configuration loading
- ☑ GitHub API interactions (direct mode)
- ☑ PR comment formatting
- ☑ Check run creation

### GitHub App Handles:

- ☑ Webhook delivery to platform
- ☑ Repository access permissions
- ☑ PR and commit status updates
- ☑ Installation across organizations

---

## Setup Guides

### For Platform Users

See the comprehensive platform guide:

- [CAPYSQUASH GitHub Integration Guide](../ecosystem%20docs/GITHUB_INTEGRATION.md)

**Quick Start:**

1. Install CAPYSQUASH GitHub App from marketplace
2. Configure repositories in CAPYSQUASH dashboard
3. Add `.capysquash.yml` to your repository (optional)
4. Open a PR with migrations → automatic analysis!

### For Self-Hosted Engine

See the engine-specific guide:

- [GitHub App Setup Guide](./github-app-setup.md)

**Quick Start:**

1. Create GitHub App
2. Generate private key
3. Set environment variables:
   ```bash
   GITHUB_APP_ID=123456
   GITHUB_APP_PRIVATE_KEY="-----BEGIN RSA..."
   GITHUB_WEBHOOK_SECRET=your_secret
   ```
4. Deploy API server
5. Configure webhook URL in GitHub App settings

---

## Migration Path

### From Personal Token to GitHub App

**Before:**

```bash
GITHUB_TOKEN=ghp_xxxxx
```

**After (Platform Mode):**

- Install CAPYSQUASH GitHub App
- Configure in platform dashboard
- Remove `GITHUB_TOKEN` from environment

**After (Direct Mode):**

```bash
GITHUB_APP_ID=123456
GITHUB_APP_PRIVATE_KEY="-----BEGIN RSA..."
GITHUB_WEBHOOK_SECRET=xxxxx
```

**Benefits:**

- ☑ 3x higher rate limits (15k vs 5k requests/hour)
- ☑ Better security (app credentials vs user credentials)
- ☑ Multi-repository support without extra configuration
- ☑ Automatic webhook setup
- ☑ Team ownership (not tied to individual user)

---

## Testing & Debugging

### Test PR Comment Formatting

Create a test PR with migrations:

```bash
# Create test branch
git checkout -b test-capysquash
echo "CREATE TABLE test (id serial);" > migrations/001_test.sql
git add migrations/
git commit -m "Test: trigger CAPYSQUASH analysis"
git push origin test-capysquash

# Open PR on GitHub
# Within 30 seconds, CAPYSQUASH comment should appear
```

### Verify .capysquash.yml Loading

Check engine logs for:

```
INFO: Loaded .capysquash.yml from repository
INFO: Safety level: standard
INFO: Migration threshold: 15
INFO: Include patterns: [migrations/**/*.sql]
```

### Test Check Run Creation

After PR analysis, check GitHub PR:

1. **Checks tab** - Should show "pgsquash/migration-analysis"
2. **Status** - Green ☑, Yellow ⚪, or Red ☒
3. **Details** - Click to see full analysis

### Debug Webhook Delivery

**GitHub → Settings → Developer settings → GitHub Apps → Your App → Advanced → Recent Deliveries**

Look for:

- ☑ Response: 200 OK
- ⚠️ Response: 400/401 - Check webhook secret
- ☒ Response: 500 - Check engine logs

---

## Configuration Examples

### Conservative Production Setup

```yaml
enabled: true
safety_level: conservative
migration_threshold: 10

checks:
  max_warnings: 0
  fail_on_critical: true
  fail_on_warnings: true
  fail_on_data_loss: true
  require_optimization: true

auto_apply:
  enabled: false
```

### Aggressive Development Setup

```yaml
enabled: true
safety_level: aggressive
migration_threshold: 20

pr_comment:
  enabled: true
  include_recommendations: true

checks:
  max_warnings: 10
  fail_on_critical: false
  fail_on_warnings: false

auto_apply:
  enabled: true
  branches: ["development"]
  exclude_branches: ["main"]
```

### Monorepo Setup

```yaml
enabled: true

projects:
  - name: "API Service"
    include:
      - "services/api/migrations/**/*.sql"
    safety_level: conservative

  - name: "Auth Service"
    include:
      - "services/auth/db/migrations/**/*.sql"
    safety_level: standard

checks:
  fail_on_critical: true
  max_warnings: 5
```

---

## API Reference

### Engine Endpoints (Direct Mode)

```
POST /webhooks/github
- Receives GitHub webhook events
- Validates signature
- Processes PR events and comments
- Returns 200 OK on success

POST /api/analyze
- Manual analysis endpoint
- Accepts migration files as multipart/form-data
- Returns analysis results JSON

POST /api/squash
- Consolidation endpoint
- Accepts migration files
- Returns consolidated SQL
```

### GitHub API Usage

The engine uses these GitHub APIs:

- `GET /repos/:owner/:repo/pulls/:number/files` - Get PR files
- `GET /repos/:owner/:repo/contents/:path` - Get file content
- `POST /repos/:owner/:repo/issues/:number/comments` - Post PR comment
- `POST /repos/:owner/:repo/check-runs` - Create check run
- `PATCH /repos/:owner/:repo/check-runs/:id` - Update check run

---

## Troubleshooting

### PR Comments Not Appearing

**Possible causes:**

1. `.capysquash.yml` has `pr_comment.enabled: false`
2. No migration files detected (check `include` patterns)
3. Repository has `enabled: false` in config
4. GitHub App lacks "Pull requests: write" permission

**Solution:**

```yaml
# .capysquash.yml
enabled: true
pr_comment:
  enabled: true
include:
  - "migrations/**/*.sql"
  - "db/migrations/**/*.sql"
```

### Check Runs Always Failing

**Possible causes:**

1. `fail_on_warnings: true` with any warnings present
2. `max_warnings: 0` with warnings present
3. Data loss operations detected with `fail_on_data_loss: true`

**Solution:**

```yaml
checks:
  max_warnings: 5  # Allow some warnings
  fail_on_warnings: false
  fail_on_critical: true  # Only fail on critical
```

### Migration Files Not Detected

**Possible causes:**

1. Files don't match `include` patterns
2. Files match `exclude` patterns
3. Non-standard migration paths

**Solution:**

```yaml
include:
  - "your/custom/path/**/*.sql"
  - "another/path/**/*.sql"
exclude:
  - "**/tests/**"  # Make sure not excluding your files
```

---

## Security Considerations

### Webhook Signature Verification

The engine verifies GitHub webhook signatures using HMAC-SHA256:

```go
signature := "sha256=" + hex.EncodeToString(hmac.New(sha256.New, secret).Sum(body))
```

### Private Key Security

**Best practices:**

- ☑ Store private key in environment variable or secrets manager
- ☑ Use `GITHUB_APP_PRIVATE_KEY_PATH` for file-based keys
- ☑ Set file permissions to 600 (read-only by owner)
- ☑ Rotate keys every 90 days
- ☒ Never commit private keys to repository

### Rate Limiting

GitHub App rate limits:

- **With GitHub App**: 15,000 requests/hour per installation
- **With Personal Token**: 5,000 requests/hour
- **Unauthenticated**: 60 requests/hour

The engine respects rate limits and includes rate limit info in logs.

---

## Related Documentation

- **Platform Integration**: [ecosystem docs/GITHUB\_INTEGRATION.md](../ecosystem%20docs/GITHUB_INTEGRATION.md)
- **GitHub App Setup**: [github-app-setup.md](./github-app-setup.md)
- **Configuration Reference**: [user docs/configuration.md](./user%20docs/configuration.md)
- **Webhook Guide**: [user docs/github-webhooks.md](./user%20docs/github-webhooks.md)

---

## Changelog

- **v1.0.0** (Oct 2025) - Initial GitHub integration alignment with CAPYSQUASH platform
- Added `.capysquash.yml` configuration support
- Enhanced PR comment formatting to match platform style
- Implemented pass/fail check logic based on thresholds
- Added multi-path migration detection
- Aligned webhook handling with platform expectations

---

**Last Updated**: October 20, 2025
**Integration Version**: v1.0
**Status**: Production Ready ☑
