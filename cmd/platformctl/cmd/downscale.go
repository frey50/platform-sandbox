package cmd

import (
	"fmt"
	"os/exec"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var downscaleCmd = &cobra.Command{
	Use:   "downscale",
	Short: "Immediately downscale an application to 0 replicas",
	Long:  `Triggers an immediate downscale by setting replicas to 0. Useful for emergency cost cutting or testing.`,
	Example: `  platformctl downscale
  platformctl downscale -a my-app -n my-ns`,
	RunE: runDownscale,
}

func init() {
	rootCmd.AddCommand(downscaleCmd)
}

func runDownscale(cmd *cobra.Command, args []string) error {
	color.Cyan("Downscaling %s/%s to 0 replicas...", namespace, appName)

	out, err := exec.Command("kubectl", "scale", "deployment", appName,
		"--replicas=0",
		"-n", namespace).CombinedOutput()
	if err != nil {
		color.Red("Failed to downscale: %v", err)
		fmt.Println(string(out))
		return err
	}

	color.Green("✓ Downscaled successfully")
	fmt.Println(string(out))

	// Show current state
	out, _ = exec.Command("kubectl", "get", "deployment", appName, "-n", namespace,
		"-o", "custom-columns=NAME:.metadata.name,REPLICAS:.spec.replicas").CombinedOutput()
	fmt.Println(string(out))

	return nil
}
