package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"friday/internal/batch"
	"friday/internal/models"
	"friday/internal/whatsapp"
)

type BatchHandler struct {
	batchRepo  *models.BatchRunRepository
	msgRepo    *models.BatchMessageRepository
	groupRepo  *models.GroupRepository
	memberRepo *models.GroupMemberRepository
	draftRepo  *models.DraftRepository
	worker     *batch.Worker
	waClient   *whatsapp.Client
}

// NewBatchHandler creates a new batch handler with required dependencies.
func NewBatchHandler(
	batchRepo *models.BatchRunRepository,
	msgRepo *models.BatchMessageRepository,
	groupRepo *models.GroupRepository,
	memberRepo *models.GroupMemberRepository,
	draftRepo *models.DraftRepository,
	worker *batch.Worker,
	waClient *whatsapp.Client,
) *BatchHandler {
	return &BatchHandler{
		batchRepo:  batchRepo,
		msgRepo:    msgRepo,
		groupRepo:  groupRepo,
		memberRepo: memberRepo,
		draftRepo:  draftRepo,
		worker:     worker,
		waClient:   waClient,
	}
}

// Request/Response types

type CreateBatchRequest struct {
	DraftID int64 `json:"draft_id"`
	GroupID int64 `json:"group_id"`
}

type BatchResponse struct {
	Success bool              `json:"success"`
	Message string            `json:"message"`
	Batch   *models.BatchRun  `json:"batch,omitempty"`
}

type BatchListResponse struct {
	Success bool               `json:"success"`
	Message string             `json:"message"`
	Batches []models.BatchRun  `json:"batches"`
	Count   int                `json:"count"`
}

type BatchDetailResponse struct {
	Success  bool                   `json:"success"`
	Message  string                 `json:"message"`
	Batch    *models.BatchRun       `json:"batch,omitempty"`
	Messages []models.BatchMessage  `json:"messages,omitempty"`
}

type ActiveBatchResponse struct {
	Success   bool                  `json:"success"`
	HasActive bool                  `json:"has_active"`
	Batch     *models.BatchRun      `json:"batch,omitempty"`
	Progress  *batch.ProgressEvent  `json:"progress,omitempty"`
}

// HandleBatches handles GET /api/batch-runs (list) and POST /api/batch-runs (create)
func (h *BatchHandler) HandleBatches(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listBatches(w, r)
	case http.MethodPost:
		h.createBatch(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleBatch handles single batch operations: GET/DELETE /api/batch-runs/{id}
// Also handles: POST /api/batch-runs/{id}/cancel and GET /api/batch-runs/{id}/stream
func (h *BatchHandler) HandleBatch(w http.ResponseWriter, r *http.Request) {
	// Extract path after /api/batch-runs/
	path := strings.TrimPrefix(r.URL.Path, "/api/batch-runs/")

	// Check for special endpoints
	if path == "active" {
		h.getActiveBatch(w, r)
		return
	}

	// Check for /cancel or /stream suffix
	if strings.Contains(path, "/cancel") {
		id, err := strconv.ParseInt(strings.TrimSuffix(path, "/cancel"), 10, 64)
		if err != nil {
			jsonError(w, "Invalid batch ID", http.StatusBadRequest)
			return
		}
		h.cancelBatch(w, r, id)
		return
	}

	if strings.Contains(path, "/stream") {
		id, err := strconv.ParseInt(strings.TrimSuffix(path, "/stream"), 10, 64)
		if err != nil {
			jsonError(w, "Invalid batch ID", http.StatusBadRequest)
			return
		}
		h.streamBatch(w, r, id)
		return
	}

	if strings.Contains(path, "/messages") {
		id, err := strconv.ParseInt(strings.TrimSuffix(path, "/messages"), 10, 64)
		if err != nil {
			jsonError(w, "Invalid batch ID", http.StatusBadRequest)
			return
		}
		h.getBatchMessages(w, r, id)
		return
	}

	// Parse batch ID
	id, err := strconv.ParseInt(path, 10, 64)
	if err != nil {
		jsonError(w, "Invalid batch ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getBatch(w, r, id)
	case http.MethodDelete:
		h.deleteBatch(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *BatchHandler) listBatches(w http.ResponseWriter, r *http.Request) {
	batches, err := h.batchRepo.GetAll()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(BatchListResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to retrieve batches: %v", err),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(BatchListResponse{
		Success: true,
		Message: "Batches retrieved successfully",
		Batches: batches,
		Count:   len(batches),
	})
}

func (h *BatchHandler) createBatch(w http.ResponseWriter, r *http.Request) {
	var req CreateBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(BatchResponse{
			Success: false,
			Message: fmt.Sprintf("Invalid JSON: %v", err),
		})
		return
	}

	// Validate draft exists
	draft, err := h.draftRepo.GetByID(req.DraftID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(BatchResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to check draft: %v", err),
		})
		return
	}
	if draft == nil {
		jsonError(w, "Draft not found", http.StatusNotFound)
		return
	}

	// Validate group exists
	group, err := h.groupRepo.GetByID(req.GroupID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(BatchResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to check group: %v", err),
		})
		return
	}
	if group == nil {
		jsonError(w, "Group not found", http.StatusNotFound)
		return
	}

	// Check group has members
	if group.MemberCount == 0 {
		jsonError(w, "Group has no members", http.StatusBadRequest)
		return
	}

	// Get group members
	members, err := h.memberRepo.GetByGroup(req.GroupID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(BatchResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to get group members: %v", err),
		})
		return
	}

	// Create batch run
	batchRun := &models.BatchRun{
		DraftID:    req.DraftID,
		GroupID:    req.GroupID,
		GroupName:  group.Name,
		DraftTitle: draft.Title,
		Status:     models.BatchStatusQueued,
		TotalCount: len(members),
	}

	if err := h.batchRepo.Create(batchRun); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(BatchResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to create batch: %v", err),
		})
		return
	}

	// Create batch messages for each member
	messages := make([]models.BatchMessage, len(members))
	for i, member := range members {
		// Try to get contact name
		var contactName *string
		if h.waClient.IsConnected() {
			contact, _ := h.waClient.FindContactByJID(member.JID)
			if contact != nil {
				contactName = &contact.Name
			}
		}

		messages[i] = models.BatchMessage{
			BatchRunID:      batchRun.ID,
			JID:             member.JID,
			ContactName:     contactName,
			Status:          models.MessageStatusPending,
			TemplateContent: draft.Content,
		}
	}

	if err := h.msgRepo.CreateMultiple(messages); err != nil {
		// Clean up the batch run
		h.batchRepo.Delete(batchRun.ID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(BatchResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to create batch messages: %v", err),
		})
		return
	}

	// Check if there's already an active batch
	activeBatchID := h.worker.GetActiveBatchID()
	message := "Batch queued successfully"
	if activeBatchID == 0 {
		message = "Batch started"
	} else {
		message = fmt.Sprintf("Batch queued (waiting for batch #%d to complete)", activeBatchID)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(BatchResponse{
		Success: true,
		Message: message,
		Batch:   batchRun,
	})
}

func (h *BatchHandler) getBatch(w http.ResponseWriter, r *http.Request, id int64) {
	batchRun, err := h.batchRepo.GetByID(id)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(BatchDetailResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to retrieve batch: %v", err),
		})
		return
	}

	if batchRun == nil {
		jsonError(w, "Batch not found", http.StatusNotFound)
		return
	}

	// Get messages
	messages, err := h.msgRepo.GetByBatchRun(id)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(BatchDetailResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to retrieve messages: %v", err),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(BatchDetailResponse{
		Success:  true,
		Message:  "Batch retrieved successfully",
		Batch:    batchRun,
		Messages: messages,
	})
}

func (h *BatchHandler) getBatchMessages(w http.ResponseWriter, r *http.Request, id int64) {
	messages, err := h.msgRepo.GetByBatchRun(id)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":  false,
			"message":  fmt.Sprintf("Failed to retrieve messages: %v", err),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"messages": messages,
		"count":    len(messages),
	})
}

func (h *BatchHandler) cancelBatch(w http.ResponseWriter, r *http.Request, id int64) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check batch exists
	batchRun, err := h.batchRepo.GetByID(id)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(BatchResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to check batch: %v", err),
		})
		return
	}

	if batchRun == nil {
		jsonError(w, "Batch not found", http.StatusNotFound)
		return
	}

	// Can only cancel queued or running batches
	if batchRun.Status != models.BatchStatusQueued && batchRun.Status != models.BatchStatusRunning {
		jsonError(w, fmt.Sprintf("Cannot cancel batch with status: %s", batchRun.Status), http.StatusBadRequest)
		return
	}

	if err := h.worker.CancelBatch(id); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(BatchResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to cancel batch: %v", err),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(BatchResponse{
		Success: true,
		Message: "Batch cancelled successfully",
	})
}

func (h *BatchHandler) deleteBatch(w http.ResponseWriter, r *http.Request, id int64) {
	found, err := h.batchRepo.Delete(id)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(BatchResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to delete batch: %v", err),
		})
		return
	}

	if !found {
		jsonError(w, "Batch not found or currently running", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(BatchResponse{
		Success: true,
		Message: "Batch deleted successfully",
	})
}

func (h *BatchHandler) getActiveBatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	activeBatch, err := h.batchRepo.GetActive()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ActiveBatchResponse{
			Success: false,
		})
		return
	}

	if activeBatch == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ActiveBatchResponse{
			Success:   true,
			HasActive: false,
		})
		return
	}

	progress, _ := h.worker.GetProgress(activeBatch.ID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ActiveBatchResponse{
		Success:   true,
		HasActive: true,
		Batch:     activeBatch,
		Progress:  progress,
	})
}

func (h *BatchHandler) streamBatch(w http.ResponseWriter, r *http.Request, id int64) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check batch exists
	batchRun, err := h.batchRepo.GetByID(id)
	if err != nil || batchRun == nil {
		http.Error(w, "Batch not found", http.StatusNotFound)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	// Subscribe to batch events
	eventCh := h.worker.Subscribe(id)
	defer h.worker.Unsubscribe(id, eventCh)

	// Send initial progress
	progress, err := h.worker.GetProgress(id)
	if err == nil && progress != nil {
		data, _ := json.Marshal(progress)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()

		// If already completed or cancelled, close the stream
		if progress.Status == "completed" || progress.Status == "cancelled" || progress.Status == "failed" {
			return
		}
	}

	// Stream events
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			// Client disconnected
			return

		case event, ok := <-eventCh:
			if !ok {
				return
			}
			data, _ := json.Marshal(event)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()

			// Close stream on terminal events
			if event.Type == "completed" || event.Type == "cancelled" {
				return
			}

		case <-ticker.C:
			// Send heartbeat/progress update every second
			progress, err := h.worker.GetProgress(id)
			if err != nil {
				continue
			}
			data, _ := json.Marshal(progress)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()

			// Close stream on terminal status
			if progress.Status == "completed" || progress.Status == "cancelled" || progress.Status == "failed" {
				return
			}
		}
	}
}
