# GitHub Integration Deployment Guide

**Status**: ✅ Fully Implemented - Production Ready

pgsquash integrates with GitHub for automated PR migration analysis and consolidation via three deployment architectures.

## Integration Modes

### 1. Platform Mode (Recommended)
- CAPYSQUASH platform orchestrates analysis
- Platform receives webhooks from GitHub
- Platform calls engine API for analysis
- Best for hosted/SaaS users

### 2. Direct Mode (Self-Hosted)
- Engine receives webhooks directly
- No platform dependency
- Perfect for self-hosted deployments
- Uses `.capysquash.yml` configuration

### 3. Hybrid Mode
- Platform provides web UI
- Engine handles webhook automation
- Can use both simultaneously

See [GITHUB_INTEGRATION.md](../../GITHUB_INTEGRATION.md) for architecture details.

## Features

**Automated PR Analysis**

- ✅ Analyzes migrations when PRs are opened/updated
- ✅ Posts platform-style formatted PR comments
- ✅ Identifies consolidation opportunities
- ✅ Multi-path detection (migrations/, db/migrations/, supabase/, prisma/)
- ✅ Configurable via `.capysquash.yml`
- ✅ Pass/fail checks with thresholds

**Bot Commands**

- `/pgsquash analyze` - Re-analyze migrations
- `/pgsquash consolidate` - Create consolidation PR

**Auto-Consolidation**

- Creates consolidation PRs when threshold is met
- Configurable safety levels
- Preserves data integrity
- Branch-specific rules

## Setup

### 1. Create GitHub App

1. Go to **Settings** → **Developer settings** → **GitHub Apps** → **New GitHub App**
2. Configure:
   - **Name**: `pgsquash-bot`
   - **Homepage URL**: Your API server URL
   - **Webhook URL**: `https://your-server.com/github/webhook`
   - **Webhook Secret**: Generate secure random string

**Permissions:**

- Contents: Read & Write
- Pull requests: Read & Write
- Webhooks: Read

**Events:**

- Pull request
- Issue comment

### 2. Environment Configuration

```bash
# GitHub App credentials
export GITHUB_TOKEN="ghp_your_token"
export GITHUB_WEBHOOK_SECRET="your_secret"

# OAuth (optional)
export GITHUB_CLIENT_ID="your_client_id"
export GITHUB_CLIENT_SECRET="your_client_secret"
export GITHUB_REDIRECT_URL="http://localhost:8080/github/callback"

# Server
export PORT="8080"
export CORS_ORIGIN="*"
```

### 3. Deploy API Server

```bash
# Build
go build -o api-server api-server/main.go

# Run
./api-server
```

Server logs:

```
✓ GitHub webhook handler initialized
✓ Webhook endpoint: /github/webhook
✓ OAuth endpoints: /github/login, /github/callback
```

### 4. Configure Webhook

In repository **Settings** → **Webhooks** → **Add webhook**:

- **Payload URL**: `https://your-server.com/github/webhook`
- **Content type**: `application/json`
- **Secret**: Match `GITHUB_WEBHOOK_SECRET`
- **Events**: Pull requests, Issue comments

### 5. Repository Configuration

**Recommended: Create `.capysquash.yml`** in repository root or `.github/` folder:

```yaml
# Enable/disable pgsquash for this repository
enabled: true

# Safety level: paranoid | conservative | standard | aggressive
safety_level: standard

# Minimum files to trigger consolidation suggestions
migration_threshold: 15

# File patterns to analyze
include:
  - "migrations/**/*.sql"
  - "db/migrations/**/*.sql"
  - "supabase/migrations/**/*.sql"

# File patterns to exclude
exclude:
  - "**/seeds/**"
  - "**/fixtures/**"

# PR comment settings
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
  fail_on_data_loss: true
  min_reduction_percent: 0

# Notifications
notifications:
  notify_users: ["@tech-lead"]
  slack_channel: ""

# Auto-apply (use with caution!)
auto_apply:
  enabled: false
  branches: ["development"]
  exclude_branches: ["main", "production"]
```

See [.capysquash.yml.example](../../../.capysquash.yml.example) for complete options.

**Legacy: `.github/pgsquash.yml`** (still supported):

```yaml
auto_analyze: true
auto_pr: false
migration_threshold: 15
safety_level: standard
```

## Usage

### Automatic Analysis

When PR with migrations is created/updated:

1. Bot analyzes migrations
2. Posts comment with results:

```
## pgsquash Migration Analysis

Migration Files: 20
Consolidation Ratio: 65.0%

### Consolidation Recommended
- Original: 150 statements
- Optimized: ~45 statements
- Savings: ~105 statements

### Redundancies
- users table: Multiple ALTER TABLE can be combined
- calculate_total function: Duplicate definition

Commands:
- /pgsquash analyze
- /pgsquash consolidate
```

### Bot Commands

**Re-analyze:**

```
/pgsquash analyze
```

**Create consolidation PR:**

```
/pgsquash consolidate
```

Creates:

- Branch: `pgsquash/consolidate-pr-{number}`
- PR with consolidated migrations
- Links to original PR

## Workflows

### Development (Fast)

`.github/pgsquash.yml`:

```yaml
auto_analyze: true
auto_pr: true
migration_threshold: 10
safety_level: standard
```

Flow:

1. Developer creates PR with migrations
2. Bot analyzes automatically
3. If ≥10 migrations, bot creates consolidation PR
4. Team reviews both PRs
5. Merge consolidation PR

### Production (Safe)

`.github/pgsquash.yml`:

```yaml
auto_analyze: true
auto_pr: false
migration_threshold: 20
safety_level: conservative
```

Flow:

1. Developer creates PR
2. Bot analyzes, posts report
3. Team reviews analysis
4. Manual `/pgsquash consolidate` if approved
5. Extra scrutiny on consolidation PR
6. Validate before merge

## Security

**Webhook Verification:**

- HMAC-SHA256 signature verification
- Validates payload integrity
- Uses `GITHUB_WEBHOOK_SECRET`

**OAuth Protection:**

- CSRF state parameter
- Prevents session hijacking
- Validates on callback

**Token Permissions:**

- Minimal required permissions
- Repository access only
- No admin rights

## API Endpoints

### Webhook

```
POST /github/webhook
```

Receives PR and comment events.

### OAuth

```
GET /github/login?state={random}
GET /github/callback?code={code}&state={state}
```

OAuth flow for user authentication.

## GitHub Actions Integration

```yaml
# .github/workflows/migrations.yml
name: Migration Analysis

on:
  pull_request:
    paths:
      - 'migrations/**/*.sql'

jobs:
  analyze:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Setup Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.25'

      - name: Build pgsquash
        run: |
          git clone https://github.com/CAPYSQUASH/pgsquash-engine
          cd pgsquash-engine
          go build -o pgsquash cmd/pgsquash/main.go

      - name: Analyze migrations
        run: |
          ./pgsquash-engine/pgsquash analyze migrations/*.sql --verbose

      - name: Squash (if many migrations)
        if: ${{ github.event.pull_request.changed_files > 15 }}
        run: |
          ./pgsquash-engine/pgsquash squash migrations/*.sql \
            --output squashed/ \
            --safety standard

      - name: Validate
        if: ${{ github.event.pull_request.changed_files > 15 }}
        run: |
          ./pgsquash-engine/pgsquash validate migrations/ squashed/
```

## Troubleshooting

### Bot Not Responding

1. Check webhook delivery in GitHub Settings → Webhooks
2. Verify webhook secret matches environment variable
3. Check server logs for errors
4. Ensure GitHub App has correct permissions

### Analysis Errors

1. Invalid SQL syntax - Fix migration syntax
2. Parser errors - Check PostgreSQL compatibility
3. Timeout - Adjust for large migration sets

### OAuth Issues

1. Invalid redirect URI - Must match GitHub App config
2. State mismatch - CSRF protection triggered, retry
3. Token expired - Re-authenticate via `/github/login`

## Further Reading

- [Production Guide](production.md) - Production deployment
- [Docker Guide](docker.md) - Container usage
- [CLI Reference](../cli-reference.md) - Command options
