package supabase

import (
    "context"
    "regexp"
    "strings"
)

// InjectCompatibilityLayer returns SQL to mock Supabase authentication for validation
// This creates:
//   - auth schema
//   - auth.users table (stub for foreign key references)
//   - Supabase roles (anon, authenticated, service_role)
//   - Mock auth.uid() function
//   - Mock auth.jwt() function
//   - Supabase Realtime publication
func (sp *SupabasePlugin) InjectCompatibilityLayer(ctx context.Context) string {
    return `-- Supabase Authentication Compatibility Layer
CREATE SCHEMA IF NOT EXISTS auth;

-- Create auth.users table stub for foreign key references
-- This allows migrations to reference auth.users(id) without errors
CREATE TABLE IF NOT EXISTS auth.users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT,
    encrypted_password TEXT,
    email_confirmed_at TIMESTAMPTZ,
    invited_at TIMESTAMPTZ,
    confirmation_token TEXT,
    confirmation_sent_at TIMESTAMPTZ,
    recovery_token TEXT,
    recovery_sent_at TIMESTAMPTZ,
    email_change_token_new TEXT,
    email_change TEXT,
    email_change_sent_at TIMESTAMPTZ,
    last_sign_in_at TIMESTAMPTZ,
    raw_app_meta_data JSONB,
    raw_user_meta_data JSONB,
    is_super_admin BOOLEAN,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    phone TEXT,
    phone_confirmed_at TIMESTAMPTZ,
    phone_change TEXT,
    phone_change_token TEXT,
    phone_change_sent_at TIMESTAMPTZ,
    confirmed_at TIMESTAMPTZ,
    email_change_token_current TEXT,
    email_change_confirm_status SMALLINT,
    banned_until TIMESTAMPTZ,
    reauthentication_token TEXT,
    reauthentication_sent_at TIMESTAMPTZ,
    is_sso_user BOOLEAN DEFAULT FALSE,
    deleted_at TIMESTAMPTZ
);

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
$$ LANGUAGE plpgsql SECURITY DEFINER STABLE;

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
$$ LANGUAGE plpgsql SECURITY DEFINER STABLE;

-- Create Supabase Realtime publication if it doesn't exist
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_publication WHERE pubname = 'supabase_realtime') THEN
    CREATE PUBLICATION supabase_realtime;
  END IF;
END
$$;`
}

// TransformSQL performs Supabase-specific SQL transformations
// Currently delegates to FixFunctionVolatility
func (sp *SupabasePlugin) TransformSQL(ctx context.Context, sql string) (string, error) {
    // Apply function volatility fixes for Supabase auth functions
    return sp.FixFunctionVolatility(ctx, sql)
}

// FixFunctionVolatility adds STABLE markers to Supabase auth functions
//
// Supabase auth functions MUST be marked STABLE because they:
//   - Read session state (JWT claims, current user ID)
//   - Are used in RLS policies and index predicates
//   - Return consistent values within a transaction
//
// This transformation:
//   1. Detects Supabase auth functions (auth.uid, auth.jwt, auth.role)
//   2. Checks if they already have volatility markers
//   3. Adds STABLE if missing
func (sp *SupabasePlugin) FixFunctionVolatility(ctx context.Context, functionSQL string) (string, error) {
    // Pattern: CREATE [OR REPLACE] FUNCTION [schema.]name(...) RETURNS type
    //          [LANGUAGE plpgsql] [SECURITY DEFINER]
    //          AS $$...
    //
    // We need to inject STABLE before AS
    funcPattern := regexp.MustCompile(
        `(?ims)(CREATE\s+(?:OR\s+REPLACE\s+)?FUNCTION\s+(?:auth\.)?([a-z_][a-z0-9_]*)\s*\([^)]*\)\s*RETURNS\s+(?:TABLE\s*\([^)]+\)|SETOF\s+[^\s]+|[^\s]+))((?:\s+(?:LANGUAGE\s+[a-z]+|SECURITY\s+DEFINER|SET\s+[^\s]+\s*=\s*[^\s]+))*?)(\s+AS\s+(?:\$\$|\$[a-z0-9_]*\$|\'))`,
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

        // Skip if not a Supabase auth function
        if !sp.isSupabaseAuthFunctionName(funcName) {
            continue
        }

        // Extract parts
        beforeModifiers := transformedSQL[match[0]+offset : match[2]+offset] // CREATE FUNCTION ... RETURNS type
        modifiers := transformedSQL[match[6]+offset : match[7]+offset]       // LANGUAGE/SECURITY DEFINER
        asKeyword := transformedSQL[match[8]+offset : match[9]+offset]       // AS $$

        // Check if volatility marker already exists
        if sp.hasVolatilityMarker(modifiers) || sp.hasVolatilityMarker(beforeModifiers) {
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

// isSupabaseAuthFunctionName checks if function name is a Supabase auth helper
func (sp *SupabasePlugin) isSupabaseAuthFunctionName(name string) bool {
    supabaseFunctions := map[string]bool{
        "uid":  true, // auth.uid()
        "jwt":  true, // auth.jwt()
        "role": true, // auth.role()
    }

    return supabaseFunctions[strings.ToLower(name)]
}

// hasVolatilityMarker checks if a string contains a volatility marker
func (sp *SupabasePlugin) hasVolatilityMarker(s string) bool {
    upper := strings.ToUpper(s)
    return strings.Contains(upper, " IMMUTABLE") ||
           strings.Contains(upper, " STABLE") ||
           strings.Contains(upper, " VOLATILE")
}
