package downloader

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"download-file-from-bucket/providers"
)

// Downloader handles downloading files from cloud storage
type Downloader struct {
	provider         providers.Provider
	concurrency      int
	verbose          bool
	createZip        bool
	zipPath          string
	handleDuplicates bool
}

// Options for configuring the downloader
type Options struct {
	Concurrency       int
	Verbose           bool
	CreateZip         bool
	ZipPath           string
	HandleDuplicates  bool
}

// NewDownloader creates a new downloader
func NewDownloader(provider providers.Provider, opts Options) *Downloader {
	if opts.Concurrency <= 0 {
		opts.Concurrency = 5 // Default concurrency
	}

	return &Downloader{
		provider:         provider,
		concurrency:      opts.Concurrency,
		verbose:          opts.Verbose,
		createZip:        opts.CreateZip,
		zipPath:          opts.ZipPath,
		handleDuplicates: opts.HandleDuplicates,
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
	ZipPath         string
	ZipSize         int64
	Objects         []providers.Object // Add this for UI calculation
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
	result.Objects = objects // Store objects for UI

	// Create zip file if requested
	if d.createZip && result.SuccessfulFiles > 0 {
		zipPath := d.zipPath
		if zipPath == "" {
			// Generate default zip filename
			zipPath = localDir + ".zip"
		}

		if d.verbose {
			fmt.Printf("Creating zip file: %s\n", zipPath)
		}

		zipSize, err := d.createZipFile(localDir, zipPath)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("failed to create zip file: %w", err))
		} else {
			result.ZipPath = zipPath
			result.ZipSize = zipSize
			if d.verbose {
				fmt.Printf("Zip file created successfully: %s (%.2f MB)\n", zipPath, float64(zipSize)/(1024*1024))
			}
		}
	}

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

	// Handle duplicate file names by adding numbers if enabled
	finalPath := localPath
	if d.handleDuplicates {
		finalPath = d.getUniqueFilePath(localPath)
	}

	// Download the object
	reader, err := d.provider.DownloadObject(ctx, obj.Key)
	if err != nil {
		return err
	}
	defer reader.Close()

	// Create local file
	file, err := os.Create(finalPath)
	if err != nil {
		// Check if the error is due to the path being a directory
		if stat, statErr := os.Stat(finalPath); statErr == nil && stat.IsDir() {
			// The path exists as a directory, try to get a unique name
			if d.handleDuplicates {
				alternativePath := d.getUniqueFilePath(finalPath)
				file, err = os.Create(alternativePath)
				if err != nil {
					return fmt.Errorf("failed to create local file %s (alternative path %s): %w", finalPath, alternativePath, err)
				}
				finalPath = alternativePath
				if d.verbose {
					fmt.Printf("Created file with alternative path due to directory conflict: %s\n", alternativePath)
				}
			} else {
				return fmt.Errorf("failed to create local file %s: path exists as directory, enable --handle-duplicates to resolve automatically", finalPath)
			}
		} else {
			return fmt.Errorf("failed to create local file %s: %w", finalPath, err)
		}
	}
	defer file.Close()

	// Copy data
	_, err = io.Copy(file, reader)
	if err != nil {
		return fmt.Errorf("failed to write data to %s: %w", localPath, err)
	}

	if d.verbose {
		fmt.Printf("Downloaded: %s -> %s\n", obj.Key, finalPath)
	}

	return nil
}

// getUniqueFilePath generates a unique file path by adding numbers if file already exists
func (d *Downloader) getUniqueFilePath(originalPath string) string {
	// Check if path exists and what type it is
	if stat, err := os.Stat(originalPath); err == nil {
		// Path exists, check if it's a directory or file
		if stat.IsDir() {
			// It's a directory, we need a different name for the file
			if d.verbose {
				fmt.Printf("Directory exists with same name, creating alternative file name: %s\n", originalPath)
			}
		} else {
			// It's a file, we need a different name
			if d.verbose {
				fmt.Printf("File exists, creating alternative name: %s\n", originalPath)
			}
		}
	} else if os.IsNotExist(err) {
		// Path doesn't exist, safe to use
		return originalPath
	} else {
		// Some other error occurred, but let's try to continue
		if d.verbose {
			fmt.Printf("Error checking path %s: %v, trying alternative name\n", originalPath, err)
		}
	}

	// Generate a unique name
	dir := filepath.Dir(originalPath)
	filename := filepath.Base(originalPath)
	ext := filepath.Ext(filename)
	nameWithoutExt := strings.TrimSuffix(filename, ext)

	counter := 1
	for {
		// Generate new filename: filename_1.ext, filename_2.ext, etc.
		newFilename := fmt.Sprintf("%s_%d%s", nameWithoutExt, counter, ext)
		newPath := filepath.Join(dir, newFilename)
		
		// Check if this new path exists (either as file or directory)
		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			if d.verbose {
				fmt.Printf("Using alternative name: %s -> %s\n", originalPath, newPath)
			}
			return newPath
		}
		
		counter++
		// Safety check to prevent infinite loop
		if counter > 9999 {
			// If we reach 9999 duplicates, use timestamp
			timestamp := time.Now().Unix()
			newFilename := fmt.Sprintf("%s_%d%s", nameWithoutExt, timestamp, ext)
			newPath := filepath.Join(dir, newFilename)
			if d.verbose {
				fmt.Printf("Too many duplicates, using timestamp: %s -> %s\n", originalPath, newPath)
			}
			return newPath
		}
	}
}

// createZipFile creates a zip file from the downloaded directory
func (d *Downloader) createZipFile(sourceDir, zipPath string) (int64, error) {
	// Create zip file
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return 0, fmt.Errorf("failed to create zip file: %w", err)
	}
	defer zipFile.Close()

	// Create zip writer
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Walk through the source directory
	err = filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the root directory itself
		if path == sourceDir {
			return nil
		}

		// Get relative path for zip entry
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		// Convert to forward slashes for zip compatibility
		relPath = strings.ReplaceAll(relPath, string(os.PathSeparator), "/")

		// Create zip entry header
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = relPath
		header.Method = zip.Deflate

		// Handle directories
		if info.IsDir() {
			header.Name += "/"
			_, err = zipWriter.CreateHeader(header)
			return err
		}

		// Create zip entry for file
		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}

		// Open source file
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		// Copy file content to zip
		_, err = io.Copy(writer, file)
		return err
	})

	if err != nil {
		return 0, fmt.Errorf("failed to create zip archive: %w", err)
	}

	// Get zip file size
	stat, err := zipFile.Stat()
	if err != nil {
		return 0, fmt.Errorf("failed to get zip file size: %w", err)
	}

	return stat.Size(), nil
}

// CreateZipFilePublic is a public wrapper for createZipFile for testing purposes
func (d *Downloader) CreateZipFilePublic(sourceDir, zipPath string) (int64, error) {
	return d.createZipFile(sourceDir, zipPath)
}

// GetUniqueFilePathPublic is a public wrapper for getUniqueFilePath for testing purposes
func (d *Downloader) GetUniqueFilePathPublic(originalPath string) string {
	// Enable verbose for testing
	d.verbose = true
	return d.getUniqueFilePath(originalPath)
}

// Close cleans up resources
func (d *Downloader) Close() error {
	return d.provider.Close()
} 