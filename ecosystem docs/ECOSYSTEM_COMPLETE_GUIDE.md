# PGSQUASH Complete Ecosystem Guide

**The definitive reference for the pgsquash ecosystem - architecture, components, integrations, development, deployment, and business model.**

**Last Updated**: October 20, 2025\
**Version**: 1.0

---

## 📋 Table of Contents

1. [Executive Summary](#executive-summary)
2. [Ecosystem Architecture](#ecosystem-architecture)
3. [Core Components](#core-components)
4. [Technical Stack](#technical-stack)
5. [Component Relationships](#component-relationships)
6. [Development Workflow](#development-workflow)
7. [Deployment Architecture](#deployment-architecture)
8. [Business Model](#business-model)
9. [Integration Ecosystem](#integration-ecosystem)
10. [Data Flow & APIs](#data-flow--apis)
11. [Security & Authentication](#security--authentication)
12. [Monitoring & Operations](#monitoring--operations)
13. [Repository Structure](#repository-structure)
14. [Team Organization](#team-organization)
15. [Quick Reference](#quick-reference)

---

## Executive Summary

### What is pgsquash?

**pgsquash** is a comprehensive PostgreSQL migration optimization platform that automatically consolidates messy migration files while ensuring safety and preserving database semantics.

### Ecosystem Components

1. **pgsquash-engine** (Go) - Open-source CLI and API for migration analysis and consolidation
2. **capysquash-platform** (Next.js) - SaaS web application for teams, automation, and billing
3. **capysquash-docs** (Fumadocs) - Public documentation, guides, and content
4. **Branding & Business Docs** - Strategy, positioning, and growth framework
5. **Deployment Infrastructure** - Docker, orchestration, and integration assets

### Value Proposition

**For Developers**: "Autopilot for your Postgres migrations"

- Reduce migration files by 60-80%
- Cut deployment time by 40-70%
- Docker-validated safety guarantees
- Zero database access required

**For Teams**: Collaboration, automation, and governance

- GitHub PR automation
- Team analytics dashboards
- Role-based access control
- Scheduled cleanups

**For Enterprises**: Compliance and control

- On-premise deployment
- SSO/SAML integration
- Audit logs
- SLA guarantees

### Business Model

- **Free Tier**: Open-source CLI drives awareness
- **Creator** ($12/mo): Solo developers, unlimited repos
- **Professional** ($29/mo): Teams up to 5, collaboration features
- **Agency** ($99/mo): Client projects, white-label reports
- **Enterprise** (Custom): SSO, compliance, on-premise

**Revenue Targets**:

- Year 1: $100-150k ARR
- Year 2: $400-600k ARR

---

## Ecosystem Architecture

### High-Level System Diagram

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          PGSQUASH ECOSYSTEM                              │
└─────────────────────────────────────────────────────────────────────────┘

┌──────────────────────────┐         ┌──────────────────────────┐
│     User Touchpoints     │         │    External Services     │
├──────────────────────────┤         ├──────────────────────────┤
│ • CLI (local binary)     │◄────────┤ • GitHub (webhooks, API) │
│ • Web UI (browser)       │         │ • Clerk (auth)           │
│ • GitHub Bot (PR)        │         │ • Stripe (billing)       │
│ • VS Code Extension      │         │ • Neon (database)        │
│ • Documentation (web)    │         │ • Vercel (hosting)       │
└──────────┬───────────────┘         │ • Upstash (Redis)        │
           │                         └──────────────────────────┘
           ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         APPLICATION LAYER                                │
├───────────────────────┬─────────────────────────┬───────────────────────┤
│  capysquash-platform  │    pgsquash-engine      │   capysquash-docs     │
│  (Next.js 15)         │    (Go 1.22)            │   (Fumadocs)          │
├───────────────────────┼─────────────────────────┼───────────────────────┤
│ • Dashboard UI        │ • CLI Tool              │ • User Guides         │
│ • Team Management     │ • API Server            │ • API Reference       │
│ • Billing/RBAC        │ • SQL Parser            │ • Tutorials           │
│ • Analytics           │ • Squasher Engine       │ • Blog Content        │
│ • GitHub Integration  │ • Validation Engine     │ • Marketing Pages     │
│ • Project Management  │ • AI Analysis           │ • SEO Content         │
│ • Automation/Webhooks │ • GitHub App Handler    │ • Case Studies        │
└───────────┬───────────┴──────────┬──────────────┴───────────────────────┘
            │                      │
            ▼                      ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                      DATA & INFRASTRUCTURE                               │
├──────────────────────┬──────────────────────────┬───────────────────────┤
│  PostgreSQL (Neon)   │    Docker Runtime        │   Redis (Upstash)     │
│  • User data         │    • Validation          │   • Rate limiting     │
│  • Organizations     │    • Schema testing      │   • API caching       │
│  • Projects/Runs     │    • Migration replay    │   • Sessions          │
│  • Subscriptions     │    • Equivalence proof   │   • Job queues        │
│  • Analytics         │                          │                       │
└──────────────────────┴──────────────────────────┴───────────────────────┘
```

### Information Flow

```
┌─────────────────────────────────────────────────────────────┐
│                  REQUEST FLOW PATTERNS                      │
└─────────────────────────────────────────────────────────────┘

Pattern 1: CLI Direct Usage
User → CLI → Parser → Squasher → Validator → Output Files

Pattern 2: Web UI Analysis
User → Browser → Next.js → Go API → Processing → Database → Dashboard

Pattern 3: GitHub Automation
GitHub Event → Webhook → Go API → Analysis → Platform API → PR Creation

Pattern 4: Scheduled Automation
Cron → Platform → GitHub API → Go API → Processing → PR Creation
```

---

## Core Components

### 1. pgsquash-engine (Migration Engine)

**Location**: `/pgsquash-engine`\
**Language**: Go 1.22+\
**License**: MIT (Open Source)\
**Current Version**: 0.8.5

#### Purpose

Core migration analysis, consolidation, and validation engine. Provides both CLI and HTTP API interfaces.

#### Key Capabilities

- **Parser-Grade Accuracy**: Uses PostgreSQL's actual parser (`pg_query_go`)
- **Dependency Resolution**: Builds dependency graphs for safe statement reordering
- **Intelligent Consolidation**: Merges migrations while preserving semantics
- **Docker Validation**: Proves mathematical equivalence via schema comparison
- **Safety Levels**: Paranoid, Conservative, Standard, Aggressive modes
- **Platform Intelligence**: Built-in Supabase, Clerk, Neon pattern recognition
- **AI Analysis**: Dead code detection, function equivalency analysis

#### Architecture

```
cmd/
├── pgsquash/           # CLI entry point (main binary)
└── api-server/         # HTTP REST API server

internal/
├── parser/             # pg_query_go bindings, SQL parsing
├── tracking/           # Object lifecycle tracking across migrations
├── squasher/           # Consolidation logic and merge strategies
├── validation/         # Docker-based schema validation
├── ai/                 # AI provider integrations (OpenAI, Anthropic)
├── github/             # GitHub App integration, webhook handling
├── plugins/            # Extensible plugin system
├── transformation/     # SQL transformations and optimizations
└── tui/                # Terminal UI (interactive mode)

migrations/             # Sample/test migration fixtures
scripts/                # Build, validation, deployment scripts
docs/                   # Engine-specific documentation
docker/                 # Dockerfiles and compose configs
```

#### Key Technologies

- **pg\_query\_go**: Official PostgreSQL parser bindings
- **Cobra**: CLI framework
- **Viper**: Configuration management
- **Bubble Tea**: Terminal UI framework
- **Docker SDK**: Validation infrastructure
- **OpenAI/Anthropic**: AI analysis

#### Entry Points

1. **CLI Binary** (`pgsquash`)
   ```bash
   pgsquash analyze migrations/
   pgsquash squash migrations/ --dry-run
   pgsquash validate original/ squashed/
   pgsquash tui migrations/  # Interactive mode
   ```

2. **HTTP API Server**
   ```bash
   api-server  # Runs on :8080
   # POST /analyze, /squash, /validate
   # POST /github/webhook
   # GET  /health
   ```

#### Distribution Channels

- **Direct Download**: Binary releases for Linux, macOS, Windows
- **GitHub Releases**: Automated builds via CI/CD
- **Docker Hub**: Container images
- **Package Managers**: (Planned) Homebrew, apt, etc.
- **Platform Integration**: Called by capysquash-platform

#### Business Function

- **Awareness**: Free tier drives top-of-funnel adoption
- **Trust**: Open source builds transparency and community
- **Differentiation**: Technical moat (pg\_query parser)
- **Viral Growth**: CLI-to-web conversion funnel

---

### 2. capysquash-platform (SaaS Application)

**Location**: `/capysquash-platform`\
**Language**: TypeScript (Next.js 15, React 19)\
**License**: Proprietary\
**Current Version**: 1.0

#### Purpose

Team collaboration, automation, billing, and analytics wrapper around the core engine.

#### Key Capabilities

- **User Management**: Clerk authentication, 3-tier RBAC
- **Organization Management**: Teams, roles, API keys, settings
- **Project Management**: Repo connections, analysis history
- **Billing**: Stripe subscriptions across 5 tiers
- **GitHub Integration**: App installation, webhook processing, PR automation
- **Analytics**: Usage dashboards, optimization metrics, team activity
- **Demo System**: Automatic sample project for new users
- **Automation**: Scheduled cleanups, notifications, workflows

#### Architecture

```
src/
├── app/                           # Next.js App Router
│   ├── (auth)/                    # Authentication pages (sign-in, sign-up)
│   ├── (dashboard)/               # Protected dashboard pages
│   │   ├── dashboard/             # Overview
│   │   ├── projects/              # Project management
│   │   ├── settings/              # User/org settings
│   │   └── admin/                 # Platform admin (ADMIN role)
│   ├── api/                       # API routes (56 endpoints)
│   │   ├── engine/                # Engine proxy endpoints
│   │   ├── dashboard/             # Dashboard data
│   │   ├── organizations/         # Org management
│   │   ├── projects/              # Project CRUD
│   │   ├── stripe/                # Billing endpoints
│   │   ├── github/                # GitHub integration
│   │   ├── admin/                 # Admin endpoints
│   │   └── webhooks/              # External webhooks
│   └── onboarding/                # New user onboarding flow
│
├── components/
│   ├── dashboard/                 # Dashboard-specific components
│   │   ├── AnalysisResults.tsx
│   │   ├── ProjectCard.tsx
│   │   ├── UsageChart.tsx
│   │   └── ...
│   └── ui/                        # shadcn/ui base components
│       ├── button.tsx
│       ├── card.tsx
│       ├── dialog.tsx
│       └── ...
│
├── lib/
│   ├── db/                        # Database layer
│   │   ├── schema.ts              # Drizzle schema (26 tables)
│   │   ├── queries.ts             # Reusable queries
│   │   └── migrations/            # Database migrations
│   ├── auth/                      # RBAC utilities
│   │   ├── permissions.ts
│   │   └── rbac.ts
│   ├── demo-data/                 # Demo migration files
│   │   └── migrations/
│   └── utils/                     # Shared utilities
│       ├── api.ts
│       ├── formatting.ts
│       └── validation.ts
│
├── hooks/                         # Custom React hooks
│   ├── useProjects.ts
│   ├── useOrganization.ts
│   └── useAnalytics.ts
│
└── types/                         # TypeScript type definitions
    ├── database.ts
    ├── api.ts
    └── github.ts

docs/                              # Platform documentation
├── API_REFERENCE.md               # Complete API docs (56 endpoints)
├── DATABASE_SCHEMA.md             # Schema documentation (26 tables)
├── QUICK_START.md                 # Setup guide
├── RBAC_GUIDE.md                  # Permission system
├── guides/                        # How-to guides
├── internal/                      # Internal developer docs
│   ├── architecture/
│   ├── deployment/
│   ├── admin/
│   └── integrations/
└── troubleshooting/

scripts/                           # Maintenance and utility scripts
├── seed-demo-data.ts              # Create demo projects
├── make-admin.ts                  # Grant admin role
├── check-permissions.ts           # Debug RBAC
├── test-apis.ts                   # Smoke tests
└── ...

public/                            # Static assets
├── logos/
├── icons/
└── avatars/

docker/                            # Docker configurations
└── nginx/                         # Nginx configs for production
```

#### Database Schema (26 Tables)

**Core** (6):

- `organizations` - Team/company accounts
- `users` - User profiles
- `projects` - Connected repositories
- `analysis_runs` - Analysis executions
- `migration_files` - File metadata
- `organization_memberships` - User-org relationships

**Payments** (4):

- `subscriptions` - Stripe subscriptions
- `usage_tracking` - Metered usage
- `subscription_plans` - Plan definitions
- `stripe_events` - Webhook event log

**GitHub** (2):

- `github_installations` - GitHub App installations
- `github_webhook_events` - Webhook event log

**Settings** (3):

- `organization_settings` - Org preferences
- `user_preferences` - User preferences
- `notification_rules` - Alert configuration

**Audit** (4):

- `activity_log` - Action audit trail
- `comments` - Project comments
- `favorites` - Starred projects
- `notification_history` - Sent notifications

**API** (2):

- `api_keys` - API authentication tokens
- `database_connections` - Saved DB configs

**Templates** (2):

- `project_templates` - Reusable project templates
- `feature_flags` - Feature toggles

**Configuration** (1):

- `subscription_plan_limits` - Plan limit definitions

#### API Endpoints (56 Total)

**Engine Integration** (2):

- `POST /api/engine/analyze` - Analyze migrations
- `POST /api/engine/squash` - Consolidate migrations

**Dashboard** (1):

- `GET /api/dashboard/metrics` - Dashboard overview

**Organizations** (17):

- CRUD operations for organizations
- Settings management
- API key management
- Database connections
- Team member management
- Notification configuration

**Projects** (8):

- CRUD operations for projects
- Analysis runs
- File management
- GitHub repository connections

**Users** (4):

- Preferences
- Recent projects
- Favorite projects
- Activity feed

**Stripe** (3):

- `POST /api/stripe/checkout` - Create checkout session
- `POST /api/stripe/portal` - Customer portal
- `POST /api/stripe/webhook` - Webhook handler

**GitHub** (3):

- `POST /api/github/webhook` - Webhook handler
- `GET /api/github/installation` - Installation status
- `GET /api/github/repositories` - Connected repos

**Admin** (15):

- Platform overview
- Revenue metrics
- Usage statistics
- User management
- Subscription management
- Activity monitoring
- Health checks

**Webhooks** (2):

- `POST /api/webhooks/clerk` - Clerk user lifecycle
- `POST /api/webhooks/github` - GitHub events

**Demo** (1):

- `POST /api/demo/setup` - Create demo project

**Debug** (1):

- `GET /api/debug/session` - Session diagnostics

#### Key Technologies

**Frontend**:

- **Next.js 15**: App Router, Server Components, API Routes
- **React 19**: Latest React features
- **TailwindCSS 4**: Utility-first styling
- **shadcn/ui**: Component library
- **Radix UI**: Headless UI primitives
- **Motion**: Animations
- **TypeScript 5**: Strict mode

**Backend**:

- **Drizzle ORM**: Type-safe database queries
- **Clerk**: Authentication and user management
- **Stripe**: Payment processing
- **Neon**: Serverless PostgreSQL
- **Upstash Redis**: Caching and rate limiting (optional)

**Developer Experience**:

- **Biome**: Linting and formatting (replaces ESLint + Prettier)
- **pnpm**: Fast package manager
- **tsx**: TypeScript execution for scripts

#### Distribution

- **Web UI**: <https://capysquash.dev> (Vercel hosted)
- **GitHub App**: GitHub Marketplace
- **Platform Partnerships**: Neon, Supabase integrations

#### Business Function

- **Monetization**: Primary revenue source (subscriptions)
- **Retention**: Team features create lock-in
- **Expansion**: Usage-based upsells
- **Enterprise**: Compliance and governance features

---

### 3. capysquash-docs (Documentation Hub)

**Location**: `/capysquash-docs`\
**Framework**: Next.js + Fumadocs\
**License**: Proprietary

#### Purpose

Public-facing documentation, guides, tutorials, and marketing content. Serves as the knowledge base for both free and paid users.

#### Content Structure

```
content/
├── docs/                          # Documentation
│   ├── getting-started/
│   │   ├── introduction.mdx
│   │   ├── quick-start.mdx
│   │   └── installation.mdx
│   ├── cli-reference/
│   │   ├── analyze.mdx
│   │   ├── squash.mdx
│   │   ├── validate.mdx
│   │   └── config.mdx
│   ├── api-reference/
│   │   ├── authentication.mdx
│   │   ├── endpoints.mdx
│   │   └── webhooks.mdx
│   ├── guides/
│   │   ├── supabase-integration.mdx
│   │   ├── neon-integration.mdx
│   │   ├── github-setup.mdx
│   │   └── ci-cd-integration.mdx
│   ├── integrations/
│   │   ├── drizzle.mdx
│   │   ├── prisma.mdx
│   │   ├── vercel.mdx
│   │   └── vs-code.mdx
│   ├── safety/
│   │   ├── safety-levels.mdx
│   │   ├── validation.mdx
│   │   └── best-practices.mdx
│   └── troubleshooting/
│       ├── common-issues.mdx
│       └── faq.mdx
│
└── blog/                          # Blog content
    ├── announcements/
    ├── case-studies/
    └── tutorials/

src/                               # Fumadocs application
├── app/
├── components/
└── lib/
```

#### Features

- **MDX Support**: Rich, interactive documentation
- **Code Examples**: Syntax-highlighted code blocks
- **Full-Text Search**: Instant search across all content
- **Version Control**: Docs versioned with product releases
- **Analytics**: Track popular pages and search terms
- **SEO Optimized**: Meta tags, sitemap, structured data

#### Distribution

- **Public Website**: docs.capysquash.dev (or subdomain)
- **In-App Links**: Deep links from platform UI
- **SEO**: Organic search traffic driver
- **Social Sharing**: Twitter, LinkedIn, Dev.to

#### Business Function

- **Activation**: Reduces time-to-value for new users
- **Support Deflection**: Self-service reduces support tickets
- **SEO**: Drives organic traffic and awareness
- **Sales Enablement**: Supports enterprise sales conversations

---

### 4. Branding & Business Documentation

**Location**: `/branding and business docs`\
**Format**: Markdown knowledge base

#### Purpose

Strategic playbooks for positioning, pricing, go-to-market, partnerships, and growth.

#### Key Documents

**Strategy & Positioning**:

- **`pgsquash-complete-strategy.md`** (2,016 lines)
  - Complete market validation
  - Competitive landscape analysis
  - Positioning framework ("Autopilot for migrations")
  - Target audience personas (vibe coders, agencies, startups)
  - Revenue model and projections
  - Marketing and growth strategies
  - Implementation roadmap

**Brand Identity**:

- **`capysquash-complete-brand-guide.md`**
  - Visual identity guidelines
  - Voice and tone
  - Messaging framework
  - Marketing materials

**Product Roadmap**:

- **`CapySquash Feature Roadmap - MoSCoW.md`**
  - Must-have features
  - Should-have features
  - Could-have features
  - Won't-have (this release)

- **`PRODUCT_ROADMAP_AND_GROWTH_STRATEGY.md`**
  - Quarter-by-quarter feature plan
  - Growth milestones
  - Partnership strategy

**Integration Strategy**:

- **`pgsquash-integration-roadmap.md`**
  - Platform partnership priorities
  - Integration specifications
  - Co-marketing plans

#### Key Strategic Insights

**Positioning**: "Autopilot for your Postgres migrations"

- Target "vibe coders" (frontend devs using Next.js + Supabase/Neon)
- Emphasis on speed and automation over power/control
- "Zero-config first run" philosophy

**Pricing Strategy**:

```
Free → Creator ($12) → Professional ($29) → Agency ($99) → Enterprise (Custom)
```

**Target Markets**:

1. **Primary**: Indie hackers, startup teams (Next.js + Supabase/Neon stack)
2. **Secondary**: Agencies building client projects
3. **Future**: Enterprise teams with compliance needs

**Revenue Targets**:

- Year 1: $100-150k ARR
- Year 2: $400-600k ARR

**Distribution Priorities**:

1. GitHub App (highest priority)
2. Supabase CLI plugin
3. Neon integration
4. Vercel Marketplace
5. VS Code extension

#### Business Function

- **Strategic Alignment**: Unified vision across all teams
- **Marketing Framework**: Guides all messaging and campaigns
- **Sales Playbook**: Supports enterprise sales motions
- **Investor Relations**: Provides narrative for fundraising

---

### 5. Deployment Infrastructure

#### Root Deployment (`/docker-compose.yml`)

**Purpose**: Deploy Go API server only (for production)

**Services**:

- `api-server` - Go engine HTTP API

**Use Cases**:

- Fly.io deployment
- Cloud container platforms
- Standalone API hosting

**Typical Setup**:

```bash
# Production API deployment
docker compose up -d

# API available at http://localhost:8080
# Web app hosted separately on Vercel
```

#### Platform Deployment (`/capysquash-platform/docker-compose.yml`)

**Purpose**: Multi-profile deployment supporting different scenarios

**Profiles**:

1. **`dev`** - API server only (for local development)
   ```bash
   docker compose --profile dev up -d
   # Then: pnpm dev (in separate terminal)
   ```

2. **`full-stack`** - Complete deployment (web + API + DB + Redis)
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

- `api-server` - Go API from pgsquash-engine
- `webapp` - Next.js application
- `postgres` - PostgreSQL 17 database
- `redis` - Redis 7 for caching (optional profile)
- `nginx` - Reverse proxy (production profile only)

**Business Function**:

- **Hosted SaaS**: Powers production capysquash.dev
- **Enterprise**: Enables on-premise deployments
- **Partners**: Supports white-label/reseller models
- **Development**: Consistent local/staging/production environments

---

## Technical Stack

### Language & Framework Summary

| Component               | Language   | Framework          | Version |
| ----------------------- | ---------- | ------------------ | ------- |
| **pgsquash-engine**     | Go         | Cobra (CLI)        | 1.22+   |
| **capysquash-platform** | TypeScript | Next.js            | 15      |
| **capysquash-docs**     | TypeScript | Next.js + Fumadocs | 15      |

### Core Dependencies by Component

#### pgsquash-engine (Go)

```go
// go.mod
module github.com/CAPYSQUASH/pgsquash-engine

go 1.22

require (
    github.com/pganalyze/pg_query_go/v5  // PostgreSQL parser
    github.com/spf13/cobra               // CLI framework
    github.com/spf13/viper               // Configuration
    github.com/charmbracelet/bubbletea   // TUI framework
    github.com/charmbracelet/bubbles     // TUI components
    github.com/docker/docker             // Docker client
    github.com/sashabaranov/go-openai    // OpenAI SDK
    github.com/anthropics/anthropic-sdk-go // Anthropic SDK
)
```

**Build Commands**:

```bash
go build -o pgsquash cmd/pgsquash/main.go
go build -o api-server cmd/api-server/main.go
go test ./...
go test ./... -cover
```

#### capysquash-platform (TypeScript)

```json
// package.json (key dependencies)
{
  "dependencies": {
    // Framework
    "next": "^15.0.0",
    "react": "^19.0.0",
    "react-dom": "^19.0.0",
    
    // Styling
    "tailwindcss": "^4.0.0",
    "@radix-ui/react-*": "latest",
    "motion": "latest",
    
    // Database
    "drizzle-orm": "latest",
    "@neondatabase/serverless": "latest",
    
    // Authentication
    "@clerk/nextjs": "latest",
    
    // Payments
    "stripe": "latest",
    
    // Utilities
    "typescript": "^5.0.0",
    "zod": "latest"
  },
  "devDependencies": {
    "@biomejs/biome": "latest",
    "drizzle-kit": "latest",
    "tsx": "latest"
  }
}
```

**Build Commands**:

```bash
pnpm install
pnpm dev
pnpm build
pnpm start
pnpm db:generate  # Generate migrations
pnpm db:migrate   # Run migrations
pnpm db:studio    # Open Drizzle Studio
```

#### capysquash-docs (TypeScript)

```json
// package.json (key dependencies)
{
  "dependencies": {
    "next": "^15.0.0",
    "fumadocs-core": "latest",
    "fumadocs-ui": "latest",
    "fumadocs-mdx": "latest",
    "shiki": "latest"
  }
}
```

---

## Component Relationships

### Communication Architecture

```
┌────────────────────────────────────────────────────────────────┐
│                    COMMUNICATION FLOW                           │
└────────────────────────────────────────────────────────────────┘

End Users
    │
    ├─────────┬──────────┬──────────┬──────────┐
    │         │          │          │          │
    ▼         ▼          ▼          ▼          ▼
  CLI      Web UI    GitHub     VS Code    Docs
    │         │       Bot        Ext.
    │         │         │          │
    ▼         ▼         ▼          │
┌─────────────────────────────────┐│
│  pgsquash-engine (Go)           ││
│  • CLI commands                 ││
│  • HTTP API Server (:8080)      ││
│  • GitHub webhook handler       ││
└──────────┬──────────────────────┘│
           │                       │
           ▼                       │
    ┌──────────────────────┐      │
    │ capysquash-platform  │◄─────┘
    │ (Next.js)            │
    │ • Web UI             │
    │ • API Routes         │
    │ • Webhooks           │
    └──────────┬───────────┘
               │
        ┌──────┼──────┬──────────┐
        │      │      │          │
        ▼      ▼      ▼          ▼
    ┌────┐ ┌─────┐ ┌──────┐ ┌───────┐
    │ DB │ │Clerk│ │Stripe│ │GitHub │
    │(PG)│ │Auth │ │ Pay  │ │ API   │
    └────┘ └─────┘ └──────┘ └───────┘
```

### Data Flow Patterns

#### Pattern 1: CLI Direct Usage

```
Developer
    ↓
    $ pgsquash analyze migrations/
    ↓
CLI reads files
    ↓
Parser (pg_query_go)
    ↓
Dependency analyzer
    ↓
Results to terminal
```

**Characteristics**:

- No network calls
- No authentication
- Local file system only
- Completely offline capable

#### Pattern 2: Web UI Analysis

```
User uploads files via browser
    ↓
Next.js API route (/api/engine/analyze)
    ↓
Validates authentication (Clerk)
    ↓
Checks permissions (RBAC)
    ↓
Forwards to Go API (HTTP POST)
    ↓
Go engine processes
    ↓
Returns results to platform
    ↓
Platform saves to database
    ↓
Dashboard displays metrics
```

**Characteristics**:

- Authenticated via Clerk JWT
- Usage tracked for billing
- Results persisted
- Team members can view

#### Pattern 3: GitHub Webhook

```
GitHub event (push, PR)
    ↓
Webhook to Go API (/github/webhook)
    ↓
Verify signature
    ↓
Clone repository
    ↓
Analyze migrations
    ↓
POST results to platform API
    ↓
Platform creates/updates PR
    ↓
Bot comments with metrics
```

**Characteristics**:

- Triggered automatically
- No user intervention
- Async processing
- Results visible in PR

#### Pattern 4: Scheduled Automation

```
Platform cron job
    ↓
Query organizations with automation enabled
    ↓
For each project:
    ↓
    Fetch repo via GitHub API
    ↓
    Send to Go engine
    ↓
    If changes detected:
        ↓
        Create cleanup PR
        ↓
        Notify team
```

**Characteristics**:

- Weekly/monthly schedule
- Pro tier and above
- Email/Slack notifications
- Team approval workflow

### Integration Points

#### Platform → Engine

**Method**: HTTP REST API\
**Authentication**: Shared secret header (`X-API-Secret`)\
**Base URL**: `${GO_ENGINE_URL}` (env var)

**Example Request**:

```typescript
const response = await fetch(`${process.env.GO_ENGINE_URL}/analyze`, {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'X-API-Secret': process.env.CAPYSQUASH_API_SECRET
  },
  body: JSON.stringify({
    migrations: filePaths.map(p => fs.readFileSync(p, 'utf-8')),
    safetyLevel: 'conservative',
    config: { /* optional */ }
  })
});

const result = await response.json();
// { summary, dependencies, recommendations }
```

#### Engine → Platform

**Method**: Callback URLs (for async operations)\
**Authentication**: Signed payloads (HMAC)\
**Use Cases**: Long-running analysis, validation results

#### GitHub Integration

**Inbound** (GitHub → System):

- Webhook events delivered to `/github/webhook`
- Signature verification via `X-Hub-Signature-256`
- Events: push, pull\_request, installation

**Outbound** (System → GitHub):

- PR creation via GitHub API
- PR comments via GitHub API
- Status checks via GitHub API
- Auth via GitHub App credentials

#### Clerk (Authentication)

**Integration Points**:

- Next.js middleware for route protection
- JWT verification in API routes
- Webhook for user lifecycle events (`/api/webhooks/clerk`)
- Organization sync

#### Stripe (Billing)

**Integration Points**:

- Checkout session creation
- Customer portal redirect
- Webhook for subscription events (`/api/webhooks/stripe` or `/api/stripe/webhook`)
- Usage reporting for metered billing

#### Neon/Supabase (Database)

**Platform Connection**:

- PostgreSQL via `DATABASE_URL`
- SSL required (`sslmode=require`)
- Connection pooling via Neon proxy
- Drizzle ORM for queries

**Engine Usage**:

- Pattern recognition for Supabase/Neon-specific constructs
- No direct database connection

---

## Development Workflow

### Local Development Setup

#### Prerequisites

- **Go** 1.22+ (for engine)
- **Node.js** 18+ (for platform/docs)
- **pnpm** (enable via `corepack enable`)
- **Docker** (for validation and local services)
- **PostgreSQL client** (optional, for debugging)

#### Engine Development

```bash
# Clone and navigate
git clone https://github.com/CAPYSQUASH/pgsquash-engine
cd pgsquash-engine

# Install dependencies
go mod tidy

# Build CLI
go build -o pgsquash cmd/pgsquash/main.go

# Try it
./pgsquash analyze migrations/

# Run tests
go test ./...
go test ./... -cover
go test ./... -race  # With race detection

# Build API server
go build -o api-server cmd/api-server/main.go

# Run API server
./api-server  # Listens on :8080

# Run validation tests (requires Docker)
./scripts/validate.sh --mode full --migrations migrations/
```

#### Platform Development

```bash
# Clone and navigate
cd capysquash-platform

# Enable pnpm
corepack enable

# Install dependencies
pnpm install

# Setup environment
cp .env.example .env.local
# Edit .env.local with your credentials

# Push database schema
pnpm db:push

# Start dev server
pnpm dev  # Runs on :3000

# In separate terminal: start API server
docker compose --profile dev up -d

# Or run everything in Docker
docker compose --profile full-stack up -d
```

#### Documentation Development

```bash
cd capysquash-docs
pnpm install
pnpm dev  # Runs on :3000
```

### Recommended Development Workflows

#### Option 1: Hybrid (Recommended for Platform Dev)

```bash
# Terminal 1: API in Docker
cd capysquash-platform
docker compose --profile dev up -d

# Terminal 2: Next.js dev server
pnpm dev

# Hot reload for frontend
# API restarts only when rebuilding container
```

**Pros**: Fast frontend iteration, realistic API environment\
**Cons**: API changes require container rebuild

#### Option 2: All Local (Recommended for Engine Dev)

```bash
# Terminal 1: API server locally
cd pgsquash-engine
./api-server

# Terminal 2: Next.js locally
cd capysquash-platform
pnpm dev

# Both have hot reload
```

**Pros**: Fastest iteration for both\
**Cons**: Need to manage both processes

#### Option 3: All Docker

```bash
cd capysquash-platform
docker compose --profile full-stack up -d

# Access at http://localhost:3000
```

**Pros**: Production-like environment\
**Cons**: Slower iteration (no hot reload)

### Testing Strategy

#### Engine Tests (Go)

```bash
cd pgsquash-engine

# All tests
go test ./...

# With coverage
go test ./... -cover

# Verbose output
go test ./... -v

# Specific package
go test ./internal/squasher/...

# Race detection
go test ./... -race

# Validation tests (requires Docker)
./scripts/validate.sh --mode full --migrations test_migrations/

# Benchmarks
go test ./internal/squasher -bench=.
```

**Coverage Target**: >60%

#### Platform Tests (TypeScript)

```bash
cd capysquash-platform

# Type checking
pnpm tsc --noEmit

# Linting
pnpm lint

# Formatting check
pnpm format:check

# Fix all issues
pnpm check:fix

# Database smoke tests
pnpm tsx scripts/test-apis.ts
pnpm tsx scripts/test-dashboard-query.ts
pnpm tsx scripts/check-permissions.ts

# Build test
pnpm build
```

### Code Quality Standards

#### Go (Engine)

**Formatting**:

```bash
go fmt ./...
goimports -w .  # If available
```

**Style**:

- Tabs for indentation (gofmt standard)
- UpperCamelCase for exported symbols
- lowerCamelCase for package-private
- Doc comments for all exported functions
- Wrap errors with context
- No naked returns in complex functions

**Example**:

```go
// AnalyzeFiles processes migration files and returns dependency graph.
// It returns an error if parsing fails or dependencies cannot be resolved.
func AnalyzeFiles(files []string, config *Config) (*AnalysisResult, error) {
    if len(files) == 0 {
        return nil, fmt.Errorf("no files provided")
    }
    // ...
}
```

#### TypeScript (Platform)

**Formatting**: Biome (2 spaces, single quotes, 100-char lines)

**Style**:

- PascalCase for components: `AnalysisResults.tsx`
- camelCase for functions: `getUserProjects()`
- kebab-case for files: `api-client.ts`
- Named exports preferred (except pages/layouts)
- No `any` without JSDoc justification
- Prefer Server Components, use Client only when needed

**Example**:

```typescript
// components/dashboard/ProjectCard.tsx
import type { Project } from '@/types/database';

interface ProjectCardProps {
  project: Project;
  onAnalyze?: (projectId: string) => void;
}

export function ProjectCard({ project, onAnalyze }: ProjectCardProps) {
  // Component code
}
```

### Git Workflow

**Branch Strategy**:

- `main` - Production-ready code
- `refactor` - Major refactoring (current branch)
- `feature/*` - Feature branches
- `fix/*` - Bug fixes
- `docs/*` - Documentation updates

**Commit Format**: Conventional Commits

```bash
# Format: type(scope): description

feat: add GitHub App webhook handling
feat(platform): implement organization settings UI
fix: resolve dependency ordering in squasher
fix(api): correct CORS headers for engine endpoint
chore: update dependencies
chore(deps): bump next from 14.0.0 to 15.0.0
docs: improve API documentation
docs(engine): add validation workflow diagram
refactor: restructure validation module
refactor(db): migrate to Drizzle ORM
```

**Pull Request Template**:

```markdown
## Description
Clear description of what this PR does

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Related Issues
Closes #123

## Changes Made
- List of specific changes
- Migration steps if database changed
- New environment variables if added

## Testing
- [ ] Tests added/updated
- [ ] Manual testing performed
- [ ] Build succeeds

## Screenshots (if UI change)
[Add screenshots]

## Checklist
- [ ] Code follows style guidelines
- [ ] Self-review completed
- [ ] Documentation updated
- [ ] No breaking changes (or documented)
```

---

## Deployment Architecture

### Production Topology

```
┌─────────────────────────────────────────────────────────────────┐
│                      PRODUCTION ARCHITECTURE                     │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────────────┐            ┌──────────────────┐          │
│  │   Vercel Edge    │            │    Fly.io        │          │
│  │   (Next.js App)  │◄──────────▶│  (Go API Server) │          │
│  │                  │   HTTPS    │                  │          │
│  │ • Server Comps   │            │ • pgsquash API   │          │
│  │ • API Routes     │            │ • Webhooks       │          │
│  │ • Static Assets  │            │ • Health Checks  │          │
│  └────────┬─────────┘            └────────┬─────────┘          │
│           │                               │                     │
│           │        ┌──────────────────────┼──────┐              │
│           │        │                      │      │              │
│           ▼        ▼                      ▼      ▼              │
│     ┌─────────┐┌────────┐          ┌────────┐┌────────┐        │
│     │ Clerk   ││ Stripe │          │  Neon  ││Upstash │        │
│     │ (Auth)  ││ (Pay)  │          │ (PG)   ││(Redis) │        │
│     └─────────┘└────────┘          └────────┘└────────┘        │
│                                                                  │
│     ┌───────────────────────────────────────────────────┐       │
│     │        GitHub (Webhooks, App API, OAuth)          │       │
│     └───────────────────────────────────────────────────┘       │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘

Traffic Flow:
  Users → Vercel Edge → Next.js App
  Users → Vercel → API Route → Fly.io API
  GitHub → Webhook → Fly.io API → Vercel API → Update DB
```

### Deployment Scenarios

#### 1. Hosted SaaS (Current Production)

**Next.js Web Application**:

- **Platform**: Vercel
- **Domain**: capysquash.dev (and www)
- **Build**: Automatic on push to `main`
- **Environment**: Production build with edge caching
- **CDN**: Vercel Edge Network (global)
- **Regions**: Auto (Vercel optimizes)

**Go API Server**:

- **Platform**: Fly.io (or alternative: Railway, Render, AWS ECS)
- **Build**: Docker image from `pgsquash-engine/docker/api-server/Dockerfile`
- **Scaling**: Auto-scale based on CPU/memory
- **Regions**: Multi-region for low latency
- **Health Checks**: `/health` endpoint every 30s
- **Monitoring**: Fly.io metrics + optional DataDog

**Database**:

- **Provider**: Neon (Serverless PostgreSQL)
- **Plan**: Pro or Business
- **Connection**: Pooled via Neon proxy
- **Backups**: Automatic (Neon managed)
- **Migrations**: Run via Drizzle during CI/CD

**Caching**:

- **Provider**: Upstash (Serverless Redis)
- **Purpose**: Rate limiting, session storage, API caching
- **Fallback**: Application continues without Redis (graceful degradation)

**External Services**:

- **Auth**: Clerk (managed SaaS)
- **Payments**: Stripe (managed SaaS)
- **GitHub**: GitHub App (managed SaaS)
- **Monitoring**: Sentry (error tracking), Vercel Analytics

**Cost Estimate (Monthly)**:

- Vercel: $20-50 (Pro plan)
- Fly.io: $30-100 (based on load)
- Neon: $20-100 (based on usage)
- Upstash: $10-30 (based on requests)
- Clerk: $25-100 (based on MAU)
- Total: \~$105-380/month

#### 2. Enterprise On-Premise

**Deployment Method**: Docker Compose (production profile)

```bash
# On customer infrastructure
cd capysquash-platform
cp .env.example .env
# Edit .env with customer credentials

docker compose --profile production up -d
```

**Included Services**:

- Next.js web application (port 3000)
- Go API server (port 8080)
- PostgreSQL 17 database (port 5432)
- Redis cache (port 6379)
- Nginx reverse proxy (ports 80, 443)

**System Requirements**:

- **CPU**: 4+ cores
- **RAM**: 8GB+ (16GB recommended)
- **Disk**: 50GB+ SSD
- **OS**: Linux (Ubuntu 22.04+ or RHEL 8+)
- **Docker**: 20.10+ and Docker Compose 2.0+

**Network Requirements**:

- Outbound HTTPS (443) for external services (if not airgapped)
- SSL certificate for custom domain
- Firewall rules for internal access

**Customization Options**:

- White-label branding
- Custom domain
- SSO/SAML integration
- Airgapped mode (no external calls)
- Custom retention policies

**Support**:

- Dedicated Slack channel
- Video call support
- Quarterly reviews
- Emergency hotline

#### 3. Hybrid (Partner/Reseller)

**Use Case**: Partners host web app, use centralized API

**Configuration**:

- Web app: Partner-hosted (Vercel or their infrastructure)
- API server: Centrally hosted by pgsquash
- Database: Partner-managed (their Neon/PostgreSQL)
- Branding: White-label with partner logo/domain

**Partner Benefits**:

- Lower infrastructure costs
- Automatic API updates
- Shared security responsibility
- Revenue sharing model

**Partner Responsibilities**:

- Customer support (Tier 1)
- Branding and customization
- Database management
- Customer billing

### Environment Variables

#### Platform (Next.js) - Complete List

```bash
# ============================================================================
# Core Application
# ============================================================================
NODE_ENV=production
NEXT_PUBLIC_APP_URL=https://capysquash.dev

# ============================================================================
# Database
# ============================================================================
DATABASE_URL=postgresql://user:password@host/database?sslmode=require

# ============================================================================
# API Connection (pgsquash-engine)
# ============================================================================
GO_ENGINE_URL=https://api.capysquash.dev
CAPYSQUASH_API_SECRET=shared_secret_between_platform_and_engine

# ============================================================================
# Authentication (Clerk)
# ============================================================================
NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY=pk_live_xxxxxxxxxxxx
CLERK_SECRET_KEY=sk_live_xxxxxxxxxxxx
CLERK_WEBHOOK_SECRET=whsec_xxxxxxxxxxxx

# Clerk URLs (usually defaults are fine)
NEXT_PUBLIC_CLERK_SIGN_IN_URL=/sign-in
NEXT_PUBLIC_CLERK_SIGN_UP_URL=/sign-up
NEXT_PUBLIC_CLERK_SIGN_IN_FORCE_REDIRECT_URL=/dashboard
NEXT_PUBLIC_CLERK_SIGN_UP_FORCE_REDIRECT_URL=/dashboard

# ============================================================================
# Payments (Stripe)
# ============================================================================
STRIPE_SECRET_KEY=sk_live_xxxxxxxxxxxx
NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY=pk_live_xxxxxxxxxxxx
STRIPE_WEBHOOK_SECRET=whsec_xxxxxxxxxxxx

# Stripe Price IDs (from Stripe dashboard)
STRIPE_PRICE_ID_CREATOR=price_xxxxxxxxxxxx
STRIPE_PRICE_ID_PRO=price_xxxxxxxxxxxx
STRIPE_PRICE_ID_AGENCY=price_xxxxxxxxxxxx
STRIPE_PRICE_ID_ENTERPRISE=price_xxxxxxxxxxxx

# ============================================================================
# GitHub Integration
# ============================================================================
GITHUB_APP_ID=123456
GITHUB_APP_PRIVATE_KEY="-----BEGIN RSA PRIVATE KEY-----\n...\n-----END RSA PRIVATE KEY-----"
GITHUB_WEBHOOK_SECRET=webhook_secret_here
GITHUB_CLIENT_ID=Iv1.xxxxxxxxxxxx
GITHUB_CLIENT_SECRET=xxxxxxxxxxxx

# ============================================================================
# Redis (Optional - for rate limiting and caching)
# ============================================================================
UPSTASH_REDIS_URL=redis://default:password@host:port
UPSTASH_REDIS_TOKEN=token_here

# ============================================================================
# Monitoring (Optional)
# ============================================================================
SENTRY_DSN=https://xxxxxxxxxxxx@sentry.io/xxxxxxxxxxxx
NEXT_PUBLIC_VERCEL_ANALYTICS_ID=xxxxxxxxxxxx

# ============================================================================
# Feature Flags (Optional)
# ============================================================================
NEXT_PUBLIC_ENABLE_DEMO_MODE=true
NEXT_PUBLIC_ENABLE_AI_ANALYSIS=false
```

#### Engine (Go API Server) - Complete List

```bash
# ============================================================================
# Server Configuration
# ============================================================================
PORT=8080
LOG_LEVEL=info  # debug, info, warn, error
CORS_ORIGIN=https://capysquash.dev,https://www.capysquash.dev,http://localhost:3000

# ============================================================================
# API Security
# ============================================================================
CAPYSQUASH_API_SECRET=shared_secret_between_platform_and_engine

# ============================================================================
# GitHub Integration (if handling webhooks directly)
# ============================================================================
GITHUB_TOKEN=ghp_xxxxxxxxxxxx
GITHUB_WEBHOOK_SECRET=webhook_secret_here
GITHUB_APP_ID=123456
GITHUB_APP_PRIVATE_KEY="-----BEGIN RSA PRIVATE KEY-----\n...\n-----END RSA PRIVATE KEY-----"

# ============================================================================
# AI Providers (Optional - for AI analysis features)
# ============================================================================
OPENAI_API_KEY=sk-xxxxxxxxxxxx
ANTHROPIC_API_KEY=sk-ant-xxxxxxxxxxxx

# ============================================================================
# Docker Validation (usually auto-detected)
# ============================================================================
DOCKER_HOST=unix:///var/run/docker.sock
```

### CI/CD Pipelines

#### Platform Deployment (Vercel)

**Trigger**: Push to `main` branch

**Automatic Steps**:

1. Vercel detects git push
2. Install dependencies (`pnpm install`)
3. Run build checks
4. Type check (`tsc --noEmit`)
5. Build Next.js (`pnpm build`)
6. Deploy to Vercel Edge
7. Run smoke tests (if configured)
8. Update production URL

**Environment Variables**: Set in Vercel dashboard

**Preview Deployments**: Automatic for all branches

#### Engine Deployment (Fly.io)

**Trigger**: Manual or automated via GitHub Actions

**Manual Deployment**:

```bash
cd pgsquash-engine
fly deploy
```

**Automated via GitHub Actions** (example):

```yaml
# .github/workflows/deploy-api.yml
name: Deploy API to Fly.io

on:
  push:
    branches: [main]
    paths:
      - 'pgsquash-engine/**'

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: superfly/flyctl-actions/setup-flyctl@master
      - run: flyctl deploy --remote-only
        working-directory: pgsquash-engine
        env:
          FLY_API_TOKEN: ${{ secrets.FLY_API_TOKEN }}
```

**Health Checks**: Automatic via Fly.io health endpoint

#### Database Migrations

**Automatic in CI**:

```yaml
# .github/workflows/migrate-db.yml
name: Run Database Migrations

on:
  push:
    branches: [main]
    paths:
      - 'capysquash-platform/src/lib/db/schema.ts'
      - 'capysquash-platform/drizzle/**'

jobs:
  migrate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-node@v3
        with:
          node-version: '18'
      - run: corepack enable
      - run: pnpm install
      - run: pnpm db:migrate
        env:
          DATABASE_URL: ${{ secrets.DATABASE_URL }}
```

**Manual Migrations**:

```bash
cd capysquash-platform
pnpm db:migrate
```

---

_(Continuing in next message due to length...)_
