# internal/types package map

## Domain Summary
- Central type system that prevents import cycles: defines migration/statement metadata, PostgreSQL catalog models, and analyzer results consumed by parser, tracking, squasher, validation, and plugins.
- Aggregates helper methods for type compatibility checks and migration type analysis so upstream packages can reason about schema changes consistently.

## Files (alphabetical)

### parser_types.go
- **Purpose**: Canonical structs and enums representing parsed migrations, statements, execution metadata, dependencies, and provenance.
- **Key Types (highlights)**
  - Migration primitives: `Migration`, `MigrationWithContent`, `Statement`, `StatementMetadata`, `StatementSummary`.
  - Classification enums: `StatementCategory`, `StatementSubcategory`, `LockLevel`, `Operation`, `RiskLevel`, `AuthPattern`, `ObjectType`, `DDLAction`.
  - Dependency models: `Dependency`, `DependencyType`, `DependencyGraphEdge`, `CrossSchemaReference`.
  - Execution metadata: `ExecutionPlan`, `ExecutionPhase`, `ExecutionMetadata`, `TransactionRequirement`, `LockRequirement`.
  - Provenance/reporting: `RedundantObject`, `ObjectChange`, `PolicyMetadata`, `FunctionMetadata`, `DataOperation`, `RiskInsight`.
  - Auth/AI support: `AuthPatternMetadata`, `AnalysisResult`, `OptimizationInsight`, `SchemaDifference`.
- **Notes**: File defines JSON tags and helper slices but contains no functions—logic lives in other packages that consume these structures.

### postgresql_types.go
- **Purpose**: Represents PostgreSQL type system (custom types, domains, enums, arrays, ranges) and exposes compatibility/size helpers.
- **Key Types**
  - `PostgreSQLTypeSystem`: Central registry containing builtin types, implicit conversions, and metadata maps.
  - Type models: `CustomType`, `Domain`, `CompositeType`, `EnumType`, `ArrayType`, `RangeType`, `TypeCompatibility`, `ConversionCategory`, `Attribute`.
- **Functions / Methods**
  - `NewPostgreSQLTypeSystem(version)`: Seeds builtin map and version-specific features.
  - Registration: `RegisterCustomType`, `RegisterDomain`, `RegisterCompositeType`, `RegisterEnumType`.
  - Lookup & compatibility:
    - `IsBuiltinType(typeName)`: Checks against builtin catalog.
    - `ParseArrayType(typeSpec)`: Parses `type[]` strings into `ArrayType`.
    - `CheckTypeCompatibility(fromType, toType)`: Returns `TypeCompatibility` describing conversion category and potential data loss.
    - `normalizeTypeName`, `hasImplicitConversion`, `hasAssignmentConversion`, `hasExplicitConversion`, `isPotentiallyLossyConversion`, `checkSpecialCases`: Internal helpers supporting compatibility analysis.
    - `isInSlice`: Generic membership helper.
  - Size & validation:
    - `GetTypeSize(typeName)`: Returns byte size / `-1` for variable.
    - `extractSizeFromSpec(typeSpec)`: Parses `varchar(255)` style limits.
    - `ValidateEnumValue(enumTypeName, value)`: Ensures enum label exists.
    - `GetCompositeTypeAttributes(typeName)`: Returns composite field list.
    - `IsCompatibleArrayDimensions(from, to)`: Checks array dimensionality match.

### type_analyzer.go
- **Purpose**: AST-driven analyzer that inspects SQL statements/migrations to determine type usage, data-loss risks, and conversion requirements.
- **Key Types**
  - `TypeAnalyzer`: Holds `PostgreSQLTypeSystem`, DB connection, caches.
  - `TypeInfo`, `TypeChange`, `TypeConversion`, `MigrationTypeAnalysis`.
- **Functions / Methods**
  - Construction & entry points:
    - `NewTypeAnalyzer(typeSystem, db)`: Initializes analyzer.
    - `(*TypeAnalyzer) AnalyzeStatement(ctx, sql)`: Returns `TypeInfo` slice for a statement.
    - `(*TypeAnalyzer) AnalyzeMigrationTypes(ctx, statements)`: Aggregates type changes across migration list.
  - AST extraction helpers:
    - `extractTypesFromNode`, `extractTypesFromCreateTable`, `extractTypesFromAlterTable`, `extractTypesFromCreateDomain`, `extractTypesFromCreateEnum`, `extractTypesFromCreateComposite`, `extractTypesFromCreateFunction`: Walk specific pg_query nodes to collect types.
    - `analyzeColumnType`: Derives `TypeInfo` for column definitions (including modifiers).
    - `extractTypeNameFromColumnDef`, `extractTypeNameFromNode`: Determine normalized type names from AST nodes.
    - `getOrCreateTypeInfo`: Ensures deduplicated `TypeInfo` instances.
    - `detectTypeChanges`: Compares before/after statements for conversion needs.
  - Conversion guidance:
    - `GenerateTypeConversion(fromType, toType, columnName)`: Builds conversion plan and warns about risks.
    - `generateDataLossDescription`: Human-readable risk explanation.

## Subdirectories
- _None._
