# Contributing to Sercha CLI

Thank you for your interest in contributing to Sercha CLI! This document provides guidelines and instructions for contributing.

## Table of Contents

- [Getting Started](#getting-started)
- [Project Structure](#project-structure)
- [Development Workflow](#development-workflow)
- [Branch Naming](#branch-naming)
- [Commit Messages](#commit-messages)
- [Pull Requests](#pull-requests)
- [Running CI Locally](#running-ci-locally)
- [Testing](#testing)
- [Release Process](#release-process)
- [Governance](#governance)

## Getting Started

### Fork and Clone

1. Fork the repository on GitHub
2. Clone your fork locally:
   ```bash
   git clone https://github.com/YOUR_USERNAME/sercha-cli.git
   cd sercha-cli
   ```
3. Add the upstream remote:
   ```bash
   git remote add upstream https://github.com/custodia-labs/sercha-cli.git
   ```
4. Keep your fork synced:
   ```bash
   git fetch upstream
   git checkout main
   git merge upstream/main
   ```

### Prerequisites

- Go 1.25 or later
- CGO enabled (for C++ integration)
- Xapian library installed:
  - macOS: `brew install xapian`
  - Ubuntu/Debian: `apt install libxapian-dev`
  - RHEL/CentOS: `yum install xapian-core-devel`
- GoReleaser (for release testing)

## Project Structure

```
sercha-cli/
├── cmd/
│   └── sercha/
│       └── main.go          # CLI entry point
├── internal/                 # Private application code
│   ├── adapters/            # Hexagonal architecture adapters
│   │   ├── driven/          # Infrastructure (storage, external services)
│   │   └── driving/         # Entry points (CLI, TUI)
│   └── core/                # Business logic
│       ├── domain/          # Domain models
│       └── ports/           # Interface definitions
├── .github/
│   ├── workflows/           # GitHub Actions
│   │   ├── release.yml      # Release automation
│   │   └── go-ci.yml        # CI checks
│   ├── PULL_REQUEST_TEMPLATE/
│   └── ISSUE_TEMPLATE/
├── .goreleaser.yml          # GoReleaser configuration
├── VERSION                  # Current version
├── go.mod                   # Go module definition
├── LICENSE
└── README.md
```

## Development Workflow

### Daily Development

```bash
# Start work
git checkout main
git pull upstream main
git checkout -b feat/my-feature

# Make changes
# ...edit files...

# Test
go mod tidy && go vet ./... && go test ./...

# Commit
git add .
git commit -m "feat(cli): add new feature"

# Push and PR
git push origin feat/my-feature
```

**All code changes must go through pull requests and pass CI.**

## Branch Naming

Use the following pattern for all branches:

```
type/short-description
```

### Branch Types

| Type | Description | Example |
|------|-------------|---------|
| `feat` | New feature | `feat/add-config-support` |
| `fix` | Bug fix | `fix/linux-arm64-cross-compile` |
| `docs` | Documentation | `docs/update-installation` |
| `style` | Code style/formatting | `style/format-main-package` |
| `refactor` | Code refactoring | `refactor/extract-parser-module` |
| `perf` | Performance improvement | `perf/optimize-search-query` |
| `test` | Tests | `test/add-cli-unit-tests` |
| `chore` | Maintenance | `chore/update-dependencies` |

### Examples

```bash
# Good branch names
feat/add-search-subcommand
fix/null-pointer-in-parser
docs/contributing-guide

# Bad branch names
my-feature          # Missing type
FEAT/add-feature    # Uppercase type
feat/Add_Feature    # Underscores and mixed case
```

## Commit Messages

We follow [Conventional Commits](https://www.conventionalcommits.org/) specification.

### Format

```
type(scope): summary

[optional body]

[optional footer(s)]
```

### Scope

The scope should be the module or area affected:

- `cli` - Command-line interface
- `parser` - Parsing logic
- `config` - Configuration handling
- `build` - Build system
- `deps` - Dependencies

### Examples

```bash
# Simple commit
feat(cli): add search subcommand

# With body
fix(parser): correct null dereference

The parser was not checking for nil values when processing
empty input strings. This caused panics in production.

Fixes #123

# Breaking change
feat(api)!: change response format

BREAKING CHANGE: The API response now returns an array
instead of an object for list endpoints.
```

### Rules

1. **Use imperative mood** - "add" not "added" or "adds"
2. **Don't capitalize** - "add feature" not "Add feature"
3. **No period at end** - "add feature" not "add feature."
4. **Keep summary under 72 characters**
5. **Reference issues** when applicable

### Git Commit Template

Enable the commit template for guidance:

```bash
git config commit.template .gitmessage.txt
```

### Commit Hook

Install the commit message validation hook:

```bash
cp .github/hooks/commit-msg .git/hooks/commit-msg
chmod +x .git/hooks/commit-msg
```

This will reject commits that don't follow Conventional Commits format.

## Pull Requests

### Before Opening a PR

1. **Sync with main**
   ```bash
   git fetch upstream
   git rebase upstream/main
   ```

2. **Run all checks**
   ```bash
   go mod tidy
   go vet ./...
   go test ./...
   go build ./...
   ```

3. **Ensure clean commits** - Squash WIP commits if needed

### PR Requirements

| Requirement | Description |
|-------------|-------------|
| CI Passing | All GitHub Actions checks must be green |
| Review | At least one approving review required |
| Up to Date | Branch must be current with `main` |
| Template | Use appropriate PR template |
| Description | Clear explanation of changes |

### PR Templates

Select the appropriate template from `.github/PULL_REQUEST_TEMPLATE/`:

- **feature.md** - For new features
- **bugfix.md** - For bug fixes
- **release.md** - For version bumps

### Review Process

1. Open PR with appropriate template
2. Wait for CI checks to pass
3. Request review from maintainers
4. Address feedback with additional commits
5. Once approved, maintainer will merge

### VERSION Changes

Pull requests that modify the `VERSION` file:
- Require owner/maintainer approval
- Must use the `release.md` template
- Should only contain the version bump (no other changes)

## Running CI Locally

Before submitting a PR, run the same checks that CI runs:

```bash
# Build
go build ./...

# Run tests
go test ./...

# Run vet
go vet ./...

# Tidy modules
go mod tidy
```

### GoReleaser Dry Run

Test the release configuration:

```bash
goreleaser check
goreleaser build --snapshot --clean --single-target
```

## Testing

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/...
```

### Writing Tests

- Place tests in `_test.go` files alongside the code
- Use table-driven tests where appropriate
- Aim for meaningful test coverage

## Release Process

Releases are fully automated through GitHub Actions.

### Release Steps

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│ Bump        │ --> │ Create PR   │ --> │ Merge to    │ --> │ Release     │
│ VERSION     │     │ (release    │     │ main        │     │ workflow    │
│             │     │ template)   │     │             │     │ triggered   │
└─────────────┘     └─────────────┘     └─────────────┘     └─────────────┘
                                                                  │
                                                                  v
┌─────────────┐     ┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│ Users       │ <-- │ Packages    │ <-- │ GoReleaser  │ <-- │ Tag created │
│ install     │     │ published   │     │ builds      │     │ & pushed    │
└─────────────┘     └─────────────┘     └─────────────┘     └─────────────┘
```

### Detailed Steps

1. **Bump VERSION**

   Edit the `VERSION` file with the new version:
   ```bash
   echo "1.2.0" > VERSION
   ```

2. **Create Release PR**

   - Create branch: `chore/release-1.2.0`
   - Use the `release.md` PR template
   - Request owner approval

3. **Merge to Main**

   Once approved, merge the PR to `main`.

4. **Automated Release Pipeline**

   The `release.yml` workflow automatically:
   - Creates git tag `v1.2.0` and pushes it
   - Builds binaries for all platforms (darwin/linux x amd64/arm64)
   - Creates GitHub Release with assets
   - Publishes Homebrew formula
   - Uploads to Cloudsmith (deb/rpm)

### What Gets Published

| Platform | Artifacts |
|----------|-----------|
| GitHub | Release page with tar.gz archives |
| Homebrew | Formula in `custodia-labs/homebrew-sercha` |
| Cloudsmith | deb and rpm packages |

### Version Format

Follow [Semantic Versioning](https://semver.org/):

```
MAJOR.MINOR.PATCH
```

- **MAJOR** - Breaking changes
- **MINOR** - New features (backwards compatible)
- **PATCH** - Bug fixes (backwards compatible)

### Pre-release Versions

For pre-releases, use suffixes:
- `1.0.0-alpha.1`
- `1.0.0-beta.1`
- `1.0.0-rc.1`

## Governance

### Roles

**Maintainers** have write access to the repository and are responsible for:
- Reviewing and merging pull requests
- Triaging issues
- Ensuring code quality and consistency
- Helping contributors

**Release Manager** (currently the project owner) is responsible for:
- Approving VERSION changes
- Coordinating releases
- Monitoring release pipelines

**Contributors** are community members who contribute through:
- Code contributions
- Documentation improvements
- Bug reports and feature requests
- Helping other users

### Decision Making

- **Routine decisions** (bug fixes, minor improvements) are made by individual maintainers through PR review
- **Significant decisions** (new features, breaking changes, architecture) require discussion in a GitHub issue and maintainer consensus
- **Disputes** are resolved through discussion; project owner makes final decision if needed

### Becoming a Maintainer

Contributors may be invited based on:
- Consistent, high-quality contributions
- Understanding of project goals and conventions
- Positive interactions with the community

## Questions?

If you have questions, please open an issue or reach out to the maintainers.

See also:
- [Code of Conduct](CODE_OF_CONDUCT.md)
