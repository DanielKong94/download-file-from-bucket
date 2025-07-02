package downloader

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"download-file-from-bucket/providers"
)

// Downloader handles downloading files from cloud storage
type Downloader struct {
	provider    providers.Provider
	concurrency int
	verbose     bool
}

// Options for configuring the downloader
type Options struct {
	Concurrency int
	Verbose     bool
}

// NewDownloader creates a new downloader
func NewDownloader(provider providers.Provider, opts Options) *Downloader {
	if opts.Concurrency <= 0 {
		opts.Concurrency = 5 // Default concurrency
	}

	return &Downloader{
		provider:    provider,
		concurrency: opts.Concurrency,
		verbose:     opts.Verbose,
	}
}

// DownloadResult represents the result of a download operation
type DownloadResult struct {
	TotalFiles      int
	SuccessfulFiles int
	FailedFiles     int
	TotalBytes      int64
	Duration        time.Duration
	Errors          []error
}

// DownloadFolder downloads all files from a folder/prefix to a local directory
func (d *Downloader) DownloadFolder(ctx context.Context, prefix, localDir string, progressCallback func(providers.DownloadProgress)) (*DownloadResult, error) {
	startTime := time.Now()
	
	// List all objects with the given prefix
	if d.verbose {
		fmt.Printf("Listing objects with prefix: %s\n", prefix)
	}
	
	objects, err := d.provider.ListObjects(ctx, prefix)
	if err != nil {
		return nil, fmt.Errorf("failed to list objects: %w", err)
	}

	if len(objects) == 0 {
		return &DownloadResult{
			Duration: time.Since(startTime),
		}, nil
	}

	if d.verbose {
		fmt.Printf("Found %d objects to download\n", len(objects))
	}

	// Create local directory if it doesn't exist
	if err := os.MkdirAll(localDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create local directory: %w", err)
	}

	// Create a channel for download jobs
	jobs := make(chan providers.Object, len(objects))
	results := make(chan providers.DownloadProgress, len(objects))

	// Start worker goroutines
	var wg sync.WaitGroup
	for i := 0; i < d.concurrency; i++ {
		wg.Add(1)
		go d.downloadWorker(ctx, &wg, jobs, results, prefix, localDir)
	}

	// Send jobs to workers
	go func() {
		defer close(jobs)
		for _, obj := range objects {
			jobs <- obj
		}
	}()

	// Collect results
	go func() {
		wg.Wait()
		close(results)
	}()

	// Process results and call progress callback
	result := &DownloadResult{}
	for progress := range results {
		result.TotalFiles++
		result.TotalBytes += progress.TotalBytes

		if progress.Error != nil {
			result.FailedFiles++
			result.Errors = append(result.Errors, progress.Error)
		} else {
			result.SuccessfulFiles++
		}

		if progressCallback != nil {
			progressCallback(progress)
		}
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

// downloadWorker is a worker goroutine that downloads objects
func (d *Downloader) downloadWorker(ctx context.Context, wg *sync.WaitGroup, jobs <-chan providers.Object, results chan<- providers.DownloadProgress, prefix, localDir string) {
	defer wg.Done()

	for obj := range jobs {
		progress := providers.DownloadProgress{
			Key:        obj.Key,
			TotalBytes: obj.Size,
		}

		// Download the object
		if err := d.downloadObject(ctx, obj, prefix, localDir); err != nil {
			progress.Error = err
		} else {
			progress.BytesDownloaded = obj.Size
			progress.Completed = true
		}

		results <- progress
	}
}

// downloadObject downloads a single object
func (d *Downloader) downloadObject(ctx context.Context, obj providers.Object, prefix, localDir string) error {
	// Calculate local file path
	relativePath := obj.Key
	if prefix != "" && len(obj.Key) > len(prefix) {
		relativePath = obj.Key[len(prefix):]
		if relativePath[0] == '/' {
			relativePath = relativePath[1:]
		}
	}
	
	localPath := filepath.Join(localDir, relativePath)
	
	// Skip if it's a directory (ends with /)
	if obj.Key[len(obj.Key)-1] == '/' {
		return os.MkdirAll(localPath, 0755)
	}

	// Create directory for the file if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory for %s: %w", localPath, err)
	}

	// Download the object
	reader, err := d.provider.DownloadObject(ctx, obj.Key)
	if err != nil {
		return err
	}
	defer reader.Close()

	// Create local file
	file, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create local file %s: %w", localPath, err)
	}
	defer file.Close()

	// Copy data
	_, err = io.Copy(file, reader)
	if err != nil {
		return fmt.Errorf("failed to write data to %s: %w", localPath, err)
	}

	if d.verbose {
		fmt.Printf("Downloaded: %s -> %s\n", obj.Key, localPath)
	}

	return nil
}

// Close cleans up resources
func (d *Downloader) Close() error {
	return d.provider.Close()
} 