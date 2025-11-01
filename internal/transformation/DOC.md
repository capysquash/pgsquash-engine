# internal/transformation package map

## Domain Summary
- Provides backup generation, rollback planning, and SQL transformation utilities that safeguard squashing operations and enable recovery when needed.
- Integrates with plugins for SQL rewrites, produces rollback scripts, and executes pg_dump-based safety nets.

## Files (alphabetical)

### backup_generator.go
- **Purpose**: Generates schema/data backups (pg_dump wrappers) with configurable formats, compression, and table/schema filters.
- **Key Types**: `BackupConfig`, `BackupFormat`, `BackupGenerator`, `BackupResult`.
- **Functions / Methods**
  - Configuration: `DefaultBackupConfig`, `NewBackupGenerator`, `findPgDumpPath`, `(*BackupGenerator) SetWorkingDirectory`.
  - Backup lifecycle: `GeneratePreMigrationBackup`, `GeneratePostMigrationBackup`, internal `generateBackup`, `buildPgDumpArgs`, `executePgDump`, `analyzeBackup`.
  - Rollback script creation: `GenerateRollbackScript`, `generateRollbackForStatement`, `generateAlterTableRollback`, `generateInsertRollback`.
  - Validation & maintenance: `ValidateBackup`, `validateSQLBackup`, `CleanupOldBackups`.

### rollback_manager.go
- **Purpose**: Manages rollback plans/scripts, metadata, and execution tracking.
- **Key Types**: `RollbackManager`, `RollbackPlan`, `RollbackScript`, `RollbackMetadata`, `RollbackExecution`.
- **Functions / Methods**
  - Planning: `NewRollbackManager`, `CreateRollbackPlan`, `analyzeRollbackPlan`, `estimateRollbackDuration`.
  - Environment introspection: `generateSchemaChecksum`, `getDatabaseVersion`.
  - Execution: `ExecuteRollbackPlan`, `executeScript`.
  - Persistence & retrieval: `ListRollbackPlans`, `GetRollbackPlan`, `GetRollbackExecution`, `savePlanToFile`, `loadPlanFromFile`, `LoadAllPlans`, `DeleteRollbackPlan`.
  - Validation: `ValidateRollbackPlan`.

### sql_transformer.go
- **Purpose**: Applies SQL transformations (commenting destructive ops, modernizing syntax, performance tweaks) with plugin hook integration.
- **Key Types**: `SQLTransformer`, `TransformationConfig`, `TransformationResult`, `TransformationApplied`.
- **Functions / Methods**
  - Configuration & batching: `DefaultTransformationConfig`, `NewSQLTransformer`, `BatchTransform`, `ValidateTransformation`.
  - Main pipeline: `(*SQLTransformer) Transform`, `applyPluginTransformations`, `transformDMLToSelect`, `transformDropToComment`, `transformUnsafeToSafe`, `transformToModernSyntax`, `applyPerformanceTransformations`.
  - DML rewrites: `convertInsertToSelect`, `convertUpdateToSelect`, `convertDeleteToSelect`, `transformDMLToSelect`.
  - Syntax/semantics fixes: `fixCommonSyntaxErrors`, `fixReturnNextWithOutParams` (delegates to postprocessing), `fixCommentSyntax`, `normalizeLanguagePosition`, `fixFunctionVolatilityMarkers` (fallback for non-plugin functions), `hasVolatilityMarker`, `determineVolatility`, `extractFunctionName`.
  - Safety transforms: `transformDropToComment`, `transformUnsafeToSafe`, `transformToModernSyntax`.

## Subdirectories
- _None._
