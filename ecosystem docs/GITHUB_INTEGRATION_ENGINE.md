# GitHub Integration Guide

**Current Status: Fully Operational** ✅

> **🎉 IMPLEMENTED:** The CAPYSQUASH GitHub App now supports full webhook automation! When you install the app and open a pull request with migration files, CAPYSQUASH will automatically analyze them and post results as a PR comment.

Automate migration analysis in your pull requests with the CAPYSQUASH GitHub App.

---

## Overview

The CAPYSQUASH GitHub App is designed to automatically analyze migration files when you create or update pull requests. It will post results as PR comments, helping your team catch migration issues before they reach production.

**Live Features:**

- ✅ Automatic analysis on PR open/update
- ✅ Results posted as PR comments
- ✅ GitHub App installation and repository access
- ✅ Manual analysis from CAPYSQUASH dashboard
- ✅ Webhook signature verification
- ✅ Async processing for long-running analysis
- ✅ Admin retry for failed webhooks
- 🚧 Configurable safety levels (coming soon)
- 🚧 Pass/fail checks based on warnings (coming soon)

> **✨ Current Workflow:** Install the GitHub App, and whenever you open a PR with migration files, CAPYSQUASH automatically analyzes them and posts results as a comment. You can also manually trigger analysis from your CAPYSQUASH project dashboard.

---

## Prerequisites

Before setting up the GitHub integration:

- **CAPYSQUASH Account** - Professional plan or higher
- **GitHub Admin Access** - Admin permissions on the repository
- **Organization Setup** - Active CAPYSQUASH organization

---

## Step 1: Create a GitHub App

### 1.1 Register a New GitHub App

1. Go to your GitHub organization settings
2. Navigate to **Settings** → **Developer settings** → **GitHub Apps**
3. Click **New GitHub App**

### 1.2 Configure App Settings

**GitHub App name:**
```
CAPYSQUASH Migration Analyzer
```

**Homepage URL:**
```
https://capysquash.dev
```

**Webhook URL:**
```
https://capysquash.dev/api/webhooks/github
```
*(This endpoint is now live! Replace with your deployment URL if self-hosting)*

**Webhook secret:**
```
Generate a random secret (save this for later):
openssl rand -hex 32
```
*(Store this for when webhook support is implemented)*

### 1.3 Set Permissions

**Repository permissions:**

- **Contents:** Read-only (to access migration files)
- **Pull requests:** Read & write (to post comments)
- **Checks:** Read & write (optional, for PR status checks)

**Organization permissions:**

- **Members:** Read-only (to verify user access)

### 1.4 Subscribe to Events

Select the following events:

- ✅ Pull request (opened, synchronize, reopened)
- ✅ Push (optional, for auto-analysis on main branch)

### 1.5 Generate Private Key

1. Scroll down to **Private keys**
2. Click **Generate a private key**
3. Save the downloaded `.pem` file securely

### 1.6 Save App Credentials

After creating the app, note these values:

- **App ID** - Found at the top of the app page
- **Client ID** - In the OAuth section
- **Client Secret** - Generate one in the OAuth section
- **Private Key** - The `.pem` file you downloaded
- **Webhook Secret** - The secret you generated earlier

---

## Step 2: Install the GitHub App

### 2.1 Install on Your Organization/Account

1. From your GitHub App settings page, click **Install App**
2. Select the organization or account
3. Choose repositories:
   - **All repositories** (easiest)
   - **Only select repositories** (more secure)
4. Click **Install**

### 2.2 Note Installation ID

After installation, you'll be redirected to:
```
https://github.com/settings/installations/INSTALLATION_ID
```

Save the `INSTALLATION_ID` for configuration.

---

## Step 3: Configure CAPYSQUASH

### 3.1 Add Environment Variables

Add these to your deployment environment (Vercel, Railway, etc.):

```bash
# GitHub App Credentials
GITHUB_APP_ID="123456"
GITHUB_APP_CLIENT_ID="Iv1.abc123def456"
GITHUB_APP_CLIENT_SECRET="abc123def456..."
GITHUB_APP_PRIVATE_KEY="-----BEGIN RSA PRIVATE KEY-----\n...\n-----END RSA PRIVATE KEY-----"
GITHUB_WEBHOOK_SECRET="your-webhook-secret-here"
```

**Note:** For `GITHUB_APP_PRIVATE_KEY`, you need to format the `.pem` file:

```bash
# Convert PEM to single-line format
awk 'NF {sub(/\r/, ""); printf "%s\\n",$0;}' your-private-key.pem
```

Or use this script:
```bash
cat your-private-key.pem | tr '\n' '|' | sed 's/|/\\n/g'
```

### 3.2 Deploy Configuration

**Vercel:**
```bash
vercel env add GITHUB_APP_ID
vercel env add GITHUB_APP_CLIENT_ID
vercel env add GITHUB_APP_CLIENT_SECRET
vercel env add GITHUB_APP_PRIVATE_KEY
vercel env add GITHUB_WEBHOOK_SECRET

# Redeploy
vercel --prod
```

**Railway:**
```bash
railway variables set GITHUB_APP_ID="123456"
railway variables set GITHUB_APP_CLIENT_ID="Iv1..."
# ... etc
```

**Docker:**
Add to your `.env.production` file and rebuild.

### 3.3 Verify Webhook Endpoint

Test that your webhook endpoint is accessible:

```bash
curl https://your-domain.com/api/webhooks/github
```

Should return 400 Bad Request with message about missing headers (expected - it requires GitHub webhook headers).

---

## Step 4: Connect GitHub App to CAPYSQUASH

### 4.1 Link Installation in CAPYSQUASH

**Automated Workflow:**

1. Install the GitHub App on your organization/repositories
2. Open a pull request with migration files
3. CAPYSQUASH automatically:
   - Detects migration files in the PR
   - Downloads and analyzes them
   - Posts results as a PR comment

**Manual Workflow:**

1. Log into CAPYSQUASH
2. Go to your **Project** or create a new one
3. When adding migrations, you can access repositories where the GitHub App is installed
4. Select the repository and branch to analyze manually

> **🎉 Automated Analysis:** Webhooks are now live! Every PR with migration files is automatically analyzed.

### 4.2 Configure Analysis Settings (Future Feature)

The following settings will be available once webhook automation is implemented:

In CAPYSQUASH Settings → GitHub Integration:

**Default Safety Level:**
- Conservative (recommended for production repos)
- Standard (balanced)
- Aggressive (development repos)

**PR Comment Settings:**
- Post analysis results as comment
- Update existing comments (vs. new comment each time)
- Include file reduction stats
- Include warnings and recommendations

**Trigger Conditions:**
- Analyze on PR open
- Analyze on PR update (new commits)
- Analyze on every commit (can be noisy)

**File Filters:**
- Include pattern: `migrations/**/*.sql` (customize for your project)
- Exclude pattern: `**/seeds/**` (optional)

---

## Step 5: Test the Integration

### 5.1 Create a Test Pull Request

1. Create a new branch in your repository:
   ```bash
   git checkout -b test-capysquash-integration
   ```

2. Add or modify migration files:
   ```bash
   # Create a test migration
   echo "CREATE TABLE test_users (id SERIAL PRIMARY KEY);" > migrations/001_test.sql

   git add migrations/001_test.sql
   git commit -m "Test: Add migration for CAPYSQUASH"
   git push origin test-capysquash-integration
   ```

3. Open a pull request on GitHub

### 5.2 Verify Analysis

Within 30 seconds, you should see:

1. **GitHub check** - "CAPYSQUASH Analysis" (in progress → complete)
2. **PR comment** - Analysis results from CAPYSQUASH bot

**Example PR Comment:**

```markdown
## 🦫 CAPYSQUASH Migration Analysis

✅ Migrations analyzed successfully!

### 📊 Results
- **Original:** 5 migration files
- **Optimized:** 2 migration files (60% reduction)
- **Time saved:** ~3 minutes per deployment
- **Warnings:** 1

### ⚠️ Warnings
1. Missing index on `posts.author_id` (foreign key without index)

### 📁 Files Analyzed
- migrations/001_create_users.sql
- migrations/002_create_posts.sql
- migrations/003_add_email_to_users.sql
- migrations/004_add_index_users_email.sql
- migrations/005_add_author_to_posts.sql

[View detailed analysis →](https://capysquash.dev/runs/abc123)
```

### 5.3 Check CAPYSQUASH Dashboard

1. Go to CAPYSQUASH dashboard
2. Navigate to **Activity** tab
3. You should see a new analysis run from GitHub
4. Click to view full details

---

## Step 6: Customize Per Repository

### 6.1 Create `.capysquash.yml` Config

Add this file to your repository root:

```yaml
# .capysquash.yml
# CAPYSQUASH GitHub Integration Configuration

# Safety level for analysis
safety_level: standard # paranoid | conservative | standard | aggressive

# File patterns to analyze
include:
  - "migrations/**/*.sql"
  - "db/migrate/*.sql"

# File patterns to exclude
exclude:
  - "**/seeds/**"
  - "**/fixtures/**"
  - "**/*_rollback.sql"

# PR comment settings
pr_comment:
  enabled: true
  update_existing: true
  include_stats: true
  include_warnings: true
  include_recommendations: true

# Pass/fail thresholds
checks:
  # Fail PR if warnings exceed this number
  max_warnings: 5

  # Fail PR if critical warnings found
  fail_on_critical: true

  # Require minimum file reduction percentage
  min_reduction_percent: 20

# Notification settings
notifications:
  # Notify these users when analysis fails
  notify_users:
    - "@team-lead"

  # Post to Slack (requires Slack integration)
  slack_channel: "#migrations"
```

### 6.2 Commit Configuration

```bash
git add .capysquash.yml
git commit -m "Add CAPYSQUASH configuration"
git push
```

The next PR will use this configuration.

---

## Step 7: Advanced Configuration

### 7.1 Monorepo Support

For monorepos with multiple services:

```yaml
# .capysquash.yml
projects:
  - name: "API Service"
    include:
      - "services/api/migrations/**/*.sql"
    safety_level: conservative

  - name: "Auth Service"
    include:
      - "services/auth/db/migrations/**/*.sql"
    safety_level: standard

  - name: "Analytics Service"
    include:
      - "services/analytics/migrations/**/*.sql"
    safety_level: aggressive
```

### 7.2 Custom PR Check Rules

```yaml
checks:
  # Don't fail on warnings, just report
  fail_on_warnings: false

  # But fail on data loss operations
  fail_on_data_loss: true

  # Fail if no migrations are optimized
  require_optimization: true

  # Require certain indexes
  required_indexes:
    - table: "users"
      column: "email"
    - table: "posts"
      column: "author_id"
```

### 7.3 Auto-Apply Optimizations

**⚠️ Use with caution in production!**

```yaml
auto_apply:
  enabled: true

  # Only auto-apply for certain branches
  branches:
    - "development"
    - "staging"

  # Never auto-apply for these branches
  exclude_branches:
    - "main"
    - "production"

  # Require approval from these users
  require_approval_from:
    - "@tech-lead"
```

---

## Troubleshooting

### Webhook Not Triggering

**Problem:** No analysis runs when opening PRs

**Solutions:**

1. **Check webhook deliveries:**
   - GitHub → Settings → Developer settings → GitHub Apps
   - Click your app → Advanced → Recent Deliveries
   - Look for failed deliveries

2. **Verify webhook URL:**
   - Ensure URL is publicly accessible
   - Test: `curl -X POST https://your-domain.com/api/webhooks/github`
   - Should return 400 or 401 (not 404)

3. **Check webhook secret:**
   - Verify `GITHUB_WEBHOOK_SECRET` matches GitHub App settings
   - Re-generate if unsure

4. **Check environment variables:**
   ```bash
   # Verify all required vars are set
   vercel env ls
   ```

### Analysis Fails

**Problem:** Analysis runs but returns errors

**Solutions:**

1. **Check migration file paths:**
   - Ensure files match include patterns in config
   - Check file permissions

2. **Review CAPYSQUASH logs:**
   - Check deployment logs (Vercel/Railway)
   - Look for errors in API routes

3. **Verify CAPYSQUASH account:**
   - Ensure organization has active subscription
   - Check usage limits not exceeded

4. **Test locally:**
   ```bash
   # Run analysis manually
   curl -X POST https://your-domain.com/api/engine/analyze \
     -H "Authorization: Bearer YOUR_API_KEY" \
     -F "files=@migrations/001.sql"
   ```

### PR Comments Not Posting

**Problem:** Analysis completes but no comment appears

**Solutions:**

1. **Check GitHub App permissions:**
   - Ensure "Pull requests" permission is "Read & write"
   - Reinstall app if permissions were changed

2. **Verify bot user:**
   - GitHub App should have bot user created
   - Check bot hasn't been blocked

3. **Check CAPYSQUASH logs:**
   - Look for GitHub API errors
   - Check rate limits

### Duplicate Installations

**Problem:** Multiple CAPYSQUASH comments on PRs

**Solutions:**

1. **Check for multiple installations:**
   - GitHub → Settings → Integrations → GitHub Apps
   - Uninstall duplicate installations

2. **Verify single webhook:**
   - Only one webhook should be configured per repository

---

## Best Practices

### 1. Start Conservative

Begin with `conservative` safety level and gradually move to `standard` or `aggressive` as confidence builds.

### 2. Configure Per Environment

Use different settings for different branches:

```yaml
branches:
  main:
    safety_level: conservative
    fail_on_warnings: true

  development:
    safety_level: aggressive
    fail_on_warnings: false
```

### 3. Review Warnings

Don't auto-merge PRs with warnings. Review them carefully, especially:
- Data loss operations
- Missing indexes on foreign keys
- Complex dependency changes

### 4. Keep Config in Version Control

Commit `.capysquash.yml` to track configuration changes over time.

### 5. Monitor Usage

Check CAPYSQUASH dashboard regularly to track:
- Analysis success rate
- Common warnings
- Time savings

### 6. Team Communication

Ensure team knows:
- How to interpret analysis results
- When to override warnings
- How to adjust configuration

---

## Security Considerations

### Secrets Management

- ✅ Store GitHub App private key securely
- ✅ Rotate webhook secret periodically (every 90 days)
- ✅ Never commit secrets to repository
- ✅ Use environment variable encryption (Vercel, etc.)

### Access Control

- ✅ Limit GitHub App to specific repositories
- ✅ Require approval for auto-apply features
- ✅ Review webhook deliveries regularly
- ✅ Monitor unusual activity in CAPYSQUASH dashboard

### Data Privacy

- ✅ Migration files are analyzed and deleted after processing
- ✅ Results stored encrypted at rest
- ✅ Only organization members can view results
- ✅ Webhook payloads are validated and sanitized

---

## Pricing & Limits

**GitHub Integration Availability:**

- ❌ Free plan - Not available
- ❌ Creator plan - Not available
- ✅ Professional plan - Unlimited repositories
- ✅ Agency plan - Unlimited repositories
- ✅ Enterprise plan - Unlimited + custom features

**Rate Limits:**

- **Webhook processing:** 100 per hour per organization
- **PR comments:** 20 per hour per repository
- **Analysis runs:** Per your subscription plan

---

## Support

Need help with GitHub integration?

- **Documentation:** [https://capysquash.dev/docs](https://capysquash.dev/docs)
- **Email:** support@capysquash.dev
- **GitHub Issues:** [Report integration issues](https://github.com/capysquash/capysquash-platform/issues)

---

## Next Steps

- **[API Documentation](../internal/API/API_REFERENCE.md)** - Programmatic access
- **[Migration Analysis Guide](./MIGRATION_ANALYSIS.md)** - Understanding analysis results

---

**Last Updated:** October 20, 2025
**Integration Version:** v1.0
