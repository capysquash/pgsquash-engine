# pg-squash Engine v0.8.0-beta - Beta Release

**Release Date**: October 2025
**Type**: Beta Release
**Status**: Feature-Complete, In Testing

---

## 🎉 Introducing pg-squash Engine

We're excited to announce the first public release of **pg-squash Engine** - the open-source PostgreSQL migration optimization engine that consolidates multiple migration files into clean, organized, and optimized migrations while preserving data integrity and dependency order.

pg-squash Engine is the core engine that powers CapySquash and other migration management tools, built with production PostgreSQL workloads in mind.

---

## ⭐ Highlights

### 🔍 Smart SQL Parsing

Uses PostgreSQL's actual parser (pg\_query\_go v6) for **100% accurate SQL analysis** - not regex-based approximations. Every statement is parsed using the same parser PostgreSQL itself uses.

### 🧠 Intelligent Plugin Architecture

Auto-detects and optimizes for popular frameworks:

- **Clerk** - JWT v2 organization claims
- **Supabase** - Auth schema, RLS policies, storage
- **Prisma** - Migration metadata, shadow databases
- **Drizzle** - IDENTITY columns, modern PostgreSQL patterns

### 🎯 Multi-Level Safety System

Choose your optimization level:

- **Conservative** - Maximum safety, minimal changes
- **Standard** - Balanced optimization (recommended)
- **Aggressive** - Maximum optimization
- **Paranoid** - Ultra-safe mode

### 🚀 Production-Ready Features

- Docker-based schema validation
- AI-powered semantic analysis
- Streaming mode for large datasets
- GitHub integration for automated workflows
- Comprehensive error recovery

---

## 🆕 What's New in v0.8.0-beta

### Core Engine

#### PostgreSQL Parser Integration

- Full integration with `pg_query_go/v6` (libpg\_query)
- Accurate parsing of complex PostgreSQL syntax
- Support for PostgreSQL 12-17 features
- Zero regex-based parsing approximations

#### Multi-Phase Processing Pipeline

1. **Parsing** - Extract statements and metadata
2. **Tracking** - Build dependency graphs and object lifecycles
3. **Analysis** - Detect redundancies and optimization opportunities
4. **Consolidation** - Apply safety-appropriate rules
5. **Generation** - Output organized, optimized migrations

#### Safety Levels

```bash
# Conservative - Production deployments
pgsquash squash migrations/*.sql --safety conservative

# Standard - Development workflows (default)
pgsquash squash migrations/*.sql --safety standard

# Aggressive - Maximum optimization
pgsquash squash migrations/*.sql --safety aggressive
```

### Plugin System

#### Auto-Discovery Architecture

Plugins automatically detect applicable frameworks from migration patterns:

```go
// No configuration required
pgsquash squash migrations/*.sql
// [plugins] Detected: prisma, clerk
```

#### 4 Production-Ready Plugins

**Authentication Plugins**:

- **Clerk** (Priority: 95)
  - JWT v2 organization claims detection
  - Helper function volatility markers
  - Mock auth layers for validation

- **Supabase** (Priority: 90)
  - `auth.uid()` pattern detection
  - RLS policy consolidation
  - Storage bucket policies

**ORM Plugins**:

- **Prisma** (Priority: 75)
  - `_prisma_migrations` table preservation
  - Enum protection (TypeScript mapping)
  - Shadow database support
  - VARCHAR(191) optimization

- **Drizzle** (Priority: 75)
  - IDENTITY column support (PostgreSQL 14+)
  - Generated column preservation
  - Sequence optimization
  - Modern SQL best practices

#### Extensibility

Add custom plugins with 3 files + 1 registration line:

```go
// internal/plugins/myservice/myservice.go
type MyServicePlugin struct { ... }

// cmd/pgsquash/main.go
plugins.Register(myservice.NewMyServicePlugin())
```

### Validation System

#### Three Validation Approaches

```bash
# TWO_CONTAINERS - Most accurate (separate containers)
pgsquash validate original/ squashed/ --mode two-containers

# TWO_DATABASES - Best balance (shared container)
pgsquash validate original/ squashed/ --mode two-databases

# SCHEMA_DIFF - Fastest (SQL diff comparison)
pgsquash validate original/ squashed/ --mode schema-diff
```

#### Automatic Extension Detection

Auto-detects and installs required PostgreSQL extensions:

- pgcrypto, pg\_stat\_statements
- vector (pgvector)
- PostGIS, TimescaleDB
- Custom extensions

### AI Integration

#### Multi-Provider Support

- **Claude** (Anthropic) - claude-4-5-sonnet
- **OpenAI** - gpt-5
- **Azure OpenAI** - Enterprise deployments

#### AI-Powered Features

```bash
# Semantic function analysis
pgsquash ai-demo --function-analysis

# Dead code detection
pgsquash analyze migrations/*.sql --ai-detect-dead-code

# Deep analysis workflow
pgsquash analyze-deep migrations/*.sql
```

**Capabilities**:

- Function semantic equivalence detection
- Dead code identification
- Authentication pattern recognition
- Performance optimization suggestions
- Complexity analysis

### Performance Optimizations

#### Streaming Mode

Automatically enabled for large migration sets (>100 files):

```bash
# Manual control
pgsquash squash migrations/*.sql --streaming --memory-limit 256
```

**Features**:

- Memory-efficient batch processing
- Configurable worker pools
- Progress tracking
- Throughput monitoring

#### Memory Management

- Configurable memory limits
- Automatic batch size calculation
- Incremental processing
- Resource cleanup

### Transformation System

#### Function Volatility Markers

Automatically adds IMMUTABLE/STABLE/VOLATILE markers to functions:

```sql
-- Before
CREATE FUNCTION clerk_user_id() RETURNS TEXT AS $$ ... $$;

-- After
CREATE FUNCTION clerk_user_id() RETURNS TEXT STABLE AS $$ ... $$;
```

Fixes: `ERROR: functions in index predicate must be marked IMMUTABLE`

#### Backup & Rollback Generation

```bash
# Generate backups before squashing
pgsquash squash migrations/*.sql --backup --backup-type schema-only

# Generate rollback scripts
pgsquash squash migrations/*.sql --generate-rollback
```

#### SQL Modernization

- Legacy SERIAL → IDENTITY conversion
- Verbose sequence → simplified syntax
- Auth pattern standardization

### GitHub Integration

#### Automated PR Analysis

```yaml
# .github/pgsquash.yml
auto_analyze: true
migration_threshold: 15
safety_level: standard
```

**Features**:

- Automatic migration analysis on PR creation
- Bot commands: `/pgsquash analyze`, `/pgsquash consolidate`
- Auto-consolidation when threshold met
- Real-time feedback as PR comments

#### API Server

```bash
# Deploy webhook server
go run docker/api-server/main.go
```

**Capabilities**:

- Webhook handling
- OAuth integration
- PR comment posting
- Status checks

### Standardized Workflows

#### Three Pre-Configured Commands

**SAFE Workflow** (Production):

```bash
pgsquash safe migrations/*.sql
```

- Conservative safety level
- TWO\_CONTAINERS validation
- Backup and rollback generation
- AI safety validation (if available)

**FAST Workflow** (Development):

```bash
pgsquash fast migrations/*.sql
```

- Standard safety level
- SCHEMA\_DIFF validation
- Streaming mode enabled
- DDL cycle detection

**ANALYZE Workflow** (Deep Dive):

```bash
pgsquash analyze-deep migrations/*.sql
```

- AI-powered semantic analysis
- Dead code detection
- Performance suggestions
- No file modifications

### Configuration System

#### Comprehensive Config Schema

```json
{
  "safety_level": "standard",
  "output": {
    "format": "organized",
    "preserve_comments": true,
    "directory": "squashed"
  },
  "plugins": {
    "auto_detect": true,
    "enabled_plugins": [],
    "disabled_plugins": []
  },
  "modern_features": {
    "vector_indexes": true,
    "generated_columns": true,
    "identity_columns": true
  },
  "third_party_integrations": {
    "clerk": { "jwt_version": "v2" },
    "supabase": { "preserve_auth_schema": true },
    "prisma": { "preserve_migration_table": true },
    "drizzle": { "prefer_identity_columns": true }
  },
  "performance": {
    "streaming_threshold_mb": 50,
    "memory_limit_mb": 512,
    "batch_size": 100,
    "workers": 4
  }
}
```

#### Auto-Generation

```bash
pgsquash init-config
# Creates: pgsquash.config.json
```

---

## 📦 Installation

### From Source

```bash
git clone https://github.com/capysquash/pg-squash-engine
cd pg-squash
go build -o pgsquash cmd/pgsquash/main.go
```

### Go Install

```bash
go install github.com/capysquash/pg-squash-engine/cmd/pgsquash@latest
```

### Docker

```bash
docker pull ghcr.io/capysquash/pg-squash:v1.0.0
```

---

## 🚀 Quick Start

### Basic Usage

```bash
# Analyze migrations
pgsquash analyze migrations/*.sql

# Squash with default settings (standard safety)
pgsquash squash migrations/*.sql --output clean_migrations/

# Validate results
pgsquash validate migrations/ clean_migrations/
```

### With Plugins (Auto-Detected)

```bash
# Prisma project
pgsquash squash prisma/migrations/*/migration.sql

# Drizzle project
pgsquash squash drizzle/*/migration.sql

# Supabase project
pgsquash squash supabase/migrations/*.sql
```

### Production Workflow

```bash
# Use SAFE workflow for production
pgsquash safe migrations/*.sql --output production/
```

---

## 📊 Technical Specifications

### Supported PostgreSQL Versions

- PostgreSQL 12, 13, 14, 15, 16, 17
- Target version configurable via `postgresql_features.target_version`

### Dependencies

```
github.com/pganalyze/pg_query_go/v6  - PostgreSQL parser
github.com/spf13/cobra               - CLI framework
github.com/anthropics/anthropic-sdk-go - Claude integration
github.com/openai/openai-go          - OpenAI integration
github.com/docker/docker             - Validation containers
github.com/fatih/color               - Terminal output
github.com/lib/pq                    - PostgreSQL driver
```

### Language Requirements

- Go 1.25.1+
- Docker (for validation)

---

## 🏗️ Architecture

### Component Overview

```
internal/
├── cli/              Command-line interface
├── parser/           SQL parsing (pg_query_go)
├── tracking/         Object lifecycle tracking
├── squasher/         Consolidation engine
├── plugins/          Plugin system (4 built-in)
├── validation/       Schema validation (Docker)
├── ai/               AI provider integration
├── transformation/   SQL transformations
├── performance/      Streaming & optimization
├── config/           Configuration management
└── types/            Shared type definitions
```

### Processing Phases

1. **Parse** - Extract statements using pg\_query\_go
2. **Track** - Build dependency graphs
3. **Analyze** - Detect redundancies
4. **Consolidate** - Apply rules based on safety level
5. **Generate** - Output organized migrations

### Output Structure

```
squashed/
├── 001_schema_foundation.sql          # Tables, types, domains
├── 002_constraints_relationships.sql  # Foreign keys
├── 003_indexes_performance.sql        # Indexes
├── 004_functions_procedures.sql       # Functions
├── 005_triggers_rules.sql             # Triggers
├── 006_permissions_security.sql       # RLS, grants
├── 007_data_migrations.sql            # Data operations
├── 008_extensions.sql                 # Extensions
├── migration_report.json              # Analysis report
└── README.md                          # Usage guide
```

---

## 📚 Documentation

Comprehensive documentation available in `/docs`:

- **[Getting Started Guide](../GETTING_STARTED.md)** - Installation and first steps
- **[User Guide](../USER_GUIDE.md)** - Complete CLI reference
- **[Architecture](../ARCHITECTURE.md)** - System design and components
- **[Configuration Reference](../CONFIGURATION.md)** - All config options
- **[Safety Levels Guide](../SAFETY_LEVELS.md)** - Choosing safety levels
- **[AI Integration Guide](../AI_INTEGRATION.md)** - AI-powered features
- **[GitHub Integration](../GITHUB_INTEGRATION.md)** - GitHub App setup
- **[Troubleshooting](../TROUBLESHOOTING.md)** - Common issues

---

## 🎯 Use Cases

### Development Teams

- Consolidate accumulated migrations before releases
- Clean up migration history after rapid prototyping
- Optimize database deployment times
- Maintain clean migration history

### DevOps/SRE

- Reduce database initialization time
- Minimize migration-related deployment failures
- Validate schema consistency across environments
- Automate migration optimization in CI/CD

### Framework Users

- **Prisma** - Optimize shadow database migrations
- **Drizzle** - Leverage modern PostgreSQL features
- **Supabase** - Preserve auth schema integrity
- **Clerk** - Maintain JWT v2 compatibility

---

## 🔐 Security & Compliance

### Data Safety

- ✅ No data is sent to external services (except AI features, opt-in)
- ✅ Schema validation in isolated Docker containers
- ✅ Automatic backup generation before consolidation
- ✅ Rollback script generation
- ✅ Dry-run mode for preview

### AI Privacy

- AI features are **opt-in** (require API keys)
- Only SQL code is analyzed (no data)
- Supports self-hosted models (Azure OpenAI)
- Can be fully disabled

---

## 🐛 Known Limitations

### v0.8.0-beta Limitations

1. **Testing Coverage**
   - Plugin system has manual testing only
   - Integration tests in development
   - Performance benchmarks pending

2. **Plugin Ecosystem**
   - 4 plugins included (Clerk, Supabase, Prisma, Drizzle)
   - Additional auth plugins planned (Auth0, NextAuth)
   - Platform plugins planned (Neon, Railway)

3. **AI Features**
   - Requires external API keys
   - API costs apply
   - Not recommended for critical production decisions

4. **Validation**
   - Requires Docker for full validation
   - Extension installation requires network access
   - Some extensions not available on all platforms

---

## 🗺️ Roadmap

### Phase 4 - Additional Auth (Planned)

- Auth0 plugin (enterprise RBAC)
- NextAuth plugin (Next.js ecosystem)
- Firebase Auth plugin (Google ecosystem)

**Estimated**: 1-2 weeks

### Phase 5 - Platform Plugins (Planned)

- Neon plugin (serverless PostgreSQL)
- Railway plugin (multi-environment)
- PlanetScale plugin (MySQL compatibility)

**Estimated**: 2-3 weeks

### Community Features

- Plugin marketplace
- Community plugin template
- Plugin version management
- Trust/security scoring

### Quality Improvements

- Comprehensive test suite
- Performance benchmarks
- CI/CD integration
- Error reporting system

---

## 🤝 Contributing

We welcome contributions! Here's how to get started:

### Adding a Plugin

```bash
# 1. Create plugin package
internal/plugins/yourservice/yourservice.go

# 2. Implement Plugin interface
type YourServicePlugin struct { ... }

# 3. Register in main.go
plugins.Register(yourservice.NewYourServicePlugin())

# 4. Add tests (recommended)
internal/plugins/yourservice/yourservice_test.go
```

### Development Setup

```bash
git clone https://github.com/capysquash/pg-squash-engine
cd pg-squash
go mod tidy
go build -o pgsquash cmd/pgsquash/main.go
go test ./...
```

### Contribution Guidelines

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

---

## 📄 License

MIT License - see [LICENSE](../../LICENSE) file for details.

---

## 🙏 Acknowledgments

### Technologies

- **PostgreSQL** - The world's most advanced open source database
- **pg\_query** - Ruby & Go PostgreSQL parser (libpg\_query)
- **pganalyze** - pg\_query\_go maintainers
- **Cobra** - CLI framework
- **Docker** - Container platform

### Frameworks Supported

- **Prisma** - Next-generation ORM
- **Drizzle** - TypeScript ORM with zero runtime overhead
- **Supabase** - Open source Firebase alternative
- **Clerk** - User management and authentication

### AI Providers

- **Anthropic** - Claude API
- **OpenAI** - GPT models
- **Microsoft** - Azure OpenAI Service

---

## Support & Community

- **Documentation**: Comprehensive guides in `/docs`
- **Issues**: [GitHub Issues](https://github.com/capysquash/pg-squash-engine/issues)
- **Discussions**: [GitHub Discussions](https://github.com/capysquash/pg-squash-engine/discussions)
- **Examples**: Sample configurations in `/examples`

---

## 📈 Release Statistics

### Code Metrics

- **Total Lines**: \~15,000+ lines of Go code
- **Packages**: 12 internal packages
- **Plugins**: 4 production-ready plugins
- **Documentation**: 2,500+ lines across 8 comprehensive guides
- **Test Coverage**: Manual validation (automated tests in development)

### Development Timeline

- **Initial Planning**: August 2024
- **Core Engine**: September 2024
- **Plugin System**: October 2024 (Phases 1-3)
- **Release**: October 2025

### Contributors

- Core Team: Dominikos Pritis  and contributors
- Community: Open for contributions

---

## 🎉 What's Next?

### Get Started

```bash
# Install
go install github.com/capysquash/pg-squash-engine/cmd/pgsquash@latest

# Try it
pgsquash analyze your-migrations/*.sql

# Optimize
pgsquash squash your-migrations/*.sql --output optimized/
```

### Join the Community

- ⭐ Star the repository
- 🐛 Report issues
- 💡 Suggest features
- 🤝 Contribute plugins
- 📢 Share your experience

---

**Thank you for using pg-squash Engine!**

_Building the future of PostgreSQL migration management, one squash at a time._ 🐘✨
