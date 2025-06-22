# Versioning and Release Process

This document describes the versioning and release process for the Flux Language.

## Overview

Flux Language uses [Semantic Versioning](https://semver.org/) with automated releases based on [Conventional Commits](https://www.conventionalcommits.org/).

## Version Format

Versions follow the format: `MAJOR.MINOR.PATCH`

- **MAJOR**: Breaking changes (backward incompatible)
- **MINOR**: New features (backward compatible)
- **PATCH**: Bug fixes (backward compatible)

## Conventional Commits

We use conventional commit messages to automatically determine version bumps:

### Commit Message Format

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

### Commit Types

| Type | Description | Version Bump |
|------|-------------|--------------|
| `feat` | New feature | Minor |
| `fix` | Bug fix | Patch |
| `perf` | Performance improvement | Patch |
| `feat!` | Breaking feature | Major |
| `fix!` | Breaking fix | Major |
| `BREAKING CHANGE:` | Breaking change | Major |

### Examples

```bash
# Minor version bump (new feature)
git commit -m "feat: add new compilation optimization"
git commit -m "feat(parser): support for new syntax"

# Patch version bump (bug fix)
git commit -m "fix: resolve memory leak in compiler"
git commit -m "perf: improve lexer performance"

# Major version bump (breaking change)
git commit -m "feat!: change API for type system"
git commit -m "fix!: remove deprecated functions"

# Other commits (no version bump)
git commit -m "docs: update README"
git commit -m "ci: update workflow"
git commit -m "refactor: clean up parser code"
```

## Automated Release Process

### Workflow Triggers

The release workflow is triggered by:
1. **Push to main branch**: Analyzes commits and creates release if needed
2. **Manual trigger**: For emergency releases

### Release Steps

1. **Test**: Run all tests and checks
2. **Version Analysis**: Determine version bump based on commit messages
3. **Build**: Create binaries for multiple platforms
4. **Release**: Create GitHub release with binaries

### Supported Platforms

| OS | Architecture | Binary Name |
|----|--------------|-------------|
| Linux | amd64 | `flux-linux-amd64` |
| Linux | arm64 | `flux-linux-arm64` |
| macOS | amd64 | `flux-darwin-amd64` |
| macOS | arm64 | `flux-darwin-arm64` |
| Windows | amd64 | `flux-windows-amd64.exe` |

## Manual Version Management

### Using the Version Script

We provide a helper script for manual version management:

```bash
# Show current version
./scripts/version.sh current

# Suggest next version based on commits
./scripts/version.sh suggest

# Create a new version
./scripts/version.sh create 1.2.3

# List all versions
./scripts/version.sh list
```

### Manual Release Process

If you need to create a release manually:

1. **Ensure clean state**:
   ```bash
   git status  # Should be clean
   git pull origin main
   ```

2. **Create version tag**:
   ```bash
   ./scripts/version.sh create 1.2.3
   ```

3. **Push tag to trigger release**:
   ```bash
   git push origin v1.2.3
   ```

## Version Information

### Build-time Version Embedding

Version information is embedded at build time using Go build flags:

```bash
go build -ldflags="-X main.Version=v1.2.3 -X main.Commit=abc1234 -X main.Date=2024-01-01T12:00:00Z"
```

### Runtime Version Information

Users can check version information:

```bash
# Show version information
flux version

# Output:
# Flux Language v1.2.3
# Commit: abc1234
# Build Date: 2024-01-01T12:00:00Z
```

## Release Artifacts

Each release includes:

1. **Source code** (automatic GitHub archives)
2. **Binaries** for all supported platforms
3. **Checksums** for verification
4. **Changelog** generated from commit messages

### Binary Naming Convention

```
flux-<os>-<arch>[.exe]
```

Examples:
- `flux-linux-amd64`
- `flux-darwin-arm64`
- `flux-windows-amd64.exe`

### Archive Format

- **Linux/macOS**: `.tar.gz`
- **Windows**: `.zip`

## Changelog Generation

Changelogs are automatically generated from commit messages, organized by type:

- ✨ **Features**: `feat:` commits
- 🐛 **Bug Fixes**: `fix:` commits  
- 🚀 **Performance**: `perf:` commits
- 📝 **Other Changes**: Other conventional commits

## Best Practices

### For Contributors

1. **Use conventional commits** for all changes
2. **Be descriptive** in commit messages
3. **Group related changes** in single commits
4. **Test thoroughly** before pushing to main

### For Maintainers

1. **Review commit messages** during PR review
2. **Squash and merge** with conventional commit format
3. **Monitor releases** for any issues
4. **Update documentation** for breaking changes

### Breaking Changes

When introducing breaking changes:

1. **Use `!` suffix** in commit type: `feat!:` or `fix!:`
2. **Include `BREAKING CHANGE:`** in commit footer
3. **Update documentation** accordingly
4. **Consider deprecation** warnings first

Example:
```
feat!: change configuration file format

BREAKING CHANGE: Configuration files now use YAML instead of JSON.
Migration guide available in MIGRATION.md.
```

## Troubleshooting

### Release Not Created

If a release isn't created automatically:

1. Check that commits follow conventional format
2. Verify workflow ran successfully in Actions tab
3. Ensure you have the right permissions
4. Check for any test failures

### Version Script Issues

```bash
# Make sure script is executable
chmod +x scripts/version.sh

# Check Git setup
git config --list | grep user

# Verify you're on main branch
git branch --show-current
```

### Build Failures

Common issues:
1. **Go version mismatch**: Update workflow if Go version changes
2. **Missing dependencies**: Run `go mod tidy`
3. **Test failures**: Fix tests before release
4. **Linting errors**: Run `gofmt -s -w .`

## Configuration

### Workflow Configuration

The release workflow is configured in `.github/workflows/build-main.yml`:

- **Go version**: Set in `GO_VERSION` environment variable
- **Platforms**: Configured in build matrix
- **Artifacts**: Uploaded as release assets

### Permissions Required

The workflow needs:
- `contents: write` - Create releases and upload assets
- `actions: read` - Access workflow artifacts

## Security Considerations

1. **Signed releases**: Consider GPG signing for releases
2. **Checksums**: Always verify binary checksums
3. **Supply chain**: Use dependabot for dependency updates
4. **Secrets**: Never commit secrets or tokens

## Future Improvements

Potential enhancements:
1. **Pre-release versions**: Support for alpha/beta releases
2. **Release candidates**: Automated RC creation
3. **Rollback mechanism**: Quick rollback for bad releases
4. **Mirror distributions**: Publish to package managers
5. **Security scanning**: Automated vulnerability checks 