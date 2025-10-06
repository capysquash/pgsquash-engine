package main

import (
	"context"
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

	// Internal packages
	"github.com/capysquash/pg-squash-engine/internal/config"
	"github.com/capysquash/pg-squash-engine/internal/github"
	"github.com/capysquash/pg-squash-engine/internal/squasher"
)

type Server struct {
    router         *mux.Router
    port           string
    corsOrigin     string
    webhookHandler *github.WebhookHandler
    oauthHandler   *github.OAuthHandler
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
    OriginalCount        int                    `json:"original_count"`
    OptimizedCount       int                    `json:"optimized_count"`
    EstimatedTimeSavings string                 `json:"estimated_time_savings"`
    SafetyLevel          string                 `json:"safety_level"`
    Operations           map[string]int         `json:"operations"`
    Warnings             []string               `json:"warnings"`
    Recommendations      []string               `json:"recommendations"`
    ProcessingTimeMs     int64                  `json:"processing_time_ms"`
    FileSizeReduction    string                 `json:"file_size_reduction"`
}

type ErrorResponse struct {
    Error string `json:"error"`
    Code  string `json:"code"`
}

func NewServer() *Server {
    // Initialize GitHub integration components
    githubToken := getEnv("GITHUB_TOKEN", "")
    webhookSecret := getEnv("GITHUB_WEBHOOK_SECRET", "")
    clientID := getEnv("GITHUB_CLIENT_ID", "")
    clientSecret := getEnv("GITHUB_CLIENT_SECRET", "")
    redirectURL := getEnv("GITHUB_REDIRECT_URL", "http://localhost:8080/github/callback")

    // Create GitHub client and analysis engine
    var webhookHandler *github.WebhookHandler
    var oauthHandler *github.OAuthHandler

    if githubToken != "" && webhookSecret != "" {
        githubClient := github.NewClient(githubToken)

        // Create analysis engine with standard config
        cfg, err := config.LoadConfig("")
        if err != nil {
            cfg = config.DefaultConfig()
        }
        engine := squasher.NewEngine(cfg)

        webhookHandler = github.NewWebhookHandler(webhookSecret, githubClient, engine)
        log.Println("✓ GitHub webhook handler initialized")
    } else {
        log.Println("⚠ GitHub webhook handler not configured (set GITHUB_TOKEN and GITHUB_WEBHOOK_SECRET)")
    }

    if clientID != "" && clientSecret != "" {
        oauthHandler = github.NewOAuthHandler(clientID, clientSecret, redirectURL)
        log.Println("✓ GitHub OAuth handler initialized")
    } else {
        log.Println("⚠ GitHub OAuth not configured (set GITHUB_CLIENT_ID and GITHUB_CLIENT_SECRET)")
    }

    s := &Server{
        router:         mux.NewRouter(),
        port:           getEnv("PORT", "8080"),
        corsOrigin:     getEnv("CORS_ORIGIN", "*"),
        webhookHandler: webhookHandler,
        oauthHandler:   oauthHandler,
    }

    s.setupRoutes()
    return s
}

func (s *Server) setupRoutes() {
    // Health check
    s.router.HandleFunc("/health", s.handleHealth).Methods("GET")
    s.router.HandleFunc("/", s.handleHealth).Methods("GET")

    // API endpoints
    api := s.router.PathPrefix("/api").Subrouter()
    api.HandleFunc("/analyze", s.handleAnalyze).Methods("POST", "OPTIONS")
    api.HandleFunc("/squash", s.handleSquash).Methods("POST", "OPTIONS")
    api.HandleFunc("/info", s.handleInfo).Methods("GET")

    // GitHub integration endpoints
    if s.webhookHandler != nil {
        github := s.router.PathPrefix("/github").Subrouter()
        github.HandleFunc("/webhook", s.webhookHandler.HandleWebhook).Methods("POST")
        log.Println("✓ GitHub webhook endpoint registered at /github/webhook")
    }

    if s.oauthHandler != nil {
        github := s.router.PathPrefix("/github").Subrouter()
        github.HandleFunc("/login", s.handleGitHubLogin).Methods("GET")
        github.HandleFunc("/callback", s.oauthHandler.HandleCallback).Methods("GET")
        log.Println("✓ GitHub OAuth endpoints registered at /github/login and /github/callback")
    }

    // Setup CORS
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
    json.NewEncoder(w).Encode(map[string]interface{}{
        "status":    "healthy",
        "timestamp": time.Now().Unix(),
        "service":   "pg-squash-api",
        "version":   "1.0.0",
    })
}

func (s *Server) handleInfo(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "service":        "pg-squash-api",
        "version":        "1.0.0",
        "capabilities":   []string{"analyze", "squash"},
        "safety_levels":  []string{"paranoid", "conservative", "standard", "aggressive"},
        "max_file_size":  "100MB",
        "max_files":      1000,
        "supported_formats": []string{"sql"},
    })
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
        s.sendError(w, "Too many files. Maximum 1000 files allowed", "TOO_MANY_FILES", http.StatusBadRequest)
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
        defer file.Close()

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
    engine := squasher.NewEngine(cfg)
    consolidatedSQL, warnings, err := engine.Squash(migrationMap)
    if err != nil {
        log.Printf("Analysis error: %v", err)
        s.sendError(w, "Analysis failed", "ANALYSIS_ERROR", http.StatusInternalServerError)
        return
    }

    // Calculate metrics
    originalLines := 0
    for _, content := range migrationMap {
        originalLines += strings.Count(content, "\n")
    }
    optimizedLines := strings.Count(consolidatedSQL, "\n")

    // Build response
    processingTime := time.Since(startTime).Milliseconds()

    response := AnalysisResponse{
        OriginalCount:        len(migrationMap),
        OptimizedCount:       optimizedLines,
        EstimatedTimeSavings: fmt.Sprintf("~%d lines reduced", originalLines-optimizedLines),
        SafetyLevel:          safetyLevel,
        Operations:           map[string]int{"analyzed": len(migrationMap)},
        Warnings:             warnings,
        Recommendations:      []string{"Review consolidation results before applying"},
        ProcessingTimeMs:     processingTime,
        FileSizeReduction:    fmt.Sprintf("%.1f%%", (1.0-float64(optimizedLines)/float64(originalLines))*100),
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
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
        defer file.Close()

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
    engine := squasher.NewEngine(cfg)

    consolidatedSQL, _, err := engine.Squash(migrationMap)
    if err != nil {
        log.Printf("Squash error: %v", err)
        s.sendError(w, "Squash operation failed", "SQUASH_ERROR", http.StatusInternalServerError)
        return
    }

    // Return the consolidated SQL
    w.Header().Set("Content-Type", "application/sql")
    w.Header().Set("Content-Disposition", "attachment; filename=\"consolidated_migration.sql\"")
    w.Write([]byte(consolidatedSQL))
}

func (s *Server) sendError(w http.ResponseWriter, message, code string, status int) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(ErrorResponse{
        Error: message,
        Code:  code,
    })
}

func (s *Server) Start() error {
    log.Printf("Starting pg-squash API server on port %s", s.port)

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

        log.Println("Shutting down server...")
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()

        if err := srv.Shutdown(ctx); err != nil {
            log.Printf("Server shutdown error: %v", err)
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
        state = "random-state-string" // In production, generate secure random state
    }

    authURL := s.oauthHandler.GetAuthorizationURL(state)
    http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}

func main() {
    log.Println("Starting pg-squash API server with GitHub integration...")

    server := NewServer()

    if err := server.Start(); err != nil && err != http.ErrServerClosed {
        log.Fatalf("Server failed to start: %v", err)
    }

    log.Println("Server stopped")
}