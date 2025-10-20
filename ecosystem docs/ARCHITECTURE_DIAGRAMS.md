# PGSQUASH Architecture Diagrams

**Visual representations of system architecture, data flow, and component relationships**

Last Updated: October 20, 2025

---

## Table of Contents

1. [System Architecture](#system-architecture)
2. [Component Relationships](#component-relationships)
3. [Data Flow Patterns](#data-flow-patterns)
4. [Deployment Topologies](#deployment-topologies)
5. [Authentication & Authorization](#authentication--authorization)
6. [Integration Architecture](#integration-architecture)
7. [Database Schema](#database-schema)

---

## System Architecture

### High-Level System Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          PGSQUASH ECOSYSTEM                                  │
│                    "Autopilot for Postgres Migrations"                       │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│                            USER INTERFACES                                   │
├──────────────┬──────────────┬──────────────┬──────────────┬────────────────┤
│              │              │              │              │                │
│   CLI Tool   │   Web UI     │  GitHub Bot  │  VS Code Ext │  Documentation │
│   (Local)    │  (Browser)   │   (PR)       │  (Editor)    │   (Web)        │
│              │              │              │              │                │
└──────┬───────┴──────┬───────┴──────┬───────┴──────┬───────┴────────────────┘
       │              │              │              │
       │              │              │              │
       ▼              ▼              ▼              │
┌────────────────────────────────────────────────────────────────────────────┐
│                         APPLICATION LAYER                                   │
├─────────────────────────────────┬──────────────────────────────────────────┤
│                                 │                                          │
│   pgsquash-engine (Go 1.22)    │    capysquash-platform (Next.js 15)      │
│   Port: 8080                    │    Port: 3000                            │
│   ─────────────────────────     │    ──────────────────────────────        │
│                                 │                                          │
│   ┌─────────────────────────┐  │    ┌──────────────────────────────┐     │
│   │   CLI Interface         │  │    │   Web Application            │     │
│   │   • analyze             │  │    │   • Dashboard                │     │
│   │   • squash              │  │    │   • Projects                 │     │
│   │   • validate            │  │    │   • Settings                 │     │
│   │   • tui (interactive)   │  │    │   • Analytics                │     │
│   └─────────────────────────┘  │    └──────────────────────────────┘     │
│                                 │                                          │
│   ┌─────────────────────────┐  │    ┌──────────────────────────────┐     │
│   │   HTTP API Server       │◄─┼────┤   API Routes (56 endpoints) │     │
│   │   • POST /analyze       │  │    │   • /api/engine/*            │     │
│   │   • POST /squash        │  │    │   • /api/organizations/*     │     │
│   │   • POST /validate      │  │    │   • /api/projects/*          │     │
│   │   • POST /github/webhook│  │    │   • /api/stripe/*            │     │
│   │   • GET  /health        │  │    │   • /api/admin/*             │     │
│   └─────────────────────────┘  │    └──────────────────────────────┘     │
│                                 │                                          │
│   ┌─────────────────────────┐  │    ┌──────────────────────────────┐     │
│   │   Core Modules          │  │    │   Business Logic             │     │
│   │   ─────────────         │  │    │   ───────────────            │     │
│   │   • Parser (pg_query)   │  │    │   • Authentication (Clerk)   │     │
│   │   • Tracker             │  │    │   • Authorization (RBAC)     │     │
│   │   • Squasher            │  │    │   • Billing (Stripe)         │     │
│   │   • Validator (Docker)  │  │    │   • Team Management          │     │
│   │   • AI Analyzer         │  │    │   • Usage Tracking           │     │
│   │   • GitHub Handler      │  │    │   • Notifications            │     │
│   └─────────────────────────┘  │    └──────────────────────────────┘     │
│                                 │                                          │
└─────────────────────────────────┴──────────────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                      DATA & INFRASTRUCTURE LAYER                             │
├────────────────┬────────────────┬────────────────┬────────────────┬─────────┤
│                │                │                │                │         │
│  PostgreSQL    │   Docker       │   Redis        │   File Storage │  Logs   │
│  (Neon)        │   Runtime      │   (Upstash)    │   (Temp)       │         │
│                │                │                │                │         │
│  • Users       │   • Validation │   • Cache      │   • Migrations │  • API  │
│  • Orgs        │   • Schema     │   • Sessions   │   • Results    │  • App  │
│  • Projects    │     Comparison │   • Rate       │   • Artifacts  │  • Err  │
│  • Runs        │   • Migration  │     Limiting   │                │         │
│  • Analytics   │     Replay     │   • Jobs       │                │         │
│                │                │                │                │         │
└────────────────┴────────────────┴────────────────┴────────────────┴─────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                         EXTERNAL SERVICES                                    │
├─────────────┬─────────────┬─────────────┬─────────────┬────────────────────┤
│             │             │             │             │                    │
│   Clerk     │   Stripe    │   GitHub    │   Vercel    │   Monitoring       │
│   (Auth)    │   (Pay)     │   (Code)    │   (Host)    │   (Observability)  │
│             │             │             │             │                    │
│  • Sign In  │  • Checkout │  • Webhooks │  • CDN      │  • Sentry (Errors) │
│  • Sign Up  │  • Portal   │  • OAuth    │  • Edge     │  • Analytics       │
│  • Orgs     │  • Webhooks │  • API      │  • Deploy   │  • DataDog (APM)   │
│  • Sessions │  • Billing  │  • PR Bot   │             │                    │
│             │             │             │             │                    │
└─────────────┴─────────────┴─────────────┴─────────────┴────────────────────┘
```

---

## Component Relationships

### Dependency Graph

```
┌─────────────────────────────────────────────────────────────┐
│              COMPONENT DEPENDENCY GRAPH                      │
└─────────────────────────────────────────────────────────────┘

     Users & External Services
            │
     ┌──────┴──────┐
     │             │
     ▼             ▼
┌─────────┐   ┌──────────┐
│  Docs   │   │ Platform │ ◄── Primary Entry Points
│ (Info)  │   │  (Web)   │
└─────────┘   └────┬─────┘
                   │
                   │ HTTP API Calls
                   ▼
            ┌─────────────┐
            │   Engine    │ ◄── Core Processing
            │    (API)    │
            └──────┬──────┘
                   │
     ┌─────────────┼─────────────┐
     │             │             │
     ▼             ▼             ▼
┌─────────┐   ┌─────────┐   ┌─────────┐
│ Parser  │   │ Squasher│   │Validator│ ◄── Engine Modules
│(pg_query│   │ (Logic) │   │ (Docker)│
└─────────┘   └─────────┘   └─────────┘
                   │
                   ▼
            ┌──────────────┐
            │   Storage    │ ◄── Data Persistence
            │ (PostgreSQL) │
            └──────────────┘


Communication Patterns:
  ────▶   Direct dependency (synchronous)
  ─ ─ ▶   Indirect/async dependency
  ◄───▶   Bidirectional communication
```

### Service Communication Matrix

```
┌────────────────────────────────────────────────────────────────┐
│           WHO TALKS TO WHO (AND HOW)                           │
└────────────────────────────────────────────────────────────────┘

Platform → Engine:
  Protocol: HTTP REST
  Auth:     X-API-Secret header
  Use:      Migration analysis, consolidation
  Endpoint: ${GO_ENGINE_URL}/analyze, /squash

Engine → Platform:
  Protocol: HTTP Callbacks
  Auth:     HMAC signatures
  Use:      Long-running job completion
  Endpoint: ${PLATFORM_URL}/api/callbacks/*

GitHub → Engine:
  Protocol: Webhooks (HTTP POST)
  Auth:     X-Hub-Signature-256
  Use:      Push events, PR events
  Endpoint: ${ENGINE_URL}/github/webhook

GitHub → Platform:
  Protocol: Webhooks (HTTP POST)
  Auth:     X-Hub-Signature-256
  Use:      Installation, repository events
  Endpoint: ${PLATFORM_URL}/api/github/webhook

Platform → Clerk:
  Protocol: HTTP REST + JWT
  Auth:     API Key + JWT verification
  Use:      Authentication, user management
  SDK:      @clerk/nextjs

Clerk → Platform:
  Protocol: Webhooks (Svix)
  Auth:     Webhook signature
  Use:      User lifecycle events
  Endpoint: ${PLATFORM_URL}/api/webhooks/clerk

Platform → Stripe:
  Protocol: HTTP REST
  Auth:     API Key
  Use:      Checkout, subscriptions
  SDK:      stripe (Node.js)

Stripe → Platform:
  Protocol: Webhooks
  Auth:     Webhook signature
  Use:      Payment events, subscription updates
  Endpoint: ${PLATFORM_URL}/api/stripe/webhook

Platform → Neon:
  Protocol: PostgreSQL Wire Protocol
  Auth:     Connection string
  Use:      Database queries
  Driver:   @neondatabase/serverless

Platform → GitHub API:
  Protocol: HTTP REST
  Auth:     GitHub App credentials
  Use:      Repository access, PR creation
  SDK:      @octokit/rest
```

---

## Data Flow Patterns

### Pattern 1: CLI Direct Usage (Offline)

```
┌────────────────────────────────────────────────────────────┐
│              CLI DIRECT FLOW (No Network)                  │
└────────────────────────────────────────────────────────────┘

Developer
    │
    ▼
$ pgsquash analyze migrations/
    │
    ├─ 1. Read migration files from disk
    │     (*.sql in specified directory)
    │
    ├─ 2. Parse SQL using pg_query_go
    │     (PostgreSQL's actual parser)
    │
    ├─ 3. Build dependency graph
    │     (Track CREATE, ALTER, DROP, etc.)
    │
    ├─ 4. Analyze patterns
    │     • Detect redundancies
    │     • Find optimization opportunities
    │     • Calculate metrics
    │
    ├─ 5. Generate report
    │     • File count reduction estimate
    │     • Complexity score
    │     • Recommendations
    │
    └─▶ Output to terminal
          ✅ No network calls
          ✅ No authentication
          ✅ Completely offline
```

### Pattern 2: Web UI Analysis (Full Stack)

```
┌────────────────────────────────────────────────────────────┐
│           WEB UI ANALYSIS FLOW (Full Stack)               │
└────────────────────────────────────────────────────────────┘

User Browser
    │
    ├─ 1. Upload migrations via web form
    │     Component: <FileUpload />
    │
    ▼
Next.js Frontend (Server Component)
    │
    ├─ 2. Validate user authentication
    │     Middleware: auth() from @clerk/nextjs
    │
    ├─ 3. Check organization permissions
    │     Function: checkPermission(userId, orgId, 'projects:write')
    │
    ▼
Next.js API Route (/api/engine/analyze)
    │
    ├─ 4. Prepare request payload
    │     { migrations: string[], safetyLevel: string, config?: {} }
    │
    ├─ 5. Forward to Go engine
    │     fetch(`${GO_ENGINE_URL}/analyze`, {
    │       headers: { 'X-API-Secret': SECRET }
    │     })
    │
    ▼
Go API Server
    │
    ├─ 6. Verify API secret
    │     Middleware: apiKeyAuth()
    │
    ├─ 7. Parse migrations
    │     Module: internal/parser
    │
    ├─ 8. Analyze dependencies
    │     Module: internal/tracking
    │
    ├─ 9. Generate consolidation plan
    │     Module: internal/squasher
    │
    ├─ 10. Return analysis results
    │      { summary, dependencies, recommendations }
    │
    ▼
Next.js API Route (response handler)
    │
    ├─ 11. Save results to database
    │      await db.insert(analysisRuns).values(...)
    │
    ├─ 12. Track usage for billing
    │      await incrementUsage(orgId, 'analyses', 1)
    │
    ├─ 13. Create activity log entry
    │      await logActivity(userId, 'analysis_completed', projectId)
    │
    ▼
Next.js Frontend (Dashboard)
    │
    └─ 14. Display results
          • Metrics cards
          • Charts and graphs
          • Downloadable report
```

### Pattern 3: GitHub Webhook (Automated)

```
┌────────────────────────────────────────────────────────────┐
│         GITHUB WEBHOOK FLOW (Automation)                   │
└────────────────────────────────────────────────────────────┘

GitHub Event (push to main)
    │
    ├─ Trigger: User pushes commit with migration changes
    │
    ▼
GitHub Webhook Delivery
    │
    ├─ POST /github/webhook
    │   Headers:
    │     X-GitHub-Event: push
    │     X-Hub-Signature-256: sha256=...
    │   Body: {
    │     repository: { ... },
    │     commits: [ ... ],
    │     pusher: { ... }
    │   }
    │
    ▼
Go API Server (/github/webhook)
    │
    ├─ 1. Verify webhook signature
    │     crypto.HmacSHA256(body, WEBHOOK_SECRET)
    │
    ├─ 2. Parse event payload
    │     Determine: repo, branch, commits
    │
    ├─ 3. Check if migrations changed
    │     Filter commits for migration file paths
    │
    ├─ 4. Clone repository
    │     git clone --depth 1 --branch main <repo>
    │
    ├─ 5. Analyze migrations
    │     Run analysis on migrations/ directory
    │
    ├─ 6. Determine if action needed
    │     if filesReduction > threshold (e.g., 15):
    │       proceed to create PR
    │
    ▼
GitHub API (Create PR)
    │
    ├─ 7. Create new branch
    │     git checkout -b pgsquash-cleanup-<timestamp>
    │
    ├─ 8. Apply squashed migrations
    │     Replace migrations/ with consolidated files
    │
    ├─ 9. Commit changes
    │     git commit -m "chore: consolidate migrations"
    │
    ├─ 10. Push branch
    │      git push origin pgsquash-cleanup-<timestamp>
    │
    ├─ 11. Create pull request
    │      POST /repos/:owner/:repo/pulls
    │      {
    │        title: "🧹 Consolidate migrations",
    │        body: "## pgsquash Analysis\n...",
    │        head: "pgsquash-cleanup-<timestamp>",
    │        base: "main"
    │      }
    │
    ├─ 12. Add PR comment with metrics
    │      POST /repos/:owner/:repo/issues/:number/comments
    │      {
    │        body: "### 📊 Results\n| Metric | Before | After |..."
    │      }
    │
    ▼
Platform API (Log activity)
    │
    ├─ 13. Record PR creation
    │      await db.insert(githubWebhookEvents).values(...)
    │
    ├─ 14. Update project stats
    │      await db.update(projects).set({ lastPrAt: now() })
    │
    ├─ 15. Notify team (if configured)
    │      • Send Slack message
    │      • Send email notification
    │
    ▼
GitHub PR (Visible to team)
    │
    └─ Team reviews and merges
       ✅ Automatic analysis
       ✅ Zero manual intervention
       ✅ Full audit trail
```

### Pattern 4: Scheduled Automation (Cron)

```
┌────────────────────────────────────────────────────────────┐
│         SCHEDULED AUTOMATION FLOW (Cron Job)               │
└────────────────────────────────────────────────────────────┘

Vercel Cron (configured in vercel.json)
    │
    ├─ Schedule: Every Monday at 9am UTC
    │   {
    │     "crons": [{
    │       "path": "/api/cron/cleanup",
    │       "schedule": "0 9 * * 1"
    │     }]
    │   }
    │
    ▼
API Route (/api/cron/cleanup)
    │
    ├─ 1. Verify cron secret
    │     if (req.headers['authorization'] !== CRON_SECRET) reject
    │
    ├─ 2. Query organizations with automation enabled
    │     SELECT * FROM organizations
    │     WHERE settings->>'autoCleanup' = 'true'
    │     AND plan IN ('professional', 'agency', 'enterprise')
    │
    ├─ 3. For each organization:
    │     │
    │     ├─ 3a. Get active projects
    │     │     SELECT * FROM projects
    │     │     WHERE org_id = $1
    │     │     AND archived_at IS NULL
    │     │
    │     ├─ 3b. Filter projects needing cleanup
    │     │     • Last cleanup > 7 days ago
    │     │     • Migration count > threshold
    │     │
    │     ├─ 3c. For each project:
    │     │     │
    │     │     ├─ i. Fetch repository via GitHub API
    │     │     │    GET /repos/:owner/:repo
    │     │     │
    │     │     ├─ ii. Clone and analyze migrations
    │     │     │     POST ${GO_ENGINE_URL}/analyze
    │     │     │
    │     │     ├─ iii. Check if cleanup beneficial
    │     │     │      if (reduction < 20%) skip
    │     │     │
    │     │     ├─ iv. Generate squashed migrations
    │     │     │     POST ${GO_ENGINE_URL}/squash
    │     │     │
    │     │     ├─ v. Create cleanup PR (see Pattern 3)
    │     │     │
    │     │     └─ vi. Notify team
    │     │           • Email: "Weekly cleanup PR ready"
    │     │           • Slack: Post to #engineering channel
    │     │
    │     └─ 4. Log cron execution
    │           await db.insert(cronLogs).values({
    │             jobType: 'weekly_cleanup',
    │             projectsProcessed: count,
    │             prsCreated: prCount,
    │             executedAt: now()
    │           })
    │
    ▼
Response
    │
    └─ Return summary
       {
         success: true,
         orgsProcessed: 15,
         projectsProcessed: 47,
         prsCreated: 12
       }
```

---

## Deployment Topologies

### Topology 1: Production SaaS

```
┌────────────────────────────────────────────────────────────────────┐
│                  PRODUCTION SAAS TOPOLOGY                           │
│                  (Current: capysquash.dev)                          │
└────────────────────────────────────────────────────────────────────┘

End Users (Global)
    │
    ├───── DNS: capysquash.dev ─────┐
    │                                │
    ▼                                ▼
┌──────────────────┐        ┌───────────────────┐
│  Vercel Edge CDN │        │    Fly.io Edge    │
│  (Global PoPs)   │        │   (Multi-region)  │
└────────┬─────────┘        └─────────┬─────────┘
         │                            │
         ▼                            ▼
┌──────────────────┐        ┌───────────────────┐
│  Next.js Web App │        │   Go API Server   │
│  (Vercel)        │◄──────▶│   (Fly.io)        │
│                  │  HTTPS │                   │
│  • SSR Pages     │        │  • HTTP API       │
│  • API Routes    │        │  • Webhooks       │
│  • Static Assets │        │  • Validation     │
└────────┬─────────┘        └─────────┬─────────┘
         │                            │
         │                            │
    ┌────┴──────┬──────────────┬──────┴─────┐
    │           │              │            │
    ▼           ▼              ▼            ▼
┌────────┐ ┌────────┐   ┌──────────┐  ┌─────────┐
│ Clerk  │ │ Stripe │   │   Neon   │  │ Upstash │
│ (Auth) │ │ (Pay)  │   │ (PG 17)  │  │ (Redis) │
│        │ │        │   │          │  │         │
│ US-E   │ │ Global │   │  US-E    │  │ Global  │
└────────┘ └────────┘   └──────────┘  └─────────┘

GitHub (Global)
    │
    ├─ Webhooks ──▶ Fly.io API
    ├─ OAuth ─────▶ Vercel Platform
    └─ API ◄──────▶ Both (bidirectional)

Monitoring:
  • Sentry (Error tracking)
  • Vercel Analytics (Web vitals)
  • Fly.io Metrics (API performance)
  • DataDog (Optional APM)

Deployment Flow:
  1. Git push to main
  2. Vercel auto-deploys web app
  3. Fly auto-deploys API (or manual: fly deploy)
  4. Database migrations via Drizzle (manual or CI)

Cost Estimate:
  ~$105-380/month based on usage
```

### Topology 2: Enterprise On-Premise

```
┌────────────────────────────────────────────────────────────────────┐
│            ENTERPRISE ON-PREMISE TOPOLOGY                           │
│            (Self-hosted in customer infrastructure)                 │
└────────────────────────────────────────────────────────────────────┘

Customer Network (VPC or On-Premise)
    │
    ▼
┌──────────────────────────────────────────────────────────────┐
│                    Nginx Reverse Proxy                        │
│                    (Port 80/443)                              │
│                    SSL Termination                            │
└─────────────────────┬───────────────────────────────────┬────┘
                      │                                    │
         ┌────────────┴───────────┐                       │
         │                        │                        │
         ▼                        ▼                        │
    ┌─────────────┐        ┌──────────────┐              │
    │  Web App    │        │  API Server  │              │
    │  (Next.js)  │◄──────▶│  (Go)        │              │
    │  Port 3000  │        │  Port 8080   │              │
    └──────┬──────┘        └──────┬───────┘              │
           │                      │                        │
           │     ┌────────────────┴──────┐                │
           │     │                       │                │
           ▼     ▼                       ▼                │
       ┌──────────────┐           ┌──────────────┐       │
       │ PostgreSQL   │           │    Redis     │       │
       │    (PG 17)   │           │  (Cache)     │       │
       │  Port 5432   │           │  Port 6379   │       │
       └──────────────┘           └──────────────┘       │
                                                          │
All in Docker Compose:                                    │
  docker compose --profile production up -d               │
                                                          │
Optional External Services (if not airgapped):            │
  ┌────────────────────────────────────────────┐         │
  │  • Clerk (or replaced with SAML/LDAP)     │◄────────┘
  │  • Stripe (or internal billing)           │
  │  • GitHub Enterprise (on-premise)         │
  └────────────────────────────────────────────┘

System Requirements:
  • CPU: 4+ cores (8+ recommended)
  • RAM: 8GB minimum (16GB+ recommended)
  • Disk: 50GB+ SSD
  • OS: Ubuntu 22.04, RHEL 8+, or similar
  • Docker 20.10+ and Docker Compose 2.0+

Network:
  • Internal access only (no public internet required)
  • Outbound HTTPS for license check (optional)
  • SSL certificate for custom domain

Customization:
  • White-label branding
  • Custom domain (e.g., migrations.company.com)
  • SSO/SAML integration
  • Custom retention policies
  • Airgapped mode

Support SLA:
  • Dedicated Slack channel
  • Video call support
  • Quarterly business reviews
  • 4-hour emergency response time
```

---

## Authentication & Authorization

### Auth Flow Diagram

```
┌────────────────────────────────────────────────────────────┐
│              AUTHENTICATION FLOW (Clerk)                   │
└────────────────────────────────────────────────────────────┘

User Browser
    │
    ├─ 1. Navigate to /dashboard
    │
    ▼
Next.js Middleware
    │
    ├─ 2. Check for Clerk session cookie
    │     const { userId } = auth()
    │
    ├─ 3a. If NO session:
    │     │
    │     └─▶ Redirect to /sign-in
    │          │
    │          ▼
    │      Clerk Sign In Page
    │          │
    │          ├─ Email/password
    │          ├─ OAuth (Google, GitHub, etc.)
    │          ├─ Magic link
    │          │
    │          ▼
    │      Clerk Verification
    │          │
    │          ├─ Verify credentials
    │          ├─ Generate JWT
    │          ├─ Set session cookie
    │          │
    │          └─▶ Redirect to /dashboard
    │
    └─ 3b. If session exists:
         │
         ├─ 4. Verify JWT signature
         │     Clerk public key validation
         │
         ├─ 5. Extract user info
         │     userId, email, orgId, etc.
         │
         ├─ 6. Attach to request context
         │     req.auth = { userId, ... }
         │
         └─▶ Proceed to route handler


API Route Request Flow:
    │
    ├─ 1. Extract token from header
    │     Authorization: Bearer <token>
    │
    ├─ 2. Verify with Clerk
    │     const { userId } = auth()
    │
    ├─ 3. Check platform role
    │     SELECT role FROM users WHERE id = userId
    │
    ├─ 4. Check organization role
    │     SELECT role FROM organization_memberships
    │     WHERE user_id = userId AND org_id = orgId
    │
    ├─ 5. Check specific permission
    │     SELECT permission FROM user_permissions
    │     WHERE user_id = userId
    │     AND permission = 'projects:write'
    │
    └─▶ Allow or deny request
```

### RBAC Permission Matrix

```
┌───────────────────────────────────────────────────────────────────┐
│              RBAC PERMISSION MATRIX                                │
└───────────────────────────────────────────────────────────────────┘

┌─────────────────┬────────────────┬─────────────────────────────────┐
│ Platform Role   │ Who Gets It    │ Permissions                     │
├─────────────────┼────────────────┼─────────────────────────────────┤
│ ADMIN           │ Founders,      │ • Manage all users              │
│                 │ Core Team      │ • View all organizations        │
│                 │                │ • Access admin dashboard        │
│                 │                │ • Modify subscription plans     │
│                 │                │ • Override limits               │
├─────────────────┼────────────────┼─────────────────────────────────┤
│ USER            │ Everyone       │ • Create organizations          │
│ (default)       │ else           │ • Join organizations            │
│                 │                │ • Manage own profile            │
└─────────────────┴────────────────┴─────────────────────────────────┘

┌─────────────────┬────────────────┬─────────────────────────────────┐
│Organization Role│ Assignment     │ Organization Permissions        │
├─────────────────┼────────────────┼─────────────────────────────────┤
│ OWNER           │ Org creator,   │ • Full organization control     │
│                 │ or transferred │ • Manage billing                │
│                 │                │ • Delete organization           │
│                 │                │ • Manage all members & roles    │
│                 │                │ • Change organization settings  │
│                 │                │ • View usage & analytics        │
├─────────────────┼────────────────┼─────────────────────────────────┤
│ ADMIN           │ Promoted by    │ • Manage projects               │
│                 │ OWNER          │ • Manage members (except OWNER) │
│                 │                │ • Manage API keys               │
│                 │                │ • View analytics                │
│                 │                │ • Configure integrations        │
├─────────────────┼────────────────┼─────────────────────────────────┤
│ MEMBER          │ Invited or     │ • Create projects               │
│                 │ default        │ • Analyze migrations            │
│                 │                │ • View own projects             │
│                 │                │ • Comment on projects           │
├─────────────────┼────────────────┼─────────────────────────────────┤
│ VIEWER          │ Restricted     │ • View projects (read-only)     │
│                 │ access         │ • View analytics (read-only)    │
│                 │                │ • No write permissions          │
└─────────────────┴────────────────┴─────────────────────────────────┘

Granular Permissions (per-resource):
  ┌─────────────────────┬──────────────────────────────────┐
  │ Permission          │ What It Allows                   │
  ├─────────────────────┼──────────────────────────────────┤
  │ projects:read       │ View project details             │
  │ projects:write      │ Create/edit projects             │
  │ projects:delete     │ Delete projects                  │
  │ runs:execute        │ Run analysis                     │
  │ runs:view           │ View analysis results            │
  │ settings:read       │ View organization settings       │
  │ settings:write      │ Modify organization settings     │
  │ billing:read        │ View billing info                │
  │ billing:write       │ Manage subscriptions             │
  │ members:read        │ View team members                │
  │ members:write       │ Invite/remove members            │
  │ apikeys:read        │ View API keys                    │
  │ apikeys:write       │ Create/revoke API keys           │
  └─────────────────────┴──────────────────────────────────┘

Permission Check Example (in code):
  ```typescript
  // In API route
  export async function POST(req: Request) {
    const { userId } = auth();
    const { orgId } = await req.json();

    // Check if user has permission
    const hasPermission = await checkPermission(
      userId,
      orgId,
      'projects:write'
    );

    if (!hasPermission) {
      return new Response('Forbidden', { status: 403 });
    }

    // Proceed with action...
  }
  ```
```

---

## Integration Architecture

### External Service Integration Map

```
┌────────────────────────────────────────────────────────────────────┐
│           EXTERNAL SERVICE INTEGRATIONS                             │
└────────────────────────────────────────────────────────────────────┘

Platform (capysquash.dev)
    │
    ├─── Clerk ──────────────────────────┐
    │    • Authentication               │
    │    • User management              │
    │    • Organization management      │
    │    • Session handling             │
    │    SDK: @clerk/nextjs             │
    │                                   │
    │    Webhooks:                      │
    │    user.created ──▶ /api/webhooks/clerk
    │    user.updated ──▶ /api/webhooks/clerk
    │    org.created  ──▶ /api/webhooks/clerk
    │                                   │
    ├─── Stripe ─────────────────────────┤
    │    • Payment processing           │
    │    • Subscription management      │
    │    • Customer portal              │
    │    • Invoice generation           │
    │    SDK: stripe (Node.js)          │
    │                                   │
    │    Webhooks:                      │
    │    checkout.session.completed ──▶ /api/stripe/webhook
    │    customer.subscription.* ─────▶ /api/stripe/webhook
    │    invoice.* ──────────────────▶ /api/stripe/webhook
    │                                   │
    ├─── GitHub ─────────────────────────┤
    │    • Repository access            │
    │    • PR creation                  │
    │    • Webhook events               │
    │    • OAuth authentication         │
    │    SDK: @octokit/rest             │
    │                                   │
    │    Webhooks:                      │
    │    push ────────────────────────▶ /github/webhook (Engine)
    │    pull_request ───────────────▶ /github/webhook (Engine)
    │    installation.* ─────────────▶ /api/github/webhook
    │                                   │
    ├─── Neon ───────────────────────────┤
    │    • PostgreSQL hosting           │
    │    • Connection pooling           │
    │    • Automatic backups            │
    │    Driver: @neondatabase/serverless
    │    Protocol: PostgreSQL wire      │
    │                                   │
    ├─── Upstash ────────────────────────┤
    │    • Redis caching                │
    │    • Rate limiting                │
    │    • Session storage              │
    │    SDK: @upstash/redis            │
    │    Protocol: Redis                │
    │                                   │
    ├─── Vercel ─────────────────────────┤
    │    • Web app hosting              │
    │    • Edge CDN                     │
    │    • Preview deployments          │
    │    • Analytics                    │
    │    • Cron jobs                    │
    │                                   │
    └─── Monitoring ─────────────────────┘
         • Sentry (error tracking)
         • Vercel Analytics (performance)
         • DataDog (optional APM)


Engine API (api.capysquash.dev)
    │
    ├─── GitHub ─────────────────────────┐
    │    • Webhook handling              │
    │    • Repository cloning            │
    │    • PR creation                   │
    │                                    │
    ├─── OpenAI (Optional) ──────────────┤
    │    • AI-powered analysis           │
    │    • Function equivalency          │
    │    • Dead code detection           │
    │    SDK: sashabaranov/go-openai     │
    │                                    │
    ├─── Anthropic (Optional) ───────────┤
    │    • Alternative AI provider       │
    │    • Claude analysis               │
    │    SDK: anthropics/anthropic-sdk-go│
    │                                    │
    └─── Docker ─────────────────────────┘
         • Schema validation
         • Migration replay
         • Equivalence testing
         Socket: unix:///var/run/docker.sock
```

---

## Database Schema

### Entity Relationship Diagram (Simplified)

```
┌────────────────────────────────────────────────────────────────────┐
│              DATABASE SCHEMA (26 Tables)                            │
│              PostgreSQL 17 via Neon                                 │
└────────────────────────────────────────────────────────────────────┘

CORE ENTITIES:

┌──────────────┐       ┌─────────────────┐       ┌──────────────┐
│organizations │       │ users           │       │ projects     │
├──────────────┤       ├─────────────────┤       ├──────────────┤
│ id (PK)      │◄──┐   │ id (PK)         │   ┌──▶│ id (PK)      │
│ name         │   │   │ clerk_id        │   │   │ name         │
│ slug         │   │   │ email           │   │   │ org_id (FK)  │
│ plan_id      │   │   │ name            │   │   │ repo_url     │
│ created_at   │   │   │ created_at      │   │   │ created_at   │
└──────────────┘   │   └─────────────────┘   │   └──────────────┘
                   │            │             │
                   │            │             │
        ┌──────────┴────────────┴─────────────┘
        │
┌───────────────────────┐
│ organization_         │
│ memberships           │
├───────────────────────┤
│ id (PK)               │
│ user_id (FK) ─────────┼──▶ users.id
│ organization_id (FK) ─┼──▶ organizations.id
│ role (OWNER/ADMIN/...)│
│ joined_at             │
└───────────────────────┘


ANALYSIS ENTITIES:

┌──────────────┐       ┌──────────────────┐
│ projects     │       │ analysis_runs    │
│              │       ├──────────────────┤
│ id (PK)      │◄──────│ id (PK)          │
│ ...          │       │ project_id (FK)  │
└──────────────┘       │ status           │
                       │ safety_level     │
                       │ file_count       │
                       │ results (JSONB)  │
                       │ started_at       │
                       └──────────────────┘
                                │
                                │
                       ┌────────▼──────────┐
                       │ migration_files   │
                       ├───────────────────┤
                       │ id (PK)           │
                       │ run_id (FK)       │
                       │ filename          │
                       │ content           │
                       │ order             │
                       └───────────────────┘


BILLING ENTITIES:

┌──────────────┐       ┌─────────────────────┐
│organizations │       │ subscriptions       │
│              │       ├─────────────────────┤
│ id (PK)      │◄──────│ id (PK)             │
│ plan_id (FK) ├──┐    │ org_id (FK)         │
└──────────────┘  │    │ stripe_customer_id  │
                  │    │ stripe_subscription │
                  │    │ status              │
                  │    │ current_period_end  │
                  │    └─────────────────────┘
                  │
         ┌────────▼─────────┐
         │subscription_plans│
         ├──────────────────┤
         │ id (PK)          │
         │ name             │
         │ price            │
         │ stripe_price_id  │
         │ features (JSONB) │
         └──────────────────┘


Complete Table List (26):

CORE (6):
  • organizations
  • users
  • projects
  • analysis_runs
  • migration_files
  • organization_memberships

PAYMENTS (4):
  • subscriptions
  • usage_tracking
  • subscription_plans
  • stripe_events

GITHUB (2):
  • github_installations
  • github_webhook_events

SETTINGS (3):
  • organization_settings
  • user_preferences
  • notification_rules

AUDIT (4):
  • activity_log
  • comments
  • favorites
  • notification_history

API (2):
  • api_keys
  • database_connections

TEMPLATES (2):
  • project_templates
  • feature_flags

CONFIG (1):
  • subscription_plan_limits
```

---

**For Complete Details**:
- See `capysquash-platform/docs/DATABASE_SCHEMA.md` for full schema documentation
- See `ECOSYSTEM_COMPLETE_GUIDE.md` for comprehensive technical guide
- See `ECOSYSTEM_QUICK_REFERENCE.md` for quick lookups

---

Last Updated: October 20, 2025
