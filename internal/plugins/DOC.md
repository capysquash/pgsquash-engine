# internal/plugins package map

## Domain Summary
- Defines a pluggable integration framework allowing service-specific behavior (auth providers, ORMs, platforms) to hook into parsing, transformation, validation, and squashing pipelines.
- Provides registry/activation logic, conflict resolution, and consolidation helpers that plugin implementations extend.

## Files (alphabetical)

### consolidation_base.go
- **Purpose**: Shared consolidation helpers for common object families (policies, functions, tables).
- **Key Types**
  - `BaseConsolidator`: Minimal struct that stores rule name and metadata.
  - `PolicyConsolidator`, `FunctionConsolidator`, `TableConsolidator`: Embed base and add specialized predicates/merge logic.
- **Functions / Methods**
  - Constructors: `NewBaseConsolidator`, `NewPolicyConsolidator`, `NewFunctionConsolidator`, `NewTableConsolidator`.
  - Policy utilities: `AllSameTargetTable`, `AllSameObjectName`, `AllSameObjectType`, `ExtractPolicyClauses`, `AllClausesIdentical`, `HaveSamePolicyLogic`.
  - Function utilities: `AllSameFunctionSignature`.
  - Table utilities: `HasCreateOperation`, `AllSameTable`, `ContainsKeyword`, `FilterByOperation`, `ConservativeMerge`.

### plugin.go
- **Purpose**: Declares the `Plugin` interface, pattern structs, consolidation rule contracts, and base implementations.
- **Key Types**
  - `Plugin`: Lifecycle hooks (Detect, Initialize, EnrichStatement, DetectPatterns/Auth, TransformSQL, InjectCompatibilityLayer, FixFunctionVolatility, ValidateSchema, GetRequiredExtensions, GetConsolidationRules, ShouldPreserve, GetConflictingPlugins).
  - `Pattern`, `PatternType`, `PatternSeverity`, `Location`: Capture detected plugin-specific patterns.
  - `ConsolidationRule`: Describes plugin-provided merge logic.
  - `BasePlugin`: Default no-op implementation used by plugin packages.
- **Functions / Methods**
  - `NewBasePlugin`: Creates base plugin with name/priority.
  - BasePlugin method overrides: `Name`, `Priority`, `GetConflictingPlugins`, `EnrichStatement`, `DetectPatterns`, `TransformSQL`, `InjectCompatibilityLayer`, `FixFunctionVolatility`, `ValidateSchema`, `GetRequiredExtensions`, `GetConsolidationRules`, `ShouldPreserve`, `DetectAuthPattern`.

### registry.go
- **Purpose**: Manages plugin registration, detection, initialization, and hook delegation.
- **Key Types**
  - `Registry`: Holds all registered and active plugins plus configuration.
- **Functions / Methods**
  - Lifecycle: `NewRegistry`, `Register`, `DiscoverAndInitialize`, `resolveConflicts`, `isPluginEnabled`, `getPluginConfig`, `getPluginNames`, `Reset`.
  - Lookup APIs: `ActivePlugins`, `GetPlugin`, `IsActive`.
  - Hook fan-out: `EnrichStatement`, `TransformSQL`, `InjectCompatibilityLayer`, `GetRequiredExtensions`, `GetConsolidationRules`, `ShouldPreserve`, `ValidateSchema`.
  - Utilities: `join`, `GlobalRegistry`, package-level `Register`.

## Subdirectories
- `auth/`: Compatibility SQL generators for auth providers.
- `clerk/`: Clerk-specific plugin implementation.
- `drizzle/`: Drizzle ORM plugin implementation.
- `prisma/`: Prisma ORM plugin implementation.
- `supabase/`: Supabase platform plugin implementation.
- `volatility/`: Shared logic for adjusting function volatility.
