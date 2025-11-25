# Apple Code Signing Setup

## Current Status

Code signing and notarization are **commented out** in `.goreleaser.yml`.

The pipeline is ready to support signing - macOS builds run natively on `macos-latest` with CGO enabled.

## Prerequisites

1. **Apple Developer Account** ($99/year)
2. **Developer ID Application Certificate**
3. **App-specific password** for notarization
4. **Team ID** (10-character string)

## Steps to Enable

### 1. Export Certificate

On your Mac with Xcode installed:

```bash
# Export your Developer ID Application certificate to .p12
# Keychain Access > My Certificates > Right-click > Export
# Save as developer-id.p12 with a password
```

### 2. Add GitHub Secrets

Add these secrets to your repository:

- `APPLE_CERTIFICATE_P12` - Base64-encoded .p12 file:
  ```bash
  base64 -i developer-id.p12 | pbcopy
  ```
- `APPLE_CERTIFICATE_PASSWORD` - Password for the .p12 file
- `APPLE_ID` - Your Apple ID email
- `APPLE_PASSWORD` - App-specific password (generate at appleid.apple.com)
- `APPLE_TEAM_ID` - Your 10-character Team ID

### 3. Update Workflow

Add certificate import step to `.github/workflows/release.yml` in the `build-macos` job:

```yaml
- name: Import Code-Signing Certificate
  uses: apple-actions/import-codesign-certs@v2
  with:
    p12-file-base64: ${{ secrets.APPLE_CERTIFICATE_P12 }}
    p12-password: ${{ secrets.APPLE_CERTIFICATE_PASSWORD }}
```

Add secrets to the build step:

```yaml
- name: Build macOS partials
  uses: goreleaser/goreleaser-action@v6
  with:
    distribution: goreleaser-pro
    version: "~> v2"
    args: release --split --id darwin-amd64 --id darwin-arm64
  env:
    GORELEASER_KEY: ${{ secrets.GORELEASER_KEY }}
    APPLE_ID: ${{ secrets.APPLE_ID }}
    APPLE_PASSWORD: ${{ secrets.APPLE_PASSWORD }}
    APPLE_TEAM_ID: ${{ secrets.APPLE_TEAM_ID }}
```

### 4. Uncomment Signing in .goreleaser.yml

Update the `signs:` section with your team name:

```yaml
signs:
  - id: macos
    ids:
      - darwin-amd64
      - darwin-arm64
    signature: "${artifact}.sig"
    cmd: codesign
    args:
      - --sign
      - "Developer ID Application: YOUR_TEAM_NAME"  # Replace with your team name
      - --timestamp
      - --options
      - runtime
      - --entitlements
      - signing/entitlements.plist
      - --verbose
      - "${artifact}"
    artifacts: binary
```

Uncomment the `notarize:` section:

```yaml
notarize:
  macos:
    enabled: true
    sign: macos
    ids:
      - darwin-amd64
      - darwin-arm64
```

### 5. Test

Create a test release to verify signing and notarization work.

## References

- [GoReleaser Signing Docs](https://goreleaser.com/customization/sign/)
- [GoReleaser Notarization Docs](https://goreleaser.com/customization/notarize/)
- [Apple Notarization Guide](https://developer.apple.com/documentation/security/notarizing_macos_software_before_distribution)
