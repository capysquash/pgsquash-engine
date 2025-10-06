// Package supabase provides Supabase platform integration for pg-squash.
// It handles auth.uid(), RLS policies, Storage bucket policies, and Realtime publications.
package supabase

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/capysquash/pg-squash/internal/plugins"
	"github.com/capysquash/pg-squash/internal/types"
)

// SupabasePlugin implements Supabase platform integration
type SupabasePlugin struct {
    *plugins.BasePlugin
    config SupabaseConfig
}

// SupabaseConfig holds Supabase-specific configuration
type SupabaseConfig struct {
    Enabled            bool   `json:"enabled"`
    JWTSecret          string `json:"jwt_secret"`
    EnableRLS          bool   `json:"enable_rls"`
    StorageIntegration bool   `json:"storage_integration"`
}

// NewSupabasePlugin creates a new Supabase plugin instance
func NewSupabasePlugin() *SupabasePlugin {
    return &SupabasePlugin{
        BasePlugin: plugins.NewBasePlugin("supabase", 90), // High priority for auth
        config: SupabaseConfig{
            Enabled:            true,
            EnableRLS:          true,
            StorageIntegration: true,
        },
    }
}

// Detect checks if migrations contain Supabase patterns
// Detection patterns (in order of specificity):
//   1. auth.uid() function calls (most specific to Supabase)
//   2. auth.users table references
//   3. storage.objects or storage.buckets tables
//   4. supabase_realtime publication
func (sp *SupabasePlugin) Detect(migrations []*types.Migration) bool {
    for _, migration := range migrations {
        for _, stmt := range migration.Statements {
            sql := stmt.SQL

            // Pattern 1: auth.uid() function (Supabase-specific)
            // Note: Don't match comments mentioning "No legacy auth.uid()"
            if strings.Contains(sql, "auth.uid()") && !strings.Contains(sql, "No legacy auth.uid()") {
                return true
            }

            // Pattern 2: auth.users table
            if strings.Contains(sql, "auth.users") {
                return true
            }

            // Pattern 3: Storage tables
            storagePatterns := []string{
                "storage.objects",
                "storage.buckets",
            }
            for _, pattern := range storagePatterns {
                if strings.Contains(sql, pattern) {
                    return true
                }
            }

            // Pattern 4: Supabase Realtime publication
            if strings.Contains(sql, "supabase_realtime") {
                return true
            }

            // Pattern 5: Supabase-specific extensions or schemas
            supabasePatterns := []string{
                "CREATE EXTENSION IF NOT EXISTS pgjwt",
                "CREATE SCHEMA IF NOT EXISTS storage",
                "CREATE SCHEMA IF NOT EXISTS auth",
            }
            for _, pattern := range supabasePatterns {
                if strings.Contains(sql, pattern) {
                    return true
                }
            }
        }
    }

    return false
}

// Initialize configures the Supabase plugin with service-specific settings
func (sp *SupabasePlugin) Initialize(ctx context.Context, config interface{}) error {
    if config == nil {
        // Use defaults
        return nil
    }

    // Type assertion to extract Supabase config
    if supabaseCfg, ok := config.(SupabaseConfig); ok {
        sp.config = supabaseCfg
    } else if configMap, ok := config.(map[string]interface{}); ok {
        // Parse from map (typical JSON unmarshal result)
        if enabled, ok := configMap["enabled"].(bool); ok {
            sp.config.Enabled = enabled
        }
        if jwtSecret, ok := configMap["jwt_secret"].(string); ok {
            sp.config.JWTSecret = jwtSecret
        }
        if enableRLS, ok := configMap["enable_rls"].(bool); ok {
            sp.config.EnableRLS = enableRLS
        }
        if storageIntegration, ok := configMap["storage_integration"].(bool); ok {
            sp.config.StorageIntegration = storageIntegration
        }
    }

    return nil
}

// EnrichStatement adds Supabase-specific metadata to parsed statements
func (sp *SupabasePlugin) EnrichStatement(ctx context.Context, stmt *types.Statement) error {
    sql := stmt.SQL

    // Detect Supabase auth patterns
    if strings.Contains(sql, "auth.uid()") {
        stmt.AuthPattern = types.AuthPatternSupabase
    }

    // Detect RLS policies
    if stmt.ObjectType == types.TypePolicy && strings.Contains(sql, "auth.uid()") {
        stmt.AuthPattern = types.AuthPatternRLS
        stmt.Category = types.CategoryCritical
    }

    // Detect Storage policies
    if stmt.ObjectType == types.TypePolicy &&
       (strings.Contains(sql, "storage.objects") || strings.Contains(sql, "storage.buckets")) {
        stmt.AuthPattern = types.AuthPatternStorage
        stmt.Category = types.CategoryCritical
    }

    // Detect Realtime publications
    if stmt.ObjectType == types.TypePublication && strings.Contains(sql, "supabase_realtime") {
        stmt.Category = types.CategoryCritical
    }

    return nil
}

// DetectPatterns analyzes SQL for Supabase-specific patterns
func (sp *SupabasePlugin) DetectPatterns(sql string) []plugins.Pattern {
    var patterns []plugins.Pattern

    // Pattern 1: auth.uid() usage
    if strings.Contains(sql, "auth.uid()") {
        patterns = append(patterns, plugins.Pattern{
            Type:     plugins.PatternTypeAuth,
            Name:     "supabase_auth_uid",
            Metadata: map[string]string{
                "function": "auth.uid()",
            },
            Severity: plugins.SeverityCritical,
        })
    }

    // Pattern 2: Storage policies
    if strings.Contains(sql, "storage.objects") || strings.Contains(sql, "storage.buckets") {
        patterns = append(patterns, plugins.Pattern{
            Type:     plugins.PatternTypeStorage,
            Name:     "supabase_storage",
            Severity: plugins.SeverityHigh,
        })
    }

    // Pattern 3: Realtime publication
    if strings.Contains(sql, "supabase_realtime") {
        patterns = append(patterns, plugins.Pattern{
            Type:     plugins.PatternTypeExtension,
            Name:     "supabase_realtime",
            Severity: plugins.SeverityHigh,
        })
    }

    return patterns
}

// GetConflictingPlugins returns plugins that conflict with Supabase
func (sp *SupabasePlugin) GetConflictingPlugins() []string {
    // Supabase doesn't conflict with other plugins (lower priority than Clerk)
    return []string{}
}

// GetRequiredExtensions returns PostgreSQL extensions required by Supabase
func (sp *SupabasePlugin) GetRequiredExtensions() []string {
    extensions := []string{"uuid-ossp"}

    // Additional extensions if storage is enabled
    if sp.config.StorageIntegration {
        extensions = append(extensions, "pg_stat_statements")
    }

    return extensions
}

// ShouldPreserve determines if a statement should never be consolidated
func (sp *SupabasePlugin) ShouldPreserve(stmt *types.Statement) bool {
    // Preserve RLS policies using auth.uid()
    if stmt.ObjectType == types.TypePolicy && stmt.AuthPattern == types.AuthPatternSupabase {
        return true
    }

    // Preserve Storage policies
    if stmt.ObjectType == types.TypePolicy && stmt.AuthPattern == types.AuthPatternStorage {
        return true
    }

    // Preserve Realtime publications
    if stmt.ObjectType == types.TypePublication && strings.Contains(stmt.SQL, "supabase_realtime") {
        return true
    }

    // Preserve auth schema functions
    if stmt.ObjectType == types.TypeFunction && sp.isSupabaseAuthFunction(stmt.SQL) {
        return true
    }

    return false
}

// isSupabaseAuthFunction checks if SQL defines a Supabase authentication helper
func (sp *SupabasePlugin) isSupabaseAuthFunction(sql string) bool {
    supabaseFunctions := []string{
        "auth.uid",
        "auth.jwt",
        "auth.role",
    }

    sqlUpper := strings.ToUpper(sql)
    for _, fn := range supabaseFunctions {
        fnUpper := strings.ToUpper(fn)
        if strings.Contains(sqlUpper, "CREATE FUNCTION "+fnUpper) ||
           strings.Contains(sqlUpper, "CREATE OR REPLACE FUNCTION "+fnUpper) {
            return true
        }
    }

    return false
}

// ValidateSchema performs Supabase-specific schema validation
func (sp *SupabasePlugin) ValidateSchema(ctx context.Context, db *sql.DB) error {
    // Verify auth schema exists
    var schemaExists bool
    err := db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM pg_namespace WHERE nspname = 'auth')").Scan(&schemaExists)
    if err != nil {
        return fmt.Errorf("failed to check auth schema: %w", err)
    }

    if !schemaExists {
        return fmt.Errorf("auth schema does not exist (required for Supabase integration)")
    }

    // Verify auth.uid() function exists
    var functionExists bool
    err = db.QueryRowContext(ctx,
        "SELECT EXISTS(SELECT 1 FROM pg_proc p JOIN pg_namespace n ON p.pronamespace = n.oid WHERE n.nspname = 'auth' AND p.proname = 'uid')").
        Scan(&functionExists)
    if err != nil {
        return fmt.Errorf("failed to check auth.uid() function: %w", err)
    }

    if !functionExists {
        return fmt.Errorf("auth.uid() function does not exist (required for Supabase integration)")
    }

    // If storage integration is enabled, verify storage schema
    if sp.config.StorageIntegration {
        var storageExists bool
        err = db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM pg_namespace WHERE nspname = 'storage')").Scan(&storageExists)
        if err != nil {
            return fmt.Errorf("failed to check storage schema: %w", err)
        }

        if !storageExists {
            return fmt.Errorf("storage schema does not exist (required for Supabase Storage integration)")
        }
    }

    return nil
}

// Note: Plugin registration happens in the squasher/engine initialization
// to avoid import cycles. Do not use init() for plugin registration.
