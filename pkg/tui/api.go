// Package tui provides a public API for the Terminal User Interface (TUI).
// This package exposes the interactive terminal interface for pgsquash-engine,
// allowing external tools like capysquash-cli to integrate the TUI functionality.
package tui

import (
	"fmt"

	"github.com/CAPYSQUASH/pgsquash-engine/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
)

// Model represents the TUI application model.
// This is an opaque wrapper around the internal TUI model.
type Model struct {
	internal *tui.Model
}

// Options contains configuration options for the TUI.
type Options struct {
	// MigrationDir is the directory containing migration files
	MigrationDir string

	// ConfigPath is the path to the configuration file
	ConfigPath string

	// InitialView specifies which view to show on startup (optional)
	InitialView ViewType

	// AltScreen enables alternate screen buffer (recommended: true)
	AltScreen bool
}

// ViewType represents different views in the TUI.
type ViewType int

const (
	// ViewDashboard is the main dashboard view
	ViewDashboard ViewType = ViewType(tui.ViewDashboard)

	// ViewAnalysis is the migration analysis view
	ViewAnalysis ViewType = ViewType(tui.ViewAnalysis)

	// ViewConfig is the configuration wizard view
	ViewConfig ViewType = ViewType(tui.ViewConfig)

	// ViewDependencyGraph is the dependency graph visualization view
	ViewDependencyGraph ViewType = ViewType(tui.ViewDependencyGraph)

	// ViewProgress is the operation progress view
	ViewProgress ViewType = ViewType(tui.ViewProgress)

	// ViewHelp is the help and keyboard shortcuts view
	ViewHelp ViewType = ViewType(tui.ViewHelp)
)

// New creates a new TUI model with the given options.
//
// Example:
//
//	model := tui.New(tui.Options{
//		MigrationDir: "./migrations",
//		ConfigPath:   ".pgsquash.config.json",
//		AltScreen:    true,
//	})
func New(opts Options) *Model {
	// Set defaults
	if opts.MigrationDir == "" {
		opts.MigrationDir = "."
	}
	if opts.ConfigPath == "" {
		opts.ConfigPath = "pgsquash.config.json"
	}

	internal := tui.NewModel(opts.MigrationDir, opts.ConfigPath)

	return &Model{
		internal: internal,
	}
}

// Run starts the TUI and blocks until the user exits.
// Returns an error if the TUI encounters a fatal error.
//
// Example:
//
//	model := tui.New(tui.Options{
//		MigrationDir: "./migrations",
//		AltScreen:    true,
//	})
//	if err := model.Run(); err != nil {
//		log.Fatal(err)
//	}
func (m *Model) Run() error {
	p := tea.NewProgram(m.internal, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}
	return nil
}

// RunWithView starts the TUI and immediately navigates to the specified view.
// This is useful for launching directly into a specific workflow.
//
// Example:
//
//	model := tui.New(tui.Options{
//		MigrationDir: "./migrations",
//		AltScreen:    true,
//	})
//	// Launch directly into analysis view
//	if err := model.RunWithView(tui.ViewAnalysis); err != nil {
//		log.Fatal(err)
//	}
func (m *Model) RunWithView(view ViewType) error {
	p := tea.NewProgram(m.internal, tea.WithAltScreen())

	// Send navigation message after program starts
	go func() {
		p.Send(tui.NavigateMsg{View: tui.ViewType(view)})
	}()

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}
	return nil
}

// RunWithOptions starts the TUI with custom tea.ProgramOption settings.
// This provides full control over the Bubbletea program configuration.
//
// Example:
//
//	model := tui.New(tui.Options{
//		MigrationDir: "./migrations",
//	})
//	opts := []tea.ProgramOption{
//		tea.WithAltScreen(),
//		tea.WithMouseCellMotion(),
//	}
//	if err := model.RunWithOptions(opts...); err != nil {
//		log.Fatal(err)
//	}
func (m *Model) RunWithOptions(opts ...tea.ProgramOption) error {
	p := tea.NewProgram(m.internal, opts...)
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}
	return nil
}

// Launch is a convenience function that creates and runs a TUI in one step.
// This is the simplest way to start the TUI with default settings.
//
// Example:
//
//	if err := tui.Launch("./migrations", "pgsquash.config.json"); err != nil {
//		log.Fatal(err)
//	}
func Launch(migrationDir, configPath string) error {
	model := New(Options{
		MigrationDir: migrationDir,
		ConfigPath:   configPath,
		AltScreen:    true,
	})
	return model.Run()
}

// LaunchWithView is a convenience function that creates and runs a TUI,
// immediately navigating to the specified view.
//
// Example:
//
//	// Launch directly into analysis view
//	if err := tui.LaunchWithView("./migrations", "", tui.ViewAnalysis); err != nil {
//		log.Fatal(err)
//	}
func LaunchWithView(migrationDir, configPath string, view ViewType) error {
	model := New(Options{
		MigrationDir: migrationDir,
		ConfigPath:   configPath,
		AltScreen:    true,
	})
	return model.RunWithView(view)
}
