package ast

import (
	"strings"

	"github.com/CAPYSQUASH/pgsquash-engine/internal/utils"
	pg_query "github.com/pganalyze/pg_query_go/v6"
)

// FunctionNormalizer provides AST-based function normalization operations.
// It replaces regex-based function fixes with accurate AST manipulation.
type FunctionNormalizer struct {
	logger *utils.Logger
}

// NewFunctionNormalizer creates a new function normalizer.
func NewFunctionNormalizer() *FunctionNormalizer {
	return &FunctionNormalizer{
		logger: utils.GetDefaultLogger().WithPrefix("AST-FUNC"),
	}
}

// NormalizeAll applies all function normalizations to the given SQL.
// This is the main entry point that replaces all regex-based function fixes.
func (fn *FunctionNormalizer) NormalizeAll(sql string) (string, error) {
	// Parse SQL to AST
	parseResult, err := pg_query.Parse(sql)
	if err != nil {
		fn.logger.Info("Failed to parse SQL for AST normalization: %v (falling back to original)", err)
		return sql, nil // Return original SQL if parse fails
	}

	modified := false
	fixCount := 0

	// Process each statement
	for _, stmt := range parseResult.Stmts {
		funcStmt := stmt.Stmt.GetCreateFunctionStmt()
		if funcStmt == nil {
			continue // Not a CREATE FUNCTION statement
		}

		// Apply normalization operations
		if fn.fixLanguageOrder(funcStmt) {
			modified = true
			fixCount++
		}

		if fn.removeRedundantLanguage(funcStmt) {
			modified = true
			fixCount++
		}

		if fn.inferMissingLanguage(funcStmt) {
			modified = true
			fixCount++
		}

		if fn.removeDuplicateOptions(funcStmt) {
			modified = true
			fixCount++
		}
	}

	if !modified {
		return sql, nil // No changes needed
	}

	fn.logger.Info("Applied %d AST-based function normalizations", fixCount)

	// Deparse back to SQL
	deparsed, err := pg_query.Deparse(parseResult)
	if err != nil {
		fn.logger.Info("Failed to deparse normalized AST: %v (falling back to original)", err)
		return sql, nil
	}

	return deparsed, nil
}

// fixLanguageOrder ensures LANGUAGE comes before VOLATILE/STABLE/IMMUTABLE.
// PostgreSQL requires: LANGUAGE plpgsql VOLATILE
// Not: VOLATILE LANGUAGE plpgsql
func (fn *FunctionNormalizer) fixLanguageOrder(funcStmt *pg_query.CreateFunctionStmt) bool {
	if funcStmt.Options == nil || len(funcStmt.Options) == 0 {
		return false
	}

	// Find positions of language and volatility options
	languageIdx := -1
	volatilityIdx := -1

	for i, opt := range funcStmt.Options {
		defElem := opt.GetDefElem()
		if defElem == nil {
			continue
		}

		switch defElem.Defname {
		case "language":
			languageIdx = i
		case "volatility":
			volatilityIdx = i
		}
	}

	// Need to reorder if volatility comes before language
	if languageIdx >= 0 && volatilityIdx >= 0 && volatilityIdx < languageIdx {
		// Swap positions
		funcStmt.Options[languageIdx], funcStmt.Options[volatilityIdx] =
			funcStmt.Options[volatilityIdx], funcStmt.Options[languageIdx]

		fn.logger.Info("Fixed LANGUAGE/VOLATILITY order in function")
		return true
	}

	return false
}

// removeRedundantLanguage removes duplicate LANGUAGE declarations.
func (fn *FunctionNormalizer) removeRedundantLanguage(funcStmt *pg_query.CreateFunctionStmt) bool {
	if funcStmt.Options == nil || len(funcStmt.Options) == 0 {
		return false
	}

	languageCount := 0
	var keepIdx int

	// Count and find first language option
	for i, opt := range funcStmt.Options {
		defElem := opt.GetDefElem()
		if defElem != nil && defElem.Defname == "language" {
			if languageCount == 0 {
				keepIdx = i
			}
			languageCount++
		}
	}

	if languageCount <= 1 {
		return false // No duplicates
	}

	// Remove duplicate language options (keep first one)
	newOptions := make([]*pg_query.Node, 0, len(funcStmt.Options))
	languagesSeen := 0

	for i, opt := range funcStmt.Options {
		defElem := opt.GetDefElem()
		if defElem != nil && defElem.Defname == "language" {
			if i == keepIdx {
				newOptions = append(newOptions, opt)
			}
			languagesSeen++
		} else {
			newOptions = append(newOptions, opt)
		}
	}

	funcStmt.Options = newOptions
	fn.logger.Info("Removed %d duplicate LANGUAGE declarations", languageCount-1)
	return true
}

// inferMissingLanguage adds missing LANGUAGE declaration based on function body.
func (fn *FunctionNormalizer) inferMissingLanguage(funcStmt *pg_query.CreateFunctionStmt) bool {
	// Check if function already has a language
	hasLanguage := false
	for _, opt := range funcStmt.Options {
		defElem := opt.GetDefElem()
		if defElem != nil && defElem.Defname == "language" {
			hasLanguage = true
			break
		}
	}

	if hasLanguage {
		return false // Already has language
	}

	// Infer language from function body or return type
	language := fn.inferLanguageFromFunction(funcStmt)
	if language == "" {
		language = "sql" // Default to SQL
	}

	// Add language option
	languageOpt := &pg_query.Node{
		Node: &pg_query.Node_DefElem{
			DefElem: &pg_query.DefElem{
				Defname: "language",
				Arg: &pg_query.Node{
					Node: &pg_query.Node_String_{
						String_: &pg_query.String{
							Sval: language,
						},
					},
				},
			},
		},
	}

	// Insert language after RETURNS but before AS (if possible)
	// For now, append to end
	funcStmt.Options = append(funcStmt.Options, languageOpt)

	fn.logger.Info("Added missing LANGUAGE %s", language)
	return true
}

// inferLanguageFromFunction infers the appropriate language from function characteristics.
func (fn *FunctionNormalizer) inferLanguageFromFunction(funcStmt *pg_query.CreateFunctionStmt) string {
	// Check if function returns TRIGGER - must be plpgsql
	if funcStmt.ReturnType != nil {
		returnType := fn.getTypeNameFromTypeName(funcStmt.ReturnType)
		if strings.ToLower(returnType) == "trigger" {
			return "plpgsql"
		}
	}

	// Check function body for plpgsql constructs
	// Function body can be in SqlBody (for BEGIN ATOMIC syntax) or in Options (for AS $$ syntax)
	bodyStr := fn.extractFunctionBody(funcStmt)
	if bodyStr != "" {
		bodyLower := strings.ToLower(bodyStr)

		// Check for plpgsql-specific constructs
		plpgsqlKeywords := []string{
			"begin", "declare", "perform ", "raise ",
			"return next", "return query", "if then", "loop",
		}

		for _, keyword := range plpgsqlKeywords {
			if strings.Contains(bodyLower, keyword) {
				return "plpgsql"
			}
		}
	}

	return "sql" // Default to SQL
}

// removeDuplicateOptions removes any duplicate options.
func (fn *FunctionNormalizer) removeDuplicateOptions(funcStmt *pg_query.CreateFunctionStmt) bool {
	if funcStmt.Options == nil || len(funcStmt.Options) == 0 {
		return false
	}

	seen := make(map[string]int) // option name -> first index
	duplicates := 0

	// Find duplicates
	for i, opt := range funcStmt.Options {
		defElem := opt.GetDefElem()
		if defElem == nil {
			continue
		}

		if firstIdx, exists := seen[defElem.Defname]; exists {
			// Duplicate found - keep first occurrence
			duplicates++
			fn.logger.Info("Found duplicate option: %s (keeping first at index %d)", defElem.Defname, firstIdx)
		} else {
			seen[defElem.Defname] = i
		}
	}

	if duplicates == 0 {
		return false
	}

	// Build new options list without duplicates
	newOptions := make([]*pg_query.Node, 0, len(funcStmt.Options))
	seenFinal := make(map[string]bool)

	for _, opt := range funcStmt.Options {
		defElem := opt.GetDefElem()
		if defElem == nil {
			newOptions = append(newOptions, opt)
			continue
		}

		if !seenFinal[defElem.Defname] {
			newOptions = append(newOptions, opt)
			seenFinal[defElem.Defname] = true
		}
	}

	funcStmt.Options = newOptions
	return true
}

// Helper functions

func (fn *FunctionNormalizer) getTypeNameFromTypeName(typeName *pg_query.TypeName) string {
	if typeName == nil {
		return ""
	}

	if len(typeName.Names) == 0 {
		return ""
	}

	// Get last component of type name (e.g., "pg_catalog.text" -> "text")
	lastNode := typeName.Names[len(typeName.Names)-1]
	str := lastNode.GetString_()
	if str != nil {
		return str.Sval
	}

	return ""
}

// extractFunctionBody extracts the function body text from CreateFunctionStmt.
// It checks both SqlBody (for BEGIN ATOMIC syntax) and Options (for AS $$ syntax).
func (fn *FunctionNormalizer) extractFunctionBody(funcStmt *pg_query.CreateFunctionStmt) string {
	// First check SqlBody (for BEGIN ATOMIC syntax)
	if funcStmt.SqlBody != nil {
		// SqlBody contains structured SQL statements
		// Try to deparse it to get the body text
		if bodyText := fn.deparseNode(funcStmt.SqlBody); bodyText != "" {
			return bodyText
		}
	}

	// Check Options for "as" DefElem (traditional $$ syntax)
	for _, opt := range funcStmt.Options {
		defElem := opt.GetDefElem()
		if defElem != nil && defElem.Defname == "as" {
			// The body is in defElem.Arg
			if defElem.Arg != nil {
				// Try to extract as list of strings
				if listNode := defElem.Arg.GetList(); listNode != nil {
					var bodyParts []string
					for _, item := range listNode.Items {
						if strNode := item.GetString_(); strNode != nil {
							bodyParts = append(bodyParts, strNode.Sval)
						}
					}
					if len(bodyParts) > 0 {
						return strings.Join(bodyParts, "\n")
					}
				}
				// Try to extract as single string
				if strNode := defElem.Arg.GetString_(); strNode != nil {
					return strNode.Sval
				}
			}
		}
	}

	return ""
}

// deparseNode attempts to deparse a Node back to SQL text.
func (fn *FunctionNormalizer) deparseNode(node *pg_query.Node) string {
	// Create a minimal parse result with just this node
	parseResult := &pg_query.ParseResult{
		Stmts: []*pg_query.RawStmt{
			{Stmt: node},
		},
	}

	deparsed, err := pg_query.Deparse(parseResult)
	if err != nil {
		return ""
	}

	return deparsed
}
