package consolidation

import (
	"fmt"
	"strings"

	"github.com/CAPYSQUASH/pgsquash-engine/internal/tracking"
	"github.com/CAPYSQUASH/pgsquash-engine/internal/types"

	"github.com/CAPYSQUASH/pgsquash-engine/internal/errors"
)

// ColumnEvolutionRule handles complex column lifecycle patterns
type ColumnEvolutionRule struct{}

// CanApply checks if the rule can be applied to the given lifecycle
func (r *ColumnEvolutionRule) CanApply(lifecycle *tracking.ObjectLifecycle) bool {
	if lifecycle.Type != types.TypeTable {
		return false
	}

	// Check if there are column-related ALTER operations
	hasColumnOps := false
	for _, event := range lifecycle.History {
		if event.Operation == types.OpAlter {
			alterSQL := strings.ToUpper(event.Statement.SQL)
			if strings.Contains(alterSQL, "ADD COLUMN") ||
				strings.Contains(alterSQL, "DROP COLUMN") ||
				strings.Contains(alterSQL, "ALTER COLUMN") ||
				strings.Contains(alterSQL, "RENAME COLUMN") {
				hasColumnOps = true
				break
			}
		}
	}

	return hasColumnOps
}

// Apply applies the consolidation rule to the given lifecycle
func (r *ColumnEvolutionRule) Apply(lifecycle *tracking.ObjectLifecycle, engine ConsolidationEngine) (*tracking.ConsolidationResult, error) {
	if !r.CanApply(lifecycle) {
		return nil, errors.New(errors.ErrorCodeConsolidationFailed, errors.CategoryConsolidation, "rule cannot be applied to lifecycle", map[string]interface{}{"rule": "ColumnEvolutionRule"})
	}

	// Track column evolution through lifecycle
	columnChanges := r.analyzeColumnEvolution(lifecycle)

	// Generate final schema with all column changes applied
	finalSQL := r.generateFinalSchema(lifecycle, columnChanges)

	// Find all column-related statements for consolidation
	var originalStmts []types.Statement
	for _, event := range lifecycle.History {
		if event.Operation == types.OpCreate ||
			(event.Operation == types.OpAlter && r.isColumnOperation(event.Statement.SQL)) {
			originalStmts = append(originalStmts, event.Statement)
		}
	}

	result := &tracking.ConsolidationResult{
		OriginalStatements: originalStmts,
		ConsolidatedSQL:    finalSQL,
		Optimizations: []string{
			fmt.Sprintf("Consolidated %d column operations into final schema", len(columnChanges)),
		},
		RiskLevel: tracking.RiskLevelMedium, // Medium risk due to schema changes
		EstimatedSavings: tracking.SquashSavings{
			StatementsReduced: len(originalStmts) - 1,
			FilesAffected:     len(originalStmts),
			LinesReduced:      len(originalStmts) * 3,
		},
	}

	return result, nil
}

// Risk returns the risk level for this rule
func (r *ColumnEvolutionRule) Risk() tracking.RiskLevel {
	return tracking.RiskLevelMedium
}

// ColumnChange tracks a column modification operation
type ColumnChange struct {
	Name      string
	Operation string // ADD, DROP, ALTER, RENAME
	DataType  string
	Default   string
	NotNull   bool
	Order     int
}

func (r *ColumnEvolutionRule) analyzeColumnEvolution(lifecycle *tracking.ObjectLifecycle) []ColumnChange {
	var changes []ColumnChange
	order := 0

	for _, event := range lifecycle.History {
		if event.Operation == types.OpAlter {
			alterSQL := event.Statement.SQL

			// Extract column operations from ALTER statements
			if change := r.extractColumnChange(alterSQL, order); change != nil {
				changes = append(changes, *change)
				order++
			}
		}
	}

	return changes
}

func (r *ColumnEvolutionRule) extractColumnChange(alterSQL string, order int) *ColumnChange {
	upperSQL := strings.ToUpper(alterSQL)

	if strings.Contains(upperSQL, "ADD COLUMN") {
		// Extract ADD COLUMN operation
		return r.parseAddColumn(alterSQL, order)
	} else if strings.Contains(upperSQL, "DROP COLUMN") {
		// Extract DROP COLUMN operation
		return r.parseDropColumn(alterSQL, order)
	} else if strings.Contains(upperSQL, "ALTER COLUMN") {
		// Extract ALTER COLUMN operation
		return r.parseAlterColumn(alterSQL, order)
	}

	return nil
}

func (r *ColumnEvolutionRule) parseAddColumn(alterSQL string, order int) *ColumnChange {
	// Simple extraction for ADD COLUMN
	lines := strings.Split(alterSQL, "\n")
	for _, line := range lines {
		upperLine := strings.ToUpper(strings.TrimSpace(line))
		if strings.Contains(upperLine, "ADD COLUMN") {
			// Extract column name and definition
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				return &ColumnChange{
					Name:      parts[3], // Column name after "ADD COLUMN"
					Operation: "ADD",
					Order:     order,
				}
			}
		}
	}
	return nil
}

func (r *ColumnEvolutionRule) parseDropColumn(alterSQL string, order int) *ColumnChange {
	// Simple extraction for DROP COLUMN
	lines := strings.Split(alterSQL, "\n")
	for _, line := range lines {
		upperLine := strings.ToUpper(strings.TrimSpace(line))
		if strings.Contains(upperLine, "DROP COLUMN") {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				return &ColumnChange{
					Name:      parts[3], // Column name after "DROP COLUMN"
					Operation: "DROP",
					Order:     order,
				}
			}
		}
	}
	return nil
}

func (r *ColumnEvolutionRule) parseAlterColumn(alterSQL string, order int) *ColumnChange {
	// Simple extraction for ALTER COLUMN
	lines := strings.Split(alterSQL, "\n")
	for _, line := range lines {
		upperLine := strings.ToUpper(strings.TrimSpace(line))
		if strings.Contains(upperLine, "ALTER COLUMN") {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				return &ColumnChange{
					Name:      parts[3], // Column name after "ALTER COLUMN"
					Operation: "ALTER",
					Order:     order,
				}
			}
		}
	}
	return nil
}

func (r *ColumnEvolutionRule) generateFinalSchema(lifecycle *tracking.ObjectLifecycle, columnChanges []ColumnChange) string {
	// Get the base CREATE statement
	var baseCreateStmt *types.Statement
	for _, event := range lifecycle.History {
		if event.Operation == types.OpCreate {
			baseCreateStmt = &event.Statement
			break
		}
	}

	if baseCreateStmt == nil {
		return ""
	}

	// Collect all ALTER statements that add columns
	var alterStmts []types.Statement
	for _, event := range lifecycle.History {
		if event.Operation == types.OpAlter {
			alterSQL := strings.ToUpper(event.Statement.SQL)
			if strings.Contains(alterSQL, "ADD COLUMN") {
				alterStmts = append(alterStmts, event.Statement)
			}
		}
	}

	// Use the working integrateAlterIntoCreate function from create_alter_rule.go
	// This properly parses and integrates ALTER TABLE ADD COLUMN statements
	finalSQL := integrateAlterIntoCreate(baseCreateStmt, alterStmts)

	// Add a comment about the column changes
	if len(columnChanges) > 0 {
		comment := fmt.Sprintf("-- Column evolution: %d operations consolidated\n", len(columnChanges))
		finalSQL = comment + finalSQL
	}

	return finalSQL
}

func (r *ColumnEvolutionRule) isColumnOperation(sql string) bool {
	upperSQL := strings.ToUpper(sql)
	return strings.Contains(upperSQL, "ADD COLUMN") ||
		strings.Contains(upperSQL, "DROP COLUMN") ||
		strings.Contains(upperSQL, "ALTER COLUMN") ||
		strings.Contains(upperSQL, "RENAME COLUMN")
}
