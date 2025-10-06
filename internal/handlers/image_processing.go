package handlers

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lehigh-university-libraries/cataloger/internal/utils"
)

func (h *Handler) processImageFile(fileData []byte, filename, imageType, provider, model string) (*ImageProcessResult, error) {
	md5Hash := utils.CalculateDataMD5(fileData)
	ext := filepath.Ext(filename)
	imageFilename := md5Hash + ext
	imageFilePath := filepath.Join("uploads", imageFilename)

	if err := os.WriteFile(imageFilePath, fileData, 0644); err != nil {
		return nil, fmt.Errorf("failed to save image: %w", err)
	}

	slog.Info("Image saved", "filename", imageFilename, "type", imageType)

	width, height, err := getImageDimensions(imageFilePath)
	if err != nil {
		slog.Warn("Failed to get image dimensions", "error", err)
		width, height = 0, 0
	}

	return &ImageProcessResult{
		ImageFilename: imageFilename,
		ImageFilePath: imageFilePath,
		ImageType:     imageType,
		Width:         width,
		Height:        height,
		Provider:      provider,
		Model:         model,
	}, nil
}

func (h *Handler) createSessionFromURL(imageURL, imageType, provider, model string) (string, error) {
	imageData, err := h.downloadImageFromURL(imageURL)
	if err != nil {
		return "", err
	}

	// Extract filename from URL
	parts := strings.Split(imageURL, "/")
	filename := parts[len(parts)-1]
	if filename == "" {
		filename = "image.jpg"
	}

	result, err := h.processImageFile(imageData, filename, imageType, provider, model)
	if err != nil {
		return "", err
	}

	sessionID := fmt.Sprintf("%s_%d", filename, time.Now().Unix())
	session := h.createImageSession(sessionID, result)
	h.sessionStore.Set(sessionID, session)

	slog.Info("Session created from URL", "session_id", sessionID, "url", imageURL)
	return sessionID, nil
}

func (h *Handler) downloadImageFromURL(imageURL string) ([]byte, error) {
	resp, err := http.Get(imageURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download image: HTTP %d", resp.StatusCode)
	}

	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read image data: %w", err)
	}

	return imageData, nil
}

func getImageDimensions(imagePath string) (int, int, error) {
	file, err := os.Open(imagePath)
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()

	img, _, err := image.DecodeConfig(file)
	if err != nil {
		return 0, 0, err
	}

	return img.Width, img.Height, nil
}
