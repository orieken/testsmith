# Releasing TestSmith

This document describes how to create and publish a new release of TestSmith.

## Prerequisites

1. **PyPI API Token**: Set up `PYPI_API_TOKEN` in GitHub repository secrets
   - Go to: Settings → Secrets and variables → Actions → New repository secret
   - Name: `PYPI_API_TOKEN`
   - Value: Your PyPI API token (starts with `pypi-`)

2. **Permissions**: You need write access to the repository and permission to create tags

## Release Process

### 1. Update Version

Edit `src/testsmith/__init__.py` and update the version:

```python
__version__ = "1.2.3"  # Update to new version
```

### 2. Commit and Push

```bash
git add src/testsmith/__init__.py
git commit -m "Bump version to 1.2.3"
git push origin main
```

### 3. Create and Push Tag

```bash
git tag v1.2.3
git push origin v1.2.3
```

**That's it!** The GitHub Actions release workflow will automatically:
- Build binaries for Linux (amd64), macOS (amd64 + arm64), and Windows (amd64)
- Run smoke tests on all binaries
- Create a GitHub Release with all binaries and checksums
- Publish the package to PyPI (for `pipx install testsmith`)

## Pre-Release Tags

For beta/RC releases, use a pre-release tag format:

```bash
git tag v1.2.3-beta.1
git push origin v1.2.3-beta.1
```

Pre-release tags will:
- ✅ Build binaries and create GitHub Release
- ❌ **Skip** PyPI publish (to avoid polluting the stable release index)

## What Happens Automatically

### CI Workflow (on every push/PR)
- Runs on: Ubuntu, macOS, Windows
- Python versions: 3.10, 3.11, 3.12
- Steps: lint (ruff + black) → test with coverage (85% minimum)
- Uploads coverage report artifact

### Release Workflow (on version tag)
1. **Build** job: Creates 4 platform-specific binaries
2. **Release** job: Creates GitHub Release with binaries + SHA256 checksums
3. **Publish** job: Publishes to PyPI (skipped for pre-release tags)

## Local Testing

### Build Binary Locally

**Linux/macOS:**
```bash
./scripts/build-local.sh
```

**Windows:**
```powershell
.\scripts\build-local.ps1
```

The binary will be in `dist/testsmith` (or `dist/testsmith.exe` on Windows).

### Test Binary on Clean System

**Linux (using Docker):**
```bash
docker run --rm -v $(pwd)/dist:/app ubuntu:24.04 /app/testsmith --version
```

**RHEL (using Docker):**
```bash
docker run --rm -v $(pwd)/dist:/app redhat/ubi9 /app/testsmith --version
```

**Windows:**
Test on a clean Windows VM or use Windows Sandbox.

## Verifying Checksums

After a release is published, verify the checksums:

```bash
# Download SHA256SUMS.txt from GitHub Release
curl -LO https://github.com/orieken/testsmith/releases/download/v1.2.3/SHA256SUMS.txt

# Download a binary
curl -LO https://github.com/orieken/testsmith/releases/download/v1.2.3/testsmith-linux-amd64

# Verify
sha256sum -c SHA256SUMS.txt --ignore-missing
```

## Installation Methods

Users can install TestSmith in three ways:

### 1. Binary (No Python Required)
```bash
# Download from GitHub Releases
curl -LO https://github.com/orieken/testsmith/releases/latest/download/testsmith-linux-amd64
chmod +x testsmith-linux-amd64
sudo mv testsmith-linux-amd64 /usr/local/bin/testsmith
```

### 2. pipx (Recommended for Python users)
```bash
pipx install testsmith
```

### 3. pip
```bash
pip install testsmith
```

## Troubleshooting

### Release workflow failed
- Check the Actions tab for error logs
- Common issues: PyPI token expired, binary build errors, test failures

### Binary doesn't work on target platform
- Verify the binary was built for the correct platform
- Check that the target system meets minimum requirements (glibc version on Linux, etc.)
- For RHEL/CentOS, the Linux binary should work on RHEL 7+

### PyPI publish failed
- Verify `PYPI_API_TOKEN` secret is set correctly
- Check that the version doesn't already exist on PyPI
- Ensure the version in `__init__.py` matches the git tag

## Platform Compatibility

- **Linux**: Ubuntu 20.04+, Debian 10+, RHEL 7+, Fedora 30+, CentOS 7+
- **macOS**: macOS 11+ (Big Sur and later)
  - Intel: `testsmith-macos-amd64`
  - Apple Silicon: `testsmith-macos-arm64`
- **Windows**: Windows 10, Windows 11, Windows Server 2019+
