package cmd

import (
	"fmt"
	"kubconfig-cli/config"
	"os"

	"github.com/spf13/cobra"
)

var DeactivateCmd = &cobra.Command{
	Use:   "deactivate",
	Short: "Deactivate the current session and revert to default kubeconfig",
	Run: func(cmd *cobra.Command, args []string) {
		// Get current kubeconfig path
		currentConfig := os.Getenv("KUBECONFIG")
		if currentConfig == "" {
			currentConfig = config.KubeConfigFile
		}

		// Get service account info before clearing config
		saConfig, err := config.GetServiceAccountFromConfig(currentConfig)
		if err != nil {
			fmt.Printf("Warning: Could not get service account info: %v\n", err)
		}

		// Clear the kubeconfig
		if err := os.WriteFile(config.KubeConfigFile, []byte(""), 0600); err != nil {
			fmt.Printf("Error clearing kubeconfig: %v\n", err)
			return
		}

		// Clean up service account and related resources
		if saConfig != nil {
			if err := config.CleanupTemporaryAccess(saConfig); err != nil {
				fmt.Printf("Warning: Error cleaning up resources: %v\n", err)
			} else {
				fmt.Printf("Cleaned up service account: %s\n", saConfig.Name)
			}
		}

		// Clean up session files
		if err := cleanupSessions(); err != nil {
			fmt.Printf("Warning: Error cleaning up sessions: %v\n", err)
		}

		if shellEval, _ := cmd.Flags().GetBool("shell-eval"); shellEval {
			fmt.Println("unset KUBECONFIG")
			return
		}

		fmt.Println("Successfully deactivated session")
	},
}
