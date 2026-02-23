package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"friday/internal/whatsapp"
)

type WhatsAppHandler struct {
	client *whatsapp.Client
}

func NewWhatsAppHandler(client *whatsapp.Client) *WhatsAppHandler {
	return &WhatsAppHandler{client: client}
}

type StatusResponse struct {
	Connected  bool   `json:"connected"`
	HasSession bool   `json:"has_session"`  // true if device was previously linked
	Connecting bool   `json:"connecting"`   // true if websocket connected but not authenticated yet
	Message    string `json:"message"`
}

type SendMessageRequest struct {
	Phone     string `json:"phone,omitempty"` // deprecated, use recipient
	Recipient string `json:"recipient"`
	Message   string `json:"message"`
}

type SendMessageResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	ID      string `json:"id,omitempty"`
}

func (h *WhatsAppHandler) HandleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	connected := h.client.IsConnected()
	hasSession := h.client.HasSession()
	connecting := h.client.IsConnecting()

	response := StatusResponse{
		Connected:  connected,
		HasSession: hasSession,
		Connecting: connecting,
		Message:    "WhatsApp client connected",
	}

	if !connected {
		if connecting {
			response.Message = "WhatsApp session restoring..."
		} else if hasSession {
			response.Message = "Session exists, attempting to connect..."
		} else {
			response.Message = "No session - QR code scan required"
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (h *WhatsAppHandler) HandleConnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if h.client.IsConnected() {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Already connected to WhatsApp",
		})
		return
	}

	err := h.client.Connect()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Failed to connect: %v", err),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Connection initiated - check logs for QR code if needed",
	})
}

// HandleDisconnect clears the WhatsApp session and disconnects the client.
// This forces a fresh QR code scan on the next connection attempt.
func (h *WhatsAppHandler) HandleDisconnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// ClearSession disconnects and removes the stored session
	if err := h.client.ClearSession(); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Failed to disconnect: %v", err),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "WhatsApp disconnected and session cleared",
	})
}

func (h *WhatsAppHandler) HandleSendMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !h.client.IsConnected() {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(SendMessageResponse{
			Success: false,
			Message: "WhatsApp client not connected",
		})
		return
	}

	var req SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(SendMessageResponse{
			Success: false,
			Message: fmt.Sprintf("Invalid JSON in request body: %v", err),
		})
		return
	}

	// Support both 'phone' (deprecated) and 'recipient' fields
	recipient := req.Recipient
	if recipient == "" && req.Phone != "" {
		recipient = req.Phone
	}

	if recipient == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(SendMessageResponse{
			Success: false,
			Message: "Recipient (phone number or contact name) is required",
		})
		return
	}

	if req.Message == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(SendMessageResponse{
			Success: false,
			Message: "Message content is required",
		})
		return
	}

	jid, err := h.client.ResolveRecipient(recipient)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(SendMessageResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to resolve recipient '%s': %v", recipient, err),
		})
		return
	}

	err = h.client.SendMessage(r.Context(), jid, req.Message)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(SendMessageResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to send message: %v", err),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(SendMessageResponse{
		Success: true,
		Message: "Message sent successfully",
	})
}
