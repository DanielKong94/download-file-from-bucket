package cmd

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"download-file-from-bucket/config"
	"download-file-from-bucket/downloader"
	"download-file-from-bucket/providers"
	"download-file-from-bucket/ui"
)

var (
	providerName     string
	concurrency      int
	accessKey        string
	secretKey        string
	region           string
	endpoint         string
	bucket           string
	createZip        bool
	zipPath          string
	removeSource     bool
	handleDuplicates bool
	quietMode        bool
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
  download-bucket clone --provider=aws --region=eu-west-1 s3://eu-bucket/files/ ./files
  download-bucket clone --zip s3://my-bucket/data/ ./local-data
  download-bucket clone --zip --zip-path=./my-archive.zip s3://my-bucket/data/ ./local-data
  download-bucket clone --quiet s3://my-bucket/data/ ./local-data`,
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
	cloneCmd.Flags().BoolVar(&createZip, "zip", false, "Create a zip file after downloading")
	cloneCmd.Flags().StringVar(&zipPath, "zip-path", "", "Path for the zip file (default: <destination>.zip)")
	cloneCmd.Flags().BoolVar(&removeSource, "remove-source", false, "Remove source directory after creating zip")
	cloneCmd.Flags().BoolVar(&handleDuplicates, "handle-duplicates", true, "Handle duplicate files by adding numbers (default: true)")
	cloneCmd.Flags().BoolVar(&quietMode, "quiet", false, "Use simple progress output instead of progress bar")
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

	// Create downloader with zip options
	dl := downloader.NewDownloader(provider, downloader.Options{
		Concurrency:      concurrency,
		Verbose:          verbose,
		CreateZip:        createZip,
		ZipPath:          zipPath,
		HandleDuplicates: handleDuplicates,
	})

	// Start download with UI
	ctx := context.Background()
	
	// First, get object list to initialize progress tracker
	objects, err := provider.ListObjects(ctx, parsedSource.Prefix)
	if err != nil {
		return fmt.Errorf("failed to list objects: %w", err)
	}

	if len(objects) == 0 {
		fmt.Printf("No files found with prefix: %s\n", parsedSource.Prefix)
		return nil
	}

	// Calculate total size
	var totalBytes int64
	for _, obj := range objects {
		totalBytes += obj.Size
	}

	// Create progress tracker
	var progressTracker *ui.ProgressTracker
	if quietMode {
		progressTracker = ui.NewProgressTrackerWithOptions(len(objects), totalBytes, verbose, true)
	} else {
		progressTracker = ui.NewProgressTracker(len(objects), totalBytes, verbose)
	}
	
	// Print start message
	progressTracker.PrintStartMessage(sourceURL, destDir)
	
	// Progress callback using the UI tracker
	progressCallback := func(progress providers.DownloadProgress) {
		progressTracker.UpdateProgress(progress)
	}
	
	// Start download
	result, err := dl.DownloadFolder(ctx, parsedSource.Prefix, destDir, progressCallback)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	// Finish progress bar
	progressTracker.Finish()

	// Handle zip creation
	var sourceRemoved bool
	if result.ZipPath != "" {
		progressTracker.PrintZipProgress(result.ZipPath)
		
		// Remove source directory if requested
		if removeSource && result.SuccessfulFiles > 0 {
			if err := removeDirectory(destDir); err != nil {
				if verbose {
					fmt.Printf("Warning: Failed to remove source directory: %v\n", err)
				}
			} else {
				sourceRemoved = true
			}
		}
	}

	// Print completion summary
	progressTracker.PrintCompletionMessage(result.ZipPath, result.ZipSize, sourceRemoved)

	// Return error if there were failures (errors are already shown in the summary)
	if len(result.Errors) > 0 {
		return fmt.Errorf("download completed with %d errors", len(result.Errors))
	}

	return nil
}

// removeDirectory removes a directory and all its contents
func removeDirectory(dir string) error {
	return os.RemoveAll(dir)
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