package cmd

import (
	"fmt"
	"os/exec"
	"strconv"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var replicas int

var scaleCmd = &cobra.Command{
	Use:   "scale",
	Short: "Scale an application to a specific replica count",
	Long:  `Manually overrides the replica count of a Deployment. Argo CD will not fight this due to ignoreDifferences.`,
	Example: `  platformctl scale --replicas 5
  platformctl scale -a my-app -n my-ns --replicas 10`,
	RunE: runScale,
}

func init() {
	rootCmd.AddCommand(scaleCmd)
	scaleCmd.Flags().IntVarP(&replicas, "replicas", "r", 0, "Target replica count (required)")
	scaleCmd.MarkFlagRequired("replicas")
}

func runScale(cmd *cobra.Command, args []string) error {
	if replicas < 0 {
		return fmt.Errorf("replicas must be >= 0")
	}

	color.Cyan("Scaling %s/%s to %d replicas...", namespace, appName, replicas)

	out, err := exec.Command("kubectl", "scale", "deployment", appName,
		"--replicas="+strconv.Itoa(replicas),
		"-n", namespace,
		"--timeout=60s").CombinedOutput()
	if err != nil {
		color.Red("Failed to scale: %v", err)
		fmt.Println(string(out))
		return err
	}

	color.Green("✓ %s", string(out))

	// Wait for rollout
	color.Yellow("Waiting for rollout to complete...")
	out, err = exec.Command("kubectl", "rollout", "status", "deployment", appName,
		"-n", namespace, "--timeout=120s").CombinedOutput()
	if err != nil {
		color.Yellow("Rollout status: %v", err)
		fmt.Println(string(out))
	} else {
		color.Green("✓ Rollout complete")
	}

	return nil
}
