package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"friday/internal/models"
	"friday/internal/template"
	"friday/internal/whatsapp"
)

type DraftHandler struct {
	repo       *models.DraftRepository
	attrRepo   *models.AttributeRepository
	waClient   *whatsapp.Client
}

func NewDraftHandler(repo *models.DraftRepository, attrRepo *models.AttributeRepository, waClient *whatsapp.Client) *DraftHandler {
	return &DraftHandler{
		repo:       repo,
		attrRepo:   attrRepo,
		waClient:   waClient,
	}
}

// Request/Response types

type CreateDraftRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

type UpdateDraftRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

type DraftResponse struct {
	Success bool                 `json:"success"`
	Message string               `json:"message"`
	Draft   *models.MessageDraft `json:"draft,omitempty"`
}

type DraftListResponse struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message"`
	Drafts  []models.MessageDraft  `json:"drafts"`
	Count   int                    `json:"count"`
}

type PreviewRequest struct {
	JID string `json:"jid"` // Contact JID to use for placeholder values
}

type PreviewResponse struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message"`
	Preview *template.PreviewResult `json:"preview,omitempty"`
}

type SendWithDraftRequest struct {
	JID string `json:"jid"` // Contact JID to send to
}

type SendWithDraftResponse struct {
	Success     bool   `json:"success"`
	Message     string `json:"message"`
	SentMessage string `json:"sent_message,omitempty"` // The actual message that was sent
}

// HandleDrafts handles GET /api/drafts (list) and POST /api/drafts (create)
func (h *DraftHandler) HandleDrafts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listDrafts(w, r)
	case http.MethodPost:
		h.createDraft(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleDraft handles single draft operations: GET/PUT/DELETE /api/drafts/{id}
func (h *DraftHandler) HandleDraft(w http.ResponseWriter, r *http.Request) {
	// Extract ID from path: /api/drafts/123 -> "123"
	path := strings.TrimPrefix(r.URL.Path, "/api/drafts/")

	// Check if this is a preview or send request
	if strings.Contains(path, "/preview") {
		id, err := strconv.ParseInt(strings.TrimSuffix(path, "/preview"), 10, 64)
		if err != nil {
			jsonError(w, "Invalid draft ID", http.StatusBadRequest)
			return
		}
		h.previewDraft(w, r, id)
		return
	}
	if strings.Contains(path, "/send") {
		id, err := strconv.ParseInt(strings.TrimSuffix(path, "/send"), 10, 64)
		if err != nil {
			jsonError(w, "Invalid draft ID", http.StatusBadRequest)
			return
		}
		h.sendWithDraft(w, r, id)
		return
	}

	id, err := strconv.ParseInt(path, 10, 64)
	if err != nil {
		jsonError(w, "Invalid draft ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getDraft(w, r, id)
	case http.MethodPut:
		h.updateDraft(w, r, id)
	case http.MethodDelete:
		h.deleteDraft(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *DraftHandler) listDrafts(w http.ResponseWriter, r *http.Request) {
	drafts, err := h.repo.GetAll()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(DraftListResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to retrieve drafts: %v", err),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(DraftListResponse{
		Success: true,
		Message: "Drafts retrieved successfully",
		Drafts:  drafts,
		Count:   len(drafts),
	})
}

func (h *DraftHandler) createDraft(w http.ResponseWriter, r *http.Request) {
	var req CreateDraftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(DraftResponse{
			Success: false,
			Message: fmt.Sprintf("Invalid JSON: %v", err),
		})
		return
	}

	// Validate required fields
	if strings.TrimSpace(req.Title) == "" {
		jsonError(w, "Title is required", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Content) == "" {
		jsonError(w, "Content is required", http.StatusBadRequest)
		return
	}

	draft := &models.MessageDraft{
		Title:   strings.TrimSpace(req.Title),
		Content: req.Content,
	}

	if err := h.repo.Create(draft); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(DraftResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to create draft: %v", err),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(DraftResponse{
		Success: true,
		Message: "Draft created successfully",
		Draft:   draft,
	})
}

func (h *DraftHandler) getDraft(w http.ResponseWriter, r *http.Request, id int64) {
	draft, err := h.repo.GetByID(id)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(DraftResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to retrieve draft: %v", err),
		})
		return
	}

	if draft == nil {
		jsonError(w, "Draft not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(DraftResponse{
		Success: true,
		Message: "Draft retrieved successfully",
		Draft:   draft,
	})
}

func (h *DraftHandler) updateDraft(w http.ResponseWriter, r *http.Request, id int64) {
	var req UpdateDraftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(DraftResponse{
			Success: false,
			Message: fmt.Sprintf("Invalid JSON: %v", err),
		})
		return
	}

	// Validate required fields
	if strings.TrimSpace(req.Title) == "" {
		jsonError(w, "Title is required", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Content) == "" {
		jsonError(w, "Content is required", http.StatusBadRequest)
		return
	}

	draft := &models.MessageDraft{
		ID:      id,
		Title:   strings.TrimSpace(req.Title),
		Content: req.Content,
	}

	found, err := h.repo.Update(draft)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(DraftResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to update draft: %v", err),
		})
		return
	}

	if !found {
		jsonError(w, "Draft not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(DraftResponse{
		Success: true,
		Message: "Draft updated successfully",
		Draft:   draft,
	})
}

func (h *DraftHandler) deleteDraft(w http.ResponseWriter, r *http.Request, id int64) {
	found, err := h.repo.Delete(id)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(DraftResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to delete draft: %v", err),
		})
		return
	}

	if !found {
		jsonError(w, "Draft not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(DraftResponse{
		Success: true,
		Message: "Draft deleted successfully",
	})
}

func (h *DraftHandler) previewDraft(w http.ResponseWriter, r *http.Request, id int64) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req PreviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(PreviewResponse{
			Success: false,
			Message: fmt.Sprintf("Invalid JSON: %v", err),
		})
		return
	}

	if req.JID == "" {
		jsonError(w, "Contact JID is required", http.StatusBadRequest)
		return
	}

	// Get the draft
	draft, err := h.repo.GetByID(id)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(PreviewResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to retrieve draft: %v", err),
		})
		return
	}
	if draft == nil {
		jsonError(w, "Draft not found", http.StatusNotFound)
		return
	}

	// Get placeholder values
	values, err := h.getPlaceholderValues(req.JID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(PreviewResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to get placeholder values: %v", err),
		})
		return
	}

	// Generate preview
	preview := template.Preview(draft.Content, values)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(PreviewResponse{
		Success: true,
		Message: "Preview generated successfully",
		Preview: &preview,
	})
}

// sendWithDraft sends a message using a draft template with placeholders filled.
func (h *DraftHandler) sendWithDraft(w http.ResponseWriter, r *http.Request, id int64) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !h.waClient.IsConnected() {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(SendWithDraftResponse{
			Success: false,
			Message: "WhatsApp client not connected",
		})
		return
	}

	var req SendWithDraftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(SendWithDraftResponse{
			Success: false,
			Message: fmt.Sprintf("Invalid JSON: %v", err),
		})
		return
	}

	if req.JID == "" {
		jsonError(w, "Contact JID is required", http.StatusBadRequest)
		return
	}

	// Get the draft
	draft, err := h.repo.GetByID(id)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(SendWithDraftResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to retrieve draft: %v", err),
		})
		return
	}
	if draft == nil {
		jsonError(w, "Draft not found", http.StatusNotFound)
		return
	}

	// Get placeholder values
	values, err := h.getPlaceholderValues(req.JID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(SendWithDraftResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to get placeholder values: %v", err),
		})
		return
	}

	// Fill placeholders
	filledMessage, missing := template.FillPlaceholders(draft.Content, values)

	// Warn if there are missing placeholders but still send
	warningMsg := ""
	if len(missing) > 0 {
		warningMsg = fmt.Sprintf(" (warning: unfilled placeholders: %v)", missing)
	}

	// Send the message
	if err := h.waClient.SendMessage(r.Context(), req.JID, filledMessage); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(SendWithDraftResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to send message: %v", err),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(SendWithDraftResponse{
		Success:     true,
		Message:     "Message sent successfully" + warningMsg,
		SentMessage: filledMessage,
	})
}

// getPlaceholderValues retrieves all placeholder values for a contact.
// This merges built-in contact fields with custom attributes.
func (h *DraftHandler) getPlaceholderValues(jid string) (map[string]string, error) {
	// Get built-in placeholders from contact
	var builtIn map[string]string
	if h.waClient.IsConnected() {
		contact, _ := h.waClient.FindContactByJID(jid)
		if contact != nil {
			builtIn = template.GetBuiltInPlaceholders(contact)
		}
	}
	if builtIn == nil {
		builtIn = map[string]string{}
	}

	// Get custom attributes
	custom, err := h.attrRepo.GetAllForContactAsMap(jid)
	if err != nil {
		return nil, err
	}

	// Merge: custom attributes override built-in values
	return template.MergePlaceholders(builtIn, custom), nil
}

// Helper function for JSON error responses
func jsonError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": false,
		"message": message,
	})
}
