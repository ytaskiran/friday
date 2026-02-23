package models

import (
	"database/sql"
	"fmt"
	"time"

	"friday/internal/database"
)

type GroupMember struct {
	ID       int64     `json:"id"`
	GroupID  int64     `json:"group_id"`
	JID      string    `json:"jid"`
	AddedAt  time.Time `json:"added_at"`
	Name     string    `json:"name,omitempty"`  // Not from DB - populated by handler from WhatsApp
	Phone    string    `json:"phone,omitempty"` // Not from DB - populated by handler from WhatsApp
}

// GroupMemberRepository handles database operations for group memberships.
type GroupMemberRepository struct {
	db *database.DB
}

// NewGroupMemberRepository creates a new group member repository.
func NewGroupMemberRepository(db *database.DB) *GroupMemberRepository {
	return &GroupMemberRepository{db: db}
}

func (r *GroupMemberRepository) Add(groupID int64, jid string) error {
	r.db.Lock()
	defer r.db.Unlock()

	query := `
		INSERT OR IGNORE INTO group_members (group_id, jid, added_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
	`

	_, err := r.db.Conn().Exec(query, groupID, jid)
	if err != nil {
		return fmt.Errorf("failed to add member to group: %w", err)
	}

	return nil
}

// AddMultiple adds multiple contacts to a group in a single transaction.
func (r *GroupMemberRepository) AddMultiple(groupID int64, jids []string) error {
	r.db.Lock()
	defer r.db.Unlock()

	tx, err := r.db.Conn().Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // No-op if committed

	query := `
		INSERT OR IGNORE INTO group_members (group_id, jid, added_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
	`

	stmt, err := tx.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, jid := range jids {
		if _, err := stmt.Exec(groupID, jid); err != nil {
			return fmt.Errorf("failed to add member %s: %w", jid, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Remove removes a contact from a group.
func (r *GroupMemberRepository) Remove(groupID int64, jid string) (bool, error) {
	r.db.Lock()
	defer r.db.Unlock()

	result, err := r.db.Conn().Exec(
		"DELETE FROM group_members WHERE group_id = ? AND jid = ?",
		groupID, jid,
	)
	if err != nil {
		return false, fmt.Errorf("failed to remove member: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected > 0, nil
}

// GetByGroup retrieves all members of a group.
// Note: Name and Phone fields are not populated - the handler must enrich these
// from WhatsApp contact data.
func (r *GroupMemberRepository) GetByGroup(groupID int64) ([]GroupMember, error) {
	r.db.RLock()
	defer r.db.RUnlock()

	query := `
		SELECT id, group_id, jid, added_at
		FROM group_members
		WHERE group_id = ?
		ORDER BY added_at ASC
	`

	rows, err := r.db.Conn().Query(query, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to query group members: %w", err)
	}
	defer rows.Close()

	members := []GroupMember{}

	for rows.Next() {
		var member GroupMember
		if err := rows.Scan(
			&member.ID,
			&member.GroupID,
			&member.JID,
			&member.AddedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan member: %w", err)
		}
		members = append(members, member)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating members: %w", err)
	}

	return members, nil
}

// GetJIDsByGroup returns just the JIDs of all members in a group.
// This is useful for batch operations where you only need the JIDs.
func (r *GroupMemberRepository) GetJIDsByGroup(groupID int64) ([]string, error) {
	r.db.RLock()
	defer r.db.RUnlock()

	query := `
		SELECT jid
		FROM group_members
		WHERE group_id = ?
		ORDER BY added_at ASC
	`

	rows, err := r.db.Conn().Query(query, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to query group members: %w", err)
	}
	defer rows.Close()

	jids := []string{}

	for rows.Next() {
		var jid string
		if err := rows.Scan(&jid); err != nil {
			return nil, fmt.Errorf("failed to scan jid: %w", err)
		}
		jids = append(jids, jid)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating jids: %w", err)
	}

	return jids, nil
}

// IsMember checks if a contact is a member of a group.
func (r *GroupMemberRepository) IsMember(groupID int64, jid string) (bool, error) {
	r.db.RLock()
	defer r.db.RUnlock()

	var exists int
	err := r.db.Conn().QueryRow(
		"SELECT 1 FROM group_members WHERE group_id = ? AND jid = ?",
		groupID, jid,
	).Scan(&exists)

	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check membership: %w", err)
	}

	return true, nil
}

// GetGroupsForContact returns all groups that a contact belongs to.
func (r *GroupMemberRepository) GetGroupsForContact(jid string) ([]int64, error) {
	r.db.RLock()
	defer r.db.RUnlock()

	query := `
		SELECT group_id
		FROM group_members
		WHERE jid = ?
	`

	rows, err := r.db.Conn().Query(query, jid)
	if err != nil {
		return nil, fmt.Errorf("failed to query groups for contact: %w", err)
	}
	defer rows.Close()

	groupIDs := []int64{}

	for rows.Next() {
		var groupID int64
		if err := rows.Scan(&groupID); err != nil {
			return nil, fmt.Errorf("failed to scan group_id: %w", err)
		}
		groupIDs = append(groupIDs, groupID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating groups: %w", err)
	}

	return groupIDs, nil
}

// Count returns the number of members in a group.
func (r *GroupMemberRepository) Count(groupID int64) (int, error) {
	r.db.RLock()
	defer r.db.RUnlock()

	var count int
	err := r.db.Conn().QueryRow(
		"SELECT COUNT(*) FROM group_members WHERE group_id = ?",
		groupID,
	).Scan(&count)

	if err != nil {
		return 0, fmt.Errorf("failed to count members: %w", err)
	}

	return count, nil
}
