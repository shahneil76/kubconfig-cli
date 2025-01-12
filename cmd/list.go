package cmd

import (
	"fmt"
	"kubconfig-cli/config"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/spf13/cobra"
)

var ListCmd = &cobra.Command{
	Use:   "list",
	Short: "List kubeconfig files in the S3 bucket",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadConfig()
		if err != nil {
			fmt.Println("Error loading configuration:", err)
			return
		}

		sess, err := config.CreateS3Session(cfg)
		if err != nil {
			fmt.Printf("Error creating AWS session: %v\n", err)
			return
		}
		svc := s3.New(sess)

		result, err := svc.ListObjectsV2(&s3.ListObjectsV2Input{
			Bucket: aws.String(cfg.S3Bucket),
		})
		if err != nil {
			fmt.Println("Error listing files:", err)
			return
		}

		fmt.Println("Available kubeconfigs:")
		for _, item := range result.Contents {
			fmt.Println("-", *item.Key)
		}
	},
}
