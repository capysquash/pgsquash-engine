# Docker Usage Guide

pg-squash can run in Docker for portability and includes Docker-based schema validation.

## Use Cases

1. **Run pg-squash in a container** - Portable CLI environment
2. **Schema validation** - Ephemeral PostgreSQL containers for testing
3. **GitHub Action** - Automated PR analysis

## Running pg-squash in Docker

### Build Image

```bash
docker build -t pgsquash:latest .
```

### Basic Usage

```bash
# Analyze migrations
docker run --rm \
  -v $(pwd)/migrations:/app/migrations \
  pgsquash:latest analyze /app/migrations/*.sql

# Squash migrations
docker run --rm \
  -v $(pwd)/migrations:/app/migrations \
  -v $(pwd)/output:/app/output \
  pgsquash:latest squash /app/migrations/*.sql --output /app/output
```

### With Validation (Requires Docker Socket)

```bash
docker run --rm \
  -v $(pwd)/migrations:/app/migrations \
  -v $(pwd)/output:/app/output \
  -v /var/run/docker.sock:/var/run/docker.sock \
  pgsquash:latest squash /app/migrations/*.sql --output /app/output
```

**Note:** Mounting Docker socket gives container access to host Docker daemon.

### Environment Variables

```bash
# Safety configuration
PGSQUASH_SAFETY_LEVEL=standard  # paranoid|conservative|standard|aggressive

# Validation
PGSQUASH_AUTO_VALIDATE=true     # Auto-validate after squashing
PGSQUASH_DOCKER_ENABLED=true    # Enable Docker validation

# AI features
ANTHROPIC_API_KEY=sk-ant-...
OPENAI_API_KEY=sk-...

# Database connection (paranoid mode)
PROD_DB_DSN=postgres://user:pass@localhost/db

# Logging
PGSQUASH_LOG_LEVEL=info         # debug|info|warning|error
```

Example with environment:

```bash
docker run --rm \
  -v $(pwd)/migrations:/app/migrations \
  -v $(pwd)/output:/app/output \
  -e PGSQUASH_SAFETY_LEVEL=conservative \
  -e PGSQUASH_AUTO_VALIDATE=true \
  -e ANTHROPIC_API_KEY=$ANTHROPIC_API_KEY \
  pgsquash:latest squash /app/migrations/*.sql --output /app/output
```

## Schema Validation with Docker

pg-squash uses ephemeral PostgreSQL containers to validate schema equivalence.

### Validation Methods

#### 1. Two Containers (Most Accurate)

Creates separate containers for original and squashed migrations.

```
Container 1 (postgres:15-alpine) → Apply original migrations
Container 2 (postgres:15-alpine) → Apply squashed migrations
Compare schemas with pg_dump
```

**Pros:** Complete isolation, accurate
**Cons:** Slower (2× startup time)

#### 2. Schema Diff (Fastest)

Uses single container with schema versioning.

```
Container 1 (postgres:15-alpine) → Apply both migration sets
Compare via schema introspection
```

**Pros:** Fast, less resource usage
**Cons:** Potential namespace conflicts

#### 3. Single Container (Simplest)

Sequential application with reset.

```
Container 1 → Apply original → Dump schema → Reset
Container 1 → Apply squashed → Dump schema → Compare
```

**Pros:** Simple, minimal resources
**Cons:** Slower than schema diff

### Configure Validation Method

In `pgsquash.config.json`:

```json
{
  "validation": {
    "method": "TWO_CONTAINERS"  // or SCHEMA_DIFF, SINGLE_CONTAINER
  }
}
```

Or via CLI:

```bash
pgsquash validate migrations/ clean/ --method TWO_CONTAINERS
```

### Extension Detection

Validator automatically detects and installs required extensions:

- uuid-ossp
- vector (pgvector)
- pg_stat_statements
- PostGIS
- pg_trgm
- btree_gist
- And more...

Example validation output:

```
Validating migrations...

✓ Starting Docker containers...
✓ Detected extensions: [uuid-ossp, vector, pg_stat_statements]
✓ Installing extensions...
✓ Applying original migrations (15 files)...
✓ Applying squashed migrations (1 file)...
✓ Comparing schemas...

✓ Validation successful: Schemas are equivalent.

Validation completed in 18.5s
```

### Validation Failure Example

```
✗ Validation failed: Schemas are different.

Differences found:
--- Original Schema
+++ Squashed Schema
@@ -42,7 +42,6 @@
 CREATE TABLE users (
     id uuid PRIMARY KEY,
     email varchar(255) NOT NULL,
-    deleted_at timestamp
 );

Error: MISSING_COLUMN
Column 'deleted_at' exists in original but not in squashed schema.

Risk Level: HIGH
Recommendation: Review table consolidation rules
```

## Docker Compose Setup

For local development with validation:

```yaml
# docker-compose.yml
version: '3.8'

services:
  pgsquash:
    build: .
    volumes:
      - ./migrations:/app/migrations
      - ./output:/app/output
      - /var/run/docker.sock:/var/run/docker.sock
    environment:
      - PGSQUASH_SAFETY_LEVEL=standard
      - PGSQUASH_AUTO_VALIDATE=true
      - ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}
    command: squash /app/migrations/*.sql --output /app/output
```

Run:

```bash
docker-compose up
```

## GitHub Action Usage

```yaml
# .github/workflows/squash-migrations.yml
name: Squash Migrations

on:
  pull_request:
    paths:
      - 'migrations/**'

jobs:
  squash:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Build pg-squash
        run: docker build -t pgsquash:latest .

      - name: Squash migrations
        run: |
          docker run --rm \
            -v ${{ github.workspace }}/migrations:/app/migrations \
            -v ${{ github.workspace }}/output:/app/output \
            -v /var/run/docker.sock:/var/run/docker.sock \
            pgsquash:latest safe /app/migrations/*.sql --output /app/output

      - name: Upload artifacts
        uses: actions/upload-artifact@v3
        with:
          name: squashed-migrations
          path: output/
```

## Production Deployment

### Container Registry

```bash
# Tag for registry
docker tag pgsquash:latest ghcr.io/yourusername/pgsquash:latest

# Push to GitHub Container Registry
docker push ghcr.io/yourusername/pgsquash:latest
```

### Security Considerations

**Docker Socket Access:**
- Mounting `/var/run/docker.sock` gives container root-equivalent access
- Only mount in trusted environments
- Consider Docker-in-Docker alternatives for sensitive environments

**Network Isolation:**
- Validation containers use bridge network
- Ephemeral containers are destroyed after use
- No persistent data in validation containers

**Secrets Management:**
- Use environment variables for API keys
- Never commit secrets to Dockerfile
- Use Docker secrets or external secret management

### Resource Limits

```bash
docker run --rm \
  --memory="512m" \
  --cpus="2" \
  -v $(pwd)/migrations:/app/migrations \
  -v $(pwd)/output:/app/output \
  pgsquash:latest squash /app/migrations/*.sql --output /app/output
```

## Troubleshooting

### "Cannot connect to Docker daemon"

Ensure Docker socket is mounted:

```bash
docker run --rm \
  -v /var/run/docker.sock:/var/run/docker.sock \
  pgsquash:latest validate ...
```

Check Docker is running:

```bash
docker ps
```

### "Permission denied" on volumes

Ensure directories are readable:

```bash
chmod -R 755 migrations/ output/
```

### Validation containers not cleaning up

Manually remove:

```bash
docker ps -a | grep pgsquash-validation | awk '{print $1}' | xargs docker rm -f
```

### Extension installation fails

Check PostgreSQL version compatibility:

```json
{
  "postgresql_features": {
    "target_version": "15.0"
  }
}
```

## Further Reading

- [Production Guide](production.md) - Production deployment best practices
- [GitHub Integration](github-integration.md) - PR automation setup
- [Configuration](../configuration.md) - Config file options
