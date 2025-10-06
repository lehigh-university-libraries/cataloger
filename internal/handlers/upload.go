package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

func (h *Handler) HandleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		h.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check if this is a JSON request with image URL
	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		h.handleURLUpload(w, r)
		return
	}

	// Handle file upload
	h.handleFileUpload(w, r)
}

func (h *Handler) handleURLUpload(w http.ResponseWriter, r *http.Request) {
	var request struct {
		ImageURL  string `json:"image_url"`
		ImageType string `json:"image_type"` // "cover", "title_page", "copyright"
		Provider  string `json:"provider"`
		Model     string `json:"model"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.writeError(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	if request.ImageURL == "" {
		h.writeError(w, "image_url is required", http.StatusBadRequest)
		return
	}

	// Validate image type
	if request.ImageType != "" && request.ImageType != "cover" && request.ImageType != "title_page" && request.ImageType != "copyright" {
		h.writeError(w, "Invalid image_type. Must be 'cover', 'title_page', or 'copyright'", http.StatusBadRequest)
		return
	}

	sessionID, err := h.createSessionFromURL(request.ImageURL, request.ImageType, request.Provider, request.Model)
	if err != nil {
		h.writeError(w, "Failed to process image URL: "+err.Error(), http.StatusBadRequest)
		return
	}

	response := map[string]any{
		"session_id": sessionID,
		"message":    "Successfully processed image from URL",
		"images":     1,
		"source":     "url",
	}

	h.writeJSON(w, response)
}

func (h *Handler) handleFileUpload(w http.ResponseWriter, r *http.Request) {
	file, header, err := r.FormFile("files")
	if err != nil {
		file, header, err = r.FormFile("file")
		if err != nil {
			h.writeError(w, "Failed to read file: "+err.Error(), http.StatusBadRequest)
			return
		}
	}
	defer file.Close()

	// Extract image type and provider/model from form data
	imageType := r.FormValue("image_type") // "cover", "title_page", "copyright"
	provider := r.FormValue("provider")
	model := r.FormValue("model")

	if err := h.ensureUploadsDir(); err != nil {
		h.writeError(w, "Failed to create uploads directory: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Limit file size to 10MB
	fileData, err := io.ReadAll(io.LimitReader(file, 10*1024*1024))
	if err != nil {
		h.writeError(w, "Failed to read file contents: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Validate file size
	if len(fileData) >= 10*1024*1024 {
		h.writeError(w, "File too large (max 10MB)", http.StatusBadRequest)
		return
	}

	// Validate image type
	if imageType != "" && imageType != "cover" && imageType != "title_page" && imageType != "copyright" {
		h.writeError(w, "Invalid image_type. Must be 'cover', 'title_page', or 'copyright'", http.StatusBadRequest)
		return
	}

	result, err := h.processImageFile(fileData, header.Filename, imageType, provider, model)
	if err != nil {
		h.writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Use filename (without extension) as session name, with timestamp for uniqueness
	baseFilename := strings.TrimSuffix(header.Filename, filepath.Ext(header.Filename))
	sessionID := fmt.Sprintf("%s_%d", baseFilename, time.Now().Unix())

	session := h.createImageSession(sessionID, result)
	h.sessionStore.Set(sessionID, session)

	response := map[string]any{
		"session_id": sessionID,
		"message":    "Successfully uploaded 1 image",
		"images":     1,
	}

	h.writeJSON(w, response)
}
