package main

import (
	"kubconfig-cli/cmd"

	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "kubconfig",
		Short: "A CLI tool for managing kubeconfigs from S3",
	}

	rootCmd.AddCommand(cmd.InitCmd)
	rootCmd.AddCommand(cmd.ListCmd)
	rootCmd.AddCommand(cmd.ActivateCmd)
	rootCmd.AddCommand(cmd.ClearCmd)
	rootCmd.AddCommand(cmd.CurrentCmd)
	rootCmd.AddCommand(cmd.CleanupCmd)
	rootCmd.AddCommand(cmd.AnalyzeCmd)
	rootCmd.AddCommand(cmd.StatusCmd)
	rootCmd.AddCommand(cmd.DeactivateCmd)
	rootCmd.AddCommand(cmd.VerifyCmd)

	// Add shell completion
	rootCmd.CompletionOptions.DisableDefaultCmd = false

	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}
