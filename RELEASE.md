# Release Process

This project uses automated semantic versioning and release management with `go-semantic-release`.

## How It Works

When you push to `main`, the release workflow automatically:
1. **Analyzes commit messages** to determine version bump
2. **Creates a git tag** based on conventional commits
3. **Runs GoReleaser** with cross-compilation toolchains to build binaries
4. **Creates a GitHub release** with:
   - Generated changelog
   - Built binaries (Linux, macOS, Windows - all amd64/arm64)
   - Archives (tar.gz for Linux/macOS, zip for Windows)
   - Checksums

## Commit Message Format

Use [Conventional Commits](https://www.conventionalcommits.org/) format:

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Types that trigger releases:

- **feat**: A new feature (triggers MINOR version bump)
  ```
  feat: add support for custom networks
  feat(ui): add dark mode toggle
  ```

- **fix**: A bug fix (triggers PATCH version bump)
  ```
  fix: resolve memory leak in event handler
  fix(logs): correct timestamp formatting
  ```

- **perf**: Performance improvements (triggers PATCH version bump)
  ```
  perf: optimize transaction processing
  ```

### Breaking changes (triggers MAJOR version bump):

Add `BREAKING CHANGE:` in the footer or use `!` after type:

```
feat!: redesign configuration API

BREAKING CHANGE: Configuration format has changed from YAML to TOML
```

### Types that DON'T trigger releases:

- **docs**: Documentation changes
- **chore**: Maintenance tasks
- **test**: Test updates
- **ci**: CI/CD changes
- **refactor**: Code refactoring without behavior changes
- **style**: Code style changes

## Example Workflow

1. **Make changes and commit with conventional format:**
   ```bash
   git add .
   git commit -m "feat: add transaction filtering"
   git push origin main
   ```

2. **CI runs automatically:**
   - Linting, tests, and coverage checks run
   - If all pass, `go-semantic-release` analyzes commits
   - A new version tag is created (e.g., `v1.2.0`)
   - GoReleaser builds binaries for all platforms
   - Binaries are uploaded to GitHub release
   - Users can download pre-built binaries

## Manual Release (if needed)

To test GoReleaser locally:

```bash
# Install goreleaser
go install github.com/goreleaser/goreleaser@latest

# Test the build (doesn't publish)
goreleaser release --snapshot --clean

# Check the dist/ folder for built binaries
ls -la dist/
```

## Supported Platforms

GoReleaser builds for:
- **Linux**: amd64 (x86_64), arm64 (aarch64)
- **macOS**: amd64 (Intel), arm64 (Apple Silicon M1/M2/M3)
- **Windows**: amd64 (x86_64), arm64

> **Note**: Uses `goreleaser-cross` Docker image with pre-configured cross-compilation toolchains (similar to Flow CLI) to build for all platforms with CGO support.

## Version Numbers

Following [Semantic Versioning](https://semver.org/):
- **MAJOR**: Breaking changes (1.0.0 → 2.0.0)
- **MINOR**: New features (1.0.0 → 1.1.0)
- **PATCH**: Bug fixes (1.0.0 → 1.0.1)

## First Release

To create your first release:

```bash
# Make sure you're on main branch
git checkout main

# Commit with a feat or fix
git commit --allow-empty -m "feat: initial release"
git push origin main
```

This will create version `v1.0.0` automatically.
