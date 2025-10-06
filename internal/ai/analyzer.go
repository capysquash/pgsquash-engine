package ai

import (
	"context"
	"fmt"
	"strings"

	"github.com/capysquash/pg-squash-engine/internal/ai/providers"
)

// Analyzer provides AI-powered analysis using the modular provider system
type Analyzer struct {
	manager *ProviderManager
}

// NewAnalyzer creates a new analyzer with the modular provider system
func NewAnalyzer() (*Analyzer, error) {
	manager, err := NewProviderManager(nil) // Use default config
	if err != nil {
		return nil, fmt.Errorf("failed to create provider manager: %w", err)
	}

	return &Analyzer{
		manager: manager,
	}, nil
}

// AreFunctionsSemanticallyEquivalent checks if two functions are semantically equivalent
func (a *Analyzer) AreFunctionsSemanticallyEquivalent(func1, func2 string) (bool, error) {
	if a.manager == nil {
		return false, fmt.Errorf("AI analyzer not initialized - no providers available")
	}

	content := func1 + "|||" + func2 // Use separator for function pairs

	req := &AnalysisRequest{
		Type:        providers.AnalysisFunctionEquivalence,
		Content:     content,
		Temperature: 0.1, // Low temperature for consistent results
		MaxTokens:   1000,
	}

	response, err := a.manager.Analyze(context.Background(), req)
	if err != nil {
		return false, err
	}

	result := strings.TrimSpace(strings.ToLower(response.Result))
	return result == "true", nil
}

// IsDeadCode determines if a function is unused/dead code
func (a *Analyzer) IsDeadCode(schema, functionName string) (bool, error) {
	if a.manager == nil {
		return false, fmt.Errorf("AI analyzer not initialized - no providers available")
	}

	req := &AnalysisRequest{
		Type:        providers.AnalysisDeadCode,
		Content:     fmt.Sprintf("Function: %s\n\nSchema:\n%s", functionName, schema),
		Temperature: 0.1,
		MaxTokens:   1000,
	}

	response, err := a.manager.Analyze(context.Background(), req)
	if err != nil {
		return false, err
	}

	result := strings.TrimSpace(strings.ToLower(response.Result))
	return result == "true", nil
}

// AnalyzeFunctionComplexity analyzes the complexity of a PostgreSQL function
func (a *Analyzer) AnalyzeFunctionComplexity(functionSQL string) (string, error) {
	if a.manager == nil {
		return "", fmt.Errorf("AI analyzer not initialized - no providers available")
	}

	req := &AnalysisRequest{
		Type:        providers.AnalysisFunctionComplexity,
		Content:     functionSQL,
		Temperature: 0.1,
		MaxTokens:   2000,
	}

	response, err := a.manager.Analyze(context.Background(), req)
	if err != nil {
		return "", err
	}

	return response.Result, nil
}

// DetectAuthPatterns identifies authentication/authorization patterns in SQL
func (a *Analyzer) DetectAuthPatterns(sqlContent string) ([]string, error) {
	if a.manager == nil {
		return nil, fmt.Errorf("AI analyzer not initialized - no providers available")
	}

	req := &AnalysisRequest{
		Type:        providers.AnalysisAuthPatterns,
		Content:     sqlContent,
		Temperature: 0.1,
		MaxTokens:   2000,
	}

	response, err := a.manager.Analyze(context.Background(), req)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(strings.ToUpper(response.Result)) == "NONE" {
		return []string{}, nil
	}

	lines := strings.Split(response.Result, "\n")
	var patterns []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			patterns = append(patterns, line)
		}
	}

	return patterns, nil
}

// SuggestOptimizations suggests performance and maintainability improvements
func (a *Analyzer) SuggestOptimizations(migrationSQL string) ([]string, error) {
	if a.manager == nil {
		return nil, fmt.Errorf("AI analyzer not initialized - no providers available")
	}

	req := &AnalysisRequest{
		Type:        providers.AnalysisOptimizations,
		Content:     migrationSQL,
		Temperature: 0.1,
		MaxTokens:   2000,
	}

	response, err := a.manager.Analyze(context.Background(), req)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(strings.ToUpper(response.Result)) == "NONE" {
		return []string{}, nil
	}

	lines := strings.Split(response.Result, "\n")
	var suggestions []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			suggestions = append(suggestions, line)
		}
	}

	return suggestions, nil
}

// AnalyzeCodeCoverage analyzes function usage and determines if code is dead
func (a *Analyzer) AnalyzeCodeCoverage(functionSQL, usageContext string) (string, error) {
	if a.manager == nil {
		return "", fmt.Errorf("AI analyzer not initialized - no providers available")
	}

	req := &AnalysisRequest{
		Type:        providers.AnalysisCodeCoverage,
		Content:     functionSQL,
		Context:     usageContext,
		Temperature: 0.1,
		MaxTokens:   1500,
	}

	response, err := a.manager.Analyze(context.Background(), req)
	if err != nil {
		return "", err
	}

	return response.Result, nil
}

// ValidateSchemaConsistency compares original vs squashed schemas
func (a *Analyzer) ValidateSchemaConsistency(originalSchema, squashedSchema string) ([]string, error) {
	if a.manager == nil {
		return nil, fmt.Errorf("AI analyzer not initialized - no providers available")
	}

	content := fmt.Sprintf("ORIGINAL SCHEMA:\n%s\n\nSQUASHED SCHEMA:\n%s", originalSchema, squashedSchema)

	req := &AnalysisRequest{
		Type:        providers.AnalysisSchemaConsistency,
		Content:     content,
		Temperature: 0.1,
		MaxTokens:   3000,
	}

	response, err := a.manager.Analyze(context.Background(), req)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(strings.ToUpper(response.Result)) == "IDENTICAL" {
		return []string{}, nil
	}

	lines := strings.Split(response.Result, "\n")
	var differences []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			differences = append(differences, line)
		}
	}

	return differences, nil
}

// AnalyzeSQLComplexity analyzes the complexity of SQL statements
func (a *Analyzer) AnalyzeSQLComplexity(sqlStatement string) (string, error) {
	if a.manager == nil {
		return "", fmt.Errorf("AI analyzer not initialized - no providers available")
	}

	req := &AnalysisRequest{
		Type:        providers.AnalysisSQLComplexity,
		Content:     sqlStatement,
		Temperature: 0.1,
		MaxTokens:   2000,
	}

	response, err := a.manager.Analyze(context.Background(), req)
	if err != nil {
		return "", err
	}

	return response.Result, nil
}

// AnalyzeWithTools performs analysis using tool-enhanced capabilities
func (a *Analyzer) AnalyzeWithTools(req *AnalysisRequest, tools []*ToolDefinition) (*AnalysisResponse, error) {
	if a.manager == nil {
		return nil, fmt.Errorf("AI analyzer not initialized - no providers available")
	}

	return a.manager.AnalyzeWithTools(context.Background(), req, tools)
}

// SubmitBatch submits a batch of analysis requests
func (a *Analyzer) SubmitBatch(batch *BatchRequest) (*BatchResponse, error) {
	if a.manager == nil {
		return nil, fmt.Errorf("AI analyzer not initialized - no providers available")
	}

	return a.manager.SubmitBatch(context.Background(), batch)
}

// GetBatchStatus retrieves the status of a batch analysis
func (a *Analyzer) GetBatchStatus(batchID string) (*BatchResponse, error) {
	if a.manager == nil {
		return nil, fmt.Errorf("AI analyzer not initialized - no providers available")
	}

	return a.manager.GetBatchStatus(context.Background(), batchID)
}

// HealthCheck performs health check on all providers
func (a *Analyzer) HealthCheck() map[ProviderType]error {
	if a.manager == nil {
		return map[ProviderType]error{
			"unknown": fmt.Errorf("AI analyzer not initialized"),
		}
	}

	return a.manager.HealthCheck(context.Background())
}

// GetCapabilities returns capabilities of all available providers
func (a *Analyzer) GetCapabilities() map[ProviderType]providers.ProviderCapabilities {
	if a.manager == nil {
		return make(map[ProviderType]providers.ProviderCapabilities)
	}

	return a.manager.GetCapabilities()
}

// GetAvailableProviders returns list of available providers
func (a *Analyzer) GetAvailableProviders() []ProviderType {
	if a.manager == nil {
		return []ProviderType{}
	}

	return a.manager.ListProviders()
}

// GetProvider returns a specific provider for advanced usage
func (a *Analyzer) GetProvider(providerType ProviderType) (Provider, error) {
	if a.manager == nil {
		return nil, fmt.Errorf("AI analyzer not initialized")
	}

	return a.manager.GetProvider(providerType)
}
