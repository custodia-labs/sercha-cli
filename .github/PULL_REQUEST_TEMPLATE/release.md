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

After this PR is merged:

1. `tag-on-version-change.yml` will create git tag `vX.Y.Z`
2. `release.yml` will build and publish:
   - GitHub Release
   - Homebrew cask
   - Cloudsmith packages

## Approvals

This PR requires owner/maintainer approval.
