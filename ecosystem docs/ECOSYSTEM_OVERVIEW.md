````markdown
# PGSQUASH Ecosystem Overview

**The definitive guide to the pgsquash ecosystem architecture, components, integrations, and business model.**

---

## 📋 Table of Contents

1. [Executive Summary](#executive-summary)
2. [Ecosystem Architecture](#ecosystem-architecture)
3. [Core Components](#core-components)
4. [Component Relationships](#component-relationships)
5. [Technical Stack](#technical-stack)
6. [Development Workflow](#development-workflow)
7. [Deployment Architecture](#deployment-architecture)
8. [Business Model](#business-model)
9. [Integration Ecosystem](#integration-ecosystem)
10. [Data Flow & APIs](#data-flow--apis)
11. [Security & Authentication](#security--authentication)
12. [Monitoring & Operations](#monitoring--operations)
13. [Repository Structure](#repository-structure)
14. [Quick Reference](#quick-reference)

---

## Executive Summary

**pgsquash** is a comprehensive PostgreSQL migration optimization platform consisting of:

- **Open-source CLI engine** (Go) - The core migration analysis and consolidation engine
- **SaaS web platform** (Next.js) - Team collaboration, automation, and billing wrapper
- **Documentation hub** (Fumadocs) - Public-facing documentation and content
- **Business strategy** - Complete go-to-market positioning and growth framework

This document maps every major part of the ecosystem—what it does, how it connects to the rest of the stack, where it lives in the codebase, and the business lane it supports. Treat this as the home base before diving into more specialized docs.

---

## Ecosystem Architecture

### High-Level System Diagram

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          PGSQUASH ECOSYSTEM                              │
└─────────────────────────────────────────────────────────────────────────┘

┌──────────────────────┐         ┌──────────────────────┐
│   User Touchpoints   │         │  External Services   │
├──────────────────────┤         ├──────────────────────┤
│ • CLI (local)        │◄────────┤ • GitHub (webhooks)  │
│ • Web UI             │         │ • Clerk (auth)       │
│ • GitHub Bot         │         │ • Stripe (billing)   │
│ • VS Code Extension  │         │ • Neon/Supabase DB   │
│ • Documentation      │         │ • Vercel (hosting)   │
└──────────┬───────────┘         └──────────────────────┘
           │
           ▼
┌─────────────────────────────────────────────────────────────────┐
│                    APPLICATION LAYER                             │
├──────────────────────┬──────────────────────┬───────────────────┤
│  capysquash-platform │   pgsquash-engine    │ capysquash-docs   │
│  (Next.js 15)        │   (Go 1.22)          │ (Fumadocs)        │
├──────────────────────┼──────────────────────┼───────────────────┤
│ • Dashboard UI       │ • CLI Tool           │ • User Guides     │
│ • Team Management    │ • API Server         │ • API Reference   │
│ • Billing/RBAC       │ • SQL Parser         │ • Tutorials       │
│ • Analytics          │ • Squasher Engine    │ • Blog            │
│ • GitHub Integration │ • Validation         │ • Marketing       │
└──────────┬───────────┴──────────┬───────────┴───────────────────┘
           │                      │
           ▼                      ▼
┌─────────────────────────────────────────────────────────────────┐
│                    DATA & INFRASTRUCTURE                         │
├──────────────────────┬──────────────────────┬───────────────────┤
│  PostgreSQL (Neon)   │   Docker Runtime     │  Redis (Upstash)  │
│  • User data         │   • Validation       │  • Rate limiting  │
│  • Projects          │   • Schema testing   │  • Caching        │
│  • Analytics         │   • Migrations       │  • Sessions       │
└──────────────────────┴──────────────────────┴───────────────────┘
```

### Three-Tier Value Proposition

1. **Free/Community Tier** - Open-source CLI drives awareness and adoption
2. **Self-Serve SaaS** - Platform enables team collaboration and automation
3. **Enterprise** - Custom deployments, compliance, and dedicated support

---

## Core Components

### 1. pgsquash-engine (Core Engine)

**Location**: `/pgsquash-engine`
**Language**: Go 1.22
**License**: MIT (Open Source)
**Primary Purpose**: PostgreSQL migration analysis, consolidation, and validation

#### Key Responsibilities

- **SQL Parsing** - Uses PostgreSQL's actual parser (`pg_query_go`) for 100% accurate SQL analysis
- **Dependency Resolution** - Builds dependency graphs to safely reorder statements
- **Migration Consolidation** - Intelligently merges migration files while preserving semantics
- **Schema Validation** - Docker-based validation proves equivalence between original and squashed migrations
- **Safety Levels** - Multiple modes (Paranoid, Conservative, Standard, Aggressive) with clear tradeoffs
- **Platform Intelligence** - Built-in knowledge of Supabase, Clerk, Auth0, Neon patterns
- **AI Integration** - Semantic analysis for function equivalency and dead code detection

#### Entry Points

- **CLI** (`cmd/pgsquash/`) - Direct developer tool, distributed as binary
- **API Server** (`cmd/api-server/`) - HTTP REST API consumed by platform
- **GitHub Handlers** - Webhook endpoints for PR automation

#### Core Modules

```
internal/
├── parser/              # pg_query_go bindings, SQL parsing
├── tracking/            # Object lifecycle tracking across migrations
├── squasher/            # Consolidation logic and merge strategies
├── validation/          # Docker-based schema validation
├── ai/                  # AI provider integrations (OpenAI, Anthropic)
├── github/              # GitHub App integration, webhook handling
├── plugins/             # Extensible plugin system
├── transformation/      # SQL transformations and optimizations
└── tui/                 # Terminal UI (interactive mode)
```

#### Distribution

- **Direct**: Binary downloads for Linux, macOS, Windows
- **GitHub**: Open-source repository drives adoption
- **Docker**: Container images for CI/CD integration
- **Platform**: Called by capysquash-platform via HTTP API

#### Business Function

- **Awareness**: Free tier drives top-of-funnel adoption
- **Trust**: Open source = transparency and community contribution
- **Differentiation**: Technical moat (pg_query parser, validation)
- **Distribution**: Viral CLI-to-web funnel

---

### 2. capysquash-platform (SaaS Application)

**Location**: `/capysquash-platform`
**Language**: TypeScript (Next.js 15, React 19)
**License**: Proprietary
**Primary Purpose**: Team collaboration, automation, billing, and analytics wrapper

#### Key Responsibilities

- **User Management** - Clerk-powered authentication with 3-tier RBAC
- **Organization Management** - Teams, roles, permissions, API keys
- **Project Management** - Repository connections, analysis history, settings
- **Billing & Subscriptions** - Stripe integration across 5 pricing tiers
- **GitHub Integration** - GitHub App installation, webhook processing, PR automation
- **Analytics & Reporting** - Usage dashboards, optimization metrics, team activity
- **Demo System** - Automatic sample project creation for new users
- **Automation** - Scheduled cleanups, notifications, workflow triggers

#### Technical Architecture

**Frontend**:
- Next.js 15 App Router
- React 19 with Server Components
- TailwindCSS 4 + shadcn/ui components
- Motion for animations
- TypeScript strict mode

**Backend**:
- Next.js API routes (56 endpoints)
- Drizzle ORM for database
- Clerk for authentication
- Stripe for payments
- Redis for caching/rate limiting

**Database Schema** (26 tables):
- **Core** (6): organizations, users, projects, runs, files, memberships
- **Payments** (4): subscriptions, usage_tracking, plans, stripe_events
- **GitHub** (2): installations, webhook_events
- **Settings** (3): org_settings, user_preferences, notification_rules
- **Audit** (4): activities, comments, favorites, notification_history
- **API** (2): api_keys, database_connections
- **Templates** (2): project_templates, feature_flags
- **Configuration** (1): subscription_plans

#### API Endpoints (56 total)

- **Engine** (2): `/api/engine/analyze`, `/api/engine/squash`
- **Dashboard** (1): `/api/dashboard/metrics`
- **Usage** (3): check, validate, increment
- **Organizations** (17): settings, API keys, database connections, notifications
- **Users** (4): preferences, recent/favorite projects
- **Stripe** (3): checkout, portal, webhook
- **GitHub** (3): installation, webhook, repositories
- **Admin** (15): overview, revenue, usage, activity, plans, health
- **Webhooks** (2): Clerk, GitHub
- **Demo** (1): setup
- **Debug** (1): session

#### Distribution

- **Web UI**: Hosted on Vercel (https://capysquash.dev)
- **GitHub App**: GitHub Marketplace integration
- **Platform Partnerships**: Neon, Supabase, Vercel marketplace

#### Business Function

- **Monetization**: Primary revenue source (Creator → Enterprise tiers)
- **Retention**: Team features create stickiness
- **Expansion**: Usage-based upsell opportunities
- **Enterprise**: SSO, audit logs, compliance features

---

### 3. capysquash-docs (Documentation Hub)

**Location**: `/capysquash-docs`
**Framework**: Next.js + Fumadocs
**License**: Proprietary
**Primary Purpose**: Public documentation, guides, and marketing content

#### Key Responsibilities

- **Product Documentation** - Complete API reference, user guides, CLI reference
- **Integration Guides** - Platform-specific setup (Supabase, Neon, Drizzle, Prisma)
- **Tutorials** - Step-by-step walkthroughs for common workflows
- **Blog** - Product updates, case studies, technical content
- **Marketing** - Feature pages, comparison guides, use cases
- **SEO** - Organic traffic generation through technical content

#### Content Structure

```
content/
├── docs/
│   ├── getting-started/      # Quick start guides
│   ├── cli-reference/         # Complete CLI documentation
│   ├── api-reference/         # HTTP API documentation
│   ├── guides/                # How-to guides
│   ├── integrations/          # Platform integrations
│   ├── safety/                # Safety levels, validation
│   └── troubleshooting/       # Common issues
└── blog/
    ├── announcements/         # Product updates
    ├── case-studies/          # Customer stories
    └── tutorials/             # Technical tutorials
```

#### Features

- **MDX Support** - Rich interactive documentation
- **Code Examples** - Syntax-highlighted code blocks
- **Search** - Full-text search across all docs
- **Version Control** - Documentation versioning aligned with releases
- **Analytics** - Track popular pages, search queries

#### Distribution

- **Public Website**: docs.capysquash.dev (or subdomain)
- **In-App Links**: Deep links from platform UI
- **SEO**: Organic search traffic
- **Community**: Shared on social media, forums

#### Business Function

- **Activation**: Reduces time-to-value for new users
- **Support Deflection**: Self-service reduces support load
- **SEO**: Drives organic traffic and awareness
- **Sales Enablement**: Supports enterprise sales process

---

### 4. Branding & Business Docs

**Location**: `/branding and business docs`
**Format**: Markdown knowledge base
**Primary Purpose**: Strategy, positioning, GTM framework

#### Key Documents

- **Complete Strategy** (`pgsquash-complete-strategy.md`) - Full market analysis, positioning, pricing
- **Brand Guide** (`capysquash-complete-brand-guide.md`) - Visual identity, voice, messaging
- **Integration Roadmap** (`pgsquash-integration-roadmap.md`) - Platform partnership strategy
- **Feature Roadmap** (`CapySquash Feature Roadmap - MoSCoW.md`) - Product development priorities
- **Product Roadmap** (`PRODUCT_ROADMAP_AND_GROWTH_STRATEGY.md`) - Growth framework

#### Key Insights

**Positioning**: "Autopilot for your Postgres migrations" - targeting "vibe coders" (frontend developers using Next.js + Supabase/Neon)

**Pricing Tiers**:
- Free: CLI unlimited, web limited (3 repos)
- Creator: $9-12/mo - Solo developers, side projects
- Professional: $19-29/mo - Teams up to 5, collaboration features
- Agency: $99/mo - Client projects, white-label reports
- Enterprise: Custom - SSO, compliance, on-premise

**Target Market**:
- Primary: Indie hackers, startup teams using Next.js + Supabase/Neon
- Secondary: Agencies building client projects
- Future: Enterprise teams with compliance requirements

**Revenue Targets**:
- Year 1: $100-150k ARR
- Year 2: $400-600k ARR

#### Business Function

- **Strategic Alignment** - Ensures all teams work toward unified goals
- **Marketing Framework** - Guides messaging, content, campaigns
- **Sales Playbook** - Supports enterprise sales motions
- **Investor Relations** - Provides narrative for fundraising

---

### 5. Shared Deployment Assets

**Location**: Root `/docker-compose.yml` + `/capysquash-platform/docker-compose.yml`
**Purpose**: Orchestrate multi-component deployments

#### Root Deployment (Production API)

**File**: `/docker-compose.yml`
**Purpose**: Deploy Go API server only (Vercel hosts web app)

```yaml
Services:
  api-server:
    - Go API on port 8080
    - GitHub webhook handlers
    - Health checks
    - Resource limits (2 CPU, 1GB RAM)
```

**Use Cases**:
- Fly.io deployment
- Cloud container platforms
- Standalone API hosting

#### Platform Deployment (Full Stack)

**File**: `/capysquash-platform/docker-compose.yml`
**Purpose**: Multi-profile deployment for different scenarios

**Profiles**:

1. **`dev`** - API server only (for local development with `pnpm dev`)
   ```bash
   docker compose --profile dev up -d
   pnpm dev  # Run in separate terminal
   ```

2. **`full-stack`** - Complete deployment (web + API + database + Redis)
   ```bash
   docker compose --profile full-stack up -d
   # Access at http://localhost:3000
   ```

3. **`production`** - Full stack + Nginx reverse proxy
   ```bash
   docker compose --profile production up -d
   # Access at http://localhost (port 80)
   ```

**Services**:
- **api-server**: Go API (from pgsquash-engine)
- **webapp**: Next.js application
- **postgres**: PostgreSQL 17 database
- **redis**: Redis 7 for caching (optional)
- **nginx**: Reverse proxy (production only)

#### Business Function

- **Hosted SaaS**: Powers capysquash.dev production
- **Enterprise**: Enables on-premise deployments
- **Partners**: Supports white-label/reseller deployments
- **Development**: Consistent local/staging/production environments

---

## Component Relationships

### Communication Patterns

```
┌────────────────────┐
│  End Users         │
└────────┬───────────┘
         │
    ┌────┼────┐
    │    │    │
    ▼    ▼    ▼
  ┌───┐┌───┐┌───┐
  │CLI││Web││Bot│  User Interfaces
  └─┬─┘└─┬─┘└─┬─┘
    │    │    │
    │    ▼    │
    │  ┌──────────────┐
    │  │  Next.js     │  Platform Layer
    │  │  (API Routes)│
    │  └──────┬───────┘
    │         │
    └────┐    │    ┌────┐
         │    │    │
         ▼    ▼    ▼
    ┌─────────────────┐
    │  Go API Server  │  Engine Layer
    │  (pgsquash-     │
    │   engine)        │
    └────────┬────────┘
             │
        ┌────┼────┐
        │    │    │
        ▼    ▼    ▼
    ┌────┐┌────┐┌────┐
    │ DB ││File││Val │  Storage/Processing
    └────┘└────┘└────┘
```

### Data Flow Patterns

#### Pattern 1: CLI Analysis (Direct)

```
Developer → CLI → Go Engine → Analysis → Terminal Output
```

1. Developer runs `pgsquash analyze migrations/`
2. CLI reads files, sends to parser
3. Engine analyzes dependencies
4. Results printed to terminal
5. No platform involvement

#### Pattern 2: Web UI Analysis (Platform)

```
User → Web UI → Next.js API → Go Engine API → Analysis → Database → Dashboard
```

1. User uploads migrations via web UI
2. Next.js API route receives files
3. API forwards to Go engine via HTTP
4. Engine processes and returns results
5. Platform stores in PostgreSQL
6. Dashboard displays analytics

#### Pattern 3: GitHub Automation (Webhook)

```
GitHub → Webhook → Go Engine → Analysis → Platform API → GitHub PR
```

1. Push event triggers GitHub webhook
2. Webhook hits Go engine directly
3. Engine analyzes repository migrations
4. Results posted to platform API
5. Platform creates PR with results
6. Bot comments on PR

#### Pattern 4: Scheduled Cleanup (Automation)

```
Cron → Platform → GitHub API → Repository → Go Engine → Analysis → PR
```

1. Platform cron job triggers
2. Fetches repositories needing cleanup
3. Clones repository via GitHub API
4. Sends migrations to Go engine
5. Engine returns squashed files
6. Platform creates cleanup PR

### Integration Points

#### Platform → Engine Communication

**Method**: HTTP REST API
**Authentication**: Shared secret (`CAPYSQUASH_API_SECRET`)
**Endpoints**:
- `POST /analyze` - Analyze migrations
- `POST /squash` - Consolidate migrations
- `POST /validate` - Validate equivalence
- `GET /health` - Health check

**Request Flow**:
```typescript
// Platform code
const response = await fetch(`${GO_ENGINE_URL}/analyze`, {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'X-API-Secret': process.env.CAPYSQUASH_API_SECRET
  },
  body: JSON.stringify({
    migrations: files,
    safetyLevel: 'conservative'
  })
});
```

#### Engine → Platform Communication

**Method**: Callback URLs (for async operations)
**Authentication**: Signed payloads
**Use Cases**:
- Long-running analysis completion
- Validation results
- Error notifications

#### GitHub Integration

**Direction**: Bidirectional

**Inbound** (GitHub → System):
- Webhook events (push, PR, installation)
- Delivered to Go engine webhook endpoint
- Engine processes and optionally notifies platform

**Outbound** (System → GitHub):
- PR creation (via GitHub API)
- PR comments (via GitHub API)
- Status checks (via GitHub API)
- Authentication via GitHub App credentials

#### External Services

**Clerk** (Authentication):
- Next.js middleware integration
- Webhook for user lifecycle events
- JWT verification for API routes

**Stripe** (Billing):
- Checkout session creation
- Customer portal
- Webhook for subscription events
- Usage-based metering

**Neon/Supabase** (Database):
- Platform uses Neon-managed PostgreSQL
- Connection via DATABASE_URL
- Drizzle ORM for queries

**Upstash Redis** (Optional):
- Rate limiting
- API response caching
- Session storage

---

## Technical Stack

### pgsquash-engine (Go)

## Component Matrix

| Component | Primary Role | Key Relationships | Dev Placement (repo · stack) | Business Placement |
|-----------|--------------|-------------------|------------------------------|--------------------|
| **pgsquash-engine** | Core migration consolidation engine, exposes CLI (`cmd/pgsquash`) and API (`cmd/api-server`) plus GitHub webhook handlers | Consumed by Next.js platform via `GO_ENGINE_URL`; CLI distributed directly to developers; GitHub App and Docker validation rely on it | `pgsquash-engine/` · Go 1.22, Docker, Fly.io manifests | Free open-source entry point that feeds self-serve funnel and powers paid automation features |
| **capysquash-platform** | Team-facing web app: onboarding, RBAC, billing, analytics, GitHub automation, demo projects | Calls engine API, persists data to Postgres/Neon, uses Clerk auth, Stripe billing, Redis caching; surfaces docs links | `capysquash-platform/` · Next.js 15, React 19, Drizzle ORM, Clerk, Stripe | Commercial SaaS plans (Creator → Enterprise). Targets startups, agencies, larger teams |
| **capysquash-docs** | Public documentation portal covering both CLI and SaaS | Pulls canonical product narratives, links into platform onboarding, referenced by support & marketing | `capysquash-docs/` · Next.js + Fumadocs, MDX content in `content/docs` | Drives activation, reduces support load, supports sales enablement |
| **Branding & business docs** | Strategy, roadmap, messaging, pricing playbooks | Inform marketing site, sales collateral, and platform roadmap prioritization | `branding and business docs/` · Markdown knowledge base | Guides GTM motion, pricing decisions, investor narratives |
| **Shared deployment assets** | Orchestrates engine + platform + data stores via Docker profiles | Production compose (`docker-compose.yml`) deploys API; platform compose coordinates API + Next.js + Postgres + Redis + Nginx | Root `docker-compose.yml`, `capysquash-platform/docker-compose.yml`, `pgsquash-engine/docker` | Enables hosted offerings, partner deployments, and enterprise/on-prem deals |

## Relationship Overview

```
┌────────────────────┐         HTTP / Webhooks         ┌──────────────────────┐
│ capysquash-platform│ ────────▶ pgsquash-engine API ─▶│ Migration Processing │
│ (Next.js SaaS)     │◀────────▶ GitHub App events     │ (Go services)        │
└─────────▲──────────┘         ▲                      └─────────┬────────────┘
          │                    │                                 │
          │ Clerk/Stripe/Auth  │ CLI invocations                 │ Docker validation
          │                    │                                 │
          ▼                    ▼                                 ▼
     Postgres (Neon)      CLI users (local)                Containerized runners
```

- **Data flow** – Platform uploads migrations or streams repo contents to the engine, which returns analysis artifacts stored in the platform database and surfaced in dashboards.
- **Automation** – GitHub webhooks hit the engine; the platform records results to show PR activity and billing usage.
- **Documentation feedback loop** – Docs cite engine and platform capabilities, while platform onboarding links back to docs for activation.

## Deployment Views

- **Local development** – Run `docker compose --profile dev` inside `capysquash-platform` to boot the Go engine container while `pnpm dev` runs the Next.js app. Developers can swap in a local Go binary for quicker iteration.
- **Full stack preview** – Use the `full-stack` profile to bring up engine, platform, Postgres, and optional Redis, mirroring the hosted environment.
- **Production** – The root compose file deploys only the Go API (commonly Fly.io or container platforms). The SaaS front end is deployed to Vercel and points to the hosted API via `GO_ENGINE_URL`. Enterprise/self-hosted customers can run `production` profile (Next.js + API + Nginx).

## Integration Touchpoints

- **Authentication** – Clerk powers user management (`NEXT_PUBLIC_CLERK_*` envs) within the platform; engine remains stateless and keyed via `CAPYSQUASH_API_SECRET`.
- **Billing** – Stripe keys live in the platform (`STRIPE_SECRET_KEY`, publishable keys) to support tiered plans (Creator, Professional, Agency, Enterprise).
- **GitHub** – Engine handles webhook endpoints (`/github/webhook`), uses PATs or GitHub App credentials, and posts PR comments; the platform surfaces activity logs.
- **PostgreSQL / Neon** – Platform uses Neon-managed database via Drizzle; engine can spin Dockerized Postgres to validate squashed migrations.
- **Redis / Upstash** – Optional caching and rate limiting for API throughput as configured in `capysquash-platform/docker-compose.yml`.
- **CI/CD** – CLI and API both integrate into pipelines through Docker images, Fly deployments, and scripted smoke tests in `capysquash-platform/scripts`.

## Ownership & Development Notes

- **Engine Engineering** – Focus on Go modules under `internal/` (parser, tracker, squasher, validation, AI providers). Release cadence tracked in `pgsquash-engine/CHANGELOG.md`.
- **Platform Engineering** – Owns Next.js app (`src/app`, `lib`, `hooks`) with documentation in `docs/internal`. Responsible for RBAC, analytics, onboarding, and integration UX.
- **Documentation Team** – Maintains MDX content in `capysquash-docs/content`, syncing product updates with platform releases; coordinates with marketing for blog posts.
- **Growth & GTM** – Works out of `branding and business docs/`, aligning pricing, positioning, and roadmap; feeds requirements back to platform and engine teams.

## Business Placement Summary

- **Free Tier / Awareness** – pgsquash-engine CLI delivers value immediately and seeds advocacy. Documentation and demo migrations (via platform) reduce friction.
- **Self-Serve SaaS** – capysquash-platform monetizes collaborators who need RBAC, dashboards, and automation. Target personas: startups exiting MVP, agencies with multiple clients.
- **Enterprise & Partnerships** – Shared deployment assets plus business strategy docs enable on-prem, SSO/SAML, compliance workflows, and partner integrations.
- **Content & Community** – capysquash-docs, blog content, and Discord setup drive retention, support deflection, and upsell cues highlighted in `branding and business docs`.

## Reference Map

- Engine product docs – `pgsquash-engine/docs/user docs/`
- Platform internal docs – `capysquash-platform/docs/internal/`
- Public documentation site – `capysquash-docs/content/docs/`
- Strategy & positioning – `branding and business docs/`
- Deployment guides – root `docker-compose.yml`, `capysquash-platform/docker-compose.yml`, `pgsquash-engine/docker/`

Keep this overview updated whenever new services, integrations, or plans land so teams can navigate the ecosystem without spelunking through multiple repositories.
````
