package consolidation

import (
	"github.com/capysquash/pg-squash-engine/internal/utils"
	"fmt"
	"regexp"
	"strings"

	"github.com/capysquash/pg-squash-engine/internal/tracking"
	"github.com/capysquash/pg-squash-engine/internal/types"

	"github.com/capysquash/pg-squash-engine/internal/errors"
)

// EnumDeduplicationRule detects and resolves duplicate ENUM type definitions
// This handles cases where multiple ENUMs with similar names or conflicting values exist
type EnumDeduplicationRule struct{}

// CanApply checks if the rule can be applied to the given lifecycle
func (r *EnumDeduplicationRule) CanApply(lifecycle *tracking.ObjectLifecycle) bool {
	// Only apply to ENUM types
	if lifecycle.Type != types.TypeEnum && lifecycle.Type != types.TypeType {
		return false
	}

	// Check if this is an ENUM (either standalone or in DO block)
	hasEnumDef := false
	for _, event := range lifecycle.History {
		sqlUpper := strings.ToUpper(event.Statement.SQL)
		if strings.Contains(sqlUpper, "CREATE TYPE") && strings.Contains(sqlUpper, "AS ENUM") {
			hasEnumDef = true
			break
		}
	}

	return hasEnumDef
}

// Apply applies the consolidation rule to the given lifecycle
func (r *EnumDeduplicationRule) Apply(lifecycle *tracking.ObjectLifecycle, engine ConsolidationEngine) (*tracking.ConsolidationResult, error) {
	if !r.CanApply(lifecycle) {
		return nil, errors.New(errors.ErrorCodeConsolidationFailed, errors.CategoryConsolidation, "rule cannot be applied to lifecycle", map[string]interface{}{"rule": "EnumDedupRule"})
	}

	tracker := engine.GetTracker()

	// Collect all ENUM definitions
	var enumStmts []types.Statement
	enumPattern := regexp.MustCompile(`(?is)CREATE\s+TYPE\s+([a-zA-Z_][a-zA-Z0-9_]*)\s+AS\s+ENUM\s*\((.*?)\)`)

	for _, event := range lifecycle.History {
		if matches := enumPattern.FindStringSubmatch(event.Statement.SQL); len(matches) > 1 {
			enumStmts = append(enumStmts, event.Statement)
		}
	}

	if len(enumStmts) == 0 {
		return nil, errors.New(errors.ErrorCodeConsolidationFailed, errors.CategoryConsolidation, "no ENUM statements found", map[string]interface{}{"object": lifecycle.Name})
	}

	// Check for duplicate ENUM types across the entire codebase
	typeName := lifecycle.Name
	allLifecycles := tracker.GetObjectsByCategory()
	var conflictingEnums []tracking.ObjectLifecycle
	var primaryEnum string // The name that should be kept

	// Collect all similar ENUMs including self
	allSimilarEnums := []string{typeName}
	for _, categoryObjects := range allLifecycles {
		for _, otherLifecycle := range categoryObjects {
			if otherLifecycle.Key == lifecycle.Key {
				continue // Skip self
			}

			// Check if other lifecycle is also an ENUM with similar name
			if (otherLifecycle.Type == types.TypeEnum || otherLifecycle.Type == types.TypeType) &&
				isSimilarEnumName(typeName, otherLifecycle.Name) {
				conflictingEnums = append(conflictingEnums, *otherLifecycle)
				allSimilarEnums = append(allSimilarEnums, otherLifecycle.Name)
			}
		}
	}

	// Determine the primary ENUM using a deterministic rule (alphabetically first without _enum suffix)
	if len(allSimilarEnums) > 1 {
		// Sort all similar ENUMs
		for i := 0; i < len(allSimilarEnums)-1; i++ {
			for j := i + 1; j < len(allSimilarEnums); j++ {
				// Prefer names without _enum suffix, otherwise alphabetical
				name1 := allSimilarEnums[i]
				name2 := allSimilarEnums[j]

				// Strip common suffixes for comparison
				clean1 := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(name1, "_enum"), "_type"), "_status")
				clean2 := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(name2, "_type"), "_status"), "_enum")

				// If one is clean and the other has suffix, prefer the clean one
				hasClean1 := clean1 == name1
				hasClean2 := clean2 == name2

				var shouldSwap bool
				if hasClean1 && !hasClean2 {
					shouldSwap = false // name1 is clean, keep it first
				} else if !hasClean1 && hasClean2 {
					shouldSwap = true // name2 is clean, move it first
				} else {
					// Both clean or both have suffix - use alphabetical
					shouldSwap = name1 > name2
				}

				if shouldSwap {
					allSimilarEnums[i], allSimilarEnums[j] = allSimilarEnums[j], allSimilarEnums[i]
				}
			}
		}
		primaryEnum = allSimilarEnums[0]
		utils.GetDefaultLogger().WithPrefix("ENUM-DEDUP").Info("EnumDeduplicationRule: Primary ENUM chosen: %s from candidates %v", primaryEnum, allSimilarEnums)
	} else {
		primaryEnum = typeName
	}

	// If this ENUM has conflicts and is NOT the primary one, skip it entirely
	if len(conflictingEnums) > 0 && lifecycle.Name != primaryEnum {
		// This is a duplicate - eliminate it
		utils.GetDefaultLogger().WithPrefix("ENUM-DEDUP").Info("EnumDeduplicationRule: Eliminating duplicate ENUM %s (keeping %s)", lifecycle.Name, primaryEnum)
		result := &tracking.ConsolidationResult{
			OriginalStatements: enumStmts,
			ConsolidatedSQL:    fmt.Sprintf("-- Duplicate ENUM eliminated: %s (similar to %s)", typeName, primaryEnum),
			Optimizations: []string{
				fmt.Sprintf("Eliminated duplicate ENUM %s (keeping %s)", typeName, primaryEnum),
			},
			Warnings: []string{
				fmt.Sprintf("ENUM %s is a duplicate of %s - eliminated to prevent conflicts", typeName, primaryEnum),
			},
			RiskLevel: tracking.RiskLevelMedium, // Medium risk since we're removing a type
			EstimatedSavings: tracking.SquashSavings{
				StatementsReduced: len(enumStmts),
				FilesAffected:     len(enumStmts),
				LinesReduced:      len(enumStmts) * 5,
			},
		}
		return result, nil
	}

	// This is the earliest/first ENUM - keep it
	firstEnum := enumStmts[0]
	warnings := []string{}

	if len(enumStmts) > 1 {
		warnings = append(warnings, fmt.Sprintf("Found %d definitions for ENUM %s, using first definition", len(enumStmts), typeName))
	}

	if len(conflictingEnums) > 0 {
		var conflictNames []string
		for _, c := range conflictingEnums {
			conflictNames = append(conflictNames, c.Name)
		}
		warnings = append(warnings, fmt.Sprintf("ENUM %s is the primary definition (duplicates eliminated: %v)", typeName, conflictNames))
	}

	result := &tracking.ConsolidationResult{
		OriginalStatements: enumStmts,
		ConsolidatedSQL:    firstEnum.SQL,
		Optimizations: []string{
			fmt.Sprintf("Deduplicated %d ENUM definition(s) for %s", len(enumStmts)-1, typeName),
		},
		Warnings:  warnings,
		RiskLevel: tracking.RiskLevelMedium, // Medium risk since we're choosing one definition
		EstimatedSavings: tracking.SquashSavings{
			StatementsReduced: len(enumStmts) - 1,
			FilesAffected:     len(enumStmts),
			LinesReduced:      (len(enumStmts) - 1) * 5,
		},
	}

	return result, nil
}

// Risk returns the risk level for this rule
func (r *EnumDeduplicationRule) Risk() tracking.RiskLevel {
	return tracking.RiskLevelMedium
}

// isSimilarEnumName checks if two ENUM names are similar enough to be duplicates
// Examples: "verification_status" and "verification_status_enum" are similar
func isSimilarEnumName(name1, name2 string) bool {
	n1 := strings.ToLower(name1)
	n2 := strings.ToLower(name2)

	// Exact match
	if n1 == n2 {
		return true
	}

	// One is a prefix of the other
	if strings.HasPrefix(n1, n2) || strings.HasPrefix(n2, n1) {
		return true
	}

	// Check for common suffix patterns (_enum, _type)
	suffixes := []string{"_enum", "_type", "_status"}
	for _, suffix := range suffixes {
		n1WithoutSuffix := strings.TrimSuffix(n1, suffix)
		n2WithoutSuffix := strings.TrimSuffix(n2, suffix)

		if n1WithoutSuffix == n2WithoutSuffix {
			return true
		}
	}

	return false
}
