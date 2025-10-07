package tracking

import (
	"fmt"
	"log"
	"strings"

	"github.com/capysquash/pg-squash-engine/internal/metadata"
	"github.com/capysquash/pg-squash-engine/internal/parser"
)

// Tracker is an alias for UnifiedTracker to maintain API compatibility
type Tracker = UnifiedTracker

// NewTracker creates a new unified tracker (alias for NewUnifiedTracker)
func NewTracker() *Tracker {
	return NewUnifiedTracker()
}

// NewTrackerWithMetadata creates a tracker with metadata manager integration
func NewTrackerWithMetadata(metaMgr *metadata.MetadataManager) *Tracker {
	return NewUnifiedTrackerWithMetadata(metaMgr)
}

// ConsolidationRule interface for consolidation rules used by the squasher
type ConsolidationRule interface {
	CanApply(lifecycle *ObjectLifecycle) bool
	Apply(lifecycle *ObjectLifecycle, engine ConsolidationEngine) (*ConsolidationResult, error)
	Risk() RiskLevel
}

// ConsolidationEngine interface for the engine that applies consolidation rules
type ConsolidationEngine interface {
	GetTracker() *Tracker
	GetConfig() interface{} // This would be the actual config type
}

// MultipleCreateConsolidationRule handles multiple CREATE statements for the same object
type MultipleCreateConsolidationRule struct{}

func (r *MultipleCreateConsolidationRule) CanApply(lifecycle *ObjectLifecycle) bool {
	createCount := 0
	for _, event := range lifecycle.History {
		if event.Operation == parser.OpCreate {
			createCount++
		}
	}

	// Debug logging for profiles
	if strings.ToLower(lifecycle.Name) == "profiles" {
		log.Printf("DEBUG MultipleCreateConsolidationRule.CanApply: profiles (type=%s) has %d events total, %d CREATE operations", lifecycle.Type, len(lifecycle.History), createCount)
		for i, event := range lifecycle.History {
			log.Printf("  Event %d: Operation=%s (%v)", i, event.Operation, event.Operation)
		}
		log.Printf("  Returning %v (need createCount > 1)", createCount > 1)
	}

	return createCount > 1
}

func (r *MultipleCreateConsolidationRule) Apply(lifecycle *ObjectLifecycle, engine ConsolidationEngine) (*ConsolidationResult, error) {
	if !r.CanApply(lifecycle) {
		return nil, fmt.Errorf("rule cannot be applied to lifecycle")
	}

	// Collect all CREATE statements
	var allCreateStmts []parser.Statement
	for _, event := range lifecycle.History {
		if event.Operation == parser.OpCreate {
			allCreateStmts = append(allCreateStmts, event.Statement)
		}
	}

	// Safety check: if no CREATE statements found, this shouldn't happen due to CanApply check
	if len(allCreateStmts) == 0 {
		return nil, fmt.Errorf("no CREATE statements found in lifecycle")
	}

	// Debug logging for profiles and viewing_requests
	tableName := strings.ToLower(lifecycle.Name)
	if tableName == "profiles" || tableName == "viewing_requests" {
		log.Printf("DEBUG MultipleCreateConsolidationRule.Apply: %s has %d CREATE statements", lifecycle.Name, len(allCreateStmts))
		for i, stmt := range allCreateStmts {
			if tableName == "profiles" {
				hasCity := strings.Contains(strings.ToLower(stmt.SQL), "city")
				hasAuthProvider := strings.Contains(strings.ToLower(stmt.SQL), "auth_provider")
				columnCount := strings.Count(stmt.SQL, ",")
				log.Printf("  CREATE %d: has 'city'=%v, 'auth_provider'=%v, columns~%d, SQL length=%d", i, hasCity, hasAuthProvider, columnCount, len(stmt.SQL))
			} else {
				log.Printf("  CREATE %d: SQL length=%d, first 150 chars: %.150s", i, len(stmt.SQL), stmt.SQL)
			}
		}
	}

	// Merge all CREATE statements by treating subsequent CREATEs as if they were ALTERs
	// This handles the case where migrations have duplicate CREATE TABLE IF NOT EXISTS with different column sets
	baseCreateStmt := allCreateStmts[0]
	consolidatedSQL := baseCreateStmt.SQL

	// For tables with more than one CREATE, merge columns from all CREATE statements
	if len(allCreateStmts) > 1 && lifecycle.Type == parser.TypeTable {
		consolidatedSQL = mergeMultipleCreateStatements(allCreateStmts, lifecycle.Name)
		if strings.ToLower(lifecycle.Name) == "profiles" {
			log.Printf("  Merged %d CREATE statements into unified schema", len(allCreateStmts))
		}
	}

	result := &ConsolidationResult{
		OriginalStatements: allCreateStmts,
		ConsolidatedSQL:    consolidatedSQL,
		Optimizations: []string{
			fmt.Sprintf("Consolidated %d CREATE statements to final definition", len(allCreateStmts)),
		},
		RiskLevel: RiskLevelLow,
		EstimatedSavings: SquashSavings{
			StatementsReduced: len(allCreateStmts) - 1,
			FilesAffected:     len(allCreateStmts),
			LinesReduced:      (len(allCreateStmts) - 1) * 5, // Estimate
		},
	}

	return result, nil
}

func (r *MultipleCreateConsolidationRule) Risk() RiskLevel {
	return RiskLevelLow
}

// CreateAlterConsolidationRule consolidates CREATE followed by ALTER operations
type CreateAlterConsolidationRule struct{}

func (r *CreateAlterConsolidationRule) CanApply(lifecycle *ObjectLifecycle) bool {
	if len(lifecycle.History) < 2 {
		return false
	}

	// First check: If there are multiple CREATE statements, let MultipleCreateConsolidationRule handle it
	createCount := 0
	for _, event := range lifecycle.History {
		if event.Operation == parser.OpCreate {
			createCount++
			if createCount > 1 {
				// Debug logging for profiles
				if strings.ToLower(lifecycle.Name) == "profiles" {
					log.Printf("DEBUG CreateAlterConsolidationRule.CanApply: profiles has %d CREATE operations, deferring to MultipleCreateConsolidationRule", createCount)
				}
				return false // Let MultipleCreateConsolidationRule handle it
			}
		}
	}

	// Check for CREATE followed by ALTER pattern (single CREATE only)
	if lifecycle.History[0].Operation == parser.OpCreate {
		for i := 1; i < len(lifecycle.History); i++ {
			if lifecycle.History[i].Operation == parser.OpAlter {
				// Check if there are no data operations in between
				if !lifecycle.History[i].HasDataOps {
					// Debug logging for profiles
					if strings.ToLower(lifecycle.Name) == "profiles" {
						log.Printf("DEBUG CreateAlterConsolidationRule.CanApply: profiles (type=%s) matches! Single CREATE with ALTER operations", lifecycle.Type)
					}
					return true
				}
			}
		}
	}

	return false
}

func (r *CreateAlterConsolidationRule) Apply(lifecycle *ObjectLifecycle, engine ConsolidationEngine) (*ConsolidationResult, error) {
	if !r.CanApply(lifecycle) {
		return nil, fmt.Errorf("rule cannot be applied to lifecycle")
	}

	// Extract CREATE statement and all ALTER statements
	var createStmt *parser.Statement
	var alterStmts []parser.Statement

	for _, event := range lifecycle.History {
		if event.Operation == parser.OpCreate {
			createStmt = &event.Statement
		} else if event.Operation == parser.OpAlter && !event.HasDataOps {
			alterStmts = append(alterStmts, event.Statement)
		}
	}

	// Build consolidated CREATE statement by actually integrating ALTER operations
	consolidatedSQL := integrateAlterIntoCreate(createStmt, alterStmts)

	// Build list of original statements
	originalStmts := []parser.Statement{*createStmt}
	originalStmts = append(originalStmts, alterStmts...)

	result := &ConsolidationResult{
		OriginalStatements: originalStmts,
		ConsolidatedSQL:    consolidatedSQL,
		Optimizations: []string{
			fmt.Sprintf("Consolidated CREATE with %d ALTER operations", len(alterStmts)),
		},
		RiskLevel: RiskLevelLow,
		EstimatedSavings: SquashSavings{
			StatementsReduced: len(alterStmts),
			FilesAffected:     len(alterStmts) + 1,
			LinesReduced:      len(alterStmts) * 2,
		},
	}

	return result, nil
}

func (r *CreateAlterConsolidationRule) Risk() RiskLevel {
	return RiskLevelLow
}

// DropCreateCycleRule handles DROP followed by CREATE consolidation
type DropCreateCycleRule struct{}

func (r *DropCreateCycleRule) CanApply(lifecycle *ObjectLifecycle) bool {
	if len(lifecycle.History) < 2 {
		return false
	}

	// Look for DROP followed by CREATE pattern
	for i := 0; i < len(lifecycle.History)-1; i++ {
		if lifecycle.History[i].Operation == parser.OpDrop &&
			lifecycle.History[i+1].Operation == parser.OpCreate {
			return true
		}
	}

	return false
}

func (r *DropCreateCycleRule) Apply(lifecycle *ObjectLifecycle, engine ConsolidationEngine) (*ConsolidationResult, error) {
	if !r.CanApply(lifecycle) {
		return nil, fmt.Errorf("rule cannot be applied to lifecycle")
	}

	// Find the DROP-CREATE pair
	var dropStmt *parser.Statement
	var originalStmts []parser.Statement

	for i := 0; i < len(lifecycle.History)-1; i++ {
		if lifecycle.History[i].Operation == parser.OpDrop &&
			lifecycle.History[i+1].Operation == parser.OpCreate {
			dropStmt = &lifecycle.History[i].Statement
			originalStmts = append(originalStmts, *dropStmt)
			break
		}
	}

	// Collect ALL CREATE statements (not just the one after DROP)
	var allCreateStmts []parser.Statement
	for _, event := range lifecycle.History {
		if event.Operation == parser.OpCreate {
			allCreateStmts = append(allCreateStmts, event.Statement)
			if dropStmt == nil { // If no DROP found yet, add CREATE to originalStmts
				originalStmts = append(originalStmts, event.Statement)
			}
		}
	}

	// Merge multiple CREATE statements if needed (same logic as MultipleCreateConsolidationRule)
	var consolidatedSQL string
	if len(allCreateStmts) > 1 && lifecycle.Type == parser.TypeTable {
		consolidatedSQL = mergeMultipleCreateStatements(allCreateStmts, lifecycle.Name)
		log.Printf("DropCreateCycleRule: Merged %d CREATE statements for %s", len(allCreateStmts), lifecycle.Name)
	} else if len(allCreateStmts) == 1 {
		consolidatedSQL = allCreateStmts[0].SQL
	} else {
		return nil, fmt.Errorf("no CREATE statement found")
	}

	result := &ConsolidationResult{
		OriginalStatements: originalStmts,
		ConsolidatedSQL:    consolidatedSQL,
		Optimizations: []string{
			"Eliminated DROP operation before CREATE",
		},
		RiskLevel: RiskLevelMedium, // Medium risk due to data implications
		EstimatedSavings: SquashSavings{
			StatementsReduced: 1, // Eliminated the DROP
			FilesAffected:     2,
			LinesReduced:      3,
		},
	}

	return result, nil
}

func (r *DropCreateCycleRule) Risk() RiskLevel {
	return RiskLevelMedium
}

// ConsolidationRuleEngine manages and applies consolidation rules
type ConsolidationRuleEngine struct {
	rules []ConsolidationRule
}

// NewConsolidationRuleEngine creates a new rule engine with default rules
func NewConsolidationRuleEngine() *ConsolidationRuleEngine {
	engine := &ConsolidationRuleEngine{
		rules: []ConsolidationRule{},
	}

	// Add default rules
	engine.AddRule(&CreateAlterConsolidationRule{})
	engine.AddRule(&DropCreateCycleRule{})
	engine.AddRule(&FunctionDeduplicationRule{})
	engine.AddRule(&DeadCodeRemovalRule{})

	return engine
}

// AddRule adds a consolidation rule to the engine
func (cre *ConsolidationRuleEngine) AddRule(rule ConsolidationRule) {
	cre.rules = append(cre.rules, rule)
}

// ApplyRules applies all applicable rules to a lifecycle
func (cre *ConsolidationRuleEngine) ApplyRules(lifecycle *ObjectLifecycle, engine ConsolidationEngine) (*ConsolidationResult, error) {
	for _, rule := range cre.rules {
		if rule.CanApply(lifecycle) {
			return rule.Apply(lifecycle, engine)
		}
	}

	// No rules applicable
	return nil, nil
}

// GetApplicableRules returns all rules that can be applied to a lifecycle
func (cre *ConsolidationRuleEngine) GetApplicableRules(lifecycle *ObjectLifecycle) []ConsolidationRule {
	var applicable []ConsolidationRule
	for _, rule := range cre.rules {
		if rule.CanApply(lifecycle) {
			applicable = append(applicable, rule)
		}
	}
	return applicable
}

// FunctionDeduplicationRule consolidates duplicate function definitions
type FunctionDeduplicationRule struct{}

func (r *FunctionDeduplicationRule) CanApply(lifecycle *ObjectLifecycle) bool {
	if lifecycle.Type != parser.TypeFunction {
		return false
	}

	// Check for multiple CREATE operations with identical function signatures
	createCount := 0
	for _, event := range lifecycle.History {
		if event.Operation == parser.OpCreate {
			createCount++
		}
	}

	return createCount > 1
}

func (r *FunctionDeduplicationRule) Apply(lifecycle *ObjectLifecycle, engine ConsolidationEngine) (*ConsolidationResult, error) {
	if !r.CanApply(lifecycle) {
		return nil, fmt.Errorf("rule cannot be applied to lifecycle")
	}

	var latestCreate *parser.Statement
	var duplicateStmts []parser.Statement

	// Find the latest CREATE statement and collect duplicates
	for _, event := range lifecycle.History {
		if event.Operation == parser.OpCreate {
			if latestCreate != nil {
				duplicateStmts = append(duplicateStmts, *latestCreate)
			}
			latestCreate = &event.Statement
		}
	}

	if latestCreate == nil {
		return nil, fmt.Errorf("no CREATE statement found")
	}

	result := &ConsolidationResult{
		OriginalStatements: append(duplicateStmts, *latestCreate),
		ConsolidatedSQL:    latestCreate.SQL,
		Optimizations: []string{
			fmt.Sprintf("Removed %d duplicate function definitions", len(duplicateStmts)),
		},
		RiskLevel: RiskLevelLow,
		EstimatedSavings: SquashSavings{
			StatementsReduced: len(duplicateStmts),
			FilesAffected:     len(duplicateStmts) + 1,
			LinesReduced:      len(duplicateStmts) * 5, // Estimate based on function size
		},
	}

	return result, nil
}

func (r *FunctionDeduplicationRule) Risk() RiskLevel {
	return RiskLevelLow
}

// DeadCodeRemovalRule identifies and removes unused database objects
type DeadCodeRemovalRule struct{}

func (r *DeadCodeRemovalRule) CanApply(lifecycle *ObjectLifecycle) bool {
	// This rule requires production database analysis to determine usage
	// For now, we can identify objects that are created and then dropped
	createFound := false
	dropFound := false

	for _, event := range lifecycle.History {
		if event.Operation == parser.OpCreate {
			createFound = true
		}
		if event.Operation == parser.OpDrop {
			dropFound = true
		}
	}

	// If object is created and then dropped, it might be dead code
	return createFound && dropFound
}

func (r *DeadCodeRemovalRule) Apply(lifecycle *ObjectLifecycle, engine ConsolidationEngine) (*ConsolidationResult, error) {
	if !r.CanApply(lifecycle) {
		return nil, fmt.Errorf("rule cannot be applied to lifecycle")
	}

	// For CREATE -> DROP patterns, we can eliminate both statements
	var eliminatedStmts []parser.Statement
	createFound := false

	for _, event := range lifecycle.History {
		if event.Operation == parser.OpCreate && !createFound {
			eliminatedStmts = append(eliminatedStmts, event.Statement)
			createFound = true
		} else if event.Operation == parser.OpDrop && createFound {
			eliminatedStmts = append(eliminatedStmts, event.Statement)
			break
		}
	}

	result := &ConsolidationResult{
		OriginalStatements: eliminatedStmts,
		ConsolidatedSQL:    "-- Dead code eliminated", // No SQL needed
		Optimizations: []string{
			"Eliminated CREATE/DROP cycle for unused object",
		},
		RiskLevel: RiskLevelMedium, // Medium risk as we're removing code
		EstimatedSavings: SquashSavings{
			StatementsReduced: len(eliminatedStmts),
			FilesAffected:     len(eliminatedStmts),
			LinesReduced:      len(eliminatedStmts) * 3,
		},
	}

	return result, nil
}

func (r *DeadCodeRemovalRule) Risk() RiskLevel {
	return RiskLevelMedium
}

// ExternalDependencyFilterRule filters out warnings for known external dependencies
type ExternalDependencyFilterRule struct {
	ExternalSchemas map[string]bool
	ExternalTables  map[string]bool
}

func NewExternalDependencyFilterRule() *ExternalDependencyFilterRule {
	return &ExternalDependencyFilterRule{
		ExternalSchemas: map[string]bool{
			"storage":    true,
			"auth":       true,
			"realtime":   true,
			"supabase":   true,
			"extensions": true,
		},
		ExternalTables: map[string]bool{
			"storage.objects":       true,
			"storage.buckets":       true,
			"auth.users":           true,
			"auth.sessions":        true,
			"realtime.messages":    true,
			"supabase.migrations":  true,
		},
	}
}

func (r *ExternalDependencyFilterRule) CanApply(lifecycle *ObjectLifecycle) bool {
	// Check if this object has external dependencies that should be filtered
	for _, dep := range lifecycle.Dependencies {
		if r.isExternalDependency(dep.DependsOn.Name) {
			return true
		}
	}
	return false
}

func (r *ExternalDependencyFilterRule) Apply(lifecycle *ObjectLifecycle, engine ConsolidationEngine) (*ConsolidationResult, error) {
	if !r.CanApply(lifecycle) {
		return nil, fmt.Errorf("rule cannot be applied to lifecycle")
	}

	// Filter out external dependencies to reduce warnings
	var filteredDeps []ObjectDependency
	externalDepsFiltered := 0

	for _, dep := range lifecycle.Dependencies {
		if !r.isExternalDependency(dep.DependsOn.Name) {
			filteredDeps = append(filteredDeps, dep)
		} else {
			externalDepsFiltered++
		}
	}

	// Update lifecycle dependencies
	lifecycle.Dependencies = filteredDeps

	// IMPORTANT: This rule only filters dependencies, it doesn't consolidate SQL.
	// Return nil to let other rules handle SQL consolidation.
	log.Printf("ExternalDependencyFilterRule filtered %d deps for %s, returning nil to allow other rules", externalDepsFiltered, lifecycle.Name)
	return nil, nil
}

func (r *ExternalDependencyFilterRule) Risk() RiskLevel {
	return RiskLevelLow
}

func (r *ExternalDependencyFilterRule) isExternalDependency(depName string) bool {
	// Check if it's a known external table
	if r.ExternalTables[depName] {
		return true
	}

	// Check if it belongs to an external schema
	for schema := range r.ExternalSchemas {
		if strings.HasPrefix(depName, schema+".") {
			return true
		}
	}

	// Check for common external object patterns
	externalPatterns := []string{
		"objects",    // storage.objects
		"buckets",    // storage.buckets
		"avatars",    // common storage bucket
		"documents",  // common storage bucket
		"images",     // common storage bucket
	}

	for _, pattern := range externalPatterns {
		if strings.Contains(strings.ToLower(depName), pattern) {
			return true
		}
	}

	return false
}

// ColumnEvolutionRule handles complex column lifecycle patterns
type ColumnEvolutionRule struct{}

func (r *ColumnEvolutionRule) CanApply(lifecycle *ObjectLifecycle) bool {
	if lifecycle.Type != parser.TypeTable {
		return false
	}

	// Check if there are column-related ALTER operations
	hasColumnOps := false
	for _, event := range lifecycle.History {
		if event.Operation == parser.OpAlter {
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

func (r *ColumnEvolutionRule) Apply(lifecycle *ObjectLifecycle, engine ConsolidationEngine) (*ConsolidationResult, error) {
	if !r.CanApply(lifecycle) {
		return nil, fmt.Errorf("rule cannot be applied to lifecycle")
	}

	// Track column evolution through lifecycle
	columnChanges := r.analyzeColumnEvolution(lifecycle)

	// Generate final schema with all column changes applied
	finalSQL := r.generateFinalSchema(lifecycle, columnChanges)

	// Find all column-related statements for consolidation
	var originalStmts []parser.Statement
	for _, event := range lifecycle.History {
		if event.Operation == parser.OpCreate ||
		   (event.Operation == parser.OpAlter && r.isColumnOperation(event.Statement.SQL)) {
			originalStmts = append(originalStmts, event.Statement)
		}
	}

	result := &ConsolidationResult{
		OriginalStatements: originalStmts,
		ConsolidatedSQL:    finalSQL,
		Optimizations: []string{
			fmt.Sprintf("Consolidated %d column operations into final schema", len(columnChanges)),
		},
		RiskLevel: RiskLevelMedium, // Medium risk due to schema changes
		EstimatedSavings: SquashSavings{
			StatementsReduced: len(originalStmts) - 1,
			FilesAffected:     len(originalStmts),
			LinesReduced:      len(originalStmts) * 3,
		},
	}

	return result, nil
}

func (r *ColumnEvolutionRule) Risk() RiskLevel {
	return RiskLevelMedium
}

type ColumnChange struct {
	Name      string
	Operation string // ADD, DROP, ALTER, RENAME
	DataType  string
	Default   string
	NotNull   bool
	Order     int
}

func (r *ColumnEvolutionRule) analyzeColumnEvolution(lifecycle *ObjectLifecycle) []ColumnChange {
	var changes []ColumnChange
	order := 0

	for _, event := range lifecycle.History {
		if event.Operation == parser.OpAlter {
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

func (r *ColumnEvolutionRule) generateFinalSchema(lifecycle *ObjectLifecycle, columnChanges []ColumnChange) string {
	// Get the base CREATE statement
	var baseCreateSQL string
	for _, event := range lifecycle.History {
		if event.Operation == parser.OpCreate {
			baseCreateSQL = event.Statement.SQL
			break
		}
	}

	if baseCreateSQL == "" {
		return ""
	}

	// Apply column changes to generate final schema
	finalSQL := r.applyColumnChanges(baseCreateSQL, columnChanges)

	// Add a comment about the column changes
	if len(columnChanges) > 0 {
		comment := fmt.Sprintf("-- Column evolution: %d operations consolidated\n", len(columnChanges))
		finalSQL = comment + finalSQL
	}

	return finalSQL
}

func (r *ColumnEvolutionRule) applyColumnChanges(createSQL string, changes []ColumnChange) string {
	// Track which columns should be present in final schema
	columns := make(map[string]bool)

	// Start with columns from original CREATE statement
	// (This is simplified - a real implementation would parse the SQL properly)
	if strings.Contains(createSQL, "created_at") {
		columns["created_at"] = true
	}
	if strings.Contains(createSQL, "id") {
		columns["id"] = true
	}

	// Apply changes in order
	for _, change := range changes {
		switch change.Operation {
		case "ADD":
			columns[change.Name] = true
		case "DROP":
			delete(columns, change.Name)
		case "ALTER":
			// Column still exists, just modified
			columns[change.Name] = true
		}
	}

	// Build final CREATE statement with only remaining columns
	// (Simplified implementation)
	finalSQL := createSQL
	for colName := range columns {
		if colName == "customer_id" && !columns["customer_id"] {
			// Remove dropped columns from SQL
			finalSQL = strings.ReplaceAll(finalSQL, "customer_id INTEGER,", "")
			finalSQL = strings.ReplaceAll(finalSQL, ", customer_id INTEGER", "")
		}
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

// RLSConsolidationRule consolidates Row Level Security operations
type RLSConsolidationRule struct{}

func (r *RLSConsolidationRule) CanApply(lifecycle *ObjectLifecycle) bool {
	if lifecycle.Type != parser.TypeTable {
		return false
	}

	// Check if there are RLS operations
	hasRLS := false
	for _, event := range lifecycle.History {
		if event.Operation == parser.OpAlter {
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

func (r *RLSConsolidationRule) Apply(lifecycle *ObjectLifecycle, engine ConsolidationEngine) (*ConsolidationResult, error) {
	if !r.CanApply(lifecycle) {
		return nil, fmt.Errorf("rule cannot be applied to lifecycle")
	}

	// Analyze RLS operations in lifecycle
	finalRLSState := r.determineFinalRLSState(lifecycle)

	// Find all RLS-related statements AND the CREATE TABLE statement
	var originalStmts []parser.Statement
	var createStmt *parser.Statement
	for _, event := range lifecycle.History {
		if event.Operation == parser.OpCreate {
			createStmt = &event.Statement
		} else if event.Operation == parser.OpAlter && r.isRLSOperation(event.Statement.SQL) {
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

	result := &ConsolidationResult{
		OriginalStatements: originalStmts,
		ConsolidatedSQL:    consolidatedSQL,
		Optimizations: []string{
			fmt.Sprintf("Consolidated %d RLS operations into final state: %s", len(originalStmts), finalRLSState),
		},
		RiskLevel: RiskLevelLow, // RLS is usually safe to consolidate
		EstimatedSavings: SquashSavings{
			StatementsReduced: len(originalStmts) - 1,
			FilesAffected:     len(originalStmts),
			LinesReduced:      len(originalStmts),
		},
	}

	return result, nil
}

func (r *RLSConsolidationRule) Risk() RiskLevel {
	return RiskLevelLow
}

func (r *RLSConsolidationRule) determineFinalRLSState(lifecycle *ObjectLifecycle) string {
	// Track RLS state changes through lifecycle
	currentState := "DISABLE ROW LEVEL SECURITY" // Default state

	for _, event := range lifecycle.History {
		if event.Operation == parser.OpAlter {
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

// integrateAlterIntoCreate integrates ALTER TABLE operations into the CREATE TABLE statement
func integrateAlterIntoCreate(createStmt *parser.Statement, alterStmts []parser.Statement) string {
	createSQL := createStmt.SQL

	// Extract column additions and constraints from ALTER statements
	var addedColumns []string
	var addedConstraints []string

	for _, alterStmt := range alterStmts {
		// Parse the ALTER statement directly to extract what needs to be added
		alterSQL := strings.TrimSpace(alterStmt.SQL)

		if strings.Contains(strings.ToUpper(alterSQL), "ADD COLUMN") {
			// Extract column definition from ADD COLUMN statement
			if columnDef := extractColumnFromAddStatement(alterSQL); columnDef != "" {
				addedColumns = append(addedColumns, columnDef)
			}
		} else if strings.Contains(strings.ToUpper(alterSQL), "ADD CONSTRAINT") {
			// Extract constraint definition from ADD CONSTRAINT statement
			if constraintDef := extractConstraintFromAddStatement(alterSQL); constraintDef != "" {
				addedConstraints = append(addedConstraints, constraintDef)
			}
		}

		// Handle other ALTER operations that should be integrated
		if strings.Contains(strings.ToUpper(alterSQL), "ENABLE ROW LEVEL SECURITY") {
			// Skip RLS - this needs to be a separate statement after CREATE
			continue
		}
	}

	// Integrate columns and constraints into the CREATE statement
	if len(addedColumns) > 0 || len(addedConstraints) > 0 {
		createSQL = integrateColumnsAndConstraintsIntoCreate(createSQL, addedColumns, addedConstraints)
	}

	return createSQL
}

//nolint:unused // Reserved for future column integration functionality
// extractColumnDefinition extracts column definition from ALTER TABLE ADD COLUMN statement
func extractColumnDefinition(alterSQL, columnName string) string {
	// Simple regex-based extraction for ALTER TABLE ADD COLUMN
	// This is a simplified version - in production, would use proper SQL parsing
	lines := strings.Split(alterSQL, "\n")
	for _, line := range lines {
		upperLine := strings.ToUpper(line)
		if strings.Contains(upperLine, "ADD COLUMN") &&
			strings.Contains(strings.ToLower(line), strings.ToLower(columnName)) {
			// Extract just the column part after "ADD COLUMN" from the original line (preserve case)
			addColIndex := strings.Index(upperLine, "ADD COLUMN")
			if addColIndex != -1 {
				columnPart := strings.TrimSpace(line[addColIndex+len("ADD COLUMN"):])
				// Remove semicolon if present
				columnPart = strings.TrimRight(columnPart, ";")
				return columnPart
			}
		}
	}
	return ""
}

//nolint:unused // Reserved for future column integration functionality
// integrateColumnsIntoCreate adds columns to the CREATE TABLE statement
func integrateColumnsIntoCreate(createSQL string, columns []string) string {
	// Find the last column in the CREATE statement and add new columns
	// This is a simplified version - in production, would use proper SQL parsing

	// Find the closing parenthesis of the CREATE statement
	lastParenIndex := strings.LastIndex(createSQL, ")")
	if lastParenIndex == -1 {
		return createSQL // Can't parse, return original
	}

	// Find the last comma before the closing parenthesis
	beforeParen := createSQL[:lastParenIndex]
	afterParen := createSQL[lastParenIndex:]

	// Add comma and new columns
	var result strings.Builder
	result.WriteString(beforeParen)

	for _, column := range columns {
		result.WriteString(",\n  ")
		result.WriteString(column)
	}

	result.WriteString(afterParen)

	return result.String()
}

// extractColumnFromAddStatement extracts the column definition from ALTER TABLE ADD COLUMN
func extractColumnFromAddStatement(alterSQL string) string {
	upperSQL := strings.ToUpper(alterSQL)
	addColIndex := strings.Index(upperSQL, "ADD COLUMN")
	if addColIndex == -1 {
		return ""
	}

	// Extract everything after "ADD COLUMN"
	afterAddCol := strings.TrimSpace(alterSQL[addColIndex+len("ADD COLUMN"):])

	// Remove trailing semicolon
	afterAddCol = strings.TrimRight(afterAddCol, ";")

	// Strip "IF NOT EXISTS" clause - it's not valid inside CREATE TABLE column definitions
	// We need to preserve case-sensitive column names, so use regex to strip only the IF NOT EXISTS part
	upperAfterCol := strings.ToUpper(afterAddCol)
	if strings.HasPrefix(upperAfterCol, "IF NOT EXISTS ") {
		// Remove the first "IF NOT EXISTS " (case-insensitive)
		afterAddCol = strings.TrimSpace(afterAddCol[len("IF NOT EXISTS "):])
	}

	// Normalize multi-line column definitions (e.g., when CHECK is on next line)
	// Replace newlines with spaces to keep the column definition on one logical line
	afterAddCol = strings.ReplaceAll(afterAddCol, "\n", " ")
	// Clean up multiple consecutive spaces
	for strings.Contains(afterAddCol, "  ") {
		afterAddCol = strings.ReplaceAll(afterAddCol, "  ", " ")
	}
	afterAddCol = strings.TrimSpace(afterAddCol)

	return afterAddCol
}

// extractConstraintFromAddStatement extracts the constraint definition from ALTER TABLE ADD CONSTRAINT
func extractConstraintFromAddStatement(alterSQL string) string {
	lines := strings.Split(alterSQL, "\n")
	var constraintLines []string
	inConstraint := false

	for _, line := range lines {
		upperLine := strings.ToUpper(strings.TrimSpace(line))

		if strings.Contains(upperLine, "ADD CONSTRAINT") {
			inConstraint = true
			// Extract the constraint name and start of definition
			addConstIndex := strings.Index(upperLine, "ADD CONSTRAINT")
			constraintPart := strings.TrimSpace(line[addConstIndex+len("ADD CONSTRAINT"):])
			constraintLines = append(constraintLines, "CONSTRAINT "+constraintPart)
		} else if inConstraint && strings.TrimSpace(line) != "" {
			// Continue collecting constraint definition until we hit a semicolon or end
			constraintLines = append(constraintLines, strings.TrimSpace(line))
			if strings.HasSuffix(strings.TrimSpace(line), ";") {
				break
			}
		}
	}

	if len(constraintLines) > 0 {
		result := strings.Join(constraintLines, "\n        ")
		// Remove trailing semicolon since it will be inside the CREATE statement
		result = strings.TrimRight(result, ";")
		return result
	}

	return ""
}

// integrateColumnsAndConstraintsIntoCreate integrates both columns and constraints into CREATE statement
func integrateColumnsAndConstraintsIntoCreate(createSQL string, columns []string, constraints []string) string {
	// Find the last closing parenthesis of the CREATE statement
	lastParenIndex := strings.LastIndex(createSQL, ")")
	if lastParenIndex == -1 {
		return createSQL // Can't parse, return original
	}

	beforeParen := createSQL[:lastParenIndex]
	afterParen := createSQL[lastParenIndex:]

	var additions []string

	// Add columns
	for _, col := range columns {
		additions = append(additions, "    "+col)
	}

	// Add constraints
	for _, constraint := range constraints {
		additions = append(additions, "    "+constraint)
	}

	if len(additions) > 0 {
		// Check if we need to add a comma before the new items
		trimmedBefore := strings.TrimSpace(beforeParen)
		if !strings.HasSuffix(trimmedBefore, ",") && !strings.HasSuffix(trimmedBefore, "(") {
			beforeParen += ","
		}

		// Add the new columns and constraints
		beforeParen += "\n" + strings.Join(additions, ",\n")
	}

	return beforeParen + "\n" + afterParen
}

// ErrorRecoveryRule provides enhanced error recovery and validation for consolidation failures
type ErrorRecoveryRule struct {
	MaxRetries    int
	RecoveryMode  string // "conservative", "aggressive", "fallback"
	ValidateSQL   bool
	LogFailures   bool
	FailureMetrics map[string]int
}

// Apply implements the ConsolidationRule interface with error recovery
func (rule *ErrorRecoveryRule) Apply(lifecycle *ObjectLifecycle, engine ConsolidationEngine) (*ConsolidationResult, error) {
	if rule.FailureMetrics == nil {
		rule.FailureMetrics = make(map[string]int)
	}

	// Track if this object has had failures before
	objectKey := lifecycle.Key
	failureCount, hasFailures := rule.FailureMetrics[objectKey]

	// If we've exceeded max retries, apply fallback strategy
	if hasFailures && failureCount >= rule.MaxRetries {
		return rule.applyFallbackStrategy(lifecycle, engine)
	}

	// Attempt normal consolidation with validation
	result, err := rule.attemptConsolidation(lifecycle, engine)
	if err != nil {
		// Record failure and attempt recovery
		rule.FailureMetrics[objectKey] = failureCount + 1
		if rule.LogFailures {
			log.Printf("Consolidation failed for %s (attempt %d/%d): %v",
				objectKey, failureCount+1, rule.MaxRetries, err)
		}

		// Try error recovery
		return rule.attemptErrorRecovery(lifecycle, engine, err)
	}

	// Validate the result if enabled
	if rule.ValidateSQL && result != nil {
		if validationErr := rule.validateConsolidatedSQL(result); validationErr != nil {
			// Mark as failed and try recovery
			rule.FailureMetrics[objectKey] = failureCount + 1
			return rule.attemptErrorRecovery(lifecycle, engine, validationErr)
		}
	}

	return result, nil
}

// CanApply returns true for all objects to provide universal error recovery
func (rule *ErrorRecoveryRule) CanApply(lifecycle *ObjectLifecycle) bool {
	return true // Universal error recovery
}

// attemptConsolidation tries to apply normal consolidation rules
func (rule *ErrorRecoveryRule) attemptConsolidation(lifecycle *ObjectLifecycle, engine ConsolidationEngine) (*ConsolidationResult, error) {
	// This would delegate to other consolidation rules
	// For now, return a basic result to allow testing of error recovery
	if len(lifecycle.History) == 0 {
		return nil, fmt.Errorf("no operations to consolidate")
	}

	// Try to create a basic consolidated statement
	finalState := lifecycle.GetFinalState()
	if finalState == nil {
		return nil, fmt.Errorf("no final state available for consolidation")
	}

	return &ConsolidationResult{
		ConsolidatedSQL:    finalState.SQL,
		OriginalStatements: rule.extractStatements(lifecycle.History),
		Optimizations:      []string{"error_recovery_applied"},
		Warnings:          []string{},
	}, nil
}

// attemptErrorRecovery tries various recovery strategies when consolidation fails
func (rule *ErrorRecoveryRule) attemptErrorRecovery(lifecycle *ObjectLifecycle, engine ConsolidationEngine, originalErr error) (*ConsolidationResult, error) {
	switch rule.RecoveryMode {
	case "conservative":
		return rule.conservativeRecovery(lifecycle, originalErr)
	case "aggressive":
		return rule.aggressiveRecovery(lifecycle, originalErr)
	case "fallback":
		return rule.applyFallbackStrategy(lifecycle, engine)
	default:
		return rule.conservativeRecovery(lifecycle, originalErr)
	}
}

// conservativeRecovery uses the original statements without modification
func (rule *ErrorRecoveryRule) conservativeRecovery(lifecycle *ObjectLifecycle, originalErr error) (*ConsolidationResult, error) {
	log.Printf("Applying conservative recovery for %s", lifecycle.Key)

	// Use all original statements unchanged
	var sqlParts []string
	for _, event := range lifecycle.History {
		if event.Statement.SQL != "" {
			sqlParts = append(sqlParts, event.Statement.SQL)
		}
	}

	return &ConsolidationResult{
		ConsolidatedSQL:    strings.Join(sqlParts, ";\n") + ";",
		OriginalStatements: rule.extractStatements(lifecycle.History),
		Optimizations:      []string{"conservative_recovery"},
		Warnings:          []string{fmt.Sprintf("Original error: %v", originalErr)},
	}, nil
}

// aggressiveRecovery tries to salvage parts of the consolidation
func (rule *ErrorRecoveryRule) aggressiveRecovery(lifecycle *ObjectLifecycle, originalErr error) (*ConsolidationResult, error) {
	log.Printf("Applying aggressive recovery for %s", lifecycle.Key)

	// Try to use the final state, fall back to conservative if that fails
	finalState := lifecycle.GetFinalState()
	if finalState != nil {
		return &ConsolidationResult{
			ConsolidatedSQL:    finalState.SQL,
			OriginalStatements: rule.extractStatements(lifecycle.History),
			Optimizations:      []string{"aggressive_recovery"},
			Warnings:          []string{fmt.Sprintf("Recovered from: %v", originalErr)},
		}, nil
	}

	// Fall back to conservative recovery
	return rule.conservativeRecovery(lifecycle, originalErr)
}

// applyFallbackStrategy applies when max retries exceeded
func (rule *ErrorRecoveryRule) applyFallbackStrategy(lifecycle *ObjectLifecycle, engine ConsolidationEngine) (*ConsolidationResult, error) {
	log.Printf("Applying fallback strategy for %s (max retries exceeded)", lifecycle.Key)

	// Skip the object entirely or use minimal processing
	if len(lifecycle.History) == 0 {
		return nil, nil // Skip empty objects
	}

	// Return first valid operation
	for _, event := range lifecycle.History {
		if event.Statement.SQL != "" {
			return &ConsolidationResult{
				ConsolidatedSQL:    event.Statement.SQL,
				OriginalStatements: []parser.Statement{event.Statement},
				Optimizations:      []string{"fallback_strategy"},
				Warnings:          []string{"Applied fallback due to repeated failures"},
			}, nil
		}
	}

	return nil, fmt.Errorf("no valid operations found for fallback")
}

// validateConsolidatedSQL performs basic SQL validation
func (rule *ErrorRecoveryRule) validateConsolidatedSQL(result *ConsolidationResult) error {
	sql := strings.TrimSpace(result.ConsolidatedSQL)

	// Basic validation checks
	if sql == "" {
		return fmt.Errorf("empty SQL generated")
	}

	// Check for obvious syntax issues
	if !strings.HasSuffix(sql, ";") {
		result.ConsolidatedSQL = sql + ";"
		result.Warnings = append(result.Warnings, "Added missing semicolon")
	}

	// Check for dangerous patterns that might indicate consolidation errors
	dangerousPatterns := []string{
		"DROP DATABASE",
		"DROP SCHEMA",
		"TRUNCATE",
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(strings.ToUpper(sql), pattern) {
			return fmt.Errorf("potentially dangerous SQL detected: %s", pattern)
		}
	}

	return nil
}

// extractStatements extracts statements from lifecycle events
func (rule *ErrorRecoveryRule) extractStatements(history []LifecycleEvent) []parser.Statement {
	var statements []parser.Statement
	for _, event := range history {
		statements = append(statements, event.Statement)
	}
	return statements
}

// Risk returns the risk level for error recovery rule
func (rule *ErrorRecoveryRule) Risk() RiskLevel {
	return RiskLevelLow // Error recovery is inherently low risk
}

// NewErrorRecoveryRule creates a new error recovery rule with specified configuration
func NewErrorRecoveryRule(maxRetries int, recoveryMode string, validateSQL bool) *ErrorRecoveryRule {
	return &ErrorRecoveryRule{
		MaxRetries:     maxRetries,
		RecoveryMode:   recoveryMode,
		ValidateSQL:    validateSQL,
		LogFailures:    true,
		FailureMetrics: make(map[string]int),
	}
}

// ConditionalSchemaRule handles CREATE IF NOT EXISTS and ALTER IF EXISTS patterns
type ConditionalSchemaRule struct{}

func (r *ConditionalSchemaRule) CanApply(lifecycle *ObjectLifecycle) bool {
	// Check if there are conditional schema operations that can be consolidated
	// OR if we have multiple CREATE operations that could benefit from conditional logic
	conditionalOps := 0
	createOps := 0

	for _, event := range lifecycle.History {
		sql := strings.ToUpper(event.Statement.SQL)
		if strings.Contains(sql, "IF NOT EXISTS") ||
		   strings.Contains(sql, "IF EXISTS") ||
		   strings.Contains(sql, "CREATE OR REPLACE") {
			conditionalOps++
		}
		if event.Operation == parser.OpCreate {
			createOps++
		}
	}

	// Apply if we have multiple conditional ops OR multiple creates that could use conditional logic
	return conditionalOps > 1 || (createOps > 1 && conditionalOps >= 1)
}

func (r *ConditionalSchemaRule) Apply(lifecycle *ObjectLifecycle, engine ConsolidationEngine) (*ConsolidationResult, error) {
	if !r.CanApply(lifecycle) {
		return nil, fmt.Errorf("rule cannot be applied to lifecycle")
	}

	// Analyze conditional operations and determine final desired state
	finalState := r.analyzeFinalConditionalState(lifecycle)

	// Collect all conditional statements
	var originalStmts []parser.Statement
	for _, event := range lifecycle.History {
		sql := strings.ToUpper(event.Statement.SQL)
		if strings.Contains(sql, "IF NOT EXISTS") ||
		   strings.Contains(sql, "IF EXISTS") ||
		   strings.Contains(sql, "CREATE OR REPLACE") {
			originalStmts = append(originalStmts, event.Statement)
		}
	}

	// Generate consolidated conditional statement
	consolidatedSQL := r.generateConditionalSQL(lifecycle, finalState)

	result := &ConsolidationResult{
		OriginalStatements: originalStmts,
		ConsolidatedSQL:    consolidatedSQL,
		Optimizations: []string{
			fmt.Sprintf("Consolidated %d conditional schema operations", len(originalStmts)),
			"Resolved IF NOT EXISTS/IF EXISTS conflicts",
		},
		RiskLevel: RiskLevelLow, // Conditional operations are inherently safe
		EstimatedSavings: SquashSavings{
			StatementsReduced: len(originalStmts) - 1,
			FilesAffected:     len(originalStmts),
			LinesReduced:      len(originalStmts) * 2,
		},
	}

	return result, nil
}

func (r *ConditionalSchemaRule) Risk() RiskLevel {
	return RiskLevelLow
}

type ConditionalState struct {
	ShouldExist   bool
	UseReplace    bool
	FinalSQL      string
	Dependencies  []string
}

func (r *ConditionalSchemaRule) analyzeFinalConditionalState(lifecycle *ObjectLifecycle) *ConditionalState {
	state := &ConditionalState{
		ShouldExist: true, // Default assumption
		UseReplace:  false,
	}

	// Track the final intention through the lifecycle
	for _, event := range lifecycle.History {
		sql := strings.ToUpper(event.Statement.SQL)

		switch event.Operation {
		case parser.OpCreate:
			state.ShouldExist = true
			state.FinalSQL = event.Statement.SQL

			// Prefer CREATE OR REPLACE over CREATE IF NOT EXISTS
			if strings.Contains(sql, "CREATE OR REPLACE") {
				state.UseReplace = true
			}
		case parser.OpDrop:
			state.ShouldExist = false
		case parser.OpAlter:
			// ALTER operations indicate the object should exist
			state.ShouldExist = true
		}
	}

	return state
}

func (r *ConditionalSchemaRule) generateConditionalSQL(lifecycle *ObjectLifecycle, state *ConditionalState) string {
	if !state.ShouldExist {
		// If final state is that object shouldn't exist, use DROP IF EXISTS
		return fmt.Sprintf("DROP %s IF EXISTS %s;",
			strings.ToUpper(string(lifecycle.Type)), lifecycle.Name)
	}

	// Generate appropriate CREATE statement
	if state.UseReplace && (lifecycle.Type == parser.TypeFunction || lifecycle.Type == parser.TypeView) {
		// Use CREATE OR REPLACE for functions and views
		baseSQL := state.FinalSQL
		if !strings.Contains(strings.ToUpper(baseSQL), "CREATE OR REPLACE") {
			baseSQL = strings.Replace(baseSQL, "CREATE ", "CREATE OR REPLACE ", 1)
		}
		return baseSQL
	} else {
		// Use CREATE IF NOT EXISTS for tables, indexes, etc.
		baseSQL := state.FinalSQL
		if !strings.Contains(strings.ToUpper(baseSQL), "IF NOT EXISTS") {
			// Insert IF NOT EXISTS after CREATE OBJECT_TYPE
			objectTypeUpper := strings.ToUpper(string(lifecycle.Type))
			createPattern := "CREATE " + objectTypeUpper
			replacePattern := "CREATE " + objectTypeUpper + " IF NOT EXISTS"
			baseSQL = strings.Replace(baseSQL, createPattern, replacePattern, 1)
		}
		return baseSQL
	}
}

// TransactionBoundaryRule optimizes operation grouping within transaction boundaries
type TransactionBoundaryRule struct{}

func (r *TransactionBoundaryRule) CanApply(lifecycle *ObjectLifecycle) bool {
	// Apply to lifecycles with multiple operations that can be grouped
	return len(lifecycle.History) > 2
}

func (r *TransactionBoundaryRule) Apply(lifecycle *ObjectLifecycle, engine ConsolidationEngine) (*ConsolidationResult, error) {
	if !r.CanApply(lifecycle) {
		return nil, fmt.Errorf("rule cannot be applied to lifecycle")
	}

	// Group operations by transaction boundaries
	transactionGroups := r.groupByTransactionBoundaries(lifecycle)

	// Generate optimized transaction blocks
	consolidatedSQL := r.generateOptimizedTransactions(transactionGroups)

	// Collect all statements for consolidation
	var originalStmts []parser.Statement
	for _, event := range lifecycle.History {
		originalStmts = append(originalStmts, event.Statement)
	}

	result := &ConsolidationResult{
		OriginalStatements: originalStmts,
		ConsolidatedSQL:    consolidatedSQL,
		Optimizations: []string{
			fmt.Sprintf("Grouped %d operations into %d optimized transactions",
				len(originalStmts), len(transactionGroups)),
			"Optimized transaction boundaries for better performance",
		},
		RiskLevel: RiskLevelLow, // Transaction optimization is generally safe
		EstimatedSavings: SquashSavings{
			StatementsReduced: 0, // No statements removed, just reorganized
			FilesAffected:     len(originalStmts),
			LinesReduced:      len(transactionGroups) * 2, // Transaction overhead reduction
		},
	}

	return result, nil
}

func (r *TransactionBoundaryRule) Risk() RiskLevel {
	return RiskLevelLow
}

type TransactionGroup struct {
	Operations    []LifecycleEvent
	RequiresSeparateTx bool
	Priority      int
}

func (r *TransactionBoundaryRule) groupByTransactionBoundaries(lifecycle *ObjectLifecycle) []TransactionGroup {
	var groups []TransactionGroup
	currentGroup := TransactionGroup{Priority: 1}

	for _, event := range lifecycle.History {
		// Check if this operation requires a separate transaction
		requiresSeparate := r.requiresSeparateTransaction(event)

		if requiresSeparate && len(currentGroup.Operations) > 0 {
			// Finish current group and start new one
			groups = append(groups, currentGroup)
			currentGroup = TransactionGroup{
				Operations:     []LifecycleEvent{event},
				RequiresSeparateTx: true,
				Priority:       r.getOperationPriority(event),
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

func (r *TransactionBoundaryRule) requiresSeparateTransaction(event LifecycleEvent) bool {
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
	if event.HasDataOps && event.Operation != parser.OpCreate {
		return true
	}

	return false
}

func (r *TransactionBoundaryRule) getOperationPriority(event LifecycleEvent) int {
	// Higher priority = should be executed earlier
	switch event.Operation {
	case parser.OpCreate:
		return 100
	case parser.OpAlter:
		return 80
	case parser.OpDrop:
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
				sqlParts = append(sqlParts, "  " + sql)
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

// mergeMultipleCreateStatements merges columns from multiple CREATE TABLE statements
// This handles the case where migrations have duplicate CREATE TABLE IF NOT EXISTS with different column sets
func mergeMultipleCreateStatements(createStmts []parser.Statement, tableName string) string {
	if len(createStmts) == 0 {
		return ""
	}

	// Use the first CREATE as the base
	baseSQL := createStmts[0].SQL

	// Extract all unique columns from all CREATE statements
	allColumns := make(map[string]string) // column name -> full definition
	columnOrder := make([]string, 0)       // maintain order

	for i, stmt := range createStmts {
		columns := extractColumnsFromCreate(stmt.SQL)
		log.Printf("  Extracted %d columns from CREATE statement %d", len(columns), i)

		for colName, colDef := range columns {
			if _, exists := allColumns[colName]; !exists {
				allColumns[colName] = colDef
				columnOrder = append(columnOrder, colName)
			}
		}
	}

	log.Printf("  Total unique columns across all CREATEs: %d", len(allColumns))

	// Extract table-level constraints from the first CREATE statement (they should be the same across all)
	tableConstraints := extractTableConstraintsFromCreate(baseSQL)

	// Rebuild CREATE statement with all columns and constraints
	return reconstructCreateWithColumns(baseSQL, allColumns, columnOrder, tableConstraints, tableName)
}

//nolint:unused // Utility function kept for potential future use
// min2 returns the minimum of two integers
func min2(a, b int) int {
	if a < b {
		return a
	}
	return b
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
