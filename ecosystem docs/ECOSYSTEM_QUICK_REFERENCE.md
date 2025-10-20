# PGSQUASH Ecosystem - Quick Reference

**Fast lookup guide for developers, product managers, and stakeholders**

Last Updated: October 20, 2025

---

## 📦 Component Matrix

| Component               | Tech Stack           | Purpose                               | Location                      | Business Role                 |
| ----------------------- | -------------------- | ------------------------------------- | ----------------------------- | ----------------------------- |
| **pgsquash-engine**     | Go 1.22              | CLI & API for migration consolidation | `/pgsquash-engine`            | Free OSS, awareness driver    |
| **capysquash-platform** | Next.js 15, React 19 | SaaS web app, team features           | `/capysquash-platform`        | Revenue source, $12-custom/mo |
| **capysquash-docs**     | Next.js + Fumadocs   | Public documentation                  | `/capysquash-docs`            | Activation, SEO, support      |
| **Business Docs**       | Markdown             | Strategy, positioning, GTM            | `/branding and business docs` | Strategic direction           |

---

## 🔗 Key URLs & Resources

### Production

- **Platform**: <https://capysquash.dev>
- **API**: <https://api.capysquash.dev> (Fly.io)
- **Docs**: <https://docs.capysquash.dev> (planned)
- **GitHub Org**: <https://github.com/CAPYSQUASH>
- **Engine Repo**: <https://github.com/CAPYSQUASH/pgsquash-engine>

### Development

- **Platform Local**: <http://localhost:3000>
- **API Local**: <http://localhost:8080>
- **Docs Local**: <http://localhost:3000>

---

## 💰 Pricing Tiers

| Tier             | Price  | Target      | Key Features                  |
| ---------------- | ------ | ----------- | ----------------------------- |
| **Free**         | $0     | Individuals | CLI unlimited, 3 repos on web |
| **Creator**      | $12/mo | Solo devs   | Unlimited repos, automation   |
| **Professional** | $29/mo | Teams (≤5)  | Collaboration, SSO            |
| **Agency**       | $99/mo | Dev shops   | 20 users, white-label         |
| **Enterprise**   | Custom | Large orgs  | On-premise, SAML, SLA         |

**Conversion Triggers**:

- Free → Creator: Hit 3-repo limit, want automation
- Creator → Pro: Add second team member
- Pro → Agency: Managing 5+ client projects
- Agency → Enterprise: Compliance, SSO needs

---

## 🏗️ Architecture at a Glance

```
Users (CLI, Web, Bot)
        ↓
┌───────────────────┬────────────────┐
│ pgsquash-engine   │ capysquash-    │
│ (Go API)          │ platform       │
│ Port 8080         │ (Next.js)      │
│                   │ Port 3000      │
└───────────────────┴────────────────┘
        ↓                  ↓
┌───────────────────────────────────┐
│ External Services                 │
│ • Clerk (auth)                    │
│ • Stripe (payments)               │
│ • GitHub (code + webhooks)        │
│ • Neon (PostgreSQL)               │
│ • Upstash (Redis)                 │
└───────────────────────────────────┘
```

---

## 🚀 Development Quick Start

### Engine (Go)

```bash
cd pgsquash-engine
go mod tidy
go build -o pgsquash cmd/pgsquash/main.go
./pgsquash analyze migrations/

# API server
go build -o api-server cmd/api-server/main.go
./api-server
```

### Platform (Next.js)

```bash
cd capysquash-platform
corepack enable
pnpm install
cp .env.example .env.local
# Edit .env.local
pnpm db:push
pnpm dev
```

### Full Stack (Docker)

```bash
# Option 1: API only (recommended)
cd capysquash-platform
docker compose --profile dev up -d
pnpm dev  # Separate terminal

# Option 2: Everything
docker compose --profile full-stack up -d
```

---

## 📡 API Endpoints Summary

### Engine API (Go - Port 8080)

```bash
POST /analyze      # Analyze migrations
POST /squash       # Consolidate migrations
POST /validate     # Validate equivalence
POST /github/webhook  # GitHub events
GET  /health       # Health check
```

**Auth**: `X-API-Secret` header

### Platform API (Next.js - /api/\*)

**56 endpoints total** organized as:

- `/api/engine/*` (2) - Engine proxy
- `/api/dashboard/*` (1) - Metrics
- `/api/organizations/*` (17) - Org management
- `/api/projects/*` (8) - Project CRUD
- `/api/users/*` (4) - User preferences
- `/api/stripe/*` (3) - Billing
- `/api/github/*` (3) - GitHub integration
- `/api/admin/*` (15) - Platform admin
- `/api/webhooks/*` (2) - External webhooks
- `/api/demo/*` (1) - Demo setup

**Auth**: Clerk JWT or API key

---

## 🗄️ Database Schema

**26 tables** across 8 categories:

- **Core** (6): organizations, users, projects, runs, files, memberships
- **Payments** (4): subscriptions, usage\_tracking, plans, stripe\_events
- **GitHub** (2): installations, webhook\_events
- **Settings** (3): org\_settings, user\_preferences, notification\_rules
- **Audit** (4): activities, comments, favorites, notifications
- **API** (2): api\_keys, database\_connections
- **Templates** (2): project\_templates, feature\_flags
- **Config** (1): subscription\_plans

**ORM**: Drizzle ORM
**Hosting**: Neon (Serverless PostgreSQL)
**Migrations**: `pnpm db:migrate`

---

## 🔐 Security & Auth

### Authentication Flow

```
User → Clerk Sign In → JWT → Platform → Verification → Authorized
```

### RBAC (3-Tier)

**Platform Roles**:

- `ADMIN` - Full platform access
- `USER` - Standard user

**Organization Roles**:

- `OWNER` - Full org control + billing
- `ADMIN` - Org management
- `MEMBER` - Project access
- `VIEWER` - Read-only

**Permissions**: Granular per-resource checks

### Inter-Service Auth

- **Platform → Engine**: Shared secret (`CAPYSQUASH_API_SECRET`)
- **GitHub → System**: Webhook signature verification
- **Stripe → System**: Webhook signature verification
- **Clerk → System**: JWT verification + webhook signatures

---

## 🎯 Business Model

### Revenue Streams

1. **Subscriptions** (Primary): $0 → $12 → $29 → $99 → Custom
2. **Usage Add-ons** (Secondary): Extra validations, large repos
3. **Enterprise Contracts** (Future): Multi-year, dedicated support

### Growth Loops

1. **PR Social Proof**: PRs show "Squashed by pgsquash" → Team sees → Some convert
2. **Agency Referral**: Agencies use for clients → Clients continue → Clients refer
3. **Community Showcase**: Users share results on Twitter → Others discover

### Revenue Targets

- **Year 1**: $100-150k ARR (163 customers)
- **Year 2**: $400-600k ARR (770 customers)

**Unit Economics**:

- CAC: <$50 organic, <$200 paid
- LTV: Creator $200, Pro $500, Agency $1,500, Enterprise $5,000+
- LTV:CAC: >3:1
- Gross Margin: >70%

---

## 🔌 Integration Priorities

### Current (Live)

✅ **GitHub App** - PR automation, webhooks
✅ **Clerk** - Authentication, orgs
✅ **Stripe** - Subscriptions, billing
✅ **Neon** - Database hosting
✅ **Vercel** - Web hosting

### Planned (By Priority)

1. **Supabase CLI Plugin** (High) - `supabase pgsquash analyze`
2. **Neon Dashboard Widget** (High) - One-click from Neon UI
3. **Vercel Marketplace** (Medium) - One-click install
4. **VS Code Extension** (Medium) - Command palette integration
5. **Drizzle/Prisma Plugins** (Medium) - Pattern recognition

---

## 📂 Repository Structure

```
pg-squash/                          # Monorepo root
├── pgsquash-engine/                # Go CLI & API
│   ├── cmd/pgsquash/               # CLI binary
│   ├── cmd/api-server/             # HTTP API
│   ├── internal/                   # Core modules
│   ├── docs/                       # Engine docs
│   └── AGENTS.md                   # Dev guidelines
│
├── capysquash-platform/            # Next.js SaaS
│   ├── src/app/                    # App Router
│   ├── src/components/             # React components
│   ├── src/lib/                    # Utilities + DB
│   ├── docs/                       # Platform docs
│   ├── scripts/                    # Maintenance
│   ├── docker-compose.yml          # Multi-profile deployment
│   └── AGENTS.md                   # Dev guidelines
│
├── capysquash-docs/                # Documentation site
│   ├── content/docs/               # MDX documentation
│   ├── content/blog/               # Blog posts
│   └── src/                        # Fumadocs app
│
├── branding and business docs/     # Strategy
│   ├── pgsquash-complete-strategy.md  # Full playbook
│   ├── capysquash-complete-brand-guide.md
│   └── pgsquash-integration-roadmap.md
│
├── docker-compose.yml              # Production API deployment
├── ECOSYSTEM_OVERVIEW.md           # Ecosystem guide
├── ECOSYSTEM_COMPLETE_GUIDE.md     # Detailed reference
└── ECOSYSTEM_QUICK_REFERENCE.md    # This file
```

---

## 🔧 Environment Variables (Essential)

### Platform (.env.local)

```bash
# Database
DATABASE_URL=postgresql://...?sslmode=require

# API Connection
GO_ENGINE_URL=http://localhost:8080
CAPYSQUASH_API_SECRET=shared_secret

# Clerk
NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY=pk_...
CLERK_SECRET_KEY=sk_...

# Stripe (optional for development)
STRIPE_SECRET_KEY=sk_...
NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY=pk_...

# GitHub (optional for development)
GITHUB_APP_ID=123456
GITHUB_APP_PRIVATE_KEY="-----BEGIN RSA..."
```

### Engine (API Server)

```bash
PORT=8080
LOG_LEVEL=info
CORS_ORIGIN=http://localhost:3000
CAPYSQUASH_API_SECRET=shared_secret

# Optional
GITHUB_TOKEN=ghp_...
OPENAI_API_KEY=sk-...
```

---

## 📊 Key Metrics Dashboard

### Product Metrics

- **North Star**: Weekly Active Projects
- **Activation**: Time to First Analysis <5min
- **Engagement**: Analyses per Active Project 2-4/month
- **Quality**: Validation Pass Rate >95%

### Business Metrics

- **MRR**: Monthly Recurring Revenue
- **Churn**: Target <5% monthly
- **CAC**: Customer Acquisition Cost
- **LTV**: Lifetime Value
- **NRR**: Net Revenue Retention >100%

---

## 🎭 Target Personas

### Primary: The Vibe Coder

**Profile**: Frontend dev, Next.js + Supabase, Cursor/v0.dev user, 0-2 years full-stack

**Pain**: "My deploys are getting slower, I have 147 migration files"

**Message**: "Keep building fast—we handle cleanup"

**Acquisition**: Twitter, Reddit, YouTube, Indie Hackers

### Secondary: The Agency Developer

**Profile**: Building 5-10 client projects/year, small team (2-5), needs professional polish

**Pain**: "Each client project accumulates mess, handoffs need to be clean"

**Message**: "Professional database hygiene for every client"

**Acquisition**: Agency communities, LinkedIn, webinars

### Expansion: The Startup Team

**Profile**: 3-10 engineers, raised seed funding, rapid growth, DevOps emerging

**Pain**: "Our CI takes forever, onboarding is painful, scared to touch DB"

**Message**: "Professional database operations without overhead"

**Acquisition**: YC networks, startup communities, case studies

---

## 🛠️ Common Commands

### Engine Development

```bash
# Build
go build -o pgsquash cmd/pgsquash/main.go
go build -o api-server cmd/api-server/main.go

# Test
go test ./...
go test ./... -cover
go test ./... -race

# Run
./pgsquash analyze migrations/
./api-server  # Port 8080

# Validation
./scripts/validate.sh --mode full --migrations test/
```

### Platform Development

```bash
# Install
corepack enable && pnpm install

# Database
pnpm db:push       # Development
pnpm db:generate   # Create migration
pnpm db:migrate    # Run migration
pnpm db:studio     # GUI

# Development
pnpm dev           # Next.js dev server
pnpm build         # Production build
pnpm start         # Production server

# Code Quality
pnpm lint          # Biome linting
pnpm format        # Format code
pnpm check:fix     # Fix all issues

# Testing
pnpm tsx scripts/test-apis.ts
pnpm tsc --noEmit  # Type check
```

### Docker Deployment

```bash
# Development (API only)
docker compose --profile dev up -d

# Full stack
docker compose --profile full-stack up -d

# Production
docker compose --profile production up -d

# Logs
docker compose logs -f api-server
docker compose logs -f webapp

# Clean up
docker compose down
docker compose down -v  # Remove volumes
```

---

## 📝 Deployment Checklist

### Platform (Vercel)

- [ ] Set all environment variables in Vercel dashboard
- [ ] Verify custom domain DNS
- [ ] Enable Vercel Analytics (optional)
- [ ] Configure build settings (auto-detected)
- [ ] Test preview deployment

### Engine (Fly.io)

- [ ] Install Fly CLI: `curl -L https://fly.io/install.sh | sh`
- [ ] Login: `flyctl auth login`
- [ ] Create app: `flyctl apps create pgsquash-api`
- [ ] Set secrets: `flyctl secrets set KEY=value`
- [ ] Deploy: `flyctl deploy`
- [ ] Monitor: `flyctl logs`

### Database (Neon)

- [ ] Create Neon project
- [ ] Copy connection string (pooled)
- [ ] Add `?sslmode=require` to connection string
- [ ] Run migrations: `pnpm db:migrate`
- [ ] Verify tables created

---

## 🆘 Troubleshooting

### Platform won't start

```bash
# Check environment variables
cat .env.local

# Verify database connection
pnpm tsx -e "import { db } from './src/lib/db'; console.log(await db.execute('SELECT 1'))"

# Clear Next.js cache
rm -rf .next
pnpm dev
```

### Engine API unreachable

```bash
# Check if running
curl http://localhost:8080/health

# Check logs
docker compose logs api-server

# Restart
docker compose restart api-server
```

### Database migration issues

```bash
# Check current schema
pnpm db:studio

# Reset database (DEV ONLY!)
pnpm db:push --force

# Create new migration
pnpm db:generate
pnpm db:migrate
```

### Build failures

```bash
# Clear dependencies
rm -rf node_modules pnpm-lock.yaml
pnpm install

# Type check
pnpm tsc --noEmit

# Fix linting
pnpm check:fix

# Try build
pnpm build
```

---

## 📚 Documentation Index

### Detailed Guides

- **[ECOSYSTEM\_OVERVIEW.md](./ECOSYSTEM_OVERVIEW.md)** - Original ecosystem overview
- **[ECOSYSTEM\_COMPLETE\_GUIDE.md](./ECOSYSTEM_COMPLETE_GUIDE.md)** - Comprehensive technical guide
- **[pg-squash-engine-e2e-task.md](./pg-squash-engine-e2e-task.md)** - E2E implementation task

### Component-Specific Docs

**Engine**:

- `pgsquash-engine/README.md` - Engine overview
- `pgsquash-engine/docs/` - User documentation
- `pgsquash-engine/AGENTS.md` - Development guidelines
- `pgsquash-engine/CHANGELOG.md` - Release notes

**Platform**:

- `capysquash-platform/README.md` - Platform overview
- `capysquash-platform/docs/API_REFERENCE.md` - All 56 endpoints
- `capysquash-platform/docs/DATABASE_SCHEMA.md` - All 26 tables
- `capysquash-platform/docs/QUICK_START.md` - Setup guide
- `capysquash-platform/docs/RBAC_GUIDE.md` - Permission system
- `capysquash-platform/AGENTS.md` - Development guidelines

**Business**:

- `branding and business docs/pgsquash-complete-strategy.md` - Complete playbook (2,016 lines)
- `branding and business docs/capysquash-complete-brand-guide.md` - Brand identity
- `branding and business docs/pgsquash-integration-roadmap.md` - Partnership strategy
- `branding and business docs/PRODUCT_ROADMAP_AND_GROWTH_STRATEGY.md` - Product/growth plan

---

## 🎯 Next Steps by Role

### For Developers

1. Read `ECOSYSTEM_OVERVIEW.md` for big picture
2. Set up local development (see Development Quick Start above)
3. Read component-specific AGENTS.md
4. Pick an issue and submit PR

### For Product Managers

1. Review pricing tiers and business model above
2. Read `pgsquash-complete-strategy.md` for full context
3. Check integration priorities
4. Review feature roadmap in business docs

### For Designers

1. Read brand guide in business docs
2. Review platform UI in `/capysquash-platform/src/components`
3. Check Figma files (if available)
4. Consult with team on design system

### For Marketers

1. Read complete strategy document
2. Review target personas above
3. Check growth loops and acquisition channels
4. Plan content calendar

### For Sales

1. Review pricing tiers and positioning
2. Read enterprise features in platform docs
3. Check case studies (when available)
4. Set up demo environment

---

## 🔗 Quick Links

- **Main Repo**: <https://github.com/CAPYSQUASH/pgsquash-engine>
- **Production**: <https://capysquash.dev>
- **Platform Docs**: `capysquash-platform/docs/`
- **Engine Docs**: `pgsquash-engine/docs/`
- **Strategy**: `branding and business docs/pgsquash-complete-strategy.md`

---

**Questions?** Check the detailed guides or ask in the team chat.

**Last Updated**: October 20, 2025
