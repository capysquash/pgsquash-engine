package cli

import (
    "fmt"
    "os"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/CAPYSQUASH/pgsquash-engine/internal/tui"
    "github.com/spf13/cobra"
)

var tuiCmd = &cobra.Command{
    Use:   "tui [migrations-dir]",
    Short: "Launch interactive TUI for migration analysis and squashing",
    Long: `Launch the interactive terminal user interface (TUI) for pgsquash.

The TUI provides a visual interface for:
  • Analyzing migrations and viewing lifecycle patterns
  • Configuring squashing settings interactively
  • Visualizing dependency graphs
  • Monitoring squashing progress in real-time

Examples:
  pgsquash tui migrations/
  pgsquash tui
`,
    RunE: runTUI,
}

var tuiAnalyzeCmd = &cobra.Command{
    Use:   "analyze [migrations-dir]",
    Short: "Launch TUI directly in analysis view",
    Long:  "Launch the TUI and immediately navigate to the analysis view.",
    RunE:  runTUIAnalyze,
}

var tuiConfigCmd = &cobra.Command{
    Use:   "config [config-path]",
    Short: "Launch TUI directly in configuration wizard",
    Long:  "Launch the TUI and immediately navigate to the configuration wizard.",
    RunE:  runTUIConfig,
}

var tuiDepGraphCmd = &cobra.Command{
    Use:   "deps [migrations-dir]",
    Short: "Launch TUI directly in dependency graph view",
    Long:  "Launch the TUI and immediately navigate to the dependency graph visualization.",
    RunE:  runTUIDepGraph,
}

func init() {
    // Add subcommands
    tuiCmd.AddCommand(tuiAnalyzeCmd)
    tuiCmd.AddCommand(tuiConfigCmd)
    tuiCmd.AddCommand(tuiDepGraphCmd)

    // Add to root command
    rootCmd.AddCommand(tuiCmd)
}

// runTUI runs the main TUI
func runTUI(cmd *cobra.Command, args []string) error {
    migrationDir := "."
    if len(args) > 0 {
        migrationDir = args[0]
    }

    // Verify migration directory exists
    if _, err := os.Stat(migrationDir); err != nil {
        if os.IsNotExist(err) {
            return fmt.Errorf("migration directory does not exist: %s", migrationDir)
        }
        return fmt.Errorf("failed to access migration directory: %w", err)
    }

    // Get config path from flag or default
    configPath, _ := cmd.Flags().GetString("config")

    // Create TUI model
    model := tui.NewModel(migrationDir, configPath)

    // Run the TUI
    p := tea.NewProgram(model, tea.WithAltScreen())
    if _, err := p.Run(); err != nil {
        return fmt.Errorf("TUI error: %w", err)
    }

    return nil
}

// runTUIAnalyze runs the TUI and navigates to analysis view
func runTUIAnalyze(cmd *cobra.Command, args []string) error {
    migrationDir := "."
    if len(args) > 0 {
        migrationDir = args[0]
    }

    // Verify migration directory exists
    if _, err := os.Stat(migrationDir); err != nil {
        if os.IsNotExist(err) {
            return fmt.Errorf("migration directory does not exist: %s", migrationDir)
        }
        return fmt.Errorf("failed to access migration directory: %w", err)
    }

    configPath, _ := cmd.Flags().GetString("config")

    // Create TUI model
    model := tui.NewModel(migrationDir, configPath)

    // Send navigation command to switch to analysis view
    // This will be handled after the model is initialized
    p := tea.NewProgram(model, tea.WithAltScreen())

    // We can send an initial message to navigate
    go func() {
        p.Send(tui.NavigateMsg{View: tui.ViewAnalysis})
    }()

    if _, err := p.Run(); err != nil {
        return fmt.Errorf("TUI error: %w", err)
    }

    return nil
}

// runTUIConfig runs the TUI and navigates to configuration view
func runTUIConfig(cmd *cobra.Command, args []string) error {
    configPath := "pgsquash.config.json"
    if len(args) > 0 {
        configPath = args[0]
    }

    // Create TUI model with current directory
    model := tui.NewModel(".", configPath)

    p := tea.NewProgram(model, tea.WithAltScreen())

    // Navigate to config view
    go func() {
        p.Send(tui.NavigateMsg{View: tui.ViewConfig})
    }()

    if _, err := p.Run(); err != nil {
        return fmt.Errorf("TUI error: %w", err)
    }

    return nil
}

// runTUIDepGraph runs the TUI and navigates to dependency graph view
func runTUIDepGraph(cmd *cobra.Command, args []string) error {
    migrationDir := "."
    if len(args) > 0 {
        migrationDir = args[0]
    }

    // Verify migration directory exists
    if _, err := os.Stat(migrationDir); err != nil {
        if os.IsNotExist(err) {
            return fmt.Errorf("migration directory does not exist: %s", migrationDir)
        }
        return fmt.Errorf("failed to access migration directory: %w", err)
    }

    configPath, _ := cmd.Flags().GetString("config")

    // Create TUI model
    model := tui.NewModel(migrationDir, configPath)

    p := tea.NewProgram(model, tea.WithAltScreen())

    // Navigate to dependency graph view
    go func() {
        p.Send(tui.NavigateMsg{View: tui.ViewDependencyGraph})
    }()

    if _, err := p.Run(); err != nil {
        return fmt.Errorf("TUI error: %w", err)
    }

    return nil
}
