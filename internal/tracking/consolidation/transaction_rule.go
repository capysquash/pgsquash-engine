package consolidation

import (
	"fmt"
	"strings"

	"github.com/capysquash/pg-squash-engine/internal/errors"
	"github.com/capysquash/pg-squash-engine/internal/tracking"
	"github.com/capysquash/pg-squash-engine/internal/types"
)

// TransactionBoundaryRule optimizes transaction boundaries for better performance
type TransactionBoundaryRule struct{}

// CanApply checks if the rule can be applied to the given lifecycle
func (r *TransactionBoundaryRule) CanApply(lifecycle *tracking.ObjectLifecycle) bool {
	// Apply to lifecycles with multiple operations that can be grouped
	return len(lifecycle.History) > 2
}

// Apply applies the consolidation rule to the given lifecycle
func (r *TransactionBoundaryRule) Apply(lifecycle *tracking.ObjectLifecycle, engine ConsolidationEngine) (*tracking.ConsolidationResult, error) {
	if !r.CanApply(lifecycle) {
		return nil, errors.New(errors.ErrorCodeConsolidationFailed, errors.CategoryConsolidation, "rule cannot be applied to lifecycle", map[string]interface{}{"rule": "TransactionRule"})
	}

	// Group operations by transaction boundaries
	transactionGroups := r.groupByTransactionBoundaries(lifecycle)

	// Generate optimized transaction blocks
	consolidatedSQL := r.generateOptimizedTransactions(transactionGroups)

	// Collect all statements for consolidation
	var originalStmts []types.Statement
	for _, event := range lifecycle.History {
		originalStmts = append(originalStmts, event.Statement)
	}

	result := &tracking.ConsolidationResult{
		OriginalStatements: originalStmts,
		ConsolidatedSQL:    consolidatedSQL,
		Optimizations: []string{
			fmt.Sprintf("Grouped %d operations into %d optimized transactions",
				len(originalStmts), len(transactionGroups)),
			"Optimized transaction boundaries for better performance",
		},
		RiskLevel: tracking.RiskLevelLow, // Transaction optimization is generally safe
		EstimatedSavings: tracking.SquashSavings{
			StatementsReduced: 0, // No statements removed, just reorganized
			FilesAffected:     len(originalStmts),
			LinesReduced:      len(transactionGroups) * 2, // Transaction overhead reduction
		},
	}

	return result, nil
}

// Risk returns the risk level for this rule
func (r *TransactionBoundaryRule) Risk() tracking.RiskLevel {
	return tracking.RiskLevelLow
}

// TransactionGroup represents a group of operations that can be executed together
type TransactionGroup struct {
	Operations         []tracking.LifecycleEvent
	RequiresSeparateTx bool
	Priority           int
}

func (r *TransactionBoundaryRule) groupByTransactionBoundaries(lifecycle *tracking.ObjectLifecycle) []TransactionGroup {
	var groups []TransactionGroup
	currentGroup := TransactionGroup{Priority: 1}

	for _, event := range lifecycle.History {
		// Check if this operation requires a separate transaction
		requiresSeparate := r.requiresSeparateTransaction(event)

		if requiresSeparate && len(currentGroup.Operations) > 0 {
			// Finish current group and start new one
			groups = append(groups, currentGroup)
			currentGroup = TransactionGroup{
				Operations:         []tracking.LifecycleEvent{event},
				RequiresSeparateTx: true,
				Priority:           r.getOperationPriority(event),
			}
		} else {
			// Add to current group
			currentGroup.Operations = append(currentGroup.Operations, event)
			if requiresSeparate {
				currentGroup.RequiresSeparateTx = true
			}
			// Update priority to highest in group
			if priority := r.getOperationPriority(event); priority > currentGroup.Priority {
				currentGroup.Priority = priority
			}
		}
	}

	// Add final group
	if len(currentGroup.Operations) > 0 {
		groups = append(groups, currentGroup)
	}

	return groups
}

func (r *TransactionBoundaryRule) requiresSeparateTransaction(event tracking.LifecycleEvent) bool {
	sql := strings.ToUpper(event.Statement.SQL)

	// Operations that typically require separate transactions
	separateOps := []string{
		"CREATE INDEX CONCURRENTLY",
		"DROP INDEX CONCURRENTLY",
		"REINDEX",
		"VACUUM",
		"ANALYZE",
		"ALTER TYPE", // Some ALTER TYPE operations
	}

	for _, op := range separateOps {
		if strings.Contains(sql, op) {
			return true
		}
	}

	// Data operations (INSERT/UPDATE/DELETE) after schema changes
	if event.HasDataOps && event.Operation != types.OpCreate {
		return true
	}

	return false
}

func (r *TransactionBoundaryRule) getOperationPriority(event tracking.LifecycleEvent) int {
	// Higher priority = should be executed earlier
	switch event.Operation {
	case types.OpCreate:
		return 100
	case types.OpAlter:
		return 80
	case types.OpDrop:
		return 60
	default:
		return 50
	}
}

func (r *TransactionBoundaryRule) generateOptimizedTransactions(groups []TransactionGroup) string {
	var sqlParts []string

	// Sort groups by priority
	for i := 0; i < len(groups)-1; i++ {
		for j := i + 1; j < len(groups); j++ {
			if groups[i].Priority < groups[j].Priority {
				groups[i], groups[j] = groups[j], groups[i]
			}
		}
	}

	for i, group := range groups {
		if len(group.Operations) == 1 && group.RequiresSeparateTx {
			// Single operation that requires separate transaction
			sqlParts = append(sqlParts,
				fmt.Sprintf("-- Operation %d: Requires separate transaction", i+1))
			sql := group.Operations[0].Statement.SQL
			if !strings.HasSuffix(strings.TrimSpace(sql), ";") {
				sql += ";"
			}
			sqlParts = append(sqlParts, sql)
		} else if len(group.Operations) > 1 {
			// Multiple operations in transaction block
			sqlParts = append(sqlParts,
				fmt.Sprintf("-- Transaction %d: %d operations", i+1, len(group.Operations)))
			sqlParts = append(sqlParts, "BEGIN;")

			for _, op := range group.Operations {
				sql := op.Statement.SQL
				if !strings.HasSuffix(strings.TrimSpace(sql), ";") {
					sql += ";"
				}
				sqlParts = append(sqlParts, "  "+sql)
			}

			sqlParts = append(sqlParts, "COMMIT;")
		} else {
			// Single operation, no transaction needed
			sql := group.Operations[0].Statement.SQL
			if !strings.HasSuffix(strings.TrimSpace(sql), ";") {
				sql += ";"
			}
			sqlParts = append(sqlParts, sql)
		}

		sqlParts = append(sqlParts, "")
	}

	return strings.Join(sqlParts, "\n")
}
