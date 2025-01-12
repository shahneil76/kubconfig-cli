package cmd

import (
	"bytes"
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"
)

var VerifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify cluster access and permissions",
	Run: func(cmd *cobra.Command, args []string) {
		if err := verifyClusterAccess(); err != nil {
			fmt.Printf("❌ %v\n", err)
			return
		}
		fmt.Println("✅ Cluster access verified")
	},
}

func verifyClusterAccess() error {
	checks := []struct {
		name    string
		command []string
	}{
		{
			name:    "Cluster connectivity",
			command: []string{"kubectl", "cluster-info"},
		},
		{
			name:    "ServiceAccount creation permission",
			command: []string{"kubectl", "auth", "can-i", "create", "serviceaccount"},
		},
		{
			name:    "ClusterRoleBinding creation permission",
			command: []string{"kubectl", "auth", "can-i", "create", "clusterrolebinding"},
		},
		{
			name:    "Secret access permission",
			command: []string{"kubectl", "auth", "can-i", "get", "secret", "-n", "kube-system"},
		},
	}

	for _, check := range checks {
		fmt.Printf("Checking %s... ", check.name)
		cmd := exec.Command(check.command[0], check.command[1:]...)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			fmt.Println("❌")
			return fmt.Errorf("%s failed: %v\n%s", check.name, err, stderr.String())
		}
		fmt.Println("✅")
	}

	return nil
}
