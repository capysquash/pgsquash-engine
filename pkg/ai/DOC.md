# pkg/ai package map

## Domain Summary
- Public-facing wrapper around `internal/ai` that re-exports provider manager types for external consumers (CLI, SDKs) without exposing internal package paths.
- Documents supported providers (Azure OpenAI, Anthropic Claude, OpenAI) and configuration defaults.

## Files (alphabetical)

### api.go
- **Purpose**: Re-export `internal/ai` provider manager, config structs, analysis request/response types, and provider capability interfaces for SDK use.
- **Key Items**
  - Types: `ProviderManager`, `ManagerConfig`, `RetryConfig`, `Provider`, `ProviderType`, `ProviderConfig`, `AnalysisRequest`, `AnalysisResponse`, `AnalysisType`, `BatchRequest`, `BatchResponse`, `ToolDefinition`, `ProviderCapabilities`, `ToolUseProvider`, `StreamingProvider`.
  - Constants: `ProviderAzureOpenAI`, `ProviderClaude`, `ProviderOpenAI`, `ProviderLocal`, `ProviderBedrock`.
  - Functions: `NewProviderManager`, `NewProviderManager` helpers for analysis and health check delegation.

## Subdirectories
- _None._
