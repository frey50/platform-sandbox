package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var clusterName string

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Bootstrap the platform (create cluster, install infra, deploy app)",
	Long: `Runs the full platform bootstrap:
  1. Creates a kind cluster (if not exists)
  2. Installs Argo CD
  3. Installs kube-downscaler
  4. Deploys the sample application
  5. Verifies everything is healthy`,
	Example: `  platformctl up
  platformctl up --cluster-name my-cluster`,
	RunE: runUp,
}

func init() {
	rootCmd.AddCommand(upCmd)
	upCmd.Flags().StringVar(&clusterName, "cluster-name", "platform-sandbox", "Name of the kind cluster")
}

func runUp(cmd *cobra.Command, args []string) error {
	color.Cyan("========================================")
	color.Cyan("     PLATFORM SANDBOX BOOTSTRAP")
	color.Cyan("========================================")
	fmt.Println()

	// Step 1: Create cluster
	color.Yellow("[1/5] Creating kind cluster '%s'...", clusterName)
	scriptPath := filepath.Join(".", "cluster-up.sh")
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		scriptPath = filepath.Join("..", "..", "cluster-up.sh")
	}

	cmdEnv := os.Environ()
	cmdEnv = append(cmdEnv, "CLUSTER_NAME="+clusterName)
	c := exec.Command("bash", scriptPath)
	c.Env = cmdEnv
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		color.Yellow("Cluster creation may have skipped (already exists) or failed: %v", err)
	}
	fmt.Println()

	// Step 2: Install Argo CD
	color.Yellow("[2/5] Installing Argo CD...")
	exec.Command("kubectl", "create", "namespace", "argocd").Run()
	c = exec.Command("kubectl", "apply", "-n", "argocd", "-f",
		"https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml")
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		color.Red("Failed to install Argo CD: %v", err)
		return err
	}
	fmt.Println()

	// Step 3: Install downscaler
	color.Yellow("[3/5] Installing kube-downscaler...")
	c = exec.Command("kubectl", "apply", "-f", "infrastructure/downscaler/")
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		color.Yellow("Downscaler install failed (may already exist): %v", err)
	}
	fmt.Println()

	// Step 4: Deploy app
	color.Yellow("[4/5] Deploying sample application...")
	c = exec.Command("kubectl", "apply", "-f", "apps/sample-app-application.yaml", "-n", "argocd")
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		color.Red("Failed to deploy app: %v", err)
		return err
	}
	fmt.Println()

	// Step 5: Verify
	color.Yellow("[5/5] Verifying deployment...")
	exec.Command("kubectl", "wait", "--for=condition=available", "deployment", appName,
		"-n", namespace, "--timeout=120s").Run()
	fmt.Println()

	color.Green("========================================")
	color.Green("     BOOTSTRAP COMPLETE ✓")
	color.Green("========================================")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  platformctl status    # Check health")
	fmt.Println("  platformctl cost      # View cost report")
	fmt.Println("  kubectl port-forward svc/argocd-server 8080:443 -n argocd")

	return nil
}
