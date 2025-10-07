# Docker Infrastructure

Complete Docker setup for pg-squash with three distinct use cases.

## Directory Structure

```
/Dockerfile                       # ← Shared by all use cases
/docker-compose.yml               # ← Complete development environment

docker/
├── README.md                     # This file
├── DOCKERFILE_USAGE.md           # How Dockerfile is used across use cases
│
├── engine/                       # Use Case 1: Containerized CLI
│   ├── quick-start.yml
│   ├── examples.sh
│   └── README.md
│
├── validation/                   # Use Case 2: Docker Validation
│   ├── with-validation.yml
│   ├── init-scripts/
│   ├── validation/
│   └── README.md
│
├── web-app/                      # Use Case 3: Full-Stack Web App
│   ├── monolithic/
│   ├── separated/
│   ├── hybrid/
│   └── README.md
│
├── dev-environment/              # Complete development setup
│   ├── full-stack.yml           # Simplified dev compose
│   └── README.md                # Points to root docker-compose.yml
│
├── scripts/                      # Helper scripts
│   ├── build.sh                 # Build Docker images
│   ├── quick-validate.sh        # ⭐ Fast validation
│   ├── validate.sh              # Full validation
│   ├── setup-validation.sh      # Setup validation
│   ├── multi-version-test.sh    # Multi-version testing
│   ├── cleanup.sh               # Cleanup containers
│   └── test-setup.sh            # Test setup
│
├── config-templates/             # Configuration templates
│   ├── pgsquash.config.json.template
│   ├── .env.template
│   └── deployment-configs/
│
├── init-scripts/                 # Shared initialization
│   ├── supabase-compat.sql
│   ├── init-postgres.sql
│   └── validation-init.sql
│
└── entrypoint.sh                 # Container entrypoint
```

## Root Files

### `/Dockerfile`
Main Dockerfile used by **all three use cases** (engine, validation, dev).

- **Engine**: CLI-only container
- **Validation**: + Docker CLI for validation
- **Dev**: Full development setup

See [DOCKERFILE_USAGE.md](../docs/docker/DOCKERFILE_USAGE.md) for details.

### `/docker-compose.yml`
Complete **development environment** with all services:
- pg-squash + Docker socket
- PostgreSQL 17, 15, 13 (multi-version)
- Redis, MinIO, Grafana, Prometheus, Traefik, pgAdmin

See [dev-environment/README.md](dev-environment/README.md) for usage.

## Quick Start

### Engine (CLI in Docker)

```bash
docker compose -f docker/engine/quick-start.yml run --rm pgsquash squash
```

### Validation (Docker-in-Docker)

```bash
./docker/scripts/quick-validate.sh
```

### Web App (Full Stack)

```bash
cd docker/web-app/monolithic
docker compose up -d
```

### Development Environment

```bash
# From project root
docker compose up -d
```

## Three Use Cases

### 1. Engine: Containerized CLI

Run pg-squash CLI in a container without installing Go.

**Directory**: `docker/engine/`

```bash
# Squash migrations
docker compose -f docker/engine/quick-start.yml run --rm pgsquash squash /app/migrations/*.sql

# Analyze migrations
docker compose -f docker/engine/quick-start.yml run --rm pgsquash analyze /app/migrations/*.sql

# Interactive shell
docker compose -f docker/engine/quick-start.yml run --rm pgsquash bash
```

**Documentation**: [engine/README.md](engine/README.md)

---

### 2. Validation: Docker Validation Containers

Spin up ephemeral PostgreSQL containers to validate migrations.

**Directory**: `docker/validation/`

```bash
# Quick validation (recommended)
./docker/scripts/quick-validate.sh

# Full validation with Docker Compose
docker compose -f docker/validation/with-validation.yml run --rm pgsquash validate

# Multi-version testing
./docker/scripts/multi-version-test.sh
```

**Documentation**: [validation/README.md](validation/README.md)

---

### 3. Web App: Full-Stack Web Application

Complete web application with Next.js frontend and Go API backend.

**Directory**: `docker/web-app/`

**Deployment Scenarios**:

#### Monolithic (Simple)
```bash
cd docker/web-app/monolithic
docker compose up -d
# Access at http://localhost:3000
```

#### Separated (Production)
```bash
cd docker/web-app/separated
docker compose up -d api-server
# + Deploy frontend to Vercel
```

#### Hybrid (Development)
```bash
cd docker/web-app/hybrid
# See README for hybrid setup
```

**Documentation**: [web-app/README.md](web-app/README.md)

---

### 4. Development Environment

Complete development environment with all services.

**File**: `/docker-compose.yml` (root)

```bash
# Start core services
docker compose up -d

# With monitoring
docker compose --profile monitoring up -d

# With all tools
docker compose --profile monitoring --profile management up -d
```

**Documentation**: [dev-environment/README.md](dev-environment/README.md)

## Scripts Reference

### Build Scripts

#### `build.sh`
Build and publish Docker images with multi-platform support.

```bash
# Build locally
./docker/scripts/build.sh

# Build and push to registry
./docker/scripts/build.sh --registry docker.io --repository myuser/pgsquash --push

# Multi-platform build
./docker/scripts/build.sh --platforms "linux/amd64,linux/arm64" --push

# Build specific version
./docker/scripts/build.sh --version v1.2.3 --push

# Help
./docker/scripts/build.sh --help
```

**Features**:
- Multi-architecture builds (amd64, arm64)
- Automatic version tagging from git
- Build cache optimization
- Security scanning with Trivy (if installed)
- Image metadata generation
- Buildx setup and management

### Validation Scripts

#### `quick-validate.sh`
One-command validation using Docker containers.

```bash
# Validate migrations in current directory
./docker/scripts/quick-validate.sh

# Validate specific directory
./docker/scripts/quick-validate.sh my-migrations/

# With custom output directory
OUTPUT_DIR=./validation-results ./docker/scripts/quick-validate.sh

# With specific safety level
PGSQUASH_SAFETY_LEVEL=aggressive ./docker/scripts/quick-validate.sh
```

**Features**:
- Automatic Docker Compose detection
- Fallback to direct Docker if needed
- Results summary with file listings

#### `validate.sh`
Full validation workflow with comprehensive reporting.

```bash
# Full validation with reporting
./docker/scripts/validate.sh

# Keep container for inspection
./docker/scripts/validate.sh --keep-container

# Help
./docker/scripts/validate.sh --help
```

**Features**:
- Comprehensive schema comparison
- Detailed validation reports with timestamps
- Original vs squashed migration comparison
- Supabase compatibility layer
- Side-by-side database comparison

#### `setup-validation.sh`
Setup complete validation environment with auto-detected extensions.

```bash
# Setup validation environment
./docker/scripts/setup-validation.sh migrations/

# With custom output directory
./docker/scripts/setup-validation.sh migrations/ /tmp/validation
```

**Features**:
- Automatic extension detection from migrations
- Auth service compatibility (Clerk, Supabase)
- Docker Compose generation
- Initialization scripts for extensions
- Ready-to-run validation script

#### `multi-version-test.sh`
Test migrations against multiple PostgreSQL versions.

```bash
# Test all default versions (17, 16, 15, 14, 13)
./docker/scripts/multi-version-test.sh

# Test specific directory
./docker/scripts/multi-version-test.sh my-migrations/

# Test specific versions
./docker/scripts/multi-version-test.sh --versions 17,15,13

# Quick test (latest and oldest only)
./docker/scripts/multi-version-test.sh --quick

# Help
./docker/scripts/multi-version-test.sh --help
```

**Features**:
- Parallel version testing
- Detailed logs per version
- Schema comparison across versions
- Summary report generation
- Quick mode for CI/CD

### Utility Scripts

#### `cleanup.sh`
Clean up Docker resources from validation and development.

```bash
# Clean validation containers only (default)
./docker/scripts/cleanup.sh

# Clean everything (containers, images, volumes)
./docker/scripts/cleanup.sh --all --volumes

# Skip confirmation prompts
./docker/scripts/cleanup.sh --all --force

# Help
./docker/scripts/cleanup.sh --help
```

**Features**:
- Label-based container cleanup
- Volume management
- Compose service cleanup
- System pruning
- Safety confirmations

#### `docker-run-examples.sh`
View and copy-paste Docker run examples.

```bash
# View all examples
./docker/scripts/docker-run-examples.sh
```

**Examples Include**:
- Basic squashing with resource limits
- Security-hardened containers
- Validation with Docker socket
- Read-only containers
- Different safety levels

#### `test-setup.sh`
Comprehensive Docker setup testing.

```bash
# Run all tests
./docker/scripts/test-setup.sh

# Help
./docker/scripts/test-setup.sh --help
```

**Tests**:
- Docker availability
- Image building
- Container runtime
- Validation workflow
- Full end-to-end workflow
- Multi-platform support

## Configuration

### Environment Variables

Copy template and customize:

```bash
cp docker/config-templates/.env.template .env
nano .env
```

### Config File

```bash
cp docker/config-templates/pgsquash.config.json.template config/pgsquash.config.json
```

## Best Practices

### Development

- Use `docker/validation/with-validation.yml`
- Mount local directories for quick iteration
- Enable verbose logging

### CI/CD

- Use `docker/engine/quick-start.yml`
- Pin image versions
- Use `--rm` for cleanup

### Production

- Use main `docker-compose.yml` or `docker/web-app/separated/`
- Set resource limits
- Use secrets management
- Enable monitoring

## Troubleshooting

### Build Issues

```bash
# Clean build
docker-compose build --no-cache

# Check Dockerfile syntax
docker build --dry-run -f Dockerfile .
```

### Validation Issues

```bash
# Check Docker socket
docker info

# List validation containers
docker ps -a --filter "label=pg-squash.type=validation"

# Clean up stuck containers
./docker/scripts/cleanup.sh
```

### Permission Issues

```bash
# Fix ownership
sudo chown -R $(id -u):$(id -g) output/

# Check user in container
docker-compose run --rm pgsquash id
```

## Advanced

### Custom PostgreSQL Image

Create `docker/validation/custom-postgres.dockerfile`:

```dockerfile
FROM postgres:15-alpine

RUN apk add --no-cache postgis
# Add more extensions
```

Build and use:

```bash
docker build -t pgsquash-postgres:custom -f docker/validation/custom-postgres.dockerfile .

# Update config
postgresql_image: pgsquash-postgres:custom
```

### Multi-Architecture Build

```bash
docker buildx create --use
docker buildx build --platform linux/amd64,linux/arm64 -t pgsquash:latest .
```

## Documentation

- [Setup Guide](../docs/docker/DOCKER_SETUP.md)
- [Use Cases](../docs/docker/DOCKER_USE_CASES.md)

## Support

Issues? Check:
1. [Troubleshooting Guide](../docs/TROUBLESHOOTING.md)
2. [GitHub Issues](https://github.com/capysquash/pg-squash-engine/issues)
3. Container logs: `docker-compose logs -f`
