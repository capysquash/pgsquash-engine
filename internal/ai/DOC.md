# internal/ai package map

## Domain Summary
- Centralizes every AI-assisted workflow that supports migration analysis, optimization, validation, and automatic repair across both the CLI (`cmd/pgsquash`) and hosted orchestration layer.
- Coordinates prompt construction, provider selection/failover, response parsing, and surfacing of typed insights (complexity, schema diffs, auth patterns).
- Owns environment-driven provider bootstrap (Azure OpenAI, Anthropic Claude, OpenAI) and exposes feature gating so downstream packages can react to runtime availability.

## Cross-Cutting Concepts
- **Structured Responses**: Helpers insist on JSON envelopes (`ParseStructuredResponse`, schemas in `structured_responses.go`) with markdown-stripping fallbacks for backward compatibility.
- **Provider Abstraction**: The `ProviderManager` shields callers from API differences and enforces retries, health caching, tool support, and prioritized failover.
- **Error Model**: Every operation wraps failures in `internal/errors`, tagging severity, error codes, suggestions, and supplemental metadata for UI surfaces.
- **Environment Contracts**: Reads `AZURE_OPENAI_*`, `ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, etc., to auto-register providers; absence results in actionable validation errors.

## Files (alphabetical)

### analyzer.go
- **Purpose**: Thin façade over `ProviderManager` that binds provider-agnostic primitives to pgsquash-specific use cases and return types.
- **Key Types**
  - `Analyzer`: Holds a lazily configured `ProviderManager`.
- **Functions / Methods**
  - `NewAnalyzer`: Instantiates a `ProviderManager` with default priority (Azure → Claude → OpenAI), returning structured validation errors if none initialize.
  - `(*Analyzer) AreFunctionsSemanticallyEquivalent(ctx, func1, func2)`: Concatenates code with a delimiter, requests `AnalysisFunctionEquivalence`, prefers structured JSON (`FunctionEquivalenceResponse`), falls back to canonicalized booleans with default confidence.
  - `(*Analyzer) IsDeadCode(ctx, schema, functionName)`: Feeds schema + function into `AnalysisDeadCode`, returning typed confidence or boolean heuristics.
  - `(*Analyzer) AnalyzeFunctionComplexity(ctx, functionSQL)`: Requests `FunctionComplexityResponse`, downgrades to plain-text reasoning while preserving safe defaults.
  - `(*Analyzer) DetectAuthPatterns(ctx, sqlContent)`: Issues `AnalysisAuthPatterns`, inflating plain-text lines into `AuthPattern` entries when JSON parsing fails and providing human summary strings.
  - `(*Analyzer) SuggestOptimizations(ctx, migrationSQL)`: Uses `AnalysisOptimizations`; constructs default `Optimization` entries with medium priority when only text is returned.
  - `(*Analyzer) AnalyzeCodeCoverage(ctx, functionSQL, usageContext)`: Dispatches `AnalysisCodeCoverage`, leveraging provider-calculated confidence but otherwise returning raw text.
  - `(*Analyzer) ValidateSchemaConsistency(ctx, originalSchema, squashedSchema)`: Builds multiplexed prompt, seeks `SchemaConsistencyResponse`, and fabricates `SchemaDifference` slices when responses are unstructured.
  - `(*Analyzer) AnalyzeSQLComplexity(ctx, sqlStatement)`: Targets `AnalysisSQLComplexity` with 2K token window; falls back to `SQLComplexityResponse` stub populated with reasoning text.
  - `(*Analyzer) AnalyzeWithTools(ctx, req, tools)`: Relays tool definitions to providers that implement tool-use extension APIs.
  - `(*Analyzer) SubmitBatch` / `GetBatchStatus`: Wrap batch workflows, surfacing provider metadata or failure suggestion strings.
  - `(*Analyzer) HealthCheck(ctx)`: Ensures manager presence and aggregates provider-specific health errors keyed by `ProviderType`.
  - `(*Analyzer) GetCapabilities`: Exposes per-provider `ProviderCapabilities` map.
  - `(*Analyzer) GetAvailableProviders`: Returns slice of active provider identifiers.
  - `(*Analyzer) GetProvider(providerType)`: Delegates to manager with initialization guard and suggestion hints.
  - `(*Analyzer) GetManager()`: Accessor retained for advanced integrators that need direct manager control.

### manager.go
- **Purpose**: Owns provider registration, prioritized routing, retry policies, and capability surfacing for all AI integrations.
- **Key Types**
  - Type aliases (`Provider`, `ProviderType`, `AnalysisRequest`, etc.) to isolate provider contracts.
  - `ProviderManager`: Thread-safe registry and router with cached provider health and configuration state.
  - `HealthCache`: Captures health status, TTL (five minutes), last check timestamp, and human-readable error string.
  - `ManagerConfig`: JSON-compatible config controlling default/fallback provider, preferred provider per `AnalysisType`, retry stanza, and load-balancing flag.
  - `RetryConfig`: Tunable retries, backoff, delay ceilings used by `analyzeWithProvider`.
- **Functions / Methods**
  - `NewProviderManager(config *ManagerConfig)`: Applies defaults (Azure AD-capable config, fallback to Claude, retriable policy), then invokes `initializeProviders`.
  - `(*ProviderManager) initializeProviders()`: Bootstraps providers conditionally per env var, injects defaults (models, timeouts), reconciles default/fallback order if requested provider missing, and errors when none are available.
  - `(*ProviderManager) GetProvider(providerType)`: Guarded lookup returning suggestion-laden errors when provider is absent.
  - `(*ProviderManager) GetDefaultProvider()`: Convenience wrapper calling `GetProvider` with current default type.
  - `(*ProviderManager) ListProviders()`: Collects active provider keys with read lock.
  - `(*ProviderManager) Analyze(ctx, req)`: Computes priority list, skips cached-unhealthy providers with reason tracking, attempts each provider with retry/backoff (`analyzeWithProvider`), annotates fallback metadata on success, aggregates errors when all fail.
  - `(*ProviderManager) analyzeWithProvider(ctx, req, providerType)`: Applies exponential backoff using `RetryConfig`, respects context cancellation, records retry counts in response metadata, and wraps terminal error with guidance.
  - `(*ProviderManager) selectProvider(req)`: Honors `PreferredProviders` mapping before default fallback.
  - `(*ProviderManager) AnalyzeWithTools(ctx, req, tools)`: Detects `ToolUseProvider` support and gracefully falls back to standard `Analyze`.
  - `(*ProviderManager) SubmitBatch` / `GetBatchStatus`: Iterates providers supporting batch, returns actionable errors when unsupported or missing.
  - `(*ProviderManager) HealthCheck(ctx)`: Delegates to provider-level `HealthCheck` without touching caches.
  - `(*ProviderManager) GetCapabilities()`: Produces map of `ProviderCapabilities` describing features and limits.

### manager_failover.go
- **Purpose**: Encapsulates prioritized routing and health cache helpers consumed by the main manager.
- **Functions / Methods**
  - `(*ProviderManager) getPrioritizedProviders(req)`: Builds ordered slice: preferred provider (if configured and available), default provider, then remaining providers deduplicated.
  - `(*ProviderManager) isProviderHealthy(ctx, providerType)`: Consults `healthCache` (TTL five minutes) and refreshes via provider `HealthCheck` on cache miss.
  - `(*ProviderManager) updateHealthCache(providerType, healthy)`: Stores current status and clears prior error string.
  - `(*ProviderManager) getProviderHealthReason(ctx, providerType)`: Returns cached failure reason or triggers fresh health check, updating cache before reporting.

### migration_fixer.go
- **Purpose**: Automates migration repair loops by analyzing validation errors, prompting AI for fixes, and applying patches safely.
- **Key Types**
  - `ValidationFunc`: Adapter signature for custom validation callbacks.
  - `MigrationFixer`: Coordinates fix attempts, verbosity, validation hooks, and provider calls.
  - `FixAttempt` / `FixResult`: Capture per-attempt outcomes and aggregated run status.
  - `ErrorAnalysis`, `MigrationFile`, `AIFix`: Internal helper structures for parsing errors, migration data, and AI responses.
- **Functions / Methods**
  - `NewMigrationFixer(provider, maxAttempts, verbose)`: Normalizes attempts (min 1), stores provider, verbosity flag.
  - `(*MigrationFixer) WithValidation(validationFunc)`: Assigns custom validation loop and returns self for chaining.
  - `(*MigrationFixer) FixMigrationsUntilValid(ctx, path, validationError)`: Iterative orchestrator that logs attempts (via `color`), tracks modified files, re-runs validation when possible, and exits early on success.
  - `(*MigrationFixer) analyzeAndFix(ctx, err, path, attempt)`: Parses validation message, composes prompt with truncated migration bodies, calls provider (`AnalysisRequest` type `migration_fix`), enforces confidence threshold (0.75), and optionally applies fix.
  - `(*MigrationFixer) parseValidationError(error)`: Regex-driven extraction of duplicate triggers/functions/publications plus migration file hints.
  - `(*MigrationFixer) readMigrationFiles(path)`: Walks directory, loading every `.sql` file into memory.
  - `(*MigrationFixer) buildFixPrompt(analysis, migrations)`: Emits structured instructions and common fix hints, truncating to avoid token blowups.
  - `(*MigrationFixer) parseAIResponse(response)`: Validates expected format segments (`FILE`, `DESCRIPTION`, `FIX_SQL`...`END_FIX`), returning actionable `AIFix` or wrapping errors with context.
  - `(*MigrationFixer) applyFix(fix, migrationPath)`: Writes `.backup` copy, prepends SQL patch to target file, and handles IO errors via error wrappers.
  - `contains(slice, item)`: Utility to deduplicate modified file tracking.

### structured_responses.go
- **Purpose**: Defines typed response contracts and parsing helpers so higher layers can rely on consistent fields regardless of provider output quirks.
- **Key Types**
  - `FunctionEquivalenceResponse`, `DeadCodeResponse`: Boolean + confidence payloads that surface reasoning and optional difference/use-site lists.
  - `FunctionComplexityResponse`: Scores, maintainability tier, performance issues, recommendations, reasoning string.
  - `AuthPatternsResponse` / `AuthPattern`: Captures detected auth/RLS pattern taxonomy, description, optional code location, and summary text.
  - `OptimizationsResponse` / `Optimization`: Encodes type, priority, impact, and actionable suggestion per optimization along with aggregate score.
  - `SchemaConsistencyResponse` / `SchemaDifference`: Flags if schemas match and enumerates categorized diffs with severity for UI triage.
  - `SQLComplexityResponse`: Score, issues, best practices violated, and reasoning trace.
- **Functions**
  - `ParseStructuredResponse[T](response string)`: Pulls JSON (optionally from fenced code blocks), unmarshals into typed struct, and wraps errors via `internal/errors` with raw response for debugging.
  - `extractJSONFromMarkdown(content string)`: Strips leading/trailing code fences when providers respond with markdown.
  - `GetStructuredPromptSuffix(schema string)`: Emits canonical suffix instructing providers to answer with JSON matching supplied schema.

### test_integration.go
- **Purpose**: Provides CLI-friendly demonstrations of AI functionality and environment readiness checks.
- **Functions**
  - `TestAIIntegration()`: Baseline script printing provider availability, capabilities, health status, and keyed environment readiness with emoji-coded output.
  - `testClaudeSpecificFeatures()`: Validates Claude provider access, reports tool/streaming/batch support flags.
  - `testOpenAISpecificFeatures()`: Mirrors provider inspection for OpenAI adapter.
  - `testAnalysisTypes()`: Enumerates supported `AnalysisType` constants for quick manual verification.
  - `RunAIIntegrationTest()`: Panic-protected entrypoint that logs recoveries through `utils` logger.
  - `DemoAICapabilities()`: Demonstrates major analyzer flows (function equivalence, dead code, auth patterns) with sample SQL, returning structured errors when providers unavailable.

## Subdirectories
- `providers/`: Concrete AI provider adapters and shared utilities. See `providers/DOC.md` for detailed mapping.
