# pgsquash Engine API - Deployment Guide

## 🚀 Quick Start

This guide covers deploying the pgsquash Engine API to Fly.io with full integration to the Platform.

---

## 📋 Prerequisites

- [ ] Fly.io account and CLI installed (`fly auth login`)
- [ ] PostgreSQL database accessible (Neon, Supabase, or Fly Postgres)
- [ ] CAPYSQUASH deployed and configured
- [ ] Environment variables ready

---

## 🔐 Environment Variables

### Required Variables

```bash
# Database Connection
DATABASE_URL=postgresql://user:password@host:port/database?sslmode=require

# Security
JWT_SECRET=your-production-jwt-secret-key-must-be-strong

# CORS Configuration
CORS_ORIGIN=https://CAPYSQUASH.dev,https://app.CAPYSQUASH.dev,http://localhost:3000

# Server Configuration
PORT=8080
```

### Optional Variables (GitHub Integration)

```bash
# GitHub OAuth (optional)
GITHUB_CLIENT_ID=your_github_oauth_client_id
GITHUB_CLIENT_SECRET=your_github_oauth_client_secret
GITHUB_REDIRECT_URL=https://api.CAPYSQUASH.dev/github/callback

# GitHub Webhook (optional)
GITHUB_TOKEN=ghp_your_personal_access_token
GITHUB_WEBHOOK_SECRET=your_webhook_secret
```

---

## 🗄️ Database Setup

### 1. Push Schema Migrations (Platform Database)

```bash
# Navigate to Platform directory
cd /Users/dominikospritis/DevFolder/pgsquash/capysquash-platform

# Push latest migrations (includes created_by index)
pnpm db:push
```

### 2. Verify Schema

The engine expects these tables in your database:

- `migration_runs` - Stores operation history
- `users` - For user authentication

Required indexes on `migration_runs`:

- `migration_run_created_by_idx` - Performance for user queries
- `migration_run_project_idx` - Performance for project queries
- `migration_run_status_idx` - Performance for status filtering

---

## 🛠️ Local Testing

### 1. Build and Test Locally

```bash
cd /Users/dominikospritis/DevFolder/pgsquash/pgsquash-engine

# Compile the binary
go build -o pgsquash-api cmd/api-server/main.go

# Create local .env file
cat > .env << 'EOF'
DATABASE_URL=postgresql://localhost:5432/CAPYSQUASH?sslmode=disable
JWT_SECRET=local-dev-secret-key-change-in-production
CORS_ORIGIN=http://localhost:3000
PORT=8080
EOF

# Run locally
export $(cat .env | xargs) && ./pgsquash-api
```

### 2. Test Endpoints

```bash
# Health check
curl http://localhost:8080/health

# Test with JWT (get token from your Platform's Clerk)
export TOKEN="your-jwt-token-from-clerk"

# Test rules endpoint
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/rules
```

---

## ☁️ Fly.io Deployment

### 1. Create Fly App

```bash
cd /Users/dominikospritis/DevFolder/pgsquash/pgsquash-engine

# Launch app (interactive)
fly launch --name pgsquash-engine --region iad
```

Answer prompts:

- Would you like to copy existing configuration? **No**
- Would you like to set up a Postgresql database? **No** (using external DB)
- Would you like to deploy now? **No** (configure secrets first)

### 2. Configure Secrets

```bash
# Set required environment variables
fly secrets set \
  DATABASE_URL="postgresql://user:password@host:port/database?sslmode=require" \
  JWT_SECRET="$(openssl rand -base64 32)" \
  CORS_ORIGIN="https://CAPYSQUASH.dev,https://app.CAPYSQUASH.dev"

# Optional: GitHub integration
fly secrets set \
  GITHUB_CLIENT_ID="your_client_id" \
  GITHUB_CLIENT_SECRET="your_client_secret" \
  GITHUB_REDIRECT_URL="https://api.CAPYSQUASH.dev/github/callback" \
  GITHUB_TOKEN="ghp_your_token" \
  GITHUB_WEBHOOK_SECRET="your_webhook_secret"
```

### 3. Create fly.toml Configuration

```toml
app = "pgsquash-engine"
primary_region = "iad"

[build]
  [build.args]
    GO_VERSION = "1.22"

[env]
  PORT = "8080"

[http_service]
  internal_port = 8080
  force_https = true
  auto_stop_machines = false
  auto_start_machines = true
  min_machines_running = 1
  processes = ["app"]

  [[http_service.checks]]
    interval = "15s"
    timeout = "10s"
    grace_period = "5s"
    method = "GET"
    path = "/health"

[[vm]]
  cpu_kind = "shared"
  cpus = 1
  memory_mb = 512
```

### 4. Create Dockerfile

```dockerfile
# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o pgsquash-api cmd/api-server/main.go

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/pgsquash-api .

# Expose port
EXPOSE 8080

# Run
CMD ["./pgsquash-api"]
```

### 5. Deploy

```bash
# Deploy to Fly.io
fly deploy

# Monitor deployment
fly logs

# Verify deployment
curl https://pgsquash-engine.fly.dev/health
```

---

## 🔗 Platform Integration

### 1. Update Platform Environment Variables

```bash
# In your Platform .env.local or .env.production
NEXT_PUBLIC_API_URL=https://pgsquash-engine.fly.dev
```

### 2. Verify JWT Token Format

The engine expects JWT tokens from Clerk with these claims:

```json
{
  "user_id": "user_abc123",
  "email": "user@example.com",
  "exp": 1234567890
}
```

Clerk tokens are automatically in this format.

### 3. Test Integration

```typescript
// In your Platform, test API connectivity:
import { getRules } from '@/src/lib/api/engine-client';

// This should work if everything is configured correctly
const rulesData = await getRules();
console.log('Rules:', rulesData);
```

---

## ✅ Post-Deployment Checklist

### API Health

- [ ] Health endpoint responds: `curl https://pgsquash-engine.fly.dev/health`
- [ ] Returns `{"status":"healthy","timestamp":...}`

### Authentication

- [ ] CAPYSQUASH can authenticate with engine
- [ ] JWT tokens are accepted
- [ ] Unauthorized requests return 401

### CORS

- [ ] CAPYSQUASH domain is in CORS\_ORIGIN
- [ ] Browser allows requests from Platform
- [ ] No CORS errors in browser console

### Database

- [ ] migration\_runs table exists
- [ ] Indexes are created
- [ ] Engine can write to database

### Functionality

- [ ] Rules endpoint returns data: `GET /api/rules`
- [ ] Plugins endpoint works: `GET /api/plugins`
- [ ] Config endpoint works: `GET /api/config`
- [ ] SSE progress works: `GET /api/operations/{id}/progress?token={jwt}`
- [ ] Operations can be created and tracked

### Performance

- [ ] API responds within 200ms for read operations
- [ ] File uploads handle up to 10MB
- [ ] Rate limiting allows 10 req/sec per user
- [ ] SSE connections stream properly

---

## 🔧 Troubleshooting

### Issue: "Unauthorized" errors

**Solution:**

1. Check JWT\_SECRET matches between Platform and engine
2. Verify Clerk token is being passed correctly
3. Check CORS\_ORIGIN includes your Platform domain

### Issue: "Connection refused" from Platform

**Solution:**

1. Verify NEXT\_PUBLIC\_API\_URL is set correctly
2. Check Fly app is running: `fly status`
3. Test health endpoint directly

### Issue: SSE not streaming

**Solution:**

1. Verify token is passed as query parameter
2. Check operation exists in database
3. Monitor with: `fly logs --app pgsquash-engine`

### Issue: Database connection errors

**Solution:**

1. Verify DATABASE\_URL is correct
2. Check database allows connections from Fly.io IPs
3. Ensure connection string includes `sslmode=require`

### Issue: Rate limiting too aggressive

**Solution:**
Adjust rate limiter in `cmd/api-server/main.go:96`:

```go
rateLimiter := middleware.NewRateLimiter(20, 40) // 20 req/s, burst 40
```

---

## 📊 Monitoring

### View Logs

```bash
# Real-time logs
fly logs

# Filter by level
fly logs --level error

# Follow specific instance
fly logs -i <instance-id>
```

### Check Metrics

```bash
# App status
fly status

# VM metrics
fly dashboard
```

### Health Checks

```bash
# Manual health check
curl https://pgsquash-engine.fly.dev/health

# Automated monitoring
# Set up UptimeRobot or similar to ping /health every 5 minutes
```

---

## 🔄 Updates and Rollbacks

### Deploy New Version

```bash
# Build and deploy
fly deploy

# Force rebuild
fly deploy --build-only
```

### Rollback

```bash
# List releases
fly releases

# Rollback to previous version
fly releases rollback <version>
```

---

## 🔒 Security Checklist

- [ ] JWT\_SECRET is strong (32+ characters, random)
- [ ] DATABASE\_URL uses SSL (`sslmode=require`)
- [ ] CORS\_ORIGIN only includes trusted domains
- [ ] Rate limiting is enabled (10 req/s per user)
- [ ] File upload limits enforced (10MB, 100 files, .sql only)
- [ ] No secrets in git repository
- [ ] Fly secrets are set (not in fly.toml)

---

## 📈 Scaling

### Increase Resources

```bash
# Scale memory
fly scale memory 1024

# Scale CPU
fly scale vm shared-cpu-2x

# Add more instances
fly scale count 2
```

### Database Connection Pooling

If you see "too many connections" errors:

1. Use PgBouncer or similar
2. Adjust pool size in `operations/tracker.go:56`

---

## 🎯 Production Checklist

Before going live:

- [ ] All environment variables set on Fly
- [ ] Database migrations applied
- [ ] SSL/TLS enabled (automatic with Fly)
- [ ] Health checks passing
- [ ] CAPYSQUASH integration tested end-to-end
- [ ] CORS configured correctly
- [ ] Rate limiting tested
- [ ] Error handling verified
- [ ] Logs are being collected
- [ ] Monitoring set up
- [ ] Backup strategy in place

---

## 📞 Support

If you encounter issues:

1. Check logs: `fly logs`
2. Review AUDIT\_REPORT.md for known issues
3. Test locally first
4. Verify environment variables
5. Check database connectivity

---

## 🚀 Deploy Command Summary

```bash
# Complete deployment in one go:
cd /Users/dominikospritis/DevFolder/pgsquash/pgsquash-engine

# 1. Test locally
go build -o pgsquash-api cmd/api-server/main.go

# 2. Create Fly app
fly launch --name pgsquash-engine --region iad --no-deploy

# 3. Set secrets
fly secrets set DATABASE_URL="..." JWT_SECRET="..." CORS_ORIGIN="..."

# 4. Deploy
fly deploy

# 5. Verify
curl https://pgsquash-engine.fly.dev/health

# 6. Update Platform
# Set NEXT_PUBLIC_API_URL=https://pgsquash-engine.fly.dev

# Done! 🎉
```

---

## 📝 Notes

- **Binary Size:** Compiled Go binary is \~43MB
- **Memory Usage:** Typically 100-200MB under normal load
- **Cold Start:** <1 second with Fly.io
- **Request Latency:** Typically <100ms for read operations
- **Database Connections:** Pooled (max 20, idle 10)
- **Rate Limit:** 10 requests/second per user, burst 20

---

## ✅ Success Criteria

Your deployment is successful when:

1. ✅ Health endpoint returns 200 OK
2. ✅ CAPYSQUASH can fetch rules, plugins, config
3. ✅ File upload and analysis works
4. ✅ SSE progress streaming works
5. ✅ Operations are tracked in database
6. ✅ No CORS errors in browser
7. ✅ Authentication works seamlessly
8. ✅ All 22 endpoints respond correctly

---

**Ready to deploy? Follow the deploy command summary above!** 🚀
