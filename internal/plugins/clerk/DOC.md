# internal/plugins/clerk package map

## Domain Summary
- Implements the Clerk auth provider plugin: detects Clerk patterns, preserves critical functions/policies, injects compatibility SQL, and manages consolidation of Clerk-specific statements.

## Files (alphabetical)

### clerk.go
- **Purpose**: Core Clerk plugin implementation.
- **Key Types**
  - `ClerkPlugin`: Extends `plugins.BasePlugin` with Clerk-specific behavior and config fields (organization support, JWT version, etc.).
- **Functions / Methods**
  - Lifecycle: `NewClerkPlugin`, `Detect`, `Initialize`.
  - Pattern detection: `hasSupabasePatterns`, `hasJWTV2OrgPattern`, `isClerkAuthFunction`, `DetectPatterns`, `DetectAuthPattern`.
  - Statement enrichment: `EnrichStatement`.
  - Conflict/requirement info: `GetConflictingPlugins`, `GetRequiredExtensions`, `ShouldPreserve`.
  - Validation: `ValidateSchema`.

### consolidation.go
- **Purpose**: Consolidation rules for Clerk-specific statements.
- **Functions**
  - `(*ClerkPlugin) GetConsolidationRules`: Returns consolidation rules bound to helpers.
  - Merge predicates/actions: `canMergeJWTV2Policies`, `mergeJWTV2Policies`, `canMergeAuthFunctions`, `mergeAuthFunctions`.

### transformations.go
- **Purpose**: SQL transformations and compatibility injection.
- **Functions**
  - `(*ClerkPlugin) InjectCompatibilityLayer`: Emits Clerk compatibility SQL (JWT helper functions, schemas).
  - `(*ClerkPlugin) TransformSQL`: Applies pre-parse fixes.
  - `(*ClerkPlugin) FixFunctionVolatility`: Adds STABLE/IMMUTABLE markers to Clerk functions.

## Subdirectories
- _None._
