# internal/parser package map

## Domain Summary
- Wraps the `pg_query` PostgreSQL parser to produce enriched `types.Statement` objects with normalized metadata, dependency graphs, naming diagnostics, pragma analysis, and plugin annotations.
- Includes normalization, keyword, and error-handling utilities plus statement-level analysis for lock levels, execution cost, and idempotency.
- Designed for resiliency: collects parse errors without aborting entire migrations, supports plugin hooks, and standardizes StructuredError emission for downstream tooling.

## Files (alphabetical)

### errors.go
- **Purpose**: Parser-specific error taxonomy, collectors, formatters, and resilient error handling.
- **Key Types**
  - `ParseError`: Captures message, severity, category, suggestions, parse context.
  - `ErrorSeverity`, `ErrorCategory`: Enum-like aliases for parse diagnostics.
  - `ParseContext`: Holds filename, line, SQL fragment, statement metadata.
  - `ErrorCollector`: Aggregates parse errors/warnings with helper methods.
  - `ErrorReporter`, `ErrorFormatter`: Produce formatted summaries.
  - `ErrorHandler`: Wraps panic recovery and orchestrates parse error escalation.
- **Functions / Methods**
  - Constructors: `NewParseError`, `NewErrorCollector`, `NewErrorReporter`, `NewErrorFormatter`, `NewErrorHandler`.
  - Collector helpers: `AddError`, `AddSyntaxError`, `AddSemanticError`, `AddDependencyError`, `AddNamingWarning`, `AddPerformanceWarning`, `GetErrors`, `GetWarnings`, `GetAllIssues`.
  - Formatter routines: `(*ErrorFormatter).FormatError`, `FormatErrorList`.
  - Handler operations: `HandleParseError`, `HandleValidationError`, `CreateContext`, `GetCollector`, `GetReporter`, `Recovery`, `ShouldContinue`, `LogSummary`.
  - Utility: `detectsMissingSemicolon`.

### normalization.go
- **Purpose**: Context-aware normalization of PostgreSQL identifiers and keyword management.
- **Key Types**
  - `NormalizationContext`: Configures defaults (schema, case sensitivity, max length, version).
  - `ContextualNormalizer`: Applies context to identifiers, schemas, tables, functions.
  - `VersionedKeywordManager`: Tracks keyword sets per PostgreSQL version.
  - `KeywordType`, `KeywordContext`, `ContextualKeywordChecker`.
  - `NormalizationError`: Structured errors for normalization routines.
- **Functions / Methods**
  - Context helpers: `DefaultNormalizationContext`, `NewContextualNormalizer`.
  - Normalization APIs: `NormalizeIdentifier`, `NormalizeSchemaName`, `NormalizeTableName`, `NormalizeFunctionName`, `ParseQualifiedName`, `BatchNormalize`, `CompareIdentifiers`.
  - Keyword management: `NewVersionedKeywordManager`, `loadBaseKeywords`, `loadVersionSpecificKeywords`, `addVersionKeywords`, `IsKeyword`, `GetKeywordType`, `IsReservedKeyword`.
  - Contextual checks: `NewContextualKeywordChecker`, `IsKeywordInContext`, and internal `isDDLKeyword`, `isDMLKeyword`, `isFunctionKeyword`, `isConstraintKeyword`.
  - Errors & validators: `NewNormalizationError`, `IsValidPostgreSQLIdentifier`, `SuggestIdentifierName`.

### parser.go
- **Purpose**: Main migration parsing pipeline—splits SQL, normalizes statements, extracts dependencies, validates naming, and enriches results.
- **Key Functions**
  - Entry points: `ParseMigration`, `ParseMigrationWithContext`.
  - Statement parsing: `parseStatementWithNormalizationAndContext`.
  - Statement analysis helpers:
    - `analyzeStatementWithNormalization`, `categorizeStatement`.
    - Object extraction: `getTableNameWithNormalization`, `extractTableDependenciesWithNormalization`, `extractTypeNameFromTypeName`, `isBuiltInType`.
    - Dependency traversal: `extractQueryDependencies`, `extractJoinDependencies`, `extractAlterTableConstraints`, `extractFunctionNameWithNormalization`, `extractGranteesWithNormalization`, `extractPrivileges`, `extractRoleNameWithNormalization`, `extractGranteeRolesWithNormalization`.
    - Mapping utilities: `mapGrantObjectType`, `mapObjectType`, `mapCommentObjectType`.
    - Comment handling: `extractComments`, `getRelevantComments`, `extractObjectNameWithNormalization`, `extractCommentObjectNameWithNormalization`, `extractCommentDependenciesWithNormalization`.
    - Schema inference: `extractSchemaWithNormalization`, `extractSchemaFromAST`, `extractSchemaFromObjectList`.
    - Naming checks: `validateNamingConventions`, `validateTableNaming`, `validateIndexNaming`, `validateFunctionNaming`, `validateConstraintNaming`, `validatePolicyNaming`.
    - Metadata enrichment: `extractCrossSchemaReferences`, `detectAuthPattern`, `isDynamicSQL`, `containsComplexLogic`.
    - SQL cleanup & helpers: `cleanSQL`, `extractNestedTypesFromDoBlock`, `enrichStatementWithPlugins`.

### statement_analyzer.go
- **Purpose**: Adds execution metadata such as lock levels, transaction requirements, idempotency, and pragma processing to parsed statements.
- **Key Types**
  - `StatementAnalyzer`: Configured per PostgreSQL version.
- **Functions / Methods**
  - `NewStatementAnalyzer`, `(*StatementAnalyzer) AnalyzeStatement`, `analyzeMetadata`.
  - Lock inference: `determineLockLevel`, `determineCreateLockLevel`, `determineAlterLockLevel`, `determineDropLockLevel`, `isConcurrent`.
  - Transaction / feature gates: `requiresNoTransaction`, `determineVersionGate`.
  - Behavioral insights: `isIdempotent`, `estimateExecutionTime`.
  - Pragma handling: `AnalyzePragmas`.
  - Utility: `FormatLockLevel`.

## Subdirectories
- _None._
