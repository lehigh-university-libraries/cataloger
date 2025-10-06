package catalog

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Client represents a catalog API client (VuFind or FOLIO)
type Client struct {
	BaseURL     string
	CatalogType string
	APIKey      string
	httpClient  *http.Client
}

// CatalogRecord represents a record from the catalog with MARC and image URLs
type CatalogRecord struct {
	ID            string `json:"id"`
	MARCRecord    string `json:"marc_record"`     // Raw MARC record (MARC21 format)
	CoverImageURL string `json:"cover_image_url"` // URL to cover image
	TitlePageURL  string `json:"title_page_url"`  // URL to title page image
	CopyrightURL  string `json:"copyright_url"`   // URL to copyright page image
}

// NewClient creates a new catalog client
func NewClient(catalogType, baseURL, apiKey string) *Client {
	return &Client{
		BaseURL:     baseURL,
		CatalogType: catalogType,
		APIKey:      apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// FetchRecords fetches records from the catalog
func (c *Client) FetchRecords(limit int) ([]CatalogRecord, error) {
	switch c.CatalogType {
	case "vufind":
		return c.fetchFromVuFind(limit)
	case "folio":
		return c.fetchFromFOLIO(limit)
	default:
		return nil, fmt.Errorf("unsupported catalog type: %s", c.CatalogType)
	}
}

// fetchFromVuFind fetches records from a VuFind instance
func (c *Client) fetchFromVuFind(limit int) ([]CatalogRecord, error) {
	// VuFind API search endpoint
	searchURL := fmt.Sprintf("%s/api/v1/search?limit=%d&sort=random", c.BaseURL, limit)

	resp, err := c.httpClient.Get(searchURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from VuFind: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("VuFind API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse VuFind response
	var vufindResp struct {
		Records []struct {
			ID     string `json:"id"`
			Fields struct {
				Title  []string `json:"title"`
				Author []string `json:"author"`
			} `json:"fields"`
		} `json:"records"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&vufindResp); err != nil {
		return nil, fmt.Errorf("failed to decode VuFind response: %w", err)
	}

	records := make([]CatalogRecord, 0, len(vufindResp.Records))
	for _, rec := range vufindResp.Records {
		// Fetch full MARC record for each ID
		marcRecord, err := c.fetchMARCFromVuFind(rec.ID)
		if err != nil {
			// Skip records we can't fetch MARC for
			continue
		}

		// Fetch image URLs if available
		coverURL, titlePageURL, copyrightURL := c.fetchImageURLsFromVuFind(rec.ID)

		records = append(records, CatalogRecord{
			ID:            rec.ID,
			MARCRecord:    marcRecord,
			CoverImageURL: coverURL,
			TitlePageURL:  titlePageURL,
			CopyrightURL:  copyrightURL,
		})
	}

	return records, nil
}

// fetchMARCFromVuFind fetches the MARC record for a specific record ID
func (c *Client) fetchMARCFromVuFind(recordID string) (string, error) {
	// VuFind MARC export endpoint
	marcURL := fmt.Sprintf("%s/Record/%s/Export?style=MARC", c.BaseURL, url.PathEscape(recordID))

	resp, err := c.httpClient.Get(marcURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch MARC: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("MARC fetch returned status %d", resp.StatusCode)
	}

	marcData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read MARC data: %w", err)
	}

	return string(marcData), nil
}

// fetchImageURLsFromVuFind fetches image URLs for a record
func (c *Client) fetchImageURLsFromVuFind(recordID string) (cover, titlePage, copyright string) {
	// This is placeholder logic - actual implementation depends on your VuFind setup
	// Some institutions store image URLs in MARC 856 fields or have separate image services

	// Example: construct cover image URL from record ID
	cover = fmt.Sprintf("%s/Cover/Show?id=%s&size=large", c.BaseURL, url.QueryEscape(recordID))

	// Title page and copyright URLs would come from your specific setup
	// For now, we'll leave them empty unless found in MARC 856 fields
	return cover, "", ""
}

// fetchFromFOLIO fetches records from a FOLIO instance
func (c *Client) fetchFromFOLIO(limit int) ([]CatalogRecord, error) {
	// FOLIO requires authentication via Okapi headers
	if c.APIKey == "" {
		return nil, fmt.Errorf("API key required for FOLIO")
	}

	// FOLIO API endpoint for MARC records
	searchURL := fmt.Sprintf("%s/source-storage/stream/marc-record-identifiers?limit=%d", c.BaseURL, limit)

	req, err := http.NewRequest("POST", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create FOLIO request: %w", err)
	}

	// Add Okapi headers
	req.Header.Set("X-Okapi-Token", c.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from FOLIO: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("FOLIO API returned status %d: %s", resp.StatusCode, string(body))
	}

	// TODO: Parse FOLIO response and fetch MARC records
	// This is a placeholder - actual implementation depends on FOLIO API structure

	return nil, fmt.Errorf("FOLIO support not yet implemented")
}
