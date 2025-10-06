package squasher

import (
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"

	"github.com/capysquash/pg-squash-engine/internal/parser"
	"github.com/capysquash/pg-squash-engine/internal/tracking"
)

// UnifiedDependencyResolver provides comprehensive dependency resolution
// for both object lifecycle analysis and SQL consolidation phases
type UnifiedDependencyResolver struct {
	// Category and type ordering rules from EnhancedDependencyResolver
	categoryOrder map[parser.Category]int
	typeOrder     map[parser.ObjectType]int

	// SQL analysis caches from DependencyResolver
	tableDeps    map[string][]string
	schemaDeps   map[string][]string
	columnDeps   map[string][]string
	functionDeps map[string][]string
}

// NewUnifiedDependencyResolver creates a new unified dependency resolver
func NewUnifiedDependencyResolver() *UnifiedDependencyResolver {
	resolver := &UnifiedDependencyResolver{
		tableDeps:    make(map[string][]string),
		schemaDeps:   make(map[string][]string),
		columnDeps:   make(map[string][]string),
		functionDeps: make(map[string][]string),
	}

	// Define strict category ordering for PostgreSQL DDL
	resolver.categoryOrder = map[parser.Category]int{
		parser.CategoryExtensions:  0, // Extensions first
		parser.CategoryFoundation:  1, // Schemas, types, domains, tables
		parser.CategoryConstraints: 2, // Primary keys, unique constraints
		parser.CategoryIndexes:     3, // Indexes (after constraints)
		parser.CategoryFunctions:   4, // Functions and procedures
		parser.CategoryTriggers:    5, // Triggers (after functions)
		parser.CategorySecurity:    6, // RLS policies, grants
		parser.CategoryData:        7, // INSERT/UPDATE data operations
	}

	// Define object type ordering within categories
	resolver.typeOrder = map[parser.ObjectType]int{
		// Extensions first
		parser.TypeExtension: 0,

		// Foundation objects - types must come before tables that use them
		parser.TypeSchema:    10,
		parser.TypeEnum:      11, // ENUMs must come before tables that use them
		parser.TypeComposite: 12, // Composite types before tables
		parser.TypeDomain:    13, // Domains before tables
		parser.TypeType:      14, // Generic types before tables
		parser.TypeSequence:  15,
		parser.TypeTable:     16,

		// Constraints
		parser.TypeConstraint: 20,

		// Indexes
		parser.TypeIndex: 30,

		// Functions and procedures
		parser.TypeFunction: 40,

		// Triggers
		parser.TypeTrigger: 50,

		// Views
		parser.TypeView: 60,

		// Security
		parser.TypePolicy: 70,
		parser.TypeRole:   71,

		// Publications, etc.
		parser.TypePublication: 80,
		parser.TypeComment:     90,
		parser.TypeDoBlock:     91,
		parser.TypeUnknown:     100,
	}

	return resolver
}

// ResolveLifecycleDependencies performs advanced dependency resolution for object lifecycles
// This replaces the functionality from EnhancedDependencyResolver
func (udr *UnifiedDependencyResolver) ResolveLifecycleDependencies(
	graph *tracking.DependencyGraph,
	lifecycles map[string]*tracking.ObjectLifecycle,
) ([]tracking.ObjectID, error) {
	log.Printf("Starting unified dependency resolution for %d objects", len(lifecycles))

	// Step 1: Group objects by category
	categoryGroups := udr.groupLifecyclesByCategory(lifecycles)

	// Step 2: Resolve dependencies within each category
	var orderedObjects []tracking.ObjectID
	for categoryOrder := 0; categoryOrder < 8; categoryOrder++ {
		category := udr.getCategoryByOrder(categoryOrder)
		if objects, exists := categoryGroups[category]; exists {
			log.Printf("Processing category %s with %d objects", category, len(objects))

			// Sort within category using topological sort with cycle breaking
			sortedObjects, err := udr.resolveLifecycleWithinCategory(objects, category, graph)
			if err != nil {
				log.Printf("Warning: Error in category %s: %v", category, err)
				// Continue with best effort
				sortedObjects = udr.fallbackSortLifecycles(objects)
			}

			orderedObjects = append(orderedObjects, sortedObjects...)
		}
	}

	// Step 3: Final validation
	if err := udr.validateLifecycleOrdering(orderedObjects, lifecycles); err != nil {
		log.Printf("Final validation warnings: %v", err)
	}

	log.Printf("Unified dependency resolution completed with %d objects ordered", len(orderedObjects))
	return orderedObjects, nil
}

// SortConsolidationResults sorts consolidated SQL results by their dependencies within a category
// This replaces the functionality from DependencyResolver
func (udr *UnifiedDependencyResolver) SortConsolidationResults(
	categoryObjects map[string]*tracking.ConsolidationResult,
	category parser.Category,
	lifecycles map[string]*tracking.ObjectLifecycle,
) []*tracking.ConsolidationResult {

	if len(categoryObjects) <= 1 {
		// Single object or empty - no sorting needed
		var results []*tracking.ConsolidationResult
		for _, result := range categoryObjects {
			results = append(results, result)
		}
		return results
	}

	log.Printf("Resolving SQL dependencies for %d objects in category %s", len(categoryObjects), category)

	// Step 1: Analyze each object's dependencies and what it provides
	dependencies := make(map[string]*SQLDependencyInfo)
	for key, result := range categoryObjects {
		deps := udr.analyzeSQLDependencies(key, result, category, lifecycles)
		dependencies[key] = deps
	}

	// Step 2: Build dependency graph and topologically sort
	sorted := udr.topologicalSortSQL(dependencies, category)

	// Step 3: Convert back to results array
	var results []*tracking.ConsolidationResult
	for _, key := range sorted {
		if result, exists := categoryObjects[key]; exists {
			results = append(results, result)
		}
	}

	log.Printf("Sorted %d SQL objects by dependencies in category %s", len(results), category)
	return results
}

// SQLDependencyInfo holds dependency information for an object (renamed to avoid conflict)
type SQLDependencyInfo struct {
	ObjectKey      string
	Result         *tracking.ConsolidationResult
	Dependencies   []string // Objects this depends on
	Provides       []string // Objects this provides/creates
	RequiredFirst  bool     // Must be first in category (extensions, schemas)
	RequiredLast   bool     // Must be last in category (data operations)
}

// groupLifecyclesByCategory groups objects by their PostgreSQL category
func (udr *UnifiedDependencyResolver) groupLifecyclesByCategory(lifecycles map[string]*tracking.ObjectLifecycle) map[parser.Category][]tracking.ObjectID {
	groups := make(map[parser.Category][]tracking.ObjectID)

	for _, lifecycle := range lifecycles {
		objectID := tracking.ObjectID{
			Type: parser.ObjectType(lifecycle.Name), // Simplified mapping
			Name: lifecycle.Name,
		}

		category := udr.determineLifecycleCategory(lifecycle)
		groups[category] = append(groups[category], objectID)
	}

	return groups
}

// determineLifecycleCategory determines the PostgreSQL category for an object lifecycle
func (udr *UnifiedDependencyResolver) determineLifecycleCategory(lifecycle *tracking.ObjectLifecycle) parser.Category {
	// Use existing category if available
	if lifecycle.Category != "" {
		return lifecycle.Category
	}

	// Fallback category determination based on object type from lifecycle
	objectType := strings.ToUpper(lifecycle.Name)
	switch {
	case strings.Contains(objectType, "EXTENSION"):
		return parser.CategoryExtensions
	case strings.Contains(objectType, "SCHEMA") || strings.Contains(objectType, "TYPE") ||
		strings.Contains(objectType, "TABLE") || strings.Contains(objectType, "SEQUENCE"):
		return parser.CategoryFoundation
	case strings.Contains(objectType, "INDEX"):
		return parser.CategoryIndexes
	case strings.Contains(objectType, "FUNCTION") || strings.Contains(objectType, "PROCEDURE"):
		return parser.CategoryFunctions
	case strings.Contains(objectType, "TRIGGER"):
		return parser.CategoryTriggers
	case strings.Contains(objectType, "VIEW"):
		return parser.CategoryFoundation
	case strings.Contains(objectType, "POLICY") || strings.Contains(objectType, "GRANT"):
		return parser.CategorySecurity
	default:
		return parser.CategoryFoundation
	}
}

// resolveLifecycleWithinCategory resolves dependencies within a specific category for lifecycles
func (udr *UnifiedDependencyResolver) resolveLifecycleWithinCategory(
	objects []tracking.ObjectID,
	category parser.Category,
	graph *tracking.DependencyGraph,
) ([]tracking.ObjectID, error) {
	if len(objects) <= 1 {
		return objects, nil
	}

	// Create sub-graph for this category
	subGraph := udr.createSubGraph(objects, graph)

	// Try topological sort
	if sorted, err := subGraph.TopologicalSort(); err == nil {
		return sorted, nil
	}

	// If cycles exist, break them intelligently
	return udr.breakCyclesAndSort(objects, subGraph, category)
}

// createSubGraph creates a dependency subgraph for specific objects
func (udr *UnifiedDependencyResolver) createSubGraph(objects []tracking.ObjectID, originalGraph *tracking.DependencyGraph) *tracking.DependencyGraph {
	subGraph := tracking.NewDependencyGraph()

	// Add all objects as nodes
	for _, obj := range objects {
		subGraph.AddNode(obj)
	}

	// Add edges only between objects in this subgraph
	allNodes := originalGraph.GetAllNodes()
	for _, obj := range objects {
		if node, exists := allNodes[obj]; exists {
			for _, dep := range node.Dependencies {
				// Only add edge if both objects are in this subgraph
				if udr.objectInList(dep, objects) {
					subGraph.AddEdge(obj, dep)
				}
			}
		}
	}

	return subGraph
}

// breakCyclesAndSort intelligently breaks cycles and provides best-effort sorting
func (udr *UnifiedDependencyResolver) breakCyclesAndSort(
	objects []tracking.ObjectID,
	subGraph *tracking.DependencyGraph,
	category parser.Category,
) ([]tracking.ObjectID, error) {
	log.Printf("Breaking cycles in category %s with %d objects", category, len(objects))

	// Detect cycles
	cycles := subGraph.DetectCycles()
	if len(cycles) == 0 {
		// No cycles, should not happen, but handle gracefully
		return udr.fallbackSortLifecycles(objects), nil
	}

	// Strategy 1: Remove least important edges in cycles
	modifiedGraph := udr.removeCyclicEdges(subGraph, cycles)
	if sorted, err := modifiedGraph.TopologicalSort(); err == nil {
		log.Printf("Successfully broke %d cycles in category %s", len(cycles), category)
		return sorted, nil
	}

	// Strategy 2: Fallback to statement-type ordering
	return udr.fallbackSortLifecycles(objects), fmt.Errorf("cycles broken with fallback sorting")
}

// removeCyclicEdges removes edges that create cycles, prioritizing less important dependencies
func (udr *UnifiedDependencyResolver) removeCyclicEdges(graph *tracking.DependencyGraph, cycles [][]tracking.ObjectID) *tracking.DependencyGraph {
	modifiedGraph := udr.cloneGraph(graph)

	for _, cycle := range cycles {
		// Find the least important edge to remove
		edgeToRemove := udr.findLeastImportantLifecycleEdge(cycle)
		if edgeToRemove.from.Name != "" && edgeToRemove.to.Name != "" {
			udr.removeGraphEdge(modifiedGraph, edgeToRemove.from, edgeToRemove.to)
			log.Printf("Removed cyclic edge: %s -> %s", edgeToRemove.from.Name, edgeToRemove.to.Name)
		}
	}

	return modifiedGraph
}

// Edge represents a dependency edge
type Edge struct {
	from, to tracking.ObjectID
	weight   int // Lower weight = less important, easier to remove
}

// findLeastImportantLifecycleEdge finds the edge in a cycle that's safest to remove
func (udr *UnifiedDependencyResolver) findLeastImportantLifecycleEdge(cycle []tracking.ObjectID) Edge {
	var leastImportant Edge
	minWeight := 1000

	for i := 0; i < len(cycle); i++ {
		from := cycle[i]
		to := cycle[(i+1)%len(cycle)]

		weight := udr.calculateLifecycleEdgeWeight(from, to)
		if weight < minWeight {
			minWeight = weight
			leastImportant = Edge{from: from, to: to, weight: weight}
		}
	}

	return leastImportant
}

// calculateLifecycleEdgeWeight calculates the importance of a dependency edge
func (udr *UnifiedDependencyResolver) calculateLifecycleEdgeWeight(from, to tracking.ObjectID) int {
	weight := 100 // Base weight

	// Critical dependencies (never remove if possible)
	if udr.isCriticalLifecycleDependency(from, to) {
		weight += 500
	}

	// Schema dependencies are important
	if strings.Contains(string(to.Type), "SCHEMA") || strings.Contains(string(from.Type), "SCHEMA") {
		weight += 200
	}

	// Table to table dependencies are important
	if strings.Contains(string(from.Type), "TABLE") && strings.Contains(string(to.Type), "TABLE") {
		weight += 150
	}

	// Function to function dependencies can often be reordered
	if strings.Contains(string(from.Type), "FUNCTION") && strings.Contains(string(to.Type), "FUNCTION") {
		weight -= 50
	}

	// View dependencies are often safe to reorder
	if strings.Contains(string(from.Type), "VIEW") || strings.Contains(string(to.Type), "VIEW") {
		weight -= 30
	}

	return weight
}

// isCriticalLifecycleDependency checks if a dependency is critical and should not be removed
func (udr *UnifiedDependencyResolver) isCriticalLifecycleDependency(from, to tracking.ObjectID) bool {
	// Foreign key relationships
	if strings.Contains(strings.ToLower(from.Name), "foreign") ||
		strings.Contains(strings.ToLower(to.Name), "foreign") {
		return true
	}

	// Extension dependencies
	if strings.Contains(string(to.Type), "EXTENSION") {
		return true
	}

	// Schema dependencies
	if strings.Contains(string(to.Type), "SCHEMA") && !strings.Contains(string(from.Type), "SCHEMA") {
		return true
	}

	return false
}

// fallbackSortLifecycles provides object-type based sorting when topological sort fails
func (udr *UnifiedDependencyResolver) fallbackSortLifecycles(objects []tracking.ObjectID) []tracking.ObjectID {
	log.Printf("Using fallback sorting for %d lifecycle objects", len(objects))

	// Sort by object type order, then by name for stability
	sort.Slice(objects, func(i, j int) bool {
		// First, sort by type order
		orderI := udr.getTypeOrder(objects[i].Type)
		orderJ := udr.getTypeOrder(objects[j].Type)

		if orderI != orderJ {
			return orderI < orderJ
		}

		// Then by name for stability
		return objects[i].Name < objects[j].Name
	})

	return objects
}

// validateLifecycleOrdering performs final validation on the ordered objects
func (udr *UnifiedDependencyResolver) validateLifecycleOrdering(
	orderedObjects []tracking.ObjectID,
	lifecycles map[string]*tracking.ObjectLifecycle,
) error {
	var warnings []string

	// Check for any obvious ordering violations
	seenObjects := make(map[string]bool)

	for _, obj := range orderedObjects {
		objKey := fmt.Sprintf("%s:%s", obj.Type, obj.Name)

		// Check if this object's dependencies have been seen
		if lifecycle, exists := lifecycles[objKey]; exists {
			for _, dep := range lifecycle.Dependencies {
				depKey := fmt.Sprintf("%s:%s", dep.DependsOn.Type, dep.DependsOn.Name)
				if !seenObjects[depKey] {
					warnings = append(warnings, fmt.Sprintf("Object %s depends on %s which appears later", objKey, depKey))
				}
			}
		}

		seenObjects[objKey] = true
	}

	if len(warnings) > 0 {
		return fmt.Errorf("ordering validation warnings: %s", strings.Join(warnings, "; "))
	}

	return nil
}

// Helper functions

func (udr *UnifiedDependencyResolver) getCategoryByOrder(order int) parser.Category {
	for category, categoryOrder := range udr.categoryOrder {
		if categoryOrder == order {
			return category
		}
	}
	return parser.CategoryFoundation
}

func (udr *UnifiedDependencyResolver) getTypeOrder(objectType parser.ObjectType) int {
	if order, exists := udr.typeOrder[objectType]; exists {
		return order
	}
	return 100 // Default order
}

func (udr *UnifiedDependencyResolver) objectInList(obj tracking.ObjectID, list []tracking.ObjectID) bool {
	for _, item := range list {
		if item.Type == obj.Type && item.Name == obj.Name {
			return true
		}
	}
	return false
}

func (udr *UnifiedDependencyResolver) cloneGraph(original *tracking.DependencyGraph) *tracking.DependencyGraph {
	clone := tracking.NewDependencyGraph()

	// Get all nodes and edges from original graph and copy them
	nodes := original.GetAllNodes()
	for _, node := range nodes {
		clone.AddNode(node.ID)
		for _, dep := range node.Dependencies {
			clone.AddEdge(node.ID, dep)
		}
	}

	return clone
}

func (udr *UnifiedDependencyResolver) removeGraphEdge(graph *tracking.DependencyGraph, from, to tracking.ObjectID) {
	// Remove edge by removing the node and re-adding without this edge
	fromNode := graph.GetNode(from)
	toNode := graph.GetNode(to)

	if fromNode != nil {
		// Remove dependency from fromNode
		newDeps := make([]tracking.ObjectID, 0, len(fromNode.Dependencies))
		for _, dep := range fromNode.Dependencies {
			if dep != to {
				newDeps = append(newDeps, dep)
			}
		}
		fromNode.Dependencies = newDeps
	}

	if toNode != nil {
		// Remove dependent from toNode
		newDependents := make([]tracking.ObjectID, 0, len(toNode.Dependents))
		for _, dep := range toNode.Dependents {
			if dep != from {
				newDependents = append(newDependents, dep)
			}
		}
		toNode.Dependents = newDependents
	}
}

// SQL dependency analysis methods (from original DependencyResolver)

// analyzeSQLDependencies extracts dependencies from SQL statements
func (udr *UnifiedDependencyResolver) analyzeSQLDependencies(
	objectKey string,
	result *tracking.ConsolidationResult,
	category parser.Category,
	lifecycles map[string]*tracking.ObjectLifecycle,
) *SQLDependencyInfo {

	info := &SQLDependencyInfo{
		ObjectKey:    objectKey,
		Result:       result,
		Dependencies: []string{},
		Provides:     []string{},
	}

	sql := result.ConsolidatedSQL
	lowercaseSQL := strings.ToLower(sql)

	// Analyze based on category
	switch category {
	case parser.CategoryExtensions:
		info.RequiredFirst = true
		info.Provides = append(info.Provides, udr.extractExtensionProvisions(sql)...)

	case parser.CategoryFoundation:
		info.Dependencies = append(info.Dependencies, udr.extractSchemaDependencies(sql)...)
		info.Dependencies = append(info.Dependencies, udr.extractExtensionDependencies(sql)...)
		info.Dependencies = append(info.Dependencies, udr.extractTypeDependencies(sql)...)
		// CRITICAL: Extract table-to-table dependencies (foreign keys) for correct ordering
		info.Dependencies = append(info.Dependencies, udr.extractTableDependencies(sql)...)
		info.Provides = append(info.Provides, udr.extractTableProvisions(sql)...)
		info.Provides = append(info.Provides, udr.extractSchemaProvisions(sql)...)
		info.Provides = append(info.Provides, udr.extractTypeProvisions(sql)...)

	case parser.CategoryConstraints:
		info.Dependencies = append(info.Dependencies, udr.extractTableDependencies(sql)...)
		info.Dependencies = append(info.Dependencies, udr.extractColumnDependencies(sql)...)

	case parser.CategoryFunctions:
		info.Dependencies = append(info.Dependencies, udr.extractTableDependencies(sql)...)
		info.Dependencies = append(info.Dependencies, udr.extractSchemaDependencies(sql)...)
		info.Dependencies = append(info.Dependencies, udr.extractExtensionDependencies(sql)...)
		// CRITICAL: Extract function-to-function dependencies for correct ordering
		info.Dependencies = append(info.Dependencies, udr.extractFunctionDependencies(sql)...)
		// CRITICAL: Declare what functions this SQL provides for dependency resolution
		info.Provides = append(info.Provides, udr.extractFunctionProvisions(sql)...)

	case parser.CategoryTriggers:
		info.Dependencies = append(info.Dependencies, udr.extractTableDependencies(sql)...)
		info.Dependencies = append(info.Dependencies, udr.extractFunctionDependencies(sql)...)

	case parser.CategoryIndexes:
		info.Dependencies = append(info.Dependencies, udr.extractTableDependencies(sql)...)
		info.Dependencies = append(info.Dependencies, udr.extractColumnDependencies(sql)...)

	case parser.CategorySecurity:
		info.Dependencies = append(info.Dependencies, udr.extractTableDependencies(sql)...)

	case parser.CategoryData:
		info.RequiredLast = true
		info.Dependencies = append(info.Dependencies, udr.extractTableDependencies(sql)...)
		info.Dependencies = append(info.Dependencies, udr.extractColumnDependencies(sql)...)
		info.Dependencies = append(info.Dependencies, udr.extractSchemaDependencies(sql)...)

		// Data operations need more careful dependency analysis
		if strings.Contains(lowercaseSQL, "insert into") {
			info.Dependencies = append(info.Dependencies, udr.extractInsertDependencies(sql)...)
		}
		if strings.Contains(lowercaseSQL, "update") {
			info.Dependencies = append(info.Dependencies, udr.extractUpdateDependencies(sql)...)
		}
	}

	// Remove duplicates
	info.Dependencies = udr.removeDuplicates(info.Dependencies)
	info.Provides = udr.removeDuplicates(info.Provides)

	return info
}

// topologicalSortSQL performs dependency-based sorting for SQL consolidation results
func (udr *UnifiedDependencyResolver) topologicalSortSQL(dependencies map[string]*SQLDependencyInfo, category parser.Category) []string {
	// Build adjacency list and in-degree count
	graph := make(map[string][]string)
	inDegree := make(map[string]int)
	allNodes := make(map[string]bool)

	// Initialize all nodes
	for key := range dependencies {
		allNodes[key] = true
		inDegree[key] = 0
		graph[key] = []string{}
	}

	// Build the graph based on dependencies
	for key, info := range dependencies {
		for _, dep := range info.Dependencies {
			// Find if any other object provides this dependency
			for otherKey, otherInfo := range dependencies {
				if otherKey == key {
					continue
				}
				for _, provides := range otherInfo.Provides {
					if udr.dependencyMatches(dep, provides) {
						// otherKey must come before key
						graph[otherKey] = append(graph[otherKey], key)
						inDegree[key]++
						break
					}
				}
			}
		}
	}

	// Kahn's algorithm for topological sorting
	var queue []string
	var result []string

	// Handle special cases first
	for key, info := range dependencies {
		if info.RequiredFirst {
			queue = append(queue, key)
		}
	}

	// Add nodes with no incoming edges
	for key, degree := range inDegree {
		if degree == 0 && !dependencies[key].RequiredFirst {
			queue = append(queue, key)
		}
	}

	// Process the queue
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		// Skip if this should be last but we haven't processed all others
		if dependencies[current].RequiredLast {
			// Check if there are still non-RequiredLast nodes to process
			hasNonLast := false
			for _, neighbor := range graph[current] {
				if !dependencies[neighbor].RequiredLast {
					hasNonLast = true
					break
				}
			}
			if hasNonLast {
				// Put it at the end of queue and continue
				queue = append(queue, current)
				continue
			}
		}

		result = append(result, current)

		// Reduce in-degree for neighbors
		for _, neighbor := range graph[current] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	// Handle remaining nodes (potential circular dependencies)
	for key := range dependencies {
		found := false
		for _, resultKey := range result {
			if resultKey == key {
				found = true
				break
			}
		}
		if !found {
			log.Printf("Warning: Object %s may have circular dependency, adding to end", key)
			result = append(result, key)
		}
	}

	// Final sort for RequiredLast items
	sort.SliceStable(result, func(i, j int) bool {
		iLast := dependencies[result[i]].RequiredLast
		jLast := dependencies[result[j]].RequiredLast

		if iLast && !jLast {
			return false // i goes after j
		}
		if !iLast && jLast {
			return true // i goes before j
		}
		return false // maintain stable order
	})

	return result
}

// SQL extraction methods (from original DependencyResolver)

// extractTableDependencies finds table references in SQL, with special focus on foreign key dependencies
func (udr *UnifiedDependencyResolver) extractTableDependencies(sql string) []string {
	var deps []string

	// Priority 1: FOREIGN KEY REFERENCES - These are critical for table creation order
	// Tables must be created before they can be referenced
	foreignKeyPattern := `(?i)FOREIGN\s+KEY\s*\([^)]+\)\s+REFERENCES\s+([a-zA-Z_][a-zA-Z0-9_]*(?:\.[a-zA-Z_][a-zA-Z0-9_]*)?)`
	fkRe := regexp.MustCompile(foreignKeyPattern)
	fkMatches := fkRe.FindAllStringSubmatch(sql, -1)
	for _, match := range fkMatches {
		if len(match) > 1 {
			referencedTable := strings.TrimSpace(match[1])
			if referencedTable != "" {
				log.Printf("Found foreign key dependency: table depends on %s", referencedTable)
				deps = append(deps, referencedTable)
			}
		}
	}

	// Priority 2: Direct REFERENCES clause (shorthand foreign key syntax)
	directRefPattern := `(?i)REFERENCES\s+([a-zA-Z_][a-zA-Z0-9_]*(?:\.[a-zA-Z_][a-zA-Z0-9_]*)?)\s*\(`
	refRe := regexp.MustCompile(directRefPattern)
	refMatches := refRe.FindAllStringSubmatch(sql, -1)
	for _, match := range refMatches {
		if len(match) > 1 {
			referencedTable := strings.TrimSpace(match[1])
			if referencedTable != "" {
				log.Printf("Found direct reference dependency: table depends on %s", referencedTable)
				deps = append(deps, referencedTable)
			}
		}
	}

	// Priority 3: COMMENT ON statements - tables must exist before comments
	commentPattern := `(?i)COMMENT\s+ON\s+(?:TABLE|COLUMN)\s+([a-zA-Z_][a-zA-Z0-9_]*)`
	commentRe := regexp.MustCompile(commentPattern)
	commentMatches := commentRe.FindAllStringSubmatch(sql, -1)
	for _, match := range commentMatches {
		if len(match) > 1 {
			tableName := strings.TrimSpace(match[1])
			if tableName != "" {
				log.Printf("Found comment dependency: comment depends on table %s", tableName)
				deps = append(deps, tableName)
			}
		}
	}

	// Priority 4: Other table dependencies (for data operations)
	patterns := []string{
		`(?i)\bFROM\s+([a-zA-Z_][a-zA-Z0-9_]*(?:\.[a-zA-Z_][a-zA-Z0-9_]*)?)`,
		`(?i)\bJOIN\s+([a-zA-Z_][a-zA-Z0-9_]*(?:\.[a-zA-Z_][a-zA-Z0-9_]*)?)`,
		`(?i)\bINSERT\s+INTO\s+([a-zA-Z_][a-zA-Z0-9_]*(?:\.[a-zA-Z_][a-zA-Z0-9_]*)?)`,
		`(?i)\bUPDATE\s+([a-zA-Z_][a-zA-Z0-9_]*(?:\.[a-zA-Z_][a-zA-Z0-9_]*)?)`,
		`(?i)\bTABLE\s+([a-zA-Z_][a-zA-Z0-9_]*(?:\.[a-zA-Z_][a-zA-Z0-9_]*)?)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(sql, -1)
		for _, match := range matches {
			if len(match) > 1 {
				tableName := strings.TrimSpace(match[1])
				if tableName != "" && !strings.Contains(tableName, "(") {
					deps = append(deps, tableName)
				}
			}
		}
	}

	return deps
}

// extractSchemaDependencies finds schema references
func (udr *UnifiedDependencyResolver) extractSchemaDependencies(sql string) []string {
	var deps []string

	// Look for schema qualified references
	re := regexp.MustCompile(`(?i)([a-zA-Z_][a-zA-Z0-9_]*)\.[a-zA-Z_][a-zA-Z0-9_]*`)
	matches := re.FindAllStringSubmatch(sql, -1)
	for _, match := range matches {
		if len(match) > 1 {
			schema := strings.TrimSpace(match[1])
			if schema != "" && schema != "public" {
				deps = append(deps, fmt.Sprintf("schema:%s", schema))
			}
		}
	}

	return deps
}

// extractExtensionDependencies finds extension requirements
func (udr *UnifiedDependencyResolver) extractExtensionDependencies(sql string) []string {
	var deps []string
	lowercaseSQL := strings.ToLower(sql)

	// Known extensions and their indicators
	extensionIndicators := map[string][]string{
		"postgis":            {"geometry", "geography", "st_", "postgis"},
		"uuid-ossp":          {"uuid_generate", "gen_random_uuid"},
		"earthdistance":      {"earth_distance", "ll_to_earth", "cube"},
		"pg_stat_statements": {"pg_stat_statements"},
		"plpgsql":            {"$function$", "$body$"},
		"cube":               {"cube"},
		"pg_trgm":            {"similarity", "word_similarity"},
	}

	for extension, indicators := range extensionIndicators {
		for _, indicator := range indicators {
			if strings.Contains(lowercaseSQL, indicator) {
				deps = append(deps, fmt.Sprintf("extension:%s", extension))
				break
			}
		}
	}

	return deps
}

// extractColumnDependencies finds column references that require prior ALTER TABLE
func (udr *UnifiedDependencyResolver) extractColumnDependencies(sql string) []string {
	var deps []string

	// Look for INSERT statements with specific columns
	insertRegex := regexp.MustCompile(`(?i)INSERT\s+INTO\s+([a-zA-Z_][a-zA-Z0-9_]*(?:\.[a-zA-Z_][a-zA-Z0-9_]*)?)\s*\(\s*([^)]+)\)`)
	matches := insertRegex.FindAllStringSubmatch(sql, -1)
	for _, match := range matches {
		if len(match) > 2 {
			tableName := strings.TrimSpace(match[1])
			columns := strings.Split(match[2], ",")
			for _, col := range columns {
				col = strings.TrimSpace(col)
				if col != "" {
					deps = append(deps, fmt.Sprintf("column:%s.%s", tableName, col))
				}
			}
		}
	}

	return deps
}

// extractFunctionDependencies finds function references
func (udr *UnifiedDependencyResolver) extractFunctionDependencies(sql string) []string {
	var deps []string

	// Look for function calls in trigger definitions (EXECUTE FUNCTION)
	reExecute := regexp.MustCompile(`(?i)EXECUTE\s+FUNCTION\s+([a-zA-Z_][a-zA-Z0-9_]*(?:\.[a-zA-Z_][a-zA-Z0-9_]*)?)`)
	matches := reExecute.FindAllStringSubmatch(sql, -1)
	for _, match := range matches {
		if len(match) > 1 {
			funcName := strings.TrimSpace(match[1])
			if funcName != "" {
				deps = append(deps, fmt.Sprintf("function:%s", funcName))
			}
		}
	}

	// Look for function calls within function bodies (function_name(...))
	// This regex matches function calls: function_name() with optional schema qualifier
	// Pattern: word characters followed by optional dot and word characters, then opening parenthesis
	reCall := regexp.MustCompile(`\b([a-zA-Z_][a-zA-Z0-9_]*(?:\.[a-zA-Z_][a-zA-Z0-9_]*)?)\s*\(`)
	callMatches := reCall.FindAllStringSubmatch(sql, -1)

	// Track seen functions to avoid duplicates and filter out common SQL keywords/functions
	seen := make(map[string]bool)
	sqlKeywords := map[string]bool{
		"create": true, "or": true, "replace": true, "function": true, "returns": true,
		"language": true, "as": true, "begin": true, "end": true, "declare": true,
		"if": true, "then": true, "else": true, "elsif": true, "case": true, "when": true,
		"return": true, "raise": true, "exception": true, "for": true, "loop": true,
		"while": true, "select": true, "insert": true, "update": true, "delete": true,
		"from": true, "where": true, "into": true, "values": true, "set": true,
		"now": true, "count": true, "sum": true, "avg": true, "max": true, "min": true,
		"coalesce": true, "nullif": true, "cast": true, "extract": true, "substring": true,
		"trim": true, "lower": true, "upper": true, "concat": true, "array_agg": true,
		"json_build_object": true, "jsonb_build_object": true, "to_jsonb": true,
		"string_agg": true, "row_number": true, "rank": true, "dense_rank": true,
		"gen_random_uuid": true, "uuid_generate_v4": true, "current_timestamp": true,
	}

	for _, match := range callMatches {
		if len(match) > 1 {
			funcName := strings.ToLower(strings.TrimSpace(match[1]))
			// Skip if already seen, is a SQL keyword, or contains dots (likely schema.table)
			if !seen[funcName] && !sqlKeywords[funcName] && funcName != "" {
				seen[funcName] = true
				deps = append(deps, fmt.Sprintf("function:%s", funcName))
			}
		}
	}

	return deps
}

// extractInsertDependencies analyzes INSERT statements for dependencies
func (udr *UnifiedDependencyResolver) extractInsertDependencies(sql string) []string {
	var deps []string

	// INSERT statements depend on the table existing and columns being present
	deps = append(deps, udr.extractTableDependencies(sql)...)
	deps = append(deps, udr.extractColumnDependencies(sql)...)

	return deps
}

// extractUpdateDependencies analyzes UPDATE statements
func (udr *UnifiedDependencyResolver) extractUpdateDependencies(sql string) []string {
	var deps []string

	// UPDATE statements depend on tables and columns
	deps = append(deps, udr.extractTableDependencies(sql)...)

	// Look for column references in SET clause
	setRegex := regexp.MustCompile(`(?i)SET\s+([^=\s]+)\s*=`)
	matches := setRegex.FindAllStringSubmatch(sql, -1)
	for _, match := range matches {
		if len(match) > 1 {
			column := strings.TrimSpace(match[1])
			if column != "" {
				// Extract table name from UPDATE statement
				updateRegex := regexp.MustCompile(`(?i)UPDATE\s+([a-zA-Z_][a-zA-Z0-9_]*(?:\.[a-zA-Z_][a-zA-Z0-9_]*)?)`)
				updateMatches := updateRegex.FindStringSubmatch(sql)
				if len(updateMatches) > 1 {
					tableName := strings.TrimSpace(updateMatches[1])
					deps = append(deps, fmt.Sprintf("column:%s.%s", tableName, column))
				}
			}
		}
	}

	return deps
}

// extractTableProvisions finds what tables/objects this SQL creates
func (udr *UnifiedDependencyResolver) extractTableProvisions(sql string) []string {
	var provides []string

	// Look for CREATE TABLE statements
	re := regexp.MustCompile(`(?i)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?([a-zA-Z_][a-zA-Z0-9_]*(?:\.[a-zA-Z_][a-zA-Z0-9_]*)?)`)
	matches := re.FindAllStringSubmatch(sql, -1)
	for _, match := range matches {
		if len(match) > 1 {
			tableName := strings.TrimSpace(match[1])
			if tableName != "" {
				provides = append(provides, tableName)
			}
		}
	}

	// Look for CREATE SEQUENCE, CREATE VIEW, etc.
	objectTypes := []string{"SEQUENCE", "VIEW", "MATERIALIZED VIEW", "TYPE"}
	for _, objType := range objectTypes {
		pattern := fmt.Sprintf(`(?i)CREATE\s+%s\s+(?:IF\s+NOT\s+EXISTS\s+)?([a-zA-Z_][a-zA-Z0-9_]*(?:\.[a-zA-Z_][a-zA-Z0-9_]*)?)`, strings.Replace(objType, " ", `\s+`, -1))
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(sql, -1)
		for _, match := range matches {
			if len(match) > 1 {
				objName := strings.TrimSpace(match[1])
				if objName != "" {
					provides = append(provides, objName)
				}
			}
		}
	}

	return provides
}

// extractFunctionProvisions finds what functions this SQL creates
func (udr *UnifiedDependencyResolver) extractFunctionProvisions(sql string) []string {
	var provides []string

	// Look for CREATE [OR REPLACE] FUNCTION statements
	re := regexp.MustCompile(`(?i)CREATE\s+(?:OR\s+REPLACE\s+)?FUNCTION\s+(?:[a-zA-Z_][a-zA-Z0-9_]*\.)?([a-zA-Z_][a-zA-Z0-9_]*)\s*\(`)
	matches := re.FindAllStringSubmatch(sql, -1)
	for _, match := range matches {
		if len(match) > 1 {
			funcName := strings.ToLower(strings.TrimSpace(match[1]))
			if funcName != "" {
				provides = append(provides, fmt.Sprintf("function:%s", funcName))
			}
		}
	}

	return provides
}

// extractSchemaProvisions finds what schemas this SQL creates
func (udr *UnifiedDependencyResolver) extractSchemaProvisions(sql string) []string {
	var provides []string

	re := regexp.MustCompile(`(?i)CREATE\s+SCHEMA\s+(?:IF\s+NOT\s+EXISTS\s+)?([a-zA-Z_][a-zA-Z0-9_]*)`)
	matches := re.FindAllStringSubmatch(sql, -1)
	for _, match := range matches {
		if len(match) > 1 {
			schema := strings.TrimSpace(match[1])
			if schema != "" {
				provides = append(provides, fmt.Sprintf("schema:%s", schema))
			}
		}
	}

	return provides
}

// extractExtensionProvisions finds what extensions this SQL creates
func (udr *UnifiedDependencyResolver) extractExtensionProvisions(sql string) []string {
	var provides []string

	re := regexp.MustCompile(`(?i)CREATE\s+EXTENSION\s+(?:IF\s+NOT\s+EXISTS\s+)?([a-zA-Z_][a-zA-Z0-9_-]*)`)
	matches := re.FindAllStringSubmatch(sql, -1)
	for _, match := range matches {
		if len(match) > 1 {
			extension := strings.TrimSpace(match[1])
			if extension != "" {
				provides = append(provides, fmt.Sprintf("extension:%s", extension))
			}
		}
	}

	return provides
}

// extractTypeDependencies finds type/enum references that need to be created first
func (udr *UnifiedDependencyResolver) extractTypeDependencies(sql string) []string {
	var deps []string

	// Look for type usage in table definitions
	typePatterns := []string{
		`(?i)(\w+_enum)\s+(?:DEFAULT|NOT\s+NULL|NULL|,|\))`,
		`(?i)(\w+_status)\s+(?:DEFAULT|NOT\s+NULL|NULL|,|\))`,
		`(?i)(\w+_type)\s+(?:DEFAULT|NOT\s+NULL|NULL|,|\))`,
		`(?i)(\w+_role)\s+(?:DEFAULT|NOT\s+NULL|NULL|,|\))`,
	}

	for _, pattern := range typePatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(sql, -1)
		for _, match := range matches {
			if len(match) > 1 {
				typeName := strings.TrimSpace(match[1])
				if typeName != "" {
					deps = append(deps, fmt.Sprintf("type:%s", typeName))
				}
			}
		}
	}

	return deps
}

// extractTypeProvisions finds what types/enums this SQL creates
func (udr *UnifiedDependencyResolver) extractTypeProvisions(sql string) []string {
	var provides []string

	// Look for CREATE TYPE statements
	patterns := []string{
		`(?i)CREATE\s+TYPE\s+(?:IF\s+NOT\s+EXISTS\s+)?([a-zA-Z_][a-zA-Z0-9_]*(?:\.[a-zA-Z_][a-zA-Z0-9_]*)?)`,
		`(?i)CREATE\s+DOMAIN\s+(?:IF\s+NOT\s+EXISTS\s+)?([a-zA-Z_][a-zA-Z0-9_]*(?:\.[a-zA-Z_][a-zA-Z0-9_]*)?)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(sql, -1)
		for _, match := range matches {
			if len(match) > 1 {
				typeName := strings.TrimSpace(match[1])
				if typeName != "" {
					provides = append(provides, fmt.Sprintf("type:%s", typeName))
				}
			}
		}
	}

	return provides
}

// dependencyMatches checks if a dependency requirement matches a provision
func (udr *UnifiedDependencyResolver) dependencyMatches(dependency, provision string) bool {
	// Handle different types of dependencies
	if strings.HasPrefix(dependency, "schema:") && strings.HasPrefix(provision, "schema:") {
		return dependency == provision
	}
	if strings.HasPrefix(dependency, "extension:") && strings.HasPrefix(provision, "extension:") {
		return dependency == provision
	}
	if strings.HasPrefix(dependency, "function:") && strings.HasPrefix(provision, "function:") {
		return dependency == provision
	}
	if strings.HasPrefix(dependency, "type:") && strings.HasPrefix(provision, "type:") {
		return dependency == provision
	}
	if strings.HasPrefix(dependency, "column:") {
		// For column dependencies, we need table to exist first
		parts := strings.Split(dependency, ":")
		if len(parts) > 1 {
			tablePart := strings.Split(parts[1], ".")[0]
			return provision == tablePart
		}
	}

	// Direct table name match
	return dependency == provision
}

// removeDuplicates removes duplicate strings from slice
func (udr *UnifiedDependencyResolver) removeDuplicates(slice []string) []string {
	keys := make(map[string]bool)
	var result []string

	for _, item := range slice {
		if !keys[item] {
			keys[item] = true
			result = append(result, item)
		}
	}

	return result
}

// EnhanceExtensionSQL adds CASCADE to extension creation where needed
func (udr *UnifiedDependencyResolver) EnhanceExtensionSQL(sql string) string {
	// Extensions that typically require CASCADE
	cascadeExtensions := map[string]bool{
		"earthdistance": true,
		"postgis":       true,
	}

	// Check if this is an extension creation
	re := regexp.MustCompile(`(?i)(CREATE\s+EXTENSION\s+(?:IF\s+NOT\s+EXISTS\s+)?)([a-zA-Z_][a-zA-Z0-9_-]*)(\s*;?)`)
	return re.ReplaceAllStringFunc(sql, func(match string) string {
		parts := re.FindStringSubmatch(match)
		if len(parts) < 4 {
			return match
		}

		prefix := parts[1]
		extensionName := parts[2]
		suffix := parts[3]

		// Add CASCADE if needed and not already present
		if cascadeExtensions[strings.ToLower(extensionName)] && !strings.Contains(strings.ToLower(match), "cascade") {
			return prefix + extensionName + " CASCADE" + suffix
		}

		return match
	})
}