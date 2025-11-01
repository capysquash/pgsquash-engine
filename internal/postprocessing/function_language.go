package postprocessing

import (
	"regexp"
	"strings"

	"github.com/CAPYSQUASH/pgsquash-engine/internal/utils"
)

// FixFunctionLanguageConflicts fixes functions with conflicting VOLATILE/LANGUAGE placement.
// This is a safety net post-processor that catches any functions where normalization didn't run
// or where SQL transformation added volatility markers incorrectly.
//
// Problem Pattern (Invalid PostgreSQL syntax):
//
//	CREATE FUNCTION foo() RETURNS text VOLATILE AS $$ ... $$ LANGUAGE plpgsql;
//
// Fixed Pattern (Valid):
//
//	CREATE FUNCTION foo() RETURNS text LANGUAGE plpgsql VOLATILE AS $$ ... $$;
//
// Root Cause:
// When volatility markers (VOLATILE/STABLE/IMMUTABLE) are added before AS $$,
// PostgreSQL requires LANGUAGE to also be before AS $$, not after the function body.
//
// This function detects and fixes three scenarios:
// 1. VOLATILE AS $$ ... $$ LANGUAGE plpgsql → LANGUAGE plpgsql VOLATILE AS $$ ... $$
// 2. STABLE AS $$ ... $$ LANGUAGE plpgsql → LANGUAGE plpgsql STABLE AS $$ ... $$
// 3. IMMUTABLE AS $$ ... $$ LANGUAGE plpgsql → LANGUAGE plpgsql IMMUTABLE AS $$ ... $$
func FixFunctionLanguageConflicts(sql string) string {
	// Regex Pattern:
	// Group 1: CREATE [OR REPLACE] FUNCTION name(...) RETURNS type
	// Group 2: Optional existing modifiers (SECURITY DEFINER, etc.)
	// Group 3: Volatility marker (VOLATILE|STABLE|IMMUTABLE)
	// Group 4: AS $$
	// Group 5: Function body
	// Group 6: $$ (closing delimiter)
	// Group 7: LANGUAGE clause (this needs to be moved)
	// Group 8: Optional SECURITY DEFINER after LANGUAGE
	//
	// We look for functions where volatility comes before AS but LANGUAGE comes after body
	// CRITICAL FIX: Instead of using (.+?) which can match across function boundaries,
	// we match specifically up to and including "$$ LANGUAGE" to ensure we only match ONE function
	// Pattern: Match everything up to the FIRST occurrence of $$ followed immediately by LANGUAGE
	// This prevents the regex from accidentally spanning multiple adjacent functions
	pattern := regexp.MustCompile(
		`(?ims)(CREATE\s+(?:OR\s+REPLACE\s+)?FUNCTION\s+(?:[a-z_][a-z0-9_]*\.)?[a-z_][a-z0-9_]*\s*\([^)]*\)\s*RETURNS\s+(?:TABLE\s*\([^)]+\)|SETOF\s+[^\s]+|[^\s]+))((?:\s+(?:SECURITY\s+DEFINER|SET\s+[^\s]+\s*=\s*[^\s]+))*?)\s+(VOLATILE|STABLE|IMMUTABLE)(\s+AS\s+\$\$)([\s\S]*?)(\$\$\s+LANGUAGE\s+[a-z]+)((?:\s+SECURITY\s+DEFINER)?)`,
	)

	matches := pattern.FindAllStringSubmatchIndex(sql, -1)
	if len(matches) == 0 {
		return sql // No conflicting functions found
	}

	transformedSQL := sql
	offset := 0
	fixedCount := 0

	for _, match := range matches {
		if len(match) < 16 {
			continue
		}

		// Extract parts (Groups changed because we now capture "$$ LANGUAGE" as combined Group 6)
		signature := transformedSQL[match[0]+offset : match[1]+offset]           // Group 1: CREATE FUNCTION...RETURNS type
		existingModifiers := transformedSQL[match[2]+offset : match[3]+offset]   // Group 2: Existing modifiers
		volatility := transformedSQL[match[4]+offset : match[5]+offset]          // Group 3: VOLATILE/STABLE/IMMUTABLE
		asKeyword := transformedSQL[match[6]+offset : match[7]+offset]           // Group 4: AS $$
		body := transformedSQL[match[8]+offset : match[9]+offset]                // Group 5: function body
		closingLangClause := transformedSQL[match[10]+offset : match[11]+offset] // Group 6: $$ LANGUAGE plpgsql (combined!)
		securityAfterLang := ""
		if match[12] >= 0 && match[13] >= 0 {
			securityAfterLang = transformedSQL[match[12]+offset : match[13]+offset] // Group 7: SECURITY DEFINER
		}

		// Extract $$ and LANGUAGE separately from the combined group
		// closingLangClause is like "$$ LANGUAGE plpgsql"
		parts := strings.Fields(closingLangClause) // Split on whitespace
		closingDelim := "$$"
		languageClause := ""
		if len(parts) >= 3 {
			// parts[0] = "$$", parts[1] = "LANGUAGE", parts[2] = "plpgsql"
			languageClause = strings.Join(parts[1:], " ") // "LANGUAGE plpgsql"
		}

		// Build corrected function:
		// signature + existingModifiers + LANGUAGE + volatility + SECURITY + AS $$ + body + $$;
		normalizedModifiers := strings.TrimSpace(existingModifiers)
		normalizedLanguage := strings.TrimSpace(languageClause)
		normalizedVolatility := strings.TrimSpace(volatility)
		normalizedSecurity := strings.TrimSpace(securityAfterLang)

		// Build modifier chain in correct order: LANGUAGE → VOLATILITY → SECURITY
		var modifiers string
		if normalizedModifiers != "" {
			// Already have modifiers, add LANGUAGE and VOLATILITY
			modifiers = normalizedModifiers + " " + normalizedLanguage + " " + normalizedVolatility
		} else {
			// No existing modifiers
			modifiers = normalizedLanguage + " " + normalizedVolatility
		}

		// Add SECURITY DEFINER at the end if it was after LANGUAGE
		if normalizedSecurity != "" {
			modifiers = modifiers + " " + normalizedSecurity
		}

		// Reconstruct: signature + " " + modifiers + AS $$ + body + $$;
		fixedFunction := signature + " " + modifiers + asKeyword + body + closingDelim + ";"

		// Calculate old function length (everything we're replacing)
		oldFunctionEnd := match[11] + offset
		if match[12] >= 0 { // Has SECURITY DEFINER after LANGUAGE
			oldFunctionEnd = match[13] + offset
		}
		oldFunction := transformedSQL[match[0]+offset : oldFunctionEnd]

		// Replace in SQL
		transformedSQL = transformedSQL[:match[0]+offset] + fixedFunction + transformedSQL[oldFunctionEnd:]
		offset += len(fixedFunction) - len(oldFunction)
		fixedCount++

		utils.GetDefaultLogger().WithPrefix("POSTPROCESS").Info("Fixed function with conflicting %s placement (moved LANGUAGE before AS)", normalizedVolatility)
	}

	if fixedCount > 0 {
		utils.GetDefaultLogger().WithPrefix("POSTPROCESS").Info("Fixed %d functions with conflicting VOLATILE/STABLE/IMMUTABLE and LANGUAGE placement", fixedCount)
	}

	return transformedSQL
}

// FixRedundantTrailingLanguageClauses removes redundant LANGUAGE clauses that appear after
// the closing $$ delimiter in function definitions. PostgreSQL rejects these as
// "conflicting or redundant options" when LANGUAGE is also in the header.
//
// This function handles all variations of trailing language clauses:
//   - $$ language 'plpgsql';   → $$;
//   - $$ language 'sql';       → $$;
//   - $$ language plpgsql;     → $$;
//   - $$ LANGUAGE SQL;         → $$;
//   - $$\nlanguage 'plpgsql';  → $$;  (with newline)
//   - $tag$ language plpgsql;  → $tag$;  (tagged dollar quotes)
//
// NOTE: This function only REMOVES trailing clauses. The FixFunctionLanguageConflicts
// function (which runs after this) handles moving LANGUAGE to the correct header position.
//
// EXECUTION: Called in post-processing phase BEFORE FixFunctionLanguageConflicts.
func FixRedundantTrailingLanguageClauses(sql string) string {
	// Pattern matches: closing dollar-quote delimiter + whitespace + language clause + optional modifiers + semicolon
	// Groups:
	//   1. Closing delimiter ($$ or $tag$)
	//   2. Language name (for logging only, not used in replacement)
	// CRITICAL: Only match common modifier keywords to prevent over-greedy matching
	// Pattern: $$ + language + (optional: security definer/invoker, volatile/stable/immutable, strict) + ;
	pattern := regexp.MustCompile(
		`(\$+(?:[a-z_][a-z0-9_]*)?\$)\s*[lL][aA][nN][gG][uU][aA][gG][eE]\s+['"]?(plpgsql|sql|c|internal)['"]?(?:\s+(?:security\s+(?:definer|invoker)|volatile|stable|immutable|strict))*\s*;`,
	)

	// Find all matches before replacement (for logging)
	matches := pattern.FindAllStringSubmatch(sql, -1)

	// Replace: keep the closing delimiter, remove language clause, keep semicolon
	fixedSQL := pattern.ReplaceAllString(sql, "$1;")

	if len(matches) > 0 {
		utils.GetDefaultLogger().WithPrefix("POSTPROCESS").Info(
			"Removed %d redundant trailing LANGUAGE clauses from function definitions",
			len(matches),
		)

		// Log examples (helpful for debugging)
		for i, match := range matches {
			if i < 3 { // Only log first 3 to avoid spam
				utils.GetDefaultLogger().WithPrefix("POSTPROCESS").Info(
					"  Example %d: Removed '$$ language %s;'",
					i+1,
					match[2], // match[2] is the language name from capture group 2
				)
			}
		}
	}

	return fixedSQL
}

// FixIncorrectLanguageDeclarations fixes functions that have incorrect LANGUAGE declarations
// based on their body content. For example, functions with BEGIN/END blocks but declared as
// LANGUAGE SQL should be changed to LANGUAGE plpgsql.
//
// Common patterns:
//   - RETURNS TRIGGER + LANGUAGE SQL → should be LANGUAGE plpgsql (triggers require plpgsql)
//   - Body has BEGIN/END + LANGUAGE SQL → should be LANGUAGE plpgsql
//   - Body has DECLARE + LANGUAGE SQL → should be LANGUAGE plpgsql
//   - Body has PERFORM + LANGUAGE SQL → should be LANGUAGE plpgsql
func FixIncorrectLanguageDeclarations(sql string) string {
	// Simpler approach: Find all functions with LANGUAGE SQL and check if body needs plpgsql
	// Pattern handles multi-line RETURNS clauses like RETURNS TABLE(...)
	pattern := regexp.MustCompile(
		`(?si)(CREATE\s+(?:OR\s+REPLACE\s+)?FUNCTION\s+[^\(]+\([^\)]*\)\s+RETURNS\s+.*?)\s+(LANGUAGE\s+SQL)(\s+.*?AS\s+\$\$)(.*?)(\$\$\s*;)`,
	)

	fixedSQL := sql
	fixedCount := 0

	matches := pattern.FindAllStringSubmatchIndex(sql, -1)
	offset := 0

	for _, match := range matches {
		// Extract components
		signature := sql[match[2]+offset : match[3]+offset]                     // Group 1: function header
		langClause := sql[match[4]+offset : match[5]+offset]                    // Group 2: "LANGUAGE SQL"
		body := strings.ToLower(sql[match[8]+offset : match[9]+offset])        // Group 4: body

		// Check if body has plpgsql-specific constructs
		hasPlpgsqlConstructs := strings.Contains(body, "begin") ||
			strings.Contains(body, "declare") ||
			strings.Contains(body, "perform ") ||
			strings.Contains(body, "raise ") ||
			strings.Contains(body, "return next") ||
			strings.Contains(body, "return query")

		// Check if function returns TRIGGER (triggers must be plpgsql)
		isTrigger := strings.Contains(strings.ToLower(signature), "returns trigger")

		// Determine if we need to change the language to plpgsql
		if hasPlpgsqlConstructs || isTrigger {
			// Replace LANGUAGE SQL with LANGUAGE plpgsql
			oldFunction := sql[match[0]+offset : match[1]+offset]
			newFunction := strings.Replace(oldFunction, langClause, "LANGUAGE plpgsql", 1)

			fixedSQL = fixedSQL[:match[0]+offset] + newFunction + fixedSQL[match[1]+offset:]
			offset += len(newFunction) - len(oldFunction)
			fixedCount++

			utils.GetDefaultLogger().WithPrefix("POSTPROCESS").Info(
				"Fixed incorrect language declaration: LANGUAGE SQL → LANGUAGE plpgsql (body contains plpgsql constructs or returns TRIGGER)",
			)
		}
	}

	if fixedCount > 0 {
		utils.GetDefaultLogger().WithPrefix("POSTPROCESS").Info(
			"Fixed %d incorrect LANGUAGE declarations",
			fixedCount,
		)
	}

	return fixedSQL
}

// FixMissingLanguageDeclarations adds LANGUAGE declarations to functions that are missing them.
// PostgreSQL requires all functions to have a LANGUAGE clause.
func FixMissingLanguageDeclarations(sql string) string {
	// Pattern to match functions without LANGUAGE
	// Look for: CREATE FUNCTION ... RETURNS ... (optional modifiers) AS $$ ... $$;
	// Where there's no LANGUAGE between RETURNS and AS
	// Made modifiers optional (*) to handle functions with no modifiers yet
	pattern := regexp.MustCompile(
		`(?si)(CREATE\s+(?:OR\s+REPLACE\s+)?FUNCTION\s+[^\(]+\([^\)]*\)\s+RETURNS\s+[^\n]+?)(\s+(?:VOLATILE|STABLE|IMMUTABLE|STRICT|SECURITY\s+(?:DEFINER|INVOKER)|\s)*)(AS\s+\$\$)(.*?)(\$\$\s*;)`,
	)

	fixedSQL := sql
	fixedCount := 0

	matches := pattern.FindAllStringSubmatchIndex(sql, -1)
	offset := 0

	for _, match := range matches {
		// Extract components
		signature := sql[match[2]+offset : match[3]+offset]                     // Group 1: function header
		modifiers := sql[match[4]+offset : match[5]+offset]                     // Group 2: modifiers between RETURNS and AS
		asClause := sql[match[6]+offset : match[7]+offset]                      // Group 3: "AS $$"
		body := strings.ToLower(sql[match[8]+offset : match[9]+offset])        // Group 4: body

		// The non-greedy RETURNS pattern can expand to include LANGUAGE, leaving only
		// whitespace in the modifiers group, which would pass the check incorrectly.
		fullSignature := sql[match[2]+offset : match[7]+offset] // Everything from CREATE to AS
		if strings.Contains(strings.ToUpper(fullSignature), "LANGUAGE") {
			continue // Skip, already has LANGUAGE
		}

		// Infer language from body or signature
		language := "sql" // default

		// Check if body has plpgsql-specific constructs
		hasPlpgsqlConstructs := strings.Contains(body, "begin") ||
			strings.Contains(body, "declare") ||
			strings.Contains(body, "perform ") ||
			strings.Contains(body, "raise ") ||
			strings.Contains(body, "return next") ||
			strings.Contains(body, "return query")

		// Check if function returns TRIGGER (triggers must be plpgsql)
		isTrigger := strings.Contains(strings.ToLower(signature), "returns trigger")

		if hasPlpgsqlConstructs || isTrigger {
			language = "plpgsql"
		}

		// Insert LANGUAGE before AS
		// Handle spacing properly whether modifiers exist or not
		languageClause := " LANGUAGE " + language
		if strings.TrimSpace(modifiers) != "" {
			// Has modifiers - add LANGUAGE after them
			languageClause = modifiers + languageClause
		}

		oldFunction := sql[match[0]+offset : match[1]+offset]
		newFunction := sql[match[2]+offset : match[3]+offset] + languageClause + " " + asClause + sql[match[8]+offset : match[9]+offset] + sql[match[10]+offset : match[11]+offset]

		fixedSQL = fixedSQL[:match[0]+offset] + newFunction + fixedSQL[match[1]+offset:]
		offset += len(newFunction) - len(oldFunction)
		fixedCount++

		utils.GetDefaultLogger().WithPrefix("POSTPROCESS").Info(
			"Added missing LANGUAGE declaration: LANGUAGE %s (inferred from %s)",
			language,
			map[bool]string{true: "body constructs", false: "function signature"}[hasPlpgsqlConstructs],
		)
	}

	if fixedCount > 0 {
		utils.GetDefaultLogger().WithPrefix("POSTPROCESS").Info(
			"Added %d missing LANGUAGE declarations",
			fixedCount,
		)
	}

	return fixedSQL
}

// RemoveDuplicateLanguageDeclarations removes duplicate LANGUAGE clauses from functions.
// Example: "LANGUAGE plpgsql STABLE LANGUAGE plpgsql" → "LANGUAGE plpgsql STABLE"
// NOTE: This function is currently disabled in the processing pipeline due to regex complexity.
// It should be reimplemented using AST-based approach instead of regex.
func RemoveDuplicateLanguageDeclarations(sql string) string {
	// Find function signatures (from CREATE FUNCTION to AS $$) and check for duplicate LANGUAGE
	// Pattern matches: CREATE [OR REPLACE] FUNCTION ... AS $$
	funcPattern := regexp.MustCompile(`(?i)(CREATE\s+(?:OR\s+REPLACE\s+)?FUNCTION\s+[^;]+?)\s+AS\s+\$\$`)

	fixedSQL := sql
	fixedCount := 0

	// Process each function signature
	matches := funcPattern.FindAllStringSubmatchIndex(sql, -1)
	offset := 0

	for _, match := range matches {
		signatureStart := match[2] + offset
		signatureEnd := match[3] + offset
		signature := fixedSQL[signatureStart:signatureEnd]

		// Count LANGUAGE occurrences in signature
		languagePattern := regexp.MustCompile(`(?i)\bLANGUAGE\s+(?:plpgsql|sql|c|internal)\b`)
		languageMatches := languagePattern.FindAllStringIndex(strings.ToUpper(signature), -1)

		if len(languageMatches) > 1 {
			// Found duplicates - keep only the first LANGUAGE declaration
			firstLangStart := languageMatches[0][0]
			firstLangEnd := languageMatches[0][1]
			firstLang := signature[firstLangStart:firstLangEnd]

			// Remove all LANGUAGE declarations
			cleanSignature := languagePattern.ReplaceAllString(signature, "")

			// Re-insert the first LANGUAGE at its original position
			// To maintain proper spacing, find where to insert it
			// Insert after RETURNS clause if present
			returnsPattern := regexp.MustCompile(`(?i)\bRETURNS\s+[^\s]+(?:\s*\([^)]*\))?`)
			returnsMatch := returnsPattern.FindStringIndex(strings.ToUpper(cleanSignature))

			var newSignature string
			if returnsMatch != nil {
				// Insert LANGUAGE after RETURNS clause
				insertPos := returnsMatch[1]
				newSignature = cleanSignature[:insertPos] + " " + firstLang + cleanSignature[insertPos:]
			} else {
				// Fallback: append LANGUAGE at end of signature
				newSignature = cleanSignature + " " + firstLang
			}

			// Replace signature in SQL
			fixedSQL = fixedSQL[:signatureStart] + newSignature + fixedSQL[signatureEnd:]
			offset += len(newSignature) - len(signature)
			fixedCount++

			utils.GetDefaultLogger().WithPrefix("POSTPROCESS").Info("Removed duplicate LANGUAGE declaration from function")
		}
	}

	if fixedCount > 0 {
		utils.GetDefaultLogger().WithPrefix("POSTPROCESS").Info("Removed %d duplicate LANGUAGE declarations", fixedCount)
	}

	return fixedSQL
}
