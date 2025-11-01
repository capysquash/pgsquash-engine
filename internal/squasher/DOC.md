# internal/squasher package map

## Domain Summary
- Core squashing engine: parses migrations, resolves dependencies, plans transactions, applies consolidation rules, runs post-processing, and emits provenance/metrics.
- Coordinates AI insights, plugin integrations, streaming pipelines, and safety-level enforcement while generating reduction plans and provenance maps.

## Files (alphabetical)

### circular_fk_handler.go
- **Purpose**: Detects circular foreign-key dependency chains and rewrites queuing/deferral to keep squashed output executable.
- **Key Types**
  - `CircularFKHandler`: Builds FK dependency graph, extracts constraints.
  - `ForeignKeyConstraint`: Captures FK metadata (columns, actions, original SQL).
- **Functions / Methods**
  - Construction & graph prep: `NewCircularFKHandler`, `(*CircularFKHandler) buildDependencyGraph`, `(*CircularFKHandler) extractForeignKeys`, `(*CircularFKHandler) buildForeignKeyFromConstraint`, `(*CircularFKHandler) getTableName`.
  - Cycle detection: `(*CircularFKHandler) DetectCircularDependencies`, `(*CircularFKHandler) detectCyclesDFS`, `(*CircularFKHandler) extractCycle`, `(*CircularFKHandler) fkInCycle`.
  - Remediation: `(*CircularFKHandler) RemoveCircularFKsFromTables`, `(*CircularFKHandler) removeConstraintsFromCreateTable`, `(*CircularFKHandler) generateAlterTableAddConstraint`.

### consolidation_plan.go
- **Purpose**: Generates human-readable consolidation plans with reductions, conflicts, and recommended actions.
- **Key Types**
  - `ConsolidationPlan`, `PlannedConsolidation`, `ConsolidationConflict`, `ConsolidationStats`.
- **Functions / Methods**
  - Planning & formatting: `(*Engine) GenerateConsolidationPlan`, `( *ConsolidationPlan) FormatPlan`, helper utilities `getRuleName`, `getRuleReason` that derive human-readable rule metadata.

### deparser.go
- **Purpose**: Thin wrapper over `pg_query` deparser with structured error handling.
- **Functions**
  - `Deparse`, `deparseNode`.

### engine.go
- **Purpose**: Master orchestration of squashing workflow (parsing, consolidation, streaming, post-processing, provenance, metrics).
- **Key Types**
  - `Engine`, `EngineConfig`, `SquashResult`, `DetailedMetrics`, `SquashStats`, transaction/streaming structs.
- **Functions / Methods**
  - Construction & configuration: `NewEngine`, `newEngineInternal`, `NewSquasherRuleEngine`.
  - Accessors & lifecycle: `(*Engine) GetTracker`, `GetSafetyLevel`, `GetConfig`, `GetAIAnalyzer`, `GetAuthCompatibilitySQL`, `Close`, `GetStats`, `GetMemoryStats`, `SetProgressCallback`, `updatePhase`.
  - User entry points: `(*Engine) Squash`, `SquashWithSeparateFiles`, `SquashStreaming`, `SquashFromDirectory`, helpers `OptimizedSquashForLargeDatasets`, `OptimizedSquashFromDirectory`.
  - Core phases: `parseAndTrackMigrations`, `analyzeDependenciesAndRisks`, `applyConsolidationRules`, `generateOptimizedSQL`, `generateDataOperationsSQL`, `generateCycleResolutionSuggestions`, `validateAgainstDatabase`.
  - Object helpers: `getObjectsByCategory`, `getObjectsByCategoryAsMap`, `buildEnumReplacementsMap`.
  - Streaming helpers: `streamParseAndTrack`, `streamProcessMigrations`.

### extension_detector.go
- **Purpose**: Identifies required extensions, detects versions/schemas, and surfaces install guidance.
- **Key Types**
  - `ExtensionDetector`, `ExtensionInfo`, `ExtensionRef`.
- **Functions / Methods**
  - Extension metadata: `NewExtensionDetector`, `(*ExtensionRef) Key`, `(*ExtensionRef) CanMergeWith`, `(*ExtensionDetector) initializeExtensions`.
  - Analysis: `(*ExtensionDetector) AnalyzeMigrations`, `detectExtensionsInContent`, `DetectExtensionRefs`, `CanMergeExtensions`, `getExtensionOrder`, `detectAuthService`.
  - Install scripts: `selectBestDockerImage`, `generateInstallationScript`, `generateValidationScript`, `GenerateDockerfile`, `GenerateInitSQL`, `generateAuthCompatibilitySQL`.

### modern_patterns.go
- **Purpose**: Applies modern pattern consolidation (JWT v2 policies, storage policies, vector indexes, session helpers).
- **Key Types**
  - `ModernPatternRule`, `ModernPatternOptimizer`.
- **Functions / Methods**
  - Optimizer lifecycle: `NewModernPatternOptimizer`, `(*ModernPatternOptimizer) ApplyModernOptimizations`.
  - Pattern gating: `meetsSafetyLevel`, `matchesModernPattern`.
  - Consolidators: `consolidateJWTV2OrgPolicies`, `consolidateStoragePolicies`, `consolidateDynamicPolicies`, `consolidateAuthFunctions`, `consolidateAuth0Policies`, `consolidateNextAuthPolicies`, `consolidateVectorIndexes`, `consolidateGeneratedColumns`, `consolidateEventSourcing`.
  - Builders & helpers: `extractPolicyTable`, `extractBucketName`, `extractFunctionSignature`, `createConsolidatedOrgPolicy`, `createConsolidatedStoragePolicy`, `createConsolidatedAuthFunctions`, `createConsolidatedAuth0Policy`, `createConsolidatedNextAuthPolicy`, `createConsolidatedVectorIndex`, `createConsolidatedGeneratedColumns`, `createConsolidatedEventTable`, `createConsolidatedEventFunction`, `extractIndexTable`, `extractAlterTable`.

### provenance.go
- **Purpose**: Produces `.squashmap.json` mapping original statements to squashed output, including stats and hashes.
- **Key Types**
  - `SquashMap`, `StatementMapping`, `SquashStatistics`, `ProvenanceTracker`.
- **Functions / Methods**
  - Tracker lifecycle: `NewProvenanceTracker`, `(*ProvenanceTracker) AddInputFile`, `AddOutputFile`, `SetCurrentSource`, `SetCurrentOutput`, `RecordMapping`, `AddWarning`, `SetStatistics`, `ComputeContentHash`, `WriteSquashMap`, `GetSquashMap`.
  - Persistence & lookup: `LoadSquashMap`, `(*SquashMap) FormatSquashMap`, `(*SquashMap) VerifyContentHash`, `(*SquashMap) FindMappingForSource`, `(*SquashMap) FindMappingForOutput`.

### safety_level_validation.go
- **Purpose**: Validates user-selected safety levels and converts string inputs.
- **Functions**
  - `ValidSafetyLevels`, `IsValidSafetyLevel`, `ValidateSafetyLevel`, `ParseSafetyLevel`.

### unified_dependency_resolver.go
- **Purpose**: Unified dependency resolver combining category/type ordering with SQL dependency extraction for consolidation ordering. Delegates to internal/tracking/dependency_graph.go (single source of truth) for all topological sorting.
- **Key Types**
  - `UnifiedDependencyResolver`.
- **Functions / Methods**
  - Lifecycle: `NewUnifiedDependencyResolver`.
  - Ordering APIs: `ResolveLifecycleDependencies`, `SortConsolidationResults`.
  - Grouping helpers: `groupLifecyclesByCategory`, `determineLifecycleCategory`, `resolveLifecycleWithinCategory`, `createSubGraph`, `breakCyclesAndSort`, `removeCyclicEdges`, `findLeastImportantLifecycleEdge`, `calculateLifecycleEdgeWeight`, `isCriticalLifecycleDependency`, `fallbackSortLifecycles`, `validateLifecycleOrdering`.
  - Graph utilities: `cloneGraph`, `removeGraphEdge`, `objectInList`, `getCategoryByOrder`, `getTypeOrder`, `getTypeOrderForLifecycle`.
  - SQL dependency analysis: `analyzeSQLDependencies`, `topologicalSortSQL` (delegates to tracking.DependencyGraph), `extractTableDependencies`, `extractSchemaDependencies`, `extractExtensionDependencies`, `extractColumnDependencies`, `extractFunctionDependencies`, `extractInsertDependencies`, `extractUpdateDependencies`, `extractTableProvisions`, `extractFunctionProvisions`, `extractSchemaProvisions`, `extractExtensionProvisions`, `extractTypeDependencies`, `extractTypeProvisions`, `dependencyMatches`, `removeDuplicates`, `EnhanceExtensionSQL`.


## Subdirectories
- _None._
