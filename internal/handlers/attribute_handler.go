package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"friday/internal/models"
)

// AttributeHandler handles HTTP requests for contact attribute operations.
type AttributeHandler struct {
	repo *models.AttributeRepository
}

// NewAttributeHandler creates a new attribute handler.
func NewAttributeHandler(repo *models.AttributeRepository) *AttributeHandler {
	return &AttributeHandler{repo: repo}
}

// Request/Response types

type SetAttributeRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type AttributeResponse struct {
	Success    bool                       `json:"success"`
	Message    string                     `json:"message"`
	Attribute  *models.ContactAttribute   `json:"attribute,omitempty"`
	Attributes []models.ContactAttribute  `json:"attributes,omitempty"`
}

type AttributeKeysResponse struct {
	Success bool              `json:"success"`
	Message string            `json:"message"`
	Keys    []string          `json:"keys"`
	Counts  map[string]int    `json:"counts,omitempty"` // Optional: count of contacts per key
}

// HandleContactAttributes handles /api/contacts/{jid}/attributes[/{key}] routes.
func (h *AttributeHandler) HandleContactAttributes(w http.ResponseWriter, r *http.Request) {
	// Parse path: /api/contacts/{jid}/attributes[/{key}]
	path := strings.TrimPrefix(r.URL.Path, "/api/contacts/")

	// Find where /attributes starts
	attrIdx := strings.Index(path, "/attributes")
	if attrIdx == -1 {
		jsonError(w, "Invalid path", http.StatusBadRequest)
		return
	}

	// Extract JID (URL-decode it since JIDs contain special characters)
	jidEncoded := path[:attrIdx]
	jid, err := url.PathUnescape(jidEncoded)
	if err != nil {
		jsonError(w, "Invalid JID encoding", http.StatusBadRequest)
		return
	}

	// Extract optional key after /attributes/
	remainder := path[attrIdx+len("/attributes"):]
	key := ""
	if len(remainder) > 1 && remainder[0] == '/' {
		key = remainder[1:]
		// URL-decode the key as well
		key, _ = url.PathUnescape(key)
	}

	// Route based on method and whether key is present
	switch r.Method {
	case http.MethodGet:
		h.getAttributes(w, r, jid)
	case http.MethodPost:
		h.setAttribute(w, r, jid)
	case http.MethodDelete:
		if key == "" {
			jsonError(w, "Attribute key is required for DELETE", http.StatusBadRequest)
			return
		}
		h.deleteAttribute(w, r, jid, key)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleAttributeKeys handles GET /api/attributes/keys
func (h *AttributeHandler) HandleAttributeKeys(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	keys, err := h.repo.GetAllUniqueKeys()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(AttributeKeysResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to get attribute keys: %v", err),
		})
		return
	}

	// Also get counts for each key
	counts, _ := h.repo.CountByKey() // Ignore error, counts are optional

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AttributeKeysResponse{
		Success: true,
		Message: "Attribute keys retrieved successfully",
		Keys:    keys,
		Counts:  counts,
	})
}

func (h *AttributeHandler) getAttributes(w http.ResponseWriter, r *http.Request, jid string) {
	attrs, err := h.repo.GetAllForContact(jid)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(AttributeResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to get attributes: %v", err),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AttributeResponse{
		Success:    true,
		Message:    fmt.Sprintf("Found %d attributes", len(attrs)),
		Attributes: attrs,
	})
}

func (h *AttributeHandler) setAttribute(w http.ResponseWriter, r *http.Request, jid string) {
	var req SetAttributeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(AttributeResponse{
			Success: false,
			Message: fmt.Sprintf("Invalid JSON: %v", err),
		})
		return
	}

	// Validate
	key := strings.TrimSpace(req.Key)
	value := strings.TrimSpace(req.Value)

	if key == "" {
		jsonError(w, "Attribute key is required", http.StatusBadRequest)
		return
	}

	// Validate key format: only alphanumeric and underscores
	for _, c := range key {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
			jsonError(w, "Attribute key must contain only letters, numbers, and underscores", http.StatusBadRequest)
			return
		}
	}

	if value == "" {
		jsonError(w, "Attribute value is required", http.StatusBadRequest)
		return
	}

	if err := h.repo.Set(jid, key, value); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(AttributeResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to set attribute: %v", err),
		})
		return
	}

	// Fetch the saved attribute to return it
	attr, _ := h.repo.Get(jid, key)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AttributeResponse{
		Success:   true,
		Message:   "Attribute saved successfully",
		Attribute: attr,
	})
}

func (h *AttributeHandler) deleteAttribute(w http.ResponseWriter, r *http.Request, jid, key string) {
	found, err := h.repo.Delete(jid, key)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(AttributeResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to delete attribute: %v", err),
		})
		return
	}

	if !found {
		jsonError(w, "Attribute not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AttributeResponse{
		Success: true,
		Message: "Attribute deleted successfully",
	})
}
