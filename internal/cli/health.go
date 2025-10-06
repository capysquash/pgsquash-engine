package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/spf13/cobra"
)

// HealthStatus represents the health check response
type HealthStatus struct {
	Status    string    `json:"status"`
	Version   string    `json:"version"`
	Timestamp time.Time `json:"timestamp"`
	System    struct {
		OS      string `json:"os"`
		Arch    string `json:"arch"`
		GoVer   string `json:"go_version"`
		NumCPU  int    `json:"num_cpu"`
		NumGo   int    `json:"num_goroutines"`
	} `json:"system"`
	Docker struct {
		Available bool   `json:"available"`
		Reason    string `json:"reason,omitempty"`
	} `json:"docker"`
}

var (
	healthJSON bool
)

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Health check endpoint for container orchestration",
	Long: `Provides a health check endpoint suitable for Docker HEALTHCHECK,
Kubernetes liveness/readiness probes, and other orchestration tools.

Returns exit code 0 if healthy, non-zero otherwise.
Output format can be plain text or JSON.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		status := HealthStatus{
			Status:    "healthy",
			Version:   getVersion(),
			Timestamp: time.Now().UTC(),
		}

		// System info
		status.System.OS = runtime.GOOS
		status.System.Arch = runtime.GOARCH
		status.System.GoVer = runtime.Version()
		status.System.NumCPU = runtime.NumCPU()
		status.System.NumGo = runtime.NumGoroutine()

		// Docker availability check
		if _, err := os.Stat("/var/run/docker.sock"); err == nil {
			status.Docker.Available = true
		} else {
			status.Docker.Available = false
			status.Docker.Reason = "docker socket not found"
		}

		// Output format
		if healthJSON {
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			if err := encoder.Encode(status); err != nil {
				return fmt.Errorf("failed to encode health status: %w", err)
			}
		} else {
			fmt.Printf("Status: %s\n", status.Status)
			fmt.Printf("Version: %s\n", status.Version)
			fmt.Printf("Docker: %v\n", status.Docker.Available)
			fmt.Printf("Timestamp: %s\n", status.Timestamp.Format(time.RFC3339))
		}

		return nil
	},
}

func init() {
	healthCmd.Flags().BoolVar(&healthJSON, "json", false, "Output in JSON format")
	rootCmd.AddCommand(healthCmd)
}

func getVersion() string {
	// This would be set via ldflags during build
	// For now, return a default
	if version := os.Getenv("PGSQUASH_VERSION"); version != "" {
		return version
	}
	return "0.8.1-beta"
}
