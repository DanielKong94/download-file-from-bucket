package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"download-file-from-bucket/config"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration for cloud providers",
	Long:  "Manage configuration for cloud storage providers",
}

var configSetCmd = &cobra.Command{
	Use:   "set <provider-name>",
	Short: "Set configuration for a cloud provider",
	Long: `Set configuration for a cloud provider.

Examples:
  download-bucket config set aws --access-key=XXX --secret-key=YYY --region=us-west-2 --bucket=my-bucket
  download-bucket config set digitalocean --access-key=XXX --secret-key=YYY --region=nyc3 --bucket=my-space`,
	Args: cobra.ExactArgs(1),
	RunE: runConfigSet,
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured providers",
	Long:  "List all configured cloud storage providers",
	RunE:  runConfigList,
}

var (
	configAccessKey string
	configSecretKey string
	configRegion    string
	configEndpoint  string
	configBucket    string
	configType      string
)

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configListCmd)

	configSetCmd.Flags().StringVar(&configAccessKey, "access-key", "", "Access key")
	configSetCmd.Flags().StringVar(&configSecretKey, "secret-key", "", "Secret key")
	configSetCmd.Flags().StringVar(&configRegion, "region", "", "Region")
	configSetCmd.Flags().StringVar(&configEndpoint, "endpoint", "", "Custom endpoint")
	configSetCmd.Flags().StringVar(&configBucket, "bucket", "", "Default bucket name")
	configSetCmd.Flags().StringVar(&configType, "type", "", "Provider type (s3, digitalocean)")

	configSetCmd.MarkFlagRequired("access-key")
	configSetCmd.MarkFlagRequired("secret-key")
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	providerName := args[0]

	// Load existing config
	cfg, err := config.LoadConfig()
	if err != nil {
		// If config doesn't exist, create a new one
		cfg = &config.Config{
			Providers: make(map[string]config.ProviderConfig),
		}
	}

	// Determine provider type
	providerType := configType
	if providerType == "" {
		switch providerName {
		case "aws", "s3":
			providerType = "s3"
		case "digitalocean", "do", "spaces":
			providerType = "digitalocean"
		default:
			return fmt.Errorf("unknown provider type for %s, please specify with --type", providerName)
		}
	}

	// Set default region if not provided
	if configRegion == "" {
		switch providerType {
		case "s3":
			configRegion = "us-east-1"
		case "digitalocean":
			configRegion = "nyc3"
		}
	}

	// Set default endpoint for DigitalOcean
	if configEndpoint == "" && providerType == "digitalocean" {
		configEndpoint = fmt.Sprintf("https://%s.digitaloceanspaces.com", configRegion)
	}

	// Create provider config
	providerConfig := config.ProviderConfig{
		Type:      providerType,
		Region:    configRegion,
		Endpoint:  configEndpoint,
		AccessKey: configAccessKey,
		SecretKey: configSecretKey,
		Bucket:    configBucket,
		Options:   make(map[string]string),
	}

	// Add to config
	cfg.Providers[providerName] = providerConfig

	// Save config
	configPath := getConfigPath()
	if err := cfg.SaveConfig(configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Configuration saved for provider '%s'\n", providerName)
	fmt.Printf("Config file: %s\n", configPath)

	return nil
}

func runConfigList(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if len(cfg.Providers) == 0 {
		fmt.Println("No providers configured.")
		fmt.Println("Use 'download-bucket config set <provider>' to add a provider.")
		return nil
	}

	fmt.Println("Configured providers:")
	for name, provider := range cfg.Providers {
		fmt.Printf("\n%s:\n", name)
		fmt.Printf("  Type: %s\n", provider.Type)
		fmt.Printf("  Region: %s\n", provider.Region)
		if provider.Endpoint != "" {
			fmt.Printf("  Endpoint: %s\n", provider.Endpoint)
		}
		if provider.Bucket != "" {
			fmt.Printf("  Default Bucket: %s\n", provider.Bucket)
		}
		fmt.Printf("  Access Key: %s***\n", maskKey(provider.AccessKey))
	}

	return nil
}

func getConfigPath() string {
	configPath, _ := rootCmd.Flags().GetString("config")
	if configPath != "" {
		return configPath
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "config.yaml"
	}

	return filepath.Join(homeDir, ".download-bucket", "config.yaml")
}

func maskKey(key string) string {
	if len(key) <= 8 {
		return "***"
	}
	return key[:4] + "***"
} 