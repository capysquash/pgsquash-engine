package providers

import (
    "context"
    "fmt"
    "net/http"
    "strings"
    "time"

    "github.com/Azure/azure-sdk-for-go/sdk/azidentity"
    "github.com/openai/openai-go"
    "github.com/openai/openai-go/azure"
    "github.com/openai/openai-go/option"
)

// AzureOpenAIProvider implements the AI provider interface for Azure OpenAI
type AzureOpenAIProvider struct {
    client       openai.Client
    config       *ProviderConfig
    capabilities ProviderCapabilities
}

// NewAzureOpenAIProvider creates a new Azure OpenAI provider instance
func NewAzureOpenAIProvider(config *ProviderConfig) (*AzureOpenAIProvider, error) {
    if config.Endpoint == "" {
        return nil, fmt.Errorf("Azure OpenAI endpoint is required")
    }

    if config.AzureDeployment == "" {
        return nil, fmt.Errorf("Azure deployment name is required")
    }

    timeout := 60 * time.Second
    if config.Timeout > 0 {
        timeout = config.Timeout
    }

    // Set default API version if not specified
    if config.AzureAPIVersion == "" {
        // Use "preview" for v1 API (recommended for latest features)
        // Alternative: "2024-10-21" for latest GA version
        config.AzureAPIVersion = "preview"
    }

    // Create HTTP client with timeout
    httpClient := &http.Client{Timeout: timeout}

    // Determine if we're using v1 API (preview or no version specified)
    // v1 API uses /openai/v1/ endpoint format
    useV1API := config.AzureAPIVersion == "preview" || config.AzureAPIVersion == ""

    // Create Azure OpenAI client based on API version and authentication method
    var client openai.Client

    if useV1API {
        // v1 API: Use /openai/v1/ endpoint format
        baseURL := config.Endpoint
        if !strings.HasSuffix(baseURL, "/") {
            baseURL += "/"
        }
        baseURL += "openai/v1/"

        if config.UseAzureAD {
            // Use Azure AD authentication
            credential, err := azidentity.NewDefaultAzureCredential(nil)
            if err != nil {
                return nil, fmt.Errorf("failed to create Azure AD credential: %w", err)
            }

            client = openai.NewClient(
                option.WithBaseURL(baseURL),
                azure.WithTokenCredential(credential),
                option.WithHTTPClient(httpClient),
            )
        } else {
            // Use API key authentication
            if config.APIKey == "" {
                return nil, fmt.Errorf("Azure OpenAI API key is required when not using Azure AD")
            }

            client = openai.NewClient(
                option.WithBaseURL(baseURL),
                option.WithAPIKey(config.APIKey),
                option.WithHTTPClient(httpClient),
            )
        }
    } else {
        // Legacy API: Use Azure-specific endpoint format with api-version
        if config.UseAzureAD {
            // Use Azure AD authentication
            credential, err := azidentity.NewDefaultAzureCredential(nil)
            if err != nil {
                return nil, fmt.Errorf("failed to create Azure AD credential: %w", err)
            }

            client = openai.NewClient(
                azure.WithEndpoint(config.Endpoint, config.AzureAPIVersion),
                azure.WithTokenCredential(credential),
                option.WithHTTPClient(httpClient),
            )
        } else {
            // Use API key authentication
            if config.APIKey == "" {
                return nil, fmt.Errorf("Azure OpenAI API key is required when not using Azure AD")
            }

            client = openai.NewClient(
                azure.WithEndpoint(config.Endpoint, config.AzureAPIVersion),
                azure.WithAPIKey(config.APIKey),
                option.WithHTTPClient(httpClient),
            )
        }
    }

    provider := &AzureOpenAIProvider{
        client: client,
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
            SupportsStreaming: true,
            SupportsTools:     true,
            SupportsBatch:     false,
            MaxTokens:         128000, // GPT-4 context window
            MaxBatchSize:      0,      // No batch support currently
        },
    }

    return provider, nil
}

// Provider interface implementation

func (a *AzureOpenAIProvider) Name() string {
    return "Azure OpenAI"
}

func (a *AzureOpenAIProvider) Type() ProviderType {
    return ProviderAzureOpenAI
}

func (a *AzureOpenAIProvider) SupportedTypes() []AnalysisType {
    return a.capabilities.SupportedTypes
}

func (a *AzureOpenAIProvider) Configure(config *ProviderConfig) error {
    if config.APIKey != "" {
        a.config.APIKey = config.APIKey
    }

    if config.Model != "" {
        a.config.Model = config.Model
    }

    if config.Endpoint != "" {
        a.config.Endpoint = config.Endpoint
    }

    if config.AzureDeployment != "" {
        a.config.AzureDeployment = config.AzureDeployment
    }

    if config.AzureAPIVersion != "" {
        a.config.AzureAPIVersion = config.AzureAPIVersion
    }

    if config.Timeout > 0 {
        // Note: Timeout change would require recreating the client
        a.config.Timeout = config.Timeout
    }

    return nil
}

func (a *AzureOpenAIProvider) HealthCheck(ctx context.Context) error {
    // Simple health check by making a minimal API call
    req := &AnalysisRequest{
        Content:     "test",
        MaxTokens:   10,
        Temperature: 0.1,
    }

    _, err := a.Analyze(ctx, req)
    return err
}

func (a *AzureOpenAIProvider) Analyze(ctx context.Context, req *AnalysisRequest) (*AnalysisResponse, error) {
    startTime := time.Now()

    systemPrompt := a.getSystemPromptForType(req.Type)
    userPrompt := a.buildUserPrompt(req)

    maxTokens := int64(1000)
    if req.MaxTokens > 0 {
        maxTokens = int64(req.MaxTokens)
    }

    temperature := float64(0.1) // Default low temperature for consistent results
    if req.Temperature > 0 {
        temperature = req.Temperature
    }

    // Build messages using openai-go types
    messages := []openai.ChatCompletionMessageParamUnion{
        {
            OfSystem: &openai.ChatCompletionSystemMessageParam{
                Content: openai.ChatCompletionSystemMessageParamContentUnion{
                    OfString: openai.String(systemPrompt),
                },
            },
        },
        {
            OfUser: &openai.ChatCompletionUserMessageParam{
                Content: openai.ChatCompletionUserMessageParamContentUnion{
                    OfString: openai.String(userPrompt),
                },
            },
        },
    }

    // Make the API call using openai-go ChatCompletion API
    resp, err := a.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
        Messages:    messages,
        Model:       openai.ChatModel(a.config.AzureDeployment),
        MaxTokens:   openai.Int(maxTokens),
        Temperature: openai.Float(temperature),
    })

    if err != nil {
        return nil, fmt.Errorf("Azure OpenAI API call failed: %w", err)
    }

    if len(resp.Choices) == 0 {
        return nil, fmt.Errorf("no choices in Azure OpenAI response")
    }

    content := resp.Choices[0].Message.Content

    // Extract token usage
    tokensUsed := int(resp.Usage.TotalTokens)
    promptTokens := int(resp.Usage.PromptTokens)
    completionTokens := int(resp.Usage.CompletionTokens)

    return &AnalysisResponse{
        Result:     content,
        Confidence: a.calculateConfidence(req.Type, content),
        Metadata: map[string]interface{}{
            "deployment":        a.config.AzureDeployment,
            "api_version":       a.config.AzureAPIVersion,
            "prompt_tokens":     promptTokens,
            "completion_tokens": completionTokens,
        },
        TokensUsed: tokensUsed,
        Duration:   time.Since(startTime),
        ProviderID: "azure-openai",
    }, nil
}

func (a *AzureOpenAIProvider) SubmitBatch(ctx context.Context, batch *BatchRequest) (*BatchResponse, error) {
    return nil, fmt.Errorf("batch processing not supported by Azure OpenAI provider")
}

func (a *AzureOpenAIProvider) GetBatchStatus(ctx context.Context, batchID string) (*BatchResponse, error) {
    return nil, fmt.Errorf("batch processing not supported by Azure OpenAI provider")
}

func (a *AzureOpenAIProvider) SupportsStreaming() bool {
    return a.capabilities.SupportsStreaming
}

func (a *AzureOpenAIProvider) SupportsTools() bool {
    return a.capabilities.SupportsTools
}

func (a *AzureOpenAIProvider) SupportsBatch() bool {
    return a.capabilities.SupportsBatch
}

// Helper methods (reusing from OpenAI provider)

func (a *AzureOpenAIProvider) getSystemPromptForType(analysisType AnalysisType) string {
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

func (a *AzureOpenAIProvider) buildUserPrompt(req *AnalysisRequest) string {
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

func (a *AzureOpenAIProvider) calculateConfidence(analysisType AnalysisType, result string) float64 {
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
