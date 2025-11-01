# internal/tracking/recovery package map

## Domain Summary
- Provides fine-grained error recovery strategies that the consolidation engine can invoke when rules fail—allowing partial rollbacks, retries, or conservative fallbacks instead of aborting the run.

## Files (alphabetical)

### fine_grained_error_recovery.go
- **Purpose**: Implements recovery orchestrator with multiple strategies (conservative, aggressive, isolate, retry) and targeted fixers for parsing, dependency, constraint, and type errors.
- **Functions / Methods**
  - Lifecycle: `NewFineGrainedErrorRecovery`.
  - Entry points: `RecoverFromError`, `RecoverFromMultipleErrors`.
  - Strategy executor: `attemptRecovery`.
  - Strategy handlers: `recoverConservative`, `recoverAggressive`, `recoverFallback`, `recoverIsolate`, `recoverRetry`.
  - Fixers: `fixParsingError`, `fixDependencyError`, `fixConstraintError`, `fixTypeMismatch`, `applyGenericFix`.
  - Analysis & utilities: `analyzeError`, `tryRetryApproach`, `applySimpleConsolidation`, `logVerbose`, `maxErrorInt`.

## Subdirectories
- _None._
