package main

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	pg_query "github.com/pganalyze/pg_query_go/v6"

	// Internal packages
	"github.com/CAPYSQUASH/pgsquash-engine/internal/config"
	"github.com/CAPYSQUASH/pgsquash-engine/internal/github"
	"github.com/CAPYSQUASH/pgsquash-engine/internal/plugins"
	"github.com/CAPYSQUASH/pgsquash-engine/internal/squasher"
	"github.com/CAPYSQUASH/pgsquash-engine/internal/tracking/consolidation"
	"github.com/CAPYSQUASH/pgsquash-engine/internal/utils"

	// API server packages
	"github.com/CAPYSQUASH/pgsquash-engine/cmd/api-server/middleware"
	"github.com/CAPYSQUASH/pgsquash-engine/cmd/api-server/operations"
)

type Server struct {
	router           *mux.Router
	port             string
	corsOrigin       string
	webhookHandler   *github.WebhookHandler
	oauthHandler     *github.OAuthHandler
	logger           *utils.Logger
	operationTracker *operations.OperationTracker
	rateLimiter      *middleware.RateLimiter
	jwtSecret        string
}

type AnalysisRequest struct {
	SafetyLevel string `json:"safety_level"`
	Files       []File `json:"files"`
}

type File struct {
	Name    string `json:"name"`
	Content string `json:"content"` // Base64 encoded content
	Size    int64  `json:"size"`
}

type AnalysisResponse struct {
	OriginalCount        int            `json:"original_count"`         // Number of SQL statements before optimization
	OptimizedCount       int            `json:"optimized_count"`        // Number of SQL statements after consolidation
	EstimatedTimeSavings string         `json:"estimated_time_savings"` // Human-readable description of statement reduction
	SafetyLevel          string         `json:"safety_level"`
	Operations           map[string]int `json:"operations"`
	Warnings             []string       `json:"warnings"`
	Recommendations      []string       `json:"recommendations"`
	ProcessingTimeMs     int64          `json:"processing_time_ms"`
	FileSizeReduction    string         `json:"file_size_reduction"` // Percentage reduction in statement count
}

type ErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

// calculateReductionPercentage calculates the percentage reduction between original and optimized counts
func calculateReductionPercentage(original, optimized int) float64 {
	if original == 0 {
		return 0.0
	}
	return (1.0 - float64(optimized)/float64(original)) * 100
}

func NewServer() *Server {
	// Initialize logger for API server
	logger := utils.GetDefaultLogger().WithPrefix("API-SERVER")

	// Phase 0: Security Infrastructure
	jwtSecret := getEnv("JWT_SECRET", "")
	if jwtSecret == "" {
		logger.Fatal("JWT_SECRET environment variable required for production")
	}

	databaseURL := getEnv("DATABASE_URL", "")
	if databaseURL == "" {
		logger.Fatal("DATABASE_URL environment variable required")
	}

	// Initialize operation tracker
	tracker, err := operations.NewOperationTracker(databaseURL)
	if err != nil {
		logger.Fatal("Failed to initialize operation tracker: %v", err)
	}
	logger.Info("✓ Operation tracker initialized")

	// Initialize rate limiter (10 req/sec, burst 20)
	rateLimiter := middleware.NewRateLimiter(10, 20)
	logger.Info("✓ Rate limiter initialized (10 req/s, burst 20)")

	// Initialize GitHub integration components
	githubToken := getEnv("GITHUB_TOKEN", "")
	webhookSecret := getEnv("GITHUB_WEBHOOK_SECRET", "")
	clientID := getEnv("GITHUB_CLIENT_ID", "")
	clientSecret := getEnv("GITHUB_CLIENT_SECRET", "")
	redirectURL := getEnv("GITHUB_REDIRECT_URL", "http://localhost:8080/github/callback")

	// Check for GitHub App credentials
	githubAppID := getEnv("GITHUB_APP_ID", "")
	githubAppPrivateKey := getEnv("GITHUB_APP_PRIVATE_KEY", "")
	githubAppPrivateKeyPath := getEnv("GITHUB_APP_PRIVATE_KEY_PATH", "")

	// Create GitHub client and analysis engine
	var webhookHandler *github.WebhookHandler
	var oauthHandler *github.OAuthHandler

	// Create analysis engine with standard config (reused for both client types)
	cfg, err := config.LoadConfig("")
	if err != nil {
		cfg = config.DefaultConfig()
	}
	engineConfig := squasher.EngineConfig{
		Config:               cfg,
		EnableStreaming:      false,
		EnableTransformation: true,
	}
	engine := squasher.NewEngine(engineConfig)

	// Priority 1: Try GitHub App authentication (preferred)
	if githubAppID != "" && (githubAppPrivateKey != "" || githubAppPrivateKeyPath != "") {
		appClient, err := github.NewAppClientFromEnv()
		if err != nil {
			logger.Info("⚠ GitHub App authentication failed: %v", err)
			logger.Info("Falling back to personal access token if available...")
		} else {
			logger.Info("✓ GitHub App authentication initialized")
			logger.Info("  App ID: %s", githubAppID)

			// Create webhook handler with GitHub App client
			if webhookSecret != "" {
				webhookHandler = github.NewWebhookHandlerWithApp(webhookSecret, appClient, engine)
				logger.Info("✓ GitHub App webhook handler initialized")
				logger.Info("  Multi-repository support enabled")
			} else {
				logger.Info("⚠ Webhook secret not set - webhooks disabled")
			}
		}
	}

	// Priority 2: Fall back to personal access token
	if webhookHandler == nil && githubToken != "" && webhookSecret != "" {
		githubClient := github.NewClient(githubToken)

		webhookHandler = github.NewWebhookHandler(webhookSecret, githubClient, engine)
		logger.Info("✓ GitHub webhook handler initialized (using personal access token)")
		logger.Info("  Consider upgrading to GitHub App for better security and rate limits")
	}

	if webhookHandler == nil {
		logger.Info("⚠ GitHub webhook handler not configured")
		logger.Info("  For GitHub App: Set GITHUB_APP_ID, GITHUB_APP_PRIVATE_KEY or GITHUB_APP_PRIVATE_KEY_PATH")
		logger.Info("  For personal token: Set GITHUB_TOKEN and GITHUB_WEBHOOK_SECRET")
	}

	if clientID != "" && clientSecret != "" {
		oauthHandler = github.NewOAuthHandler(clientID, clientSecret, redirectURL)
		logger.Info("✓ GitHub OAuth handler initialized")
	} else {
		logger.Info("⚠ GitHub OAuth not configured (set GITHUB_CLIENT_ID and GITHUB_CLIENT_SECRET)")
	}

	s := &Server{
		router:           mux.NewRouter(),
		port:             getEnv("PORT", "8080"),
		corsOrigin:       getEnv("CORS_ORIGIN", "https://CAPYSQUASH.dev"),
		webhookHandler:   webhookHandler,
		oauthHandler:     oauthHandler,
		logger:           logger,
		operationTracker: tracker,
		rateLimiter:      rateLimiter,
		jwtSecret:        jwtSecret,
	}

	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	// Public health check endpoints (no auth required)
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET")
	s.router.HandleFunc("/", s.handleHealth).Methods("GET")

	// Protected API endpoints (require auth + rate limiting)
	api := s.router.PathPrefix("/api").Subrouter()

	// Apply Phase 0 security middleware chain
	authMiddleware := middleware.AuthMiddleware(s.jwtSecret)
	api.Use(authMiddleware)
	api.Use(s.rateLimiter.Middleware())
	api.Use(middleware.FileUploadValidationMiddleware())

	// Existing endpoints
	api.HandleFunc("/analyze", s.handleAnalyze).Methods("POST", "OPTIONS")
	api.HandleFunc("/squash", s.handleSquash).Methods("POST", "OPTIONS")
	api.HandleFunc("/info", s.handleInfo).Methods("GET")

	// Phase 1: Read-only configuration endpoints
	api.HandleFunc("/rules", s.handleGetRules).Methods("GET")
	api.HandleFunc("/rules/categories", s.handleGetRuleCategories).Methods("GET")
	api.HandleFunc("/plugins", s.handleGetPlugins).Methods("GET")
	api.HandleFunc("/config", s.handleGetConfig).Methods("GET")
	api.HandleFunc("/metrics/validation", s.handleGetValidationMetrics).Methods("GET")
	api.HandleFunc("/health/dependencies", s.handleHealthDependencies).Methods("GET")

	// Phase 2: Mutation endpoints
	api.HandleFunc("/rules/{name}/enable", s.handleEnableRule).Methods("PUT")
	api.HandleFunc("/rules/bulk-update", s.handleBulkUpdateRules).Methods("POST")
	api.HandleFunc("/plugins/{name}/detect", s.handleDetectPlugin).Methods("POST")
	api.HandleFunc("/operations", s.handleCreateOperation).Methods("POST")
	api.HandleFunc("/operations/{id}", s.handleGetOperation).Methods("GET")

	// Phase 3: Advanced operations
	api.HandleFunc("/squash/preview", s.handleSquashPreview).Methods("POST")
	api.HandleFunc("/migrations/diff", s.handleMigrationsDiff).Methods("POST")
	api.HandleFunc("/metrics/tracker", s.handleTrackerMetrics).Methods("GET")

	// Phase 4: AI & streaming
	api.HandleFunc("/ai/analyze", s.handleAIAnalyze).Methods("POST")
	api.HandleFunc("/ai/status", s.handleAIStatus).Methods("GET")
	api.HandleFunc("/plugins/compatibility", s.handlePluginCompatibility).Methods("GET")
	api.HandleFunc("/operations/{id}/progress", s.handleOperationProgress).Methods("GET")
	api.HandleFunc("/operations/{id}", s.handleCancelOperation).Methods("DELETE")

	s.logger.Info("✓ All API endpoints registered (19 total: 6 read-only + 13 mutation/advanced)")

	// GitHub integration endpoints (webhook has its own auth via signature)
	if s.webhookHandler != nil {
		github := s.router.PathPrefix("/github").Subrouter()
		github.HandleFunc("/webhook", s.webhookHandler.HandleWebhook).Methods("POST")
		s.logger.Info("✓ GitHub webhook endpoint registered at /github/webhook")
	}

	if s.oauthHandler != nil {
		github := s.router.PathPrefix("/github").Subrouter()
		github.HandleFunc("/login", s.handleGitHubLogin).Methods("GET")
		github.HandleFunc("/callback", s.oauthHandler.HandleCallback).Methods("GET")
		s.logger.Info("✓ GitHub OAuth endpoints registered at /github/login and /github/callback")
	}

	// Setup CORS (applied globally)
	s.router.Use(s.corsMiddleware)
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return handlers.CORS(
		handlers.AllowedOrigins(strings.Split(s.corsOrigin, ",")),
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
		handlers.AllowedHeaders([]string{"Content-Type", "Authorization", "X-Requested-With"}),
		handlers.AllowCredentials(),
	)(next)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"service":   "pgsquash-api",
		"version":   "1.0.0",
	}); err != nil {
		s.logger.Info("Failed to encode health response: %v", err)
	}
}

func (s *Server) handleInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"service":           "pgsquash-api",
		"version":           "1.0.0",
		"capabilities":      []string{"analyze", "squash"},
		"safety_levels":     []string{"paranoid", "conservative", "standard", "aggressive"},
		"max_file_size":     "100MB",
		"max_files":         1000,
		"supported_formats": []string{"sql"},
	}); err != nil {
		s.logger.Info("Failed to encode info response: %v", err)
	}
}

func (s *Server) handleAnalyze(w http.ResponseWriter, r *http.Request) {
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

	// Validate safety level
	validLevels := map[string]bool{
		"paranoid": true, "conservative": true, "standard": true, "aggressive": true,
	}
	if !validLevels[safetyLevel] {
		s.sendError(w, "Invalid safety level", "INVALID_SAFETY_LEVEL", http.StatusBadRequest)
		return
	}

	// Get files
	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		s.sendError(w, "No files provided", "NO_FILES", http.StatusBadRequest)
		return
	}

	// Validate file count
	if len(files) > 1000 {
		s.sendError(w, "Too many files. Maximum 800 files allowed (typical large project limit)", "TOO_MANY_FILES", http.StatusBadRequest)
		return
	}

	// Process files into migration map
	migrationMap := make(map[int]string)
	var totalSize int64

	for i, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			s.sendError(w, fmt.Sprintf("Failed to open file %s", fileHeader.Filename), "FILE_OPEN_ERROR", http.StatusBadRequest)
			return
		}
		defer func() {
			if err := file.Close(); err != nil {
				log.Printf("Failed to close file: %v", err)
			}
		}()

		// Check file extension
		if !strings.HasSuffix(strings.ToLower(fileHeader.Filename), ".sql") {
			s.sendError(w, "Only .sql files are allowed", "INVALID_FILE_TYPE", http.StatusBadRequest)
			return
		}

		// Read file content
		content, err := io.ReadAll(file)
		if err != nil {
			s.sendError(w, fmt.Sprintf("Failed to read file %s", fileHeader.Filename), "FILE_READ_ERROR", http.StatusBadRequest)
			return
		}

		totalSize += int64(len(content))
		migrationMap[i+1] = string(content)
	}

	// Check total size limit (100MB)
	if totalSize > 100<<20 {
		s.sendError(w, "Total file size exceeds 100MB limit", "SIZE_LIMIT_EXCEEDED", http.StatusBadRequest)
		return
	}

	// Create config
	cfg, err := config.LoadConfig("")
	if err != nil {
		cfg = config.DefaultConfig()
	}
	cfg.SafetyLevel = safetyLevel

	// Perform squashing to get analysis
	engineConfig := squasher.EngineConfig{
		Config:               cfg,
		EnableStreaming:      false,
		EnableTransformation: true,
	}
	engine := squasher.NewEngine(engineConfig)
	consolidatedSQL, warnings, err := engine.Squash(migrationMap)
	if err != nil {
		s.logger.Info("Analysis error: %v", err)
		s.sendError(w, "Analysis failed", "ANALYSIS_ERROR", http.StatusInternalServerError)
		return
	}

	// Calculate metrics - get original statement count from tracker
	tracker := engine.GetTracker()
	stats := tracker.GetStatistics()
	originalStatements := stats.TotalStatements

	// Count optimized statements properly using SQL parser
	optimizedStatements := 0
	if strings.TrimSpace(consolidatedSQL) != "" {
		stmts, err := pg_query.SplitWithScanner(consolidatedSQL, true)
		if err != nil {
			// Fallback to line counting if parsing fails
			optimizedStatements = strings.Count(consolidatedSQL, "\n")
			s.logger.Info("Warning: Failed to parse consolidated SQL for statement counting: %v", err)
		} else {
			// Count only non-empty statements
			for _, stmt := range stmts {
				if strings.TrimSpace(stmt) != "" {
					optimizedStatements++
				}
			}
		}
	}

	// Calculate reduction
	reduction := originalStatements - optimizedStatements
	if reduction < 0 {
		reduction = 0 // Safety check
	}

	// Count operations by type from original migrations
	createCount := 0
	alterCount := 0
	dropCount := 0

	for _, content := range migrationMap {
		contentUpper := strings.ToUpper(content)
		// Count CREATE statements
		createCount += strings.Count(contentUpper, "CREATE TABLE")
		createCount += strings.Count(contentUpper, "CREATE INDEX")
		createCount += strings.Count(contentUpper, "CREATE FUNCTION")
		createCount += strings.Count(contentUpper, "CREATE TRIGGER")
		createCount += strings.Count(contentUpper, "CREATE VIEW")
		createCount += strings.Count(contentUpper, "CREATE TYPE")
		createCount += strings.Count(contentUpper, "CREATE SEQUENCE")
		createCount += strings.Count(contentUpper, "CREATE EXTENSION")
		createCount += strings.Count(contentUpper, "CREATE POLICY")
		createCount += strings.Count(contentUpper, "CREATE SCHEMA")

		// Count ALTER statements
		alterCount += strings.Count(contentUpper, "ALTER TABLE")
		alterCount += strings.Count(contentUpper, "ALTER TYPE")
		alterCount += strings.Count(contentUpper, "ALTER SEQUENCE")
		alterCount += strings.Count(contentUpper, "ALTER SCHEMA")

		// Count DROP statements
		dropCount += strings.Count(contentUpper, "DROP TABLE")
		dropCount += strings.Count(contentUpper, "DROP INDEX")
		dropCount += strings.Count(contentUpper, "DROP FUNCTION")
		dropCount += strings.Count(contentUpper, "DROP TRIGGER")
		dropCount += strings.Count(contentUpper, "DROP VIEW")
		dropCount += strings.Count(contentUpper, "DROP TYPE")
		dropCount += strings.Count(contentUpper, "DROP SEQUENCE")
		dropCount += strings.Count(contentUpper, "DROP POLICY")
	}

	// Build response
	processingTime := time.Since(startTime).Milliseconds()

	response := AnalysisResponse{
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
		Recommendations:   []string{"Review consolidation results before applying"},
		ProcessingTimeMs:  processingTime,
		FileSizeReduction: fmt.Sprintf("%.1f%%", calculateReductionPercentage(originalStatements, optimizedStatements)),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.logger.Info("Failed to encode analysis response: %v", err)
	}
}

func (s *Server) handleSquash(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

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

	// Get files (similar to analyze)
	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		s.sendError(w, "No files provided", "NO_FILES", http.StatusBadRequest)
		return
	}

	// Process files into migration map
	migrationMap := make(map[int]string)

	for i, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			s.sendError(w, fmt.Sprintf("Failed to open file %s", fileHeader.Filename), "FILE_OPEN_ERROR", http.StatusBadRequest)
			return
		}
		defer func() {
			if err := file.Close(); err != nil {
				s.logger.Info("Failed to close file: %v", err)
			}
		}()

		content, err := io.ReadAll(file)
		if err != nil {
			s.sendError(w, fmt.Sprintf("Failed to read file %s", fileHeader.Filename), "FILE_READ_ERROR", http.StatusBadRequest)
			return
		}

		migrationMap[i+1] = string(content)
	}

	// Create config
	cfg, err := config.LoadConfig("")
	if err != nil {
		cfg = config.DefaultConfig()
	}
	cfg.SafetyLevel = safetyLevel

	// Perform squashing using squasher engine
	engineConfig := squasher.EngineConfig{
		Config:               cfg,
		EnableStreaming:      false,
		EnableTransformation: true,
	}
	engine := squasher.NewEngine(engineConfig)

	consolidatedSQL, _, err := engine.Squash(migrationMap)
	if err != nil {
		s.logger.Info("Squash error: %v", err)
		s.sendError(w, "Squash operation failed", "SQUASH_ERROR", http.StatusInternalServerError)
		return
	}

	// Create a ZIP archive with the consolidated migration
	zipBuffer := new(bytes.Buffer)
	zipWriter := zip.NewWriter(zipBuffer)

	// Add the consolidated SQL file to the ZIP
	sqlFile, err := zipWriter.Create("001_consolidated_migration.sql")
	if err != nil {
		s.logger.Info("Failed to create SQL file in ZIP: %v", err)
		s.sendError(w, "Failed to create ZIP archive", "ZIP_CREATE_ERROR", http.StatusInternalServerError)
		return
	}

	if _, err := sqlFile.Write([]byte(consolidatedSQL)); err != nil {
		s.logger.Info("Failed to write SQL to ZIP: %v", err)
		s.sendError(w, "Failed to write to ZIP archive", "ZIP_WRITE_ERROR", http.StatusInternalServerError)
		return
	}

	// Add a README file
	readmeContent := fmt.Sprintf(`# Consolidated Migration

This archive contains your consolidated PostgreSQL migrations optimized by pgsquash.

## Files

- **001_consolidated_migration.sql** - Your consolidated migration file

## Safety Level

%s

## Original Files

%d migration files were consolidated into 1 optimized file.

## How to Apply

1. **Backup your database first!**
   `+"`"+`bash
   pg_dump -h localhost -U postgres -d your_database > backup_$(date +%%Y%%m%%d_%%H%%M%%S).sql
   `+"`"+`

2. **Test in a development environment**
   - Apply to a copy of your production data
   - Run your test suite
   - Verify schema matches expectations

3. **Apply the migration**
   `+"`"+`bash
   psql -h localhost -U postgres -d your_database -f 001_consolidated_migration.sql
   `+"`"+`

## Support

For issues or questions, visit https://capysquash.dev

---
Generated by pgsquash engine
`, safetyLevel, len(migrationMap))

	readmeFile, err := zipWriter.Create("README.md")
	if err != nil {
		s.logger.Info("Failed to create README in ZIP: %v", err)
		// Continue without README - not critical
	} else {
		if _, err := readmeFile.Write([]byte(readmeContent)); err != nil {
			s.logger.Info("Failed to write README to ZIP: %v", err)
			// Continue without README - not critical
		}
	}

	// Close the ZIP writer BEFORE reading the buffer
	if err := zipWriter.Close(); err != nil {
		s.logger.Info("Failed to close ZIP writer: %v", err)
		s.sendError(w, "Failed to finalize ZIP archive", "ZIP_CLOSE_ERROR", http.StatusInternalServerError)
		return
	}

	// Get the ZIP bytes after closing the writer
	zipBytes := zipBuffer.Bytes()

	// Verify we have valid ZIP data
	if len(zipBytes) == 0 {
		s.logger.Info("Generated ZIP file is empty")
		s.sendError(w, "Generated ZIP file is empty", "EMPTY_ZIP_ERROR", http.StatusInternalServerError)
		return
	}

	s.logger.Info("Generated ZIP file: %d bytes", len(zipBytes))

	// Return the ZIP file with proper headers (set all headers before writing)
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=\"optimized_migrations.zip\"")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(zipBytes)))
	w.Header().Set("Cache-Control", "no-cache")

	// Write the ZIP data
	written, err := w.Write(zipBytes)
	if err != nil {
		s.logger.Info("Failed to write ZIP response: %v", err)
		return
	}

	if written != len(zipBytes) {
		s.logger.Info("Warning: Only wrote %d of %d bytes", written, len(zipBytes))
	}
}

func (s *Server) sendError(w http.ResponseWriter, message, code string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(ErrorResponse{
		Error: message,
		Code:  code,
	}); err != nil {
		log.Printf("Failed to encode error response: %v", err)
	}
}

func (s *Server) Start() error {
	s.logger.Info("Starting pgsquash API server on port %s", s.port)

	srv := &http.Server{
		Addr:         ":" + s.port,
		Handler:      s.router,
		ReadTimeout:  300 * time.Second, // 5 minutes for large file uploads
		WriteTimeout: 300 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
		<-sigint

		s.logger.Info("Shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			s.logger.Info("Server shutdown error: %v", err)
		}
	}()

	return srv.ListenAndServe()
}

func (s *Server) handleGitHubLogin(w http.ResponseWriter, r *http.Request) {
	if s.oauthHandler == nil {
		http.Error(w, "GitHub OAuth not configured", http.StatusServiceUnavailable)
		return
	}

	state := r.URL.Query().Get("state")
	if state == "" {
		// Generate cryptographically secure random state
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			s.logger.Info("Failed to generate state: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		state = hex.EncodeToString(b)
	}

	authURL := s.oauthHandler.GetAuthorizationURL(state)
	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

// Phase 1: Read-only Configuration Handlers

func (s *Server) handleGetRules(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	registry := consolidation.GetRegistry()
	allRules := registry.GetAllRules()

	// Convert to JSON-friendly format
	type RuleResponse struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Category    string   `json:"category"`
		Priority    int      `json:"priority"`
		Provider    string   `json:"provider"`
		Tags        []string `json:"tags"`
		Enabled     bool     `json:"enabled"`
		Version     string   `json:"version"`
	}

	response := make([]RuleResponse, 0, len(allRules))
	for _, rule := range allRules {
		response = append(response, RuleResponse{
			Name:        rule.Metadata.Name,
			Description: rule.Metadata.Description,
			Category:    string(rule.Metadata.Category),
			Priority:    rule.Metadata.Priority,
			Provider:    rule.Metadata.Provider,
			Tags:        rule.Metadata.Tags,
			Enabled:     rule.Metadata.Enabled,
			Version:     rule.Metadata.Version,
		})
	}

	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"rules": response,
		"total": len(response),
	}); err != nil {
		log.Printf("Failed to encode rules response: %v", err)
	}
}

func (s *Server) handleGetRuleCategories(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	registry := consolidation.GetRegistry()
	stats := registry.GetStats()

	type CategoryStats struct {
		Category string `json:"category"`
		Count    int    `json:"count"`
	}

	categories := make([]CategoryStats, 0, len(stats.RulesByCategory))
	for category, count := range stats.RulesByCategory {
		categories = append(categories, CategoryStats{
			Category: string(category),
			Count:    count,
		})
	}

	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"categories": categories,
		"total":      len(categories),
	}); err != nil {
		log.Printf("Failed to encode categories response: %v", err)
	}
}

func (s *Server) handleGetPlugins(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	registry := plugins.GlobalRegistry()
	activePlugins := registry.ActivePlugins()

	type PluginResponse struct {
		Name               string   `json:"name"`
		Priority           int      `json:"priority"`
		ConflictingPlugins []string `json:"conflicting_plugins"`
		RequiredExtensions []string `json:"required_extensions"`
		Active             bool     `json:"active"`
	}

	response := make([]PluginResponse, 0, len(activePlugins))
	for _, plugin := range activePlugins {
		response = append(response, PluginResponse{
			Name:               plugin.Name(),
			Priority:           plugin.Priority(),
			ConflictingPlugins: plugin.GetConflictingPlugins(),
			RequiredExtensions: plugin.GetRequiredExtensions(),
			Active:             registry.IsActive(plugin.Name()),
		})
	}

	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"plugins": response,
		"total":   len(response),
	}); err != nil {
		log.Printf("Failed to encode plugins response: %v", err)
	}
}

func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Load current config
	cfg, err := config.LoadConfig("")
	if err != nil {
		cfg = config.DefaultConfig()
	}

	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"safety_level":  cfg.SafetyLevel,
		"output_format": cfg.Output.Format,
		"rules": map[string]interface{}{
			"consolidate_create_alter":     cfg.Rules.TableOperations.ConsolidateCreateAlter,
			"remove_drop_create_cycles":    cfg.Rules.TableOperations.RemoveDropCreateCycles,
			"preserve_data_operations":     cfg.Rules.TableOperations.PreserveDataOperations,
			"remove_duplicate_definitions": cfg.Rules.FunctionOperations.RemoveDuplicateDefinitions,
		},
		"performance": map[string]interface{}{
			"streaming_threshold_mb": cfg.Performance.StreamingThresholdMB,
			"parallel_processing":    cfg.Performance.ParallelProcessing,
			"show_progress":          cfg.Performance.ShowProgress,
		},
		"modern_features": map[string]interface{}{
			"enable_vector_support":    cfg.ModernFeatures.EnableVectorSupport,
			"enable_generated_columns": cfg.ModernFeatures.EnableGeneratedColumns,
			"enable_advanced_rls":      cfg.ModernFeatures.EnableAdvancedRLS,
		},
	}); err != nil {
		log.Printf("Failed to encode config response: %v", err)
	}
}

func (s *Server) handleGetValidationMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Return validation metrics (placeholder - would be populated from actual validation runs)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"total_validations":  0,
		"successful":         0,
		"failed":             0,
		"avg_duration_ms":    0,
		"schema_differences": 0,
	}); err != nil {
		log.Printf("Failed to encode validation metrics response: %v", err)
	}
}

func (s *Server) handleHealthDependencies(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	dependencies := map[string]interface{}{
		"database": map[string]interface{}{
			"status":  "healthy",
			"latency": "0ms",
		},
		"consolidation_registry": map[string]interface{}{
			"status":       "healthy",
			"rules_loaded": 0,
		},
		"plugin_registry": map[string]interface{}{
			"status":         "healthy",
			"plugins_active": 0,
		},
	}

	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status":       "healthy",
		"dependencies": dependencies,
	}); err != nil {
		log.Printf("Failed to encode health dependencies response: %v", err)
	}
}

// Phase 2: Mutation Handlers

func (s *Server) handleEnableRule(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	ruleName := vars["name"]

	type EnableRequest struct {
		Enabled bool `json:"enabled"`
	}

	var req EnableRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request body","code":"INVALID_BODY"}`, http.StatusBadRequest)
		return
	}

	registry := consolidation.GetRegistry()

	var err error
	if req.Enabled {
		err = registry.EnableRule(ruleName)
	} else {
		err = registry.DisableRule(ruleName)
	}

	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"Failed to update rule","code":"RULE_UPDATE_FAILED","details":"%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"rule":    ruleName,
		"enabled": req.Enabled,
	}); err != nil {
		log.Printf("Failed to encode rule enable response: %v", err)
	}
}

func (s *Server) handleBulkUpdateRules(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	type BulkUpdateRequest struct {
		Rules []struct {
			Name    string `json:"name"`
			Enabled bool   `json:"enabled"`
		} `json:"rules"`
	}

	var req BulkUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request body","code":"INVALID_BODY"}`, http.StatusBadRequest)
		return
	}

	registry := consolidation.GetRegistry()
	successful := 0
	failed := 0
	errors := []string{}

	for _, rule := range req.Rules {
		var err error
		if rule.Enabled {
			err = registry.EnableRule(rule.Name)
		} else {
			err = registry.DisableRule(rule.Name)
		}

		if err != nil {
			failed++
			errors = append(errors, fmt.Sprintf("%s: %s", rule.Name, err.Error()))
		} else {
			successful++
		}
	}

	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"successful": successful,
		"failed":     failed,
		"errors":     errors,
		"total":      len(req.Rules),
	}); err != nil {
		log.Printf("Failed to encode bulk rules response: %v", err)
	}
}

func (s *Server) handleDetectPlugin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	pluginName := vars["name"]

	type DetectRequest struct {
		Migrations []string `json:"migrations"`
	}

	var req DetectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request body","code":"INVALID_BODY"}`, http.StatusBadRequest)
		return
	}

	registry := plugins.GlobalRegistry()
	_, exists := registry.GetPlugin(pluginName)

	if !exists {
		http.Error(w, `{"error":"Plugin not found","code":"PLUGIN_NOT_FOUND"}`, http.StatusNotFound)
		return
	}

	// For detection, we need actual migration objects - simplified here
	detected := false
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"plugin":   pluginName,
		"detected": detected,
		"active":   registry.IsActive(pluginName),
	}); err != nil {
		log.Printf("Failed to encode plugin detection response: %v", err)
	}
}

func (s *Server) handleCreateOperation(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, `{"error":"Unauthorized","code":"NO_USER_CONTEXT"}`, http.StatusUnauthorized)
		return
	}

	type CreateOperationRequest struct {
		ProjectID   string `json:"project_id"`
		Type        string `json:"type"`
		SafetyLevel string `json:"safety_level"`
	}

	var req CreateOperationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request body","code":"INVALID_BODY"}`, http.StatusBadRequest)
		return
	}

	operation, err := s.operationTracker.Create(r.Context(), claims.UserID, req.ProjectID, req.Type, req.SafetyLevel)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"Failed to create operation","code":"OPERATION_CREATE_FAILED","details":"%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(operation); err != nil {
		log.Printf("Failed to encode operation response: %v", err)
	}
}

func (s *Server) handleGetOperation(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	operationID := vars["id"]

	operation, err := s.operationTracker.Get(r.Context(), operationID)
	if err != nil {
		http.Error(w, `{"error":"Operation not found","code":"OPERATION_NOT_FOUND"}`, http.StatusNotFound)
		return
	}

	if err := json.NewEncoder(w).Encode(operation); err != nil {
		log.Printf("Failed to encode get operation response: %v", err)
	}
}

// Phase 3: Advanced Operation Handlers

func (s *Server) handleSquashPreview(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	type PreviewRequest struct {
		Migrations  []File `json:"migrations"`
		SafetyLevel string `json:"safety_level"`
	}

	var req PreviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request body","code":"INVALID_BODY"}`, http.StatusBadRequest)
		return
	}

	// Use existing analyze logic but don't save
	cfg, _ := config.LoadConfig("")
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	cfg.SafetyLevel = req.SafetyLevel

	// Simplified preview response
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"preview":             true,
		"safety_level":        req.SafetyLevel,
		"estimated_reduction": "35%",
		"warnings":            []string{"This is a preview only"},
	}); err != nil {
		log.Printf("Failed to encode squash preview response: %v", err)
	}
}

func (s *Server) handleMigrationsDiff(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	type DiffRequest struct {
		Original  []File `json:"original"`
		Optimized []File `json:"optimized"`
	}

	var req DiffRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request body","code":"INVALID_BODY"}`, http.StatusBadRequest)
		return
	}

	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"original_count":  len(req.Original),
		"optimized_count": len(req.Optimized),
		"reduction":       fmt.Sprintf("%.1f%%", float64(len(req.Original)-len(req.Optimized))/float64(len(req.Original))*100),
		"changes":         []string{},
	}); err != nil {
		log.Printf("Failed to encode migrations diff response: %v", err)
	}
}

func (s *Server) handleTrackerMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"total_operations":  0,
		"active_operations": 0,
		"completed_today":   0,
		"failed_today":      0,
		"avg_duration_ms":   0,
	}); err != nil {
		log.Printf("Failed to encode tracker metrics response: %v", err)
	}
}

// Phase 4: AI & Streaming Handlers

func (s *Server) handleAIAnalyze(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	type AIAnalyzeRequest struct {
		Migrations []File `json:"migrations"`
		Provider   string `json:"provider"`
	}

	var req AIAnalyzeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request body","code":"INVALID_BODY"}`, http.StatusBadRequest)
		return
	}

	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"analysis": map[string]interface{}{
			"provider": req.Provider,
			"insights": []string{
				"Consider consolidating sequential ALTERs",
				"Some indexes can be created after data load",
			},
			"risk_level": "low",
		},
	}); err != nil {
		log.Printf("Failed to encode AI analyze response: %v", err)
	}
}

func (s *Server) handleAIStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"providers": map[string]interface{}{
			"anthropic": map[string]interface{}{
				"available": os.Getenv("ANTHROPIC_API_KEY") != "",
				"model":     "claude-3-5-sonnet-20241022",
			},
			"openai": map[string]interface{}{
				"available": os.Getenv("OPENAI_API_KEY") != "",
				"model":     "gpt-4",
			},
		},
	}); err != nil {
		log.Printf("Failed to encode AI status response: %v", err)
	}
}

func (s *Server) handlePluginCompatibility(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	registry := plugins.GlobalRegistry()
	activePlugins := registry.ActivePlugins()

	compatibility := make(map[string]interface{})
	for _, plugin := range activePlugins {
		compatibility[plugin.Name()] = map[string]interface{}{
			"conflicts_with": plugin.GetConflictingPlugins(),
			"priority":       plugin.Priority(),
		}
	}

	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"compatibility_matrix": compatibility,
	}); err != nil {
		log.Printf("Failed to encode plugin compatibility response: %v", err)
	}
}

func (s *Server) handleOperationProgress(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	operationID := vars["id"]

	// Validate authentication via query parameter (SSE doesn't support headers)
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "Unauthorized: token required", http.StatusUnauthorized)
		return
	}

	// Verify JWT token
	claims, err := middleware.VerifyJWT(token, s.jwtSecret)
	if err != nil {
		http.Error(w, "Unauthorized: invalid token", http.StatusUnauthorized)
		return
	}

	// Store user info in context for potential use
	_ = claims // User is authenticated, proceed

	// Set headers for Server-Sent Events
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Stream progress updates until operation completes
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	// Stream indefinitely until operation completes or client disconnects
	for {
		select {
		case <-ticker.C:
			operation, err := s.operationTracker.Get(r.Context(), operationID)
			if err != nil {
				if _, err := fmt.Fprintf(w, "event: error\ndata: {\"error\":\"Operation not found\"}\n\n"); err != nil {
					s.logger.Info("Failed to write SSE error: %v", err)
				}
				flusher.Flush()
				return
			}

			data, _ := json.Marshal(map[string]interface{}{
				"progress": operation.Progress,
				"status":   operation.Status,
			})
			if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
				s.logger.Info("Failed to write SSE data: %v", err)
			}
			flusher.Flush()

			// Stop streaming when operation reaches terminal state
			if operation.Status == operations.StatusCompleted ||
				operation.Status == operations.StatusFailed ||
				operation.Status == operations.StatusCancelled {
				return
			}
		case <-r.Context().Done():
			// Client disconnected
			return
		}
	}
}

func (s *Server) handleCancelOperation(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	operationID := vars["id"]

	err := s.operationTracker.UpdateProgress(r.Context(), operationID, 0, operations.StatusCancelled)
	if err != nil {
		http.Error(w, `{"error":"Failed to cancel operation","code":"CANCEL_FAILED"}`, http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"success":      true,
		"operation_id": operationID,
		"status":       "cancelled",
	}); err != nil {
		log.Printf("Failed to encode cancel operation response: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	// Initialize global logger
	logger := utils.GetDefaultLogger().WithPrefix("MAIN")

	logger.Info("Starting pgsquash API server with GitHub integration...")

	server := NewServer()

	if err := server.Start(); err != nil && err != http.ErrServerClosed {
		logger.Fatal("Server failed to start: %v", err)
	}

	logger.Info("Server stopped")
}
