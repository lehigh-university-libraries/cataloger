package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/lehigh-university-libraries/cataloger/internal/models"
)

func (h *Handler) HandleSessions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		sessions := h.sessionStore.GetAll()
		sessionList := make([]*models.CatalogSession, 0, len(sessions))
		for _, session := range sessions {
			sessionList = append(sessionList, session)
		}
		h.writeJSON(w, sessionList)
	default:
		h.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) HandleSessionDetail(w http.ResponseWriter, r *http.Request) {
	sessionID := strings.TrimPrefix(r.URL.Path, "/api/sessions/")

	session, ok := h.getSessionOrError(w, sessionID)
	if !ok {
		return
	}

	switch r.Method {
	case "GET":
		h.writeJSON(w, session)
	case "PUT":
		var updatedSession models.CatalogSession
		if err := json.NewDecoder(r.Body).Decode(&updatedSession); err != nil {
			h.writeError(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		h.sessionStore.Set(sessionID, &updatedSession)
		h.writeJSON(w, updatedSession)
	default:
		h.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
