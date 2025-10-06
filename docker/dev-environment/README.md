# Development Environment

Complete local development environment with all services and tools.

## Overview

This is the **development setup** that includes:
- pg-squash CLI with Docker socket access
- PostgreSQL 17 (primary) + PostgreSQL 15 (secondary) + PostgreSQL 13 (legacy)
- Redis for caching
- MinIO for S3-compatible storage
- Grafana + Prometheus for monitoring
- Traefik reverse proxy
- pgAdmin for database management
- File browser for managing migrations

## File Location

The main docker-compose file is at the **root** of the project:

```
/docker-compose.yml
```

This file is kept at the root for convenience and convention. The `full-stack.yml` in this directory is a simplified version for development.

## Quick Start

### Full Environment (All Services)

```bash
# From project root
docker compose up -d

# Or with specific profiles
docker compose --profile monitoring --profile management up -d
```

### Simplified Development

```bash
# Just pg-squash + PostgreSQL
docker compose -f docker/dev-environment/full-stack.yml up -d
```

## Services

### Core Services (Always Running)

- **pgsquash** - Main application with Docker socket access
- **postgres-primary** (PostgreSQL 17) - Primary database

### Optional Services (Profiles)

- **postgres-secondary** (PostgreSQL 15) - Profile: `multi-version`
- **postgres-legacy** (PostgreSQL 13) - Profile: `legacy`
- **redis** - Profile: `caching`
- **minio** - Profile: `storage`
- **grafana** + **prometheus** - Profile: `monitoring`
- **traefik** - Profile: `proxy`
- **pgadmin** + **filebrowser** - Profile: `management`

## Usage Examples

### Basic Development

```bash
# Start core services only
docker compose up -d pgsquash postgres-primary

# Access PostgreSQL
docker compose exec postgres-primary psql -U pgsquash -d pgsquash_primary

# Run pg-squash
docker compose exec pgsquash pgsquash squash /app/migrations/*.sql
```

### With Multi-Version Testing

```bash
# Start with multiple PostgreSQL versions
docker compose --profile multi-version up -d

# Test against PG 17
docker compose exec postgres-primary psql -U pgsquash

# Test against PG 15
docker compose exec postgres-secondary psql -U pgsquash

# Test against PG 13
docker compose --profile legacy up -d postgres-legacy
docker compose exec postgres-legacy psql -U pgsquash
```

### With Monitoring

```bash
# Start with monitoring stack
docker compose --profile monitoring up -d

# Access Grafana
open http://localhost:3000
# Default: admin / pgsquash_grafana

# Access Prometheus
open http://localhost:9090
```

### With Database Management

```bash
# Start with pgAdmin
docker compose --profile management up -d

# Access pgAdmin
open http://localhost:5050
# Default: admin@pgsquash.localhost / pgsquash_pgadmin

# Access File Browser
open http://localhost:8081
```

## Environment Variables

Create a `.env` file in the project root:

```bash
# Core configuration
BUILD_VERSION=dev
PGSQUASH_SAFETY_LEVEL=standard
PGSQUASH_AUTO_VALIDATE=true
PGSQUASH_LOG_LEVEL=info

# Database
POSTGRES_PASSWORD=your_secure_password_here

# Ports (optional overrides)
POSTGRES_PRIMARY_PORT=5432
POSTGRES_SECONDARY_PORT=5433
POSTGRES_LEGACY_PORT=5434
REDIS_PORT=6379
GRAFANA_PORT=3000
PROMETHEUS_PORT=9090
PGADMIN_PORT=5050

# Monitoring
GRAFANA_PASSWORD=your_grafana_password

# Storage
MINIO_ROOT_USER=pgsquash
MINIO_ROOT_PASSWORD=your_minio_password

# SSL (for Traefik)
ACME_EMAIL=your-email@example.com
```

## Volumes

All data is persisted in Docker volumes:

```bash
# List all pg-squash volumes
docker volume ls | grep pgsquash

# Backup a database
docker compose exec postgres-primary pg_dump -U pgsquash pgsquash_primary > backup.sql

# Remove all volumes (⚠️ DESTRUCTIVE)
docker compose down -v
```

## Networks

All services run in the `pgsquash-net` bridge network:

```bash
# Inspect network
docker network inspect pgsquash-net

# Services can communicate via container names
# Example: pgsquash -> postgres-primary:5432
```

## Resource Limits

Default resource limits are set for each service. Adjust in `docker-compose.yml`:

```yaml
deploy:
  resources:
    limits:
      cpus: '2.0'
      memory: 1G
    reservations:
      cpus: '1.0'
      memory: 512M
```

## Profiles Summary

| Profile | Services | Purpose |
|---------|----------|---------|
| (none) | pgsquash + postgres-primary | Basic development |
| `multi-version` | + postgres-secondary (PG 15) | Multi-version testing |
| `legacy` | + postgres-legacy (PG 13) | Legacy compatibility |
| `caching` | + redis | Caching layer |
| `storage` | + minio | S3-compatible storage |
| `monitoring` | + grafana + prometheus | Metrics and dashboards |
| `proxy` | + traefik | Reverse proxy + SSL |
| `management` | + pgadmin + filebrowser | Database & file management |

## Comparison with Other Use Cases

### vs Engine (docker/engine/)
- **Engine**: Minimal CLI-only container
- **Dev Environment**: Full stack with all services

### vs Validation (docker/validation/)
- **Validation**: Ephemeral containers for testing
- **Dev Environment**: Persistent development databases

### vs Web App (docker/web-app/)
- **Web App**: Production deployment scenarios
- **Dev Environment**: Local development with all tools

## Common Tasks

### Reset Everything

```bash
# Stop all services
docker compose down

# Remove volumes
docker compose down -v

# Rebuild and start fresh
docker compose build --no-cache
docker compose up -d
```

### Update Images

```bash
# Pull latest images
docker compose pull

# Rebuild pg-squash
docker compose build pgsquash

# Restart
docker compose up -d
```

### View Logs

```bash
# All services
docker compose logs -f

# Specific service
docker compose logs -f pgsquash

# Last 100 lines
docker compose logs --tail=100 pgsquash
```

### Exec into Containers

```bash
# pg-squash CLI
docker compose exec pgsquash bash

# PostgreSQL
docker compose exec postgres-primary psql -U pgsquash

# Redis CLI
docker compose exec redis redis-cli
```

## Troubleshooting

### Port Conflicts

If ports are already in use:

```bash
# Check what's using port 5432
lsof -i :5432

# Override port in .env
echo "POSTGRES_PRIMARY_PORT=5433" >> .env
docker compose up -d
```

### Permission Issues

```bash
# Fix output directory permissions
sudo chown -R $(id -u):$(id -g) output/

# Fix log directory permissions
sudo chown -R $(id -u):$(id -g) logs/
```

### Docker Socket Access

```bash
# Test Docker socket access
docker compose exec pgsquash docker ps

# If it fails, check Docker socket permissions
ls -la /var/run/docker.sock
```

### Database Connection Issues

```bash
# Test database connection
docker compose exec pgsquash psql postgresql://pgsquash:password@postgres-primary:5432/pgsquash_primary

# Check database logs
docker compose logs postgres-primary
```

## Production Warning

⚠️ **This setup is for DEVELOPMENT ONLY**

For production deployments, use:
- **Engine**: [docker/engine/](../engine/) - CLI-only
- **Validation**: [docker/validation/](../validation/) - Validation workflows
- **Web App**: [docker/web-app/](../web-app/) - Production web deployment

Production considerations:
- Remove developer tools (pgAdmin, filebrowser)
- Use managed databases (not containers)
- Enable SSL/TLS
- Use secrets management
- Configure proper backup strategies
- Set up monitoring and alerting
- Implement proper logging

## Next Steps

- **For CLI usage**: See [Engine](../engine/)
- **For validation**: See [Validation](../validation/)
- **For production web app**: See [Web App](../web-app/)

## Related Documentation

- [Root docker-compose.yml](../../docker-compose.yml) - Complete configuration
- [docker/README.md](../README.md) - Main Docker documentation
- [docs/DEVELOPMENT.md](../../docs/DEVELOPMENT.md) - Development guide
