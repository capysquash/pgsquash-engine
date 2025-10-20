# Docker Publishing Guide

Complete guide for building, tagging, and publishing pgsquash Docker images to container registries.

## Table of Contents

- [Quick Start](#quick-start)
- [Image Registries](#image-registries)
- [Building Images](#building-images)
- [Multi-Platform Builds](#multi-Platform-builds)
- [Tagging Strategy](#tagging-strategy)
- [Publishing to Registries](#publishing-to-registries)
- [Automated Publishing (CI/CD)](#automated-publishing-cicd)
- [Security Best Practices](#security-best-practices)
- [Troubleshooting](#troubleshooting)

## Quick Start

### Build and Test Locally

```bash
# Build for local architecture
docker build -t pgsquash:dev .

# Test the image
docker run --rm pgsquash:dev --version
docker run --rm pgsquash:dev --help
```

### Build for Multiple Platforms

```bash
# Setup buildx (one-time)
docker buildx create --name pgsquash-builder --use
docker buildx inspect --bootstrap

# Build for multiple Platforms
docker buildx build \
  --Platform linux/amd64,linux/arm64 \
  -t yourusername/pgsquash:latest \
  --push \
  .
```

## Image Registries

pgsquash supports publishing to multiple container registries:

### 1. GitHub Container Registry (GHCR) - Recommended

**Registry**: `ghcr.io`
**Image Path**: `ghcr.io/OWNER/pgsquash`

**Benefits**:

- Free for public repositories
- Integrated with GitHub Actions
- Automatic permissions from repository access
- Excellent for open source projects

**Authentication**:

```bash
echo $GITHUB_TOKEN | docker login ghcr.io -u USERNAME --password-stdin
```

### 2. Docker Hub

**Registry**: `docker.io` (default)
**Image Path**: `docker.io/USERNAME/pgsquash`

**Benefits**:

- Most widely used
- Free tier available
- Good for public images
- Official Docker registry

**Authentication**:

```bash
docker login
# Or with token:
echo $DOCKERHUB_TOKEN | docker login -u USERNAME --password-stdin
```

### 3. Azure Container Registry (ACR)

**Registry**: `REGISTRYNAME.azurecr.io`
**Image Path**: `REGISTRYNAME.azurecr.io/pgsquash`

**Benefits**:

- Enterprise-grade
- Integrated with Azure services
- Geo-replication support
- Advanced security features

**Authentication**:

```bash
az acr login --name REGISTRYNAME
```

### 4. AWS Elastic Container Registry (ECR)

**Registry**: `ACCOUNT.dkr.ecr.REGION.amazonaws.com`
**Image Path**: `ACCOUNT.dkr.ecr.REGION.amazonaws.com/pgsquash`

**Authentication**:

```bash
aws ecr get-login-password --region REGION | \
  docker login --username AWS --password-stdin \
  ACCOUNT.dkr.ecr.REGION.amazonaws.com
```

### 5. Google Container Registry (GCR)

**Registry**: `gcr.io`
**Image Path**: `gcr.io/PROJECT-ID/pgsquash`

**Authentication**:

```bash
gcloud auth configure-docker
```

## Building Images

### Standard Build

```bash
# Basic build
docker build -t pgsquash:latest .

# With build arguments
docker build \
  --build-arg BUILD_VERSION=1.0.0 \
  --build-arg BUILD_DATE=$(date -u +'%Y-%m-%dT%H:%M:%SZ') \
  --build-arg GIT_COMMIT=$(git rev-parse --short HEAD) \
  -t pgsquash:1.0.0 \
  .
```

### Build with Cache

```bash
# Use cache from previous builds
docker build \
  --cache-from pgsquash:latest \
  -t pgsquash:latest \
  .

# With buildx cache
docker buildx build \
  --cache-from type=registry,ref=pgsquash:buildcache \
  --cache-to type=registry,ref=pgsquash:buildcache,mode=max \
  -t pgsquash:latest \
  .
```

### Build API Server

```bash
# Build API server image
docker build \
  -f docker/api-server/Dockerfile \
  --build-arg BUILD_VERSION=1.0.0 \
  -t pgsquash-api:1.0.0 \
  .
```

## Multi-Platform Builds

pgsquash supports multiple architectures for broad compatibility.

### Supported Platforms

- `linux/amd64` - Intel/AMD 64-bit (most common)
- `linux/arm64` - ARM 64-bit (Apple Silicon, AWS Graviton, Raspberry Pi 4+)
- `linux/arm/v7` - ARM 32-bit (older Raspberry Pi)

### Setup Buildx

```bash
# Create builder instance
docker buildx create \
  --name pgsquash-builder \
  --driver docker-container \
  --use

# Inspect and bootstrap
docker buildx inspect --bootstrap

# Verify Platforms
docker buildx inspect | grep Platforms
```

### Multi-Platform Build

```bash
# Build for multiple Platforms and push
docker buildx build \
  --Platform linux/amd64,linux/arm64 \
  --build-arg BUILD_VERSION=1.0.0 \
  --build-arg BUILD_DATE=$(date -u +'%Y-%m-%dT%H:%M:%SZ') \
  --build-arg GIT_COMMIT=$(git rev-parse --short HEAD) \
  -t ghcr.io/CAPYSQUASH/pgsquash:1.0.0 \
  -t ghcr.io/CAPYSQUASH/pgsquash:latest \
  --push \
  .
```

### Build for Specific Platform

```bash
# Build only for ARM64 (e.g., Apple Silicon, AWS Graviton)
docker buildx build \
  --Platform linux/arm64 \
  -t pgsquash:arm64 \
  --load \
  .

# Build only for AMD64
docker buildx build \
  --Platform linux/amd64 \
  -t pgsquash:amd64 \
  --load \
  .
```

## Tagging Strategy

### Semantic Versioning

Follow semantic versioning for releases:

```bash
# Full version tag
docker tag pgsquash:build ghcr.io/CAPYSQUASH/pgsquash:1.2.3

# Major.minor tag
docker tag pgsquash:build ghcr.io/CAPYSQUASH/pgsquash:1.2

# Major tag
docker tag pgsquash:build ghcr.io/CAPYSQUASH/pgsquash:1

# Latest tag
docker tag pgsquash:build ghcr.io/CAPYSQUASH/pgsquash:latest
```

### Git-Based Tagging

```bash
# Tag with git commit SHA
GIT_SHA=$(git rev-parse --short HEAD)
docker tag pgsquash:build ghcr.io/CAPYSQUASH/pgsquash:sha-${GIT_SHA}

# Tag with git branch
GIT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
docker tag pgsquash:build ghcr.io/CAPYSQUASH/pgsquash:${GIT_BRANCH}
```

### Environment-Based Tagging

```bash
# Development
docker tag pgsquash:build ghcr.io/CAPYSQUASH/pgsquash:dev

# Staging
docker tag pgsquash:build ghcr.io/CAPYSQUASH/pgsquash:staging

# Production
docker tag pgsquash:build ghcr.io/CAPYSQUASH/pgsquash:prod
```

### Combined Tagging Script

```bash
#!/bin/bash
set -e

VERSION=${1:-dev}
REGISTRY=${2:-ghcr.io/CAPYSQUASH}
IMAGE_NAME=pgsquash

# Build
docker buildx build \
  --Platform linux/amd64,linux/arm64 \
  --build-arg BUILD_VERSION=${VERSION} \
  --build-arg BUILD_DATE=$(date -u +'%Y-%m-%dT%H:%M:%SZ') \
  --build-arg GIT_COMMIT=$(git rev-parse --short HEAD) \
  -t ${REGISTRY}/${IMAGE_NAME}:${VERSION} \
  -t ${REGISTRY}/${IMAGE_NAME}:latest \
  --push \
  .

echo "✅ Published ${REGISTRY}/${IMAGE_NAME}:${VERSION}"
```

## Publishing to Registries

### GitHub Container Registry (GHCR)

```bash
# 1. Create GitHub Personal Access Token with `write:packages` scope
#    https://github.com/settings/tokens

# 2. Login to GHCR
echo $GITHUB_TOKEN | docker login ghcr.io -u YOUR_USERNAME --password-stdin

# 3. Build and tag
docker buildx build \
  --Platform linux/amd64,linux/arm64 \
  -t ghcr.io/OWNER/pgsquash:1.0.0 \
  -t ghcr.io/OWNER/pgsquash:latest \
  --push \
  .

# 4. Verify
docker pull ghcr.io/OWNER/pgsquash:latest
docker run --rm ghcr.io/OWNER/pgsquash:latest --version
```

### Docker Hub

```bash
# 1. Create Docker Hub account and access token
#    https://hub.docker.com/settings/security

# 2. Login
docker login -u YOUR_USERNAME

# 3. Build and push
docker buildx build \
  --Platform linux/amd64,linux/arm64 \
  -t YOUR_USERNAME/pgsquash:1.0.0 \
  -t YOUR_USERNAME/pgsquash:latest \
  --push \
  .

# 4. Verify
docker pull YOUR_USERNAME/pgsquash:latest
```

### Private Registry

```bash
# 1. Login to private registry
docker login registry.example.com

# 2. Build and push
docker buildx build \
  --Platform linux/amd64,linux/arm64 \
  -t registry.example.com/pgsquash:1.0.0 \
  --push \
  .

# 3. Pull from private registry
docker pull registry.example.com/pgsquash:1.0.0
```

## Automated Publishing (CI/CD)

### GitHub Actions

The project includes `.github/workflows/docker-publish.yml` for automated publishing.

**Triggers**:

- Push to `main` branch → publishes `edge` tag
- Push tags matching `v*` → publishes version tags
- Manual workflow dispatch

**Registries**:

- GitHub Container Registry (GHCR)
- Docker Hub (if credentials configured)

**Setup**:

1. **For GHCR** (automatic):
   - No setup needed, uses `GITHUB_TOKEN`

2. **For Docker Hub**:
   ```bash
   # Add repository secrets:
   # - DOCKERHUB_USERNAME
   # - DOCKERHUB_TOKEN
   ```

**Manual Trigger**:

```bash
# Via GitHub CLI
gh workflow run docker-publish.yml \
  -f push_to_registry=true \
  -f Platforms=linux/amd64,linux/arm64
```

### GitLab CI

Example `.gitlab-ci.yml`:

```yaml
build-docker:
  stage: build
  image: docker:latest
  services:
    - docker:dind
  before_script:
    - docker login -u $CI_REGISTRY_USER -p $CI_REGISTRY_PASSWORD $CI_REGISTRY
  script:
    - docker buildx create --use
    - docker buildx build
        --Platform linux/amd64,linux/arm64
        --build-arg BUILD_VERSION=$CI_COMMIT_TAG
        -t $CI_REGISTRY_IMAGE:$CI_COMMIT_TAG
        -t $CI_REGISTRY_IMAGE:latest
        --push .
  only:
    - tags
```

### Using Helper Script

The project includes `docker/scripts/build.sh` for simplified publishing:

```bash
# Build locally
./docker/scripts/build.sh

# Build and push to GHCR
./docker/scripts/build.sh \
  --registry ghcr.io \
  --repository CAPYSQUASH/pgsquash \
  --version 1.0.0 \
  --push

# Build and push to Docker Hub
./docker/scripts/build.sh \
  --registry docker.io \
  --repository yourusername/pgsquash \
  --version 1.0.0 \
  --Platforms linux/amd64,linux/arm64 \
  --push

# Build and push to both registries
./docker/scripts/build.sh \
  --registries "ghcr.io/CAPYSQUASH,docker.io/yourusername" \
  --version 1.0.0 \
  --push
```

## Security Best Practices

### Image Signing

Sign images for verification:

```bash
# Install cosign
brew install cosign  # macOS
# or download from https://github.com/sigstore/cosign/releases

# Generate key pair
cosign generate-key-pair

# Sign image
cosign sign --key cosign.key ghcr.io/CAPYSQUASH/pgsquash:1.0.0

# Verify signature
cosign verify --key cosign.pub ghcr.io/CAPYSQUASH/pgsquash:1.0.0
```

### Vulnerability Scanning

Scan images before publishing:

```bash
# Using Trivy
trivy image pgsquash:latest

# Using Grype
grype pgsquash:latest

# Using Docker Scout
docker scout cves pgsquash:latest
```

### SBOM Generation

Generate Software Bill of Materials:

```bash
# Using Syft
syft pgsquash:latest -o spdx-json > sbom.json

# Using Docker
docker sbom pgsquash:latest
```

### Best Practices

1. **Never commit secrets** to Dockerfiles or images
2. **Use multi-stage builds** to minimize final image size
3. **Run as non-root user** (already implemented)
4. **Keep base images updated** regularly
5. **Scan for vulnerabilities** before publishing
6. **Sign images** for production use
7. **Use specific tags**, avoid `:latest` in production
8. **Implement image provenance** for supply chain security

## Troubleshooting

### Build Failures

**CGO\_ENABLED errors**:

```bash
# pgsquash requires CGO for pg_query_go
# Ensure build uses proper C compiler
docker build --build-arg CGO_ENABLED=1 .
```

**Platform-specific issues**:

```bash
# Build only for your Platform to debug
docker build --Platform linux/$(uname -m) .
```

### Push Failures

**Authentication errors**:

```bash
# Re-login to registry
docker logout ghcr.io
docker login ghcr.io

# Check token permissions
# GHCR needs: write:packages, read:packages
```

**Rate limiting**:

```bash
# Docker Hub free tier: 100 pulls/6hrs
# Solution: Authenticate or upgrade plan
docker login

# Use alternative registry
# GHCR has no rate limits for authenticated users
```

### Multi-Platform Issues

**QEMU not found**:

```bash
# Install QEMU for cross-Platform builds
docker run --privileged --rm tonistiigi/binfmt --install all
```

**Buildx not available**:

```bash
# Update Docker to latest version
# Or enable experimental features:
export DOCKER_CLI_EXPERIMENTAL=enabled
```

## Additional Resources

- [Docker Documentation](https://docs.docker.com/)
- [Buildx Documentation](https://docs.docker.com/buildx/working-with-buildx/)
- [GitHub Container Registry Guide](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry)
- [Docker Hub Documentation](https://docs.docker.com/docker-hub/)
- [Trivy Scanning](https://github.com/aquasecurity/trivy)
- [Cosign Image Signing](https://github.com/sigstore/cosign)

## Next Steps

- [Docker Deployment Guide](DOCKER_DEPLOYMENT_GUIDE.md) - Deploy images to various environments
- [Docker Best Practices](DOCKER_BEST_PRACTICES.md) - Optimization and security guidelines
- [Main Docker README](../../docker/README.md) - Overview of Docker setup
