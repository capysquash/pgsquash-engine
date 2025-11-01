# internal/errors package map

## Domain Summary
- Centralizes structured error handling for the engine, providing consistent severity/category metadata, helpful context, aggregation utilities, and formatting helpers.
- Offers convenience constructors for common error families (parsing, validation, transformation, type normalization) and warning/critical flows.
- Bridges engine internals, CLI, and hosted services by enforcing error codes/severities that downstream consumers (CLI UX, telemetry, GitHub automation) can treat uniformly.
- Supplies rich context capture (file, schema, SQL, suggestions, additional metadata) so diagnostics remain actionable across logs, TUI panels, and API responses.

## Cross-Cutting Concepts
- **Severity Ladder**: `SeverityInfo` → `SeverityCritical` determines `CanContinue` defaults and drives CLI/TUI colour coding.
- **Category Catalog**: Unified categories (syntax, validation, transformation, cycle, optimization, backup, etc.) align with warning manager taxonomy and analytics dashboards.
- **Error Codes**: Stable `ErrorCode` strings group related issues (validation failures, transformation errors, normalization issues) for telemetry and automated workflows.
- **Fluent Builder Pattern**: `StructuredError` exposes `With*` helpers for chaining context fields, suggestions, and inner errors without allocating new structs.
- **Collectors & Formatters**: `ErrorCollector` and `ErrorFormatter` provide reusable plumbing for aggregating issues, generating summaries, and rendering human-readable reports.

## Files (alphabetical)

### errors.go
- **Purpose**: Implements the structured error model, enum definitions, collectors/formatters, and helper constructors.
- **Key Types**
  - Severity & Category enums with string helpers.
  - `ErrorCode`: Stable identifiers for engine error/warning conditions.
  - `ErrorContext`: Optional metadata (file, line, object, SQL, etc.).
  - `StructuredError`: Core error type supporting suggestions, inner errors, and continuation flags.
  - `ErrorCollector`: Aggregates errors/warnings with summary stats and context-aware filtering.
  - `ErrorSummary`: Roll-up counts by category/severity/code for dashboards and CLI output.
  - `ErrorFormatter`: Produces human-readable summaries and numbered lists.
  - Convenience types: `ValidationErrors`, `ConfigValidationError`, `JSONSyntaxError`, `JSONTypeError`, `GenericJSONError`.
- **Functions / Methods**
  - Enum helpers: `Severity.String`.
  - Constructors: `NewError`, `NewParseError`, `NewValidationError`, `NewTransformationError`, `NewTypeError`, `NewNormalizationError`, `NewWarning`, `NewCriticalError`, `New`, `Wrap`.
  - Context builders on `StructuredError`: `WithContext`, `WithFile`, `WithLine`, `WithColumn`, `WithObject`, `WithSchema`, `WithStatement`, `WithTypeName`, `WithSQLQuery`, `WithAdditional`, `WithSuggestion`, `WithInnerError`, `WithCanContinue`.
  - Error interface hooks: `(*StructuredError).Error`, `(*StructuredError).Unwrap`.
  - Collector operations: `NewErrorCollector`, `(*ErrorCollector).AddError`, `HasErrors`, `HasWarnings`, `GetErrors`, `GetWarnings`, `GetAllIssues`, `Clear`, `Summary`.
  - Formatter routines: `NewErrorFormatter`, `(*ErrorFormatter).FormatError`, `FormatSummary`, `FormatErrorList`.
  - `(*ValidationErrors).Error` and other error type `Error()` methods.

## Subdirectories
- _None._
