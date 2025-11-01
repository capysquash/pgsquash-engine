# internal/tracking package map

## Domain Summary
- Tracks object lifecycles, detects DDL cycles, analyzes data operations, and manages dependency graphs that feed the squashing/consolidation pipeline.
- Provides risk assessment, streaming integration, and unified tracking APIs consumed by the squasher and reporting layers.

## Files (alphabetical)

### advanced_ddl_cycle_detection.go
- **Purpose**: Detects and classifies DDL cycles (simple, complex, dependency, transient, versioning, constraint) while estimating severity and optimization potential.
- **Key Types**: `AdvancedDDLCycleDetector`, `DDLCycle`, `DDLCycleOperation`, `CycleDetectionConfig`, cycle type/severity enums.
- **Functions / Methods**
  - Lifecycle: `NewAdvancedDDLCycleDetector`, `(*AdvancedDDLCycleDetector) DetectCycles`.
  - Cycle discovery: `detectSimpleCycles`, `detectComplexCycles`, `detectDependencyCycles`, `detectTransientCycles`, `detectVersioningCycles`, `detectConstraintCycles`.
  - Graph helpers: `buildObjectGraph`, `dfsDetectCycle`, `buildDependencyGraph`, `hasCircularDependency`, `buildConstraintDependencies`, `hasConstraintCycle`.
  - Cycle analysis: `buildComplexCycle`, `analyzeCycles`, `calculateCycleSeverity`, `canOptimizeCycle`, `canOptimizeSimpleCycle`, `canOptimizeComplexCycle`, `canOptimizeVersioningCycle`.
  - SQL inspectors: `extractBaseName`, `extractObjectReferences`, `extractForeignKeyReferences`, `extractConstraintName`, `extractConstraintDependencies`.
  - Utility: `maxInt`.

### data_operation_tracker.go
- **Purpose**: Records INSERT/UPDATE/DELETE operations, builds dependency graphs from ASTs, and exposes execution ordering plus statistics.
- **Key Types**: `DataOperationTracker`, `DataOperation`, `OperationDependency`.
- **Functions / Methods**
  - Lifecycle & queries: `NewDataOperationTracker`, `AddOperation`, `GetSortedOperations`, `GetStatistics`.
  - Graph helpers: `buildDependencyGraph`, `topologicalSort`.
  - AST extraction: `extractTableNameFromAST`, `extractRelationName`, `extractDependenciesFromAST`, `extractFromSelectStmt`, `extractFromNode`, `extractFromWhereClause`.

### dependency_graph.go
- **Purpose**: Core dependency graph implementation reused by trackers and consolidation rules.
- **Key Types**: `DependencyGraph`, `DependencyNode`.
- **Functions / Methods**
  - Mutation: `AddNode`, `AddEdge`, `RemoveNode`.
  - Ordering & cycles: `TopologicalSort`, `DetectCycles`, `dfsDetectCycle`, `GetDependencyChain`, `collectDependencies`, `GetLevelOrder`.
  - Introspection: `GetNode`, `GetAllNodes`, `IsEmpty`, `Size`, `HasCycles`.
  - Utilities: `containsObjectID`, `removeObjectID`.

### risk_assessment.go
- **Purpose**: Applies rule-based scoring to lifecycle events and surfaces lifecycle/permission summaries for downstream consumers.
- **Key Types**: `RiskAssessment`, concrete `RiskRule` implementations, `ObjectLifecycle`, `UnifiedTracker`.
- **Functions / Methods**
  - Assessment core: `(*RiskAssessment) AddRule`, `Evaluate`, `GetRiskRules`.
  - Rule behavior: `DataLossRiskRule`, `CircularDependencyRiskRule`, `CrossSchemaRiskRule`, `ProductionUsageRiskRule`, `ConstraintModificationRiskRule`, `PermissionChangeRiskRule` each implement `Evaluate`/`Description`.
  - Lifecycle summaries: `(ObjectLifecycle) CanSquash`, `GetFinalState`, `GetAlterStatements`, `HasConflicts`, `GetHighestRiskLevel`, `GetConsolidatedPermissions`, `GetPermissionStatements`.
  - Tracker reporting: `(UnifiedTracker) GetRedundantObjects`, `analyzeRedundancy`, `GetObjectsByCategory`, `GetStatistics`, `ValidateConsistency`, `GetResourceChanges`, `GetResourceChangesByType`.

### streaming_integration.go
- **Purpose**: Streams large migration directories through the tracker with batching, memory controls, and live progress callbacks.
- **Key Types**: `StreamingTracker`, `TrackingResult`, `StreamingStats`, `MemoryOptimizedTracker`.
- **Functions / Methods**
  - Streaming tracker: `NewStreamingTracker`, `SetProgressCallback`, `ProcessDirectory`, `trackingWorker`, `processTracking`, `progressMonitor`, `GetTracker`, `GetStreamingStats`, `GetCombinedStats`, `Stop`.
  - Memory-constrained variant: `NewMemoryOptimizedTracker`, `(*MemoryOptimizedTracker) ProcessWithMemoryConstraints`.

### tracker_types.go
- **Purpose**: Convenience constructors exposing a simplified tracker API surface.
- **Functions**: `NewTracker`, `NewTrackerWithMetadata`.

### unified_tracker.go
- **Purpose**: Primary lifecycle tracker that integrates metadata, dependency graphs, permission auditing, and cycle detection.
- **Key Types**: `UnifiedTracker`, `ChangeTracker`, `ObjectLifecycle`, `LifecycleEvent`, `ResourceChange`, `ObjectInfo`, `ObjectID`.
- **Functions / Methods**
  - Value helpers: `(ObjectInfo) String`, `FullName`, `(ObjectID) String`.
  - Change tracker: `NewChangeTracker`, `TrackStatement`, `GetChanges`, `GetChangesByObject`, `getChangeType`, `createObjectInfo`, `createSourceRange`, `getObjectKey`, `resolveSchema`.
  - Tracker setup: `NewUnifiedTracker`, `NewUnifiedTrackerWithMetadata`, `NewDependencyGraph`, `NewRiskAssessment`.
  - Lifecycle ingestion: `ProcessMigration`, `createLifecycleEvent`, `createObjectLifecycle`, `createResourceChange`.
  - Mapping helpers: `mapOperationToChangeType`, `mapObjectTypeToResourceType`.
  - Processing utilities: `processPermissionEvent`, `processDependencies`, `extractRequiredObjects`, `parseObjectID`, `inferDependencyType`, `isRequiredDependency`, `extractDescription`, `extractTags`, `extractDatabaseMetadata`, `makeKey`.
  - Queries & metrics: `GetObjects`, `GetDependencyGraph`, `GetActualDependencyGraph`, `EnableStreamingMode`, `DisableStreamingMode`, `IsStreamingMode`, `GetProcessingStats`, `ClearProcessedMigrations`.
  - Cycle reporting: `DetectDDLCycles`, `GetDetectedCycles`, `HasDDLCycles`, `GetCriticalCycles`, `IsObjectInCycle`.

## Subdirectories
- `consolidation/`: Consolidation rule implementations and registry (see `internal/tracking/consolidation/DOC.md`).
- `recovery/`: Fine-grained error recovery strategies (see `internal/tracking/recovery/DOC.md`).
- `reporting/`: Progress reporting adapters (see `internal/tracking/reporting/DOC.md`).
