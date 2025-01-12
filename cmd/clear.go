package cmd

import (
	"fmt"
	"kubconfig-cli/config"
	"os"

	"github.com/spf13/cobra"
)

var ClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear the active kubeconfig",
	Run: func(cmd *cobra.Command, args []string) {
		if err := os.WriteFile(config.KubeConfigFile, []byte(""), 0644); err != nil {
			fmt.Printf("Error clearing kubeconfig: %v\n", err)
			return
		}
		fmt.Println("Kubeconfig cleared successfully.")
	},
}
