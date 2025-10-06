package squasher

import (
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"
)

// ExtensionDetector analyzes migrations to detect required PostgreSQL extensions
type ExtensionDetector struct {
	// Known extensions and their installation requirements
	extensionMap map[string]ExtensionInfo
}

// ExtensionInfo holds information about a PostgreSQL extension
type ExtensionInfo struct {
	Name              string   // Extension name
	PackageName       string   // APT package name for installation
	DockerImage       string   // Preferred Docker image that includes this extension
	Dependencies      []string // Other extensions this depends on
	InstallCommand    string   // Custom installation command if needed
	ValidationSQL     string   // SQL to test if extension is available
	RequiresCASCADE   bool     // Whether this extension typically needs CASCADE
	RequiresPostGIS   bool     // Whether this requires PostGIS base
}

// NewExtensionDetector creates a new extension detector with known extensions
func NewExtensionDetector() *ExtensionDetector {
	detector := &ExtensionDetector{
		extensionMap: make(map[string]ExtensionInfo),
	}

	// Initialize known extensions
	detector.initializeExtensions()
	return detector
}

// initializeExtensions populates the known extensions map
func (ed *ExtensionDetector) initializeExtensions() {
	// Extension definitions with Debian-based Docker images
	// Migrated from Alpine to standard postgres (Debian) images
	// PostGIS uses the official PostGIS image which is Debian-based
	extensions := []ExtensionInfo{
		{
			Name:            "postgis",
			PackageName:     "postgresql-15-postgis-3",
			DockerImage:     "postgis/postgis:15-3.3", // Debian-based PostGIS image
			Dependencies:    []string{},
			ValidationSQL:   "SELECT PostGIS_version();",
			RequiresCASCADE: true,
		},
		{
			Name:            "earthdistance",
			PackageName:     "postgresql-contrib",
			DockerImage:     "postgres:15", // Standard Debian-based postgres
			Dependencies:    []string{"cube"},
			ValidationSQL:   "SELECT earth_distance(ll_to_earth(0,0), ll_to_earth(0,1));",
			RequiresCASCADE: true,
		},
		{
			Name:            "cube",
			PackageName:     "postgresql-contrib",
			DockerImage:     "postgres:15",
			Dependencies:    []string{},
			ValidationSQL:   "SELECT cube(array[1,2,3]);",
			RequiresCASCADE: false,
		},
		{
			Name:            "uuid-ossp",
			PackageName:     "postgresql-contrib",
			DockerImage:     "postgres:15",
			Dependencies:    []string{},
			ValidationSQL:   "SELECT uuid_generate_v4();",
			RequiresCASCADE: false,
		},
		{
			Name:            "pg_trgm",
			PackageName:     "postgresql-contrib",
			DockerImage:     "postgres:15",
			Dependencies:    []string{},
			ValidationSQL:   "SELECT similarity('hello', 'hallo');",
			RequiresCASCADE: false,
		},
		{
			Name:            "pg_stat_statements",
			PackageName:     "postgresql-contrib",
			DockerImage:     "postgres:15",
			Dependencies:    []string{},
			InstallCommand:  "shared_preload_libraries = 'pg_stat_statements'",
			ValidationSQL:   "SELECT query FROM pg_stat_statements LIMIT 1;",
			RequiresCASCADE: false,
		},
		{
			Name:            "btree_gin",
			PackageName:     "postgresql-contrib",
			DockerImage:     "postgres:15",
			Dependencies:    []string{},
			ValidationSQL:   "SELECT 1;", // Simple validation
			RequiresCASCADE: false,
		},
		{
			Name:            "plpgsql",
			PackageName:     "", // Built-in extension
			DockerImage:     "postgres:15",
			Dependencies:    []string{},
			ValidationSQL:   "SELECT 1;",
			RequiresCASCADE: false,
		},
	}

	for _, ext := range extensions {
		ed.extensionMap[strings.ToLower(ext.Name)] = ext
	}
}

// AuthServiceType represents different authentication services
type AuthServiceType string

const (
	AuthServiceNone     AuthServiceType = "none"
	AuthServiceClerk    AuthServiceType = "clerk"
	AuthServiceSupabase AuthServiceType = "supabase"
	AuthServiceAuth0    AuthServiceType = "auth0"
)

// ExtensionAnalysis holds the results of extension detection
type ExtensionAnalysis struct {
	RequiredExtensions    []string                    // List of extensions found
	ExtensionDetails      map[string]ExtensionInfo    // Detailed info for each extension
	RecommendedDockerBase string                      // Best Docker image to use
	InstallationScript    string                      // Script to install extensions
	ValidationScript      string                      // Script to validate extensions
	MissingExtensions     []string                    // Extensions we don't know about
	AuthService           AuthServiceType             // Detected authentication service
	AuthCompatibilitySQL  string                      // SQL to create service compatibility
}

// AnalyzeMigrations scans migration content to detect required extensions
func (ed *ExtensionDetector) AnalyzeMigrations(migrations map[int]string) *ExtensionAnalysis {
	analysis := &ExtensionAnalysis{
		RequiredExtensions: []string{},
		ExtensionDetails:   make(map[string]ExtensionInfo),
		MissingExtensions:  []string{},
	}

	extensionSet := make(map[string]bool)

	log.Printf("Analyzing %d migrations for extension requirements", len(migrations))

	// Scan all migrations for extension references
	for migrationID, content := range migrations {
		extensions := ed.detectExtensionsInContent(content)
		log.Printf("Migration %d: found extensions %v", migrationID, extensions)

		for _, ext := range extensions {
			extensionSet[ext] = true
		}
	}

	// Process detected extensions
	for ext := range extensionSet {
		if info, known := ed.extensionMap[strings.ToLower(ext)]; known {
			analysis.RequiredExtensions = append(analysis.RequiredExtensions, ext)
			analysis.ExtensionDetails[ext] = info

			// Add dependencies
			for _, dep := range info.Dependencies {
				if !extensionSet[dep] {
					analysis.RequiredExtensions = append(analysis.RequiredExtensions, dep)
					if depInfo, exists := ed.extensionMap[dep]; exists {
						analysis.ExtensionDetails[dep] = depInfo
					}
				}
			}
		} else {
			analysis.MissingExtensions = append(analysis.MissingExtensions, ext)
			log.Printf("Warning: Unknown extension detected: %s", ext)
		}
	}

	// Sort for consistent output
	sort.Strings(analysis.RequiredExtensions)
	sort.Strings(analysis.MissingExtensions)

	// Determine best Docker base image
	analysis.RecommendedDockerBase = ed.selectBestDockerImage(analysis.ExtensionDetails)

	// Detect authentication service
	analysis.AuthService = ed.detectAuthService(migrations)
	analysis.AuthCompatibilitySQL = ed.generateAuthCompatibilitySQL(analysis.AuthService)

	// Generate installation and validation scripts
	analysis.InstallationScript = ed.generateInstallationScript(analysis.ExtensionDetails)
	analysis.ValidationScript = ed.generateValidationScript(analysis.ExtensionDetails)

	log.Printf("Extension analysis complete: %d extensions, auth service: %s, base image: %s",
		len(analysis.RequiredExtensions), analysis.AuthService, analysis.RecommendedDockerBase)

	return analysis
}

// detectExtensionsInContent scans SQL content for extension references
func (ed *ExtensionDetector) detectExtensionsInContent(content string) []string {
	var extensions []string

	// Pattern 1: Explicit CREATE EXTENSION statements
	createExtRegex := regexp.MustCompile(`(?i)CREATE\s+EXTENSION\s+(?:IF\s+NOT\s+EXISTS\s+)?(?:"?)([a-zA-Z0-9_-]+)(?:"?)`)
	matches := createExtRegex.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 1 {
			extensions = append(extensions, strings.TrimSpace(match[1]))
		}
	}

	// Pattern 2: Function calls that indicate extension usage
	extensionIndicators := map[string][]string{
		"postgis": {
			"ST_", "st_", "geometry", "geography", "PostGIS",
			"ST_Point", "ST_Distance", "ST_Contains", "ST_Within",
		},
		"uuid-ossp": {
			"uuid_generate_v1", "uuid_generate_v4", "uuid_generate_v1mc",
			"uuid_generate_v3", "uuid_generate_v5", "gen_random_uuid",
		},
		"earthdistance": {
			"earth_distance", "ll_to_earth", "earth_box", "earth",
		},
		"pg_trgm": {
			"similarity", "word_similarity", "show_trgm", "show_limit",
		},
		"cube": {
			"cube(", "cube_distance", "cube_dim", "cube_ll_coord",
		},
		"pg_stat_statements": {
			"pg_stat_statements", "pg_stat_statements_reset",
		},
		"btree_gin": {
			"gin_extract_value", "gin_extract_query", "gin_consistent",
		},
	}

	contentLower := strings.ToLower(content)

	for extension, indicators := range extensionIndicators {
		for _, indicator := range indicators {
			if strings.Contains(contentLower, strings.ToLower(indicator)) {
				// Avoid duplicates
				found := false
				for _, existing := range extensions {
					if strings.ToLower(existing) == extension {
						found = true
						break
					}
				}
				if !found {
					extensions = append(extensions, extension)
				}
				break // Found one indicator, no need to check others for this extension
			}
		}
	}

	return extensions
}

// selectBestDockerImage determines the best Docker image based on required extensions
// Migrated to Debian-based images for production reliability
func (ed *ExtensionDetector) selectBestDockerImage(extensionDetails map[string]ExtensionInfo) string {
	// Priority order for Docker images (Debian-based)
	imagePriority := map[string]int{
		"postgis/postgis:15-3.3": 1, // PostGIS (Debian-based, includes most extensions)
		"postgres:15":            2, // Standard PostgreSQL (Debian-based)
	}

	bestImage := "postgres:15" // Default (Debian-based)
	bestPriority := 999

	// Check if PostGIS is required
	for _, info := range extensionDetails {
		if info.RequiresPostGIS || strings.Contains(strings.ToLower(info.Name), "postgis") {
			return "postgis/postgis:15-3.3" // Debian-based PostGIS image
		}

		if priority, exists := imagePriority[info.DockerImage]; exists && priority < bestPriority {
			bestImage = info.DockerImage
			bestPriority = priority
		}
	}

	return bestImage
}

// generateInstallationScript creates a script to install required extensions
func (ed *ExtensionDetector) generateInstallationScript(extensionDetails map[string]ExtensionInfo) string {
	var script strings.Builder

	script.WriteString("#!/bin/bash\n")
	script.WriteString("# Auto-generated extension installation script\n")
	script.WriteString("set -e\n\n")

	// APT packages to install
	packages := make(map[string]bool)
	for _, info := range extensionDetails {
		if info.PackageName != "" {
			packages[info.PackageName] = true
		}
	}

	if len(packages) > 0 {
		script.WriteString("# Install required packages\n")
		script.WriteString("apt-get update && apt-get install -y")
		for pkg := range packages {
			script.WriteString(" " + pkg)
		}
		script.WriteString("\n\n")
	}

	// Custom installation commands
	for _, info := range extensionDetails {
		if info.InstallCommand != "" {
			script.WriteString(fmt.Sprintf("# Configure %s\n", info.Name))
			script.WriteString("echo \"" + info.InstallCommand + "\" >> /etc/postgresql/postgresql.conf\n\n")
		}
	}

	script.WriteString("# Restart PostgreSQL if needed\n")
	script.WriteString("# service postgresql restart\n")

	return script.String()
}

// generateValidationScript creates a script to validate extension availability
func (ed *ExtensionDetector) generateValidationScript(extensionDetails map[string]ExtensionInfo) string {
	var script strings.Builder

	script.WriteString("-- Auto-generated extension validation script\n")
	script.WriteString("\\set ON_ERROR_STOP on\n\n")

	// Sort extensions for consistent output
	var extensions []string
	for name := range extensionDetails {
		extensions = append(extensions, name)
	}
	sort.Strings(extensions)

	for _, name := range extensions {
		info := extensionDetails[name]
		script.WriteString(fmt.Sprintf("-- Validate %s extension\n", name))

		if info.ValidationSQL != "" {
			script.WriteString("DO $$\n")
			script.WriteString("BEGIN\n")
			script.WriteString(fmt.Sprintf("  PERFORM %s\n", info.ValidationSQL))
			script.WriteString("EXCEPTION\n")
			script.WriteString(fmt.Sprintf("  WHEN OTHERS THEN\n"))
			script.WriteString(fmt.Sprintf("    RAISE NOTICE 'Extension %s not available: %%', SQLERRM;\n", name))
			script.WriteString("END\n")
			script.WriteString("$$;\n\n")
		}
	}

	return script.String()
}

// GenerateDockerfile creates a Dockerfile with required extensions
func (ed *ExtensionDetector) GenerateDockerfile(analysis *ExtensionAnalysis) string {
	var dockerfile strings.Builder

	dockerfile.WriteString(fmt.Sprintf("FROM %s\n\n", analysis.RecommendedDockerBase))

	if len(analysis.ExtensionDetails) > 0 {
		dockerfile.WriteString("# Install required PostgreSQL extensions\n")
		dockerfile.WriteString("USER root\n")

		// Collect APT packages
		packages := make(map[string]bool)
		for _, info := range analysis.ExtensionDetails {
			if info.PackageName != "" {
				packages[info.PackageName] = true
			}
		}

		if len(packages) > 0 {
			dockerfile.WriteString("RUN apt-get update && apt-get install -y")
			for pkg := range packages {
				dockerfile.WriteString(" " + pkg)
			}
			dockerfile.WriteString(" && apt-get clean && rm -rf /var/lib/apt/lists/*\n")
		}

		dockerfile.WriteString("USER postgres\n\n")
	}

	dockerfile.WriteString("# Copy initialization scripts\n")
	dockerfile.WriteString("COPY init-extensions.sql /docker-entrypoint-initdb.d/01-init-extensions.sql\n")

	return dockerfile.String()
}

// GenerateInitSQL creates an SQL script to initialize extensions
func (ed *ExtensionDetector) GenerateInitSQL(analysis *ExtensionAnalysis) string {
	var script strings.Builder

	script.WriteString("-- Auto-generated extension initialization script\n")
	script.WriteString("-- This script creates required extensions for migration validation\n\n")

	// Sort extensions to ensure consistent dependency order
	extensionOrder := ed.getExtensionOrder(analysis.ExtensionDetails)

	for _, extName := range extensionOrder {
		info := analysis.ExtensionDetails[extName]
		cascade := ""
		if info.RequiresCASCADE {
			cascade = " CASCADE"
		}

		script.WriteString(fmt.Sprintf("-- Create %s extension\n", extName))
		script.WriteString(fmt.Sprintf("CREATE EXTENSION IF NOT EXISTS \"%s\"%s;\n\n", extName, cascade))
	}

	return script.String()
}

// getExtensionOrder returns extensions in dependency order
func (ed *ExtensionDetector) getExtensionOrder(extensionDetails map[string]ExtensionInfo) []string {
	var ordered []string
	processed := make(map[string]bool)

	// Simple dependency resolution - could be enhanced with topological sort
	for len(ordered) < len(extensionDetails) {
		for name, info := range extensionDetails {
			if processed[name] {
				continue
			}

			// Check if all dependencies are processed
			canProcess := true
			for _, dep := range info.Dependencies {
				if _, exists := extensionDetails[dep]; exists && !processed[dep] {
					canProcess = false
					break
				}
			}

			if canProcess {
				ordered = append(ordered, name)
				processed[name] = true
			}
		}
	}

	return ordered
}

// detectAuthService analyzes migrations to detect authentication service patterns
func (ed *ExtensionDetector) detectAuthService(migrations map[int]string) AuthServiceType {
	for _, content := range migrations {
		contentLower := strings.ToLower(content)

		// Clerk patterns - most specific first
		if strings.Contains(content, "auth.jwt()") && strings.Contains(content, "'o'->>'id'") {
			log.Printf("Detected Clerk authentication service")
			return AuthServiceClerk
		}

		// Supabase patterns - look for actual function calls, not just mentions
		if strings.Contains(content, "auth.users") ||
		   (strings.Contains(content, "auth.uid()") && !strings.Contains(content, "No legacy auth.uid()")) {
			log.Printf("Detected Supabase authentication service")
			return AuthServiceSupabase
		}

		// Auth0 patterns - more specific
		if strings.Contains(contentLower, "auth0") ||
		   (strings.Contains(content, "\"sub\"") && strings.Contains(content, "\"iss\"")) {
			log.Printf("Detected Auth0 authentication service")
			return AuthServiceAuth0
		}
	}

	return AuthServiceNone
}

// generateAuthCompatibilitySQL creates SQL to mock authentication service functions
func (ed *ExtensionDetector) generateAuthCompatibilitySQL(authService AuthServiceType) string {
	switch authService {
	case AuthServiceClerk:
		return `-- Clerk Authentication Compatibility Layer
CREATE SCHEMA IF NOT EXISTS auth;

-- Create common Supabase/Clerk roles (used in RLS policies)
DO $$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'anon') THEN
    CREATE ROLE anon NOLOGIN;
  END IF;
  IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'authenticated') THEN
    CREATE ROLE authenticated NOLOGIN;
  END IF;
  IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'service_role') THEN
    CREATE ROLE service_role NOLOGIN;
  END IF;
END
$$;

-- Mock Clerk auth.jwt() function
CREATE OR REPLACE FUNCTION auth.jwt() RETURNS jsonb AS $$
BEGIN
  -- Return a mock Clerk JWT payload for validation purposes
  RETURN jsonb_build_object(
    'o', jsonb_build_object(
      'id', 'org_mock_organization_id',
      'role', 'admin',
      'name', 'Mock Organization',
      'slug', 'mock-org'
    ),
    'sub', 'user_mock_user_id',
    'email', 'mock@clerk.dev',
    'email_verified', true,
    'iat', extract(epoch from now()),
    'exp', extract(epoch from now()) + 3600,
    'iss', 'https://mock-clerk.clerk.accounts.dev'
  );
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- Mock current_user_id helper (common in Clerk setups)
CREATE OR REPLACE FUNCTION current_user_id() RETURNS text AS $$
BEGIN
  RETURN (auth.jwt() ->> 'sub')::text;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- Mock organization helpers
CREATE OR REPLACE FUNCTION current_organization_id() RETURNS text AS $$
BEGIN
  RETURN (auth.jwt()->'o'->>'id')::text;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

CREATE OR REPLACE FUNCTION current_organization_role() RETURNS text AS $$
BEGIN
  RETURN (auth.jwt()->'o'->>'role')::text;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

CREATE OR REPLACE FUNCTION current_organization_name() RETURNS text AS $$
BEGIN
  RETURN (auth.jwt()->'o'->>'name')::text;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- Create Supabase Realtime publication if it doesn't exist (often used with Clerk)
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_publication WHERE pubname = 'supabase_realtime') THEN
    CREATE PUBLICATION supabase_realtime;
  END IF;
END
$$;`

	case AuthServiceSupabase:
		return `-- Supabase Authentication Compatibility Layer
CREATE SCHEMA IF NOT EXISTS auth;

-- Create Supabase roles (used in RLS policies)
DO $$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'anon') THEN
    CREATE ROLE anon NOLOGIN;
  END IF;
  IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'authenticated') THEN
    CREATE ROLE authenticated NOLOGIN;
  END IF;
  IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'service_role') THEN
    CREATE ROLE service_role NOLOGIN;
  END IF;
END
$$;

-- Mock Supabase auth.uid() function
CREATE OR REPLACE FUNCTION auth.uid() RETURNS uuid AS $$
BEGIN
  RETURN 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11'::uuid;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- Mock Supabase auth.jwt() function
CREATE OR REPLACE FUNCTION auth.jwt() RETURNS jsonb AS $$
BEGIN
  RETURN jsonb_build_object(
    'sub', 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
    'email', 'mock@supabase.io',
    'role', 'authenticated',
    'iat', extract(epoch from now()),
    'exp', extract(epoch from now()) + 3600
  );
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- Create Supabase Realtime publication if it doesn't exist
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_publication WHERE pubname = 'supabase_realtime') THEN
    CREATE PUBLICATION supabase_realtime;
  END IF;
END
$$;`

	case AuthServiceAuth0:
		return `-- Auth0 Authentication Compatibility Layer
CREATE SCHEMA IF NOT EXISTS auth;

-- Mock Auth0 JWT function
CREATE OR REPLACE FUNCTION auth.jwt() RETURNS jsonb AS $$
BEGIN
  RETURN jsonb_build_object(
    'sub', 'auth0|mock_user_id',
    'email', 'mock@auth0.com',
    'email_verified', true,
    'iss', 'https://mock.auth0.com/',
    'aud', 'mock_audience',
    'iat', extract(epoch from now()),
    'exp', extract(epoch from now()) + 3600
  );
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;`

	default:
		return "-- No authentication service detected"
	}
}