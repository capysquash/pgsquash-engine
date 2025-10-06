package supabase

import (
	"strings"

	"github.com/capysquash/pg-squash-engine/internal/plugins"
	"github.com/capysquash/pg-squash-engine/internal/types"
)

// GetConsolidationRules returns Supabase-specific consolidation rules
func (sp *SupabasePlugin) GetConsolidationRules() []plugins.ConsolidationRule {
    return []plugins.ConsolidationRule{
        {
            Name:        "supabase_rls_policies",
            Priority:    85,
            ObjectType:  types.TypePolicy,
            AuthPattern: types.AuthPatternRLS,
            Conflicts:   []string{"general_policy_consolidation"},
            CanMerge:    sp.canMergeRLSPolicies,
            Merge:       sp.mergeRLSPolicies,
        },
        {
            Name:        "supabase_storage_policies",
            Priority:    80,
            ObjectType:  types.TypePolicy,
            AuthPattern: types.AuthPatternStorage,
            Conflicts:   []string{"general_policy_consolidation"},
            CanMerge:    sp.canMergeStoragePolicies,
            Merge:       sp.mergeStoragePolicies,
        },
        {
            Name:        "supabase_auth_functions",
            Priority:    90,
            ObjectType:  types.TypeFunction,
            AuthPattern: types.AuthPatternSupabase,
            Conflicts:   []string{"function_deduplication"},
            CanMerge:    sp.canMergeAuthFunctions,
            Merge:       sp.mergeAuthFunctions,
        },
    }
}

// canMergeRLSPolicies checks if Supabase RLS policies can be merged
// Conservative approach: Only merge if policies are for same table and operation
func (sp *SupabasePlugin) canMergeRLSPolicies(statements []*types.Statement) bool {
    if len(statements) < 2 {
        return false
    }

    // All statements must be policies with same target table
    firstPolicy := statements[0]
    for _, stmt := range statements[1:] {
        if stmt.ObjectType != types.TypePolicy {
            return false
        }
        // Only merge policies on the same table
        if !sp.sameTargetTable(firstPolicy.SQL, stmt.SQL) {
            return false
        }
    }

    return false // Be conservative - don't merge RLS policies automatically
}

// sameTargetTable checks if two policy statements target the same table
func (sp *SupabasePlugin) sameTargetTable(sql1, sql2 string) bool {
    table1 := sp.extractPolicyTargetTable(sql1)
    table2 := sp.extractPolicyTargetTable(sql2)
    return table1 != "" && table1 == table2
}

// extractPolicyTargetTable extracts the target table from a CREATE POLICY statement
func (sp *SupabasePlugin) extractPolicyTargetTable(sql string) string {
    // Pattern: CREATE POLICY ... ON [schema.]table
    onPattern := strings.Index(strings.ToUpper(sql), " ON ")
    if onPattern == -1 {
        return ""
    }

    afterOn := sql[onPattern+4:]
    parts := strings.Fields(afterOn)
    if len(parts) > 0 {
        return strings.TrimSpace(parts[0])
    }

    return ""
}

// mergeRLSPolicies merges Supabase RLS policies (conservative - keep all)
func (sp *SupabasePlugin) mergeRLSPolicies(statements []*types.Statement) *types.Statement {
    if len(statements) == 0 {
        return nil
    }

    // Return first statement (conservative - don't actually merge)
    return statements[0]
}

// canMergeStoragePolicies checks if Supabase Storage policies can be merged
func (sp *SupabasePlugin) canMergeStoragePolicies(statements []*types.Statement) bool {
    // Never merge storage policies - they're critical for access control
    return false
}

// mergeStoragePolicies merges Supabase Storage policies (should never be called)
func (sp *SupabasePlugin) mergeStoragePolicies(statements []*types.Statement) *types.Statement {
    if len(statements) == 0 {
        return nil
    }
    return statements[0]
}

// canMergeAuthFunctions checks if Supabase auth functions can be merged
func (sp *SupabasePlugin) canMergeAuthFunctions(statements []*types.Statement) bool {
    // Never merge auth functions - they are critical for RLS
    return false
}

// mergeAuthFunctions merges Supabase auth functions (should never be called)
func (sp *SupabasePlugin) mergeAuthFunctions(statements []*types.Statement) *types.Statement {
    if len(statements) == 0 {
        return nil
    }
    // Return last definition (most recent)
    return statements[len(statements)-1]
}
