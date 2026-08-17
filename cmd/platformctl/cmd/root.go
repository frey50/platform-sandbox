package cmd

import (
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	namespace string
	appName   string
)

var rootCmd = &cobra.Command{
	Use:   "platformctl",
	Short: "Platform Sandbox CLI - manage your GitOps K8s platform",
	Long: `platformctl is the operator CLI for the Platform Sandbox.

It provides a unified interface for:
  - Checking cluster and application health
  - Scaling applications manually
  - Triggering downscales and upscales
  - Viewing cost savings reports
  - Verifying prerequisites and setup

Examples:
  platformctl status              # Show overall platform status
  platformctl verify              # Check prerequisites
  platformctl scale --replicas 5  # Scale sample-app to 5 replicas
  platformctl cost                # Show cost savings report
  platformctl logs                # Stream application logs`,
	SilenceUsage: true,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&namespace, "namespace", "n", "sample-app", "Target namespace")
	rootCmd.PersistentFlags().StringVarP(&appName, "app", "a", "sample-app", "Target application name")

	if os.Getenv("NO_COLOR") != "" {
		color.NoColor = true
	}
}
