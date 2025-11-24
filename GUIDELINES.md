# Contributor Quickstart

This guide helps you get up and running with Sercha CLI development quickly.

## Project Structure

```
sercha-cli/
├── cmd/
│   └── sercha/
│       └── main.go          # CLI entry point
├── internal/                 # Private application code
├── pkg/                      # Public library code
├── .github/
│   ├── workflows/           # GitHub Actions
│   │   ├── release.yml      # Release automation
│   │   ├── go-ci.yml        # CI checks
│   │   └── tag-on-version-change.yml
│   ├── PULL_REQUEST_TEMPLATE/
│   └── ISSUE_TEMPLATE/
├── .goreleaser.yml          # GoReleaser configuration
├── VERSION                  # Current version
├── go.mod                   # Go module definition
├── LICENSE
└── README.md
```

## Running the CLI Locally

### Basic Run

```bash
go run ./cmd/sercha/main.go
```

### With Version Info

```bash
go run -ldflags "-X main.version=dev" ./cmd/sercha/main.go
```

## Building

### Standard Build

```bash
go build -o sercha ./cmd/sercha/main.go
```

### Build with CGO

This project uses CGO for C++ integration. Ensure CGO is enabled:

```bash
CGO_ENABLED=1 go build -o sercha ./cmd/sercha/main.go
```

### Build with Version

```bash
go build -ldflags "-s -w -X main.version=1.0.0" -o sercha ./cmd/sercha/main.go
```

## Running Checks

### Go Vet

Static analysis for common mistakes:

```bash
go vet ./...
```

### Go Test

Run all tests:

```bash
go test ./...
```

With verbose output:

```bash
go test -v ./...
```

With coverage:

```bash
go test -cover ./...
```

### Go Build

Verify the project compiles:

```bash
go build ./...
```

### All Checks (CI Equivalent)

```bash
go mod tidy && go vet ./... && go test ./... && go build ./...
```

## GoReleaser

### Validate Configuration

```bash
goreleaser check
```

### Dry-Run Build

Build for your current platform:

```bash
goreleaser build --snapshot --clean --single-target
```

Build all platforms (requires appropriate toolchains):

```bash
goreleaser build --snapshot --clean
```

### Full Dry-Run Release

```bash
goreleaser release --snapshot --clean --skip=publish
```

## Repository Rules

### Pull Request Requirements

- **All changes must go through PRs** - Direct pushes to `main` are not allowed
- **CI must pass** - All checks (build, test, vet) must be green
- **Review required** - At least one approval is needed
- **VERSION changes** - Require owner/maintainer approval

### Tag Protection

- Tags are created automatically when VERSION changes are merged to `main`
- Manual tag creation is discouraged
- Tags trigger the release workflow

### Branch Protection

- `main` branch is protected
- Force pushes are disabled
- Branch must be up to date before merging

## Local Development Workflow

1. **Sync your fork**
   ```bash
   git fetch upstream
   git checkout main
   git merge upstream/main
   ```

2. **Create a branch**
   ```bash
   git checkout -b feat/my-feature
   ```

3. **Make changes and test**
   ```bash
   # Edit files...
   go mod tidy
   go vet ./...
   go test ./...
   go build ./...
   ```

4. **Commit with conventional format**
   ```bash
   git add .
   git commit -m "feat(cli): add new feature"
   ```

5. **Push and create PR**
   ```bash
   git push origin feat/my-feature
   # Open PR on GitHub
   ```

## Environment Setup

### Required Tools

| Tool | Version | Purpose |
|------|---------|---------|
| Go | 1.25+ | Build and test |
| GoReleaser | Latest | Release automation |
| Git | 2.x | Version control |

### Optional Tools

| Tool | Purpose |
|------|---------|
| golangci-lint | Extended linting |
| pre-commit | Git hooks |

## Getting Help

- Check existing [issues](https://github.com/custodia-labs/sercha-cli/issues)
- Read the [Contributing Guide](CONTRIBUTING.md)
- Review the [Development Workflow](DEVELOPMENT_WORKFLOW.md)
