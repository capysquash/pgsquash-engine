package clerk

import (
    "context"
    "regexp"
    "strings"
)

// InjectCompatibilityLayer returns SQL to mock Clerk authentication for validation
// This creates:
//   - auth schema
//   - Common Clerk/Supabase roles (anon, authenticated, service_role)
//   - Mock auth.jwt() function returning Clerk JWT v2 payload
//   - Helper functions (current_user_id, current_organization_id, etc.)
func (cp *ClerkPlugin) InjectCompatibilityLayer(ctx context.Context) string {
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
$$ LANGUAGE plpgsql SECURITY DEFINER STABLE;

-- Mock current_user_id helper (common in Clerk setups)
CREATE OR REPLACE FUNCTION current_user_id() RETURNS text AS $$
BEGIN
  RETURN (auth.jwt() ->> 'sub')::text;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER STABLE;

-- Mock organization helpers
CREATE OR REPLACE FUNCTION current_organization_id() RETURNS text AS $$
BEGIN
  RETURN (auth.jwt()->'o'->>'id')::text;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER STABLE;

CREATE OR REPLACE FUNCTION current_organization_role() RETURNS text AS $$
BEGIN
  RETURN (auth.jwt()->'o'->>'role')::text;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER STABLE;

CREATE OR REPLACE FUNCTION current_organization_name() RETURNS text AS $$
BEGIN
  RETURN (auth.jwt()->'o'->>'name')::text;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER STABLE;

-- Create Supabase Realtime publication if it doesn't exist (often used with Clerk)
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_publication WHERE pubname = 'supabase_realtime') THEN
    CREATE PUBLICATION supabase_realtime;
  END IF;
END
$$;`
}

// TransformSQL performs Clerk-specific SQL transformations
// Currently delegates to FixFunctionVolatility
func (cp *ClerkPlugin) TransformSQL(ctx context.Context, sql string) (string, error) {
    // Apply function volatility fixes for Clerk auth functions
    return cp.FixFunctionVolatility(ctx, sql)
}

// FixFunctionVolatility adds STABLE markers to Clerk auth functions
//
// Clerk auth functions MUST be marked STABLE because they:
//   - Read session state (auth.jwt())
//   - Are used in index predicates and RLS policies
//   - Return consistent values within a transaction
//
// PostgreSQL Requirements:
//   - Functions in index predicates MUST have volatility markers
//   - Functions accessing database state should be STABLE (not IMMUTABLE)
//   - STABLE allows index usage while maintaining correctness
//
// This transformation:
//   1. Detects Clerk auth functions (clerk_user_id, clerk_is_admin, etc.)
//   2. Checks if they already have volatility markers
//   3. Adds STABLE if missing
func (cp *ClerkPlugin) FixFunctionVolatility(ctx context.Context, functionSQL string) (string, error) {
    // Pattern: CREATE [OR REPLACE] FUNCTION [schema.]name(...) RETURNS type
    //          [LANGUAGE plpgsql] [SECURITY DEFINER]
    //          AS $$...
    //
    // We need to inject STABLE before AS
    funcPattern := regexp.MustCompile(
        `(?ims)(CREATE\s+(?:OR\s+REPLACE\s+)?FUNCTION\s+(?:[a-z_][a-z0-9_]*\.)?([a-z_][a-z0-9_]*)\s*\([^)]*\)\s*RETURNS\s+(?:TABLE\s*\([^)]+\)|SETOF\s+[^\s]+|[^\s]+))((?:\s+(?:LANGUAGE\s+[a-z]+|SECURITY\s+DEFINER|SET\s+[^\s]+\s*=\s*[^\s]+))*?)(\s+AS\s+(?:\$\$|\$[a-z0-9_]*\$|\'))`,
    )

    matches := funcPattern.FindAllStringSubmatchIndex(functionSQL, -1)
    if len(matches) == 0 {
        return functionSQL, nil // No functions found
    }

    transformedSQL := functionSQL
    offset := 0

    for _, match := range matches {
        if len(match) < 10 {
            continue
        }

        // Extract function name (capture group 2)
        funcNameStart := match[4] + offset
        funcNameEnd := match[5] + offset
        funcName := transformedSQL[funcNameStart:funcNameEnd]

        // Skip if not a Clerk auth function
        if !cp.isClerkAuthFunctionName(funcName) {
            continue
        }

        // Extract parts
        beforeModifiers := transformedSQL[match[0]+offset : match[2]+offset] // CREATE FUNCTION ... RETURNS type
        modifiers := transformedSQL[match[6]+offset : match[7]+offset]       // LANGUAGE/SECURITY DEFINER
        asKeyword := transformedSQL[match[8]+offset : match[9]+offset]       // AS $$

        // Check if volatility marker already exists
        if cp.hasVolatilityMarker(modifiers) || cp.hasVolatilityMarker(beforeModifiers) {
            continue
        }

        // Insert STABLE before AS
        newModifiers := modifiers + " STABLE"
        replacement := beforeModifiers + newModifiers + asKeyword

        transformedSQL = transformedSQL[:match[0]+offset] + replacement + transformedSQL[match[9]+offset:]
        offset += len(replacement) - (match[9] - match[0])
    }

    return transformedSQL, nil
}

// isClerkAuthFunctionName checks if function name is a Clerk auth helper
func (cp *ClerkPlugin) isClerkAuthFunctionName(name string) bool {
    clerkFunctions := map[string]bool{
        "clerk_user_id":             true,
        "clerk_is_admin":            true,
        "clerk_organization_id":     true,
        "current_user_id":           true,
        "current_organization_id":   true,
        "current_organization_role": true,
        "current_organization_name": true,
    }

    return clerkFunctions[strings.ToLower(name)]
}

// hasVolatilityMarker checks if a string contains a volatility marker
func (cp *ClerkPlugin) hasVolatilityMarker(s string) bool {
    upper := strings.ToUpper(s)
    return strings.Contains(upper, " IMMUTABLE") ||
           strings.Contains(upper, " STABLE") ||
           strings.Contains(upper, " VOLATILE")
}
