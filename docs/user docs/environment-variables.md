# Environment Variables

Environment variables let you configure pgsquash without editing config files - great for different environments, CI/CD pipelines, and team setups.

## Quick Reference by Use Case

### Debugging a tricky migration?

- **`PGSQUASH_LOG_LEVEL=debug`** - See exactly what pgsquash is thinking (CLI only)

### Testing against your real database?

- **`PROD_DB_DSN`** - Validate squashed output matches your production schema (CLI only)

### Setting up the API Server?

- **`JWT_SECRET`** - Required for API authentication
- **`DATABASE_URL`** - Required for operation tracking
- **`PORT`** - API server port (default: 8080)
- **`CORS_ORIGIN`** - Allowed origins for CORS

### Setting up CI/CD or webhooks?

- **`ANTHROPIC_API_KEY`** / **`OPENAI_API_KEY`** - Enable AI-powered analysis (CLI and API)
- **`GITHUB_TOKEN`** - Automate PR comments and consolidation (API server only)
- **`GITHUB_WEBHOOK_SECRET`** - Secure webhook endpoints (API server only)

---

## Core Configuration

### PGSQUASH\_LOG\_LEVEL

**Type:** String
**Values:** `info` (default), `debug`
**When to use:** Hit a weird squashing error? Crank up the logs.

```bash
export PGSQUASH_LOG_LEVEL=debug
pgsquash analyze migrations/*.sql
```

You'll see:

- Which patterns are detected (Supabase RLS, Clerk schemas, etc.)
- Why statements are being preserved or merged
- Dependency resolution decisions
- Parser output for problematic SQL

---

### PROD\_DB\_DSN

**Type:** String (PostgreSQL connection string)
**When to use:** Validate squashed migrations against your actual production schema

```bash
export PROD_DB_DSN="postgres://user:pass@host:5432/dbname"
pgsquash squash migrations/*.sql --backup --rollback
```

**Format:** `postgres://username:password@hostname:port/database`

⚠️ **Never commit database credentials**
Use `.env.local` or your deployment Platform's secrets:

- **Vercel:** Environment Variables in project settings
- **Fly.io:** `flyctl secrets set PROD_DB_DSN="..."`
- **Railway:** Environment variables in service settings
- **Docker:** `--env-file .env.production`

**Supabase users:** Get your connection string from Project Settings → Database → Connection String (use "Connection Pooling" for production)

---

### PGSQUASH\_VERSION

**Type:** String
**When to use:** CI/CD builds, API deployments

```bash
export PGSQUASH_VERSION=1.0.0
pgsquash health
```

Set this in your build pipeline to track which version analyzed migrations.

---

## API Server Configuration

**When to use:** Running the REST API server for hosted orchestration, GitHub webhooks, or team collaboration.

### JWT\_SECRET

**Type:** String (required for API server)
**When to use:** Securing API endpoints with JWT authentication

```bash
# Generate a secure secret
openssl rand -base64 32

export JWT_SECRET="your-secure-jwt-secret"
api-server
```

⚠️ **Security:** This secret is **required** for production. Never use default values or commit secrets to version control.

**Purpose:**

- Authenticates API requests
- Signs and verifies JWT tokens
- Prevents unauthorized access to operations

---

### DATABASE\_URL

**Type:** String (PostgreSQL connection string, required for API server)
**When to use:** Operation tracking and API server state management

```bash
export DATABASE_URL="postgres://user:pass@localhost:5432/pgsquash_api"
api-server
```

**Format:** `postgres://username:password@hostname:port/database`

**Purpose:**

- Tracks operation status and progress
- Stores API request metadata
- Manages user sessions and state

**Note:** This is separate from `PROD_DB_DSN` (used for CLI validation). The API server requires its own database for tracking operations.

---

## AI Provider Configuration

**Optional:** Enable AI-powered analysis to detect duplicate functions, dead code, and optimization opportunities.

### ANTHROPIC\_API\_KEY

**Type:** String
**When to use:** You want Claude to analyze your migrations for semantic issues

```bash
export ANTHROPIC_API_KEY="sk-ant-api03-..."
pgsquash ai-test  # verify it works
pgsquash squash migrations/*.sql --ai
```

**Get your key:** <https://console.anthropic.com/>

**What it does:**

- Detects semantically identical functions with different names
- Finds unreachable code paths in stored procedures
- Suggests index optimizations based on query patterns
- Identifies auth pattern conflicts (multiple auth providers)

**Example output:**

```
⚠️ AI Analysis:
- Functions calculate_total() and compute_sum() are semantically identical
- Index idx_users_email is redundant with idx_users_email_status
- RLS policy users_select is unreachable (earlier policy matches first)
```

---

### OPENAI\_API\_KEY

**Type:** String
**When to use:** Prefer GPT-4 for analysis, or need Azure OpenAI support

```bash
export OPENAI_API_KEY="sk-..."
pgsquash squash migrations/*.sql --ai
```

**Get your key:** <https://Platform.openai.com/>

Provides similar analysis to Claude - choose based on your preference or existing API access.

---

### AZURE\_OPENAI\_ENDPOINT

**Type:** String (URL)
**When to use:** Enterprise setups with Azure OpenAI deployments

```bash
export AZURE_OPENAI_ENDPOINT="https://your-resource.openai.azure.com/"
export AZURE_OPENAI_API_KEY="your-api-key"
export AZURE_OPENAI_DEPLOYMENT="gpt-4"
pgsquash squash migrations/*.sql --ai
```

**Required:** All three Azure variables must be set together.

---

### AZURE\_OPENAI\_API\_KEY

**Type:** String
**When to use:** Part of Azure OpenAI setup (see AZURE\_OPENAI\_ENDPOINT above)

```bash
export AZURE_OPENAI_API_KEY="your-azure-key"
```

**Note:** Only required if not using Azure AD authentication.

---

### AZURE\_OPENAI\_DEPLOYMENT

**Type:** String
**Description:** Name of your Azure OpenAI deployment.

```bash
export AZURE_OPENAI_DEPLOYMENT="gpt-4"
```

**Default:** Uses the deployment name from your Azure configuration.

---

### AZURE\_OPENAI\_API\_VERSION

**Type:** String
**Description:** Azure OpenAI API version to use.

```bash
export AZURE_OPENAI_API_VERSION="2023-12-01-preview"
```

**Default:** Latest stable version

---

### AZURE\_OPENAI\_USE\_AD

**Type:** Boolean (`true` or `false`)
**Description:** Use Azure Active Directory authentication instead of API key.

```bash
export AZURE_OPENAI_USE_AD=true
export AZURE_OPENAI_ENDPOINT="https://your-resource.openai.azure.com/"
export AZURE_OPENAI_DEPLOYMENT="gpt-4"
```

**Default:** `false`

---

## API Server & GitHub Integration

**When to use:** Setting up automatic PR analysis, webhook automation, or team collaboration features.

See [github-webhooks.md](github-webhooks.md) for complete setup guide.

### PORT

**Type:** Integer
**Default:** `8080`
**When to use:** Running API server on non-standard port (Fly.io uses 8080 by default)

```bash
export PORT=3000
api-server
```

---

### CORS\_ORIGIN

**Type:** String (comma-separated URLs)
**When to use:** Building a web UI that calls the API server

```bash
export CORS_ORIGIN="https://app.example.com,https://staging.example.com"
api-server
```

⚠️ **Security:** Never use `*` in production - restrict to your actual domains.

---

### GITHUB\_TOKEN

**Type:** String
**When to use:** Enabling PR comments, consolidation bots, webhook integration

```bash
export GITHUB_TOKEN="ghp_..."
api-server
```

**Required permissions:** `repo`, `workflow`
**Get your token:** <https://github.com/settings/tokens>

**What it enables:**

- Automatic PR comments with analysis results
- Bot commands (`/pgsquash analyze`, `/pgsquash consolidate`)
- Creating consolidation PRs
- Reading migration files from private repos

---

### GITHUB\_WEBHOOK\_SECRET

**Type:** String
**When to use:** Securing your webhook endpoint (prevents unauthorized requests)

```bash
# Generate a secure secret
openssl rand -hex 32

export GITHUB_WEBHOOK_SECRET="your-webhook-secret"
api-server
```

⚠️ **Security:** Use a strong random secret - this prevents attackers from triggering fake PR analysis.

**Setup:** Set the same secret in:

1. Your API server environment (`GITHUB_WEBHOOK_SECRET`)
2. GitHub webhook settings (Repository → Settings → Webhooks → Secret)

---

### GITHUB\_CLIENT\_ID

**Type:** String
**Description:** GitHub OAuth application client ID.

```bash
export GITHUB_CLIENT_ID="Iv1.abc123..."
export GITHUB_CLIENT_SECRET="secret123..."
export GITHUB_REDIRECT_URL="https://yourapp.com/github/callback"
api-server
```

**Setup:** Create a GitHub OAuth App at <https://github.com/settings/developers>

---

### GITHUB\_CLIENT\_SECRET

**Type:** String
**Description:** GitHub OAuth application client secret.

```bash
export GITHUB_CLIENT_SECRET="secret123..."
```

**Security:** Never expose this value publicly.

---

### GITHUB\_REDIRECT\_URL

**Type:** String (URL)
**Description:** OAuth callback URL after GitHub authentication.

```bash
export GITHUB_REDIRECT_URL="https://yourapp.com/github/callback"
```

**Must Match:** The callback URL configured in your GitHub OAuth app.

---

## Configuration Priority

Environment variables **do not override** configuration file settings for most options. Only specific environment variables are supported:

**CLI Supported Variables:**

- `PGSQUASH_LOG_LEVEL` - Sets logging verbosity
- `PROD_DB_DSN` - Database connection for paranoid mode and backups
- AI provider keys (`ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, etc.)

**API Server Supported Variables:**

- `JWT_SECRET` - Required for authentication
- `DATABASE_URL` - Required for operation tracking
- `PORT`, `CORS_ORIGIN` - Server configuration
- GitHub integration variables (`GITHUB_TOKEN`, etc.)

**Priority Order:**

1. **CLI Flags** (highest priority) - e.g., `--safety conservative`
2. **Environment Variables** - Only for supported variables listed above
3. **Config File** (`pgsquash.config.json`)
4. **Built-in Defaults** (lowest priority)

### Example

```bash
# Config file sets safety_level: "standard"
# Override with CLI flag (NOT environment variable):
pgsquash squash migrations/*.sql --safety conservative
```

**Note:** Unlike some tools, pgsquash does **not** support environment variables like `PGSQUASH_SAFETY_LEVEL` or `PGSQUASH_CONFIG_PATH`. Use CLI flags or the config file instead.

---

## Security Best Practices

### Local Development

Use a `.env` file (add to `.gitignore`):

```bash
# .env
export ANTHROPIC_API_KEY="sk-ant-..."
export OPENAI_API_KEY="sk-..."
export PROD_DB_DSN="postgres://localhost:5432/dev"
```

Load it:

```bash
source .env
pgsquash analyze migrations/*.sql
```

### Production Deployment

1. **Use secrets management:** AWS Secrets Manager, HashiCorp Vault, GitHub Secrets
2. **Rotate keys regularly:** Especially API keys and tokens
3. **Restrict access:** Use principle of least privilege
4. **Monitor usage:** Track API calls and database connections
5. **Never commit secrets:** Use `.gitignore` for `.env` files

### Docker

Pass environment variables securely:

```bash
# Using env file
docker run --env-file .env pgsquash

# Individual variables
docker run -e ANTHROPIC_API_KEY="$ANTHROPIC_API_KEY" pgsquash

# Docker Compose with secrets
docker-compose --env-file .env up
```

---

## Quick Reference

### CLI Environment Variables

| Variable                   | Type    | Default | Used By          |
| -------------------------- | ------- | ------- | ---------------- |
| `PGSQUASH_LOG_LEVEL`       | string  | `info`  | CLI              |
| `PROD_DB_DSN`              | string  | -       | CLI              |
| `ANTHROPIC_API_KEY`        | string  | -       | CLI, AI features |
| `OPENAI_API_KEY`           | string  | -       | CLI, AI features |
| `AZURE_OPENAI_ENDPOINT`    | URL     | -       | CLI, AI features |
| `AZURE_OPENAI_API_KEY`     | string  | -       | CLI, AI features |
| `AZURE_OPENAI_DEPLOYMENT`  | string  | -       | CLI, AI features |
| `AZURE_OPENAI_API_VERSION` | string  | latest  | CLI, AI features |
| `AZURE_OPENAI_USE_AD`      | boolean | `false` | CLI, AI features |

### API Server Environment Variables

| Variable                | Type   | Default | Required |
| ----------------------- | ------ | ------- | -------- |
| `JWT_SECRET`            | string | -       | **Yes**  |
| `DATABASE_URL`          | string | -       | **Yes**  |
| `PORT`                  | int    | `8080`  | No       |
| `CORS_ORIGIN`           | string | -       | No       |
| `GITHUB_TOKEN`          | string | -       | No       |
| `GITHUB_WEBHOOK_SECRET` | string | -       | No       |
| `GITHUB_CLIENT_ID`      | string | -       | No       |
| `GITHUB_CLIENT_SECRET`  | string | -       | No       |
| `GITHUB_REDIRECT_URL`   | URL    | -       | No       |

**Note:** `PGSQUASH_DOCKER_NETWORK` and `PGSQUASH_CONFIG_PATH` are not supported environment variables. Use Docker Compose network configuration and the `--config` CLI flag instead.

---

## See Also

- [Configuration File Reference](configuration.md)
- [AI Features](ai-features.md)
- [API Server Documentation](../cmd/api-server/README.md)
- [GitHub Integration](internal/deployments/github-integration.md)
