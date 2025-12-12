// Package microsoft provides OAuth2 and connector support for Microsoft Graph API.
//
// This package provides:
//   - OAuth2 authentication handler for Microsoft identity platform
//   - Rate limiting for Microsoft Graph API requests
//   - Error handling for Microsoft Graph API responses
//   - HTTP client with authentication support
//
// Microsoft Graph endpoints use the "common" tenant for multi-tenant support,
// allowing both personal Microsoft accounts and Azure AD accounts.
//
// # OAuth2 Flow
//
// Microsoft uses standard OAuth2 with PKCE:
//   - Auth URL: https://login.microsoftonline.com/common/oauth2/v2.0/authorize
//   - Token URL: https://login.microsoftonline.com/common/oauth2/v2.0/token
//
// The "offline_access" scope is required for refresh tokens.
//
// # Delta Query
//
// Microsoft Graph supports incremental sync via delta queries:
//   - Mail: /me/mailFolders/{id}/messages/delta
//   - OneDrive: /me/drive/root/delta
//   - Calendar: /me/calendarView/delta
//
// Delta queries return @odata.deltaLink for subsequent requests.
// A 410 Gone response indicates the delta token has expired and a full sync is required.
//
// # Rate Limits
//
// Microsoft Graph allows approximately 10,000 requests per 10 minutes per app.
// This package implements conservative rate limiting to avoid hitting quotas.
package microsoft
