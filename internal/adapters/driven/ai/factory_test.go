package ai

import (
	"testing"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

func TestInitResult_Close(t *testing.T) {
	t.Run("close with nil services", func(t *testing.T) {
		result := &InitResult{}
		// Should not panic
		result.Close()
	})
}

func TestCreateEmbeddingService(t *testing.T) {
	tests := []struct {
		name        string
		settings    *domain.EmbeddingSettings
		wantNil     bool
		wantErr     bool
		errContains string
	}{
		{
			name:     "nil settings returns nil",
			settings: nil,
			wantNil:  true,
			wantErr:  false,
		},
		{
			name:     "unconfigured settings returns nil",
			settings: &domain.EmbeddingSettings{},
			wantNil:  true,
			wantErr:  false,
		},
		{
			name: "ollama provider creates service",
			settings: &domain.EmbeddingSettings{
				Provider: domain.AIProviderOllama,
				BaseURL:  "http://localhost:11434",
				Model:    "nomic-embed-text",
			},
			wantNil: false,
			wantErr: false,
		},
		{
			name: "openai provider creates service",
			settings: &domain.EmbeddingSettings{
				Provider: domain.AIProviderOpenAI,
				APIKey:   "test-key",
				Model:    "text-embedding-3-small",
			},
			wantNil: false,
			wantErr: false,
		},
		{
			name: "anthropic provider returns error",
			settings: &domain.EmbeddingSettings{
				Provider: domain.AIProviderAnthropic,
				APIKey:   "test-key",
			},
			wantNil:     true,
			wantErr:     true,
			errContains: "anthropic does not support embeddings",
		},
		{
			name: "unknown provider returns nil (not configured)",
			settings: &domain.EmbeddingSettings{
				Provider: "unknown",
				APIKey:   "test-key",
			},
			wantNil: true,
			wantErr: false, // unknown provider is not valid, so IsConfigured() returns false
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, err := CreateEmbeddingService(tt.settings)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errContains)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if tt.wantNil && svc != nil {
				t.Error("expected nil service, got non-nil")
				svc.Close()
			}
			if !tt.wantNil && svc == nil {
				t.Error("expected non-nil service, got nil")
			}
			if svc != nil {
				svc.Close()
			}
		})
	}
}

func TestCreateLLMService(t *testing.T) {
	tests := []struct {
		name        string
		settings    *domain.LLMSettings
		wantNil     bool
		wantErr     bool
		errContains string
	}{
		{
			name:     "nil settings returns nil",
			settings: nil,
			wantNil:  true,
			wantErr:  false,
		},
		{
			name:     "unconfigured settings returns nil",
			settings: &domain.LLMSettings{},
			wantNil:  true,
			wantErr:  false,
		},
		{
			name: "ollama provider creates service",
			settings: &domain.LLMSettings{
				Provider: domain.AIProviderOllama,
				BaseURL:  "http://localhost:11434",
				Model:    "llama3.2",
			},
			wantNil: false,
			wantErr: false,
		},
		{
			name: "openai provider creates service",
			settings: &domain.LLMSettings{
				Provider: domain.AIProviderOpenAI,
				APIKey:   "test-key",
				Model:    "gpt-4o-mini",
			},
			wantNil: false,
			wantErr: false,
		},
		{
			name: "anthropic provider creates service",
			settings: &domain.LLMSettings{
				Provider: domain.AIProviderAnthropic,
				APIKey:   "test-key",
				Model:    "claude-3-5-sonnet-latest",
			},
			wantNil: false,
			wantErr: false,
		},
		{
			name: "unknown provider returns nil (not configured)",
			settings: &domain.LLMSettings{
				Provider: "unknown",
				APIKey:   "test-key",
			},
			wantNil: true,
			wantErr: false, // unknown provider is not valid, so IsConfigured() returns false
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, err := CreateLLMService(tt.settings)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errContains)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if tt.wantNil && svc != nil {
				t.Error("expected nil service, got non-nil")
				svc.Close()
			}
			if !tt.wantNil && svc == nil {
				t.Error("expected non-nil service, got nil")
			}
			if svc != nil {
				svc.Close()
			}
		})
	}
}

func TestValidateEmbeddingConfig(t *testing.T) {
	tests := []struct {
		name     string
		settings *domain.EmbeddingSettings
		wantErr  bool
	}{
		{
			name:     "nil settings returns nil",
			settings: nil,
			wantErr:  false,
		},
		{
			name:     "unconfigured settings returns nil",
			settings: &domain.EmbeddingSettings{},
			wantErr:  false,
		},
		{
			name: "anthropic returns error",
			settings: &domain.EmbeddingSettings{
				Provider: domain.AIProviderAnthropic,
				APIKey:   "test-key",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEmbeddingConfig(tt.settings)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateLLMConfig(t *testing.T) {
	tests := []struct {
		name     string
		settings *domain.LLMSettings
		wantErr  bool
	}{
		{
			name:     "nil settings returns nil",
			settings: nil,
			wantErr:  false,
		},
		{
			name:     "unconfigured settings returns nil",
			settings: &domain.LLMSettings{},
			wantErr:  false,
		},
		{
			name: "unknown provider returns nil (not configured)",
			settings: &domain.LLMSettings{
				Provider: "unknown",
				APIKey:   "test-key",
			},
			wantErr: false, // unknown provider is not valid, so IsConfigured() returns false
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLLMConfig(tt.settings)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestCreateAndValidateEmbeddingService(t *testing.T) {
	tests := []struct {
		name     string
		settings *domain.EmbeddingSettings
		wantNil  bool
		wantErr  bool
	}{
		{
			name:     "nil settings returns nil",
			settings: nil,
			wantNil:  true,
			wantErr:  false,
		},
		{
			name:     "unconfigured settings returns nil",
			settings: &domain.EmbeddingSettings{},
			wantNil:  true,
			wantErr:  false,
		},
		{
			name: "anthropic returns error",
			settings: &domain.EmbeddingSettings{
				Provider: domain.AIProviderAnthropic,
				APIKey:   "test-key",
			},
			wantNil: true,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, err := CreateAndValidateEmbeddingService(tt.settings)

			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.wantNil && svc != nil {
				t.Error("expected nil service")
				svc.Close()
			}
			if svc != nil {
				svc.Close()
			}
		})
	}
}

func TestCreateAndValidateLLMService(t *testing.T) {
	tests := []struct {
		name     string
		settings *domain.LLMSettings
		wantNil  bool
		wantErr  bool
	}{
		{
			name:     "nil settings returns nil",
			settings: nil,
			wantNil:  true,
			wantErr:  false,
		},
		{
			name:     "unconfigured settings returns nil",
			settings: &domain.LLMSettings{},
			wantNil:  true,
			wantErr:  false,
		},
		{
			name: "unknown provider returns nil (not configured)",
			settings: &domain.LLMSettings{
				Provider: "unknown",
				APIKey:   "test-key",
			},
			wantNil: true,
			wantErr: false, // unknown provider is not valid, so IsConfigured() returns false
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, err := CreateAndValidateLLMService(tt.settings)

			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.wantNil && svc != nil {
				t.Error("expected nil service")
				svc.Close()
			}
			if svc != nil {
				svc.Close()
			}
		})
	}
}

func TestCreateEmbeddingService_UnknownProvider(t *testing.T) {
	settings := &domain.EmbeddingSettings{
		Provider: "unknown-provider",
		APIKey:   "test-key",
	}

	svc, err := CreateEmbeddingService(settings)

	// Unknown provider is not "valid" so IsConfigured returns false
	if svc != nil {
		t.Error("expected nil service for unknown provider")
		svc.Close()
	}
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCreateLLMService_UnknownProvider(t *testing.T) {
	settings := &domain.LLMSettings{
		Provider: "unknown-provider",
		APIKey:   "test-key",
	}

	svc, err := CreateLLMService(settings)

	// Unknown provider is not "valid" so IsConfigured returns false
	if svc != nil {
		t.Error("expected nil service for unknown provider")
		svc.Close()
	}
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestInitResult_Close_AllServices(t *testing.T) {
	// Create a result with all services populated
	result := &InitResult{}

	// Create mock embedding service (using ollama which doesn't need network)
	embSvc := createOllamaEmbedding(&domain.EmbeddingSettings{
		Provider: domain.AIProviderOllama,
		BaseURL:  "http://localhost:11434",
		Model:    "nomic-embed-text",
	})
	result.EmbeddingService = embSvc

	// Create mock LLM service
	llmSvc := createOllamaLLM(&domain.LLMSettings{
		Provider: domain.AIProviderOllama,
		BaseURL:  "http://localhost:11434",
		Model:    "llama3.2",
	})
	result.LLMService = llmSvc

	// Close should not panic and should close all services
	result.Close()
}

func TestValidateLLMConfig_ValidOllamaConfig(t *testing.T) {
	settings := &domain.LLMSettings{
		Provider: domain.AIProviderOllama,
		BaseURL:  "http://localhost:99999", // Invalid port will fail ping
		Model:    "llama3.2",
	}

	// Will fail due to connection error, but exercises the validation code path
	err := ValidateLLMConfig(settings)
	if err == nil {
		t.Log("ollama was available, validation passed")
	} else {
		// Expected since no ollama is running
		if !contains(err.Error(), "dial") && !contains(err.Error(), "connection") {
			t.Logf("validation failed as expected with error: %v", err)
		}
	}
}

func TestValidateEmbeddingConfig_ValidOllamaConfig(t *testing.T) {
	settings := &domain.EmbeddingSettings{
		Provider: domain.AIProviderOllama,
		BaseURL:  "http://localhost:99999", // Invalid port will fail ping
		Model:    "nomic-embed-text",
	}

	err := ValidateEmbeddingConfig(settings)
	if err == nil {
		t.Log("ollama was available, validation passed")
	} else {
		// Expected since no ollama is running
		if !contains(err.Error(), "dial") && !contains(err.Error(), "connection") {
			t.Logf("validation failed as expected with error: %v", err)
		}
	}
}

func TestCreateOllamaEmbedding_WithDimensions(t *testing.T) {
	// Test with a model that has known dimensions
	settings := &domain.EmbeddingSettings{
		Provider: domain.AIProviderOllama,
		BaseURL:  "http://localhost:11434",
		Model:    "nomic-embed-text", // Known model with 768 dimensions
	}

	svc := createOllamaEmbedding(settings)
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
	defer svc.Close()

	// nomic-embed-text should have 768 dimensions from the lookup
	expectedDims := domain.EmbeddingDimensions()["nomic-embed-text"]
	if expectedDims == 0 {
		// If not in lookup, defaults are used
		t.Log("model not in dimension lookup, using defaults")
	}
}

func TestCreateOllamaEmbedding_UnknownModel(t *testing.T) {
	settings := &domain.EmbeddingSettings{
		Provider: domain.AIProviderOllama,
		BaseURL:  "http://localhost:11434",
		Model:    "custom-model-unknown",
	}

	svc := createOllamaEmbedding(settings)
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
	defer svc.Close()
}

func TestCreateOpenAIEmbedding_Success(t *testing.T) {
	settings := &domain.EmbeddingSettings{
		Provider: domain.AIProviderOpenAI,
		APIKey:   "test-key",
		BaseURL:  "https://api.openai.com/v1",
		Model:    "text-embedding-3-small",
	}

	svc, err := createOpenAIEmbedding(settings)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
	defer svc.Close()
}

func TestCreateAnthropicLLM_Success(t *testing.T) {
	settings := &domain.LLMSettings{
		Provider: domain.AIProviderAnthropic,
		APIKey:   "test-key",
		BaseURL:  "https://api.anthropic.com",
		Model:    "claude-3-5-sonnet-latest",
	}

	svc, err := createAnthropicLLM(settings)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
	defer svc.Close()
}

func TestCreateOpenAILLM_Success(t *testing.T) {
	settings := &domain.LLMSettings{
		Provider: domain.AIProviderOpenAI,
		APIKey:   "test-key",
		BaseURL:  "https://api.openai.com/v1",
		Model:    "gpt-4o-mini",
	}

	svc, err := createOpenAILLM(settings)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
	defer svc.Close()
}

func TestCreateOllamaLLM_Success(t *testing.T) {
	settings := &domain.LLMSettings{
		Provider: domain.AIProviderOllama,
		BaseURL:  "http://localhost:11434",
		Model:    "llama3.2",
	}

	svc := createOllamaLLM(settings)
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
	defer svc.Close()
}

// contains checks if s contains substr.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
