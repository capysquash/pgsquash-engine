# internal/plugins/volatility package map

## Domain Summary
- Shared tooling for detecting and fixing PostgreSQL function volatility markers across plugins (Clerk, Supabase, etc.).
- Provides both regex- and AST-based fixers plus registries that map function names to required volatility levels.

## Files (alphabetical)

### fixer.go
- **Purpose**: Regex-oriented volatility fixer and function registry utilities.
- **Key Types**
  - `VolatilityType`: Enum of PostgreSQL volatility levels (`IMMUTABLE`, `STABLE`, `VOLATILE`).
  - `FunctionRegistry`: Tracks functions and desired volatility.
  - `VolatilityFixer`: Regex-based fixer that inserts markers before `AS` clause.
- **Functions / Methods**
  - Registry helpers: `NewFunctionRegistry`, `Register`, `RegisterMultiple`, `GetVolatility`, `IsRegistered`, preset constructors (`CreateSupabaseRegistry`, `CreateClerkRegistry`, `CreateDrizzleRegistry`, `CreatePrismaRegistry`).
  - Fixer lifecycle: `NewVolatilityFixer`, `(*VolatilityFixer) Fix`.
  - Utility: `hasVolatilityMarker`, preset registries (`CreateClerkRegistry`, `CreateSupabaseRegistry`, etc.).

### fixer_ast.go
- **Purpose**: AST-driven volatility fixer leveraging `pg_query` for more reliable detection.
- **Key Types**
  - `ASTVolatilityFixer`: Uses parsed AST to inspect function options.
- **Functions / Methods**
  - `NewASTVolatilityFixer`, `(*ASTVolatilityFixer) Fix`.
  - Helpers: `extractFunctionName`, `hasVolatilityInAST`, `addVolatilityToFunction`, `containsFunctionName`, `findASKeywordIndex`, `FixSQL`.

## Subdirectories
- _None._
