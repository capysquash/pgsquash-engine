package postprocessing

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/CAPYSQUASH/pgsquash-engine/internal/utils"
)

// FixMalformedDropTriggers fixes DROP TRIGGER statements with incorrect syntax.
// PostgreSQL requires: DROP TRIGGER IF EXISTS trigger_name ON table_name;
// This function fixes: DROP TRIGGER IF EXISTS table_name.trigger_name;
func FixMalformedDropTriggers(sql string) string {
	lines := strings.Split(sql, "\n")
	for i, line := range lines {
		upperLine := strings.ToUpper(strings.TrimSpace(line))

		// Check if this is a DROP TRIGGER statement with qualified name
		if strings.HasPrefix(upperLine, "DROP TRIGGER IF EXISTS") && strings.Contains(line, ".") {
			// Extract the qualified name
			// Format: DROP TRIGGER IF EXISTS table_name.trigger_name;
			parts := strings.Fields(line)
			if len(parts) >= 5 {
				qualifiedName := parts[4] // table_name.trigger_name or table_name.trigger_name;
				qualifiedName = strings.TrimSuffix(qualifiedName, ";")

				// Split on the dot
				dotIndex := strings.Index(qualifiedName, ".")
				if dotIndex > 0 {
					tableName := qualifiedName[:dotIndex]
					triggerName := qualifiedName[dotIndex+1:]

					// Reconstruct with correct syntax
					lines[i] = fmt.Sprintf("DROP TRIGGER IF EXISTS %s ON %s;", triggerName, tableName)
					utils.GetDefaultLogger().WithPrefix("POSTPROCESS").Info("Fixed malformed DROP TRIGGER: %s.%s -> %s ON %s", tableName, triggerName, triggerName, tableName)
				}
			}
		}
	}

	return strings.Join(lines, "\n")
}

// FixMissingSemicolons adds missing semicolons to SQL statements.
// PostgreSQL requires all statements to end with a semicolon.
func FixMissingSemicolons(sql string) string {
	lines := strings.Split(sql, "\n")
	var result []string
	fixedCount := 0

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip empty lines, comments, and lines that already end with semicolon
		if trimmed == "" || strings.HasPrefix(trimmed, "--") || strings.HasSuffix(trimmed, ";") {
			result = append(result, line)
			continue
		}

		// Skip lines that are part of function bodies (contain $$)
		if strings.Contains(line, "$$") {
			result = append(result, line)
			continue
		}

		// Skip lines that look like they're in the middle of a statement
		// (e.g., column definitions, constraint definitions)
		if i+1 < len(lines) {
			nextLine := strings.TrimSpace(lines[i+1])
			if strings.HasPrefix(nextLine, ",") || strings.HasPrefix(nextLine, ")") {
				result = append(result, line)
				continue
			}
		}

		// Check if this looks like a complete statement that needs a semicolon
		upperLine := strings.ToUpper(trimmed)
		needsSemicolon := strings.HasPrefix(upperLine, "CREATE ") ||
			strings.HasPrefix(upperLine, "ALTER ") ||
			strings.HasPrefix(upperLine, "DROP ") ||
			strings.HasPrefix(upperLine, "INSERT ") ||
			strings.HasPrefix(upperLine, "UPDATE ") ||
			strings.HasPrefix(upperLine, "DELETE ") ||
			strings.HasPrefix(upperLine, "GRANT ") ||
			strings.HasPrefix(upperLine, "REVOKE ")

		if needsSemicolon {
			result = append(result, line+";")
			fixedCount++
			utils.GetDefaultLogger().WithPrefix("POSTPROCESS").Info("Added missing semicolon to: %s", trimmed[:min(50, len(trimmed))])
		} else {
			result = append(result, line)
		}
	}

	if fixedCount > 0 {
		utils.GetDefaultLogger().WithPrefix("POSTPROCESS").Info("Added %d missing semicolons", fixedCount)
	}

	return strings.Join(result, "\n")
}

// FixMalformedFunctions repairs common function definition issues from consolidation.
// Issues fixed:
// 1. Missing AS keyword before function body
// 2. Duplicate LANGUAGE clauses
// 3. Standalone LANGUAGE lines after $$
// 4. Multiple volatility markers (STABLE and IMMUTABLE together)
// 5. Orphaned function bodies without CREATE FUNCTION headers
func FixMalformedFunctions(sql string) string {
	lines := strings.Split(sql, "\n")
	var result []string
	fixedCount := 0
	inOrphanedBody := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		upperLine := strings.ToUpper(trimmed)

		// Skip empty lines and comments
		if trimmed == "" || strings.HasPrefix(trimmed, "--") {
			result = append(result, line)
			continue
		}

		// Detect orphaned function bodies (bodies without CREATE FUNCTION)
		// These start with AS $$ or LANGUAGE clauses not preceded by CREATE FUNCTION
		if !inOrphanedBody && (strings.HasPrefix(upperLine, "AS $$") || strings.HasPrefix(upperLine, "LANGUAGE ")) {
			// Check if previous non-empty line was CREATE FUNCTION
			foundCreateFunction := false
			for j := i - 1; j >= 0; j-- {
				prevTrimmed := strings.TrimSpace(lines[j])
				if prevTrimmed == "" || strings.HasPrefix(prevTrimmed, "--") {
					continue
				}
				if strings.Contains(strings.ToUpper(prevTrimmed), "CREATE") && strings.Contains(strings.ToUpper(prevTrimmed), "FUNCTION") {
					foundCreateFunction = true
				}
				break
			}

			if !foundCreateFunction {
				utils.GetDefaultLogger().WithPrefix("POSTPROCESS").Info("Skipping orphaned function body at line %d", i+1)
				inOrphanedBody = true
				continue
			}
		}

		// Exit orphaned body mode when we see end delimiter
		if inOrphanedBody && strings.Contains(trimmed, "$$") && !strings.HasPrefix(upperLine, "AS $$") {
			inOrphanedBody = false
			continue
		}

		// Skip lines while in orphaned body
		if inOrphanedBody {
			continue
		}

		// Fix: Standalone LANGUAGE lines after $$
		if strings.HasPrefix(upperLine, "LANGUAGE ") && i > 0 {
			prevLine := strings.TrimSpace(lines[i-1])
			if strings.HasSuffix(prevLine, "$$") || strings.HasSuffix(prevLine, "$$;") {
				utils.GetDefaultLogger().WithPrefix("POSTPROCESS").Info("Removing standalone LANGUAGE line after $$")
				fixedCount++
				continue
			}
		}

		// Fix: Multiple volatility markers in same line
		if strings.Contains(upperLine, "STABLE") && strings.Contains(upperLine, "IMMUTABLE") {
			// Keep IMMUTABLE, remove STABLE (more restrictive)
			fixedLine := regexp.MustCompile(`(?i)\bSTABLE\b`).ReplaceAllString(line, "")
			fixedLine = regexp.MustCompile(`\s+`).ReplaceAllString(fixedLine, " ")
			result = append(result, fixedLine)
			fixedCount++
			utils.GetDefaultLogger().WithPrefix("POSTPROCESS").Info("Removed STABLE in favor of IMMUTABLE")
			continue
		}

		result = append(result, line)
	}

	if fixedCount > 0 {
		utils.GetDefaultLogger().WithPrefix("POSTPROCESS").Info("Fixed %d malformed function issues", fixedCount)
	}

	return strings.Join(result, "\n")
}

// RemoveOrphanedAlterStatements removes ALTER statements for objects that don't exist.
// This happens when an object is created and altered in different migrations,
// but the consolidation removes the CREATE.
func RemoveOrphanedAlterStatements(sql string) string {
	lines := strings.Split(sql, "\n")

	// First pass: collect all created objects
	createdObjects := make(map[string]bool)
	for _, line := range lines {
		upperLine := strings.ToUpper(strings.TrimSpace(line))

		// Match: CREATE TABLE schema.name or CREATE TABLE name
		if strings.HasPrefix(upperLine, "CREATE TABLE") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				objectName := strings.Trim(parts[2], `";,()`)
				createdObjects[objectName] = true
			}
		}

		// Match: CREATE INDEX, CREATE FUNCTION, etc.
		if strings.HasPrefix(upperLine, "CREATE INDEX") ||
			strings.HasPrefix(upperLine, "CREATE UNIQUE INDEX") ||
			strings.HasPrefix(upperLine, "CREATE FUNCTION") ||
			strings.HasPrefix(upperLine, "CREATE TYPE") {
			parts := strings.Fields(line)
			for i, part := range parts {
				if strings.ToUpper(part) == "INDEX" ||
					strings.ToUpper(part) == "FUNCTION" ||
					strings.ToUpper(part) == "TYPE" {
					if i+1 < len(parts) {
						objectName := strings.Trim(parts[i+1], `";,()`)
						createdObjects[objectName] = true
					}
					break
				}
			}
		}
	}

	// Second pass: remove ALTER statements for non-existent objects
	var result []string
	removedCount := 0

	for _, line := range lines {
		upperLine := strings.ToUpper(strings.TrimSpace(line))

		// Check if this is an ALTER statement
		if strings.HasPrefix(upperLine, "ALTER TABLE") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				objectName := strings.Trim(parts[2], `";,()`)
				if !createdObjects[objectName] {
					utils.GetDefaultLogger().WithPrefix("POSTPROCESS").Info("Removing orphaned ALTER TABLE for non-existent object: %s", objectName)
					removedCount++
					continue
				}
			}
		}

		result = append(result, line)
	}

	if removedCount > 0 {
		utils.GetDefaultLogger().WithPrefix("POSTPROCESS").Info("Removed %d orphaned ALTER statements", removedCount)
	}

	return strings.Join(result, "\n")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
