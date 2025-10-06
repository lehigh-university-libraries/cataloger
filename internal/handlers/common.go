package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/lehigh-university-libraries/cataloger/internal/cataloging"
	"github.com/lehigh-university-libraries/cataloger/internal/models"
	"github.com/lehigh-university-libraries/cataloger/internal/storage"
)

type Handler struct {
	sessionStore      *storage.SessionStore
	catalogingService *cataloging.Service
}

type ImageProcessResult struct {
	ImageFilename string
	ImageFilePath string
	ImageType     string
	Width         int
	Height        int
	Provider      string
	Model         string
}

func New() *Handler {
	return &Handler{
		sessionStore:      storage.New(),
		catalogingService: cataloging.NewService(),
	}
}

// Response helpers
func (h *Handler) writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("Unable to encode JSON response", "err", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (h *Handler) writeError(w http.ResponseWriter, message string, code int) {
	slog.Error(message)
	http.Error(w, message, code)
}

// Session helpers
func (h *Handler) getSessionOrError(w http.ResponseWriter, sessionID string) (*models.CatalogSession, bool) {
	session, exists := h.sessionStore.Get(sessionID)
	if !exists {
		h.writeError(w, "Session not found", http.StatusNotFound)
		return nil, false
	}
	return session, true
}

// File operation helpers
func (h *Handler) ensureUploadsDir() error {
	uploadsDir := "uploads"
	return os.MkdirAll(uploadsDir, 0755)
}

func (h *Handler) createImageSession(sessionID string, result *ImageProcessResult) *models.CatalogSession {
	session := &models.CatalogSession{
		ID:        sessionID,
		Images:    []models.ImageItem{},
		Provider:  result.Provider,
		Model:     result.Model,
		CreatedAt: time.Now(),
	}

	imageItem := models.ImageItem{
		ID:          "img_1",
		ImagePath:   result.ImageFilename,
		ImageURL:    "/static/uploads/" + result.ImageFilename,
		ImageType:   result.ImageType,
		ImageWidth:  result.Width,
		ImageHeight: result.Height,
	}

	session.Images = []models.ImageItem{imageItem}

	// Generate MARC record if this is a title page
	if result.ImageType == "title_page" {
		slog.Info("Generating MARC record for title page", "session_id", sessionID, "provider", result.Provider, "model", result.Model)
		marcRecord, err := h.catalogingService.GenerateMARCFromImage(result.ImageFilePath, result.Provider, result.Model)
		if err != nil {
			slog.Error("Failed to generate MARC record", "error", err)
			// Don't fail the session creation, just log the error
			session.MARCRecord = "Error generating MARC record: " + err.Error()
		} else {
			session.MARCRecord = marcRecord
			slog.Info("MARC record generated successfully", "session_id", sessionID, "length", len(marcRecord))
		}
	}

	return session
}
