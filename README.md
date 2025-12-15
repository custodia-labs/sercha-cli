<!-- Improved compatibility of back to top link -->
<a id="readme-top"></a>

<!-- PROJECT SHIELDS -->
<p align="center">

[![GitHub Release][release-shield]][release-url]
[![License][license-shield]][license-url]
[![Go Report Card][goreport-shield]][goreport-url]
[![Release Workflow][release-workflow-shield]][release-workflow-url]
[![CI Workflow][ci-workflow-shield]][ci-workflow-url]
[![Contributions Welcome][contributions-shield]][contributions-url]

</p>

<!-- BANNER -->
<p align="center">
  <img src=".github/assets/banner.png" alt="Sercha" width="100%">
</p>

<!-- PROJECT TITLE -->
<div align="center">

  <p align="center">
    A unified, compact version for local, private search
    <br />
    <br />
    <a href="https://github.com/custodia-labs/sercha-cli/issues/new?labels=bug&template=bug_report.md">Report Bug</a>
    &middot;
    <a href="https://github.com/custodia-labs/sercha-cli/issues/new?labels=enhancement&template=feature_request.md">Request Feature</a>
  </p>
</div>

<!-- TABLE OF CONTENTS -->
<details>
  <summary>Table of Contents</summary>
  <ol>
    <li>
      <a href="#about-the-project">About The Project</a>
      <ul>
        <li><a href="#built-with">Built With</a></li>
      </ul>
    </li>
    <li>
      <a href="#getting-started">Getting Started</a>
      <ul>
        <li><a href="#macos">macOS</a></li>
        <li><a href="#ubuntu--debian">Ubuntu / Debian</a></li>
        <li><a href="#rhel--centos--fedora">RHEL / CentOS / Fedora</a></li>
        <li><a href="#direct-binary-download">Direct Binary Download</a></li>
        <li><a href="#from-source">From Source</a></li>
      </ul>
    </li>
    <li><a href="#usage">Usage</a></li>
    <li><a href="#development">Development</a></li>
    <li><a href="#contributing">Contributing</a></li>
    <li><a href="#license">License</a></li>
  </ol>
</details>

<!-- ABOUT THE PROJECT -->
## About The Project

Sercha CLI is a powerful, privacy-focused search tool designed for local environments. Built with performance and security in mind, it provides fast, efficient search capabilities without relying on external services or compromising your data privacy.

**Why Sercha?**
* **Privacy First**: All searches happen locally on your machine - your data never leaves your control
* **Fast & Efficient**: Optimized for speed with CGO-enabled performance
* **Cross-Platform**: Native builds for macOS (Apple Silicon) and Linux (x86_64 & ARM64)
* **Easy Installation**: Multiple installation methods including Homebrew, apt, yum, and direct binaries

<p align="right">(<a href="#readme-top">back to top</a>)</p>

### Built With

* [![Go][Go-badge]][Go-url] - Go 1.24+
* **CGO Enabled** - For enhanced performance with C/C++ integration
* **GoReleaser Pro** - Enterprise-grade release automation
* **GitHub Actions** - Automated CI/CD pipeline

<p align="right">(<a href="#readme-top">back to top</a>)</p>

<!-- GETTING STARTED -->
## Getting Started

Choose your preferred installation method below based on your operating system.

### macOS

#### Homebrew (Recommended)

```bash
brew tap custodia-labs/sercha
brew install sercha
```

**Note**: On first run, macOS may block the binary. If you see "killed", run:
```bash
xattr -d com.apple.quarantine $(which sercha)
```

<p align="right">(<a href="#readme-top">back to top</a>)</p>

### Ubuntu / Debian

#### Ubuntu 24.04 (Noble) and 22.04 (Jammy)

```bash
curl -1sLf 'https://dl.cloudsmith.io/public/custodia-labs/sercha/setup.deb.sh' | sudo bash
sudo apt-get install -y sercha
```

<p align="right">(<a href="#readme-top">back to top</a>)</p>

### RHEL / CentOS / Fedora

```bash
curl -1sLf 'https://dl.cloudsmith.io/public/custodia-labs/sercha/setup.rpm.sh' | sudo bash
sudo yum install -y sercha
```

<p align="right">(<a href="#readme-top">back to top</a>)</p>

### Direct Binary Download

Download the latest release for your platform from [GitHub Releases][release-url]:

- **macOS (Apple Silicon)**: `sercha_*_darwin_arm64.tar.gz`
- **Linux (x86_64)**: `sercha_*_linux_amd64.tar.gz`
- **Linux (ARM64)**: `sercha_*_linux_arm64.tar.gz`

**Note**: Binary downloads require Xapian to be installed manually:
- macOS: `brew install xapian`
- Ubuntu/Debian: `sudo apt install libxapian30`
- RHEL/CentOS: `sudo yum install xapian-core-libs`

Extract and move to your PATH:
```bash
tar -xzf sercha_*.tar.gz
sudo mv sercha /usr/local/bin/
```

<p align="right">(<a href="#readme-top">back to top</a>)</p>

### From Source

Requires Go 1.24 or later:

```bash
go install github.com/custodia-labs/sercha-cli/cmd/sercha@latest
```

<p align="right">(<a href="#readme-top">back to top</a>)</p>

<!-- USAGE -->
## Usage

### Quick Start

```bash
# 1. Add a local folder as a source
sercha source add filesystem -c path=~/Documents

# 2. Sync documents from the source
sercha sync

# 3. Search your documents
sercha search "meeting notes"

# 4. Or launch the interactive TUI
sercha tui
```

### Documentation

For detailed guides and configuration options, visit the [Sercha Documentation](https://docs.sercha.dev):

- [Quickstart Guide](https://docs.sercha.dev/cli/quickstart) - Get up and running in 5 minutes
- [Supported Connectors](https://docs.sercha.dev/cli/supported-connectors) - GitHub, Google Drive, Gmail, and more
- [AI Models](https://docs.sercha.dev/cli/models/overview) - Text, hybrid, and AI-assisted search
- [TUI Guide](https://docs.sercha.dev/cli/tui/overview) - Interactive terminal interface
- [MCP Server](https://docs.sercha.dev/cli/mcp/overview) - AI tool integration

<p align="right">(<a href="#readme-top">back to top</a>)</p>

<!-- DEVELOPMENT -->
## Development

### Prerequisites

- Go 1.24 or later
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

For more detailed development instructions, see [CONTRIBUTING.md](CONTRIBUTING.md).

<p align="right">(<a href="#readme-top">back to top</a>)</p>

<!-- CONTRIBUTING -->
## Contributing

Contributions are what make the open source community such an amazing place to learn, inspire, and create. Any contributions you make are **greatly appreciated**.

If you have a suggestion that would make this better, please fork the repo and create a pull request. You can also simply open an issue with the tag "enhancement".

**Please read:**

- [Contributing Guide](CONTRIBUTING.md) - How to contribute, development workflow, and governance
- [Code of Conduct](CODE_OF_CONDUCT.md) - Community standards

### Quick Links

- [PR Templates](.github/PULL_REQUEST_TEMPLATE/) - Use these when opening PRs
- [Issue Templates](.github/ISSUE_TEMPLATE/) - Use these when reporting bugs or requesting features

<p align="right">(<a href="#readme-top">back to top</a>)</p>

<!-- LICENSE -->
## License

Distributed under the Apache 2.0 License. See [LICENSE](LICENSE) for details.

<p align="right">(<a href="#readme-top">back to top</a>)</p>

<!-- MARKDOWN LINKS & IMAGES -->
[release-shield]: https://img.shields.io/github/v/release/custodia-labs/sercha-cli
[release-url]: https://github.com/custodia-labs/sercha-cli/releases/latest
[license-shield]: https://img.shields.io/badge/License-Apache_2.0-blue.svg
[license-url]: https://opensource.org/licenses/Apache-2.0
[goreport-shield]: https://goreportcard.com/badge/github.com/custodia-labs/sercha-cli
[goreport-url]: https://goreportcard.com/report/github.com/custodia-labs/sercha-cli
[release-workflow-shield]: https://github.com/custodia-labs/sercha-cli/actions/workflows/release.yml/badge.svg
[release-workflow-url]: https://github.com/custodia-labs/sercha-cli/actions/workflows/release.yml
[ci-workflow-shield]: https://github.com/custodia-labs/sercha-cli/actions/workflows/go-ci.yml/badge.svg
[ci-workflow-url]: https://github.com/custodia-labs/sercha-cli/actions/workflows/go-ci.yml
[contributions-shield]: https://img.shields.io/badge/contributions-welcome-brightgreen.svg
[contributions-url]: https://github.com/custodia-labs/sercha-cli/blob/main/CONTRIBUTING.md
[Go-badge]: https://img.shields.io/badge/Go-00ADD8?style=flat&logo=go&logoColor=white
[Go-url]: https://go.dev/
