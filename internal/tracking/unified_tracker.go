// Package tracking provides database object lifecycle tracking and dependency management.
// It maintains state of database objects across migrations, analyzes dependencies,
// and supports the migration consolidation pipeline.
package tracking

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/capysquash/pg-squash/internal/metadata"
	"github.com/capysquash/pg-squash/internal/parser"
)

// UnifiedTracker provides comprehensive object lifecycle tracking with advanced metadata integration
type UnifiedTracker struct {
	objects         map[string]*ObjectLifecycle
	migrations      []*parser.Migration
	dependencies    map[string][]ObjectDependency
	metadataManager *metadata.MetadataManager
	dependencyGraph *DependencyGraph
	riskAssessment  *RiskAssessment
	processedEvents map[string]*LifecycleEvent
	resourceChanges []*ResourceChange
	changeTracker   *ChangeTracker

	// DDL cycle detection
	cycleDetector *AdvancedDDLCycleDetector
	detectedCycles []DDLCycle

	// Streaming processing support
	streamingMode    bool
	migrationCounter int64
	processedCount   int64
}

// ObjectLifecycle tracks complete database object lifecycle with advanced metadata integration
type ObjectLifecycle struct {
	Key          string // schema.name.type
	Name         string
	Schema       string
	Type         parser.ObjectType
	History      []LifecycleEvent
	Permissions  []PermissionEvent
	Dependencies []ObjectDependency
	Category     parser.Category
	WasDropped   bool

	// Analysis results
	IsRedundant   bool
	CanBeSquashed bool
	RiskLevel     RiskLevel

	// Metadata integration
	Metadata     ObjectMetadata
	CreatedAt    time.Time
	LastModified time.Time

	// Consolidation state
	ConsolidationResult *ConsolidationResult

	// Resource change tracking
	ResourceChanges []*ResourceChange

	// Absorbed from ChangeTracker
	changeCounter    int64
	operationContext *ChangeContext
}

// LifecycleEvent represents an event in an object's lifecycle with enhanced context
type LifecycleEvent struct {
	ID           string // Unique event identifier
	Migration    string
	Sequence     int
	Operation    parser.Operation
	Statement    parser.Statement
	Timestamp    time.Time        // When this event occurred
	Context      OperationContext // Enhanced context
	HasDataOps   bool
	Dependencies []string     // Dependencies at this point in time
	RiskLevel    RiskLevel    // Risk assessment for this operation
	SourceRange  *SourceRange // Source location information
}

// PermissionEvent tracks GRANT/REVOKE operations
type PermissionEvent struct {
	Operation parser.Operation // GRANT or REVOKE
	Grantee   string
	Privilege string
	Statement parser.Statement
}

// ObjectDependency provides enhanced dependency tracking
type ObjectDependency struct {
	ObjectID       ObjectID       `json:"object_id"`
	DependsOn      ObjectID       `json:"depends_on"`
	DependencyType DependencyType `json:"dependency_type"`
	IsRequired     bool           `json:"is_required"`
	Context        string         `json:"context,omitempty"`
}

// ObjectID provides comprehensive object identification
type ObjectID struct {
	Type   parser.ObjectType `json:"type"`
	Schema string            `json:"schema"`
	Name   string            `json:"name"`
}

// DependencyType represents different types of dependencies
type DependencyType string

const (
	DependencyTypeForeignKey  DependencyType = "FOREIGN_KEY"
	DependencyTypeFunction    DependencyType = "FUNCTION"
	DependencyTypeView        DependencyType = "VIEW"
	DependencyTypeType        DependencyType = "TYPE"
	DependencyTypeTrigger     DependencyType = "TRIGGER"
	DependencyTypeConstraint  DependencyType = "CONSTRAINT"
	DependencyTypeSequence    DependencyType = "SEQUENCE"
	DependencyTypeExtension   DependencyType = "EXTENSION"
	DependencyTypeCrossSchema DependencyType = "CROSS_SCHEMA"
)

// ===== ABSORBED TYPES FROM resource_tracker.go =====

// ResourceChangeType represents the type of resource change
type ResourceChangeType string

const (
	ResourceChangeCreate ResourceChangeType = "CREATE"
	ResourceChangeAlter  ResourceChangeType = "ALTER"
	ResourceChangeDrop   ResourceChangeType = "DROP"
	ResourceChangeRename ResourceChangeType = "RENAME"
	ResourceChangeMove   ResourceChangeType = "MOVE"
	ResourceChangeData   ResourceChangeType = "DATA"
	ResourceChangeGrant  ResourceChangeType = "GRANT"
	ResourceChangeRevoke ResourceChangeType = "REVOKE"
)

// ResourceType represents the type of database resource - use parser.ObjectType for consistency
type ResourceType = parser.ObjectType

// SourceRange represents the source location of a change
type SourceRange struct {
	Filename  string `json:"filename"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
	StartCol  int    `json:"start_col"`
	EndCol    int    `json:"end_col"`
	Text      string `json:"text"`
	Context   string `json:"context,omitempty"`
}

// ObjectInfo represents information about a database object
type ObjectInfo struct {
	Name       string                 `json:"name"`
	Schema     string                 `json:"schema"`
	Type       ResourceType           `json:"type"`
	Parent     *ObjectInfo            `json:"parent,omitempty"`
	Children   []*ObjectInfo          `json:"children,omitempty"`
	Properties map[string]interface{} `json:"properties,omitempty"`
}

// String returns a string representation of the object
func (oi *ObjectInfo) String() string {
	if oi.Schema != "" && oi.Schema != "public" {
		return fmt.Sprintf("%s.%s (%s)", oi.Schema, oi.Name, oi.Type)
	}
	return fmt.Sprintf("%s (%s)", oi.Name, oi.Type)
}

// FullName returns the fully qualified name
func (oi *ObjectInfo) FullName() string {
	if oi.Schema != "" {
		return fmt.Sprintf("%s.%s", oi.Schema, oi.Name)
	}
	return oi.Name
}

// ChangeContext provides context information for a resource change
type ChangeContext struct {
	Database     string            `json:"database"`
	SearchPath   []string          `json:"search_path"`
	Transaction  string            `json:"transaction,omitempty"`
	User         string            `json:"user,omitempty"`
	Application  string            `json:"application,omitempty"`
	Environment  string            `json:"environment,omitempty"`
	Tags         map[string]string `json:"tags,omitempty"`
	Reason       string            `json:"reason,omitempty"`
	TicketNumber string            `json:"ticket_number,omitempty"`
}

// ResourceChange represents a tracked change to a database resource
type ResourceChange struct {
	ID           string                 `json:"id"`
	Type         ResourceChangeType     `json:"type"`
	Object       *ObjectInfo            `json:"object"`
	Range        *SourceRange           `json:"range"`
	Context      *ChangeContext         `json:"context"`
	Dependencies []string               `json:"dependencies,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	Timestamp    time.Time              `json:"timestamp"`
	Migration    string                 `json:"migration"`
	Statement    *parser.Statement      `json:"statement,omitempty"`
}

// DatabaseMetadata contains metadata about the database structure
type DatabaseMetadata struct {
	Schemas    map[string]*SchemaInfo    `json:"schemas"`
	SearchPath []string                  `json:"search_path"`
	Version    string                    `json:"version"`
	Extensions map[string]*ExtensionInfo `json:"extensions"`
	Settings   map[string]string         `json:"settings"`
	Cache      *MetadataCache            `json:"-"`
}

// SchemaInfo contains information about a database schema
type SchemaInfo struct {
	Name        string                     `json:"name"`
	Owner       string                     `json:"owner"`
	Comment     string                     `json:"comment,omitempty"`
	Tables      map[string]*TableInfo      `json:"tables,omitempty"`
	Views       map[string]*ViewInfo       `json:"views,omitempty"`
	Functions   map[string]*FunctionInfo   `json:"functions,omitempty"`
	Indexes     map[string]*IndexInfo      `json:"indexes,omitempty"`
	Constraints map[string]*ConstraintInfo `json:"constraints,omitempty"`
	Sequences   map[string]*SequenceInfo   `json:"sequences,omitempty"`
	Types       map[string]*TypeInfo       `json:"types,omitempty"`
}

// TableInfo contains information about a database table
type TableInfo struct {
	Name        string                     `json:"name"`
	Schema      string                     `json:"schema"`
	Owner       string                     `json:"owner"`
	Comment     string                     `json:"comment,omitempty"`
	Tablespace  string                     `json:"tablespace,omitempty"`
	Columns     map[string]*ColumnInfo     `json:"columns"`
	Constraints map[string]*ConstraintInfo `json:"constraints"`
	Indexes     map[string]*IndexInfo      `json:"indexes"`
	Triggers    map[string]*TriggerInfo    `json:"triggers"`
	RowCount    int64                      `json:"row_count,omitempty"`
	Size        int64                      `json:"size,omitempty"`
}

// ColumnInfo contains information about a table column
type ColumnInfo struct {
	Name         string      `json:"name"`
	DataType     string      `json:"data_type"`
	IsNullable   bool        `json:"is_nullable"`
	DefaultValue *string     `json:"default_value,omitempty"`
	IsIdentity   bool        `json:"is_identity"`
	IsGenerated  bool        `json:"is_generated"`
	Comment      string      `json:"comment,omitempty"`
	Ordinal      int         `json:"ordinal"`
	Constraints  []string    `json:"constraints,omitempty"`
	Properties   interface{} `json:"properties,omitempty"`
}

// ViewInfo contains information about a database view
type ViewInfo struct {
	Name        string                 `json:"name"`
	Schema      string                 `json:"schema"`
	Owner       string                 `json:"owner"`
	Comment     string                 `json:"comment,omitempty"`
	Definition  string                 `json:"definition"`
	Columns     map[string]*ColumnInfo `json:"columns"`
	IsUpdatable bool                   `json:"is_updatable"`
}

// FunctionInfo contains information about a database function
type FunctionInfo struct {
	Name         string                 `json:"name"`
	Schema       string                 `json:"schema"`
	Owner        string                 `json:"owner"`
	Comment      string                 `json:"comment,omitempty"`
	Language     string                 `json:"language"`
	ReturnType   string                 `json:"return_type"`
	Parameters   []ParameterInfo        `json:"parameters"`
	Definition   string                 `json:"definition"`
	Volatility   string                 `json:"volatility"`
	IsStrict     bool                   `json:"is_strict"`
	IsSecDefiner bool                   `json:"is_sec_definer"`
	Properties   map[string]interface{} `json:"properties,omitempty"`
}

// ParameterInfo contains information about function parameters
type ParameterInfo struct {
	Name     string `json:"name"`
	DataType string `json:"data_type"`
	Mode     string `json:"mode"` // IN, OUT, INOUT
	Default  string `json:"default,omitempty"`
}

// IndexInfo contains information about a database index
type IndexInfo struct {
	Name       string   `json:"name"`
	Schema     string   `json:"schema"`
	Table      string   `json:"table"`
	Columns    []string `json:"columns"`
	IsUnique   bool     `json:"is_unique"`
	IsPrimary  bool     `json:"is_primary"`
	IsPartial  bool     `json:"is_partial"`
	Method     string   `json:"method"`
	Definition string   `json:"definition"`
	Size       int64    `json:"size,omitempty"`
}

// ConstraintInfo contains information about a database constraint
type ConstraintInfo struct {
	Name              string   `json:"name"`
	Schema            string   `json:"schema"`
	Table             string   `json:"table"`
	Type              string   `json:"type"`
	Columns           []string `json:"columns"`
	ReferencedTable   string   `json:"referenced_table,omitempty"`
	ReferencedColumns []string `json:"referenced_columns,omitempty"`
	Definition        string   `json:"definition"`
	IsDeferrable      bool     `json:"is_deferrable"`
	IsDeferred        bool     `json:"is_deferred"`
}

// SequenceInfo contains information about a database sequence
type SequenceInfo struct {
	Name        string `json:"name"`
	Schema      string `json:"schema"`
	Owner       string `json:"owner"`
	Comment     string `json:"comment,omitempty"`
	DataType    string `json:"data_type"`
	StartValue  int64  `json:"start_value"`
	MinValue    *int64 `json:"min_value,omitempty"`
	MaxValue    *int64 `json:"max_value,omitempty"`
	Increment   int64  `json:"increment"`
	CycleOption bool   `json:"cycle_option"`
	CacheSize   int64  `json:"cache_size"`
	LastValue   int64  `json:"last_value"`
}

// TypeInfo contains information about a user-defined type
type TypeInfo struct {
	Name       string                 `json:"name"`
	Schema     string                 `json:"schema"`
	Owner      string                 `json:"owner"`
	Comment    string                 `json:"comment,omitempty"`
	Type       string                 `json:"type"` // enum, composite, domain, etc.
	Definition string                 `json:"definition"`
	Properties map[string]interface{} `json:"properties,omitempty"`
}

// TriggerInfo contains information about a database trigger
type TriggerInfo struct {
	Name        string   `json:"name"`
	Schema      string   `json:"schema"`
	Table       string   `json:"table"`
	Events      []string `json:"events"`
	Timing      string   `json:"timing"`
	Orientation string   `json:"orientation"`
	Condition   string   `json:"condition,omitempty"`
	Definition  string   `json:"definition"`
	IsEnabled   bool     `json:"is_enabled"`
}

// ExtensionInfo contains information about a database extension
type ExtensionInfo struct {
	Name        string `json:"name"`
	Schema      string `json:"schema"`
	Version     string `json:"version"`
	Relocatable bool   `json:"relocatable"`
	Comment     string `json:"comment,omitempty"`
}

// MetadataCache provides caching for database metadata
type MetadataCache struct {
	LastUpdated time.Time     `json:"last_updated"`
	TTL         time.Duration `json:"ttl"`
	Enabled     bool          `json:"enabled"`
}

// ChangeTracker provides simple change tracking (compatibility with resource_tracker.go)
type ChangeTracker struct {
	changes         []*ResourceChange
	changesByObject map[string][]*ResourceChange
	context         *ChangeContext
	sequenceCounter int64
	metadata        *DatabaseMetadata
}

// NewChangeTracker creates a new change tracker
func NewChangeTracker(ctx *ChangeContext) *ChangeTracker {
	return &ChangeTracker{
		changes:         make([]*ResourceChange, 0),
		changesByObject: make(map[string][]*ResourceChange),
		context:         ctx,
		sequenceCounter: 0,
	}
}

// TrackStatement tracks a statement as a resource change
func (ct *ChangeTracker) TrackStatement(stmt *parser.Statement, migrationFile string) *ResourceChange {
	ct.sequenceCounter++

	change := &ResourceChange{
		ID:        fmt.Sprintf("%s-%d", migrationFile, ct.sequenceCounter),
		Type:      ct.getChangeType(stmt.Operation),
		Object:    ct.createObjectInfo(stmt),
		Range:     ct.createSourceRange(stmt, migrationFile),
		Context:   ct.context,
		Migration: migrationFile,
		Statement: stmt,
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	ct.changes = append(ct.changes, change)

	// Group by object for easy lookup
	objectKey := ct.getObjectKey(stmt)
	ct.changesByObject[objectKey] = append(ct.changesByObject[objectKey], change)

	return change
}

// GetChanges returns all tracked changes
func (ct *ChangeTracker) GetChanges() []*ResourceChange {
	return ct.changes
}

// GetChangesByObject returns changes for a specific object
func (ct *ChangeTracker) GetChangesByObject(objectKey string) []*ResourceChange {
	return ct.changesByObject[objectKey]
}

// Helper methods
func (ct *ChangeTracker) getChangeType(op parser.Operation) ResourceChangeType {
	switch op {
	case parser.OpCreate:
		return ResourceChangeCreate
	case parser.OpAlter:
		return ResourceChangeAlter
	case parser.OpDrop:
		return ResourceChangeDrop
	// Note: OpRename not defined in parser, handle as ALTER
	// case parser.OpRename:
	//	return ResourceChangeRename
	case parser.OpGrant:
		return ResourceChangeGrant
	case parser.OpRevoke:
		return ResourceChangeRevoke
	default:
		return ResourceChangeAlter
	}
}

func (ct *ChangeTracker) createObjectInfo(stmt *parser.Statement) *ObjectInfo {
	return &ObjectInfo{
		Name:   stmt.ObjectName,
		Schema: ct.resolveSchema(stmt.ObjectName),
		Type:   stmt.ObjectType,
	}
}

func (ct *ChangeTracker) createSourceRange(stmt *parser.Statement, filename string) *SourceRange {
	return &SourceRange{
		Filename:  filename,
		StartLine: stmt.Line,
		EndLine:   stmt.Line, // Single line for now
		Text:      stmt.SQL,
	}
}

func (ct *ChangeTracker) getObjectKey(stmt *parser.Statement) string {
	schema := ct.resolveSchema(stmt.ObjectName)
	return fmt.Sprintf("%s.%s.%s", schema, stmt.ObjectName, stmt.ObjectType)
}

func (ct *ChangeTracker) resolveSchema(objectName string) string {
	if strings.Contains(objectName, ".") {
		parts := strings.Split(objectName, ".")
		return parts[0]
	}
	return "public"
}

// RiskLevel represents the risk level of operations
type RiskLevel string

const (
	RiskLevelLow      RiskLevel = "LOW"
	RiskLevelMedium   RiskLevel = "MEDIUM"
	RiskLevelHigh     RiskLevel = "HIGH"
	RiskLevelCritical RiskLevel = "CRITICAL"
)

// OperationContext provides context for lifecycle events
type OperationContext struct {
	MigrationFile   string          `json:"migration_file"`
	LineNumber      int             `json:"line_number"`
	PreConditions   map[string]bool `json:"pre_conditions"`
	PostConditions  map[string]bool `json:"post_conditions"`
	ConflictsWith   []string        `json:"conflicts_with"`
	RequiredObjects []ObjectID      `json:"required_objects"`
}

// ObjectMetadata stores metadata about database objects
type ObjectMetadata struct {
	Source       string      `json:"source"` // Migration file source
	Description  string      `json:"description"`
	Tags         []string    `json:"tags"`
	DatabaseMeta interface{} `json:"database_meta"` // From metadata manager
}

// ConsolidationResult stores the result of object consolidation
type ConsolidationResult struct {
	OriginalStatements []parser.Statement `json:"original_statements"`
	ConsolidatedSQL    string             `json:"consolidated_sql"`
	Optimizations      []string           `json:"optimizations"`
	Warnings           []string           `json:"warnings"`
	RiskLevel          RiskLevel          `json:"risk_level"`
	EstimatedSavings   SquashSavings      `json:"estimated_savings"`
}

// DependencyGraph manages object dependencies with cycle detection
type DependencyGraph struct {
	nodes map[ObjectID]*DependencyNode
	edges map[ObjectID][]ObjectID
}

// DependencyNode represents a node in the dependency graph
type DependencyNode struct {
	ID           ObjectID
	Dependencies []ObjectID
	Dependents   []ObjectID
	Level        int  // Topological level
	InCycle      bool // Whether this node is part of a dependency cycle
}

// RiskAssessment evaluates risks for lifecycle operations
type RiskAssessment struct {
	rules []RiskRule
}

// RiskRule defines a rule for risk assessment
type RiskRule interface {
	Evaluate(event *LifecycleEvent, lifecycle *ObjectLifecycle) RiskLevel
	Description() string
}

// RedundancyReport provides analysis of redundant operations
type RedundancyReport struct {
	Object      string
	Type        parser.ObjectType
	Pattern     RedundancyPattern
	CanSquash   bool
	Explanation string
	Events      []LifecycleEvent
	Savings     SquashSavings
}

// RedundancyPattern identifies types of redundancy
type RedundancyPattern string

const (
	PatternCreateAlterSequence RedundancyPattern = "CREATE_ALTER_SEQUENCE"
	PatternDropCreateSequence  RedundancyPattern = "DROP_CREATE_SEQUENCE"
	PatternDuplicateOperations RedundancyPattern = "DUPLICATE_OPERATIONS"
	PatternUnusedObject        RedundancyPattern = "UNUSED_OBJECT"
	PatternDuplicateComments   RedundancyPattern = "DUPLICATE_COMMENTS"
	PatternRedundantDoBlocks   RedundancyPattern = "REDUNDANT_DO_BLOCKS"
	PatternDuplicateIndexes    RedundancyPattern = "DUPLICATE_INDEXES"
)

// SquashSavings quantifies potential savings from squashing
type SquashSavings struct {
	StatementsReduced int
	FilesAffected     int
	LinesReduced      int
}

// TrackerStats provides comprehensive tracking statistics
type TrackerStats struct {
	TotalObjects      int
	TotalMigrations   int
	TotalStatements   int
	DataOperations    int
	ResourceChanges   int
	ObjectsByType     map[parser.ObjectType]int
	ObjectsByCategory map[parser.Category]int
	ChangesByType     map[ResourceChangeType]int
}

// NewUnifiedTracker creates a comprehensive tracker with all features
func NewUnifiedTracker() *UnifiedTracker {
	// Configure cycle detection
	cycleConfig := CycleDetectionConfig{
		EnableDeepScan:         true,
		EnableDependencyTrack:  true,
		EnableVersionDetection: true,
		MaxCycleDepth:         10,
		MinCycleLength:        2,
	}

	return &UnifiedTracker{
		objects:         make(map[string]*ObjectLifecycle),
		dependencies:    make(map[string][]ObjectDependency),
		dependencyGraph: NewDependencyGraph(),
		riskAssessment:  NewRiskAssessment(),
		processedEvents: make(map[string]*LifecycleEvent),
		resourceChanges: make([]*ResourceChange, 0),
		changeTracker: NewChangeTracker(&ChangeContext{
			Database:   "",
			SearchPath: []string{"public"},
		}),
		cycleDetector:  NewAdvancedDDLCycleDetector(cycleConfig),
		detectedCycles: make([]DDLCycle, 0),
	}
}

// NewUnifiedTrackerWithMetadata creates a tracker with metadata manager integration
func NewUnifiedTrackerWithMetadata(metaMgr *metadata.MetadataManager) *UnifiedTracker {
	tracker := NewUnifiedTracker()
	tracker.metadataManager = metaMgr
	return tracker
}

// NewDependencyGraph creates a new dependency graph
func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		nodes: make(map[ObjectID]*DependencyNode),
		edges: make(map[ObjectID][]ObjectID),
	}
}

// NewRiskAssessment creates a new risk assessment with default rules
func NewRiskAssessment() *RiskAssessment {
	assessment := &RiskAssessment{
		rules: []RiskRule{},
	}

	// Add default risk rules
	assessment.AddRule(&DataLossRiskRule{})
	assessment.AddRule(&CircularDependencyRiskRule{})
	assessment.AddRule(&CrossSchemaRiskRule{})
	assessment.AddRule(&ProductionUsageRiskRule{})

	return assessment
}

// ProcessMigration processes a migration with comprehensive tracking
func (ut *UnifiedTracker) ProcessMigration(m *parser.Migration, sequence int) {
	if m == nil {
		return // Skip nil migrations
	}

	// Only store migrations in memory if not in streaming mode
	if !ut.streamingMode {
		ut.migrations = append(ut.migrations, m)
	}
	ut.migrationCounter++

	for stmtIndex, stmt := range m.Statements {
		// Process all statements, not just those with ObjectName
		// Some statements like INSERT/UPDATE/DELETE don't have ObjectName but should be tracked for data operations
		if stmt.ObjectName == "" && !stmt.IsDataOp {
			continue // Only skip if it's not a data operation and has no object name
		}

		// Create enhanced lifecycle event
		event := ut.createLifecycleEvent(stmt, m.Filename, sequence, stmtIndex)

		// Skip if already processed
		if _, processed := ut.processedEvents[event.ID]; processed {
			continue
		}

		// Handle statements with ObjectName (schema objects)
		if stmt.ObjectName != "" {
			key := makeKey(stmt.ObjectName, stmt.ObjectType)
			objectID := ObjectID{
				Type:   stmt.ObjectType,
				Schema: stmt.Schema,
				Name:   stmt.ObjectName,
			}

			lifecycle, exists := ut.objects[key]
			if !exists {
				lifecycle = ut.createObjectLifecycle(stmt, objectID)
				ut.objects[key] = lifecycle
			}

			// Add event to object lifecycle
			lifecycle.History = append(lifecycle.History, *event)

			// Debug: track profiles events
			if strings.ToLower(stmt.ObjectName) == "profiles" {
				log.Printf("DEBUG Tracker: Adding %s event for profiles (type=%s, key=%s) from %s, total events now: %d", stmt.Operation, stmt.ObjectType, key, m.Filename, len(lifecycle.History))
				if stmt.Operation == parser.OpCreate {
					log.Printf("  -> This is a CREATE operation (OpCreate=%v)", parser.OpCreate)
				}
			}

			// Process permissions if applicable
			if stmt.Operation == parser.OpGrant || stmt.Operation == parser.OpRevoke {
				ut.processPermissionEvent(stmt, lifecycle)
			}

			// Track dependencies
			ut.processDependencies(stmt, objectID, lifecycle)

			// Update dependency graph
			ut.dependencyGraph.AddNode(objectID)
		}

		// Resource change tracking (for all statements)
		resourceChange := ut.createResourceChange(stmt, m.Filename, sequence)
		if resourceChange != nil {
			ut.resourceChanges = append(ut.resourceChanges, resourceChange)
		}

		// Mark as processed
		ut.processedEvents[event.ID] = event
	}

	// Increment processed count for streaming mode progress tracking
	ut.processedCount++
}

// createLifecycleEvent creates an enhanced lifecycle event
func (ut *UnifiedTracker) createLifecycleEvent(stmt parser.Statement, migrationFile string, sequence int, stmtIndex int) *LifecycleEvent {
	eventID := fmt.Sprintf("%s:%d:%d:%s:%s", migrationFile, sequence, stmtIndex, stmt.ObjectType, stmt.ObjectName)

	return &LifecycleEvent{
		ID:           eventID,
		Migration:    migrationFile,
		Sequence:     sequence,
		Operation:    stmt.Operation,
		Statement:    stmt,
		Timestamp:    time.Now(),
		HasDataOps:   stmt.IsDataOp,
		Dependencies: stmt.Dependencies,
		Context: OperationContext{
			MigrationFile:   migrationFile,
			LineNumber:      stmt.Line,
			PreConditions:   make(map[string]bool),
			PostConditions:  make(map[string]bool),
			RequiredObjects: ut.extractRequiredObjects(stmt),
		},
		SourceRange: &SourceRange{
			Filename:  migrationFile,
			StartLine: stmt.Line,
			EndLine:   stmt.Line, // Single line for now
		},
	}
}

// createObjectLifecycle creates a new object lifecycle
func (ut *UnifiedTracker) createObjectLifecycle(stmt parser.Statement, objectID ObjectID) *ObjectLifecycle {
	key := makeKey(stmt.ObjectName, stmt.ObjectType)

	lifecycle := &ObjectLifecycle{
		Key:       key,
		Name:      stmt.ObjectName,
		Schema:    stmt.Schema,
		Type:      stmt.ObjectType,
		History:   []LifecycleEvent{},
		Category:  stmt.Category,
		CreatedAt: time.Now(),
		Metadata: ObjectMetadata{
			Source:      "migration",
			Description: extractDescription(stmt),
			Tags:        extractTags(stmt),
		},
		ResourceChanges: make([]*ResourceChange, 0),
	}

	// Enrich with database metadata if available
	if ut.metadataManager != nil {
		ctx := context.Background()
		if dbMeta, err := ut.metadataManager.GetMetadata(ctx, ""); err == nil {
			lifecycle.Metadata.DatabaseMeta = ut.extractDatabaseMetadata(dbMeta, objectID)
		}
	}

	return lifecycle
}

// createResourceChange creates a resource change from a statement
func (ut *UnifiedTracker) createResourceChange(stmt parser.Statement, migrationFile string, sequence int) *ResourceChange {
	changeType := ut.mapOperationToChangeType(stmt.Operation)
	resourceType := ut.mapObjectTypeToResourceType(stmt.ObjectType)

	return &ResourceChange{
		ID:   fmt.Sprintf("%s:%d:%s", migrationFile, sequence, stmt.ObjectName),
		Type: changeType,
		Object: &ObjectInfo{
			Name:   stmt.ObjectName,
			Schema: stmt.Schema,
			Type:   resourceType,
		},
		Range: &SourceRange{
			Filename:  migrationFile,
			StartLine: stmt.Line,
			EndLine:   stmt.Line,
		},
		Context: &ChangeContext{
			Database:   "",
			SearchPath: []string{"public"},
		},
		Metadata: map[string]interface{}{
			"migration": migrationFile,
			"sequence":  sequence,
			"sql":       stmt.SQL,
			"author":    "migration",
		},
		Timestamp: time.Now(),
		Migration: migrationFile,
		Statement: &stmt,
	}
}

// mapOperationToChangeType maps parser operations to resource change types
func (ut *UnifiedTracker) mapOperationToChangeType(op parser.Operation) ResourceChangeType {
	switch op {
	case parser.OpCreate:
		return ResourceChangeCreate
	case parser.OpAlter:
		return ResourceChangeAlter
	case parser.OpDrop:
		return ResourceChangeDrop
	case parser.OpGrant:
		return ResourceChangeGrant
	case parser.OpRevoke:
		return ResourceChangeRevoke
	default:
		return ResourceChangeAlter // Default fallback
	}
}

// mapObjectTypeToResourceType maps parser object types to resource types
func (ut *UnifiedTracker) mapObjectTypeToResourceType(objType parser.ObjectType) ResourceType {
	// Since ResourceType is now an alias for parser.ObjectType, just return it directly
	return ResourceType(objType)
}

// Helper methods for tracker functionality

// processPermissionEvent processes GRANT/REVOKE operations
func (ut *UnifiedTracker) processPermissionEvent(stmt parser.Statement, lifecycle *ObjectLifecycle) {
	for _, grantee := range stmt.Grantees {
		for _, privilege := range stmt.Privileges {
			permEvent := PermissionEvent{
				Operation: stmt.Operation,
				Grantee:   grantee,
				Privilege: privilege,
				Statement: stmt,
			}
			lifecycle.Permissions = append(lifecycle.Permissions, permEvent)
		}
	}
}

// processDependencies processes object dependencies with enhanced tracking
func (ut *UnifiedTracker) processDependencies(stmt parser.Statement, objectID ObjectID, lifecycle *ObjectLifecycle) {
	var dependencies []ObjectDependency

	for _, depName := range stmt.Dependencies {
		depType := ut.inferDependencyType(stmt, depName)
		dep := ObjectDependency{
			ObjectID:       objectID,
			DependsOn:      ut.parseObjectID(depName),
			DependencyType: depType,
			IsRequired:     ut.isRequiredDependency(stmt, depName),
			Context:        fmt.Sprintf("Referenced in %s operation", stmt.Operation),
		}
		dependencies = append(dependencies, dep)

		// Add to dependency graph
		ut.dependencyGraph.AddEdge(objectID, dep.DependsOn)
	}

	key := makeKey(stmt.ObjectName, stmt.ObjectType)
	ut.dependencies[key] = dependencies
}

// extractRequiredObjects extracts required objects from a statement
func (ut *UnifiedTracker) extractRequiredObjects(stmt parser.Statement) []ObjectID {
	var objects []ObjectID
	for _, dep := range stmt.Dependencies {
		objects = append(objects, ut.parseObjectID(dep))
	}
	return objects
}

// parseObjectID parses an object identifier string into ObjectID
func (ut *UnifiedTracker) parseObjectID(identifier string) ObjectID {
	// Handle prefixed identifiers (e.g., "REFERENCES:table_name", "CONSTRAINT:constraint_name")
	if strings.Contains(identifier, ":") {
		parts := strings.SplitN(identifier, ":", 2)
		if len(parts) == 2 {
			prefix := parts[0]
			actualIdentifier := parts[1]

			switch prefix {
			case "REFERENCES":
				// Foreign key reference - parse as table dependency
				log.Printf("Parsing foreign key reference dependency: %s", actualIdentifier)
				return ut.parseObjectID(actualIdentifier) // Recursive call without prefix
			case "CONSTRAINT":
				return ObjectID{Name: actualIdentifier, Type: parser.TypeConstraint}
			case "COLUMN":
				// Column dependencies should reference the table
				if strings.Contains(actualIdentifier, ".") {
					tableParts := strings.Split(actualIdentifier, ".")
					return ObjectID{Name: tableParts[0], Type: parser.TypeTable}
				}
				return ObjectID{Name: actualIdentifier, Type: parser.TypeTable} // Assume table reference
			default:
				// Fallback to parsing the actual identifier without prefix
				return ut.parseObjectID(actualIdentifier)
			}
		}
	}

	// Regular identifier parsing
	parts := strings.Split(identifier, ".")
	switch len(parts) {
	case 1:
		return ObjectID{Name: parts[0], Type: parser.TypeTable} // Default assumption
	case 2:
		return ObjectID{Schema: parts[0], Name: parts[1], Type: parser.TypeTable}
	case 3:
		return ObjectID{Schema: parts[0], Name: parts[1], Type: parser.ObjectType(parts[2])}
	default:
		return ObjectID{Name: identifier, Type: parser.TypeUnknown}
	}
}

// inferDependencyType infers the type of dependency from context
func (ut *UnifiedTracker) inferDependencyType(stmt parser.Statement, depName string) DependencyType {
	switch stmt.ObjectType {
	case parser.TypeTrigger:
		return DependencyTypeFunction
	case parser.TypeView:
		return DependencyTypeView
	case parser.TypeConstraint:
		if stmt.Operation == parser.OpCreate && strings.Contains(strings.ToLower(stmt.SQL), "foreign key") {
			return DependencyTypeForeignKey
		}
		return DependencyTypeConstraint
	default:
		if strings.Contains(depName, ".") {
			return DependencyTypeCrossSchema
		}
		return DependencyTypeFunction // Default assumption
	}
}

// isRequiredDependency checks if a dependency is required
func (ut *UnifiedTracker) isRequiredDependency(stmt parser.Statement, depName string) bool {
	// Foreign keys and triggers are typically required
	return stmt.ObjectType == parser.TypeConstraint || stmt.ObjectType == parser.TypeTrigger
}

// extractDescription extracts description from statement
func extractDescription(stmt parser.Statement) string {
	if len(stmt.Comments) > 0 {
		return strings.Join(stmt.Comments, " ")
	}
	return fmt.Sprintf("%s operation on %s", stmt.Operation, stmt.ObjectName)
}

// extractTags extracts tags from statement
func extractTags(stmt parser.Statement) []string {
	var tags []string

	if stmt.AuthPattern != parser.AuthPatternNone {
		tags = append(tags, "auth:"+string(stmt.AuthPattern))
	}
	if stmt.IsDynamic {
		tags = append(tags, "dynamic")
	}
	if stmt.IsDataOp {
		tags = append(tags, "data-operation")
	}

	return tags
}

// extractDatabaseMetadata extracts relevant metadata based on object type
func (ut *UnifiedTracker) extractDatabaseMetadata(dbMeta *metadata.DatabaseMetadata, objectID ObjectID) interface{} {
	switch objectID.Type {
	case parser.TypeTable:
		if schema, exists := dbMeta.Schemas[objectID.Schema]; exists {
			if table, exists := schema.Tables[objectID.Name]; exists {
				return table
			}
		}
	case parser.TypeFunction:
		if schema, exists := dbMeta.Schemas[objectID.Schema]; exists {
			if funcs, exists := schema.Functions[objectID.Name]; exists {
				return funcs
			}
		}
	}
	return nil
}

// makeKey creates a unique key for an object
func makeKey(name string, objType parser.ObjectType) string {
	return fmt.Sprintf("%s::%s", strings.ToLower(name), objType)
}

// String returns string representation of ObjectID
func (oid ObjectID) String() string {
	if oid.Schema != "" {
		return fmt.Sprintf("%s.%s.%s", oid.Type, oid.Schema, oid.Name)
	}
	return fmt.Sprintf("%s.%s", oid.Type, oid.Name)
}

// GetObjects returns all tracked objects
func (ut *UnifiedTracker) GetObjects() map[string]*ObjectLifecycle {
	return ut.objects
}

// GetDependencyGraph returns the dependency graph as a map for compatibility
func (ut *UnifiedTracker) GetDependencyGraph() map[string][]string {
	graph := make(map[string][]string)
	for key, deps := range ut.dependencies {
		stringDeps := make([]string, len(deps))
		for i, dep := range deps {
			stringDeps[i] = makeKey(dep.DependsOn.Name, dep.DependsOn.Type)
		}
		graph[key] = stringDeps
	}
	return graph
}

// GetActualDependencyGraph returns the actual dependency graph with full functionality
func (ut *UnifiedTracker) GetActualDependencyGraph() *DependencyGraph {
	return ut.dependencyGraph
}

// EnableStreamingMode enables streaming mode to reduce memory usage for large migration sets
func (ut *UnifiedTracker) EnableStreamingMode() {
	ut.streamingMode = true
}

// DisableStreamingMode disables streaming mode and restores full migration storage
func (ut *UnifiedTracker) DisableStreamingMode() {
	ut.streamingMode = false
}

// IsStreamingMode returns whether streaming mode is enabled
func (ut *UnifiedTracker) IsStreamingMode() bool {
	return ut.streamingMode
}

// GetProcessingStats returns processing statistics for streaming mode
func (ut *UnifiedTracker) GetProcessingStats() (processed, total int64) {
	return ut.processedCount, ut.migrationCounter
}

// ClearProcessedMigrations removes migration references to free memory in streaming mode
// This should only be called when all necessary analysis is complete
func (ut *UnifiedTracker) ClearProcessedMigrations() {
	if ut.streamingMode {
		ut.migrations = nil // Clear migration slice to free memory
	}
}

// DetectDDLCycles analyzes tracked objects for DDL cycles and stores results
func (ut *UnifiedTracker) DetectDDLCycles() error {
	if ut.cycleDetector == nil {
		return fmt.Errorf("DDL cycle detector not initialized")
	}

	// Run cycle detection on all tracked objects
	cycles, err := ut.cycleDetector.DetectCycles(ut.objects)
	if err != nil {
		return fmt.Errorf("failed to detect DDL cycles: %w", err)
	}

	ut.detectedCycles = cycles

	// Update risk assessment based on detected cycles
	for _, cycle := range cycles {
		for _, objectName := range cycle.Objects {
			if lifecycle, exists := ut.objects[objectName]; exists {
				switch cycle.Severity {
				case SeverityCritical:
					lifecycle.RiskLevel = RiskLevelHigh
				case SeverityHigh:
					lifecycle.RiskLevel = RiskLevelMedium
				case SeverityMedium:
					if lifecycle.RiskLevel == RiskLevelLow {
						lifecycle.RiskLevel = RiskLevelMedium
					}
				}

				// Mark objects involved in cycles as non-squashable if severity is high
				if cycle.Severity == SeverityCritical || cycle.Severity == SeverityHigh {
					lifecycle.CanBeSquashed = false
				}
			}
		}
	}

	return nil
}

// GetDetectedCycles returns all detected DDL cycles
func (ut *UnifiedTracker) GetDetectedCycles() []DDLCycle {
	return ut.detectedCycles
}

// HasDDLCycles returns true if any DDL cycles were detected
func (ut *UnifiedTracker) HasDDLCycles() bool {
	return len(ut.detectedCycles) > 0
}

// GetCriticalCycles returns only critical DDL cycles
func (ut *UnifiedTracker) GetCriticalCycles() []DDLCycle {
	var critical []DDLCycle
	for _, cycle := range ut.detectedCycles {
		if cycle.Severity == SeverityCritical {
			critical = append(critical, cycle)
		}
	}
	return critical
}

// IsObjectInCycle checks if an object is part of any DDL cycle
func (ut *UnifiedTracker) IsObjectInCycle(objectKey string) bool {
	for _, cycle := range ut.detectedCycles {
		for _, objectName := range cycle.Objects {
			if objectName == objectKey {
				return true
			}
		}
	}
	return false
}
