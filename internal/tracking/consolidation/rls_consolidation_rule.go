package consolidation

import (
	"fmt"
	"strings"

	"github.com/capysquash/pg-squash-engine/internal/errors"
	"github.com/capysquash/pg-squash-engine/internal/tracking"
	"github.com/capysquash/pg-squash-engine/internal/types"
)

// RLSConsolidationRule consolidates Row Level Security operations
type RLSConsolidationRule struct{}

// CanApply checks if the rule can be applied to the given lifecycle
func (r *RLSConsolidationRule) CanApply(lifecycle *tracking.ObjectLifecycle) bool {
	if lifecycle.Type != types.TypeTable {
		return false
	}

	// Check if there are RLS operations
	hasRLS := false
	for _, event := range lifecycle.History {
		if event.Operation == types.OpAlter {
			alterSQL := strings.ToUpper(event.Statement.SQL)
			if strings.Contains(alterSQL, "ENABLE ROW LEVEL SECURITY") ||
				strings.Contains(alterSQL, "DISABLE ROW LEVEL SECURITY") ||
				strings.Contains(alterSQL, "FORCE ROW LEVEL SECURITY") {
				hasRLS = true
				break
			}
		}
	}

	return hasRLS
}

// Apply applies the consolidation rule to the given lifecycle
func (r *RLSConsolidationRule) Apply(lifecycle *tracking.ObjectLifecycle, engine ConsolidationEngine) (*tracking.ConsolidationResult, error) {
	if !r.CanApply(lifecycle) {
		return nil, errors.New(errors.ErrorCodeConsolidationFailed, errors.CategoryConsolidation, "rule cannot be applied to lifecycle", map[string]interface{}{"rule": "RLSConsolidationRule"})
	}

	// Analyze RLS operations in lifecycle
	finalRLSState := r.determineFinalRLSState(lifecycle)

	// Find all RLS-related statements AND the CREATE TABLE statement
	var originalStmts []types.Statement
	var createStmt *types.Statement
	for _, event := range lifecycle.History {
		if event.Operation == types.OpCreate {
			createStmt = &event.Statement
		} else if event.Operation == types.OpAlter && r.isRLSOperation(event.Statement.SQL) {
			originalStmts = append(originalStmts, event.Statement)
		}
	}

	// Generate consolidated SQL: CREATE TABLE + RLS ALTER
	var consolidatedSQL string
	if createStmt != nil {
		consolidatedSQL = createStmt.SQL
		// Add RLS operation if there is one
		if finalRLSState != "" {
			tableName := lifecycle.Name
			rlsStatement := fmt.Sprintf("\nALTER TABLE %s %s;", tableName, finalRLSState)
			consolidatedSQL += rlsStatement
		}
	} else if finalRLSState != "" {
		// Fallback: no CREATE found, just the RLS ALTER
		tableName := lifecycle.Name
		consolidatedSQL = fmt.Sprintf("ALTER TABLE %s %s;", tableName, finalRLSState)
	}

	result := &tracking.ConsolidationResult{
		OriginalStatements: originalStmts,
		ConsolidatedSQL:    consolidatedSQL,
		Optimizations: []string{
			fmt.Sprintf("Consolidated %d RLS operations into final state: %s", len(originalStmts), finalRLSState),
		},
		RiskLevel: tracking.RiskLevelLow, // RLS is usually safe to consolidate
		EstimatedSavings: tracking.SquashSavings{
			StatementsReduced: len(originalStmts) - 1,
			FilesAffected:     len(originalStmts),
			LinesReduced:      len(originalStmts),
		},
	}

	return result, nil
}

// Risk returns the risk level for this rule
func (r *RLSConsolidationRule) Risk() tracking.RiskLevel {
	return tracking.RiskLevelLow
}

func (r *RLSConsolidationRule) determineFinalRLSState(lifecycle *tracking.ObjectLifecycle) string {
	// Track RLS state changes through lifecycle
	currentState := "DISABLE ROW LEVEL SECURITY" // Default state

	for _, event := range lifecycle.History {
		if event.Operation == types.OpAlter {
			alterSQL := strings.ToUpper(event.Statement.SQL)

			if strings.Contains(alterSQL, "ENABLE ROW LEVEL SECURITY") {
				currentState = "ENABLE ROW LEVEL SECURITY"
			} else if strings.Contains(alterSQL, "DISABLE ROW LEVEL SECURITY") {
				currentState = "DISABLE ROW LEVEL SECURITY"
			} else if strings.Contains(alterSQL, "FORCE ROW LEVEL SECURITY") {
				currentState = "FORCE ROW LEVEL SECURITY"
			}
		}
	}

	return currentState
}

func (r *RLSConsolidationRule) isRLSOperation(sql string) bool {
	upperSQL := strings.ToUpper(sql)
	return strings.Contains(upperSQL, "ENABLE ROW LEVEL SECURITY") ||
		strings.Contains(upperSQL, "DISABLE ROW LEVEL SECURITY") ||
		strings.Contains(upperSQL, "FORCE ROW LEVEL SECURITY")
}
