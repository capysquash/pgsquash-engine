# internal/patterns package map

## Domain Summary
- Central repository of precompiled regular expressions used throughout the engine for SQL identification, transformation heuristics, dependency extraction, and function analysis.
- Compiles patterns once to avoid repetitive regex creation across parsers, transformers, and dependency resolvers.

## Files (alphabetical)

### patterns.go
- **Purpose**: Declares and compiles regex patterns reused across packages (`parser`, `transformation`, `squasher`, etc.).
- **Variables**
  - Parsing helpers: `FunctionPattern`, `CreateTablePattern`, `AlterTablePattern`, `CreateSchemaPattern`, `CreateIndexPattern`.
  - Transformation heuristics: `InsertPattern`, `UpdatePattern`, `DeletePattern`, `DropTablePattern`, `DropColumnPattern`, `AlterTypePattern`, `NoIndexPattern`, `LargeScanPattern`, `OldJoinPattern`, `OldFunctionPattern`.
  - Detailed DML patterns: `InsertDetailPattern`, `UpdateDetailPattern`, `SetClausePattern`, `DeleteDetailPattern`.
  - Function analysis: `ReturnsTablePattern`, `FunctionBodyPattern`, `ReturnNextPattern`, `FunctionWhitespacePattern`, `FunctionSignaturePattern`, `FunctionNamePattern`.
  - Dependency resolution: `ForeignKeyPattern`, `DirectRefPattern`, `SQLCommentPattern`, `CommentOnPattern`, `QualifiedNamePattern`, `InsertIntoPattern`, `ExecuteFunctionPattern`, `FunctionCallPattern`, `SetParameterPattern`, `UpdateTablePattern`, `CreateTableDetailPattern`, `CreateFunctionDetailPattern`, etc.
- **Functions**
  - `BuildPatternList(patterns ...*regexp.Regexp) []*regexp.Regexp`: Utility that compacts nil-safe regex slices for callers needing ordered pattern lists.
  - `BuildPatternWithEscape(pattern string) *regexp.Regexp`: Builds a single regex while escaping SQL meta-characters to avoid accidental broad matches.

- **Usage**: Patterns and helpers are referenced by other packages via import to avoid recompilation and maintain consistent matching semantics.

## Subdirectories
- _None._
