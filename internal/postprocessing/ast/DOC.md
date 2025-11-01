# internal/postprocessing/ast package map

## Domain Summary
- AST-focused utilities that normalize function definitions and rewrite enum references using `pg_query` parse trees, enabling more accurate post-processing than regex alone.

## Files (alphabetical)

### enum_replacer.go
- **Purpose**: Rewrites references to consolidated ENUM types within the AST.
- **Key Types**
  - `EnumReplacer`: Holds replacement map and logger.
- **Functions / Methods**
  - `NewEnumReplacer`, `(*EnumReplacer) ReplaceEnumReferences`, `(*EnumReplacer) GetReplacedCount`.
  - Helpers for traversing AST nodes, updating `TypeName` entries, and serializing back to SQL.

### function_normalizer.go
- **Purpose**: Normalizes function definitions (LANGUAGE clauses, dollar-quote structure, duplicate declarations) via AST manipulation.
- **Key Types**
  - `FunctionNormalizer`: Coordinates parsing, normalization, and reconstruction.
- **Functions / Methods**
  - Constructors: `NewFunctionNormalizer`.
  - Entry points: `(*FunctionNormalizer) NormalizeAll`, `normalizeCreateFunction`, `deparseNode`.
  - Helpers: `visitCreateFunction`, `visitCreateTable`, `visitAlterTable`, `visitNode`, `extractFunctionBody`, `inferMissingLanguage`, `inferLanguageFromFunction`, `fixLanguageOrder`, `removeDuplicateOptions`, `removeRedundantLanguage`, `replaceTypeNameIfNeeded`, `traverseAndReplace`, `getTypeNameFromTypeName` (all cooperate to walk/modify AST nodes safely).

### function_normalizer_test.go
- **Purpose**: Tests ensuring normalization handles edge cases (auth.jwt, double dollar quotes, comment preservation).
- **Functions**
  - `TestFunctionNormalizer_FixLanguageOrder`, `TestFunctionNormalizer_InferMissingLanguage`, `TestFunctionNormalizer_RemoveDuplicateLanguage` validate the most critical normalization paths.
- **Notes**: Reference for expected before/after SQL scenarios.

## Subdirectories
- _None._
