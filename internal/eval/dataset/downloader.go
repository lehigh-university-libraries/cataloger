package dataset

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	// HuggingFace dataset repository
	HFDatasetRepo = "instdin/institutional-books-1.0"

	// HuggingFace URLs
	HFResolveURL = "https://huggingface.co/datasets/%s/resolve/main/%s"

	// Default cache directory (similar to Python's datasets library)
	DefaultCacheDir = "~/.cache/huggingface/datasets"
)

// DownloadConfig configures dataset downloading
type DownloadConfig struct {
	CacheDir      string
	ForceDownload bool
	Token         string // HuggingFace token for private datasets
}

// Downloader handles downloading and caching datasets from HuggingFace
type Downloader struct {
	config DownloadConfig
}

// NewDownloader creates a new dataset downloader
func NewDownloader(config DownloadConfig) *Downloader {
	if config.CacheDir == "" {
		config.CacheDir = DefaultCacheDir
	}

	// Expand ~ to home directory
	if strings.HasPrefix(config.CacheDir, "~") {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			config.CacheDir = filepath.Join(homeDir, config.CacheDir[1:])
		}
	}

	return &Downloader{
		config: config,
	}
}

// DownloadDataset downloads the Institutional Books dataset from HuggingFace
// Returns the path to the cached dataset file
func (d *Downloader) DownloadDataset(filename string) (string, error) {
	// Create cache directory if it doesn't exist
	cacheDir := filepath.Join(d.config.CacheDir, HFDatasetRepo)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create cache directory: %w", err)
	}

	cachedPath := filepath.Join(cacheDir, filename)

	// Check if file already exists in cache
	if !d.config.ForceDownload {
		if _, err := os.Stat(cachedPath); err == nil {
			slog.Info("Using cached dataset", "path", cachedPath)
			return cachedPath, nil
		}
	}

	// Download the file
	slog.Info("Downloading dataset from HuggingFace", "repo", HFDatasetRepo, "file", filename)

	url := fmt.Sprintf(HFResolveURL, HFDatasetRepo, filename)

	if err := d.downloadFile(url, cachedPath); err != nil {
		return "", fmt.Errorf("failed to download dataset: %w", err)
	}

	slog.Info("Dataset downloaded successfully", "path", cachedPath)
	return cachedPath, nil
}

// downloadFile downloads a file from a URL to a local path
func (d *Downloader) downloadFile(url, destPath string) error {
	// Create HTTP request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add HuggingFace token if provided
	if d.config.Token != "" {
		req.Header.Set("Authorization", "Bearer "+d.config.Token)
	}

	// Execute request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	// Create temporary file
	tempPath := destPath + ".tmp"
	out, err := os.Create(tempPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	// Download with progress tracking
	totalSize := resp.ContentLength
	downloaded := int64(0)

	// Create a buffer for reading
	buf := make([]byte, 32*1024) // 32KB buffer

	for {
		nr, er := resp.Body.Read(buf)
		if nr > 0 {
			nw, ew := out.Write(buf[0:nr])
			if nw < 0 || nr < nw {
				nw = 0
				if ew == nil {
					ew = fmt.Errorf("invalid write result")
				}
			}
			downloaded += int64(nw)

			// Log progress every 10MB
			if downloaded%(10*1024*1024) == 0 {
				progress := float64(downloaded) / float64(totalSize) * 100
				slog.Debug("Download progress",
					"downloaded_mb", downloaded/(1024*1024),
					"total_mb", totalSize/(1024*1024),
					"progress", fmt.Sprintf("%.1f%%", progress))
			}

			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}

	if err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("download failed: %w", err)
	}

	// Move temp file to final location
	if err := os.Rename(tempPath, destPath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to move file: %w", err)
	}

	return nil
}

// ListAvailableFiles lists available dataset files from HuggingFace
// Note: This is a simplified version - HuggingFace doesn't have a direct API for this
// In practice, you'd need to know the filenames or use the HuggingFace Hub API
func (d *Downloader) ListAvailableFiles() []string {
	// Common file patterns for Institutional Books dataset
	return []string{
		"data/train-00000-of-00001.parquet", // If using Parquet format
		"institutional-books-1.0.jsonl",     // If using JSONL format
		"train.jsonl",
		"test.jsonl",
		"validation.jsonl",
	}
}

// GetCachePath returns the path where a dataset file would be cached
func (d *Downloader) GetCachePath(filename string) string {
	cacheDir := filepath.Join(d.config.CacheDir, HFDatasetRepo)
	return filepath.Join(cacheDir, filename)
}

// ClearCache removes all cached dataset files
func (d *Downloader) ClearCache() error {
	cacheDir := filepath.Join(d.config.CacheDir, HFDatasetRepo)
	slog.Info("Clearing cache", "path", cacheDir)
	return os.RemoveAll(cacheDir)
}

// LoadOrDownload loads a dataset from cache or downloads if not present
func LoadOrDownload(filename string, config DownloadConfig) (*Loader, error) {
	downloader := NewDownloader(config)

	// Download or use cached version
	datasetPath, err := downloader.DownloadDataset(filename)
	if err != nil {
		return nil, err
	}

	// Create loader with the downloaded/cached path
	return NewLoader(datasetPath), nil
}
