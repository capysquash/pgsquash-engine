# internal/builder package map

## Domain Summary
- Provides a fluent SQL generation layer tailored to PostgreSQL, used to rebuild statements from parser output or construct migrations programmatically.
- Encapsulates formatting, quoting, and reconstruction logic so higher-level packages can emit consistent SQL without manual string building.
- Bridges AST output from the parser (`internal/types` + `pg_query`) back into normalized SQL while honoring formatting preferences.

## Cross-Cutting Concepts
- **Formatting Profiles**: `BuildOptions` and `FormatStyle` drive whitespace, indentation, quote style, and comment preservation—`FormatPretty` inserts newlines/indentation, `FormatDense` squeezes output, and `FormatCompact` leaves explicit separators only.
- **Quoting Rules**: `needsQuoting` respects PostgreSQL keyword lists, mixed case, special characters, and numeric prefixes; honors `UseDoubleQuotes` and `NormalizeNames` flags.
- **Error Buffering**: The builder collects non-fatal issues in `errors []error`, allowing callers to inspect `HasErrors`/`Errors` alongside partial SQL output (current generator methods rely on this when fallbacks trigger).
- **AST Round-Tripping**: `FromStatement` first attempts AST-based regeneration (via `pg_query.Deparse` for create/alter paths) before falling back to stored SQL when parsing fails.

## Files (alphabetical)

### sql.go
- **Purpose**: Implements the `SQLBuilder` fluent API, associated definition structs, keyword helpers, and conversion utilities from parsed statements.
- **Key Types**
  - `SQLBuilder`: Buffer + formatting state used to incrementally assemble SQL.
  - `BuildOptions`: User-configurable formatting settings (style, indentation, quoting).
  - `FormatStyle`: Enum of supported formatting modes (`FormatCompact`, `FormatPretty`, `FormatDense`).
  - Definition structs (`TableDefinition`, `ColumnDefinition`, `ConstraintDefinition`, `IndexDefinition`, `IndexColumn`, `FunctionDefinition`, `ParameterDefinition`) that describe SQL objects the builder can render.
- **Functions / Methods**
  - Constructors: `DefaultBuildOptions` (Pretty style, double quotes, comments enabled) and `NewSQLBuilder` (ensures defaults when options nil).
  - Fluent primitives: `P`, `Statement`, `S`, `NL`, `Indent`, `Dedent`, `Wrap`, `Quote`, `QuoteQualifiedName` (drops `public` schema), `Comment` (respects `PreserveComments`), `String`, `Errors`, `HasErrors`, `Reset`.
  - High-level builders:
    - `CreateTable`: Emits column/constraint lists with indentation, supports `IF NOT EXISTS`, table inheritance, and optional tablespace.
    - `CreateIndex`: Handles uniqueness, `IF NOT EXISTS`, non-default methods, expression columns, per-column ordering, and partial-index predicate.
    - `CreateFunction`: Automatically strips duplicate LANGUAGE suffixes from the body, infers language from body content, and re-applies volatility/strict/security flags prior to emitting the body.
  - Internal helpers: `buildColumnDefinition` (generated columns, defaults, collation), `buildConstraintDefinition` (foreign key clauses, deferrable rules, check expressions), `needsQuoting`.
  - Keyword utility: `IsPostgreSQLKeyword` (hand-maintained keyword map used by quoting rules).
  - Statement reconstruction:
    - `FromStatement`: Routes operations to specialized handlers; defaults to original SQL when type unsupported.
    - `fromASTStatement`: Uses stored `pg_query.ParseResult` when available; falls back on raw SQL on errors.
    - `fromDropStatement`: Guards against unknown object types by writing advisory comments instead of invalid SQL.
    - `fromGrantStatement` / `fromRevokeStatement`: Rebuild grant syntax using stored privilege lists and grantees.

## Subdirectories
- _None._
