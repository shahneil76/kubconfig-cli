package cmd

import (
	"fmt"
	"kubconfig-cli/config"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

var (
	olderThan int
)

var CleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Clean up cached kubeconfig files",
	Run: func(cmd *cobra.Command, args []string) {
		files, err := os.ReadDir(config.CacheDir)
		if err != nil {
			fmt.Printf("Error reading cache directory: %v\n", err)
			return
		}

		cutoff := time.Now().Add(-time.Duration(olderThan) * 24 * time.Hour)
		cleaned := 0

		for _, file := range files {
			if file.IsDir() {
				continue
			}

			path := filepath.Join(config.CacheDir, file.Name())
			info, err := file.Info()
			if err != nil {
				continue
			}

			if info.ModTime().Before(cutoff) {
				if err := os.Remove(path); err != nil {
					fmt.Printf("Error removing %s: %v\n", file.Name(), err)
					continue
				}
				cleaned++
			}
		}

		fmt.Printf("Cleaned up %d cached kubeconfig files\n", cleaned)
	},
}

func init() {
	CleanupCmd.Flags().IntVarP(&olderThan, "older-than", "o", 30, "Clean up files older than N days")
}

func cleanupSessions() error {
	files, err := os.ReadDir(config.SessionDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		path := filepath.Join(config.SessionDir, file.Name())
		if err := os.Remove(path); err != nil {
			fmt.Printf("Error removing %s: %v\n", file.Name(), err)
			continue
		}
	}

	return nil
}
