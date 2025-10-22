# pgsquash API Server

HTTP API server for pgsquash with GitHub integration support.

## Features

- **REST API** for migration analysis and squashing
- **GitHub Webhooks** for automated PR analysis
- **GitHub OAuth** for user authentication
- **AI-powered analysis** (optional)
- **CORS support** for Platforms

## Quick Start

### 1. Environment Setup

Create `.env` file:

```bash
# Required for GitHub integration
GITHUB_TOKEN=ghp_your_token_here
GITHUB_WEBHOOK_SECRET=your_webhook_secret
GITHUB_CLIENT_ID=your_oauth_client_id
GITHUB_CLIENT_SECRET=your_oauth_client_secret

# Optional: Custom port
API_PORT=8080

# Optional: CORS origins (comma-separated)
CORS_ORIGIN=https://yourapp.com,http://localhost:3000

# Optional: AI providers
ANTHROPIC_API_KEY=sk-ant-...
OPENAI_API_KEY=sk-...
```

### 2. Start API Server

```bash
cd docker/api-server
docker compose up -d
```

### 3. Verify Health

```bash
curl http://localhost:8080/health
```

Response:

```json
{
  "status": "healthy",
  "timestamp": 1234567890,
  "service": "pgsquash-api",
  "version": "0.9.5-beta"
}
```

## API Endpoints

### Health & Info

- `GET /health` - Health check
- `GET /api/info` - Service information

### Migration Operations

- `POST /api/analyze` - Analyze migrations
- `POST /api/squash` - Squash migrations

### GitHub Integration

- `POST /github/webhook` - GitHub webhook handler
- `GET /github/login` - GitHub OAuth login
- `GET /github/callback` - GitHub OAuth callback

## Usage Examples

### Analyze Migrations

```bash
curl -X POST http://localhost:8080/api/analyze \
  -F "safety_level=standard" \
  -F "files=@migrations/001.sql" \
  -F "files=@migrations/002.sql"
```

Response:

```json
{
  "original_count": 156,
  "optimized_count": 45,
  "estimated_time_savings": "~111 statements reduced",
  "safety_level": "standard",
  "warnings": [],
  "recommendations": ["Review consolidation results before applying"],
  "processing_time_ms": 123,
  "file_size_reduction": "71.2%"
}
```

**Note**: `original_count` and `optimized_count` represent the number of SQL statements (not files or lines). The reduction percentage shows how many statements were consolidated.

### Squash Migrations

```bash
curl -X POST http://localhost:8080/api/squash \
  -F "safety_level=conservative" \
  -F "files=@migrations/001.sql" \
  -F "files=@migrations/002.sql" \
  -o consolidated.sql
```

## GitHub Integration

### Setup Webhook

1. Go to your GitHub repository settings
2. Navigate to "Webhooks" → "Add webhook"
3. Set payload URL: `https://your-domain.com/github/webhook`
4. Set content type: `application/json`
5. Set secret: Same as `GITHUB_WEBHOOK_SECRET` in `.env`
6. Select events: `Pull requests`, `Pushes`

### OAuth Flow

1. User visits: `http://localhost:8080/github/login`
2. Redirects to GitHub for authorization
3. GitHub redirects back to: `http://localhost:8080/github/callback?code=...`
4. API server exchanges code for access token
5. Token stored securely (OS keychain + file fallback)

## Configuration

All configuration via environment variables:

| Variable                | Required | Default                                 | Description                             |
| ----------------------- | -------- | --------------------------------------- | --------------------------------------- |
| `GITHUB_TOKEN`          | No       | -                                       | GitHub personal access token            |
| `GITHUB_WEBHOOK_SECRET` | No       | -                                       | Webhook signature verification secret   |
| `GITHUB_CLIENT_ID`      | No       | -                                       | OAuth app client ID                     |
| `GITHUB_CLIENT_SECRET`  | No       | -                                       | OAuth app client secret                 |
| `GITHUB_REDIRECT_URL`   | No       | `http://localhost:8080/github/callback` | OAuth redirect URL                      |
| `CORS_ORIGIN`           | No       | `https://CAPYSQUASH.dev`                | Allowed CORS origins (comma-separated)  |
| `API_PORT`              | No       | `8080`                                  | Server port                             |
| `LOG_LEVEL`             | No       | `info`                                  | Log level (debug, info, warning, error) |
| `LOG_FORMAT`            | No       | `json`                                  | Log format (json, text)                 |
| `ANTHROPIC_API_KEY`     | No       | -                                       | Anthropic Claude API key                |
| `OPENAI_API_KEY`        | No       | -                                       | OpenAI API key                          |
| `AZURE_OPENAI_ENDPOINT` | No       | -                                       | Azure OpenAI endpoint                   |

## Production Deployment

### With SSL/TLS (Recommended)

Use a reverse proxy like Traefik, Nginx, or Caddy:

```yaml
# docker-compose.prod.yml
services:
  api-server:
    # ... same as docker-compose.yml
    environment:
      - GITHUB_REDIRECT_URL=https://api.yourdomain.com/github/callback
      - CORS_ORIGIN=https://yourdomain.com

  traefik:
    image: traefik:v3.0
    command:
      - "--providers.docker=true"
      - "--entrypoints.websecure.address=:443"
      - "--certificatesresolvers.letsencrypt.acme.tlschallenge=true"
      - "--certificatesresolvers.letsencrypt.acme.email=admin@yourdomain.com"
    ports:
      - "443:443"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
```

### Resource Limits

```yaml
services:
  api-server:
    deploy:
      resources:
        limits:
          cpus: '2.0'
          memory: 2G
        reservations:
          cpus: '0.5'
          memory: 512M
```

### Health Checks

API server includes built-in health checks:

```bash
# Check container health
docker inspect --format='{{.State.Health.Status}}' pgsquash-api-server

# View health logs
docker inspect --format='{{range .State.Health.Log}}{{.Output}}{{end}}' pgsquash-api-server
```

## Monitoring

### Logs

```bash
# Follow logs
docker compose logs -f api-server

# Export logs
docker compose logs api-server > api-server.log

# JSON log parsing
docker compose logs api-server | jq '.level' | sort | uniq -c
```

### Metrics

If Prometheus integration is enabled (future):

```bash
curl http://localhost:8080/metrics
```

## Troubleshooting

### Container won't start

```bash
# Check logs
docker compose logs api-server

# Verify environment variables
docker compose config

# Test health endpoint
curl http://localhost:8080/health
```

### GitHub webhook not receiving events

1. Check webhook secret matches `.env`
2. Verify webhook URL is publicly accessible
3. Check GitHub webhook delivery logs
4. Inspect API server logs: `docker compose logs api-server | grep webhook`

### CORS errors

Add your origin to `CORS_ORIGIN`:

```bash
CORS_ORIGIN=https://yourapp.com,http://localhost:3000
```

### Permission denied errors

Ensure files are readable:

```bash
chmod 644 .env
chmod 755 docker/api-server
```

## Development

### Local Development

```bash
# Build and run with live logs
docker compose up --build

# Run in background
docker compose up -d

# Rebuild after code changes
docker compose build api-server
docker compose up -d api-server
```

### Debug Mode

```bash
# Enable debug logging
LOG_LEVEL=debug docker compose up
```

### Testing

```bash
# Test health endpoint
curl http://localhost:8080/health

# Test info endpoint
curl http://localhost:8080/api/info

# Test analyze with sample file
curl -X POST http://localhost:8080/api/analyze \
  -F "safety_level=standard" \
  -F "files=@test.sql"
```

## Security

- **Never commit** `.env` file with secrets
- Use **strong webhook secrets** (32+ characters)
- Enable **HTTPS** in production
- Rotate **GitHub tokens** regularly
- Limit **CORS origins** to trusted domains
- Use **read-only** file mounts where possible

## Further Reading

- [Main Documentation](../../README.md)
- [CLI Reference](../../docs/cli-reference.md)
- [GitHub Integration Guide](../../docs/internal/deployments/github-integration.md)
- [Docker Publishing Guide](../../docs/internal/deployments/DOCKER_PUBLISHING_GUIDE.md)
