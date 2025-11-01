# internal/cli package map

## Domain Summary
- Provides the Cobra-based command-line interface for pgsquash, wiring together analysis, squashing, validation, AI tooling, and the interactive TUI.
- Owns command registration, global flag parsing, workflow presets, and user-experience safeguards (branch safety prompts, quiet/verbose handling, branding).
- Acts as the glue between orchestration layers (CLI, Docker health checks, hosted API) and engine packages, coordinating configuration loading, AI integration, validation, and file IO for end-users.

## Cross-Cutting Concepts
- **Global Flags & Modes**: Persistent flags (`--config`, `--verbose`, `--quiet`, `--no-emoji`, `--tui`, etc.) and workflow-specific toggles control downstream behavior such as streaming, explain/dry-run, safety level, and validation approach.
- **AI Feature Gates**: Handlers gracefully degrade when AI providers are unavailable, logging warnings and invoking non-AI fallbacks for workflows/explain commands.
- **Validation & Reporting**: Common helpers standardize validation outputs (Docker schema diff vs two-database) and textual reporting (analysis summaries, squashing warnings, backup paths).
- **Streaming Pipelines**: Commands select streaming engines automatically for large inputs (>100 migrations) while exposing flags for manual control (`--streaming`, `--memory-limit`).
- **Branding & Versioning**: `SetBrandName` and `SetVersionInfo` inject alternate product names and build metadata, allowing white-label binaries to reuse CLI logic verbatim.

## Files (alphabetical)

### branch_safety.go
- **Purpose**: Protects against dangerous squashing on non-protected git branches.
- **Key Types**
  - `BranchSafetyChecker`: Encapsulates protected branch list and user prompts.
- **Functions / Methods**
  - `NewBranchSafetyChecker`: Constructor with default protected branch names.
  - `(*BranchSafetyChecker) GetCurrentBranch`: Reads current git branch via `git branch --show-current`.
  - `(*BranchSafetyChecker) IsProtectedBranch`: Case-insensitive membership check.
  - `(*BranchSafetyChecker) CheckBranchSafety`: Core guard that enforces `--branch-check`, handles warnings, and interactive confirmation unless forced.
  - `(*BranchSafetyChecker) promptUserForConfirmation`: Renders warning banner and reads `yes` confirmation.
  - `(*BranchSafetyChecker) FormatBranchWarning`: Formats status line for CLI output.

### health.go
- **Purpose**: Implements the `pgsquash health` command and CLI branding/version helpers.
- **Key Types**
  - `HealthStatus`: Minimal JSON payload for orchestration probes.
  - `DetailedHealthStatus`: Extended payload including runtime and Docker diagnostics.
- **Top-Level Variables**
  - `healthCmd`, option flags (`healthText`, `healthDetailed`), `versionInfo`.
- **Functions**
  - `init`: Registers health flags and attaches command to `rootCmd`.
  - `SetVersionInfo`: Updates runtime version metadata injected via ldflags.
  - `SetBrandName`: Rebrands CLI copy/examples for alternate binaries (e.g., capysquash).
  - `getVersion`: Resolves effective version (env override → ldflag fallback).

### root.go
- **Purpose**: Central command wiring, global flags, and handlers for all core CLI actions and workflows.
- **Top-Level Commands**
  - `rootCmd`, `analyzeCmd`, `squashCmd`, `validateCmd`, `initConfigCmd`, `aiTestCmd`, `aiDemoCmd`, `aiFixCmd`, `safeCmd`, `fastCmd`, `analyzeDeepCmd`.
  - TUI toggle integration (`--tui` flags on analyze/squash) and explain/dry-run toggles that imply safe defaults.
- **Functions**
  - Initialization & execution:
    - `init`: Defines persistent/command-specific flags and attaches commands.
    - `Execute`: Configures `PersistentPreRun` for quiet/no-emoji handling, logging verbosity, and emoji suppression before executing root command.
  - Primary command handlers:
    - `runAnalyze`: Loads config, optionally launches TUI analyzer, streams large datasets with memory-optimized tracker, and prints redundancy & optimization summaries.
    - `runSquash`: Handles squashing pipeline (streaming vs standard engine selection, explain/dry-run semantics, provenance output, auto-validation, branch safety checks).
    - `runValidate`: Runs Docker-backed schema comparisons (two containers/databases/schema diff), handles Supabase/Clerk auth compatibility, and writes validation reports with optional editor launch.
    - `runInitConfig`: Generates or overwrites `pgsquash.config.json`, surfacing warnings when file exists unless `--force`.
    - `runAITest`: Invokes AI integration smoke test and prints provider health/capabilities.
    - `runAIDemo`: Runs AI capability showcase (function equivalence, dead code, auth patterns) for demonstration.
    - `runAIFix`: Orchestrates AI-driven migration fixing loop with optional auto-apply, Docker validation between attempts, and verbose logging.
  - Workflow entry points:
    - `runSafeWorkflow`: Applies conservative settings, backups/rollback, and executes AI-assisted SAFE flow.
    - `runFastWorkflow`: Tunes for dev speed (streaming, schema-diff validation, transformations) using AI optimizations.
    - `runAnalyzeWorkflow`: Runs deep analysis workflow with AI augmentations.
  - Workflow helpers:
    - `executeSquashWithAIValidation`: SAFE-oriented squashing with AI auth/pattern checks and Docker validation.
    - `executeSquashWithAIOptimization`: FAST-oriented squashing with AI complexity/optimization feedback.
    - `executeAIComprehensiveAnalysis`: AI-backed reporting for ANALYZE workflow.
    - `executeSquashWithValidation`: Common helper for Docker validation without AI extras.
    - `executeComprehensiveAnalysis`: Non-AI analysis pipeline used as fallback/utility.
  - Shared utilities:
    - `runValidationCheck`: Internal helper powering automatic validation after squashing, returning structured validation results.
    - `openInEditor`: Opens generated reports in `$EDITOR` (respects user environment).
    - `extractFunctionsFromSQL` / `extractFunctionName`: Parse function names from SQL text for AI analysis payloads.
    - `min`: Simple two-value minimum helper.
    - `loadSingleMigration` / `loadMigrations`: Read SQL files into `MigrationWithContent` structures (with progress hooks and error wrapping).
    - `printAnalysisReport`: Formats tracker results, redundancies, auth pattern findings, and warnings for CLI output (with emoji gating).
    - `printSquashSummary`: Summarizes squashing metrics (input count, output size, duration, warnings, output path) and integrates branch safety messaging.
    - `createBackupConfig`: Builds transformation backup configuration used by squasher engine (backup location, retention).
    - `createTransformationConfig`: Builds SQL transformation configuration (modernization flags, AI transformation toggles).

### tui.go
- **Purpose**: Adds top-level `tui` command family and delegates to public TUI API for various entry points.
- **Commands**
  - `tuiCmd`: Launches main interface.
  - `tuiAnalyzeCmd`, `tuiConfigCmd`, `tuiDepGraphCmd`: Direct entry points to specific views.
- **Functions**
  - `init`: Registers TUI commands with subcommands.
  - `runTUI`: Validates migration directory and launches default TUI.
  - `runTUIAnalyze`: Launches TUI focused on analysis view.
  - `runTUIConfig`: Launches configuration wizard view.
  - `runTUIDepGraph`: Launches dependency graph visualization.

## Subdirectories
- _None._
