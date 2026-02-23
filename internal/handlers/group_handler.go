package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"friday/internal/models"
	"friday/internal/whatsapp"
)

type GroupHandler struct {
	groupRepo  *models.GroupRepository
	memberRepo *models.GroupMemberRepository
	waClient   *whatsapp.Client
}

// NewGroupHandler creates a new group handler with required dependencies.
func NewGroupHandler(groupRepo *models.GroupRepository, memberRepo *models.GroupMemberRepository, waClient *whatsapp.Client) *GroupHandler {
	return &GroupHandler{
		groupRepo:  groupRepo,
		memberRepo: memberRepo,
		waClient:   waClient,
	}
}

// Request/Response types

type CreateGroupRequest struct {
	Name string `json:"name"`
}

type UpdateGroupRequest struct {
	Name string `json:"name"`
}

type AddMembersRequest struct {
	JIDs []string `json:"jids"`
}

type GroupResponse struct {
	Success bool                  `json:"success"`
	Message string                `json:"message"`
	Group   *models.ContactGroup  `json:"group,omitempty"`
}

type GroupListResponse struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message"`
	Groups  []models.ContactGroup  `json:"groups"`
	Count   int                    `json:"count"`
}

// GroupDetailResponse includes the group with its members.
type GroupDetailResponse struct {
	Success bool                  `json:"success"`
	Message string                `json:"message"`
	Group   *models.ContactGroup  `json:"group,omitempty"`
	Members []GroupMemberInfo     `json:"members,omitempty"`
}

// GroupMemberInfo combines member data with contact info from WhatsApp.
type GroupMemberInfo struct {
	ID      int64  `json:"id"`
	JID     string `json:"jid"`
	Name    string `json:"name"`
	Phone   string `json:"phone"`
	AddedAt string `json:"added_at"`
}

type MembersResponse struct {
	Success bool              `json:"success"`
	Message string            `json:"message"`
	Members []GroupMemberInfo `json:"members,omitempty"`
	Count   int               `json:"count"`
}

// HandleGroups handles GET /api/groups (list) and POST /api/groups (create)
func (h *GroupHandler) HandleGroups(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listGroups(w, r)
	case http.MethodPost:
		h.createGroup(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleGroup handles single group operations: GET/PUT/DELETE /api/groups/{id}
// Also handles member operations: POST/GET /api/groups/{id}/members
func (h *GroupHandler) HandleGroup(w http.ResponseWriter, r *http.Request) {
	// Extract path after /api/groups/
	path := strings.TrimPrefix(r.URL.Path, "/api/groups/")

	// Check if this is a members operation: /api/groups/{id}/members
	if strings.Contains(path, "/members") {
		parts := strings.Split(path, "/members")
		id, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			jsonError(w, "Invalid group ID", http.StatusBadRequest)
			return
		}

		// Check if there's a JID after /members/
		memberJID := ""
		if len(parts) > 1 && parts[1] != "" {
			memberJID = strings.TrimPrefix(parts[1], "/")
		}

		switch r.Method {
		case http.MethodGet:
			h.getMembers(w, r, id)
		case http.MethodPost:
			h.addMembers(w, r, id)
		case http.MethodDelete:
			if memberJID != "" {
				h.removeMember(w, r, id, memberJID)
			} else {
				jsonError(w, "Member JID required for deletion", http.StatusBadRequest)
			}
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	// Parse group ID
	id, err := strconv.ParseInt(path, 10, 64)
	if err != nil {
		jsonError(w, "Invalid group ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getGroup(w, r, id)
	case http.MethodPut:
		h.updateGroup(w, r, id)
	case http.MethodDelete:
		h.deleteGroup(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *GroupHandler) listGroups(w http.ResponseWriter, r *http.Request) {
	groups, err := h.groupRepo.GetAll()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(GroupListResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to retrieve groups: %v", err),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(GroupListResponse{
		Success: true,
		Message: "Groups retrieved successfully",
		Groups:  groups,
		Count:   len(groups),
	})
}

func (h *GroupHandler) createGroup(w http.ResponseWriter, r *http.Request) {
	var req CreateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(GroupResponse{
			Success: false,
			Message: fmt.Sprintf("Invalid JSON: %v", err),
		})
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		jsonError(w, "Group name is required", http.StatusBadRequest)
		return
	}

	// Check if name already exists
	existing, err := h.groupRepo.GetByName(name)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(GroupResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to check group name: %v", err),
		})
		return
	}
	if existing != nil {
		jsonError(w, "A group with this name already exists", http.StatusConflict)
		return
	}

	group := &models.ContactGroup{
		Name: name,
	}

	if err := h.groupRepo.Create(group); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(GroupResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to create group: %v", err),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(GroupResponse{
		Success: true,
		Message: "Group created successfully",
		Group:   group,
	})
}

func (h *GroupHandler) getGroup(w http.ResponseWriter, r *http.Request, id int64) {
	group, err := h.groupRepo.GetByID(id)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(GroupDetailResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to retrieve group: %v", err),
		})
		return
	}

	if group == nil {
		jsonError(w, "Group not found", http.StatusNotFound)
		return
	}

	// Get members with contact info
	members, err := h.getMembersWithInfo(id)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(GroupDetailResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to retrieve members: %v", err),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(GroupDetailResponse{
		Success: true,
		Message: "Group retrieved successfully",
		Group:   group,
		Members: members,
	})
}

func (h *GroupHandler) updateGroup(w http.ResponseWriter, r *http.Request, id int64) {
	var req UpdateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(GroupResponse{
			Success: false,
			Message: fmt.Sprintf("Invalid JSON: %v", err),
		})
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		jsonError(w, "Group name is required", http.StatusBadRequest)
		return
	}

	// Check if another group has this name
	existing, err := h.groupRepo.GetByName(name)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(GroupResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to check group name: %v", err),
		})
		return
	}
	if existing != nil && existing.ID != id {
		jsonError(w, "A group with this name already exists", http.StatusConflict)
		return
	}

	group := &models.ContactGroup{
		ID:   id,
		Name: name,
	}

	found, err := h.groupRepo.Update(group)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(GroupResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to update group: %v", err),
		})
		return
	}

	if !found {
		jsonError(w, "Group not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(GroupResponse{
		Success: true,
		Message: "Group updated successfully",
		Group:   group,
	})
}

func (h *GroupHandler) deleteGroup(w http.ResponseWriter, r *http.Request, id int64) {
	found, err := h.groupRepo.Delete(id)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(GroupResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to delete group: %v", err),
		})
		return
	}

	if !found {
		jsonError(w, "Group not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(GroupResponse{
		Success: true,
		Message: "Group deleted successfully",
	})
}

// Member operations

func (h *GroupHandler) getMembers(w http.ResponseWriter, r *http.Request, groupID int64) {
	// Verify group exists
	group, err := h.groupRepo.GetByID(groupID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(MembersResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to check group: %v", err),
		})
		return
	}
	if group == nil {
		jsonError(w, "Group not found", http.StatusNotFound)
		return
	}

	members, err := h.getMembersWithInfo(groupID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(MembersResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to retrieve members: %v", err),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(MembersResponse{
		Success: true,
		Message: "Members retrieved successfully",
		Members: members,
		Count:   len(members),
	})
}

func (h *GroupHandler) addMembers(w http.ResponseWriter, r *http.Request, groupID int64) {
	// Verify group exists
	group, err := h.groupRepo.GetByID(groupID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(MembersResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to check group: %v", err),
		})
		return
	}
	if group == nil {
		jsonError(w, "Group not found", http.StatusNotFound)
		return
	}

	var req AddMembersRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(MembersResponse{
			Success: false,
			Message: fmt.Sprintf("Invalid JSON: %v", err),
		})
		return
	}

	if len(req.JIDs) == 0 {
		jsonError(w, "At least one JID is required", http.StatusBadRequest)
		return
	}

	// Add members
	if err := h.memberRepo.AddMultiple(groupID, req.JIDs); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(MembersResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to add members: %v", err),
		})
		return
	}

	// Return updated member list
	members, _ := h.getMembersWithInfo(groupID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(MembersResponse{
		Success: true,
		Message: fmt.Sprintf("Added %d members to group", len(req.JIDs)),
		Members: members,
		Count:   len(members),
	})
}

func (h *GroupHandler) removeMember(w http.ResponseWriter, r *http.Request, groupID int64, jid string) {
	// Verify group exists
	group, err := h.groupRepo.GetByID(groupID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(MembersResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to check group: %v", err),
		})
		return
	}
	if group == nil {
		jsonError(w, "Group not found", http.StatusNotFound)
		return
	}

	found, err := h.memberRepo.Remove(groupID, jid)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(MembersResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to remove member: %v", err),
		})
		return
	}

	if !found {
		jsonError(w, "Member not found in group", http.StatusNotFound)
		return
	}

	// Return updated member list
	members, _ := h.getMembersWithInfo(groupID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(MembersResponse{
		Success: true,
		Message: "Member removed successfully",
		Members: members,
		Count:   len(members),
	})
}

// getMembersWithInfo enriches member data with contact info from WhatsApp.
func (h *GroupHandler) getMembersWithInfo(groupID int64) ([]GroupMemberInfo, error) {
	members, err := h.memberRepo.GetByGroup(groupID)
	if err != nil {
		return nil, err
	}

	result := make([]GroupMemberInfo, len(members))
	for i, m := range members {
		info := GroupMemberInfo{
			ID:      m.ID,
			JID:     m.JID,
			Phone:   extractPhone(m.JID),
			Name:    extractPhone(m.JID), // Default to phone
			AddedAt: m.AddedAt.Format("2006-01-02 15:04"),
		}

		// Try to get contact info from WhatsApp
		if h.waClient.IsConnected() {
			contact, _ := h.waClient.FindContactByJID(m.JID)
			if contact != nil {
				info.Name = contact.Name
				info.Phone = contact.Phone
			}
		}

		result[i] = info
	}

	return result, nil
}

// extractPhone extracts the phone number from a JID.
// JID format: "1234567890@s.whatsapp.net"
func extractPhone(jid string) string {
	parts := strings.Split(jid, "@")
	if len(parts) > 0 {
		return parts[0]
	}
	return jid
}
