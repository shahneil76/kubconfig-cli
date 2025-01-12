package cmd

import (
	"fmt"
	"kubconfig-cli/config"
	"os"

	"github.com/spf13/cobra"
)

var InitCmd = &cobra.Command{
	Use:   "init",
	Short: "Configure the CLI with S3 bucket and AWS credentials",
	Run: func(cmd *cobra.Command, args []string) {
		var cfg config.Config

		// Interactive configuration
		fmt.Print("Enter S3 bucket name: ")
		fmt.Scanln(&cfg.S3Bucket)

		fmt.Print("Enter AWS region: ")
		fmt.Scanln(&cfg.Region)

		fmt.Print("Enter AWS access key: ")
		fmt.Scanln(&cfg.AWSAccessKey)

		fmt.Print("Enter AWS secret key: ")
		fmt.Scanln(&cfg.AWSSecretKey)

		fmt.Print("Enter S3 endpoint URL (optional, press Enter for AWS S3): ")
		var endpoint string
		fmt.Scanln(&endpoint)

		if endpoint != "" {
			cfg.S3Endpoint = config.FormatEndpointURL(endpoint)
			cfg.ForcePathStyle = true
			fmt.Printf("Using S3 endpoint: %s\n", cfg.S3Endpoint)
			fmt.Println("Enabled path-style addressing for S3 compatible service")
		}

		// Create necessary directories
		if err := os.MkdirAll(config.CacheDir, 0755); err != nil {
			fmt.Printf("Error creating cache directory: %v\n", err)
			return
		}

		if err := config.SaveConfig(cfg); err != nil {
			fmt.Printf("Error saving configuration: %v\n", err)
			return
		}

		fmt.Println("Configuration saved successfully")
	},
}
