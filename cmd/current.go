package cmd

import (
	"fmt"
	"kubconfig-cli/config"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var CurrentCmd = &cobra.Command{
	Use:   "current",
	Short: "Show the currently active kubeconfig",
	Run: func(cmd *cobra.Command, args []string) {
		// Check if ~/.kube/config exists
		if _, err := os.Stat(config.KubeConfigFile); err != nil {
			if os.IsNotExist(err) {
				fmt.Println("No kubeconfig is currently active")
				return
			}
			fmt.Printf("Error checking kubeconfig: %v\n", err)
			return
		}

		// Execute kubectl config current-context
		kubectlCmd := exec.Command("kubectl", "config", "current-context")
		output, err := kubectlCmd.Output()
		if err != nil {
			fmt.Printf("Error getting current context: %v\n", err)
			return
		}

		fmt.Printf("Current context: %s", string(output))
	},
}
