package github

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/capysquash/pg-squash-engine/internal/parser"
	"github.com/capysquash/pg-squash-engine/internal/squasher"
	"github.com/capysquash/pg-squash-engine/internal/tracking"
)

// WebhookHandler handles GitHub webhook events
type WebhookHandler struct {
    secret          string
    githubClient    *Client
    analysisEngine  *squasher.Engine
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(secret string, client *Client, engine *squasher.Engine) *WebhookHandler {
    return &WebhookHandler{
        secret:         secret,
        githubClient:   client,
        analysisEngine: engine,
    }
}

// PullRequestEvent represents a GitHub pull request webhook event
type PullRequestEvent struct {
    Action      string `json:"action"`
    Number      int    `json:"number"`
    PullRequest struct {
        Number  int    `json:"number"`
        Head    Branch `json:"head"`
        Base    Branch `json:"base"`
        HTMLURL string `json:"html_url"`
    } `json:"pull_request"`
    Repository Repository `json:"repository"`
}

// IssueCommentEvent represents a GitHub issue comment webhook event
type IssueCommentEvent struct {
    Action  string `json:"action"`
    Issue   Issue  `json:"issue"`
    Comment struct {
        Body string `json:"body"`
    } `json:"comment"`
    Repository Repository `json:"repository"`
}

// Branch represents a git branch
type Branch struct {
    Ref string `json:"ref"`
    SHA string `json:"sha"`
}

// Repository represents a GitHub repository
type Repository struct {
    FullName string `json:"full_name"`
    Owner    struct {
        Login string `json:"login"`
    } `json:"owner"`
    Name string `json:"name"`
}

// Issue represents a GitHub issue/PR
type Issue struct {
    Number      int  `json:"number"`
    PullRequest *struct{} `json:"pull_request,omitempty"`
}

// HandleWebhook processes incoming GitHub webhooks
func (h *WebhookHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
    // Verify webhook signature
    if !h.verifySignature(r) {
        http.Error(w, "Invalid signature", http.StatusUnauthorized)
        return
    }

    // Parse event type
    eventType := r.Header.Get("X-GitHub-Event")

    body, err := io.ReadAll(r.Body)
    if err != nil {
        http.Error(w, "Failed to read body", http.StatusBadRequest)
        return
    }

    switch eventType {
    case "pull_request":
        h.handlePullRequest(r.Context(), body)
    case "issue_comment":
        h.handleIssueComment(r.Context(), body)
    default:
        w.WriteHeader(http.StatusOK)
        return
    }

    w.WriteHeader(http.StatusOK)
}

// handlePullRequest processes pull request events
func (h *WebhookHandler) handlePullRequest(ctx context.Context, body []byte) error {
    var event PullRequestEvent
    if err := json.Unmarshal(body, &event); err != nil {
        return fmt.Errorf("parse PR event: %w", err)
    }

    // Only process opened and synchronized events
    if event.Action != "opened" && event.Action != "synchronize" {
        return nil
    }

    // Get changed migration files
    files, err := h.githubClient.GetPRFiles(ctx, event.Repository.FullName, event.PullRequest.Number)
    if err != nil {
        return fmt.Errorf("get PR files: %w", err)
    }

    migrationFiles := h.filterMigrationFiles(files)
    if len(migrationFiles) == 0 {
        return nil // No migration files changed
    }

    // Analyze migrations
    analysis, err := h.analyzeMigrations(ctx, event.Repository.FullName, migrationFiles)
    if err != nil {
        return fmt.Errorf("analyze migrations: %w", err)
    }

    // Post analysis as PR comment
    comment := h.formatAnalysisComment(analysis, len(migrationFiles))
    if err := h.githubClient.PostPRComment(ctx, event.Repository.FullName, event.PullRequest.Number, comment); err != nil {
        return fmt.Errorf("post comment: %w", err)
    }

    // Check if auto-consolidation is enabled via repo config
    config, err := h.githubClient.GetRepoConfig(ctx, event.Repository.FullName, event.PullRequest.Head.SHA)
    if err == nil && config.AutoPR && analysis.ShouldConsolidate {
        // Create consolidation PR
        if err := h.createConsolidationPR(ctx, event, analysis); err != nil {
            return fmt.Errorf("create consolidation PR: %w", err)
        }
    }

    return nil
}

// handleIssueComment processes comment events for bot commands
func (h *WebhookHandler) handleIssueComment(ctx context.Context, body []byte) error {
    var event IssueCommentEvent
    if err := json.Unmarshal(body, &event); err != nil {
        return fmt.Errorf("parse comment event: %w", err)
    }

    // Only process comments on PRs
    if event.Issue.PullRequest == nil {
        return nil
    }

    // Check for bot commands
    command := h.parseCommand(event.Comment.Body)
    if command == "" {
        return nil
    }

    switch command {
    case "analyze":
        return h.handleAnalyzeCommand(ctx, event)
    case "consolidate":
        return h.handleConsolidateCommand(ctx, event)
    default:
        return nil
    }
}

// parseCommand extracts bot commands from comment body
func (h *WebhookHandler) parseCommand(body string) string {
    lines := strings.Split(body, "\n")
    for _, line := range lines {
        line = strings.TrimSpace(line)
        if strings.HasPrefix(line, "/pgsquash ") {
            parts := strings.Fields(line)
            if len(parts) >= 2 {
                return parts[1]
            }
        }
    }
    return ""
}

// handleAnalyzeCommand handles /pgsquash analyze command
func (h *WebhookHandler) handleAnalyzeCommand(ctx context.Context, event IssueCommentEvent) error {
    // Get PR files
    files, err := h.githubClient.GetPRFiles(ctx, event.Repository.FullName, event.Issue.Number)
    if err != nil {
        return err
    }

    migrationFiles := h.filterMigrationFiles(files)
    if len(migrationFiles) == 0 {
        h.githubClient.PostPRComment(ctx, event.Repository.FullName, event.Issue.Number,
            "No migration files found in this PR.")
        return nil
    }

    analysis, err := h.analyzeMigrations(ctx, event.Repository.FullName, migrationFiles)
    if err != nil {
        return err
    }

    comment := h.formatAnalysisComment(analysis, len(migrationFiles))
    return h.githubClient.PostPRComment(ctx, event.Repository.FullName, event.Issue.Number, comment)
}

// handleConsolidateCommand handles /pgsquash consolidate command
func (h *WebhookHandler) handleConsolidateCommand(ctx context.Context, event IssueCommentEvent) error {
    // Get PR details
    pr, err := h.githubClient.GetPullRequest(ctx, event.Repository.FullName, event.Issue.Number)
    if err != nil {
        return err
    }

    files, err := h.githubClient.GetPRFiles(ctx, event.Repository.FullName, event.Issue.Number)
    if err != nil {
        return err
    }

    migrationFiles := h.filterMigrationFiles(files)
    analysis, err := h.analyzeMigrations(ctx, event.Repository.FullName, migrationFiles)
    if err != nil {
        return err
    }

    // Create consolidation PR
    return h.createConsolidationPR(ctx, PullRequestEvent{
        Repository: event.Repository,
        PullRequest: struct {
            Number  int    `json:"number"`
            Head    Branch `json:"head"`
            Base    Branch `json:"base"`
            HTMLURL string `json:"html_url"`
        }{
            Number: pr.Number,
            Head:   pr.Head,
            Base:   pr.Base,
        },
    }, analysis)
}

// filterMigrationFiles filters for SQL migration files
func (h *WebhookHandler) filterMigrationFiles(files []PRFile) []PRFile {
    migrations := []PRFile{}
    for _, file := range files {
        if strings.HasSuffix(strings.ToLower(file.Filename), ".sql") {
            migrations = append(migrations, file)
        }
    }
    return migrations
}

// AnalysisResult contains migration analysis results
type AnalysisResult struct {
    OriginalCount      int
    OptimizedCount     int
    ConsolidationRatio float64
    ShouldConsolidate  bool
    Redundancies       []tracking.RedundancyReport
    Warnings           []string
    ConsolidatedSQL    string
}

// analyzeMigrations performs analysis on migration files
func (h *WebhookHandler) analyzeMigrations(ctx context.Context, repo string, files []PRFile) (*AnalysisResult, error) {
    // Download file contents
    migrations := make([]parser.Migration, 0, len(files))

    for i, file := range files {
        content, err := h.githubClient.GetFileContent(ctx, repo, file.Filename, file.SHA)
        if err != nil {
            return nil, fmt.Errorf("get file %s: %w", file.Filename, err)
        }

        m, err := parser.ParseMigration(content, file.Filename)
        if err != nil {
            return nil, fmt.Errorf("parse %s: %w", file.Filename, err)
        }
        m.Sequence = i + 1
        migrations = append(migrations, *m)
    }

    // Track objects and find redundancies
    tracker := tracking.NewTracker()
    for i, m := range migrations {
        tracker.ProcessMigration(&m, i)
    }

    redundancies := tracker.GetRedundantObjects()
    stats := tracker.GetStatistics()

    // Run squashing to get optimized count and SQL
    migrationMap := make(map[int]string)
    for i, file := range files {
        content, _ := h.githubClient.GetFileContent(ctx, repo, file.Filename, file.SHA)
        migrationMap[i+1] = content
    }

    consolidatedSQL, warnings, err := h.analysisEngine.Squash(migrationMap)
    if err != nil {
        return nil, fmt.Errorf("squash analysis: %w", err)
    }

    // Calculate consolidation ratio
    optimizedCount := strings.Count(consolidatedSQL, "\n")
    originalCount := stats.TotalStatements
    ratio := 0.0
    if originalCount > 0 {
        ratio = float64(optimizedCount) / float64(originalCount)
    }

    return &AnalysisResult{
        OriginalCount:      len(migrations),
        OptimizedCount:     optimizedCount,
        ConsolidationRatio: ratio,
        ShouldConsolidate:  len(migrations) >= 15 || ratio < 0.7,
        Redundancies:       redundancies,
        Warnings:           warnings,
        ConsolidatedSQL:    consolidatedSQL,
    }, nil
}

// formatAnalysisComment formats analysis results as a PR comment
func (h *WebhookHandler) formatAnalysisComment(analysis *AnalysisResult, fileCount int) string {
    var comment strings.Builder

    comment.WriteString("## 🔍 pg-squash Migration Analysis\n\n")
    comment.WriteString(fmt.Sprintf("**Migration Files:** %d\n", fileCount))
    comment.WriteString(fmt.Sprintf("**Consolidation Ratio:** %.1f%%\n\n", analysis.ConsolidationRatio*100))

    if analysis.ShouldConsolidate {
        comment.WriteString("### ✅ Consolidation Recommended\n\n")
        comment.WriteString("This PR can benefit from migration consolidation:\n")
        comment.WriteString(fmt.Sprintf("- Original statements: %d\n", analysis.OriginalCount))
        comment.WriteString(fmt.Sprintf("- After consolidation: ~%d statements\n", analysis.OptimizedCount))
        comment.WriteString(fmt.Sprintf("- Savings: ~%d statements\n\n", analysis.OriginalCount-analysis.OptimizedCount))
    } else {
        comment.WriteString("### ℹ️ Consolidation Status\n\n")
        comment.WriteString("Migrations appear well-optimized. No consolidation needed.\n\n")
    }

    if len(analysis.Redundancies) > 0 {
        comment.WriteString("### 🔄 Redundancies Found\n\n")
        for _, r := range analysis.Redundancies {
            comment.WriteString(fmt.Sprintf("- **%s** (%s): %s\n", r.Object, r.Type, r.Explanation))
        }
        comment.WriteString("\n")
    }

    if len(analysis.Warnings) > 0 {
        comment.WriteString("### ⚠️ Warnings\n\n")
        for _, w := range analysis.Warnings {
            comment.WriteString(fmt.Sprintf("- %s\n", w))
        }
        comment.WriteString("\n")
    }

    comment.WriteString("---\n")
    comment.WriteString("💡 **Commands:**\n")
    comment.WriteString("- `/pgsquash analyze` - Re-analyze migrations\n")
    comment.WriteString("- `/pgsquash consolidate` - Create consolidation PR\n")

    return comment.String()
}

// createConsolidationPR creates a new PR with consolidated migrations
func (h *WebhookHandler) createConsolidationPR(ctx context.Context, event PullRequestEvent, analysis *AnalysisResult) error {
    // Create a new branch
    baseBranch := event.PullRequest.Base.Ref
    newBranch := fmt.Sprintf("pg-squash/consolidate-pr-%d", event.PullRequest.Number)

    if err := h.githubClient.CreateBranch(ctx, event.Repository.FullName, newBranch, event.PullRequest.Head.SHA); err != nil {
        return fmt.Errorf("create branch: %w", err)
    }

    // Commit consolidated migration
    commitMsg := fmt.Sprintf("Consolidate migrations from PR #%d\n\nAutomatic consolidation by pg-squash bot", event.PullRequest.Number)
    filename := "migrations/001_consolidated.sql"

    if err := h.githubClient.CreateOrUpdateFile(ctx, event.Repository.FullName, filename, analysis.ConsolidatedSQL, commitMsg, newBranch); err != nil {
        return fmt.Errorf("commit file: %w", err)
    }

    // Create PR
    prTitle := fmt.Sprintf("🤖 Consolidated migrations from PR #%d", event.PullRequest.Number)
    prBody := fmt.Sprintf(`## Automated Migration Consolidation

This PR consolidates migrations from #%d.

### Summary
- Original migrations: %d files
- Consolidated to: 1 file
- Statement reduction: %d → %d

### Changes
- Removed redundant operations
- Optimized dependency order
- Preserved data integrity

---
*Generated by [pg-squash](https://github.com/capysquash/pg-squash)*
`, event.PullRequest.Number, analysis.OriginalCount, analysis.OriginalCount, analysis.OptimizedCount)

    prNumber, err := h.githubClient.CreatePullRequest(ctx, event.Repository.FullName, prTitle, prBody, newBranch, baseBranch)
    if err != nil {
        return fmt.Errorf("create PR: %w", err)
    }

    // Comment on original PR
    linkComment := fmt.Sprintf("✅ Created consolidation PR: #%d", prNumber)
    return h.githubClient.PostPRComment(ctx, event.Repository.FullName, event.PullRequest.Number, linkComment)
}

// verifySignature validates the webhook signature
func (h *WebhookHandler) verifySignature(r *http.Request) bool {
    signature := r.Header.Get("X-Hub-Signature-256")
    if signature == "" {
        return false
    }

    body, err := io.ReadAll(r.Body)
    if err != nil {
        return false
    }
    r.Body = io.NopCloser(strings.NewReader(string(body)))

    mac := hmac.New(sha256.New, []byte(h.secret))
    mac.Write(body)
    expectedMAC := "sha256=" + hex.EncodeToString(mac.Sum(nil))

    return hmac.Equal([]byte(signature), []byte(expectedMAC))
}
