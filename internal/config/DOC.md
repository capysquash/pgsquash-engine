# internal/config package map

## Domain Summary
- Defines the primary engine configuration schema (`pgsquash.config.json`) and helpers to load, merge, validate, and persist user settings.
- Adds support for GitHub-hosted CAPYSQUASH automation via `.capysquash.yml`, with logic for per-branch/project overrides and GitHub workflow controls.
- Provides ergonomic error reporting (line/column context, typed validation failures) so CLI surfaces actionable guidance when configuration is malformed.
- Centralizes feature flags for AI integrations, Docker validation, transformation workflows, and plugin ecosystems consumed across CLI and engine packages.

## Cross-Cutting Concepts
- **Defaults vs Overrides**: `DefaultConfig` seeds sane values; `mergeConfigs` overlays user JSON while preserving boolean intent and combining slices/maps carefully.
- **Environment Awareness**: DSNs, output directories, vector feature toggles, and validation sockets can be derived from env vars to support CI/CD.
- **Validation Pipeline**: `Validate` checks enum membership, positive numeric ranges, mutual exclusivity of options (e.g., vector vs planet-scale), and prerequisite credentials for integrations.
- **Rich Error Types**: Custom JSON error types capture file path, line/column, context string, and suggested fixes; aggregated validation errors bubble up as multi-line reports.
- **Plugin/AI Feature Flags**: Configuration exposes fine-grained toggles for plugin adapters (`supabase`, `drizzle`, etc.) and AI provider hints (temperature, token caps, confidence thresholds) consumed by other packages.

## Files (alphabetical)

### capysquash.go
- **Purpose**: Models `.capysquash.yml` and provides routines to discover, parse, merge, and apply GitHub-specific automation settings.
- **Key Types**
  - `CapySquashConfig`: High-level GitHub automation settings (safety level, thresholds, include/exclude patterns, notifications, auto-apply).
  - Nested configs: `PRCommentConfig`, `ChecksConfig`, `RequiredIndex`, `NotificationsConfig`, `AutoApplyConfig`, `ProjectConfig`, `BranchConfig`.
- **Functions / Methods**
  - `DefaultCapySquashConfig`: Returns opinionated defaults used when no YAML exists.
  - `LoadCapySquashConfig`: Searches common paths, loads YAML, applies defaults.
  - `LoadCapySquashConfigFromRepo`: Repository-root aware loader.
  - `applyCapySquashDefaults`: Fills unset fields with default values.
  - `(*CapySquashConfig) MergeWithEngineConfig`: Overlays GitHub-specific options onto engine `Config`.
  - `(*CapySquashConfig) GetProjectConfig`: Finds monorepo project block matching a set of files.
  - `(*CapySquashConfig) GetBranchConfig`: Retrieves branch-specific overrides.
  - `(*CapySquashConfig) ShouldAnalyze`: Checks include/exclude globs against changed files.
  - `(*CapySquashConfig) ShouldAutoApply`: Determines whether auto-apply is allowed for a branch.
  - `matchesGlob`: Lightweight glob matcher supporting `*` and `**`.
  - `ParseCapySquashYAML`: Helper for parsing YAML payloads (e.g., tests).

### config.go
- **Purpose**: Core configuration schema plus I/O utilities for JSON-based engine configuration.
- **Key Types**
  - `Config`: Top-level settings covering safety level, IO, rules, performance, modern PostgreSQL features, third-party integrations, plugin toggles, validation, and AI options.
  - Nested structs (selected): `OutputConfig`, `RulesConfig`, `TableRulesConfig`, `IndexRulesConfig`, `FunctionRulesConfig`, `PerformanceConfig`, `ModernFeaturesConfig`, `ConflictResolutionConfig`, `PostgreSQLFeaturesConfig`, `ThirdPartyConfig` (with `Auth0Config`, `NextAuthConfig`, `SupabaseConfig`, `ClerkConfig`, `VectorConfig`, `PlanetScaleConfig`), `PluginSettings`, `ValidationConfig`, `AIConfig`.
  - Error wrappers: `ValidationErrors`, `ConfigValidationError`, `JSONSyntaxError`, `JSONTypeError`, `GenericJSONError`.
- **Functions**
  - `DefaultConfig`: Produces a fully populated default configuration (env-aware DSN, output directory, feature toggles, validation defaults, AI defaults such as provider hints and temperature).
  - `mergeConfigs`: Applies top-level overlay logic while calling granular helpers for each nested struct.
  - Merge helpers: `mergeOutputConfig`, `mergeRulesConfig`, `mergePerformanceConfig`, `mergeModernFeaturesConfig`, `mergeConflictResolutionConfig`, `mergePostgreSQLFeaturesConfig`, `mergeThirdPartyConfig`, `mergeAuth0Config`, `mergeNextAuthConfig`, `mergeSupabaseConfig`, `mergeClerkConfig`, `mergeVectorConfig`, `mergePlanetScaleConfig`, `mergePluginSettings`, `mergeValidationConfig`, `mergeAIConfig`; each ensures user overrides blend with defaults without clobbering explicit false/nil values or expanding slices incorrectly.
  - `LoadConfig`: Reads JSON from disk, unwraps syntax/type errors into rich structs, merges with defaults, applies CapySquash overrides if present, and runs validation.
  - `(*Config) Validate`: Ensures enums, numeric ranges, inter-field dependencies (e.g., data-loss safeguards when aggressive safety selected), integration credentials, and plugin settings align with supported values.
  - Error helpers: `(*ValidationErrors).Error`, `(*ConfigValidationError).Error`, `(*JSONSyntaxError).Error`, `(*JSONTypeError).Error`, `(*GenericJSONError).Error` for granular feedback.
  - `getJSONErrorContext`: Calculates line/column/context for meaningful JSON error reporting.
  - `(*Config) SaveToFile`: Persists config with helpful comment header (documenting file naming strategies) and pretty JSON formatting.

## Subdirectories
- _None._
