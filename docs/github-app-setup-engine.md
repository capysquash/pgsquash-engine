# GitHub App Setup Guide

Complete guide to setting up the pgsquash GitHub App for automated migration analysis and consolidation.

## Table of Contents

- [Why Use a GitHub App?](#why-use-a-github-app)
- [Prerequisites](#prerequisites)
- [Quick Setup (5 Minutes)](#quick-setup-5-minutes)
- [Detailed Setup](#detailed-setup)
- [Configuration](#configuration)
- [Testing](#testing)
- [Troubleshooting](#troubleshooting)

---

## Why Use a GitHub App?

### GitHub App vs Personal Access Token

| Feature | Personal Token | GitHub App |
|---------|---------------|------------|
| **Multi-repo support** | One token per repo | One app for unlimited repos |
| **Permissions** | Broad, user-level | Fine-grained, repo-specific |
| **Security** | User credentials | App credentials + installation tokens |
| **Rate limits** | 5,000 req/hour | 15,000 req/hour per installation |
| **Revocation** | Revokes user access | Revokes only app access |
| **Installation** | Manual webhook setup | Automatic webhook configuration |
| **Audit trail** | User actions | App actions (clearer attribution) |

### Benefits for Your Team

✅ **Better Security** - App credentials instead of user tokens
✅ **Higher Rate Limits** - 3x more API calls
✅ **Easier Management** - One app for all repositories
✅ **Automatic Webhooks** - No manual webhook configuration
✅ **Team Ownership** - Not tied to a specific user account
✅ **Per-Repo Configuration** - Use `.capysquash.yml` for custom settings

---

## Prerequisites

- GitHub organization or personal account
- Deployed API server (see [API Server README](../cmd/api-server/README.md))
- API server must be accessible via HTTPS
- 10 minutes of setup time

---

## Quick Setup (5 Minutes)

### Step 1: Create GitHub App (2 minutes)

1. Go to https://github.com/settings/apps/new
2. **GitHub App name**: `pgsquash` (or `pgsquash-{your-org}`)
3. **Homepage URL**: `https://capysquash.dev`
4. **Webhook URL**: `https://your-api-server.fly.dev/github/webhook`
5. **Webhook secret**: Generate with: `openssl rand -hex 32`

**Permissions:**
- ✅ Repository permissions:
  - Contents: **Read & write**
  - Pull requests: **Read & write**
  - Issues: **Read & write**
  - Checks: **Read & write**
  - Metadata: **Read-only**

**Subscribe to events:**
- ✅ Pull request
- ✅ Pull request review
- ✅ Pull request review comment
- ✅ Issue comment
- ✅ Push

6. Click **Create GitHub App**

### Step 2: Generate Private Key (1 minute)

1. Scroll down to **Private keys**
2. Click **Generate a private key**
3. Save the `.pem` file securely
4. Note the **App ID** (shown at the top of the page)

### Step 3: Install the App (1 minute)

1. Click **Install App** in the left sidebar
2. Select your organization or personal account
3. Choose **All repositories** or **Only select repositories**
4. Click **Install**
5. Note the **Installation ID** from the URL (after `/settings/installations/`)

### Step 4: Configure API Server (1 minute)

Set environment variables on your API server:

```bash
# Fly.io
fly secrets set \
  GITHUB_APP_ID=123456 \
  GITHUB_WEBHOOK_SECRET=your_webhook_secret \
  GITHUB_APP_PRIVATE_KEY="$(cat pgsquash.2024-10-20.private-key.pem)"

# Or use file path
fly secrets set \
  GITHUB_APP_ID=123456 \
  GITHUB_WEBHOOK_SECRET=your_webhook_secret \
  GITHUB_APP_PRIVATE_KEY_PATH=/app/private-key.pem
```

**Railway:**
```bash
railway variables set GITHUB_APP_ID=123456
railway variables set GITHUB_WEBHOOK_SECRET=your_webhook_secret
railway variables set GITHUB_APP_PRIVATE_KEY="$(cat pgsquash.*.private-key.pem)"
```

**Docker:**
```bash
docker run -d \
  -e GITHUB_APP_ID=123456 \
  -e GITHUB_WEBHOOK_SECRET=your_webhook_secret \
  -e GITHUB_APP_PRIVATE_KEY="$(cat pgsquash.*.private-key.pem)" \
  -p 8080:8080 \
  pgsquash/api-server:latest
```

### Step 5: Test (30 seconds)

1. Open a pull request with migration changes
2. Watch for pgsquash comment within 30 seconds
3. ✅ Success! You're done.

---

## Detailed Setup

### Creating the GitHub App

#### Option A: Manual Creation

1. Navigate to GitHub App creation page:
   - **Organization**: `https://github.com/organizations/{org}/settings/apps/new`
   - **Personal**: `https://github.com/settings/apps/new`

2. **Fill in basic information:**

   - **GitHub App name**: `pgsquash` (must be globally unique)
     - If taken, try: `pgsquash-{your-org}` or `pgsquash-{your-company}`

   - **Description**:
     ```
     Autopilot for your Postgres migrations - automatically consolidate and
     optimize migration files with safety guarantees
     ```

   - **Homepage URL**: `https://capysquash.dev`

   - **Callback URL**: `https://capysquash.dev/github/callback`
     - Required if you want OAuth integration
     - Can leave blank if only using webhooks

3. **Webhook configuration:**

   - **Webhook URL**: `https://your-api-server.fly.dev/github/webhook`
     - Replace `your-api-server.fly.dev` with your actual API server domain
     - Must be HTTPS
     - Must be publicly accessible

   - **Webhook secret**: Generate a secure random string:
     ```bash
     openssl rand -hex 32
     ```
     Save this - you'll need it for API server configuration

   - **Active**: ✅ Check this box

4. **Permissions:** Set the minimum required permissions:

   | Permission | Access | Reason |
   |------------|--------|--------|
   | Contents | Read & write | Read migrations, create consolidation PRs |
   | Pull requests | Read & write | Comment on PRs, create consolidation PRs |
   | Issues | Read & write | Post analysis comments |
   | Checks | Read & write | Create check runs with analysis results |
   | Metadata | Read-only | Repository metadata (automatically granted) |

5. **Subscribe to events:** Select these events:

   - ✅ **Pull request** - Triggers analysis when PRs are opened/synchronized
   - ✅ **Pull request review** - Allows responding to reviews
   - ✅ **Pull request review comment** - Process review comments
   - ✅ **Issue comment** - Listen for bot commands (e.g., `/pgsquash analyze`)
   - ✅ **Push** - Optional: Analyze on every push

6. **Where can this GitHub App be installed?**

   - Select **Any account** if you want to make it publicly available
   - Select **Only on this account** if it's for internal use only

7. Click **Create GitHub App**

#### Option B: Using Manifest

You can use the provided manifest file for faster setup:

1. Go to: `https://github.com/settings/apps/new?state=setup`

2. Upload the manifest:
   ```bash
   cat .github/github-app-manifest.json
   ```

3. Update the webhook URL in the manifest before uploading

4. Click **Create GitHub App from manifest**

### Generating and Managing Private Key

The private key is used to authenticate as the GitHub App and generate installation access tokens.

#### Generate Private Key

1. On your GitHub App settings page, scroll to **Private keys**
2. Click **Generate a private key**
3. Save the downloaded `.pem` file securely
4. The file name format: `pgsquash.YYYY-MM-DD.private-key.pem`

**Security best practices:**

```bash
# Set restrictive permissions
chmod 600 pgsquash.*.private-key.pem

# Store in secure location
mkdir -p ~/.pgsquash/keys
mv pgsquash.*.private-key.pem ~/.pgsquash/keys/

# Never commit to git
echo "*.private-key.pem" >> .gitignore
```

#### Rotate Private Key

For security, rotate your private key every 90 days:

1. Generate a new private key on GitHub
2. Update your API server environment variables
3. Delete the old key from GitHub after confirming the new one works
4. Securely delete the old `.pem` file

### Installing the App

#### Install on Organization

1. From your GitHub App settings, click **Install App** in left sidebar
2. Select your organization
3. Choose repository access:
   - **All repositories** - Automatically includes new repos
   - **Only select repositories** - More control, must add repos manually
4. Click **Install**

#### Install on Personal Account

Same process as organization, but select your personal account.

#### Finding Installation ID

After installation, you'll be redirected to a URL like:
```
https://github.com/settings/installations/12345678
```

The number `12345678` is your **Installation ID**. Save this for API server configuration.

Alternatively, you can find it via API:
```bash
# List all installations (requires App JWT)
curl -H "Authorization: Bearer {APP_JWT}" \
  https://api.github.com/app/installations
```

---

## Configuration

### API Server Environment Variables

**Required variables:**

```bash
# GitHub App credentials
GITHUB_APP_ID=123456                    # Found on GitHub App settings page
GITHUB_WEBHOOK_SECRET=abc123...         # Webhook secret you generated

# Private key (choose one method):
# Method 1: Inline (preferred for platforms like Fly.io, Railway)
GITHUB_APP_PRIVATE_KEY="-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA...
-----END RSA PRIVATE KEY-----"

# Method 2: File path (preferred for self-hosted)
GITHUB_APP_PRIVATE_KEY_PATH=/app/keys/private-key.pem
```

**Optional variables:**

```bash
# OAuth configuration (for web-based installation flow)
GITHUB_CLIENT_ID=Iv1.abcd1234...        # Found on GitHub App settings
GITHUB_CLIENT_SECRET=abc123def456...    # Generated on GitHub App settings
GITHUB_REDIRECT_URL=https://capysquash.dev/github/callback

# API server configuration
PORT=8080
CORS_ORIGIN=https://capysquash.dev
DATABASE_URL=postgres://...             # For storing GitHub installations
```

### Platform-Specific Configuration

#### Fly.io

**Set secrets:**
```bash
# Read private key from file
fly secrets set GITHUB_APP_PRIVATE_KEY="$(cat pgsquash.*.private-key.pem)"

# Or set other variables
fly secrets set \
  GITHUB_APP_ID=123456 \
  GITHUB_WEBHOOK_SECRET=your_secret
```

**Verify secrets:**
```bash
fly secrets list
```

**Update fly.toml:**
```toml
[env]
  PORT = "8080"
  CORS_ORIGIN = "https://capysquash.dev"

[deploy]
  strategy = "rolling"
```

#### Railway

**Via CLI:**
```bash
railway variables set GITHUB_APP_ID=123456
railway variables set GITHUB_WEBHOOK_SECRET=your_secret
railway variables set GITHUB_APP_PRIVATE_KEY="$(cat pgsquash.*.private-key.pem)"
```

**Via Dashboard:**
1. Go to your project
2. Click **Variables**
3. Add each variable manually
4. Railway automatically redeploys

#### Docker

**Using environment file:**
```bash
# Create .env file
cat > .env << EOF
GITHUB_APP_ID=123456
GITHUB_WEBHOOK_SECRET=your_secret
GITHUB_APP_PRIVATE_KEY=$(cat pgsquash.*.private-key.pem)
EOF

# Run container
docker run -d \
  --env-file .env \
  -p 8080:8080 \
  pgsquash/api-server:latest
```

**Using secrets:**
```bash
# Create secret
docker secret create pgsquash_private_key pgsquash.*.private-key.pem

# Use in docker-compose.yml
services:
  api-server:
    image: pgsquash/api-server:latest
    environment:
      - GITHUB_APP_ID=123456
      - GITHUB_WEBHOOK_SECRET=your_secret
    secrets:
      - pgsquash_private_key
    environment:
      - GITHUB_APP_PRIVATE_KEY_PATH=/run/secrets/pgsquash_private_key

secrets:
  pgsquash_private_key:
    external: true
```

#### Self-Hosted (systemd)

**Create service file:**
```bash
sudo nano /etc/systemd/system/pgsquash-api.service
```

**Service configuration:**
```ini
[Unit]
Description=pgsquash API Server
After=network.target

[Service]
Type=simple
User=pgsquash
WorkingDirectory=/opt/pgsquash
Environment="GITHUB_APP_ID=123456"
Environment="GITHUB_WEBHOOK_SECRET=your_secret"
Environment="GITHUB_APP_PRIVATE_KEY_PATH=/opt/pgsquash/keys/private-key.pem"
Environment="PORT=8080"
ExecStart=/opt/pgsquash/api-server
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

**Enable and start:**
```bash
sudo systemctl daemon-reload
sudo systemctl enable pgsquash-api
sudo systemctl start pgsquash-api
sudo systemctl status pgsquash-api
```

### Repository Configuration

**Option 1: `.capysquash.yml` (Recommended)**

Create `.capysquash.yml` in the root or `.github/` folder of each repository:

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
```

See [.capysquash.yml.example](../.capysquash.yml.example) for complete configuration options.

**Option 2: `.pgsquash.config.json` (Legacy)**

Create `.pgsquash.config.json` in the root of each repository:

```json
{
  "enabled": true,
  "auto_analyze": true,
  "auto_pr": false,
  "migration_threshold": 15,
  "consolidation_ratio": 0.7,
  "safety_level": "standard",
  "migrations_dir": "migrations",
  "patterns": {
    "include": ["**/*.sql"],
    "exclude": ["**/archive/**", "**/*.bak.sql"]
  }
}
```

**Configuration options:**

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | boolean | `true` | Enable/disable pgsquash for this repo |
| `auto_analyze` | boolean | `true` | Automatically analyze PRs |
| `auto_pr` | boolean | `false` | Automatically create consolidation PRs |
| `migration_threshold` | number | `15` | Min migrations to trigger auto-consolidation |
| `consolidation_ratio` | number | `0.7` | Max ratio to trigger (lower = more consolidation) |
| `safety_level` | string | `"standard"` | `paranoid`, `conservative`, `standard`, `aggressive` |
| `migrations_dir` | string | `"migrations"` | Path to migration directory |

---

## Testing

### Test Webhook Delivery

1. **Create a test PR:**
   ```bash
   git checkout -b test-pgsquash
   echo "CREATE TABLE test_table (id serial PRIMARY KEY);" > migrations/001_test.sql
   git add migrations/001_test.sql
   git commit -m "Test pgsquash integration"
   git push origin test-pgsquash
   ```

2. **Open PR on GitHub**

3. **Check for bot comment within 30 seconds:**
   ```markdown
   ## 🔍 pgsquash Migration Analysis

   **Migration Files:** 1
   **Consolidation Ratio:** 100.0%

   ### ℹ️ Consolidation Status
   Migrations appear well-optimized. No consolidation needed.
   ```

4. **Verify webhook delivery:**
   - Go to your GitHub App settings
   - Click **Advanced** tab
   - Check **Recent Deliveries**
   - Should show 200 response

### Test Bot Commands

**Comment on PR:**
```
/pgsquash analyze
```

**Expected:** Bot re-analyzes and posts updated comment

### Test Check Run (if enabled)

1. Open PR with migrations
2. Go to **Checks** tab
3. Should see `pgsquash / migration-analysis`
4. Click to view detailed analysis

---

## Troubleshooting

### Webhook Not Firing

**Symptoms:** No bot comments on PRs

**Check:**
1. Verify webhook URL is correct and accessible:
   ```bash
   curl https://your-api-server.fly.dev/health
   ```

2. Check webhook deliveries in GitHub App settings
   - Look for failed deliveries
   - Check error messages

3. Verify webhook secret matches:
   ```bash
   fly secrets list | grep WEBHOOK_SECRET
   ```

4. Check API server logs:
   ```bash
   fly logs
   ```

### "Installation not found" Error

**Symptoms:** Error message about missing installation

**Solutions:**
1. Verify app is installed on the repository
2. Check installation ID in logs
3. Reinstall the app if necessary

### Authentication Errors

**Symptoms:** 401 Unauthorized or authentication failures

**Check:**
1. Verify App ID is correct:
   ```bash
   fly secrets list | grep APP_ID
   ```

2. Verify private key is valid:
   ```bash
   # Test key format
   openssl rsa -in private-key.pem -check
   ```

3. Ensure private key matches the active key on GitHub

### Rate Limit Errors

**Symptoms:** API returns 403 with rate limit message

**Solutions:**
1. Check current rate limits:
   ```bash
   curl -H "Authorization: Bearer {INSTALLATION_TOKEN}" \
     https://api.github.com/rate_limit
   ```

2. Implement rate limit backoff in your code
3. Consider caching responses

### Bot Not Responding to Commands

**Symptoms:** `/pgsquash` commands don't work

**Check:**
1. Verify "Issue comment" event is enabled in GitHub App
2. Check command syntax is correct
3. Verify bot has `issues: write` permission
4. Check API server logs for command processing

---

## Next Steps

- [GitHub Webhooks Guide](../docs/user%20docs/github-webhooks.md) - Complete webhook documentation
- [API Server README](../cmd/api-server/README.md) - API server deployment
- [Configuration Guide](../docs/user%20docs/configuration.md) - Advanced configuration options

---

## Support

- **Issues**: https://github.com/capysquash/pgsquash-engine/issues
- **Discussions**: https://github.com/capysquash/pgsquash-engine/discussions
- **Documentation**: https://capysquash.dev/docs
