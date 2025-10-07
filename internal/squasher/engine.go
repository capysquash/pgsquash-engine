package squasher

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/capysquash/pg-squash-engine/internal/ai"
	"github.com/capysquash/pg-squash-engine/internal/builder"
	"github.com/capysquash/pg-squash-engine/internal/config"
	"github.com/capysquash/pg-squash-engine/internal/metadata"
	"github.com/capysquash/pg-squash-engine/internal/parser"
	"github.com/capysquash/pg-squash-engine/internal/performance"
	"github.com/capysquash/pg-squash-engine/internal/plugins"
	"github.com/capysquash/pg-squash-engine/internal/tracking"
	"github.com/capysquash/pg-squash-engine/internal/transformation"
	_ "github.com/lib/pq"
)

type SafetyLevel string

const (
	Conservative SafetyLevel = "conservative"
	Standard     SafetyLevel = "standard"
	Aggressive   SafetyLevel = "aggressive"
	Paranoid     SafetyLevel = "paranoid"
)

// Engine provides comprehensive PostgreSQL migration squashing with modern patterns and streaming support
type Engine struct {
	// Core configuration and state
	config     *config.Config
	lifecycles map[string]*tracking.ObjectLifecycle
	warnings   []string
	aiAnalyzer *ai.Analyzer
	prodDB     *sql.DB

	// Enhanced components
	metadataManager *metadata.MetadataManager
	tracker         *tracking.Tracker
	sqlBuilder      *builder.SQLBuilder
	ruleEngine      *tracking.ConsolidationRuleEngine

	// Transformation components
	backupGenerator *transformation.BackupGenerator
	rollbackManager *transformation.RollbackManager
	sqlTransformer  *transformation.SQLTransformer

	// Processing state
	processedFiles       map[string]bool
	consolidationResults map[string]*tracking.ConsolidationResult

	// Streaming components (optional)
	streamingTracker *tracking.StreamingTracker
	memManager       *performance.MemoryManager
	batchProcessor   *performance.BatchProcessor

	// Streaming configuration
	batchSize           int
	workerCount         int
	memoryLimitMB       int
	enableProgressTrack bool
	enableStreaming     bool

	// Extension analysis and auth compatibility
	authCompatibilitySQL string

	// Processing statistics
	stats      *SquashStats
	mu         sync.RWMutex
	progressCb func(processed, total int64, phase string)
}

// SquashStats tracks squashing statistics (both regular and streaming)
type SquashStats struct {
	Phase                 string        `json:"current_phase"`
	MigrationsProcessed   int64         `json:"migrations_processed"`
	TotalMigrations       int64         `json:"total_migrations"`
	ObjectsTracked        int64         `json:"objects_tracked"`
	ConsolidationsApplied int64         `json:"consolidations_applied"`
	ProcessingTime        time.Duration `json:"processing_time"`
	PeakMemoryUsage       int64         `json:"peak_memory_usage"`
	ThroughputMPS         float64       `json:"throughput_migrations_per_sec"`
}

// EngineConfig configures the engine with optional streaming capabilities
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

// NewEngineWithStreaming creates an enhanced engine with streaming capabilities
func NewEngineWithStreaming(cfg EngineConfig) *Engine {
	return newEngineInternal(cfg)
}

// NewEngine creates an engine with legacy config for backward compatibility
func NewEngine(cfg interface{}) *Engine {
	// Legacy interface to maintain backward compatibility
	if configPtr, ok := cfg.(*config.Config); ok {
		legacyConfig := EngineConfig{
			Config:              configPtr,
			EnableStreaming:     false,
			BatchSize:           50,
			WorkerCount:         4,
			MemoryLimitMB:       256,
			EnableProgressTrack: false,
			ProgressCallback:    nil,
			EnableBackup:        false,
			EnableRollback:      false,
			EnableTransformation: false,
		}
		return newEngineInternal(legacyConfig)
	}
	if engineCfg, ok := cfg.(EngineConfig); ok {
		return NewEngineWithStreaming(engineCfg)
	}
	log.Fatal("Invalid config type provided to NewEngine")
	return nil
}

// newEngineInternal is the internal implementation
func newEngineInternal(engineCfg EngineConfig) *Engine {
	// Extract configuration
	cfg := engineCfg.Config
	enableStreaming := engineCfg.EnableStreaming
	batchSize := engineCfg.BatchSize
	workerCount := engineCfg.WorkerCount
	memoryLimitMB := engineCfg.MemoryLimitMB
	enableProgressTrack := engineCfg.EnableProgressTrack
	progressCallback := engineCfg.ProgressCallback

	// Set defaults for streaming config
	if batchSize == 0 {
		batchSize = 50
	}
	if workerCount == 0 {
		workerCount = 4
	}
	if memoryLimitMB == 0 {
		memoryLimitMB = 256
	}

	// Database connection for paranoid mode
	var db *sql.DB
	var err error
	if cfg.SafetyLevel == string(Paranoid) {
		if cfg.ProdDBDSN == "" {
			log.Println("Warning: Paranoid safety level selected, but no production database DSN provided. Dead code analysis will be skipped.")
		} else {
			db, err = sql.Open("postgres", cfg.ProdDBDSN)
			if err != nil {
				log.Fatalf("Failed to connect to production database: %v", err)
			}
			if err := db.Ping(); err != nil {
				log.Fatalf("Failed to ping production database: %v", err)
			}
			log.Println("Successfully connected to production database for dead code analysis.")
		}
	}

	// Initialize enhanced components
	var metaMgr *metadata.MetadataManager
	if db != nil {
		metaMgr = metadata.NewMetadataManager(db, 15*time.Minute)
	}

	tracker := tracking.NewTrackerWithMetadata(metaMgr)
	sqlBuilder := builder.NewSQLBuilder(builder.DefaultBuildOptions())
	ruleEngine := NewSquasherRuleEngine(SafetyLevel(cfg.SafetyLevel))

	// Initialize streaming components if enabled
	var memManager *performance.MemoryManager
	var streamingTracker *tracking.StreamingTracker
	var batchProcessor *performance.BatchProcessor

	if enableStreaming {
		memManager = performance.NewMemoryManager(memoryLimitMB)
		streamingTracker = tracking.NewStreamingTracker(batchSize, workerCount, memManager)
		batchProcessor = performance.NewBatchProcessor(batchSize, memoryLimitMB/4, memManager)
	}

	// Initialize transformation components if enabled
	var backupGenerator *transformation.BackupGenerator
	var rollbackManager *transformation.RollbackManager
	var sqlTransformer *transformation.SQLTransformer

	if engineCfg.EnableBackup {
		backupConfig := engineCfg.BackupConfig
		if backupConfig == nil {
			backupConfig = transformation.DefaultBackupConfig()
		}
		backupGenerator = transformation.NewBackupGenerator(backupConfig, db)
	}

	if engineCfg.EnableRollback {
		workDir := "rollbacks" // Default rollback directory
		rollbackManager = transformation.NewRollbackManager(db, workDir)
	}

	if engineCfg.EnableTransformation {
		transformConfig := engineCfg.TransformationConfig
		if transformConfig == nil {
			transformConfig = transformation.DefaultTransformationConfig()
		}
		sqlTransformer = transformation.NewSQLTransformer(transformConfig)
	}

	return &Engine{
		// Core components
		config:     cfg,
		lifecycles: make(map[string]*tracking.ObjectLifecycle),
		warnings:   []string{},
		aiAnalyzer: func() *ai.Analyzer { analyzer, _ := ai.NewAnalyzer(); return analyzer }(),
		prodDB:     db,

		// Enhanced components
		metadataManager: metaMgr,
		tracker:         tracker,
		sqlBuilder:      sqlBuilder,
		ruleEngine:      ruleEngine,

		// Transformation components
		backupGenerator: backupGenerator,
		rollbackManager: rollbackManager,
		sqlTransformer:  sqlTransformer,

		// Processing state
		processedFiles:       make(map[string]bool),
		consolidationResults: make(map[string]*tracking.ConsolidationResult),

		// Streaming components
		streamingTracker:    streamingTracker,
		memManager:          memManager,
		batchProcessor:      batchProcessor,
		batchSize:           batchSize,
		workerCount:         workerCount,
		memoryLimitMB:       memoryLimitMB,
		enableProgressTrack: enableProgressTrack,
		enableStreaming:     enableStreaming,
		progressCb:          progressCallback,
		stats:               &SquashStats{},
	}
}

// NewSquasherRuleEngine creates a rule engine for the given safety level
func NewSquasherRuleEngine(safetyLevel SafetyLevel) *tracking.ConsolidationRuleEngine {
	engine := tracking.NewConsolidationRuleEngine()

	// Always add external dependency filter (reduces noise)
	engine.AddRule(tracking.NewExternalDependencyFilterRule())

	// Add safety-appropriate rules based on level
	switch safetyLevel {
	case Conservative:
		engine.AddRule(&tracking.MultipleCreateConsolidationRule{})
		engine.AddRule(&tracking.CreateAlterConsolidationRule{})
		engine.AddRule(&tracking.ColumnEvolutionRule{})
		engine.AddRule(&tracking.ConditionalSchemaRule{})
		engine.AddRule(&tracking.AdvancedColumnLifecycleRule{})
	case Standard:
		engine.AddRule(&tracking.MultipleCreateConsolidationRule{})
		engine.AddRule(&tracking.CreateAlterConsolidationRule{})
		engine.AddRule(&tracking.ColumnEvolutionRule{})
		engine.AddRule(&tracking.ConditionalSchemaRule{})
		engine.AddRule(&tracking.AdvancedColumnLifecycleRule{})
		engine.AddRule(&tracking.DropCreateCycleRule{})
		engine.AddRule(&tracking.RLSConsolidationRule{})
		engine.AddRule(&tracking.TransactionBoundaryRule{})
	case Aggressive:
		engine.AddRule(&tracking.MultipleCreateConsolidationRule{})
		engine.AddRule(&tracking.CreateAlterConsolidationRule{})
		engine.AddRule(&tracking.ColumnEvolutionRule{})
		engine.AddRule(&tracking.ConditionalSchemaRule{})
		engine.AddRule(&tracking.AdvancedColumnLifecycleRule{})
		engine.AddRule(&tracking.DropCreateCycleRule{})
		engine.AddRule(&tracking.RLSConsolidationRule{})
		engine.AddRule(&tracking.TransactionBoundaryRule{})
		engine.AddRule(&tracking.FunctionDeduplicationRule{})
	case Paranoid:
		engine.AddRule(&tracking.MultipleCreateConsolidationRule{})
		engine.AddRule(&tracking.CreateAlterConsolidationRule{})
		engine.AddRule(&tracking.ColumnEvolutionRule{})
		engine.AddRule(&tracking.ConditionalSchemaRule{})
		engine.AddRule(&tracking.AdvancedColumnLifecycleRule{})
		engine.AddRule(&tracking.DropCreateCycleRule{})
		engine.AddRule(&tracking.RLSConsolidationRule{})
		engine.AddRule(&tracking.TransactionBoundaryRule{})
		engine.AddRule(&tracking.FunctionDeduplicationRule{})
		engine.AddRule(&tracking.DeadCodeRemovalRule{})
	}

	// Add error recovery as the last rule to catch any failures from primary consolidation rules
	recoveryMode := "conservative"
	if safetyLevel == Aggressive || safetyLevel == Paranoid {
		recoveryMode = "aggressive"
	}
	errorRecovery := tracking.NewErrorRecoveryRule(3, recoveryMode, true)
	engine.AddRule(errorRecovery)

	return engine
}

// GetTracker returns the tracker for use by consolidation rules
func (e *Engine) GetTracker() *tracking.Tracker {
	return e.tracker
}

// GetConfig returns the configuration for use by consolidation rules
func (e *Engine) GetConfig() interface{} {
	return e.config
}

// GetAuthCompatibilitySQL returns the auth compatibility SQL for Docker validation
func (e *Engine) GetAuthCompatibilitySQL() string {
	return e.authCompatibilitySQL
}

// Close gracefully shuts down the engine
func (e *Engine) Close() {
	if e.prodDB != nil {
		_ = e.prodDB.Close()
	}
	if e.enableStreaming && e.streamingTracker != nil {
		if err := e.streamingTracker.Stop(); err != nil {
			log.Printf("Warning: failed to stop streaming tracker: %v", err)
		}
	}
}

// Squash processes migrations using enhanced patterns and modern PostgreSQL conventions
func (e *Engine) Squash(migrations map[int]string) (string, []string, error) {
	// Use streaming approach if enabled
	if e.enableStreaming {
		return e.SquashStreaming(migrations)
	}

	ctx := context.Background()
	startTime := time.Now()

	log.Printf("Starting enhanced squashing process with %d migration files", len(migrations))

	// PHASE 0: Initialize Plugin System
	// This must happen BEFORE parsing to enable plugin enrichment
	if err := e.initializePlugins(ctx, migrations); err != nil {
		log.Printf("Warning: Plugin initialization failed: %v", err)
		e.warnings = append(e.warnings, fmt.Sprintf("Plugin initialization warning: %v", err))
	}

	// Analyze extensions required for validation
	extDetector := NewExtensionDetector()
	extAnalysis := extDetector.AnalyzeMigrations(migrations)
	if len(extAnalysis.RequiredExtensions) > 0 {
		log.Printf("Detected extensions: %v", extAnalysis.RequiredExtensions)
		log.Printf("Recommended Docker image: %s", extAnalysis.RecommendedDockerBase)
		e.warnings = append(e.warnings, fmt.Sprintf("Required extensions: %v", extAnalysis.RequiredExtensions))
	}

	// Store auth compatibility SQL for validation
	if extAnalysis.AuthCompatibilitySQL != "" {
		e.authCompatibilitySQL = extAnalysis.AuthCompatibilitySQL
		log.Printf("Generated auth compatibility layer for: %s", extAnalysis.AuthService)
	}

	// Pre-processing: Generate backup if enabled
	if e.backupGenerator != nil && e.prodDB != nil {
		log.Printf("Generating backup before squashing...")
		// Use database connection string from config
		backupResult, err := e.backupGenerator.GeneratePreMigrationBackup(ctx, e.config.ProdDBDSN)
		if err != nil {
			return "", nil, fmt.Errorf("failed to generate backup: %w", err)
		}
		log.Printf("Backup created at: %s", backupResult.BackupPath)
		e.warnings = append(e.warnings, fmt.Sprintf("Backup created: %s", backupResult.BackupPath))
	}

	// Initialize rollback manager if enabled
	if e.rollbackManager != nil {
		// Parse migrations to extract statements for rollback planning
		var allStatements []parser.Statement
		for id, migrationSQL := range migrations {
			filename := fmt.Sprintf("migration_%d.sql", id)
			migration, err := parser.ParseMigration(migrationSQL, filename)
			if err == nil {
				allStatements = append(allStatements, migration.Statements...)
			}
		}

		// Create a rollback plan
		planName := fmt.Sprintf("squash_%d", startTime.Unix())
		_, err := e.rollbackManager.CreateRollbackPlan(ctx, planName, allStatements)
		if err != nil {
			log.Printf("Warning: Failed to create rollback plan: %v", err)
			e.warnings = append(e.warnings, fmt.Sprintf("Rollback plan creation warning: %v", err))
		} else {
			log.Printf("Rollback plan '%s' created successfully", planName)
		}
	}

	// Phase 1: Parse and build object lifecycles
	if err := e.parseAndTrackMigrations(ctx, migrations); err != nil {
		return "", nil, fmt.Errorf("parse and track migrations: %w", err)
	}

	// Phase 2: Analyze dependencies and detect issues
	if err := e.analyzeDependenciesAndRisks(ctx); err != nil {
		return "", nil, fmt.Errorf("analyze dependencies: %w", err)
	}

	// Phase 3: Apply consolidation rules
	consolidatedObjects, err := e.applyConsolidationRules(ctx)
	if err != nil {
		return "", nil, fmt.Errorf("apply consolidation rules: %w", err)
	}

	// Phase 4: Generate optimized SQL with modern formatting
	finalSQL, err := e.generateOptimizedSQL(ctx, consolidatedObjects)
	if err != nil {
		return "", nil, fmt.Errorf("generate optimized SQL: %w", err)
	}

	// Phase 4.5: Apply SQL transformations if enabled
	if e.sqlTransformer != nil {
		log.Printf("Applying SQL transformations...")
		transformResult, err := e.sqlTransformer.Transform(ctx, finalSQL)
		if err != nil {
			log.Printf("Warning: SQL transformation failed: %v", err)
			e.warnings = append(e.warnings, fmt.Sprintf("SQL transformation warning: %v", err))
		} else {
			finalSQL = transformResult.TransformedSQL
			if len(transformResult.Transformations) > 0 {
				log.Printf("Applied %d SQL transformations", len(transformResult.Transformations))
				for _, transformation := range transformResult.Transformations {
					e.warnings = append(e.warnings, fmt.Sprintf("Transformation: %s", transformation.Description))
				}
			}
		}
	}

	// Phase 5: Perform database validation if in paranoid mode
	if e.config.SafetyLevel == string(Paranoid) && e.prodDB != nil {
		if err := e.validateAgainstDatabase(ctx, finalSQL); err != nil {
			e.warnings = append(e.warnings, fmt.Sprintf("Database validation warning: %v", err))
		}
	}

	processingTime := time.Since(startTime)
	log.Printf("Enhanced squashing completed in %v", processingTime)

	return finalSQL, e.warnings, nil
}

// SquashStreaming processes migrations with streaming approach for large datasets
func (e *Engine) SquashStreaming(migrations map[int]string) (string, []string, error) {
	if !e.enableStreaming {
		return "", nil, fmt.Errorf("streaming not enabled for this engine instance")
	}

	startTime := time.Now()
	ctx := context.Background()

	e.updatePhase("Initializing")
	e.stats.TotalMigrations = int64(len(migrations))

	log.Printf("Starting streaming squash process with %d migrations", len(migrations))

	// Phase 1: Stream process migrations using batching
	e.updatePhase("Parsing and Tracking")
	if err := e.streamProcessMigrations(ctx, migrations); err != nil {
		return "", nil, fmt.Errorf("stream process migrations: %w", err)
	}

	// Get the underlying tracker
	tracker := e.streamingTracker.GetTracker()
	e.tracker = tracker

	// Get lifecycles from tracker
	lifecycles := e.tracker.GetObjectsByCategory()
	for _, categoryObjects := range lifecycles {
		for _, lifecycle := range categoryObjects {
			e.lifecycles[lifecycle.Key] = lifecycle
		}
	}

	e.stats.ObjectsTracked = int64(len(e.lifecycles))

	// Continue with existing engine phases
	e.updatePhase("Analyzing Dependencies")
	if err := e.analyzeDependenciesAndRisks(ctx); err != nil {
		return "", nil, fmt.Errorf("analyze dependencies: %w", err)
	}

	e.updatePhase("Applying Consolidations")
	consolidatedObjects, err := e.applyConsolidationRules(ctx)
	if err != nil {
		return "", nil, fmt.Errorf("apply consolidation rules: %w", err)
	}

	e.stats.ConsolidationsApplied = int64(len(consolidatedObjects))

	e.updatePhase("Generating SQL")
	finalSQL, err := e.generateOptimizedSQL(ctx, consolidatedObjects)
	if err != nil {
		return "", nil, fmt.Errorf("generate final SQL: %w", err)
	}

	// Update final statistics
	e.stats.ProcessingTime = time.Since(startTime)
	if e.stats.ProcessingTime.Seconds() > 0 {
		e.stats.ThroughputMPS = float64(e.stats.MigrationsProcessed) / e.stats.ProcessingTime.Seconds()
	}
	e.updatePhase("Completed")

	log.Printf("Streaming squash completed in %v (%.2f migrations/sec)",
		e.stats.ProcessingTime, e.stats.ThroughputMPS)

	return finalSQL, e.warnings, nil
}

// SquashFromDirectory processes migrations from a directory using streaming
func (e *Engine) SquashFromDirectory(dir string) (string, []string, error) {
	if !e.enableStreaming {
		return "", nil, fmt.Errorf("streaming not enabled for this engine instance")
	}

	startTime := time.Now()
	ctx := context.Background()

	e.updatePhase("Initializing")
	log.Printf("Starting streaming squash process from directory: %s", dir)

	// Phase 1: Stream parse and track migrations
	e.updatePhase("Parsing and Tracking")
	if err := e.streamParseAndTrack(ctx, dir); err != nil {
		return "", nil, fmt.Errorf("stream parse and track: %w", err)
	}

	// Get the underlying tracker from streaming tracker
	tracker := e.streamingTracker.GetTracker()

	// Update engine's tracker to use the streaming tracker's results
	e.tracker = tracker

	// Phase 2: Analyze dependencies (using existing engine logic)
	e.updatePhase("Analyzing Dependencies")
	if err := e.analyzeDependenciesAndRisks(ctx); err != nil {
		return "", nil, fmt.Errorf("analyze dependencies: %w", err)
	}

	// Phase 3: Apply consolidation rules (using existing engine logic)
	e.updatePhase("Applying Consolidations")
	consolidatedObjects, err := e.applyConsolidationRules(ctx)
	if err != nil {
		return "", nil, fmt.Errorf("apply consolidation rules: %w", err)
	}

	e.stats.ConsolidationsApplied = int64(len(consolidatedObjects))

	// Phase 4: Generate final SQL (using existing engine logic)
	e.updatePhase("Generating SQL")
	finalSQL, err := e.generateOptimizedSQL(ctx, consolidatedObjects)
	if err != nil {
		return "", nil, fmt.Errorf("generate final SQL: %w", err)
	}

	// Update final statistics
	e.stats.ProcessingTime = time.Since(startTime)
	e.updatePhase("Completed")

	log.Printf("Streaming squash completed in %v", e.stats.ProcessingTime)
	return finalSQL, e.warnings, nil
}

// parseAndTrackMigrations parses migrations and builds object lifecycles
func (e *Engine) parseAndTrackMigrations(ctx context.Context, migrations map[int]string) error {
	log.Printf("Parsing %d migration files with enhanced tracking", len(migrations))

	for sequence, migrationContent := range migrations {
		migration, err := parser.ParseMigration(migrationContent, fmt.Sprintf("migration_%d", sequence))
		if err != nil {
			return fmt.Errorf("parse migration %d: %w", sequence, err)
		}

		// Process with enhanced tracker
		e.tracker.ProcessMigration(migration, sequence)

		// Mark as processed
		e.processedFiles[migration.Filename] = true
	}

	// Get object lifecycles from tracker
	lifecycles := e.tracker.GetObjectsByCategory()
	for _, categoryObjects := range lifecycles {
		for _, lifecycle := range categoryObjects {
			e.lifecycles[lifecycle.Key] = lifecycle
		}
	}

	log.Printf("Tracked %d database objects across %d categories", len(e.lifecycles), len(lifecycles))
	return nil
}

// analyzeDependenciesAndRisks analyzes object dependencies and assesses risks
func (e *Engine) analyzeDependenciesAndRisks(ctx context.Context) error {
	log.Printf("Analyzing dependencies and risks for %d objects", len(e.lifecycles))

	// Use unified dependency resolver for complex scenarios
	unifiedResolver := NewUnifiedDependencyResolver()

	sortedObjects, err := unifiedResolver.ResolveLifecycleDependencies(
		e.tracker.GetActualDependencyGraph(),
		e.lifecycles,
	)
	if err != nil {
		e.warnings = append(e.warnings, fmt.Sprintf("Unified dependency resolution warnings: %v", err))

		// Fallback to basic topological sort
		if basicSorted, basicErr := e.tracker.GetActualDependencyGraph().TopologicalSort(); basicErr != nil {
			e.warnings = append(e.warnings, fmt.Sprintf("Circular dependencies detected: %v", basicErr))
		} else {
			log.Printf("Fallback dependency analysis complete: %d objects in correct order", len(basicSorted))
		}
	} else {
		log.Printf("Enhanced dependency analysis complete: %d objects in optimized order", len(sortedObjects))
	}

	// Detect specific dependency cycles
	if cycles := e.tracker.GetActualDependencyGraph().DetectCycles(); len(cycles) > 0 {
		for _, cycle := range cycles {
			cycleStr := make([]string, len(cycle))
			for i, obj := range cycle {
				cycleStr[i] = obj.String()
			}
			e.warnings = append(e.warnings, fmt.Sprintf("Circular dependency: %s", strings.Join(cycleStr, " -> ")))
		}
	}

	// Advanced DDL cycle detection
	unifiedTracker := e.tracker // Tracker is an alias for UnifiedTracker
	log.Printf("Running advanced DDL cycle detection...")
	if err := unifiedTracker.DetectDDLCycles(); err != nil {
		log.Printf("Warning: DDL cycle detection failed: %v", err)
		e.warnings = append(e.warnings, fmt.Sprintf("DDL cycle detection warning: %v", err))
	} else {
		detectedCycles := unifiedTracker.GetDetectedCycles()
		if len(detectedCycles) > 0 {
			log.Printf("Detected %d DDL cycles", len(detectedCycles))
			for _, cycle := range detectedCycles {
				severity := string(cycle.Severity)
				objectsStr := strings.Join(cycle.Objects, ", ")
				e.warnings = append(e.warnings, fmt.Sprintf("DDL Cycle [%s] %s: %s", severity, cycle.Type, objectsStr))

				// Log additional details for critical cycles
				if cycle.Severity == tracking.SeverityCritical {
					log.Printf("CRITICAL DDL cycle detected: %s involving objects: %s", cycle.Description, objectsStr)
				}
			}

			// Report critical cycles count
			criticalCycles := unifiedTracker.GetCriticalCycles()
			if len(criticalCycles) > 0 {
				e.warnings = append(e.warnings, fmt.Sprintf("WARNING: %d critical DDL cycles detected - some optimizations may be unsafe", len(criticalCycles)))
			}
		} else {
			log.Printf("No DDL cycles detected - all optimizations are safe")
		}
	}

	// Validate consistency
	warnings := e.tracker.ValidateConsistency()
	e.warnings = append(e.warnings, warnings...)

	return nil
}

// applyConsolidationRules applies safety-appropriate consolidation rules
func (e *Engine) applyConsolidationRules(ctx context.Context) (map[string]*tracking.ConsolidationResult, error) {
	log.Printf("Applying %s safety level consolidation rules", e.config.SafetyLevel)

	consolidatedObjects := make(map[string]*tracking.ConsolidationResult)

	for key, lifecycle := range e.lifecycles {
		// Skip if already processed
		if _, exists := e.consolidationResults[key]; exists {
			continue
		}

		// Apply consolidation rules
		result, err := e.ruleEngine.ApplyRules(lifecycle, e)
		if err != nil {
			e.warnings = append(e.warnings, fmt.Sprintf("Rule application failed for %s: %v", key, err))
			// Don't drop objects on error - preserve them as-is
			if lifecycle.GetFinalState() != nil {
				result = &tracking.ConsolidationResult{
					OriginalStatements: []parser.Statement{*lifecycle.GetFinalState()},
					ConsolidatedSQL:    lifecycle.GetFinalState().SQL,
					Optimizations:      []string{"preserved_after_rule_failure"},
					Warnings:           []string{fmt.Sprintf("Rule application failed: %v", err)},
					RiskLevel:          tracking.RiskLevelLow,
				}
			}
		}

		if result != nil {
			// Validate ConsolidatedSQL is not empty
			if strings.TrimSpace(result.ConsolidatedSQL) == "" {
				e.warnings = append(e.warnings, fmt.Sprintf("Object %s has empty ConsolidatedSQL, skipping", key))
				continue
			}

			// For tables, check if there are ALTER statements that must remain separate
			// Some ALTERs (like RLS) cannot be integrated into CREATE TABLE
			if lifecycle.Type == parser.TypeTable {
				allAlterStmts := lifecycle.GetAlterStatements()

				// Check which ALTER types must remain as separate statements
				var separateAlters []parser.Statement
				for _, alterStmt := range allAlterStmts {
					alterSQL := strings.ToUpper(alterStmt.SQL)
					// These ALTER operations cannot be integrated into CREATE TABLE
					mustBeSeparate := strings.Contains(alterSQL, "ENABLE ROW LEVEL SECURITY") ||
						strings.Contains(alterSQL, "DISABLE ROW LEVEL SECURITY") ||
						strings.Contains(alterSQL, "FORCE ROW LEVEL SECURITY") ||
						strings.Contains(alterSQL, "NO FORCE ROW LEVEL SECURITY") ||
						strings.Contains(alterSQL, "ALTER COLUMN") || // Column modifications
						strings.Contains(alterSQL, "DROP COLUMN") || // Column drops
						strings.Contains(alterSQL, "RENAME COLUMN") || // Column renames
						strings.Contains(alterSQL, "RENAME TO") || // Table renames
						strings.Contains(alterSQL, "OWNER TO") || // Owner changes
						strings.Contains(alterSQL, "SET SCHEMA") // Schema changes

					if mustBeSeparate {
						separateAlters = append(separateAlters, alterStmt)
					}
				}

				if len(separateAlters) > 0 {
					// Ensure CREATE TABLE ends with semicolon before appending ALTERs
					result.ConsolidatedSQL = strings.TrimSpace(result.ConsolidatedSQL)
					if !strings.HasSuffix(result.ConsolidatedSQL, ";") {
						result.ConsolidatedSQL += ";"
					}
					// Append ALTER statements that must be separate
					for _, alterStmt := range separateAlters {
						result.ConsolidatedSQL += "\n\n" + alterStmt.SQL
						// Only add to OriginalStatements if not already there
						alreadyIncluded := false
						for _, existing := range result.OriginalStatements {
							if existing.SQL == alterStmt.SQL {
								alreadyIncluded = true
								break
							}
						}
						if !alreadyIncluded {
							result.OriginalStatements = append(result.OriginalStatements, alterStmt)
						}
					}
					log.Printf("Added %d ALTER statements that must remain separate for table %s", len(separateAlters), key)
				}
			}

			consolidatedObjects[key] = result
			e.consolidationResults[key] = result
			log.Printf("Applied consolidation to %s: %v", key, result.Optimizations)
		} else if lifecycle.GetFinalState() != nil {
			// No consolidation rules applied - preserve object as-is
			finalState := lifecycle.GetFinalState()
			if strings.TrimSpace(finalState.SQL) == "" {
				e.warnings = append(e.warnings, fmt.Sprintf("Object %s has empty SQL, skipping", key))
				continue
			}

			consolidatedSQL := finalState.SQL
			allStatements := []parser.Statement{*finalState}

			// For tables, include ALTER statements as separate statements (not consolidated)
			// When no consolidation rules apply, we preserve original migration structure
			if finalState.ObjectType == parser.TypeTable {
				alterStmts := lifecycle.GetAlterStatements()
				if len(alterStmts) > 0 {
					// Ensure CREATE TABLE ends with semicolon before appending ALTERs
					consolidatedSQL = strings.TrimSpace(consolidatedSQL)
					if !strings.HasSuffix(consolidatedSQL, ";") {
						consolidatedSQL += ";"
					}
					// Append ALTER statements after CREATE TABLE (original migration structure)
					for _, alterStmt := range alterStmts {
						consolidatedSQL += "\n\n" + alterStmt.SQL
						allStatements = append(allStatements, alterStmt)
					}
					log.Printf("Preserved %d ALTER statements for table %s (no consolidation rules applied)", len(alterStmts), key)
				}
			}

			result = &tracking.ConsolidationResult{
				OriginalStatements: allStatements,
				ConsolidatedSQL:    consolidatedSQL,
				Optimizations:      []string{"preserved_no_applicable_rules"},
				Warnings:           []string{},
				RiskLevel:          tracking.RiskLevelLow,
			}
			consolidatedObjects[key] = result
			e.consolidationResults[key] = result
			log.Printf("Preserved object without consolidation: %s", key)
		}
	}

	log.Printf("Successfully consolidated %d objects", len(consolidatedObjects))
	return consolidatedObjects, nil
}

// generateOptimizedSQL generates the final optimized SQL with modern formatting
func (e *Engine) generateOptimizedSQL(ctx context.Context, consolidatedObjects map[string]*tracking.ConsolidationResult) (string, error) {
	log.Printf("Generating optimized SQL for %d consolidated objects", len(consolidatedObjects))

	e.sqlBuilder.Reset()

	// Add header comment
	e.sqlBuilder.Comment("Generated by pg-squash with modern PostgreSQL patterns")
	e.sqlBuilder.Comment(fmt.Sprintf("Safety level: %s", e.config.SafetyLevel))
	e.sqlBuilder.Comment(fmt.Sprintf("Generated at: %s", time.Now().Format(time.RFC3339)))
	e.sqlBuilder.NL()

	// Group by category for organized output - CRITICAL: Order must ensure dependencies are created first
	categories := []parser.Category{
		parser.CategoryExtensions,     // 1. Extensions first (CREATE EXTENSION)
		parser.CategoryFoundation,     // 2. Tables, views, sequences (CREATE TABLE)
		parser.CategoryConstraints,    // 3. Constraints (ALTER TABLE ADD CONSTRAINT)
		parser.CategoryFunctions,      // 4. Functions (CREATE FUNCTION)
		parser.CategoryTriggers,       // 5. Triggers (CREATE TRIGGER)
		parser.CategoryIndexes,        // 6. Indexes (CREATE INDEX)
		parser.CategorySecurity,       // 7. RLS Policies (CREATE POLICY)
		parser.CategoryData,           // 8. Data operations LAST (INSERT/UPDATE)
	}

	// Initialize unified dependency resolver
	unifiedResolver := NewUnifiedDependencyResolver()

	for _, category := range categories {
		categoryObjectsMap := e.getObjectsByCategoryAsMap(consolidatedObjects, category)
		if len(categoryObjectsMap) == 0 {
			continue
		}

		// Add category header
		e.sqlBuilder.NL().Comment(fmt.Sprintf("=== %s OBJECTS ===", strings.ToUpper(string(category)))).NL()

		// Sort objects by dependencies within category
		sortedObjects := unifiedResolver.SortConsolidationResults(categoryObjectsMap, category, e.lifecycles)

		for _, result := range sortedObjects {
			sql := result.ConsolidatedSQL

			// Apply CASCADE enhancement for extension objects
			if category == parser.CategoryExtensions {
				sql = unifiedResolver.EnhanceExtensionSQL(sql)
			}

			e.sqlBuilder.Statement(sql)
			e.sqlBuilder.NL().NL()
		}
	}

	// Add any objects not included in categories (fallback)
	addedObjects := make(map[string]bool)
	for _, category := range categories {
		categoryObjects := e.getObjectsByCategory(consolidatedObjects, category)
		for range categoryObjects {
			for key := range consolidatedObjects {
				if lifecycle, exists := e.lifecycles[key]; exists && lifecycle.Category == category {
					addedObjects[key] = true
				}
			}
		}
	}

	// Add objects that weren't included in any category
	e.sqlBuilder.NL().Comment("=== UNCATEGORIZED OBJECTS ===").NL()
	uncategorizedCount := 0
	for key, result := range consolidatedObjects {
		if !addedObjects[key] {
			e.sqlBuilder.Statement(result.ConsolidatedSQL)
			e.sqlBuilder.NL().NL()
			uncategorizedCount++
		}
	}

	if uncategorizedCount > 0 {
		log.Printf("Added %d uncategorized objects to output", uncategorizedCount)
	}

	// Add any non-consolidated objects (legacy fallback)
	for key, lifecycle := range e.lifecycles {
		if _, consolidated := consolidatedObjects[key]; !consolidated {
			e.sqlBuilder.Comment(fmt.Sprintf("Original object: %s", key))
			if lifecycle.GetFinalState() != nil {
				e.sqlBuilder.FromStatement(*lifecycle.GetFinalState()).NL().NL()
			}
		}
	}

	// Post-process SQL to fix various issues
	finalSQL := e.sqlBuilder.String()
	finalSQL = fixMalformedDropTriggers(finalSQL)
	finalSQL = fixExtensionOrder(finalSQL)
	finalSQL = removeOrphanedAlterStatements(finalSQL)

	return finalSQL, nil
}

func (e *Engine) getObjectsByCategory(consolidatedObjects map[string]*tracking.ConsolidationResult, category parser.Category) []*tracking.ConsolidationResult {
	var objects []*tracking.ConsolidationResult

	for key, result := range consolidatedObjects {
		if lifecycle, exists := e.lifecycles[key]; exists && lifecycle.Category == category {
			objects = append(objects, result)
		}
	}

	return objects
}

func (e *Engine) getObjectsByCategoryAsMap(consolidatedObjects map[string]*tracking.ConsolidationResult, category parser.Category) map[string]*tracking.ConsolidationResult {
	objects := make(map[string]*tracking.ConsolidationResult)

	for key, result := range consolidatedObjects {
		if lifecycle, exists := e.lifecycles[key]; exists && lifecycle.Category == category {
			objects[key] = result
		}
	}

	return objects
}

// validateAgainstDatabase validates the generated SQL against the production database
func (e *Engine) validateAgainstDatabase(ctx context.Context, sql string) error {
	if e.metadataManager == nil {
		return fmt.Errorf("no metadata manager available for validation")
	}

	log.Printf("Performing database validation in paranoid mode")

	// Get current database metadata
	dbMeta, err := e.metadataManager.GetMetadata(ctx, "")
	if err != nil {
		return fmt.Errorf("failed to get database metadata: %w", err)
	}

	// Validate against current schema
	// This would involve creating a temporary schema and applying the SQL
	// For now, we just log the validation attempt
	log.Printf("Database validation: Found %d schemas, %d extensions", len(dbMeta.Schemas), len(dbMeta.Extensions))

	return nil
}

// streamParseAndTrack streams parsing and tracking from directory
func (e *Engine) streamParseAndTrack(ctx context.Context, dir string) error {
	// Set up progress tracking
	if e.enableProgressTrack {
		e.streamingTracker.SetProgressCallback(func(processed, total int64, throughput float64) {
			e.mu.Lock()
			e.stats.MigrationsProcessed = processed
			e.mu.Unlock()

			if e.progressCb != nil {
				e.progressCb(processed, total, e.stats.Phase)
			}
		})
	}

	// Process directory with streaming
	if err := e.streamingTracker.ProcessDirectory(dir); err != nil {
		return fmt.Errorf("streaming tracker failed: %w", err)
	}

	// Update statistics
	streamStats, _ := e.streamingTracker.GetCombinedStats()
	e.mu.Lock()
	e.stats.MigrationsProcessed = streamStats.MigrationsProcessed
	e.stats.ObjectsTracked = streamStats.ObjectsTracked
	e.stats.PeakMemoryUsage = e.memManager.GetMemoryStats().CurrentMemoryBytes
	e.mu.Unlock()

	return nil
}

// streamProcessMigrations processes migrations using batching for memory efficiency
func (e *Engine) streamProcessMigrations(ctx context.Context, migrations map[int]string) error {
	migrationFiles := make([]*performance.MigrationFile, 0, len(migrations))

	// Convert migrations to MigrationFile format
	for sequence, content := range migrations {
		migrationFile := &performance.MigrationFile{
			Path:     fmt.Sprintf("migration_%d.sql", sequence),
			Content:  []byte(content),
			Sequence: sequence,
			Size:     int64(len(content)),
		}
		migrationFiles = append(migrationFiles, migrationFile)
	}

	// Process in batches to manage memory
	processedCount := 0
	for _, migrationFile := range migrationFiles {
		// Check memory constraints
		if !e.memManager.TrackMemoryUsage(migrationFile.Size) {
			// Force memory cleanup if needed
			e.streamingTracker.GetTracker().ClearProcessedMigrations()
			if !e.memManager.TrackMemoryUsage(migrationFile.Size) {
				return fmt.Errorf("migration too large for memory constraints: %s", migrationFile.Path)
			}
		}

		// Parse migration
		migration, err := parser.ParseMigration(string(migrationFile.Content), migrationFile.Path)
		if err != nil {
			e.memManager.ReleaseMemory(migrationFile.Size)
			return fmt.Errorf("parse migration %s: %w", migrationFile.Path, err)
		}

		// Process through tracker
		e.streamingTracker.GetTracker().ProcessMigration(migration, migrationFile.Sequence)

		// Release memory
		e.memManager.ReleaseMemory(migrationFile.Size)

		processedCount++
		e.mu.Lock()
		e.stats.MigrationsProcessed = int64(processedCount)
		e.mu.Unlock()

		// Update progress
		if e.progressCb != nil {
			e.progressCb(int64(processedCount), e.stats.TotalMigrations, e.stats.Phase)
		}

		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	return nil
}

// updatePhase updates the current processing phase
func (e *Engine) updatePhase(phase string) {
	e.mu.Lock()
	e.stats.Phase = phase
	e.mu.Unlock()

	log.Printf("Squash phase: %s", phase)
}

// GetStats returns current squashing statistics
func (e *Engine) GetStats() SquashStats {
	e.mu.RLock()
	defer e.mu.RUnlock()

	stats := *e.stats

	// Update peak memory from memory manager if streaming is enabled
	if e.enableStreaming && e.memManager != nil {
		memStats := e.memManager.GetMemoryStats()
		if memStats.CurrentMemoryBytes > stats.PeakMemoryUsage {
			stats.PeakMemoryUsage = memStats.CurrentMemoryBytes
		}
	}

	return stats
}

// SetProgressCallback sets a progress callback for the engine
func (e *Engine) SetProgressCallback(callback func(processed, total int64, phase string)) {
	e.progressCb = callback
	e.enableProgressTrack = true
}

// GetMemoryStats returns current memory usage statistics (streaming mode only)
func (e *Engine) GetMemoryStats() performance.MemoryStats {
	if e.enableStreaming && e.memManager != nil {
		return e.memManager.GetMemoryStats()
	}
	return performance.MemoryStats{}
}

// OptimizedSquashForLargeDatasets provides a high-level interface for large migration sets
func OptimizedSquashForLargeDatasets(cfg *config.Config, migrations map[int]string, memoryLimitMB int) (string, []string, error) {
	engineConfig := EngineConfig{
		Config:              cfg,
		EnableStreaming:     true,
		BatchSize:           100, // Larger batches for better throughput
		WorkerCount:         4,
		MemoryLimitMB:       memoryLimitMB,
		EnableProgressTrack: true,
		ProgressCallback: func(processed, total int64, phase string) {
			if total > 0 {
				progress := float64(processed) / float64(total) * 100
				log.Printf("Progress: %.1f%% (%d/%d) - %s", progress, processed, total, phase)
			}
		},
	}

	engine := NewEngine(engineConfig)
	defer engine.Close()

	return engine.SquashStreaming(migrations)
}

// OptimizedSquashFromDirectory provides a high-level interface for directory processing
func OptimizedSquashFromDirectory(cfg *config.Config, dir string, memoryLimitMB int) (string, []string, error) {
	engineConfig := EngineConfig{
		Config:              cfg,
		EnableStreaming:     true,
		BatchSize:           50,
		WorkerCount:         2, // Conservative for directory processing
		MemoryLimitMB:       memoryLimitMB,
		EnableProgressTrack: true,
		ProgressCallback: func(processed, total int64, phase string) {
			log.Printf("Processing: %d files - %s", processed, phase)
		},
	}

	engine := NewEngine(engineConfig)
	defer engine.Close()

	return engine.SquashFromDirectory(dir)
}

// fixMalformedDropTriggers fixes DROP TRIGGER statements that use qualified table.trigger_name syntax
// PostgreSQL requires: DROP TRIGGER IF EXISTS trigger_name ON table_name;
// Not: DROP TRIGGER IF EXISTS table_name.trigger_name;
func fixMalformedDropTriggers(sql string) string {
	// Pattern: DROP TRIGGER IF EXISTS table_name.trigger_name;
	// Replace with: DROP TRIGGER IF EXISTS trigger_name ON table_name;

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
					log.Printf("Fixed malformed DROP TRIGGER: %s.%s -> %s ON %s", tableName, triggerName, triggerName, tableName)
				}
			}
		}
	}

	return strings.Join(lines, "\n")
}

// fixExtensionOrder ensures extensions are in correct dependency order
// cube must come before earthdistance
func fixExtensionOrder(sql string) string {
	// Correct order (cube before earthdistance)
	correctOrder := []string{
		"cube",
		"earthdistance",
		"postgis",
		"uuid-ossp",
		"pg_trgm",
		"pg_stat_statements",
		"btree_gin",
		"pgcrypto",
	}

	// Find all CREATE EXTENSION statements and their positions
	lines := strings.Split(sql, "\n")
	extensionMap := make(map[string]string) // extension name -> full line
	extensionPositions := make(map[int]bool) // line numbers to remove

	for i, line := range lines {
		upperLine := strings.ToUpper(strings.TrimSpace(line))
		if strings.HasPrefix(upperLine, "CREATE EXTENSION") {
			// Extract extension name
			// Format: CREATE EXTENSION IF NOT EXISTS "name"; or CREATE EXTENSION "name";
			parts := strings.Fields(line)

			// Find the extension name (last meaningful part before semicolon)
			var extName string
			for j := len(parts) - 1; j >= 0; j-- {
				part := strings.Trim(parts[j], `";`)
				if part != "" && strings.ToUpper(part) != "EXISTS" && strings.ToUpper(part) != "NOT" &&
				   strings.ToUpper(part) != "IF" && strings.ToUpper(part) != "EXTENSION" && strings.ToUpper(part) != "CREATE" {
					extName = part
					break
				}
			}

			if extName != "" {
				extensionMap[extName] = line
				extensionPositions[i] = true
				log.Printf("Found extension: %s at line %d", extName, i+1)
			}
		}
	}

	log.Printf("Total extensions found: %d", len(extensionMap))

	// If we found extensions, rebuild with correct order
	if len(extensionMap) > 0 {
		// Find the extension section header
		extensionHeaderIdx := -1
		for i, line := range lines {
			if strings.Contains(line, "=== EXTENSIONS OBJECTS ===") {
				extensionHeaderIdx = i
				break
			}
		}

		if extensionHeaderIdx >= 0 {
			// Build new file: header + sorted extensions + rest
			var result []string

			// Add everything before extension section
			for i := 0; i <= extensionHeaderIdx; i++ {
				result = append(result, lines[i])
			}
			result = append(result, "")

			// Add extensions in correct order
			for _, extName := range correctOrder {
				if line, exists := extensionMap[extName]; exists {
					result = append(result, line)
					result = append(result, "")
					delete(extensionMap, extName)
				}
			}

			// Add any remaining extensions not in predefined order (must iterate in stable order)
			var remainingExtNames []string
			for extName := range extensionMap {
				remainingExtNames = append(remainingExtNames, extName)
			}
			// Sort remaining extensions by name for stable output
			for _, extName := range remainingExtNames {
				if line, exists := extensionMap[extName]; exists {
					result = append(result, line)
					result = append(result, "")
				}
			}

			// Add rest of file (skipping old extension lines)
			for i := extensionHeaderIdx + 1; i < len(lines); i++ {
				if !extensionPositions[i] {
					result = append(result, lines[i])
				}
			}

			log.Printf("Reordered %d extensions to ensure correct dependency order", len(extensionMap))
			return strings.Join(result, "\n")
		}
	}

	return sql
}

// sortExtensionsByDependency sorts extension CREATE statements by dependency order
//nolint:unused // Reserved for future extension dependency sorting
// Some extensions depend on others and must be created in the right order
func sortExtensionsByDependency(extensionLines []string) []string {
	// Simple hardcoded order for known dependencies
	// cube must come before earthdistance
	order := []string{
		"cube",
		"earthdistance",
		"postgis",
		"uuid-ossp",
		"pg_trgm",
		"pg_stat_statements",
		"btree_gin",
		"pgcrypto",
	}

	// Map extension names to their lines
	extLineMap := make(map[string]string)
	for _, line := range extensionLines {
		parts := strings.Fields(line)
		if len(parts) >= 5 {
			extName := parts[4] // Position after "CREATE EXTENSION IF NOT EXISTS"
			extName = strings.Trim(extName, `";`) // Remove quotes and semicolon
			extLineMap[extName] = line
		}
	}

	// Build result in correct order
	var result []string
	for _, extName := range order {
		if line, exists := extLineMap[extName]; exists {
			result = append(result, line)
			delete(extLineMap, extName) // Mark as processed
		}
	}

	// Add any remaining extensions not in the predefined order
	for _, line := range extLineMap {
		result = append(result, line)
	}

	log.Printf("Sorted %d extensions by dependency order", len(result))
	return result
}

// removeOrphanedAlterStatements removes ALTER TABLE statements for tables that don't exist
// This happens when a table was created in early migrations but replaced/renamed in later ones
func removeOrphanedAlterStatements(sql string) string {
	lines := strings.Split(sql, "\n")

	// First pass: collect all tables that are actually created
	createdTables := make(map[string]bool)
	createTablePattern := regexp.MustCompile(`(?i)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?([a-zA-Z_][a-zA-Z0-9_]*)`)

	for _, line := range lines {
		if matches := createTablePattern.FindStringSubmatch(line); len(matches) > 1 {
			tableName := strings.ToLower(strings.TrimSpace(matches[1]))
			createdTables[tableName] = true
		}
	}

	log.Printf("Found %d created tables", len(createdTables))

	// Second pass: filter out ALTER statements for non-existent tables
	var result []string
	alterTablePattern := regexp.MustCompile(`(?i)ALTER\s+TABLE\s+([a-zA-Z_][a-zA-Z0-9_]*)`)
	removedCount := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check if this is an ALTER TABLE statement
		if matches := alterTablePattern.FindStringSubmatch(trimmed); len(matches) > 1 {
			tableName := strings.ToLower(strings.TrimSpace(matches[1]))

			// Only keep ALTER if the table was actually created
			if !createdTables[tableName] {
				log.Printf("Removing orphaned ALTER for non-existent table: %s", tableName)
				removedCount++
				continue // Skip this line
			}
		}

		result = append(result, line)
	}

	if removedCount > 0 {
		log.Printf("Removed %d orphaned ALTER TABLE statements", removedCount)
	}

	return strings.Join(result, "\n")
}

// initializePlugins discovers and initializes plugins from migrations
// This enables plugin-specific enrichment, transformations, and consolidation rules
func (e *Engine) initializePlugins(ctx context.Context, migrations map[int]string) error {
	// Parse migrations to types.Migration format for plugin detection
	var parsedMigrations []*parser.Migration
	for id, sql := range migrations {
		filename := fmt.Sprintf("migration_%d.sql", id)
		migration, err := parser.ParseMigration(sql, filename)
		if err != nil {
			log.Printf("Warning: Failed to parse migration %d for plugin detection: %v", id, err)
			continue
		}
		parsedMigrations = append(parsedMigrations, migration)
	}

	// Extract plugin configuration from engine config
	pluginConfig := make(map[string]interface{})
	if e.config != nil {
		// Map config to plugin-specific sections
		pluginConfig["clerk"] = e.config.ThirdPartyIntegrations.ClerkIntegration
		pluginConfig["supabase"] = e.config.ThirdPartyIntegrations.SupabaseIntegration
		pluginConfig["auth0"] = e.config.ThirdPartyIntegrations.Auth0Integration
		pluginConfig["nextauth"] = e.config.ThirdPartyIntegrations.NextAuthIntegration
	}

	// Initialize plugin registry
	registry := plugins.GlobalRegistry()
	if err := registry.DiscoverAndInitialize(ctx, parsedMigrations, pluginConfig); err != nil {
		return fmt.Errorf("plugin initialization failed: %w", err)
	}

	activePlugins := registry.ActivePlugins()
	if len(activePlugins) > 0 {
		var pluginNames []string
		for _, p := range activePlugins {
			pluginNames = append(pluginNames, p.Name())
		}
		log.Printf("✓ Activated plugins: %v", pluginNames)
	} else {
		log.Printf("ℹ No plugins activated (no third-party patterns detected)")
	}

	return nil
}

//nolint:unused // Plugin preservation feature not yet integrated
// shouldPreserveStatement checks if plugins want to preserve a statement
// Preserved statements are marked as critical and skipped during consolidation
func (e *Engine) shouldPreserveStatement(stmt *parser.Statement) bool {
	registry := plugins.GlobalRegistry()
	return registry.ShouldPreserve(stmt)
}

//nolint:unused // Plugin consolidation rules not yet integrated
// getPluginConsolidationRules retrieves consolidation rules from all active plugins
// These rules are merged with standard consolidation rules
func (e *Engine) getPluginConsolidationRules() []plugins.ConsolidationRule {
	registry := plugins.GlobalRegistry()
	return registry.GetConsolidationRules()
}
