// Package google provides shared infrastructure for Google API connectors.
//
// This package contains common utilities used by the gmail, drive, and calendar
// connectors including:
//   - TokenSource adapter to bridge Sercha's TokenProvider to oauth2.TokenSource
//   - Service factories for creating Google API clients
//   - Error handling for common Google API errors (401, 403, 404, 429)
//   - Rate limiting to respect Google API quotas
//
// # Usage
//
// Each Google connector (gmail, drive, calendar) uses this package to create
// authenticated API clients:
//
//	ts := google.NewTokenSource(ctx, tokenProvider)
//	svc, err := google.NewGmailService(ctx, ts)
//
// # OAuth2 Scopes
//
// Google connectors use these scopes:
//   - https://www.googleapis.com/auth/userinfo.email (non-sensitive)
//   - https://www.googleapis.com/auth/userinfo.profile (non-sensitive)
//   - https://www.googleapis.com/auth/gmail.readonly (restricted)
//   - https://www.googleapis.com/auth/drive.readonly (restricted)
//   - https://www.googleapis.com/auth/calendar.readonly (sensitive)
//
// For user-created internal apps, restricted scopes don't require verification.
package google
