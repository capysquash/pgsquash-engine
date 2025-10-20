# Cross-Compilation Setup

This document describes the modern, robust ARM64 cross-compilation setup for pgsquash Engine.

## Overview

pgsquash Engine uses **goreleaser-cross** for CGO-enabled cross-compilation to all supported Platforms. This approach provides:

- ✅ **Native ARM64 support** for Linux and macOS
- ✅ **No complex toolchain setup** - everything is pre-built in goreleaser-cross
- ✅ **Reproducible builds** via Docker containers
- ✅ **macOS SDK support** for proper macOS cross-compilation

## Supported Platforms

| Platform | Architecture          | Status                       |
| -------- | --------------------- | ---------------------------- |
| Linux    | AMD64 (x86\_64)       | ✅ Supported                  |
| Linux    | ARM64 (aarch64)       | ✅ Supported                  |
| macOS    | AMD64 (Intel)         | ✅ Supported                  |
| macOS    | ARM64 (Apple Silicon) | ✅ Supported                  |
| Windows  | AMD64                 | 🚧 Possible (not configured) |
| Windows  | ARM64                 | 🚧 Possible (not configured) |

## Technical Details

### goreleaser-cross

[goreleaser-cross](https://github.com/goreleaser/goreleaser-cross) is the official GoReleaser Docker image with pre-built C/C++ cross-compiler toolchains.

**Included toolchains:**

- `x86_64-linux-gnu-gcc/g++` - Linux AMD64
- `aarch64-linux-gnu-gcc/g++` - Linux ARM64
- `o64-clang/clang++` - macOS AMD64 (with SDK)
- `oa64-clang/clang++` - macOS ARM64 (with SDK)
- `x86_64-w64-mingw32-gcc/g++` - Windows AMD64
- `aarch64-w64-mingw32-gcc/g++` - Windows ARM64

### Build Configuration

Each Platform has its own build target in [.goreleaser.yml](../../.goreleaser.yml):

```yaml
builds:
  - id: linux-amd64
    env:
      - CGO_ENABLED=1
      - CC=x86_64-linux-gnu-gcc
      - CXX=x86_64-linux-gnu-g++
    goos: [linux]
    goarch: [amd64]

  - id: linux-arm64
    env:
      - CGO_ENABLED=1
      - CC=aarch64-linux-gnu-gcc
      - CXX=aarch64-linux-gnu-g++
    goos: [linux]
    goarch: [arm64]

  - id: darwin-amd64
    env:
      - CGO_ENABLED=1
      - CC=o64-clang
      - CXX=o64-clang++
    goos: [darwin]
    goarch: [amd64]

  - id: darwin-arm64
    env:
      - CGO_ENABLED=1
      - CC=oa64-clang
      - CXX=oa64-clang++
    goos: [darwin]
    goarch: [arm64]
```

### GitHub Actions Workflow

The [release workflow](../../.github/workflows/release.yml) uses the goreleaser-cross distribution:

```yaml
- name: Set up QEMU
  uses: docker/setup-qemu-action@v3

- name: Set up Docker Buildx
  uses: docker/setup-buildx-action@v3

- name: Run GoReleaser with goreleaser-cross
  uses: goreleaser/goreleaser-action@v6
  with:
    distribution: goreleaser-cross
    version: latest
    args: release --clean
```

## Why goreleaser-cross?

### Previous Approaches (Not Used)

1. **Zig Compiler** - Works well for Linux ARM64, but has SDK issues for macOS cross-compilation
2. **Manual Toolchains** - Complex setup, hard to maintain
3. **Bazel** - Powerful but would require complete build system rewrite

### Benefits of goreleaser-cross

- ✅ **Official GoReleaser tool** - Well-maintained and documented
- ✅ **macOS SDK included** - Proper framework support (CoreFoundation, Security, resolv)
- ✅ **Zero host dependencies** - Everything runs in Docker
- ✅ **Proven solution** - Used by major projects like [Bytebase](https://github.com/bytebase/bytebase)
- ✅ **No hacks or workarounds** - Production-grade toolchains

## Local Development

### Building Locally

To build for your current Platform:

```bash
go build -o pgsquash ./cmd/pgsquash
```

### Testing Cross-Compilation Locally

You can test the goreleaser-cross setup locally:

```bash
# Install goreleaser
brew install goreleaser/tap/goreleaser-cross

# Run a test build (snapshot, no release)
goreleaser release --snapshot --clean
```

### Manual Cross-Compilation

For development/testing, you can cross-compile manually using the toolchains:

```bash
# Linux ARM64
CGO_ENABLED=1 CC=aarch64-linux-gnu-gcc GOOS=linux GOARCH=arm64 \
  go build -o pgsquash-linux-arm64 ./cmd/pgsquash

# macOS ARM64 (requires macOS SDK - use goreleaser-cross Docker)
docker run --rm -v $(pwd):/workspace -w /workspace \
  goreleaser/goreleaser-cross:latest \
  bash -c "CGO_ENABLED=1 CC=oa64-clang GOOS=darwin GOARCH=arm64 \
    go build -o pgsquash-darwin-arm64 ./cmd/pgsquash"
```

## Troubleshooting

### Build Fails with "unable to find dynamic system library 'resolv'"

This happens when using Zig without proper macOS SDK. Solution: Use goreleaser-cross which includes the SDK.

### ARM64 Build Fails with Assembly Errors

This happens when cross-compiling without proper ARM64 toolchain. Solution: Ensure you're using goreleaser-cross distribution in GitHub Actions.

### macOS Binary Doesn't Work

Ensure you're using the correct clang compiler (`o64-clang` for AMD64, `oa64-clang` for ARM64) with goreleaser-cross.

## References

- [goreleaser-cross GitHub](https://github.com/goreleaser/goreleaser-cross)
- [GoReleaser CGO Cross-Compilation Cookbook](https://goreleaser.com/cookbooks/cgo-and-crosscompiling/)
- [Bytebase's Cross-Compilation Guide](https://www.bytebase.com/blog/how-to-cross-compile-with-cgo-use-goreleaser-and-github-action/)
- [goreleaser-cross Example](https://github.com/goreleaser/goreleaser-cross-example)

## Future Enhancements

- [ ] Add Windows ARM64 support
- [ ] Add Linux ARMv7 support (for Raspberry Pi)
- [ ] Consider s390x support for IBM mainframes
- [ ] Benchmark build times across Platforms
