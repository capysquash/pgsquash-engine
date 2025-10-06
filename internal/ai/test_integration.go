package ai

import (
	"fmt"
	"log"
	"os"

	"github.com/capysquash/pg-squash-engine/internal/ai/providers"
)

// TestAIIntegration demonstrates the new modular AI system
func TestAIIntegration() {
	fmt.Println("🤖 Testing pg-squash AI Integration")
	fmt.Println("====================================")

	// Test Analyzer
	fmt.Println("\n1. Testing Analyzer...")
	analyzer, err := NewAnalyzer()
	if err == nil && analyzer != nil {
		fmt.Println("✅ Modern analyzer available")

		providers := analyzer.GetAvailableProviders()
		fmt.Printf("   Available providers: %v\n", providers)

		capabilities := analyzer.GetCapabilities()
		for providerType, caps := range capabilities {
			fmt.Printf("   %s: %d analysis types, streaming=%v, tools=%v, batch=%v\n",
				providerType, len(caps.SupportedTypes), caps.SupportsStreaming, caps.SupportsTools, caps.SupportsBatch)
		}

		// Health check
		healthResults := analyzer.HealthCheck()
		fmt.Println("   Health check results:")
		for providerType, err := range healthResults {
			if err == nil {
				fmt.Printf("     %s: ✅ Healthy\n", providerType)
			} else {
				fmt.Printf("     %s: ❌ Error: %v\n", providerType, err)
			}
		}
	} else {
		fmt.Println("❌ Modern analyzer not available (no API keys configured)")
	}

	// Test API Key Configuration
	fmt.Println("\n2. Testing API Key Configuration...")
	if os.Getenv("ANTHROPIC_API_KEY") != "" || os.Getenv("OPENAI_API_KEY") != "" {
		fmt.Println("✅ API keys configured")
	} else {
		fmt.Println("❌ No API keys available")
	}

	// Test provider-specific functionality
	fmt.Println("\n3. Testing Provider-Specific Features...")

	if claudeKey := os.Getenv("ANTHROPIC_API_KEY"); claudeKey != "" {
		fmt.Println("✅ Claude API key found")
		testClaudeSpecificFeatures()
	} else {
		fmt.Println("⚠️  Claude API key not found (set ANTHROPIC_API_KEY)")
	}

	if openaiKey := os.Getenv("OPENAI_API_KEY"); openaiKey != "" {
		fmt.Println("✅ OpenAI API key found")
		testOpenAISpecificFeatures()
	} else {
		fmt.Println("⚠️  OpenAI API key not found (set OPENAI_API_KEY)")
	}

	// Test analysis types
	fmt.Println("\n4. Testing Analysis Types...")
	testAnalysisTypes()

	fmt.Println("\n🎉 Integration test complete!")
}

func testClaudeSpecificFeatures() {
	fmt.Println("   Testing Claude-specific features...")

	// Test direct provider access
	analyzer, err := NewAnalyzer()
	if err != nil || analyzer == nil {
		return
	}

	claudeProvider, err := analyzer.GetProvider(ProviderClaude)
	if err != nil {
		fmt.Printf("     ❌ Error getting Claude provider: %v\n", err)
		return
	}

	fmt.Printf("     ✅ Claude provider: %s\n", claudeProvider.Name())
	fmt.Printf("     Supports streaming: %v\n", claudeProvider.SupportsStreaming())
	fmt.Printf("     Supports tools: %v\n", claudeProvider.SupportsTools())
	fmt.Printf("     Supports batch: %v\n", claudeProvider.SupportsBatch())
}

func testOpenAISpecificFeatures() {
	fmt.Println("   Testing OpenAI-specific features...")

	analyzer, err := NewAnalyzer()
	if err != nil || analyzer == nil {
		return
	}

	openaiProvider, err := analyzer.GetProvider(ProviderOpenAI)
	if err != nil {
		fmt.Printf("     ❌ Error getting OpenAI provider: %v\n", err)
		return
	}

	fmt.Printf("     ✅ OpenAI provider: %s\n", openaiProvider.Name())
	fmt.Printf("     Supports streaming: %v\n", openaiProvider.SupportsStreaming())
	fmt.Printf("     Supports tools: %v\n", openaiProvider.SupportsTools())
	fmt.Printf("     Supports batch: %v\n", openaiProvider.SupportsBatch())
}

func testAnalysisTypes() {
	analysisTypes := []providers.AnalysisType{
		providers.AnalysisFunctionEquivalence,
		providers.AnalysisDeadCode,
		providers.AnalysisFunctionComplexity,
		providers.AnalysisAuthPatterns,
		providers.AnalysisOptimizations,
		providers.AnalysisCodeCoverage,
		providers.AnalysisSchemaConsistency,
		providers.AnalysisSQLComplexity,
		providers.AnalysisBatchProcessing,
	}

	fmt.Printf("   Available analysis types: %d\n", len(analysisTypes))
	for i, analysisType := range analysisTypes {
		fmt.Printf("     %d. %s\n", i+1, analysisType)
	}
}

// RunAIIntegrationTest runs the integration test (can be called from CLI)
func RunAIIntegrationTest() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Integration test panic: %v", r)
		}
	}()

	TestAIIntegration()
}

// DemoAICapabilities demonstrates AI capabilities with sample data
func DemoAICapabilities() error {
	fmt.Println("🎯 Demonstrating AI Capabilities")
	fmt.Println("================================")

	analyzer, err := NewAnalyzer()
	if err != nil || analyzer == nil {
		return fmt.Errorf("no AI providers available: %v", err)
	}

	// Example 1: Function Equivalence
	fmt.Println("\n📝 Example 1: Function Equivalence Analysis")
	func1 := `CREATE FUNCTION get_user_count() RETURNS integer AS $$
BEGIN
    RETURN (SELECT COUNT(*) FROM users);
END;
$$ LANGUAGE plpgsql;`

	func2 := `CREATE FUNCTION get_user_count() RETURNS int AS $$
    SELECT count(*)::int FROM users;
$$ LANGUAGE sql;`

	equivalent, err := analyzer.AreFunctionsSemanticallyEquivalent(func1, func2)
	if err != nil {
		fmt.Printf("   ❌ Error: %v\n", err)
	} else {
		fmt.Printf("   Functions are equivalent: %v\n", equivalent)
	}

	// Example 2: Dead Code Detection
	fmt.Println("\n🗑️  Example 2: Dead Code Detection")
	schema := `
CREATE TABLE users (id serial, name text);
CREATE FUNCTION get_user_by_id(user_id int) RETURNS users AS $$
    SELECT * FROM users WHERE id = user_id;
$$ LANGUAGE sql;

CREATE FUNCTION unused_function() RETURNS void AS $$
    -- This function is never called
$$ LANGUAGE sql;`

	isDead, err := analyzer.IsDeadCode(schema, "unused_function")
	if err != nil {
		fmt.Printf("   ❌ Error: %v\n", err)
	} else {
		fmt.Printf("   Function 'unused_function' is dead code: %v\n", isDead)
	}

	// Example 3: Auth Pattern Detection
	fmt.Println("\n🔐 Example 3: Authentication Pattern Detection")
	authSQL := `
CREATE POLICY user_policy ON users
    FOR ALL TO authenticated
    USING (auth.jwt() ->> 'sub' = user_id::text);

CREATE OR REPLACE FUNCTION get_current_user_id()
RETURNS uuid AS $$
BEGIN
    RETURN (auth.jwt() ->> 'sub')::uuid;
END;
$$ LANGUAGE plpgsql;`

	patterns, err := analyzer.DetectAuthPatterns(authSQL)
	if err != nil {
		fmt.Printf("   ❌ Error: %v\n", err)
	} else {
		fmt.Printf("   Detected %d authentication patterns:\n", len(patterns))
		for i, pattern := range patterns {
			fmt.Printf("     %d. %s\n", i+1, pattern)
		}
	}

	fmt.Println("\n✨ AI capabilities demonstration complete!")
	return nil
}
