# Rule Registry System

The Rule Registry System provides dynamic, plugin-extensible consolidation rule management in pgsquash. It allows plugins to register custom consolidation rules with priorities, metadata, and runtime enable/disable controls.

## Overview

The Rule Registry System replaces the static rule list with a flexible, priority-based registry that supports:

- **Dynamic Registration**: Plugins can register rules at runtime
- **Priority-Based Execution**: Rules execute in priority order (highest first)
- **Metadata**: Rules have descriptive metadata (name, description, category, tags)
- **Enable/Disable**: Rules can be enabled or disabled at runtime
- **Conflict Resolution**: Automatic conflict resolution based on policy
- **Provider Tracking**: Rules are tracked by provider (core, supabase, clerk, etc.)

## Architecture

```
┌────────────────────────────────────────────────────────────┐
│                     RuleRegistry                            │
│  (Global Singleton, Thread-Safe)                           │
├────────────────────────────────────────────────────────────┤
│ • rules: map[string]*RegisteredRule                        │
│ • rulesByCategory: map[Category][]*RegisteredRule          │
│ • rulesByProvider: map[string][]*RegisteredRule            │
│ • conflictPolicy: ConflictPolicy                           │
├────────────────────────────────────────────────────────────┤
│ Register(rule, metadata)                                   │
│ Unregister(name)                                           │
│ EnableRule(name) / DisableRule(name)                       │
│ GetEnabledRules() → sorted by priority                     │
│ GetRulesByCategory(category)                               │
│ GetRulesByProvider(provider)                               │
│ GetApplicableRules(lifecycle) → filtered + sorted          │
└────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌────────────────────────────────────────────────────────────┐
│             ConsolidationRuleEngine                         │
│  (Uses Registry or Legacy Mode)                            │
├────────────────────────────────────────────────────────────┤
│ • registry: *RuleRegistry                                  │
│ • useRegistry: bool (true for new, false for legacy)       │
├────────────────────────────────────────────────────────────┤
│ ApplyRules(lifecycle, engine)                              │
│   → Gets applicable rules from registry (by priority)      │
│   → Applies first matching rule                            │
└────────────────────────────────────────────────────────────┘
```

## Core Concepts

### RegisteredRule

A `RegisteredRule` wraps a `ConsolidationRule` with metadata:

```go
type RegisteredRule struct {
    Rule     ConsolidationRule
    Metadata RuleMetadata
}
```

### RuleMetadata

Complete metadata describing a rule:

```go
type RuleMetadata struct {
    Name        string       // Unique identifier (e.g., "create_alter_consolidation")
    Description string       // Human-readable description
    Category    RuleCategory // Category for organization
    Priority    int          // Execution priority (higher = earlier)
    Provider    string       // Provider name (e.g., "core", "clerk")
    Tags        []string     // Tags for filtering (e.g., "safe", "aggressive")
    Enabled     bool         // Whether rule is enabled
    Version     string       // Rule version (e.g., "1.0.0")
}
```

### RuleCategory

Pre-defined categories for rule organization:

- `CategoryTableOps`: Table CREATE/ALTER consolidation
- `CategoryFunctionOps`: Function deduplication
- `CategoryDeadCode`: Dead code removal
- `CategorySecurity`: RLS, policies, auth
- `CategoryOptimization`: Performance optimizations
- `CategoryExtension`: Extension-specific rules
- `CategoryPluginAuth`: Plugin authentication patterns
- `CategoryPluginORM`: ORM-specific patterns

### Priority System

Rules execute in **descending priority order** (highest priority first):

| Priority Range | Use Case                  | Example Rules                                                            |
| -------------- | ------------------------- | ------------------------------------------------------------------------ |
| 100-90         | Critical table operations | `multiple_create_consolidation` (100), `create_alter_consolidation` (90) |
| 89-70          | Standard consolidation    | `drop_create_cycle` (85), `function_deduplication` (80)                  |
| 69-50          | Type and schema handling  | `column_evolution` (65), `rls_consolidation` (60)                        |
| 49-30          | Optimization and recovery | `transaction_boundary` (40), `error_recovery` (30)                       |

**Best Practice**: Leave gaps between priorities (e.g., 100, 90, 80) to allow future rules to slot in between.

## Usage

### 1. Registering Core Rules

Core rules are registered automatically when creating a new engine:

```go
engine := consolidation.NewConsolidationRuleEngine()
// Core rules are automatically registered from registry
```

Manual core rule registration:

```go
registry := consolidation.GetRegistry()
err := consolidation.RegisterCoreRules(registry)
if err != nil {
    log.Fatalf("Failed to register core rules: %v", err)
}
```

### 2. Registering Custom Rules

Plugins can register custom rules:

```go
// Get global registry
registry := consolidation.GetRegistry()

// Register custom rule
customRule := &MyCustomRule{}
metadata := consolidation.RuleMetadata{
    Name:        "my_custom_rule",
    Description: "Custom consolidation logic for my plugin",
    Category:    consolidation.CategoryPluginAuth,
    Priority:    75, // Between function_deduplication (80) and enum_deduplication (75)
    Provider:    "my_plugin",
    Tags:        []string{"safe", "auth"},
    Enabled:     true,
    Version:     "1.0.0",
}

err := registry.Register(customRule, metadata)
if err != nil {
    log.Printf("Failed to register rule: %v", err)
}
```

### 3. Enabling/Disabling Rules

Enable or disable rules at runtime:

```go
registry := consolidation.GetRegistry()

// Disable aggressive rules
err := registry.DisableRule("dead_code_removal")

// Enable specific rule
err = registry.EnableRule("dead_code_removal")
```

### 4. Querying Rules

Query rules by various criteria:

```go
registry := consolidation.GetRegistry()

// Get all enabled rules (sorted by priority)
enabledRules := registry.GetEnabledRules()

// Get rules by category
tableRules := registry.GetRulesByCategory(consolidation.CategoryTableOps)

// Get rules by provider
clerkRules := registry.GetRulesByProvider("clerk")

// Get rules by tag
safeRules := registry.GetRulesByTag("safe")

// Get applicable rules for a specific lifecycle
lifecycle := tracker.GetObjectLifecycle("public.users")
applicableRules := registry.GetApplicableRules(lifecycle)
```

### 5. Registry Statistics

Get statistics about registered rules:

```go
registry := consolidation.GetRegistry()
stats := registry.GetStats()

fmt.Printf("Total rules: %d\n", stats.TotalRules)
fmt.Printf("Enabled: %d, Disabled: %d\n", stats.EnabledRules, stats.DisabledRules)
fmt.Printf("Rules by category: %v\n", stats.RulesByCategory)
fmt.Printf("Rules by provider: %v\n", stats.RulesByProvider)
```

## Conflict Resolution

When multiple rules have the same name, the registry uses a conflict policy:

### ConflictPolicyHighestPriority (Default)

Replace existing rule only if new rule has higher priority:

```go
registry := consolidation.NewRuleRegistry(consolidation.ConflictPolicyHighestPriority)

// Register initial rule with priority 50
registry.Register(ruleV1, RuleMetadata{Name: "my_rule", Priority: 50})

// Register replacement with priority 75 (succeeds)
registry.Register(ruleV2, RuleMetadata{Name: "my_rule", Priority: 75})

// Register replacement with priority 40 (fails - lower priority)
registry.Register(ruleV3, RuleMetadata{Name: "my_rule", Priority: 40}) // Error
```

### ConflictPolicyFirstRegistered

First registered rule wins, subsequent registrations fail:

```go
registry := consolidation.NewRuleRegistry(consolidation.ConflictPolicyFirstRegistered)

registry.Register(ruleV1, RuleMetadata{Name: "my_rule", Priority: 50})
registry.Register(ruleV2, RuleMetadata{Name: "my_rule", Priority: 75}) // Error: already registered
```

### ConflictPolicyError

Always fail on conflicts (strictest):

```go
registry := consolidation.NewRuleRegistry(consolidation.ConflictPolicyError)

registry.Register(ruleV1, RuleMetadata{Name: "my_rule", Priority: 50})
registry.Register(ruleV2, RuleMetadata{Name: "my_rule", Priority: 75}) // Error: conflict
```

## Plugin Integration

Plugins register rules during initialization:

```go
// Example: Clerk plugin registering auth-specific rules
type ClerkPlugin struct {
    registry *consolidation.RuleRegistry
}

func (p *ClerkPlugin) Initialize() error {
    // Get global registry
    p.registry = consolidation.GetRegistry()

    // Register JWT validation rule
    jwtRule := &ClerkJWTValidationRule{}
    jwtMetadata := consolidation.RuleMetadata{
        Name:        "clerk_jwt_validation",
        Description: "Validates Clerk JWT v2 patterns",
        Category:    consolidation.CategoryPluginAuth,
        Priority:    95, // High priority for auth rules
        Provider:    "clerk",
        Tags:        []string{"auth", "jwt", "clerk"},
        Enabled:     true,
        Version:     "2.0.0",
    }

    if err := p.registry.Register(jwtRule, jwtMetadata); err != nil {
        return fmt.Errorf("failed to register JWT rule: %w", err)
    }

    // Register user table consolidation rule
    userTableRule := &ClerkUserTableRule{}
    userMetadata := consolidation.RuleMetadata{
        Name:        "clerk_user_table",
        Description: "Handles Clerk user table patterns",
        Category:    consolidation.CategoryTableOps,
        Priority:    85,
        Provider:    "clerk",
        Tags:        []string{"auth", "table", "clerk"},
        Enabled:     true,
        Version:     "1.0.0",
    }

    return p.registry.Register(userTableRule, userMetadata)
}
```

## Core Rules Reference

All core rules with their priorities:

| Rule Name                       | Priority | Category             | Tags                   | Description                                              |
| ------------------------------- | -------- | -------------------- | ---------------------- | -------------------------------------------------------- |
| `multiple_create_consolidation` | 100      | table\_operations    | safe, standard         | Consolidates multiple CREATE statements                  |
| `create_alter_consolidation`    | 90       | table\_operations    | safe, standard         | Combines CREATE and ALTER into single CREATE             |
| `drop_create_cycle`             | 85       | table\_operations    | safe, optimization     | Removes redundant DROP/CREATE cycles                     |
| `function_deduplication`        | 80       | function\_operations | safe, deduplication    | Removes duplicate function definitions                   |
| `publication_deduplication`     | 80       | optimization         | safe, replication      | Removes duplicate publication additions                  |
| `enum_deduplication`            | 75       | table\_operations    | safe, types            | Removes duplicate ENUM definitions                       |
| `dead_code_removal`             | 70       | dead\_code           | aggressive, cleanup    | Removes unreferenced functions (**disabled by default**) |
| `column_evolution`              | 65       | table\_operations    | standard, schema       | Consolidates column modifications                        |
| `rls_consolidation`             | 60       | security             | safe, security, rls    | Consolidates RLS policy operations                       |
| `do_block_enum_type`            | 55       | table\_operations    | safe, types, do\_block | Handles DO block ENUM operations                         |
| `external_dependency_filter`    | 50       | optimization         | standard, dependencies | Filters external dependency operations                   |
| `conditional_schema`            | 45       | table\_operations    | safe, schema           | Handles conditional schema operations                    |
| `transaction_boundary`          | 40       | optimization         | safe, transactions     | Manages transaction preservation                         |
| `error_recovery`                | 30       | optimization         | safe, recovery         | Handles error recovery and retry                         |

## Advanced Usage

### Rule Priorities by Safety Level

Adjust rule priorities based on safety level:

```go
registry := consolidation.GetRegistry()

switch safetyLevel {
case "paranoid":
    // Disable aggressive rules
    registry.DisableRule("dead_code_removal")

case "aggressive":
    // Enable all rules and increase cleanup priority
    registry.EnableRule("dead_code_removal")

    // Re-register with higher priority if needed
    deadCodeRule := &consolidation.DeadCodeRemovalRule{}
    metadata := consolidation.RuleMetadata{
        Name:     "dead_code_removal",
        Priority: 95, // Bump to high priority in aggressive mode
        ...
    }
    registry.Register(deadCodeRule, metadata)
}
```

### Custom Rule Selection

Select rules by custom criteria:

```go
registry := consolidation.GetRegistry()

// Get all safe auth rules
var authRules []*consolidation.RegisteredRule
for _, rule := range registry.GetEnabledRules() {
    if rule.Metadata.Category == consolidation.CategoryPluginAuth &&
       hasTag(rule.Metadata.Tags, "safe") {
        authRules = append(authRules, rule)
    }
}
```

### Legacy Mode

Use legacy static rules (backward compatibility):

```go
// Create engine with legacy rules (no registry)
engine := consolidation.NewLegacyConsolidationRuleEngine()

// Or switch existing engine to legacy mode
engine.UseLegacyRules()
```

## Thread Safety

The `RuleRegistry` is **thread-safe**:

- All operations use `sync.RWMutex`
- Reads use `RLock()` for concurrent access
- Writes use `Lock()` for exclusive access
- Safe to use from multiple goroutines

## Best Practices

### 1. Priority Assignment

- **100-90**: Critical table operations (CREATE, ALTER)
- **89-70**: Standard consolidation (functions, cycles)
- **69-50**: Type and schema handling
- **49-30**: Optimization and recovery
- **29-10**: Cosmetic and minor rules

Leave gaps (e.g., 10 priority units) to allow future rules to fit between existing ones.

### 2. Rule Naming

Use snake\_case with descriptive names:

- ✅ `clerk_jwt_validation`
- ✅ `supabase_rls_consolidation`
- ❌ `clerkRule1`
- ❌ `MyRule`

### 3. Provider Naming

Use consistent provider names:

- Core: `"core"`
- Plugins: Plugin name in lowercase (e.g., `"clerk"`, `"supabase"`, `"prisma"`)

### 4. Tags

Use consistent, lowercase tags:

- Safety: `"safe"`, `"aggressive"`, `"standard"`
- Type: `"auth"`, `"orm"`, `"replication"`, `"security"`
- Scope: `"table"`, `"function"`, `"type"`, `"schema"`

### 5. Enable/Disable Defaults

- **Safe rules**: Enabled by default
- **Aggressive rules**: Disabled by default (e.g., `dead_code_removal`)
- **Plugin rules**: Enabled by default (plugins control enablement)

### 6. Version Management

Use semantic versioning:

- `"1.0.0"` - Initial rule
- `"1.1.0"` - Minor enhancements (backward compatible)
- `"2.0.0"` - Breaking changes

## Testing

### Testing Custom Rules

Test rule registration and execution:

```go
func TestCustomRuleRegistration(t *testing.T) {
    // Create test registry
    registry := consolidation.NewRuleRegistry(consolidation.ConflictPolicyError)

    // Register custom rule
    customRule := &MyCustomRule{}
    metadata := consolidation.RuleMetadata{
        Name:     "test_rule",
        Category: consolidation.CategoryTableOps,
        Priority: 50,
        Provider: "test",
        Enabled:  true,
    }

    err := registry.Register(customRule, metadata)
    assert.NoError(t, err)

    // Verify registration
    registered, err := registry.GetRule("test_rule")
    assert.NoError(t, err)
    assert.Equal(t, "test_rule", registered.Metadata.Name)

    // Test priority ordering
    rules := registry.GetEnabledRules()
    assert.True(t, len(rules) > 0)
}
```

### Testing Priority Conflicts

Test conflict resolution:

```go
func TestConflictResolution(t *testing.T) {
    registry := consolidation.NewRuleRegistry(consolidation.ConflictPolicyHighestPriority)

    // Register lower priority rule
    ruleV1 := &MyRule{}
    metadata1 := consolidation.RuleMetadata{
        Name:     "my_rule",
        Priority: 50,
    }
    err := registry.Register(ruleV1, metadata1)
    assert.NoError(t, err)

    // Register higher priority rule (should replace)
    ruleV2 := &MyRule{}
    metadata2 := consolidation.RuleMetadata{
        Name:     "my_rule",
        Priority: 75,
    }
    err = registry.Register(ruleV2, metadata2)
    assert.NoError(t, err)

    // Verify replacement
    registered, _ := registry.GetRule("my_rule")
    assert.Equal(t, 75, registered.Metadata.Priority)
}
```

## Migration from Legacy Rules

### Before (Static Rules)

```go
engine := &ConsolidationRuleEngine{
    rules: []ConsolidationRule{},
}
engine.AddRule(&CreateAlterConsolidationRule{})
engine.AddRule(&FunctionDeduplicationRule{})
```

### After (Registry)

```go
// Get global registry
registry := consolidation.GetRegistry()

// Register with metadata
registry.Register(&CreateAlterConsolidationRule{}, RuleMetadata{
    Name:     "create_alter_consolidation",
    Priority: 90,
    Category: CategoryTableOps,
    Enabled:  true,
})

// Engine uses registry automatically
engine := consolidation.NewConsolidationRuleEngine()
```

## Troubleshooting

### Rule Not Executing

1. **Check if rule is enabled**:
   ```go
   registered, _ := registry.GetRule("my_rule")
   fmt.Printf("Enabled: %v\n", registered.Metadata.Enabled)
   ```

2. **Check rule priority**:
   ```go
   rules := registry.GetEnabledRules()
   for _, r := range rules {
       fmt.Printf("%s: priority %d\n", r.Metadata.Name, r.Metadata.Priority)
   }
   ```

3. **Check if rule applies**:
   ```go
   lifecycle := tracker.GetObjectLifecycle("public.users")
   applicable := registry.GetApplicableRules(lifecycle)
   fmt.Printf("Applicable rules: %d\n", len(applicable))
   ```

### Conflict Errors

```
ERROR: rule 'my_rule' already registered with higher or equal priority
```

**Solution**: Use higher priority or unregister existing rule first:

```go
registry.Unregister("my_rule")
registry.Register(newRule, metadata)
```

### Performance Issues

If rule lookup is slow:

1. **Use category/provider filtering**:
   ```go
   // Instead of GetEnabledRules()
   rules := registry.GetRulesByCategory(CategoryTableOps)
   ```

2. **Cache applicable rules**:
   ```go
   // Cache rules for lifecycle type
   cachedRules := registry.GetRulesByCategory(CategoryTableOps)
   ```

## Further Reading

- [Configuration Reference](./configuration.md) - Rule-related configuration options
- [Plugin Development](../internal/plugins/README.md) - Creating plugins with custom rules
- [Safety Levels](./safety-levels.md) - How rules behave at different safety levels
- [Architecture](./architecture.md) - Overall system architecture
