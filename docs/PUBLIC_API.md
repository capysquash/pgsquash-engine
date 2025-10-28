# pgsquash-engine Public API Documentation

## Overview

This document describes the public API surface of `pgsquash-engine` available through the `pkg/` packages. These packages are designed for building external tools, API servers, and integrations.

**Version**: 1.0.0
**Last Updated**: October 29, 2025
**Status**: Stable

---

## Package Index

| Package | Purpose | Status |
|---------|---------|--------|
| [pkg/engine](#pkgengine) | Core migration squashing and analysis | ✅ Stable |
| [pkg/plugins](#pkgplugins) | Plugin registration and detection | ✅ Stable |
| [pkg/github](#pkggithub) | GitHub App and OAuth integration | 🆕 New |
| [pkg/ai](#pkgai) | AI-powered analysis with multi-provider support | 🆕 New |
| [pkg/rules](#pkgrules) | Consolidation rule management | 🆕 New |

---

## pkg/engine

**Import**: `github.com/CAPYSQUASH/pgsquash-engine/pkg/engine`

Core API for migration squashing and analysis.

### Types

#### Config

```go
type Config struct {
    SafetyLevel      SafetyLevel
    OutputFormat     OutputFormat
    SeparateDataOps  bool
    EnableStreaming  bool
    MemoryLimitMB    int
    BatchSize        int
    WorkerCount      int
    ProgressCallback ProgressCallback
    EnableBackup     bool
    BackupPath       string
    EnableRollback   bool
    // ... additional fields
}
```

**Configuration for squashing operations.**

#### SafetyLevel

```go
type SafetyLevel string

const (
    Conservative SafetyLevel = "conservative" // Minimal consolidation
    Standard     SafetyLevel = "standard"     // Balanced (recommended)
    Aggressive   SafetyLevel = "aggressive"   // Maximum consolidation
    Paranoid     SafetyLevel = "paranoid"     // Preserve everything
)
```

#### SquashResult

```go
type SquashResult struct {
    SQL                 string                 // Consolidated SQL
    BaselineSQL         string                 // DDL-only SQL
    DataOperationsSQL   string                 // Data operations SQL
    Warnings            []string               // Warnings generated
    FilesProcessed      int                    // Number of files processed
    ObjectsConsolidated int                    // Objects consolidated
    ProcessingTime      string                 // Duration
    Extensions          []string               // Required extensions
    ProvenanceInfo      *ProvenanceInfo        // Metadata
    DetailedMetrics     *DetailedMetrics       // Comprehensive metrics
    RecommendedActions  []RecommendedAction    // Suggested next steps
}
```

#### AnalysisResult

```go
type AnalysisResult struct {
    TotalFiles      int
    TotalStatements int
    TotalObjects    int
    Redundancies    []Redundancy
    ObjectsByType   map[string]int
    Warnings        []string
}
```

### Functions

#### DefaultConfig

```go
func DefaultConfig() *Config
```

Returns configuration with sensible defaults.

**Example**:
```go
config := engine.DefaultConfig()
config.SafetyLevel = engine.Standard
```

#### SquashDirectory

```go
func SquashDirectory(directory string, config *Config) (*SquashResult, error)
```

Consolidates all migration files in a directory.

**Parameters**:
- `directory`: Path to migrations directory
- `config`: Configuration (nil uses defaults)

**Returns**: Squashed result with consolidated SQL

**Example**:
```go
result, err := engine.SquashDirectory("./migrations", nil)
if err != nil {
    log.Fatal(err)
}
fmt.Println(result.SQL)
```

#### SquashFiles

```go
func SquashFiles(migrations map[int]string, config *Config) (*SquashResult, error)
```

Consolidates specific migration files.

**Parameters**:
- `migrations`: Map of file paths or SQL content
- `config`: Configuration (nil uses defaults)

**Example**:
```go
migrations := map[int]string{
    1: "001_create_users.sql",
    2: "002_add_email.sql",
}
result, err := engine.SquashFiles(migrations, nil)
```

#### AnalyzeDirectory

```go
func AnalyzeDirectory(directory string, config *Config) (*AnalysisResult, error)
```

Analyzes migrations without making changes.

**Example**:
```go
analysis, err := engine.AnalyzeDirectory("./migrations", nil)
fmt.Printf("Found %d redundancies\n", len(analysis.Redundancies))
```

---

## pkg/plugins

**Import**: `github.com/CAPYSQUASH/pgsquash-engine/pkg/plugins`

Plugin registration and detection for ORM/platform integrations.

### Available Plugins

| Plugin | Description | Provider |
|--------|-------------|----------|
| **Supabase** | RLS policy optimization, auth schema handling, storage integration | supabase |
| **Clerk** | JWT v2 support, organization handling | clerk |
| **Prisma** | Migration table handling, shadow database optimizations | prisma |
| **Drizzle** | Identity column preference, sequence optimization | drizzle |

### Types

#### PluginInfo

```go
type PluginInfo struct {
    Name        string   `json:"name"`
    Version     string   `json:"version"`
    Description string   `json:"description"`
    Provider    string   `json:"provider"`
    Detected    bool     `json:"detected"`
    Patterns    []string `json:"patterns"`
}
```

#### DetectionResult

```go
type DetectionResult struct {
    Detected []PluginInfo          `json:"detected"`
    Count    int                   `json:"count"`
    Details  map[string][]string   `json:"details"`
}
```

#### CompatibilityMatrix

```go
type CompatibilityMatrix struct {
    Compatible   []string          `json:"compatible"`
    Incompatible []string          `json:"incompatible"`
    Warnings     []string          `json:"warnings"`
    Details      map[string]string `json:"details"`
}
```

### Functions

#### RegisterDefault

```go
func RegisterDefault() error
```

Registers all built-in plugins. Call during initialization.

**Example**:
```go
if err := plugins.RegisterDefault(); err != nil {
    log.Fatal(err)
}
```

#### GetAvailablePlugins

```go
func GetAvailablePlugins() []PluginInfo
```

Returns information about all available plugins.

**Example**:
```go
allPlugins := plugins.GetAvailablePlugins()
for _, p := range allPlugins {
    fmt.Printf("%s v%s: %s\n", p.Name, p.Version, p.Description)
}
```

#### DetectPlugins

```go
func DetectPlugins(ctx context.Context, migrations []string) (*DetectionResult, error)
```

Detects which plugins are applicable to migrations.

**Example**:
```go
result, err := plugins.DetectPlugins(ctx, migrations)
for _, plugin := range result.Detected {
    fmt.Printf("Detected: %s\n", plugin.Name)
}
```

#### CheckCompatibility

```go
func CheckCompatibility(pluginNames []string) (*CompatibilityMatrix, error)
```

Checks compatibility between detected plugins.

**Example**:
```go
matrix, err := plugins.CheckCompatibility([]string{"supabase", "clerk"})
for _, warning := range matrix.Warnings {
    log.Printf("Warning: %s\n", warning)
}
```

---

## pkg/github

**Import**: `github.com/CAPYSQUASH/pgsquash-engine/pkg/github`

GitHub App and OAuth integration for automated migration analysis.

### Types

#### AppClient

```go
type AppClient struct {
    // Wraps GitHub App authentication
}
```

**Methods**:
- `GetInstallationClient(ctx, installationID) (*InstallationClient, error)`
- `GetInstallationForRepo(ctx, owner, repo) (int64, error)`
- `GetInstallationClientForRepo(ctx, owner, repo) (*InstallationClient, error)`
- `ListInstallations(ctx) ([]*Installation, error)`

#### InstallationClient

```go
type InstallationClient struct {
    // Represents GitHub App installation
}
```

**Methods**:
- `CreateCheckRun(ctx, owner, repo, checkRun) (*CheckRun, error)`
- `UpdateCheckRun(ctx, owner, repo, checkRunID, updates) (*CheckRun, error)`
- `CreateIssueComment(ctx, owner, repo, issueNumber, comment) error`
- `GetPullRequest(ctx, owner, repo, number) (*PullRequest, error)`

#### WebhookHandler

```go
type WebhookHandler struct {
    // Processes GitHub webhook events
}
```

**Methods**:
- `HandleWebhook(w http.ResponseWriter, r *http.Request) error`

#### OAuthHandler

```go
type OAuthHandler struct {
    // Manages OAuth authentication flow
}
```

**Methods**:
- `GetAuthorizationURL(state) string`
- `ExchangeCodeForToken(ctx, code) (string, error)`
- `HandleCallback(w http.ResponseWriter, r *http.Request)`

### Functions

#### NewAppClientFromEnv

```go
func NewAppClientFromEnv() (*AppClient, error)
```

Creates GitHub App client from environment variables:
- `GITHUB_APP_ID`
- `GITHUB_APP_PRIVATE_KEY` or `GITHUB_APP_PRIVATE_KEY_PATH`

**Example**:
```go
appClient, err := github.NewAppClientFromEnv()
if err != nil {
    log.Fatal(err)
}

installClient, err := appClient.GetInstallationClientForRepo(ctx, "owner", "repo")
```

#### NewWebhookHandlerWithApp

```go
func NewWebhookHandlerWithApp(secret string, appClient *AppClient, engine *squasher.Engine) *WebhookHandler
```

Creates webhook handler with GitHub App authentication.

**Example**:
```go
handler := github.NewWebhookHandlerWithApp(webhookSecret, appClient, engine)
http.HandleFunc("/github/webhook", func(w http.ResponseWriter, r *http.Request) {
    handler.HandleWebhook(w, r)
})
```

#### NewOAuthHandlerFromEnv

```go
func NewOAuthHandlerFromEnv() (*OAuthHandler, error)
```

Creates OAuth handler from environment variables:
- `GITHUB_CLIENT_ID`
- `GITHUB_CLIENT_SECRET`
- `GITHUB_REDIRECT_URL`

**Example**:
```go
oauth, err := github.NewOAuthHandlerFromEnv()
authURL := oauth.GetAuthorizationURL(state)
```

---

## pkg/ai

**Import**: `github.com/CAPYSQUASH/pgsquash-engine/pkg/ai`

AI-powered migration analysis with multi-provider support and automatic failover.

### Provider Priority

1. **Azure OpenAI** (default) - Enterprise-grade, recommended for production
2. **Anthropic Claude** (fallback) - High-quality analysis
3. **OpenAI GPT** (last resort) - Widely available

### Types

#### ProviderManager

```go
type ProviderManager struct {
    // Manages multiple AI providers
}
```

**Methods**:
- `Analyze(ctx, req) (*AnalysisResponse, error)`
- `AnalyzeWithTools(ctx, req, tools) (*AnalysisResponse, error)`
- `GetProvider(providerType) (Provider, error)`
- `ListProviders() []ProviderType`
- `HealthCheck(ctx) map[ProviderType]error`
- `GetCapabilities() map[ProviderType]ProviderCapabilities`

#### ManagerConfig

```go
type ManagerConfig struct {
    DefaultProvider  ProviderType
    FallbackProvider ProviderType
    EnableFallback   bool
    ProviderConfigs  map[ProviderType]*ProviderConfig
    RetrySettings    RetryConfig
}
```

#### AnalysisRequest

```go
type AnalysisRequest struct {
    Type        AnalysisType
    Context     map[string]interface{}
    Migrations  []string
    Options     map[string]interface{}
}
```

#### AnalysisResponse

```go
type AnalysisResponse struct {
    Analysis   string
    Confidence float64
    Metadata   map[string]interface{}
}
```

### Constants

#### ProviderType

```go
const (
    ProviderAzureOpenAI ProviderType = "azure-openai"
    ProviderClaude      ProviderType = "claude"
    ProviderOpenAI      ProviderType = "openai"
)
```

#### AnalysisType

```go
const (
    AnalysisTypeOptimization AnalysisType = "optimization"
    AnalysisTypeSemantics    AnalysisType = "semantics"
    AnalysisTypeDeadCode     AnalysisType = "dead_code"
    AnalysisTypeAuthPatterns AnalysisType = "auth_patterns"
    AnalysisTypeValidation   AnalysisType = "validation"
    AnalysisTypeRepair       AnalysisType = "repair"
)
```

### Functions

#### NewProviderManager

```go
func NewProviderManager(config *ManagerConfig) (*ProviderManager, error)
```

Creates AI provider manager. If config is nil, uses defaults with auto-detection.

**Example**:
```go
manager, err := ai.NewProviderManager(nil)
if err != nil {
    log.Fatal(err)
}

req := &ai.AnalysisRequest{
    Type:       ai.AnalysisTypeOptimization,
    Migrations: []string{migrationSQL},
}

resp, err := manager.Analyze(ctx, req)
fmt.Printf("Analysis: %s (confidence: %.2f)\n", resp.Analysis, resp.Confidence)
```

### Environment Variables

**Azure OpenAI**:
- `AZURE_OPENAI_ENDPOINT`: Endpoint URL (e.g., `https://myroomieai-eu.openai.azure.com/`)
- `AZURE_OPENAI_DEPLOYMENT`: Deployment name (e.g., `gpt-4`)
- `AZURE_OPENAI_API_KEY`: API key (or use Azure AD)

**Anthropic Claude**:
- `ANTHROPIC_API_KEY`: API key

**OpenAI**:
- `OPENAI_API_KEY`: API key

---

## pkg/rules

**Import**: `github.com/CAPYSQUASH/pgsquash-engine/pkg/rules`

Consolidation rule management for dynamic rule configuration.

### Types

#### RuleRegistry

```go
type RuleRegistry struct {
    // Manages consolidation rules
}
```

**Methods**:
- `GetAllRules() []*RegisteredRule`
- `GetEnabledRules() []*RegisteredRule`
- `GetRulesByCategory(category) []*RegisteredRule`
- `EnableRule(name) error`
- `DisableRule(name) error`
- `GetStats() RegistryStats`

#### RegisteredRule

```go
type RegisteredRule struct {
    Rule     ConsolidationRule
    Metadata RuleMetadata
}
```

#### RuleMetadata

```go
type RuleMetadata struct {
    Name        string
    Description string
    Category    RuleCategory
    Priority    int
    Provider    string
    Tags        []string
    Enabled     bool
    Version     string
}
```

#### RegistryStats

```go
type RegistryStats struct {
    TotalRules      int
    EnabledRules    int
    DisabledRules   int
    RulesByCategory map[RuleCategory]int
    RulesByProvider map[string]int
}
```

### Constants

#### RuleCategory

```go
const (
    CategoryTableOps     RuleCategory = "table_operations"
    CategoryIndexOps     RuleCategory = "index_operations"
    CategoryFunctionOps  RuleCategory = "function_operations"
    CategoryDeadCode     RuleCategory = "dead_code"
    CategorySecurity     RuleCategory = "security"
    CategoryOptimization RuleCategory = "optimization"
    CategoryExtension    RuleCategory = "extension"
    CategoryPluginAuth   RuleCategory = "plugin_auth"
    CategoryPluginORM    RuleCategory = "plugin_orm"
)
```

### Functions

#### GetRegistry

```go
func GetRegistry() *RuleRegistry
```

Returns the global rule registry singleton.

**Example**:
```go
registry := rules.GetRegistry()

// List all rules
allRules := registry.GetAllRules()
for _, rule := range allRules {
    fmt.Printf("%s: %s\n", rule.Metadata.Name, rule.Metadata.Description)
}

// Enable/disable rules
registry.EnableRule("create_alter_consolidation")
registry.DisableRule("dead_code_removal")

// Get statistics
stats := registry.GetStats()
fmt.Printf("Enabled: %d/%d rules\n", stats.EnabledRules, stats.TotalRules)
```

### Core Rules

| Name | Description | Category | Priority | Default |
|------|-------------|----------|----------|---------|
| `create_alter_consolidation` | Consolidates CREATE + ALTER | table_operations | 100 | ✅ Enabled |
| `drop_create_sequence` | Removes DROP-CREATE cycles | table_operations | 95 | ✅ Enabled |
| `enum_deduplication` | Removes duplicate ENUMs | table_operations | 75 | ✅ Enabled |
| `column_evolution` | Consolidates column mods | table_operations | 65 | ✅ Enabled |
| `dead_code_removal` | Removes dead code | dead_code | 50 | ❌ Disabled |

---

## Integration Examples

### Building an API Server

```go
package main

import (
    "github.com/CAPYSQUASH/pgsquash-engine/pkg/engine"
    "github.com/CAPYSQUASH/pgsquash-engine/pkg/github"
    "github.com/CAPYSQUASH/pgsquash-engine/pkg/ai"
    "github.com/CAPYSQUASH/pgsquash-engine/pkg/rules"
    "github.com/CAPYSQUASH/pgsquash-engine/pkg/plugins"
)

func main() {
    // Register plugins
    plugins.RegisterDefault()

    // Initialize GitHub integration
    githubApp, _ := github.NewAppClientFromEnv()

    // Initialize AI manager
    aiManager, _ := ai.NewProviderManager(nil)

    // Setup rule registry
    registry := rules.GetRegistry()

    // Create API server handlers
    http.HandleFunc("/api/squash", handleSquash)
    http.HandleFunc("/api/analyze", handleAnalyze)
    http.HandleFunc("/github/webhook", handleGitHubWebhook)
    http.HandleFunc("/api/rules", handleRules)

    http.ListenAndServe(":8080", nil)
}
```

### Analyzing Migrations

```go
// Analyze migrations
result, err := engine.AnalyzeDirectory("./migrations", nil)

// Detect plugins
detection, err := plugins.DetectPlugins(ctx, migrations)

// Run AI analysis
aiResp, err := aiManager.Analyze(ctx, &ai.AnalysisRequest{
    Type:       ai.AnalysisTypeOptimization,
    Migrations: migrations,
})

// Get rule statistics
stats := rules.GetRegistry().GetStats()
```

---

## Migration Guide

### From Internal Usage

If you were using internal packages directly, migrate to the public API:

**Before** (internal):
```go
import "github.com/CAPYSQUASH/pgsquash-engine/internal/github"
```

**After** (public):
```go
import "github.com/CAPYSQUASH/pgsquash-engine/pkg/github"
```

All type names and function signatures remain the same.

### Version Compatibility

- ✅ All public API is stable and backwards compatible
- ✅ Type aliases ensure no breaking changes
- ✅ Semantic versioning follows Go module conventions

---

## Support

- **Issues**: https://github.com/CAPYSQUASH/pgsquash-engine/issues
- **Documentation**: https://capysquash.dev
- **Email**: support@capysquash.dev

---

**Generated**: October 29, 2025
**API Version**: 1.0.0
**Engine Version**: 0.9.5
