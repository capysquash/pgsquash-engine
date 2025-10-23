package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	pg_query "github.com/pganalyze/pg_query_go/v6"

	"github.com/CAPYSQUASH/pgsquash-engine/internal/config"
	"github.com/CAPYSQUASH/pgsquash-engine/internal/squasher"
	"github.com/CAPYSQUASH/pgsquash-engine/pkg/engine"
)

// DetailedAnalysisResponse extends AnalysisResponse with enhanced metrics
type DetailedAnalysisResponse struct {
	OriginalCount        int                        `json:"original_count"`
	OptimizedCount       int                        `json:"optimized_count"`
	EstimatedTimeSavings string                     `json:"estimated_time_savings"`
	SafetyLevel          string                     `json:"safety_level"`
	Operations           map[string]int             `json:"operations"`
	Warnings             []string                   `json:"warnings"`
	Recommendations      []engine.RecommendedAction `json:"recommendations"`
	ProcessingTimeMs     int64                      `json:"processing_time_ms"`
	FileSizeReduction    string                     `json:"file_size_reduction"`

	// Enhanced metrics
	DetailedMetrics *engine.DetailedMetrics `json:"detailed_metrics,omitempty"`
}

func (s *Server) handleAnalyzeDetailed(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	startTime := time.Now()

	// Parse multipart form
	err := r.ParseMultipartForm(100 << 20) // 100MB max
	if err != nil {
		s.sendError(w, "Failed to parse form data", "INVALID_FORM", http.StatusBadRequest)
		return
	}

	// Get safety level
	safetyLevel := r.FormValue("safety_level")
	if safetyLevel == "" {
		safetyLevel = "standard"
	}

	// Get files
	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		s.sendError(w, "No files provided", "NO_FILES", http.StatusBadRequest)
		return
	}

	// Process files into migration map
	migrationMap := make(map[int]string)
	var totalOriginalSize int64

	for i, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			s.sendError(w, fmt.Sprintf("Failed to open file %s", fileHeader.Filename), "FILE_OPEN_ERROR", http.StatusBadRequest)
			return
		}
		defer file.Close()

		content, err := io.ReadAll(file)
		if err != nil {
			s.sendError(w, fmt.Sprintf("Failed to read file %s", fileHeader.Filename), "FILE_READ_ERROR", http.StatusBadRequest)
			return
		}

		totalOriginalSize += int64(len(content))
		migrationMap[i+1] = string(content)
	}

	// Create config
	cfg, err := config.LoadConfig("")
	if err != nil {
		cfg = config.DefaultConfig()
	}
	cfg.SafetyLevel = safetyLevel

	// Perform analysis using squasher engine
	engineConfig := squasher.EngineConfig{
		Config:               cfg,
		EnableStreaming:      false,
		EnableTransformation: true,
	}
	eng := squasher.NewEngine(engineConfig)
	consolidatedSQL, warnings, err := eng.Squash(migrationMap)
	if err != nil {
		s.logger.Info("Analysis error: %v", err)
		s.sendError(w, "Analysis failed", "ANALYSIS_ERROR", http.StatusInternalServerError)
		return
	}

	// Calculate metrics
	tracker := eng.GetTracker()
	stats := tracker.GetStatistics()
	originalStatements := stats.TotalStatements

	// Count optimized statements
	optimizedStatements := 0
	if strings.TrimSpace(consolidatedSQL) != "" {
		stmts, err := pg_query.SplitWithScanner(consolidatedSQL, true)
		if err != nil {
			optimizedStatements = strings.Count(consolidatedSQL, "\n")
			s.logger.Info("Warning: Failed to parse consolidated SQL: %v", err)
		} else {
			for _, stmt := range stmts {
				if strings.TrimSpace(stmt) != "" {
					optimizedStatements++
				}
			}
		}
	}

	// Count operations by type
	createCount, alterCount, dropCount := countOperations(migrationMap)

	// Calculate sizes
	optimizedSize := int64(len(consolidatedSQL))
	reduction := originalStatements - optimizedStatements
	if reduction < 0 {
		reduction = 0
	}

	// Build detailed metrics
	detailedMetrics := engine.CalculateDetailedMetrics(
		&engine.SquashResult{
			FilesProcessed:      len(files),
			ObjectsConsolidated: reduction,
			Warnings:            warnings,
		},
		totalOriginalSize,
		optimizedSize,
	)

	// Generate recommendations
	recommendations := engine.GenerateRecommendations(
		&engine.SquashResult{
			FilesProcessed:      len(files),
			ObjectsConsolidated: reduction,
			Warnings:            warnings,
		},
		detailedMetrics,
	)

	// Build response
	processingTime := time.Since(startTime).Milliseconds()

	response := DetailedAnalysisResponse{
		OriginalCount:        originalStatements,
		OptimizedCount:       optimizedStatements,
		EstimatedTimeSavings: fmt.Sprintf("~%d statements reduced", reduction),
		SafetyLevel:          safetyLevel,
		Operations: map[string]int{
			"creates":      createCount,
			"alters":       alterCount,
			"drops":        dropCount,
			"consolidated": reduction,
		},
		Warnings:          warnings,
		Recommendations:   recommendations,
		ProcessingTimeMs:  processingTime,
		FileSizeReduction: fmt.Sprintf("%.1f%%", calculateReductionPercentage(originalStatements, optimizedStatements)),
		DetailedMetrics:   detailedMetrics,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.logger.Info("Failed to encode detailed analysis response: %v", err)
	}
}

// countOperations counts CREATE, ALTER, and DROP operations in migrations
func countOperations(migrationMap map[int]string) (creates, alters, drops int) {
	for _, content := range migrationMap {
		contentUpper := strings.ToUpper(content)

		// Count CREATE statements
		creates += strings.Count(contentUpper, "CREATE TABLE")
		creates += strings.Count(contentUpper, "CREATE INDEX")
		creates += strings.Count(contentUpper, "CREATE FUNCTION")
		creates += strings.Count(contentUpper, "CREATE TRIGGER")
		creates += strings.Count(contentUpper, "CREATE VIEW")
		creates += strings.Count(contentUpper, "CREATE TYPE")
		creates += strings.Count(contentUpper, "CREATE SEQUENCE")
		creates += strings.Count(contentUpper, "CREATE EXTENSION")
		creates += strings.Count(contentUpper, "CREATE POLICY")
		creates += strings.Count(contentUpper, "CREATE SCHEMA")

		// Count ALTER statements
		alters += strings.Count(contentUpper, "ALTER TABLE")
		alters += strings.Count(contentUpper, "ALTER TYPE")
		alters += strings.Count(contentUpper, "ALTER SEQUENCE")
		alters += strings.Count(contentUpper, "ALTER SCHEMA")

		// Count DROP statements
		drops += strings.Count(contentUpper, "DROP TABLE")
		drops += strings.Count(contentUpper, "DROP INDEX")
		drops += strings.Count(contentUpper, "DROP FUNCTION")
		drops += strings.Count(contentUpper, "DROP TRIGGER")
		drops += strings.Count(contentUpper, "DROP VIEW")
		drops += strings.Count(contentUpper, "DROP TYPE")
		drops += strings.Count(contentUpper, "DROP SEQUENCE")
		drops += strings.Count(contentUpper, "DROP POLICY")
	}

	return creates, alters, drops
}
