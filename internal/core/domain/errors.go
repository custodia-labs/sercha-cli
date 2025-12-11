package domain

import "errors"

// Domain errors represent business logic failures.
// These are distinct from infrastructure errors.
var (
	// ErrNotFound indicates a requested entity does not exist.
	ErrNotFound = errors.New("not found")

	// ErrAlreadyExists indicates an entity already exists.
	ErrAlreadyExists = errors.New("already exists")

	// ErrInvalidInput indicates malformed or invalid input.
	ErrInvalidInput = errors.New("invalid input")

	// ErrNotImplemented indicates functionality is not yet available.
	ErrNotImplemented = errors.New("not implemented")

	// ErrUnsupportedType indicates an unknown connector or normaliser type.
	ErrUnsupportedType = errors.New("unsupported type")

	// ErrSyncInProgress indicates a sync is already running.
	ErrSyncInProgress = errors.New("sync in progress")

	// ErrLLMUnavailable indicates the LLM service is not configured.
	// Features requiring LLM (query rewriting, summarisation) are disabled.
	ErrLLMUnavailable = errors.New("LLM service unavailable")

	// ErrEmbeddingUnavailable indicates the embedding service is not configured.
	// Vector/semantic search is disabled without embeddings.
	ErrEmbeddingUnavailable = errors.New("embedding service unavailable")

	// ErrSearchUnavailable indicates the search engine is not configured.
	// Full-text/keyword search is disabled.
	ErrSearchUnavailable = errors.New("search engine unavailable")

	// ErrVectorIndexUnavailable indicates the vector index is not configured.
	// Semantic similarity search is disabled.
	ErrVectorIndexUnavailable = errors.New("vector index unavailable")

	// Authentication Errors.

	// ErrAuthRequired indicates the connector requires authentication but none is configured.
	ErrAuthRequired = errors.New("authentication required")

	// ErrAuthExpired indicates the authentication has expired and refresh failed.
	ErrAuthExpired = errors.New("authentication expired")

	// ErrAuthInvalid indicates the authentication credentials are invalid.
	ErrAuthInvalid = errors.New("authentication invalid")

	// ErrTokenRefreshFailed indicates token refresh operation failed.
	ErrTokenRefreshFailed = errors.New("token refresh failed")

	// Connector Errors.

	// ErrConnectorValidation indicates connector validation failed.
	// The source is misconfigured or credentials are invalid.
	ErrConnectorValidation = errors.New("connector validation failed")

	// ErrConnectorClosed indicates the connector has been closed.
	ErrConnectorClosed = errors.New("connector closed")

	// ErrRateLimited indicates the API rate limit was exceeded.
	ErrRateLimited = errors.New("rate limited")

	// ErrAuthProviderInUse indicates an auth provider cannot be deleted because sources depend on it.
	ErrAuthProviderInUse = errors.New("auth provider is in use by one or more sources")
)
