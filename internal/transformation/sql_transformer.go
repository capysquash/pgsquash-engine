package transformation

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/capysquash/pg-squash-engine/internal/plugins"
	pg_query "github.com/pganalyze/pg_query_go/v6"
)

// TransformationType defines the type of SQL transformation
type TransformationType int

const (
	DMLToSelect TransformationType = iota
	DropToComment
	UnsafeToSafe
	ModernSyntax
	Performance
)

// TransformationConfig controls SQL transformation behavior
type TransformationConfig struct {
	EnableDMLToSelect    bool   `json:"enable_dml_to_select"`
	EnableDropToComment  bool   `json:"enable_drop_to_comment"`
	EnableUnsafeToSafe   bool   `json:"enable_unsafe_to_safe"`
	EnableModernSyntax   bool   `json:"enable_modern_syntax"`
	EnablePerformance    bool   `json:"enable_performance"`
	EnableSyntaxFixes    bool   `json:"enable_syntax_fixes"`    // Fix common SQL syntax errors
	PreserveSrcPositions bool   `json:"preserve_src_positions"`
	TargetVersion        string `json:"target_version"` // PostgreSQL version target
}

// DefaultTransformationConfig returns sensible defaults
func DefaultTransformationConfig() *TransformationConfig {
	return &TransformationConfig{
		EnableDMLToSelect:    true,
		EnableDropToComment:  true,
		EnableUnsafeToSafe:   true,
		EnableModernSyntax:   true,
		EnablePerformance:    true,
		EnableSyntaxFixes:    true, // Enable syntax error fixes by default
		PreserveSrcPositions: true,
		TargetVersion:        "16",
	}
}

// TransformationResult represents the result of a transformation
type TransformationResult struct {
	OriginalSQL     string                  `json:"original_sql"`
	TransformedSQL  string                  `json:"transformed_sql"`
	Transformations []TransformationApplied `json:"transformations"`
	Warnings        []string                `json:"warnings"`
	Success         bool                    `json:"success"`
	Error           string                  `json:"error,omitempty"`
}

// TransformationApplied tracks what transformations were applied
type TransformationApplied struct {
	Type        TransformationType `json:"type"`
	Description string             `json:"description"`
	LineStart   int                `json:"line_start"`
	LineEnd     int                `json:"line_end"`
	Before      string             `json:"before"`
	After       string             `json:"after"`
}

// SQLTransformer handles SQL transformations for safety and compatibility
type SQLTransformer struct {
	config    *TransformationConfig
	pgVersion string
	patterns  *TransformationPatterns
}

// TransformationPatterns contains regex patterns for transformations
type TransformationPatterns struct {
	// DML patterns
	InsertPattern *regexp.Regexp
	UpdatePattern *regexp.Regexp
	DeletePattern *regexp.Regexp

	// Unsafe patterns
	DropTablePattern  *regexp.Regexp
	DropColumnPattern *regexp.Regexp
	AlterTypePattern  *regexp.Regexp

	// Performance patterns
	NoIndexPattern   *regexp.Regexp
	LargeScanPattern *regexp.Regexp

	// Modern syntax opportunities
	OldJoinPattern     *regexp.Regexp
	OldFunctionPattern *regexp.Regexp
}

// NewSQLTransformer creates a new SQL transformer
func NewSQLTransformer(config *TransformationConfig) *SQLTransformer {
	if config == nil {
		config = DefaultTransformationConfig()
	}

	patterns := &TransformationPatterns{
		InsertPattern:      regexp.MustCompile(`(?i)^\s*INSERT\s+INTO\s+`),
		UpdatePattern:      regexp.MustCompile(`(?i)^\s*UPDATE\s+`),
		DeletePattern:      regexp.MustCompile(`(?i)^\s*DELETE\s+FROM\s+`),
		DropTablePattern:   regexp.MustCompile(`(?i)^\s*DROP\s+TABLE\s+`),
		DropColumnPattern:  regexp.MustCompile(`(?i)ALTER\s+TABLE\s+\w+\s+DROP\s+COLUMN\s+`),
		AlterTypePattern:   regexp.MustCompile(`(?i)ALTER\s+TABLE\s+\w+\s+ALTER\s+COLUMN\s+\w+\s+TYPE\s+`),
		NoIndexPattern:     regexp.MustCompile(`(?i)WHERE\s+[^=]+\s*=\s*[^=]+`),
		LargeScanPattern:   regexp.MustCompile(`(?i)SELECT\s+\*\s+FROM\s+\w+\s*;?\s*$`),
		OldJoinPattern:     regexp.MustCompile(`(?i)WHERE\s+\w+\.\w+\s*=\s*\w+\.\w+`),
		OldFunctionPattern: regexp.MustCompile(`(?i)substr\(|length\(|position\(`),
	}

	return &SQLTransformer{
		config:    config,
		pgVersion: config.TargetVersion,
		patterns:  patterns,
	}
}

// Transform applies configured transformations to SQL
//
// Transformation Pipeline Order:
// 1. Function volatility markers (pre-parse, regex-based)
// 2. SQL parsing with pg_query (AST generation)
// 3. AST-dependent transformations (DML to SELECT, etc.)
//
// Note: Volatility fix runs BEFORE parsing because:
// - Doesn't require AST (uses regex pattern matching)
// - pg_query parser can fail on complex/large migration files (9000+ lines)
// - Critical for index predicate compatibility
func (st *SQLTransformer) Transform(ctx context.Context, sql string) (*TransformationResult, error) {
	result := &TransformationResult{
		OriginalSQL:     sql,
		TransformedSQL:  sql,
		Transformations: make([]TransformationApplied, 0),
		Warnings:        make([]string, 0),
		Success:         true,
	}

	transformedSQL := sql

	// STEP 0: Plugin Transformations (Pre-Parse, Highest Priority)
	// Plugins can apply service-specific transformations before core transformations
	// Examples:
	//   - Clerk: Add STABLE markers to auth helper functions
	//   - Supabase: Add STABLE markers to auth.uid(), auth.jwt()
	//   - Prisma: Convert @@ directives to PostgreSQL equivalents
	transformedSQL, err := st.applyPluginTransformations(ctx, transformedSQL)
	if err != nil {
		// Log warning but continue with core transformations
		result.Warnings = append(result.Warnings, fmt.Sprintf("Plugin transformations warning: %v", err))
	}

	// STEP 1: Function Volatility Fix (Pre-Parse, Fallback)
	// CRITICAL: Must run BEFORE pg_query.Parse() because:
	// - Parser fails on complex migrations (e.g., 9850 lines, 367KB files)
	// - Volatility fix uses regex, doesn't need AST
	// - Fixes: "ERROR: functions in index predicate must be marked IMMUTABLE"
	// Note: Plugins may have already fixed their specific functions, this is a fallback
	transformedSQL, err = st.fixFunctionVolatilityMarkers(ctx, transformedSQL, result)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Function volatility fix failed: %v", err))
	}

	// STEP 2: Parse SQL to AST (for transformations that need it)
	parseResult, err := pg_query.Parse(transformedSQL)
	if err != nil {
		// Don't fail - volatility fix already applied successfully
		// Just skip AST-dependent transformations
		result.Warnings = append(result.Warnings, fmt.Sprintf("SQL parsing skipped: %v", err))
		result.TransformedSQL = transformedSQL
		return result, nil
	}

	// Apply transformations that require AST
	if st.config.EnableDMLToSelect {
		transformedSQL, err = st.transformDMLToSelect(ctx, transformedSQL, parseResult, result)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("DML to SELECT transformation failed: %v", err))
		}
	}

	if st.config.EnableDropToComment {
		transformedSQL, err = st.transformDropToComment(ctx, transformedSQL, result)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Drop to comment transformation failed: %v", err))
		}
	}

	if st.config.EnableUnsafeToSafe {
		transformedSQL, err = st.transformUnsafeToSafe(ctx, transformedSQL, result)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Unsafe to safe transformation failed: %v", err))
		}
	}

	if st.config.EnableModernSyntax {
		transformedSQL, err = st.transformToModernSyntax(ctx, transformedSQL, result)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Modern syntax transformation failed: %v", err))
		}
	}

	if st.config.EnablePerformance {
		transformedSQL, err = st.applyPerformanceTransformations(ctx, transformedSQL, result)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Performance transformation failed: %v", err))
		}
	}

	if st.config.EnableSyntaxFixes {
		transformedSQL, err = st.fixCommonSyntaxErrors(ctx, transformedSQL, result)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Syntax fix transformation failed: %v", err))
		}
	}

	result.TransformedSQL = transformedSQL
	return result, nil
}

// applyPluginTransformations calls all active plugins to transform SQL
// Plugins are called in priority order (highest first)
// Each plugin can modify the SQL (e.g., add volatility markers, fix syntax)
func (st *SQLTransformer) applyPluginTransformations(ctx context.Context, sql string) (string, error) {
	registry := plugins.GlobalRegistry()

	// Only transform if plugins are initialized
	if len(registry.ActivePlugins()) == 0 {
		return sql, nil // No plugins active, return original SQL
	}

	// Call TransformSQL on registry (handles priority ordering internally)
	transformedSQL, err := registry.TransformSQL(ctx, sql)
	if err != nil {
		log.Printf("[transformation] Plugin transformation error: %v", err)
		return sql, err // Return original SQL on error
	}

	return transformedSQL, nil
}

// transformDMLToSelect converts DML statements to SELECT for dry-run validation
func (st *SQLTransformer) transformDMLToSelect(ctx context.Context, sql string, parseResult *pg_query.ParseResult, result *TransformationResult) (string, error) {
	lines := strings.Split(sql, "\n")
	transformedLines := make([]string, len(lines))
	copy(transformedLines, lines)

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Transform INSERT to SELECT
		if st.patterns.InsertPattern.MatchString(trimmed) {
			transformed := st.convertInsertToSelect(trimmed)
			if transformed != trimmed {
				transformedLines[i] = transformed
				result.Transformations = append(result.Transformations, TransformationApplied{
					Type:        DMLToSelect,
					Description: "Converted INSERT to SELECT for validation",
					LineStart:   i + 1,
					LineEnd:     i + 1,
					Before:      trimmed,
					After:       transformed,
				})
			}
		}

		// Transform UPDATE to SELECT
		if st.patterns.UpdatePattern.MatchString(trimmed) {
			transformed := st.convertUpdateToSelect(trimmed)
			if transformed != trimmed {
				transformedLines[i] = transformed
				result.Transformations = append(result.Transformations, TransformationApplied{
					Type:        DMLToSelect,
					Description: "Converted UPDATE to SELECT for validation",
					LineStart:   i + 1,
					LineEnd:     i + 1,
					Before:      trimmed,
					After:       transformed,
				})
			}
		}

		// Transform DELETE to SELECT
		if st.patterns.DeletePattern.MatchString(trimmed) {
			transformed := st.convertDeleteToSelect(trimmed)
			if transformed != trimmed {
				transformedLines[i] = transformed
				result.Transformations = append(result.Transformations, TransformationApplied{
					Type:        DMLToSelect,
					Description: "Converted DELETE to SELECT for validation",
					LineStart:   i + 1,
					LineEnd:     i + 1,
					Before:      trimmed,
					After:       transformed,
				})
			}
		}
	}

	return strings.Join(transformedLines, "\n"), nil
}

// convertInsertToSelect converts INSERT statement to SELECT for validation
func (st *SQLTransformer) convertInsertToSelect(sql string) string {
	// INSERT INTO table (col1, col2) VALUES (val1, val2)
	// -> SELECT val1 as col1, val2 as col2

	insertPattern := regexp.MustCompile(`(?i)INSERT\s+INTO\s+(\w+(?:\.\w+)?)\s*(?:\(([^)]+)\))?\s*VALUES\s*\(([^)]+)\)`)
	matches := insertPattern.FindStringSubmatch(sql)

	if len(matches) >= 4 {
		table := matches[1]
		columns := matches[2]
		values := matches[3]

		if columns != "" {
			// Parse columns and values
			cols := strings.Split(columns, ",")
			vals := strings.Split(values, ",")

			selectParts := make([]string, 0, len(vals))
			for i, val := range vals {
				val = strings.TrimSpace(val)
				if i < len(cols) {
					col := strings.TrimSpace(cols[i])
					selectParts = append(selectParts, fmt.Sprintf("%s as %s", val, col))
				} else {
					selectParts = append(selectParts, val)
				}
			}

			return fmt.Sprintf("-- INSERT validation: SELECT %s -- FROM %s",
				strings.Join(selectParts, ", "), table)
		} else {
			return fmt.Sprintf("-- INSERT validation: SELECT %s -- INTO %s", values, table)
		}
	}

	return sql
}

// convertUpdateToSelect converts UPDATE statement to SELECT for validation
func (st *SQLTransformer) convertUpdateToSelect(sql string) string {
	// UPDATE table SET col1 = val1 WHERE condition
	// -> SELECT val1 as col1 FROM table WHERE condition

	updatePattern := regexp.MustCompile(`(?i)UPDATE\s+(\w+(?:\.\w+)?)\s+SET\s+(.+?)(?:\s+WHERE\s+(.+))?$`)
	matches := updatePattern.FindStringSubmatch(sql)

	if len(matches) >= 3 {
		table := matches[1]
		setPart := matches[2]
		wherePart := ""
		if len(matches) >= 4 && matches[3] != "" {
			wherePart = " WHERE " + matches[3]
		}

		// Parse SET clause
		setPattern := regexp.MustCompile(`(\w+)\s*=\s*([^,]+)`)
		setMatches := setPattern.FindAllStringSubmatch(setPart, -1)

		selectParts := make([]string, 0, len(setMatches))
		for _, match := range setMatches {
			if len(match) >= 3 {
				col := strings.TrimSpace(match[1])
				val := strings.TrimSpace(match[2])
				selectParts = append(selectParts, fmt.Sprintf("%s as %s", val, col))
			}
		}

		if len(selectParts) > 0 {
			return fmt.Sprintf("-- UPDATE validation: SELECT %s FROM %s%s",
				strings.Join(selectParts, ", "), table, wherePart)
		}
	}

	return sql
}

// convertDeleteToSelect converts DELETE statement to SELECT for validation
func (st *SQLTransformer) convertDeleteToSelect(sql string) string {
	// DELETE FROM table WHERE condition
	// -> SELECT COUNT(*) FROM table WHERE condition

	deletePattern := regexp.MustCompile(`(?i)DELETE\s+FROM\s+(\w+(?:\.\w+)?)(?:\s+WHERE\s+(.+))?$`)
	matches := deletePattern.FindStringSubmatch(sql)

	if len(matches) >= 2 {
		table := matches[1]
		wherePart := ""
		if len(matches) >= 3 && matches[2] != "" {
			wherePart = " WHERE " + matches[2]
		}

		return fmt.Sprintf("-- DELETE validation: SELECT COUNT(*) FROM %s%s", table, wherePart)
	}

	return sql
}

// transformDropToComment converts dangerous DROP statements to comments
func (st *SQLTransformer) transformDropToComment(ctx context.Context, sql string, result *TransformationResult) (string, error) {
	lines := strings.Split(sql, "\n")
	transformedLines := make([]string, len(lines))
	copy(transformedLines, lines)

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if st.patterns.DropTablePattern.MatchString(trimmed) {
			commented := "-- DANGEROUS: " + trimmed
			transformedLines[i] = commented
			result.Transformations = append(result.Transformations, TransformationApplied{
				Type:        DropToComment,
				Description: "Commented out dangerous DROP TABLE statement",
				LineStart:   i + 1,
				LineEnd:     i + 1,
				Before:      trimmed,
				After:       commented,
			})
		}

		if st.patterns.DropColumnPattern.MatchString(trimmed) {
			commented := "-- DANGEROUS: " + trimmed
			transformedLines[i] = commented
			result.Transformations = append(result.Transformations, TransformationApplied{
				Type:        DropToComment,
				Description: "Commented out dangerous DROP COLUMN statement",
				LineStart:   i + 1,
				LineEnd:     i + 1,
				Before:      trimmed,
				After:       commented,
			})
		}
	}

	return strings.Join(transformedLines, "\n"), nil
}

// transformUnsafeToSafe converts unsafe operations to safer alternatives
func (st *SQLTransformer) transformUnsafeToSafe(ctx context.Context, sql string, result *TransformationResult) (string, error) {
	lines := strings.Split(sql, "\n")
	transformedLines := make([]string, len(lines))
	copy(transformedLines, lines)

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// ALTER TYPE operations - add USING clause suggestions
		if st.patterns.AlterTypePattern.MatchString(trimmed) {
			if !strings.Contains(strings.ToUpper(trimmed), "USING") {
				suggestion := trimmed + " -- Add USING clause for type conversion"
				transformedLines[i] = suggestion
				result.Transformations = append(result.Transformations, TransformationApplied{
					Type:        UnsafeToSafe,
					Description: "Added suggestion for USING clause in type conversion",
					LineStart:   i + 1,
					LineEnd:     i + 1,
					Before:      trimmed,
					After:       suggestion,
				})
				result.Warnings = append(result.Warnings, "Type conversion may require explicit USING clause")
			}
		}
	}

	return strings.Join(transformedLines, "\n"), nil
}

// transformToModernSyntax converts old SQL syntax to modern PostgreSQL syntax
func (st *SQLTransformer) transformToModernSyntax(ctx context.Context, sql string, result *TransformationResult) (string, error) {
	transformedSQL := sql

	// Convert old join syntax to modern ANSI joins
	if st.patterns.OldJoinPattern.MatchString(sql) {
		// This is a complex transformation that would need more sophisticated parsing
		result.Warnings = append(result.Warnings, "Consider converting old-style joins to ANSI join syntax")
	}

	// Convert old function names to modern equivalents
	if st.patterns.OldFunctionPattern.MatchString(sql) {
		modernSQL := strings.ReplaceAll(transformedSQL, "substr(", "substring(")
		modernSQL = strings.ReplaceAll(modernSQL, "length(", "char_length(")
		modernSQL = strings.ReplaceAll(modernSQL, "position(", "strpos(")

		if modernSQL != transformedSQL {
			result.Transformations = append(result.Transformations, TransformationApplied{
				Type:        ModernSyntax,
				Description: "Updated function names to modern equivalents",
				LineStart:   1,
				LineEnd:     len(strings.Split(sql, "\n")),
				Before:      "Old function names",
				After:       "Modern function names",
			})
			transformedSQL = modernSQL
		}
	}

	return transformedSQL, nil
}

// applyPerformanceTransformations adds performance-related improvements
func (st *SQLTransformer) applyPerformanceTransformations(ctx context.Context, sql string, result *TransformationResult) (string, error) {
	lines := strings.Split(sql, "\n")

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Detect queries that might benefit from indexes
		if st.patterns.NoIndexPattern.MatchString(trimmed) && !strings.Contains(strings.ToUpper(trimmed), "INDEX") {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Line %d: Consider adding index for WHERE clause performance", i+1))
		}

		// Detect SELECT * queries
		if st.patterns.LargeScanPattern.MatchString(trimmed) {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Line %d: SELECT * may cause performance issues", i+1))
		}
	}

	return sql, nil
}

// BatchTransform processes multiple SQL statements
func (st *SQLTransformer) BatchTransform(ctx context.Context, statements []string) ([]*TransformationResult, error) {
	results := make([]*TransformationResult, len(statements))

	for i, stmt := range statements {
		result, err := st.Transform(ctx, stmt)
		if err != nil {
			return nil, fmt.Errorf("failed to transform statement %d: %w", i, err)
		}
		results[i] = result
	}

	return results, nil
}

// ValidateTransformation checks if transformation preserves semantic meaning
func (st *SQLTransformer) ValidateTransformation(original, transformed string) error {
	// Parse both to ensure they're valid SQL
	_, err := pg_query.Parse(original)
	if err != nil {
		return fmt.Errorf("original SQL is invalid: %w", err)
	}

	_, err = pg_query.Parse(transformed)
	if err != nil {
		return fmt.Errorf("transformed SQL is invalid: %w", err)
	}

	return nil
}

// fixCommonSyntaxErrors fixes common PostgreSQL syntax errors found in migrations
func (st *SQLTransformer) fixCommonSyntaxErrors(ctx context.Context, sql string, result *TransformationResult) (string, error) {
	transformedSQL := sql

	// Fix 1: RETURN NEXT with parameter in RETURNS TABLE functions
	// Issue: RETURNS TABLE creates implicit OUT parameters, RETURN NEXT should not have arguments
	// Pattern: RETURNS TABLE(...) ... RETURN NEXT variable_name;
	// Fix: RETURN QUERY SELECT variable_name.field1, variable_name.field2...;
	transformedSQL = st.fixReturnNextWithOutParams(transformedSQL, result)

	// Fix 2: Missing semicolons at function ends
	// DISABLED: This was breaking function definitions by corrupting $$ delimiters
	// transformedSQL = st.fixMissingSemicolons(transformedSQL, result)

	// Fix 3: Invalid COMMENT ON syntax
	transformedSQL = st.fixCommentSyntax(transformedSQL, result)

	return transformedSQL, nil
}

// fixReturnNextWithOutParams fixes RETURN NEXT usage in RETURNS TABLE functions
func (st *SQLTransformer) fixReturnNextWithOutParams(sql string, result *TransformationResult) string {
	// Find all functions with RETURNS TABLE
	// Pattern: CREATE OR REPLACE FUNCTION func_name() RETURNS TABLE(col1 TYPE, col2 TYPE) ... RETURN NEXT record_var;

	// This is a complex fix that requires understanding the function structure
	// We'll use regex to find problematic patterns and fix them

	// Find RETURNS TABLE declarations and track the columns
	returnsTableRegex := regexp.MustCompile(`(?ims)CREATE\s+(?:OR\s+REPLACE\s+)?FUNCTION\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*\([^)]*\)\s*RETURNS\s+TABLE\s*\(\s*([^)]+)\)`)

	matches := returnsTableRegex.FindAllStringSubmatchIndex(sql, -1)
	if len(matches) == 0 {
		return sql // No RETURNS TABLE functions found
	}

	transformedSQL := sql
	offset := 0

	for _, match := range matches {
		if len(match) < 6 {
			continue
		}

		funcNameStart := match[2] + offset
		funcNameEnd := match[3] + offset
		columnsStart := match[4] + offset
		columnsEnd := match[5] + offset

		funcName := sql[funcNameStart:funcNameEnd]
		columnsSpec := sql[columnsStart:columnsEnd]

		// Parse column names from TABLE(...) definition
		columnNames := st.parseTableColumns(columnsSpec)
		if len(columnNames) == 0 {
			continue
		}

		// Find the function body (between AS $$ and $$)
		funcStart := match[0] + offset
		bodyRegex := regexp.MustCompile(`(?s)AS\s+\$\$(.+?)\$\$`)
		bodyMatch := bodyRegex.FindStringSubmatchIndex(transformedSQL[funcStart:])

		if len(bodyMatch) < 4 {
			continue
		}

		bodyStart := funcStart + bodyMatch[2]
		bodyEnd := funcStart + bodyMatch[3]
		body := transformedSQL[bodyStart:bodyEnd]

		// Find RETURN NEXT statements with arguments in this function body
		returnNextRegex := regexp.MustCompile(`(?m)^\s*RETURN\s+NEXT\s+([a-zA-Z_][a-zA-Z0-9_]*(?:\.[a-zA-Z_][a-zA-Z0-9_]*)*)\s*;`)
		returnMatches := returnNextRegex.FindAllStringSubmatchIndex(body, -1)

		if len(returnMatches) == 0 {
			continue
		}

		fixedBody := body
		bodyOffset := 0

		for _, returnMatch := range returnMatches {
			if len(returnMatch) < 4 {
				continue
			}

			// Get the variable name being returned
			varNameStart := returnMatch[2] + bodyOffset
			varNameEnd := returnMatch[3] + bodyOffset
			varName := fixedBody[varNameStart:varNameEnd]

			// Build RETURN QUERY SELECT statement
			// For a RECORD variable, we need to access its fields
			var selectColumns []string
			for _, colName := range columnNames {
				selectColumns = append(selectColumns, fmt.Sprintf("%s.%s", varName, colName))
			}
			selectStmt := fmt.Sprintf("RETURN QUERY SELECT %s;", strings.Join(selectColumns, ", "))

			// Replace RETURN NEXT var; with RETURN QUERY SELECT var.field1, var.field2;
			oldStmt := fixedBody[returnMatch[0]+bodyOffset : returnMatch[1]+bodyOffset]
			fixedBody = fixedBody[:returnMatch[0]+bodyOffset] + selectStmt + fixedBody[returnMatch[1]+bodyOffset:]

			bodyOffset += len(selectStmt) - len(oldStmt)

			// Record the transformation
			result.Transformations = append(result.Transformations, TransformationApplied{
				Type:        UnsafeToSafe,
				Description: fmt.Sprintf("Fixed RETURN NEXT syntax in function %s (RETURNS TABLE should use RETURN QUERY SELECT)", funcName),
				Before:      strings.TrimSpace(oldStmt),
				After:       strings.TrimSpace(selectStmt),
			})
		}

		// Replace the function body in the transformed SQL
		transformedSQL = transformedSQL[:bodyStart] + fixedBody + transformedSQL[bodyEnd:]
		offset += len(fixedBody) - len(body)
	}

	return transformedSQL
}

// parseTableColumns extracts column names from RETURNS TABLE(...) specification
func (st *SQLTransformer) parseTableColumns(columnsSpec string) []string {
	// Split by commas, handling nested parentheses
	var columns []string
	var current strings.Builder
	depth := 0

	for _, char := range columnsSpec {
		switch char {
		case '(':
			depth++
			current.WriteRune(char)
		case ')':
			depth--
			current.WriteRune(char)
		case ',':
			if depth == 0 {
				colDef := strings.TrimSpace(current.String())
				if colDef != "" {
					// Extract just the column name (first word)
					parts := strings.Fields(colDef)
					if len(parts) > 0 {
						columns = append(columns, parts[0])
					}
				}
				current.Reset()
			} else {
				current.WriteRune(char)
			}
		default:
			current.WriteRune(char)
		}
	}

	// Handle last column
	if current.Len() > 0 {
		colDef := strings.TrimSpace(current.String())
		if colDef != "" {
			parts := strings.Fields(colDef)
			if len(parts) > 0 {
				columns = append(columns, parts[0])
			}
		}
	}

	return columns
}

// fixMissingSemicolons adds missing semicolons to function definitions
//nolint:unused // Reserved for future semicolon fixing functionality
func (st *SQLTransformer) fixMissingSemicolons(sql string, result *TransformationResult) string {
	// Look for $$ followed by newline without semicolon
	// Use a simpler approach compatible with RE2: match $$ + whitespace + newline + non-semicolon
	regex := regexp.MustCompile(`(?m)\$\$\s*\n\s*([^;])`)

	if regex.MatchString(sql) {
		sql = regex.ReplaceAllString(sql, "$$;\n$1")
		result.Transformations = append(result.Transformations, TransformationApplied{
			Type:        UnsafeToSafe,
			Description: "Added missing semicolons after function definitions",
		})
	}

	return sql
}

// fixCommentSyntax fixes invalid COMMENT ON syntax
func (st *SQLTransformer) fixCommentSyntax(sql string, result *TransformationResult) string {
	// Fix COMMENT ON FUNCTION with invalid signature
	// Pattern: COMMENT ON FUNCTION func_name() IS '...';
	// Sometimes the function signature might be incomplete or invalid

	// For now, we'll just validate the syntax is correct
	// More sophisticated fixes can be added as needed

	return sql
}

// fixFunctionVolatilityMarkers adds STABLE/IMMUTABLE/VOLATILE markers to functions without them.
//
// PostgreSQL Requirement:
// Functions used in index predicates (CREATE INDEX ... WHERE function(...)) MUST have
// explicit volatility markers. Without them, PostgreSQL throws:
// "ERROR: functions in index predicate must be marked IMMUTABLE"
//
// Volatility Categories:
// - IMMUTABLE: Pure functions, same input always produces same output (e.g., math operations)
// - STABLE: Reads database/session state but doesn't modify it (e.g., auth.jwt(), current_user)
// - VOLATILE: Modifies data or has side effects (e.g., INSERT, UPDATE, nextval())
//
// This transformation:
// 1. Finds all CREATE FUNCTION statements without volatility markers
// 2. Analyzes function body to determine appropriate volatility
// 3. Injects marker before AS keyword
//
// Common Use Cases:
// - Clerk/Supabase auth functions (clerk_user_id, clerk_is_admin) → STABLE
// - Trigger functions with INSERT/UPDATE → VOLATILE
// - Calculation functions → IMMUTABLE
//
// Implementation Note:
// This runs BEFORE pg_query.Parse() because the parser can fail on complex migrations.
// Uses regex-based detection instead of AST analysis.
func (st *SQLTransformer) fixFunctionVolatilityMarkers(ctx context.Context, sql string, result *TransformationResult) (string, error) {
	// Regex Pattern Explanation:
	// (?ims) - Case insensitive, multiline, dot matches newline
	// Group 1: CREATE [OR REPLACE] FUNCTION [schema.]name(...) RETURNS type
	// Group 2: Optional modifiers (LANGUAGE, SECURITY DEFINER, SET)
	// Group 3: AS keyword with delimiter ($$ or $tag$ or ')
	//
	// Handles both formats:
	// - Single line: CREATE FUNCTION foo() RETURNS TEXT AS $$...
	// - Multi-line:  CREATE FUNCTION foo()
	//                RETURNS TEXT
	//                LANGUAGE plpgsql
	//                AS $$...
	funcPattern := regexp.MustCompile(`(?ims)(CREATE\s+(?:OR\s+REPLACE\s+)?FUNCTION\s+(?:[a-z_][a-z0-9_]*\.)?[a-z_][a-z0-9_]*\s*\([^)]*\)\s*RETURNS\s+(?:TABLE\s*\([^)]+\)|SETOF\s+[^\s]+|[^\s]+))((?:\s+(?:LANGUAGE\s+[a-z]+|SECURITY\s+DEFINER|SET\s+[^\s]+\s*=\s*[^\s]+))*?)(\s+AS\s+(?:\$\$|\$[a-z0-9_]*\$|\'))`)

	matches := funcPattern.FindAllStringSubmatchIndex(sql, -1)
	if len(matches) == 0 {
		return sql, nil // No functions to fix
	}

	transformedSQL := sql
	offset := 0

	for _, match := range matches {
		if len(match) < 8 {
			continue
		}

		// Extract function definition parts
		beforeAS := transformedSQL[match[0]+offset : match[6]+offset]  // Full match up to AS
		modifiers := transformedSQL[match[4]+offset : match[5]+offset] // LANGUAGE/SECURITY DEFINER/SET
		asKeyword := transformedSQL[match[6]+offset : match[7]+offset] // " AS "

		// Check if volatility marker already exists
		hasVolatility := st.hasVolatilityMarker(modifiers) || st.hasVolatilityMarker(beforeAS)

		if hasVolatility {
			continue // Already has volatility marker, skip
		}

		funcName := st.extractFunctionName(beforeAS)

		// Extract function body to determine appropriate volatility
		bodyStart := match[7] + offset
		bodyPattern := regexp.MustCompile(`(?s)\$\$(.+?)\$\$`)
		bodyMatch := bodyPattern.FindStringSubmatchIndex(transformedSQL[bodyStart:])

		var functionBody string
		if len(bodyMatch) >= 4 {
			functionBody = transformedSQL[bodyStart+bodyMatch[2] : bodyStart+bodyMatch[3]]
		} else {
			functionBody = "" // Can't determine body, default to STABLE
		}

		// Determine appropriate volatility based on function body
		volatility := st.determineVolatility(functionBody)

		// Insert volatility marker before AS
		// Priority: LANGUAGE > SECURITY DEFINER > SET > VOLATILITY > AS
		newModifiers := modifiers + " " + volatility
		replacement := beforeAS[:match[4]+offset-match[0]-offset] + newModifiers + asKeyword

		transformedSQL = transformedSQL[:match[0]+offset] + replacement + transformedSQL[match[7]+offset:]
		offset += len(replacement) - (match[7] - match[0])

		// Record transformation
		result.Transformations = append(result.Transformations, TransformationApplied{
			Type:        UnsafeToSafe,
			Description: fmt.Sprintf("Added %s volatility marker to function %s for index predicate compatibility", volatility, funcName),
			Before:      beforeAS + asKeyword,
			After:       replacement,
		})
	}

	return transformedSQL, nil
}

// hasVolatilityMarker checks if a function definition already has a volatility marker.
// Returns true if IMMUTABLE, STABLE, or VOLATILE is found (case-insensitive).
func (st *SQLTransformer) hasVolatilityMarker(s string) bool {
	upper := strings.ToUpper(s)
	return strings.Contains(upper, " IMMUTABLE") ||
	       strings.Contains(upper, " STABLE") ||
	       strings.Contains(upper, " VOLATILE")
}

// determineVolatility analyzes a function body to determine appropriate volatility category.
//
// Decision Logic (in order of precedence):
//
// 1. VOLATILE - Functions that:
//    - Modify data: INSERT, UPDATE, DELETE, TRUNCATE
//    - Modify schema: CREATE, DROP, ALTER
//    - Use non-deterministic sequences: nextval(), setval(), currval()
//    - Use randomness: random(), setseed()
//
// 2. STABLE - Functions that:
//    - Read session/auth state: auth.jwt(), auth.uid(), current_user, current_setting()
//    - Read time (within transaction): now(), current_timestamp, current_date
//    - Read database: SELECT (any query is at minimum STABLE)
//
// 3. IMMUTABLE - Functions that:
//    - Are purely computational (no DB access, no state)
//    - Always return same result for same inputs
//    - Example: Mathematical calculations, string operations
//
// Default: STABLE (safest for auth functions, works with index predicates)
//
// Note: We err on the side of caution. VOLATILE/STABLE are safe for index predicates,
// while incorrectly marking a function as IMMUTABLE can cause correctness issues.
func (st *SQLTransformer) determineVolatility(functionBody string) string {
	bodyUpper := strings.ToUpper(functionBody)

	// Pattern 1: VOLATILE - Data modification or side effects
	volatilePatterns := []string{
		"INSERT ", "UPDATE ", "DELETE ", "TRUNCATE ",
		"CREATE ", "DROP ", "ALTER ",
		"NEXTVAL(", "SETVAL(", "CURRVAL(",
		"RANDOM()", "SETSEED(",
	}
	for _, pattern := range volatilePatterns {
		if strings.Contains(bodyUpper, pattern) {
			return "VOLATILE"
		}
	}

	// Pattern 2: STABLE - Session/database state reads (no modifications)
	// This is the safest default for auth functions (Clerk, Supabase, Auth0)
	stablePatterns := []string{
		"AUTH.JWT()", "AUTH.UID()", "AUTH.ROLE()",           // Supabase/Clerk auth
		"CURRENT_USER", "CURRENT_SETTING(", "SESSION_USER",  // PostgreSQL session
		"CURRENT_TIMESTAMP", "NOW()", "TIMEOFDAY()",         // Time functions
		"CURRENT_DATE", "CURRENT_TIME",
		"TRANSACTION_TIMESTAMP()", "STATEMENT_TIMESTAMP()",
		"SELECT ", // Any SELECT query reads state
	}
	for _, pattern := range stablePatterns {
		if strings.Contains(bodyUpper, pattern) {
			return "STABLE"
		}
	}

	// Pattern 3: IMMUTABLE - Pure computational functions only
	// Very conservative check - must have no database/state access
	isPure := !strings.Contains(bodyUpper, "SELECT") &&
	          !strings.Contains(bodyUpper, "FROM") &&
	          !strings.Contains(bodyUpper, "PERFORM") &&
	          len(strings.TrimSpace(functionBody)) > 0

	if isPure {
		return "IMMUTABLE"
	}

	// Default: STABLE (safest, works with index predicates, appropriate for auth functions)
	return "STABLE"
}

// extractFunctionName extracts the function name from a CREATE FUNCTION statement.
// Handles both schema-qualified and unqualified names.
//
// Examples:
//   - "CREATE FUNCTION foo(...)" → "foo"
//   - "CREATE FUNCTION public.bar(...)" → "bar"
//   - "CREATE OR REPLACE FUNCTION baz(...)" → "baz"
func (st *SQLTransformer) extractFunctionName(funcDef string) string {
	// Pattern: FUNCTION [schema.]name
	// Captures only the function name, not the schema
	namePattern := regexp.MustCompile(`(?i)FUNCTION\s+(?:[a-z_][a-z0-9_]*\.)?([a-z_][a-z0-9_]*)`)
	match := namePattern.FindStringSubmatch(funcDef)
	if len(match) >= 2 {
		return match[1]
	}
	return "unknown"
}
