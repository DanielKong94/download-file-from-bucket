package cmd

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"download-file-from-bucket/config"
	"download-file-from-bucket/downloader"
	"download-file-from-bucket/providers"
)

var (
	providerName  string
	concurrency   int
	accessKey     string
	secretKey     string
	region        string
	endpoint      string
	bucket        string
)

var cloneCmd = &cobra.Command{
	Use:   "clone <source> <destination>",
	Short: "Clone a folder from cloud storage to local directory",
	Long: `Clone (download) an entire folder from cloud storage to a local directory.

Source formats:
  s3://bucket-name/path/to/folder/
  spaces://space-name/path/to/folder/
  https://region.digitaloceanspaces.com/space-name/path/to/folder/

Examples:
  download-bucket clone s3://my-bucket/data/ ./local-data
  download-bucket clone --provider=digitalocean spaces://my-space/images/ ./images
  download-bucket clone --provider=aws --region=eu-west-1 s3://eu-bucket/files/ ./files`,
	Args: cobra.ExactArgs(2),
	RunE: runClone,
}

func init() {
	rootCmd.AddCommand(cloneCmd)

	cloneCmd.Flags().StringVar(&providerName, "provider", "", "Cloud provider (aws, digitalocean)")
	cloneCmd.Flags().IntVar(&concurrency, "concurrency", 5, "Number of concurrent downloads")
	cloneCmd.Flags().StringVar(&accessKey, "access-key", "", "Access key (overrides config)")
	cloneCmd.Flags().StringVar(&secretKey, "secret-key", "", "Secret key (overrides config)")
	cloneCmd.Flags().StringVar(&region, "region", "", "Region (overrides config)")
	cloneCmd.Flags().StringVar(&endpoint, "endpoint", "", "Custom endpoint (overrides config)")
	cloneCmd.Flags().StringVar(&bucket, "bucket", "", "Bucket name (overrides URL)")
}

func runClone(cmd *cobra.Command, args []string) error {
	sourceURL := args[0]
	destDir := args[1]
	
	verbose, _ := cmd.Flags().GetBool("verbose")
	
	// Parse the source URL
	parsedSource, err := parseSourceURL(sourceURL)
	if err != nil {
		return fmt.Errorf("invalid source URL: %w", err)
	}

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Determine provider configuration
	providerConfig, err := getProviderConfig(cfg, parsedSource)
	if err != nil {
		return fmt.Errorf("failed to get provider config: %w", err)
	}

	// Create provider
	opts := providers.GetProviderOptions(
		providerConfig.Type,
		providerConfig.Region,
		providerConfig.Endpoint,
		providerConfig.AccessKey,
		providerConfig.SecretKey,
		providerConfig.Bucket,
		providerConfig.Options,
	)

	provider, err := providers.NewProvider(opts)
	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}
	defer provider.Close()

	// Create downloader
	dl := downloader.NewDownloader(provider, downloader.Options{
		Concurrency: concurrency,
		Verbose:     verbose,
	})

	// Progress callback
	var lastProgress time.Time
	progressCallback := func(progress providers.DownloadProgress) {
		if verbose && time.Since(lastProgress) > time.Second {
			if progress.Error != nil {
				fmt.Printf("Error downloading %s: %v\n", progress.Key, progress.Error)
			} else if progress.Completed {
				fmt.Printf("Completed: %s (%d bytes)\n", progress.Key, progress.BytesDownloaded)
			}
			lastProgress = time.Now()
		}
	}

	fmt.Printf("Cloning %s to %s...\n", sourceURL, destDir)
	
	// Start download
	ctx := context.Background()
	result, err := dl.DownloadFolder(ctx, parsedSource.Prefix, destDir, progressCallback)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	// Print results
	fmt.Printf("\nDownload completed!\n")
	fmt.Printf("Files: %d total, %d successful, %d failed\n", 
		result.TotalFiles, result.SuccessfulFiles, result.FailedFiles)
	fmt.Printf("Total size: %.2f MB\n", float64(result.TotalBytes)/(1024*1024))
	fmt.Printf("Duration: %v\n", result.Duration)

	if len(result.Errors) > 0 {
		fmt.Printf("\nErrors:\n")
		for _, err := range result.Errors {
			fmt.Printf("  - %v\n", err)
		}
		return fmt.Errorf("download completed with %d errors", len(result.Errors))
	}

	return nil
}

// SourceInfo holds parsed information from the source URL
type SourceInfo struct {
	Provider string
	Bucket   string
	Prefix   string
	Region   string
}

// parseSourceURL parses the source URL and extracts provider, bucket, and prefix
func parseSourceURL(sourceURL string) (*SourceInfo, error) {
	// Handle different URL formats
	if strings.HasPrefix(sourceURL, "s3://") {
		return parseS3URL(sourceURL)
	} else if strings.HasPrefix(sourceURL, "spaces://") {
		return parseSpacesURL(sourceURL)
	} else if strings.Contains(sourceURL, "digitaloceanspaces.com") {
		return parseDigitalOceanURL(sourceURL)
	}

	return nil, fmt.Errorf("unsupported URL format: %s", sourceURL)
}

func parseS3URL(sourceURL string) (*SourceInfo, error) {
	// s3://bucket-name/path/to/folder/
	u, err := url.Parse(sourceURL)
	if err != nil {
		return nil, err
	}

	return &SourceInfo{
		Provider: "s3",
		Bucket:   u.Host,
		Prefix:   strings.TrimPrefix(u.Path, "/"),
	}, nil
}

func parseSpacesURL(sourceURL string) (*SourceInfo, error) {
	// spaces://space-name/path/to/folder/
	u, err := url.Parse(sourceURL)
	if err != nil {
		return nil, err
	}

	return &SourceInfo{
		Provider: "digitalocean",
		Bucket:   u.Host,
		Prefix:   strings.TrimPrefix(u.Path, "/"),
	}, nil
}

func parseDigitalOceanURL(sourceURL string) (*SourceInfo, error) {
	// https://region.digitaloceanspaces.com/space-name/path/to/folder/
	u, err := url.Parse(sourceURL)
	if err != nil {
		return nil, err
	}

	// Extract region from hostname
	parts := strings.Split(u.Host, ".")
	if len(parts) < 2 || !strings.Contains(u.Host, "digitaloceanspaces.com") {
		return nil, fmt.Errorf("invalid DigitalOcean Spaces URL")
	}

	region := parts[0]
	pathParts := strings.SplitN(strings.TrimPrefix(u.Path, "/"), "/", 2)
	if len(pathParts) < 1 {
		return nil, fmt.Errorf("bucket name not found in URL")
	}

	bucket := pathParts[0]
	prefix := ""
	if len(pathParts) > 1 {
		prefix = pathParts[1]
	}

	return &SourceInfo{
		Provider: "digitalocean",
		Bucket:   bucket,
		Prefix:   prefix,
		Region:   region,
	}, nil
}

// getProviderConfig gets the provider configuration, merging CLI flags with config file
func getProviderConfig(cfg *config.Config, source *SourceInfo) (*config.ProviderConfig, error) {
	var providerConfig config.ProviderConfig

	// Start with config from file if available
	if providerName != "" {
		if pc, exists := cfg.Providers[providerName]; exists {
			providerConfig = pc
		}
	} else if source.Provider != "" {
		// Try to find a matching provider in config
		for _, pc := range cfg.Providers {
			if pc.Type == source.Provider {
				providerConfig = pc
				break
			}
		}
	}

	// Set provider type if not set
	if providerConfig.Type == "" {
		if source.Provider != "" {
			providerConfig.Type = source.Provider
		} else {
			return nil, fmt.Errorf("provider type not specified")
		}
	}

	// Override with CLI flags
	if accessKey != "" {
		providerConfig.AccessKey = accessKey
	}
	if secretKey != "" {
		providerConfig.SecretKey = secretKey
	}
	if region != "" {
		providerConfig.Region = region
	}
	if endpoint != "" {
		providerConfig.Endpoint = endpoint
	}
	if bucket != "" {
		providerConfig.Bucket = bucket
	} else if source.Bucket != "" {
		providerConfig.Bucket = source.Bucket
	}

	// Set region from URL if available and not already set
	if providerConfig.Region == "" && source.Region != "" {
		providerConfig.Region = source.Region
	}

	// Set default endpoint for DigitalOcean if not set
	if providerConfig.Type == "digitalocean" && providerConfig.Endpoint == "" {
		if providerConfig.Region != "" {
			providerConfig.Endpoint = fmt.Sprintf("https://%s.digitaloceanspaces.com", providerConfig.Region)
		}
	}

	// Validate required fields
	if providerConfig.AccessKey == "" {
		return nil, fmt.Errorf("access key not provided")
	}
	if providerConfig.SecretKey == "" {
		return nil, fmt.Errorf("secret key not provided")
	}
	if providerConfig.Bucket == "" {
		return nil, fmt.Errorf("bucket name not provided")
	}

	return &providerConfig, nil
} 