package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var follow bool
var tail int

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Stream logs from the application",
	Long:  `Shows logs from all pods matching the application label. Supports following and tailing.`,
	Example: `  platformctl logs
  platformctl logs --follow
  platformctl logs --tail 50`,
	RunE: runLogs,
}

func init() {
	rootCmd.AddCommand(logsCmd)
	logsCmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow log output")
	logsCmd.Flags().IntVarP(&tail, "tail", "t", 50, "Number of lines to show from the end")
}

func runLogs(cmd *cobra.Command, args []string) error {
	color.Cyan("Fetching logs for %s/%s...", namespace, appName)

	kubectlArgs := []string{
		"logs", "-n", namespace,
		"-l", "app.kubernetes.io/name=" + appName,
		"--tail=" + fmt.Sprintf("%d", tail),
	}

	if follow {
		kubectlArgs = append(kubectlArgs, "--follow")
		color.Yellow("Following logs (Ctrl+C to stop)...")
	}

	c := exec.Command("kubectl", kubectlArgs...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}
