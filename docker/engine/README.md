# Engine: Containerized CLI Application

Run pg-squash CLI tool (the engine) inside a Docker container for portability and consistency.

## Purpose

- Run pg-squash without installing Go
- Consistent environment across teams
- Easy CI/CD integration
- Isolated from host system

## Quick Start

```bash
# 1. Squash migrations
docker compose -f docker/engine/quick-start.yml run --rm pgsquash squash /app/migrations/*.sql

# 2. Analyze migrations
docker compose -f docker/engine/quick-start.yml run --rm pgsquash analyze /app/migrations/*.sql

# 3. Interactive shell
docker compose -f docker/engine/quick-start.yml run --rm pgsquash bash
```

## Configuration

Set environment variables before running:

```bash
export MIGRATIONS_DIR=./my-migrations
export OUTPUT_DIR=./output
export PGSQUASH_SAFETY_LEVEL=aggressive

docker compose -f docker/engine/quick-start.yml run --rm pgsquash squash
```

## Examples

View all usage examples:

```bash
./docker/engine/examples.sh
```

## Files

- `quick-start.yml` - Docker Compose configuration
- `examples.sh` - Usage examples with resource limits

## Resource Limits

Default limits:
- Memory: 512MB (limit), 256MB (reservation)
- CPU: 1.0 core (limit), 0.5 core (reservation)

## Use Cases

✅ **Good for**:
- CI/CD pipelines
- Team consistency
- No Go installation required
- Reproducible builds

❌ **Not suitable for**:
- Validation workflows (use Validation)
- Web application (use Web App)
- Heavy parallel processing

## Next Steps

- **For validation**: See [Validation](../validation/)
- **For web app**: See [Web App](../web-app/)
- **For development**: See [dev-environment](../dev-environment/)
