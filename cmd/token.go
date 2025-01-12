package cmd

import (
	"encoding/json"
	"os"
	"time"

	"github.com/spf13/cobra"
)

type ExecCredential struct {
	ApiVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Status     struct {
		Token               string    `json:"token"`
		ExpirationTimestamp time.Time `json:"expirationTimestamp"`
	} `json:"status"`
}

var TokenCmd = &cobra.Command{
	Use:    "token",
	Short:  "Internal command for token management",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		origToken, _ := cmd.Flags().GetString("original-token")
		expiryStr, _ := cmd.Flags().GetString("expiry")

		expiry, err := time.Parse(time.RFC3339, expiryStr)
		if err != nil {
			os.Exit(1)
		}

		// Check if token has expired
		if time.Now().After(expiry) {
			os.Exit(1)
		}

		// Return valid credentials
		creds := ExecCredential{
			ApiVersion: "client.authentication.k8s.io/v1beta1",
			Kind:       "ExecCredential",
			Status: struct {
				Token               string    `json:"token"`
				ExpirationTimestamp time.Time `json:"expirationTimestamp"`
			}{
				Token:               origToken,
				ExpirationTimestamp: expiry,
			},
		}

		json.NewEncoder(os.Stdout).Encode(creds)
	},
}

func init() {
	TokenCmd.Flags().String("original-token", "", "Original token")
	TokenCmd.Flags().String("original-cert", "", "Original certificate")
	TokenCmd.Flags().String("original-key", "", "Original key")
	TokenCmd.Flags().String("expiry", "", "Token expiry time")
}
