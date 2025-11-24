# Contributing to Sercha CLI

Thank you for your interest in contributing to Sercha CLI! This document provides guidelines and instructions for contributing.

## Table of Contents

- [Getting Started](#getting-started)
- [Development Workflow](#development-workflow)
- [Branch Naming](#branch-naming)
- [Commit Messages](#commit-messages)
- [Pull Requests](#pull-requests)
- [Running CI Locally](#running-ci-locally)
- [Testing](#testing)
- [Release Process](#release-process)

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
- GoReleaser (for release testing)

## Development Workflow

1. Create a new branch from `main`
2. Make your changes
3. Run tests and linting locally
4. Commit using conventional commit format
5. Push to your fork
6. Open a pull request

**All code changes must go through pull requests and pass CI.**

## Branch Naming

Use the following pattern for branch names:

```
type/short-description
```

Where `type` is one of:
- `feat` - New feature
- `fix` - Bug fix
- `docs` - Documentation only
- `style` - Code style changes (formatting, etc.)
- `refactor` - Code refactoring
- `perf` - Performance improvements
- `test` - Adding or updating tests
- `chore` - Maintenance tasks

**Examples:**
- `feat/add-config-support`
- `fix/linux-arm64-cross-compile`
- `docs/update-installation`

## Commit Messages

We use [Conventional Commits](https://www.conventionalcommits.org/) format:

```
type(scope): summary
```

**Examples:**
- `feat(cli): add search subcommand`
- `fix(parser): correct null dereference`
- `docs(readme): update installation instructions`
- `test(cli): add unit tests for config loader`

### Rules

- Use imperative mood ("add" not "added")
- Keep summary under 72 characters
- Reference issues when applicable

### Git Commit Template

You can use our commit template:
```bash
git config commit.template .gitmessage.txt
```

## Pull Requests

### Requirements

- All PRs must pass CI checks
- All PRs require at least one review
- VERSION changes require owner approval
- Use the appropriate PR template from `.github/PULL_REQUEST_TEMPLATE/`

### PR Process

1. Ensure your branch is up to date with `main`
2. Open a PR with a clear title and description
3. Select the appropriate PR template (feature, bugfix, or release)
4. Wait for CI to pass
5. Address review feedback
6. Squash and merge when approved

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

Releases are automated through GitHub Actions:

1. **Bump VERSION**: Update the `VERSION` file with the new version number
2. **Submit PR**: Create a PR with the VERSION change (use `release.md` template)
3. **Merge**: After approval and CI passes, merge to `main`
4. **Auto-tag**: The `tag-on-version-change.yml` workflow creates a git tag
5. **Release**: GoReleaser builds and publishes:
   - GitHub Release with binaries
   - Homebrew cask
   - Cloudsmith packages (deb/rpm)

### Version Format

Use semantic versioning: `MAJOR.MINOR.PATCH`

- `MAJOR`: Breaking changes
- `MINOR`: New features (backwards compatible)
- `PATCH`: Bug fixes (backwards compatible)

## Questions?

If you have questions, please open an issue or reach out to the maintainers.

See also:
- [Code of Conduct](CODE_OF_CONDUCT.md)
- [Development Workflow](DEVELOPMENT_WORKFLOW.md)
- [Contributor Guidelines](GUIDELINES.md)
