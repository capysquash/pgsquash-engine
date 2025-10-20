# AI-Powered Migration Analysis

> **Looking for AI validation workflows?** See [AI Validation Usage Guide](ai-validation-usage.md) for post-processing validation, semantic checking, and quality assessment.

pgsquash integrates with leading AI providers to enhance migration analysis, semantic understanding, and intelligent optimization.

## Overview

pgsquash supports AI providers for semantic analysis beyond static parsing:

- Function equivalence detection
- Dead code identification
- Authentication pattern recognition
- Performance optimization suggestions
- Complexity analysis

**This guide covers:** AI configuration, provider setup, and analysis features.

**For validation workflows:** See [AI Validation Usage](ai-validation-usage.md).

## Supported Providers

### Claude (Anthropic) - Recommended

**Model:** claude-3-5-sonnet-20241022

Best for semantic analysis and code understanding.

**Capabilities:**

- Function semantic equivalence
- Dead code detection
- Authentication pattern recognition
- Performance optimization
- Complexity analysis

### OpenAI

**Model:** gpt-4

General analysis and optimization suggestions.

**Capabilities:**

- Function semantic equivalence
- Dead code detection
- Performance optimization

### Azure OpenAI

**Model:** Configurable (typically gpt-4)

Enterprise OpenAI deployment for organizations using Azure.

**Capabilities:**

- Function semantic equivalence
- Dead code detection
- Performance optimization

## Setup

### Claude (Recommended)

```bash
# Get API key from https://console.anthropic.com/
export ANTHROPIC_API_KEY="sk-ant-api03-..."

# Add to shell profile
echo 'export ANTHROPIC_API_KEY="sk-ant-..."' >> ~/.zshrc
source ~/.zshrc
```

### OpenAI

```bash
# Get API key from https://platform.openai.com/
export OPENAI_API_KEY="sk-..."

# Add to shell profile
echo 'export OPENAI_API_KEY="sk-..."' >> ~/.zshrc
source ~/.zshrc
```

### Azure OpenAI

```bash
export AZURE_OPENAI_API_KEY="..."
export AZURE_OPENAI_ENDPOINT="https://your-endpoint.openai.azure.com/"
export AZURE_OPENAI_DEPLOYMENT="gpt-4"
```

### Test Configuration

```bash
pgsquash ai-test
```

Output:

```
Testing AI Provider Integrations

Testing Claude (Anthropic)...
✓ Provider: Claude
  Status: Available
  Model: claude-3-5-sonnet-20241022

Testing OpenAI...
✓ Provider: OpenAI
  Status: Available
  Model: gpt-4

AI Integration: 2 providers available
```

## Usage

### Demo Mode

See AI capabilities with sample code:

```bash
pgsquash ai-demo
```

Shows:

- Function semantic equivalence
- Dead code detection
- Auth pattern recognition
- Performance suggestions

### Deep Analysis

Comprehensive AI-powered analysis:

```bash
export ANTHROPIC_API_KEY="sk-ant-..."
pgsquash analyze-deep migrations/*.sql
```

Output includes:

- Authentication patterns detected
- Dead code functions
- Semantically equivalent function pairs
- Complexity warnings
- Performance optimizations

### Fast Workflow (AI-Enhanced)

Development workflow with AI optimizations:

```bash
pgsquash fast migrations/*.sql --output dev/
```

AI features:

- Function deduplication
- Performance suggestions
- Dead code warnings

## Features

### Function Semantic Equivalence

Detects functionally identical code with different implementations:

```sql
-- Migration 001
CREATE FUNCTION calculate_total(a INT, b INT) RETURNS INT AS $$
BEGIN
    RETURN a + b;
END;
$$ LANGUAGE plpgsql;

-- Migration 050
CREATE FUNCTION compute_sum(x INT, y INT) RETURNS INT AS $$
BEGIN
    RETURN x + y;
END;
$$ LANGUAGE plpgsql;
```

AI detects: These functions are semantically equivalent.

### Dead Code Detection

Identifies unused functions and procedures:

```sql
-- Never referenced in migrations or code
CREATE FUNCTION old_cleanup() RETURNS VOID AS $$
BEGIN
    DELETE FROM deprecated_table WHERE created_at < NOW() - INTERVAL '90 days';
END;
$$ LANGUAGE plpgsql;
```

AI reports: `old_cleanup` appears to be dead code (no references found).

### Authentication Pattern Recognition

Detects authentication/security patterns:

```sql
-- Supabase RLS
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
CREATE POLICY users_select ON users FOR SELECT USING (auth.uid() = id);

-- Clerk JWT v2
CREATE FUNCTION clerk_user_id() RETURNS TEXT AS $$
BEGIN
    RETURN (SELECT auth.jwt()->>'sub');
END;
$$ LANGUAGE plpgsql STABLE SECURITY DEFINER;
```

AI detects:

- SUPABASE\_AUTH: RLS policies detected
- CLERK\_JWT\_V2: JWT validation functions

### Performance Optimization

Suggests query and schema improvements:

```sql
-- Query with missing index
SELECT * FROM events WHERE user_id = 123 AND created_at > NOW() - INTERVAL '30 days';
```

AI suggests:

- Add composite index: `CREATE INDEX idx_events_user_created ON events(user_id, created_at);`
- Consider BRIN index for `created_at` if table is large
- Use table partitioning for time-series data

### Complexity Analysis

Identifies complex migrations needing review:

```sql
-- High complexity: many operations, dependencies
CREATE TABLE complex (...);
ALTER TABLE complex ADD COLUMN ...;
CREATE INDEX ...;
ALTER TABLE complex ADD CONSTRAINT ...;
CREATE TRIGGER complex_trigger ...;
CREATE FUNCTION complex_fn() ...;
```

AI warns: High complexity migration (6 operations, 4 dependencies). Consider review.

## Configuration

### Basic Configuration

In `pgsquash.config.json`:

```json
{
  "ai": {
    "enabled": true,
    "provider": "claude",
    "confidence_threshold": 0.7,
    "max_retries": 3,
    "timeout_seconds": 30
  }
}
```

### Complete Configuration Reference

All available AI configuration options:

```json
{
  "ai": {
    "enabled": false,
    "provider": "auto",
    "max_retries": 3,
    "timeout_seconds": 60,
    "enable_semantic_analysis": false,
    "enable_dead_code_detection": false,
    "enable_auth_pattern_detection": true,
    "enable_post_processing_validation": false,
    "enable_auto_repair": false,
    "confidence_threshold": 0.85
  }
}
```

### Configuration Options Explained

| Option                              | Type    | Default  | Description                                                          |
| ----------------------------------- | ------- | -------- | -------------------------------------------------------------------- |
| `enabled`                           | boolean | `false`  | Enable AI features (requires API keys)                               |
| `provider`                          | string  | `"auto"` | AI provider: `"auto"`, `"claude"`, `"openai"`, `"azure-openai"`      |
| `max_retries`                       | integer | `3`      | Maximum retry attempts for AI calls (must be >= 0)                   |
| `timeout_seconds`                   | integer | `60`     | Timeout for AI operations in seconds (must be > 0)                   |
| `enable_semantic_analysis`          | boolean | `false`  | Use AI for semantic function comparison                              |
| `enable_dead_code_detection`        | boolean | `false`  | Use AI for dead code detection                                       |
| `enable_auth_pattern_detection`     | boolean | `true`   | Use AI to detect auth patterns (enabled by default if AI is enabled) |
| `enable_post_processing_validation` | boolean | `false`  | Use AI for post-processing validation                                |
| `enable_auto_repair`                | boolean | `false`  | Allow AI to automatically fix issues (requires manual review)        |
| `confidence_threshold`              | float   | `0.85`   | Minimum confidence for AI suggestions (0.0-1.0)                      |

### Disable AI Features

```json
{
  "ai": {
    "enabled": false
  }
}
```

## Cost Management

AI features use provider APIs (costs apply).

### Detailed Cost Breakdown

| Migration Size               | Tokens (Input/Output) | Cost (Claude) | Cost (OpenAI GPT-4) | Cost (Azure OpenAI) |
| ---------------------------- | --------------------- | ------------- | ------------------- | ------------------- |
| Small (10 files, 1K lines)   | 10K / 2K              | \~$0.06       | \~$0.16             | Varies by agreement |
| Medium (50 files, 10K lines) | 50K / 5K              | \~$0.24       | \~$0.65             | Varies by agreement |
| Large (200 files, 50K lines) | 200K / 15K            | \~$0.83       | \~$2.60             | Varies by agreement |
| Enterprise (1000+ files)     | 1M / 50K              | \~$3.75       | \~$11.50            | Varies by agreement |

**Note**: Azure OpenAI costs depend on your enterprise agreement.

### Cost Optimization Tips

1. **Use `confidence_threshold`**: Set to 0.7-0.8 to reduce unnecessary AI calls
2. **Enable selectively**: Only enable features you need (e.g., just auth pattern detection)
3. **Batch operations**: Analyze multiple files together instead of individually
4. **Cache results**: AI analysis results are cached per migration hash
5. **Use workflows selectively**: Not every operation needs AI analysis. Use `analyze-deep` for audits and `fast` or `safe` for consolidation.
6. **Disable for CI/CD**: Run AI analysis manually, not in automated pipelines.
7. **Set appropriate timeouts**: Lower timeouts prevent expensive long-running calls

## When to Use AI Features

**Good use cases:**

- Function deduplication (aggressive mode)
- Dead code cleanup before production
- Performance audit
- Security pattern verification
- Technical debt analysis

**Not recommended:**

- Critical production deployments (use validation instead)
- Compliance decisions (use manual review)
- When provider unavailable
- Cost-sensitive environments

## Security & Privacy

### Data Handling

- **Migration SQL sent to AI providers**: Yes, migration SQL is sent for analysis
- **API keys stored**: Only in environment variables or config files (never sent externally)
- **Response caching**: Local only, never sent to external services
- **Data retention**: Check your AI provider's data retention policy

### Enterprise Considerations

For sensitive migrations:

1. **Use Azure OpenAI**: Data processed in your Azure tenant (regional compliance)
2. **Review SQL before analysis**: Redact sensitive table/column names if needed
3. **Disable auto-repair**: Require manual review for all AI-suggested changes
4. **Audit logging**: Enable detailed logs for compliance requirements
5. **Test in staging first**: Never run AI analysis directly on production migrations

### Redacting Sensitive Information

If your migrations contain sensitive information, consider redacting them manually before running analysis commands like `analyze-deep`. The tool does not currently offer an automatic redaction feature via command-line flags.

## Best Practices

### 1. Start with Conservative Settings

```json
{
  "ai": {
    "enabled": true,
    "provider": "claude",
    "confidence_threshold": 0.85,
    "enable_semantic_analysis": true,
    "enable_dead_code_detection": false,
    "enable_auth_pattern_detection": true,
    "enable_auto_repair": false
  }
}
```

**Rationale**: Enable only core features, require high confidence (0.85), disable auto-repair for safety.

### 2. Always Review AI Suggestions

AI suggestions are surfaced in workflows like `analyze-deep` and `fast`. These should be **reviewed**, not blindly accepted. The `ai-fix` command provides an interactive loop for reviewing and applying changes.

```bash
# Step 1: Use ai-fix for an interactive repair session
pgsquash ai-fix migrations/

# Step 2: For analysis, use analyze-deep and review the output
pgsquash analyze-deep migrations/

# Step 3: For AI-enhanced squashing, use the fast or safe workflows
pgsquash fast migrations/*.sql --output dev/
```

### 3. Appropriate Confidence Thresholds

| Use Case              | Recommended Threshold            |
| --------------------- | -------------------------------- |
| Production migrations | 0.85-0.95 (very high confidence) |
| Staging environment   | 0.75-0.85 (high confidence)      |
| Development/Testing   | 0.65-0.75 (moderate confidence)  |
| Exploratory analysis  | 0.50-0.65 (exploratory)          |

### 4. Enable Features Progressively

Gradual rollout approach:

- **Week 1**: Auth pattern detection only
- **Week 2**: Add semantic analysis
- **Week 3**: Add dead code detection
- **Week 4**: Enable post-processing validation

This allows you to evaluate each feature's value and accuracy before adding more.

### 5. Monitor AI Usage

Track costs and performance:

```bash
# Check AI usage statistics (if available)
pgsquash ai-stats

# Example output:
# AI Usage Statistics
# ===================
# Provider: claude
# Total requests: 47
# Successful: 45 (95.7%)
# Failed: 2 (4.3%)
# Average latency: 312ms
# Total tokens: 125K input, 28K output
# Estimated cost: $0.45
```

## Troubleshooting

### API Key Not Working

```bash
# Verify key is set
echo $ANTHROPIC_API_KEY

# Test manually
curl https://api.anthropic.com/v1/messages \
  -H "x-api-key: $ANTHROPIC_API_KEY" \
  -H "anthropic-version: 2023-06-01" \
  -H "content-type: application/json" \
  -d '{"model":"claude-3-5-sonnet-20241022","max_tokens":1024,"messages":[{"role":"user","content":"test"}]}'
```

### Timeout Errors

Increase timeout in config:

```json
{
  "ai": {
    "timeout_seconds": 60
  }
}
```

### Rate Limits

Reduce concurrency:

```json
{
  "ai": {
    "max_concurrent_requests": 1
  }
}
```

### Provider Unavailable

AI features gracefully degrade:

- Analysis continues without AI
- Warnings logged
- No errors thrown

## Further Reading

- [CLI Reference](./cli-reference.md) - AI commands
- [Configuration](./configuration.md) - AI config options
- [Troubleshooting](./troubleshooting.md) - Common issues
