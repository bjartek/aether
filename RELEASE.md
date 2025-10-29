# Release Process

This project uses automated semantic versioning and release management.

## How It Works

### 1. Semantic Release (Automated Versioning)

When you push to `main`, semantic-release automatically:
- Analyzes commit messages to determine version bump
- Generates a changelog
- Creates a git tag
- Creates a GitHub release

### 2. GoReleaser (Binary Distribution)

When a new tag is created, GoReleaser automatically:
- Builds binaries for multiple platforms (Linux, macOS, Windows)
- Creates archives (tar.gz, zip)
- Uploads binaries to the GitHub release
- Generates checksums

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
   - If all pass, semantic-release analyzes commits
   - A new version tag is created (e.g., `v1.2.0`)

3. **Release workflow triggers:**
   - GoReleaser builds binaries for all platforms
   - Binaries are uploaded to GitHub releases
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
- **Linux**: amd64, arm64
- **macOS**: amd64 (Intel), arm64 (Apple Silicon)
- **Windows**: amd64, arm64

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
