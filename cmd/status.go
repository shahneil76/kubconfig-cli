package cmd

import (
	"fmt"
	"kubconfig-cli/config"
	"time"

	"github.com/spf13/cobra"
)

var StatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check current kubeconfig token status",
	Run: func(cmd *cobra.Command, args []string) {
		expired, expiry, err := config.VerifyTokenExpiry(config.KubeConfigFile)
		if err != nil {
			fmt.Printf("Error checking token status: %v\n", err)
			return
		}

		if expired {
			fmt.Printf("Token has expired (expired at %s)\n", expiry.Format(time.RFC3339))
		} else {
			remaining := time.Until(expiry).Round(time.Second)
			fmt.Printf("Token is valid (expires in %s)\n", remaining)
		}
	},
}
