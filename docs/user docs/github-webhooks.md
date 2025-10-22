# GitHub Webhooks Guide

**Status**: ☑ Fully Implemented - Production Ready

Stop reviewing migrations manually - catch issues before they hit production.

## Why Automate Migration Analysis?

**The problem:** Your team keeps shipping PRs with:

- Missing indexes on foreign keys
- Broken RLS policies (wrong table name, typo in condition)
- Duplicate column definitions from merge conflicts
- Migrations that work locally but fail in staging

**Manual review doesn't scale:** You're not going to catch `auth.users` vs `public.users` at 4pm on a Friday.

**The solution:** Automatic PR analysis that comments on every migration change with:

- Schema validation results
- Dependency issues
- Performance warnings (missing indexes, sequential scans)
- Integration compatibility (Supabase, Clerk, Prisma)
- Pass/fail checks with configurable thresholds

## What You Get

☑ **Instant PR feedback** - Analysis runs on every push
☑ **Bot commands** - `/pgsquash analyze` or `/pgsquash consolidate` in PR comments
☑ **Zero config for common stacks** - Auto-detects Supabase, Clerk, Prisma
☑ **PR comments with validation results** - Posts analysis findings directly on the PR
☑ **GitHub check runs** - Pass/fail status based on thresholds
☑ **Per-repository configuration** - Use `.capysquash.yml` for custom settings
☑ **Multi-path detection** - Automatically finds migrations/, db/migrations/, supabase/, prisma/
☑ **Platform-style formatted comments** - Rich output with emojis and actionable recommendations

## Integration Options

### Option 1: CAPYSQUASH Platform (Recommended)

- Install GitHub App from CAPYSQUASH
- Web UI for project management
- See [Platform Guide](../../ecosystem%20docs/GITHUB_INTEGRATION.md)

### Option 2: Direct Engine Webhooks (Self-Hosted)

- Deploy engine API server
- Configure webhooks to engine directly
- No platform dependency
- This guide covers this option

## Prerequisites

- pgsquash API server deployed (see [cmd/api-server/README.md](../cmd/api-server/README.md))
- GitHub repository with migration files
- GitHub personal access token (5 minutes to set up)

## Quick Setup (5 Minutes)

### Step 1: Get a GitHub Token (2 minutes)

1. Go to <https://github.com/settings/tokens/new>
2. Give it a name like "pgsquash webhook"
3. Select permissions:
   - ☑ `repo` (Full repository access)
   - ☑ `write:repo_hook` (Webhook management)
4. Click **Generate token**
5. Copy the token (starts with `ghp_`)

### Step 2: Generate Webhook Secret (30 seconds)

```bash
openssl rand -hex 32
```

Copy this secret - you'll need it twice (GitHub webhook settings + API server config).

### Step 3: Deploy API Server

**Option A: Fly.io (Recommended for teams)**

```bash
# Clone and build
git clone https://github.com/CAPYSQUASH/pgsquash-engine
cd pgsquash-engine

# Set secrets
fly secrets set \
  GITHUB_TOKEN="ghp_your_token_here" \
  GITHUB_WEBHOOK_SECRET="your_secret_here"

# Deploy
fly deploy
```

**Option B: Self-hosted Docker**

```bash
docker run -d -p 8080:8080 \
  -e GITHUB_TOKEN="ghp_your_token_here" \
  -e GITHUB_WEBHOOK_SECRET="your_secret_here" \
  --name pgsquash-api \
  pgsquash/api-server:latest
```

**Option C: Local testing**

```bash
# Build
go build -o api-server cmd/api-server/main.go

# Configure
export GITHUB_TOKEN="ghp_your_token_here"
export GITHUB_WEBHOOK_SECRET="your_secret_here"

# Run
./api-server
```

### Step 4: Configure GitHub Webhook (1 minute)

1. Go to your repo → **Settings** → **Webhooks** → **Add webhook**
2. Fill in:
   - **Payload URL:** `https://your-api-server.com/github/webhook`
   - **Content type:** `application/json`
   - **Secret:** Paste your webhook secret from Step 2
   - **Events:** Select "Pull requests" and "Issue comments"
3. Click **Add webhook**
4. GitHub will send a test ping - check that it shows a green ☑

### Step 5: Test the Integration

Open a pull request that touches migration files. Within seconds, you'll see a comment like this:

```markdown
## 🔍 pgsquash Migration Analysis

**Files analyzed:** 3 migrations
**Consolidation potential:** 45%

### Issues Found
⚠️ Missing index on `posts.user_id` (foreign key without index - slow joins)
⚠️ RLS policy `posts_select` references undefined function `is_admin()`
☑ Supabase auth patterns detected and validated

### Recommendations
- Add `CREATE INDEX idx_posts_user_id ON posts(user_id);`
- Define `is_admin()` before creating policies

---
💡 Run `/pgsquash consolidate` to create a cleanup PR
```

**Example PR with migration changes:**

```sql
-- migrations/023_add_comments.sql
CREATE TABLE comments (
  id SERIAL PRIMARY KEY,
  post_id INTEGER REFERENCES posts(id),
  user_id INTEGER REFERENCES users(id),
  content TEXT NOT NULL
);
```

Bot catches:

- ☑ Dependencies are correct (posts and users tables exist)
- ⚠️ Missing indexes on foreign keys
- ℹ️ Could consolidate with earlier table creations (if 15+ migrations exist)

## Webhook Events

### Pull Request Events

**Triggers:**

- Pull request opened
- Pull request synchronized (new commits pushed)

**Actions:**

1. Detects migration files (`.sql` files)
2. Downloads file contents
3. Parses and analyzes migrations
4. Posts analysis comment with:
   - Number of migration files
   - Consolidation ratio
   - Redundancy warnings
   - Consolidation recommendations

**Example comment:**

```markdown
## 🔍 pgsquash Migration Analysis

**Migration Files:** 12
**Consolidation Ratio:** 68.5%

### ☑ Consolidation Recommended

This PR can benefit from migration consolidation:
- Original statements: 142
- After consolidation: ~97 statements
- Savings: ~45 statements

### 🔄 Redundancies Found

- **users_email_idx** (index): Duplicate index on users.email
- **posts_published_at_idx** (index): Overlaps with posts_published_user composite index

---
💡 **Commands:**
- `/pgsquash analyze` - Re-analyze migrations
- `/pgsquash consolidate` - Create consolidation PR
```

### Issue Comment Events

**Triggers:**

- Comment created on pull request

**Supported commands:**

#### `/pgsquash analyze`

Re-run migration analysis on demand.

**Usage:**

```
/pgsquash analyze
```

**Result:** Posts fresh analysis comment with current state.

#### `/pgsquash consolidate`

Create a new pull request with consolidated migrations.

**Usage:**

```
/pgsquash consolidate
```

**Result:**

1. Creates new branch: `pgsquash/consolidate-pr-<number>`
2. Commits consolidated migration file
3. Opens PR with consolidation summary
4. Links back to original PR

**Example consolidation PR:**

```markdown
## 🤖 Automated Migration Consolidation

This PR consolidates migrations from #42.

### Summary
- Original migrations: 12 files
- Consolidated to: 1 file
- Statement reduction: 142 → 97

### Changes
- Removed redundant operations
- Optimized dependency order
- Preserved data integrity

---
*Generated by [pgsquash](https://github.com/CAPYSQUASH/pgsquash)*
```

## Repository Configuration

### Using `.capysquash.yml` (Recommended)

Create `.capysquash.yml` in your repository root or `.github/` folder:

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

# PR comment settings
pr_comment:
  enabled: true
  update_existing: true
  include_stats: true
  include_warnings: true
  include_recommendations: true

# Pass/fail thresholds
checks:
  max_warnings: 5          # Fail if more than 5 warnings
  fail_on_critical: true   # Fail on critical warnings
  fail_on_data_loss: true  # Fail on data loss operations

# Auto-apply (use with caution!)
auto_apply:
  enabled: false
  branches: ["development"]
  exclude_branches: ["main", "production"]
```

**Key Configuration Options:**

| Option                    | Type    | Default    | Description                             |
| ------------------------- | ------- | ---------- | --------------------------------------- |
| `enabled`                 | boolean | `true`     | Enable/disable pgsquash                 |
| `safety_level`            | string  | `standard` | Analysis safety level                   |
| `migration_threshold`     | integer | `15`       | Min files for consolidation suggestions |
| `pr_comment.enabled`      | boolean | `true`     | Post PR comments                        |
| `checks.max_warnings`     | integer | `5`        | Max warnings before failing             |
| `checks.fail_on_critical` | boolean | `true`     | Fail on critical warnings               |
| `auto_apply.enabled`      | boolean | `false`    | Auto-create consolidation PRs           |

See [.capysquash.yml.example](../../.capysquash.yml.example) for complete configuration options.

### Legacy: `.pgsquash.json`

Still supported for backward compatibility:

```json
{
  "enabled": true,
  "auto_pr": false,
  "consolidation_threshold": 15,
  "consolidation_ratio": 0.7,
  "migrations_dir": "migrations"
}
```

**Migration path:** The engine loads `.capysquash.yml` first, then falls back to `.pgsquash.json`.

## Auto-Consolidation

Enable automatic consolidation PR creation for qualifying pull requests

### Consolidation Rules

Auto-consolidation is triggered when **both** conditions are met:

1. **Migration count** ≥ `consolidation_threshold` (default: 15 files)
2. **Consolidation ratio** < `consolidation_ratio` (default: 0.7 or 70%)

**Consolidation ratio calculation:**

```
ratio = optimized_statements / original_statements
```

**Example:**

- Original: 142 statements across 20 files
- Optimized: 97 statements in 1 file
- Ratio: 97/142 = 0.68 (68%)
- Result: Auto-consolidation triggered ☑

### Disable Auto-Consolidation

To analyze only (no automatic PRs):

```json
{
  "enabled": true,
  "auto_pr": false
}
```

Or trigger consolidation manually with `/pgsquash consolidate` command.

## Security

### Webhook Signature Verification

All webhook requests are verified using HMAC-SHA256:

```go
// Automatic verification in API server
signature := r.Header.Get("X-Hub-Signature-256")
mac := hmac.New(sha256.New, []byte(secret))
mac.Write(body)
expectedMAC := "sha256=" + hex.EncodeToString(mac.Sum(nil))
verified := hmac.Equal([]byte(signature), []byte(expectedMAC))
```

**Security guarantees:**

- Requests without valid signatures are rejected (401 Unauthorized)
- HMAC prevents request tampering
- Secret never exposed in logs or responses

### Token Permissions

**Minimum required permissions:**

For personal access tokens:

- `repo` - Read/write repository content
- `write:repo_hook` - Optional, for webhook management

For GitHub Apps:

- **Repository permissions:**
  - Contents: Read & write
  - Pull requests: Read & write
  - Issues: Read & write (for commenting)
- **Subscribe to events:**
  - Pull request
  - Issue comment

### Best Practices

1. **Rotate secrets regularly** - Change webhook secret every 90 days
2. **Use environment variables** - Never commit secrets to repository
3. **Restrict token scope** - Use repository-specific tokens when possible
4. **Enable HTTPS only** - Ensure webhook URL uses HTTPS
5. **Monitor webhook deliveries** - Check GitHub webhook delivery logs regularly

## Deployment Patterns

### Single Repository

Deploy one API server instance per repository for isolated analysis.

```bash
# Deploy to Fly.io
fly deploy --app pgsquash-myrepo

# Configure webhook
Payload URL: https://pgsquash-myrepo.fly.dev/github/webhook
```

### Multi-Repository (GitHub App)

Deploy one API server for multiple repositories using GitHub App authentication.

**Steps:**

1. Create GitHub App:
   - Go to **Settings** → **Developer settings** → **GitHub Apps** → **New**
   - Configure permissions (see above)
   - Set webhook URL: `https://your-api-server.com/github/webhook`
   - Generate and save app credentials

2. Install app on repositories:
   - Install GitHub App on target repositories
   - App automatically configures webhooks

3. Configure API server with app credentials:

```bash
export GITHUB_APP_ID="123456"
export GITHUB_APP_INSTALLATION_ID="7890123"
export GITHUB_APP_PRIVATE_KEY_PATH="/path/to/private-key.pem"
export GITHUB_WEBHOOK_SECRET="your_secret"
```

**Benefits:**

- Centralized management
- Fine-grained permissions
- Easy installation across organization

### Behind API Gateway

Deploy API server behind authentication gateway (e.g., Next.js API routes).

```
GitHub Webhook → API Gateway (auth) → pgsquash API → Response
```

**Example Next.js proxy:**

```typescript
// pages/api/github/webhook.ts
export default async function handler(req: NextApiRequest, res: NextApiResponse) {
  // Verify GitHub signature
  const signature = req.headers['x-hub-signature-256'];
  if (!verifySignature(req.body, signature)) {
    return res.status(401).json({ error: 'Invalid signature' });
  }

  // Forward to pgsquash API
  const response = await fetch(`${process.env.GO_ENGINE_URL}/github/webhook`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-Hub-Signature-256': signature,
    },
    body: JSON.stringify(req.body),
  });

  return res.status(response.status).json(await response.json());
}
```

## Monitoring

### Webhook Delivery Logs

Check webhook delivery status in GitHub:

**Repository** → **Settings** → **Webhooks** → Select webhook → **Recent Deliveries**

**What to check:**

- Response code (should be 200)
- Delivery latency
- Response body
- Failed deliveries (redelivery available)

### API Server Logs

Enable debug logging to see webhook processing details:

```bash
export PGSQUASH_LOG_LEVEL=debug
./api-server
```

**Log entries to watch:**

```
[INFO] ☑ GitHub webhook handler initialized
[INFO] ☑ GitHub webhook endpoint registered at /github/webhook
[DEBUG] Received webhook: pull_request (action: opened)
[DEBUG] Found 12 migration files in PR #42
[INFO] Posted analysis comment to PR #42
```

### Health Monitoring

Set up health check monitoring:

```bash
# Check API server health
curl https://your-api-server.com/health
```

**Response:**

```json
{
  "status": "healthy",
  "timestamp": 1234567890,
  "service": "pgsquash-api",
  "version": "1.0.0"
}
```

Use this endpoint for:

- Uptime monitoring (UptimeRobot, Pingdom, etc.)
- Load balancer health checks
- CI/CD deployment validation

## Troubleshooting

### Webhook Not Firing

**Symptoms:** No bot comments on pull requests

**Solutions:**

1. **Check webhook configuration:**
   - Verify payload URL is correct
   - Ensure "Pull requests" event is selected
   - Check webhook secret matches API server

2. **Check delivery logs:**
   - Go to **Settings** → **Webhooks** → **Recent Deliveries**
   - Look for failed deliveries
   - Check response codes and error messages

3. **Verify API server is running:**
   ```bash
   curl https://your-api-server.com/health
   ```

4. **Check API server logs:**
   ```bash
   # Fly.io
   fly logs

   # Docker
   docker logs <container-id>
   ```

### Bot Comments Not Appearing

**Symptoms:** Webhook delivers successfully but no comment posted

**Solutions:**

1. **Verify GitHub token has correct permissions:**
   - Token needs `repo` scope
   - Check token hasn't expired

2. **Check for migration files:**
   - Bot only comments when `.sql` files are detected
   - Verify migrations are in tracked directory

3. **Check API server logs for errors:**
   ```bash
   grep "ERROR" api-server.log
   ```

### Invalid Signature Errors

**Symptoms:** Webhook returns 401 Unauthorized

**Solutions:**

1. **Verify webhook secret matches:**
   - Check `GITHUB_WEBHOOK_SECRET` in API server
   - Verify secret in GitHub webhook configuration
   - Secrets must match exactly

2. **Regenerate secret:**
   ```bash
   # Generate new secret
   openssl rand -hex 32

   # Update in both places:
   # 1. GitHub webhook settings
   # 2. API server environment variables
   ```

3. **Check for whitespace issues:**
   - Secrets should have no trailing spaces
   - Use quotes when setting environment variables

### Auto-Consolidation Not Working

**Symptoms:** No consolidation PR created despite qualifying migrations

**Solutions:**

1. **Verify repository configuration:**
   ```bash
   # Check .pgsquash.json exists in repo root
   cat .pgsquash.json
   ```

2. **Check thresholds:**
   - Verify migration count ≥ `consolidation_threshold`
   - Verify ratio < `consolidation_ratio`

3. **Check API server logs:**
   - Look for "Auto-consolidation enabled" messages
   - Check for errors during PR creation

4. **Verify token permissions:**
   - Token needs `repo` scope for PR creation
   - GitHub App needs "Pull requests: write" permission

### High Latency

**Symptoms:** Slow webhook responses, timeout errors

**Solutions:**

1. **Enable streaming mode for large migrations:**
   ```json
   {
     "streaming": {
       "enabled": true,
       "batch_size": 50,
       "workers": 8
     }
   }
   ```

2. **Increase server resources:**
   ```bash
   # Fly.io: Scale up memory
   fly scale memory 512

   # Docker: Increase memory limit
   docker run -m 512m ...
   ```

3. **Reduce worker count:**
   - Lower `workers` value if CPU-bound
   - Monitor memory usage during processing

4. **Use caching:**
   - Cache parsed migrations
   - Cache analysis results for unchanged files

## Advanced Configuration

### Custom Migration Directory

Configure non-standard migration directory:

```json
{
  "migrations_dir": "db/migrations",
  "patterns": {
    "include": ["*.sql"],
    "exclude": ["*_test.sql"]
  }
}
```

### Safety Level Configuration

Set default safety level for analysis:

```json
{
  "safety_level": "standard",
  "allow_override": false
}
```

**Options:**

- `paranoid` - No consolidation, maximum safety
- `conservative` - Minimal consolidation
- `standard` - Balanced (default)
- `aggressive` - Maximum consolidation

See [safety-levels.md](./safety-levels.md) for details.

### Custom Comment Templates

Customize bot comment formatting:

```json
{
  "comment_template": {
    "header": "## 🔍 Custom Analysis",
    "consolidation_threshold": 10,
    "show_redundancies": true,
    "show_warnings": true,
    "show_commands": true
  }
}
```

### Rate Limiting

Configure rate limits to prevent abuse:

```json
{
  "rate_limit": {
    "enabled": true,
    "max_requests_per_hour": 100,
    "max_requests_per_day": 500
  }
}
```

## API Reference

### Webhook Endpoint

**Endpoint:** `POST /github/webhook`

**Headers:**

- `X-GitHub-Event`: Event type (`pull_request`, `issue_comment`)
- `X-Hub-Signature-256`: HMAC signature for verification
- `Content-Type`: `application/json`

**Events:**

#### Pull Request Event

```json
{
  "action": "opened",
  "number": 42,
  "pull_request": {
    "number": 42,
    "head": {
      "ref": "feature-branch",
      "sha": "abc123"
    },
    "base": {
      "ref": "main"
    }
  },
  "repository": {
    "full_name": "owner/repo",
    "name": "repo"
  }
}
```

#### Issue Comment Event

```json
{
  "action": "created",
  "issue": {
    "number": 42,
    "pull_request": {}
  },
  "comment": {
    "body": "/pgsquash analyze"
  },
  "repository": {
    "full_name": "owner/repo"
  }
}
```

**Response:**

```
HTTP/1.1 200 OK
```

**Error responses:**

```
HTTP/1.1 401 Unauthorized
Invalid signature

HTTP/1.1 500 Internal Server Error
Failed to process webhook
```

## Examples

### Example 1: Basic Setup

```bash
# 1. Generate secret
SECRET=$(openssl rand -hex 32)

# 2. Deploy API server
fly deploy \
  --env GITHUB_TOKEN="ghp_..." \
  --env GITHUB_WEBHOOK_SECRET="$SECRET"

# 3. Add webhook in GitHub
# Payload URL: https://pgsquash.fly.dev/github/webhook
# Secret: <your-secret>
# Events: Pull requests, Issue comments

# 4. Open PR with migrations
# Bot comments automatically!
```

### Example 2: Auto-Consolidation

```bash
# 1. Create repository config
cat > .pgsquash.json <<EOF
{
  "enabled": true,
  "auto_pr": true,
  "consolidation_threshold": 10,
  "migrations_dir": "migrations"
}
EOF

# 2. Commit and push config
git add .pgsquash.json
git commit -m "Enable pgsquash auto-consolidation"
git push

# 3. Open PR with 10+ migrations
# Bot creates consolidation PR automatically!
```

### Example 3: Manual Consolidation

```bash
# 1. Open PR with migrations
# 2. Wait for analysis comment
# 3. Comment on PR: /pgsquash consolidate
# 4. Bot creates consolidation PR
```

## Related Documentation

- [API Server README](../cmd/api-server/README.md) - API server deployment and configuration
- [Configuration Guide](./configuration.md) - pgsquash configuration reference
- [Environment Variables](./environment-variables.md) - All environment variables
- [Safety Levels](./safety-levels.md) - Consolidation safety levels
- [Troubleshooting](./troubleshooting.md) - Common issues and solutions

## Summary

GitHub webhooks enable automated migration analysis in your development workflow:

1. **Easy setup** - Configure in minutes with personal access token
2. **Automatic analysis** - Get instant feedback on every PR
3. **Bot commands** - Trigger actions with `/pgsquash` commands
4. **Auto-consolidation** - Optionally automate consolidation PRs
5. **Secure** - HMAC signature verification prevents tampering
6. **Flexible deployment** - Single repo or multi-repo with GitHub Apps

The webhook integration makes pgsquash a seamless part of your code review process, catching migration issues before they reach production.
