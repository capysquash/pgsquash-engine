# internal/validation package map

## Domain Summary
- Executes schema validation workflows (Docker-based diffing, AI semantic checks, schema normalization) to ensure squashed output matches original behavior and satisfies safety levels.
- Produces detailed metrics, detects risky changes, generates repair suggestions, and exposes helper APIs for plugins and hosted orchestration.

## Files (alphabetical)

### ai_validator.go
- **Purpose**: AI-assisted validation (semantic analysis, dependency checks, quality reports, repair suggestions).
- **Key Types**: `AIValidator`, `AIValidationConfig`, `AIValidationResult`, `SemanticIssue`, `DependencyIssue`, `QualityReport`, `RepairSuggestion`.
- **Functions / Methods**
  - Entry points: `NewAIValidator`, `ValidatePostProcessing`.
  - Semantic analysis pipeline: `validateSemantics`, `validateDependencies`, `generateQualityReport`, `suggestOptimizations`, `evaluateOverallSuccess`.
  - Prompt handling: `chunkSQL`, `truncateForPrompt`.
  - Response parsing: `parseConfidence`, `parseSemanticIssues`, `parseDependencyIssues`, `parseQualityReport`.
  - Reporting: `printSummary`, `logInfo`.

### metrics.go
- **Purpose**: Collects and exports validation metrics (timings, counts, schema stats) to JSON/streams.
- **Key Types**: `ValidationMetrics`.
- **Functions / Methods**
  - Construction & lifecycle: `NewValidationMetrics`, `GetSnapshot`, `Reset`.
  - Recording helpers: `RecordValidation`, `RecordQuery`, `RecordError`, `RecordWarning`, `RecordDockerValidation`, `RecordSchemaObject`, `RecordExtensionInstall`, `RecordFix`, `UpdateResourceMetrics`.
  - Metadata setters: `SetValidationTimes`, `SetValidationLevel`, `SetPostgreSQLVersion`.
  - Exporters: `ExportJSON`, `ExportPrometheus`.
  - Utilities: `copyInt64Map`, `calculateSuccessRate`.

### publication_dedup.go
- **Purpose**: Removes duplicate publication `ALTER ... ADD TABLE` statements prior to validation to avoid false failures.
- **Functions**: `deduplicatePublicationStatements`, `preprocessMigrationSQL`.

### schema_changes.go
- **Purpose**: Concrete `SchemaChange` implementations (tables, columns, indexes, functions, constraints) used by the schema diff engine.
- **Key Types**: `SchemaChange` interface plus `TableCreateChange`, `TableDropChange`, `ColumnAddChange`, `ColumnDropChange`, `ColumnModifyChange`, `IndexCreateChange`, `IndexDropChange`, `IndexModifyChange`, `FunctionCreateChange`, `FunctionDropChange`, `FunctionModifyChange`, `ConstraintAddChange`, `ConstraintDropChange`, `ConstraintModifyChange`.
- **Functions / Methods**
  - Table changes: `(*TableCreateChange) Type`, `Risk`, `Description`, `SQL`, `ObjectID`, `Details`; `(*TableDropChange)` equivalents.
  - Column changes: `(*ColumnAddChange) Type/Risk/Description/SQL/ObjectID/Details`; `(*ColumnDropChange) ...`; `(*ColumnModifyChange) ...`.
  - Index changes: `(*IndexCreateChange) ...`; `(*IndexDropChange) ...`; `(*IndexModifyChange) ...`.
  - Function changes: `(*FunctionCreateChange) ...`; `(*FunctionDropChange) ...`; `(*FunctionModifyChange) ...`.
  - Constraint changes: `(*ConstraintAddChange) ...`; `(*ConstraintDropChange) ...`; `(*ConstraintModifyChange) ...`.
  - Builder helpers: `convertToBuilderTableDef`, `convertToBuilderIndexDef`, `convertToBuilderFunctionDef`, `formatConstraintDef`.

### schema_diff.go
- **Purpose**: Performs schema diff comparison between original and squashed outputs, classifies changes, assigns risk.
- **Key Types**: `Schema`, `SchemaDiffer`, `DiffConfig`, `ExpressionValidator`, `ObjectID`, `SchemaChange`.
- **Functions / Methods**
  - Construction: `(ObjectID) String`, `NewSchema`, `DefaultDiffConfig`, `NewSchemaDiffer`.
  - Diff pipeline: `(*SchemaDiffer) Compare`, `compareTables`, `compareTableDefinitions`, `compareColumns`, `columnsEqual`, `compareIndexes`, `indexesEqual`, `compareFunctions`, `functionsEqual`, `compareTableConstraints`, `constraintsEqual`.
  - Helpers: `stringSlicesEqual`, `getRiskPriority`.
  - Expression validation: `NewExpressionValidator`, `(*ExpressionValidator) ExpressionsEqual`, `ValidateExpression`.

### schema_normalizer.go
- **Purpose**: Normalizes pg_dump output (strip comments, sort blocks, canonicalize functions) for stable schema comparisons.
- **Key Types**: `SchemaNormalizer`, `SchemaValidator`, `NormalizedSchema`, `SchemaDiff`.
- **Functions / Methods**
  - Construction: `DefaultSchemaNormalizer`.
  - Dump utilities: `(*SchemaValidator) DumpAndNormalizeSchema`, `DumpAndNormalizeContainerSchema`, `DumpAndNormalizeContainerDatabase`.
  - Normalization pipeline: `Normalize`, `stripComments`, `removeOwnership`, `removePrivileges`, `removeOIDs`, `normalizeWhitespace`, `canonicalizeFunctions`, `sortBlocks`, `splitIntoBlocks`, `getBlockType`, `extractObjectName`.
  - Diff helpers: `(*NormalizedSchema) extractObjects`, `CompareNormalizedSchemas`, `(*SchemaDiff) compareObjects`, `extractShortName`, `(*SchemaDiff) FormatDiff`.

### validator.go
- **Purpose**: Main validation orchestrator—manages Docker environments, applies schema diff approaches, runs compatibility SQL, integrates plugins/AI, writes reports.
- **Key Types**: `ValidationConfig`, `SchemaValidator`, `ValidationResult`, `DockerValidationResult`, `ValidationFix`, `ContainerInfo`.
- **Functions / Methods**
  - Configuration: `DefaultValidationConfig`, `NewSchemaValidator`.
  - Migration validation: `ValidateMigrations`, `validateMigrationStructure`, `validateStatement`, `validateDependencies`, `validateWithDatabase`, `validatePerformance`, `validateNamingConventions`, `validateObjectSpecificRules`, `validateStatementConstraints`, `validatePostgreSQLFeatures`, `analyzeStatementPerformance`, `extractExpressionFromStatement`, `canExplainStatement`, `SortValidationResults`.
  - Docker orchestration: `ValidateWithDocker`, `runDockerValidation`, `validateWithTwoDatabases`, `validateWithTwoContainers`, `validateWithSchemaDiff`.
  - Extension discovery: `detectRequiredExtensions`, `scanDirectoryForExtensions`, `resolveExtensionAlias`, `extractExtensionsFromSQL`, `deduplicateExtensions`, `fixDuplicateFunctions`.
  - Container management: `createEnhancedContainer`, `findAvailablePort`, `isPortAvailable`, `ensureDockerImageAvailable`, `stopAndRemoveContainer`, `waitForContainerStart`, `waitForPostgreSQLReady`, `waitForContainer`, `installExtensions`, `installExtensionsViaPackageManager`, `execInContainer`, `setupDatabases`.
  - Migration execution: `applyMigrationsToDatabase`, `executeSQLFile`, `splitSQLStatements`, `applyMigrationsToContainer`.
  - Comparison helpers: `compareSchemasWithNormalization`, `dumpContainerSchema`, `compareSchemas`, `getTables`, `getIndexes`, `compareStringSlices`, `getDefaultExtensionMap`, `getFunctions`, `getTriggers`, `getViews`, `getSequences`, `getCustomTypes`, `getExtensions`.
  - Fixes & reporting: `fixSQLIssues`, `applySQLFixes`, `logInfo`, `Close`, `SetAIValidator`, `ValidateWithAI`, `getPluginCompatibilitySQL`.

## Subdirectories
- _None._
