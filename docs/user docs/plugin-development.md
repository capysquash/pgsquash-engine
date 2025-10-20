# Plugin Development Guide

Teach pgsquash about your custom auth provider, ORM, or internal Platform.

## When to Build a Plugin

**Do you need a plugin?** Only if you're hitting one of these scenarios:

### Scenario 1: Custom Auth Provider

**Problem:** Your migrations use `internal_auth.verify_session()` and pgsquash doesn't recognize it as auth-related, so it merges policies incorrectly.

**Solution:** Build an auth plugin that detects your patterns and preserves auth logic.

**Examples of plugins that solve this:**

- **Clerk plugin** - Handles JWT v2 organization claims (`auth.jwt()->'o'->>'id'`)
- **Supabase plugin** - Preserves RLS policies, storage buckets, auth helper functions

### Scenario 2: ORM-Specific Metadata

**Problem:** Your ORM adds tracking tables (like Prisma's `_prisma_migrations`) and pgsquash treats them as regular tables, breaking migration tracking.

**Solution:** Build an ORM plugin that preserves metadata tables and their specific formats.

**Examples:**

- **Prisma plugin** - Preserves `_prisma_migrations`, handles enum conflicts
- **Drizzle plugin** - Handles `GENERATED ALWAYS AS IDENTITY`, column generation

### Scenario 3: Platform-Specific Schemas

**Problem:** Your Platform injects schemas at runtime (like Supabase's `auth.users`) and validation fails because pgsquash doesn't know about them.

**Solution:** Build a Platform plugin that injects mock schemas during validation.

**Examples:**

- **Supabase plugin** - Injects `auth`, `storage`, `realtime` schemas with proper structure

### Scenario 4: Custom SQL Dialects

**Problem:** Your internal tooling generates SQL that's valid in your environment but non-standard, causing parse errors.

**Solution:** Build a transform plugin that normalizes your SQL before parsing.

## Should You Build a Plugin or Contribute to Core?

| Your Situation                                 | Recommendation                               |
| ---------------------------------------------- | -------------------------------------------- |
| Using a public service (Auth0, NextAuth, Neon) | **Contribute to core** - Others will benefit |
| Internal/proprietary system                    | **Build a plugin** - Keep it private         |
| Experimental/unstable patterns                 | **Build a plugin** - Easier to iterate       |
| Widely-used open-source tool                   | **Contribute to core** - We'll maintain it   |

**Built-in plugins you can reference:**

- `internal/plugins/auth/` - Generic auth pattern detection and handling
- `internal/plugins/clerk/` - Auth provider (JWT claims, organization support)
- `internal/plugins/supabase/` - Platform (RLS, storage, auth schemas)
- `internal/plugins/prisma/` - ORM (metadata tables, enums)
- `internal/plugins/drizzle/` - ORM (identity columns, generated columns)
- `internal/plugins/volatility/` - PostgreSQL function volatility detection and fixing

## Architecture

### Plugin Lifecycle

```
1. Registration → 2. Detection → 3. Initialization → 4-7. Processing
```

**Phase 1: Registration** (at startup)

- Plugin registers with global registry
- Stored with name and priority

**Phase 2: Detection** (per squash operation)

- Registry calls `Detect()` on all registered plugins
- Plugin scans migrations for identifying patterns
- Returns `true` if plugin should activate

**Phase 3: Initialization** (after detection)

- Registry calls `Initialize()` with config
- Plugin loads settings from `pgsquash.config.json`
- Prepares internal state

**Phase 4-7: Processing** (during migration analysis)

- **Parser phase**: `EnrichStatement()` adds metadata
- **Transform phase**: `TransformSQL()` modifies SQL
- **Consolidation phase**: `GetConsolidationRules()`, `ShouldPreserve()`
- **Validation phase**: `InjectCompatibilityLayer()`, `ValidateSchema()`

### Plugin Interface

```go
type Plugin interface {
    // Lifecycle
    Name() string
    Priority() int
    Detect(migrations []*types.Migration) bool
    Initialize(ctx context.Context, config interface{}) error

    // Parser hooks
    EnrichStatement(ctx context.Context, stmt *types.Statement) error
    DetectPatterns(sql string) []Pattern
    DetectAuthPattern(stmt *types.Statement) string

    // Transformation hooks
    TransformSQL(ctx context.Context, sql string) (string, error)
    InjectCompatibilityLayer(ctx context.Context) string
    FixFunctionVolatility(ctx context.Context, functionSQL string) (string, error)

    // Validation hooks
    ValidateSchema(ctx context.Context, db *sql.DB) error
    GetRequiredExtensions() []string

    // Consolidation hooks
    GetConsolidationRules() []ConsolidationRule
    ShouldPreserve(stmt *types.Statement) bool
    GetConflictingPlugins() []string
}
```

See [Plugin Interface Reference](#plugin-interface-reference) for detailed documentation.

## Quick Start

### Step 1: Create Plugin Package

```bash
mkdir internal/plugins/myauth
```

**Directory structure:**

```
internal/plugins/myauth/
├── myauth.go           # Main plugin implementation
├── consolidation.go    # Consolidation rules (optional)
└── transformations.go  # SQL transformations (optional)
```

### Step 2: Implement Plugin

**File:** `internal/plugins/myauth/myauth.go`

```go
package myauth

import (
    "context"
    "strings"

    "github.com/CAPYSQUASH/pgsquash-engine/internal/plugins"
    "github.com/CAPYSQUASH/pgsquash-engine/internal/types"
)

// MyAuthPlugin implements authentication integration
type MyAuthPlugin struct {
    *plugins.BasePlugin
    config MyAuthConfig
}

// MyAuthConfig holds plugin-specific configuration
type MyAuthConfig struct {
    Enabled        bool   `json:"enabled"`
    APIVersion     string `json:"api_version"`
    PreserveHelper bool   `json:"preserve_helper"`
}

// NewMyAuthPlugin creates a new plugin instance
func NewMyAuthPlugin() *MyAuthPlugin {
    return &MyAuthPlugin{
        BasePlugin: plugins.NewBasePlugin("myauth", 92), // Auth tier priority
        config: MyAuthConfig{
            Enabled:        true,
            APIVersion:     "v1",
            PreserveHelper: true,
        },
    }
}

// Detect checks if migrations contain MyAuth patterns
func (p *MyAuthPlugin) Detect(migrations []*types.Migration) bool {
    for _, migration := range migrations {
        for _, stmt := range migration.Statements {
            sql := stmt.SQL

            // Pattern 1: MyAuth helper function
            if strings.Contains(sql, "myauth.get_user_id()") {
                return true
            }

            // Pattern 2: MyAuth schema
            if strings.Contains(sql, "CREATE SCHEMA myauth") {
                return true
            }

            // Pattern 3: MyAuth claims access
            if strings.Contains(sql, "myauth.claims()") {
                return true
            }
        }
    }

    return false
}

// Initialize configures the plugin with user settings
func (p *MyAuthPlugin) Initialize(ctx context.Context, config interface{}) error {
    if config == nil {
        return nil // Use defaults
    }

    // Parse config (if provided as map[string]interface{})
    if configMap, ok := config.(map[string]interface{}); ok {
        if enabled, ok := configMap["enabled"].(bool); ok {
            p.config.Enabled = enabled
        }
        if apiVersion, ok := configMap["api_version"].(string); ok {
            p.config.APIVersion = apiVersion
        }
    }

    return nil
}

// InjectCompatibilityLayer provides mock auth schema for validation
func (p *MyAuthPlugin) InjectCompatibilityLayer(ctx context.Context) string {
    return `
-- MyAuth compatibility layer for validation
CREATE SCHEMA IF NOT EXISTS myauth;

CREATE OR REPLACE FUNCTION myauth.get_user_id()
RETURNS TEXT
LANGUAGE sql STABLE
AS $$
    SELECT 'user_test_id'::text;
$$;

CREATE OR REPLACE FUNCTION myauth.claims()
RETURNS JSONB
LANGUAGE sql STABLE
AS $$
    SELECT '{"user_id": "test", "role": "user"}'::jsonb;
$$;
`
}

// ShouldPreserve marks auth helper functions as critical
func (p *MyAuthPlugin) ShouldPreserve(stmt *types.Statement) bool {
    if stmt.ObjectType != types.TypeFunction {
        return false
    }

    // Never consolidate auth helper functions
    authFunctions := []string{
        "myauth.get_user_id",
        "myauth.claims",
        "myauth.verify_token",
    }

    for _, fn := range authFunctions {
        if strings.Contains(stmt.ObjectName, fn) {
            return true
        }
    }

    return false
}

// GetRequiredExtensions returns PostgreSQL extensions needed
func (p *MyAuthPlugin) GetRequiredExtensions() []string {
    return []string{"pgcrypto"} // For token hashing
}
```

### Step 3: Register Plugin

**File:** `cmd/pgsquash/main.go`

```go
package main

import (
    "github.com/CAPYSQUASH/pgsquash-engine/internal/cli"
    "github.com/CAPYSQUASH/pgsquash-engine/internal/plugins"

    // Import built-in plugins
    "github.com/CAPYSQUASH/pgsquash-engine/internal/plugins/clerk"
    "github.com/CAPYSQUASH/pgsquash-engine/internal/plugins/supabase"
    "github.com/CAPYSQUASH/pgsquash-engine/internal/plugins/prisma"
    "github.com/CAPYSQUASH/pgsquash-engine/internal/plugins/drizzle"

    // Import your plugin
    "github.com/CAPYSQUASH/pgsquash-engine/internal/plugins/myauth"
)

func init() {
    // Register built-in plugins
    plugins.Register(clerk.NewClerkPlugin())
    plugins.Register(supabase.NewSupabasePlugin())
    plugins.Register(prisma.NewPrismaPlugin())
    plugins.Register(drizzle.NewDrizzlePlugin())

    // Register your plugin
    plugins.Register(myauth.NewMyAuthPlugin())
}

func main() {
    cli.Execute()
}
```

### Step 4: Build and Test

```bash
# Build
go build -o pgsquash cmd/pgsquash/main.go

# Test detection
echo "CREATE FUNCTION myauth.get_user_id() RETURNS TEXT AS \$\$ SELECT 'test'; \$\$ LANGUAGE sql;" > test.sql
./pgsquash analyze test.sql

# Expected output:
# [plugins] Detecting plugins from 1 migrations...
# [plugins] Detected: myauth
# [plugins] Initialized: myauth
```

## Plugin Interface Reference

### Required Methods

#### `Name() string`

Returns unique plugin identifier.

**Usage:** Registry uses this for conflict resolution and logging.

**Example:**

```go
func (p *MyAuthPlugin) Name() string {
    return "myauth"
}
```

#### `Priority() int`

Returns plugin priority for conflict resolution (higher = higher priority).

**Priority tiers:**

- **90-100**: Authentication services (Clerk: 95, Supabase: 90)
- **70-89**: ORMs (Prisma: 75, Drizzle: 75)
- **50-69**: Platforms (future: Neon, Railway)
- **0-49**: Utilities

**Example:**

```go
func (p *MyAuthPlugin) Priority() int {
    return 92 // Auth tier
}
```

**Conflict resolution:**

If two plugins detect the same migration:

1. Both plugins run `Detect()`
2. Registry sorts by priority (highest first)
3. Higher priority plugin activates
4. Lower priority plugin excluded

#### `Detect(migrations []*types.Migration) bool`

Analyzes migrations to determine if plugin should activate.

**Parameters:**

- `migrations`: All migration files being processed

**Returns:**

- `true`: Plugin should activate
- `false`: Plugin not applicable

**Best practices:**

- Check multiple patterns (most specific first)
- Avoid false positives (could break other projects)
- Fast execution (called for every plugin on every squash)

**Example:**

```go
func (p *MyAuthPlugin) Detect(migrations []*types.Migration) bool {
    for _, migration := range migrations {
        for _, stmt := range migration.Statements {
            sql := stmt.SQL

            // Pattern 1: Service-specific function (most reliable)
            if strings.Contains(sql, "myauth.get_user_id()") {
                return true
            }

            // Pattern 2: Service schema
            if strings.Contains(sql, "CREATE SCHEMA myauth") {
                return true
            }

            // Pattern 3: Service-specific comments
            if strings.Contains(sql, "-- MyAuth migration") {
                return true
            }
        }
    }

    return false
}
```

**Common patterns to detect:**

- Service-specific functions (`auth.uid()`, `clerk_user_id()`)
- Service schemas (`CREATE SCHEMA auth`, `CREATE SCHEMA storage`)
- Service tables (`_prisma_migrations`, `auth.users`)
- Service comments (`-- CreateTable`, `-- Supabase migration`)
- Service-specific syntax patterns

#### `Initialize(ctx context.Context, config interface{}) error`

Configures plugin with user settings from `pgsquash.config.json`.

**Parameters:**

- `ctx`: Context for cancellation
- `config`: Plugin configuration (from config file)

**Returns:**

- `error`: If configuration is invalid

**Configuration lookup:**

Registry checks multiple paths:

1. `plugins.myauth.*`
2. `myauth.*`
3. `third_party_integrations.myauth_integration.*`

**Example:**

```go
func (p *MyAuthPlugin) Initialize(ctx context.Context, config interface{}) error {
    if config == nil {
        return nil // Use defaults
    }

    configMap, ok := config.(map[string]interface{})
    if !ok {
        return fmt.Errorf("invalid config type: %T", config)
    }

    // Parse configuration
    if enabled, ok := configMap["enabled"].(bool); ok {
        p.config.Enabled = enabled
    }

    if apiVersion, ok := configMap["api_version"].(string); ok {
        p.config.APIVersion = apiVersion
    }

    // Validate config
    if p.config.APIVersion != "v1" && p.config.APIVersion != "v2" {
        return fmt.Errorf("unsupported API version: %s", p.config.APIVersion)
    }

    return nil
}
```

**Config file example:**

```json
{
  "plugins": {
    "myauth": {
      "enabled": true,
      "api_version": "v2",
      "preserve_helper": true
    }
  }
}
```

### Optional Methods (BasePlugin Defaults)

All optional methods have default implementations in `BasePlugin`:

```go
type MyAuthPlugin struct {
    *plugins.BasePlugin  // Embeds defaults
    config MyAuthConfig
}
```

Override only the methods you need.

#### `EnrichStatement(ctx context.Context, stmt *types.Statement) error`

Adds plugin-specific metadata to parsed statements.

**When called:** After pg\_query parsing, before tracking.

**Use cases:**

- Mark auth functions with volatility hints
- Tag ORM tables for preservation
- Add cross-reference information

**Example:**

```go
func (p *MyAuthPlugin) EnrichStatement(ctx context.Context, stmt *types.Statement) error {
    // Mark auth functions as STABLE
    if stmt.ObjectType == types.TypeFunction {
        if strings.Contains(stmt.ObjectName, "myauth.") {
            stmt.Metadata["volatility"] = "STABLE"
            stmt.Metadata["security"] = "DEFINER"
        }
    }

    // Mark auth policies for preservation
    if stmt.ObjectType == types.TypePolicy {
        if strings.Contains(stmt.SQL, "myauth.get_user_id()") {
            stmt.Metadata["preserve"] = "true"
            stmt.Metadata["reason"] = "auth_policy"
        }
    }

    return nil
}
```

#### `TransformSQL(ctx context.Context, sql string) (string, error)`

Modifies SQL before pg\_query parsing.

**When called:** Before pg\_query parsing.

**Use cases:**

- Fix service-specific syntax
- Add missing keywords
- Convert proprietary SQL to standard

**Example:**

```go
func (p *MyAuthPlugin) TransformSQL(ctx context.Context, sql string) (string, error) {
    // Add STABLE marker to auth functions
    if strings.Contains(sql, "CREATE FUNCTION myauth.") {
        if !strings.Contains(sql, "STABLE") &&
           !strings.Contains(sql, "VOLATILE") &&
           !strings.Contains(sql, "IMMUTABLE") {
            // Insert STABLE before AS keyword
            sql = strings.Replace(sql, " AS ", " STABLE AS ", 1)
        }
    }

    return sql, nil
}
```

#### `InjectCompatibilityLayer(ctx context.Context) string`

Returns SQL to execute before migrations for validation.

**When called:** During validation setup.

**Use cases:**

- Create mock auth functions
- Create service schemas
- Create placeholder roles

**Example:**

```go
func (p *MyAuthPlugin) InjectCompatibilityLayer(ctx context.Context) string {
    return `
-- MyAuth compatibility layer
CREATE SCHEMA IF NOT EXISTS myauth;

CREATE OR REPLACE FUNCTION myauth.get_user_id()
RETURNS TEXT
LANGUAGE sql STABLE
AS $$ SELECT 'test_user'::text; $$;

CREATE OR REPLACE FUNCTION myauth.claims()
RETURNS JSONB
LANGUAGE sql STABLE
AS $$ SELECT '{}'::jsonb; $$;

CREATE ROLE IF NOT EXISTS authenticated;
GRANT USAGE ON SCHEMA myauth TO authenticated;
`
}
```

#### `GetConsolidationRules() []ConsolidationRule`

Returns custom consolidation rules for plugin-specific patterns.

**When called:** During consolidation phase.

**Use cases:**

- Define how to merge auth policies
- Protect critical statements from consolidation
- Custom merging logic for service patterns

**Example:**

See [Advanced: Consolidation Rules](#advanced-consolidation-rules).

#### `ShouldPreserve(stmt *types.Statement) bool`

Determines if statement should never be consolidated.

**When called:** During consolidation analysis.

**Returns:**

- `true`: Statement is critical, never consolidate
- `false`: Statement can be consolidated (if safe)

**Example:**

```go
func (p *MyAuthPlugin) ShouldPreserve(stmt *types.Statement) bool {
    // Preserve all auth helper functions
    if stmt.ObjectType == types.TypeFunction {
        authFunctions := []string{
            "myauth.get_user_id",
            "myauth.claims",
            "myauth.verify_token",
        }

        for _, fn := range authFunctions {
            if strings.Contains(stmt.ObjectName, fn) {
                return true
            }
        }
    }

    // Preserve auth policies
    if stmt.ObjectType == types.TypePolicy {
        if strings.Contains(stmt.SQL, "myauth.") {
            return true
        }
    }

    return false
}
```

#### `ValidateSchema(ctx context.Context, db *sql.DB) error`

Performs plugin-specific schema validation.

**When called:** During validation phase (with database connection).

**Use cases:**

- Verify auth functions exist
- Check RLS policies are correct
- Validate service-specific constraints

**Example:**

```go
func (p *MyAuthPlugin) ValidateSchema(ctx context.Context, db *sql.DB) error {
    // Check if auth schema exists
    var exists bool
    err := db.QueryRowContext(ctx,
        "SELECT EXISTS(SELECT 1 FROM pg_namespace WHERE nspname = 'myauth')").Scan(&exists)
    if err != nil {
        return fmt.Errorf("failed to check myauth schema: %w", err)
    }

    if !exists {
        return fmt.Errorf("myauth schema not found (required for auth functions)")
    }

    // Check required functions exist
    requiredFunctions := []string{"get_user_id", "claims", "verify_token"}
    for _, fn := range requiredFunctions {
        var fnExists bool
        err := db.QueryRowContext(ctx,
            "SELECT EXISTS(SELECT 1 FROM pg_proc p JOIN pg_namespace n ON p.pronamespace = n.oid WHERE n.nspname = 'myauth' AND p.proname = $1)",
            fn).Scan(&fnExists)
        if err != nil {
            return fmt.Errorf("failed to check function %s: %w", fn, err)
        }

        if !fnExists {
            return fmt.Errorf("required function myauth.%s not found", fn)
        }
    }

    return nil
}
```

#### `GetRequiredExtensions() []string`

Returns PostgreSQL extensions required by plugin.

**When called:** During validation setup.

**Example:**

```go
func (p *MyAuthPlugin) GetRequiredExtensions() []string {
    return []string{
        "pgcrypto",  // For password hashing
        "uuid-ossp", // For UUID generation
    }
}
```

#### `GetConflictingPlugins() []string`

Returns names of plugins that conflict with this one.

**When called:** During detection phase.

**Use cases:**

- Prevent multiple auth providers from activating
- Resolve ORM conflicts

**Example:**

```go
func (p *MyAuthPlugin) GetConflictingPlugins() []string {
    // MyAuth conflicts with other auth providers
    return []string{"clerk", "supabase", "auth0"}
}
```

**Conflict resolution:**

```
Detected: clerk (95) + myauth (92) + supabase (90)
Clerk conflicts with: [myauth, supabase]
Result: clerk active, myauth and supabase excluded
```

## Advanced: Consolidation Rules

Consolidation rules define how plugin-specific statements can be merged.

### Rule Structure

```go
type ConsolidationRule struct {
    Name        string                              // Rule identifier
    Priority    int                                 // Higher = higher priority
    ObjectType  types.ObjectType                   // Target object type
    AuthPattern types.AuthPatternType              // Target auth pattern
    Conflicts   []string                            // Conflicting rule names
    CanMerge    func([]*types.Statement) bool      // Check if mergeable
    Merge       func([]*types.Statement) *types.Statement // Perform merge
}
```

### Example: Auth Policy Consolidation

**File:** `internal/plugins/myauth/consolidation.go`

```go
package myauth

import (
    "strings"

    "github.com/CAPYSQUASH/pgsquash-engine/internal/plugins"
    "github.com/CAPYSQUASH/pgsquash-engine/internal/types"
)

// GetConsolidationRules returns MyAuth-specific rules
func (p *MyAuthPlugin) GetConsolidationRules() []plugins.ConsolidationRule {
    return []plugins.ConsolidationRule{
        {
            Name:        "myauth_policy_consolidation",
            Priority:    80,
            ObjectType:  types.TypePolicy,
            AuthPattern: "myauth_pattern",
            Conflicts:   []string{},
            CanMerge:    p.canMergeAuthPolicies,
            Merge:       p.mergeAuthPolicies,
        },
    }
}

// canMergeAuthPolicies checks if policies can be safely merged
func (p *MyAuthPlugin) canMergeAuthPolicies(statements []*types.Statement) bool {
    if len(statements) < 2 {
        return false
    }

    // All must be policies
    for _, stmt := range statements {
        if stmt.ObjectType != types.TypePolicy {
            return false
        }
    }

    // All must target same table
    firstTable := extractPolicyTable(statements[0].SQL)
    for _, stmt := range statements[1:] {
        if extractPolicyTable(stmt.SQL) != firstTable {
            return false
        }
    }

    // All must have same auth logic
    firstLogic := extractAuthLogic(statements[0].SQL)
    for _, stmt := range statements[1:] {
        if extractAuthLogic(stmt.SQL) != firstLogic {
            return false
        }
    }

    return true
}

// mergeAuthPolicies merges multiple policies into one
func (p *MyAuthPlugin) mergeAuthPolicies(statements []*types.Statement) *types.Statement {
    if len(statements) == 0 {
        return nil
    }

    // Use first statement as base
    merged := *statements[0]

    // Combine policy names
    var policyNames []string
    for _, stmt := range statements {
        policyNames = append(policyNames, stmt.ObjectName)
    }
    merged.ObjectName = strings.Join(policyNames, "_")

    // Note: SQL consolidation logic here...
    // (combine policy commands while preserving auth logic)

    return &merged
}

// Helper: Extract table name from policy SQL
func extractPolicyTable(sql string) string {
    // Pattern: CREATE POLICY name ON table_name
    parts := strings.Split(sql, " ON ")
    if len(parts) < 2 {
        return ""
    }
    tablePart := strings.TrimSpace(parts[1])
    return strings.Fields(tablePart)[0]
}

// Helper: Extract auth logic from policy SQL
func extractAuthLogic(sql string) string {
    // Pattern: USING (auth_expression)
    if !strings.Contains(sql, "USING (") {
        return ""
    }
    start := strings.Index(sql, "USING (") + 7
    // Find matching closing paren...
    return sql[start : start+50] // Simplified
}
```

## Testing Plugins

### Unit Tests

**File:** `internal/plugins/myauth/myauth_test.go`

```go
package myauth

import (
    "testing"

    "github.com/CAPYSQUASH/pgsquash-engine/internal/types"
    "github.com/stretchr/testify/assert"
)

func TestMyAuthPlugin_Detect(t *testing.T) {
    plugin := NewMyAuthPlugin()

    tests := []struct {
        name     string
        sql      string
        expected bool
    }{
        {
            name:     "detects myauth function",
            sql:      "SELECT myauth.get_user_id();",
            expected: true,
        },
        {
            name:     "detects myauth schema",
            sql:      "CREATE SCHEMA myauth;",
            expected: true,
        },
        {
            name:     "ignores other patterns",
            sql:      "CREATE TABLE users (id SERIAL);",
            expected: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            migrations := []*types.Migration{
                {
                    Statements: []types.Statement{
                        {SQL: tt.sql},
                    },
                },
            }

            result := plugin.Detect(migrations)
            assert.Equal(t, tt.expected, result)
        })
    }
}

func TestMyAuthPlugin_ShouldPreserve(t *testing.T) {
    plugin := NewMyAuthPlugin()

    // Should preserve auth functions
    stmt := &types.Statement{
        ObjectType: types.TypeFunction,
        ObjectName: "myauth.get_user_id",
    }
    assert.True(t, plugin.ShouldPreserve(stmt))

    // Should not preserve regular functions
    stmt2 := &types.Statement{
        ObjectType: types.TypeFunction,
        ObjectName: "public.calculate_total",
    }
    assert.False(t, plugin.ShouldPreserve(stmt2))
}
```

### Integration Tests

```bash
# Create test migrations
mkdir -p test/myauth
cat > test/myauth/001_setup.sql <<EOF
CREATE SCHEMA myauth;

CREATE FUNCTION myauth.get_user_id()
RETURNS TEXT
LANGUAGE sql
AS \$\$ SELECT 'test'; \$\$;
EOF

cat > test/myauth/002_policies.sql <<EOF
CREATE POLICY user_policy ON users
USING (myauth.get_user_id() = user_id);
EOF

# Test detection
./pgsquash analyze test/myauth/*.sql
# Expected: [plugins] Detected: myauth

# Test squashing
./pgsquash squash test/myauth/*.sql --output test/squashed.sql

# Verify: Check that auth function wasn't consolidated
grep "myauth.get_user_id" test/squashed.sql
```

## Configuration

### Plugin-Specific Config

**File:** `pgsquash.config.json`

```json
{
  "plugins": {
    "auto_detect": true,
    "enabled_plugins": ["myauth"],
    "disabled_plugins": [],
    "myauth": {
      "enabled": true,
      "api_version": "v2",
      "preserve_helper": true,
      "validation": {
        "strict": true,
        "required_functions": [
          "get_user_id",
          "claims",
          "verify_token"
        ]
      }
    }
  }
}
```

### Disabling Auto-Detection

```json
{
  "plugins": {
    "auto_detect": false,
    "enabled_plugins": ["myauth", "prisma"],
    "disabled_plugins": ["clerk", "supabase"]
  }
}
```

## Best Practices

### 1. Pattern Detection

**Do:**

- Check multiple patterns (most specific first)
- Use service-specific identifiers
- Test with real-world migrations

**Don't:**

- Use overly broad patterns (false positives)
- Check only one pattern (fragile)
- Ignore edge cases

**Example:**

```go
// ❌ Too broad - will match unrelated migrations
if strings.Contains(sql, "auth") {
    return true
}

// ✅ Specific - unlikely to false-match
if strings.Contains(sql, "myauth.get_user_id()") ||
   strings.Contains(sql, "CREATE SCHEMA myauth") ||
   (strings.Contains(sql, "-- MyAuth") && strings.Contains(sql, "migration")) {
    return true
}
```

### 2. Priority Assignment

**Priority tiers:**

```go
// Auth services: 90-100
NewBasePlugin("myauth", 92)

// ORMs: 70-89
NewBasePlugin("myorm", 75)

// Platforms: 50-69
NewBasePlugin("myPlatform", 60)

// Utilities: 0-49
NewBasePlugin("myutil", 30)
```

**Reasoning:**

- Auth plugins must run first (critical for security)
- ORMs need consistent schema handling
- Platform plugins configure environment
- Utilities provide optional enhancements

### 3. Consolidation Safety

**Conservative approach:**

```go
func (p *MyAuthPlugin) ShouldPreserve(stmt *types.Statement) bool {
    // When in doubt, preserve
    if stmt.ObjectType == types.TypeFunction {
        if strings.Contains(stmt.ObjectName, "myauth.") {
            return true // Never consolidate auth functions
        }
    }
    return false
}
```

**Rationale:** Breaking auth = security vulnerability. Better to be conservative.

### 4. Error Handling

**Fail gracefully:**

```go
func (p *MyAuthPlugin) Initialize(ctx context.Context, config interface{}) error {
    if config == nil {
        // Use defaults, don't fail
        return nil
    }

    // Validate but provide helpful messages
    if err := p.parseConfig(config); err != nil {
        return fmt.Errorf("myauth plugin config invalid: %w (see docs/plugin-development.md)", err)
    }

    return nil
}
```

### 5. Testing

**Test matrix:**

```go
func TestMyAuthPlugin(t *testing.T) {
    tests := []struct {
        name        string
        migrations  []*types.Migration
        shouldDetect bool
        shouldError  bool
    }{
        {"with myauth schema", createMigration("CREATE SCHEMA myauth;"), true, false},
        {"with myauth function", createMigration("SELECT myauth.get_user_id();"), true, false},
        {"without myauth", createMigration("CREATE TABLE users (id INT);"), false, false},
        {"with invalid syntax", createMigration("INVALID SQL"), false, true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            plugin := NewMyAuthPlugin()
            detected := plugin.Detect(tt.migrations)
            assert.Equal(t, tt.shouldDetect, detected)
        })
    }
}
```

## Examples

### Example 1: Minimal Auth Plugin

Simplest possible plugin - just detection and preservation.

```go
package simpleauth

import (
    "strings"

    "github.com/CAPYSQUASH/pgsquash-engine/internal/plugins"
    "github.com/CAPYSQUASH/pgsquash-engine/internal/types"
)

type SimpleAuthPlugin struct {
    *plugins.BasePlugin
}

func NewSimpleAuthPlugin() *SimpleAuthPlugin {
    return &SimpleAuthPlugin{
        BasePlugin: plugins.NewBasePlugin("simpleauth", 88),
    }
}

func (p *SimpleAuthPlugin) Detect(migrations []*types.Migration) bool {
    for _, m := range migrations {
        for _, stmt := range m.Statements {
            if strings.Contains(stmt.SQL, "simpleauth.") {
                return true
            }
        }
    }
    return false
}

func (p *SimpleAuthPlugin) ShouldPreserve(stmt *types.Statement) bool {
    return stmt.ObjectType == types.TypeFunction &&
           strings.Contains(stmt.ObjectName, "simpleauth.")
}

// All other methods use BasePlugin defaults
```

### Example 2: ORM Plugin with Transformations

ORM plugin that normalizes syntax.

```go
package myorm

import (
    "context"
    "regexp"
    "strings"

    "github.com/CAPYSQUASH/pgsquash-engine/internal/plugins"
    "github.com/CAPYSQUASH/pgsquash-engine/internal/types"
)

type MyORMPlugin struct {
    *plugins.BasePlugin
}

func NewMyORMPlugin() *MyORMPlugin {
    return &MyORMPlugin{
        BasePlugin: plugins.NewBasePlugin("myorm", 72),
    }
}

func (p *MyORMPlugin) Detect(migrations []*types.Migration) bool {
    for _, m := range migrations {
        // Check for _myorm_migrations table
        for _, stmt := range m.Statements {
            if strings.Contains(stmt.SQL, "_myorm_migrations") {
                return true
            }
        }
    }
    return false
}

func (p *MyORMPlugin) TransformSQL(ctx context.Context, sql string) (string, error) {
    // Transform MyORM's "AUTO INCREMENT" to PostgreSQL "SERIAL"
    autoIncrementPattern := regexp.MustCompile(`(?i)INT\s+AUTO\s+INCREMENT`)
    sql = autoIncrementPattern.ReplaceAllString(sql, "SERIAL")

    // Transform MyORM's TEXT(65535) to PostgreSQL TEXT
    textSizePattern := regexp.MustCompile(`(?i)TEXT\(\d+\)`)
    sql = textSizePattern.ReplaceAllString(sql, "TEXT")

    return sql, nil
}

func (p *MyORMPlugin) ShouldPreserve(stmt *types.Statement) bool {
    // Preserve ORM metadata table
    if stmt.ObjectType == types.TypeTable {
        if strings.Contains(stmt.ObjectName, "_myorm_migrations") {
            return true
        }
    }
    return false
}
```

### Example 3: Platform Plugin with Validation

Platform plugin that injects compatibility layer and validates schema.

```go
package myPlatform

import (
    "context"
    "database/sql"
    "fmt"
    "strings"

    "github.com/CAPYSQUASH/pgsquash-engine/internal/plugins"
    "github.com/CAPYSQUASH/pgsquash-engine/internal/types"
)

type MyPlatformPlugin struct {
    *plugins.BasePlugin
}

func NewMyPlatformPlugin() *MyPlatformPlugin {
    return &MyPlatformPlugin{
        BasePlugin: plugins.NewBasePlugin("myPlatform", 55),
    }
}

func (p *MyPlatformPlugin) Detect(migrations []*types.Migration) bool {
    for _, m := range migrations {
        for _, stmt := range m.Statements {
            // Platform-specific role
            if strings.Contains(stmt.SQL, "CREATE ROLE myPlatform_user") {
                return true
            }
            // Platform-specific extension
            if strings.Contains(stmt.SQL, "CREATE EXTENSION myPlatform_pgvector") {
                return true
            }
        }
    }
    return false
}

func (p *MyPlatformPlugin) InjectCompatibilityLayer(ctx context.Context) string {
    return `
-- MyPlatform compatibility layer
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE ROLE IF NOT EXISTS myPlatform_user;
CREATE ROLE IF NOT EXISTS myPlatform_admin;

GRANT myPlatform_user TO CURRENT_USER;
`
}

func (p *MyPlatformPlugin) ValidateSchema(ctx context.Context, db *sql.DB) error {
    // Check required roles exist
    var exists bool
    err := db.QueryRowContext(ctx,
        "SELECT EXISTS(SELECT 1 FROM pg_roles WHERE rolname = 'myPlatform_user')").
        Scan(&exists)
    if err != nil {
        return fmt.Errorf("failed to check myPlatform_user role: %w", err)
    }

    if !exists {
        return fmt.Errorf("myPlatform_user role not found (required by MyPlatform)")
    }

    return nil
}

func (p *MyPlatformPlugin) GetRequiredExtensions() []string {
    return []string{"uuid-ossp", "myPlatform_pgvector"}
}
```

## Troubleshooting

### Plugin Not Detecting

**Problem:** Plugin doesn't activate despite matching patterns

**Solutions:**

1. **Enable debug logging:**
   ```bash
   PGSQUASH_LOG_LEVEL=debug ./pgsquash squash migrations/*.sql
   ```

2. **Check detection logic:**
   ```go
   // Add logging to Detect()
   func (p *MyAuthPlugin) Detect(migrations []*types.Migration) bool {
       fmt.Printf("[DEBUG] Checking %d migrations\n", len(migrations))
       for _, m := range migrations {
           for _, stmt := range m.Statements {
               if strings.Contains(stmt.SQL, "myauth.") {
                   fmt.Printf("[DEBUG] Found myauth pattern in: %s\n", m.Filename)
                   return true
               }
           }
       }
       fmt.Printf("[DEBUG] No myauth patterns found\n")
       return false
   }
   ```

3. **Check priority conflicts:**
   - Higher priority plugin may be excluding yours
   - Check `GetConflictingPlugins()` on other plugins

### Plugin Detected But Not Active

**Problem:** Plugin detected but methods not being called

**Solutions:**

1. **Check initialization:**
   ```bash
   # Look for initialization message
   ./pgsquash squash migrations/*.sql 2>&1 | grep "Initialized"
   ```

2. **Check config:**
   ```json
   {
     "plugins": {
       "myauth": {
         "enabled": true  // Must be true
       }
     }
   }
   ```

3. **Check conflicts:**
   ```bash
   # Higher priority plugin may have excluded yours
   ./pgsquash squash migrations/*.sql 2>&1 | grep "Excluding"
   ```

### Consolidation Not Working

**Problem:** Plugin rules not being applied

**Solutions:**

1. **Check rule priority:**
   ```go
   // Higher priority = runs first
   ConsolidationRule{
       Priority: 100, // Higher than default rules
       // ...
   }
   ```

2. **Check CanMerge logic:**
   ```go
   func (p *MyAuthPlugin) canMerge(statements []*types.Statement) bool {
       fmt.Printf("[DEBUG] Checking merge for %d statements\n", len(statements))
       // Add debug logging
       return result
   }
   ```

3. **Check ShouldPreserve:**
   ```go
   // ShouldPreserve = true overrides all consolidation
   func (p *MyAuthPlugin) ShouldPreserve(stmt *types.Statement) bool {
       preserve := /* your logic */
       fmt.Printf("[DEBUG] ShouldPreserve(%s) = %v\n", stmt.ObjectName, preserve)
       return preserve
   }
   ```

## Built-in Plugin Reference

pgsquash includes several built-in plugins that demonstrate best practices and handle common third-party integrations.

### Auth Plugin (`internal/plugins/auth/`)

**Purpose:** Generic authentication pattern detection and handling for custom auth systems.

**Priority:** 95 (runs before most plugins)

**Detects:**
- Custom auth schemas and tables
- Session management tables
- Authentication helper functions
- JWT validation functions
- Custom user tables with auth patterns

**Use Cases:**
- Projects with custom authentication systems
- Internal auth implementations
- Auth systems that don't use Clerk, Supabase, or Auth0

**Detection Patterns:**
```sql
-- Detects tables with auth-related names
CREATE TABLE user_sessions (...)
CREATE TABLE authentication_tokens (...)
CREATE TABLE api_keys (...)

-- Detects auth functions
CREATE FUNCTION verify_jwt_token(...) ...
CREATE FUNCTION validate_session(...) ...
```

**Configuration:**
```json
{
  "plugins": {
    "enabled_plugins": ["auth"],
    "auth": {
      "enabled": true,
      "detect_custom_patterns": true
    }
  }
}
```

**What It Does:**
- Marks auth-related tables and functions for preservation
- Groups authentication objects together in output
- Prevents unsafe consolidation of session tables
- Maintains auth function signatures

**Example:**
```bash
# Enable auth plugin explicitly
pgsquash squash migrations/*.sql --config pgsquash.config.json

# Output will preserve:
-- === AUTHENTICATION ===
CREATE TABLE user_sessions (...);
CREATE FUNCTION verify_session_token(...) ...;
-- (Not merged with other tables)
```

---

### Clerk Plugin (`internal/plugins/clerk/`)

**Purpose:** Handle Clerk authentication patterns, JWT v2 tokens, and organization support.

**Priority:** 90

**Detects:**
- Clerk JWT claims (`auth.jwt()->>'id'`)
- Organization metadata (`auth.jwt()->'o'->>'id'`)
- Public metadata support
- Clerk user ID patterns

**Detection Patterns:**
```sql
-- JWT v2 organization claims
WHERE auth.jwt()->'o'->>'id' = organization_id

-- Clerk user ID references
user_id VARCHAR(255) -- Clerk uses string IDs

-- Public metadata
metadata JSONB -- Clerk public_metadata
```

**Configuration:**
```json
{
  "third_party_integrations": {
    "clerk_integration": {
      "enabled": true,
      "jwt_version": "v2",
      "organization_support": true,
      "public_metadata_support": true
    }
  }
}
```

**What It Does:**
- Preserves Clerk-specific JWT claim syntax
- Handles organization-scoped RLS policies
- Maintains public metadata JSONB columns
- Groups Clerk auth patterns together

**Reference:** [Clerk Documentation](https://clerk.com/docs)

---

### Supabase Plugin (`internal/plugins/supabase/`)

**Purpose:** Handle Supabase-specific patterns including RLS policies, storage buckets, and auth schemas.

**Priority:** 90

**Detects:**
- `auth.users` references
- `storage.buckets` and `storage.objects`
- RLS policies using `auth.uid()`
- Supabase realtime patterns

**Detection Patterns:**
```sql
-- Auth schema references
FOREIGN KEY (user_id) REFERENCES auth.users(id)

-- Storage integration
CREATE POLICY storage_policy ON storage.objects
  FOR SELECT USING (auth.uid() = owner_id);

-- RLS with auth.uid()
CREATE POLICY users_policy ON users
  FOR SELECT USING (auth.uid() = id);
```

**Configuration:**
```json
{
  "third_party_integrations": {
    "supabase_integration": {
      "enabled": true,
      "enable_rls": true,
      "storage_integration": true
    }
  }
}
```

**Validation Support:**

The Supabase plugin injects compatibility layers during validation:

```sql
-- Auto-generated auth.users stub
CREATE SCHEMA IF NOT EXISTS auth;
CREATE TABLE auth.users (
    id UUID PRIMARY KEY,
    email VARCHAR(255)
);

-- Auth helper functions
CREATE FUNCTION auth.uid() RETURNS UUID ...
CREATE FUNCTION auth.jwt() RETURNS JSONB ...
```

**What It Does:**
- Preserves references to `auth.users` and `storage` tables
- Groups RLS policies logically
- Injects auth schema stubs for validation
- Handles storage bucket policies

**Reference:** [Supabase Documentation](https://supabase.com/docs)

---

### Prisma Plugin (`internal/plugins/prisma/`)

**Purpose:** Handle Prisma ORM metadata tables, enum handling, and migration tracking.

**Priority:** 80

**Detects:**
- `_prisma_migrations` table
- Prisma enum tables (`_prisma_enum_*`)
- Shadow database patterns
- Prisma-specific constraints

**Detection Patterns:**
```sql
-- Migration tracking
CREATE TABLE "_prisma_migrations" (
    id VARCHAR(36) PRIMARY KEY,
    checksum VARCHAR(64) NOT NULL,
    ...
);

-- Enum handling
CREATE TABLE "_prisma_enum_UserRole" (...);
```

**Configuration:**
```json
{
  "plugins": {
    "enabled_plugins": ["prisma"]
  },
  "third_party_integrations": {
    "prisma_integration": {
      "enabled": true,
      "preserve_migrations_table": true
    }
  }
}
```

**What It Does:**
- Preserves `_prisma_migrations` table structure exactly
- Handles Prisma enum conflicts
- Maintains shadow database compatibility
- Groups Prisma metadata tables

**Reference:** [Prisma Documentation](https://www.prisma.io/docs)

---

### Drizzle Plugin (`internal/plugins/drizzle/`)

**Purpose:** Handle Drizzle ORM patterns including generated columns and identity columns.

**Priority:** 80

**Detects:**
- `GENERATED ALWAYS AS IDENTITY`
- Generated columns
- Drizzle migration metadata
- Drizzle-specific column defaults

**Detection Patterns:**
```sql
-- Identity columns
id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY

-- Generated columns
full_name TEXT GENERATED ALWAYS AS (first_name || ' ' || last_name) STORED
```

**Configuration:**
```json
{
  "plugins": {
    "enabled_plugins": ["drizzle"]
  },
  "modern_features": {
    "enable_generated_columns": true
  }
}
```

**What It Does:**
- Preserves `GENERATED ALWAYS AS IDENTITY` syntax
- Maintains generated column definitions
- Handles Drizzle column defaults correctly
- Groups generated columns appropriately

**Reference:** [Drizzle Documentation](https://orm.drizzle.team/)

---

### Volatility Plugin (`internal/plugins/volatility/`)

**Purpose:** Detect and fix missing PostgreSQL function volatility declarations (VOLATILE, STABLE, IMMUTABLE).

**Priority:** 70

**Detects:**
- Functions without volatility declarations
- Functions with incorrect volatility
- Performance-critical functions

**What It Does:**
- Analyzes function bodies to infer volatility
- Adds missing volatility declarations
- Warns about performance implications
- Suggests optimal volatility levels

**Detection Patterns:**
```sql
-- Missing volatility (defaults to VOLATILE)
CREATE FUNCTION count_users() RETURNS INTEGER AS $$
BEGIN
    RETURN (SELECT COUNT(*) FROM users);
END;
$$ LANGUAGE plpgsql;
-- Should be STABLE

-- Plugin fixes to:
CREATE FUNCTION count_users() RETURNS INTEGER AS $$
BEGIN
    RETURN (SELECT COUNT(*) FROM users);
END;
$$ LANGUAGE plpgsql STABLE;
```

**Configuration:**
```json
{
  "plugins": {
    "enabled_plugins": ["volatility"],
    "volatility": {
      "auto_fix": true,
      "warn_only": false
    }
  }
}
```

**Volatility Levels:**

| Level     | Description                   | Use Case                  |
| --------- | ----------------------------- | ------------------------- |
| VOLATILE  | Modifies database state       | INSERT, UPDATE, DELETE    |
| STABLE    | No modifications, may vary    | SELECT with WHERE         |
| IMMUTABLE | Always same result for input  | Math functions, constants |

**What It Fixes:**

```sql
-- Before
CREATE FUNCTION get_user_count() RETURNS INTEGER AS $$
    SELECT COUNT(*) FROM users
$$ LANGUAGE sql;

-- After (plugin adds STABLE)
CREATE FUNCTION get_user_count() RETURNS INTEGER AS $$
    SELECT COUNT(*) FROM users
$$ LANGUAGE sql STABLE;
```

**Performance Impact:**

Functions without proper volatility can't be optimized:
- Missing `IMMUTABLE`: Function called for every row instead of once
- Missing `STABLE`: Function re-evaluated in same query
- Incorrect `VOLATILE`: Prevents query optimization

**Configuration Options:**

```json
{
  "plugins": {
    "volatility": {
      "auto_fix": true,           // Automatically add volatility
      "warn_only": false,          // Only warn, don't fix
      "strict_mode": false,        // Require explicit volatility
      "analyze_function_body": true // Infer from function code
    }
  }
}
```

**Reference:** [PostgreSQL Volatility Documentation](https://www.postgresql.org/docs/current/xfunc-volatility.html)

---

### Using Multiple Plugins

Plugins work together based on priority:

```json
{
  "plugins": {
    "auto_detect": true,
    "enabled_plugins": ["auth", "clerk", "supabase", "volatility"]
  }
}
```

**Execution Order (by priority):**
1. Auth Plugin (95)
2. Clerk Plugin (90)
3. Supabase Plugin (90)
4. Prisma Plugin (80)
5. Drizzle Plugin (80)
6. Volatility Plugin (70)

**Conflict Resolution:**

If plugins conflict, higher priority wins. Use `disabled_plugins` to exclude:

```json
{
  "plugins": {
    "auto_detect": true,
    "disabled_plugins": ["auth"]  // Disable if Clerk is primary
  }
}
```

---

## Related Documentation

- [Architecture Guide](./architecture.md) - System architecture overview
- [Configuration Guide](./configuration.md) - Configuration reference
- [Pattern Detection Guide](./patterns.md) - Pattern detection system
- [Plugins README](../internal/plugins/README.md) - Internal plugin documentation
- [Troubleshooting](./troubleshooting.md) - Common issues and solutions

## Summary

Creating a pgsquash plugin:

1. **Create plugin package** - `internal/plugins/yourservice/`
2. **Implement interface** - Embed `BasePlugin`, override needed methods
3. **Register plugin** - Add to `cmd/pgsquash/main.go`
4. **Test thoroughly** - Unit tests + integration tests
5. **Document configuration** - Add config examples
6. **Follow best practices** - Conservative consolidation, clear errors

**Minimum required methods:**

- `Name()` - Unique identifier
- `Priority()` - Conflict resolution
- `Detect()` - Pattern matching

**Common optional methods:**

- `ShouldPreserve()` - Mark critical statements
- `InjectCompatibilityLayer()` - Validation support
- `TransformSQL()` - Fix syntax issues

The plugin system enables pgsquash to intelligently handle any authentication provider, ORM, or Platform without modifying the core engine.
