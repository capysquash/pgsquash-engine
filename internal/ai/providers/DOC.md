# internal/ai/providers package map

## Domain Summary
- Houses provider implementations that let pgsquash talk to Azure OpenAI, Anthropic Claude, and public OpenAI endpoints with a uniform contract.
- Exposes the shared abstractions (`Provider`, `AnalysisRequest`, `ProviderCapabilities`, tool/batch interfaces) consumed by `ProviderManager`.
- Supplies prompt scaffolding, response confidence heuristics, and validation routines so every provider behaves consistently.

## Cross-Cutting Notes
- **Authentication Inputs**: Pulls from `AZURE_OPENAI_*`, `ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, and supports Azure AD token credentials via `azidentity`.
- **Prompt Hygiene**: Every adapter calls `GetSystemPromptForType` and `BuildUserPrompt` to enforce JSON-only responses and shared guidance per analysis type.
- **Capability Reporting**: Providers populate `ProviderCapabilities` (supported analysis types, streaming/tool/batch booleans, token limits) for downstream discovery API responses.

## Files (alphabetical)

### azure_openai.go
- **Purpose**: Adapter for Azure-hosted OpenAI deployments, handling Azure AD or API key auth and API-version differences.
- **Key Types**
  - `AzureOpenAIProvider`: Implements `Provider` interface using the official `openai-go` client with Azure extensions.
- **Functions / Methods**
  - `NewAzureOpenAIProvider(config)`: Validates `Endpoint`/`AzureDeployment`, defaults API version to `preview`, chooses Azure AD vs API key auth, normalizes timeout, and creates `openai-go` client with `/openai/v1/` routing when using the v1 API.
  - `(*AzureOpenAIProvider) Name` / `Type` / `SupportedTypes`: Return provider metadata and supported analysis type slice (equivalence, dead code, complexity, auth, optimizations, coverage, schema, SQL).
  - `(*AzureOpenAIProvider) Configure(config)`: Applies live updates to stored config fields (API key, model, endpoint, deployment, API version, timeout).
  - `(*AzureOpenAIProvider) HealthCheck(ctx)`: Issues a small `Analyze` call (test content, 10 tokens) and surfaces errors directly.
  - `(*AzureOpenAIProvider) Analyze(ctx, req)`: Builds chat messages, enforces temperature/token defaults, calls Azure Chat Completions API, and returns `AnalysisResponse` containing confidence, deployment metadata, token usage data, and elapsed duration.
  - `(*AzureOpenAIProvider) SubmitBatch` / `GetBatchStatus`: Return `errors.ErrorCodeValidationFailed` advising users to leverage Claude for batch support.
  - `(*AzureOpenAIProvider) SupportsStreaming`, `SupportsTools`, `SupportsBatch`: Mirror capability struct (streaming/tools true, batch false).

### claude.go
- **Purpose**: Adapter for Anthropic Claude via the official SDK, including tool registration and streaming support.
- **Key Types**
  - `ClaudeProvider`: Implements `Provider`, `ToolUseProvider`, and `StreamingProvider`.
- **Functions / Methods**
  - `NewClaudeProvider(config)`: Requires API key, defaults to Claude Sonnet 4.5, seeds capability struct (streaming/tools/batch true, 200K token window, 100K batch size).
  - `Name` / `Type` / `SupportedTypes`: Standard metadata surfaces for manager.
  - `Configure(config)`: Rebuilds client when API key changes, updates model selection at runtime.
  - `HealthCheck(ctx)`: Sends minimal message to confirm connectivity.
  - `Analyze(ctx, req)`: Builds prompts via `common.go`, feeds them to Claude Messages API, merges reasoning across content blocks (text, thinking) and records token usage in metadata.
  - `SubmitBatch` / `GetBatchStatus`: Currently return forward-looking validation errors noting future implementation.
  - `SupportsStreaming` / `SupportsTools` / `SupportsBatch`: Expose capability flags from configuration.
  - `AnalyzeWithTools(ctx, req, tools)`: Reuses standard messaging while counting tool handles in metadata (SDK lacks explicit tool invocation yet).
  - `RegisterTool`, `ListTools`: Manage in-memory slice of tool definitions.
  - `AnalyzeStream(ctx, req)`: Goroutine wrapper that emits single-response channel today (placeholder for Anthropic streaming).
  - Helpers: `extractTextFromResponse` flattens Anthropic polymorphic content; `calculateConfidence` provides heuristics tuned for Claude output.

### common.go
- **Purpose**: Provider-neutral helpers for prompt scaffolding, confidence heuristics, and request validation.
- **Functions**
  - `GetSystemPromptForType(analysisType)`: Supplies tailored expert instructions per analysis (equivalence, dead code, complexity, auth, optimizations, coverage, schema, SQL).
  - `BuildUserPrompt(req)`: Generates detailed user prompts including task instructions, example acceptance criteria, JSON schema expectations, and attaches request context when provided.
  - `CalculateConfidence(analysisType, result)`: Heuristic estimator that checks response structure (booleans, keyword presence, length) to produce repeatable confidence signals.
  - `ValidateAnalysisRequest(req)`: Validates basic invariants (non-nil request, content presence, non-negative tokens, temperature bounds 0-2) pre-dispatch.

### openai.go
- **Purpose**: Adapter for OpenAI's public API using the go-openai client.
- **Key Types**
  - `OpenAIProvider`: Concrete `Provider` that leverages `go-openai` chat completion endpoints.
- **Functions / Methods**
  - `NewOpenAIProvider(config)`: Requires API key, sets default model `gpt-4`, optional custom base URL, builds client with optional HTTP timeout override, and seeds capability struct (tools true, streaming/batch false).
  - `Name` / `Type` / `SupportedTypes`: Mirror provider identity and supported analysis list.
  - `Configure(config)`: Applies runtime changes to API key/model/endpoint, recreating client when necessary.
  - `HealthCheck(ctx)`: Uses `Analyze` on a tiny prompt to verify connectivity.
  - `Analyze(ctx, req)`: Builds `ChatCompletionRequest`, relays temperature/tokens, and returns `AnalysisResponse` including finish reason and token accounting.
  - `SubmitBatch`, `GetBatchStatus`: Return unsupported-operation errors with suggestion to use Claude.
  - `SupportsStreaming`, `SupportsTools`, `SupportsBatch`: Reflect capabilities (only tools true).

### types.go
- **Purpose**: Defines shared contracts and capability descriptors used across providers.
- **Key Types**
  - `ProviderType`: Identifiers (`openai`, `claude`, `local`, `bedrock`, `azure-openai`) referenced throughout AI layer.
  - `AnalysisRequest`: Core payload describing analysis type, content, optional context/metadata, temperature, token budget.
  - `AnalysisResponse`: Result string plus confidence, metadata map, token counts, duration, provider ID.
  - `AnalysisType`: Constants covering function equivalence, dead code, complexity, auth patterns, optimizations, coverage, schema consistency, SQL complexity, and batch mode marker.
  - `ProviderConfig`: Provider-level configuration (API key, model, endpoint, timeout, retry defaults, Azure deployment/API version, Azure AD flag).
  - `BatchRequest` / `BatchResponse`: Structures supporting batch operations (ID, request list, metadata, callback URL, status, progress, completion timestamp).
  - Interfaces: `Provider` (core contract), `ToolDefinition`, `ToolUseProvider` (tool execution), `StreamingProvider` (streamed responses).
  - `ProviderCapabilities`: Snapshot of supported analysis types, streaming/tool/batch support, token and batch size limits.

## Notes
- All adapters rely on helpers from `common.go` for deterministic prompt shaping and heuristic confidence scores.
- Capability maps surfaced through `ProviderManager.GetCapabilities` mirror `ProviderCapabilities`, ensuring downstream callers (CLI/TUI/rest) stay in sync with provider abilities.
