# Changelog

All notable changes to pg-squash Engine will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

> **⚠️ Important Notice**
>
> This tool modifies SQL migration files. Always maintain backups of your original migrations before running consolidation operations. While pg-squash includes validation and safety checks, you should verify the output matches your expectations. Test consolidated migrations in a non-production environment first.

---

## [Unreleased]

## [0.8.1-beta] - 2025-10-06

### Added

#### Core Engine
- PostgreSQL parser integration using `pg_query_go/v6` for 100% accurate SQL parsing
- Multi-phase processing pipeline (Parse → Track → Analyze → Consolidate → Generate)
- Four safety levels: Paranoid, Conservative, Standard, and Aggressive
- Dependency-aware consolidation with automatic topological sorting
- Object lifecycle tracking across migrations

#### Plugin System
- Auto-discovery plugin architecture with priority-based execution
- **Clerk Plugin** (Priority: 95) - JWT v2 organization claims, helper function volatility markers
- **Supabase Plugin** (Priority: 90) - Auth schema detection, RLS policy consolidation, storage bucket policies
- **Prisma Plugin** (Priority: 75) - Migration table preservation, enum protection, VARCHAR(191) optimization
- **Drizzle Plugin** (Priority: 75) - IDENTITY column support, generated column preservation, modern SQL patterns
- Plugin extensibility with simple registration API

#### Validation System
- Three validation modes:
  - `TWO_CONTAINERS` - Most accurate (separate containers)
  - `TWO_DATABASES` - Best balance (shared container)
  - `SCHEMA_DIFF` - Fastest (SQL diff comparison)
- Automatic PostgreSQL extension detection and installation
- Docker-based schema validation with isolation guarantees

#### AI Integration
- Multi-provider support (Claude/Anthropic, OpenAI, Azure OpenAI)
- Semantic function analysis and equivalence detection
- Dead code identification
- Authentication pattern recognition
- Performance optimization suggestions
- Optional opt-in features (requires API keys)

#### Performance
- Streaming mode for large migration sets (500+ files)
- Memory-efficient batch processing
- Configurable worker pools
- Progress tracking and throughput monitoring
- Automatic batch size calculation

#### Transformation System
- Function volatility marker injection (IMMUTABLE/STABLE/VOLATILE)
- Backup generation (schema-only, data-only, full)
- Rollback script generation
- SQL modernization (SERIAL → IDENTITY conversion)
- Auth pattern standardization

#### GitHub Integration
- Automated PR analysis via webhooks
- Bot commands: `/pgsquash analyze`, `/pgsquash consolidate`
- Auto-consolidation when migration threshold met
- Configurable via `.github/pgsquash.yml`
- API server for webhook handling

#### Standardized Workflows
- `safe` command - Production workflow (conservative, full validation, backups)
- `fast` command - Development workflow (balanced optimization, quick validation)
- `analyze-deep` command - Deep analysis without modifications

#### CLI Commands
- `analyze` - Analyze migrations without modifications
- `squash` - Consolidate migrations with configurable safety
- `validate` - Validate original vs squashed schemas
- `init-config` - Generate default configuration
- `ai-test` - Test AI provider connectivity
- `ai-demo` - Demonstrate AI capabilities
- `health` - Health check endpoint
- `version` - Display version information

#### Configuration
- Comprehensive JSON configuration system
- Auto-generation via `init-config`
- Per-plugin configuration options
- Performance tuning parameters
- Modern PostgreSQL feature toggles
- Third-party integration settings

### Documentation
- Comprehensive README with quick start guide
- Detailed CLI reference documentation
- Safety levels guide with use case recommendations
- Configuration reference with all options
- Architecture documentation with system design
- AI features guide
- GitHub integration setup guide
- Troubleshooting guide
- Docker deployment documentation
- Production deployment guide

### Changed
- Module renamed from initial structure to `github.com/capysquash/pg-squash-engine`
- Project reorganized following standard Go project layout
- Documentation restructured (public docs in `docs/`, internal in `docs/internal/`)

### Known Limitations
- No automated test coverage (manual testing only)
- Plugin system requires integration tests
- AI features require external API keys (with associated costs)
- Full validation requires Docker installation
- Some PostgreSQL extensions not available on all platforms

### Technical Details
- **Language**: Go 1.25.1+
- **PostgreSQL Support**: Versions 12-17
- **Key Dependencies**:
  - `github.com/pganalyze/pg_query_go/v6` - PostgreSQL parser
  - `github.com/spf13/cobra` - CLI framework
  - `github.com/anthropics/anthropic-sdk-go` - Claude API
  - `github.com/docker/docker` - Container validation
  - `github.com/fatih/color` - Terminal output
  - `github.com/lib/pq` - PostgreSQL driver

## Roadmap

### Phase 1 (Week 1)
- Test coverage >60%
- Example projects
- CI test enforcement

### Phase 2 (Week 2)
- Performance benchmarks
- Security audit
- Multi-platform binaries

### Phase 3 (Week 3-4)
- Additional auth plugins (Auth0, NextAuth)
- Platform plugins (Neon, Railway)
- PostgreSQL 18 support
- 1.0.0 release

[unreleased]: https://github.com/capysquash/pg-squash-engine/compare/v0.8.1-beta...HEAD
[0.8.1-beta]: https://github.com/capysquash/pg-squash-engine/releases/tag/v0.8.1-beta
