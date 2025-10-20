# Advanced Features Guide

Enterprise-scale migration management—when the basic CLI isn't enough.

## When You Need These Features

**Are you hitting these limits?**

- Migration folder is 1GB+ and basic analysis takes 10+ minutes
- Multi-tenant SaaS with 100+ schemas that need independent tracking
- Team of 10+ developers creating merge conflicts constantly
- Custom SQL generation for dynamic tenant provisioning
- Need to analyze migrations without database access

**If yes, these advanced features solve those problems.**

## Overview

These subsystems handle what the basic CLI can't:

- **Streaming Processor** - Process 10GB+ migration folders without running out of memory
- **Metadata Manager** - Cache schema information to skip expensive database queries
- **SQL Builder** - Programmatically generate migrations (useful for multi-tenant setups)
- **Object Tracker** - Track object lifecycles across thousands of files
- **Performance Manager** - Parallel processing with memory management

**Real-world usage:**

- **Agency managing 50+ client projects** - Metadata caching skips repetitive queries
- **Multi-tenant SaaS with 200 schemas** - Streaming processor handles massive folders
- **Platform generating tenant databases** - SQL builder creates consistent migrations programmatically

## SQL Builder System

**When to use:** Programmatically generating migrations (multi-tenant SaaS, dynamic schema provisioning).

### Use Case: Multi-Tenant Schema Generation

**Problem:** You onboard 20 new customers per week, each needs identical schema with custom naming (`tenant_123_users`, `tenant_456_users`). Manually creating migrations doesn't scale.

**Solution:** Generate migrations programmatically with consistent formatting.

```go
import "github.com/CAPYSQUASH/pgsquash-engine/internal/builder"

// Generate tenant-specific schema
func GenerateTenantSchema(tenantID string) string {
    b := builder.NewSQLBuilder(nil)

    tableName := fmt.Sprintf("tenant_%s_users", tenantID)

    b.P("CREATE", "TABLE", tableName).
        NL().
        P("(").
        NL().Indent().
        P("id", "SERIAL", "PRIMARY", "KEY,").
        NL().
        P("name", "VARCHAR(255)", "NOT", "NULL,").
        NL().
        P("created_at", "TIMESTAMP", "DEFAULT", "NOW()").
        NL().Dedent().
        P(")").
        String() // Returns the built SQL string

    return sql
}

// Result for tenant 123:
// CREATE TABLE tenant_123_users
// (
//   id SERIAL PRIMARY KEY,
//   name VARCHAR(255) NOT NULL,
//   created_at TIMESTAMP DEFAULT NOW()
// )
```

**Why use the builder instead of string templates?**

- Automatic indentation and formatting
- Handles identifier quoting correctly
- Validates syntax as you build
- Easier to maintain than string concatenation

### Build Options

```go
options := &builder.BuildOptions{
    FormatStyle:      builder.FormatPretty,  // Pretty, Compact, or Dense
    IndentSize:       2,                     // Spaces per indent level
    MaxLineLength:    120,                   // Wrap at this column
    UseDoubleQuotes:  true,                  // PostgreSQL standard quotes
    NormalizeNames:   true,                  // Normalize identifiers
    PreserveComments: true,                  // Keep SQL comments
    PostgreSQLMode:   true,                  // PostgreSQL-specific formatting
}

b := builder.NewSQLBuilder(options)
```

### Format Styles

**FormatPretty** (default):

```sql
CREATE TABLE users (
  id SERIAL PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  email VARCHAR(255) UNIQUE
);
```

**FormatCompact**:

```sql
CREATE TABLE users (id SERIAL PRIMARY KEY, name VARCHAR(255) NOT NULL, email VARCHAR(255) UNIQUE);
```

**FormatDense**:

```sql
CREATE TABLE users (id SERIAL PRIMARY KEY,name VARCHAR(255) NOT NULL,email VARCHAR(255) UNIQUE);
```

### Fluent Interface Methods

| Method           | Purpose                         | Example                             |
| ---------------- | ------------------------------- | ----------------------------------- |
| `P(...string)`   | Add phrases separated by spaces | `b.P("CREATE", "TABLE")`            |
| `S()`            | Add single space                | `b.P("NOT").S().P("NULL")`          |
| `NL()`           | Add newline with indentation    | `b.P("(").NL()`                     |
| `Indent()`       | Increase indent level           | `b.Indent().P("id").Dedent()`       |
| `Dedent()`       | Decrease indent level           | `b.Dedent().P(")")`                 |
| `Statement(sql)` | Add complete statement          | `b.Statement("CREATE TABLE users")` |
| `String()`       | Return final SQL string         | `sql := b.String()`                 |

### Building Complex Statements

```go
// CREATE TABLE with multiple columns, constraints, indexes
func BuildUsersTable() string {
    b := builder.NewSQLBuilder(nil)

    b.P("CREATE", "TABLE", "users").NL().
    P("(").NL().Indent().

    // Columns
    P("id", "INTEGER", "GENERATED", "BY", "DEFAULT", "AS", "IDENTITY", "PRIMARY", "KEY,").NL().
    P("email", "VARCHAR(255)", "NOT", "NULL", "UNIQUE,").NL().
    P("name", "VARCHAR(255),").NL().
    P("created_at", "TIMESTAMP", "DEFAULT", "NOW(),").NL().
    P("updated_at", "TIMESTAMP", "DEFAULT", "NOW()").NL().

    Dedent().P(")").P(";")

    return b.String()
}
```

### Building Functions

```go
func BuildAuthFunction() string {
    b := builder.NewSQLBuilder(nil)

    b.P("CREATE", "OR", "REPLACE", "FUNCTION", "auth.get_user_id()").NL().
    P("RETURNS", "TEXT").NL().
    P("LANGUAGE", "sql", "STABLE", "SECURITY", "DEFINER").NL().
    P("AS", "$$").NL().Indent().
    P("SELECT", "current_setting('request.jwt.claim.sub', true)").NL().
    Dedent().P("$$").P(";")

    return b.String()
}
```

### Building Policies

```go
func BuildRLSPolicy() string {
    b := builder.NewSQLBuilder(nil)

    b.P("CREATE", "POLICY", "user_isolation").NL().
    P("ON", "users").NL().
    P("FOR", "SELECT").NL().
    P("USING", "(").Indent().NL().
    P("auth.get_user_id()", "=", "user_id").NL().
    Dedent().P(")").P(";")

    return b.String()
}
```

### Error Handling

```go
b := builder.NewSQLBuilder(nil)
b.P("CREATE", "TABLE")
// ... build SQL ...

if errs := b.GetErrors(); len(errs) > 0 {
    for _, err := range errs {
        log.Printf("Builder error: %v", err)
    }
}
```

### Integration with Deparsing

The SQL builder is used by the squasher to generate consolidated SQL from parsed ASTs:

```go
// Internal squasher usage
func (s *Squasher) GenerateSQL(lifecycle *ObjectLifecycle) string {
    builder := builder.NewSQLBuilder(s.config.BuildOptions)

    // Add CREATE statement
    builder.Statement(lifecycle.History[0].Statement.SQL)

    // Add modifications
    for _, event := range lifecycle.History[1:] {
        if event.Operation == types.OpAlter {
            builder.NL().NL()
            builder.Statement(event.Statement.SQL)
        }
    }

    return builder.Build()
}
```

## Metadata Manager

The metadata manager provides database schema introspection with caching and lazy-loading.

### Purpose

- **Schema introspection** - Extract complete database metadata
- **Caching** - Reduce database queries with TTL-based cache
- **Search path resolution** - Resolve unqualified object names
- **Extension detection** - Discover installed PostgreSQL extensions

### Basic Usage

```go
import "github.com/CAPYSQUASH/pgsquash-engine/internal/metadata"

// Create manager with database connection
manager := metadata.NewMetadataManager(db, 5*time.Minute) // 5min cache TTL

// Get database metadata (cached)
dbMeta, err := manager.GetDatabaseMetadata(ctx, "mydb")
if err != nil {
    log.Fatal(err)
}

// Access schemas
for schemaName, schema := range dbMeta.Schemas {
    fmt.Printf("Schema: %s\n", schemaName)

    // Access tables
    for tableName, table := range schema.Tables {
        fmt.Printf("  Table: %s.%s\n", schemaName, tableName)

        // Access columns
        for _, col := range table.Columns {
            fmt.Printf("    Column: %s %s\n", col.Name, col.DataType)
        }
    }
}
```

### Metadata Structures

**DatabaseMetadata**:

```go
type DatabaseMetadata struct {
    Database   string                        // Database name
    Schemas    map[string]*SchemaMetadata    // All schemas
    SearchPath []string                      // Current search path
    Extensions map[string]*ExtensionMetadata // Installed extensions
    Version    *PostgreSQLVersion            // PostgreSQL version
}
```

**SchemaMetadata**:

```go
type SchemaMetadata struct {
    Name              string
    Tables            map[string]*TableMetadata
    Views             map[string]*ViewMetadata
    Functions         map[string][]*FunctionMetadata // Overloaded functions
    Sequences         map[string]*SequenceMetadata
    Types             map[string]*TypeMetadata
    Indexes           map[string]*IndexMetadata
}
```

**TableMetadata**:

```go
type TableMetadata struct {
    Name        string
    Schema      string
    Columns     []*ColumnMetadata
    Constraints []*ConstraintMetadata
    Indexes     []*IndexMetadata
    Triggers    []*TriggerMetadata
    Policies    []*PolicyMetadata
    RowSecurity bool // RLS enabled?
}
```

### Getting Specific Objects

```go
// Get table metadata
table, err := manager.GetTableMetadata(ctx, "public", "users")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Table: %s.%s\n", table.Schema, table.Name)
for _, col := range table.Columns {
    nullable := "NULL"
    if !col.IsNullable {
        nullable = "NOT NULL"
    }
    fmt.Printf("  %s %s %s\n", col.Name, col.DataType, nullable)
}

// Get function metadata (handles overloading)
functions, err := manager.GetFunctionMetadata(ctx, "public", "calculate_total")
if err != nil {
    log.Fatal(err)
}

for _, fn := range functions {
    fmt.Printf("Function: %s(%s) RETURNS %s\n",
        fn.Name, strings.Join(fn.Parameters, ", "), fn.ReturnType)
}
```

### Search Path Resolution

```go
// Resolve unqualified name using search path
objectID, err := manager.ResolveObjectName(ctx, "users", types.TypeTable)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Resolved: %s.%s\n", objectID.Schema, objectID.Name)
// Example output: Resolved: public.users
```

### Extension Detection

```go
// Get installed extensions
extensions, err := manager.GetExtensions(ctx)
if err != nil {
    log.Fatal(err)
}

for extName, ext := range extensions {
    fmt.Printf("Extension: %s (version %s)\n", extName, ext.Version)
}

// Check if specific extension exists
hasUUID, _ := manager.HasExtension(ctx, "uuid-ossp")
if hasUUID {
    fmt.Println("uuid-ossp extension available")
}
```

### Caching Behavior

```go
// First call: Queries database
meta1, _ := manager.GetDatabaseMetadata(ctx, "mydb")
fmt.Printf("Cache stats: Hits=%d, Misses=%d\n",
    manager.GetCacheHits(), manager.GetCacheMisses())
// Output: Cache stats: Hits=0, Misses=1

// Second call: Returns cached data (within TTL)
meta2, _ := manager.GetDatabaseMetadata(ctx, "mydb")
fmt.Printf("Cache stats: Hits=%d, Misses=%d\n",
    manager.GetCacheHits(), manager.GetCacheMisses())
// Output: Cache stats: Hits=1, Misses=1

// Invalidate cache manually
manager.InvalidateCache("mydb")

// Force refresh
meta3, _ := manager.GetDatabaseMetadata(ctx, "mydb")
// Fresh data from database
```

### Integration with Validation

The metadata manager is used during validation to check for:

- **Missing dependencies** - Objects that don't exist
- **Type mismatches** - Incompatible column types
- **Constraint violations** - FK references to non-existent columns

```go
// Validate that all referenced tables exist
func ValidateForeignKeys(manager *metadata.MetadataManager, fkDeps []FKDependency) error {
    for _, fk := range fkDeps {
        table, err := manager.GetTableMetadata(ctx, fk.Schema, fk.Table)
        if err != nil {
            return fmt.Errorf("referenced table %s.%s not found", fk.Schema, fk.Table)
        }

        // Check referenced column exists
        found := false
        for _, col := range table.Columns {
            if col.Name == fk.Column {
                found = true
                break
            }
        }
        if !found {
            return fmt.Errorf("referenced column %s.%s.%s not found",
                fk.Schema, fk.Table, fk.Column)
        }
    }
    return nil
}
```

## Object Tracking System

The object tracker maintains the complete lifecycle of database objects across migrations.

### Purpose

- **Lifecycle tracking** - Track CREATE, ALTER, DROP operations
- **Dependency analysis** - Build dependency graph between objects
- **Change detection** - Identify what changed between migration versions
- **Consolidation support** - Determine which operations can be merged

### Basic Usage

```go
import "github.com/CAPYSQUASH/pgsquash-engine/internal/tracking"

// Create tracker
tracker := tracking.NewTracker() // Use NewTracker() not NewUnifiedTracker()

// Process migrations
for i, migration := range migrations {
    tracker.ProcessMigration(migration, i) // No ctx parameter, no error return
}

// Get tracked objects by category
lifecycles := tracker.GetObjectsByCategory()
for category, objects := range lifecycles {
    fmt.Printf("Category: %s\n", category)
    for key, lifecycle := range objects {
        fmt.Printf("  Object: %s\n", key)
        fmt.Printf("    Type: %s\n", lifecycle.Type)
        fmt.Printf("    Events: %d\n", len(lifecycle.History))
        fmt.Printf("    Dropped: %v\n", lifecycle.WasDropped)
    }
}
```

### Object Lifecycle

```go
type ObjectLifecycle struct {
    Key          string           // "schema.name.type"
    Name         string           // Object name
    Schema       string           // Schema name
    Type         types.ObjectType // TABLE, FUNCTION, INDEX, etc.
    History      []LifecycleEvent // All operations on this object
    Dependencies []ObjectDependency
    WasDropped   bool             // True if object was dropped
    IsRedundant  bool             // True if can be consolidated
    RiskLevel    RiskLevel        // Low, Medium, High, Critical
}
```

### Lifecycle Events

```go
type LifecycleEvent struct {
    ID           string          // Unique event ID
    Migration    string          // Migration filename
    Sequence     int             // Migration sequence number
    Operation    types.Operation // CREATE, ALTER, DROP, etc.
    Statement    types.Statement // Original SQL
    Timestamp    time.Time       // When event occurred
    Dependencies []string        // Objects this depends on
    RiskLevel    RiskLevel       // Risk assessment
}
```

### Tracking Object Changes

```go
// Create table
migration1 := &types.Migration{
    Filename: "001_create_users.sql",
    Statements: []types.Statement{
        {
            ObjectType: types.TypeTable,
            ObjectName: "users",
            Operation:  types.OpCreate,
            SQL:        "CREATE TABLE users (id SERIAL PRIMARY KEY);",
        },
    },
}
tracker.ProcessMigration(migration1, 0) // No ctx parameter

// Alter table
migration2 := &types.Migration{
    Filename: "002_add_email.sql",
    Statements: []types.Statement{
        {
            ObjectType: types.TypeTable,
            ObjectName: "users",
            Operation:  types.OpAlter,
            SQL:        "ALTER TABLE users ADD COLUMN email VARCHAR(255);",
        },
    },
}
tracker.ProcessMigration(migration2, 1) // No ctx parameter

// Get lifecycle (use correct method)
allObjects := tracker.GetObjectsByCategory()
lifecycle := allObjects[types.CategoryFoundation]["public.users.table"]
fmt.Printf("History:\n")
for _, event := range lifecycle.History {
    fmt.Printf("  [%d] %s: %s\n", event.Sequence, event.Operation, event.Migration)
}
// Output:
// History:
//   [0] CREATE: 001_create_users.sql
//   [1] ALTER: 002_add_email.sql
```

### Dependency Tracking

```go
// Tables with foreign key dependencies
dependencies := tracker.GetDependencies("public.posts.table")
for _, dep := range dependencies {
    fmt.Printf("%s depends on %s\n", dep.ObjectID, dep.DependsOn)
}
// Output: public.posts.table depends on public.users.table

// Get dependency graph
graph := tracker.GetDependencyGraph()
sorted, err := graph.TopologicalSort()
if err != nil {
    log.Fatal("Circular dependency detected:", err)
}

fmt.Println("Creation order:")
for _, objectID := range sorted {
    fmt.Printf("  %s\n", objectID)
}
```

### Redundancy Detection

```go
// Find redundant operations that can be consolidated
redundancies := tracker.GetRedundancies()
for _, redundancy := range redundancies {
    fmt.Printf("Redundant: %s (%s)\n", redundancy.ObjectKey, redundancy.Reason)
    fmt.Printf("  Can consolidate: %v\n", redundancy.CanConsolidate)
    fmt.Printf("  Savings: %d operations\n", redundancy.OperationsSaved)
}

// Example output:
// Redundant: public.users.table (multiple ALTER operations)
//   Can consolidate: true
//   Savings: 5 operations
```

### Risk Assessment

```go
type RiskLevel int

const (
    RiskLow      RiskLevel = iota // Safe to consolidate
    RiskMedium                     // Consolidate with caution
    RiskHigh                       // Review before consolidating
    RiskCritical                   // Do not consolidate
)

// Get objects by risk level
criticalObjects := tracker.GetObjectsByRisk(tracking.RiskCritical)
for _, obj := range criticalObjects {
    fmt.Printf("Critical: %s - %s\n", obj.Key, obj.RiskReason)
}
```

### Statistics

```go
stats := tracker.GetStatistics()
fmt.Printf("Statistics:\n")
fmt.Printf("  Total Objects: %d\n", stats.TotalObjects)
fmt.Printf("  Total Events: %d\n", stats.TotalEvents)
fmt.Printf("  Dropped Objects: %d\n", stats.DroppedObjects)
fmt.Printf("  Redundant Operations: %d\n", stats.RedundantOperations)
fmt.Printf("  Dependencies: %d\n", stats.TotalDependencies)
```

## Streaming Processor

**When you need it:** Migration folder is 1GB+, or you're running out of memory during analysis.

### The Problem at Scale

**Scenario:** Multi-tenant SaaS with 3 years of migrations

- 847 migration files
- 2.3 GB total size
- Each tenant schema has 50+ tables

**Without streaming:**

```bash
pgsquash analyze migrations/*.sql

# Result:
# Loading migrations... 100%
# Parsing...
# killed: out of memory
```

Basic mode loads everything into memory at once—doesn't work at this scale.

**With streaming:**

```bash
pgsquash analyze migrations/*.sql --stream --workers 8

# Result:
# Processing batch 1/17... ✓
# Processing batch 2/17... ✓
# ...
# Complete: 847 files analyzed in 4m 23s
# Memory usage: 512 MB (constant)
```

### How It Works

1. **Batching** - Processes 50 files at a time (configurable)
2. **Parallel workers** - Uses all CPU cores (default: 8 workers)
3. **Constant memory** - Discards processed batches, never loads everything
4. **Progress tracking** - Shows which batch is running

### When to Use Streaming

| Scenario                 | Use Streaming? | Why                                           |
| ------------------------ | -------------- | --------------------------------------------- |
| < 50 files               | No             | Basic mode is faster (less overhead)          |
| 50-200 files             | Optional       | Depends on file sizes                         |
| 100-300 files            | Yes            | Significantly faster with parallel processing |
| 1000+ files              | Required       | Won't fit in memory otherwise                 |
| Agency with 20+ projects | Yes            | Process all projects in single run            |

### Basic Usage

```bash
# Enable streaming mode
pgsquash squash migrations/*.sql \
  --stream \
  --workers 8 \
  --batch-size 50 \
  --memory-limit 512
```

### Configuration

```json
{
  "performance": {
    "streaming": {
      "enabled": true,
      "batch_size": 50,
      "workers": 8,
      "memory_limit_mb": 512,
      "progress_interval": 5
    }
  }
}
```

### Programmatic Usage

```go
import "github.com/CAPYSQUASH/pgsquash-engine/internal/performance"

// Create memory manager
memManager := performance.NewMemoryManager(512 * 1024 * 1024) // 512MB

// Create streaming processor
processor := performance.NewStreamingProcessor(
    50,  // batch size
    8,   // workers
    memManager,
)

// Start processing
processor.Start()

// Process directory
err := processor.ProcessDirectory("migrations/")
if err != nil {
    log.Fatal(err)
}

// Get results
for result := range processor.GetOutput() {
    if len(result.Errors) > 0 {
        for _, err := range result.Errors {
            log.Printf("Error in %s: %v\n", result.OriginalFile.Path, err)
        }
    } else {
        fmt.Printf("Processed: %s\n", result.OriginalFile.Path)
    }
}

// Wait for completion
processor.Wait()

// Get statistics
stats := processor.GetStats()
fmt.Printf("Files processed: %d\n", stats.FilesProcessed)
fmt.Printf("Throughput: %.2f MB/s\n", stats.ThroughputMBps)
```

### Performance Comparison

**Standard mode** (all in memory):

```
50 migrations × 2MB each = 100MB memory
Processing time: 45 seconds
```

**Streaming mode** (8 workers, 50 batch):

```
50 migrations in memory at once = 100MB memory
Processing time: 12 seconds (3.75× faster)
```

### Memory Management

```go
// Monitor memory usage
memManager.StartMonitoring(ctx)

// Check current usage
usage := memManager.GetCurrentUsage()
fmt.Printf("Memory: %d MB / %d MB\n", usage/1024/1024, memManager.GetLimit()/1024/1024)

// Wait for memory availability
memManager.WaitForMemory(ctx, 50*1024*1024) // Wait for 50MB free

// Release memory after processing
memManager.ReleaseMemory(fileSize)
```

### Progress Tracking

```go
// Create progress tracker
progressTracker := performance.NewProgressTracker(totalFiles)

// Update progress
progressTracker.Increment()

// Get progress
progress := progressTracker.GetProgress()
fmt.Printf("Progress: %.1f%% (%d/%d)\n",
    progress.Percentage, progress.Completed, progress.Total)

// Estimate time remaining
eta := progressTracker.EstimateTimeRemaining()
fmt.Printf("ETA: %s\n", eta)
```

## Performance Optimization

### Batch Processing

Process multiple files efficiently:

```go
// Create batch processor
batchProcessor := performance.NewBatchProcessor(50) // 50 files per batch

// Add files to batch
for _, file := range files {
    batchProcessor.Add(file)
}

// Process all batches
results := batchProcessor.ProcessAll()
```

### Parallel Execution

Utilize multiple CPU cores:

```go
// Create parallel executor
executor := performance.NewParallelExecutor(8) // 8 workers

// Submit tasks
for _, migration := range migrations {
    executor.Submit(func() error {
        return processMigration(migration)
    })
}

// Wait for all tasks
executor.Wait()

// Check for errors
if executor.HasErrors() {
    for _, err := range executor.GetErrors() {
        log.Printf("Error: %v\n", err)
    }
}
```

### Caching Strategies

Reduce redundant processing:

```go
// Cache parsed migrations
cache := performance.NewMigrationCache(100) // 100 entry limit

// Check cache before parsing
if cached, found := cache.Get(filename); found {
    return cached
}

// Parse and cache
migration := parser.Parse(content)
cache.Set(filename, migration)
```

### Memory Profiling

Identify memory bottlenecks:

```bash
# Run with profiling
pgsquash squash migrations/*.sql --profile-memory

# Analyze profile
go tool pprof mem.prof
```

### Performance Benchmarks

Typical performance (400 migrations, 2MB average):

| Mode                   | Memory | Time | Throughput   |
| ---------------------- | ------ | ---- | ------------ |
| Standard               | 2GB    | 180s | 11 files/s   |
| Streaming (4 workers)  | 400MB  | 60s  | 16.7 files/s |
| Streaming (8 workers)  | 400MB  | 35s  | 28.6 files/s |
| Streaming (16 workers) | 600MB  | 25s  | 40 files/s   |

## Best Practices

### 1. Choose the Right Mode

**Use standard mode when:**

- < 100 migration files
- Files are small (< 100KB)
- Plenty of memory available (> 2GB)

**Use streaming mode when:**

- 100+ migration files
- Large files (> 1MB)
- Limited memory (< 1GB)
- Want maximum speed

### 2. Tune Worker Count

```bash
# CPU-bound workload (parsing): workers = CPU cores
pgsquash squash --stream --workers $(nproc)

# I/O-bound workload (database validation): workers = 2× CPU cores
pgsquash validate --stream --workers $(($(nproc) * 2))
```

### 3. Monitor Memory Usage

```bash
# Set appropriate limit (80% of available RAM)
AVAILABLE=$(free -m | awk 'NR==2{print $7}')
LIMIT=$((AVAILABLE * 80 / 100))
pgsquash squash --stream --memory-limit $LIMIT
```

### 4. Handle Errors Gracefully

```go
processor.SetErrorHandler(func(err error, file string) bool {
    log.Printf("Error in %s: %v\n", file, err)
    // Return true to continue, false to abort
    return true
})
```

### 5. Optimize for Your Use Case

**Fast prototyping** (speed over memory):

```bash
pgsquash squash --stream --workers 16 --batch-size 100
```

**Production** (safety and logging):

```bash
pgsquash squash --safety conservative --verbose
```

**Constrained environment** (minimal memory):

```bash
pgsquash squash --stream --workers 2 --batch-size 10 --memory-limit 128
```

## Troubleshooting

### High Memory Usage

**Problem:** Process using too much memory

**Solutions:**

1. Enable streaming mode: `--stream`
2. Reduce batch size: `--batch-size 20`
3. Reduce workers: `--workers 4`
4. Set memory limit: `--memory-limit 256`

### Slow Processing

**Problem:** Processing takes too long

**Solutions:**

1. Enable streaming: `--stream`
2. Increase workers: `--workers $(nproc)`
3. Increase batch size: `--batch-size 100`
4. Use SSD for migration files

### Out of Memory Errors

**Problem:** Process crashes with OOM

**Solutions:**

```bash
# Enable streaming with strict memory limit
pgsquash squash \
  --stream \
  --workers 2 \
  --batch-size 10 \
  --memory-limit 128

# Monitor memory during execution
watch -n 1 'ps aux | grep pgsquash'
```

## Related Documentation

- [Architecture Guide](./architecture.md) - System architecture overview
- [Configuration Guide](./configuration.md) - Configuration reference
- [Performance Guide](./quickstart.md#performance-tuning) - Performance tuning tips
- [CLI Reference](./cli-reference.md) - Command-line options
- [Troubleshooting](./troubleshooting.md) - Common issues and solutions

## Summary

pgsquash's advanced features enable:

1. **SQL Builder** - Programmatic, formatted SQL generation
2. **Metadata Manager** - Efficient database introspection with caching
3. **Object Tracker** - Complete lifecycle and dependency tracking
4. **Streaming Processor** - Memory-efficient processing at scale
5. **Performance Tools** - Parallel execution and optimization

These features work together to handle enterprise-scale migrations efficiently while maintaining accuracy and safety. Use standard mode for small projects, streaming mode for large-scale deployments.
