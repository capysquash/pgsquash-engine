# AI-Powered Post-Processing Validation

> **New to AI features?** Start with the [AI Features Overview](ai-features.md) for configuration, provider setup, and core AI capabilities.

## Overview

The AI validation system provides semantic validation, dependency checking, and quality assessment beyond traditional syntax validation. It uses LLM-powered analysis to detect logical errors, security issues, and optimization opportunities.

**This guide covers:** AI validation workflows, post-processing checks, and semantic analysis.

**For AI configuration:** See [AI Features](ai-features.md).

## Features

### 1. Semantic Validation

- **Logic Errors**: Detects contradictory constraints, impossible conditions
- **Data Integrity**: Identifies missing foreign keys, orphaned references
- **Security Issues**: Flags overly permissive RLS policies, SQL injection risks
- **Inconsistencies**: Catches mismatched types, conflicting naming patterns

### 2. Dependency Validation

- **Circular Dependencies**: Detects circular references between objects
- **Missing Dependencies**: Identifies objects used before creation
- **Invalid Ordering**: Flags incorrect dependency order

### 3. Quality Reports

- **Overall Score**: 1-10 quality assessment
- **Maintainability**: High/medium/low maintainability rating
- **Complexity**: Complexity score (1-10)
- **Best Practices**: Lists followed and violated best practices

### 4. Optimization Suggestions

- Performance improvements
- Maintainability enhancements
- Security hardening recommendations

## Usage

### Basic Setup

```go
package main

import (
    "context"
    "log"

    "github.com/capysquash/pgsquash-engine/internal/ai"
    "github.com/capysquash/pgsquash-engine/internal/validation"
    "github.com/capysquash/pgsquash-engine/internal/types"
)

func main() {
    // 1. Initialize AI analyzer
    analyzer, err := ai.NewAnalyzer()
    if err != nil {
        log.Fatalf("Failed to create AI analyzer: %v", err)
    }

    // 2. Create AI validation config
    aiConfig := &validation.AIValidationConfig{
        EnableSemanticValidation: true,
        EnableDependencyChecks:   true,
        EnableQualityReports:     true,
        EnableAutoRepair:         false, // Conservative default
        ConfidenceThreshold:      0.85,
    }

    // 3. Create AI validator
    aiValidator := validation.NewAIValidator(analyzer, aiConfig, true)

    // 4. Create schema validator
    schemaValidator := validation.NewSchemaValidator(
        &validation.ValidationConfig{
            Level:         validation.ValidationLevelStandard,
            DockerApproach: validation.ApproachTwoDatabases,
            Verbose:       true,
        },
        nil, // No database connection needed for AI validation
        nil, // No progress reporter
    )

    // 5. Set AI validator on schema validator
    schemaValidator.SetAIValidator(aiValidator)

    // 6. Perform validation
    ctx := context.Background()
    migrations := loadMigrations() // Your migration loading logic
    squashedSQL := readSquashedOutput() // Your squashed SQL

    result, err := schemaValidator.ValidateWithAI(ctx, migrations, squashedSQL)
    if err != nil {
        log.Fatalf("AI validation failed: %v", err)
    }

    // 7. Check results
    if result.Success {
        log.Println("☑ AI validation passed!")
    } else {
        log.Println("☒ AI validation found issues:")
        for _, issue := range result.SemanticIssues {
            log.Printf("  - [%s] %s", issue.Severity, issue.Description)
        }
    }
}
```

### Integration with Docker Validation

```go
// Perform Docker validation first
dockerResult, err := schemaValidator.ValidateWithDocker(ctx, originalPath, squashedPath)
if err != nil {
    log.Fatalf("Docker validation failed: %v", err)
}

// If Docker validation passes, run AI validation
if dockerResult.Success {
    // Read squashed SQL for AI analysis
    squashedSQL, err := os.ReadFile(squashedPath)
    if err != nil {
        log.Fatalf("Failed to read squashed SQL: %v", err)
    }

    // Run AI validation
    aiResult, err := schemaValidator.ValidateWithAI(ctx, migrations, string(squashedSQL))
    if err != nil {
        log.Fatalf("AI validation failed: %v", err)
    }

    // Check AI results
    if aiResult.Success {
        log.Println("☑ All validations passed!")
    } else {
        log.Println("⚠️  AI detected potential issues:")
        printAIResults(aiResult)
    }
}
```

### Configuration via pgsquash.config.json

```json
{
  "ai": {
    "enabled": true,
    "provider": "claude",
    "max_retries": 3,
    "timeout_seconds": 60,
    "enable_semantic_analysis": true,
    "enable_dead_code_detection": false,
    "enable_auth_pattern_detection": true,
    "enable_post_processing_validation": true,
    "enable_auto_repair": false,
    "confidence_threshold": 0.85
  },
  "validation": {
    "mode": "TWO_DATABASES",
    "enable_sql_fixes": true,
    "verbose": true
  }
}
```

## Output Examples

### Semantic Issue Example

```json
{
  "type": "logic_error",
  "severity": "critical",
  "description": "RLS policy 'user_access' references non-existent auth.uid() function",
  "location": "public.users",
  "reasoning": "The policy attempts to call auth.uid() but no such function exists in the schema",
  "confidence": 0.95,
  "suggestion": "Create auth.uid() function or use a different authentication method"
}
```

### Quality Report Example

```json
{
  "overall_score": 8,
  "maintainability": "high",
  "complexity": 4,
  "best_practices": [
    "Uses RLS for data security",
    "Proper foreign key constraints",
    "Consistent naming conventions"
  ],
  "violations": [
    "Missing indexes on foreign keys",
    "No comments on complex functions"
  ],
  "summary": "Well-structured schema with good security practices. Consider adding indexes for better performance."
}
```

### Dependency Issue Example

```json
{
  "type": "missing",
  "description": "Table 'orders' references 'users' table which is created later in the migration",
  "objects": ["orders", "users"],
  "confidence": 0.92,
  "suggestion": "Reorder migrations to create 'users' table before 'orders' table"
}
```

## Environment Variables

Required environment variables for AI features:

```bash
# For Claude (Anthropic)
export ANTHROPIC_API_KEY="sk-ant-..."

# For OpenAI
export OPENAI_API_KEY="sk-..."

# For Azure OpenAI
export AZURE_OPENAI_ENDPOINT="https://your-resource.openai.azure.com"
export AZURE_OPENAI_KEY="your-key"
# Or use Azure AD authentication
export AZURE_OPENAI_USE_AD="true"
```

## Best Practices

### 1. Use Conservative Confidence Thresholds

Start with a high threshold (0.85) and lower it only if needed:

```go
aiConfig := &validation.AIValidationConfig{
    ConfidenceThreshold: 0.85, // Only show high-confidence issues
}
```

### 2. Enable Selectively in CI/CD

For faster CI/CD, enable only critical checks:

```go
aiConfig := &validation.AIValidationConfig{
    EnableSemanticValidation: true,  // Critical logic errors
    EnableDependencyChecks:   true,  // Dependency issues
    EnableQualityReports:     false, // Skip in CI
    EnableAutoRepair:         false, // Never auto-repair in CI
}
```

### 3. Review AI Suggestions

Always review AI suggestions before applying:

```go
for _, suggestion := range result.RepairSuggestions {
    if suggestion.Confidence > 0.9 && suggestion.AutoApply {
        // High confidence, safe to consider
        log.Printf("High-confidence suggestion: %s", suggestion.Fix)
    }
}
```

### 4. Handle Large Migrations

For very large migrations, AI validation automatically chunks SQL:

```go
// The validator automatically splits large SQL into ~3000 token chunks
// No special handling needed - it's automatic
```

## Performance Considerations

- **Docker Validation**: 5-30 seconds (container startup + migrations)
- **AI Validation**: 10-60 seconds (depends on SQL size and provider)
- **Total Validation Time**: \~15-90 seconds for typical migrations

### Optimization Tips

1. **Use Faster Providers**: Claude Haiku or GPT-3.5 for speed
2. **Parallel Validation**: Run Docker and AI validation concurrently (not yet implemented)
3. **Caching**: AI validation results can be cached based on SQL hash (future enhancement)

## Troubleshooting

### Issue: "AI analyzer not initialized"

**Solution**: Ensure environment variables are set:

```bash
export ANTHROPIC_API_KEY="sk-ant-..."
# or
export OPENAI_API_KEY="sk-..."
```

### Issue: "Low confidence warnings"

**Solution**: Adjust confidence threshold:

```go
aiConfig.ConfidenceThreshold = 0.75 // Lower threshold to see more issues
```

### Issue: "Timeout errors"

**Solution**: Increase timeout:

```json
{
  "ai": {
    "timeout_seconds": 120
  }
}
```

## Future Enhancements

- **Parallel Validation**: Run Docker and AI validation concurrently
- **Result Caching**: Cache AI results based on SQL content hash
- **Auto-Repair Mode**: Safe automatic fixes for high-confidence issues
- **Custom Prompts**: Allow users to define custom validation rules
- **Multi-Provider Consensus**: Use multiple AI providers for higher confidence

## Related Documentation

- [AI Configuration Reference](configuration.md#ai-configuration)
- [Validation Approaches](safety-levels.md#validation-approaches)
- [CLI Reference](cli-reference.md#ai-commands)
