# internal/transaction package map

## Domain Summary
- Provides transaction planning and lock analysis for migration squashing.
- Groups statements into optimized transaction batches based on metadata (locks, concurrent operations, execution time).
- Single source of truth for transaction boundary logic used across consolidation and execution phases.

## Files (alphabetical)

### planner.go
- **Purpose**: Plans transaction boundaries, analyzes lock conflicts, and generates execution plans for migrations.
- **Key Types**
  - `TransactionPlanner`: Main planner orchestrator.
  - `TransactionPlan`: Complete execution plan with batches, conflicts, and warnings.
  - `TransactionBatch`: Group of statements that can run in a single transaction.
  - `LockConflict`: Potential lock conflict between statements.
- **Functions / Methods**
  - Lifecycle: `NewTransactionPlanner`.
  - Planning: `(*TransactionPlanner) PlanTransactions` - Groups statements by transaction boundaries using parser metadata.
  - Analysis: `analyzeLockConflicts`, `detectConflict`, `generateWarnings`.
  - Formatting: `FormatPlan`, `FormatLockAnalysis`.
  - Utilities: `countConcurrent`, `countNoTxn`, `countPreserved`.

## Subdirectories
- _None._
