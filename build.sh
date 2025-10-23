#!/bin/bash
# =============================================================================
# Build Script for pgsquash-engine
# =============================================================================
# Builds both the CLI tool and API server with proper version information
# and places them in the correct locations

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Build information
VERSION="${VERSION:-0.9.5-beta}"
BUILD_DATE=$(date -u +'%Y-%m-%dT%H:%M:%SZ')
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

echo -e "${BLUE}==============================================================================${NC}"
echo -e "${BLUE}Building pgsquash-engine${NC}"
echo -e "${BLUE}==============================================================================${NC}"
echo ""
echo -e "${GREEN}Version:${NC}     ${VERSION}"
echo -e "${GREEN}Build Date:${NC}  ${BUILD_DATE}"
echo -e "${GREEN}Git Commit:${NC}  ${GIT_COMMIT}"
echo ""

# Get script directory (project root)
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR"

# Create bin directory if it doesn't exist
mkdir -p bin

# Build flags
LDFLAGS="-X 'main.version=${VERSION}' -X 'main.buildDate=${BUILD_DATE}' -X 'main.gitCommit=${GIT_COMMIT}'"

echo -e "${BLUE}==============================================================================${NC}"
echo -e "${BLUE}Step 1: Clean previous builds${NC}"
echo -e "${BLUE}==============================================================================${NC}"
rm -f bin/pgsquash bin/api-server pgsquash main
echo -e "${GREEN}☑ Cleaned old binaries${NC}"
echo ""

echo -e "${BLUE}==============================================================================${NC}"
echo -e "${BLUE}Step 2: Download Go dependencies${NC}"
echo -e "${BLUE}==============================================================================${NC}"
go mod download
go mod verify
echo -e "${GREEN}☑ Dependencies downloaded and verified${NC}"
echo ""

echo -e "${BLUE}==============================================================================${NC}"
echo -e "${BLUE}Step 3: Build CLI tool (pgsquash)${NC}"
echo -e "${BLUE}==============================================================================${NC}"
echo "Building: cmd/pgsquash → bin/pgsquash"
CGO_ENABLED=1 go build -v \
  -ldflags="${LDFLAGS}" \
  -o bin/pgsquash \
  ./cmd/pgsquash

if [ -f bin/pgsquash ]; then
  echo -e "${GREEN}☑ CLI built successfully${NC}"
  ls -lh bin/pgsquash

  # Test the binary
  echo ""
  echo "Testing binary..."
  ./bin/pgsquash --version
else
  echo -e "${RED}✗ CLI build failed${NC}"
  exit 1
fi
echo ""

echo -e "${BLUE}==============================================================================${NC}"
echo -e "${BLUE}Step 4: Build API Server${NC}"
echo -e "${BLUE}==============================================================================${NC}"
echo "Building: cmd/api-server → bin/api-server"
CGO_ENABLED=1 go build -v \
  -ldflags="${LDFLAGS}" \
  -o bin/api-server \
  ./cmd/api-server

if [ -f bin/api-server ]; then
  echo -e "${GREEN}☑ API Server built successfully${NC}"
  ls -lh bin/api-server

  # Test the binary (just check if it exists and is executable)
  echo ""
  echo "Testing binary..."
  chmod +x bin/api-server
  echo "Binary is executable"
else
  echo -e "${RED}✗ API Server build failed${NC}"
  exit 1
fi
echo ""

echo -e "${BLUE}==============================================================================${NC}"
echo -e "${BLUE}Step 5: Create convenience symlinks${NC}"
echo -e "${BLUE}==============================================================================${NC}"

# Create symlink in project root for convenience
ln -sf bin/pgsquash pgsquash
echo -e "${GREEN}☑ Created symlink: pgsquash → bin/pgsquash${NC}"

ln -sf bin/api-server main
echo -e "${GREEN}☑ Created symlink: main → bin/api-server${NC}"
echo ""

echo -e "${BLUE}==============================================================================${NC}"
echo -e "${BLUE}Build Summary${NC}"
echo -e "${BLUE}==============================================================================${NC}"
echo ""
echo -e "${GREEN}☑ All binaries built successfully!${NC}"
echo ""
echo "Binaries location:"
echo "  ► CLI:        $(pwd)/bin/pgsquash"
echo "  ► API Server: $(pwd)/bin/api-server"
echo ""
echo "Convenience symlinks:"
echo "  ► ./pgsquash  → bin/pgsquash"
echo "  ► ./main      → bin/api-server"
echo ""
echo "Build information:"
echo "  ► Version:     ${VERSION}"
echo "  ► Build Date:  ${BUILD_DATE}"
echo "  ► Git Commit:  ${GIT_COMMIT}"
echo ""
echo -e "${YELLOW}Next steps:${NC}"
echo "  1. Test CLI:        ./pgsquash --help"
echo "  2. Test API:        ./main (or ./bin/api-server)"
echo "  3. Deploy to Fly:   fly deploy"
echo ""
echo -e "${GREEN}Ready to deploy! 🚀${NC}"
