package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "download-bucket",
	Short: "Download files and folders from cloud storage buckets",
	Long: `A fast and flexible tool to download files and folders from various cloud storage providers.
	
Supports:
- AWS S3
- DigitalOcean Spaces
- Any S3-compatible service

Examples:
  download-bucket clone s3://my-bucket/folder/ ./local-folder
  download-bucket clone --provider=digitalocean spaces://my-space/data/ ./data
  download-bucket config set aws --access-key=XXX --secret-key=YYY --region=us-west-2`,
}

// Execute executes the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().Bool("verbose", false, "Enable verbose output")
	rootCmd.PersistentFlags().String("config", "", "Config file path (default is $HOME/.download-bucket/config.yaml)")
} 