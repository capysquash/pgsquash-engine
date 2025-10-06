package main

import (
	"fmt"
	"os"

	"github.com/capysquash/pg-squash-engine/internal/cli"
	"github.com/capysquash/pg-squash-engine/internal/plugins"
	"github.com/capysquash/pg-squash-engine/internal/plugins/clerk"
	"github.com/capysquash/pg-squash-engine/internal/plugins/drizzle"
	"github.com/capysquash/pg-squash-engine/internal/plugins/prisma"
	"github.com/capysquash/pg-squash-engine/internal/plugins/supabase"
)

func init() {
	// Register all available plugins at application startup
	// This must happen before any CLI commands execute
	registerPlugins()
}

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// registerPlugins registers all built-in plugins with the global registry
// Plugins are registered early so they're available for auto-discovery during migration processing
func registerPlugins() {
	// Register auth service plugins
	if err := plugins.Register(clerk.NewClerkPlugin()); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to register Clerk plugin: %v\n", err)
	}

	if err := plugins.Register(supabase.NewSupabasePlugin()); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to register Supabase plugin: %v\n", err)
	}

	// Register ORM plugins
	if err := plugins.Register(prisma.NewPrismaPlugin()); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to register Prisma plugin: %v\n", err)
	}

	if err := plugins.Register(drizzle.NewDrizzlePlugin()); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to register Drizzle plugin: %v\n", err)
	}

	// Future plugins can be registered here:
	// plugins.Register(auth0.NewAuth0Plugin())
	// plugins.Register(nextauth.NewNextAuthPlugin())
}