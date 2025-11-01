# internal/utils package map

## Domain Summary
- Shared utility helpers spanning enum validation, structured logging, parenthesis parsing, SQL/token extraction, string normalization, and warning aggregation. These utilities keep higher-level packages lean while enforcing consistent behavior.

## Files (alphabetical)

### enum_validation.go
- **Purpose**: Validates enum-like identifiers used across the engine (`types.ObjectType`, `types.Operation`, `types.Category`).
- **Functions**
  - `ValidObjectTypes()`: Returns supported object types.
  - `IsValidObjectType(t)`: Boolean guard for object type membership.
  - `ValidateObjectType(t)`: Returns structured error when object type invalid.
  - `ValidOperations()`: Lists supported operation enums.
  - `IsValidOperation(op)`: Checks if operation is recognized.
  - `ValidateOperation(op)`: Produces validation error for unsupported operations.
  - `ValidCategories()`: Enumerates supported categories.
  - `IsValidCategory(c)`: Membership check for categories.
  - `ValidateCategory(c)`: Structured error when category invalid.
  - `IsDDLOperation(op)`: True when operation is DDL.
  - `IsDMLOperation(op)`: True when operation is DML.
  - `IsSecurityOperation(op)`: True when operation targets auth/security.

### logger.go
- **Purpose**: Lightweight structured logger with log levels, prefixes, and global singleton.
- **Functions / Methods**
  - `LogLevel.String()`: Human-readable level string.
  - `NewLogger(minLevel, output)`: Constructs logger with mutex-protected writer.
  - `(*Logger) WithPrefix(prefix)`: Returns child logger with appended prefix.
  - `(*Logger) Debug/Info/Warn/Error/Fatal`: Level-specific formatted logging (Fatal exits).
  - `(*Logger) StandardLogger(level)`: Adapts logger to `*log.Logger`.
  - `SetDefaultLogger(logger)`, `GetDefaultLogger()`: Manage global singleton.
  - Internal helpers: `(*Logger) log`, `(*logWriter) Write`.

### parens.go
- **Purpose**: Balanced-parentheses parsing helpers for SQL inspection.
- **Functions**
  - `ExtractBalancedParentheses(text)`: Returns first balanced block.
  - `ExtractAllBalancedParentheses(text)`: Returns all balanced segments.
  - `HasBalancedParentheses(text)`: Boolean check.
  - `StripOutermostParentheses(text)`: Removes wrapping parentheses pair when balanced.

### sql_parsing.go
- **Purpose**: Regex-assisted SQL parsing helpers for object extraction.
- **Functions**
  - `ExtractPolicyTargetTable(sql)`: Pulls table from `ALTER POLICY`.
  - `ExtractFunctionName(sql)`: Finds function identifier (schema-qualified aware).
  - `ExtractTableName(sql)`: Returns table name from CREATE/ALTER.
  - `ExtractSchemaName(sql)`: Extracts schema qualifier.
  - `ExtractIndexName(sql)`: Extracts index identifier.
  - `IsDMLStatement(sql)`: Detects INSERT/UPDATE/DELETE.
  - `IsDDLStatement(sql)`: Detects CREATE/ALTER/DROP.

### strings.go
- **Purpose**: String normalization utilities used by parser and transformers.
- **Functions**
  - `NormalizeObjectName(name)`: Lowercases and strips quotes safely.
  - `ContainsKeyword(sql, keyword)`: Case-insensitive keyword search.
  - `HasClause(sql, clause)`: Clause detection for heuristics.
  - `HasVolatilityMarker(sql)`: Checks for IMMUTABLE/STABLE/VOLATILE.
  - `TrimSQLComments(sql)`: Removes line/block comments.
  - `NormalizeSQLWhitespace(sql)`: Collapses whitespace for comparisons.

### warning_manager.go
- **Purpose**: Collects, deduplicates, and formats warnings emitted across engine.
- **Functions / Methods**
  - `NewWarningManager()`: Constructor with hashing map.
  - `(*WarningManager) AddWarning/AddRawWarning/AddRawWarnings`: Push warnings (structured or string).
  - `GetWarnings()`: Returns deduplicated slice.
  - `GetWarningsByCategory()`, `GetWarningsBySeverity()`: Grouping helpers.
  - `Count()`, `CountBySeverity()`, `HasCriticalWarnings()`: Aggregate stats.
  - Internal helpers: `hashWarning`, `categorizeRawWarning`, `FormatWarnings` (pretty output), `SeverityString`.

## Subdirectories
- _None._
