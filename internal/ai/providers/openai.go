package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// OpenAIProvider implements the AI provider interface for OpenAI
type OpenAIProvider struct {
	client       *http.Client
	config       *ProviderConfig
	capabilities ProviderCapabilities
}

// OpenAI API structures (keeping existing ones from analyzer.go)
type OpenAIRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// NewOpenAIProvider creates a new OpenAI provider instance
func NewOpenAIProvider(config *ProviderConfig) (*OpenAIProvider, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}

	timeout := 60 * time.Second
	if config.Timeout > 0 {
		timeout = config.Timeout
	}

	model := "gpt-4"
	if config.Model != "" {
		model = config.Model
	}

	provider := &OpenAIProvider{
		client: &http.Client{Timeout: timeout},
		config: config,
		capabilities: ProviderCapabilities{
			SupportedTypes: []AnalysisType{
				AnalysisFunctionEquivalence,
				AnalysisDeadCode,
				AnalysisFunctionComplexity,
				AnalysisAuthPatterns,
				AnalysisOptimizations,
				AnalysisCodeCoverage,
				AnalysisSchemaConsistency,
				AnalysisSQLComplexity,
			},
			SupportsStreaming: false,
			SupportsTools:     true,
			SupportsBatch:     false,
			MaxTokens:         128000, // GPT-4 context window
			MaxBatchSize:      0,      // No batch support
		},
	}

	// Update config with defaults
	if provider.config.Model == "" {
		provider.config.Model = model
	}
	if provider.config.Endpoint == "" {
		provider.config.Endpoint = "https://api.openai.com/v1/chat/completions"
	}

	return provider, nil
}

// Provider interface implementation

func (o *OpenAIProvider) Name() string {
	return "OpenAI"
}

func (o *OpenAIProvider) Type() ProviderType {
	return ProviderOpenAI
}

func (o *OpenAIProvider) SupportedTypes() []AnalysisType {
	return o.capabilities.SupportedTypes
}

func (o *OpenAIProvider) Configure(config *ProviderConfig) error {
	if config.APIKey != "" {
		o.config.APIKey = config.APIKey
	}

	if config.Model != "" {
		o.config.Model = config.Model
	}

	if config.Endpoint != "" {
		o.config.Endpoint = config.Endpoint
	}

	if config.Timeout > 0 {
		o.client.Timeout = config.Timeout
	}

	return nil
}

func (o *OpenAIProvider) HealthCheck(ctx context.Context) error {
	// Simple health check by making a minimal API call
	req := &AnalysisRequest{
		Content:     "test",
		MaxTokens:   10,
		Temperature: 0.1,
	}

	_, err := o.Analyze(ctx, req)
	return err
}

func (o *OpenAIProvider) Analyze(ctx context.Context, req *AnalysisRequest) (*AnalysisResponse, error) {
	startTime := time.Now()

	systemPrompt := o.getSystemPromptForType(req.Type)
	userPrompt := o.buildUserPrompt(req)

	maxTokens := 1000
	if req.MaxTokens > 0 {
		maxTokens = req.MaxTokens
	}

	temperature := 0.1 // Default low temperature for consistent results
	if req.Temperature > 0 {
		temperature = req.Temperature
	}

	result, tokenUsage, err := o.makeAPICall(systemPrompt, userPrompt, maxTokens, temperature)
	if err != nil {
		return nil, fmt.Errorf("OpenAI API call failed: %w", err)
	}

	return &AnalysisResponse{
		Result:     result,
		Confidence: o.calculateConfidence(req.Type, result),
		Metadata: map[string]interface{}{
			"model":             o.config.Model,
			"prompt_tokens":     tokenUsage.PromptTokens,
			"completion_tokens": tokenUsage.CompletionTokens,
		},
		TokensUsed: tokenUsage.TotalTokens,
		Duration:   time.Since(startTime),
		ProviderID: "openai",
	}, nil
}

func (o *OpenAIProvider) SubmitBatch(ctx context.Context, batch *BatchRequest) (*BatchResponse, error) {
	return nil, fmt.Errorf("batch processing not supported by OpenAI provider")
}

func (o *OpenAIProvider) GetBatchStatus(ctx context.Context, batchID string) (*BatchResponse, error) {
	return nil, fmt.Errorf("batch processing not supported by OpenAI provider")
}

func (o *OpenAIProvider) SupportsStreaming() bool {
	return o.capabilities.SupportsStreaming
}

func (o *OpenAIProvider) SupportsTools() bool {
	return o.capabilities.SupportsTools
}

func (o *OpenAIProvider) SupportsBatch() bool {
	return o.capabilities.SupportsBatch
}

// Helper methods

func (o *OpenAIProvider) getSystemPromptForType(analysisType AnalysisType) string {
	basePrompt := "You are a PostgreSQL expert specializing in database optimization, migration analysis, and SQL best practices. Provide precise, accurate responses."

	switch analysisType {
	case AnalysisFunctionEquivalence:
		return basePrompt + " Focus on semantic equivalence of PostgreSQL functions, considering inputs, outputs, and side effects."
	case AnalysisDeadCode:
		return basePrompt + " Analyze PostgreSQL schemas to identify unused functions, triggers, and database objects."
	case AnalysisFunctionComplexity:
		return basePrompt + " Evaluate PostgreSQL function complexity, maintainability, and performance characteristics."
	case AnalysisAuthPatterns:
		return basePrompt + " Identify authentication and authorization patterns in PostgreSQL schemas, including RLS, JWT, and third-party integrations."
	case AnalysisOptimizations:
		return basePrompt + " Suggest performance and maintainability optimizations for PostgreSQL migrations and schemas."
	case AnalysisSQLComplexity:
		return basePrompt + " Analyze SQL statement complexity, performance implications, and best practice adherence."
	default:
		return basePrompt
	}
}

func (o *OpenAIProvider) buildUserPrompt(req *AnalysisRequest) string {
	switch req.Type {
	case AnalysisFunctionEquivalence:
		parts := strings.Split(req.Content, "|||")
		if len(parts) == 2 {
			return fmt.Sprintf(`Analyze these two PostgreSQL functions and determine if they are semantically equivalent.

Two functions are semantically equivalent if they:
1. Have the same input parameters (names can differ, but types and order must match)
2. Produce identical outputs for all possible inputs
3. Have the same side effects (or lack thereof)
4. Handle edge cases identically

Function 1:
%s

Function 2:
%s

Respond with only 'true' if they are semantically equivalent, or 'false' if they are not.`, parts[0], parts[1])
		}
		return req.Content

	case AnalysisDeadCode:
		return fmt.Sprintf(`Analyze this PostgreSQL schema and determine if the specified function is dead code.

A function is considered dead code if:
1. It's not called by any other functions in the schema
2. It's not referenced in any triggers
3. It's not used in any policies or constraints
4. It's not referenced in any views
5. It appears to be unused application entry point

Schema and Analysis Request:
%s

Context (if provided):
%s

Respond with only 'true' if the function appears to be dead code, or 'false' if it's used.`, req.Content, req.Context)

	case AnalysisAuthPatterns:
		return fmt.Sprintf(`Analyze this SQL content and identify authentication/authorization patterns.

Look for:
1. JWT token processing (auth.jwt(), claims extraction)
2. RLS (Row Level Security) policies
3. User/role-based access patterns
4. Supabase auth patterns
5. Clerk auth patterns
6. Custom authentication schemes

SQL Content:
%s

List the detected patterns, one per line, or 'NONE' if no auth patterns found.`, req.Content)

	default:
		prompt := req.Content
		if req.Context != "" {
			prompt += "\n\nContext:\n" + req.Context
		}
		return prompt
	}
}

func (o *OpenAIProvider) makeAPICall(systemPrompt, userPrompt string, maxTokens int, temperature float64) (string, Usage, error) {
	reqBody := OpenAIRequest{
		Model: o.config.Model,
		Messages: []Message{
			{
				Role:    "system",
				Content: systemPrompt,
			},
			{
				Role:    "user",
				Content: userPrompt,
			},
		},
		MaxTokens:   maxTokens,
		Temperature: temperature,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", Usage{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(context.Background(), "POST", o.config.Endpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", Usage{}, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+o.config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(req)
	if err != nil {
		return "", Usage{}, fmt.Errorf("API request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", Usage{}, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var result OpenAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", Usage{}, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Choices) == 0 {
		return "", Usage{}, fmt.Errorf("no choices in API response")
	}

	return result.Choices[0].Message.Content, result.Usage, nil
}

func (o *OpenAIProvider) calculateConfidence(analysisType AnalysisType, result string) float64 {
	// Simple confidence calculation based on result characteristics
	result = strings.TrimSpace(strings.ToLower(result))

	switch analysisType {
	case AnalysisFunctionEquivalence, AnalysisDeadCode:
		// For boolean responses, high confidence if it's a clear true/false
		if result == "true" || result == "false" {
			return 0.95
		}
		return 0.7
	default:
		// For other types, base confidence on result length and structure
		if len(result) > 50 && strings.Contains(result, "\n") {
			return 0.85
		}
		return 0.75
	}
}
