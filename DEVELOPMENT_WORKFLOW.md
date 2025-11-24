# Development Workflow

This document describes the development workflow for Sercha CLI, including branch naming conventions, commit message format, PR expectations, and the release process.

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
refactor/cli-argument-handling

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

### Types

Same as branch types: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `chore`

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

## Pull Request Expectations

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

## Release Workflow

Releases are fully automated through GitHub Actions.

### Release Steps

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│ Bump        │ --> │ Create PR   │ --> │ Merge to    │ --> │ Auto-tag    │
│ VERSION     │     │ (release    │     │ main        │     │ created     │
│             │     │ template)   │     │             │     │             │
└─────────────┘     └─────────────┘     └─────────────┘     └─────────────┘
                                                                   │
                                                                   v
┌─────────────┐     ┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│ Users       │ <-- │ Packages    │ <-- │ GoReleaser  │ <-- │ Release     │
│ install     │     │ published   │     │ builds      │     │ workflow    │
│             │     │             │     │             │     │ triggered   │
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

4. **Automatic Tag Creation**

   The `tag-on-version-change.yml` workflow:
   - Detects VERSION file change
   - Creates git tag `v1.2.0`
   - Pushes tag to repository

5. **GoReleaser Builds**

   The `release.yml` workflow:
   - Builds binaries for all platforms (darwin/linux × amd64/arm64)
   - Creates GitHub Release with assets
   - Publishes Homebrew cask
   - Uploads to Cloudsmith (deb/rpm)

### What Gets Published

| Platform | Artifacts |
|----------|-----------|
| GitHub | Release page with tar.gz archives |
| Homebrew | Cask in `custodia-labs/homebrew-sercha` |
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

## Quick Reference

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

### Release

```bash
# Bump version
git checkout -b chore/release-X.Y.Z
echo "X.Y.Z" > VERSION
git add VERSION
git commit -m "chore(release): bump version to X.Y.Z"
git push origin chore/release-X.Y.Z
# Open PR with release template
```
