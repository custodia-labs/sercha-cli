## Release Version

<!-- Which version is being released? -->

**Version:**

## Changes in This Release

<!-- Summary of what's included in this release -->

### Features

-

### Bug Fixes

-

### Other Changes

-

## Pre-Release Testing

<!-- Confirm testing has been completed -->

- [ ] All CI checks pass
- [ ] GoReleaser dry-run successful (`goreleaser build --snapshot --clean`)
- [ ] Manual testing of key features
- [ ] Version number follows semantic versioning

## Checklist

- [ ] VERSION file updated with new version
- [ ] No other file changes in this PR
- [ ] CHANGELOG updated (if maintained)
- [ ] Release notes prepared

## Post-Merge

After this PR is merged, `release.yml` will automatically:

1. Create and push git tag `vX.Y.Z`
2. Build binaries for all platforms
3. Publish:
   - GitHub Release
   - Homebrew cask
   - Cloudsmith packages

## Approvals

This PR requires owner/maintainer approval.
