package cmd

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify prerequisites and platform health",
	Long:  `Checks that all required tools are installed and the cluster is accessible.`,
	RunE:  runVerify,
}

func init() {
	rootCmd.AddCommand(verifyCmd)
}

type check struct {
	Name string
	Cmd  string
	Args []string
	Hint string
}

func runVerify(cmd *cobra.Command, args []string) error {
	color.Cyan("========================================")
	color.Cyan("     PLATFORM SANDBOX VERIFICATION")
	color.Cyan("========================================")
	fmt.Println()

	checks := []check{
		{Name: "Docker", Cmd: "docker", Args: []string{"--version"}, Hint: "Install: https://docs.docker.com/get-docker/"},
		{Name: "kind", Cmd: "kind", Args: []string{"--version"}, Hint: "Install: https://kind.sigs.k8s.io/docs/user/quick-start/"},
		{Name: "kubectl", Cmd: "kubectl", Args: []string{"version", "--client"}, Hint: "Install: https://kubernetes.io/docs/tasks/tools/"},
		{Name: "Helm", Cmd: "helm", Args: []string{"version"}, Hint: "Install: https://helm.sh/docs/intro/install/"},
	}

	if runtime.GOOS != "windows" {
		checks = append(checks, check{Name: "bash", Cmd: "bash", Args: []string{"--version"}, Hint: "Required for cluster-up.sh"})
	}

	allPassed := true
	for _, c := range checks {
		if err := exec.Command(c.Cmd, c.Args...).Run(); err != nil {
			color.Red("  ✗ %s: NOT FOUND", c.Name)
			fmt.Printf("    → %s\n", c.Hint)
			allPassed = false
		} else {
			out, _ := exec.Command(c.Cmd, c.Args...).Output()
			color.Green("  ✓ %s: %s", c.Name, trimVersion(string(out)))
		}
	}

	fmt.Println()
	color.Cyan("Cluster Check:")
	if err := exec.Command("kubectl", "cluster-info").Run(); err != nil {
		color.Red("  ✗ Kubernetes cluster: NOT ACCESSIBLE")
		fmt.Println("    → Run: ./cluster-up.sh")
		allPassed = false
	} else {
		color.Green("  ✓ Kubernetes cluster: ACCESSIBLE")
		out, _ := exec.Command("kubectl", "config", "current-context").Output()
		fmt.Printf("    → Context: %s", out)
	}

	fmt.Println()
	if allPassed {
		color.Green("========================================")
		color.Green("     ALL CHECKS PASSED ✓")
		color.Green("========================================")
	} else {
		color.Yellow("========================================")
		color.Yellow("     SOME CHECKS FAILED")
		color.Yellow("========================================")
	}

	return nil
}

func trimVersion(s string) string {
	if len(s) > 50 {
		return s[:50] + "..."
	}
	return s
}
