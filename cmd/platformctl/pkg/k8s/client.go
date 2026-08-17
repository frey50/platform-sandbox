package k8s

import (
	"fmt"
	"os/exec"
	"strings"
)

// Client wraps kubectl operations for reuse across commands.
type Client struct {
	Namespace string
}

// NewClient creates a new K8s client for the given namespace.
func NewClient(namespace string) *Client {
	return &Client{Namespace: namespace}
}

// GetDeploymentReplicas returns the current replica count of a deployment.
func (c *Client) GetDeploymentReplicas(name string) (int, error) {
	out, err := exec.Command("kubectl", "get", "deployment", name,
		"-n", c.Namespace,
		"-o", "jsonpath={.spec.replicas}").CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("failed to get replicas: %w (output: %s)", err, string(out))
	}

	var replicas int
	fmt.Sscanf(string(out), "%d", &replicas)
	return replicas, nil
}

// ScaleDeployment scales a deployment to the given replica count.
func (c *Client) ScaleDeployment(name string, replicas int) error {
	out, err := exec.Command("kubectl", "scale", "deployment", name,
		fmt.Sprintf("--replicas=%d", replicas),
		"-n", c.Namespace).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to scale: %w (output: %s)", err, string(out))
	}
	return nil
}

// GetPodStatus returns a summary of pod statuses in the namespace.
func (c *Client) GetPodStatus() (running, pending, failed int, err error) {
	out, err := exec.Command("kubectl", "get", "pods", "-n", c.Namespace,
		"-o", "jsonpath={range .items[*]}{.status.phase}{\"\\n\"}{end}").CombinedOutput()
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to get pods: %w", err)
	}

	for _, phase := range strings.Split(string(out), "\n") {
		switch phase {
		case "Running":
			running++
		case "Pending":
			pending++
		case "Failed", "Error", "CrashLoopBackOff":
			failed++
		}
	}
	return running, pending, failed, nil
}
