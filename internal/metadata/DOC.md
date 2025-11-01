# internal/metadata package map

## Domain Summary
- Introspects live PostgreSQL databases to collect rich metadata (schemas, tables, functions, indexes, policies, extensions, etc.) used during planning, squashing, and validation.
- Provides schema comparison utilities that detect drift, missing dependencies, type mismatches, and potential breaking changes between generated SQL and production environments.
- Caches catalog lookups with TTL and exposes search helpers so other packages can resolve objects without re-querying the database.

## Files (alphabetical)

### manager.go
- **Purpose**: Metadata ingestion and caching layer that queries PostgreSQL catalogs, hydrates strongly typed structures, and offers search helpers.
- **Key Types**
  - `MetadataManager`: Caches per-database metadata with TTL, tracks cache stats, orchestrates fetch routines.
  - Rich metadata models: `DatabaseMetadata`, `SchemaMetadata`, `TableMetadata`, `ColumnMetadata`, `ConstraintMetadata`, `IndexMetadata`, `ViewMetadata`, `MaterializedViewMetadata`, `FunctionMetadata` (+ `Parameter`), `SequenceMetadata`, `TypeMetadata`, `TriggerMetadata`, `PolicyMetadata`, `ExtensionMetadata`, `PostgreSQLVersion`.
- **Primary Functions / Methods**
  - Construction & caching: `NewMetadataManager`, `(*MetadataManager) GetMetadata`, `loadAndCacheMetadata`, `InvalidateMetadata`, `GetCacheStats`.
  - Search utilities on `DatabaseMetadata`: `SearchObject`, `SearchTable`, `SearchFunctions`, `GetSearchPath`, `SetSearchPath`.
  - System helpers: `IsSystemSchema`, `IsSystemTable`, `IsSystemFunction`.
  - Dependency analysis: `(*DatabaseMetadata) AnalyzeViewDependencies`, local `contains`.
  - Loader pipeline:
    - Entry: `loadMetadataFromDB`.
    - Version/search path: `loadVersion`, `loadSearchPath`.
    - Schema-level: `loadSchemas`, `loadTablesForSchema`, `loadViewsForSchema`, `loadMaterializedViewsForSchema`, `loadFunctionsForSchema`, `loadSequencesForSchema`, `loadTypesForSchema`, `loadExtensions`.
    - Table detail loaders: `loadColumnsForTable`, `loadConstraintsForTable`, `loadIndexesForTable`, `loadTriggersForTable`, `loadPoliciesForTable`.
  - Support routines inside loaders (e.g., parsing version strings, extension feature flags).

### schema_comparator.go
- **Purpose**: Compares generated SQL (“squashed” migrations) to production metadata, surfacing incompatibilities, missing dependencies, and drift.
- **Key Types**
  - `SchemaComparator`: Wrapper requiring a `MetadataManager`.
  - Result/diagnostic structs: `ComparisonResult`, `MissingDependency`, `TypeMismatch`, `ConstraintConflict`, `BreakingChange`, `SchemaDrift`.
- **Functions / Methods**
  - Lifecycle: `NewSchemaComparator`, `(*SchemaComparator) CompareSchema`.
  - Extension tracking: `extractRequiredExtensions`, `extractExtensionName`.
  - Dependency validation: `validateStatementDependencies`, helper `isOptionalDependency`.
  - Structural comparison:
    - Tables/columns/constraints: `compareTableSchemas`, `compareColumns`, `compareConstraints`.
    - Functions: `compareFunctionSignatures`.
    - Types: `compareTypes`, `areTypesCompatible`, `normalizeType`, `isBreakingTypeChange`.
  - Drift detection & utilities: `detectSchemaDrift`, `extractAllObjects`, `min`.

## Subdirectories
- _None._
