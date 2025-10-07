package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	SafetyLevel            string                   `json:"safety_level"`
	ProdDBDSN              string                   `json:"prod_db_dsn"`
	Output                 OutputConfig             `json:"output"`
	Rules                  RulesConfig              `json:"rules"`
	ExcludePatterns        []string                 `json:"exclude_patterns"`
	IncludeSchemas         []string                 `json:"include_schemas"`
	Performance            PerformanceConfig        `json:"performance"`
	ModernFeatures         ModernFeaturesConfig     `json:"modern_features"`
	ConflictResolution     ConflictResolutionConfig `json:"conflict_resolution"`
	PostgreSQLFeatures     PostgreSQLFeaturesConfig `json:"postgresql_features"`
	ThirdPartyIntegrations ThirdPartyConfig         `json:"third_party_integrations"`
	Plugins                PluginSettings           `json:"plugins"`    // Plugin system configuration
	Validation             ValidationConfig         `json:"validation"` // Docker validation configuration
}

type OutputConfig struct {
	Format                   string `json:"format"`
	PreserveComments         bool   `json:"preserve_comments"`
	AddConsolidationComments bool   `json:"add_consolidation_comments"`
	FileNaming               string `json:"file_naming"`
	Directory                string `json:"directory"`
}

type RulesConfig struct {
	TableOperations    TableRulesConfig    `json:"table_operations"`
	IndexOperations    IndexRulesConfig    `json:"index_operations"`
	FunctionOperations FunctionRulesConfig `json:"function_operations"`
}

type TableRulesConfig struct {
	ConsolidateCreateAlter bool `json:"consolidate_create_alter"`
	RemoveDropCreateCycles bool `json:"remove_drop_create_cycles"`
	PreserveDataOperations bool `json:"preserve_data_operations"`
}

type IndexRulesConfig struct {
	ConsolidateRecreations    bool `json:"consolidate_recreations"`
	PreserveUniqueConstraints bool `json:"preserve_unique_constraints"`
}

type FunctionRulesConfig struct {
	RemoveDuplicateDefinitions bool `json:"remove_duplicate_definitions"`
	PreserveSignatureChanges   bool `json:"preserve_signature_changes"`
}

type PerformanceConfig struct {
	StreamingThresholdMB int  `json:"streaming_threshold_mb"`
	ParallelProcessing   bool `json:"parallel_processing"`
	ShowProgress         bool `json:"show_progress"`
}

// New configuration structures for modern PostgreSQL features

type ModernFeaturesConfig struct {
	EnableVectorSupport    bool `json:"enable_vector_support"`
	EnableGeneratedColumns bool `json:"enable_generated_columns"`
	EnableEventSourcing    bool `json:"enable_event_sourcing"`
	EnableMergeStatements  bool `json:"enable_merge_statements"`
	EnableMultirangeTypes  bool `json:"enable_multirange_types"`
	EnableAdvancedRLS      bool `json:"enable_advanced_rls"`
}

type ConflictResolutionConfig struct {
	EnablePrioritySystem  bool   `json:"enable_priority_system"`
	StrictModeEnabled     bool   `json:"strict_mode_enabled"`
	AllowOverlappingRules bool   `json:"allow_overlapping_rules"`
	ConflictLogLevel      string `json:"conflict_log_level"`
}

type PostgreSQLFeaturesConfig struct {
	TargetVersion          string   `json:"target_version"`
	EnabledExtensions      []string `json:"enabled_extensions"`
	OptimizeForPerformance bool     `json:"optimize_for_performance"`
	UseModernSyntax        bool     `json:"use_modern_syntax"`
	ValidateCompatibility  bool     `json:"validate_compatibility"`
}

type ThirdPartyConfig struct {
	Auth0Integration       Auth0Config       `json:"auth0_integration"`
	NextAuthIntegration    NextAuthConfig    `json:"nextauth_integration"`
	SupabaseIntegration    SupabaseConfig    `json:"supabase_integration"`
	ClerkIntegration       ClerkConfig       `json:"clerk_integration"`
	VectorIntegration      VectorConfig      `json:"vector_integration"`
	PlanetScaleIntegration PlanetScaleConfig `json:"planetscale_integration"`
}

type Auth0Config struct {
	Enabled       bool     `json:"enabled"`
	Domain        string   `json:"domain"`
	CustomClaims  []string `json:"custom_claims"`
	RoleClaimPath string   `json:"role_claim_path"`
}

type NextAuthConfig struct {
	Enabled         bool     `json:"enabled"`
	SessionStrategy string   `json:"session_strategy"`
	DatabaseTables  []string `json:"database_tables"`
}

type SupabaseConfig struct {
	Enabled            bool   `json:"enabled"`
	JWTSecret          string `json:"jwt_secret"`
	EnableRLS          bool   `json:"enable_rls"`
	StorageIntegration bool   `json:"storage_integration"`
}

type ClerkConfig struct {
	Enabled               bool   `json:"enabled"`
	JWTVersion            string `json:"jwt_version"`
	OrganizationSupport   bool   `json:"organization_support"`
	PublicMetadataSupport bool   `json:"public_metadata_support"`
}

type VectorConfig struct {
	Enabled          bool     `json:"enabled"`
	DefaultIndexType string   `json:"default_index_type"`
	OptimizeQueries  bool     `json:"optimize_queries"`
	SupportedOps     []string `json:"supported_ops"`
}

type PlanetScaleConfig struct {
	Enabled                bool `json:"enabled"`
	DisableForeignKeys     bool `json:"disable_foreign_keys"`
	OptimizeForReplication bool `json:"optimize_for_replication"`
}

// PluginSettings configures the plugin system behavior
type PluginSettings struct {
	AutoDetect      bool     `json:"auto_detect"`      // Automatically detect and enable plugins (default: true)
	EnabledPlugins  []string `json:"enabled_plugins"`  // Explicitly enabled plugins (empty = auto-detect all)
	DisabledPlugins []string `json:"disabled_plugins"` // Explicitly disabled plugins
	Verbose         bool     `json:"verbose"`          // Log plugin activity (default: false)
}

// ValidationConfig configures Docker-based validation behavior
type ValidationConfig struct {
	Mode                   string `json:"mode"`                      // Validation approach: TWO_CONTAINERS, TWO_DATABASES, or SCHEMA_DIFF
	DockerImage            string `json:"docker_image"`              // PostgreSQL Docker image (default: postgres:15)
	TimeoutSeconds         int    `json:"timeout_seconds"`           // Validation timeout in seconds (default: 120)
	ContainerReadyTimeout  int    `json:"container_ready_timeout"`   // Container startup timeout in seconds (default: 30)
	EnableExtensionDetection bool `json:"enable_extension_detection"` // Auto-detect and install extensions (default: true)
	AutoInstallExtensions  bool   `json:"auto_install_extensions"`   // Automatically install detected extensions (default: true)
	EnableSQLFixes         bool   `json:"enable_sql_fixes"`          // Apply automatic SQL fixes during validation (default: false)
	Verbose                bool   `json:"verbose"`                   // Show detailed validation output (default: true)
}

func DefaultConfig() *Config {
	return &Config{
		SafetyLevel: "standard",
		ProdDBDSN:   os.Getenv("PROD_DB_DSN"), // Default to environment variable
		Output: OutputConfig{
			Format:                   "organized",
			PreserveComments:         true,
			AddConsolidationComments: true,
			FileNaming:               "semantic",
			Directory:                "squashed",
		},
		Rules: RulesConfig{
			TableOperations: TableRulesConfig{
				ConsolidateCreateAlter: true,
				RemoveDropCreateCycles: true,
				PreserveDataOperations: true,
			},
			IndexOperations: IndexRulesConfig{
				ConsolidateRecreations:    true,
				PreserveUniqueConstraints: true,
			},
			FunctionOperations: FunctionRulesConfig{
				RemoveDuplicateDefinitions: true,
				PreserveSignatureChanges:   false,
			},
		},
		ExcludePatterns: []string{"auth.*", "extensions.*"},
		IncludeSchemas:  []string{"public"},
		Performance: PerformanceConfig{
			StreamingThresholdMB: 5,
			ParallelProcessing:   true,
			ShowProgress:         true,
		},
		ModernFeatures: ModernFeaturesConfig{
			EnableVectorSupport:    true,
			EnableGeneratedColumns: true,
			EnableEventSourcing:    true,
			EnableMergeStatements:  true,
			EnableMultirangeTypes:  true,
			EnableAdvancedRLS:      true,
		},
		ConflictResolution: ConflictResolutionConfig{
			EnablePrioritySystem:  true,
			StrictModeEnabled:     false,
			AllowOverlappingRules: false,
			ConflictLogLevel:      "warn",
		},
		PostgreSQLFeatures: PostgreSQLFeaturesConfig{
			TargetVersion:          "15",
			EnabledExtensions:      []string{"vector", "pg_stat_statements", "uuid-ossp"},
			OptimizeForPerformance: true,
			UseModernSyntax:        true,
			ValidateCompatibility:  true,
		},
		ThirdPartyIntegrations: ThirdPartyConfig{
			Auth0Integration: Auth0Config{
				Enabled:       false,
				Domain:        "",
				CustomClaims:  []string{"permissions", "role"},
				RoleClaimPath: "https://myapp.com/role",
			},
			NextAuthIntegration: NextAuthConfig{
				Enabled:         false,
				SessionStrategy: "database",
				DatabaseTables:  []string{"accounts", "sessions", "users"},
			},
			SupabaseIntegration: SupabaseConfig{
				Enabled:            true,
				JWTSecret:          "",
				EnableRLS:          true,
				StorageIntegration: true,
			},
			ClerkIntegration: ClerkConfig{
				Enabled:               false,
				JWTVersion:            "v2",
				OrganizationSupport:   true,
				PublicMetadataSupport: true,
			},
			VectorIntegration: VectorConfig{
				Enabled:          true,
				DefaultIndexType: "ivfflat",
				OptimizeQueries:  true,
				SupportedOps:     []string{"vector_cosine_ops", "vector_l2_ops", "vector_ip_ops"},
			},
			PlanetScaleIntegration: PlanetScaleConfig{
				Enabled:                false,
				DisableForeignKeys:     true,
				OptimizeForReplication: true,
			},
		},
		Plugins: PluginSettings{
			AutoDetect:      true,     // Automatically detect applicable plugins
			EnabledPlugins:  []string{}, // Empty = enable all detected plugins
			DisabledPlugins: []string{}, // Explicitly disable specific plugins
			Verbose:         false,    // Don't log plugin details by default
		},
		Validation: ValidationConfig{
			Mode:                     "TWO_DATABASES", // Best balance of speed and accuracy
			DockerImage:              "postgres:15",   // Default PostgreSQL version
			TimeoutSeconds:           120,             // 2 minute timeout for validation
			ContainerReadyTimeout:    30,              // 30 second timeout for container startup
			EnableExtensionDetection: true,            // Auto-detect required extensions
			AutoInstallExtensions:    true,            // Auto-install detected extensions
			EnableSQLFixes:           false,           // Manual review recommended by default
			Verbose:                  true,            // Show detailed validation output
		},
	}
}

func LoadConfig(configPath string) (*Config, error) {
	cfg := DefaultConfig()

	if configPath == "" {
		configPath = "pgsquash.config.json"
	}

	if _, err := os.Stat(configPath); err == nil {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

func (c *Config) SaveToFile(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
