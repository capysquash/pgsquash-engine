# GitHub Integration

pg-squash integrates with GitHub for automated PR migration analysis and consolidation.

## Features

**Automated PR Analysis**
- Analyzes migrations when PRs are opened/updated
- Posts analysis results as PR comments
- Identifies consolidation opportunities

**Bot Commands**
- `/pgsquash analyze` - Re-analyze migrations
- `/pgsquash consolidate` - Create consolidation PR

**Auto-Consolidation**
- Creates consolidation PRs when threshold is met
- Configurable safety levels
- Preserves data integrity

## Setup

### 1. Create GitHub App

1. Go to **Settings** → **Developer settings** → **GitHub Apps** → **New GitHub App**
2. Configure:
   - **Name**: `pg-squash-bot`
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

Create `.github/pgsquash.yml`:

```yaml
# Auto-analyze on PR events
auto_analyze: true

# Auto-create consolidation PRs
auto_pr: false

# Migration threshold for auto-consolidation
migration_threshold: 15

# Safety level
safety_level: standard  # conservative|standard|aggressive
```

## Usage

### Automatic Analysis

When PR with migrations is created/updated:

1. Bot analyzes migrations
2. Posts comment with results:

```
## pg-squash Migration Analysis

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
- Branch: `pg-squash/consolidate-pr-{number}`
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

      - name: Build pg-squash
        run: |
          git clone https://github.com/capysquash/pg-squash-engine
          cd pg-squash-engine
          go build -o pgsquash cmd/pgsquash/main.go

      - name: Analyze migrations
        run: |
          ./pg-squash-engine/pgsquash analyze migrations/*.sql --verbose

      - name: Squash (if many migrations)
        if: ${{ github.event.pull_request.changed_files > 15 }}
        run: |
          ./pg-squash-engine/pgsquash squash migrations/*.sql \
            --output squashed/ \
            --safety standard

      - name: Validate
        if: ${{ github.event.pull_request.changed_files > 15 }}
        run: |
          ./pg-squash-engine/pgsquash validate migrations/ squashed/
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
