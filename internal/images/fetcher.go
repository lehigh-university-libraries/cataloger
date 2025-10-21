package images

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Fetcher retrieves book images from various sources
type Fetcher struct {
	HTTPClient *http.Client
}

// NewFetcher creates a new image fetcher
func NewFetcher() *Fetcher {
	return &Fetcher{
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ImageSet represents the three key images for cataloging
type ImageSet struct {
	CoverPath         string
	TitlePagePath     string
	CopyrightPagePath string
}

// OpenLibraryBooksResponse represents the Open Library Books API response
type OpenLibraryBooksResponse map[string]struct {
	InfoURL      string `json:"info_url"`
	BibKey       string `json:"bib_key"`
	PreviewURL   string `json:"preview_url"`
	ThumbnailURL string `json:"thumbnail_url"`
	Details      struct {
		InfoURL     string   `json:"info_url"`
		OCLCNumbers []string `json:"oclc_numbers"`
		LCCN        []string `json:"lccn"`
		ISBN10      []string `json:"isbn_10"`
		ISBN13      []string `json:"isbn_13"`
		IA          []string `json:"ia"` // Internet Archive identifiers
		Key         string   `json:"key"`
		Title       string   `json:"title"`
	} `json:"details"`
}

// FetchImagesForISBN retrieves cover, title page, and copyright page images for a given ISBN
func (f *Fetcher) FetchImagesForISBN(isbn string, outputDir string) (*ImageSet, error) {
	slog.Info("Fetching images for ISBN", "isbn", isbn)

	imageSet := &ImageSet{}

	// Step 1: Get cover image from Open Library Covers API
	coverPath := filepath.Join(outputDir, fmt.Sprintf("%s_cover.jpg", isbn))
	if err := f.downloadCoverImage(isbn, coverPath); err != nil {
		slog.Warn("Failed to download cover image", "isbn", isbn, "error", err)
	} else {
		imageSet.CoverPath = coverPath
		slog.Info("Downloaded cover image", "isbn", isbn, "path", coverPath)
	}

	// Rate limiting: Sleep between Open Library API calls
	// Open Library allows 100 req/5min, so ~1 req/sec is safe
	time.Sleep(1 * time.Second)

	// Step 2: Try to get interior pages from Internet Archive
	titlePath := filepath.Join(outputDir, fmt.Sprintf("%s_title.jpg", isbn))
	copyrightPath := filepath.Join(outputDir, fmt.Sprintf("%s_copyright.jpg", isbn))

	iaID, err := f.getInternetArchiveID(isbn)
	if err == nil {
		slog.Info("Found Internet Archive identifier", "isbn", isbn, "ia_id", iaID)

		// Rate limiting: Sleep before hitting Internet Archive
		time.Sleep(500 * time.Millisecond)

		if err := f.downloadInteriorPages(iaID, titlePath, copyrightPath); err == nil {
			imageSet.TitlePagePath = titlePath
			imageSet.CopyrightPagePath = copyrightPath
			slog.Info("Downloaded interior pages from IA", "isbn", isbn, "ia_id", iaID)
		} else {
			slog.Warn("Failed to download interior pages from IA", "isbn", isbn, "ia_id", iaID, "error", err)
		}
	} else {
		slog.Debug("No Internet Archive ID found", "isbn", isbn, "error", err)
	}

	// Step 3: If we don't have interior pages yet, try Google Books
	if imageSet.TitlePagePath == "" || imageSet.CopyrightPagePath == "" {
		slog.Info("Trying Google Books for interior pages", "isbn", isbn)

		// Rate limiting
		time.Sleep(500 * time.Millisecond)

		if err := f.downloadGoogleBooksPages(isbn, imageSet, outputDir, titlePath, copyrightPath); err == nil {
			slog.Info("Downloaded pages from Google Books", "isbn", isbn)
		} else {
			slog.Warn("Failed to download pages from Google Books", "isbn", isbn, "error", err)
		}
	}

	// Check if we got at least one image
	if imageSet.CoverPath == "" && imageSet.TitlePagePath == "" && imageSet.CopyrightPagePath == "" {
		return nil, fmt.Errorf("no images could be downloaded for ISBN %s", isbn)
	}

	return imageSet, nil
}

// downloadCoverImage downloads a book cover from Open Library Covers API
func (f *Fetcher) downloadCoverImage(isbn, outputPath string) error {
	// Open Library Covers API: https://covers.openlibrary.org/b/isbn/{ISBN}-L.jpg
	url := fmt.Sprintf("https://covers.openlibrary.org/b/isbn/%s-L.jpg", isbn)

	resp, err := f.HTTPClient.Get(url)
	if err != nil {
		return fmt.Errorf("failed to fetch cover: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("cover API returned status %d", resp.StatusCode)
	}

	// Check if it's a placeholder image (sometimes OL returns a tiny placeholder)
	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read cover data: %w", err)
	}

	// If image is too small, it's probably a placeholder
	if len(imageData) < 1000 {
		return fmt.Errorf("cover image too small (likely placeholder)")
	}

	if err := os.WriteFile(outputPath, imageData, 0644); err != nil {
		return fmt.Errorf("failed to write cover file: %w", err)
	}

	return nil
}

// getInternetArchiveID queries Open Library Books API to get the Internet Archive identifier
func (f *Fetcher) getInternetArchiveID(isbn string) (string, error) {
	url := fmt.Sprintf("https://openlibrary.org/api/books?bibkeys=ISBN:%s&format=json&jscmd=details", isbn)

	resp, err := f.HTTPClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to query Open Library: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("open Library API returned status %d", resp.StatusCode)
	}

	var result OpenLibraryBooksResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode Open Library response: %w", err)
	}

	// Extract Internet Archive ID from the response
	key := fmt.Sprintf("ISBN:%s", isbn)
	if book, ok := result[key]; ok {
		if len(book.Details.IA) > 0 {
			return book.Details.IA[0], nil
		}
	}

	return "", fmt.Errorf("no Internet Archive identifier found for ISBN %s", isbn)
}

// downloadInteriorPages downloads title and copyright pages from Internet Archive
func (f *Fetcher) downloadInteriorPages(iaID, titlePath, copyrightPath string) error {
	// Internet Archive BookReader Images:
	// https://archive.org/download/{identifier}/{identifier}_jp2.zip/{identifier}_jp2/{identifier}_{page}.jp2
	//
	// For most books:
	// - Title page is typically pages 5-10
	// - Copyright page is typically pages 3-8
	//
	// We'll use heuristics to try common page numbers

	titlePageNums := []int{7, 6, 5, 8, 9, 10}
	copyrightPageNums := []int{4, 5, 3, 6, 2}

	// Try to download title page
	titleDownloaded := false
	for i, pageNum := range titlePageNums {
		url := fmt.Sprintf("https://archive.org/download/%s/page/n%d_w800.jpg", iaID, pageNum)
		if err := f.downloadImage(url, titlePath); err == nil {
			titleDownloaded = true
			slog.Debug("Downloaded title page", "ia_id", iaID, "page", pageNum)
			break
		}
		// Small delay between page attempts to avoid hammering IA
		if i < len(titlePageNums)-1 {
			time.Sleep(200 * time.Millisecond)
		}
	}

	if !titleDownloaded {
		slog.Warn("Could not download title page", "ia_id", iaID)
	}

	// Try to download copyright page
	copyrightDownloaded := false
	for i, pageNum := range copyrightPageNums {
		url := fmt.Sprintf("https://archive.org/download/%s/page/n%d_w800.jpg", iaID, pageNum)
		if err := f.downloadImage(url, copyrightPath); err == nil {
			copyrightDownloaded = true
			slog.Debug("Downloaded copyright page", "ia_id", iaID, "page", pageNum)
			break
		}
		// Small delay between page attempts to avoid hammering IA
		if i < len(copyrightPageNums)-1 {
			time.Sleep(200 * time.Millisecond)
		}
	}

	if !copyrightDownloaded {
		slog.Warn("Could not download copyright page", "ia_id", iaID)
	}

	if !titleDownloaded && !copyrightDownloaded {
		return fmt.Errorf("failed to download any interior pages")
	}

	return nil
}

// downloadImage downloads an image from a URL to a file
func (f *Fetcher) downloadImage(url, outputPath string) error {
	resp, err := f.HTTPClient.Get(url)
	if err != nil {
		return fmt.Errorf("failed to fetch image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("image URL returned status %d", resp.StatusCode)
	}

	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read image data: %w", err)
	}

	// Check if image is reasonably sized
	if len(imageData) < 1000 {
		return fmt.Errorf("image too small (likely invalid)")
	}

	// Google Books placeholder images are typically around 7-12KB
	// Real book page images are usually 50KB+
	if len(imageData) < 20000 {
		return fmt.Errorf("image too small (likely placeholder), size: %d bytes", len(imageData))
	}

	if err := os.WriteFile(outputPath, imageData, 0644); err != nil {
		return fmt.Errorf("failed to write image file: %w", err)
	}

	return nil
}

// downloadGoogleBooksPages attempts to download interior pages from Google Books
func (f *Fetcher) downloadGoogleBooksPages(isbn string, imageSet *ImageSet, outputDir, titlePath, copyrightPath string) error {
	// Google Books API to get volume info
	url := fmt.Sprintf("https://www.googleapis.com/books/v1/volumes?q=isbn:%s", isbn)

	resp, err := f.HTTPClient.Get(url)
	if err != nil {
		return fmt.Errorf("failed to query Google Books API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("google Books API returned status %d", resp.StatusCode)
	}

	var result struct {
		Items []struct {
			ID         string `json:"id"`
			VolumeInfo struct {
				Title      string            `json:"title"`
				ImageLinks map[string]string `json:"imageLinks"`
				AccessInfo struct {
					Viewability string `json:"viewability"`
				} `json:"accessInfo"`
			} `json:"volumeInfo"`
			AccessInfo struct {
				Viewability string `json:"viewability"`
			} `json:"accessInfo"`
		} `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode Google Books response: %w", err)
	}

	if len(result.Items) == 0 {
		return fmt.Errorf("no books found in Google Books for ISBN %s", isbn)
	}

	volumeID := result.Items[0].ID
	viewability := result.Items[0].AccessInfo.Viewability

	// Only proceed if the book has some preview available
	if viewability == "NO_PAGES" {
		return fmt.Errorf("no preview pages available in Google Books for ISBN %s", isbn)
	}

	slog.Info("Found Google Books volume", "isbn", isbn, "volume_id", volumeID, "viewability", viewability)

	// Try to download cover if we don't have one yet
	if imageSet.CoverPath == "" {
		coverURL := fmt.Sprintf("https://books.google.com/books/content?id=%s&printsec=frontcover&img=1&zoom=1&hl=en&w=1280", volumeID)
		coverPath := filepath.Join(outputDir, fmt.Sprintf("%s_cover.jpg", isbn))
		if err := f.downloadImage(coverURL, coverPath); err == nil {
			imageSet.CoverPath = coverPath
			slog.Info("Downloaded cover from Google Books", "isbn", isbn)
		} else {
			slog.Debug("Failed to download cover from Google Books", "isbn", isbn, "error", err)
		}
		time.Sleep(500 * time.Millisecond)
	}

	// Try to download specific pages using Google Books image API
	// Google Books uses page IDs like "PA1", "PA2", etc for page numbers
	// Title page is typically around PA5-PA10, copyright around PA2-PA6

	titleDownloaded := false
	copyrightDownloaded := false

	// Try title page - typically pages 5-10
	// Use zoom=1 for larger images (Google Books uses zoom=5 for thumbnails, zoom=1 for larger)
	titlePages := []string{"PA7", "PA6", "PA5", "PA8", "PA9", "PA10", "PP1", "PP2"}
	for _, pageID := range titlePages {
		url := fmt.Sprintf("https://books.google.com/books/content?id=%s&pg=%s&img=1&zoom=1&hl=en&w=1280", volumeID, pageID)
		slog.Debug("Trying title page URL", "url", url)
		if err := f.downloadImage(url, titlePath); err == nil {
			titleDownloaded = true
			imageSet.TitlePagePath = titlePath
			slog.Debug("Downloaded title page from Google Books", "isbn", isbn, "page", pageID)
			break
		}
		// Increased sleep to avoid rate limiting
		time.Sleep(500 * time.Millisecond)
	}

	// Try copyright page - typically pages 2-6
	// Use zoom=1 for larger images
	copyrightPages := []string{"PA4", "PA5", "PA3", "PA6", "PA2", "PP3", "PP4"}
	for _, pageID := range copyrightPages {
		url := fmt.Sprintf("https://books.google.com/books/content?id=%s&pg=%s&img=1&zoom=1&hl=en&w=1280", volumeID, pageID)
		if err := f.downloadImage(url, copyrightPath); err == nil {
			copyrightDownloaded = true
			imageSet.CopyrightPagePath = copyrightPath
			slog.Debug("Downloaded copyright page from Google Books", "isbn", isbn, "page", pageID)
			break
		}
		// Increased sleep to avoid rate limiting
		time.Sleep(500 * time.Millisecond)
	}

	if !titleDownloaded && !copyrightDownloaded {
		return fmt.Errorf("failed to download any interior pages from Google Books")
	}

	return nil
}

// CleanISBN removes hyphens and normalizes ISBN
func CleanISBN(isbn string) string {
	return strings.ReplaceAll(strings.TrimSpace(isbn), "-", "")
}
