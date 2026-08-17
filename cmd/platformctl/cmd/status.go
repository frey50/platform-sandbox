package cmd

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show platform and application status",
	Long:  `Displays the health of the cluster, Argo CD sync status, and application pods.`,
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	color.Cyan("========================================")
	color.Cyan("     PLATFORM SANDBOX STATUS")
	color.Cyan("========================================")
	fmt.Println()

	// Cluster nodes
	color.Yellow("Cluster Nodes:")
	out, err := exec.Command("kubectl", "get", "nodes", "-o", "wide").CombinedOutput()
	if err != nil {
		color.Red("  ✗ Failed to get nodes: %v", err)
	} else {
		fmt.Println(string(out))
	}

	// Argo CD Application status
	color.Yellow("Argo CD Application:")
	out, err = exec.Command("kubectl", "get", "application", appName, "-n", "argocd", "-o", "custom-columns=NAME:.metadata.name,SYNC:.status.sync.status,HEALTH:.status.health.status").CombinedOutput()
	if err != nil {
		color.Red("  ✗ Argo CD app not found or Argo CD not installed")
		fmt.Println("    → Run: make deploy-argocd && make deploy")
	} else {
		lines := strings.Split(string(out), "\n")
		for i, line := range lines {
			if i == 0 {
				fmt.Println("  " + line)
				continue
			}
			if strings.Contains(line, "Synced") && strings.Contains(line, "Healthy") {
				color.Green("  " + line)
			} else if line != "" {
				color.Yellow("  " + line)
			}
		}
	}
	fmt.Println()

	// Application pods
	color.Yellow("Application Pods (%s):", namespace)
	out, err = exec.Command("kubectl", "get", "pods", "-n", namespace, "-o", "wide").CombinedOutput()
	if err != nil {
		color.Red("  ✗ No pods found in namespace %s", namespace)
	} else {
		fmt.Println(string(out))
	}

	// Deployment replicas
	color.Yellow("Deployment Scale:")
	out, err = exec.Command("kubectl", "get", "deployment", appName, "-n", namespace, "-o", "custom-columns=NAME:.metadata.name,DESIRED:.spec.replicas,CURRENT:.status.replicas,AVAILABLE:.status.availableReplicas").CombinedOutput()
	if err != nil {
		color.Red("  ✗ Deployment not found")
	} else {
		fmt.Println(string(out))
	}

	// HPA status
	color.Yellow("HPA Status:")
	out, err = exec.Command("kubectl", "get", "hpa", appName, "-n", namespace, "-o", "custom-columns=NAME:.metadata.name,MIN:.spec.minReplicas,MAX:.spec.maxReplicas,CURRENT:.status.currentReplicas,TARGET:.status.targetResource.utilizationPercentage").CombinedOutput()
	if err != nil {
		color.Yellow("  (HPA not found or metrics-server not ready)")
	} else {
		fmt.Println(string(out))
	}

	// Downscaler status
	color.Yellow("Downscaler:")
	out, err = exec.Command("kubectl", "get", "pods", "-n", "kube-system", "-l", "app=kube-downscaler", "-o", "custom-columns=NAME:.metadata.name,STATUS:.status.phase").CombinedOutput()
	if err != nil {
		out, err = exec.Command("kubectl", "get", "pods", "-n", "kube-downscaler", "-l", "app.kubernetes.io/name=kube-downscaler", "-o", "custom-columns=NAME:.metadata.name,STATUS:.status.phase").CombinedOutput()
	}
	if err != nil {
		color.Yellow("  (Downscaler not found — may be in kube-system or kube-downscaler namespace)")
	} else {
		fmt.Println(string(out))
	}

	fmt.Println()
	color.Cyan("========================================")
	return nil
}
