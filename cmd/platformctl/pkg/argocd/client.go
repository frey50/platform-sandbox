package argocd

import (
	"fmt"
	"os/exec"
	"strings"
)

// Client wraps Argo CD CLI operations.
type Client struct {
	Namespace string
}

// NewClient creates a new Argo CD client.
func NewClient(namespace string) *Client {
	return &Client{Namespace: namespace}
}

// GetAppStatus returns the sync and health status of an application.
func (c *Client) GetAppStatus(name string) (syncStatus, healthStatus string, err error) {
	out, err := exec.Command("kubectl", "get", "application", name,
		"-n", c.Namespace,
		"-o", "jsonpath={.status.sync.status} {.status.health.status}").CombinedOutput()
	if err != nil {
		return "", "", fmt.Errorf("failed to get app status: %w (output: %s)", err, string(out))
	}

	parts := strings.Fields(string(out))
	if len(parts) >= 2 {
		return parts[0], parts[1], nil
	}
	return string(out), "", nil
}

// SyncApp triggers a manual sync of an Argo CD application.
func (c *Client) SyncApp(name string) error {
	out, err := exec.Command("argocd", "app", "sync", name).CombinedOutput()
	if err != nil {
		// If argocd CLI is not installed, fall back to kubectl
		out, err = exec.Command("kubectl", "patch", "application", name,
			"-n", c.Namespace,
			"--type", "merge",
			"-p", `{"operation":{"sync":{"revision":"HEAD"}}}`).CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to sync app: %w (output: %s)", err, string(out))
		}
	}
	_ = out
	return nil
}
