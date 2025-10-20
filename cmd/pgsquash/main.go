package main

import (
	"os"

	"github.com/CAPYSQUASH/pgsquash-engine/internal/cli"
	"github.com/CAPYSQUASH/pgsquash-engine/internal/errors"
	"github.com/CAPYSQUASH/pgsquash-engine/internal/plugins"
	"github.com/CAPYSQUASH/pgsquash-engine/internal/plugins/clerk"
	"github.com/CAPYSQUASH/pgsquash-engine/internal/plugins/drizzle"
	"github.com/CAPYSQUASH/pgsquash-engine/internal/plugins/prisma"
	"github.com/CAPYSQUASH/pgsquash-engine/internal/plugins/supabase"
	"github.com/CAPYSQUASH/pgsquash-engine/internal/utils"
)

// pgsquash: The PostgreSQL migration consolidation engine
//
// Intelligently consolidates migration histories using parser-grade accuracy,
// dependency-aware processing, and safety-first validation strategies.

// Version information (set via ldflags during build)
var (
	version   = "0.8.5-beta" // Default version, can be overridden via ldflags: -ldflags "-X main.version=x.y.z"
	buildDate = "unknown"
	gitCommit = "unknown"
)

func init() {
	// Initialize global logger (use INFO level by default, can be configured via env var)
	logLevel := utils.LogLevelInfo
	if os.Getenv("PGSQUASH_LOG_LEVEL") == "debug" {
		logLevel = utils.LogLevelDebug
	}
	logger := utils.NewLogger(logLevel, os.Stdout)
	utils.SetDefaultLogger(logger)

	// Set version information for CLI commands
	cli.SetVersionInfo(version, buildDate, gitCommit)

	// Register all available plugins at application startup
	// This must happen before any CLI commands execute
	registerPlugins()
}

func main() {
	logger := utils.GetDefaultLogger()

	if err := cli.Execute(); err != nil {
		// Use structured error if possible
		if structErr, ok := err.(*errors.StructuredError); ok {
			logger.Error("%s", structErr.Error())
			if structErr.Severity == errors.SeverityCritical {
				os.Exit(2)
			}
		} else {
			logger.Error("Command execution failed: %v", err)
		}
		os.Exit(1)
	}
}

// registerPlugins registers all built-in plugins with the global registry
// Plugins are registered early so they're available for auto-discovery during migration processing
func registerPlugins() {
	logger := utils.GetDefaultLogger().WithPrefix("PLUGINS")

	// Register auth service plugins
	if err := plugins.Register(clerk.NewClerkPlugin()); err != nil {
		logger.Warn("Failed to register Clerk plugin: %v", err)
	}

	if err := plugins.Register(supabase.NewSupabasePlugin()); err != nil {
		logger.Warn("Failed to register Supabase plugin: %v", err)
	}

	// Register ORM plugins
	if err := plugins.Register(prisma.NewPrismaPlugin()); err != nil {
		logger.Warn("Failed to register Prisma plugin: %v", err)
	}

	if err := plugins.Register(drizzle.NewDrizzlePlugin()); err != nil {
		logger.Warn("Failed to register Drizzle plugin: %v", err)
	}

	// Future plugins can be registered here:
	// plugins.Register(auth0.NewAuth0Plugin())
	// plugins.Register(nextauth.NewNextAuthPlugin())
}