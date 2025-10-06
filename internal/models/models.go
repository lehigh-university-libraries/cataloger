package models

import "time"

// CatalogSession represents a book cataloging session
type CatalogSession struct {
	ID         string      `json:"id"`
	Images     []ImageItem `json:"images"`
	MARCRecord string      `json:"marc_record,omitempty"`
	Provider   string      `json:"provider,omitempty"`
	Model      string      `json:"model,omitempty"`
	CreatedAt  time.Time   `json:"created_at"`
}

// ImageItem represents an uploaded book image
type ImageItem struct {
	ID          string `json:"id"`
	ImagePath   string `json:"image_path"`
	ImageURL    string `json:"image_url"`
	ImageType   string `json:"image_type"` // "cover", "title_page", "copyright"
	ImageWidth  int    `json:"image_width"`
	ImageHeight int    `json:"image_height"`
}
