# internal/postprocessing package map

## Domain Summary
- Runs the post-processing pipeline over consolidated SQL, blending regex and AST fixes to repair syntax, normalize function language clauses, reorder extensions, patch enum references, and tidy return semantics.
- Provides reusable helpers that ensure final output is production-safe even when upstream consolidation introduced ordering or language anomalies.

## Files (alphabetical)

### extension_ordering.go
- **Purpose**: Reorders `CREATE EXTENSION` statements according to dependency rules.
- **Functions**
  - `FixExtensionOrder`: Extracts extension statements, enforces canonical ordering, and reinserts into extension section.
  - `SortExtensionsByDependency`: Utility for future enhancements; currently hardcodes known dependencies.

### function_language.go
- **Purpose**: Normalizes function `LANGUAGE` clauses (remove duplicates, infer correct language, add missing clauses).
- **Functions**
  - `FixFunctionLanguageConflicts`, `FixRedundantTrailingLanguageClauses`, `FixIncorrectLanguageDeclarations`, `FixMissingLanguageDeclarations`, `RemoveDuplicateLanguageDeclarations`.
  - Helpers to detect PL/pgSQL constructs, comment handling, and SQL normalization.

### function_returns.go
- **Purpose**: Repairs `RETURN NEXT` usage and OUT parameter handling inside functions.
- **Functions**
  - `FixReturnNextWithOutParams`, `WrapReturnNext`, plus helpers for `RETURN QUERY` transformation and OUT parameter detection.

### postprocessor.go
- **Purpose**: Orchestrates the multi-phase post-processing pipeline (syntax fixes, language normalization, function body fixes, final cleanup).
- **Key Types**
  - `Processor`: Holds logger, config, AST helpers (`FunctionNormalizer`, `EnumReplacer`), feature flags.
- **Functions / Methods**
  - Constructors: `NewProcessor` and `NewProcessorAST` (preconfigures AST-first mode).
  - Runtime toggles: `SetUseASTForFunctions`, `SetUseASTForEnums`.
  - Pipeline execution: `(*Processor) Apply`, internal helper `fixEliminatedEnumReferences`.

### postprocessor_ast.go
- **Purpose**: AST-driven post-processing helpers (function normalization, enum replacement).
- **Functions / Types**
  - `normalizeFunction`, `normalizeFunctionLanguage`, `replaceLanguageClause`.
  - Entry points: `NormalizeFunctions`, `ReplaceEnumReferences`.

### syntax_fixes.go
- **Purpose**: Fixes general syntax issues (malformed DROP TRIGGER, orphaned ALTER, missing semicolons, function scaffolding).
- **Functions**
  - High-level repairs: `FixMalformedDropTriggers`, `RemoveOrphanedAlterStatements`, `FixMalformedFunctions`, `FixMissingSemicolons`, `applyRegexFunctionFixes`.
  - Helpers: `parseTableColumns` plus other internal utilities that sanitize CREATE TABLE sections before reinsertion.

## Subdirectories
- `ast/`: AST helpers such as `FunctionNormalizer` and `EnumReplacer`; see `internal/postprocessing/ast/DOC.md`.
