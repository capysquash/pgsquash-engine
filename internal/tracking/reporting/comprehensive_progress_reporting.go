package reporting

import (
	"fmt"
	"sync"
	"time"

	"github.com/fatih/color"
)

// ProgressStage represents different stages of the squashing process
type ProgressStage string

const (
	StageInitialization    ProgressStage = "INITIALIZATION"
	StageParsing           ProgressStage = "PARSING"
	StageAnalysis          ProgressStage = "ANALYSIS"
	StageLifecycleTracking ProgressStage = "LIFECYCLE_TRACKING"
	StageConsolidation     ProgressStage = "CONSOLIDATION"
	StageOptimization      ProgressStage = "OPTIMIZATION"
	StageValidation        ProgressStage = "VALIDATION"
	StageOutput            ProgressStage = "OUTPUT"
	StageComplete          ProgressStage = "COMPLETE"
)

// ProgressMetrics tracks detailed metrics about the squashing process
type ProgressMetrics struct {
	StartTime       time.Time     `json:"start_time"`
	CurrentStage    ProgressStage `json:"current_stage"`
	OverallProgress float64       `json:"overall_progress"`
	StageProgress   float64       `json:"stage_progress"`

	// File processing metrics
	TotalFiles     int `json:"total_files"`
	ProcessedFiles int `json:"processed_files"`
	SkippedFiles   int `json:"skipped_files"`
	FailedFiles    int `json:"failed_files"`

	// Statement metrics
	TotalStatements        int `json:"total_statements"`
	ProcessedStatements    int `json:"processed_statements"`
	ConsolidatedStatements int `json:"consolidated_statements"`
	OptimizedStatements    int `json:"optimized_statements"`

	// Object tracking metrics
	TrackedObjects      int `json:"tracked_objects"`
	AnalyzedObjects     int `json:"analyzed_objects"`
	ConsolidatedObjects int `json:"consolidated_objects"`

	// Performance metrics
	MemoryUsageMB          float64       `json:"memory_usage_mb"`
	ThroughputStmtsPerSec  float64       `json:"throughput_stmts_per_sec"`
	EstimatedTimeRemaining time.Duration `json:"estimated_time_remaining"`

	// Stage durations
	StageDurations map[ProgressStage]time.Duration `json:"stage_durations"`

	// Error metrics
	ErrorCount      int `json:"error_count"`
	RecoveredErrors int `json:"recovered_errors"`

	// Optimizations applied
	OptimizationsApplied []string `json:"optimizations_applied"`
	LinesSaved           int      `json:"lines_saved"`
	SizeSavedBytes       int64    `json:"size_saved_bytes"`
}

// ProgressEvent represents a progress update event
type ProgressEvent struct {
	Timestamp time.Time              `json:"timestamp"`
	Stage     ProgressStage          `json:"stage"`
	Progress  float64                `json:"progress"`
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Metrics   *ProgressMetrics       `json:"metrics,omitempty"`
}

// ProgressCallback is called when progress updates occur
type ProgressCallback func(*ProgressEvent)

// ComprehensiveProgressReporter provides detailed progress reporting
type ComprehensiveProgressReporter struct {
	metrics   *ProgressMetrics
	callbacks []ProgressCallback
	verbose   bool
	colored   bool
	mu        sync.RWMutex

	// Stage management
	currentStage   ProgressStage
	stageStartTime time.Time

	// Colors for output
	successColor *color.Color
	infoColor    *color.Color
	warningColor *color.Color
	errorColor   *color.Color

	// ETA calculation
	stageWeights  map[ProgressStage]float64
	completedWork float64
}

// ReporterConfig configures the progress reporter
type ReporterConfig struct {
	Verbose        bool
	Colored        bool
	EnableETACalc  bool
	Callbacks      []ProgressCallback
	UpdateInterval time.Duration
}

// NewComprehensiveProgressReporter creates a new progress reporter
func NewComprehensiveProgressReporter(config ReporterConfig) *ComprehensiveProgressReporter {
	reporter := &ComprehensiveProgressReporter{
		metrics: &ProgressMetrics{
			StartTime:            time.Now(),
			CurrentStage:         StageInitialization,
			StageDurations:       make(map[ProgressStage]time.Duration),
			OptimizationsApplied: make([]string, 0),
		},
		callbacks:      config.Callbacks,
		verbose:        config.Verbose,
		colored:        config.Colored,
		currentStage:   StageInitialization,
		stageStartTime: time.Now(),

		// Define stage weights for ETA calculation
		stageWeights: map[ProgressStage]float64{
			StageInitialization:    0.05, // 5%
			StageParsing:           0.15, // 15%
			StageAnalysis:          0.10, // 10%
			StageLifecycleTracking: 0.20, // 20%
			StageConsolidation:     0.25, // 25%
			StageOptimization:      0.15, // 15%
			StageValidation:        0.05, // 5%
			StageOutput:            0.05, // 5%
		},
	}

	// Initialize colors if enabled
	if config.Colored {
		reporter.successColor = color.New(color.FgGreen, color.Bold)
		reporter.infoColor = color.New(color.FgCyan)
		reporter.warningColor = color.New(color.FgYellow)
		reporter.errorColor = color.New(color.FgRed, color.Bold)
	}

	return reporter
}

// StartStage begins a new processing stage
func (r *ComprehensiveProgressReporter) StartStage(stage ProgressStage, message string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Complete previous stage
	if r.currentStage != stage {
		r.metrics.StageDurations[r.currentStage] = time.Since(r.stageStartTime)
		r.completedWork += r.stageWeights[r.currentStage]
	}

	// Start new stage
	r.currentStage = stage
	r.stageStartTime = time.Now()
	r.metrics.CurrentStage = stage
	r.metrics.StageProgress = 0.0

	r.updateOverallProgress()

	event := &ProgressEvent{
		Timestamp: time.Now(),
		Stage:     stage,
		Progress:  0.0,
		Message:   message,
		Metrics:   r.copyMetrics(),
	}

	r.logMessage(fmt.Sprintf("🔄 Starting %s: %s", stage, message))
	r.notifyCallbacks(event)
}

// UpdateStageProgress updates progress within the current stage
func (r *ComprehensiveProgressReporter) UpdateStageProgress(progress float64, message string, details map[string]interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.metrics.StageProgress = progress
	r.updateOverallProgress()
	r.calculateETA()

	event := &ProgressEvent{
		Timestamp: time.Now(),
		Stage:     r.currentStage,
		Progress:  progress,
		Message:   message,
		Details:   details,
		Metrics:   r.copyMetrics(),
	}

	if r.verbose || progress == 1.0 {
		r.logProgress(progress, message)
	}

	r.notifyCallbacks(event)
}

// UpdateFileMetrics updates file processing metrics
func (r *ComprehensiveProgressReporter) UpdateFileMetrics(total, processed, skipped, failed int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.metrics.TotalFiles = total
	r.metrics.ProcessedFiles = processed
	r.metrics.SkippedFiles = skipped
	r.metrics.FailedFiles = failed
}

// UpdateStatementMetrics updates statement processing metrics
func (r *ComprehensiveProgressReporter) UpdateStatementMetrics(total, processed, consolidated, optimized int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.metrics.TotalStatements = total
	r.metrics.ProcessedStatements = processed
	r.metrics.ConsolidatedStatements = consolidated
	r.metrics.OptimizedStatements = optimized

	// Calculate throughput
	elapsed := time.Since(r.metrics.StartTime).Seconds()
	if elapsed > 0 {
		r.metrics.ThroughputStmtsPerSec = float64(processed) / elapsed
	}
}

// UpdateObjectMetrics updates object tracking metrics
func (r *ComprehensiveProgressReporter) UpdateObjectMetrics(tracked, analyzed, consolidated int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.metrics.TrackedObjects = tracked
	r.metrics.AnalyzedObjects = analyzed
	r.metrics.ConsolidatedObjects = consolidated
}

// ReportError reports an error that occurred during processing
func (r *ComprehensiveProgressReporter) ReportError(err error, context map[string]interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.metrics.ErrorCount++

	details := map[string]interface{}{
		"error":   err.Error(),
		"context": context,
	}

	event := &ProgressEvent{
		Timestamp: time.Now(),
		Stage:     r.currentStage,
		Progress:  r.metrics.StageProgress,
		Message:   fmt.Sprintf("Error: %s", err.Error()),
		Details:   details,
		Metrics:   r.copyMetrics(),
	}

	r.logError(err.Error())
	r.notifyCallbacks(event)
}

// ReportRecoveredError reports an error that was successfully recovered
func (r *ComprehensiveProgressReporter) ReportRecoveredError(err error, recovery string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.metrics.RecoveredErrors++

	details := map[string]interface{}{
		"error":    err.Error(),
		"recovery": recovery,
	}

	event := &ProgressEvent{
		Timestamp: time.Now(),
		Stage:     r.currentStage,
		Progress:  r.metrics.StageProgress,
		Message:   fmt.Sprintf("Recovered from error: %s", err.Error()),
		Details:   details,
		Metrics:   r.copyMetrics(),
	}

	r.logWarning(fmt.Sprintf("Recovered from error: %s (Recovery: %s)", err.Error(), recovery))
	r.notifyCallbacks(event)
}

// ReportOptimization reports an optimization that was applied
func (r *ComprehensiveProgressReporter) ReportOptimization(optimization string, linesSaved int, bytesSaved int64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.metrics.OptimizationsApplied = append(r.metrics.OptimizationsApplied, optimization)
	r.metrics.LinesSaved += linesSaved
	r.metrics.SizeSavedBytes += bytesSaved

	details := map[string]interface{}{
		"optimization": optimization,
		"lines_saved":  linesSaved,
		"bytes_saved":  bytesSaved,
	}

	event := &ProgressEvent{
		Timestamp: time.Now(),
		Stage:     r.currentStage,
		Progress:  r.metrics.StageProgress,
		Message:   fmt.Sprintf("Optimization applied: %s", optimization),
		Details:   details,
		Metrics:   r.copyMetrics(),
	}

	if r.verbose {
		r.logSuccess(fmt.Sprintf("✨ %s (saved %d lines, %d bytes)", optimization, linesSaved, bytesSaved))
	}

	r.notifyCallbacks(event)
}

// Complete marks the entire process as complete
func (r *ComprehensiveProgressReporter) Complete() {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Complete current stage
	r.metrics.StageDurations[r.currentStage] = time.Since(r.stageStartTime)

	// Set final metrics
	r.currentStage = StageComplete
	r.metrics.CurrentStage = StageComplete
	r.metrics.OverallProgress = 1.0
	r.metrics.StageProgress = 1.0
	r.completedWork = 1.0

	totalDuration := time.Since(r.metrics.StartTime)

	event := &ProgressEvent{
		Timestamp: time.Now(),
		Stage:     StageComplete,
		Progress:  1.0,
		Message:   "Squashing process completed successfully",
		Details: map[string]interface{}{
			"total_duration": totalDuration.String(),
			"final_metrics":  r.metrics,
		},
		Metrics: r.copyMetrics(),
	}

	r.printFinalSummary(totalDuration)
	r.notifyCallbacks(event)
}

// GetMetrics returns a copy of current metrics
func (r *ComprehensiveProgressReporter) GetMetrics() *ProgressMetrics {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.copyMetrics()
}

// Helper methods

func (r *ComprehensiveProgressReporter) updateOverallProgress() {
	stageWeight := r.stageWeights[r.currentStage]
	stageContribution := stageWeight * r.metrics.StageProgress
	r.metrics.OverallProgress = r.completedWork + stageContribution
}

func (r *ComprehensiveProgressReporter) calculateETA() {
	if r.metrics.OverallProgress <= 0 {
		return
	}

	elapsed := time.Since(r.metrics.StartTime)
	remaining := elapsed * time.Duration((1.0-r.metrics.OverallProgress)/r.metrics.OverallProgress)
	r.metrics.EstimatedTimeRemaining = remaining
}

func (r *ComprehensiveProgressReporter) copyMetrics() *ProgressMetrics {
	// Create a deep copy of metrics
	copied := &ProgressMetrics{
		StartTime:              r.metrics.StartTime,
		CurrentStage:           r.metrics.CurrentStage,
		OverallProgress:        r.metrics.OverallProgress,
		StageProgress:          r.metrics.StageProgress,
		TotalFiles:             r.metrics.TotalFiles,
		ProcessedFiles:         r.metrics.ProcessedFiles,
		SkippedFiles:           r.metrics.SkippedFiles,
		FailedFiles:            r.metrics.FailedFiles,
		TotalStatements:        r.metrics.TotalStatements,
		ProcessedStatements:    r.metrics.ProcessedStatements,
		ConsolidatedStatements: r.metrics.ConsolidatedStatements,
		OptimizedStatements:    r.metrics.OptimizedStatements,
		TrackedObjects:         r.metrics.TrackedObjects,
		AnalyzedObjects:        r.metrics.AnalyzedObjects,
		ConsolidatedObjects:    r.metrics.ConsolidatedObjects,
		MemoryUsageMB:          r.metrics.MemoryUsageMB,
		ThroughputStmtsPerSec:  r.metrics.ThroughputStmtsPerSec,
		EstimatedTimeRemaining: r.metrics.EstimatedTimeRemaining,
		ErrorCount:             r.metrics.ErrorCount,
		RecoveredErrors:        r.metrics.RecoveredErrors,
		LinesSaved:             r.metrics.LinesSaved,
		SizeSavedBytes:         r.metrics.SizeSavedBytes,
		StageDurations:         make(map[ProgressStage]time.Duration),
		OptimizationsApplied:   make([]string, len(r.metrics.OptimizationsApplied)),
	}

	// Copy maps and slices
	for stage, duration := range r.metrics.StageDurations {
		copied.StageDurations[stage] = duration
	}
	copy(copied.OptimizationsApplied, r.metrics.OptimizationsApplied)

	return copied
}

func (r *ComprehensiveProgressReporter) notifyCallbacks(event *ProgressEvent) {
	for _, callback := range r.callbacks {
		if callback != nil {
			go callback(event) // Run callbacks asynchronously
		}
	}
}

func (r *ComprehensiveProgressReporter) logMessage(message string) {
	if r.colored && r.infoColor != nil {
		_, _ = r.infoColor.Println(message)
	} else {
		fmt.Println(message)
	}
}

func (r *ComprehensiveProgressReporter) logProgress(progress float64, message string) {
	progressBar := r.generateProgressBar(progress)
	fullMessage := fmt.Sprintf("%s %.1f%% - %s", progressBar, progress*100, message)

	if r.colored && r.infoColor != nil {
		_, _ = r.infoColor.Println(fullMessage)
	} else {
		fmt.Println(fullMessage)
	}
}

func (r *ComprehensiveProgressReporter) logSuccess(message string) {
	if r.colored && r.successColor != nil {
		_, _ = r.successColor.Println(message)
	} else {
		fmt.Println(message)
	}
}

func (r *ComprehensiveProgressReporter) logWarning(message string) {
	if r.colored && r.warningColor != nil {
		_, _ = r.warningColor.Println(message)
	} else {
		fmt.Println(message)
	}
}

func (r *ComprehensiveProgressReporter) logError(message string) {
	if r.colored && r.errorColor != nil {
		_, _ = r.errorColor.Println("☒ " + message)
	} else {
		fmt.Println("☒ " + message)
	}
}

func (r *ComprehensiveProgressReporter) generateProgressBar(progress float64) string {
	const barWidth = 30
	filled := int(progress * barWidth)
	bar := "["

	for i := 0; i < barWidth; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}

	bar += "]"
	return bar
}

func (r *ComprehensiveProgressReporter) printFinalSummary(totalDuration time.Duration) {
	r.logSuccess("\n🎉 Migration squashing completed successfully!")
	r.logSuccess("📊 Final Statistics:")
	r.logSuccess(fmt.Sprintf("   ► Total duration: %s", totalDuration))
	r.logSuccess(fmt.Sprintf("   ► Files processed: %d (skipped: %d, failed: %d)",
		r.metrics.ProcessedFiles, r.metrics.SkippedFiles, r.metrics.FailedFiles))
	r.logSuccess(fmt.Sprintf("   ► Statements: %d → %d (%.1f%% reduction)",
		r.metrics.TotalStatements, r.metrics.ConsolidatedStatements,
		(1.0-float64(r.metrics.ConsolidatedStatements)/float64(r.metrics.TotalStatements))*100))
	r.logSuccess(fmt.Sprintf("   ► Objects consolidated: %d", r.metrics.ConsolidatedObjects))
	r.logSuccess(fmt.Sprintf("   ► Lines saved: %d", r.metrics.LinesSaved))
	r.logSuccess(fmt.Sprintf("   ► Size saved: %.2f KB", float64(r.metrics.SizeSavedBytes)/1024))
	r.logSuccess(fmt.Sprintf("   ► Throughput: %.1f statements/sec", r.metrics.ThroughputStmtsPerSec))

	if r.metrics.ErrorCount > 0 {
		r.logWarning(fmt.Sprintf("   ⚠️  Errors encountered: %d (recovered: %d)",
			r.metrics.ErrorCount, r.metrics.RecoveredErrors))
	}

	if len(r.metrics.OptimizationsApplied) > 0 {
		r.logSuccess(fmt.Sprintf("   ✨ Optimizations applied: %d", len(r.metrics.OptimizationsApplied)))
		if r.verbose {
			for _, opt := range r.metrics.OptimizationsApplied {
				r.logSuccess(fmt.Sprintf("     - %s", opt))
			}
		}
	}
}
