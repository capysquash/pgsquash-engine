package clerk

import (
	"strings"

	"github.com/capysquash/pg-squash/internal/plugins"
	"github.com/capysquash/pg-squash/internal/types"
)

// GetConsolidationRules returns Clerk-specific consolidation rules
func (cp *ClerkPlugin) GetConsolidationRules() []plugins.ConsolidationRule {
    return []plugins.ConsolidationRule{
        {
            Name:        "clerk_jwt_v2_policies",
            Priority:    100,
            ObjectType:  types.TypePolicy,
            AuthPattern: types.AuthPatternClerkJWTV2,
            Conflicts:   []string{"general_policy_consolidation"},
            CanMerge:    cp.canMergeJWTV2Policies,
            Merge:       cp.mergeJWTV2Policies,
        },
        {
            Name:        "clerk_auth_functions",
            Priority:    95,
            ObjectType:  types.TypeFunction,
            AuthPattern: types.AuthPatternClerk,
            Conflicts:   []string{"function_deduplication"},
            CanMerge:    cp.canMergeAuthFunctions,
            Merge:       cp.mergeAuthFunctions,
        },
    }
}

// canMergeJWTV2Policies checks if Clerk JWT v2 RLS policies can be merged
// Conservative approach: Only merge if policies are identical
func (cp *ClerkPlugin) canMergeJWTV2Policies(statements []*types.Statement) bool {
    if len(statements) < 2 {
        return false
    }

    // All statements must be policies with same target table
    firstPolicy := statements[0]
    for _, stmt := range statements[1:] {
        if stmt.ObjectType != types.TypePolicy {
            return false
        }
        if stmt.ObjectName != firstPolicy.ObjectName {
            return false
        }
    }

    // Check if policy definitions are semantically identical
    // (Different CREATE vs ALTER, but same USING/WITH CHECK clauses)
    return cp.haveSamePolicyLogic(statements)
}

// haveSamePolicyLogic checks if policies have the same security logic
func (cp *ClerkPlugin) haveSamePolicyLogic(statements []*types.Statement) bool {
    // Extract USING and WITH CHECK clauses
    var usingClauses []string
    var withCheckClauses []string

    for _, stmt := range statements {
        sqlUpper := strings.ToUpper(stmt.SQL)

        // Extract USING clause
        if strings.Contains(sqlUpper, "USING (") {
            usingStart := strings.Index(sqlUpper, "USING (")
            usingClause := cp.extractBalancedParens(stmt.SQL[usingStart:])
            usingClauses = append(usingClauses, usingClause)
        }

        // Extract WITH CHECK clause
        if strings.Contains(sqlUpper, "WITH CHECK (") {
            checkStart := strings.Index(sqlUpper, "WITH CHECK (")
            checkClause := cp.extractBalancedParens(stmt.SQL[checkStart:])
            withCheckClauses = append(withCheckClauses, checkClause)
        }
    }

    // All USING clauses must be identical
    if len(usingClauses) > 0 {
        first := strings.TrimSpace(usingClauses[0])
        for _, clause := range usingClauses[1:] {
            if strings.TrimSpace(clause) != first {
                return false
            }
        }
    }

    // All WITH CHECK clauses must be identical
    if len(withCheckClauses) > 0 {
        first := strings.TrimSpace(withCheckClauses[0])
        for _, clause := range withCheckClauses[1:] {
            if strings.TrimSpace(clause) != first {
                return false
            }
        }
    }

    return true
}

// extractBalancedParens extracts text within balanced parentheses
func (cp *ClerkPlugin) extractBalancedParens(text string) string {
    depth := 0
    start := strings.Index(text, "(")
    if start == -1 {
        return ""
    }

    for i := start; i < len(text); i++ {
        switch text[i] {
        case '(':
            depth++
        case ')':
            depth--
            if depth == 0 {
                return text[start : i+1]
            }
        }
    }

    return text[start:]
}

// mergeJWTV2Policies merges identical Clerk JWT v2 policies
func (cp *ClerkPlugin) mergeJWTV2Policies(statements []*types.Statement) *types.Statement {
    if len(statements) == 0 {
        return nil
    }

    // Use the first statement as base, but ensure it's CREATE POLICY
    baseStmt := statements[0]
    if baseStmt.Operation != types.OpCreate {
        // Find a CREATE statement if available
        for _, stmt := range statements {
            if stmt.Operation == types.OpCreate {
                baseStmt = stmt
                break
            }
        }
    }

    // Return the consolidated policy (keep first/CREATE version)
    return baseStmt
}

// canMergeAuthFunctions checks if Clerk auth helper functions can be merged
// Auth functions should NEVER be merged - they are critical
func (cp *ClerkPlugin) canMergeAuthFunctions(statements []*types.Statement) bool {
    // Never merge auth functions - each definition might have subtle differences
    // that affect RLS policy behavior
    return false
}

// mergeAuthFunctions merges Clerk auth functions (should never be called)
func (cp *ClerkPlugin) mergeAuthFunctions(statements []*types.Statement) *types.Statement {
    // This should never be called since canMergeAuthFunctions returns false
    // But if it is, return the last definition (most recent)
    if len(statements) == 0 {
        return nil
    }
    return statements[len(statements)-1]
}
