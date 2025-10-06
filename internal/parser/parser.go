package parser

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/capysquash/pg-squash-engine/internal/plugins"
	"github.com/capysquash/pg-squash-engine/internal/types"
	pg_query "github.com/pganalyze/pg_query_go/v6"
)

// Type aliases for backward compatibility
// All types now live in internal/types to avoid import cycles
type Migration = types.Migration
type Statement = types.Statement
type ObjectType = types.ObjectType
type Operation = types.Operation
type Category = types.Category
type CrossSchemaRef = types.CrossSchemaRef
type AuthPatternType = types.AuthPatternType

// Re-export constants from types package for backward compatibility
const (
	TypeTable           = types.TypeTable
	TypeIndex           = types.TypeIndex
	TypeFunction        = types.TypeFunction
	TypeTrigger         = types.TypeTrigger
	TypeView            = types.TypeView
	TypeSequence        = types.TypeSequence
	TypeConstraint      = types.TypeConstraint
	TypePolicy          = types.TypePolicy
	TypeRole            = types.TypeRole
	TypeSchema          = types.TypeSchema
	TypeExtension       = types.TypeExtension
	TypePublication     = types.TypePublication
	TypeComment         = types.TypeComment
	TypeDoBlock         = types.TypeDoBlock
	TypeType            = types.TypeType
	TypeDomain          = types.TypeDomain
	TypeEnum            = types.TypeEnum
	TypeComposite       = types.TypeComposite
	TypeSubscription    = types.TypeSubscription
	TypeStatistic       = types.TypeStatistic
	TypeGeneratedColumn = types.TypeGeneratedColumn
	TypeMultirangeType  = types.TypeMultirangeType
	TypeVectorIndex     = types.TypeVectorIndex
	TypeEventTrigger    = types.TypeEventTrigger
	TypeUnknown         = types.TypeUnknown

	AuthPatternNone       = types.AuthPatternNone
	AuthPatternSupabase   = types.AuthPatternSupabase
	AuthPatternClerk      = types.AuthPatternClerk
	AuthPatternClerkJWTV2 = types.AuthPatternClerkJWTV2
	AuthPatternAuth0      = types.AuthPatternAuth0
	AuthPatternNextAuth   = types.AuthPatternNextAuth
	AuthPatternFirebase   = types.AuthPatternFirebase
	AuthPatternRLS        = types.AuthPatternRLS
	AuthPatternStorage    = types.AuthPatternStorage
	AuthPatternCustomJWT  = types.AuthPatternCustomJWT

	OpCreate  = types.OpCreate
	OpAlter   = types.OpAlter
	OpDrop    = types.OpDrop
	OpInsert  = types.OpInsert
	OpUpdate  = types.OpUpdate
	OpDelete  = types.OpDelete
	OpGrant   = types.OpGrant
	OpRevoke  = types.OpRevoke
	OpComment = types.OpComment

	CategoryFoundation  = types.CategoryFoundation
	CategoryConstraints = types.CategoryConstraints
	CategoryIndexes     = types.CategoryIndexes
	CategoryFunctions   = types.CategoryFunctions
	CategoryTriggers    = types.CategoryTriggers
	CategorySecurity    = types.CategorySecurity
	CategoryData        = types.CategoryData
	CategoryExtensions  = types.CategoryExtensions
	CategoryCritical    = types.CategoryCritical
)

// ParseMigration parses a migration file with enhanced PostgreSQL-specific processing
func ParseMigration(content string, filename string) (*Migration, error) {
	ctx := context.Background()
	return ParseMigrationWithContext(ctx, content, filename)
}

// ParseMigrationWithContext parses a migration file with context and enhanced error handling
func ParseMigrationWithContext(ctx context.Context, content string, filename string) (*Migration, error) {
	// Initialize error handler
	errorHandler := NewErrorHandler(ctx)
	defer errorHandler.Recovery(filename, 0)

	// Initialize normalizer with PostgreSQL defaults
	normalizer := NewContextualNormalizer(DefaultNormalizationContext())

	// Clean and normalize SQL content
	cleanContent := cleanSQL(content)

	// Extract comments separately
	cleanContent, comments := extractComments(cleanContent)

	// Split into individual statements
	stmts, err := pg_query.SplitWithScanner(cleanContent, true)
	if err != nil {
		parseCtx := errorHandler.CreateContext(filename, 0, nil)
		errorHandler.HandleParseError(err, parseCtx)
		if !errorHandler.ShouldContinue() {
			return nil, fmt.Errorf("failed to split statements: %w", err)
		}
		// If we should continue, create empty stmts slice
		stmts = []string{}
	}

	migration := &Migration{
		Filename:   filename,
		Statements: make([]Statement, 0),
		Size:       int64(len(content)),
	}

	for i, stmtStr := range stmts {
		if strings.TrimSpace(stmtStr) == "" {
			continue
		}

		stmt, err := parseStatementWithNormalizationAndContext(stmtStr, i, normalizer, errorHandler, filename)
		if err != nil {
			// Error was already handled by parseStatementWithNormalizationAndContext
			if !errorHandler.ShouldContinue() {
				break
			}
			continue
		}

		// Assign relevant comments to statement
		stmt.Comments = getRelevantComments(comments, i)
		stmt.Category = categorizeStatement(*stmt)

		migration.Statements = append(migration.Statements, *stmt)
	}

	// Log summary of any errors or warnings
	errorHandler.LogSummary()

	// Return migration even if there were warnings
	if errorHandler.GetCollector().HasErrors() && !errorHandler.ShouldContinue() {
		return migration, fmt.Errorf("parsing failed with errors")
	}

	// Ensure migration is never nil when successful
	if migration == nil {
		migration = &Migration{
			Filename:   filename,
			Statements: make([]Statement, 0),
			Size:       int64(len(content)),
		}
	}

	return migration, nil
}

// parseStatementWithNormalization parses a statement with PostgreSQL-specific normalization
func parseStatementWithNormalization(sql string, line int, normalizer *ContextualNormalizer) (*Statement, error) {
	ctx := context.Background()
	errorHandler := NewErrorHandler(ctx)
	return parseStatementWithNormalizationAndContext(sql, line, normalizer, errorHandler, "")
}

// parseStatementWithNormalizationAndContext parses a statement with context and error handling
func parseStatementWithNormalizationAndContext(sql string, line int, normalizer *ContextualNormalizer, errorHandler *ErrorHandler, filename string) (*Statement, error) {
	defer errorHandler.Recovery(filename, line)

	parsed, err := pg_query.Parse(sql)
	if err != nil {
		parseCtx := errorHandler.CreateContext(filename, line, nil)
		parseCtx.StatementText = sql
		errorHandler.HandleParseError(err, parseCtx)
		return nil, err
	}

	if len(parsed.Stmts) == 0 {
		parseCtx := errorHandler.CreateContext(filename, line, nil)
		parseCtx.StatementText = sql
		err := fmt.Errorf("no statements found")
		errorHandler.HandleValidationError(err.Error(), parseCtx)
		return nil, err
	}

	stmt := &Statement{
		SQL:       strings.TrimSpace(sql),
		ParseTree: parsed,
		Line:      line,
	}

	// Analyze first statement with normalization
	rawStmt := parsed.Stmts[0]
	analyzeStatementWithNormalization(rawStmt, stmt, normalizer)

	// Check for IF NOT EXISTS for supported node types
	switch n := rawStmt.Stmt.Node.(type) {
	case *pg_query.Node_CreateStmt:
		stmt.IfNotExists = n.CreateStmt.IfNotExists
	case *pg_query.Node_IndexStmt:
		stmt.IfNotExists = n.IndexStmt.IfNotExists
	case *pg_query.Node_CreateExtensionStmt:
		stmt.IfNotExists = n.CreateExtensionStmt.IfNotExists
	}

	// Enhanced analysis for new features
	stmt.Schema = extractSchemaWithNormalization(stmt, normalizer)
	stmt.CrossSchema = extractCrossSchemaReferences(stmt)
	stmt.AuthPattern = detectAuthPattern(stmt)
	stmt.IsDynamic = isDynamicSQL(stmt)

	// Add naming convention warnings if applicable
	if stmt.ObjectName != "" {
		parseCtx := errorHandler.CreateContext(filename, line, stmt)
		validateNamingConventions(stmt, errorHandler.GetCollector(), parseCtx)
	}

	// Plugin enrichment: Allow plugins to add metadata to statements
	// This is called after pg_query parsing but before returning the statement
	// Plugins can enhance auth patterns, mark critical statements, add dependencies, etc.
	enrichStatementWithPlugins(context.Background(), stmt)

	return stmt, nil
}

// analyzeStatementWithNormalization analyzes statements with PostgreSQL-specific normalization
func analyzeStatementWithNormalization(raw *pg_query.RawStmt, stmt *Statement, normalizer *ContextualNormalizer) {
	switch node := raw.Stmt.Node.(type) {
	case *pg_query.Node_CreateStmt:
		stmt.ObjectType = TypeTable
		stmt.Operation = OpCreate
		stmt.ObjectName = getTableNameWithNormalization(node.CreateStmt.Relation, normalizer)
		stmt.Dependencies = extractTableDependenciesWithNormalization(node.CreateStmt, normalizer)

	case *pg_query.Node_AlterTableStmt:
		stmt.ObjectType = TypeTable
		stmt.Operation = OpAlter
		stmt.ObjectName = getTableNameWithNormalization(node.AlterTableStmt.Relation, normalizer)

		// Extract constraint information from ALTER TABLE commands
		stmt.Dependencies = extractAlterTableConstraints(node.AlterTableStmt, normalizer)

	case *pg_query.Node_DropStmt:
		stmt.Operation = OpDrop
		if len(node.DropStmt.Objects) > 0 {
			stmt.ObjectType = mapObjectType(node.DropStmt.RemoveType)
			if list := node.DropStmt.Objects[0]; list != nil {
				stmt.ObjectName = extractObjectNameWithNormalization(list, normalizer)
			}
		}

	case *pg_query.Node_IndexStmt:
		stmt.ObjectType = TypeIndex
		stmt.Operation = OpCreate
		stmt.ObjectName = normalizer.NormalizeIdentifier(node.IndexStmt.Idxname)
		if node.IndexStmt.Relation != nil {
			stmt.Dependencies = []string{getTableNameWithNormalization(node.IndexStmt.Relation, normalizer)}
		}

	case *pg_query.Node_CreateFunctionStmt:
		stmt.ObjectType = TypeFunction
		stmt.Operation = OpCreate
		if node.CreateFunctionStmt.Funcname != nil {
			stmt.ObjectName = extractFunctionNameWithNormalization(node.CreateFunctionStmt.Funcname, normalizer)
		}

	case *pg_query.Node_CreateTrigStmt:
		stmt.ObjectType = TypeTrigger
		stmt.Operation = OpCreate
		stmt.ObjectName = normalizer.NormalizeIdentifier(node.CreateTrigStmt.Trigname)
		if node.CreateTrigStmt.Relation != nil {
			stmt.Dependencies = []string{getTableNameWithNormalization(node.CreateTrigStmt.Relation, normalizer)}
		}

	case *pg_query.Node_ViewStmt:
		stmt.ObjectType = TypeView
		stmt.Operation = OpCreate
		if node.ViewStmt.View != nil {
			stmt.ObjectName = getTableNameWithNormalization(node.ViewStmt.View, normalizer)
		}

	case *pg_query.Node_CreatePolicyStmt:
		stmt.ObjectType = TypePolicy
		stmt.Operation = OpCreate
		stmt.ObjectName = normalizer.NormalizeIdentifier(node.CreatePolicyStmt.PolicyName)
		if node.CreatePolicyStmt.Table != nil {
			stmt.Dependencies = []string{getTableNameWithNormalization(node.CreatePolicyStmt.Table, normalizer)}
		}
		// Check for RESTRICTIVE policies
		if strings.Contains(stmt.SQL, "AS RESTRICTIVE") {
			stmt.Comments = append(stmt.Comments, "-- RESTRICTIVE policy")
		}

	case *pg_query.Node_InsertStmt:
		stmt.Operation = OpInsert
		stmt.IsDataOp = true
		stmt.ObjectName = getTableNameWithNormalization(node.InsertStmt.Relation, normalizer)
		// Check for ON CONFLICT clause
		if node.InsertStmt.OnConflictClause != nil {
			stmt.Comments = append(stmt.Comments, "-- Contains ON CONFLICT clause")
		}

	case *pg_query.Node_UpdateStmt:
		stmt.Operation = OpUpdate
		stmt.IsDataOp = true
		stmt.ObjectName = getTableNameWithNormalization(node.UpdateStmt.Relation, normalizer)

	case *pg_query.Node_DeleteStmt:
		stmt.Operation = OpDelete
		stmt.IsDataOp = true
		stmt.ObjectName = getTableNameWithNormalization(node.DeleteStmt.Relation, normalizer)

	case *pg_query.Node_GrantStmt:
		stmt.Operation = OpGrant
		stmt.ObjectType = mapGrantObjectType(node.GrantStmt.Objtype)
		if len(node.GrantStmt.Objects) > 0 {
			stmt.ObjectName = extractObjectNameWithNormalization(node.GrantStmt.Objects[0], normalizer)
		}
		stmt.Grantees = extractGranteesWithNormalization(node.GrantStmt.Grantees, normalizer)
		stmt.Privileges = extractPrivileges(node.GrantStmt.Privileges)

	case *pg_query.Node_GrantRoleStmt:
		if node.GrantRoleStmt.IsGrant {
			stmt.Operation = OpGrant
		} else {
			stmt.Operation = OpRevoke
		}
		stmt.ObjectType = TypeRole
		if len(node.GrantRoleStmt.GrantedRoles) > 0 {
			stmt.ObjectName = extractRoleNameWithNormalization(node.GrantRoleStmt.GrantedRoles[0], normalizer)
		}
		stmt.Grantees = extractGranteeRolesWithNormalization(node.GrantRoleStmt.GranteeRoles, normalizer)

	case *pg_query.Node_CreateExtensionStmt:
		stmt.ObjectType = TypeExtension
		stmt.Operation = OpCreate
		stmt.ObjectName = normalizer.NormalizeIdentifier(node.CreateExtensionStmt.Extname)

	case *pg_query.Node_CommentStmt:
		stmt.ObjectType = TypeUnknown // Will be determined by objtype
		stmt.Operation = OpComment
		// Extract object name and type from comment statement
		if node.CommentStmt.Object != nil {
			stmt.ObjectName = extractCommentObjectNameWithNormalization(node.CommentStmt, normalizer)
			stmt.ObjectType = mapCommentObjectType(node.CommentStmt.Objtype)
		}

	case *pg_query.Node_AlterPublicationStmt:
		stmt.ObjectType = TypePublication
		stmt.Operation = OpAlter
		stmt.ObjectName = normalizer.NormalizeIdentifier(node.AlterPublicationStmt.Pubname)

	case *pg_query.Node_DoStmt:
		// Check if this DO block contains CREATE TYPE statements
		if nestedTypes := extractNestedTypesFromDoBlock(stmt.SQL); len(nestedTypes) > 0 {
			// Treat DO blocks with CREATE TYPE as TYPE statements
			if len(nestedTypes) == 1 {
				stmt.ObjectType = TypeEnum
				stmt.Operation = OpCreate
				stmt.ObjectName = normalizer.NormalizeIdentifier(nestedTypes[0])
			} else {
				// Multiple types - treat as generic TYPE block
				stmt.ObjectType = TypeType
				stmt.Operation = OpCreate
				stmt.ObjectName = "multiple_types_block"
				stmt.Dependencies = nestedTypes
			}
		} else {
			// Regular DO block without types
			stmt.ObjectType = TypeDoBlock
			stmt.Operation = Operation("DO_BLOCK")
			stmt.ObjectName = "anonymous_block"
		}

	case *pg_query.Node_CreateEnumStmt:
		stmt.ObjectType = TypeEnum
		stmt.Operation = OpCreate
		if len(node.CreateEnumStmt.TypeName) > 0 {
			stmt.ObjectName = normalizer.NormalizeIdentifier(node.CreateEnumStmt.TypeName[len(node.CreateEnumStmt.TypeName)-1].GetString_().Sval)
		}

	case *pg_query.Node_CompositeTypeStmt:
		stmt.ObjectType = TypeComposite
		stmt.Operation = OpCreate
		if node.CompositeTypeStmt.Typevar != nil {
			stmt.ObjectName = normalizer.NormalizeIdentifier(node.CompositeTypeStmt.Typevar.Relname)
		}

	case *pg_query.Node_CreateDomainStmt:
		stmt.ObjectType = TypeDomain
		stmt.Operation = OpCreate
		if len(node.CreateDomainStmt.Domainname) > 0 {
			stmt.ObjectName = normalizer.NormalizeIdentifier(node.CreateDomainStmt.Domainname[len(node.CreateDomainStmt.Domainname)-1].GetString_().Sval)
		}

	default:
		stmt.ObjectType = TypeUnknown
		stmt.Operation = Operation("UNKNOWN")
	}
}

func categorizeStatement(stmt Statement) Category {
	if stmt.IsDataOp {
		return CategoryData
	}

	switch stmt.ObjectType {
	case TypeTable, TypeSequence, TypeType, TypeEnum, TypeComposite, TypeDomain:
		return CategoryFoundation
	case TypeConstraint:
		return CategoryConstraints
	case TypeIndex:
		return CategoryIndexes
	case TypeFunction, TypeDoBlock:
		return CategoryFunctions
	case TypeTrigger:
		return CategoryTriggers
	case TypePolicy, TypeRole, TypePublication:
		return CategorySecurity
	case TypeExtension:
		return CategoryExtensions
	case TypeComment:
		// Comments follow the object they're commenting on
		return CategoryFoundation // Default, could be enhanced
	default:
		return CategoryFoundation
	}
}

// getTableNameWithNormalization extracts and normalizes table names with schema awareness
func getTableNameWithNormalization(rangeVar *pg_query.RangeVar, normalizer *ContextualNormalizer) string {
	if rangeVar == nil {
		return ""
	}

	schema := normalizer.NormalizeSchemaName(rangeVar.Schemaname)
	table := normalizer.NormalizeIdentifier(rangeVar.Relname)

	if rangeVar.Schemaname != "" {
		return fmt.Sprintf("%s.%s", schema, table)
	}
	return table
}

// extractTableDependenciesWithNormalization extracts foreign key dependencies with normalization
func extractTableDependenciesWithNormalization(createStmt *pg_query.CreateStmt, normalizer *ContextualNormalizer) []string {
	var deps []string

	// Extract foreign key dependencies from table elements
	for _, tableElt := range createStmt.TableElts {
		// Check table-level constraints
		if constraint := tableElt.GetConstraint(); constraint != nil {
			if constraint.Contype == pg_query.ConstrType_CONSTR_FOREIGN {
				if constraint.Pktable != nil {
					refTableName := getTableNameWithNormalization(constraint.Pktable, normalizer)
					if refTableName != "" {
						deps = append(deps, fmt.Sprintf("REFERENCES:%s", refTableName))
					}
				}
			}
		}

		// Check column-level constraints (inline REFERENCES clauses)
		if columnDef := tableElt.GetColumnDef(); columnDef != nil {
			for _, colConstraint := range columnDef.Constraints {
				if constraint := colConstraint.GetConstraint(); constraint != nil {
					if constraint.Contype == pg_query.ConstrType_CONSTR_FOREIGN {
						if constraint.Pktable != nil {
							refTableName := getTableNameWithNormalization(constraint.Pktable, normalizer)
							if refTableName != "" {
								deps = append(deps, fmt.Sprintf("REFERENCES:%s", refTableName))
							}
						}
					}
				}
			}
		}
	}

	return deps
}

// extractAlterTableConstraints extracts constraint information from ALTER TABLE statements
func extractAlterTableConstraints(alterStmt *pg_query.AlterTableStmt, normalizer *ContextualNormalizer) []string {
	var constraints []string

	// Process each ALTER TABLE command
	for _, cmd := range alterStmt.Cmds {
		if alterCmd := cmd.GetAlterTableCmd(); alterCmd != nil {
			switch alterCmd.Subtype {
			case pg_query.AlterTableType_AT_AddConstraint:
				// Extract constraint name from ADD CONSTRAINT commands
				if constraint := alterCmd.Def.GetConstraint(); constraint != nil {
					if constraint.Conname != "" {
						constraintName := normalizer.NormalizeIdentifier(constraint.Conname)
						constraints = append(constraints, fmt.Sprintf("CONSTRAINT:%s", constraintName))
					}

					// Extract foreign key dependencies
					if constraint.Contype == pg_query.ConstrType_CONSTR_FOREIGN {
						if constraint.Pktable != nil {
							refTableName := getTableNameWithNormalization(constraint.Pktable, normalizer)
							if refTableName != "" {
								constraints = append(constraints, fmt.Sprintf("REFERENCES:%s", refTableName))
							}
						}
					}
				}
			case pg_query.AlterTableType_AT_AddColumn:
				// Track column additions for consolidation
				if columnDef := alterCmd.Def.GetColumnDef(); columnDef != nil {
					columnName := normalizer.NormalizeIdentifier(columnDef.Colname)
					constraints = append(constraints, fmt.Sprintf("COLUMN:%s", columnName))
				}
			}
		}
	}

	return constraints
}

// extractFunctionNameWithNormalization extracts function names with normalization
func extractFunctionNameWithNormalization(funcname []*pg_query.Node, normalizer *ContextualNormalizer) string {
	if len(funcname) == 0 {
		return ""
	}

	var parts []string
	for _, part := range funcname {
		if str := part.GetString_(); str != nil {
			parts = append(parts, normalizer.NormalizeIdentifier(str.Sval))
		}
	}

	return strings.Join(parts, ".")
}

// extractGranteesWithNormalization extracts grantees with normalization
func extractGranteesWithNormalization(grantees []*pg_query.Node, normalizer *ContextualNormalizer) []string {
	var names []string
	for _, grantee := range grantees {
		if roleSpec := grantee.GetRoleSpec(); roleSpec != nil {
			names = append(names, normalizer.NormalizeIdentifier(roleSpec.Rolename))
		}
	}
	return names
}

func extractPrivileges(privileges []*pg_query.Node) []string {
	var names []string
	for _, privilege := range privileges {
		if privilege.GetAccessPriv() != nil {
			if privilege.GetAccessPriv().PrivName != "" {
				names = append(names, privilege.GetAccessPriv().PrivName)
			} else {
				names = append(names, "ALL") // ALL PRIVILEGES
			}
		} else if str := privilege.GetString_(); str != nil {
			names = append(names, str.Sval)
		} else {
			names = append(names, "UNKNOWN_PRIVILEGE")
		}
	}
	if len(names) == 0 {
		names = append(names, "ALL") // Default to ALL if no specific privileges
	}
	return names
}

// extractRoleNameWithNormalization extracts role names with normalization
func extractRoleNameWithNormalization(role *pg_query.Node, normalizer *ContextualNormalizer) string {
	if roleSpec := role.GetRoleSpec(); roleSpec != nil {
		return normalizer.NormalizeIdentifier(roleSpec.Rolename)
	}
	if str := role.GetString_(); str != nil {
		return normalizer.NormalizeIdentifier(str.Sval)
	}
	return ""
}

// extractGranteeRolesWithNormalization extracts grantee roles with normalization
func extractGranteeRolesWithNormalization(granteeRoles []*pg_query.Node, normalizer *ContextualNormalizer) []string {
	var names []string
	for _, grantee := range granteeRoles {
		if name := extractRoleNameWithNormalization(grantee, normalizer); name != "" {
			names = append(names, name)
		}
	}
	return names
}

func mapGrantObjectType(objType pg_query.ObjectType) ObjectType {
	switch objType {
	case pg_query.ObjectType_OBJECT_TABLE:
		return TypeTable
	case pg_query.ObjectType_OBJECT_SEQUENCE:
		return TypeSequence
	case pg_query.ObjectType_OBJECT_SCHEMA:
		return TypeSchema
	case pg_query.ObjectType_OBJECT_FUNCTION:
		return TypeFunction
	default:
		return TypeUnknown
	}
}

func mapObjectType(removeType pg_query.ObjectType) ObjectType {
	switch removeType {
	case pg_query.ObjectType_OBJECT_TABLE:
		return TypeTable
	case pg_query.ObjectType_OBJECT_INDEX:
		return TypeIndex
	case pg_query.ObjectType_OBJECT_FUNCTION:
		return TypeFunction
	case pg_query.ObjectType_OBJECT_TRIGGER:
		return TypeTrigger
	case pg_query.ObjectType_OBJECT_VIEW:
		return TypeView
	case pg_query.ObjectType_OBJECT_SEQUENCE:
		return TypeSequence
	default:
		return TypeUnknown
	}
}

// extractObjectNameWithNormalization extracts object names with normalization
func extractObjectNameWithNormalization(obj *pg_query.Node, normalizer *ContextualNormalizer) string {
	if list := obj.GetList(); list != nil {
		var parts []string
		for _, item := range list.Items {
			if str := item.GetString_(); str != nil {
				parts = append(parts, normalizer.NormalizeIdentifier(str.Sval))
			}
		}
		return strings.Join(parts, ".")
	}
	return ""
}

func extractComments(content string) (string, []string) {
	lines := strings.Split(content, "\n")
	var cleanLines []string
	var comments []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "--") {
			comments = append(comments, trimmed)
		} else {
			cleanLines = append(cleanLines, line)
		}
	}

	return strings.Join(cleanLines, "\n"), comments
}

func getRelevantComments(comments []string, stmtIndex int) []string {
	// Simple heuristic: assign comments to nearby statements
	// This could be enhanced with more sophisticated logic
	var relevant []string

	if stmtIndex < len(comments) {
		relevant = append(relevant, comments[stmtIndex])
	}

	return relevant
}

// Helper functions for enhanced PostgreSQL parsing
// extractCommentObjectNameWithNormalization extracts comment object names with normalization
func extractCommentObjectNameWithNormalization(commentStmt *pg_query.CommentStmt, normalizer *ContextualNormalizer) string {
	// Extract object name from comment statement with normalization
	if commentStmt.Object != nil {
		return extractObjectNameWithNormalization(commentStmt.Object, normalizer)
	}
	return ""
}

func mapCommentObjectType(objtype pg_query.ObjectType) ObjectType {
	switch objtype {
	case pg_query.ObjectType_OBJECT_TABLE:
		return TypeTable
	case pg_query.ObjectType_OBJECT_COLUMN:
		return TypeTable // Column comments are table-related
	case pg_query.ObjectType_OBJECT_FUNCTION:
		return TypeFunction
	default:
		return TypeComment
	}
}

// Enhanced parsing for conditional DDL
func hasConditionalClause(sql string) bool {
	sqlUpper := strings.ToUpper(sql)
	return strings.Contains(sqlUpper, "IF NOT EXISTS") ||
		strings.Contains(sqlUpper, "IF EXISTS")
}

// Parse function bodies with dollar quoting
func extractFunctionBody(createFuncStmt *pg_query.CreateFunctionStmt) string {
	// This would need to handle dollar quoting properly
	// For now, return a placeholder
	return "function_body"
}

// Processing state to prevent duplicate analysis
type StatementProcessingState struct {
	StatementID      string
	ProcessedBy      []string
	FinalState       *Statement
	ConflictFlags    []string
	DependencySource string // Which function detected the dependencies
}

// extractSchemaWithNormalization extracts schema names with normalization
func extractSchemaWithNormalization(stmt *Statement, normalizer *ContextualNormalizer) string {
	sql := strings.ToLower(stmt.SQL)

	// Check for explicit schema qualification
	schemaPattern := regexp.MustCompile(`(?i)(storage|auth|public|extensions)\.`)
	if matches := schemaPattern.FindStringSubmatch(sql); len(matches) > 1 {
		return matches[1]
	}

	// Storage-specific patterns
	if strings.Contains(sql, "storage.buckets") || strings.Contains(sql, "storage.objects") {
		return "storage"
	}

	// Auth-specific patterns
	if strings.Contains(sql, "auth.jwt()") || strings.Contains(sql, "auth.uid()") {
		return "auth"
	}

	// Default to public schema
	return "public"
}

// extractFunctionDependencies extracts function dependencies from function definitions
func extractFunctionDependencies(funcStmt *pg_query.CreateFunctionStmt) []string {
	var deps []string

	// This would require parsing the function body
	// For now, return common dependencies based on function patterns
	if funcStmt != nil {
		// Common function dependencies could be extracted here
		// by analyzing function parameters, return types, etc.
	}

	return deps
}

// getTableName extracts table name without normalization (for compatibility)
func getTableName(rangeVar *pg_query.RangeVar) string {
	if rangeVar == nil {
		return ""
	}

	if rangeVar.Schemaname != "" {
		return fmt.Sprintf("%s.%s", rangeVar.Schemaname, rangeVar.Relname)
	}
	return rangeVar.Relname
}

// validateNamingConventions checks object names against PostgreSQL naming conventions
func validateNamingConventions(stmt *Statement, collector *ErrorCollector, ctx *ParseContext) {
	if stmt.ObjectName == "" {
		return
	}

	objectName := stmt.ObjectName

	// Check length limits (PostgreSQL limit is 63 characters)
	if len(objectName) > 63 {
		collector.AddNamingWarning(
			fmt.Sprintf("Object name '%s' exceeds PostgreSQL limit of 63 characters", objectName),
			"Consider shortening the name",
			ctx,
		)
	}

	// Check for reserved words
	keywordManager := NewVersionedKeywordManager(15) // PostgreSQL 15
	if keywordManager.IsReservedKeyword(objectName) {
		collector.AddNamingWarning(
			fmt.Sprintf("Object name '%s' is a PostgreSQL reserved keyword", objectName),
			"Consider using a different name or quoting the identifier",
			ctx,
		)
	}

	// Check naming conventions based on object type
	switch stmt.ObjectType {
	case TypeTable:
		validateTableNaming(objectName, collector, ctx)
	case TypeIndex:
		validateIndexNaming(objectName, collector, ctx)
	case TypeFunction:
		validateFunctionNaming(objectName, collector, ctx)
	case TypeConstraint:
		validateConstraintNaming(objectName, collector, ctx)
	case TypePolicy:
		validatePolicyNaming(objectName, collector, ctx)
	}
}

// validateTableNaming validates table naming conventions
func validateTableNaming(name string, collector *ErrorCollector, ctx *ParseContext) {
	// Prefer snake_case for table names
	if strings.Contains(name, " ") {
		collector.AddNamingWarning(
			fmt.Sprintf("Table name '%s' contains spaces", name),
			"Use snake_case naming convention",
			ctx,
		)
	}

	// Check for camelCase
	if name != strings.ToLower(name) && !strings.Contains(name, "_") {
		collector.AddNamingWarning(
			fmt.Sprintf("Table name '%s' appears to use camelCase", name),
			"PostgreSQL convention prefers snake_case for table names",
			ctx,
		)
	}
}

// validateIndexNaming validates index naming conventions
func validateIndexNaming(name string, collector *ErrorCollector, ctx *ParseContext) {
	// Index names should be descriptive
	commonBadPrefixes := []string{"idx_", "index_", "i_"}
	for _, prefix := range commonBadPrefixes {
		if strings.HasPrefix(strings.ToLower(name), prefix) {
			collector.AddNamingWarning(
				fmt.Sprintf("Index name '%s' uses generic prefix '%s'", name, prefix),
				"Consider more descriptive naming like 'tablename_column_idx'",
				ctx,
			)
			break
		}
	}
}

// validateFunctionNaming validates function naming conventions
func validateFunctionNaming(name string, collector *ErrorCollector, ctx *ParseContext) {
	// Function names should be descriptive verbs
	if len(name) < 3 {
		collector.AddNamingWarning(
			fmt.Sprintf("Function name '%s' is very short", name),
			"Use descriptive verb-based names for functions",
			ctx,
		)
	}
}

// validateConstraintNaming validates constraint naming conventions
func validateConstraintNaming(name string, collector *ErrorCollector, ctx *ParseContext) {
	// Constraint names should indicate their type
	constraintPrefixes := []string{"pk_", "fk_", "ck_", "uq_"}
	hasPrefix := false
	for _, prefix := range constraintPrefixes {
		if strings.HasPrefix(strings.ToLower(name), prefix) {
			hasPrefix = true
			break
		}
	}

	if !hasPrefix {
		collector.AddNamingWarning(
			fmt.Sprintf("Constraint name '%s' doesn't indicate its type", name),
			"Consider prefixes like pk_, fk_, ck_, uq_ for primary key, foreign key, check, unique constraints",
			ctx,
		)
	}
}

// validatePolicyNaming validates RLS policy naming conventions
func validatePolicyNaming(name string, collector *ErrorCollector, ctx *ParseContext) {
	// Policy names should indicate their purpose
	if !strings.Contains(strings.ToLower(name), "policy") &&
		!strings.Contains(strings.ToLower(name), "access") &&
		!strings.Contains(strings.ToLower(name), "security") {

		collector.AddNamingWarning(
			fmt.Sprintf("Policy name '%s' doesn't clearly indicate its security purpose", name),
			"Consider including words like 'policy', 'access', or 'security' in RLS policy names",
			ctx,
		)
	}
}

// extractCrossSchemaReferences finds references to objects in other schemas
func extractCrossSchemaReferences(stmt *Statement) []CrossSchemaRef {
	var refs []CrossSchemaRef
	sql := strings.ToLower(stmt.SQL)

	// Storage schema references
	if strings.Contains(sql, "storage.buckets") {
		refs = append(refs, CrossSchemaRef{
			Schema:     "storage",
			ObjectType: TypeTable,
			ObjectName: "buckets",
		})
	}

	if strings.Contains(sql, "storage.objects") {
		refs = append(refs, CrossSchemaRef{
			Schema:     "storage",
			ObjectType: TypeTable,
			ObjectName: "objects",
		})
	}

	// Auth schema references
	authFuncs := []string{"auth.jwt()", "auth.uid()", "auth.role()"}
	for _, authFunc := range authFuncs {
		if strings.Contains(sql, authFunc) {
			funcName := strings.TrimSuffix(strings.TrimPrefix(authFunc, "auth."), "()")
			refs = append(refs, CrossSchemaRef{
				Schema:     "auth",
				ObjectType: TypeFunction,
				ObjectName: funcName,
			})
		}
	}

	return refs
}

// detectAuthPattern identifies authentication and authorization patterns
func detectAuthPattern(stmt *Statement) AuthPatternType {
	sql := strings.ToLower(stmt.SQL)

	// JWT v2 patterns (Clerk with organization claims)
	jwtV2Patterns := []string{
		"auth.jwt()->'o'->",  // Organization claims
		"auth.jwt()->'v'",    // Version claim
		"auth.jwt()->'fva'",  // Factor verification age (MFA)
		"auth.jwt()->>'sub'", // Subject ID
		"clerk_user_id()",    // Clerk user functions
		"clerk_org_id()",
		"validate_jwt_version()",
	}

	for _, pattern := range jwtV2Patterns {
		if strings.Contains(sql, pattern) {
			return AuthPatternClerkJWTV2
		}
	}

	// Storage policies
	if strings.Contains(sql, "storage.objects") && strings.Contains(sql, "create policy") {
		return AuthPatternStorage
	}

	// RLS policies
	if strings.Contains(sql, "row level security") || strings.Contains(sql, "create policy") {
		return AuthPatternRLS
	}

	// Basic Clerk patterns
	clerkPatterns := []string{
		"clerk_is_admin()",
		"is_authenticated()",
		"handle_clerk_user(",
	}

	for _, pattern := range clerkPatterns {
		if strings.Contains(sql, pattern) {
			return AuthPatternClerk
		}
	}

	// Legacy Supabase patterns
	supabasePatterns := []string{
		"auth.uid()",
		"auth.role()",
		"current_setting('request.jwt.claims'",
	}

	for _, pattern := range supabasePatterns {
		if strings.Contains(sql, pattern) {
			return AuthPatternSupabase
		}
	}

	return AuthPatternNone
}

// isDynamicSQL detects dynamic SQL generation patterns
func isDynamicSQL(stmt *Statement) bool {
	sql := strings.ToLower(stmt.SQL)

	// Dynamic SQL patterns
	dynamicPatterns := []string{
		"execute format(",
		"execute concat(",
		"execute '",
		"$$ begin", // PL/pgSQL blocks that might contain dynamic SQL
		"foreach",  // Loop constructs
		"array[",   // Array-based dynamic generation
	}

	for _, pattern := range dynamicPatterns {
		if strings.Contains(sql, pattern) {
			return true
		}
	}

	// DO blocks with complex logic
	if stmt.ObjectType == TypeDoBlock && containsComplexLogic(sql) {
		return true
	}

	return false
}

// containsComplexLogic checks if a DO block contains complex dynamic logic
func containsComplexLogic(sql string) bool {
	complexPatterns := []string{
		"information_schema.tables", // Schema introspection
		"pg_proc",                   // Function introspection
		"pg_trigger",                // Trigger introspection
		"if exists (select",         // Conditional existence checks
		"foreach",                   // Iteration
		"execute format",            // Dynamic execution
		"raise notice",              // Logging/notifications
	}

	count := 0
	for _, pattern := range complexPatterns {
		if strings.Contains(sql, pattern) {
			count++
		}
	}

	// If multiple complex patterns are present, likely dynamic
	return count >= 2
}

// cleanSQL removes comments and normalizes whitespace
func cleanSQL(sql string) string {
	lines := strings.Split(sql, "\n")
	var cleanLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Only remove lines that start with comments
		if !strings.HasPrefix(trimmed, "--") && trimmed != "" {
			cleanLines = append(cleanLines, line)
		}
	}

	cleaned := strings.Join(cleanLines, "\n")

	// Remove multi-line comments
	multiCommentPattern := regexp.MustCompile(`/\*[\s\S]*?\*/`)
	cleaned = multiCommentPattern.ReplaceAllString(cleaned, "")

	// Normalize whitespace but preserve line breaks for SQL
	cleaned = strings.TrimSpace(cleaned)

	return cleaned
}

// extractNestedTypesFromDoBlock extracts CREATE TYPE statements from DO blocks
func extractNestedTypesFromDoBlock(doBlockSQL string) []string {
	var nestedTypes []string

	// Pattern to match CREATE TYPE ... AS ENUM statements within DO blocks
	enumPattern := regexp.MustCompile(`CREATE\s+TYPE\s+([a-zA-Z_][a-zA-Z0-9_]*)\s+AS\s+ENUM`)
	matches := enumPattern.FindAllStringSubmatch(doBlockSQL, -1)

	for _, match := range matches {
		if len(match) > 1 {
			nestedTypes = append(nestedTypes, match[1]) // Type name
		}
	}

	return nestedTypes
}

// enrichStatementWithPlugins calls all active plugins to enrich statement metadata
// This allows plugins to:
//   - Add auth pattern information (Clerk JWT v2, Supabase auth.uid())
//   - Mark critical statements (preserve auth functions, RLS policies)
//   - Add plugin-specific metadata for validation and consolidation
func enrichStatementWithPlugins(ctx context.Context, stmt *Statement) {
	registry := plugins.GlobalRegistry()

	// Only enrich if plugins are initialized
	if len(registry.ActivePlugins()) == 0 {
		return // No plugins active, skip enrichment
	}

	// Call enrichment on all active plugins (ordered by priority)
	if err := registry.EnrichStatement(ctx, stmt); err != nil {
		// Log error but don't fail parsing
		log.Printf("[parser] Plugin enrichment warning: %v", err)
	}
}
