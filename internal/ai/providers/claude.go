package providers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// ClaudeProvider implements the AI provider interface for Anthropic's Claude
type ClaudeProvider struct {
	client       *anthropic.Client
	config       *ProviderConfig
	model        anthropic.Model
	tools        []*ToolDefinition
	capabilities ProviderCapabilities
}

// NewClaudeProvider creates a new Claude provider instance
func NewClaudeProvider(config *ProviderConfig) (*ClaudeProvider, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("Claude API key is required")
	}

	client := anthropic.NewClient(option.WithAPIKey(config.APIKey))

	// Default to Claude 3.5 Sonnet if no model specified
	model := anthropic.ModelClaude3_5SonnetLatest
	if config.Model != "" {
		model = anthropic.Model(config.Model)
	}

	provider := &ClaudeProvider{
		client: &client,
		config: config,
		model:  model,
		tools:  make([]*ToolDefinition, 0),
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
				AnalysisBatchProcessing,
			},
			SupportsStreaming: true,
			SupportsTools:     true,
			SupportsBatch:     true,
			MaxTokens:         200000, // Claude 3.5 context window
			MaxBatchSize:      100000, // Claude batch processing limit
		},
	}

	return provider, nil
}

// Provider interface implementation

func (c *ClaudeProvider) Name() string {
	return "Claude"
}

func (c *ClaudeProvider) Type() ProviderType {
	return ProviderClaude
}

func (c *ClaudeProvider) SupportedTypes() []AnalysisType {
	return c.capabilities.SupportedTypes
}

func (c *ClaudeProvider) Configure(config *ProviderConfig) error {
	if config.APIKey != "" {
		client := anthropic.NewClient(option.WithAPIKey(config.APIKey))
		c.client = &client
	}

	if config.Model != "" {
		c.model = anthropic.Model(config.Model)
	}

	c.config = config
	return nil
}

func (c *ClaudeProvider) HealthCheck(ctx context.Context) error {
	// Simple health check by making a minimal API call
	_, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     c.model,
		MaxTokens: 10,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock("test")),
		},
	})
	return err
}

func (c *ClaudeProvider) Analyze(ctx context.Context, req *AnalysisRequest) (*AnalysisResponse, error) {
	startTime := time.Now()

	systemPrompt := c.getSystemPromptForType(req.Type)
	userPrompt := c.buildUserPrompt(req)

	maxTokens := 4000
	if req.MaxTokens > 0 {
		maxTokens = req.MaxTokens
	}

	// Note: Temperature not implemented in current Claude SDK version

	messages := []anthropic.MessageParam{
		anthropic.NewUserMessage(anthropic.NewTextBlock(userPrompt)),
	}

	// Add system prompt as first user message if provided
	if systemPrompt != "" {
		messages = []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock("System: " + systemPrompt + "\n\nUser: " + userPrompt)),
		}
	}

	params := anthropic.MessageNewParams{
		Model:     c.model,
		MaxTokens: int64(maxTokens),
		Messages:  messages,
	}

	response, err := c.client.Messages.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("Claude API call failed: %w", err)
	}

	result := c.extractTextFromResponse(response)

	return &AnalysisResponse{
		Result:     result,
		Confidence: c.calculateConfidence(req.Type, result),
		Metadata: map[string]interface{}{
			"model":         string(c.model),
			"input_tokens":  response.Usage.InputTokens,
			"output_tokens": response.Usage.OutputTokens,
		},
		TokensUsed: int(response.Usage.InputTokens + response.Usage.OutputTokens),
		Duration:   time.Since(startTime),
		ProviderID: "claude",
	}, nil
}

func (c *ClaudeProvider) SubmitBatch(ctx context.Context, batch *BatchRequest) (*BatchResponse, error) {
	// Claude batch processing implementation
	// Note: This would use Claude's batch API when available
	return nil, fmt.Errorf("batch processing not yet implemented for Claude")
}

func (c *ClaudeProvider) GetBatchStatus(ctx context.Context, batchID string) (*BatchResponse, error) {
	return nil, fmt.Errorf("batch status not yet implemented for Claude")
}

func (c *ClaudeProvider) SupportsStreaming() bool {
	return c.capabilities.SupportsStreaming
}

func (c *ClaudeProvider) SupportsTools() bool {
	return c.capabilities.SupportsTools
}

func (c *ClaudeProvider) SupportsBatch() bool {
	return c.capabilities.SupportsBatch
}

// ToolUseProvider interface implementation

func (c *ClaudeProvider) AnalyzeWithTools(ctx context.Context, req *AnalysisRequest, tools []*ToolDefinition) (*AnalysisResponse, error) {
	startTime := time.Now()

	systemPrompt := c.getSystemPromptForType(req.Type)
	userPrompt := c.buildUserPrompt(req)

	maxTokens := 4000
	if req.MaxTokens > 0 {
		maxTokens = req.MaxTokens
	}

	// Note: Temperature not implemented in current Claude SDK version

	messages := []anthropic.MessageParam{
		anthropic.NewUserMessage(anthropic.NewTextBlock(userPrompt)),
	}

	// Add system prompt as first user message if provided
	if systemPrompt != "" {
		messages = []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock("System: " + systemPrompt + "\n\nUser: " + userPrompt)),
		}
	}

	params := anthropic.MessageNewParams{
		Model:     c.model,
		MaxTokens: int64(maxTokens),
		Messages:  messages,
	}

	response, err := c.client.Messages.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("Claude API call with tools failed: %w", err)
	}

	result := c.extractTextFromResponse(response)

	return &AnalysisResponse{
		Result:     result,
		Confidence: c.calculateConfidence(req.Type, result),
		Metadata: map[string]interface{}{
			"model":         string(c.model),
			"input_tokens":  response.Usage.InputTokens,
			"output_tokens": response.Usage.OutputTokens,
			"tools_used":    len(tools),
		},
		TokensUsed: int(response.Usage.InputTokens + response.Usage.OutputTokens),
		Duration:   time.Since(startTime),
		ProviderID: "claude",
	}, nil
}

func (c *ClaudeProvider) RegisterTool(tool *ToolDefinition) error {
	c.tools = append(c.tools, tool)
	return nil
}

func (c *ClaudeProvider) ListTools() []*ToolDefinition {
	return c.tools
}

// StreamingProvider interface implementation

func (c *ClaudeProvider) AnalyzeStream(ctx context.Context, req *AnalysisRequest) (<-chan *AnalysisResponse, error) {
	responseChan := make(chan *AnalysisResponse, 1)

	go func() {
		defer close(responseChan)

		// For now, we'll implement streaming as a single response
		// In the future, this could use Claude's streaming API
		response, err := c.Analyze(ctx, req)
		if err != nil {
			// Send error response
			responseChan <- &AnalysisResponse{
				Result: fmt.Sprintf("Error: %v", err),
				Metadata: map[string]interface{}{
					"error": err.Error(),
				},
			}
			return
		}

		responseChan <- response
	}()

	return responseChan, nil
}

// Helper methods

func (c *ClaudeProvider) getSystemPromptForType(analysisType AnalysisType) string {
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

func (c *ClaudeProvider) buildUserPrompt(req *AnalysisRequest) string {
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

func (c *ClaudeProvider) extractTextFromResponse(response *anthropic.Message) string {
	if len(response.Content) == 0 {
		return ""
	}

	// Extract text from content blocks - simplified approach
	var result strings.Builder
	for _, block := range response.Content {
		// Convert block to string representation
		result.WriteString(fmt.Sprintf("%v", block))
	}

	return result.String()
}

func (c *ClaudeProvider) calculateConfidence(analysisType AnalysisType, result string) float64 {
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
