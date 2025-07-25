package ui

import (
	"fmt"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/schollz/progressbar/v3"

	"download-file-from-bucket/providers"
)

// ProgressTracker manages download progress display
type ProgressTracker struct {
	totalFiles     int64
	totalBytes     int64
	completedFiles int64
	completedBytes int64
	failedFiles    int64
	startTime      time.Time
	progressBar    *progressbar.ProgressBar
	lastUpdate     time.Time
	lastStatsUpdate time.Time
	mutex          sync.Mutex
	verbose        bool
	quiet          bool
	errors         []error
}

// NewProgressTracker creates a new progress tracker
func NewProgressTracker(totalFiles int, totalBytes int64, verbose bool) *ProgressTracker {
	return NewProgressTrackerWithOptions(totalFiles, totalBytes, verbose, false)
}

// NewProgressTrackerWithOptions creates a new progress tracker with options
func NewProgressTrackerWithOptions(totalFiles int, totalBytes int64, verbose, quiet bool) *ProgressTracker {
	// Create progress bar
	bar := progressbar.NewOptions64(
		totalBytes,
		progressbar.OptionSetDescription("Downloading..."),
		progressbar.OptionSetWriter(color.Output),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetWidth(50),
		progressbar.OptionThrottle(500*time.Millisecond), // Increased throttle to reduce updates
		progressbar.OptionShowCount(),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        color.GreenString("█"),
			SaucerHead:    color.GreenString("█"),
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
		progressbar.OptionClearOnFinish(), // Clear progress bar when done
		progressbar.OptionSetRenderBlankState(false), // Don't render blank state
	)

	return &ProgressTracker{
		totalFiles:  int64(totalFiles),
		totalBytes:  totalBytes,
		startTime:   time.Now(),
		progressBar: bar,
		verbose:     verbose,
		quiet:       quiet,
		errors:      make([]error, 0),
	}
}

// UpdateProgress updates the progress based on a download progress event
func (pt *ProgressTracker) UpdateProgress(progress providers.DownloadProgress) {
	pt.mutex.Lock()
	defer pt.mutex.Unlock()

	now := time.Now()

	if progress.Error != nil {
		pt.failedFiles++
		pt.errors = append(pt.errors, progress.Error)
		// Only show critical errors immediately, others will be in summary
		if pt.verbose && len(pt.errors) <= 3 {
			color.Red("✗ %s - %v", progress.Key, progress.Error)
		}
	} else if progress.Completed {
		pt.completedFiles++
		pt.completedBytes += progress.BytesDownloaded
		
		// Update progress bar
		pt.progressBar.Add64(progress.BytesDownloaded)
		
		if pt.verbose && time.Since(pt.lastUpdate) > 2*time.Second {
			// Only show occasional updates in verbose mode to avoid spam
			color.Green("✓ %s (%.2f KB)", 
				progress.Key, 
				float64(progress.BytesDownloaded)/1024)
			pt.lastUpdate = now
		}
	}

	// Update description with current stats (but not too frequently)
	if time.Since(pt.lastStatsUpdate) > 1*time.Second {
		description := fmt.Sprintf("Files: %d/%d | Speed: %s/s | ETA: %s",
			pt.completedFiles,
			pt.totalFiles,
			pt.formatBytes(pt.calculateSpeed()),
			pt.calculateETA())
		
		pt.progressBar.Describe(description)
		pt.lastStatsUpdate = now
	}
	
	// For quiet mode, show simple progress every 10% or every 5 seconds
	if pt.quiet && (time.Since(pt.lastUpdate) > 5*time.Second || 
		(pt.completedBytes > 0 && (pt.completedBytes*10/pt.totalBytes) != ((pt.completedBytes-progress.BytesDownloaded)*10/pt.totalBytes))) {
		percentage := int(pt.completedBytes * 100 / pt.totalBytes)
		fmt.Printf("Progress: %d%% (%s/%s) - %d/%d files\n", 
			percentage,
			pt.formatBytes(pt.completedBytes),
			pt.formatBytes(pt.totalBytes),
			pt.completedFiles,
			pt.totalFiles)
		pt.lastUpdate = now
	}
}

// PrintStartMessage prints an attractive start message
func (pt *ProgressTracker) PrintStartMessage(source, destination string) {
	color.Cyan("\n╭─────────────────────────────────────────────────╮")
	color.Cyan("│              Download Starting                  │")
	color.Cyan("╰─────────────────────────────────────────────────╯")
	
	fmt.Printf("📁 Source:      %s\n", color.BlueString(source))
	fmt.Printf("📂 Destination: %s\n", color.BlueString(destination))
	fmt.Printf("📊 Total Files: %s\n", color.YellowString("%d", pt.totalFiles))
	fmt.Printf("💾 Total Size:  %s\n", color.YellowString(pt.formatBytes(pt.totalBytes)))
	fmt.Printf("⏰ Started:     %s\n", color.MagentaString(pt.startTime.Format("15:04:05")))
	fmt.Println()
}

// PrintCompletionMessage prints a summary when download is complete
func (pt *ProgressTracker) PrintCompletionMessage(zipPath string, zipSize int64, sourceRemoved bool) {
	pt.mutex.Lock()
	defer pt.mutex.Unlock()

	duration := time.Since(pt.startTime)
	avgSpeed := float64(pt.completedBytes) / duration.Seconds()

	fmt.Println()
	if pt.failedFiles == 0 {
		color.Green("✅ Download Completed Successfully!")
	} else {
		color.Yellow("⚠️ Download Completed with Some Errors")
	}

	color.Cyan("\n╭─────────────────────────────────────────────────╮")
	color.Cyan("│                   Summary                       │")
	color.Cyan("╰─────────────────────────────────────────────────╯")

	fmt.Printf("📊 Files:       %s successful, %s failed, %s total\n",
		color.GreenString("%d", pt.completedFiles),
		color.RedString("%d", pt.failedFiles),
		color.BlueString("%d", pt.totalFiles))

	fmt.Printf("💾 Data:        %s downloaded\n",
		color.GreenString(pt.formatBytes(pt.completedBytes)))

	fmt.Printf("⚡ Speed:       %s/s average\n",
		color.MagentaString(pt.formatBytes(int64(avgSpeed))))

	fmt.Printf("⏱️  Duration:    %s\n",
		color.CyanString(pt.formatDuration(duration)))

	if zipPath != "" {
		fmt.Printf("📦 Zip:         %s (%s)\n",
			color.BlueString(zipPath),
			color.GreenString(pt.formatBytes(zipSize)))
		
		compressionRatio := (1.0 - float64(zipSize)/float64(pt.completedBytes)) * 100
		fmt.Printf("🗜️  Compression: %s saved\n",
			color.YellowString("%.1f%%", compressionRatio))
	}

	if sourceRemoved {
		fmt.Printf("🗑️  Cleanup:     %s\n", color.GreenString("Source directory removed"))
	}

	if len(pt.errors) > 0 {
		color.Red("\n❌ Errors encountered:")
		for i, err := range pt.errors {
			if i < 5 { // Show only first 5 errors
				color.Red("   • %v", err)
			} else if i == 5 {
				color.Red("   • ... and %d more errors", len(pt.errors)-5)
				break
			}
		}
	}

	fmt.Println()
}

// PrintZipProgress prints zip creation progress
func (pt *ProgressTracker) PrintZipProgress(zipPath string) {
	color.Yellow("\n📦 Creating zip archive: %s", zipPath)
}

// calculateSpeed calculates current download speed
func (pt *ProgressTracker) calculateSpeed() int64 {
	elapsed := time.Since(pt.startTime).Seconds()
	if elapsed == 0 {
		return 0
	}
	return int64(float64(pt.completedBytes) / elapsed)
}

// calculateETA calculates estimated time to completion
func (pt *ProgressTracker) calculateETA() string {
	if pt.completedBytes == 0 {
		return "calculating..."
	}

	remainingBytes := pt.totalBytes - pt.completedBytes
	if remainingBytes <= 0 {
		return "00:00"
	}

	speed := pt.calculateSpeed()
	if speed == 0 {
		return "calculating..."
	}

	etaSeconds := float64(remainingBytes) / float64(speed)
	eta := time.Duration(etaSeconds) * time.Second

	return pt.formatDuration(eta)
}

// formatBytes formats bytes into human readable format
func (pt *ProgressTracker) formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// formatDuration formats duration into human readable format
func (pt *ProgressTracker) formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%02ds", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%02dm%02ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%02dh%02dm%02ds", int(d.Hours()), int(d.Minutes())%60, int(d.Seconds())%60)
}

// Finish completes the progress bar
func (pt *ProgressTracker) Finish() {
	pt.progressBar.Finish()
	fmt.Println() // Add single line break after progress bar
} 