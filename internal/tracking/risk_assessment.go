package tracking

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/capysquash/pg-squash-engine/internal/parser"
)

// AddRule adds a risk assessment rule
func (ra *RiskAssessment) AddRule(rule RiskRule) {
	ra.rules = append(ra.rules, rule)
}

// Evaluate assesses the risk level for a lifecycle event
func (ra *RiskAssessment) Evaluate(event *LifecycleEvent, lifecycle *ObjectLifecycle) RiskLevel {
	maxRisk := RiskLevelLow

	for _, rule := range ra.rules {
		risk := rule.Evaluate(event, lifecycle)
		if risk > maxRisk {
			maxRisk = risk
		}
	}

	return maxRisk
}

// GetRiskRules returns all configured risk rules
func (ra *RiskAssessment) GetRiskRules() []RiskRule {
	return ra.rules
}

// Risk rule implementations

// DataLossRiskRule evaluates risk of data loss
type DataLossRiskRule struct{}

func (r *DataLossRiskRule) Evaluate(event *LifecycleEvent, lifecycle *ObjectLifecycle) RiskLevel {
	if event.Operation == parser.OpDrop {
		return RiskLevelHigh
	}
	if event.Operation == parser.OpAlter && event.HasDataOps {
		return RiskLevelMedium
	}
	return RiskLevelLow
}

func (r *DataLossRiskRule) Description() string {
	return "Evaluates risk of data loss from operations"
}

// CircularDependencyRiskRule evaluates circular dependency risks
type CircularDependencyRiskRule struct{}

func (r *CircularDependencyRiskRule) Evaluate(event *LifecycleEvent, lifecycle *ObjectLifecycle) RiskLevel {
	// Check if this operation might create circular dependencies
	if len(event.Dependencies) > 0 {
		// Simplified implementation - would need actual cycle detection
		return RiskLevelMedium
	}
	return RiskLevelLow
}

func (r *CircularDependencyRiskRule) Description() string {
	return "Evaluates risk of circular dependencies"
}

// CrossSchemaRiskRule evaluates cross-schema dependency risks
type CrossSchemaRiskRule struct{}

func (r *CrossSchemaRiskRule) Evaluate(event *LifecycleEvent, lifecycle *ObjectLifecycle) RiskLevel {
	// Check for cross-schema references
	for _, dep := range event.Dependencies {
		if strings.Contains(dep, ".") {
			return RiskLevelMedium
		}
	}
	return RiskLevelLow
}

func (r *CrossSchemaRiskRule) Description() string {
	return "Evaluates risk of cross-schema dependencies"
}

// ProductionUsageRiskRule evaluates production usage risks
type ProductionUsageRiskRule struct{}

func (r *ProductionUsageRiskRule) Evaluate(event *LifecycleEvent, lifecycle *ObjectLifecycle) RiskLevel {
	// Check if object is likely used in production
	if lifecycle.Type == parser.TypeTable || lifecycle.Type == parser.TypeView {
		return RiskLevelMedium
	}
	return RiskLevelLow
}

func (r *ProductionUsageRiskRule) Description() string {
	return "Evaluates risk based on production usage patterns"
}

// ConstraintModificationRiskRule evaluates risks from constraint modifications
type ConstraintModificationRiskRule struct{}

func (r *ConstraintModificationRiskRule) Evaluate(event *LifecycleEvent, lifecycle *ObjectLifecycle) RiskLevel {
	if lifecycle.Type == parser.TypeConstraint {
		if event.Operation == parser.OpDrop {
			return RiskLevelHigh
		}
		if event.Operation == parser.OpAlter {
			return RiskLevelMedium
		}
	}
	return RiskLevelLow
}

func (r *ConstraintModificationRiskRule) Description() string {
	return "Evaluates risk from constraint modifications"
}

// PermissionChangeRiskRule evaluates risks from permission changes
type PermissionChangeRiskRule struct{}

func (r *PermissionChangeRiskRule) Evaluate(event *LifecycleEvent, lifecycle *ObjectLifecycle) RiskLevel {
	if event.Operation == parser.OpRevoke {
		return RiskLevelMedium
	}
	if event.Operation == parser.OpGrant && len(lifecycle.Permissions) > 10 {
		return RiskLevelMedium // Complex permission structure
	}
	return RiskLevelLow
}

func (r *PermissionChangeRiskRule) Description() string {
	return "Evaluates risk from permission changes"
}

// Analysis methods for ObjectLifecycle

// CanSquash determines if an object's lifecycle can be safely squashed
func (obj *ObjectLifecycle) CanSquash() bool {
	// Check for data operations between schema changes
	for i := 0; i < len(obj.History)-1; i++ {
		if obj.History[i].HasDataOps {
			return false
		}
	}

	// Check for high-risk operations
	for _, event := range obj.History {
		if event.RiskLevel == RiskLevelHigh || event.RiskLevel == RiskLevelCritical {
			return false
		}
	}

	return true
}

// GetFinalState returns the final state of the object
// For tables, this returns the CREATE statement, not ALTER statements
func (obj *ObjectLifecycle) GetFinalState() *parser.Statement {
	if len(obj.History) == 0 {
		return nil
	}

	// For tables, prefer returning CREATE over ALTER
	if obj.Type == parser.TypeTable {
		// Debug: log history for profiles and viewing_requests tables
		tableName := strings.ToLower(obj.Name)
		if tableName == "profiles" || tableName == "viewing_requests" {
			log.Printf("DEBUG GetFinalState: %s table has %d history events:", obj.Name, len(obj.History))
			for i, event := range obj.History {
				log.Printf("  Event %d: Operation=%s, SQL length=%d", i, event.Operation, len(event.Statement.SQL))
			}
		}

		// Find the LAST CREATE statement (most recent schema definition)
		// When there are multiple CREATE TABLE IF NOT EXISTS statements,
		// the last one represents the most up-to-date schema
		var lastCreate *parser.Statement
		createCount := 0
		for i := range obj.History {
			if obj.History[i].Operation == parser.OpCreate {
				lastCreate = &obj.History[i].Statement
				createCount++
			}
		}
		if createCount > 1 {
			log.Printf("Table %s has %d CREATE statements, using last one", obj.Name, createCount)
		}
		if tableName == "viewing_requests" {
			if lastCreate != nil {
				log.Printf("DEBUG GetFinalState: viewing_requests returning CREATE statement with SQL length=%d", len(lastCreate.SQL))
			} else {
				log.Printf("DEBUG GetFinalState: viewing_requests has NO CREATE statement, returning nil!")
			}
		}
		if lastCreate != nil {
			return lastCreate
		}
	}

	// Return the last non-DROP operation
	for i := len(obj.History) - 1; i >= 0; i-- {
		if obj.History[i].Operation != parser.OpDrop {
			return &obj.History[i].Statement
		}
	}

	return nil
}

// GetAlterStatements returns all unique ALTER statements for the object (used for tables with column additions)
func (obj *ObjectLifecycle) GetAlterStatements() []parser.Statement {
	seen := make(map[string]bool)
	var alters []parser.Statement

	for _, event := range obj.History {
		if event.Operation == parser.OpAlter {
			// Normalize the SQL for deduplication (trim whitespace)
			normalizedSQL := strings.TrimSpace(event.Statement.SQL)

			// Skip if we've already seen this exact ALTER statement
			if seen[normalizedSQL] {
				continue
			}
			seen[normalizedSQL] = true

			// Ensure the ALTER statement ends with a semicolon
			if !strings.HasSuffix(normalizedSQL, ";") {
				normalizedSQL += ";"
			}

			// Create a new statement with the normalized SQL
			stmt := event.Statement
			stmt.SQL = normalizedSQL
			alters = append(alters, stmt)
		}
	}
	return alters
}

// HasConflicts checks for conflicting operations
func (obj *ObjectLifecycle) HasConflicts() bool {
	// Check for operations that might conflict
	operations := make(map[parser.Operation]bool)
	for _, event := range obj.History {
		if operations[event.Operation] && event.Operation == parser.OpCreate {
			return true // Duplicate CREATE operation
		}
		operations[event.Operation] = true
	}
	return false
}

// GetHighestRiskLevel returns the highest risk level in the lifecycle
func (obj *ObjectLifecycle) GetHighestRiskLevel() RiskLevel {
	maxRisk := RiskLevelLow
	for _, event := range obj.History {
		if event.RiskLevel > maxRisk {
			maxRisk = event.RiskLevel
		}
	}
	return maxRisk
}

// GetConsolidatedPermissions returns the final permission state for an object
func (obj *ObjectLifecycle) GetConsolidatedPermissions() []PermissionEvent {
	if len(obj.Permissions) == 0 {
		return nil
	}

	// Track the final state of each grantee-privilege combination
	finalPermissions := make(map[string]PermissionEvent)

	for _, permEvent := range obj.Permissions {
		key := fmt.Sprintf("%s::%s", permEvent.Grantee, permEvent.Privilege)

		switch permEvent.Operation {
		case parser.OpGrant:
			// Grant the permission
			finalPermissions[key] = permEvent
		case parser.OpRevoke:
			// Revoke the permission (remove from final state)
			delete(finalPermissions, key)
		}
	}

	// Convert back to slice
	consolidated := make([]PermissionEvent, 0, len(finalPermissions))
	for _, permEvent := range finalPermissions {
		consolidated = append(consolidated, permEvent)
	}

	return consolidated
}

// GetPermissionStatements returns consolidated GRANT statements for this object
func (obj *ObjectLifecycle) GetPermissionStatements() []parser.Statement {
	consolidatedPerms := obj.GetConsolidatedPermissions()
	if len(consolidatedPerms) == 0 {
		return nil
	}

	// Group by grantee to create efficient GRANT statements
	granteePerms := make(map[string][]string)
	for _, perm := range consolidatedPerms {
		granteePerms[perm.Grantee] = append(granteePerms[perm.Grantee], perm.Privilege)
	}

	var statements []parser.Statement
	for grantee, privileges := range granteePerms {
		// Create a single GRANT statement for this grantee
		stmt := parser.Statement{
			Operation:  parser.OpGrant,
			ObjectType: obj.Type,
			ObjectName: obj.Name,
			Grantees:   []string{grantee},
			Privileges: privileges,
			Category:   parser.CategorySecurity,
			SQL:        fmt.Sprintf("GRANT %s ON %s TO %s", strings.Join(privileges, ", "), obj.Name, grantee),
		}
		statements = append(statements, stmt)
	}

	return statements
}

// Analysis methods for UnifiedTracker

// GetRedundantObjects analyzes objects for redundancy patterns
func (ut *UnifiedTracker) GetRedundantObjects() []RedundancyReport {
	reports := []RedundancyReport{}

	for _, obj := range ut.objects {
		if report := ut.analyzeRedundancy(obj); report != nil {
			reports = append(reports, *report)
		}
	}

	// Sort by potential savings
	sort.Slice(reports, func(i, j int) bool {
		return reports[i].Savings.StatementsReduced > reports[j].Savings.StatementsReduced
	})

	return reports
}

// analyzeRedundancy analyzes an object lifecycle for redundancy patterns
func (ut *UnifiedTracker) analyzeRedundancy(obj *ObjectLifecycle) *RedundancyReport {
	if len(obj.History) < 2 {
		return nil
	}

	// Pattern: CREATE -> multiple ALTERs
	if obj.History[0].Operation == parser.OpCreate {
		alterCount := 0
		hasDataOps := false
		filesAffected := make(map[string]bool)

		for i := 1; i < len(obj.History); i++ {
			event := obj.History[i]
			if event.Operation == parser.OpAlter {
				alterCount++
				filesAffected[event.Migration] = true
			}
			if event.HasDataOps {
				hasDataOps = true
			}
		}

		if alterCount > 0 && !hasDataOps {
			return &RedundancyReport{
				Object:      obj.Name,
				Type:        obj.Type,
				Pattern:     PatternCreateAlterSequence,
				CanSquash:   true,
				Explanation: fmt.Sprintf("CREATE followed by %d ALTER operations can be consolidated", alterCount),
				Events:      obj.History,
				Savings: SquashSavings{
					StatementsReduced: alterCount,
					FilesAffected:     len(filesAffected),
					LinesReduced:      alterCount * 2, // Estimate
				},
			}
		}
	}

	// Pattern: DROP -> CREATE
	for i := 0; i < len(obj.History)-1; i++ {
		if obj.History[i].Operation == parser.OpDrop &&
			obj.History[i+1].Operation == parser.OpCreate {
			return &RedundancyReport{
				Object:      obj.Name,
				Type:        obj.Type,
				Pattern:     PatternDropCreateSequence,
				CanSquash:   true,
				Explanation: "DROP followed by CREATE can be consolidated into single CREATE",
				Events:      obj.History[i : i+2],
				Savings: SquashSavings{
					StatementsReduced: 1, // Remove the DROP
					FilesAffected:     2,
					LinesReduced:      3, // Estimate
				},
			}
		}
	}

	// Pattern: Duplicate operations
	operations := make(map[parser.Operation][]LifecycleEvent)
	for _, event := range obj.History {
		operations[event.Operation] = append(operations[event.Operation], event)
	}

	for op, events := range operations {
		if len(events) > 1 && op == parser.OpCreate {
			// Multiple CREATE statements for same object
			return &RedundancyReport{
				Object:      obj.Name,
				Type:        obj.Type,
				Pattern:     PatternDuplicateOperations,
				CanSquash:   true,
				Explanation: fmt.Sprintf("Multiple %s operations detected", op),
				Events:      events,
				Savings: SquashSavings{
					StatementsReduced: len(events) - 1,
					FilesAffected:     len(events),
					LinesReduced:      (len(events) - 1) * 5, // Estimate
				},
			}
		}
	}

	return nil
}

// GetObjectsByCategory returns objects grouped by category
func (ut *UnifiedTracker) GetObjectsByCategory() map[parser.Category][]*ObjectLifecycle {
	categories := make(map[parser.Category][]*ObjectLifecycle)

	for _, obj := range ut.objects {
		categories[obj.Category] = append(categories[obj.Category], obj)
	}

	return categories
}

// GetStatistics returns comprehensive tracking statistics
func (ut *UnifiedTracker) GetStatistics() TrackerStats {
	stats := TrackerStats{
		TotalObjects:    len(ut.objects),
		TotalMigrations: len(ut.migrations),
		ResourceChanges: len(ut.resourceChanges),
	}

	// Count by type
	stats.ObjectsByType = make(map[parser.ObjectType]int)
	stats.ObjectsByCategory = make(map[parser.Category]int)
	stats.ChangesByType = make(map[ResourceChangeType]int)

	for _, obj := range ut.objects {
		stats.ObjectsByType[obj.Type]++
		stats.ObjectsByCategory[obj.Category]++
	}

	// Count statements and changes
	for _, migration := range ut.migrations {
		stats.TotalStatements += len(migration.Statements)
		for _, stmt := range migration.Statements {
			if stmt.IsDataOp {
				stats.DataOperations++
			}
		}
	}

	for _, change := range ut.resourceChanges {
		stats.ChangesByType[change.Type]++
	}

	return stats
}

// ValidateConsistency checks for consistency issues in tracking
func (ut *UnifiedTracker) ValidateConsistency() []string {
	var warnings []string

	// Check for objects that are used but never created
	for objKey, deps := range ut.dependencies {
		for _, dep := range deps {
			depKey := makeKey(dep.DependsOn.Name, dep.DependsOn.Type)
			if _, exists := ut.objects[depKey]; !exists {
				warnings = append(warnings,
					fmt.Sprintf("Object %s depends on %s which is never created",
						objKey, dep.DependsOn.Name))
			}
		}
	}

	// Check for circular dependencies
	cycles := ut.dependencyGraph.DetectCycles()
	if len(cycles) > 0 {
		for i, cycle := range cycles {
			var cycleNames []string
			for _, obj := range cycle {
				cycleNames = append(cycleNames, obj.String())
			}
			warnings = append(warnings,
				fmt.Sprintf("Circular dependency detected (cycle %d): %s",
					i+1, strings.Join(cycleNames, " -> ")))
		}
	}

	// Check for objects with high risk levels
	for _, obj := range ut.objects {
		if obj.GetHighestRiskLevel() == RiskLevelCritical {
			warnings = append(warnings,
				fmt.Sprintf("Object %s has critical risk level operations", obj.Name))
		}
	}

	return warnings
}

// GetResourceChanges returns all resource changes
func (ut *UnifiedTracker) GetResourceChanges() []*ResourceChange {
	return ut.resourceChanges
}

// GetResourceChangesByType returns resource changes filtered by type
func (ut *UnifiedTracker) GetResourceChangesByType(changeType ResourceChangeType) []*ResourceChange {
	var filtered []*ResourceChange
	for _, change := range ut.resourceChanges {
		if change.Type == changeType {
			filtered = append(filtered, change)
		}
	}
	return filtered
}
