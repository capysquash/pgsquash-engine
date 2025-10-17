package consolidation

import (
	"github.com/capysquash/pg-squash-engine/internal/utils"
	"fmt"
	"regexp"
	"strings"

	"github.com/capysquash/pg-squash-engine/internal/tracking"
	"github.com/capysquash/pg-squash-engine/internal/types"

	"github.com/capysquash/pg-squash-engine/internal/errors"
)

// DropCreateCycleRule handles DROP followed by CREATE consolidation
// This includes tables and views (DROP VIEW + CREATE VIEW → CREATE OR REPLACE VIEW)
type DropCreateCycleRule struct{}

// CanApply checks if the rule can be applied to the given lifecycle
func (r *DropCreateCycleRule) CanApply(lifecycle *tracking.ObjectLifecycle) bool {
	if len(lifecycle.History) < 2 {
		return false
	}

	// Look for DROP followed by CREATE pattern
	for i := 0; i < len(lifecycle.History)-1; i++ {
		if lifecycle.History[i].Operation == types.OpDrop &&
			lifecycle.History[i+1].Operation == types.OpCreate {
			return true
		}
	}

	// Also check for DROP VIEW pattern (might not have subsequent CREATE in same lifecycle)
	if lifecycle.Type == types.TypeView {
		for _, event := range lifecycle.History {
			if event.Operation == types.OpDrop {
				return true
			}
		}
	}

	return false
}

// Apply applies the consolidation rule to the given lifecycle
func (r *DropCreateCycleRule) Apply(lifecycle *tracking.ObjectLifecycle, engine ConsolidationEngine) (*tracking.ConsolidationResult, error) {
	if !r.CanApply(lifecycle) {
		return nil, errors.New(errors.ErrorCodeConsolidationFailed, errors.CategoryConsolidation, "rule cannot be applied to lifecycle", map[string]interface{}{"rule": "DropCreateCycleRule"})
	}

	// Find the DROP-CREATE pair
	var dropStmt *types.Statement
	var originalStmts []types.Statement

	for i := 0; i < len(lifecycle.History)-1; i++ {
		if lifecycle.History[i].Operation == types.OpDrop &&
			lifecycle.History[i+1].Operation == types.OpCreate {
			dropStmt = &lifecycle.History[i].Statement
			originalStmts = append(originalStmts, *dropStmt)
			break
		}
	}

	// Collect ALL CREATE statements (not just the one after DROP)
	var allCreateStmts []types.Statement
	for _, event := range lifecycle.History {
		if event.Operation == types.OpCreate {
			allCreateStmts = append(allCreateStmts, event.Statement)
			if dropStmt == nil { // If no DROP found yet, add CREATE to originalStmts
				originalStmts = append(originalStmts, event.Statement)
			}
		}
	}

	// Merge multiple CREATE statements if needed
	var consolidatedSQL string
	optimizations := []string{}

	if lifecycle.Type == types.TypeView {
		// For VIEWs: Convert DROP VIEW + CREATE VIEW to CREATE OR REPLACE VIEW
		if len(allCreateStmts) == 1 {
			createViewSQL := allCreateStmts[0].SQL

			// Replace "CREATE VIEW" with "CREATE OR REPLACE VIEW"
			if strings.Contains(strings.ToUpper(createViewSQL), "CREATE VIEW") {
				consolidatedSQL = regexp.MustCompile(`(?i)CREATE\s+VIEW`).ReplaceAllString(createViewSQL, "CREATE OR REPLACE VIEW")
				optimizations = append(optimizations, "Converted DROP VIEW + CREATE VIEW to CREATE OR REPLACE VIEW")
			} else {
				consolidatedSQL = createViewSQL
				optimizations = append(optimizations, "Eliminated DROP operation before CREATE VIEW")
			}
		} else if len(allCreateStmts) > 1 {
			// Multiple CREATE VIEW statements - use the last one with CREATE OR REPLACE
			lastCreate := allCreateStmts[len(allCreateStmts)-1].SQL
			consolidatedSQL = regexp.MustCompile(`(?i)CREATE\s+VIEW`).ReplaceAllString(lastCreate, "CREATE OR REPLACE VIEW")
			optimizations = append(optimizations, fmt.Sprintf("Merged %d CREATE VIEW statements into CREATE OR REPLACE VIEW", len(allCreateStmts)))
		} else {
			return nil, errors.New(errors.ErrorCodeConsolidationFailed, errors.CategoryConsolidation, "no CREATE statement found for VIEW", map[string]interface{}{"object": lifecycle.Name})
		}

		utils.GetDefaultLogger().WithPrefix("DROP-CREATE").Info("DropCreateCycleRule: Converted VIEW %s to CREATE OR REPLACE pattern", lifecycle.Name)
	} else if len(allCreateStmts) > 1 && lifecycle.Type == types.TypeTable {
		// For TABLES: Merge multiple CREATE statements
		consolidatedSQL = mergeMultipleCreateStatements(allCreateStmts, lifecycle.Name)
		optimizations = append(optimizations, fmt.Sprintf("Merged %d CREATE statements for table %s", len(allCreateStmts), lifecycle.Name))
		utils.GetDefaultLogger().WithPrefix("DROP-CREATE").Info("DropCreateCycleRule: Merged %d CREATE statements for %s", len(allCreateStmts), lifecycle.Name)
	} else if len(allCreateStmts) == 1 {
		consolidatedSQL = allCreateStmts[0].SQL
		optimizations = append(optimizations, "Eliminated DROP operation before CREATE")
	} else {
		return nil, errors.New(errors.ErrorCodeConsolidationFailed, errors.CategoryConsolidation, "no CREATE statement found", map[string]interface{}{"object": lifecycle.Name})
	}

	result := &tracking.ConsolidationResult{
		OriginalStatements: originalStmts,
		ConsolidatedSQL:    consolidatedSQL,
		Optimizations:      optimizations,
		RiskLevel:          tracking.RiskLevelMedium, // Medium risk due to data implications
		EstimatedSavings: tracking.SquashSavings{
			StatementsReduced: len(originalStmts),
			FilesAffected:     len(originalStmts) + len(allCreateStmts),
			LinesReduced:      len(originalStmts) * 3,
		},
	}

	return result, nil
}

// Risk returns the risk level for this rule
func (r *DropCreateCycleRule) Risk() tracking.RiskLevel {
	return tracking.RiskLevelMedium
}

// Helper functions for DROP CREATE consolidation

// mergeMultipleCreateStatements merges multiple CREATE statements for the same table
func mergeMultipleCreateStatements(createStmts []types.Statement, tableName string) string {
	if len(createStmts) == 0 {
		return ""
	}

	// Use the LAST CREATE as the base (for DDL cycles: CREATE→DROP→CREATE, we want the final version)
	baseSQL := createStmts[len(createStmts)-1].SQL

	// Extract all unique columns from all CREATE statements
	allColumns := make(map[string]string) // column name -> full definition
	columnOrder := make([]string, 0)       // maintain order

	for i, stmt := range createStmts {
		columns := extractColumnsFromCreate(stmt.SQL)
		utils.GetDefaultLogger().WithPrefix("DROP-CREATE").Info("  Extracted %d columns from CREATE statement %d", len(columns), i)

		for colName, colDef := range columns {
			// Always update column definition (later statements override earlier ones)
			// This ensures the LAST CREATE statement's schema is preserved in DDL cycles
			if _, exists := allColumns[colName]; !exists {
				columnOrder = append(columnOrder, colName)
			}
			allColumns[colName] = colDef
		}
	}

	utils.GetDefaultLogger().WithPrefix("DROP-CREATE").Info("  Total unique columns across all CREATEs: %d", len(allColumns))

	// Extract table-level constraints from the LAST CREATE statement (this has the final schema)
	tableConstraints := extractTableConstraintsFromCreate(baseSQL)

	// Rebuild CREATE statement with all columns and constraints
	return reconstructCreateWithColumns(baseSQL, allColumns, columnOrder, tableConstraints, tableName)
}

// extractTableConstraintsFromCreate extracts table-level constraints (not column definitions)
func extractTableConstraintsFromCreate(createSQL string) []string {
	var constraints []string

	firstParen := strings.Index(createSQL, "(")
	lastParen := strings.LastIndex(createSQL, ")")

	if firstParen == -1 || lastParen == -1 || firstParen >= lastParen {
		return constraints
	}

	columnList := createSQL[firstParen+1 : lastParen]
	parts := splitColumnDefinitions(columnList)

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}

		// Skip JSON keys
		if strings.HasPrefix(trimmed, "\"") || strings.HasPrefix(trimmed, "'") {
			continue
		}

		upper := strings.ToUpper(trimmed)

		// Only include table-level constraints
		if strings.HasPrefix(upper, "CONSTRAINT ") ||
			strings.HasPrefix(upper, "PRIMARY KEY (") ||
			strings.HasPrefix(upper, "FOREIGN KEY (") ||
			strings.HasPrefix(upper, "UNIQUE (") {
			constraints = append(constraints, trimmed)
		}
	}

	return constraints
}

// extractColumnsFromCreate extracts column definitions from a CREATE TABLE statement
func extractColumnsFromCreate(createSQL string) map[string]string {
	columns := make(map[string]string)

	// Find the column list between first ( and last )
	firstParen := strings.Index(createSQL, "(")
	lastParen := strings.LastIndex(createSQL, ")")

	if firstParen == -1 || lastParen == -1 || firstParen >= lastParen {
		return columns
	}

	columnList := createSQL[firstParen+1 : lastParen]

	// Split by commas (being careful with nested parentheses in constraints)
	parts := splitColumnDefinitions(columnList)

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}

		// Skip JSON keys (they start with quotes)
		if strings.HasPrefix(trimmed, "\"") || strings.HasPrefix(trimmed, "'") {
			continue
		}

		// Skip standalone comments (lines that start with --)
		if strings.HasPrefix(trimmed, "--") {
			continue
		}

		// Skip table-level constraints (CONSTRAINT, PRIMARY KEY, FOREIGN KEY, CHECK, UNIQUE without column name)
		upper := strings.ToUpper(trimmed)
		if strings.HasPrefix(upper, "CONSTRAINT ") ||
			strings.HasPrefix(upper, "PRIMARY KEY") ||
			strings.HasPrefix(upper, "FOREIGN KEY") ||
			(strings.HasPrefix(upper, "CHECK") && !strings.Contains(trimmed, " ")) ||
			(strings.HasPrefix(upper, "UNIQUE") && strings.HasPrefix(upper, "UNIQUE (")) {
			continue
		}

		// Extract column name (first word, remove quotes if present)
		words := strings.Fields(trimmed)
		if len(words) > 0 {
			colName := strings.ToLower(strings.Trim(words[0], "\"'"))
			columns[colName] = trimmed
		}
	}

	return columns
}

// splitColumnDefinitions splits column definitions by comma, respecting nested parentheses and curly braces (for JSON/JSONB)
func splitColumnDefinitions(columnList string) []string {
	var parts []string
	var current strings.Builder
	parenDepth := 0
	braceDepth := 0
	inString := false
	var stringDelim rune

	for i, char := range columnList {
		// Handle string literals (both single and double quotes)
		if (char == '\'' || char == '"') && (i == 0 || columnList[i-1] != '\\') {
			if !inString {
				inString = true
				stringDelim = char
			} else if char == stringDelim {
				inString = false
			}
			current.WriteRune(char)
			continue
		}

		// If we're in a string, don't process special characters
		if inString {
			current.WriteRune(char)
			continue
		}

		switch char {
		case '(':
			parenDepth++
			current.WriteRune(char)
		case ')':
			parenDepth--
			current.WriteRune(char)
		case '{':
			braceDepth++
			current.WriteRune(char)
		case '}':
			braceDepth--
			current.WriteRune(char)
		case ',':
			// Only split on commas at depth 0 for both parens and braces
			if parenDepth == 0 && braceDepth == 0 {
				parts = append(parts, current.String())
				current.Reset()
			} else {
				current.WriteRune(char)
			}
		default:
			current.WriteRune(char)
		}
	}

	// Add the last part
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}

// reconstructCreateWithColumns rebuilds a CREATE TABLE statement with merged columns
func reconstructCreateWithColumns(baseSQL string, columns map[string]string, columnOrder []string, tableConstraints []string, tableName string) string {
	// Extract the CREATE TABLE ... ( part
	firstParen := strings.Index(baseSQL, "(")
	if firstParen == -1 {
		return baseSQL
	}

	header := baseSQL[:firstParen+1]

	// Rebuild column list
	var columnDefs []string
	for _, colName := range columnOrder {
		if colDef, exists := columns[colName]; exists {
			columnDefs = append(columnDefs, "  "+colDef)
		}
	}

	// Add table-level constraints
	for _, constraint := range tableConstraints {
		columnDefs = append(columnDefs, "  "+constraint)
	}

	// Find everything after the last ) (like table options, semicolon, etc.)
	lastParen := strings.LastIndex(baseSQL, ")")
	suffix := ""
	if lastParen != -1 && lastParen+1 < len(baseSQL) {
		suffix = baseSQL[lastParen:]
	} else {
		suffix = "\n);"
	}

	// Reconstruct
	result := header + "\n"
	result += strings.Join(columnDefs, ",\n")
	result += "\n" + suffix

	return result
}
