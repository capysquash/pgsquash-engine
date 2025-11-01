# internal package map

## Domain Summary
- Engines the entire squash workflow: parsing PostgreSQL migrations, consolidating statements, tracking lifecycle state, validating results, and emitting CLI/TUI experiences.
- Exposes composable domains (AI, parser, squasher, transformation, validation, plugins, tracking, performance) that higher layers (`cmd/pgsquash`, `cmd/api-server`) orchestrate.
- Shares foundational types, error handling, and utilities so packages can interoperate without import cycles.

## Subpackages (alphabetical)
Each subpackage now ships with an exhaustive function/class inventory in its local `DOC.md`; refer to those documents for per-file details and timestamps of the latest audit (November 1 2025).
- `ai/`: Provider-agnostic AI analysis and remediation helpers. See `internal/ai/DOC.md`.
- `builder/`: SQL reconstruction and formatting engine used during consolidation.
- `cli/`: Cobra command wiring for the standalone CLI.
- `config/`: JSON/YAML configuration schemas, loaders, and validators.
- `errors/`: Structured error model, severities, categories, and collectors.
- `fileutil/`: Opinionated file I/O helpers with backup semantics.
- `metadata/`: PostgreSQL catalog introspection and schema comparison.
- `parser/`: pg_query-backed SQL parser with normalization and metadata enrichment.
- `patterns/`: Centralized regex patterns reused by parsing and transformation.
- `performance/`: Memory management, streaming ingestion, and progress reporting.
- `plugins/`: Plugin registry plus concrete adapters for auth providers and ORMs.
- `postprocessing/`: Regex and AST fixups that polish consolidated SQL.
- `squasher/`: Core consolidation engine, provenance writer, safety validation.
- `tracking/`: Object lifecycle tracking, consolidation rules, recovery strategies, and reporting surfaces.
- `transformation/`: Backup generation, rollback management, SQL rewrites.
- `types/`: Shared struct and enum definitions consumed across packages.
- `utils/`: Logger, string/SQL helpers, enum validation, warning management.
- `validation/`: Docker/AI validation workflows, schema diffing, risk metrics.

## Cross-Package Patterns
- **Structured Errors**: Nearly every package wraps failures with `internal/errors.StructuredError`, enabling consistent telemetry and UX.
- **Typed Metadata**: `internal/types` supplies migration, statement, and PostgreSQL type models used by parser, tracking, squasher, plugins, and validation.
- **Plugin & AI Hooks**: Parser, squasher, transformation, and validation call into `plugins` and `ai` to extend behavior or request semantic guidance.
- **Streaming Workflow**: `performance`, `tracking`, and `squasher` coordinate to process large migration directories incrementally.

## Key Entry Points
- CLI (`cmd/pgsquash`) constructs `squasher.Engine`, `parser.Parser`, `validation.Validator`, and optional AI/streaming/tracking components from this package tree.
- Hosted API layer (`cmd/api-server`) mirrors CLI orchestration, relying on the same subpackages for parity between local and hosted runs.
