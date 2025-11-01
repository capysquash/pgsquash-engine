# internal/plugins/supabase package map

## Domain Summary
- Handles Supabase-specific migrations: detects JWT/RLS/storage patterns, injects Supabase compatibility SQL, consolidates redundant policies/functions, and validates that critical helper objects remain intact.

## Files (alphabetical)

### consolidation.go
- **Purpose**: Consolidation rules for Supabase row-level security policies, storage policies, and auth helper functions.
- **Functions / Methods**
  - `(*SupabasePlugin) GetConsolidationRules()`: Registers rule set for policies/functions.
  - `(*SupabasePlugin) canMergeRLSPolicies`, `mergeRLSPolicies`: Collapse multiple `ALTER POLICY` statements into canonical definitions.
  - `(*SupabasePlugin) canMergeStoragePolicies`, `mergeStoragePolicies`: Merge storage bucket policies while preserving priority/order.
  - `(*SupabasePlugin) canMergeAuthFunctions`, `mergeAuthFunctions`: Combine duplicate Supabase auth helper functions.

### supabase.go
- **Purpose**: Core plugin implementation responsible for detection, enrichment, auth pattern reporting, and schema validation.
- **Key Types**
  - `SupabasePlugin`: Extends `plugins.BasePlugin`, tracks detected policies, storage helpers, and auth functions.
- **Functions / Methods**
  - Lifecycle: `NewSupabasePlugin`, `Detect`, `Initialize`.
  - Detection helpers:
    - `DetectAuthPattern`: Returns auth pattern identifiers when Supabase functions are spotted.
    - `EnrichStatement`: Annotates statements with Supabase metadata for consolidation.
    - `DetectPatterns`: Emits plugin pattern findings (JWT helpers, storage policies, multi-tenant hints).
    - `isSupabaseAuthFunction`: Identifies Supabase-provided SQL functions.
  - Platform wiring: `GetConflictingPlugins`, `GetRequiredExtensions`, `ShouldPreserve`, `ValidateSchema`.

### transformations.go
- **Purpose**: Emits compatibility SQL blocks and performs rewrites to maintain Supabase semantics.
- **Functions / Methods**
  - `InjectCompatibilityLayer`: Outputs SQL for Supabase auth schemas, JWT helpers, and storage metadata tables.
  - `TransformSQL`: Applies pre-parse fixes specific to Supabase output (e.g., quoting JWT helpers).
  - `FixFunctionVolatility`: Adds appropriate volatility to Supabase helper functions.

## Subdirectories
- _None._
