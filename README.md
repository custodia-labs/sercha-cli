# sercha-cli

Sercha CLI - a unified, compact version for local, private search

## Installation

### Homebrew (macOS)

```bash
brew install custodia-labs/sercha/sercha
```

### From Source

```bash
go install github.com/custodia-labs/sercha-cli/cmd/sercha@latest
```

## Usage

```bash
sercha --version
```

## Development

### Prerequisites

- Go 1.25 or later
- CGO enabled

### Build

```bash
go build -o sercha ./cmd/sercha/main.go
```

### Test

```bash
go test ./...
```

### Run Locally

```bash
go run ./cmd/sercha/main.go
```

For more detailed development instructions, see [GUIDELINES.md](GUIDELINES.md).

## Contributing

We welcome contributions! Please read:

- [Contributing Guide](CONTRIBUTING.md) - How to contribute
- [Development Workflow](DEVELOPMENT_WORKFLOW.md) - Branch naming, commits, PRs, releases
- [Code of Conduct](CODE_OF_CONDUCT.md) - Community standards
- [Governance](GOVERNANCE.md) - Project governance

### Quick Links

- [PR Templates](.github/PULL_REQUEST_TEMPLATE/) - Use these when opening PRs
- [Issue Templates](.github/ISSUE_TEMPLATE/) - Use these when reporting bugs or requesting features

## License

MIT License - see [LICENSE](LICENSE) for details.
