package consolidation

import (
	"github.com/CAPYSQUASH/pgsquash-engine/internal/utils"
	"fmt"
	"strings"

	"github.com/CAPYSQUASH/pgsquash-engine/internal/tracking"
	"github.com/CAPYSQUASH/pgsquash-engine/internal/types"

	"github.com/CAPYSQUASH/pgsquash-engine/internal/errors"
)

// MultipleCreateConsolidationRule handles multiple CREATE statements for the same object
type MultipleCreateConsolidationRule struct{}

// CanApply checks if the rule can be applied to the given lifecycle
func (r *MultipleCreateConsolidationRule) CanApply(lifecycle *tracking.ObjectLifecycle) bool {
	createCount := 0
	for _, event := range lifecycle.History {
		if event.Operation == types.OpCreate {
			createCount++
		}
	}

	// Debug logging for profiles
	if strings.ToLower(lifecycle.Name) == "profiles" {
		utils.GetDefaultLogger().WithPrefix("MULTIPLE-CREATE").Info("DEBUG MultipleCreateConsolidationRule.CanApply: profiles (type=%s) has %d events total, %d CREATE operations", lifecycle.Type, len(lifecycle.History), createCount)
		for i, event := range lifecycle.History {
			utils.GetDefaultLogger().WithPrefix("MULTIPLE-CREATE").Info("  Event %d: Operation=%s (%v)", i, event.Operation, event.Operation)
		}
		utils.GetDefaultLogger().WithPrefix("MULTIPLE-CREATE").Info("  Returning %v (need createCount > 1)", createCount > 1)
	}

	return createCount > 1
}

// Apply applies the consolidation rule to the given lifecycle
func (r *MultipleCreateConsolidationRule) Apply(lifecycle *tracking.ObjectLifecycle, engine ConsolidationEngine) (*tracking.ConsolidationResult, error) {
	if !r.CanApply(lifecycle) {
		return nil, errors.New(errors.ErrorCodeConsolidationFailed, errors.CategoryConsolidation, "rule cannot be applied to lifecycle", map[string]interface{}{"rule": "MultipleCreateConsolidationRule"})
	}

	// Collect all CREATE statements
	var allCreateStmts []types.Statement
	for _, event := range lifecycle.History {
		if event.Operation == types.OpCreate {
			allCreateStmts = append(allCreateStmts, event.Statement)
		}
	}

	// Safety check: if no CREATE statements found, this shouldn't happen due to CanApply check
	if len(allCreateStmts) == 0 {
		return nil, errors.New(errors.ErrorCodeConsolidationFailed, errors.CategoryConsolidation, "no CREATE statements found in lifecycle", map[string]interface{}{"object": lifecycle.Name})
	}

	// Debug logging for profiles and viewing_requests
	tableName := strings.ToLower(lifecycle.Name)
	if tableName == "profiles" || tableName == "viewing_requests" {
		utils.GetDefaultLogger().WithPrefix("MULTIPLE-CREATE").Info("DEBUG MultipleCreateConsolidationRule.Apply: %s has %d CREATE statements", lifecycle.Name, len(allCreateStmts))
		for i, stmt := range allCreateStmts {
			if tableName == "profiles" {
				hasCity := strings.Contains(strings.ToLower(stmt.SQL), "city")
				hasAuthProvider := strings.Contains(strings.ToLower(stmt.SQL), "auth_provider")
				columnCount := strings.Count(stmt.SQL, ",")
				utils.GetDefaultLogger().WithPrefix("MULTIPLE-CREATE").Info("  CREATE %d: has 'city'=%v, 'auth_provider'=%v, columns~%d, SQL length=%d", i, hasCity, hasAuthProvider, columnCount, len(stmt.SQL))
			} else {
				utils.GetDefaultLogger().WithPrefix("MULTIPLE-CREATE").Info("  CREATE %d: SQL length=%d, first 150 chars: %.150s", i, len(stmt.SQL), stmt.SQL)
			}
		}
	}

	// Merge all CREATE statements by treating subsequent CREATEs as if they were ALTERs
	// This handles the case where migrations have duplicate CREATE TABLE IF NOT EXISTS with different column sets
	// Use the LAST CREATE statement as base (for DDL cycles: CREATE→DROP→CREATE, we want the final version)
	baseCreateStmt := allCreateStmts[len(allCreateStmts)-1]
	consolidatedSQL := baseCreateStmt.SQL

	// For tables with more than one CREATE, merge columns from all CREATE statements
	if len(allCreateStmts) > 1 && lifecycle.Type == types.TypeTable {
		consolidatedSQL = mergeMultipleCreateStatements(allCreateStmts, lifecycle.Name)
		if strings.ToLower(lifecycle.Name) == "profiles" {
			utils.GetDefaultLogger().WithPrefix("MULTIPLE-CREATE").Info("  Merged %d CREATE statements into unified schema", len(allCreateStmts))
		}
	}

	result := &tracking.ConsolidationResult{
		OriginalStatements: allCreateStmts,
		ConsolidatedSQL:    consolidatedSQL,
		Optimizations: []string{
			fmt.Sprintf("Consolidated %d CREATE statements to final definition", len(allCreateStmts)),
		},
		RiskLevel: tracking.RiskLevelLow,
		EstimatedSavings: tracking.SquashSavings{
			StatementsReduced: len(allCreateStmts) - 1,
			FilesAffected:     len(allCreateStmts),
			LinesReduced:      (len(allCreateStmts) - 1) * 5, // Estimate
		},
	}

	return result, nil
}

// Risk returns the risk level for this rule
func (r *MultipleCreateConsolidationRule) Risk() tracking.RiskLevel {
	return tracking.RiskLevelLow
}

// Note: mergeMultipleCreateStatements is defined in drop_create_rule.go
// and shared across multiple rules
