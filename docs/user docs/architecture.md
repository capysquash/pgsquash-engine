# pgsquash Architecture

This document provides a comprehensive overview of pgsquash's architecture, design patterns, and internal components.

## Table of Contents

- [System Overview](#system-overview)
- [Core Architecture](#core-architecture)
- [Processing Pipeline](#processing-pipeline)
- [Component Details](#component-details)
- [Data Flow](#data-flow)
- [Design Patterns](#design-patterns)
- [Performance Optimizations](#performance-optimizations)
- [Extension Points](#extension-points)

## System Overview

pgsquash is a sophisticated PostgreSQL migration consolidator built on a multi-phase processing architecture that leverages PostgreSQL's actual parser for 100% accurate SQL analysis.

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         CLI Interface                             │
│                    (cmd/pgsquash/main.go)                        │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Command Router                               │
│              (internal/cli/root.go)                              │
│  ┌──────────┬──────────┬──────────┬──────────┬─────────────┐   │
│  │ analyze  │  squash  │ validate │ ai-test  │ safe/fast   │   │
│  └──────────┴──────────┴──────────┴──────────┴─────────────┘   │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Squasher Engine                              │
│                 (internal/squasher/engine.go)                    │
│                                                                   │
│  ┌─────────────┐  ┌──────────────┐  ┌────────────────┐         │
│  │   Parser    │→ │   Tracker    │→ │  Consolidator  │         │
│  └─────────────┘  └──────────────┘  └────────────────┘         │
│         │                │                    │                  │
│         ▼                ▼                    ▼                  │
│  ┌─────────────┐  ┌──────────────┐  ┌────────────────┐         │
│  │ pg_query_go │  │  Dependency  │  │  SQL Builder   │         │
│  │   (Parse)   │  │    Graph     │  │   (Output)     │         │
│  └─────────────┘  └──────────────┘  └────────────────┘         │
└─────────────────────────────────────────────────────────────────┘
                             │
                ┌────────────┼────────────┐
                │            │            │
                ▼            ▼            ▼
     ┌─────────────┐ ┌────────────┐ ┌──────────────┐
     │     AI      │ │ Validation │ │Transformation│
     │  Analysis   │ │  (Docker)  │ │   (SQL)      │
     └─────────────┘ └────────────┘ └──────────────┘
```

### Key Design Principles

1. **Accurate Parsing**: Uses PostgreSQL's official parser (pg\_query\_go)
2. **Safety-First**: Multiple safety levels from paranoid to aggressive
3. **Dependency-Aware**: Comprehensive dependency tracking and resolution
4. **Streaming Support**: Memory-efficient processing for large datasets
5. **Extensible**: Modular rule engine for consolidation strategies
6. **Observable**: Comprehensive progress tracking and metrics

## Core Architecture

### Multi-Phase Processing

pgsquash processes migrations through six distinct phases (see [Migration Consolidation Strategy](internal/decisions/migration-consolidation-strategy.md) for complete details):

```
Phase 0: PLUGIN INITIALIZATION
├─ Auto-detect third-party patterns (Clerk, Supabase, Prisma, etc.)
├─ Initialize plugins by priority (Auth: 90-100, ORMs: 70-89)
├─ Prepare SQL transformations
└─ Enable plugin-specific consolidation rules

Phase 1: PARSING
├─ Read migration files
├─ Parse SQL using pg_query_go
├─ Extract metadata (objects, operations, dependencies)
└─ Build Statement objects

Phase 2: TRACKING
├─ Track object lifecycles
├─ Build dependency graph
├─ Detect redundancies
└─ Identify consolidation opportunities

Phase 3: ANALYSIS
├─ Resolve dependencies
├─ Detect circular dependencies
├─ Perform DDL cycle detection (DROP → CREATE patterns)
└─ Run AI analysis (optional)

Phase 4: CONSOLIDATION
├─ Apply safety-level-appropriate rules
├─ Consolidate CREATE + ALTER sequences
├─ Remove redundant operations
├─ Apply plugin-specific consolidation
└─ Optimize object definitions

Phase 5: POST-PROCESSING
├─ Fix malformed DROP TRIGGER syntax
├─ Reorder extensions by dependency (cube before earthdistance)
├─ Remove orphaned ALTER statements
├─ Fix malformed function definitions
├─ Add missing semicolons
└─ Rewrite eliminated ENUM references

Phase 6: GENERATION
├─ Sort by dependencies
├─ Group by 8 categories (Extensions → Data)
├─ Format SQL output with modern PostgreSQL syntax
└─ Generate final migrations
```

### Component Hierarchy

```
internal/
├── cli/              # Command-line interface
│   └── root.go       # Command definitions and handlers
├── parser/           # SQL parsing and analysis
│   ├── parser.go     # Main parsing logic
│   ├── normalization.go  # SQL normalization
│   └── errors.go     # Error handling
├── tracking/         # Object lifecycle tracking
│   ├── tracker.go    # Main tracker
│   ├── lifecycle.go  # Object lifecycle management
│   ├── dependency_graph.go  # Dependency resolution
│   └── streaming_tracker.go # Streaming support
├── squasher/         # Consolidation engine
│   ├── engine.go     # Main squasher engine
│   ├── unified_dependency_resolver.go
│   └── modern_patterns.go
├── builder/          # SQL generation
│   └── sql_builder.go
├── config/           # Configuration management
│   └── config.go
├── ai/               # AI-powered analysis
│   ├── analyzer.go
│   ├── manager.go
│   └── providers/
├── validation/       # Schema validation
│   └── validator.go
├── transformation/   # SQL transformation
│   ├── sql_transformer.go
│   ├── backup_generator.go
│   └── rollback_manager.go
├── performance/      # Performance optimization
│   ├── memory_manager.go
│   └── batch_processor.go
├── metadata/         # Database metadata
│   └── metadata_manager.go
├── diff/             # Schema comparison and diffing
│   └── differ.go
├── errors/           # Error handling framework
│   └── errors.go     # Error codes, categories, severities
├── fileutil/         # File utilities
│   └── fileutil.go
├── github/           # GitHub webhook integration
│   └── github.go     # PR analysis and automation
├── output/           # Output organization
│   └── organizer.go
├── patterns/         # SQL pattern detection
│   └── detector.go   # Auth patterns, ORM detection
├── plugins/          # Plugin system
│   ├── plugin.go     # Plugin interface
│   ├── registry.go   # Plugin registration
│   ├── auth/         # Generic auth plugin
│   ├── clerk/        # Clerk auth plugin
│   ├── supabase/     # Supabase platform plugin
│   ├── prisma/       # Prisma ORM plugin
│   ├── drizzle/      # Drizzle ORM plugin
│   └── volatility/   # Function volatility plugin
├── tui/              # Terminal user interface
│   └── tui.go        # Interactive TUI components
├── types/            # Core type definitions
│   └── types.go      # Shared types across modules
└── utils/            # Utility functions
    └── utils.go      # Helper functions
```

### Supporting Modules

Beyond the core processing pipeline, pgsquash includes several supporting modules:

**diff/** - Schema Comparison
- Byte-level schema comparison using pg_dump
- Schema difference detection and reporting
- Validation result formatting
- Used by the validate command for schema equivalence checking

**errors/** - Error Handling Framework
- Structured error types with error codes (ErrorCodeSyntaxError, ErrorCodeValidationFailed, etc.)
- Error categories (CategoryParsing, CategoryValidation, CategoryDependency, etc.)
- Severity levels (SeverityError, SeverityWarning, SeverityInfo)
- Contextual error messages with file locations, line numbers, and suggestions
- Error wrapping and chaining for detailed error traces

**fileutil/** - File Utilities
- File discovery and pattern matching
- Migration file reading and writing
- Path normalization and validation
- Backup file management

**github/** - GitHub Integration
- Webhook handling for pull request events
- Automated migration analysis on PR creation
- PR comment generation with analysis results
- CI/CD integration support for GitHub Actions

**output/** - Output Organization
- Migration file organization by category (extensions, schema, constraints, etc.)
- Semantic file naming strategies
- Comment preservation and generation
- SQL formatting and prettification

**patterns/** - Pattern Detection
- Authentication provider detection (Supabase auth.users, Clerk JWT, Auth0)
- ORM pattern recognition (Prisma _prisma_migrations, Drizzle schemas)
- Storage schema detection (Supabase storage.buckets)
- SQL dialect and PostgreSQL version detection
- Auth pattern analysis for RLS policies and security

**plugins/** - Plugin System
- Plugin interface defining lifecycle hooks (Detect, Initialize, EnrichStatement, etc.)
- Plugin registry with priority-based ordering
- Auto-detection of third-party patterns in migrations
- Built-in plugins:
  - **auth/**: Generic authentication pattern handling
  - **clerk/**: Clerk JWT v2, organization support, metadata
  - **supabase/**: RLS policies, storage buckets, auth schema compatibility
  - **prisma/**: Metadata tables, enum handling, migration tracking
  - **drizzle/**: Identity columns, generated columns
  - **volatility/**: PostgreSQL function volatility detection and fixing
- Custom plugin support for proprietary systems

**tui/** - Terminal User Interface
- Interactive dashboard with migration overview
- Real-time analysis view with lifecycle patterns
- Configuration wizard for interactive setup
- Dependency graph visualization
- Progress monitoring during squashing operations
- Built with Bubble Tea framework

**types/** - Core Type Definitions
- Migration and Statement structures
- Object lifecycle representations
- Dependency graph types
- Configuration types
- Shared enums and constants used across all modules

**utils/** - Utility Functions
- String manipulation and formatting
- Collection utilities (map, filter, reduce operations)
- Common helper functions
- Logging utilities



## Processing Pipeline

### Detailed Phase Breakdown

#### Phase 1: Parser Layer

**Purpose**: Convert raw SQL into structured data using PostgreSQL's actual parser

**Components**:

- `parser.ParseMigration()`: Main entry point
- `pg_query.SplitWithScanner()`: Statement splitting
- `pg_query.Parse()`: AST generation

**Process**:

```go
// Parse migration file
migration, err := parser.ParseMigration(content, filename)

// Each statement becomes:
type Statement struct {
    SQL          string         // Original SQL
    ParseTree    *pg_query.ParseResult  // AST
    ObjectType   ObjectType     // TABLE, INDEX, etc.
    ObjectName   string         // Object identifier
    Operation    Operation      // CREATE, ALTER, etc.
    Category     Category       // Categorization
    Dependencies []string       // Object dependencies
}
```

**Key Features**:

- **Error Recovery**: Continues parsing after errors
- **Normalization**: Handles PostgreSQL-specific syntax
- **Metadata Extraction**: Identifies objects, operations, dependencies
- **Comment Preservation**: Tracks and preserves comments

#### Phase 2: Tracking Layer

**Purpose**: Build complete object lifecycles across all migrations

**Components**:

- `Tracker`: Main tracking engine
- `ObjectLifecycle`: Lifecycle management
- `DependencyGraph`: Dependency tracking

**Process**:

```go
// Track each migration
tracker := tracking.NewTracker()
tracker.ProcessMigration(migration, sequence)

// Object lifecycle:
type ObjectLifecycle struct {
    Key           string
    ObjectType    ObjectType
    CreatedAt     int              // Migration sequence
    Modifications []Statement       // All changes
    FinalState    *Statement       // Consolidated state
    Dependencies  []string
    Category      Category
}
```

**Tracking Capabilities**:

- CREATE → ALTER → DROP sequences
- Multi-stage modifications
- Dependency chains
- Cross-schema references
- RLS policy evolution

#### Phase 3: Analysis Layer

**Purpose**: Analyze dependencies, detect issues, and assess risks

**Components**:

- `UnifiedDependencyResolver`: Advanced dependency resolution
- `DependencyGraph`: Graph operations
- DDL Cycle Detector: Circular dependency detection

**Process**:

```go
// Build dependency graph
graph := tracker.GetActualDependencyGraph()

// Topological sort
sorted, err := graph.TopologicalSort()

// Detect cycles
cycles := graph.DetectCycles()

// DDL cycle detection
tracker.DetectDDLCycles()
```

**Analysis Types**:

1. **Dependency Analysis**:
   - Table → Index dependencies
   - Function → Table dependencies
   - Trigger → Function dependencies
   - Cross-schema dependencies

2. **Cycle Detection**:
   - Simple cycles (A → B → A)
   - Complex cycles (A → B → C → A)
   - Self-references
   - Mutual dependencies

3. **Risk Assessment**:
   - Breaking change detection
   - Data loss risks
   - Constraint violations

#### Phase 4: Consolidation Layer

**Purpose**: Apply safety-appropriate consolidation rules

**Rule Engine Architecture**:

```go
type ConsolidationRule interface {
    Name() string
    Description() string
    Applies(lifecycle *ObjectLifecycle, engine RuleContext) bool
    Apply(lifecycle *ObjectLifecycle, engine RuleContext) (*ConsolidationResult, error)
    Priority() int
    SafetyLevel() SafetyLevel
}
```

**Built-in Rules**:

1. **CreateAlterConsolidationRule**:
   - Merges CREATE + ALTER statements
   - Consolidates column definitions
   - Preserves column order

2. **DropCreateCycleRule**:
   - Detects DROP → CREATE patterns
   - Removes redundant drops
   - Preserves data migrations

3. **FunctionDeduplicationRule**:
   - Detects duplicate function definitions
   - Uses semantic equivalence checking
   - AI-powered comparison (optional)

4. **RLSConsolidationRule**:
   - Groups related RLS policies
   - Consolidates policy definitions
   - Preserves security semantics

5. **ColumnEvolutionRule**:
   - Tracks column lifecycle
   - Consolidates column modifications
   - Handles renames and type changes

**Rule Application**:

```go
// Apply rules for each object
for _, lifecycle := range lifecycles {
    result, err := ruleEngine.ApplyRules(lifecycle, engine)

    // Result contains:
    type ConsolidationResult struct {
        OriginalStatements []Statement
        ConsolidatedSQL    string
        Optimizations      []string
        Warnings           []string
        RiskLevel          RiskLevel
    }
}
```

#### Phase 5: Builder Layer

**Purpose**: Generate optimized SQL with proper formatting and ordering

**Components**:

- `SQLBuilder`: SQL generation with formatting
- Category ordering: Extensions → Tables → Constraints → Functions → Triggers → Indexes → Security → Data

**Generation Process**:

```go
builder := builder.NewSQLBuilder(options)

// Add header
builder.Comment("Generated by pgsquash")

// Process by category (in dependency order)
categories := []Category{
    CategoryExtensions,
    CategoryFoundation,
    CategoryConstraints,
    CategoryFunctions,
    CategoryTriggers,
    CategoryIndexes,
    CategorySecurity,
    CategoryData,
}

for _, category := range categories {
    objects := getObjectsByCategory(category)
    sorted := sortByDependencies(objects)

    for _, obj := range sorted {
        builder.Statement(obj.ConsolidatedSQL)
    }
}

finalSQL := builder.String()
```

**Output Features**:

- Dependency-aware ordering
- Category grouping
- Comment preservation
- Modern PostgreSQL syntax
- Readable formatting

## Component Details

### Parser Component

**Location**: `internal/parser/`

**Responsibilities**:

- Parse SQL using pg\_query\_go
- Extract metadata (objects, operations, dependencies)
- Normalize PostgreSQL-specific syntax
- Handle parsing errors with recovery

**Key Types**:

```go
type Migration struct {
    Filename   string
    Sequence   int
    Statements []Statement
    Size       int64
}

type Statement struct {
    SQL          string
    ParseTree    *pg_query.ParseResult
    ObjectType   ObjectType
    ObjectName   string
    Operation    Operation
    Category     Category
    Dependencies []string
    Schema       string
    AuthPattern  AuthPatternType
}
```

**Special Features**:

- **Auth Pattern Detection**: Identifies Supabase, Clerk, Auth0 patterns
- **Extension Support**: Handles pgvector, PostGIS, etc.
- **Storage Support**: Recognizes Supabase storage policies
- **Modern Syntax**: Supports PostgreSQL 15+ features

### Tracking Component

**Location**: `internal/tracking/`

**Responsibilities**:

- Track object lifecycles across migrations
- Build and maintain dependency graph
- Detect redundancies and optimization opportunities
- Validate consistency

**Key Types**:

```go
type Tracker struct {
    lifecycles    map[string]*ObjectLifecycle
    dependencies  *DependencyGraph
    sequences     map[string][]int
    metadata      *metadata.MetadataManager
}

type ObjectLifecycle struct {
    Key           string
    ObjectType    ObjectType
    CreatedAt     int
    Modifications []Statement
    FinalState    *Statement
    DroppedAt     int
    Dependencies  []string
    Category      Category
}
```

**Streaming Support**:

```go
type StreamingTracker struct {
    tracker       *Tracker
    batchSize     int
    workerCount   int
    memManager    *performance.MemoryManager
    progressCb    ProgressCallback
}
```

### Squasher Engine

**Location**: `internal/squasher/engine.go`

**Responsibilities**:

- Orchestrate the entire squashing process
- Manage component lifecycle
- Apply consolidation rules
- Generate final output

**Configuration**:

```go
type EngineConfig struct {
    Config              *config.Config
    EnableStreaming     bool
    BatchSize           int
    WorkerCount         int
    MemoryLimitMB       int
    EnableProgressTrack bool
    ProgressCallback    func(processed, total int64, phase string)

    // Transformation options
    EnableBackup         bool
    EnableRollback       bool
    EnableTransformation bool
    BackupConfig         *transformation.BackupConfig
    TransformationConfig *transformation.TransformationConfig
}
```

**Processing Modes**:

1. **Standard Mode**: Load all migrations into memory
2. **Streaming Mode**: Process migrations in batches
3. **High-Performance Mode**: Optimized for 500+ migrations

### AI Integration

**Location**: `internal/ai/`

**Responsibilities**:

- Semantic function analysis
- Dead code detection
- Authentication pattern recognition
- Performance optimization suggestions

**Provider Architecture**:

```go
type Provider interface {
    Name() string
    Analyze(ctx context.Context, request *AnalysisRequest) (*AnalysisResponse, error)
    SupportedAnalysisTypes() []AnalysisType
    IsAvailable() bool
}

// Supported providers:
- Claude (Anthropic)
- OpenAI (GPT-4)
```

**Analysis Types**:

- Function semantic equivalence
- Dead code identification
- Authentication pattern detection
- Schema consistency validation
- Performance optimization suggestions
- Complexity analysis

### Validation Component

**Location**: `internal/validation/`

**Responsibilities**:

- Docker-based schema validation
- Schema diff generation
- Extension detection and installation
- SQL error fixing

**Validation Approaches**:

1. **TWO\_CONTAINERS**: Most accurate
   - Original migrations → Container 1
   - Squashed migrations → Container 2
   - Compare schemas using pg\_dump

2. **SCHEMA\_DIFF**: Fastest
   - Apply both sets to same container
   - Use schema versioning
   - Compare versions

3. **SINGLE\_CONTAINER**: Simplest
   - Apply original migrations
   - Reset database
   - Apply squashed migrations

### Transformation Component

**Location**: `internal/transformation/`

**Responsibilities**:

- SQL modernization
- Backup generation
- Rollback script creation
- Safety transformations

**Transformation Types**:

```go
type Transformation struct {
    Type        TransformationType
    Description string
    Original    string
    Transformed string
    RiskLevel   RiskLevel
}

// Types:
- UnsafeToSafe: Convert unsafe ops to safe equivalents
- ModernSyntax: Use modern PostgreSQL syntax
- Performance: Apply performance optimizations
- DMLToSelect: Convert destructive DML to SELECT
```

## Data Flow

### Complete Request Flow

```
1. CLI Command
   └─> cmd/pgsquash/main.go
       └─> cli.Execute()

2. Command Router
   └─> internal/cli/root.go
       └─> runSquash() / runAnalyze() / etc.

3. Load Configuration
   └─> config.LoadConfig()
       └─> Read pgsquash.config.json

4. Initialize Engine
   └─> squasher.NewEngine(config)
       ├─> parser.NewParser()
       ├─> tracking.NewTracker()
       ├─> builder.NewSQLBuilder()
       └─> Optional: ai.NewAnalyzer()

5. Process Migrations
   └─> engine.Squash(migrations)
       ├─> parseAndTrackMigrations()
       │   ├─> parser.ParseMigration() for each file
       │   └─> tracker.ProcessMigration()
       │
       ├─> analyzeDependenciesAndRisks()
       │   ├─> Build dependency graph
       │   ├─> Detect cycles
       │   └─> DDL cycle detection
       │
       ├─> applyConsolidationRules()
       │   └─> ruleEngine.ApplyRules() for each object
       │
       └─> generateOptimizedSQL()
           ├─> Sort by dependencies
           ├─> Group by categories
           └─> builder.Build()

6. Optional Validation
   └─> validator.ValidateWithDocker()
       ├─> Start Docker containers
       ├─> Apply migrations
       └─> Compare schemas

7. Write Output
   └─> Write final SQL to file
```

### Streaming Data Flow

For large datasets (>100 migrations):

```
1. Initialize Streaming Components
   ├─> MemoryManager (track memory usage)
   ├─> BatchProcessor (process in batches)
   └─> StreamingTracker (streaming lifecycle tracking)

2. Batch Processing
   └─> For each batch of migrations:
       ├─> Check memory constraints
       ├─> Parse batch
       ├─> Track lifecycles
       ├─> Release memory
       └─> Update progress

3. Consolidation (uses standard engine)
   └─> Same as regular flow, using streamed data

4. Efficient Output
   └─> Stream SQL generation to file
```

## Design Patterns

### 1. Strategy Pattern

**Used For**: Consolidation rules

```go
type ConsolidationRule interface {
    Apply(*ObjectLifecycle, RuleContext) (*ConsolidationResult, error)
}

// Different strategies for different safety levels
engine.AddRule(&CreateAlterConsolidationRule{})
engine.AddRule(&DropCreateCycleRule{})
engine.AddRule(&FunctionDeduplicationRule{})
```

### 2. Builder Pattern

**Used For**: SQL generation

```go
builder := NewSQLBuilder(options)
builder.Comment("Header")
builder.Statement("CREATE TABLE...")
builder.NL()
sql := builder.String()
```

### 3. Observer Pattern

**Used For**: Progress tracking

```go
engine.SetProgressCallback(func(processed, total int64, phase string) {
    fmt.Printf("Progress: %.1f%%\n", float64(processed)/float64(total)*100)
})
```

### 4. Factory Pattern

**Used For**: AI provider creation

```go
manager := NewProviderManager(config)
provider := manager.GetProvider("claude")
```

### 5. Chain of Responsibility

**Used For**: Error recovery

```go
errorHandler.HandleParseError(err, context)
if !errorHandler.ShouldContinue() {
    return err
}
// Continue with recovery
```

## Performance Optimizations

### Memory Management

**Streaming Mode**:

- Process migrations in batches
- Release memory after each batch
- Configurable memory limits
- Automatic memory pressure detection

```go
memManager := performance.NewMemoryManager(memoryLimitMB)
if !memManager.TrackMemoryUsage(size) {
    // Force cleanup
    tracker.ClearProcessedMigrations()
}
```

### Parallel Processing

**Worker Pools**:

- Concurrent parsing
- Parallel consolidation
- Configurable worker count

```go
config := EngineConfig{
    WorkerCount: runtime.NumCPU(),
    BatchSize: 50,
}
```

### Caching

**Metadata Caching**:

- Cache database metadata
- Configurable TTL
- Automatic invalidation

```go
metadataManager := metadata.NewMetadataManager(db, 15*time.Minute)
```

## Extension Points

### Custom Consolidation Rules

Implement the `ConsolidationRule` interface:

```go
type MyCustomRule struct{}

func (r *MyCustomRule) Name() string {
    return "custom_rule"
}

func (r *MyCustomRule) Applies(lifecycle *ObjectLifecycle, engine RuleContext) bool {
    // Check if rule applies
    return lifecycle.ObjectType == TypeTable
}

func (r *MyCustomRule) Apply(lifecycle *ObjectLifecycle, engine RuleContext) (*ConsolidationResult, error) {
    // Apply custom consolidation logic
    return &ConsolidationResult{
        ConsolidatedSQL: "...",
        Optimizations: []string{"custom_optimization"},
    }, nil
}

// Register rule
engine.ruleEngine.AddRule(&MyCustomRule{})
```

### Custom AI Providers

Implement the `Provider` interface:

```go
type MyProvider struct{}

func (p *MyProvider) Analyze(ctx context.Context, req *AnalysisRequest) (*AnalysisResponse, error) {
    // Custom analysis logic
    return &AnalysisResponse{
        Result: "analysis result",
    }, nil
}

// Register provider
manager.RegisterProvider(MyProvider{})
```

### Custom Validation Strategies

Implement custom validation logic:

```go
type MyValidator struct{}

func (v *MyValidator) Validate(original, squashed string) (*ValidationResult, error) {
    // Custom validation logic
    return &ValidationResult{
        Success: true,
    }, nil
}
```

## Testing Architecture

pgsquash includes comprehensive testing:

- **Unit Tests**: Test individual components
- **Integration Tests**: Test component interactions
- **Validation Tests**: Test schema equivalence
- **Performance Tests**: Benchmark large datasets

```bash
# Run all tests
go test ./...

# Test specific package
go test ./internal/parser

# Run with coverage
go test -cover ./...
```

---

This architecture enables pgsquash to safely and efficiently consolidate PostgreSQL migrations while maintaining 100% schema equivalence and providing extensive customization options.
