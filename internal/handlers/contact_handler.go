package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"friday/internal/whatsapp"
)

type ContactHandler struct {
	client *whatsapp.Client
}

func NewContactHandler(client *whatsapp.Client) *ContactHandler {
	return &ContactHandler{client: client}
}

type ContactListResponse struct {
	Success  bool               `json:"success"`
	Message  string             `json:"message"`
	Contacts []whatsapp.Contact `json:"contacts,omitempty"`
	Count    int                `json:"count"`
}

type ContactSearchResponse struct {
	Success  bool               `json:"success"`
	Message  string             `json:"message"`
	Query    string             `json:"query"`
	Contacts []whatsapp.Contact `json:"contacts,omitempty"`
	Count    int                `json:"count"`
}

type PhoneValidationRequest struct {
	Phones []string `json:"phones"`
}

type PhoneValidationResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Results map[string]bool `json:"results,omitempty"`
}

// HandleGetContacts returns all WhatsApp contacts
func (h *ContactHandler) HandleGetContacts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !h.client.IsConnected() {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ContactListResponse{
			Success: false,
			Message: "WhatsApp client not connected",
			Count:   0,
		})
		return
	}

	contacts, err := h.client.GetContacts()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ContactListResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to retrieve contacts: %v", err),
			Count:   0,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ContactListResponse{
		Success:  true,
		Message:  "Contacts retrieved successfully",
		Contacts: contacts,
		Count:    len(contacts),
	})
}

// HandleSearchContacts searches contacts by name or phone
func (h *ContactHandler) HandleSearchContacts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !h.client.IsConnected() {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ContactSearchResponse{
			Success: false,
			Message: "WhatsApp client not connected",
			Count:   0,
		})
		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("q"))

	contacts, err := h.client.SearchContacts(query)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ContactSearchResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to search contacts: %v", err),
			Query:   query,
			Count:   0,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ContactSearchResponse{
		Success:  true,
		Message:  "Contact search completed successfully",
		Query:    query,
		Contacts: contacts,
		Count:    len(contacts),
	})
}

// HandleValidatePhones checks if phone numbers are registered on WhatsApp
func (h *ContactHandler) HandleValidatePhones(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !h.client.IsConnected() {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(PhoneValidationResponse{
			Success: false,
			Message: "WhatsApp client not connected",
		})
		return
	}

	var req PhoneValidationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(PhoneValidationResponse{
			Success: false,
			Message: fmt.Sprintf("Invalid JSON in request body: %v", err),
		})
		return
	}

	if len(req.Phones) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(PhoneValidationResponse{
			Success: false,
			Message: "At least one phone number is required",
		})
		return
	}

	results, err := h.client.ValidatePhones(req.Phones)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(PhoneValidationResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to validate phone numbers: %v", err),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(PhoneValidationResponse{
		Success: true,
		Message: "Phone validation completed successfully",
		Results: results,
	})
}
