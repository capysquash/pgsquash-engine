// Package plugins provides a public API for pgsquash plugin management.
//
// This package exports plugin registration functionality for use by external tools
// while keeping the internal plugin implementation details private.
package plugins

import (
	internal_plugins "github.com/CAPYSQUASH/pgsquash-engine/internal/plugins"
	"github.com/CAPYSQUASH/pgsquash-engine/internal/plugins/clerk"
	"github.com/CAPYSQUASH/pgsquash-engine/internal/plugins/drizzle"
	"github.com/CAPYSQUASH/pgsquash-engine/internal/plugins/prisma"
	"github.com/CAPYSQUASH/pgsquash-engine/internal/plugins/supabase"
)

// RegisterDefault registers all built-in pgsquash plugins.
// This includes plugins for popular platforms and ORMs:
//   - Supabase (RLS policies, storage, auth)
//   - Clerk (JWT v2 auth)
//   - Prisma (ORM migrations)
//   - Drizzle (ORM migrations)
//
// This should be called during application initialization, typically in init().
// Registration failures are logged as warnings but don't prevent startup.
//
// Example:
//   if err := plugins.RegisterDefault(); err != nil {
//       log.Printf("Warning: Some plugins failed to register: %v", err)
//   }
func RegisterDefault() error {
	// Register auth service plugins
	if err := internal_plugins.Register(clerk.NewClerkPlugin()); err != nil {
		return err
	}

	if err := internal_plugins.Register(supabase.NewSupabasePlugin()); err != nil {
		return err
	}

	// Register ORM plugins
	if err := internal_plugins.Register(prisma.NewPrismaPlugin()); err != nil {
		return err
	}

	if err := internal_plugins.Register(drizzle.NewDrizzlePlugin()); err != nil {
		return err
	}

	return nil
}

// Note: Custom plugin registration is not exposed in the public API.
// The Plugin interface is complex and considered internal.
// If you need custom plugin support, please file an issue at:
// https://github.com/CAPYSQUASH/pgsquash-engine/issues
