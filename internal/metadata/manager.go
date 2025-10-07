// Package metadata provides database metadata extraction and management.
// It handles introspection of PostgreSQL databases, schema analysis,
// and metadata collection for migration squashing operations.
package metadata

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/capysquash/pg-squash-engine/internal/parser"
)

// MetadataManager provides comprehensive PostgreSQL metadata management
// with lazy-loading, caching, and search path resolution
type MetadataManager struct {
	cache      map[string]*DatabaseMetadata
	lastUpdate map[string]time.Time
	cacheTTL   time.Duration
	mutex      sync.RWMutex
	db         *sql.DB
	hitCount   int64
	missCount  int64
}

// DatabaseMetadata represents complete PostgreSQL database metadata
type DatabaseMetadata struct {
	Database   string                        `json:"database"`
	Schemas    map[string]*SchemaMetadata    `json:"schemas"`
	SearchPath []string                      `json:"search_path"`
	Extensions map[string]*ExtensionMetadata `json:"extensions"`
	IsSystemDB bool                          `json:"is_system_db"`
	Version    *PostgreSQLVersion            `json:"version"`
	CreatedAt  time.Time                     `json:"created_at"`
}

// SchemaMetadata represents a PostgreSQL schema with all its objects
type SchemaMetadata struct {
	Name              string                               `json:"name"`
	Tables            map[string]*TableMetadata            `json:"tables"`
	Views             map[string]*ViewMetadata             `json:"views"`
	MaterializedViews map[string]*MaterializedViewMetadata `json:"materialized_views"`
	Functions         map[string][]*FunctionMetadata       `json:"functions"` // Support overloading
	Sequences         map[string]*SequenceMetadata         `json:"sequences"`
	Types             map[string]*TypeMetadata             `json:"types"`
	Indexes           map[string]*IndexMetadata            `json:"indexes"`
	IsSystem          bool                                 `json:"is_system"`
}

// TableMetadata represents comprehensive table information
type TableMetadata struct {
	Name        string                `json:"name"`
	Schema      string                `json:"schema"`
	Columns     []*ColumnMetadata     `json:"columns"`
	Constraints []*ConstraintMetadata `json:"constraints"`
	Indexes     []*IndexMetadata      `json:"indexes"`
	Triggers    []*TriggerMetadata    `json:"triggers"`
	Policies    []*PolicyMetadata     `json:"policies"`
	Comment     string                `json:"comment,omitempty"`
	Tablespace  string                `json:"tablespace,omitempty"`
	RowSecurity bool                  `json:"row_security"`
}

// ColumnMetadata represents detailed column information
type ColumnMetadata struct {
	Name               string `json:"name"`
	DataType           string `json:"data_type"`
	IsNullable         bool   `json:"is_nullable"`
	DefaultValue       string `json:"default_value,omitempty"`
	IsGenerated        bool   `json:"is_generated"`
	GenerationExpr     string `json:"generation_expr,omitempty"`
	IsIdentity         bool   `json:"is_identity"`
	IdentityGeneration string `json:"identity_generation,omitempty"`
	Collation          string `json:"collation,omitempty"`
	Comment            string `json:"comment,omitempty"`
}

// ConstraintMetadata represents table constraints
type ConstraintMetadata struct {
	Name              string   `json:"name"`
	Type              string   `json:"type"` // PRIMARY KEY, FOREIGN KEY, UNIQUE, CHECK
	Columns           []string `json:"columns"`
	RefTable          string   `json:"ref_table,omitempty"`
	RefColumns        []string `json:"ref_columns,omitempty"`
	OnDelete          string   `json:"on_delete,omitempty"`
	OnUpdate          string   `json:"on_update,omitempty"`
	CheckExpression   string   `json:"check_expression,omitempty"`
	IsDeferrable      bool     `json:"is_deferrable"`
	InitiallyDeferred bool     `json:"initially_deferred"`
}

// IndexMetadata represents index information with parent table references
type IndexMetadata struct {
	Name          string         `json:"name"`
	Schema        string         `json:"schema"`
	Table         string         `json:"table"`
	Columns       []string       `json:"columns"`
	Expressions   []string       `json:"expressions,omitempty"`
	Method        string         `json:"method"` // BTREE, HASH, GIN, GIST, etc.
	IsUnique      bool           `json:"is_unique"`
	IsPrimary     bool           `json:"is_primary"`
	IsPartial     bool           `json:"is_partial"`
	WhereClause   string         `json:"where_clause,omitempty"`
	Definition    string         `json:"definition"`
	TableMetadata *TableMetadata `json:"-"` // Back-reference
}

// ViewMetadata represents view information
type ViewMetadata struct {
	Name         string   `json:"name"`
	Schema       string   `json:"schema"`
	Definition   string   `json:"definition"`
	Dependencies []string `json:"dependencies"`
	Comment      string   `json:"comment,omitempty"`
}

// MaterializedViewMetadata represents materialized view information
type MaterializedViewMetadata struct {
	*ViewMetadata
	HasData    bool     `json:"has_data"`
	Indexes    []string `json:"indexes"`
	Tablespace string   `json:"tablespace,omitempty"`
}

// FunctionMetadata represents function information with overloading support
type FunctionMetadata struct {
	Name       string       `json:"name"`
	Schema     string       `json:"schema"`
	Signature  string       `json:"signature"`
	Language   string       `json:"language"`
	ReturnType string       `json:"return_type"`
	Parameters []*Parameter `json:"parameters"`
	Body       string       `json:"body"`
	Volatility string       `json:"volatility"` // VOLATILE, STABLE, IMMUTABLE
	IsStrict   bool         `json:"is_strict"`
	Security   string       `json:"security"` // DEFINER, INVOKER
	Comment    string       `json:"comment,omitempty"`
}

// Parameter represents function parameter
type Parameter struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Mode    string `json:"mode"` // IN, OUT, INOUT
	Default string `json:"default,omitempty"`
}

// SequenceMetadata represents sequence information
type SequenceMetadata struct {
	Name      string `json:"name"`
	Schema    string `json:"schema"`
	DataType  string `json:"data_type"`
	Start     int64  `json:"start"`
	Increment int64  `json:"increment"`
	MinValue  int64  `json:"min_value"`
	MaxValue  int64  `json:"max_value"`
	Cache     int64  `json:"cache"`
	Cycle     bool   `json:"cycle"`
	OwnedBy   string `json:"owned_by,omitempty"`
}

// TypeMetadata represents custom type information
type TypeMetadata struct {
	Name       string   `json:"name"`
	Schema     string   `json:"schema"`
	Type       string   `json:"type"` // ENUM, COMPOSITE, DOMAIN, RANGE
	Definition string   `json:"definition"`
	Elements   []string `json:"elements,omitempty"` // For ENUM types
	Comment    string   `json:"comment,omitempty"`
}

// TriggerMetadata represents trigger information
type TriggerMetadata struct {
	Name      string   `json:"name"`
	Table     string   `json:"table"`
	Schema    string   `json:"schema"`
	Function  string   `json:"function"`
	Events    []string `json:"events"` // INSERT, UPDATE, DELETE
	Timing    string   `json:"timing"` // BEFORE, AFTER, INSTEAD OF
	Level     string   `json:"level"`  // ROW, STATEMENT
	Condition string   `json:"condition,omitempty"`
	IsEnabled bool     `json:"is_enabled"`
}

// PolicyMetadata represents RLS policy information
type PolicyMetadata struct {
	Name       string   `json:"name"`
	Table      string   `json:"table"`
	Schema     string   `json:"schema"`
	Command    string   `json:"command"`    // ALL, SELECT, INSERT, UPDATE, DELETE
	Permissive bool     `json:"permissive"` // PERMISSIVE or RESTRICTIVE
	Roles      []string `json:"roles"`
	Using      string   `json:"using,omitempty"`
	WithCheck  string   `json:"with_check,omitempty"`
}

// ExtensionMetadata represents extension information
type ExtensionMetadata struct {
	Name        string `json:"name"`
	Schema      string `json:"schema,omitempty"`
	Version     string `json:"version"`
	Comment     string `json:"comment,omitempty"`
	Relocatable bool   `json:"relocatable"`
}

// PostgreSQLVersion represents PostgreSQL version information
type PostgreSQLVersion struct {
	Major    int             `json:"major"`
	Minor    int             `json:"minor"`
	Patch    int             `json:"patch"`
	Features map[string]bool `json:"features"`
}

// NewMetadataManager creates a new metadata manager
func NewMetadataManager(db *sql.DB, cacheTTL time.Duration) *MetadataManager {
	if cacheTTL == 0 {
		cacheTTL = 15 * time.Minute
	}

	return &MetadataManager{
		cache:      make(map[string]*DatabaseMetadata),
		lastUpdate: make(map[string]time.Time),
		cacheTTL:   cacheTTL,
		db:         db,
	}
}

// GetMetadata retrieves database metadata with caching
func (m *MetadataManager) GetMetadata(ctx context.Context, database string) (*DatabaseMetadata, error) {
	m.mutex.RLock()
	if meta, exists := m.cache[database]; exists {
		if time.Since(m.lastUpdate[database]) < m.cacheTTL {
			m.mutex.RUnlock()
			m.hitCount++
			return meta, nil
		}
	}
	m.mutex.RUnlock()

	m.missCount++
	return m.loadAndCacheMetadata(ctx, database)
}

// loadAndCacheMetadata loads metadata from database and caches it
func (m *MetadataManager) loadAndCacheMetadata(ctx context.Context, database string) (*DatabaseMetadata, error) {
	meta, err := m.loadMetadataFromDB(ctx, database)
	if err != nil {
		return nil, err
	}

	m.mutex.Lock()
	m.cache[database] = meta
	m.lastUpdate[database] = time.Now()
	m.mutex.Unlock()

	return meta, nil
}

// InvalidateMetadata removes cached metadata
func (m *MetadataManager) InvalidateMetadata(database string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	delete(m.cache, database)
	delete(m.lastUpdate, database)
}

// GetCacheStats returns cache performance statistics
func (m *MetadataManager) GetCacheStats() (hits, misses int64, ratio float64) {
	total := m.hitCount + m.missCount
	if total == 0 {
		return 0, 0, 0.0
	}
	return m.hitCount, m.missCount, float64(m.hitCount) / float64(total)
}

// PostgreSQL-specific search path resolution
func (dbMeta *DatabaseMetadata) SearchObject(searchPath []string, objectName string) (schema, object string) {
	if len(searchPath) == 0 {
		searchPath = dbMeta.GetSearchPath()
	}

	for _, schemaName := range searchPath {
		schema, exists := dbMeta.Schemas[schemaName]
		if !exists {
			continue
		}

		// Check multiple object types in priority order
		if _, exists := schema.Tables[objectName]; exists {
			return schemaName, objectName
		}
		if _, exists := schema.Views[objectName]; exists {
			return schemaName, objectName
		}
		if _, exists := schema.MaterializedViews[objectName]; exists {
			return schemaName, objectName
		}
		if _, exists := schema.Functions[objectName]; exists {
			return schemaName, objectName
		}
		if _, exists := schema.Sequences[objectName]; exists {
			return schemaName, objectName
		}
		if _, exists := schema.Types[objectName]; exists {
			return schemaName, objectName
		}
	}
	return "", ""
}

// SearchTable searches for a table in the search path
func (dbMeta *DatabaseMetadata) SearchTable(searchPath []string, tableName string) (string, *TableMetadata) {
	if len(searchPath) == 0 {
		searchPath = dbMeta.GetSearchPath()
	}

	for _, schemaName := range searchPath {
		if schema, exists := dbMeta.Schemas[schemaName]; exists {
			if table, exists := schema.Tables[tableName]; exists {
				return schemaName, table
			}
		}
	}
	return "", nil
}

// SearchFunctions searches for functions supporting overloading
func (dbMeta *DatabaseMetadata) SearchFunctions(searchPath []string, funcName string) ([]string, []*FunctionMetadata) {
	if len(searchPath) == 0 {
		searchPath = dbMeta.GetSearchPath()
	}

	var schemas []string
	var functions []*FunctionMetadata

	for _, schemaName := range searchPath {
		if schema, exists := dbMeta.Schemas[schemaName]; exists {
			if funcs, exists := schema.Functions[funcName]; exists {
				for _, function := range funcs {
					schemas = append(schemas, schemaName)
					functions = append(functions, function)
				}
			}
		}
	}
	return schemas, functions
}

// GetSearchPath returns the database search path with proper defaults
func (dbMeta *DatabaseMetadata) GetSearchPath() []string {
	if len(dbMeta.SearchPath) == 0 {
		return []string{"public"} // PostgreSQL default
	}
	return dbMeta.SearchPath
}

// SetSearchPath sets the database search path with system path filtering
func (dbMeta *DatabaseMetadata) SetSearchPath(path []string) {
	var userPath []string
	for _, schema := range path {
		if !IsSystemSchema(schema) {
			userPath = append(userPath, schema)
		}
	}
	dbMeta.SearchPath = userPath
}

// IsSystemSchema checks if a schema is a PostgreSQL system schema
func IsSystemSchema(schemaName string) bool {
	systemSchemas := map[string]bool{
		"information_schema": true,
		"pg_catalog":         true,
		"pg_toast":           true,
	}

	schemaLower := strings.ToLower(schemaName)
	return systemSchemas[schemaLower] ||
		strings.HasPrefix(schemaLower, "pg_temp_") ||
		strings.HasPrefix(schemaLower, "pg_toast_")
}

// IsSystemTable checks if a table is a PostgreSQL system table
func IsSystemTable(tableName string) bool {
	tableLower := strings.ToLower(tableName)
	return strings.HasPrefix(tableLower, "pg_") ||
		strings.HasPrefix(tableLower, "sql_")
}

// IsSystemFunction checks if a function is a PostgreSQL system function
func IsSystemFunction(funcName string, schemaName string) bool {
	if schemaName != "" && !IsSystemSchema(schemaName) {
		return false
	}

	// Simplified check - would need comprehensive system function catalog
	funcLower := strings.ToLower(funcName)
	systemFunctions := map[string]bool{
		"current_user":      true,
		"current_timestamp": true,
		"now":               true,
		"version":           true,
		// Add more system functions as needed
	}

	return systemFunctions[funcLower]
}

// AnalyzeViewDependencies extracts dependencies from view definitions
func (dbMeta *DatabaseMetadata) AnalyzeViewDependencies(view *ViewMetadata) ([]string, error) {
	if view.Definition == "" {
		return nil, fmt.Errorf("view definition is empty")
	}

	// Parse view definition to extract table references
	// This would use the parser to analyze the SQL
	migration, err := parser.ParseMigration(view.Definition, "")
	if err != nil {
		return nil, fmt.Errorf("failed to analyze view %s: %w", view.Name, err)
	}

	var dependencies []string
	for _, stmt := range migration.Statements {
		for _, dep := range stmt.Dependencies {
			if !contains(dependencies, dep) {
				dependencies = append(dependencies, dep)
			}
		}
	}

	return dependencies, nil
}

// contains checks if a string slice contains a value
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// loadMetadataFromDB loads comprehensive metadata from PostgreSQL
func (m *MetadataManager) loadMetadataFromDB(ctx context.Context, database string) (*DatabaseMetadata, error) {
	meta := &DatabaseMetadata{
		Database:   database,
		Schemas:    make(map[string]*SchemaMetadata),
		Extensions: make(map[string]*ExtensionMetadata),
		CreatedAt:  time.Now(),
	}

	// Load PostgreSQL version
	version, err := m.loadVersion(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading PostgreSQL version: %w", err)
	}
	meta.Version = version

	// Load search path
	searchPath, err := m.loadSearchPath(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading search path: %w", err)
	}
	meta.SearchPath = searchPath

	// Load schemas
	schemas, err := m.loadSchemas(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading schemas: %w", err)
	}
	meta.Schemas = schemas

	// Load extensions
	extensions, err := m.loadExtensions(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading extensions: %w", err)
	}
	meta.Extensions = extensions

	return meta, nil
}

// loadVersion loads PostgreSQL version information
func (m *MetadataManager) loadVersion(ctx context.Context) (*PostgreSQLVersion, error) {
	var versionString string
	err := m.db.QueryRowContext(ctx, "SELECT version()").Scan(&versionString)
	if err != nil {
		return nil, err
	}

	// Parse version string - simplified implementation
	version := &PostgreSQLVersion{
		Features: make(map[string]bool),
	}

	// Extract major.minor from version string
	if strings.Contains(versionString, "PostgreSQL") {
		parts := strings.Fields(versionString)
		if len(parts) >= 2 {
			versionParts := strings.Split(parts[1], ".")
			if len(versionParts) >= 2 {
				_, _ = fmt.Sscanf(versionParts[0], "%d", &version.Major)
				_, _ = fmt.Sscanf(versionParts[1], "%d", &version.Minor)
				if len(versionParts) >= 3 {
					_, _ = fmt.Sscanf(versionParts[2], "%d", &version.Patch)
				}
			}
		}
	}

	// Set feature flags based on version
	if version.Major >= 14 {
		version.Features["multirange_types"] = true
	}
	if version.Major >= 15 {
		version.Features["merge_statements"] = true
		version.Features["enhanced_rls"] = true
	}

	return version, nil
}

// loadSearchPath loads the current search path
func (m *MetadataManager) loadSearchPath(ctx context.Context) ([]string, error) {
	var searchPathStr string
	err := m.db.QueryRowContext(ctx, "SHOW search_path").Scan(&searchPathStr)
	if err != nil {
		return []string{"public"}, nil // Return default on error
	}

	// Parse search path string
	paths := strings.Split(searchPathStr, ",")
	for i, path := range paths {
		paths[i] = strings.TrimSpace(path)
		// Handle "$user" placeholder
		if paths[i] == "$user" {
			var currentUser string
			if err := m.db.QueryRowContext(ctx, "SELECT current_user").Scan(&currentUser); err == nil {
				paths[i] = currentUser
			}
		}
	}

	return paths, nil
}

// loadSchemas loads all schema metadata
func (m *MetadataManager) loadSchemas(ctx context.Context) (map[string]*SchemaMetadata, error) {
	schemas := make(map[string]*SchemaMetadata)

	// Load schema names
	rows, err := m.db.QueryContext(ctx, `
		SELECT schema_name
		FROM information_schema.schemata
		WHERE schema_name NOT IN ('information_schema', 'pg_catalog', 'pg_toast')
		ORDER BY schema_name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var schemaName string
		if err := rows.Scan(&schemaName); err != nil {
			continue
		}

		schema := &SchemaMetadata{
			Name:              schemaName,
			Tables:            make(map[string]*TableMetadata),
			Views:             make(map[string]*ViewMetadata),
			MaterializedViews: make(map[string]*MaterializedViewMetadata),
			Functions:         make(map[string][]*FunctionMetadata),
			Sequences:         make(map[string]*SequenceMetadata),
			Types:             make(map[string]*TypeMetadata),
			Indexes:           make(map[string]*IndexMetadata),
			IsSystem:          IsSystemSchema(schemaName),
		}

		// Load tables for this schema
		tables, err := m.loadTablesForSchema(ctx, schemaName)
		if err != nil {
			return nil, fmt.Errorf("loading tables for schema %s: %w", schemaName, err)
		}
		schema.Tables = tables

		schemas[schemaName] = schema
	}

	return schemas, nil
}

// loadTablesForSchema loads table metadata for a specific schema
func (m *MetadataManager) loadTablesForSchema(ctx context.Context, schemaName string) (map[string]*TableMetadata, error) {
	tables := make(map[string]*TableMetadata)

	rows, err := m.db.QueryContext(ctx, `
		SELECT table_name, table_comment
		FROM information_schema.tables t
		LEFT JOIN (
			SELECT schemaname, tablename, description as table_comment
			FROM pg_tables pt
			JOIN pg_description pd ON pd.objoid = (
				SELECT oid FROM pg_class WHERE relname = pt.tablename
				AND relnamespace = (SELECT oid FROM pg_namespace WHERE nspname = pt.schemaname)
			) AND pd.objsubid = 0
		) tc ON tc.schemaname = t.table_schema AND tc.tablename = t.table_name
		WHERE table_schema = $1 AND table_type = 'BASE TABLE'
		ORDER BY table_name
	`, schemaName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var tableName string
		var comment sql.NullString
		if err := rows.Scan(&tableName, &comment); err != nil {
			continue
		}

		table := &TableMetadata{
			Name:        tableName,
			Schema:      schemaName,
			Columns:     []*ColumnMetadata{},
			Constraints: []*ConstraintMetadata{},
			Indexes:     []*IndexMetadata{},
			Triggers:    []*TriggerMetadata{},
			Policies:    []*PolicyMetadata{},
		}

		if comment.Valid {
			table.Comment = comment.String
		}

		tables[tableName] = table
	}

	return tables, nil
}

// loadExtensions loads extension metadata
func (m *MetadataManager) loadExtensions(ctx context.Context) (map[string]*ExtensionMetadata, error) {
	extensions := make(map[string]*ExtensionMetadata)

	rows, err := m.db.QueryContext(ctx, `
		SELECT extname, extversion, nspname, comment
		FROM pg_extension e
		LEFT JOIN pg_namespace n ON n.oid = e.extnamespace
		LEFT JOIN pg_description d ON d.objoid = e.oid AND d.objsubid = 0
		ORDER BY extname
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var name, version string
		var schema, comment sql.NullString
		if err := rows.Scan(&name, &version, &schema, &comment); err != nil {
			continue
		}

		ext := &ExtensionMetadata{
			Name:    name,
			Version: version,
		}

		if schema.Valid {
			ext.Schema = schema.String
		}
		if comment.Valid {
			ext.Comment = comment.String
		}

		extensions[name] = ext
	}

	return extensions, nil
}
