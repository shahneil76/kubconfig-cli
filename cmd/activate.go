package cmd

import (
	"fmt"
	"io"
	"kubconfig-cli/config"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/spf13/cobra"
)

func init() {
	ActivateCmd.Flags().DurationP("session", "s", 8*time.Hour, "Session duration (e.g., 2h, 30m, 1h30m)")
	ActivateCmd.MarkFlagRequired("session")
	config.StartCleanupRoutine()
}

var ActivateCmd = &cobra.Command{
	Use:   "activate [KUBECONFIG_NAME]",
	Short: "Activate a kubeconfig from the S3 bucket",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		kubeconfigName := args[0]
		sessionDuration, err := cmd.Flags().GetDuration("session")
		if err != nil || sessionDuration <= 0 {
			fmt.Println("Error: Valid session duration is required (e.g., --session 2h)")
			return
		}

		// Validate session duration
		if sessionDuration > 24*time.Hour {
			fmt.Println("Error: Maximum session duration is 24 hours")
			return
		}
		if sessionDuration < 10*time.Minute {
			fmt.Printf("Warning: Minimum session duration is 10 minutes, adjusting...\n")
			sessionDuration = 10 * time.Minute
		}

		// Load config
		cfg, err := config.LoadConfig()
		if err != nil {
			fmt.Printf("Error loading configuration: %v\n", err)
			return
		}

		// Validate kubeconfig name
		if err := config.ValidateKubeconfigName(kubeconfigName); err != nil {
			fmt.Printf("Invalid kubeconfig name: %v\n", err)
			return
		}

		// Download from S3 and set as current context
		originalConfig, err := downloadFromS3(cfg, kubeconfigName)
		if err != nil {
			fmt.Printf("Error downloading kubeconfig: %v\n", err)
			return
		}

		// Create temporary kubeconfig with original config
		tempKubeconfig := filepath.Join(config.CacheDir, fmt.Sprintf("%s.tmp", kubeconfigName))
		if err := os.WriteFile(tempKubeconfig, originalConfig, 0600); err != nil {
			fmt.Printf("Error creating temporary kubeconfig: %v\n", err)
			return
		}
		defer os.Remove(tempKubeconfig)

		// Set KUBECONFIG to use original config for creating resources
		originalKubeconfig := os.Getenv("KUBECONFIG")
		os.Setenv("KUBECONFIG", tempKubeconfig)
		defer os.Setenv("KUBECONFIG", originalKubeconfig)

		// Create temporary access
		saConfig, err := config.CreateTemporaryAccess(sessionDuration)
		if err != nil {
			fmt.Printf("Error creating temporary access: %v\n", err)
			return
		}

		// Get token with TTL
		token, err := config.GetTemporaryToken(saConfig)
		if err != nil {
			fmt.Printf("Error getting token: %v\n", err)
			return
		}

		// Modify the original kubeconfig with the temporary token
		sessionKubeconfig, err := config.ModifyKubeconfigWithToken(originalConfig, token)
		if err != nil {
			fmt.Printf("Error modifying kubeconfig: %v\n", err)
			return
		}

		// Save the modified config
		if err := os.WriteFile(config.KubeConfigFile, sessionKubeconfig, 0600); err != nil {
			fmt.Printf("Error saving kubeconfig: %v\n", err)
			return
		}

		fmt.Printf("Successfully activated '%s' (session expires at %s)\n",
			kubeconfigName,
			saConfig.ExpiresAt.Format(time.RFC3339))
	},
}

func downloadFromS3(cfg config.Config, kubeconfigName string) ([]byte, error) {
	sess, err := config.CreateS3Session(cfg)
	if err != nil {
		return nil, fmt.Errorf("error creating AWS session: %v", err)
	}

	svc := s3.New(sess)

	// Download the file
	output, err := svc.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(cfg.S3Bucket),
		Key:    aws.String(kubeconfigName),
	})
	if err != nil {
		return nil, fmt.Errorf("error fetching from S3: %v", err)
	}
	defer output.Body.Close()

	// Create cache directory if it doesn't exist
	if err := os.MkdirAll(config.CacheDir, 0700); err != nil {
		return nil, fmt.Errorf("error creating cache directory: %v", err)
	}

	// Create cache file
	file, err := os.Create(filepath.Join(config.CacheDir, kubeconfigName))
	if err != nil {
		return nil, fmt.Errorf("error creating cache file: %v", err)
	}
	defer file.Close()

	// Copy the data
	if _, err := io.Copy(file, output.Body); err != nil {
		return nil, fmt.Errorf("error saving to cache: %v", err)
	}

	return os.ReadFile(filepath.Join(config.CacheDir, kubeconfigName))
}

func copyFile(src, dst string) error {
	// Ensure the destination directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("error creating destination directory: %v", err)
	}

	// Read source file
	input, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("error reading source file: %v", err)
	}

	// Write to destination (will overwrite if exists)
	if err := os.WriteFile(dst, input, 0644); err != nil {
		return fmt.Errorf("error writing destination file: %v", err)
	}

	return nil
}
